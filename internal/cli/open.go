package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

// `mcp-tools open web` is a thin alias for `mcp-tools web` (no flags).
// Kept for backward compatibility with existing scripts and shell history.
var openWebModeOverride string

var openCmd = &cobra.Command{
	Use:   "open web",
	Short: "Abre el navegador en el panel web (alias de `mcp-tools web`).",
	Long:  "Lee el puerto del unit systemd (si está activo) y abre http://<bind>:<port>/ (default 0.0.0.0:8888).",
	Args:  cobra.ExactArgs(1),
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openWebModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && args[0] != "web" {
		return fmt.Errorf("subcomando desconocido %q (solo 'web' está soportado)", args[0])
	}
	mode, err := systemd.DetectMode(parseModeOverride(openWebModeOverride))
	if err != nil {
		return err
	}
	return runWebOpen(mode)
}