// Package installer implements the mcp-tools install TUI (10 phases + --dry).
package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/tui/theme"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusOK      Status = "ok"
	StatusFail    Status = "fail"
)

// DryCmd is a captured command in dry-run mode.
type DryCmd struct {
	StepKey string
	Cmd     string
}

// StepFn is what each step implements. Log is used to append dry-run commands (as
// "$ <cmd>") or to print progress lines.
type StepFn func(dry bool, log func(string)) error

// Step is one item in the installer sequence.
type Step struct {
	Key   string
	Label string
	Phase string
	Run   StepFn
}

// Phases in display order.
var Phases = []string{"Preparación", "Build", "Instalación", "Arranque"}

// Model is the bubbletea state for the installer.
type Model struct {
	steps       []Step
	dry         bool
	states      map[string]Status
	durations   map[string]time.Duration
	errors      map[string]string
	dryCommands []DryCmd
	spinner     spinner.Model
	current     int
	totalMs     time.Duration
	startTime   time.Time
	done        bool
	failed      bool
	width       int
}

// stepDoneMsg is emitted after Step.Run returns.
type stepDoneMsg struct {
	key      string
	elapsed  time.Duration
	err      error
	captured []string // "$ ..." lines emitted by the step (dry-run)
}

// New constructs an installer Model wired to the given steps.
func New(steps []Step, dry bool) Model {
	states := map[string]Status{}
	for _, s := range steps {
		states[s.Key] = StatusPending
	}
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Cyan
	return Model{
		steps:     steps,
		dry:       dry,
		states:    states,
		durations: map[string]time.Duration{},
		errors:    map[string]string{},
		spinner:   sp,
		current:   0,
		startTime: time.Now(),
	}
}

func (m Model) Init() tea.Cmd { return tea.Batch(m.spinner.Tick, m.runNext()) }

