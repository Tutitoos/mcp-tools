package tools

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func mem0Tool() Tool {
	return Tool{
		Key:       "mem0",
		Label:     "mem0-mcp-selfhosted",
		Summary:   "Memoria persistente (host binary; requiere qdrant + ollama)",
		Deploy:    DeployHost,
		Deps:      []string{"qdrant", "ollama"},
		DefaultOn: true,
		Install:   installMem0,
		Upgrade:   upgradeMem0,
		Uninstall: uninstallMem0,
		Status:    statusMem0,
	}
}

const mem0GitURL = "git+https://github.com/elvismdev/mem0-mcp-selfhosted.git"

func installMem0(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf(`$ uv tool install --from %s mem0-mcp-selfhosted`, mem0GitURL))
		log(fmt.Sprintf("$ ln -snf %s/scripts/wrappers/mem0-launcher %s/.local/bin/mem0-launcher", mustRepoRoot(), home))
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "install", "--from", mem0GitURL, "mem0-mcp-selfhosted")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	if err := runCombined(cmd, "uv tool install mem0-mcp-selfhosted"); err != nil {
		return err
	}
	return installMem0Launcher(home)
}

func upgradeMem0(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log("$ uv tool upgrade mem0-mcp-selfhosted")
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "upgrade", "mem0-mcp-selfhosted")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	if err := runCombined(cmd, "uv tool upgrade mem0-mcp-selfhosted"); err != nil {
		// Not installed? fall back to install.
		if strings.Contains(err.Error(), "not installed") {
			return installMem0(dry, log)
		}
		return err
	}
	return installMem0Launcher(home)
}

func uninstallMem0(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	launcher := filepath.Join(home, ".local", "bin", "mem0-launcher")
	if dry {
		log("$ uv tool uninstall mem0-mcp-selfhosted")
		log("$ rm -f " + launcher)
		return nil
	}
	if _, err := exec.LookPath("uv"); err == nil || fileExists(uvBin(home)) {
		cmd := exec.Command(resolveUV(home), "tool", "uninstall", "mem0-mcp-selfhosted")
		cmd.Env = withLocalBinPath(os.Environ(), home)
		cmd.Env = append(cmd.Env, "HOME="+home)
		// Best-effort; uv exits non-zero if the package was already removed.
		_ = runCombined(cmd, "uv tool uninstall mem0-mcp-selfhosted")
	}
	_ = os.Remove(launcher)
	return nil
}

func statusMem0() (StatusPayload, error) {
	p := StatusPayload{}
	bin := which("mem0-mcp-selfhosted")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	return p, nil
}

// installMem0Launcher symlinks the repo wrapper into ~/.local/bin. The
// wrapper sources .env.mem0 and execs mem0-mcp-selfhosted.
func installMem0Launcher(home string) error {
	src := filepath.Join(mustRepoRoot(), "scripts", "wrappers", "mem0-launcher")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("wrapper mem0-launcher no encontrado en repo: %w", err)
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(binDir, "mem0-launcher")
	_ = os.Remove(dst)
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("symlink mem0-launcher: %w", err)
	}
	return nil
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func resolveUV(home string) string {
	if p := which("uv"); p != "" {
		return p
	}
	return uvBin(home)
}

// mustRepoRoot returns config.RepoRoot() indirected via env to avoid an import
// cycle when this file is compiled early.
func mustRepoRoot() string {
	if r := os.Getenv("MCP_TOOLS_ROOT"); r != "" {
		return r
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, "mcp-tools")
	}
	return ""
}

// safety net for imports the linter might complain about if the file is trimmed later.
var _ = errors.New
