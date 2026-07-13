package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/orchestrator"
)

var instructionsDry bool

var instructionsCmd = &cobra.Command{
	Use:   "instructions",
	Short: "Genera y verifica RULES.md, AGENTS.md y CLAUDE.md desde instructions/.",
	Long: "Los tres ficheros de instrucciones del repo son artefactos generados desde instructions/ " +
		"(fuente canónica única; la tabla de routing vive ahí una sola vez). " +
		"`sync` los regenera; `check` falla si están desactualizados. " +
		"La distribución a clientes (imports, symlinks, bloques marcados) sigue siendo cosa del panel/`RunRules`, no de este comando.",
}

var instructionsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenera los artefactos (usa --dry para previsualizar).",
	RunE: func(cmd *cobra.Command, args []string) error {
		return orchestrator.SyncInstructions(instructionsDry, os.Stdout)
	},
}

var instructionsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Falla si algún artefacto generado no coincide con instructions/.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return orchestrator.CheckInstructions(os.Stdout)
	},
}

func init() {
	instructionsSyncCmd.Flags().BoolVar(&instructionsDry, "dry", false, "muestra qué cambiaría sin escribir")
	instructionsCmd.AddCommand(instructionsSyncCmd, instructionsCheckCmd)
	rootCmd.AddCommand(instructionsCmd)
}
