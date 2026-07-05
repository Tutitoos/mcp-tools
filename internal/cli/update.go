package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

var (
	updateSelf  bool
	updateTools bool
	updateDry   bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Actualiza el binario mcp-tools + los componentes instalados.",
	Long:  "Sin flags: hace self-update (git pull + make install) y luego actualiza cada tool. --self salta tools; --tools salta self.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateSelf, "self", false, "solo actualiza mcp-tools (git pull + make install)")
	updateCmd.Flags().BoolVar(&updateTools, "tools", false, "solo actualiza los tools seleccionados; salta self-update")
	updateCmd.Flags().BoolVar(&updateDry, "dry", false, "no ejecuta comandos; solo muestra qué haría")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if updateSelf && updateTools {
		return errors.New("--self y --tools son mutuamente excluyentes; sin flags corren ambos")
	}
	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }

	// Self-update: skip when --tools.
	if !updateTools {
		if err := runSelfUpdate(updateDry, logf); err != nil {
			return err
		}
	}
	// Tool updates: skip when --self.
	if !updateSelf {
		st, err := state.Load()
		if err != nil {
			return err
		}
		if len(st.Selected) == 0 {
			logf("SKIP: state vacío — nada que actualizar (corre 'mcp-tools install' primero)")
			return nil
		}
		sudoKeys, tuiKeys, interKeys := partitionByStdio(st.Selected)
		if err := runInlineTools("upgrade", sudoKeys, updateDry, logf); err != nil {
			return err
		}
		if err := runToolSteps("upgrade", tuiKeys, updateDry); err != nil {
			return err
		}
		if err := runInlineTools("upgrade", interKeys, updateDry, logf); err != nil {
			return err
		}
		if updateDry {
			return nil
		}
		st.Versions = collectVersions(st.Selected)
		if err := st.Save(); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
		fmt.Fprintf(os.Stdout, "── update completo — %d tools revisados\n", len(st.Selected))
	}
	return nil
}

func runSelfUpdate(dry bool, log func(string)) error {
	root := config.RepoRoot()
	if dry {
		log(fmt.Sprintf("$ git -C %s fetch --tags origin main", root))
		log(fmt.Sprintf("$ git -C %s pull --ff-only origin main", root))
		log(fmt.Sprintf("$ make -C %s install", root))
		return nil
	}
	// Is it a git checkout?
	if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		log(fmt.Sprintf("SKIP self-update: %s no es git checkout. Clónalo con `git clone git@github.com:Tutitoos/mcp-tools.git %s`.", root, root))
		return nil
	}
	if err := runCmdWithLog("git", []string{"-C", root, "fetch", "--tags", "origin", "main"}); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	local, err1 := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	remote, err2 := exec.Command("git", "-C", root, "rev-parse", "origin/main").Output()
	if err1 == nil && err2 == nil && strings.TrimSpace(string(local)) == strings.TrimSpace(string(remote)) {
		log(fmt.Sprintf("mcp-tools ya actualizado (%s)", strings.TrimSpace(string(local))[:7]))
		return nil
	}
	if err := runCmdWithLog("git", []string{"-C", root, "pull", "--ff-only", "origin", "main"}); err != nil {
		return fmt.Errorf("git pull --ff-only (¿cambios locales sin commit? prueba `git stash` y reintenta): %w", err)
	}
	if err := runCmdWithLog("make", []string{"-C", root, "install"}); err != nil {
		return fmt.Errorf("make install: %w", err)
	}
	if v, err := exec.Command("git", "-C", root, "describe", "--tags", "--always").Output(); err == nil {
		log(fmt.Sprintf("mcp-tools actualizado a %s", strings.TrimSpace(string(v))))
	}
	return nil
}

// runCmdWithLog runs an exec.Command with combined output threaded into a buffer
// (surfaced on error) — but stdio inherit for interactive make/apt output would
// hide too much; use a bytes.Buffer and wrap on failure.
func runCmdWithLog(bin string, args []string) error {
	c := exec.Command(bin, args...)
	c.Env = os.Environ()
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s %s: %w\n%s", bin, strings.Join(args, " "), err, strings.TrimSpace(buf.String()))
	}
	return nil
}
