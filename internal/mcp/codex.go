package mcp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/state"
)

// ConfigureCodex registers each ServerSpec in the OpenAI Codex CLI config at
// ~/.codex/config.toml under `[mcp_servers.<name>]` tables. Preserves every
// other top-level key (model, approval_policy, etc.). SKIP silently if the
// parent dir is missing (Codex not installed) or if `codex` CLI is absent —
// the latter is the cheapest signal that the user hasn't installed Codex yet.
func ConfigureCodex(st state.State, log func(string)) error {
	if _, err := exec.LookPath("codex"); err != nil {
		log("  SKIP Codex (codex CLI not found)")
		return nil
	}
	home, _ := os.UserHomeDir()
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

	existing := readCodexConfig(file)
	wanted := map[string]bool{}
	for _, s := range Servers(st) {
		wanted[s.Name] = true
		setCodexServer(existing, s)
	}
	// Prune obsolete mcp_tools_* tables we no longer own.
	for k := range existing {
		if strings.HasPrefix(k, "mcp_servers.") {
			name := strings.TrimPrefix(k, "mcp_servers.")
			if strings.HasPrefix(name, "mcp_tools_") && !wanted[name] {
				delete(existing, k)
				log(fmt.Sprintf("  prune Codex %s (obsolete)", name))
			}
		}
	}

	if err := writeCodexConfig(file, existing); err != nil {
		return err
	}
	log(fmt.Sprintf("  OK Codex %s", file))
	return nil
}

// codexConfig is the in-memory representation of config.toml: a flat
// key→value map. Nested tables (mcp_servers.<name>.*) are flattened with dot
// notation — sufficient for our writes (only mcp_servers.<name>.*) and
// preserves user-edited keys without re-serialising TOML ourselves.
type codexConfig map[string]string

func readCodexConfig(file string) codexConfig {
	out := codexConfig{}
	data, err := os.ReadFile(file)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if eq := strings.Index(line, "="); eq > 0 {
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			out[key] = val
		}
	}
	return out
}

func writeCodexConfig(file string, cfg codexConfig) error {
	var b bytes.Buffer
	b.WriteString("# managed by mcp-tools — do not edit by hand\n")
	for k, v := range cfg {
		b.WriteString(k)
		b.WriteString(" = ")
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return os.WriteFile(file, b.Bytes(), 0o600)
}

// setCodexServer flattens a ServerSpec into Codex's TOML shape:
//
//	[mcp_servers.<name>]
//	command = "..."
//	args = [...]
//	[mcp_servers.<name>.env]
//	HOME = "..."
func setCodexServer(cfg codexConfig, s ServerSpec) {
	prefix := "mcp_servers." + s.Name
	cfg[prefix] = "" // section header marker
	cfg[prefix+".command"] = tomlString(s.Wrapper)
	if len(s.Args) > 0 {
		parts := make([]string, len(s.Args))
		for i, a := range s.Args {
			parts[i] = tomlString(a)
		}
		cfg[prefix+".args"] = "[" + strings.Join(parts, ", ") + "]"
	} else {
		cfg[prefix+".args"] = "[]"
	}
	cfg[prefix+".env.HOME"] = "${HOME}"
}

func tomlString(s string) string {
	// Minimal TOML basic string escape: backslash and double-quote.
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + esc.Replace(s) + `"`
}
