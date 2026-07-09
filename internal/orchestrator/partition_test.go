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

// TestValidatePartitionOrderCatchesLateDependency guards the latent
// architectural risk found in this session's bug hunt: PartitionByStdio
// only preserves order WITHIN a bucket, not across the fixed
// sudo→tui→inter execution sequence. mem0 genuinely declares
// Deps: []string{"qdrant", "ollama"} (see mem0.go) — here we construct an
// artificial bucket split (not what PartitionByStdio produces today,
// since all three currently land in tui together) with qdrant scheduled
// AFTER mem0, to prove the guardrail catches that shape if it ever
// arises from a future tool's Deploy/Interactive combination.
func TestValidatePartitionOrderCatchesLateDependency(t *testing.T) {
	keys := []string{"qdrant", "mem0"}
	err := validatePartitionOrder(keys, nil, []string{"mem0"}, []string{"qdrant"})
	if err == nil {
		t.Fatal("expected an error when a dependency is scheduled to run after its dependent")
	}
}

// TestValidatePartitionOrderAllowsCurrentRegistry confirms today's only
// Deps user (mem0 → qdrant, ollama) does not trip the guardrail: all
// three land in the tui bucket together via the real PartitionByStdio.
func TestValidatePartitionOrderAllowsCurrentRegistry(t *testing.T) {
	keys := []string{"qdrant", "ollama", "mem0"}
	sudo, tui, inter := PartitionByStdio(keys)
	if err := validatePartitionOrder(keys, sudo, tui, inter); err != nil {
		t.Fatalf("current registry should not trip the guardrail: %v", err)
	}
}

// TestValidatePartitionOrderIgnoresDepsOutsideBatch confirms a dependency
// that isn't part of the current key batch (e.g. already installed in a
// prior Configure call) is not flagged — only deps scheduled WITHIN the
// same batch matter.
func TestValidatePartitionOrderIgnoresDepsOutsideBatch(t *testing.T) {
	keys := []string{"mem0"}
	if err := validatePartitionOrder(keys, nil, []string{"mem0"}, nil); err != nil {
		t.Fatalf("dependency outside the batch should be ignored: %v", err)
	}
}
