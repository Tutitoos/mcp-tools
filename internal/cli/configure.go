package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var configureDry bool

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Reabre el multi-select TUI para añadir/quitar componentes.",
	RunE:  runConfigure,
}

func init() {
	configureCmd.Flags().BoolVar(&configureDry, "dry", false, "no ejecuta comandos; solo muestra el diff")
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	if len(st.Selected) == 0 {
		return errors.New("state ausente — corre 'mcp-tools install' primero")
	}

	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }
	if err := runBootstrap(configureDry, logf); err != nil {
		return err
	}

	avail := availableTools()
	pre := map[string]bool{}
	for _, k := range st.Selected {
		pre[k] = true
	}
	model, err := runToolSelect(avail, pre)
	if err != nil {
		return err
	}
	if !model.Confirmed() {
		return errors.New("cancelado por el user")
	}
	newSelected := model.Selected()

	toAdd := diffKeys(newSelected, st.Selected)
	toRemove := diffKeys(st.Selected, newSelected)

	// Reverse-deps: uninstall dependents first.
	toRemoveSorted, err := tools.TopoSort(toRemove)
	if err != nil {
		return err
	}
	slices.Reverse(toRemoveSorted)

	toAddSorted, err := tools.TopoSort(toAdd)
	if err != nil {
		return err
	}

	rmSudo, rmTui, rmInter := partitionByStdio(toRemoveSorted)
	addSudo, addTui, addInter := partitionByStdio(toAddSorted)
	// Uninstall: sudo primero (revertir cambios de sistema), TUI en medio,
	// interactive al final (unregister plugin). Install: mismo orden.
	if err := runInlineTools("uninstall", rmSudo, configureDry, logf); err != nil {
		return err
	}
	if err := runToolSteps("uninstall", rmTui, configureDry); err != nil {
		return err
	}
	if err := runInlineTools("uninstall", rmInter, configureDry, logf); err != nil {
		return err
	}
	if err := runInlineTools("install", addSudo, configureDry, logf); err != nil {
		return err
	}
	if err := runToolSteps("install", addTui, configureDry); err != nil {
		return err
	}
	if err := runInlineTools("install", addInter, configureDry, logf); err != nil {
		return err
	}

	stNew := state.State{Selected: newSelected}
	if err := RunMcpConfig(configureDry, stNew, os.Stdout); err != nil {
		return fmt.Errorf("mcp-config: %w", err)
	}
	var buf bytes.Buffer
	_ = RunSkills(configureDry, &buf)
	_ = RunRules(configureDry, &buf)
	if s := strings.TrimRight(buf.String(), "\n"); s != "" {
		fmt.Fprintln(os.Stdout, s)
	}

	if configureDry {
		fmt.Fprintln(os.Stdout, "SKIP (dry) — no toca state.json")
		return nil
	}
	stNew.Versions = collectVersions(newSelected)
	if err := stNew.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	unchanged := len(newSelected) - len(toAdd)
	fmt.Fprintf(os.Stdout, "── configure completo — +%d añadidos, -%d eliminados, =%d sin cambios\n", len(toAdd), len(toRemove), unchanged)
	return nil
}

// diffKeys returns items in `a` that are not in `b`, preserving `a` order.
func diffKeys(a, b []string) []string {
	inB := map[string]bool{}
	for _, k := range b {
		inB[k] = true
	}
	var out []string
	for _, k := range a {
		if !inB[k] {
			out = append(out, k)
		}
	}
	return out
}
