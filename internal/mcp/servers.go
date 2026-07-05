package mcp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// ServerSpec is one MCP server entry the installer registers in every client.
type ServerSpec struct {
	Name    string
	Wrapper string
	Args    []string
}

// Servers returns the ServerSpec list to register in Claude/OpenCode/OMP,
// filtered by the user's persisted selection: only tools that expose an
// MCP stdio surface AND aren't SelfRegisters=true land here.
func Servers(st state.State) []ServerSpec {
	out := []ServerSpec{}
	for _, key := range st.Selected {
		t, err := tools.Get(key)
		if err != nil || t.SelfRegisters {
			continue
		}
		switch key {
		case "codebase-memory":
			out = append(out, ServerSpec{
				Name:    "mcp_tools_codebase_memory",
				Wrapper: "codebase-memory-mcp",
				Args:    []string{"--ui=false"},
			})
		case "mem0":
			out = append(out, ServerSpec{
				Name:    "mcp_tools_mem0",
				Wrapper: filepath.Join(config.WrapperDir(), "mem0-launcher"),
				Args:    []string{},
			})
		case "headroom":
			out = append(out, ServerSpec{
				Name:    "mcp_tools_headroom",
				Wrapper: "headroom",
				Args:    []string{"mcp", "serve"},
			})
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
