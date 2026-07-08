package tools

import (
	"path/filepath"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// TestOllamaComposeFilesRelative enforces the contract that every path
// returned by OllamaComposeFiles is relative to config.RepoRoot(). Mixes of
// absolute + relative paths used to slip through because every caller pinned
// cmd.Dir = RepoRoot(); a future caller that forgets cmd.Dir would otherwise
// silently resolve the relative entry against the wrong cwd. See REVIEW-rd2 (H25).
func TestOllamaComposeFilesRelative(t *testing.T) {
	cases := []struct {
		name     string
		selected []string
	}{
		{"no selection", nil},
		{"only qdrant", []string{"qdrant"}},
		{"with nvidia-toolkit", []string{"qdrant", "ollama", "nvidia-toolkit"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := state.State{Selected: tc.selected}
			files := OllamaComposeFiles(st)
			if len(files) == 0 {
				t.Fatalf("OllamaComposeFiles returned no files")
			}
			for _, f := range files {
				if filepath.IsAbs(f) {
					t.Errorf("path %q is absolute; OllamaComposeFiles must return relative paths", f)
				}
			}
		})
	}
}
