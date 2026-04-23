package store

import "time"

type Project struct {
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	DailyBudgetCents   *int64     `json:"daily_budget_cents,omitempty"`
	MonthlyBudgetCents *int64     `json:"monthly_budget_cents,omitempty"`
	TotalBudgetCents   *int64     `json:"total_budget_cents,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type Usage struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Model     string    `json:"model"`
	TokensIn  int64     `json:"tokens_in"`
	TokensOut int64     `json:"tokens_out"`
	CostCents int64     `json:"cost_cents"`
	LatencyMs int64     `json:"latency_ms"`
	Tag       string    `json:"tag"`
	CreatedAt time.Time `json:"created_at"`
}

type Model struct {
	ID                    int64     `json:"id"`
	Name                  string    `json:"name"`
	InputPerMillionCents  int64     `json:"input_per_million_cents"`
	OutputPerMillionCents int64     `json:"output_per_million_cents"`
	CreatedAt             time.Time `json:"created_at"`
}

type UsageAggregate struct {
	CostCents  int64
	Tokens     int64
	TokensIn   int64
	TokensOut  int64
	EventCount int64
}

type ProjectUsageAggregate struct {
	ProjectID   int64
	ProjectName string
	UsageAggregate
}