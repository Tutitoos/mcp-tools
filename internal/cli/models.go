package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/tui/modelselect"
	"github.com/Tutitoos/mcp-tools/internal/tui/selectmodel"
)

var (
	modelsDry bool
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Multi-select TUI de modelos Ollama (instalados + curados).",
	Long: "Marca/desmarca modelos; al confirmar aplica el diff (ollama pull/rm). " +
		"Sub-comandos no-interactivos: list, pull <tag>, rm <tag>.",
	RunE: runModelsTUI,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Imprime los modelos instalados como JSON",
	RunE:  runModelsList,
}

var modelsPullCmd = &cobra.Command{
	Use:   "pull <tag>",
	Short: "Descarga un modelo (docker exec mcp-tools-ollama ollama pull <tag>)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOllamaExec("pull", args[0])
	},
}

var modelsRmCmd = &cobra.Command{
	Use:   "rm <tag>",
	Short: "Borra un modelo (docker exec mcp-tools-ollama ollama rm <tag>)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOllamaExec("rm", args[0])
	},
}

func init() {
	modelsCmd.Flags().BoolVar(&modelsDry, "dry", false, "no ejecuta pull/rm; solo muestra el diff")
	modelsCmd.AddCommand(modelsListCmd, modelsPullCmd, modelsRmCmd)
	rootCmd.AddCommand(modelsCmd)
}

func runModelsTUI(cmd *cobra.Command, args []string) error {
	if err := ensureOllamaRunning(); err != nil {
		return err
	}
	installed, err := listInstalledModels()
	if err != nil {
		return err
	}
	items := buildItems(installed)

	p := tea.NewProgram(modelselect.New(items))
	res, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := res.(modelselect.Model)
	if !ok {
		return errors.New("modelselect: modelo inesperado")
	}
	if !m.Confirmed() {
		return errors.New("cancelado por el user")
	}
	newSelected := m.Selected()

	previouslyInstalled := installedTags(installed)
	toPull := diffKeys(newSelected, previouslyInstalled)
	toRemove := diffKeys(previouslyInstalled, newSelected)

	if modelsDry {
		for _, t := range toPull {
			fmt.Fprintln(os.Stdout, "$ docker exec mcp-tools-ollama ollama pull "+t)
		}
		for _, t := range toRemove {
			fmt.Fprintln(os.Stdout, "$ docker exec mcp-tools-ollama ollama rm "+t)
		}
		return nil
	}

	var errs []error
	for _, t := range toRemove {
		fmt.Fprintln(os.Stdout, "· rm "+t)
		if err := runOllamaExec("rm", t); err != nil {
			errs = append(errs, fmt.Errorf("rm %s: %w", t, err))
		}
	}
	for _, t := range toPull {
		fmt.Fprintln(os.Stdout, "· pull "+t)
		if err := runOllamaExec("pull", t); err != nil {
			errs = append(errs, fmt.Errorf("pull %s: %w", t, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func runModelsList(cmd *cobra.Command, args []string) error {
	if err := ensureOllamaRunning(); err != nil {
		return err
	}
	items, err := listInstalledModels()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

// installedModel is one row from `ollama list`.
type installedModel struct {
	Tag      string `json:"tag"`
	Size     string `json:"size,omitempty"`
	Modified string `json:"modified,omitempty"`
}

func listInstalledModels() ([]installedModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ollama list: %w", err)
	}
	return parseOllamaList(string(out)), nil
}

// parseOllamaList reads the tabular `ollama list` output. Header line skipped;
// columns are NAME, ID, SIZE (2 tokens), MODIFIED (rest).
func parseOllamaList(s string) []installedModel {
	scanner := bufio.NewScanner(strings.NewReader(s))
	first := true
	var out []installedModel
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		m := installedModel{Tag: fields[0]}
		if len(fields) >= 4 {
			m.Size = fields[2] + " " + fields[3]
		}
		if len(fields) > 4 {
			m.Modified = strings.Join(fields[4:], " ")
		}
		out = append(out, m)
	}
	return out
}

func installedTags(items []installedModel) []string {
	out := make([]string, len(items))
	for i, m := range items {
		out[i] = m.Tag
	}
	return out
}

// buildItems merges installed + curated. Curated tags land under LLM/Embed;
// installed-but-not-curated land under "Otros instalados". Tags are matched
// by HasPrefix (curated + ":") so a curated entry like `bge-m3` matches the
// installed `bge-m3:f32` or `bge-m3:latest` (no `:latest` normalization).
func buildItems(installed []installedModel) []modelselect.Item {
	installedSet := map[string]installedModel{}
	for _, m := range installed {
		installedSet[m.Tag] = m
	}

	var items []modelselect.Item
	seen := map[string]bool{}
	for _, m := range selectmodel.LLMModels {
		tag := m.Value
		items = append(items, modelselect.Item{
			Tag:       tag,
			Label:     m.Label,
			Section:   "LLM",
			Installed: fuzzyInstalled(tag, installedSet),
			Size:      sizeOf(tag, installedSet),
		})
		seen[tag] = true
	}
	for _, m := range selectmodel.EmbedModels {
		tag := m.Value
		items = append(items, modelselect.Item{
			Tag:       tag,
			Label:     m.Label,
			Section:   "Embed",
			Installed: fuzzyInstalled(tag, installedSet),
			Size:      sizeOf(tag, installedSet),
		})
		seen[tag] = true
	}
	tags := make([]string, 0, len(installedSet))
	for tag := range installedSet {
		if seen[tag] {
			continue
		}
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	for _, tag := range tags {
		m := installedSet[tag]
		items = append(items, modelselect.Item{
			Tag:       tag,
			Label:     tag,
			Section:   "Otros instalados",
			Installed: true,
			Size:      m.Size,
		})
	}
	return items
}

// fuzzyInstalled matches a curated tag against installed tags: exact match OR
// `installed` has the curated tag as a prefix followed by `:` / `@` (covers
// ollama naming `bge-m3:f32` against curated `bge-m3`).
func fuzzyInstalled(tag string, installed map[string]installedModel) bool {
	if _, ok := installed[tag]; ok {
		return true
	}
	prefix := tag + ":"
	for existing := range installed {
		if strings.HasPrefix(existing, prefix) || strings.HasPrefix(existing, tag+"@") {
			return true
		}
	}
	return false
}

func sizeOf(tag string, installed map[string]installedModel) string {
	if m, ok := installed[tag]; ok {
		return m.Size
	}
	prefix := tag + ":"
	for existing, m := range installed {
		if strings.HasPrefix(existing, prefix) || strings.HasPrefix(existing, tag+"@") {
			return m.Size
		}
	}
	return ""
}

func ensureOllamaRunning() error {
	out, err := exec.Command("docker", "container", "inspect", "-f", "{{.State.Status}}", "mcp-tools-ollama").Output()
	if err != nil {
		return errors.New("ollama no está corriendo; arranca con 'mcp-tools up'")
	}
	if strings.TrimSpace(string(out)) != "running" {
		return errors.New("ollama no está corriendo; arranca con 'mcp-tools up'")
	}
	return nil
}

func runOllamaExec(subcmd, tag string) error {
	cmd := exec.Command("docker", "exec", "mcp-tools-ollama", "ollama", subcmd, tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
