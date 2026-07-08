package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Arranca los servicios Docker (ollama, qdrant)",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load()
		if err != nil {
			return fmt.Errorf("state.json: %w", err)
		}
		files := tools.OllamaComposeFiles(st)
		return docker.RunWithFiles(files, "up", "-d")
	},
}

func init() { rootCmd.AddCommand(upCmd) }
