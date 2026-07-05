// Package tools is the single source of truth for the components mcp-tools
// manages: install / upgrade / uninstall / status for each is a plain Go
// closure declared next to the Tool struct. Adding a component = write a new
// file in this package + append an entry to Registry().
package tools

import (
	"errors"
	"fmt"
)

// Deploy discriminates how a tool lands on the host.
type Deploy int

const (
	// DeployHost — plain binary in $HOME/.cargo/bin, ~/.local/bin, etc.
	DeployHost Deploy = iota
	// DeployDocker — service in dockers/compose.yaml.
	DeployDocker
	// DeploySudo — requires root (e.g. nvidia-container-toolkit apt install).
	DeploySudo
)

// String returns the human label used in `mcp-tools status --table`.
func (d Deploy) String() string {
	switch d {
	case DeployHost:
		return "Host"
	case DeployDocker:
		return "Docker"
	case DeploySudo:
		return "Sudo"
	}
	return "?"
}

// StatusPayload is what Tool.Status returns, serialised straight to JSON by
// the `mcp-tools <tool> status` subcommand.
type StatusPayload struct {
	Installed  bool           `json:"installed"`
	Version    string         `json:"version,omitempty"`
	Binary     string         `json:"binary,omitempty"`
	MCPClients []string       `json:"mcp_clients,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// Tool is one managed component.
type Tool struct {
	Key           string
	Label         string
	Summary       string
	Deploy        Deploy
	Deps          []string
	DefaultOn     bool
	SelfRegisters bool // if true, mcp-tools mcp-config does NOT wire this tool
	Interactive   bool // hereda stdio; NO puede correr dentro del TUI Bubbletea

	Install   func(dry bool, log func(string)) error
	Upgrade   func(dry bool, log func(string)) error
	Uninstall func(dry bool, log func(string)) error
	Status    func() (StatusPayload, error)
}

// Registry returns the canonical component list in declaration order (deps
// implicitly precede their consumers so TopoSort is stable). DefaultOn for
// nvidia-toolkit resolves at call time via `nvidia-smi -L`.
func Registry() []Tool {
	return []Tool{
		nvidiaToolkitTool(),
		qdrantTool(),
		ollamaTool(),
		codebaseMemoryTool(),
		mem0Tool(),
		headroomTool(),
		rtkTool(),
		claudeMemTool(),
		codegraphTool(),
	}
}

// Get returns the Tool whose Key matches, or an error naming the unknown key.
func Get(key string) (Tool, error) {
	for _, t := range Registry() {
		if t.Key == key {
			return t, nil
		}
	}
	return Tool{}, fmt.Errorf("unknown tool %q", key)
}

// Keys returns every Key in Registry declaration order.
func Keys() []string {
	reg := Registry()
	out := make([]string, len(reg))
	for i, t := range reg {
		out[i] = t.Key
	}
	return out
}

// TopoSort orders `keys` so every Tool appears after its Deps. Deps outside
// `keys` are ignored — only intra-set ordering is enforced. Cycles → error.
func TopoSort(keys []string) ([]string, error) {
	in := map[string]bool{}
	for _, k := range keys {
		in[k] = true
	}
	visited := map[string]int{} // 0 unvisited, 1 on stack, 2 done
	order := make([]string, 0, len(keys))
	var visit func(k string) error
	visit = func(k string) error {
		if visited[k] == 2 {
			return nil
		}
		if visited[k] == 1 {
			return fmt.Errorf("dependency cycle at %q", k)
		}
		visited[k] = 1
		t, err := Get(k)
		if err != nil {
			return err
		}
		for _, dep := range t.Deps {
			if !in[dep] {
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[k] = 2
		order = append(order, k)
		return nil
	}
	// Iterate in registry order for a deterministic result on independent keys.
	reg := Registry()
	seen := map[string]bool{}
	for _, t := range reg {
		if !in[t.Key] || seen[t.Key] {
			continue
		}
		seen[t.Key] = true
		if err := visit(t.Key); err != nil {
			return nil, err
		}
	}
	// Any keys not matching the registry → surface as error.
	for _, k := range keys {
		if _, err := Get(k); err != nil {
			return nil, err
		}
	}
	if len(order) != len(keys) {
		return nil, errors.New("topo sort produced short result — duplicate keys?")
	}
	return order, nil
}
