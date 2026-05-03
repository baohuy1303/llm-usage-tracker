package client

import (
	"context"
	"time"
)

// Usage mirrors the wire shape of an event row. cost_dollars comes from the
// server's MarshalJSON on store.Usage; latency_ms is nullable.
type Usage struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Model       string    `json:"model"`
	TokensIn    int64     `json:"tokens_in"`
	TokensOut   int64     `json:"tokens_out"`
	CostDollars float64   `json:"cost_dollars"`
	LatencyMs   *int64    `json:"latency_ms"`
	Tag         string    `json:"tag"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddUsageRequest is the body for POST /projects/{id}/usage.
type AddUsageRequest struct {
	Model     string `json:"model"`
	TokensIn  int64  `json:"tokens_in"`
	TokensOut int64  `json:"tokens_out"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
	Tag       string `json:"tag,omitempty"`
}

// UsageResult wraps the created Usage plus the post-write budget snapshot.
type UsageResult struct {
	Usage
	BudgetStatus *BudgetStatus `json:"budget_status,omitempty"`
}

// EventsPage is the cursor-paginated event listing response.
type EventsPage struct {
	Events     []Usage `json:"events"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

// RangeStats is the aggregate response for /projects/{id}/usage/range.
type RangeStats struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	CostDollars float64 `json:"cost_dollars"`
	Tokens      int64   `json:"tokens"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
	EventCount  int64   `json:"event_count"`
}

type ProjectSummaryRow struct {
	ProjectID   int64   `json:"project_id"`
	ProjectName string  `json:"project_name"`
	CostDollars float64 `json:"cost_dollars"`
	Tokens      int64   `json:"tokens"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
	EventCount  int64   `json:"event_count"`
}

// SummaryStats is the response for /usage/summary.
type SummaryStats struct {
	From             string              `json:"from"`
	To               string              `json:"to"`
	TotalCostDollars float64             `json:"total_cost_dollars"`
	TotalTokens      int64               `json:"total_tokens"`
	TotalTokensIn    int64               `json:"total_tokens_in"`
	TotalTokensOut   int64               `json:"total_tokens_out"`
	TotalEventCount  int64               `json:"total_event_count"`
	Projects         []ProjectSummaryRow `json:"projects"`
}

func (c *Client) AddUsage(ctx context.Context, projectID int64, in AddUsageRequest) (*UsageResult, error) {
	var out UsageResult
	if err := c.do(ctx, "POST", "/projects/"+itoa(projectID)+"/usage", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListProjectEvents fetches a single page of events for one project.
// from/to/cursor may be empty. limit==0 lets the server use its default (30).
func (c *Client) ListProjectEvents(ctx context.Context, projectID int64, from, to, cursor string, limit int) (*EventsPage, error) {
	params := map[string]string{"from": from, "to": to, "cursor": cursor}
	if limit > 0 {
		params["limit"] = itoa(int64(limit))
	}
	var out EventsPage
	if err := c.do(ctx, "GET", "/projects/"+itoa(projectID)+"/usage/events"+queryString(params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAllEvents fetches a single page of events across all projects.
func (c *Client) ListAllEvents(ctx context.Context, from, to, cursor string, limit int) (*EventsPage, error) {
	params := map[string]string{"from": from, "to": to, "cursor": cursor}
	if limit > 0 {
		params["limit"] = itoa(int64(limit))
	}
	var out EventsPage
	if err := c.do(ctx, "GET", "/usage/events"+queryString(params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ProjectRange queries an arbitrary date window for one project.
func (c *Client) ProjectRange(ctx context.Context, projectID int64, from, to string) (*RangeStats, error) {
	params := map[string]string{"from": from, "to": to}
	var out RangeStats
	if err := c.do(ctx, "GET", "/projects/"+itoa(projectID)+"/usage/range"+queryString(params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AllProjectsSummary queries an arbitrary date window across all projects.
func (c *Client) AllProjectsSummary(ctx context.Context, from, to string) (*SummaryStats, error) {
	params := map[string]string{"from": from, "to": to}
	var out SummaryStats
	if err := c.do(ctx, "GET", "/usage/summary"+queryString(params), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
