package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var restartCmd = &cobra.Command{
	Use:   "restart <servicio>",
	Short: "Recrea un servicio releyendo .env (equivalente a up -d --force-recreate)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, _ := state.Load()
		files := tools.OllamaComposeFiles(st)
		return docker.RunWithFiles(files, "up", "-d", "--force-recreate", args[0])
	},
}

func init() { rootCmd.AddCommand(restartCmd) }
