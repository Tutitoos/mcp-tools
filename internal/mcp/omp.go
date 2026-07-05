package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// ConfigureOMP merges the ServerSpec list into ~/.omp/agent/mcp.json under `.mcpServers`.
// Preserves other keys (disabledServers, etc.). SKIP silently if the parent dir is missing.
func ConfigureOMP(st state.State, log func(string)) error {
	home, _ := os.UserHomeDir()
	file := filepath.Join(home, ".omp/agent/mcp.json")
	parent := filepath.Dir(file)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			log(fmt.Sprintf("  SKIP OMP (%s missing — OMP not installed?)", parent))
			return nil
		}
		return err
	}
	if err := Backup(file); err != nil {
		return err
	}

	fallback := map[string]any{
		"$schema":         "https://raw.githubusercontent.com/can1357/oh-my-pi/main/packages/coding-agent/src/config/mcp-schema.json",
		"mcpServers":      map[string]any{},
		"disabledServers": []any{},
	}
	cfg, err := LoadJSON(file, fallback)
	if err != nil {
		return err
	}

	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	wanted := map[string]bool{}
	for _, s := range Servers(st) {
		wanted[s.Name] = true
		servers[s.Name] = map[string]any{
			"command": s.Wrapper,
			"args":    argsToAny(s.Args),
			"env":     map[string]any{"HOME": home},
			"enabled": true,
		}
	}
	// Prune obsolete mcp_tools_* keys we no longer own.
	for k := range servers {
		if strings.HasPrefix(k, "mcp_tools_") && !wanted[k] {
			delete(servers, k)
			log(fmt.Sprintf("  prune OMP %s (obsolete)", k))
		}
	}
	cfg["mcpServers"] = servers

	if err := WriteJSON(file, cfg); err != nil {
		return err
	}
	log(fmt.Sprintf("  OK OMP %s", file))
	return nil
}
