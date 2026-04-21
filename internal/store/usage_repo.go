package store

import (
	"context"
	"database/sql"
	"time"
)

type UsageRepo struct {
	db *sql.DB
}

func NewUsageRepo(db *sql.DB) *UsageRepo {
	return &UsageRepo{db: db}
}

func (r *UsageRepo) Create(ctx context.Context, usage *Usage) error {
	query := `
		INSERT INTO usage_events (project_id, model, tokens_in, tokens_out, cost_cents, latency_ms, tag)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, usage.ProjectID, usage.Model, usage.TokensIn, usage.TokensOut, usage.CostCents, usage.LatencyMs, usage.Tag)
	if err != nil {
		return err
	}
	usage.ID, err = result.LastInsertId()
	return err
}

func (r *UsageRepo) List(ctx context.Context, projectID int64) ([]Usage, error) {
	query := `
		SELECT id, project_id, model, tokens_in, tokens_out, cost_cents, latency_ms, tag, created_at
		FROM usage_events
		WHERE project_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []Usage
	for rows.Next() {
		var u Usage
		if err := rows.Scan(&u.ID, &u.ProjectID, &u.Model, &u.TokensIn, &u.TokensOut, &u.CostCents, &u.LatencyMs, &u.Tag, &u.CreatedAt); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, nil
}

func (r *UsageRepo) SumCostByDay(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_cents), 0) FROM usage_events WHERE project_id = ? AND DATE(created_at) = DATE(?)`,
		projectID, date.Format("2006-01-02"),
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) SumCostByMonth(ctx context.Context, projectID int64, month time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_cents), 0) FROM usage_events WHERE project_id = ? AND strftime('%Y-%m', created_at) = ?`,
		projectID, month.Format("2006-01"),
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) SumTokensByDay(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens_in + tokens_out), 0) FROM usage_events WHERE project_id = ? AND DATE(created_at) = DATE(?)`,
		projectID, date.Format("2006-01-02"),
	).Scan(&total)
	return total, err
}