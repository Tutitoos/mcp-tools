package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

// ConfigureGemini registers each ServerSpec in the Google Gemini CLI config
// at ~/.gemini/settings.json under `mcpServers` (same shape as Claude's
// ~/.claude.json). Preserves every other top-level key. SKIP silently if the
// parent dir is missing or `gemini` CLI is absent — the latter is the
// cheapest signal that the user hasn't installed Gemini yet.
func ConfigureGemini(st state.State, log func(string)) error {
	if _, err := exec.LookPath("gemini"); err != nil {
		log("  SKIP Gemini (gemini CLI not found)")
		return nil
	}
	home, err := config.HomeDir()
	if err != nil {
		return fmt.Errorf("gemini mcp: %w", err)
	}
	file := filepath.Join(home, ".gemini", "settings.json")
	parent := filepath.Dir(file)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			log(fmt.Sprintf("  SKIP Gemini (%s missing — Gemini CLI not installed?)", parent))
			return nil
		}
		return err
	}
	if err := Backup(file); err != nil {
		return err
	}

	fallback := map[string]any{
		"mcpServers": map[string]any{},
	}
	cfg, err := LoadJSON(file, fallback)
	if err != nil {
		return err
	}

	section, _ := cfg["mcpServers"].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}

	wanted := map[string]bool{}
	for _, s := range Servers(st) {
		wanted[s.Name] = true
		section[s.Name] = map[string]any{
			"command": s.Wrapper,
			"args":    argsToAny(s.Args),
			"env":     serverEnvironment(s, home),
			"enabled": true,
		}
	}
	for k := range section {
		if strings.HasPrefix(k, "mcp_tools_") && !wanted[k] {
			delete(section, k)
			log(fmt.Sprintf("  prune Gemini %s (obsolete)", k))
		}
	}
	cfg["mcpServers"] = section

	if err := WriteJSON(file, cfg); err != nil {
		return err
	}
	log(fmt.Sprintf("  OK Gemini %s", file))
	return nil
}
