package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

// mcp-tools web -- gestión del panel web.
//
// Sin flags: abre el navegador en http://<bind>:<port>/ (mismo
// comportamiento que `mcp-tools open web`).
//
// Flags:
//
//	--enable         habilita + arranca el servicio systemd
//	--disable        para + deshabilita el servicio systemd
//	--set-port N     reconfigura el puerto y reinicia si está activo
//	--restart        reinicia el servicio systemd (recarga el binario)
//	--status         imprime estado del servicio + journal
//	--mode user|system|auto  (default auto)
var (
	webEnable       bool
	webDisable      bool
	webSetPort      int
	webShowStatus   bool
	webRestart      bool
	webModeOverride string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Gestiona el panel web (browser, enable, disable, port, status).",
	Long: "Sin flags: abre el navegador en http://<bind>:<port>/ (default 0.0.0.0:8888). " +
		"--enable / --disable controlan el servicio systemd. " +
		"--set-port N reconfigura el puerto y reinicia. " +
		"--restart reinicia el servicio (útil tras make install). " +
		"--status muestra el estado y el journal.",
	RunE: runWeb,
}

func init() {
	webCmd.Flags().BoolVar(&webEnable, "enable", false, "habilita + arranca el servicio systemd")
	webCmd.Flags().BoolVar(&webDisable, "disable", false, "stop + deshabilita el servicio systemd")
	webCmd.Flags().IntVar(&webSetPort, "set-port", 0, "reconfigura el puerto del panel y reinicia si está activo")
	webCmd.Flags().BoolVar(&webShowStatus, "status", false, "muestra el estado del servicio + últimas líneas del journal")
	webCmd.Flags().BoolVar(&webRestart, "restart", false, "reinicia el servicio systemd (recarga el binario tras make install)")
	webCmd.Flags().StringVar(&webModeOverride, "mode", "", "user|system|auto (default auto)")
	rootCmd.AddCommand(webCmd)
}

// runWeb dispatches based on the active flag combination. The valid
// shapes are:
//
//	no flags            → openBrowser(port)
//	--enable            → systemd.Enable
//	--disable           → systemd.Disable
//	--set-port N        → systemd.SetPort + restart-if-active
//	--restart           → systemd.Restart
//	--status            → systemd.Status + journal tail
//
// Flag combinations are mutually exclusive; combining --enable with
// --disable, --restart, or any of them with --status, is an error.
func runWeb(cmd *cobra.Command, args []string) error {
	mode, err := systemd.DetectMode(parseModeOverride(webModeOverride))
	if err != nil {
		return err
	}

	if err := validateWebFlags(); err != nil {
		return err
	}

	switch {
	case webShowStatus:
		return runWebStatus(mode)
	case webEnable:
		return runWebEnable(mode)
	case webDisable:
		return runWebDisable(mode)
	case webRestart:
		return runWebRestart(mode)
	case webSetPort > 0:
		return runWebSetPort(mode, webSetPort)
	default:
		return runWebOpen(mode)
	}
}

// runWebOpen is the default action: launch the browser on the unit's
// port (falling back to DefaultPort when the unit isn't loaded yet).
func runWebOpen(mode systemd.Mode) error {
	port := systemd.CurrentPort(mode)
	if port == 0 {
		port = DefaultPort
	}
	bind := systemd.CurrentBind(mode)
	if bind == "" {
		bind = DefaultBind
	}
	url := webURL(bind, port)
	if err := openBrowser(url); err != nil {
		fmt.Fprintf(os.Stderr, "no pude abrir el navegador (%v). Abre %s manualmente.\n", err, url)
		return err
	}
	fmt.Fprintf(os.Stdout, "%s abierto en tu navegador\n", url)
	return nil
}

