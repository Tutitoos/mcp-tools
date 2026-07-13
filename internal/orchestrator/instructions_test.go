package orchestrator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRootForTest walks up from the test's cwd to the directory containing
// go.mod, so the drift test works in any checkout (CI included) without
// depending on MCP_TOOLS_ROOT or $HOME.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test cwd")
		}
		dir = parent
	}
}

// TestInstructionArtifactsInSync is the anti-drift contract: RULES.md,
// AGENTS.md and CLAUDE.md are generated from instructions/ and MUST match
// what the sources render to. If this fails, someone hand-edited a generated
// file or changed instructions/ without running `mcp-tools instructions sync`.
func TestInstructionArtifactsInSync(t *testing.T) {
	root := repoRootForTest(t)
	rendered, err := RenderInstructionArtifacts(root)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, a := range instructionArtifacts {
		current, err := os.ReadFile(filepath.Join(root, a.Path))
		if err != nil {
			t.Fatalf("%s: %v", a.Path, err)
		}
		if !bytes.Equal(current, rendered[a.Path]) {
			t.Errorf("%s desactualizado respecto a instructions/ — corre `mcp-tools instructions sync` y commitea", a.Path)
		}
		if !bytes.Contains(current, []byte("GENERATED FILE")) {
			t.Errorf("%s no lleva el header de generado", a.Path)
		}
	}
}

// TestInsertHeaderRespectsFrontmatter pins that the generated header lands
// AFTER the YAML frontmatter (parsers require frontmatter first) and at the
// top otherwise.
func TestInsertHeaderRespectsFrontmatter(t *testing.T) {
	withFM := []byte("---\ndescription: x\n---\n# body\n")
	got := string(insertHeader(withFM))
	if !strings.HasPrefix(got, "---\ndescription: x\n---\n<!-- GENERATED") {
		t.Errorf("header debe ir tras el frontmatter:\n%s", got)
	}
	plain := []byte("# body\n")
	if got := string(insertHeader(plain)); !strings.HasPrefix(got, "<!-- GENERATED") {
		t.Errorf("header debe ir al inicio sin frontmatter:\n%s", got)
	}
}

// TestRenderPreservesRTKBlock pins the CLAUDE.md contract: the marked block
// owned by the third-party `rtk init` tool survives regeneration verbatim.
func TestRenderPreservesRTKBlock(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("instructions/core.md", "# core\n")
	writeFile("instructions/repo-agents.md", "# agents body\n")
	writeFile("CLAUDE.md", "old body\n<!-- rtk-instructions v2 -->\nrtk stuff\n<!-- /rtk-instructions -->\n")

	rendered, err := RenderInstructionArtifacts(root)
	if err != nil {
		t.Fatal(err)
	}
	claude := string(rendered["CLAUDE.md"])
	if !strings.Contains(claude, "# agents body") {
		t.Errorf("cuerpo compartido ausente:\n%s", claude)
	}
	if !strings.Contains(claude, "<!-- rtk-instructions v2 -->\nrtk stuff\n<!-- /rtk-instructions -->") {
		t.Errorf("bloque rtk no preservado:\n%s", claude)
	}
	if strings.Contains(claude, "old body") {
		t.Errorf("contenido viejo no debe sobrevivir fuera del bloque marcado:\n%s", claude)
	}
	// AGENTS.md shares the same body but never carries the rtk block.
	agents := string(rendered["AGENTS.md"])
	if strings.Contains(agents, "rtk-instructions") {
		t.Errorf("AGENTS.md no debe llevar el bloque rtk:\n%s", agents)
	}
}
