package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

var openWebModeOverride string

var openCmd = &cobra.Command{
	Use:   "open web",
	Short: "Abre el navegador en el panel web.",
	Long:  "Lee el puerto del unit systemd (si está activo) y abre http://127.0.0.1:<port>/.",
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openWebModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(openCmd)
}

// portFromUnit returns the --port argument baked into the systemd unit's
// ExecStart line, or 0 if the unit isn't loaded.
func portFromUnit(mode systemd.Mode) int {
	prefix := systemd.SystemctlPrefix(mode)
	out, err := exec.Command("systemctl", append(prefix, "show", "mcp-tools-web.service", "-p", "ExecStart")...).CombinedOutput()
	if err != nil {
		return 0
	}
	// ExecStart= shows as "ExecStart={ path argv0 ; ... }"; pull the first
	// digit-pair after "--port ".
	re := regexp.MustCompile(`--port\s+(\d+)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) < 2 {
		return 0
	}
	var p int
	fmt.Sscanf(m[1], "%d", &p)
	return p
}

func runOpen(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && args[0] != "web" {
		return fmt.Errorf("subcomando desconocido %q (solo 'web' está soportado)", args[0])
	}
	port := 8080
	mode, _ := systemd.DetectMode(parseModeOverride(openWebModeOverride))
	if mode != systemd.ModeNone {
		if p := portFromUnit(mode); p > 0 {
			port = p
		}
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	if err := openBrowser(url); err != nil {
		fmt.Fprintf(os.Stderr, "no pude abrir el navegador (%v). Abre %s manualmente.\n", err, url)
		return err
	}
	fmt.Fprintf(os.Stdout, "%s abierto en tu navegador\n", url)
	return nil
}

// shared errNoBrowser is here so tests can stub.
var errNoBrowser = errors.New("no browser launcher found")