package cli

import "github.com/spf13/cobra"

var codegraphCmd = &cobra.Command{
	Use:   "codegraph",
	Short: "Gestiona CodeGraph MCP (auto-registro en 8 IDEs, opt-in)",
}

func init() {
	codegraphCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("codegraph", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("codegraph", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("codegraph", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("codegraph")},
	)
	rootCmd.AddCommand(codegraphCmd)
}
