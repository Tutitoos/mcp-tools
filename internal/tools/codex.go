package tools

import "fmt"

// codexTool is the OpenAI Codex CLI. It's a CLIENT that consumes MCP servers
// registered in ~/.codex/config.toml (see mcp.ConfigureCodex). We install the
// npm package globally; on a root-owned prefix (/usr/local) that needs
// elevation, so Deploy = DeploySudo and runNpmGlobal pre-checks writability
// instead of dying mid-install (INS-04).
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
	return runNpmGlobal("install", codexPackage)
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
	return runNpmGlobal("install", codexPackage)
}

func uninstallCodex(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm uninstall -g %s", codexPackage))
		return nil
	}
	return runNpmGlobal("uninstall", codexPackage)
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
