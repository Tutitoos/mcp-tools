package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func headroomTool() Tool {
	return Tool{
		Key:       "headroom",
		Label:     "Headroom",
		Summary:   "Compresión + retrieval MCP + proxy opcional (host, uv tool)",
		Deploy:    DeployHost,
		DefaultOn: true,
		Install:   installHeadroom,
		Upgrade:   upgradeHeadroom,
		Uninstall: uninstallHeadroom,
		Status:    statusHeadroom,
	}
}

const headroomSpec = "headroom-ai[mcp,proxy]"

func installHeadroom(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf(`$ uv tool install "%s"`, headroomSpec))
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "install", headroomSpec)
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(out.String())
		if strings.Contains(msg, "Failed to build") && strings.Contains(msg, "mitmproxy") {
			log("WARN extras [proxy] fallaron; reintentando sin el proxy extra")
			retry := exec.Command(uvBin(home), "tool", "install", "headroom-ai[mcp]")
			retry.Env = cmd.Env
			return runCombined(retry, "uv tool install headroom-ai[mcp]")
		}
		return fmt.Errorf("uv tool install %s: %w\n%s", headroomSpec, err, msg)
	}
	return nil
}

func upgradeHeadroom(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	if dry {
		log("$ uv tool upgrade headroom-ai")
		return nil
	}
	cmd := exec.Command(uvBin(home), "tool", "upgrade", "headroom-ai")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	return runCombined(cmd, "uv tool upgrade headroom-ai")
}

func uninstallHeadroom(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log("$ uv tool uninstall headroom-ai")
		return nil
	}
	cmd := exec.Command(resolveUV(home), "tool", "uninstall", "headroom-ai")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	// Best-effort — uv errors if already gone.
	_ = runCombined(cmd, "uv tool uninstall headroom-ai")
	return nil
}

func statusHeadroom() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	p := StatusPayload{MCPClients: []string{}}
	bin := which("headroom")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	// OMP
	if hasKeyIn(filepath.Join(home, ".omp/agent/mcp.json"), []string{"mcpServers", "mcp_tools_headroom"}) {
		p.MCPClients = append(p.MCPClients, "omp")
	}
	// OpenCode
	if hasKeyIn(filepath.Join(home, ".config/opencode/opencode.json"), []string{"mcp", "mcp_tools_headroom"}) {
		p.MCPClients = append(p.MCPClients, "opencode")
	}
	// Claude
	if out, err := exec.Command("claude", "mcp", "list").Output(); err == nil {
		if strings.Contains(string(out), "mcp_tools_headroom") {
			p.MCPClients = append(p.MCPClients, "claude")
		}
	}
	return p, nil
}
