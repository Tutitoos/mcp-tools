package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var (
	uninstallDry   bool
	uninstallForce bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <tool>",
	Short: "Desinstala un componente y lo quita del state.",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallDry, "dry", false, "no ejecuta comandos; solo muestra qué haría")
	uninstallCmd.Flags().BoolVar(&uninstallForce, "force", false, "ignora el reverse-dep check y marca los dependents como broken en state")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	key := args[0]
	t, err := tools.Get(key)
	if err != nil {
		return err
	}
	st, err := state.Load()
	if err != nil {
		return err
	}
	if !st.Has(key) {
		return fmt.Errorf("%q no está en el state; nada que desinstalar", key)
	}
	// Reverse-dep check: any other selected tool that requires this one blocks
	// removal unless --force.
	blocker := ""
	for _, other := range st.Selected {
		if other == key {
			continue
		}
		ot, err := tools.Get(other)
		if err != nil {
			continue
		}
		for _, dep := range ot.Deps {
			if dep == key {
				blocker = other
				break
			}
		}
		if blocker != "" {
			break
		}
	}
	if blocker != "" && !uninstallForce {
		return fmt.Errorf("no se puede desinstalar %s: %s lo requiere. Usa --force para saltarlo", key, blocker)
	}

	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }
	if err := t.Uninstall(uninstallDry, logf); err != nil {
		return fmt.Errorf("uninstall %s: %w", key, err)
	}

	if uninstallDry {
		fmt.Fprintf(os.Stdout, "SKIP (dry) — no toca state.json\n")
		return nil
	}
	st.Selected = slices.DeleteFunc(st.Selected, func(k string) bool { return k == key })
	delete(st.Versions, key)
	if err := st.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	fmt.Fprintf(os.Stdout, "OK %s desinstalado\n", key)
	if blocker != "" && uninstallForce {
		fmt.Fprintf(os.Stdout, "WARN %s sigue seleccionado pero %s ha desaparecido — %s puede quedar roto\n", blocker, key, blocker)
	}
	return nil
}
