package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallSerenaNoopsCleanlyWhenNotInstalled is a regression guard: the
// old uninstallSerena discarded every error from `uv tool uninstall`
// (`_ = runCombined(...)`) and always returned nil, so a caller could never
// tell "nothing was installed" apart from "the uninstall genuinely failed".
func TestUninstallSerenaNoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin") // no serena, no uv anywhere

	var logs []string
	if err := uninstallSerena(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallSerena: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallSerenaInvokesUVWhenInstalled confirms the uv uninstall
// command is actually attempted (not silently skipped) once serena is
// detected as installed.
func TestUninstallSerenaInvokesUVWhenInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	writeStub(t, binDir, "serena")
	marker := filepath.Join(home, "uv-uninstall-was-called")
	writeStubWithMarker(t, binDir, "uv", marker)

	var logs []string
	if err := uninstallSerena(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallSerena: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("uv tool uninstall was never invoked: %v; logs=%v", err, logs)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("serena was installed — must not report 'nada que desinstalar', got %v", logs)
	}
}

// writeStub writes a no-op executable stub named name into dir.
func writeStub(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeStubWithMarker writes an executable stub named name into dir that
// touches marker whenever it is invoked with "uninstall" as an argument.
func writeStubWithMarker(t *testing.T, dir, name, marker string) {
	t.Helper()
	script := "#!/bin/sh\nfor a in \"$@\"; do if [ \"$a\" = uninstall ]; then touch \"" + marker + "\"; fi; done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