func runWebEnable(mode systemd.Mode) error {
	if mode == systemd.ModeNone {
		return errNoSystemd("--enable")
	}
	if err := systemd.Enable(mode); err != nil {
		return fmt.Errorf("--enable: %w", err)
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web habilitado y arrancado")
	return nil
}

func runWebRestart(mode systemd.Mode) error {
	if mode == systemd.ModeNone {
		fmt.Fprintln(os.Stdout, "── mcp-tools-web no está instalado; skip restart")
		return nil
	}
	// DetectMode only reports whether systemd itself is reachable, not
	// whether THIS unit was ever installed — a host can have a live
	// user session with no mcp-tools-web.service on disk yet (e.g.
	// `make install` on a fresh machine, before the first --enable).
	// Treat a missing unit file the same as ModeNone: skip, don't error.
	unitPath, err := systemd.UnitPath(mode)
	if err != nil {
		return fmt.Errorf("--restart: %w", err)
	}
	if _, err := os.Stat(unitPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stdout, "── mcp-tools-web no está instalado; skip restart")
		return nil
	}
	if err := systemd.Restart(mode); err != nil {
		return fmt.Errorf("--restart: %w", err)
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web reiniciado")
	return nil
}

func runWebDisable(mode systemd.Mode) error {
	if mode == systemd.ModeNone {
		return errNoSystemd("--disable")
	}
	if err := systemd.Disable(mode); err != nil {
		return fmt.Errorf("--disable: %w", err)
	}
	fmt.Fprintln(os.Stdout, "── mcp-tools-web detenido y deshabilitado")
	return nil
}

// runWebSetPort re-renders the unit with a new port and restarts it if
// it was active. The bind is preserved (parsed from the current unit).
func runWebSetPort(mode systemd.Mode, port int) error {
	if err := validatePort(port); err != nil {
		return fmt.Errorf("--set-port: %w", err)
	}
	if name, hit := dangerousPorts[port]; hit {
		fmt.Fprintf(os.Stderr, "AVISO: puerto %d suele estar en uso por %s. Continúa bajo tu responsabilidad.\n", port, name)
	}
	if mode == systemd.ModeNone {
		return errNoSystemd("--set-port")
	}
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	envPath := filepath.Join(repoRoot(), ".env")
	bind, err := systemd.SetPort(mode, port, "", binPath, envPath)
	if err != nil {
		return fmt.Errorf("--set-port: %w", err)
	}
	url := webURL(bind, port)
	fmt.Fprintf(os.Stdout, "── mcp-tools-web reconfigurado: %s\n", url)
	fmt.Fprintf(os.Stdout, "  unit:    %s\n", unitPathOrEmpty(mode))
	fmt.Fprintf(os.Stdout, "  active:  %s\n", systemd.ActiveState(mode))
	fmt.Fprintf(os.Stdout, "  enabled: %s\n", systemd.EnabledState(mode))
	return nil
}

func runWebStatus(mode systemd.Mode) error {
	if mode == systemd.ModeNone {
		return errNoSystemd("--status")
	}
	port := systemd.CurrentPort(mode)
	bind := systemd.CurrentBind(mode)
	if port == 0 {
		port = DefaultPort
	}
	if bind == "" {
		bind = DefaultBind
	}
	fmt.Fprintf(os.Stdout, "systemd mode: %s\n", mode)
	fmt.Fprintf(os.Stdout, "unit:        %s\n", unitPathOrEmpty(mode))
	fmt.Fprintf(os.Stdout, "active:      %s\n", systemd.ActiveState(mode))
	fmt.Fprintf(os.Stdout, "enabled:     %s\n", systemd.EnabledState(mode))
	fmt.Fprintf(os.Stdout, "url:         %s\n", webURL(bind, port))
	tail, err := systemd.JournalTail(mode, 20)
	if err != nil {
		fmt.Fprintf(os.Stdout, "journal:     (no se pudo leer: %v)\n", err)
		return nil
	}
	fmt.Fprintf(os.Stdout, "── journal (últimas 20 líneas)\n")
	fmt.Fprintln(os.Stdout, tail)
	return nil
}

func unitPathOrEmpty(mode systemd.Mode) string {
	p, err := systemd.UnitPath(mode)
	if err != nil {
		return ""
	}
	return p
}

func errNoSystemd(flag string) error {
	return fmt.Errorf("%s: systemd no disponible en este host", flag)
}

// validateWebFlags rejects mutually-exclusive flag combinations.
// Exported via the test file in this package; production dispatcher
// (runWeb) uses the same helper.
func validateWebFlags() error {
	switch {
	case webRestart && (webEnable || webDisable || webSetPort > 0 || webShowStatus):
		return errMutexRestartOther
	case webEnable && webDisable:
		return errMutexEnableDisable
	case webEnable && webSetPort > 0:
		return errMutexEnablePort
	case webDisable && webSetPort > 0:
		return errMutexDisablePort
	case webShowStatus && (webEnable || webDisable || webSetPort > 0):
		return errMutexStatusFlags
	}
	return nil
}

// validatePort enforces the IANA valid port range.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return errInvalidPort
	}
	return nil
}

// webURL assembles the panel URL with IPv6-safe hostport formatting.
func webURL(bind string, port int) string {
	return "http://" + net.JoinHostPort(bind, strconv.Itoa(port)) + "/"
}

// Sentinel errors used by validateWebFlags. Defined here so the runWeb
// dispatcher returns the same value the tests assert against.
var (
	errMutexEnableDisable = errString("--enable y --disable son mutuamente excluyentes")
	errMutexEnablePort    = errString("--enable y --set-port son mutuamente excluyentes")
	errMutexDisablePort   = errString("--disable y --set-port son mutuamente excluyentes")
	errMutexStatusFlags   = errString("--status no se puede combinar con --enable/--disable/--set-port")
	errMutexRestartOther  = errString("--restart no se puede combinar con --enable, --disable, --set-port ni --status")
	errInvalidPort        = errString("puerto fuera de rango (1..65535)")
)

type errString string

func (e errString) Error() string { return string(e) }

// keep the errors import in this file (used by runWebOpen's fallthrough
// for the browser-launch failure).
var _ = errors.Is
