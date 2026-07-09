package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-tools",
	Short: "Panel de administración web auto-hospedado para tu stack MCP.",
	Long: "mcp-tools es un panel de administración web auto-hospedado para tu stack MCP. " +
		"`mcp-tools install` arranca el servicio systemd; `mcp-tools open web` lo abre en tu navegador; " +
		"`mcp-tools update` actualiza el binario; `mcp-tools version` muestra versión.",
	Version:       fmt.Sprintf("%s (%s, %s)", version.Version, version.Commit, version.Date),
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command; exits with 1 on any error.
func Execute() {
	if err := config.EnsureRuntimePath(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not augment PATH: %v\n", err)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
