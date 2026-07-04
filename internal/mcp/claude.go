package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// ConfigureClaude registers each ServerSpec in Claude Code via `claude mcp add-json`.
// SKIP silently if the `claude` CLI is not on PATH.
func ConfigureClaude(log func(string)) error {
	if _, err := exec.LookPath("claude"); err != nil {
		log("SKIP Claude Code (claude CLI not found)")
		return nil
	}
	home, _ := os.UserHomeDir()

	for _, s := range Servers() {
		// remove is idempotent; ignore exit code (server may not exist yet)
		removeCmd := exec.Command("claude", "mcp", "remove", "--scope", "user", s.Name)
		removeCmd.Stdout = nil
		removeCmd.Stderr = nil
		_ = removeCmd.Run()

		spec := map[string]any{
			"type":    "stdio",
			"command": s.Wrapper,
			"args":    s.Args,
			"env":     map[string]any{"HOME": home},
		}
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

	log(fmt.Sprintf("OK Claude Code (%d servers via claude mcp add-json --scope user)", len(Servers())))
	return nil
}
