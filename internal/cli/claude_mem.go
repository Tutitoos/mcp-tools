package cli

import "github.com/spf13/cobra"

var claudeMemCmd = &cobra.Command{
	Use:   "claude-mem",
	Short: "Gestiona claude-mem (plugin de Claude Code, requiere Node ≥ 20)",
}

func init() {
	claudeMemCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("claude-mem", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("claude-mem", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("claude-mem", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("claude-mem")},
	)
	rootCmd.AddCommand(claudeMemCmd)
}
