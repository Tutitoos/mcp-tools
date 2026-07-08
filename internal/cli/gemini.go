package cli

import "github.com/spf13/cobra"

var geminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Gestiona Google Gemini CLI (npm i -g @google/gemini-cli; requiere sudo)",
}

func init() {
	geminiCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("gemini", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("gemini", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("gemini", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("gemini")},
	)
	rootCmd.AddCommand(geminiCmd)
}
