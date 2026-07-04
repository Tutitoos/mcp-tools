package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Estado de los 5 contenedores",
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Run("ps")
	},
}

func init() { rootCmd.AddCommand(psCmd) }
