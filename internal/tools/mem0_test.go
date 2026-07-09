package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallMem0NoopsCleanlyWhenNotInstalled is a regression guard: the
// old uninstallMem0 ran `uv tool uninstall mem0-mcp-selfhosted` unconditionally
// whenever uv was present (regardless of whether mem0 itself was ever
// installed) and discarded its error, so "nothing was installed" and "the
// uninstall genuinely failed" were indistinguishable.
func TestUninstallMem0NoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	var logs []string
	if err := uninstallMem0(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallMem0: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallMem0InvokesUVWhenInstalled confirms the uv uninstall command
// actually runs once mem0 is detected as installed via its binary on PATH.
func TestUninstallMem0InvokesUVWhenInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	writeStub(t, binDir, "mem0-mcp-selfhosted")
	marker := filepath.Join(home, "uv-uninstall-was-called")
	writeStubWithMarker(t, binDir, "uv", marker)

	var logs []string
	if err := uninstallMem0(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallMem0: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("uv tool uninstall was never invoked: %v; logs=%v", err, logs)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("mem0 was installed — must not report 'nada que desinstalar', got %v", logs)
	}
}

// TestUninstallMem0DetectsViaLauncherFallback confirms the launcher-file
// fallback (for a host where the binary isn't on PATH yet) still counts as
// "installed" rather than short-circuiting to the noop path.
func TestUninstallMem0DetectsViaLauncherFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin") // no mem0-mcp-selfhosted, no uv

	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localBin, "mem0-launcher"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var logs []string
	if err := uninstallMem0(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallMem0: %v", err)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("launcher existed on disk — must not report 'nada que desinstalar', got %v", logs)
	}
	if _, err := os.Stat(filepath.Join(localBin, "mem0-launcher")); !os.IsNotExist(err) {
		t.Errorf("launcher should have been removed, stat err = %v", err)
	}
}
