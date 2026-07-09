package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUninstallRTKNoopsCleanlyWhenNotInstalled is a regression guard: the
// old uninstallRTK always returned nil after silently swallowing every
// cargo error (`_ = exec.Command(...).Run()`), so a caller could never
// tell "nothing was installed" apart from "the uninstall genuinely
// failed". It must now say so explicitly.
func TestUninstallRTKNoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin") // no cargo, no rtk anywhere on PATH

	var logs []string
	if err := uninstallRTK(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallRTK: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallRTKRemovesHookEvenWithoutBinary confirms the omp hook is
// still cleaned up (and the "nada que desinstalar" short-circuit is NOT
// taken) when the rtk binary itself is absent but a stale hook file
// remains on disk.
func TestUninstallRTKRemovesHookEvenWithoutBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	hookDir := filepath.Join(home, ".omp", "extensions")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(hookDir, "rtk.ts")
	if err := os.WriteFile(hook, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logs []string
	if err := uninstallRTK(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallRTK: %v", err)
	}
	if _, err := os.Stat(hook); !os.IsNotExist(err) {
		t.Errorf("hook file should have been removed, stat err = %v", err)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("hook existed on disk — must not report 'nada que desinstalar', got %v", logs)
	}
}

func anyContains(lines []string, substr string) bool {
	for _, l := range lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}
