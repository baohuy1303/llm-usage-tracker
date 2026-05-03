package screens

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-usage-tracker/cmd/tui/client"
	"llm-usage-tracker/cmd/tui/ui"
)

// ModelsListScreen mirrors ProjectsListScreen for /models. Keeps the same
// keybind grammar (c/e/d/r) so muscle memory carries between tabs.
type ModelsListScreen struct {
	client *client.Client

	models     []client.Model
	cursor     int
	loading    bool
	err        error
	flash      string
	confirmDel bool
}

func NewModelsList(c *client.Client) *ModelsListScreen {
	return &ModelsListScreen{client: c}
}

func (s *ModelsListScreen) Name() string { return "models" }

type modelsLoadedMsg struct {
	models []client.Model
	err    error
}

type modelDeletedMsg struct {
	id  int64
	err error
}

func (s *ModelsListScreen) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ms, err := s.client.ListModels(ctx)
		return modelsLoadedMsg{models: ms, err: err}
	}
}

func (s *ModelsListScreen) doDelete(id int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.client.DeleteModel(ctx, id)
		return modelDeletedMsg{id: id, err: err}
	}
}

func (s *ModelsListScreen) Init() tea.Cmd { return s.fetch() }

func (s *ModelsListScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case modelsLoadedMsg:
		s.loading = false
		s.models = m.models
		s.err = m.err
		if s.cursor >= len(s.models) {
			s.cursor = 0
		}

	case modelDeletedMsg:
		s.confirmDel = false
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.flash = fmt.Sprintf("Deleted model %d", m.id)
		s.loading = true
		return s, s.fetch()

	case tea.KeyMsg:
		if s.confirmDel {
			switch m.String() {
			case "y", "Y":
				if s.cursor < len(s.models) {
					s.loading = true
					return s, s.doDelete(s.models[s.cursor].ID)
				}
				s.confirmDel = false
			case "n", "N", "esc":
				s.confirmDel = false
			}
			return s, nil
		}
		switch m.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.models)-1 {
				s.cursor++
			}
		case "?":
			return s, func() tea.Msg { return ui.PushScreenMsg{Screen: NewHelp()} }
		case "r":
			s.loading = true
			s.flash = ""
			return s, s.fetch()
		case "c":
			return s, func() tea.Msg {
				return ui.PushScreenMsg{Screen: NewModelForm(s.client, nil)}
			}
		case "e":
			if s.cursor < len(s.models) {
				m := s.models[s.cursor]
				return s, func() tea.Msg {
					return ui.PushScreenMsg{Screen: NewModelForm(s.client, &m)}
				}
			}
		case "d":
			if s.cursor < len(s.models) {
				s.confirmDel = true
			}
		}
	}
	return s, nil
}

func (s *ModelsListScreen) View() string {
	out := ui.Title.Render("Models") + "\n\n"

	if s.loading && len(s.models) == 0 {
		return out + ui.StatusBar.Render("loading…")
	}
	if s.err != nil {
		out += ui.ErrorText.Render(s.err.Error()) + "\n\n"
	}
	if s.flash != "" {
		out += ui.SuccessText.Render(s.flash) + "\n\n"
	}
	if len(s.models) == 0 {
		out += ui.Subtitle.Render("No models. Press c to create one.")
		return out
	}

	out += ui.Subtitle.Render(fmt.Sprintf("  %-4s %-30s %18s %18s", "ID", "Name", "Input ¢/M", "Output ¢/M")) + "\n"

	for i, m := range s.models {
		row := fmt.Sprintf("  %-4d %-30s %18d %18d",
			m.ID, truncate(m.Name, 30), m.InputPerMillionCents, m.OutputPerMillionCents)
		if i == s.cursor {
			row = lipgloss.NewStyle().
				Background(ui.ColorBgSubtle).
				Foreground(ui.ColorAccent).
				Render(row)
		}
		out += row + "\n"
	}

	if s.confirmDel && s.cursor < len(s.models) {
		m := s.models[s.cursor]
		out += "\n" + ui.WarnText.Render(fmt.Sprintf("Delete model %q (id=%d)? (y/n)", m.Name, m.ID))
	} else {
		out += "\n" + ui.HintBar([][2]string{
			{"c", "create"},
			{"e", "edit"},
			{"d", "delete"},
			{"r", "refresh"},
		})
	}
	return out
}
