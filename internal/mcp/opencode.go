package mcp

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigureOpenCode merges the ServerSpec list into ~/.config/opencode/opencode.json
// under `.mcp`. Preserves every other key. SKIP silently if the parent dir is missing
// (OpenCode not installed).
func ConfigureOpenCode(log func(string)) error {
	home, _ := os.UserHomeDir()
	file := filepath.Join(home, ".config/opencode/opencode.json")
	parent := filepath.Dir(file)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			log(fmt.Sprintf("SKIP OpenCode (%s missing — OpenCode not installed?)", parent))
			return nil
		}
		return err
	}
	if err := Backup(file); err != nil {
		return err
	}

	fallback := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"mcp":     map[string]any{},
	}
	cfg, err := LoadJSON(file, fallback)
	if err != nil {
		return err
	}

	mcpSection, _ := cfg["mcp"].(map[string]any)
	if mcpSection == nil {
		mcpSection = map[string]any{}
	}

	for _, s := range Servers() {
		cmdList := append([]any{s.Wrapper}, argsToAny(s.Args)...)
		mcpSection[s.Name] = map[string]any{
			"type":        "local",
			"command":     cmdList,
			"enabled":     true,
			"environment": map[string]any{"HOME": home},
		}
	}
	cfg["mcp"] = mcpSection

	if err := WriteJSON(file, cfg); err != nil {
		return err
	}
	log(fmt.Sprintf("OK OpenCode %s", file))
	return nil
}

func argsToAny(args []string) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = a
	}
	return out
}
