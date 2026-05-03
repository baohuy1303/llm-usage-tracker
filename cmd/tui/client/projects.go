package client

import (
	"context"
	"time"
)

// Project mirrors the wire shape from /projects endpoints. Money is dollar
// floats. Pointer fields are nullable in the API.
type Project struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	DailyBudgetDollars   *float64   `json:"daily_budget_dollars,omitempty"`
	MonthlyBudgetDollars *float64   `json:"monthly_budget_dollars,omitempty"`
	TotalBudgetDollars   *float64   `json:"total_budget_dollars,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty"`

	// Only present on GET /projects/{id} (the budget_status block).
	BudgetStatus *BudgetStatus `json:"budget_status,omitempty"`
}

// BudgetStatus is the live spend-vs-budget block on Project responses.
// Per-window pointers are nil when the project has no budget for that window.
type BudgetStatus struct {
	Daily   *BudgetWindow `json:"daily,omitempty"`
	Monthly *BudgetWindow `json:"monthly,omitempty"`
	Total   *BudgetWindow `json:"total,omitempty"`
}

type BudgetWindow struct {
	SpentDollars  float64 `json:"spent_dollars"`
	BudgetDollars float64 `json:"budget_dollars"`
	Percent       float64 `json:"percent"`
	OverBudget    bool    `json:"over_budget"`
}

// ProjectInput is the request body shape for create + update. Pointer fields
// translate to JSON null/absent for "no budget enforcement".
type ProjectInput struct {
	Name                 string   `json:"name,omitempty"`
	DailyBudgetDollars   *float64 `json:"daily_budget_dollars,omitempty"`
	MonthlyBudgetDollars *float64 `json:"monthly_budget_dollars,omitempty"`
	TotalBudgetDollars   *float64 `json:"total_budget_dollars,omitempty"`
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.do(ctx, "GET", "/projects", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetProject(ctx context.Context, id int64) (*Project, error) {
	var out Project
	if err := c.do(ctx, "GET", "/projects/"+itoa(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateProject(ctx context.Context, in ProjectInput) (*Project, error) {
	var out Project
	if err := c.do(ctx, "POST", "/projects/create", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateProject(ctx context.Context, id int64, in ProjectInput) (*Project, error) {
	var out Project
	if err := c.do(ctx, "PATCH", "/projects/"+itoa(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteProject(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", "/projects/"+itoa(id), nil, nil)
}
