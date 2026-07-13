package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// This file implements the REPO-ARTIFACT layer of the instruction pipeline.
// Two layers exist, deliberately separate:
//
//  1. Repo artifacts (this file): RULES.md, CLAUDE.md and AGENTS.md at the
//     repo root are GENERATED from the canonical sources in instructions/
//     and committed. `mcp-tools instructions sync` regenerates them;
//     `mcp-tools instructions check` (and the drift test in
//     instructions_test.go) fail when a generated file was edited by hand
//     or the sources changed without regenerating.
//
//  2. Client installs (sync.go RunRules/RunSkills): the generated RULES.md
//     is distributed to each client by its own mechanism (@import for
//     Claude Code, symlink for OMP, marked block for OpenCode). RunRules
//     stays the ONLY distribution pipeline; it consumes the generated
//     artifact and knows nothing about instructions/.
//
// Content policy: the routing table lives ONCE in instructions/core.md.
// CLAUDE.md and AGENTS.md share instructions/repo-agents.md so they cannot
// drift apart (they historically did). Skills document per-tool detail and
// point back to the core for routing.

// instructionArtifact maps generated repo files to their canonical sources.
type instructionArtifact struct {
	// Path is the generated file, relative to the repo root.
	Path string
	// Sources are concatenated in order, relative to the repo root.
	Sources []string
	// PreserveStart/PreserveEnd delimit an optional marked block in the
	// EXISTING generated file that is re-appended verbatim after the
	// rendered body. Used for CLAUDE.md's rtk block, which the third-party
	// `rtk init` tool owns and rewrites in place.
	PreserveStart, PreserveEnd string
}

var instructionArtifacts = []instructionArtifact{
	{Path: "RULES.md", Sources: []string{"instructions/core.md"}},
	{Path: "AGENTS.md", Sources: []string{"instructions/repo-agents.md"}},
	{
		Path:          "CLAUDE.md",
		Sources:       []string{"instructions/repo-agents.md"},
		PreserveStart: "<!-- rtk-instructions",
		PreserveEnd:   "<!-- /rtk-instructions -->",
	},
}

// generatedHeader marks a file as generated. It is inserted after the YAML
// frontmatter when the source has one (frontmatter must stay first for
// parsers), otherwise at the very top.
const generatedHeader = "<!-- GENERATED FILE - do not edit. Source: instructions/. Regenerate: mcp-tools instructions sync -->"

// RenderInstructionArtifacts renders every artifact from its sources under
// root and returns path→content. It reads the CURRENT artifact on disk only
// to carry over preserved marked blocks.
func RenderInstructionArtifacts(root string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(instructionArtifacts))
	for _, a := range instructionArtifacts {
		var body bytes.Buffer
		for i, src := range a.Sources {
			b, err := os.ReadFile(filepath.Join(root, src))
			if err != nil {
				return nil, fmt.Errorf("instructions: leer fuente %s: %w", src, err)
			}
			if i > 0 {
				body.WriteString("\n")
			}
			body.Write(b)
		}
		rendered := insertHeader(body.Bytes())
		if a.PreserveStart != "" {
			if existing, err := os.ReadFile(filepath.Join(root, a.Path)); err == nil {
				if block, ok := extractMarkedBlock(existing, a.PreserveStart, a.PreserveEnd); ok {
					rendered = append(rendered, '\n')
					rendered = append(rendered, block...)
					if !bytes.HasSuffix(rendered, []byte("\n")) {
						rendered = append(rendered, '\n')
					}
				}
			}
		}
		out[a.Path] = rendered
	}
	return out, nil
}

// insertHeader places generatedHeader after the YAML frontmatter (if any),
// else at the top.
func insertHeader(body []byte) []byte {
	header := []byte(generatedHeader + "\n")
	if bytes.HasPrefix(body, []byte("---\n")) {
		if end := bytes.Index(body[4:], []byte("\n---\n")); end >= 0 {
			cut := 4 + end + len("\n---\n")
			out := make([]byte, 0, len(body)+len(header)+1)
			out = append(out, body[:cut]...)
			out = append(out, header...)
			out = append(out, body[cut:]...)
			return out
		}
	}
	return append(header, body...)
}

// extractMarkedBlock returns the lines from the first line containing
// startMark through the first subsequent line containing endMark, inclusive.
func extractMarkedBlock(content []byte, startMark, endMark string) ([]byte, bool) {
	lines := strings.SplitAfter(string(content), "\n")
	start := -1
	for i, l := range lines {
		if start == -1 && strings.Contains(l, startMark) {
			start = i
			continue
		}
		if start != -1 && strings.Contains(l, endMark) {
			block := strings.Join(lines[start:i+1], "")
			if !strings.HasSuffix(block, "\n") {
				block += "\n"
			}
			return []byte(block), true
		}
	}
	return nil, false
}

// SyncInstructions regenerates the repo artifacts in place. With dry=true it
// only reports what would change.
func SyncInstructions(dry bool, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	root := config.RepoRoot()
	rendered, err := RenderInstructionArtifacts(root)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "── instructions sync (RULES.md, AGENTS.md, CLAUDE.md ← instructions/)")
	for _, a := range instructionArtifacts {
		dst := filepath.Join(root, a.Path)
		current, _ := os.ReadFile(dst)
		if bytes.Equal(current, rendered[a.Path]) {
			fmt.Fprintf(out, "  OK %s (sin cambios)\n", a.Path)
			continue
		}
		if dry {
			fmt.Fprintf(out, "  ~  %s (cambiaría, dry-run)\n", a.Path)
			continue
		}
		if err := os.WriteFile(dst, rendered[a.Path], 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "  OK %s regenerado\n", a.Path)
	}
	return nil
}

// CheckInstructions fails if any generated artifact differs from what the
// sources render to — i.e. someone edited a generated file by hand, or
// edited instructions/ without running `mcp-tools instructions sync`.
func CheckInstructions(out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	return checkInstructionsAt(config.RepoRoot(), out)
}

func checkInstructionsAt(root string, out io.Writer) error {
	rendered, err := RenderInstructionArtifacts(root)
	if err != nil {
		return err
	}
	var stale []string
	for _, a := range instructionArtifacts {
		current, err := os.ReadFile(filepath.Join(root, a.Path))
		if err != nil || !bytes.Equal(current, rendered[a.Path]) {
			stale = append(stale, a.Path)
			continue
		}
		fmt.Fprintf(out, "  OK %s\n", a.Path)
	}
	if len(stale) > 0 {
		return fmt.Errorf("instructions check: desactualizados: %s — corre `mcp-tools instructions sync` y commitea (los generados no se editan a mano)", strings.Join(stale, ", "))
	}
	return nil
}
