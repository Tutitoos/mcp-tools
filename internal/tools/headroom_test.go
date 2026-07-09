package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallHeadroomNoopsCleanlyWhenNotInstalled mirrors the serena
// regression guard: the old uninstallHeadroom discarded every error from
// `uv tool uninstall` and always returned nil.
func TestUninstallHeadroomNoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	var logs []string
	if err := uninstallHeadroom(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallHeadroom: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallHeadroomInvokesUVWhenInstalled confirms the uv uninstall
// command actually runs once headroom is detected as installed.
func TestUninstallHeadroomInvokesUVWhenInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	writeStub(t, binDir, "headroom")
	marker := filepath.Join(home, "uv-uninstall-was-called")
	writeStubWithMarker(t, binDir, "uv", marker)

	var logs []string
	if err := uninstallHeadroom(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallHeadroom: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("uv tool uninstall was never invoked: %v; logs=%v", err, logs)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("headroom was installed — must not report 'nada que desinstalar', got %v", logs)
	}
}
