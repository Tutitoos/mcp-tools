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

const codegraphInstallURL = "https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh"

func installCodegraph(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log(fmt.Sprintf("$ curl -fsSL %s | sh", codegraphInstallURL))
		log("$ codegraph install --yes")
		return nil
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return errors.New("curl no está en PATH; instala curl antes de codegraph")
	}
	// 1. Upstream bundle installer.
	curl := exec.Command("curl", "-fsSL", codegraphInstallURL)
	shell := exec.Command("sh")
	shell.Env = append(os.Environ(), "HOME="+home)
	pipe, err := curl.StdoutPipe()
	if err != nil {
		return err
	}
	shell.Stdin = pipe
	var out bytes.Buffer
	shell.Stdout = &out
	shell.Stderr = &out
	if err := shell.Start(); err != nil {
		return fmt.Errorf("codegraph install.sh start: %w", err)
	}
	if err := curl.Run(); err != nil {
		return fmt.Errorf("codegraph curl: %w", err)
	}
	if err := shell.Wait(); err != nil {
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
		// Fallback: some upstream releases prompt without --yes.
		fallback := exec.Command(bin, "install")
		fallback.Env = os.Environ()
		fallback.Stdin = strings.NewReader("y\n")
		if err2 := runCombined(fallback, "codegraph install (stdin y)"); err2 != nil {
			return fmt.Errorf("codegraph auto-register: %w (fallback: %v)", err, err2)
		}
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
		return nil
	}
	cmd := exec.Command(bin, "uninstall", "--yes")
	cmd.Env = os.Environ()
	_ = runCombined(cmd, "codegraph uninstall --yes")
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
