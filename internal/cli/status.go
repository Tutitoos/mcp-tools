package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

var statusTable bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Estado de todos los componentes (JSON por default; --table para ANSI).",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusTable, "table", false, "renderiza una tabla ANSI en lugar de JSON")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("state.json: %w", err)
	}
	reg := tools.Registry()
	rows := make([]statusRow, 0, len(reg))
	for _, t := range reg {
		row := statusRow{Key: t.Key, Selected: st.Has(t.Key)}
		if t.Status != nil {
			p, err := t.Status()
			if err != nil {
				row.Extra = map[string]any{"error": err.Error()}
			} else {
				row.Payload = p
			}
		}
		rows = append(rows, row)
	}
	if statusTable {
		return renderTable(rows, os.Stdout)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

type statusRow struct {
	Key      string              `json:"key"`
	Selected bool                `json:"selected"`
	Payload  tools.StatusPayload `json:"payload,omitempty"`
	Extra    map[string]any      `json:"extra,omitempty"`
}

func renderTable(rows []statusRow, out io.Writer) error {
	head := fmt.Sprintf("%-20s  %-9s  %-10s  %s\n", "KEY", "SELECTED", "INSTALLED", "VERSION")
	fmt.Fprint(out, head)
	fmt.Fprintln(out, strings.Repeat("─", 60))
	for _, r := range rows {
		sel := "×"
		if r.Selected {
			sel = "✔"
		}
		ins := "×"
		if r.Payload.Installed {
			ins = "✔"
		}
		version := r.Payload.Version
		if len(version) > 30 {
			version = version[:27] + "…"
		}
		fmt.Fprintf(out, "%-20s  %-9s  %-10s  %s\n", r.Key, sel, ins, version)
	}
	return nil
}
