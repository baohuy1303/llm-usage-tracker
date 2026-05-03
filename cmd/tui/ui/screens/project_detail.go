package screens

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-usage-tracker/cmd/tui/client"
	"llm-usage-tracker/cmd/tui/ui"
)

// ProjectDetailScreen drills into one project. Shows budget_status (live spend
// vs limits) plus the latest 10 events. From here you can press u to add a
// usage event for this project.
type ProjectDetailScreen struct {
	client    *client.Client
	projectID int64

	project *client.Project
	events  []client.Usage
	loading bool
	err     error
}

func NewProjectDetail(c *client.Client, projectID int64) *ProjectDetailScreen {
	return &ProjectDetailScreen{client: c, projectID: projectID, loading: true}
}

func (s *ProjectDetailScreen) Name() string { return "project detail" }

type projectDetailLoadedMsg struct {
	project *client.Project
	events  []client.Usage
	err     error
}

func (s *ProjectDetailScreen) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p, err := s.client.GetProject(ctx, s.projectID)
		if err != nil {
			return projectDetailLoadedMsg{err: err}
		}
		page, err := s.client.ListProjectEvents(ctx, s.projectID, "", "", "", 10)
		if err != nil {
			return projectDetailLoadedMsg{project: p, err: err}
		}
		return projectDetailLoadedMsg{project: p, events: page.Events}
	}
}

func (s *ProjectDetailScreen) Init() tea.Cmd { return s.fetch() }

func (s *ProjectDetailScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case projectDetailLoadedMsg:
		s.loading = false
		s.project = m.project
		s.events = m.events
		s.err = m.err

	case tea.KeyMsg:
		switch m.String() {
		case "r":
			s.loading = true
			return s, s.fetch()
		case "?":
			return s, func() tea.Msg { return ui.PushScreenMsg{Screen: NewHelp()} }
		case "u":
			return s, func() tea.Msg {
				return ui.PushScreenMsg{Screen: NewUsageForm(s.client, s.projectID)}
			}
		case "e":
			if s.project != nil {
				p := *s.project
				return s, func() tea.Msg {
					return ui.PushScreenMsg{Screen: NewProjectForm(s.client, &p)}
				}
			}
		}
	}
	return s, nil
}

func (s *ProjectDetailScreen) View() string {
	if s.loading && s.project == nil {
		return ui.StatusBar.Render("loading…")
	}
	if s.err != nil {
		return ui.ErrorText.Render(s.err.Error())
	}
	if s.project == nil {
		return ui.StatusBar.Render("no project")
	}

	out := ui.Title.Render(fmt.Sprintf("Project #%d — %s", s.project.ID, s.project.Name)) + "\n\n"

	out += "Budgets\n"
	out += fmt.Sprintf("  Daily:   %s\n", fmtBudget(s.project.DailyBudgetDollars))
	out += fmt.Sprintf("  Monthly: %s\n", fmtBudget(s.project.MonthlyBudgetDollars))
	out += fmt.Sprintf("  Total:   %s\n\n", fmtBudget(s.project.TotalBudgetDollars))

	if bs := s.project.BudgetStatus; bs != nil {
		out += "Live status\n"
		out += renderWindow("  Daily:  ", bs.Daily) + "\n"
		out += renderWindow("  Month:  ", bs.Monthly) + "\n"
		out += renderWindow("  Total:  ", bs.Total) + "\n\n"
	}

	out += ui.Title.Render("Last 10 events") + "\n"
	if len(s.events) == 0 {
		out += ui.Subtitle.Render("  no events yet") + "\n"
	} else {
		out += ui.Subtitle.Render(fmt.Sprintf("  %-19s %-25s %10s %10s %10s %12s",
			"Time", "Model", "TokensIn", "TokensOut", "Cost", "Latency")) + "\n"
		for _, e := range s.events {
			lat := "—"
			if e.LatencyMs != nil {
				lat = fmt.Sprintf("%dms", *e.LatencyMs)
			}
			out += fmt.Sprintf("  %-19s %-25s %10d %10d %10s %12s\n",
				e.CreatedAt.Format("2006-01-02 15:04:05"),
				truncate(e.Model, 25),
				e.TokensIn, e.TokensOut,
				fmt.Sprintf("$%.5f", e.CostDollars),
				lat,
			)
		}
	}

	out += "\n" + ui.HelpDesc.Render("u add usage  e edit project  r refresh  esc back")
	return out
}

func renderWindow(label string, w *client.BudgetWindow) string {
	if w == nil {
		return ui.HelpDesc.Render(label + "no budget set")
	}
	line := fmt.Sprintf("%s$%.5f / $%.2f (%.1f%%)", label, w.SpentDollars, w.BudgetDollars, w.Percent)
	if w.OverBudget {
		return ui.ErrorText.Render(line + " ⚠ over")
	}
	return line
}
