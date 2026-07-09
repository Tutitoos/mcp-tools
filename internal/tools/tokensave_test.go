package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUninstallTokensaveNoopsCleanlyWhenNotInstalled is a regression guard:
// the old uninstallTokensave discarded every error from `tokensave
// uninstall` and `cargo uninstall tokensave` (`_ = cmd.Run()`) and always
// returned nil, so a caller could never tell "nothing was installed" apart
// from "the uninstall genuinely failed". It must now say so explicitly.
func TestUninstallTokensaveNoopsCleanlyWhenNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin") // no cargo, no tokensave anywhere

	var logs []string
	if err := uninstallTokensave(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallTokensave: %v", err)
	}
	if !anyContains(logs, "nada que desinstalar") {
		t.Errorf("expected a 'nada que desinstalar' log line, got %v", logs)
	}
}

// TestUninstallTokensaveRunsClientCleanupViaDirectPath guards a bug found
// while fixing the noop-detection above: the installed check consulted
// which("tokensave") (a PATH lookup) OR the direct ~/.cargo/bin/tokensave
// path, but the actual `tokensave uninstall` invocation (which strips MCP
// client configs/hooks/CLAUDE.md rules) only fired when `bin` came from
// `which`. A binary present on disk but not yet on $PATH (e.g. a fresh
// install before EnsureRuntimePath re-runs) would skip client cleanup
// entirely and jump straight to `cargo uninstall`, leaving every
// ~/.claude.json / opencode.json / omp mcp.json entry orphaned — exactly
// the "configs colgando" outcome the function's own comment warns against.
func TestUninstallTokensaveRunsClientCleanupViaDirectPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin") // deliberately NOT on PATH

	cargoBinDir := filepath.Join(home, ".cargo", "bin")
	if err := os.MkdirAll(cargoBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "uninstall-was-called")
	stub := "#!/bin/sh\nif [ \"$1\" = uninstall ]; then touch \"" + marker + "\"; fi\nexit 0\n"
	stubPath := filepath.Join(cargoBinDir, "tokensave")
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}

	var logs []string
	if err := uninstallTokensave(false, func(s string) { logs = append(logs, s) }); err != nil {
		t.Fatalf("uninstallTokensave: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("tokensave uninstall was never invoked via the direct path (client configs left orphaned): %v; logs=%v", err, logs)
	}
	if anyContains(logs, "nada que desinstalar") {
		t.Errorf("binary existed on disk — must not report 'nada que desinstalar', got %v", logs)
	}
}
