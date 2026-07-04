// Package selectmodel implements the mem0 model-selector TUI.
package selectmodel

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/tui/theme"
)

type ModelOption struct {
	Label string
	Value string
}

// Literals mirror scripts/installer/select-model.tsx lines 26-45.
var LLMModels = []ModelOption{
	{"qwen2.5:7b        7B    multilingüe, tool calling maduro (default)", "qwen2.5:7b"},
	{"qwen3:8b          8B    siguiente gen de qwen, mejor calidad", "qwen3:8b"},
	{"mistral-nemo:12b  12B   Mistral+NVIDIA, contexto 128k", "mistral-nemo:12b"},
	{"llama3.1:8b       8B    Meta, menos multilingüe que qwen", "llama3.1:8b"},
	{"mistral:7b        7B    function calling desde v0.3", "mistral:7b"},
	{"qwen3:4b          4B    compacto dentro de qwen3", "qwen3:4b"},
	{"qwen2.5:3b        3B    ligero dentro de qwen2.5", "qwen2.5:3b"},
	{"llama3.2:3b       3B    Meta ligero", "llama3.2:3b"},
	{"granite3.1-moe:3b 3B    IBM MoE, punchea por encima", "granite3.1-moe:3b"},
	{"smollm2:1.7b      1.7B  mínimo viable, solo probar", "smollm2:1.7b"},
}

var EmbedModels = []ModelOption{
	{"bge-m3                  1024 dims, multilingüe 100+ (default)", "bge-m3"},
	{"mxbai-embed-large       mixedbread.ai (verificar dim con ollama show)", "mxbai-embed-large"},
	{"snowflake-arctic-embed  familia Snowflake, varias variantes", "snowflake-arctic-embed"},
	{"nomic-embed-text        contexto largo (verificar dim)", "nomic-embed-text"},
	{"all-minilm              mínimo (22m/33m params), solo pruebas", "all-minilm"},
}

var KindOptions = []ModelOption{
	{"LLM     (MEM0_LLM_MODEL)   — el que extrae memorias", "llm"},
	{"Embed   (MEM0_EMBED_MODEL) — vectores en qdrant", "embed"},
}

type phase int

const (
	phaseChooseKind phase = iota
	phaseChooseModel
	phaseConfirm
	phasePulling
	phaseRestarting
	phaseDone
	phaseError
)

type cmdDoneMsg struct {
	err error
}

type Model struct {
	current   map[string]string
	phase     phase
	kindIdx   int
	modelIdx  int
	kind      string  // "llm" | "embed"
	selected  string
	err       error
	spinner   spinner.Model
	cancelled bool
}

func New(current map[string]string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Cyan
	return Model{
		current: current,
		phase:   phaseChooseKind,
		spinner: sp,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" {
			m.cancelled = true
			return m, tea.Quit
		}
		switch m.phase {
		case phaseChooseKind:
			return m.updateChooseKind(key)
		case phaseChooseModel:
			return m.updateChooseModel(key)
		case phaseConfirm:
			return m.updateConfirm(key)
		case phaseDone, phaseError:
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case cmdDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseError
			return m, nil
		}
		switch m.phase {
		case phasePulling:
			// Edit .env.mem0
			updates := map[string]string{m.envVar(): m.selected}
			if m.kind == "llm" && (strings.HasPrefix(m.selected, "qwen3") || strings.HasPrefix(m.selected, "deepseek-r1")) {
				updates["MEM0_OLLAMA_THINK"] = "false"
			}
			if err := config.UpdateEnv(config.EnvMem0File(), updates); err != nil {
				m.err = err
				m.phase = phaseError
				return m, nil
			}
			m.phase = phaseRestarting
			return m, tea.Batch(m.spinner.Tick, m.runRecreate())
		case phaseRestarting:
			m.phase = phaseDone
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateChooseKind(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.kindIdx > 0 {
			m.kindIdx--
		}
	case "down", "j":
		if m.kindIdx < len(KindOptions)-1 {
			m.kindIdx++
		}
	case "enter":
		m.kind = KindOptions[m.kindIdx].Value
		m.phase = phaseChooseModel
	}
	return m, nil
}

func (m Model) updateChooseModel(key string) (tea.Model, tea.Cmd) {
	list := m.modelList()
	switch key {
	case "up", "k":
		if m.modelIdx > 0 {
			m.modelIdx--
		}
	case "down", "j":
		if m.modelIdx < len(list)-1 {
			m.modelIdx++
		}
	case "esc":
		m.phase = phaseChooseKind
		m.modelIdx = 0
	case "enter":
		m.selected = list[m.modelIdx].Value
		m.phase = phaseConfirm
	}
	return m, nil
}

func (m Model) updateConfirm(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y", "enter":
		m.phase = phasePulling
		return m, tea.Batch(m.spinner.Tick, m.runPull())
	case "n", "N", "esc":
		m.cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) runPull() tea.Cmd {
	sel := m.selected
	return func() tea.Msg {
		err := exec.Command("docker", "exec", "mcp-tools-ollama", "ollama", "pull", sel).Run()
		return cmdDoneMsg{err: err}
	}
}

func (m Model) runRecreate() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "compose",
			"-f", "dockers/compose.yaml",
			"--env-file", ".env",
			"up", "-d", "--force-recreate", "mcp_tools_mem0",
		)
		cmd.Dir = config.RepoRoot()
		err := cmd.Run()
		return cmdDoneMsg{err: err}
	}
}

