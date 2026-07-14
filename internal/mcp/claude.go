package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

func claudeServerSpec(s ServerSpec, home string) map[string]any {
	spec := map[string]any{
		"type":    "stdio",
		"command": s.Wrapper,
		"env":     serverEnvironment(s, home),
	}
	if len(s.Args) > 0 {
		spec["args"] = s.Args
	}
	return spec
}

// ConfigureClaude registers each ServerSpec in Claude Code via `claude mcp add-json`.
// SKIP silently if the `claude` CLI is not on PATH.
func ConfigureClaude(st state.State, log func(string)) error {
	if _, err := exec.LookPath("claude"); err != nil {
		log("  SKIP Claude Code (claude CLI not found)")
		return nil
	}
	home, err := config.HomeDir()
	if err != nil {
		return fmt.Errorf("claude mcp: %w", err)
	}

	wanted := map[string]bool{}
	for _, s := range Servers(st) {
		wanted[s.Name] = true
	}

	// Prune obsolete mcp_tools_* entries the user still has registered.
	listCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	listOut, err := exec.CommandContext(listCtx, "claude", "mcp", "list").Output()
	if err != nil {
		log(fmt.Sprintf("  SKIP Claude prune — 'claude mcp list' falló: %v", err))
	} else {
		for _, line := range strings.Split(string(listOut), "\n") {
			name, _, ok := strings.Cut(line, ":")
			name = strings.TrimSpace(name)
			if !ok || !strings.HasPrefix(name, "mcp_tools_") || wanted[name] {
				continue
			}
			rm := exec.Command("claude", "mcp", "remove", "--scope", "user", name)
			if err := rm.Run(); err == nil {
				log(fmt.Sprintf("  prune Claude %s (obsolete)", name))
			}
		}
	}

	for _, s := range Servers(st) {
		// remove is idempotent; ignore exit code (server may not exist yet)
		removeCmd := exec.Command("claude", "mcp", "remove", "--scope", "user", s.Name)
		removeCmd.Stdout = nil
		removeCmd.Stderr = nil
		_ = removeCmd.Run()

		spec := claudeServerSpec(s, home)
		blob, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("claude %s: marshal: %w", s.Name, err)
		}
		addCmd := exec.Command("claude", "mcp", "add-json", "--scope", "user", s.Name, string(blob))
		out, err := addCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("claude mcp add-json %s: %w: %s", s.Name, err, string(out))
		}
	}

	log(fmt.Sprintf("  OK Claude Code (%d servers)", len(Servers(st))))
	return nil
}
