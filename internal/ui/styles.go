// Package ui provides shared TUI styles and key bindings.
package ui

import "github.com/charmbracelet/lipgloss"

// Color palette — muted, terminal-friendly.
var (
	ColorPrimary   = lipgloss.Color("99")  // Purple
	ColorSecondary = lipgloss.Color("39")  // Blue
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorDanger    = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("240") // Gray
	ColorHighlight = lipgloss.Color("229") // Light yellow
	ColorWhite     = lipgloss.Color("15")  // White
	ColorCyan      = lipgloss.Color("51")  // Cyan
)

// Layout styles.
var (
	// TitleStyle for the top title bar.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1)

	// PanelTitleStyle for section headers within panels.
	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			PaddingLeft(1)

	// ActivePanelTitleStyle for the currently focused panel header.
	ActivePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				PaddingLeft(1)

	// SummaryBarStyle for the fixed bottom bar.
	SummaryBarStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Background(lipgloss.Color("236")).
			PaddingLeft(1).
			PaddingRight(1)

	// SelectedRowStyle for the currently highlighted table row.
	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorHighlight).
				Background(lipgloss.Color("236"))

	// NormalRowStyle for non-selected table rows.
	NormalRowStyle = lipgloss.NewStyle()

	// HeaderStyle for table headers.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238"))

	// StatusRunningStyle for running indicator.
	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	// StatusIdleStyle for idle indicator.
	StatusIdleStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// StatusDoneStyle for done indicator.
	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// DetailBorderStyle for inline detail expansion.
	DetailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				PaddingLeft(1).
				PaddingRight(1).
				MarginLeft(2)

	// HelpStyle for help text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ErrorStyle for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// AliveStyle for active PID indicators.
	AliveStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// DeadStyle for dead PID indicators.
	DeadStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// MutedStyle for secondary text.
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// CursorStyle for the selection cursor.
	CursorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)
)
