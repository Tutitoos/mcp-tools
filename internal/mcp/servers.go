package mcp

import (
	"errors"
	"fmt"
	"os"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// ServerSpec is one MCP server entry the installer registers in every client.
type ServerSpec struct {
	Name    string
	Wrapper string
	Args    []string
}

// Servers is the canonical list of three MCP servers this repo owns.
func Servers() []ServerSpec {
	return []ServerSpec{
		{
			Name:    "mcp_tools_codebase_memory",
			Wrapper: config.WrapperPath("codebase-memory"),
			Args:    []string{"--ui=false"},
		},
		{
			Name:    "mcp_tools_mem0",
			Wrapper: config.WrapperPath("mem0"),
			Args:    []string{},
		},
		{
			Name:    "mcp_tools_headroom",
			Wrapper: config.WrapperPath("headroom"),
			Args:    []string{},
		},
	}
}

// EnsureWrappers returns an error if any wrapper file is missing.
func EnsureWrappers() error {
	for _, s := range Servers() {
		if _, err := os.Stat(s.Wrapper); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("wrapper %s missing — corre 'mcp-tools install' primero", s.Wrapper)
			}
			return err
		}
	}
	return nil
}
