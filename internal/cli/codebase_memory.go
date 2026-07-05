package cli

import "github.com/spf13/cobra"

var codebaseMemoryCmd = &cobra.Command{
	Use:   "codebase-memory",
	Short: "Gestiona codebase-memory-mcp: install / upgrade / status / uninstall",
}

func init() {
	codebaseMemoryCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("codebase-memory", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("codebase-memory", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("codebase-memory", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("codebase-memory")},
	)
	rootCmd.AddCommand(codebaseMemoryCmd)
}
