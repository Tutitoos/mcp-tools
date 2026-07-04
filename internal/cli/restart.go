package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var restartCmd = &cobra.Command{
	Use:   "restart <servicio>",
	Short: "Recrea un servicio releyendo .env / .env.mem0 (equivalente a up -d --force-recreate)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Run("up", "-d", "--force-recreate", args[0])
	},
}

func init() { rootCmd.AddCommand(restartCmd) }
