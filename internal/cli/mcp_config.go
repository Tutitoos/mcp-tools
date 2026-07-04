package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/mcp"
)

var mcpConfigCmd = &cobra.Command{
	Use:   "mcp-config",
	Short: "Re-registra los MCPs en Claude Code, OpenCode y OMP",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMcpConfig(false)
	},
}

func init() { rootCmd.AddCommand(mcpConfigCmd) }

// RunMcpConfig executes the mcp-config workflow. dry=true prints intended actions
// without touching the filesystem.
func RunMcpConfig(dry bool) error {
	if err := mcp.EnsureWrappers(); err != nil {
		return err
	}
	if dry {
		fmt.Println("SKIP (dry) — would register 3 servers in Claude Code / OpenCode / OMP")
		return nil
	}
	if err := mcp.ConfigureClaude(func(s string) { fmt.Println(s) }); err != nil {
		return err
	}
	if err := mcp.ConfigureOpenCode(func(s string) { fmt.Println(s) }); err != nil {
		return err
	}
	if err := mcp.ConfigureOMP(func(s string) { fmt.Println(s) }); err != nil {
		return err
	}
	return nil
}
