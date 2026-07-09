package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// `mcp-tools stop` is a thin alias for `mcp-tools web --disable`. Kept
// for backward compatibility with existing scripts.
var stopModeOverride string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Detiene + deshabilita el servicio systemd (alias de `mcp-tools web --disable`).",
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().StringVar(&stopModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	mode, err := detectModeOrNone(stopModeOverride)
	if err != nil {
		return err
	}
	if err := runWebDisable(mode); err != nil {
		// keep the historical message wording for backward compat
		if errors.Is(err, errNoSystemd("--disable")) {
			return fmt.Errorf("stop: systemd no disponible en este host")
		}
		return err
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web detenido")
	return nil
}

func detectModeOrNone(override string) (modeValue, error) {
	m, err := detectMode(override)
	if err != nil {
		return "", err
	}
	return m, nil
}
