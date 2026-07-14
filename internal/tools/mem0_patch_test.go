package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeMem0Server lays out the uv-tool venv shape under a temp home and
// writes server.py with the given content, returning its path.
func writeFakeMem0Server(t *testing.T, home, content string) string {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "uv", "tools",
		"mem0-mcp-selfhosted", "lib", "python3.11", "site-packages",
		"mem0_mcp_selfhosted")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "server.py")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func pristineMem0Server() string {
	var b strings.Builder
	b.WriteString("# fixture: pinned mem0-mcp-selfhosted server.py excerpt\n")
	for _, p := range mem0EntityFilterPatches {
		b.WriteString(p.orig)
		b.WriteString("\n")
	}
	return b.String()
}

// TestPatchMem0EntityFiltersRewritesAndIsIdempotent covers the contract the
// installer relies on: a pristine pinned server.py gets both call sites
// rewritten to filters= form, and a second run (e.g. panel re-install)
// leaves the file byte-identical instead of erroring or double-patching.
func TestPatchMem0EntityFiltersRewritesAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	path := writeFakeMem0Server(t, home, pristineMem0Server())

	if err := patchMem0EntityFilters(home, func(string) {}); err != nil {
		t.Fatalf("first patch: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range mem0EntityFilterPatches {
		if strings.Contains(string(got), p.orig) {
			t.Errorf("%s: original top-level entity kwargs still present", p.name)
		}
		if !strings.Contains(string(got), p.patched) {
			t.Errorf("%s: patched filters= block missing", p.name)
		}
	}

	if err := patchMem0EntityFilters(home, func(string) {}); err != nil {
		t.Fatalf("second patch (idempotency): %v", err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Error("second run mutated an already-patched file")
	}
}

// TestPatchMem0EntityFiltersFailsOnSourceDrift guards the pin-bump
// escape hatch: if upstream code no longer matches either the original or
// the patched block, the installer must fail loudly instead of silently
// shipping a server with search/get_all still broken.
func TestPatchMem0EntityFiltersFailsOnSourceDrift(t *testing.T) {
	home := t.TempDir()
	writeFakeMem0Server(t, home, "# drifted upstream source, no known blocks\n")

	err := patchMem0EntityFilters(home, func(string) {})
	if err == nil {
		t.Fatal("expected drift error, got nil")
	}
	if !strings.Contains(err.Error(), "no coincide con el pin") {
		t.Errorf("error should name the pin mismatch, got: %v", err)
	}
}

// TestPatchMem0EntityFiltersErrorsWhenVenvMissing: a missing uv install must
// produce an actionable error, not a silent no-op that leaves the tool broken.
func TestPatchMem0EntityFiltersErrorsWhenVenvMissing(t *testing.T) {
	err := patchMem0EntityFilters(t.TempDir(), func(string) {})
	if err == nil {
		t.Fatal("expected error for missing venv, got nil")
	}
}
