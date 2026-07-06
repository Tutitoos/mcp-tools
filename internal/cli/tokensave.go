package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var tokensaveCmd = &cobra.Command{
	Use:   "tokensave",
	Short: "Gestiona TokenSave MCP: install / upgrade / status / uninstall / cap",
	Long:  "TokenSave se instala como cargo tool y se autoregistra en Claude / OpenCode / OMP y demás agentes detectados (nombre MCP nativo: tokensave).",
}

var tokensaveCapCmd = &cobra.Command{
	Use:   "cap",
	Short: "Envuelve tokensave en un cgroup de 30 GiB RAM en los MCP clients",
	Long: "Instala ~/.local/bin/tokensave-capped (wrapper systemd-run --scope) y reescribe la\n" +
		"entrada 'tokensave' en los configs MCP de Claude Code / OpenCode / OMP para apuntar\n" +
		"al wrapper. Cada spawn del MCP arranca en un cgroup transient con MemoryMax=30G,\n" +
		"MemoryHigh=28G, sin swap.\n\n" +
		"Idempotente. Corre esto tras cualquier 'tokensave install' o 'mcp-tools tokensave\n" +
		"upgrade' — tokensave reescribe los configs con el binario crudo, borrando el wrap.",
	RunE: runTokensaveCap,
}

var tokensaveUncapCmd = &cobra.Command{
	Use:   "uncap",
	Short: "Restaura la entrada 'tokensave' apuntando al binario directo (deshace cap)",
	RunE:  runTokensaveUncap,
}

func init() {
	tokensaveCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("tokensave", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("tokensave", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("tokensave", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("tokensave")},
		tokensaveCapCmd,
		tokensaveUncapCmd,
	)
	rootCmd.AddCommand(tokensaveCmd)
}

// wrapperTemplate is written verbatim to ~/.local/bin/tokensave-capped.
// %s is replaced with the resolved path to the real tokensave binary.
const wrapperTemplate = `#!/bin/bash
# Memory-capped tokensave launcher (managed by 'mcp-tools tokensave cap').
# Do NOT hand-edit — re-run 'mcp-tools tokensave cap' to regenerate.
#
# Each MCP client spawn of tokensave runs inside a transient systemd scope
# with a 30 GiB hard RAM cap enforced by the kernel via cgroup v2.
set -euo pipefail
exec systemd-run \
  --user \
  --scope \
  --quiet \
  --collect \
  --property=MemoryMax=30G \
  --property=MemoryHigh=28G \
  --property=MemorySwapMax=0 \
  --property=TasksMax=512 \
  %s "$@"
`

// tokensaveClientTargets are the MCP client configs the cap/uncap subcommands
// visit. Order is stable so the output prints Claude → OpenCode → OMP.
type tokensaveClientTarget struct {
	label string
	path  string
	root  string // top-level key that holds the MCP server map
}

func tokensaveTargets(home string) []tokensaveClientTarget {
	return []tokensaveClientTarget{
		{"Claude", filepath.Join(home, ".claude.json"), "mcpServers"},
		{"OpenCode", filepath.Join(home, ".config/opencode/opencode.json"), "mcp"},
		{"OMP", filepath.Join(home, ".omp/agent/mcp.json"), "mcpServers"},
	}
}

func runTokensaveCap(cmd *cobra.Command, _ []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("tokensave cap requires systemd (Linux only); current OS: %s", runtime.GOOS)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	realBin := filepath.Join(home, ".cargo/bin/tokensave")
	if _, err := os.Stat(realBin); err != nil {
		return fmt.Errorf("tokensave no instalado en %s — corre 'mcp-tools tokensave install' primero", realBin)
	}

	fmt.Fprintln(os.Stdout, "── tokensave cap (30 GiB, cgroup enforced)")

	wrapper := filepath.Join(home, ".local/bin/tokensave-capped")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(wrapperTemplate, realBin)
	if err := os.WriteFile(wrapper, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write wrapper: %w", err)
	}
	fmt.Fprintf(os.Stdout, "  OK wrapper %s\n", wrapper)

	for _, t := range tokensaveTargets(home) {
		status, err := recapTokensaveConfig(t.path, t.root, wrapper)
		if err != nil {
			return fmt.Errorf("%s: %w", t.label, err)
		}
		fmt.Fprintf(os.Stdout, "  %-4s %-9s %s\n", recapIcon(status), t.label, status)
	}

	fmt.Fprintln(os.Stdout, "── tokensave cap completo — reinicia sesiones MCP para respawn con cap")
	return nil
}

func runTokensaveUncap(cmd *cobra.Command, _ []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("tokensave uncap requires systemd (Linux only); current OS: %s", runtime.GOOS)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	realBin := filepath.Join(home, ".cargo/bin/tokensave")
	if _, err := os.Stat(realBin); err != nil {
		return fmt.Errorf("tokensave no instalado en %s — corre 'mcp-tools tokensave install' primero", realBin)
	}

	fmt.Fprintln(os.Stdout, "── tokensave uncap (restaurar entry directo)")
	for _, t := range tokensaveTargets(home) {
		status, err := recapTokensaveConfig(t.path, t.root, realBin)
		if err != nil {
			return fmt.Errorf("%s: %w", t.label, err)
		}
		fmt.Fprintf(os.Stdout, "  %-4s %-9s %s\n", recapIcon(status), t.label, status)
	}
	fmt.Fprintln(os.Stdout, "── tokensave uncap completo (wrapper file no borrado — sigue disponible)")
	return nil
}

// recapTokensaveConfig rewrites the `tokensave` MCP server entry inside path
// so its command points at bin. Idempotent. Handles both shapes:
//   - Claude / OMP: `"command": "<bin>", "args": [...]`
//   - OpenCode:     `"command": ["<bin>", ...args]`
//
// Returns a short status: "rewritten" / "already-set" / "no-entry" / "no-file".
func recapTokensaveConfig(path, root, bin string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "no-file", nil
		}
		return "", err
	}
	var d map[string]any
	if err := json.Unmarshal(b, &d); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	servers, _ := d[root].(map[string]any)
	if servers == nil {
		return "no-entry", nil
	}
	entry, ok := servers["tokensave"].(map[string]any)
	if !ok {
		return "no-entry", nil
	}

	changed := false
	switch cmdVal := entry["command"].(type) {
	case []any:
		// OpenCode shape: replace the binary slot (index 0), keep the tail.
		if len(cmdVal) == 0 {
			cmdVal = []any{bin}
			changed = true
		} else if s, _ := cmdVal[0].(string); s != bin {
			cmdVal[0] = bin
			changed = true
		}
		entry["command"] = cmdVal
	case string:
		// Claude / OMP shape.
		if cmdVal != bin {
			entry["command"] = bin
			changed = true
		}
	default:
		return "", fmt.Errorf("unexpected command shape in %s: %T", path, cmdVal)
	}
	if !changed {
		return "already-set", nil
	}
	servers["tokensave"] = entry
	d[root] = servers

	out, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", err
	}
	return "rewritten", nil
}

func recapIcon(status string) string {
	switch status {
	case "rewritten":
		return "OK"
	case "already-set":
		return "·"
	case "no-entry", "no-file":
		return "SKIP"
	}
	return "?"
}
