package cli

import "github.com/spf13/cobra"

var mem0Cmd = &cobra.Command{
	Use:   "mem0",
	Short: "Gestiona mem0-mcp-selfhosted: install / upgrade / status / uninstall",
	Long:  "mem0 requiere qdrant + ollama; ambos se instalan como parte del mismo tool set (mcp-tools install).",
}

func init() {
	mem0Cmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("mem0", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("mem0", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("mem0", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("mem0")},
	)
	rootCmd.AddCommand(mem0Cmd)
}
