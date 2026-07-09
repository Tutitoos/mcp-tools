package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureRuntimePathPrependsHomeBins is a regression guard for the
// mcp-tools-web systemd unit bug: the daemon inherits a minimal PATH with
// no $HOME/.local/bin or $HOME/.cargo/bin, so host-tool installs are
// invisible to exec.LookPath. EnsureRuntimePath must prepend both, in
// that order.
func TestEnsureRuntimePathPrependsHomeBins(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := EnsureRuntimePath(); err != nil {
		t.Fatalf("EnsureRuntimePath: %v", err)
	}

	got := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	want := []string{"/tmp/fakehome/.local/bin", "/tmp/fakehome/.cargo/bin"}
	if len(got) < 4 {
		t.Fatalf("PATH = %q, want at least 4 entries", os.Getenv("PATH"))
	}
	if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("PATH prefix = %v, want %v", got[:2], want)
	}
	if got[2] != "/usr/bin" || got[3] != "/bin" {
		t.Fatalf("PATH suffix = %v, want [/usr/bin /bin]", got[2:4])
	}
}

// TestEnsureRuntimePathIdempotent confirms a second call does not
// duplicate entries already present (whether pre-existing or added by a
// prior call).
func TestEnsureRuntimePathIdempotent(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("PATH", "/tmp/fakehome/.local/bin:/usr/bin")

	if err := EnsureRuntimePath(); err != nil {
		t.Fatalf("EnsureRuntimePath (1st): %v", err)
	}
	if err := EnsureRuntimePath(); err != nil {
		t.Fatalf("EnsureRuntimePath (2nd): %v", err)
	}

	got := os.Getenv("PATH")
	if n := strings.Count(got, "/tmp/fakehome/.local/bin"); n != 1 {
		t.Fatalf("count(.local/bin) = %d, want 1 (PATH=%q)", n, got)
	}
	if n := strings.Count(got, "/tmp/fakehome/.cargo/bin"); n != 1 {
		t.Fatalf("count(.cargo/bin) = %d, want 1 (PATH=%q)", n, got)
	}
	if !strings.Contains(got, "/usr/bin") {
		t.Fatalf("PATH = %q, want /usr/bin still present", got)
	}
}

// TestEnsureRuntimePathEmptyStart guards the empty-$PATH edge case: no
// leading/trailing/doubled separators, which some shells parse as "."
// (a security-relevant footgun).
func TestEnsureRuntimePathEmptyStart(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("PATH", "")

	if err := EnsureRuntimePath(); err != nil {
		t.Fatalf("EnsureRuntimePath: %v", err)
	}

	got := os.Getenv("PATH")
	want := "/tmp/fakehome/.local/bin:/tmp/fakehome/.cargo/bin"
	if got != want {
		t.Fatalf("PATH = %q, want %q", got, want)
	}
	if strings.Contains(got, "::") || strings.HasPrefix(got, ":") || strings.HasSuffix(got, ":") {
		t.Fatalf("PATH = %q contains an empty entry (security footgun)", got)
	}
}

// TestEnsureRuntimePathMakesLookPathFindHomeBinBinary is the mechanical
// reproducer for symptom 2 ("codegraph binario no encontrado tras
// install.sh"): a binary installed to $HOME/.local/bin must become
// resolvable via exec.LookPath once EnsureRuntimePath runs.
func TestEnsureRuntimePathMakesLookPathFindHomeBinBinary(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(binDir, "codegraph-fake")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := EnsureRuntimePath(); err != nil {
		t.Fatalf("EnsureRuntimePath: %v", err)
	}

	got, err := exec.LookPath("codegraph-fake")
	if err != nil {
		t.Fatalf("LookPath after EnsureRuntimePath: %v (PATH=%q)", err, os.Getenv("PATH"))
	}
	if got != stub {
		t.Fatalf("LookPath = %q, want %q", got, stub)
	}
}