func (m Model) envVar() string {
	if m.kind == "llm" {
		return "MEM0_LLM_MODEL"
	}
	return "MEM0_EMBED_MODEL"
}

func (m Model) modelList() []ModelOption {
	if m.kind == "llm" {
		return LLMModels
	}
	return EmbedModels
}

func (m Model) currentValue() string { return m.current[m.envVar()] }

func (m Model) needsThinkFlag() bool {
	return m.kind == "llm" &&
		m.selected != "" &&
		(strings.HasPrefix(m.selected, "qwen3") || strings.HasPrefix(m.selected, "deepseek-r1"))
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(theme.Magenta.Render("mem0") + theme.Dim.Render("  cambiar modelo") + "\n")
	fmt.Fprintf(&b, "%s%s%s%s%s\n\n",
		theme.Dim.Render("actual · LLM="),
		theme.Cyan.Render(fallback(m.current["MEM0_LLM_MODEL"], "?")),
		theme.Dim.Render(" · Embed="),
		theme.Cyan.Render(fallback(m.current["MEM0_EMBED_MODEL"], "?")),
		"",
	)

	switch m.phase {
	case phaseChooseKind:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("¿Qué modelo cambiar?") + "\n\n")
		for i, opt := range KindOptions {
			b.WriteString(renderOption(opt.Label, i == m.kindIdx) + "\n")
		}
		b.WriteString("\n" + theme.Dim.Render("↑↓ navega · enter selecciona · q sale") + "\n")

	case phaseChooseModel:
		title := "LLM"
		hint := "requisito: el tag debe llevar `tools` en https://ollama.com/library"
		if m.kind == "embed" {
			title = "Embeddings"
			hint = "aviso: si cambia la dim, hay que resetear la colección qdrant o cambiar MEM0_COLLECTION"
		}
		fmt.Fprintf(&b, "%s · actual: %s\n%s\n\n",
			lipgloss.NewStyle().Bold(true).Render(title),
			theme.Cyan.Render(m.currentValue()),
			theme.Dim.Render(hint),
		)
		for i, opt := range m.modelList() {
			b.WriteString(renderOption(opt.Label, i == m.modelIdx) + "\n")
		}
		b.WriteString("\n" + theme.Dim.Render("↑↓ navega · enter selecciona · esc vuelve · q sale") + "\n")

	case phaseConfirm:
		fmt.Fprintf(&b, "Selección: %s\n\n", theme.CyanBold.Render(m.selected))
		b.WriteString(theme.Dim.Render("Se ejecutará:") + "\n")
		fmt.Fprintf(&b, "   $ docker exec mcp-tools-ollama ollama pull %s\n", m.selected)
		think := ""
		if m.needsThinkFlag() {
			think = "  + MEM0_OLLAMA_THINK=false"
		}
		fmt.Fprintf(&b, "   · escribir %s=%s en .env.mem0%s\n", m.envVar(), m.selected, think)
		b.WriteString("   $ docker compose -f dockers/compose.yaml --env-file .env up -d --force-recreate mcp_tools_mem0\n")
		b.WriteString("\nConfirmar (" + theme.CyanBold.Render("y") + "/" + theme.Dim.Render("N") + "): ")

	case phasePulling:
		fmt.Fprintf(&b, "%s Descargando %s (puede tardar según tamaño)...\n", m.spinner.View(), m.selected)

	case phaseRestarting:
		fmt.Fprintf(&b, "%s Recreando mcp-tools-mem0 con el env nuevo...\n", m.spinner.View())

	case phaseDone:
		fmt.Fprintf(&b, "%s  %s=%s\n", theme.ChipGreen.Render(" OK "), m.envVar(), m.selected)
		if m.kind == "embed" {
			b.WriteString("\n" + theme.Yellow.Render("Aviso embeddings:") + "\n")
			fmt.Fprintf(&b, "  Si %s tiene dim distinta a la colección actual, se rompe.\n", m.selected)
			b.WriteString("  Cambia MEM0_COLLECTION a un nombre nuevo, o borra la anterior:\n")
			fmt.Fprintf(&b, "    $ curl -X DELETE http://127.0.0.1:6333/collections/%s\n", fallback(m.current["MEM0_COLLECTION"], "<colección>"))
		}
		b.WriteString("\n" + theme.Dim.Render("Pulsa cualquier tecla para salir.") + "\n")

	case phaseError:
		b.WriteString(theme.ChipRed.Render(" ERROR ") + "\n\n" + fmt.Sprintf("%v", m.err) + "\n")
	}

	return b.String()
}

func renderOption(label string, selected bool) string {
	if selected {
		return theme.CyanBold.Render("▸ ") + label
	}
	return "  " + theme.Dim.Render(label)
}

func fallback(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ExitCode returns the shell exit code the CLI should exit with after the TUI finishes.
func (m Model) ExitCode() int {
	if m.phase == phaseError {
		return 1
	}
	return 0
}

// StartTime is exposed so callers can time the run if desired.
func (m Model) StartTime() time.Time { return time.Now() }
