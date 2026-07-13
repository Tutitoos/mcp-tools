package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/web"
)

var (
	servePort     int
	serveBind     string
	serveUnixSock string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca la API + el panel web embebido.",
	Long:  "Bind por defecto: 127.0.0.1:8888 (loopback; usa --bind 0.0.0.0 para exponer a la LAN). Flags --port/--bind/--unix-socket. Maneja SIGINT/SIGTERM con 10s de gracia.",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", DefaultPort, "puerto TCP")
	serveCmd.Flags().StringVar(&serveBind, "bind", DefaultBind, "dirección TCP")
	serveCmd.Flags().StringVar(&serveUnixSock, "unix-socket", "", "path al unix socket (alternativa a --port/--bind)")
	rootCmd.AddCommand(serveCmd)
}

// runServe is the systemd unit's ExecStart target. It boots the API +
// embedded SPA and blocks until SIGINT / SIGTERM.
func runServe(cmd *cobra.Command, args []string) error {
	addr, listener, err := bindListener()
	if err != nil {
		return err
	}
	defer listener.Close()
	srv := web.Listen()

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "mcp-tools web listening on %s\n", addr)
		if serveUnixSock != "" {
			if err := os.Chmod(serveUnixSock, 0o660); err != nil {
				fmt.Fprintf(os.Stderr, "warn: chmod unix socket: %v\n", err)
			}
		}
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stdout, "\nreceived %s, shutting down…\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

// bindListener resolves --unix-socket / --port/--bind into a ready
// net.Listener.
func bindListener() (string, net.Listener, error) {
	if serveUnixSock != "" {
		if err := os.Remove(serveUnixSock); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", nil, fmt.Errorf("unix socket: %w", err)
		}
		ln, err := net.Listen("unix", serveUnixSock)
		if err != nil {
			return "", nil, fmt.Errorf("listen unix %s: %w", serveUnixSock, err)
		}

		return "unix://" + serveUnixSock, ln, nil
	}
	addr := net.JoinHostPort(serveBind, strconv.Itoa(servePort))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, fmt.Errorf("listen tcp %s: %w", addr, err)
	}
	return addr, ln, nil
}
