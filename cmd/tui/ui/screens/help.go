package screens

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-usage-tracker/cmd/tui/ui"
)

// HelpScreen is a static keybind reference. Pushed onto the stack from any
// screen via "?", popped via Esc.
type HelpScreen struct{}

func NewHelp() *HelpScreen { return &HelpScreen{} }

func (h *HelpScreen) Name() string  { return "help" }
func (h *HelpScreen) Init() tea.Cmd { return nil }

func (h *HelpScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	return h, nil
}

func (h *HelpScreen) View() string {
	rows := [][2]string{
		{"Global", ""},
		{"q / ctrl+c", "quit"},
		{"?", "open this help"},
		{"esc", "back / close drill-down"},
		{"r", "refresh current screen"},
		{"tab", "cycle top-level tabs (when not in a drill-down)"},
		{"", ""},
		{"Lists", ""},
		{"↑↓ / j k", "navigate items"},
		{"enter", "drill into selected item"},
		{"c", "create new item"},
		{"e", "edit selected item"},
		{"d", "delete selected item"},
		{"/", "filter list (where supported)"},
		{"", ""},
		{"Forms", ""},
		{"tab / shift+tab", "next / previous field"},
		{"enter", "submit (when on submit button)"},
		{"esc", "cancel and go back"},
		{"", ""},
		{"Pagination (events browser)", ""},
		{"n", "next page (uses next_cursor from response)"},
		{"p", "previous page (re-fetch first page)"},
	}

	var out string
	for _, row := range rows {
		if row[1] == "" {
			// Section header.
			out += "\n" + ui.Title.Render(row[0]) + "\n"
			continue
		}
		out += "  " + ui.HelpKey.Render(padRight(row[0], 16)) + ui.HelpDesc.Render(row[1]) + "\n"
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(out)
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}
