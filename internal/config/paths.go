package config

import (
	"os"
	"path/filepath"
)

// homeDir returns $HOME (falls back to os.UserHomeDir if $HOME unset).
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, err := os.UserHomeDir()
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

// WrapperPath returns the absolute path to the wrapper for the given MCP name.
// e.g. WrapperPath("mem0") -> $HOME/.local/bin/mcp-tools-mem0-docker
func WrapperPath(name string) string {
	return filepath.Join(WrapperDir(), "mcp-tools-"+name+"-docker")
}
