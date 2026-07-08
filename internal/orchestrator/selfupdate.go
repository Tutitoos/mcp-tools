package orchestrator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// RunSelfUpdate runs the git-based self-update (mirrors the legacy
// `internal/cli.runSelfUpdate`). Used by both `mcp-tools update --self`
// and the web panel's /api/update/self.
func RunSelfUpdate(dry bool, log LogFn) error {
	if log == nil {
		log = func(string) {}
	}
	root := repoRoot()
	if dry {
		log(fmt.Sprintf("$ git -C %s fetch --tags origin main", root))
		log(fmt.Sprintf("$ git -C %s pull --ff-only origin main", root))
		log(fmt.Sprintf("$ make -C %s install", root))
		return nil
	}
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
	binDir := os.Getenv("MCP_TOOLS_BIN")
	if binDir == "" {
		binDir = config.WrapperDir()
	}
	binPath := filepath.Join(binDir, "mcp-tools")
	if err := runCmdWithLog(binPath, []string{"--version"}); err != nil {
		log(fmt.Sprintf("WARN mcp-tools instalado en %s está roto (--version falla). Re-corre 'make -C %s install' o revisa el PATH.", binPath, root))
		return fmt.Errorf("post-install verify: %w", err)
	}
	if v, err := exec.Command("git", "-C", root, "describe", "--tags", "--always").Output(); err == nil {
		log(fmt.Sprintf("mcp-tools actualizado a %s", strings.TrimSpace(string(v))))
	}
	return nil
}

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