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
	Env     map[string]string
	EnvKeys []string
}

// mcpServers is the canonical list of tools that ARE MCP servers. Tools not
// in this map are system tools or self-register and are not wired by mcp-config.
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
	"mongodb": {
		Name:    "mcp_tools_mongodb",
		Wrapper: "mongodb-mcp-server",
		Args:    []string{"--readOnly"},
		EnvKeys: []string{"MDB_MCP_CONNECTION_STRING", "MDB_MCP_API_CLIENT_ID", "MDB_MCP_API_CLIENT_SECRET"},
	},
	"redis": {
		Name:    "mcp_tools_redis",
		Wrapper: "redis-mcp-server",
		EnvKeys: []string{"REDIS_HOST", "REDIS_PORT", "REDIS_DB", "REDIS_USERNAME", "REDIS_PWD", "REDIS_SSL"},
	},
	"docker-mcp-toolkit": {
		Name:    "mcp_tools_docker_toolkit",
		Wrapper: "docker",
		Args:    []string{"mcp", "gateway", "run"},
		Env:     map[string]string{"DOCKER_MCP_IN_CONTAINER": "1"},
	},
	"sentry": {
		Name:    "mcp_tools_sentry",
		Wrapper: "mcp-remote",
		Args:    []string{"https://mcp.sentry.dev/mcp"},
	},
}

// Servers returns the ServerSpec list to register in Claude/OpenCode/OMP,
// filtered by the user's persisted selection. Tools that aren't MCP servers
// (per mcpServers) are silently skipped.
func Servers(st state.State) []ServerSpec {
	out := []ServerSpec{}
	fileEnv, _ := config.LoadEnv(config.EnvFile())
	for _, key := range st.Selected {
		if s, ok := mcpServers[key]; ok {
			s.Env = resolveServerEnv(s, fileEnv)
			out = append(out, s)
		}
	}
	return out
}

func resolveServerEnv(s ServerSpec, fileEnv map[string]string) map[string]string {
	env := make(map[string]string, len(s.Env)+len(s.EnvKeys))
	for key, value := range s.Env {
		env[key] = value
	}
	for _, key := range s.EnvKeys {
		value := os.Getenv(key)
		if value == "" {
			value = fileEnv[key]
		}
		if value != "" {
			env[key] = value
		}
	}
	return env
}

func serverEnvironment(s ServerSpec, home string) map[string]string {
	env := make(map[string]string, len(s.Env)+1)
	env["HOME"] = home
	for key, value := range s.Env {
		env[key] = value
	}
	return env
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
