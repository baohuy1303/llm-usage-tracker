package ui

import "github.com/charmbracelet/lipgloss"

// Styles are shared across all screens. Defined once here so colors stay
// consistent and a future theme switch is a single edit.
var (
	ColorPrimary  = lipgloss.Color("#7D56F4")
	ColorAccent   = lipgloss.Color("#04B575")
	ColorWarn     = lipgloss.Color("#FFB454")
	ColorError    = lipgloss.Color("#E06C75")
	ColorMuted    = lipgloss.Color("#6C7086")
	ColorBg       = lipgloss.Color("#1E1E2E")
	ColorBgSubtle = lipgloss.Color("#2A2A3A")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Padding(0, 1)

	Subtitle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	ErrorText = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true).
			Padding(0, 1)

	SuccessText = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Padding(0, 1)

	WarnText = lipgloss.NewStyle().
			Foreground(ColorWarn).
			Padding(0, 1)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	HelpKey = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	HelpDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
