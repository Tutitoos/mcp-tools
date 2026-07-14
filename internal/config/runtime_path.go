package config

import (
	"os"
	"path/filepath"
	"strings"
)

// EnsureRuntimePath prepends $HOME/.local/bin and $HOME/.cargo/bin to
// $PATH if they aren't already present, and exports $HOME itself if it
// isn't already set. Idempotent. Must be called before any
// exec.LookPath / exec.Command / os.Environ read that needs to see
// host-tool installs (codebase-memory-mcp, serena, tokensave,
// etc.) — systemd system-mode services inherit a minimal PATH
// (/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin) AND no
// $HOME at all, so the mcp-tools-web daemon can't find tools it already
// installed into the user's home directory, AND any child installer
// script that references "$HOME" itself (e.g. omp's install.sh doing
// INSTALL_DIR="${PI_INSTALL_DIR:-$HOME/.local/bin}") silently writes to
// the wrong place ("/.local/bin" when $HOME is empty) instead of
// failing loudly.
func EnsureRuntimePath() error {
	home, err := HomeDir()
	if err != nil {
		return err
	}

	if os.Getenv("HOME") == "" {
		if err := os.Setenv("HOME", home); err != nil {
			return err
		}
	}

	want := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".cargo", "bin"),
	}

	rawCurrent := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	current := make([]string, 0, len(rawCurrent))
	seen := make(map[string]bool, len(rawCurrent))
	for _, p := range rawCurrent {
		if p == "" {
			continue
		}
		current = append(current, p)
		seen[p] = true
	}

	missing := make([]string, 0, len(want))
	for _, w := range want {
		if !seen[w] {
			missing = append(missing, w)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	newPath := append(missing, current...)
	return os.Setenv("PATH", strings.Join(newPath, string(os.PathListSeparator)))
}
