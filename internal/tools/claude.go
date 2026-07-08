package tools

import (
	"os"
	"os/exec"
)

// claudeTool is the Claude Code CLI by Anthropic. It's a CLIENT that consumes
// the MCP servers mcp-config registers into ~/.claude.json. We only install
// the binary here — wiring servers into it is mcp.ConfigureClaude's job.
func claudeTool() Tool {
	return Tool{
		Key:       "claude",
		Label:     "Claude Code CLI",
		Summary:   "Cliente MCP de Anthropic (curl installer oficial; ~/.local/bin)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installClaude,
		Upgrade:   upgradeClaude,
		Uninstall: uninstallClaude,
		Status:    statusClaude,
	}
}

// claudeInstallURL is the official Anthropic-provided installer. It writes
// the binary to ~/.local/bin/claude and adds it to PATH (idempotent).
const claudeInstallURL = "https://claude.ai/install.sh"

func installClaude(dry bool, log func(string)) error {
	if dry {
		log("$ curl -fsSL " + claudeInstallURL + " | bash")
		return nil
	}
	cmd := exec.Command("bash", "-c", "curl -fsSL "+claudeInstallURL+" | bash")
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func upgradeClaude(dry bool, log func(string)) error {
	// The official installer is idempotent: re-running it picks the latest
	// stable release and replaces the existing binary in place.
	return installClaude(dry, log)
}

func uninstallClaude(dry bool, log func(string)) error {
	bin, err := exec.LookPath("claude")
	if err != nil {
		log("  claude no está en PATH — nada que desinstalar")
		return nil
	}
	if dry {
		log("$ rm -f " + bin)
		return nil
	}
	if err := os.Remove(bin); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func statusClaude() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("claude"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}
