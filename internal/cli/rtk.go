package cli

import "github.com/spf13/cobra"

var rtkCmd = &cobra.Command{
	Use:   "rtk",
	Short: "Gestiona RTK: install / upgrade / status / uninstall",
	Long:  "RTK vive en ~/.cargo/bin/rtk y se hookea en OMP + Claude Code (no es un MCP).",
}

func init() {
	rtkCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("rtk", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("rtk", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("rtk", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("rtk")},
	)
	rootCmd.AddCommand(rtkCmd)
}
