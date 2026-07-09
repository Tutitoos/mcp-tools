package orchestrator

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// TestConfigureAppliesSelectionDiff reproduces B1: Configure used to diff
// `prev.Selected` against itself (`newSelected := prev.Selected`), so
// toAdd/toRemove were always empty and every POST /api/configure was a
// no-op logged as "── sin cambios" regardless of the requested selection.
//
// Run with dry=true against the REAL tool registry: every tool's
// Install/Uninstall closure honors `dry` and performs no real work (see
// e.g. internal/tools/tokensave.go), so this exercises the full
// diff-and-apply path without touching the host or state.json.
func TestConfigureAppliesSelectionDiff(t *testing.T) {
	prev := state.State{Selected: []string{"tokensave"}}
	next := []string{"tokensave", "headroom"}

	var lines []string
	log := func(l string) { lines = append(lines, l) }

	got, err := Configure(context.Background(), prev, next, true, log)
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	// dry=true returns prev unchanged — state.json is never touched.
	if !reflect.DeepEqual(got.Selected, prev.Selected) {
		t.Errorf("dry Configure mutated returned Selected: got %v, want unchanged %v", got.Selected, prev.Selected)
	}

	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "sin cambios") {
		t.Errorf("Configure(prev=[tokensave], next=[tokensave,headroom]) logged \"sin cambios\" despite a real selection diff (B1 regression). Log:\n%s", joined)
	}
	if !strings.Contains(joined, "── install ") {
		t.Errorf("Configure did not run an install step for the added tool (toAdd still empty — B1 regression). Log:\n%s", joined)
	}
}

// TestConfigureRemovesDeselectedTool is the mirror case: dropping a tool
// from the selection must drive an uninstall, not silently no-op.
func TestConfigureRemovesDeselectedTool(t *testing.T) {
	prev := state.State{Selected: []string{"tokensave", "headroom"}}
	next := []string{"tokensave"}

	var lines []string
	log := func(l string) { lines = append(lines, l) }

	if _, err := Configure(context.Background(), prev, next, true, log); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "sin cambios") {
		t.Errorf("Configure logged \"sin cambios\" despite removing a tool (B1 regression). Log:\n%s", joined)
	}
	if !strings.Contains(joined, "── uninstall ") {
		t.Errorf("Configure did not run an uninstall step for the removed tool (toRemove still empty — B1 regression). Log:\n%s", joined)
	}
}

// TestConfigureNoChangeIsNoop verifies the genuinely-unchanged case is
// still correctly reported (this is the one path where "sin cambios" is
// the RIGHT answer).
func TestConfigureNoChangeIsNoop(t *testing.T) {
	t.Setenv("MCP_TOOLS_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	prev := state.State{Selected: []string{"tokensave"}}
	var lines []string
	log := func(l string) { lines = append(lines, l) }
	got, err := Configure(context.Background(), prev, []string{"tokensave"}, false, log)
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if !reflect.DeepEqual(got.Selected, prev.Selected) {
		t.Errorf("no-op Configure changed Selected: got %v", got.Selected)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "sin cambios") {
		t.Errorf("expected \"sin cambios\" log for an identical selection, got:\n%s", joined)
	}
}

// TestConfigureRejectsUnknownKey verifies an unregistered key is rejected
// before any install/uninstall runs, per the plan's validation step.
func TestConfigureRejectsUnknownKey(t *testing.T) {
	prev := state.State{Selected: nil}
	_, err := Configure(context.Background(), prev, []string{"not-a-real-tool"}, true, func(string) {})
	if err == nil {
		t.Fatal("expected an error for an unknown tool key, got nil")
	}
	if !strings.Contains(err.Error(), "not-a-real-tool") {
		t.Errorf("error %v does not mention the offending key", err)
	}
}
