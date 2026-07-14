package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The documented contract is `MCP_TOOLS_BIND=0.0.0.0` in the repo .env
// (or the process env) — AUDIT-2026-07-14 found the CLI read a variable
// that was never documented (MCP_TOOLS_WEB_BIND) and ignored .env
// entirely, so a user following the docs stayed on loopback.
func TestBindFromEnvReadsRepoEnvFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MCP_TOOLS_BIND=0.0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_TOOLS_ROOT", dir)
	t.Setenv("MCP_TOOLS_BIND", "")
	os.Unsetenv("MCP_TOOLS_BIND")

	if got := BindFromEnv(); got != "0.0.0.0" {
		t.Fatalf("BindFromEnv() = %q, want 0.0.0.0 (from .env)", got)
	}
}

func TestBindFromEnvProcessEnvWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MCP_TOOLS_BIND=0.0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_TOOLS_ROOT", dir)
	t.Setenv("MCP_TOOLS_BIND", "192.168.1.10")

	if got := BindFromEnv(); got != "192.168.1.10" {
		t.Fatalf("BindFromEnv() = %q, want process env to win", got)
	}
}

func TestBindFromEnvEmptyWhenUnset(t *testing.T) {
	dir := t.TempDir() // no .env at all
	t.Setenv("MCP_TOOLS_ROOT", dir)
	os.Unsetenv("MCP_TOOLS_BIND")

	if got := BindFromEnv(); got != "" {
		t.Fatalf("BindFromEnv() = %q, want empty", got)
	}
}
