// Package toolselect implements the multi-select TUI shown by
// `mcp-tools install` and `mcp-tools configure`. Dependency toggles are
// enforced (can't uncheck a tool while a checked tool depends on it).
package toolselect

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tutitoos/mcp-tools/internal/tools"
	"github.com/Tutitoos/mcp-tools/internal/tui/theme"
)

// Model is the bubbletea state for the multi-select TUI.
type Model struct {
	available []tools.Tool
	checked   map[string]bool
	cursor    int
	confirmed bool
	cancelled bool
	notice    string
}

// New builds a Model. `available` is the list to display (caller decides which
// tools to include — e.g. drop nvidia-toolkit on hosts without a GPU).
// `preChecked` seeds the initial checkbox state.
func New(available []tools.Tool, preChecked map[string]bool) Model {
	checked := map[string]bool{}
	for _, t := range available {
		if preChecked[t.Key] {
			checked[t.Key] = true
		}
	}
	return Model{available: available, checked: checked}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	m.notice = ""
	switch km.String() {
	case "ctrl+c", "q", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "enter":
		m.confirmed = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.available)-1 {
			m.cursor++
		}
	case " ", "x":
		m.toggle(m.cursor)
	case "a":
		m.checkAll()
	case "n":
		m.uncheckAll()
	}
	return m, nil
}

// toggle flips the row at idx respecting deps in both directions.
func (m *Model) toggle(idx int) {
	t := m.available[idx]
	if m.checked[t.Key] {
		// Uncheck — reject if any currently-checked tool declares this as a dep.
		if blocker := m.reverseDep(t.Key); blocker != "" {
			m.notice = fmt.Sprintf("no puedo desmarcar %s: %s lo requiere", t.Key, blocker)
			fmt.Print("\a") // BEL
			return
		}
		delete(m.checked, t.Key)
		return
	}
	// Check — pull in any missing deps automatically.
	for _, dep := range t.Deps {
		if !m.checked[dep] {
			m.checked[dep] = true
			m.notice = fmt.Sprintf("auto-marcado %s (dep de %s)", dep, t.Key)
		}
	}
	m.checked[t.Key] = true
}

func (m *Model) checkAll() {
	for _, t := range m.available {
		m.checked[t.Key] = true
	}
}

func (m *Model) uncheckAll() {
	// Uncheck bottom-up but honor deps: keep a key checked if any other checked
	// tool still needs it.
	m.checked = map[string]bool{}
}

func (m Model) reverseDep(key string) string {
	for _, other := range m.available {
		if other.Key == key || !m.checked[other.Key] {
			continue
		}
		for _, dep := range other.Deps {
			if dep == key {
				return other.Key
			}
		}
	}
	return ""
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(theme.Magenta.Render("mcp-tools") + theme.Dim.Render("  seleccionar componentes") + "\n")
	b.WriteString(theme.Dim.Render("↑/↓ moverse · space toggle · a todos · n ninguno · enter confirmar · q cancelar") + "\n\n")

	// Column widths.
	keyW, depW := 20, 8
	for _, t := range m.available {
		if len(t.Key) > keyW {
			keyW = len(t.Key)
		}
		if l := len(t.Deploy.String()); l > depW {
			depW = l
		}
	}

	for i, t := range m.available {
		mark := "[ ]"
		if m.checked[t.Key] {
			mark = "[x]"
		}
		row := fmt.Sprintf("  %s  %s  %s  %s",
			mark,
			padRight(t.Key, keyW),
			padRight(t.Deploy.String(), depW),
			t.Summary,
		)
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("▸") + row[1:]
		}
		b.WriteString(row + "\n")
	}
	if m.notice != "" {
		b.WriteString("\n" + theme.Yellow.Render("· "+m.notice) + "\n")
	}
	return b.String()
}

// Confirmed reports whether the user pressed enter (as opposed to cancelling).
func (m Model) Confirmed() bool { return m.confirmed && !m.cancelled }

// Selected returns the checked keys, ordered topologically by tools.TopoSort.
func (m Model) Selected() []string {
	if !m.Confirmed() {
		return nil
	}
	var keys []string
	for _, t := range m.available {
		if m.checked[t.Key] {
			keys = append(keys, t.Key)
		}
	}
	sorted, err := tools.TopoSort(keys)
	if err != nil {
		// TopoSort failure means an unknown key or cycle — neither is
		// reachable here (available is a subset of Registry, Deps are
		// registry-declared). Return unsorted rather than lose the pick.
		return keys
	}
	return sorted
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
