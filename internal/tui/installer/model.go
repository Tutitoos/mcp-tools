// Package installer runs a sequence of Steps with a Bubbletea progress UI.
// Used by mcp-tools install/configure/update to show per-tool progress with
// OK/Fail/Pending glyphs and optional dry-run capture.
package installer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// StepFn is what each step implements. `log` appends dry-run commands ("$ …")
// or progress lines to the model's dry-command capture buffer.
type StepFn func(dry bool, log func(string)) error

// Step is one item in the install / configure / update sequence.
type Step struct {
	Key   string
	Label string
	Phase string
	Run   StepFn
}

// Model is the bubbletea state.
type Model struct {
	steps       []Step
	phases      []string
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
	captured []string
}

// New constructs a Model. `phases` is the display order; a Step whose Phase is
// not listed lands under a synthesised final group.
func New(steps []Step, phases []string, dry bool) Model {
	states := map[string]Status{}
	for _, s := range steps {
		states[s.Key] = StatusPending
	}
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Cyan
	return Model{
		steps:     steps,
		phases:    phases,
		dry:       dry,
		states:    states,
		durations: map[string]time.Duration{},
		errors:    map[string]string{},
		spinner:   sp,
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
			if !m.done {
				key := "cancel"
				if m.current < len(m.steps) {
					key = m.steps[m.current].Key
					m.states[key] = StatusFail
				}
				m.failed = true
				m.errors[key] = "cancelado por el user (ctrl+c)"
			}
			m.done = true
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
	b.WriteString(theme.Magenta.Render("mcp-tools") + theme.Dim.Render("  progress") + "\n\n")
	if m.dry {
		b.WriteString(theme.ChipYellow.Render(" DRY RUN ") + "\n\n")
	}
	for _, phase := range m.phases {
		steps := m.stepsInPhase(phase)
		if len(steps) == 0 {
			continue
		}
		b.WriteString(theme.PhaseAccent.Render(phase) + "\n")
		for _, s := range steps {
			b.WriteString(" " + m.renderStep(s) + "\n")
		}
		b.WriteString("\n")
	}
	if !m.done {
		return b.String()
	}
	if m.failed {
		b.WriteString(theme.ChipRed.Render(" ERROR ") + theme.Dim.Render(fmt.Sprintf("  tras %.1fs", m.totalMs.Seconds())) + "\n\n")
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(0, 1).
			MarginBottom(1).
			Width(clampWidth(m.width))
		for k, v := range m.errors {
			title := theme.Red.Bold(true).Render("● " + k)
			b.WriteString(box.Render(title+"\n"+strings.TrimSpace(v)) + "\n")
		}
		return b.String()
	}
	if m.dry {
		b.WriteString(theme.ChipGreen.Render(" DRY RUN OK ") + theme.Dim.Render(fmt.Sprintf("  %.1fs · %d comandos", m.totalMs.Seconds(), len(m.dryCommands))) + "\n\n")
		home, _ := os.UserHomeDir()
		for _, s := range m.steps {
			cmds := m.commandsFor(s.Key)
			if len(cmds) == 0 {
				continue
			}
			b.WriteString(theme.CyanBold.Render(s.Key) + "\n")
			for _, c := range cmds {
				b.WriteString(theme.Dim.Render("  $ "+strings.ReplaceAll(c, home, "~")) + "\n")
			}
		}
		return b.String()
	}
	b.WriteString(theme.ChipGreen.Render(" DONE ") + theme.Dim.Render(fmt.Sprintf("  %.1fs", m.totalMs.Seconds())) + "\n")
	return b.String()
}

func (m Model) renderStep(s Step) string {
	st := m.states[s.Key]
	dt, hasDT := m.durations[s.Key]
	label := padRight(s.Label, 44)
	var line string
	if st == StatusRunning {
		line = m.spinner.View() + "  " + label
	} else {
		glyph := theme.StatusStyle(string(st)).Render(theme.StatusGlyph(string(st)))
		if st == StatusFail {
			label = theme.Red.Render(label)
		}
		line = glyph + "  " + label
	}
	if hasDT {
		line += theme.Dim.Render(fmt.Sprintf("%.1fs", dt.Seconds()))
	}
	return line
}

func (m Model) stepsInPhase(phase string) []Step {
	var out []Step
	for _, s := range m.steps {
		if s.Phase == phase {
			out = append(out, s)
		}
	}
	return out
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

// ExitCode returns the shell exit code to use after the TUI quits.
func (m Model) ExitCode() int {
	if m.failed {
		return 1
	}
	return 0
}

// DryCommands surfaces the captured "$ …" lines so a caller can print them
// after the TUI has torn down (e.g. for scripting friendliness).
func (m Model) DryCommands() []DryCmd { return m.dryCommands }

// Errors returns collected step errors keyed by step Key.
func (m Model) Errors() map[string]string { return m.errors }

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func clampWidth(w int) int {
	switch {
	case w < 40:
		return 40
	case w > 120:
		return 120
	}
	return w
}
