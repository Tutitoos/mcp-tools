package tools

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func claudeMemTool() Tool {
	return Tool{
		Key:           "claude-mem",
		Label:         "claude-mem",
		Summary:       "Plugin de Claude Code (opt-in; requiere Node ≥ 20)",
		Deploy:        DeployHost,
		DefaultOn:     false,
		SelfRegisters: true,
		Interactive:   true,
		Install:       installClaudeMem,
		Upgrade:       installClaudeMem, // `npx claude-mem@latest install` is idempotent + reinstalls
		Uninstall:     uninstallClaudeMem,
		Status:        statusClaudeMem,
	}
}

func installClaudeMem(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log("$ npx --yes claude-mem@latest install")
		return nil
	}
	// TODO(security): pin claude-mem to a stable version. `@latest` pulls
	// whatever is on npm at install time and is propagated to all users.
	// See docs/REVIEW-rd2.md (H28).
	cmd := exec.Command("npx", "--yes", "claude-mem@latest", "install")
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude-mem install: %w", err)
	}
	return nil
}

func uninstallClaudeMem(dry bool, log func(string)) error {
	if err := ensureNodeMin(20); err != nil {
		return err
	}
	if dry {
		log("$ npx --yes claude-mem@latest uninstall")
		return nil
	}
	// npx fetches its own copy of claude-mem to run `uninstall`, so this
	// works to strip stray MCP configs/hooks/CLAUDE.md rules even if the
	// local ~/.local/bin/claude-mem binary was already removed by hand —
	// unlike the cargo/uv-installed tools, PATH presence isn't the source
	// of truth here, so it is deliberately NOT gated on which("claude-mem").
	// TODO(security): mirror H28 — see docs/REVIEW-rd2.md.
	cmd := exec.Command("npx", "--yes", "claude-mem@latest", "uninstall")
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Best-effort — user may already have removed the plugin — but surface
	// a genuine failure instead of discarding it outright.
	if err := cmd.Run(); err != nil {
		log(fmt.Sprintf("WARN npx claude-mem uninstall: %v", err))
	}
	return nil
}

func statusClaudeMem() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("claude-mem"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}

// ensureNodeMin returns an error if `node --version` reports below minMajor.
func ensureNodeMin(minMajor int) error {
	bin := which("node")
	if bin == "" {
		return errors.New("node no está en PATH — claude-mem requiere Node ≥ 20 (usa nvm o el package manager del sistema)")
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return fmt.Errorf("node --version: %w", err)
	}
	v := strings.TrimSpace(string(out))
	v = strings.TrimPrefix(v, "v")
	dot := strings.Index(v, ".")
	if dot < 0 {
		return fmt.Errorf("node --version salida inesperada: %q", strings.TrimSpace(string(out)))
	}
	major, err := strconv.Atoi(v[:dot])
	if err != nil {
		return fmt.Errorf("node --version parse: %w", err)
	}
	if major < minMajor {
		return fmt.Errorf("node %s < %d requerido por claude-mem", strings.TrimSpace(string(out)), minMajor)
	}
	return nil
}
