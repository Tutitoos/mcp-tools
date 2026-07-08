package cli

import "github.com/spf13/cobra"

var codexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Gestiona OpenAI Codex CLI (npm i -g @openai/codex; requiere sudo)",
}

func init() {
	codexCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("codex", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("codex", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("codex", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("codex")},
	)
	rootCmd.AddCommand(codexCmd)
}
