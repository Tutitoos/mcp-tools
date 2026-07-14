package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

// ConfigureCodex registers each ServerSpec in the OpenAI Codex CLI config at
// ~/.codex/config.toml under `[mcp_servers.<name>]` tables. Preserves every
// other top-level key and every non-owned section (model, approval_policy,
// user-added mcp_servers.*, etc.). SKIP silently if the parent dir is
// missing (Codex not installed) or if `codex` CLI is absent — the latter is
// the cheapest signal that the user hasn't installed Codex yet.
func ConfigureCodex(st state.State, log func(string)) error {
	if _, err := exec.LookPath("codex"); err != nil {
		log("  SKIP Codex (codex CLI not found)")
		return nil
	}
	return rewriteCodexConfig(st, log)
}

// rewriteCodexConfig performs the actual config.toml rewrite. Split out from
// ConfigureCodex so it's testable without requiring the `codex` CLI on PATH.
func rewriteCodexConfig(st state.State, log func(string)) error {
	home, err := config.HomeDir()
	if err != nil {
		return fmt.Errorf("codex mcp: %w", err)
	}
	file := filepath.Join(home, ".codex", "config.toml")
	parent := filepath.Dir(file)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			log(fmt.Sprintf("  SKIP Codex (%s missing — Codex not installed?)", parent))
			return nil
		}
		return err
	}
	if err := Backup(file); err != nil {
		return err
	}

	raw, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(raw)
	wasEmpty := strings.TrimSpace(text) == ""

	wanted := map[string]bool{}
	specs := Servers(st)
	for _, s := range specs {
		wanted[s.Name] = true
	}
	for _, name := range ownedTopLevelSections(text) {
		if !wanted[name] {
			log(fmt.Sprintf("  prune Codex %s (obsolete)", strings.TrimPrefix(name, "mcp_tools_")))
		}
	}

	out := stripOwnedSections(text)
	if wasEmpty {
		out = "# managed by mcp-tools — do not edit MCP sections by hand\n"
	} else if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	for _, s := range specs {
		out += renderCodexServer(s)
	}

	if err := os.WriteFile(file, []byte(out), 0o600); err != nil {
		return err
	}
	log(fmt.Sprintf("  OK Codex %s", file))
	return nil
}

// stripOwnedSections removes every `[mcp_servers.mcp_tools_<name>]` section
// (and its `[mcp_servers.mcp_tools_<name>.env]` sub-table) from a Codex
// config.toml, line by line, so it can be replaced by a fresh render.
// Sections we don't own (model, approval_policy, other mcp_servers.*, etc.)
// are passed through byte-for-byte. A line-based scan is used instead of a
// regex: RE2 has no lookahead, so "consume lines up to the next [section]
// header" can't be expressed as a single safe pattern without risking
// swallowing a following user section.
func stripOwnedSections(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	skip := false
	for _, line := range lines {
		if name, ok := sectionHeaderName(line); ok {
			skip = strings.HasPrefix(name, "mcp_servers.mcp_tools_")
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
	// (trailing newline re-added by the caller before appending sections)
}

// ownedTopLevelSections returns the mcp_tools_* server names (without the
// ".env" sub-table) currently present in text, for obsolete-section logging.
func ownedTopLevelSections(text string) []string {
	var names []string
	for _, line := range strings.Split(text, "\n") {
		name, ok := sectionHeaderName(line)
		if !ok || !strings.HasPrefix(name, "mcp_servers.mcp_tools_") || strings.HasSuffix(name, ".env") {
			continue
		}
		names = append(names, strings.TrimPrefix(name, "mcp_servers."))
	}
	return names
}

// sectionHeaderName reports whether line is a bare `[section.name]` header
// (no trailing comment/content after the closing bracket) and, if so,
// returns the name between the brackets.
func sectionHeaderName(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return "", false
	}
	return t[1 : len(t)-1], true
}

// renderCodexServer emits one server's `[mcp_servers.<name>]` +
// `[mcp_servers.<name>.env]` block, deterministically.
func renderCodexServer(s ServerSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n[mcp_servers.%s]\n", s.Name)
	fmt.Fprintf(&b, "command = %s\n", tomlString(s.Wrapper))
	args := make([]string, len(s.Args))
	for i, a := range s.Args {
		args[i] = tomlString(a)
	}
	fmt.Fprintf(&b, "args = [%s]\n", strings.Join(args, ", "))
	fmt.Fprintf(&b, "\n[mcp_servers.%s.env]\n", s.Name)
	b.WriteString("HOME = \"${HOME}\"\n")
	keys := make([]string, 0, len(s.Env))
	for key := range s.Env {
		if key != "HOME" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s = %s\n", key, tomlString(s.Env[key]))
	}
	return b.String()
}

func tomlString(s string) string {
	// Minimal TOML basic string escape: backslash and double-quote.
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + esc.Replace(s) + `"`
}
