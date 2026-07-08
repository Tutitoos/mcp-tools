package cli

import "github.com/spf13/cobra"

var claudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Gestiona Claude Code CLI (curl installer oficial, ~/.local/bin)",
}

func init() {
	claudeCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("claude", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("claude", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("claude", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("claude")},
	)
	rootCmd.AddCommand(claudeCmd)
}
