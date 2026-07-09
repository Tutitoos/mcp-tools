package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func serenaTool() Tool {
	return Tool{
		Key:       "serena",
		Label:     "Serena semantic code MCP",
		Summary:   "MCP para retrieval + edición semántica basada en LSP (uv tool, Python 3.13)",
		Deploy:    DeployHost,
		Install:   installSerena,
		Upgrade:   upgradeSerena,
		Uninstall: uninstallSerena,
		Status:    statusSerena,
	}
}

const serenaSpec = "serena-agent"

func installSerena(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log("$ uv tool install -p 3.13 " + serenaSpec)
		log("$ serena init --yes")
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "install", "-p", "3.13", serenaSpec)
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv tool install %s: %w\n%s", serenaSpec, err, strings.TrimSpace(out.String()))
	}
	// Always run `serena init --yes` with stdio captured. Bubbletea owns
	// the terminal during mcp-tools install, so we can't pass os.Stdin
	// through; serena would otherwise interleave its setup probing (and
	// a flurry of harmless 'claude: not found' / 'codex: not found'
	// stderr lines for IDEs the user hasn't installed yet) into the
	// progress UI render. Capturing also keeps the install idempotent
	// and non-interactive.
	initCmd := exec.Command("serena", "init", "--yes")
	initCmd.Env = withLocalBinPath(os.Environ(), home)
	initCmd.Env = append(initCmd.Env, "HOME="+home)
	var initOut bytes.Buffer
	initCmd.Stdout = &initOut
	initCmd.Stderr = &initOut
	if err := initCmd.Run(); err != nil {
		if strings.Contains(initOut.String(), "unknown flag") || strings.Contains(initOut.String(), "unrecognized") {
			log("WARN serena init --yes no soportado por esta versión; salta auto-register. Corre 'serena init' manualmente tras el install.")
			return nil
		}
		return fmt.Errorf("serena init: %w\n%s", err, strings.TrimSpace(initOut.String()))
	}
	return nil
}

func upgradeSerena(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log("$ uv tool upgrade " + serenaSpec)
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "upgrade", serenaSpec)
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	return runCombined(cmd, "uv tool upgrade "+serenaSpec)
}

func uninstallSerena(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log("$ uv tool uninstall " + serenaSpec)
		return nil
	}
	if which("serena") == "" {
		log("  serena no está instalado — nada que desinstalar")
		return nil
	}
	cmd := exec.Command(resolveUV(home), "tool", "uninstall", serenaSpec)
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	if err := runCombined(cmd, "uv tool uninstall "+serenaSpec); err != nil {
		log(fmt.Sprintf("WARN uv tool uninstall %s: %v", serenaSpec, err))
	}
	return nil
}

func statusSerena() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	p := StatusPayload{MCPClients: []string{}}
	bin := which("serena")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	// OMP
	if hasKeyIn(filepath.Join(home, ".omp/agent/mcp.json"), []string{"mcpServers", "mcp_tools_serena"}) {
		p.MCPClients = append(p.MCPClients, "omp")
	}
	// OpenCode
	if hasKeyIn(filepath.Join(home, ".config/opencode/opencode.json"), []string{"mcp", "mcp_tools_serena"}) {
		p.MCPClients = append(p.MCPClients, "opencode")
	}
	// Claude
	if out, err := exec.Command("claude", "mcp", "list").Output(); err == nil {
		if strings.Contains(string(out), "mcp_tools_serena") {
			p.MCPClients = append(p.MCPClients, "claude")
		}
	}
	return p, nil
}
