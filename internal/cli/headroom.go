package cli

import "github.com/spf13/cobra"

var headroomCmd = &cobra.Command{
	Use:   "headroom",
	Short: "Gestiona Headroom: install / upgrade / status / uninstall",
	Long:  "Headroom se instala como uv tool (headroom-ai[mcp,proxy]) y se registra como MCP mcp_tools_headroom.",
}

func init() {
	headroomCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("headroom", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("headroom", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("headroom", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("headroom")},
	)
	rootCmd.AddCommand(headroomCmd)
}
