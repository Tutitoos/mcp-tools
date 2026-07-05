// Package theme centralises the lipgloss styles shared across mcp-tools TUIs.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Magenta  = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	Cyan     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	CyanBold = Cyan.Bold(true)
	Green    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Red      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Yellow   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	Dim      = lipgloss.NewStyle().Faint(true)

	ChipYellow = lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0")).Bold(true).Padding(0, 1)
	ChipGreen  = lipgloss.NewStyle().Background(lipgloss.Color("10")).Foreground(lipgloss.Color("0")).Bold(true).Padding(0, 1)
	ChipRed    = lipgloss.NewStyle().Background(lipgloss.Color("9")).Foreground(lipgloss.Color("0")).Bold(true).Padding(0, 1)

	PhaseAccent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("14")).
			PaddingLeft(1)
)

// StatusGlyph returns the single-character status marker matching the installer TSX.
func StatusGlyph(status string) string {
	switch status {
	case "pending":
		return "○"
	case "running":
		return "◐"
	case "ok":
		return "✔"
	case "fail":
		return "✘"
	}
	return "?"
}

// StatusStyle returns the color style for a step status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return Dim
	case "running":
		return Cyan
	case "ok":
		return Green
	case "fail":
		return Red
	}
	return Dim
}
