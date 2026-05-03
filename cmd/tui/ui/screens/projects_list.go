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

// ProjectsListScreen is the home screen and the projects tab.
//
// State machine:
//
//	idle      → user can navigate, hit c/e/d/enter
//	loading   → request in flight; ignore input until done
//	confirmDel→ asking the user to confirm deletion (y/n)
type ProjectsListScreen struct {
	client *client.Client

	projects []client.Project
	cursor   int
	loading  bool
	err      error
	flash    string // transient success message

	confirmDel bool // showing the y/n confirmation
}

func NewProjectsList(c *client.Client) *ProjectsListScreen {
	return &ProjectsListScreen{client: c}
}

func (s *ProjectsListScreen) Name() string { return "projects" }

func (s *ProjectsListScreen) Init() tea.Cmd {
	return s.fetch()
}

// projectsLoadedMsg arrives when ListProjects returns.
type projectsLoadedMsg struct {
	projects []client.Project
	err      error
}

// projectDeletedMsg arrives after a successful DELETE.
type projectDeletedMsg struct {
	id  int64
	err error
}

func (s *ProjectsListScreen) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ps, err := s.client.ListProjects(ctx)
		return projectsLoadedMsg{projects: ps, err: err}
	}
}

func (s *ProjectsListScreen) doDelete(id int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.client.DeleteProject(ctx, id)
		return projectDeletedMsg{id: id, err: err}
	}
}

func (s *ProjectsListScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case projectsLoadedMsg:
		s.loading = false
		s.projects = m.projects
		s.err = m.err
		if s.cursor >= len(s.projects) {
			s.cursor = 0
		}

	case projectDeletedMsg:
		s.confirmDel = false
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.flash = fmt.Sprintf("Deleted project %d", m.id)
		s.loading = true
		return s, s.fetch()

	case tea.KeyMsg:
		// Confirmation dialog takes priority.
		if s.confirmDel {
			switch m.String() {
			case "y", "Y":
				if s.cursor < len(s.projects) {
					s.loading = true
					return s, s.doDelete(s.projects[s.cursor].ID)
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
			if s.cursor < len(s.projects)-1 {
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
				return ui.PushScreenMsg{Screen: NewProjectForm(s.client, nil)}
			}
		case "e":
			if s.cursor < len(s.projects) {
				p := s.projects[s.cursor]
				return s, func() tea.Msg {
					return ui.PushScreenMsg{Screen: NewProjectForm(s.client, &p)}
				}
			}
		case "d":
			if s.cursor < len(s.projects) {
				s.confirmDel = true
			}
		case "enter":
			if s.cursor < len(s.projects) {
				p := s.projects[s.cursor]
				return s, func() tea.Msg {
					return ui.PushScreenMsg{Screen: NewProjectDetail(s.client, p.ID)}
				}
			}
		case "u":
			// Quick path to add usage for the selected project.
			if s.cursor < len(s.projects) {
				p := s.projects[s.cursor]
				return s, func() tea.Msg {
					return ui.PushScreenMsg{Screen: NewUsageForm(s.client, p.ID)}
				}
			}
		}
	}
	return s, nil
}

func (s *ProjectsListScreen) View() string {
	var b string
	b += ui.Title.Render("Projects") + "\n\n"

	if s.loading && len(s.projects) == 0 {
		return b + ui.StatusBar.Render("loading…")
	}
	if s.err != nil {
		b += ui.ErrorText.Render(s.err.Error()) + "\n\n"
	}
	if s.flash != "" {
		b += ui.SuccessText.Render(s.flash) + "\n\n"
	}

	if len(s.projects) == 0 {
		b += ui.Subtitle.Render("No projects yet. Press c to create one.")
		return b
	}

	// Header row.
	b += ui.Subtitle.Render(fmt.Sprintf("  %-4s %-30s %-22s %-22s %-22s", "ID", "Name", "Daily", "Monthly", "Total")) + "\n"

	for i, p := range s.projects {
		row := fmt.Sprintf("  %-4d %-30s %-22s %-22s %-22s",
			p.ID,
			truncate(p.Name, 30),
			fmtBudget(p.DailyBudgetDollars),
			fmtBudget(p.MonthlyBudgetDollars),
			fmtBudget(p.TotalBudgetDollars),
		)
		if i == s.cursor {
			row = lipgloss.NewStyle().
				Background(ui.ColorBgSubtle).
				Foreground(ui.ColorAccent).
				Render(row)
		}
		b += row + "\n"
	}

	if s.confirmDel && s.cursor < len(s.projects) {
		p := s.projects[s.cursor]
		b += "\n" + ui.WarnText.Render(fmt.Sprintf("Delete project %q (id=%d)? (y/n)", p.Name, p.ID))
	} else {
		b += "\n" + ui.HelpDesc.Render("c create  e edit  d delete  enter detail  u add usage  r refresh")
	}

	return b
}

func fmtBudget(d *float64) string {
	if d == nil {
		return "—"
	}
	return fmt.Sprintf("$%.2f", *d)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}
