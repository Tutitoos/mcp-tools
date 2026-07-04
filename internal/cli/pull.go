package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var pullCmd = &cobra.Command{
	Use:   "pull <tag>",
	Short: "Descarga un modelo Ollama (docker exec mcp-tools-ollama ollama pull <tag>)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Exec("mcp-tools-ollama", "ollama", "pull", args[0]).Run()
	},
}

func init() { rootCmd.AddCommand(pullCmd) }
