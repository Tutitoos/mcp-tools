package mcp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

type ServerSpec struct {
	Name    string
	Wrapper string
	Args    []string
}

// mcpServers is the canonical list of tools that ARE MCP servers.
// Tools not in this map (nvidia-toolkit, qdrant, ollama, rtk, tokensave,
// claude-mem) are "system tools" and are NOT registered with MCP clients.
// To register a new tool as MCP, add it here AND to tools.Registry().
var mcpServers = map[string]ServerSpec{
	"codebase-memory": {
		Name:    "mcp_tools_codebase_memory",
		Wrapper: "codebase-memory-mcp",
		Args:    []string{"--ui=true"},
	},
	"mem0": {
		Name:    "mcp_tools_mem0",
		Wrapper: filepath.Join(config.WrapperDir(), "mem0-launcher"),
		Args:    []string{},
	},
	"headroom": {
		Name:    "mcp_tools_headroom",
		Wrapper: "headroom",
		Args:    []string{"mcp", "serve"},
	},
	"serena": {
		Name:    "mcp_tools_serena",
		Wrapper: "serena",
		Args:    []string{"start-mcp-server", "--context", "agent", "--project-from-cwd"},
	},
}

// Servers returns the ServerSpec list to register in Claude/OpenCode/OMP,
// filtered by the user's persisted selection. Tools that aren't MCP servers
// (per mcpServers) are silently skipped.
func Servers(st state.State) []ServerSpec {
	out := []ServerSpec{}
	for _, key := range st.Selected {
		if s, ok := mcpServers[key]; ok {
			out = append(out, s)
		}
	}
	return out
}

// EnsureWrappers verifies each server's Wrapper is reachable — a bare name goes
// through PATH, an absolute path through os.Stat.
func EnsureWrappers(st state.State) error {
	for _, s := range Servers(st) {
		if !filepath.IsAbs(s.Wrapper) {
			if _, err := exec.LookPath(s.Wrapper); err != nil {
				return fmt.Errorf("comando %q no está en PATH — corre 'mcp-tools install' primero", s.Wrapper)
			}
			continue
		}
		if _, err := os.Stat(s.Wrapper); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("wrapper %s missing — corre 'mcp-tools install' primero", s.Wrapper)
			}
			return err
		}
	}
	return nil
}
