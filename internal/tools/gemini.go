package tools

import (
	"fmt"
	"os"
	"os/exec"
)

// geminiTool is the Google Gemini CLI. CLIENT that consumes MCP servers
// registered in ~/.gemini/settings.json (see mcp.ConfigureGemini). npm
// global install writes to /usr/local/{lib/node_modules,bin} → DeploySudo.
func geminiTool() Tool {
	return Tool{
		Key:       "gemini",
		Label:     "Google Gemini CLI",
		Summary:   "Cliente MCP de Google (npm i -g @google/gemini-cli; requiere sudo)",
		Deploy:    DeploySudo,
		DefaultOn: false,
		Install:   installGemini,
		Upgrade:   upgradeGemini,
		Uninstall: uninstallGemini,
		Status:    statusGemini,
	}
}

const geminiPackage = "@google/gemini-cli"

func installGemini(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm install -g %s", geminiPackage))
		return nil
	}
	cmd := exec.Command("npm", "install", "-g", geminiPackage)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install -g %s: %w", geminiPackage, err)
	}
	return nil
}

func upgradeGemini(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm install -g %s  # upgrade", geminiPackage))
		return nil
	}
	cmd := exec.Command("npm", "install", "-g", geminiPackage)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm upgrade %s: %w", geminiPackage, err)
	}
	return nil
}

func uninstallGemini(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm uninstall -g %s", geminiPackage))
		return nil
	}
	cmd := exec.Command("npm", "uninstall", "-g", geminiPackage)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm uninstall -g %s: %w", geminiPackage, err)
	}
	return nil
}

func statusGemini() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("gemini"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}
