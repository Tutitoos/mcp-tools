package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Reconstruye las imágenes locales tras editar Dockerfiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return docker.Run("build")
	},
}

func init() { rootCmd.AddCommand(buildCmd) }
