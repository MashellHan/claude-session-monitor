package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Panel1   key.Binding
	Panel2   key.Binding
	Panel3   key.Binding
	Refresh  key.Binding
	Filter   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev panel"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next panel"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "expand/collapse"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "sessions"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "agents"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "tasks"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter active"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
