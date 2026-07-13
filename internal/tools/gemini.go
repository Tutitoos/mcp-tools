package tools

import "fmt"

// geminiTool is the Google Gemini CLI. CLIENT that consumes MCP servers
// registered in ~/.gemini/settings.json (see mcp.ConfigureGemini). npm
// global install may write to a root-owned prefix (/usr/local) →
// DeploySudo; runNpmGlobal pre-checks writability instead of dying
// mid-install (INS-04).
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
	return runNpmGlobal("install", geminiPackage)
}

func upgradeGemini(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm install -g %s  # upgrade", geminiPackage))
		return nil
	}
	// `npm install -g <pkg>` re-resolves to latest tag and overwrites — same
	// effect as `npm update -g` without needing the package to be pre-installed.
	return runNpmGlobal("install", geminiPackage)
}

func uninstallGemini(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ npm uninstall -g %s", geminiPackage))
		return nil
	}
	return runNpmGlobal("uninstall", geminiPackage)
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
