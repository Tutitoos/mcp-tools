package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Arranca los 5 contenedores",
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Run("up", "-d")
	},
}

func init() { rootCmd.AddCommand(upCmd) }
