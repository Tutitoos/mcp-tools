package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

var (
	installPort         int
	installBind         string
	installModeOverride string

	installNoOpenBrowser bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Arranca el panel web como servicio systemd.",
	Long: "Escribe el unit file mcp-tools-web.service, lo habilita y lo inicia. " +
		"Abre el navegador en http://<bind>:<port>/ cuando termina.",
	RunE: runInstall,
}

func init() {
	installCmd.Flags().IntVar(&installPort, "port", 0, "puerto del panel (default 8888)")
	installCmd.Flags().StringVar(&installBind, "bind", "", "dirección de escucha (default 127.0.0.1, loopback-only; usa 0.0.0.0 para exponer el panel a la LAN)")
	installCmd.Flags().StringVar(&installModeOverride, "mode", "", "user|system|auto (default auto)")

	installCmd.Flags().BoolVar(&installNoOpenBrowser, "no-open", false, "no abre el navegador al terminar")
	rootCmd.AddCommand(installCmd)
}

var dangerousPorts = map[int]string{
	22: "SSH", 25: "SMTP", 80: "HTTP", 443: "HTTPS",
	3306: "MySQL", 5432: "PostgreSQL", 6379: "Redis", 27017: "MongoDB",
	8000: "dev-server",
}

func runInstall(cmd *cobra.Command, args []string) error {
	port, err := resolvePort(installPort)
	if err != nil {
		return err
	}
	if name, hit := dangerousPorts[port]; hit {
		fmt.Fprintf(os.Stderr, "AVISO: puerto %d suele estar en uso por %s. Continúa bajo tu responsabilidad.\n", port, name)
	}

	bind, err := resolveBind(installBind)
	if err != nil {
		return err
	}
	if bind == "" {
		bind = DefaultBind
	}

	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	envPath := filepath.Join(config.RepoRoot(), ".env")

	override := systemd.Mode("")
	switch strings.ToLower(installModeOverride) {
	case "user":
		override = systemd.ModeUser
	case "system":
		override = systemd.ModeSystem
	}
	mode, err := systemd.DetectMode(override)
	if err != nil {
		return err
	}

	if mode == systemd.ModeNone {
		return printNoSystemdFallback(port, bind)
	}

	if err := systemd.Install(mode, port, bind, binPath, envPath); err != nil {
		_, _ = systemd.Status(mode)
		return fmt.Errorf("systemd install: %w", err)
	}

	hostport := net.JoinHostPort(bind, strconv.Itoa(port))
	url := "http://" + hostport + "/"
	switch {
	case installNoOpenBrowser:
		// user opted out; nothing to print — the URL is shown in the summary below
	case !hasBrowserLauncher():
		fmt.Fprintf(os.Stdout, "── sin navegador local: abre %s manualmente\n", url)
	default:
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "AVISO: no pude abrir el navegador (%v). Abre %s manualmente.\n", err, url)
		}
	}

	unitPath, _ := systemd.UnitPath(mode)
	fmt.Fprintf(os.Stdout, "── mcp-tools-web activo\n")
	fmt.Fprintf(os.Stdout, "  URL:    %s\n", url)
	fmt.Fprintf(os.Stdout, "  Unit:   %s\n", unitPath)
	if bind == "0.0.0.0" || bind == "::" {
		fmt.Fprintf(os.Stdout, "  AVISO:  bind=%s expone el panel en toda la red. Usa 127.0.0.1 si quieres loopback-only.\n", bind)
	}
	journalHint := "journalctl -u mcp-tools-web.service -f"
	if mode == systemd.ModeUser {
		journalHint = "journalctl --user -u mcp-tools-web.service -f"
	}
	fmt.Fprintf(os.Stdout, "  Logs:   %s\n", journalHint)
	fmt.Fprintf(os.Stdout, "  Stop:   mcp-tools stop\n")
	return nil
}

func resolvePort(flagVal int) (int, error) {
	if flagVal > 0 {
		if flagVal > 65535 {
			return 0, fmt.Errorf("puerto %d fuera de rango (1..65535)", flagVal)
		}
		return flagVal, nil
	}
	if v := os.Getenv("MCP_TOOLS_WEB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 65535 {
			return p, nil
		}
		return 0, fmt.Errorf("MCP_TOOLS_WEB_PORT=%q no es un puerto válido", v)
	}
	for {
		fmt.Fprintf(os.Stderr, "Puerto del panel [%d]: ", DefaultPort)
		line, err := readPromptLine()
		if err != nil {
			return DefaultPort, nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return DefaultPort, nil
		}
		p, err := strconv.Atoi(line)
		if err != nil || p < 1 || p > 65535 {
			fmt.Fprintf(os.Stderr, "  puerto inválido; introduce un número entre 1 y 65535\n")
			continue
		}
		return p, nil
	}
}

// resolveBind resolves the panel bind address: explicit --bind flag,
// then MCP_TOOLS_BIND (process env, then repo .env — the documented
// contract), then loopback.
func resolveBind(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if v := config.BindFromEnv(); v != "" {
		return v, nil
	}
	return DefaultBind, nil
}

func openBrowser(url string) error {
	for _, bin := range []string{"xdg-open", "open", "wslview"} {
		if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		cmd := exec.Command(bin, url)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	return errors.New("no browser launcher found (xdg-open/open/wslview)")
}

// hasBrowserLauncher reports whether any of the browser launchers
// openBrowser would try is on PATH. Used by the web/install CLI to
// short-circuit the browser attempt on headless hosts and print an
// informational URL instead of erroring out.
func hasBrowserLauncher() bool {
	for _, bin := range []string{"xdg-open", "open", "wslview"} {
		if _, err := exec.LookPath(bin); err == nil {
			return true
		}
	}
	return false
}

func printNoSystemdFallback(port int, bind string) error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	envPath := filepath.Join(config.RepoRoot(), ".env")
	hostport := net.JoinHostPort(bind, strconv.Itoa(port))
	fmt.Fprintf(os.Stdout, "systemd no disponible en este host.\n")
	fmt.Fprintf(os.Stdout, "para arrancar el panel en foreground:\n")
	fmt.Fprintf(os.Stdout, "  %s serve --port %d --bind %s\n", binPath, port, bind)
	fmt.Fprintf(os.Stdout, "o en segundo plano:\n")
	fmt.Fprintf(os.Stdout, "  nohup %s serve --port %d --bind %s >/tmp/mcp-tools-web.log 2>&1 &\n", binPath, port, bind)
	fmt.Fprintf(os.Stdout, "URL:   http://%s/\n", hostport)
	fmt.Fprintf(os.Stdout, ".env:  %s\n", envPath)
	return nil
}
