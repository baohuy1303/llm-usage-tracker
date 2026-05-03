// Package ui hosts the bubbletea root model and all screens.
//
// Architecture: the App holds one active Screen. Messages flow through App,
// which routes them to the active screen's Update. Screens return commands
// that emit Messages — including navigation messages handled here.
//
// Tab order across the top-level screens is:
//
//	projects → models → events → range → usage
//
// Drill-downs (project_detail, forms, help) are pushed onto a stack and
// popped on Esc.
package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-usage-tracker/cmd/tui/client"
)

// Screen is the interface every UI screen implements. Mirrors bubbletea's
// Model trio with a name for tab bar display.
type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (Screen, tea.Cmd)
	View() string
	Name() string
}

// PushScreenMsg is emitted by a screen to push a new screen on top of itself
// (drill-down). Esc unwinds.
type PushScreenMsg struct{ Screen Screen }

// PopScreenMsg unwinds the topmost screen. Equivalent to Esc.
type PopScreenMsg struct{}

// SwitchTabMsg switches to a different top-level tab.
type SwitchTabMsg struct{ Index int }

// FatalErrMsg is for unrecoverable startup errors (bad BASE_URL, etc.).
type FatalErrMsg struct{ Err error }

// App is the root tea.Model. Holds the API client, the tab bar, the active
// drill-down stack, and the most recent fatal error if any.
type App struct {
	client *client.Client
	keys   GlobalKeys

	tabs       []Screen // top-level: index 0..N-1
	activeTab  int
	stack      []Screen // drill-down stack on top of active tab
	width      int
	height     int
	fatalErr   error
}

func NewApp(c *client.Client, tabs []Screen) *App {
	return &App{
		client:    c,
		keys:      DefaultKeys(),
		tabs:      tabs,
		activeTab: 0,
	}
}

func (a *App) Init() tea.Cmd {
	if len(a.tabs) == 0 {
		return nil
	}
	return a.tabs[0].Init()
}

// active returns the currently visible screen (top of stack, or current tab).
func (a *App) active() Screen {
	if len(a.stack) > 0 {
		return a.stack[len(a.stack)-1]
	}
	return a.tabs[a.activeTab]
}

// setActive replaces the screen at the top of stack/tab.
func (a *App) setActive(s Screen) {
	if len(a.stack) > 0 {
		a.stack[len(a.stack)-1] = s
	} else {
		a.tabs[a.activeTab] = s
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height

	case tea.KeyMsg:
		// Global quit. Even fatal screens respect this.
		switch m.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
		if a.fatalErr != nil {
			return a, nil
		}
		// Esc backs out of a drill-down if any.
		if m.String() == "esc" && len(a.stack) > 0 {
			a.stack = a.stack[:len(a.stack)-1]
			return a, nil
		}
		// Tab cycles top-level tabs (only when no drill-down active).
		if m.String() == "tab" && len(a.stack) == 0 {
			a.activeTab = (a.activeTab + 1) % len(a.tabs)
			return a, a.tabs[a.activeTab].Init()
		}

	case FatalErrMsg:
		a.fatalErr = m.Err
		return a, nil

	case PushScreenMsg:
		a.stack = append(a.stack, m.Screen)
		return a, m.Screen.Init()

	case PopScreenMsg:
		if len(a.stack) > 0 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, nil

	case SwitchTabMsg:
		if m.Index >= 0 && m.Index < len(a.tabs) {
			a.activeTab = m.Index
			a.stack = nil
			return a, a.tabs[a.activeTab].Init()
		}
		return a, nil
	}

	// Forward everything else to the active screen.
	scr, cmd := a.active().Update(msg)
	a.setActive(scr)
	return a, cmd
}

func (a *App) View() string {
	if a.fatalErr != nil {
		return ErrorText.Render("Fatal: ") + a.fatalErr.Error() +
			"\n\n" + StatusBar.Render("Set BASE_URL to point at a running Pulse API. Press q to quit.")
	}

	logo := a.renderLogo()
	header := a.renderTabBar()
	body := a.active().View()
	footer := StatusBar.Render(a.footerHint())

	return lipgloss.JoinVertical(lipgloss.Left, logo, header, body, footer)
}

// pulseLogo is the block-letter wordmark shown at the top of every screen.
// Kept as a single multi-line string so it renders as one block under the
// gradient-style style in renderLogo.
const pulseLogo = `█▀█ █░█ █░░ █▀ █▀▀
█▀▀ █▄█ █▄▄ ▄█ ██▄`

func (a *App) renderLogo() string {
	logoStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)
	tagline := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Italic(true).
		Render("// llm usage tracker")

	left := logoStyle.Render(pulseLogo)
	right := lipgloss.NewStyle().
		Padding(1, 0, 0, 2). // align tagline near the wordmark's baseline
		Render(tagline)

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, left, right)
	return lipgloss.NewStyle().Padding(0, 1).Render(row)
}

func (a *App) renderTabBar() string {
	tabs := make([]string, len(a.tabs))
	for i, s := range a.tabs {
		label := fmt.Sprintf(" %s ", s.Name())
		if i == a.activeTab && len(a.stack) == 0 {
			tabs[i] = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Render(label)
		} else {
			tabs[i] = lipgloss.NewStyle().
				Background(ColorBgSubtle).
				Foreground(ColorMuted).
				Render(label)
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().Padding(0, 1).Render(bar)
}

func (a *App) footerHint() string {
	if len(a.stack) > 0 {
		return HintBar([][2]string{
			{"esc", "back"},
			{"?", "help"},
			{"q", "quit"},
		})
	}
	return HintBar([][2]string{
		{"tab", "next"},
		{"r", "refresh"},
		{"?", "help"},
		{"q", "quit"},
	})
}

// Width and Height expose the terminal size to screens that need to layout against it.
func (a *App) Width() int  { return a.width }
func (a *App) Height() int { return a.height }
