package ui

import "github.com/charmbracelet/lipgloss"

// HelpView renders the help overlay showing all key bindings.
func HelpView(width int) string {
	helpText := `
  Key Bindings
  ────────────────────────────────
  ↑/k         Move cursor up
  ↓/j         Move cursor down
  Enter       Toggle detail expansion
  Tab         Switch panel focus
  1/2/3       Jump to panel
  r           Force refresh
  f           Toggle active-only filter
  q/Ctrl+C    Quit
  ?           Toggle this help
`
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(40).
		Align(lipgloss.Left)

	return style.Render(helpText)
}
