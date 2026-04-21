package store

import "time"

type Project struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Budget    int64      `json:"budget"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
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