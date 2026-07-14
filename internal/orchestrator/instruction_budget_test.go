package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// estimateTokens is a conservative ~4 chars/token approximation for mixed
// Markdown/code. Good enough to catch regressions; not a tokenizer.
func estimateTokens(b []byte) int { return (len(b) + 3) / 4 }

var frontmatterRE = regexp.MustCompile(`(?s)^---\n.*?\n---\n`)

// budgetCategory groups instruction files under a shared token ceiling.
// Categories are deliberately separate (decisión del refactor de
// instruction-sources del 2026-07-13): counting on-demand skill
// bodies as always-on would make the metric meaningless, and shrinking
// RULES.md by silently inflating the skills must show up as a failure of
// the total-corpus budget, not as a win.
type budgetCategory struct {
	name      string
	maxTokens int
	parts     func(root string) (map[string][]byte, error)
}

func skillPaths(root string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("skills/*/SKILL.md: %v (matches=%d)", err, len(matches))
	}
	return matches, nil
}

func readAll(root string, rels ...string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(rels))
	for _, rel := range rels {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return nil, err
		}
		out[rel] = b
	}
	return out, nil
}

// TestInstructionBudget is the anti-regrowth gate: it fails when the
// instruction corpus crosses per-category ceilings, and always logs the
// per-file breakdown so the report is useful even when green.
//
// Ceilings = post-refactor measurements (2026-07-13: always-on 4,356 tok,
// on-demand 8,674 tok, total 13,725 tok) + ~10-15% headroom. Raising a
// ceiling is allowed but must be a deliberate, reviewed change with a
// justification in the commit message — that is the entire point.
func TestInstructionBudget(t *testing.T) {
	root := repoRootForTest(t)

	categories := []budgetCategory{
		{
			// What a Claude Code session carries permanently: the global
			// rules + the repo file + every skill-listing description.
			// (AGENTS.md-clients carry less; CLAUDE.md is the worst case
			// because it also holds the rtk block.)
			name:      "always-on (RULES + CLAUDE + skill frontmatters)",
			maxTokens: 5000,
			parts: func(root string) (map[string][]byte, error) {
				out, err := readAll(root, "RULES.md", "CLAUDE.md")
				if err != nil {
					return nil, err
				}
				skills, err := skillPaths(root)
				if err != nil {
					return nil, err
				}
				for _, p := range skills {
					b, err := os.ReadFile(p)
					if err != nil {
						return nil, err
					}
					fm := frontmatterRE.Find(b)
					if len(fm) == 0 {
						return nil, fmt.Errorf("%s: sin frontmatter", p)
					}
					rel, _ := filepath.Rel(root, p)
					out[rel+" (frontmatter)"] = fm
				}
				return out, nil
			},
		},
		{
			name:      "rules core (RULES.md alone)",
			maxTokens: 2000,
			parts: func(root string) (map[string][]byte, error) {
				return readAll(root, "RULES.md")
			},
		},
		{
			// Loaded only when a trigger fires. The 1k allowance above the
			// previous ceiling is the vendored Anthropic frontend-design skill.
			name:      "on-demand (skill bodies)",
			maxTokens: 10800,
			parts: func(root string) (map[string][]byte, error) {
				skills, err := skillPaths(root)
				if err != nil {
					return nil, err
				}
				out := make(map[string][]byte, len(skills))
				for _, p := range skills {
					b, err := os.ReadFile(p)
					if err != nil {
						return nil, err
					}
					rel, _ := filepath.Rel(root, p)
					out[rel+" (body)"] = frontmatterRE.ReplaceAll(b, nil)
				}
				return out, nil
			},
		},
		{
			// Injected on every post-edit turn — small text × high
			// frequency. Guard messages join this category when they are
			// shortened (phase 3 / step 9); the ceiling then tightens.
			name:      "high-frequency (post-task nudge)",
			maxTokens: 400,
			parts: func(root string) (map[string][]byte, error) {
				return readAll(root, "plugins/mcp-tools-plugin/src/nudges/post-task-maintenance.md")
			},
		},
		{
			// Everything: prevents shrinking always-on by inflating skills.
			// Includes the vendored Anthropic frontend-design skill verbatim.
			name:      "total corpus (rules + repo files + skills)",
			maxTokens: 16000,
			parts: func(root string) (map[string][]byte, error) {
				out, err := readAll(root, "RULES.md", "AGENTS.md", "CLAUDE.md")
				if err != nil {
					return nil, err
				}
				skills, err := skillPaths(root)
				if err != nil {
					return nil, err
				}
				for _, p := range skills {
					b, err := os.ReadFile(p)
					if err != nil {
						return nil, err
					}
					rel, _ := filepath.Rel(root, p)
					out[rel] = b
				}
				return out, nil
			},
		},
	}

	for _, cat := range categories {
		t.Run(cat.name, func(t *testing.T) {
			parts, err := cat.parts(root)
			if err != nil {
				t.Fatal(err)
			}
			var report strings.Builder
			total := 0
			for name, content := range parts {
				n := estimateTokens(content)
				total += n
				fmt.Fprintf(&report, "  %-56s %6d tok\n", name, n)
			}
			fmt.Fprintf(&report, "  %-56s %6d / %d tok\n", "TOTAL", total, cat.maxTokens)
			t.Log("\n" + report.String())
			if total > cat.maxTokens {
				t.Errorf("presupuesto excedido: %d > %d tokens — recorta o justifica subir el límite en el commit", total, cat.maxTokens)
			}
		})
	}
}
