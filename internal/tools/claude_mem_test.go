package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallClaudeMemWarnsOnFailure is a regression guard: the old
// uninstallClaudeMem discarded the error from `npx claude-mem@latest
// uninstall` (`_ = cmd.Run()`) and always returned nil, so a genuine
// failure (network unreachable, npx crash, etc.) was silently
// indistinguishable from a clean uninstall. It's intentionally NOT gated
// on which("claude-mem") — npx fetches its own copy to strip stray MCP
// configs/hooks even if the local binary was already removed by hand, so
// this test only exercises failure surfacing, not an install pre-check.
func TestUninstallClaudeMemWarnsOnFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := t.TempDir()

	writeExecutable(t, binDir, "node", "#!/bin/sh\necho v20.11.0\nexit 0\n")
	// npx fails loudly (simulates network-unreachable / npm registry down).
	writeExecutable(t, binDir, "npx", "#!/bin/sh\necho boom >&2\nexit 1\n")
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	var logs []string
	if err := uninstallClaudeMem(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallClaudeMem: %v (should be best-effort, not fatal)", err)
	}
	if !anyContains(logs, "WARN") {
		t.Errorf("expected a WARN log line surfacing the npx failure, got %v", logs)
	}
}

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