func (m Model) runNext() tea.Cmd {
	if m.current >= len(m.steps) {
		return nil
	}
	step := m.steps[m.current]
	return func() tea.Msg {
		var captured []string
		log := func(s string) { captured = append(captured, s) }
		start := time.Now()
		err := step.Run(m.dry, log)
		return stepDoneMsg{key: step.Key, elapsed: time.Since(start), err: err, captured: captured}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if m.done {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stepDoneMsg:
		m.durations[msg.key] = msg.elapsed
		for _, line := range msg.captured {
			m.dryCommands = append(m.dryCommands, DryCmd{StepKey: msg.key, Cmd: strings.TrimPrefix(line, "$ ")})
		}
		if msg.err != nil {
			m.states[msg.key] = StatusFail
			m.errors[msg.key] = msg.err.Error()
			m.failed = true
			m.totalMs = time.Since(m.startTime)
			m.done = true
			return m, tea.Quit
		}
		m.states[msg.key] = StatusOK
		m.current++
		if m.current >= len(m.steps) {
			m.totalMs = time.Since(m.startTime)
			m.done = true
			return m, tea.Quit
		}
		m.states[m.steps[m.current].Key] = StatusRunning
		return m, tea.Batch(m.spinner.Tick, m.runNext())
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(theme.Magenta.Render("mcp-tools") + theme.Dim.Render("  installer") + "\n")
	b.WriteString(theme.Dim.Render("self-hosted MCP servers para Claude Code, OpenCode y OMP") + "\n\n")

	if m.dry {
		b.WriteString(theme.ChipYellow.Render(" DRY RUN ") + theme.Dim.Render("  no se ejecuta nada; solo se muestra qué haría") + "\n\n")
	}

	// Phases
	for _, phase := range Phases {
		stepsInPhase := m.stepsInPhase(phase)
		if len(stepsInPhase) == 0 {
			continue
		}
		b.WriteString(theme.PhaseAccent.Render(phase) + "\n")
		for _, s := range stepsInPhase {
			idx := m.stepIndex(s.Key) + 1
			st := m.states[s.Key]
			dt, hasDT := m.durations[s.Key]
			numStr := fmt.Sprintf("%02d  ", idx)
			label := padRight(s.Label, 52)
			var line string
			switch st {
			case StatusRunning:
				line = theme.Dim.Render(numStr) + m.spinner.View() + "  " + label
			default:
				glyph := theme.StatusStyle(string(st)).Render(theme.StatusGlyph(string(st))) + "  "
				labelStr := label
				if st == StatusFail {
					labelStr = theme.Red.Render(label)
				}
				line = theme.Dim.Render(numStr) + glyph + labelStr
			}
			if hasDT {
				line += theme.Dim.Render(fmt.Sprintf("%.1fs", dt.Seconds()))
			}
			b.WriteString(" " + line + "\n")
		}
		b.WriteString("\n")
	}

	if !m.done {
		return b.String()
	}

	// Footer
	if m.failed {
		b.WriteString(theme.ChipRed.Render(" ERROR ") + theme.Dim.Render(fmt.Sprintf("  tras %.1fs", m.totalMs.Seconds())) + "\n\n")
		boxWidth := m.width - 2 // leave 1 col margin each side
		if boxWidth < 40 {
			boxWidth = 40 // sane minimum
		}
		if boxWidth > 120 {
			boxWidth = 120 // cap so lines wrap on ultra-wide terminals too
		}
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(0, 1).
			MarginBottom(1).
			Width(boxWidth)
		for k, v := range m.errors {
			title := theme.Red.Bold(true).Render("● " + k)
			b.WriteString(errBox.Render(title+"\n"+strings.TrimSpace(v)) + "\n")
		}
		b.WriteString(theme.Dim.Render("Corrige el error y relanza `mcp-tools install` — es idempotente.") + "\n")
		return b.String()
	}

	if m.dry {
		phaseCount := m.stepsWithCommands()
		b.WriteString(theme.ChipGreen.Render(" DRY RUN OK ") +
			theme.Dim.Render(fmt.Sprintf("  %.1fs · %d comandos en %d pasos", m.totalMs.Seconds(), len(m.dryCommands), phaseCount)) + "\n\n")
		b.WriteString(theme.Dim.Render("Comandos que ejecutaría (sh -c):") + "\n")
		home, _ := os.UserHomeDir()
		for _, s := range m.steps {
			cmds := m.commandsFor(s.Key)
			if len(cmds) == 0 {
				continue
			}
			idx := m.stepIndex(s.Key) + 1
			b.WriteString("\n " + theme.CyanBold.Render(fmt.Sprintf("%02d", idx)) + theme.Dim.Render("  ") + s.Label + "\n")
			for _, c := range cmds {
				b.WriteString(theme.Dim.Render("     $ " + strings.ReplaceAll(c, home, "~")) + "\n")
			}
		}
		b.WriteString("\n" + theme.Dim.Render("Relanza sin ") + theme.Yellow.Render("--dry") + theme.Dim.Render(" para aplicar.") + "\n")
		return b.String()
	}

	// Success
	b.WriteString(theme.ChipGreen.Render(" INSTALADO ") + theme.Dim.Render(fmt.Sprintf("  %.1fs", m.totalMs.Seconds())) + "\n\n")
	b.WriteString(theme.Dim.Render("Próximos pasos:") + "\n")
	b.WriteString("  → Reinicia tu cliente MCP (Claude Code, OpenCode, OMP).\n")
	b.WriteString("  → Verifica: " + theme.CyanBold.Render("claude mcp list") + " · los 3 servers como ✔ Connected.\n")
	b.WriteString("  → Config avanzada: " + theme.CyanBold.Render("docs/ADVANCED.md") + "\n")
	return b.String()
}

// stepsInPhase returns the ordered slice of steps in the named phase.
func (m Model) stepsInPhase(phase string) []Step {
	var out []Step
	for _, s := range m.steps {
		if s.Phase == phase {
			out = append(out, s)
		}
	}
	return out
}

func (m Model) stepIndex(key string) int {
	for i, s := range m.steps {
		if s.Key == key {
			return i
		}
	}
	return -1
}

func (m Model) commandsFor(key string) []string {
	var out []string
	for _, c := range m.dryCommands {
		if c.StepKey == key {
			out = append(out, c.Cmd)
		}
	}
	return out
}

func (m Model) stepsWithCommands() int {
	seen := map[string]bool{}
	for _, c := range m.dryCommands {
		seen[c.StepKey] = true
	}
	return len(seen)
}

// ExitCode returns the shell exit code to use after the TUI quits.
func (m Model) ExitCode() int {
	if m.failed {
		return 1
	}
	return 0
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// Helper for mem0-src step: reads .env and validates MEM0_SRC_PATH.
func CheckMem0Src(dry bool, log func(string)) error {
	envPath := config.EnvFile()
	if _, err := os.Stat(envPath); err != nil {
		if errors.Is(err, os.ErrNotExist) && dry {
			log("· skip (.env aún no existe; env step lo creará)")
			return nil
		}
		return fmt.Errorf(".env ausente en %s — el paso 'env' debería haberlo generado", envPath)
	}
	env, err := config.LoadEnv(envPath)
	if err != nil {
		return err
	}
	srcPath := env["MEM0_SRC_PATH"]
	if srcPath == "" {
		return fmt.Errorf("MEM0_SRC_PATH ausente en .env")
	}
	pyproject := filepath.Join(srcPath, "pyproject.toml")
	if _, err := os.Stat(pyproject); err != nil {
		return fmt.Errorf("MEM0_SRC_PATH no existe o no es un repo válido: %s\nClona: git clone https://github.com/elvismdev/mem0-mcp-selfhosted %s", srcPath, srcPath)
	}
	return nil
}
