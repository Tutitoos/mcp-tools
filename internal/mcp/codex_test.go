package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// TestConfigureCodexPreservesUserSections verifies rewriteCodexConfig
// rewrites only its own `[mcp_servers.mcp_tools_*]` sections and leaves
// every other top-level key and user-added section byte-for-byte intact.
// See B3 in REVIEW: the old flat key=value writer dropped section headers
// entirely. Exercises rewriteCodexConfig directly (not ConfigureCodex) so
// the test doesn't depend on the `codex` CLI being installed on the runner.
func TestConfigureCodexPreservesUserSections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(codexDir, "config.toml")
	fixture := "model = \"gpt-4\"\n\n[mcp_servers.user_kept]\ncommand = \"foo\"\n"
	if err := os.WriteFile(configPath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	var logged []string
	log := func(line string) { logged = append(logged, line) }

	if err := rewriteCodexConfig(state.State{Selected: []string{"serena"}}, log); err != nil {
		t.Fatalf("rewriteCodexConfig: %v", err)
	}

	out, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read back config.toml: %v", err)
	}
	text := string(out)

	if !strings.Contains(text, `model = "gpt-4"`) {
		t.Errorf("user top-level key model was dropped:\n%s", text)
	}
	if n := strings.Count(text, "[mcp_servers.user_kept]"); n != 1 {
		t.Errorf("[mcp_servers.user_kept] appears %d times, want 1:\n%s", n, text)
	}
	if !strings.Contains(text, `command = "foo"`) {
		t.Errorf("user_kept command was dropped:\n%s", text)
	}
	if !strings.Contains(text, "[mcp_servers.mcp_tools_serena]") {
		t.Errorf("mcp_tools_serena section missing:\n%s", text)
	}
	if !strings.Contains(text, `command = "serena"`) {
		t.Errorf("mcp_tools_serena command missing/wrong:\n%s", text)
	}
	wantArgs := `args = ["start-mcp-server", "--context", "agent", "--project-from-cwd"]`
	if !strings.Contains(text, wantArgs) {
		t.Errorf("mcp_tools_serena args missing/wrong, want %q:\n%s", wantArgs, text)
	}
	if !strings.Contains(text, "[mcp_servers.mcp_tools_serena.env]") {
		t.Errorf("mcp_tools_serena.env section missing:\n%s", text)
	}
	if !strings.Contains(text, `HOME = "${HOME}"`) {
		t.Errorf("mcp_tools_serena.env HOME missing:\n%s", text)
	}
}
