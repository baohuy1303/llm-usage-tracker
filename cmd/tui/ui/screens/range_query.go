package screens

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"llm-usage-tracker/cmd/tui/client"
	"llm-usage-tracker/cmd/tui/ui"
)

// RangeQueryScreen runs ad-hoc range or summary queries. Two modes:
//
//	scope=project: GET /projects/{id}/usage/range  -> RangeStats
//	scope=all:     GET /usage/summary               -> SummaryStats
//
// Form-driven so devs can poke at arbitrary windows from the TUI.
type RangeQueryScreen struct {
	client *client.Client

	form *huh.Form

	loaded   bool
	projects []client.Project

	scope        string // "project" or "all"
	projectIDStr string
	from         string
	to           string

	loading bool
	err     error
	single  *client.RangeStats
	summary *client.SummaryStats
	hasRun  bool
}

func NewRangeQuery(c *client.Client) *RangeQueryScreen {
	now := time.Now().UTC()
	return &RangeQueryScreen{
		client: c,
		scope:  "project",
		from:   now.AddDate(0, 0, -7).Format("2006-01-02"),
		to:     now.Format("2006-01-02"),
	}
}

func (s *RangeQueryScreen) Name() string { return "range" }

type rangeProjectsLoadedMsg struct {
	projects []client.Project
	err      error
}

type rangeResultMsg struct {
	single  *client.RangeStats
	summary *client.SummaryStats
	err     error
}

func (s *RangeQueryScreen) loadProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ps, err := s.client.ListProjects(ctx)
		return rangeProjectsLoadedMsg{projects: ps, err: err}
	}
}

func (s *RangeQueryScreen) Init() tea.Cmd { return s.loadProjects() }

func (s *RangeQueryScreen) buildForm() {
	if len(s.projects) > 0 && s.projectIDStr == "" {
		s.projectIDStr = strconv.FormatInt(s.projects[0].ID, 10)
	}

	projectOpts := make([]huh.Option[string], 0, len(s.projects))
	for _, p := range s.projects {
		projectOpts = append(projectOpts, huh.NewOption(
			fmt.Sprintf("#%d %s", p.ID, p.Name),
			strconv.FormatInt(p.ID, 10),
		))
	}

	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Scope").
				Options(
					huh.NewOption("Single project", "project"),
					huh.NewOption("All projects (summary)", "all"),
				).
				Value(&s.scope),
			huh.NewSelect[string]().
				Title("Project (when scope = single project)").
				Options(projectOpts...).
				Value(&s.projectIDStr),
			huh.NewInput().
				Title("From").
				Description("YYYY-MM-DD or RFC3339.").
				Value(&s.from).
				Validate(validateNonEmpty),
			huh.NewInput().
				Title("To").
				Description("YYYY-MM-DD or RFC3339.").
				Value(&s.to).
				Validate(validateNonEmpty),
		),
	).WithShowHelp(false).WithShowErrors(true)
}

func (s *RangeQueryScreen) runQuery() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if s.scope == "all" {
			summary, err := s.client.AllProjectsSummary(ctx, s.from, s.to)
			return rangeResultMsg{summary: summary, err: err}
		}

		pid, err := strconv.ParseInt(s.projectIDStr, 10, 64)
		if err != nil {
			return rangeResultMsg{err: fmt.Errorf("invalid project id")}
		}
		stats, err := s.client.ProjectRange(ctx, pid, s.from, s.to)
		return rangeResultMsg{single: stats, err: err}
	}
}

func (s *RangeQueryScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case rangeProjectsLoadedMsg:
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.projects = m.projects
		s.loaded = true
		s.buildForm()
		return s, s.form.Init()

	case rangeResultMsg:
		s.loading = false
		s.single = m.single
		s.summary = m.summary
		s.err = m.err
		s.hasRun = true
		// Rebuild the form so the user can run another query (the previous
		// run completed it, freezing all inputs).
		s.buildForm()
		return s, s.form.Init()

	case tea.KeyMsg:
		if m.String() == "?" {
			return s, func() tea.Msg { return ui.PushScreenMsg{Screen: NewHelp()} }
		}
	}

	if !s.loaded {
		return s, nil
	}

	form, cmd := s.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		s.form = f
	}
	if s.form.State == huh.StateCompleted && !s.loading {
		s.loading = true
		s.single = nil
		s.summary = nil
		return s, tea.Batch(cmd, s.runQuery())
	}
	return s, cmd
}

func (s *RangeQueryScreen) View() string {
	out := ui.Title.Render("Range Query") + "\n\n"

	if !s.loaded && s.err == nil {
		return out + ui.StatusBar.Render("loading projects…")
	}
	if !s.loaded && s.err != nil {
		return out + ui.ErrorText.Render(s.err.Error())
	}

	out += s.form.View() + "\n\n"

	if s.loading {
		out += ui.StatusBar.Render("running query…") + "\n"
	}
	if s.err != nil {
		out += ui.ErrorText.Render(s.err.Error()) + "\n"
	}

	if s.single != nil {
		out += ui.SuccessText.Render("Result") + "\n"
		out += fmt.Sprintf("  From:    %s\n", s.single.From)
		out += fmt.Sprintf("  To:      %s\n", s.single.To)
		out += fmt.Sprintf("  Cost:    $%.5f\n", s.single.CostDollars)
		out += fmt.Sprintf("  Tokens:  %d (in %d / out %d)\n", s.single.Tokens, s.single.TokensIn, s.single.TokensOut)
		out += fmt.Sprintf("  Events:  %d\n", s.single.EventCount)
	}
	if s.summary != nil {
		out += ui.SuccessText.Render("Summary") + "\n"
		out += fmt.Sprintf("  From:           %s\n", s.summary.From)
		out += fmt.Sprintf("  To:             %s\n", s.summary.To)
		out += fmt.Sprintf("  Total cost:     $%.5f\n", s.summary.TotalCostDollars)
		out += fmt.Sprintf("  Total tokens:   %d (in %d / out %d)\n", s.summary.TotalTokens, s.summary.TotalTokensIn, s.summary.TotalTokensOut)
		out += fmt.Sprintf("  Total events:   %d\n\n", s.summary.TotalEventCount)
		out += "  Per-project breakdown:\n"
		for _, row := range s.summary.Projects {
			out += fmt.Sprintf("    #%-3d %-25s $%-10s %d events\n",
				row.ProjectID,
				truncate(row.ProjectName, 25),
				fmt.Sprintf("%.5f", row.CostDollars),
				row.EventCount,
			)
		}
	}

	if s.hasRun {
		out += "\n" + ui.HelpDesc.Render("change inputs and submit again to re-run, or esc to leave")
	}
	return out
}

func validateNonEmpty(s string) error {
	if s == "" {
		return fmt.Errorf("required")
	}
	return nil
}
