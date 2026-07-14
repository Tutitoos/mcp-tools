package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// TestServersCoversMCPTools verifies that every key in the canonical MCP tools
// map (mcpServers) appears in the Servers() output when selected, with a
// non-empty Wrapper. Adding a new MCP server requires updating both the map
// and this fixture so the registry can't fall out of sync silently.
// See H29 (review ronda 2, 2026-07-08).
func TestServersCoversMCPTools(t *testing.T) {
	wantKeys := []string{"codebase-memory", "mem0", "headroom", "serena", "mongodb", "redis", "docker-mcp-toolkit", "sentry"}
	st := state.State{Selected: wantKeys}
	got := Servers(st)
	if len(got) != len(wantKeys) {
		t.Fatalf("Servers returned %d entries, want %d (one per MCP tool)", len(got), len(wantKeys))
	}
	seen := map[string]bool{}
	for _, s := range got {
		if s.Wrapper == "" {
			t.Errorf("server %q has empty Wrapper", s.Name)
		}
		if s.Name == "" {
			t.Errorf("server has empty Name; check mcpServers map")
		}
		seen[s.Name] = true
	}
	for _, k := range wantKeys {
		// Walk mcpServers for the expected resulting Name.
		spec, ok := mcpServers[k]
		if !ok {
			t.Errorf("mcpServers map is missing key %q — add it", k)
			continue
		}
		if !seen[spec.Name] {
			t.Errorf("server %q (from key %q) not present in Servers() output", spec.Name, k)
		}
	}
}

// TestServersSkipsNonMCPTools ensures tools that are NOT in mcpServers
// (system tools like nvidia-toolkit / rtk / tokensave) are silently dropped
// instead of surfacing as MCP servers.
func TestServersSkipsNonMCPTools(t *testing.T) {
	st := state.State{Selected: []string{"nvidia-toolkit", "rtk", "tokensave", "qdrant", "ollama"}}
	got := Servers(st)
	if len(got) != 0 {
		t.Fatalf("Servers returned %d entries for system-only selection, want 0", len(got))
	}
}

func TestServersResolvesConfiguredEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("MDB_MCP_CONNECTION_STRING=from-file\nREDIS_HOST=redis.internal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_TOOLS_ROOT", root)
	t.Setenv("MDB_MCP_CONNECTION_STRING", "from-process")

	servers := Servers(state.State{Selected: []string{"mongodb", "redis", "docker-mcp-toolkit"}})
	if got := servers[0].Env["MDB_MCP_CONNECTION_STRING"]; got != "from-process" {
		t.Fatalf("MongoDB env = %q, want process env precedence", got)
	}
	if got := servers[1].Env["REDIS_HOST"]; got != "redis.internal" {
		t.Fatalf("Redis env = %q, want repo .env value", got)
	}
	if got := servers[2].Env["DOCKER_MCP_IN_CONTAINER"]; got != "1" {
		t.Fatalf("Docker env = %q, want headless gateway flag", got)
	}
	env := serverEnvironment(servers[0], "/home/test")
	if env["HOME"] != "/home/test" || env["MDB_MCP_CONNECTION_STRING"] != "from-process" {
		t.Fatalf("rendered environment lost HOME or server env: %v", env)
	}
}
