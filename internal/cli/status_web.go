package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"


)

// `mcp-tools status-web` is a thin alias for `mcp-tools web --status`.
// Kept for backward compatibility with existing scripts.
var statusWebModeOverride string

var statusWebCmd = &cobra.Command{
	Use:   "status-web",
	Short: "Estado del servicio systemd mcp-tools-web (alias de `mcp-tools web --status`).",
	RunE:  runStatusWeb,
}

func init() {
	statusWebCmd.Flags().StringVar(&statusWebModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(statusWebCmd)
}

func runStatusWeb(cmd *cobra.Command, args []string) error {
	mode, err := detectMode(statusWebModeOverride)
	if err != nil {
		return err
	}
	if err := runWebStatus(mode); err != nil {
		if errors.Is(err, errNoSystemd("--status")) {
			return fmt.Errorf("status-web: systemd no disponible en este host")
		}
		return err
	}
	return nil
}

// keep fmt / os reference for any future expansion; the imports are
// also used by sibling files in this package.
var _ = fmt.Sprint
var _ = os.Stdout