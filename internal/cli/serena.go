package cli

import "github.com/spf13/cobra"

var serenaCmd = &cobra.Command{
	Use:   "serena",
	Short: "Gestiona Serena MCP: install / upgrade / status / uninstall",
	Long:  "Serena se instala como uv tool (serena-agent, Python 3.13) y se registra como MCP mcp_tools_serena.",
}

func init() {
	serenaCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("serena", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("serena", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("serena", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("serena")},
	)
	rootCmd.AddCommand(serenaCmd)
}
