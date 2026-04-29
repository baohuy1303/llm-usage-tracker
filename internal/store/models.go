package store

import (
	"encoding/json"
	"time"
)

// MillicentsPerDollar is the conversion factor between internal storage
// (millicents) and the dollar floats exposed in the API.
const MillicentsPerDollar = 100_000

// MillicentsToDollars returns a dollar float from a millicent integer.
func MillicentsToDollars(m int64) float64 {
	return float64(m) / MillicentsPerDollar
}

// DollarsToMillicents converts a dollar float (from API input) into millicents.
// Truncates toward zero — fine because input precision is 5 decimals.
func DollarsToMillicents(d float64) int64 {
	return int64(d * MillicentsPerDollar)
}

type Project struct {
	ID                      int64      `json:"id"`
	Name                    string     `json:"name"`
	DailyBudgetMillicents   *int64     `json:"-"`
	MonthlyBudgetMillicents *int64     `json:"-"`
	TotalBudgetMillicents   *int64     `json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
	DeletedAt               *time.Time `json:"deleted_at,omitempty"`
}

// MarshalJSON converts millicent-storage budget fields into dollar floats
// for the API response. omitempty is preserved for nil budgets.
func (p Project) MarshalJSON() ([]byte, error) {
	type Alias Project
	toFloat := func(m *int64) *float64 {
		if m == nil {
			return nil
		}
		v := MillicentsToDollars(*m)
		return &v
	}
	return json.Marshal(&struct {
		Alias
		DailyBudgetDollars   *float64 `json:"daily_budget_dollars,omitempty"`
		MonthlyBudgetDollars *float64 `json:"monthly_budget_dollars,omitempty"`
		TotalBudgetDollars   *float64 `json:"total_budget_dollars,omitempty"`
	}{
		Alias:                Alias(p),
		DailyBudgetDollars:   toFloat(p.DailyBudgetMillicents),
		MonthlyBudgetDollars: toFloat(p.MonthlyBudgetMillicents),
		TotalBudgetDollars:   toFloat(p.TotalBudgetMillicents),
	})
}

type Usage struct {
	ID             int64     `json:"id"`
	ProjectID      int64     `json:"project_id"`
	Model          string    `json:"model"`
	TokensIn       int64     `json:"tokens_in"`
	TokensOut      int64     `json:"tokens_out"`
	CostMillicents int64     `json:"-"`
	LatencyMs      *int64    `json:"latency_ms,omitempty"`
	Tag            string    `json:"tag"`
	CreatedAt      time.Time `json:"created_at"`
}

// MarshalJSON emits the cost as a dollar float in `cost_dollars`.
func (u Usage) MarshalJSON() ([]byte, error) {
	type Alias Usage
	return json.Marshal(&struct {
		Alias
		CostDollars float64 `json:"cost_dollars"`
	}{
		Alias:       Alias(u),
		CostDollars: MillicentsToDollars(u.CostMillicents),
	})
}

type Model struct {
	ID                    int64     `json:"id"`
	Name                  string    `json:"name"`
	InputPerMillionCents  int64     `json:"input_per_million_cents"`
	OutputPerMillionCents int64     `json:"output_per_million_cents"`
	CreatedAt             time.Time `json:"created_at"`
}

// UsageAggregate sums cost (millicents) and tokens over a query window.
type UsageAggregate struct {
	CostMillicents int64
	Tokens         int64
	TokensIn       int64
	TokensOut      int64
	EventCount     int64
}

type ProjectUsageAggregate struct {
	ProjectID   int64
	ProjectName string
	UsageAggregate
}
