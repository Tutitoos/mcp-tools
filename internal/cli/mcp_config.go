package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/mcp"
)

var mcpConfigCmd = &cobra.Command{
	Use:   "mcp-config",
	Short: "Re-registra los MCPs en Claude Code, OpenCode y OMP",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMcpConfig(false, os.Stdout)
	},
}

func init() { rootCmd.AddCommand(mcpConfigCmd) }

// RunMcpConfig executes the mcp-config workflow. dry=true prints intended actions
// without touching the filesystem.
func RunMcpConfig(dry bool, out io.Writer) error {
	if err := mcp.EnsureWrappers(); err != nil {
		return err
	}
	if dry {
		fmt.Fprintln(out, "SKIP (dry) — would register 3 servers in Claude Code / OpenCode / OMP")
		return nil
	}
	log := func(s string) { fmt.Fprintln(out, s) }
	if err := mcp.ConfigureClaude(log); err != nil {
		return err
	}
	if err := mcp.ConfigureOpenCode(log); err != nil {
		return err
	}
	if err := mcp.ConfigureOMP(log); err != nil {
		return err
	}
	return nil
}
