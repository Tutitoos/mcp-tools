package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Configura el context cap operativo de OMP (compaction threshold)",
	Long: "Envuelve 'omp config' para leer/escribir las claves compaction.* y limitar\n" +
		"cuánto contexto acumula OMP antes de compactar la sesión. Requiere el\n" +
		"binario 'omp' en PATH — mcp-tools no gestiona la instalación de omp.",
	RunE: runTokensShow,
}

func init() {
	tokensCmd.AddCommand(&cobra.Command{
		Use:   "set <tokens>",
		Short: "Establece compaction.thresholdTokens y activa compaction",
		Args:  cobra.ExactArgs(1),
		RunE:  runTokensSet,
	})
	rootCmd.AddCommand(tokensCmd)
}

// Verified against `omp config list --json` (schema real, no aspiracional).
var tokensKeys = []string{
	"compaction.enabled",
	"compaction.thresholdTokens",
	"compaction.reserveTokens",
	"compaction.keepRecentTokens",
	"compaction.strategy",
	"compaction.midTurnEnabled",
	"contextPromotion.enabled",
}

func runTokensShow(cmd *cobra.Command, _ []string) error {
	if err := ensureOMP(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "── tokens (OMP compaction config)")
	for _, k := range tokensKeys {
		val, err := ompConfigGet(k)
		if err != nil {
			fmt.Fprintf(os.Stdout, "  ERR %s: %v\n", k, err)
			continue
		}
		fmt.Fprintf(os.Stdout, "  %s = %s\n", k, val)
	}
	return nil
}

func runTokensSet(cmd *cobra.Command, args []string) error {
	if err := ensureOMP(); err != nil {
		return err
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		return fmt.Errorf("tokens debe ser un entero positivo (recibido: %q)", args[0])
	}
	updates := [][2]string{
		{"compaction.enabled", "true"},
		{"compaction.thresholdTokens", strconv.Itoa(n)},
		{"compaction.midTurnEnabled", "true"},
		{"contextPromotion.enabled", "false"},
	}
	fmt.Fprintln(os.Stdout, "── tokens set")
	for _, kv := range updates {
		if err := ompConfigSet(kv[0], kv[1]); err != nil {
			return fmt.Errorf("%s: %w", kv[0], err)
		}
		fmt.Fprintf(os.Stdout, "  OK %s = %s\n", kv[0], kv[1])
	}
	fmt.Fprintf(os.Stdout, "── tokens completo — cap operativo = %d tokens · reinicia sesión OMP\n", n)
	return nil
}

func ensureOMP() error {
	if _, err := exec.LookPath("omp"); err != nil {
		return fmt.Errorf("omp no está en PATH — instálalo primero (mcp-tools no gestiona omp)")
	}
	return nil
}

func ompConfigGet(key string) (string, error) {
	out, err := exec.Command("omp", "config", "get", key, "--json").Output()
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		return string(out), nil
	}
	if v, ok := payload["value"]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	return "(unset)", nil
}

func ompConfigSet(key, value string) error {
	cmd := exec.Command("omp", "config", "set", key, value)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
