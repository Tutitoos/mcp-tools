package orchestrator

import (
	"reflect"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// TestPartitionByStdio covers the rules from the partitionByStdio doc
// comment: sudo first, tui in the middle, inter last, unknown → tui, and
// the order-within-slice preservation contract.
func TestPartitionByStdio(t *testing.T) {
	cases := []struct {
		name      string
		keys      []string
		wantSudo  []string
		wantTui   []string
		wantInter []string
	}{
		{
			name:      "empty input",
			keys:      nil,
			wantSudo:  nil,
			wantTui:   nil,
			wantInter: nil,
		},
		{
			name:      "all silent docker tools",
			keys:      []string{"qdrant", "ollama"},
			wantSudo:  nil,
			wantTui:   []string{"qdrant", "ollama"},
			wantInter: nil,
		},
		{
			name:      "sudo tool first",
			keys:      []string{"nvidia-toolkit", "qdrant"},
			wantSudo:  []string{"nvidia-toolkit"},
			wantTui:   []string{"qdrant"},
			wantInter: nil,
		},
		{
			name:      "interactive tool last",
			keys:      []string{"claude-mem", "ollama"},
			wantSudo:  nil,
			wantTui:   []string{"ollama"},
			wantInter: []string{"claude-mem"},
		},
		{
			name:      "unknown key falls into tui bucket",
			keys:      []string{"unknown-tool"},
			wantSudo:  nil,
			wantTui:   []string{"unknown-tool"},
			wantInter: nil,
		},
		{
			name:      "preserves input order within each bucket",
			keys:      []string{"qdrant", "nvidia-toolkit", "ollama", "claude-mem", "codebase-memory"},
			wantSudo:  []string{"nvidia-toolkit"},
			wantTui:   []string{"qdrant", "ollama", "codebase-memory"},
			wantInter: []string{"claude-mem"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sudo, tui, inter := PartitionByStdio(tc.keys)
			if !reflect.DeepEqual(sudo, tc.wantSudo) {
				t.Errorf("sudo = %v, want %v", sudo, tc.wantSudo)
			}
			if !reflect.DeepEqual(tui, tc.wantTui) {
				t.Errorf("tui = %v, want %v", tui, tc.wantTui)
			}
			if !reflect.DeepEqual(inter, tc.wantInter) {
				t.Errorf("inter = %v, want %v", inter, tc.wantInter)
			}
		})
	}
}

// TestPickToolFn verifies the verb dispatch table used by runInlineTools.
func TestPickToolFn(t *testing.T) {
	t1, err := tools.Get("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if fn := pickToolFn(t1, "install"); fn == nil {
		t.Error("install verb should return a closure for ollama")
	}
	if fn := pickToolFn(t1, "upgrade"); fn == nil {
		t.Error("upgrade verb should return a closure for ollama")
	}
	if fn := pickToolFn(t1, "uninstall"); fn == nil {
		t.Error("uninstall verb should return a closure for ollama")
	}
	if fn := pickToolFn(t1, "bogus"); fn != nil {
		t.Error("bogus verb should return nil")
	}
}

// TestDiffKeys preserves `a` order and excludes items in `b`.
func TestDiffKeys(t *testing.T) {
	a := []string{"qdrant", "ollama", "claude"}
	b := []string{"ollama"}
	got := diffKeys(a, b)
	want := []string{"qdrant", "claude"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diffKeys(%v, %v) = %v, want %v", a, b, got, want)
	}
}
