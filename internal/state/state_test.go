package state

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// withDataDir temporarily redirects config.DataDir() to a fresh temp dir so the
// tests can exercise Load/Save without touching the real $MCP_TOOLS_DATA.
func withDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MCP_TOOLS_DATA", dir)
	return dir
}

// being silently accepted and producing a degraded State.
func TestLoadRejectsFutureSchema(t *testing.T) {
	dir := withDataDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	bogus := `{"version": 999, "selected": ["a"]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(bogus), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for future schema version, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "schema v999") {
		t.Fatalf("error %q does not mention the offending schema version", msg)
	}
	if !strings.Contains(msg, "soporta v1") {
		t.Fatalf("error %q does not mention the supported schema version", msg)
	}
}

// TestStateRoundTrip verifies Save then Load returns a State with identical
// Selected and Versions.
func TestStateRoundTrip(t *testing.T) {
	_ = withDataDir(t)

	in := State{
		Version:  SchemaVersion,
		Selected: []string{"a", "b", "c"},
		Versions: map[string]string{"a": "1.0", "b": "2.0"},
	}
	if err := in.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := len(out.Selected), len(in.Selected); got != want {
		t.Fatalf("Selected len = %d, want %d", got, want)
	}
	for i, k := range in.Selected {
		if out.Selected[i] != k {
			t.Fatalf("Selected[%d] = %q, want %q", i, out.Selected[i], k)
		}
	}
	if got, want := len(out.Versions), len(in.Versions); got != want {
		t.Fatalf("Versions len = %d, want %d", got, want)
	}
	for k, v := range in.Versions {
		if out.Versions[k] != v {
			t.Fatalf("Versions[%q] = %q, want %q", k, out.Versions[k], v)
		}
	}
}

// TestSaveConcurrentDoesNotError is a regression guard: Save() used to
// write to a FIXED tempfile path (path+".tmp") with no synchronization.
// Install/uninstall/configure all persist state from background jobs
// (internal/web/job.go), so two concurrent operations -- two browser
// tabs, or two devices on the LAN since MCP_TOOLS_BIND defaults to
// 0.0.0.0 -- could race: both goroutines' WriteFile+Rename sequences
// interleaved on the same tmp name, and the loser's os.Rename found its
// source already moved away by the winner and failed with ENOENT --
// surfacing a spurious "save state" error on an operation whose actual
// install/uninstall work had already succeeded. Reproduced directly
// before the fix: 50 concurrent Save() calls reliably produced several
// such rename errors.
func TestSaveConcurrentDoesNotError(t *testing.T) {
	_ = withDataDir(t)

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := State{Selected: []string{"tool"}, Versions: map[string]string{"tool": "v1"}}
			if err := s.Save(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Save() under concurrency: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() after concurrent saves: %v", err)
	}
	if len(got.Selected) != 1 || got.Selected[0] != "tool" {
		t.Errorf("state.json unexpected after concurrent Save(): got %+v", got)
	}
}
