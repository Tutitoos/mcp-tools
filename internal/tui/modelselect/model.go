// Package modelselect implements the multi-select TUI for
// `mcp-tools models`: rows are Ollama model tags (curated + already-installed),
// space toggles, enter confirms; the caller computes the pull/rm diff.
package modelselect

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tutitoos/mcp-tools/internal/tui/theme"
)

// Item is one row: a tag + display label + optional installed metadata.
type Item struct {
	Tag       string
	Label     string
	Section   string // "LLM", "Embed", "Otros instalados"
	Installed bool
	Size      string
}

type Model struct {
	items     []Item
	checked   map[string]bool
	cursor    int
	confirmed bool
	cancelled bool
}

// New wires the items list. Items already Installed start pre-checked.
func New(items []Item) Model {
	checked := map[string]bool{}
	for _, it := range items {
		if it.Installed {
			checked[it.Tag] = true
		}
	}
	return Model{items: items, checked: checked}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
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
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ", "x":
		tag := m.items[m.cursor].Tag
		if m.checked[tag] {
			delete(m.checked, tag)
		} else {
			m.checked[tag] = true
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(theme.Magenta.Render("mcp-tools") + theme.Dim.Render("  modelos Ollama") + "\n")
	b.WriteString(theme.Dim.Render("↑/↓ · space toggle · enter confirmar · q cancelar") + "\n\n")

	lastSection := ""
	for i, it := range m.items {
		if it.Section != lastSection {
			if lastSection != "" {
				b.WriteString("\n")
			}
			b.WriteString(theme.PhaseAccent.Render(it.Section) + "\n")
			lastSection = it.Section
		}
		mark := "[ ]"
		if m.checked[it.Tag] {
			mark = "[x]"
		}
		row := fmt.Sprintf("  %s  %s", mark, it.Label)
		if it.Installed && it.Size != "" {
			row += theme.Dim.Render(fmt.Sprintf("   [%s]", it.Size))
		}
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("▸") + row[1:]
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}

// Confirmed reports whether the user pressed enter (as opposed to cancelling).
func (m Model) Confirmed() bool { return m.confirmed && !m.cancelled }

// Selected returns the tags marked checked at confirmation time.
func (m Model) Selected() []string {
	if !m.Confirmed() {
		return nil
	}
	var out []string
	for _, it := range m.items {
		if m.checked[it.Tag] {
			out = append(out, it.Tag)
		}
	}
	return out
}
