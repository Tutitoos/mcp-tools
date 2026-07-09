package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func tokensaveTool() Tool {
	return Tool{
		Key:           "tokensave",
		Label:         "TokenSave semantic code MCP",
		Summary:       "MCP semántico Rust — 40+ tools, 30+ lenguajes, autodetecta agentes",
		Deploy:        DeployHost,
		DefaultOn:     false,
		SelfRegisters: true,
		Install: func(dry bool, log func(string)) error {
			return installTokensave(dry, log, false)
		},
		Upgrade: func(dry bool, log func(string)) error {
			return installTokensave(dry, log, true)
		},
		Uninstall: uninstallTokensave,
		Status:    statusTokensave,
	}
}

const tokensaveCrate = "tokensave"

func installTokensave(dry bool, log func(string), force bool) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureCargo(dry, log, home); err != nil {
		return err
	}
	args := []string{"install", tokensaveCrate, "--locked"}
	if force {
		args = append(args, "--force")
	}
	if dry {
		log("$ cargo " + strings.Join(args, " "))
		log("$ tokensave install --git-hook no")
		return nil
	}
	cmd := exec.Command(cargoBin(home), args...)
	cmd.Env = withCargoPath(os.Environ(), home)
	if err := runCombined(cmd, "cargo install tokensave"); err != nil {
		return err
	}
	bin := filepath.Join(home, ".cargo/bin/tokensave")
	reg := exec.Command(bin, "install", "--git-hook", "no")
	reg.Env = withCargoPath(os.Environ(), home)
	reg.Env = append(reg.Env, "HOME="+home)
	if err := runCombined(reg, "tokensave install --git-hook no"); err != nil {
		return err
	}
	return nil
}

func uninstallTokensave(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log("$ tokensave uninstall  # strip MCP configs, hooks y CLAUDE.md rules de cada agente")
		log("$ cargo uninstall tokensave  # remove el binario")
		return nil
	}
	bin := which("tokensave")
	directBin := filepath.Join(home, ".cargo/bin/tokensave")
	installed := bin != "" || fileExists(directBin)
	if !installed {
		log("  tokensave no está instalado — nada que desinstalar")
		return nil
	}
	if bin == "" {
		bin = directBin
	}
	// tokensave uninstall PRIMERO (mientras binario existe) para limpiar registros
	// en ~/.claude.json, ~/.config/opencode/*, ~/.omp/*, etc.
	// Después cargo uninstall borra el binario. Invertir dejaría configs colgando.
	// Best-effort: si tokensave uninstall falla, seguimos con cargo (pero avisamos).
	cmd := exec.Command(bin, "uninstall")
	cmd.Env = withCargoPath(os.Environ(), home)
	cmd.Env = append(cmd.Env, "HOME="+home)
	if err := runCombined(cmd, "tokensave uninstall"); err != nil {
		log(fmt.Sprintf("WARN tokensave uninstall: %v (continuando con cargo uninstall)", err))
	}
	var cargoCmd *exec.Cmd
	if cbin, err := exec.LookPath("cargo"); err == nil {
		cargoCmd = exec.Command(cbin, "uninstall", tokensaveCrate)
	} else if _, err := os.Stat(cargoBin(home)); err == nil {
		cargoCmd = exec.Command(cargoBin(home), "uninstall", tokensaveCrate)
	}
	if cargoCmd != nil {
		if err := runCombined(cargoCmd, "cargo uninstall tokensave"); err != nil {
			log(fmt.Sprintf("WARN cargo uninstall tokensave: %v", err))
		}
	} else {
		log("WARN tokensave binario presente pero cargo no está disponible para desinstalarlo — bórralo a mano: " + directBin)
	}
	log("NOTE: si `tokensave uninstall` no cubrió algún cliente, revisa manualmente ~/.claude.json, ~/.config/opencode/opencode.json y ~/.omp/agent/mcp.json por entradas `tokensave` residuales")
	return nil
}

func statusTokensave() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	p := StatusPayload{MCPClients: []string{}}
	bin := which("tokensave")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	if hasKeyIn(filepath.Join(home, ".omp/agent/mcp.json"), []string{"mcpServers", "tokensave"}) {
		p.MCPClients = append(p.MCPClients, "omp")
	}
	if hasKeyIn(filepath.Join(home, ".config/opencode/opencode.json"), []string{"mcp", "tokensave"}) {
		p.MCPClients = append(p.MCPClients, "opencode")
	}
	if hasKeyIn(filepath.Join(home, ".claude.json"), []string{"mcpServers", "tokensave"}) {
		p.MCPClients = append(p.MCPClients, "claude")
	}
	return p, nil
}
