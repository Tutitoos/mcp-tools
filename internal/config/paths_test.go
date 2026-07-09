package config

import "testing"

// TestHomeDirFallsBackWithoutEnv is a regression guard for the
// mcp-tools-web systemd unit bug: in system mode (no explicit User=),
// systemd does NOT populate $HOME by default, so os.UserHomeDir() alone
// fails with "$HOME is not defined" even though the process has a
// perfectly valid home directory. HomeDir must still resolve it via
// os/user.Current().
func TestHomeDirFallsBackWithoutEnv(t *testing.T) {
	t.Setenv("HOME", "")

	h, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() with $HOME unset: %v", err)
	}
	if h == "" {
		t.Fatal("HomeDir() with $HOME unset returned empty string")
	}
}

// TestHomeDirPrefersEnv confirms an explicit $HOME override still wins
// over the os/user fallback.
func TestHomeDirPrefersEnv(t *testing.T) {
	t.Setenv("HOME", "/custom/home")

	h, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir(): %v", err)
	}
	if h != "/custom/home" {
		t.Fatalf("HomeDir() = %q, want %q", h, "/custom/home")
	}
}

// TestRepoRootFallsBackWithoutEnv guards the silent-degradation variant of
// the same bug: RepoRoot must not silently collapse to a relative
// "mcp-tools" path when $HOME is unset and MCP_TOOLS_ROOT isn't set either.
func TestRepoRootFallsBackWithoutEnv(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("MCP_TOOLS_ROOT", "")

	root := RepoRoot()
	if root == "" || root == "mcp-tools" {
		t.Fatalf("RepoRoot() with $HOME unset = %q, want an absolute path", root)
	}
}
