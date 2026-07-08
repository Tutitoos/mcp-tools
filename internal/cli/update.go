package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/orchestrator"
)

var updateSelf bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Actualiza el binario mcp-tools vía git pull + make install.",
	Long:  "Sin flags: hace self-update (git pull + make install). El upgrade de tools se hace desde el panel web.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateSelf, "self", false, "alias del comportamiento por defecto;保留 por compatibilidad")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }
	_ = updateSelf // currently a no-op alias
	ctx := context.Background()
	return orchestrator.UpdateSelf(ctx, false, logf)
}