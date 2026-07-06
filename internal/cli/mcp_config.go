package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

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
	var errs []error
	if err := mcp.ConfigureClaude(st, log); err != nil {
		errs = append(errs, fmt.Errorf("claude: %w", err))
	}
	if err := mcp.ConfigureOpenCode(st, log); err != nil {
		errs = append(errs, fmt.Errorf("opencode: %w", err))
	}
	if err := mcp.ConfigureOMP(st, log); err != nil {
		errs = append(errs, fmt.Errorf("omp: %w", err))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
