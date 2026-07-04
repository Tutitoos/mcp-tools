package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/version"
)

var rootCmd = &cobra.Command{
	Use:           "mcp-tools",
	Short:         "Stack self-hosted de MCP servers en Docker",
	Long:          "mcp-tools orquesta el stack Docker de servidores MCP y su registro en Claude Code, OpenCode y OMP.",
	Version:       fmt.Sprintf("%s (%s, %s)", version.Version, version.Commit, version.Date),
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command; exits with 1 on any error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
