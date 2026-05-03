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

// UsageFormScreen posts a usage event manually. Project ID is fixed; the
// model is a dropdown populated on Init from /models. If projectID == 0 (the
// "no project picked" case via the Usage tab), the form also shows a project
// picker.
type UsageFormScreen struct {
	client    *client.Client
	projectID int64

	form   *huh.Form
	loaded bool

	models   []client.Model
	projects []client.Project // only populated when projectID == 0

	// form state
	projectIDStr string
	modelName    string
	tokensIn     string
	tokensOut    string
	latencyMs    string
	tag          string

	submitting bool
	err        error
	result     *client.UsageResult
	done       bool
}

// NewUsageForm builds the form scoped to one project. Pass projectID=0 to
// show a project picker as the first field.
func NewUsageForm(c *client.Client, projectID int64) *UsageFormScreen {
	s := &UsageFormScreen{client: c, projectID: projectID}
	if projectID > 0 {
		s.projectIDStr = strconv.FormatInt(projectID, 10)
	}
	return s
}

func (s *UsageFormScreen) Name() string { return "usage form" }

type formDataLoadedMsg struct {
	models   []client.Model
	projects []client.Project
	err      error
}

func (s *UsageFormScreen) loadFormData() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ms, err := s.client.ListModels(ctx)
		if err != nil {
			return formDataLoadedMsg{err: err}
		}
		var ps []client.Project
		if s.projectID == 0 {
			ps, err = s.client.ListProjects(ctx)
			if err != nil {
				return formDataLoadedMsg{models: ms, err: err}
			}
		}
		return formDataLoadedMsg{models: ms, projects: ps}
	}
}

func (s *UsageFormScreen) Init() tea.Cmd {
	return s.loadFormData()
}

func (s *UsageFormScreen) buildForm() {
	groups := []*huh.Group{}

	if s.projectID == 0 {
		options := make([]huh.Option[string], 0, len(s.projects))
		for _, p := range s.projects {
			options = append(options, huh.NewOption(
				fmt.Sprintf("#%d %s", p.ID, p.Name),
				strconv.FormatInt(p.ID, 10),
			))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Options(options...).
				Value(&s.projectIDStr),
		))
	}

	modelOptions := make([]huh.Option[string], 0, len(s.models))
	for _, m := range s.models {
		modelOptions = append(modelOptions, huh.NewOption(m.Name, m.Name))
	}
	if s.modelName == "" && len(modelOptions) > 0 {
		s.modelName = modelOptions[0].Value
	}

	groups = append(groups, huh.NewGroup(
		huh.NewSelect[string]().
			Title("Model").
			Options(modelOptions...).
			Value(&s.modelName),
		huh.NewInput().
			Title("Tokens In").
			Value(&s.tokensIn).
			Validate(validatePositiveInt),
		huh.NewInput().
			Title("Tokens Out").
			Value(&s.tokensOut).
			Validate(validatePositiveInt),
		huh.NewInput().
			Title("Latency (ms)").
			Description("Optional. Leave blank if not measured.").
			Value(&s.latencyMs).
			Validate(validateOptionalNonNegInt),
		huh.NewInput().
			Title("Tag").
			Description("Optional. e.g. \"chat\", \"summarize\".").
			Value(&s.tag),
	))

	s.form = huh.NewForm(groups...).WithShowHelp(false).WithShowErrors(true)
}

type usageSubmittedMsg struct {
	result *client.UsageResult
	err    error
}

func (s *UsageFormScreen) submit() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pid, err := strconv.ParseInt(s.projectIDStr, 10, 64)
		if err != nil {
			return usageSubmittedMsg{err: fmt.Errorf("invalid project id")}
		}
		in, _ := strconv.ParseInt(s.tokensIn, 10, 64)
		out, _ := strconv.ParseInt(s.tokensOut, 10, 64)

		req := client.AddUsageRequest{
			Model:     s.modelName,
			TokensIn:  in,
			TokensOut: out,
			Tag:       s.tag,
		}
		if s.latencyMs != "" {
			lat, _ := strconv.ParseInt(s.latencyMs, 10, 64)
			req.LatencyMs = &lat
		}

		res, err := s.client.AddUsage(ctx, pid, req)
		return usageSubmittedMsg{result: res, err: err}
	}
}

func (s *UsageFormScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case formDataLoadedMsg:
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.models = m.models
		s.projects = m.projects
		s.loaded = true
		s.buildForm()
		return s, s.form.Init()

	case usageSubmittedMsg:
		s.submitting = false
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.result = m.result
		s.done = true
		// Don't auto-pop — let the user see the budget_status response.
		return s, nil

	case tea.KeyMsg:
		if s.done && m.String() == "enter" {
			return s, func() tea.Msg { return ui.PopScreenMsg{} }
		}
		if m.String() == "esc" && !s.submitting {
			return s, func() tea.Msg { return ui.PopScreenMsg{} }
		}
	}

	if !s.loaded || s.done {
		return s, nil
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

func (s *UsageFormScreen) View() string {
	out := ui.Title.Render("Add Usage Event") + "\n\n"

	if !s.loaded && s.err == nil {
		return out + ui.StatusBar.Render("loading models…")
	}
	if s.err != nil && !s.loaded {
		return out + ui.ErrorText.Render(s.err.Error())
	}

	if s.done && s.result != nil {
		out += ui.SuccessText.Render(fmt.Sprintf("Event #%d created. Cost: $%.5f", s.result.ID, s.result.CostDollars)) + "\n\n"
		if bs := s.result.BudgetStatus; bs != nil {
			out += "Post-write budget status\n"
			out += renderWindow("  Daily:  ", bs.Daily) + "\n"
			out += renderWindow("  Month:  ", bs.Monthly) + "\n"
			out += renderWindow("  Total:  ", bs.Total) + "\n\n"
		}
		out += ui.HelpDesc.Render("press enter to go back")
		return out
	}

	out += s.form.View()
	if s.submitting {
		out += "\n\n" + ui.StatusBar.Render("submitting…")
	}
	if s.err != nil {
		out += "\n\n" + ui.ErrorText.Render(s.err.Error())
	}
	return out
}

func validatePositiveInt(s string) error {
	if s == "" {
		return fmt.Errorf("required")
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be an integer")
	}
	if v < 0 {
		return fmt.Errorf("must be 0 or greater")
	}
	return nil
}

func validateOptionalNonNegInt(s string) error {
	if s == "" {
		return nil
	}
	return validatePositiveInt(s)
}
