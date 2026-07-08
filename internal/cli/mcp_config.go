package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/mcp"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

var mcpConfigCmd = &cobra.Command{
	Use:   "mcp-config",
	Short: "Re-registra los MCPs en Claude Code, OpenCode y OMP",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load()
		if err != nil {
			return fmt.Errorf("state.json: %w", err)
		}
		return RunMcpConfig(false, st, os.Stdout)
	},
}

func init() { rootCmd.AddCommand(mcpConfigCmd) }

// RunMcpConfig executes the mcp-config workflow against the given state.
// Callers pass an in-memory state so a fresh selection can be registered
// before the state file is written to disk. dry=true prints intended actions.
func RunMcpConfig(dry bool, st state.State, out io.Writer) error {
	if !dry {
		if err := mcp.EnsureWrappers(st); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "── configure MCP clients")
	if dry {
		fmt.Fprintf(out, "  SKIP (dry) — would register %d servers in Claude Code / OpenCode / OMP\n", len(mcp.Servers(st)))
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
		failures = append(failures, clientErr{
			client: "claude",
			err:    err,
			hint:   "revisa ~/.claude.json y 'claude mcp list'",
		})
	}
	if err := mcp.ConfigureOpenCode(st, log); err != nil {
		failures = append(failures, clientErr{
			client: "opencode",
			err:    err,
			hint:   "revisa ~/.config/opencode/opencode.json",
		})
	}
	if err := mcp.ConfigureOMP(st, log); err != nil {
		failures = append(failures, clientErr{
			client: "omp",
			err:    err,
			hint:   "revisa ~/.omp/agent/mcp.json",
		})
	}
	if len(failures) == 0 {
		return nil
	}
	// Build a multi-client recovery message. We do NOT roll back successful
	// clients automatically: rolling back would destructively mutate the
	// other configs and the user must decide.
	var b strings.Builder
	fmt.Fprintf(&b, "mcp-config parcial — %d/%d clientes fallaron:\n", len(failures), 3)
	for _, f := range failures {
		fmt.Fprintf(&b, "  %s: %v — %s\n", f.client, f.err, f.hint)
	}
	b.WriteString("Para reintentar: mcp-tools mcp-config")
	return fmt.Errorf("%s", b.String())
}
