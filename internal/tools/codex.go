package tools

import (
	"fmt"
	"os"
	"os/exec"
)

// codexTool is the OpenAI Codex CLI. It's a CLIENT that consumes MCP servers
// registered in ~/.codex/config.toml (see mcp.ConfigureCodex). We install the
// npm package globally; npm writes to /usr/local/{lib/node_modules,bin} and
// requires root, so Deploy = DeploySudo.
func codexTool() Tool {
	return Tool{
		Key:       "codex",
		Label:     "OpenAI Codex CLI",
		Summary:   "Cliente MCP de OpenAI (npm i -g @openai/codex; requiere sudo)",
		Deploy:    DeploySudo,
		DefaultOn: false,
		Install:   installCodex,
		Upgrade:   upgradeCodex,
		Uninstall: uninstallCodex,
		Status:    statusCodex,
	}
}

const codexPackage = "@openai/codex"

func installCodex(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm install -g %s", codexPackage))
		return nil
	}
	cmd := exec.Command("npm", "install", "-g", codexPackage)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install -g %s: %w", codexPackage, err)
	}
	return nil
}

func upgradeCodex(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm install -g %s  # upgrade (npm -g es idempotente)", codexPackage))
		return nil
	}
	// `npm install -g <pkg>` re-resolves to latest tag and overwrites — same
	// effect as `npm update -g` without needing the package to be pre-installed.
	cmd := exec.Command("npm", "install", "-g", codexPackage)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm upgrade %s: %w", codexPackage, err)
	}
	return nil
}

func uninstallCodex(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm uninstall -g %s", codexPackage))
		return nil
	}
	cmd := exec.Command("npm", "uninstall", "-g", codexPackage)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm uninstall -g %s: %w", codexPackage, err)
	}
	return nil
}

func statusCodex() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("codex"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}
