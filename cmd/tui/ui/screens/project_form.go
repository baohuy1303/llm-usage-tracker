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

// ProjectFormScreen is the create + edit form for a project. existing == nil
// means create mode; non-nil means edit (PATCH).
type ProjectFormScreen struct {
	client   *client.Client
	existing *client.Project // nil = create mode

	form *huh.Form

	name    string
	daily   string
	monthly string
	total   string

	submitting bool
	err        error
	done       bool
}

func NewProjectForm(c *client.Client, existing *client.Project) *ProjectFormScreen {
	s := &ProjectFormScreen{client: c, existing: existing}
	if existing != nil {
		s.name = existing.Name
		if existing.DailyBudgetDollars != nil {
			s.daily = strconv.FormatFloat(*existing.DailyBudgetDollars, 'f', -1, 64)
		}
		if existing.MonthlyBudgetDollars != nil {
			s.monthly = strconv.FormatFloat(*existing.MonthlyBudgetDollars, 'f', -1, 64)
		}
		if existing.TotalBudgetDollars != nil {
			s.total = strconv.FormatFloat(*existing.TotalBudgetDollars, 'f', -1, 64)
		}
	}
	s.buildForm()
	return s
}

func (s *ProjectFormScreen) Name() string { return "project form" }

func (s *ProjectFormScreen) buildForm() {
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Required. Must be unique.").
				Value(&s.name).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Daily Budget (dollars)").
				Description("Optional. Leave blank for no daily limit.").
				Value(&s.daily).
				Validate(validateOptionalDollar),
			huh.NewInput().
				Title("Monthly Budget (dollars)").
				Description("Optional. Leave blank for no monthly limit.").
				Value(&s.monthly).
				Validate(validateOptionalDollar),
			huh.NewInput().
				Title("Total Budget (dollars)").
				Description("Optional. Leave blank for no all-time cap.").
				Value(&s.total).
				Validate(validateOptionalDollar),
		),
	).WithShowHelp(false).WithShowErrors(true)
}

func (s *ProjectFormScreen) Init() tea.Cmd {
	return s.form.Init()
}

type projectSavedMsg struct {
	project *client.Project
	err     error
}

func (s *ProjectFormScreen) submit() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		input := client.ProjectInput{
			Name:                 s.name,
			DailyBudgetDollars:   parseOptionalDollar(s.daily),
			MonthlyBudgetDollars: parseOptionalDollar(s.monthly),
			TotalBudgetDollars:   parseOptionalDollar(s.total),
		}

		var p *client.Project
		var err error
		if s.existing == nil {
			p, err = s.client.CreateProject(ctx, input)
		} else {
			p, err = s.client.UpdateProject(ctx, s.existing.ID, input)
		}
		return projectSavedMsg{project: p, err: err}
	}
}

func (s *ProjectFormScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	if s.done {
		return s, nil
	}

	switch m := msg.(type) {
	case projectSavedMsg:
		s.submitting = false
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.done = true
		// Pop back to the projects list.
		return s, func() tea.Msg { return ui.PopScreenMsg{} }

	case tea.KeyMsg:
		if m.String() == "esc" && !s.submitting {
			return s, func() tea.Msg { return ui.PopScreenMsg{} }
		}
	}

	form, cmd := s.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		s.form = f
	}

	if s.form.State == huh.StateCompleted && !s.submitting {
		s.submitting = true
		return s, tea.Batch(cmd, s.submit())
	}
	return s, cmd
}

func (s *ProjectFormScreen) View() string {
	title := "Create Project"
	if s.existing != nil {
		title = fmt.Sprintf("Edit Project #%d", s.existing.ID)
	}

	out := ui.Title.Render(title) + "\n\n" + s.form.View()
	if s.submitting {
		out += "\n\n" + ui.StatusBar.Render("submitting…")
	}
	if s.err != nil {
		out += "\n\n" + ui.ErrorText.Render(s.err.Error())
	}
	return out
}

// validateOptionalDollar accepts an empty string ("no value") or a positive
// float. Negative or zero is rejected since the API enforces budget > 0.
func validateOptionalDollar(s string) error {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v <= 0 {
		return fmt.Errorf("must be greater than 0")
	}
	return nil
}

func parseOptionalDollar(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
