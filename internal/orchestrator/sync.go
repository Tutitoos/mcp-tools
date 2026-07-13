package orchestrator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/mcp"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

// RunMcpConfig executes the mcp-config workflow against the given state.
// This is the orchestrator's port of the legacy cli.RunMcpConfig.
func RunMcpConfig(dry bool, st state.State, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if !dry {
		if err := mcp.EnsureWrappers(st); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "── configure MCP clients")
	if dry {
		fmt.Fprintf(out, "  SKIP (dry) — would register %d servers in Claude Code / OpenCode / OMP / Codex / Gemini\n", len(mcp.Servers(st)))
		return nil
	}
	log := func(s string) { fmt.Fprintln(out, s) }
	type clientErr struct {
		client string
		err    error
		hint   string
	}
	var failures []clientErr
	if err := mcp.ConfigureClaude(st, log); err != nil {
		failures = append(failures, clientErr{client: "claude", err: err, hint: "revisa ~/.claude.json y 'claude mcp list'"})
	}
	if err := mcp.ConfigureOpenCode(st, log); err != nil {
		failures = append(failures, clientErr{client: "opencode", err: err, hint: "revisa ~/.config/opencode/opencode.json"})
	}
	if err := mcp.ConfigureOMP(st, log); err != nil {
		failures = append(failures, clientErr{client: "omp", err: err, hint: "revisa ~/.omp/agent/mcp.json"})
	}
	if err := mcp.ConfigureCodex(st, log); err != nil {
		failures = append(failures, clientErr{client: "codex", err: err, hint: "revisa ~/.codex/config.toml"})
	}
	if err := mcp.ConfigureGemini(st, log); err != nil {
		failures = append(failures, clientErr{client: "gemini", err: err, hint: "revisa ~/.gemini/settings.json"})
	}
	if len(failures) == 0 {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mcp-config parcial — %d/%d clientes fallaron:\n", len(failures), 5)
	for _, f := range failures {
		fmt.Fprintf(&b, "  %s: %v — %s\n", f.client, f.err, f.hint)
	}
	b.WriteString("Para reintentar: panel web /settings → \"Re-run mcp-config\" (POST /api/mcp-config/sync)")
	return fmt.Errorf("%s", b.String())
}

// RunSkills is the orchestrator's port of the legacy cli.RunSkills. It
// creates symlinks under the supported client directories.
func RunSkills(dry bool, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	src := filepath.Join(config.RepoRoot(), "skills")
	targets := []string{
		filepath.Join(home, ".claude/skills"),
		filepath.Join(home, ".config/opencode/skills"),
		filepath.Join(home, ".omp/agent/skills"),
	}
	stale := []string{
		filepath.Join(home, ".claude/skills/codebase-memory-mcp"),
		filepath.Join(home, ".config/opencode/skills/codebase-memory-mcp"),
		filepath.Join(home, ".omp/agent/skills/codebase-memory-mcp"),
	}

	fmt.Fprintln(out, "── install skills (Claude Code, OpenCode, OMP)")

	stalesRemoved := 0
	for _, s := range stale {
		if _, err := os.Lstat(s); err != nil {
			continue
		}
		if !dry {
			if err := os.RemoveAll(s); err != nil {
				return err
			}
		}
		stalesRemoved++
	}
	if stalesRemoved > 0 {
		fmt.Fprintf(out, "  · %d stale dir(s) removed\n", stalesRemoved)
	}

	symlinksTotal := 0
	for _, t := range targets {
		if !dry {
			if err := os.MkdirAll(t, 0o755); err != nil {
				return err
			}
		}
		for _, name := range skillNames {
			dst := filepath.Join(t, name)
			srcPath := filepath.Join(src, name)
			if !dry {
				_ = os.Remove(dst)
				if err := os.Symlink(srcPath, dst); err != nil {
					return fmt.Errorf("symlink %s: %w", dst, err)
				}
			}
			symlinksTotal++
		}
	}

	if !dry {
		for _, t := range targets {
			for _, name := range skillNames {
				f := filepath.Join(t, name, "SKILL.md")
				if _, err := os.Stat(f); err != nil {
					return fmt.Errorf("FAIL %s: %w", f, err)
				}
			}
		}
		fmt.Fprintf(out, "  OK %d symlinks verificados (%d skills × %d clients)\n", symlinksTotal, len(skillNames), len(targets))
	} else {
		fmt.Fprintf(out, "  OK %d symlinks (dry — %d skills × %d clients)\n", symlinksTotal, len(skillNames), len(targets))
	}
	return nil
}

// RunRules is the orchestrator's port of the legacy cli.RunRules. It
// installs the mcp-tools RULES.md as an @import (Claude Code), a symlink
// (OMP), and a marked block (OpenCode).
func RunRules(dry bool, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	rulesSrc := filepath.Join(config.RepoRoot(), "RULES.md")
	if _, err := os.Stat(rulesSrc); err != nil {
		return fmt.Errorf("RULES.md no existe en %s", rulesSrc)
	}
	fmt.Fprintln(out, "── install rules (Claude Code, OpenCode, OMP)")

	// OMP: symlink as rule file.
	ompRules := filepath.Join(home, ".omp/rules")
	ompTarget := filepath.Join(ompRules, "mcp-tools.md")
	if !dry {
		if err := os.MkdirAll(ompRules, 0o755); err != nil {
			return err
		}
		_ = os.Remove(ompTarget)
		if err := os.Symlink(rulesSrc, ompTarget); err != nil {
			return err
		}
	}

	// Claude Code: @import in ~/.claude/CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude/CLAUDE.md")
	importLine := "@" + rulesSrc
	if !dry {
		if err := os.MkdirAll(filepath.Dir(claudeMd), 0o755); err != nil {
			return err
		}
		content := ""
		if b, err := os.ReadFile(claudeMd); err == nil {
			content = string(b)
		}
		if !lineExists(content, importLine) {
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += importLine + "\n"
			if err := os.WriteFile(claudeMd, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}

	// OpenCode: marked block in ~/.config/opencode/AGENTS.md.
	opencodeAgents := filepath.Join(home, ".config/opencode/AGENTS.md")
	if !dry {
		if err := os.MkdirAll(filepath.Dir(opencodeAgents), 0o755); err != nil {
			return err
		}
		content := ""
		if b, err := os.ReadFile(opencodeAgents); err == nil {
			content = string(b)
		}
		content = ruleBlockRE.ReplaceAllString(content, "")

		rulesBody, err := os.ReadFile(rulesSrc)
		if err != nil {
			return err
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += markerStart + "\n" + string(rulesBody)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += markerEnd + "\n"
		if err := os.WriteFile(opencodeAgents, []byte(content), 0o644); err != nil {
			return err
		}
	}

	if !dry {
		for _, f := range []string{ompTarget, claudeMd, opencodeAgents} {
			if _, err := os.Stat(f); err != nil {
				return fmt.Errorf("FAIL %s: %w", f, err)
			}
			fmt.Fprintf(out, "  OK %s\n", f)
		}
	}
	return nil
}

func lineExists(content, want string) bool {
	for _, line := range strings.Split(content, "\n") {
		if line == want {
			return true
		}
	}
	return false
}

const (
	markerStart = "<!-- mcp-tools:start -->"
	markerEnd   = "<!-- mcp-tools:end -->"
)

// skillNames mirrors the legacy internal/cli.skillsNames constant.
var skillNames = []string{"codebase-memory", "mem0", "serena", "tokensave"}
