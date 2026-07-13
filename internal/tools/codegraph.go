package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func codegraphTool() Tool {
	return Tool{
		Key:           "codegraph",
		Label:         "CodeGraph MCP",
		Summary:       "Grafo semántico + auto-registro en 8 IDEs (opt-in)",
		Deploy:        DeployHost,
		DefaultOn:     false,
		SelfRegisters: true,
		Install:       installCodegraph,
		Upgrade:       installCodegraph, // installer script is idempotent
		Uninstall:     uninstallCodegraph,
		Status:        statusCodegraph,
	}
}

// Pinned to a reviewed commit of colbymchenry/codegraph `main` (2026-07-13)
// + SHA-256 of the script at that commit. fetchVerified refuses to execute
// anything else. Bump = review the new script, then update BOTH constants.
// Closes AUDIT-2026-07-11 F7 / REVIEW-rd2 H27.
const (
	codegraphInstallURL    = "https://raw.githubusercontent.com/colbymchenry/codegraph/e871c49a3173a637172f501f21f6a2753ea5a39f/install.sh"
	codegraphInstallSHA256 = "f4e90c6e0c1d2ac95a43fa6e82e4caf76fabdb18310afc72597314b58632e56c"
)

func installCodegraph(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ curl -fsSL %s | sh  # sha256-verified", codegraphInstallURL))
		log("$ codegraph install --yes")
		return nil
	}
	// 1. Upstream bundle installer, pinned + checksum-verified.
	script, err := fetchVerified(codegraphInstallURL, codegraphInstallSHA256)
	if err != nil {
		return fmt.Errorf("codegraph install.sh: %w", err)
	}
	shell := exec.Command("sh")
	shell.Env = append(os.Environ(), "HOME="+home)
	shell.Stdin = bytes.NewReader(script)
	var out bytes.Buffer
	shell.Stdout = &out
	shell.Stderr = &out
	if err := shell.Run(); err != nil {
		return fmt.Errorf("codegraph install.sh: %w\n%s", err, strings.TrimSpace(out.String()))
	}

	// 2. Auto-register in every discoverable IDE.
	bin := which("codegraph")
	if bin == "" {
		return errors.New("codegraph binario no encontrado tras install.sh — inspecciona ~/.local/bin y ~/.codegraph")
	}
	reg := exec.Command(bin, "install", "--yes")
	reg.Env = os.Environ()
	if err := runCombined(reg, "codegraph install --yes"); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unknown flag") || strings.Contains(errStr, "unrecognized") {
			log("WARN codegraph install --yes no soportado por esta versión; salta auto-register. Corre 'codegraph install' manualmente.")
			return nil
		}
		return fmt.Errorf("codegraph auto-register: %w", err)
	}
	return nil
}

func uninstallCodegraph(dry bool, log func(string)) error {
	if dry {
		log("$ codegraph uninstall --yes")
		return nil
	}
	bin := which("codegraph")
	if bin == "" {
		log("  codegraph no está instalado — nada que desinstalar")
		return nil
	}
	cmd := exec.Command(bin, "uninstall", "--yes")
	cmd.Env = os.Environ()
	if err := runCombined(cmd, "codegraph uninstall --yes"); err != nil {
		log(fmt.Sprintf("WARN codegraph uninstall --yes: %v", err))
	}
	return nil
}

func statusCodegraph() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("codegraph"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}
