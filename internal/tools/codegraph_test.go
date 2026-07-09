package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallCodegraphNoopsCleanlyWhenNotInstalled is a regression guard:
// the old uninstallCodegraph returned nil silently (no log at all) when
// codegraph wasn't on PATH, and separately discarded the actual "codegraph
// uninstall --yes" error when it was — both paths reported bare success.
func TestUninstallCodegraphNoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	var logs []string
	if err := uninstallCodegraph(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallCodegraph: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallCodegraphWarnsOnFailure confirms a genuine uninstall failure
// is surfaced to the caller instead of being silently swallowed.
func TestUninstallCodegraphWarnsOnFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	if err := os.WriteFile(filepath.Join(binDir, "codegraph"), []byte("#!/bin/sh\necho boom >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var logs []string
	if err := uninstallCodegraph(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallCodegraph: %v", err)
	}
	if !anyContains(logs, "WARN") {
		t.Errorf("expected a WARN log line surfacing the uninstall failure, got %v", logs)
	}
}
