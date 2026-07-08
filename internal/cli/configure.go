package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var configureDry bool

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Reabre el multi-select TUI para añadir/quitar componentes.",
	RunE:  runConfigure,
}

func init() {
	configureCmd.Flags().BoolVar(&configureDry, "dry", false, "no ejecuta comandos; solo muestra el diff")
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	if len(st.Selected) == 0 {
		return errors.New("state ausente — corre 'mcp-tools install' primero")
	}

	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }
	if err := runBootstrap(configureDry, logf); err != nil {
		return err
	}

	avail := availableTools()
	pre := map[string]bool{}
	for _, k := range st.Selected {
		pre[k] = true
	}
	model, err := runToolSelect(avail, pre)
	if err != nil {
		return err
	}
	if !model.Confirmed() {
		return errors.New("cancelado por el user")
	}
	newSelected := model.Selected()

	toAdd := diffKeys(newSelected, st.Selected)
	toRemove := diffKeys(st.Selected, newSelected)

	// Reverse-deps: uninstall dependents first.
	toRemoveSorted, err := tools.TopoSort(toRemove)
	if err != nil {
		return err
	}
	slices.Reverse(toRemoveSorted)

	toAddSorted, err := tools.TopoSort(toAdd)
	if err != nil {
		return err
	}

	rmSudo, rmTui, rmInter := partitionByStdio(toRemoveSorted)
	addSudo, addTui, addInter := partitionByStdio(toAddSorted)
	// Uninstall: sudo primero (revertir cambios de sistema), TUI en medio,
	// interactive al final (unregister plugin). Install: mismo orden.
	if err := runInlineTools("uninstall", rmSudo, configureDry, logf); err != nil {
		return err
	}
	if err := runToolSteps("uninstall", rmTui, configureDry); err != nil {
		return err
	}
	if err := runInlineTools("uninstall", rmInter, configureDry, logf); err != nil {
		return err
	}
	if err := runInlineTools("install", addSudo, configureDry, logf); err != nil {
		return err
	}
	if err := runToolSteps("install", addTui, configureDry); err != nil {
		return err
	}
	if err := runInlineTools("install", addInter, configureDry, logf); err != nil {
		return err
	}

	stNew := state.State{Selected: newSelected, Versions: st.Versions}
	if err := RunMcpConfig(configureDry, stNew, os.Stdout); err != nil {
		return fmt.Errorf("mcp-config: %w", err)
	}

	// 5. Skills + rules (idempotentes; sirven a los 3 clientes). Errores
	//    se acumulan y se reportan al final — los tools ya están instalados
	//    y mcp-config ya corrió; un fallo aquí no debe revertir state.
	var buf bytes.Buffer
	var skErrs []error
	if err := RunSkills(configureDry, &buf); err != nil {
		skErrs = append(skErrs, fmt.Errorf("skills: %w", err))
	}
	if err := RunRules(configureDry, &buf); err != nil {
		skErrs = append(skErrs, fmt.Errorf("rules: %w", err))
	}
	if s := strings.TrimRight(buf.String(), "\n"); s != "" {
		fmt.Fprintln(os.Stdout, s)
	}

	// 6. Persistir state ANTES de un eventual return por error (H31): el
	//    state debe reflejar la nueva selección aunque skills/rules fallen.
	if configureDry {
		fmt.Fprintln(os.Stdout, "SKIP (dry) — no toca state.json")
		if len(skErrs) > 0 {
			return errors.Join(skErrs...)
		}
		return nil
	}
	stNew.Versions = collectVersions(newSelected)
	if err := stNew.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if len(skErrs) > 0 {
		return errors.Join(skErrs...)
	}
	unchanged := len(newSelected) - len(toAdd)
	fmt.Fprintf(os.Stdout, "── configure completo — +%d añadidos, -%d eliminados, =%d sin cambios\n", len(toAdd), len(toRemove), unchanged)
	return nil
}

// diffKeys returns items in `a` that are not in `b`, preserving `a` order.
func diffKeys(a, b []string) []string {
	inB := map[string]bool{}
	for _, k := range b {
		inB[k] = true
	}
	var out []string
	for _, k := range a {
		if !inB[k] {
			out = append(out, k)
		}
	}
	return out
}
