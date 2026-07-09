package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

// `mcp-tools restart` is a thin wrapper around `systemctl restart
// mcp-tools-web.service`. Kept as a separate verb (vs. `mcp-tools web
// --enable`) because the semantics differ: restart re-runs the unit
// without changing its enable state.
var restartModeOverride string

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Reinicia el servicio systemd mcp-tools-web.",
	RunE:  runRestart,
}

func init() {
	restartCmd.Flags().StringVar(&restartModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	mode, err := detectMode(restartModeOverride)
	if err != nil {
		return err
	}
	if mode == systemd.ModeNone {
		return fmt.Errorf("restart: systemd no disponible en este host")
	}
	if err := systemd.Restart(mode); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web reiniciado")
	return nil
}
