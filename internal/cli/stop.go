package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

var stopModeOverride string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Detiene el servicio systemd mcp-tools-web.",
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().StringVar(&stopModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	mode, err := systemd.DetectMode(parseModeOverride(stopModeOverride))
	if err != nil {
		return err
	}
	if mode == systemd.ModeNone {
		return fmt.Errorf("stop: systemd no disponible en este host")
	}
	if err := systemd.Stop(mode); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web detenido")
	return nil
}

func parseModeOverride(s string) systemd.Mode {
	switch s {
	case "user":
		return systemd.ModeUser
	case "system":
		return systemd.ModeSystem
	}
	return systemd.Mode("")
}