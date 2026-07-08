package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

var statusWebModeOverride string

var statusWebCmd = &cobra.Command{
	Use:   "status-web",
	Short: "Estado del servicio systemd mcp-tools-web + últimas líneas del journal.",
	RunE:  runStatusWeb,
}

func init() {
	statusWebCmd.Flags().StringVar(&statusWebModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(statusWebCmd)
}

func runStatusWeb(cmd *cobra.Command, args []string) error {
	mode, err := systemd.DetectMode(parseModeOverride(statusWebModeOverride))
	if err != nil {
		return err
	}
	if mode == systemd.ModeNone {
		return fmt.Errorf("status-web: systemd no disponible en este host")
	}
	state, _ := systemd.Status(mode)
	fmt.Fprintf(os.Stdout, "systemd mode: %s\n", mode)
	fmt.Fprintf(os.Stdout, "state:       %s\n", state)
	unitPath, _ := systemd.UnitPath(mode)
	fmt.Fprintf(os.Stdout, "unit:        %s\n", unitPath)

	// Last 20 journal lines.
	prefix := systemd.SystemctlPrefix(mode)
	journalArgs := append(prefix, "log", "--no-pager", "-n", "20", "-u", "mcp-tools-web.service")
	out, err := exec.Command("journalctl", journalArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "journal:     (no se pudo leer: %v)\n", err)
		return nil
	}
	fmt.Fprintf(os.Stdout, "── journal (últimas 20 líneas)\n")
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}