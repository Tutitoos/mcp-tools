package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Para los servicios Docker (mantiene volúmenes)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Run("stop")
	},
}

func init() { rootCmd.AddCommand(stopCmd) }
