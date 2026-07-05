package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// makeToolAction produces a RunE that invokes Install/Upgrade/Uninstall on the
// registered tool with a stdout-oriented logger. `verb` selects which closure.
func makeToolAction(key, verb string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		t, err := tools.Get(key)
		if err != nil {
			return err
		}
		log := func(s string) { fmt.Fprintln(os.Stdout, s) }
		switch verb {
		case "install":
			return t.Install(false, log)
		case "upgrade":
			if t.Upgrade == nil {
				return fmt.Errorf("%s: upgrade no soportado (correr install a mano)", key)
			}
			return t.Upgrade(false, log)
		case "uninstall":
			return t.Uninstall(false, log)
		}
		return fmt.Errorf("verb desconocido: %s", verb)
	}
}

// makeToolStatus produces a RunE that prints Status() as pretty JSON.
func makeToolStatus(key string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		t, err := tools.Get(key)
		if err != nil {
			return err
		}
		if t.Status == nil {
			return fmt.Errorf("%s: sin status disponible", key)
		}
		payload, err := t.Status()
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
}
