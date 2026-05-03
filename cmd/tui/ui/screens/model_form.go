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

// ModelFormScreen is the create + edit form for a model. Pricing is in cents
// per million tokens (the API's native unit for model pricing).
type ModelFormScreen struct {
	client   *client.Client
	existing *client.Model

	form *huh.Form

	name      string
	inputCpm  string
	outputCpm string

	submitting bool
	err        error
	done       bool
}

func NewModelForm(c *client.Client, existing *client.Model) *ModelFormScreen {
	s := &ModelFormScreen{client: c, existing: existing}
	if existing != nil {
		s.name = existing.Name
		s.inputCpm = strconv.FormatInt(existing.InputPerMillionCents, 10)
		s.outputCpm = strconv.FormatInt(existing.OutputPerMillionCents, 10)
	}
	s.buildForm()
	return s
}

func (s *ModelFormScreen) Name() string { return "model form" }

func (s *ModelFormScreen) buildForm() {
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Required. Must be unique. e.g. \"my-custom-model\".").
				Value(&s.name).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Input price (cents per million tokens)").
				Description("Required. Integer. e.g. \"50\" = $0.50 / 1M input tokens.").
				Value(&s.inputCpm).
				Validate(validateNonNegInt),
			huh.NewInput().
				Title("Output price (cents per million tokens)").
				Description("Required. Integer. e.g. \"200\" = $2.00 / 1M output tokens.").
				Value(&s.outputCpm).
				Validate(validateNonNegInt),
		),
	).WithShowHelp(false).WithShowErrors(true)
}

func (s *ModelFormScreen) Init() tea.Cmd { return s.form.Init() }

type modelSavedMsg struct {
	model *client.Model
	err   error
}

func (s *ModelFormScreen) submit() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		in, _ := strconv.ParseInt(s.inputCpm, 10, 64)
		out, _ := strconv.ParseInt(s.outputCpm, 10, 64)
		input := client.ModelInput{
			Name:                  s.name,
			InputPerMillionCents:  &in,
			OutputPerMillionCents: &out,
		}

		var m *client.Model
		var err error
		if s.existing == nil {
			m, err = s.client.CreateModel(ctx, input)
		} else {
			m, err = s.client.UpdateModel(ctx, s.existing.ID, input)
		}
		return modelSavedMsg{model: m, err: err}
	}
}

func (s *ModelFormScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	if s.done {
		return s, nil
	}
	switch m := msg.(type) {
	case modelSavedMsg:
		s.submitting = false
		if m.err != nil {
			s.err = m.err
			return s, nil
		}
		s.done = true
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

func (s *ModelFormScreen) View() string {
	title := "Create Model"
	if s.existing != nil {
		title = fmt.Sprintf("Edit Model #%d", s.existing.ID)
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

// validateNonNegInt allows non-negative integers. The API rejects negative
// pricing; zero is allowed (e.g. for free-tier models).
func validateNonNegInt(s string) error {
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
