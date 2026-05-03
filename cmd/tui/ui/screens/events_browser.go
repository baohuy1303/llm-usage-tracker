package screens

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-usage-tracker/cmd/tui/client"
	"llm-usage-tracker/cmd/tui/ui"
)

// EventsBrowserScreen lists raw usage events across all projects with cursor
// pagination. Filters (from/to dates) are applied via the f keybind which
// drops into a small inline editor.
//
// Pagination model:
//   - cursors stack tracks all next_cursor values we've followed
//   - n: fetch next page (uses page.NextCursor)
//   - p: pop one cursor and re-fetch (so previous page returns)
type EventsBrowserScreen struct {
	client *client.Client

	page    *client.EventsPage
	cursors []string // stack of cursors used to reach this page (empty = first page)
	loading bool
	err     error

	// Filter state. Applied via "f" → inline edit mode.
	from        string
	to          string
	editingFrom bool
	editingTo   bool
	editBuf     string
}

func NewEventsBrowser(c *client.Client) *EventsBrowserScreen {
	return &EventsBrowserScreen{client: c}
}

func (s *EventsBrowserScreen) Name() string { return "events" }

type eventsLoadedMsg struct {
	page *client.EventsPage
	err  error
}

func (s *EventsBrowserScreen) fetch(cursor string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		page, err := s.client.ListAllEvents(ctx, s.from, s.to, cursor, 30)
		return eventsLoadedMsg{page: page, err: err}
	}
}

func (s *EventsBrowserScreen) Init() tea.Cmd {
	s.loading = true
	return s.fetch("")
}

func (s *EventsBrowserScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case eventsLoadedMsg:
		s.loading = false
		s.page = m.page
		s.err = m.err

	case tea.KeyMsg:
		// Inline edit modes for from/to: capture all keys until enter or esc.
		if s.editingFrom || s.editingTo {
			switch m.Type {
			case tea.KeyEnter:
				if s.editingFrom {
					s.from = s.editBuf
				} else {
					s.to = s.editBuf
				}
				s.editingFrom = false
				s.editingTo = false
				s.cursors = nil
				s.loading = true
				return s, s.fetch("")
			case tea.KeyEsc:
				s.editingFrom = false
				s.editingTo = false
			case tea.KeyBackspace:
				if len(s.editBuf) > 0 {
					s.editBuf = s.editBuf[:len(s.editBuf)-1]
				}
			default:
				if m.Type == tea.KeyRunes {
					s.editBuf += string(m.Runes)
				}
			}
			return s, nil
		}

		switch m.String() {
		case "?":
			return s, func() tea.Msg { return ui.PushScreenMsg{Screen: NewHelp()} }
		case "r":
			s.loading = true
			s.cursors = nil
			return s, s.fetch("")
		case "n":
			if s.page != nil && s.page.HasMore && s.page.NextCursor != "" {
				s.cursors = append(s.cursors, s.page.NextCursor)
				s.loading = true
				return s, s.fetch(s.page.NextCursor)
			}
		case "p":
			if len(s.cursors) > 0 {
				s.cursors = s.cursors[:len(s.cursors)-1]
				cursor := ""
				if len(s.cursors) > 0 {
					cursor = s.cursors[len(s.cursors)-1]
				}
				s.loading = true
				return s, s.fetch(cursor)
			}
		case "f":
			s.editBuf = s.from
			s.editingFrom = true
		case "t":
			s.editBuf = s.to
			s.editingTo = true
		case "x":
			// Clear filters.
			s.from = ""
			s.to = ""
			s.cursors = nil
			s.loading = true
			return s, s.fetch("")
		}
	}
	return s, nil
}

func (s *EventsBrowserScreen) View() string {
	out := ui.Title.Render("Usage Events (all projects)") + "\n"

	// Filter bar.
	out += ui.Subtitle.Render(fmt.Sprintf("  filters: from=%q  to=%q  page=%d", s.from, s.to, len(s.cursors)+1)) + "\n\n"

	if s.editingFrom {
		out += ui.WarnText.Render(fmt.Sprintf("editing from: %s_  (enter to apply, esc to cancel)", s.editBuf)) + "\n\n"
	}
	if s.editingTo {
		out += ui.WarnText.Render(fmt.Sprintf("editing to: %s_  (enter to apply, esc to cancel)", s.editBuf)) + "\n\n"
	}

	if s.loading && s.page == nil {
		return out + ui.StatusBar.Render("loading…")
	}
	if s.err != nil {
		out += ui.ErrorText.Render(s.err.Error()) + "\n\n"
	}
	if s.page == nil || len(s.page.Events) == 0 {
		out += ui.Subtitle.Render("no events in this window")
		out += "\n\n" + ui.HelpDesc.Render("f set from  t set to  x clear filters  r refresh")
		return out
	}

	out += ui.Subtitle.Render(fmt.Sprintf("  %-6s %-19s %-4s %-22s %10s %10s %10s %10s",
		"ID", "Time", "PID", "Model", "Tokens In", "Tokens Out", "Cost", "Latency")) + "\n"

	for _, e := range s.page.Events {
		lat := "—"
		if e.LatencyMs != nil {
			lat = fmt.Sprintf("%dms", *e.LatencyMs)
		}
		out += fmt.Sprintf("  %-6d %-19s %-4d %-22s %10d %10d %10s %10s\n",
			e.ID,
			e.CreatedAt.Format("2006-01-02 15:04:05"),
			e.ProjectID,
			truncate(e.Model, 22),
			e.TokensIn, e.TokensOut,
			fmt.Sprintf("$%.5f", e.CostDollars),
			lat,
		)
	}

	pageInfo := fmt.Sprintf("page %d", len(s.cursors)+1)
	if s.page.HasMore {
		pageInfo += " (more)"
	} else {
		pageInfo += " (end)"
	}

	out += "\n" + ui.HelpDesc.Render(fmt.Sprintf("%s   n next  p prev  f set from  t set to  x clear  r refresh", pageInfo))
	return out
}
