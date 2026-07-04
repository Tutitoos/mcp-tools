package cli

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/tui/selectmodel"
)

var selectModelCmd = &cobra.Command{
	Use:   "select-model",
	Short: "Selector TUI para cambiar el LLM o el embed de mem0",
	RunE: func(cmd *cobra.Command, args []string) error {
		envPath := config.EnvMem0File()
		if _, err := os.Stat(envPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf(".env.mem0 no existe en %s; créalo desde el bloque del README", envPath)
			}
			return err
		}
		current, err := config.LoadEnv(envPath)
		if err != nil {
			return err
		}

		p := tea.NewProgram(selectmodel.New(current))
		result, err := p.Run()
		if err != nil {
			return err
		}
		if m, ok := result.(selectmodel.Model); ok {
			if code := m.ExitCode(); code != 0 {
				os.Exit(code)
			}
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(selectModelCmd) }
