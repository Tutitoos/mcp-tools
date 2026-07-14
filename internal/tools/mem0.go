package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
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

// Pinned to elvismdev/mem0-mcp-selfhosted main HEAD as of 2026-07-13
// (closes H6/F7, review y auditoría 2026-07). `uv tool install` resolves
// the `@<sha>` rev, so installs are reproducible; upgrading = reviewing
// upstream and bumping this SHA.
const mem0GitURL = "git+https://github.com/elvismdev/mem0-mcp-selfhosted.git@a4f538afc60ca13a9f5975e6a11fd36e578393ac"

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
		log(fmt.Sprintf("$ ln -snf %s/scripts/wrappers/mem0-launcher %s/.local/bin/mem0-launcher", config.RepoRoot(), home))
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
		log(fmt.Sprintf("$ uv tool install --force --from %s mem0-mcp-selfhosted", mem0GitURL))
		return nil
	}
	// With a pinned rev, `uv tool upgrade` is a no-op by design; upgrade
	// converges the install onto the current pin instead.
	cmd := exec.Command(uvBin(home), "tool", "install", "--force", "--from", mem0GitURL, "mem0-mcp-selfhosted")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	if err := runCombined(cmd, "uv tool install --force mem0-mcp-selfhosted"); err != nil {
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
	installed := which("mem0-mcp-selfhosted") != "" || fileExists(launcher)
	if !installed {
		log("  mem0 no está instalado — nada que desinstalar")
		return nil
	}
	if _, err := exec.LookPath("uv"); err == nil || fileExists(uvBin(home)) {
		cmd := exec.Command(resolveUV(home), "tool", "uninstall", "mem0-mcp-selfhosted")
		cmd.Env = withLocalBinPath(os.Environ(), home)
		cmd.Env = append(cmd.Env, "HOME="+home)
		if err := runCombined(cmd, "uv tool uninstall mem0-mcp-selfhosted"); err != nil {
			log(fmt.Sprintf("WARN uv tool uninstall mem0-mcp-selfhosted: %v", err))
		}
	} else {
		log("WARN mem0 instalado pero uv no está disponible para desinstalarlo — bórralo a mano")
	}
	_ = os.Remove(launcher)
	return nil
}

func statusMem0() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	p := StatusPayload{MCPClients: []string{}}
	bin := which("mem0-mcp-selfhosted")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	// OMP
	if hasKeyIn(filepath.Join(home, ".omp/agent/mcp.json"), []string{"mcpServers", "mcp_tools_mem0"}) {
		p.MCPClients = append(p.MCPClients, "omp")
	}
	// OpenCode
	if hasKeyIn(filepath.Join(home, ".config/opencode/opencode.json"), []string{"mcp", "mcp_tools_mem0"}) {
		p.MCPClients = append(p.MCPClients, "opencode")
	}
	// Claude
	if out, err := exec.Command("claude", "mcp", "list").Output(); err == nil {
		if strings.Contains(string(out), "mcp_tools_mem0") {
			p.MCPClients = append(p.MCPClients, "claude")
		}
	}
	return p, nil
}

// installMem0Launcher symlinks the repo wrapper into ~/.local/bin. The
// wrapper sources .env.mem0 and execs mem0-mcp-selfhosted.
func installMem0Launcher(home string) error {
	src := filepath.Join(config.RepoRoot(), "scripts", "wrappers", "mem0-launcher")
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
