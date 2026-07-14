package mcp

import (
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// TestServersCoversMCPTools verifies that every key in the canonical MCP tools
// map (mcpServers) appears in the Servers() output when selected, with a
// non-empty Wrapper. Adding a new MCP server requires updating both the map
// and this fixture so the registry can't fall out of sync silently.
// See H29 (review ronda 2, 2026-07-08).
func TestServersCoversMCPTools(t *testing.T) {
	wantKeys := []string{"codebase-memory", "mem0", "headroom", "serena"}
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
