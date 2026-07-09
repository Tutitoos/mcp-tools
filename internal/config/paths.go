package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// HomeDir returns the invoking user's home directory. It prefers $HOME (so
// an explicit override wins), but falls back to os/user.Current().HomeDir —
// a direct /etc/passwd lookup — when $HOME isn't set in the process
// environment. This matters for the mcp-tools-web systemd unit: in system
// mode (no explicit User=), systemd does NOT populate $HOME by default, so
// os.UserHomeDir() alone fails with "$HOME is not defined" even though the
// process has a perfectly valid home directory (e.g. root's /root).
func HomeDir() (string, error) {
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver el home del usuario (ni $HOME ni os/user): %w", err)
	}
	if u.HomeDir == "" {
		return "", errors.New("home de usuario vacío — establece $HOME antes de correr install")
	}
	return u.HomeDir, nil
}

// homeDir is the no-error convenience wrapper used by the path helpers
// below; they have historically returned "" on failure rather than
// propagating an error, and callers such as RepoRoot/DataDir predate
// HomeDir's explicit error. Keeping that contract here.
func homeDir() string {
	h, err := HomeDir()
	if err != nil {
		return ""
	}
	return h
}

// RepoRoot returns MCP_TOOLS_ROOT or $HOME/mcp-tools.
func RepoRoot() string {
	if r := os.Getenv("MCP_TOOLS_ROOT"); r != "" {
		return r
	}
	return filepath.Join(homeDir(), "mcp-tools")
}

// DataDir returns MCP_TOOLS_DATA (from env), or reads from .env, or defaults to $HOME/mcp-tools-data.
func DataDir() string {
	if d := os.Getenv("MCP_TOOLS_DATA"); d != "" {
		return d
	}
	if env, err := LoadEnv(EnvFile()); err == nil {
		if v, ok := env["MCP_TOOLS_DATA"]; ok && v != "" {
			return v
		}
	}
	return filepath.Join(homeDir(), "mcp-tools-data")
}

// EnvFile returns <RepoRoot>/.env.
func EnvFile() string { return filepath.Join(RepoRoot(), ".env") }

// EnvMem0File returns <RepoRoot>/.env.mem0.
func EnvMem0File() string { return filepath.Join(RepoRoot(), ".env.mem0") }

// ComposeFile returns <RepoRoot>/dockers/compose.yaml.
func ComposeFile() string { return filepath.Join(RepoRoot(), "dockers/compose.yaml") }

// WrapperDir returns $HOME/.local/bin.
func WrapperDir() string { return filepath.Join(homeDir(), ".local/bin") }

// PluginsDir returns <RepoRoot>/plugins, the workspace directory holding
// locally-developed OMP plugin packages (one subdir per package with its
// own package.json).
func PluginsDir() string { return filepath.Join(RepoRoot(), "plugins") }

// OmpPluginsLockfile returns $HOME/.omp/plugins/omp-plugins.lock.json — the
// runtime state file OMP writes when a plugin is linked/installed/enabled.
func OmpPluginsLockfile() string {
	return filepath.Join(homeDir(), ".omp", "plugins", "omp-plugins.lock.json")
}
