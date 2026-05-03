package ui

import "github.com/charmbracelet/bubbles/key"

// Keys is the global keymap. Individual screens may add their own bindings,
// but anything in here behaves the same everywhere.
type GlobalKeys struct {
	Quit    key.Binding
	Help    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Tab     key.Binding
}

func DefaultKeys() GlobalKeys {
	return GlobalKeys{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next screen"),
		),
	}
}
