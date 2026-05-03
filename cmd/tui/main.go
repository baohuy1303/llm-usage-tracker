// Package main is the Pulse TUI entry point.
//
// Build:   go build -o pulse-tui ./cmd/tui
// Run:     ./pulse-tui          (assumes API at http://localhost:8080)
//          BASE_URL=http://my-host:8080 ./pulse-tui
//
// The TUI is a thin HTTP client over the Pulse API. It does not import the
// internal/ packages or open SQLite/Redis directly; everything goes through
// the HTTP surface so the TUI can run alongside (or away from) the API.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	tea "github.com/charmbracelet/bubbletea"

	"llm-usage-tracker/cmd/tui/client"
	"llm-usage-tracker/cmd/tui/ui"
	"llm-usage-tracker/cmd/tui/ui/screens"
)

func main() {
	// .env is optional; in CI/prod the env vars come from the environment.
	_ = godotenv.Load()

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	timeout := 5 * time.Second
	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}

	c := client.New(baseURL, timeout)

	tabs := []ui.Screen{
		screens.NewProjectsList(c),
		screens.NewModelsList(c),
		screens.NewEventsBrowser(c),
		screens.NewRangeQuery(c),
		screens.NewUsageForm(c, 0), // 0 = no project pre-selected; the form shows a project picker
	}

	app := ui.NewApp(c, tabs)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui exited with error: %v\n", err)
		os.Exit(1)
	}
}
