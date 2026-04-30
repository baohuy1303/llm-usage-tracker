package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// sqliteTimeFormat matches how SQLite stores CURRENT_TIMESTAMP values,
// so string comparisons in WHERE ... BETWEEN work correctly.
const sqliteTimeFormat = "2006-01-02 15:04:05"

type UsageRepo struct {
	db *sql.DB
}

func NewUsageRepo(db *sql.DB) *UsageRepo {
	return &UsageRepo{db: db}
}

func (r *UsageRepo) Create(ctx context.Context, usage *Usage) error {
	query := `
		INSERT INTO usage_events (project_id, model, tokens_in, tokens_out, cost_millicents, latency_ms, tag)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, usage.ProjectID, usage.Model, usage.TokensIn, usage.TokensOut, usage.CostMillicents, nullIntFromPtr(usage.LatencyMs), usage.Tag)
	if err != nil {
		return err
	}
	usage.ID, err = result.LastInsertId()
	return err
}

// ListEventsFilter bundles the optional filters for ListEvents.
// Any nil field means "no filter". AfterTime+AfterID together form a keyset cursor
// for pagination — the next page starts strictly after (AfterTime, AfterID) in
// (created_at DESC, id DESC) order.
type ListEventsFilter struct {
	ProjectID *int64
	From      *time.Time
	To        *time.Time
	AfterTime *time.Time
	AfterID   *int64
	Limit     int
}

func (r *UsageRepo) ListEvents(ctx context.Context, f ListEventsFilter) ([]Usage, error) {
	var q strings.Builder
	q.WriteString(`SELECT id, project_id, model, tokens_in, tokens_out, cost_millicents, latency_ms, tag, created_at
		FROM usage_events WHERE 1=1`)

	args := []any{}
	if f.ProjectID != nil {
		q.WriteString(" AND project_id = ?")
		args = append(args, *f.ProjectID)
	}
	if f.From != nil {
		q.WriteString(" AND created_at >= ?")
		args = append(args, f.From.UTC().Format(sqliteTimeFormat))
	}
	if f.To != nil {
		q.WriteString(" AND created_at <= ?")
		args = append(args, f.To.UTC().Format(sqliteTimeFormat))
	}
	if f.AfterTime != nil && f.AfterID != nil {
		// Keyset predicate: strictly past (AfterTime, AfterID) in DESC order.
		q.WriteString(" AND (created_at < ? OR (created_at = ? AND id < ?))")
		t := f.AfterTime.UTC().Format(sqliteTimeFormat)
		args = append(args, t, t, *f.AfterID)
	}
	q.WriteString(" ORDER BY created_at DESC, id DESC LIMIT ?")
	args = append(args, f.Limit)

	rows, err := r.db.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []Usage
	for rows.Next() {
		var u Usage
		var latency sql.NullInt64
		if err := rows.Scan(&u.ID, &u.ProjectID, &u.Model, &u.TokensIn, &u.TokensOut, &u.CostMillicents, &latency, &u.Tag, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.LatencyMs = ptrFromNullInt(latency)
		usages = append(usages, u)
	}
	return usages, nil
}

// RecentLatencyStats holds an average latency over the latest N events.
// AvgLatencyMs is nil when no events in the window have a non-null latency_ms.
type RecentLatencyStats struct {
	AvgLatencyMs      *float64
	EventsWithLatency int64
	EventsTotal       int64
}

// GetRecentLatencyStats computes the mean latency over the latest `limit` events
// for a project. SQLite's AVG() natively ignores NULLs, so events without a
// recorded latency don't break the math.
func (r *UsageRepo) GetRecentLatencyStats(ctx context.Context, projectID int64, limit int) (*RecentLatencyStats, error) {
	query := `
		SELECT AVG(latency_ms), COUNT(latency_ms), COUNT(*)
		FROM (
			SELECT latency_ms
			FROM usage_events
			WHERE project_id = ?
			ORDER BY created_at DESC, id DESC
			LIMIT ?
		)`
	var avg sql.NullFloat64
	var withLatency, total int64
	err := r.db.QueryRowContext(ctx, query, projectID, limit).Scan(&avg, &withLatency, &total)
	if err != nil {
		return nil, err
	}
	stats := &RecentLatencyStats{
		EventsWithLatency: withLatency,
		EventsTotal:       total,
	}
	if avg.Valid {
		v := avg.Float64
		stats.AvgLatencyMs = &v
	}
	return stats, nil
}

// UsageMetricsRow is one row from AggregateForMetrics: cumulative event counts,
// cost, and tokens grouped by (project_id, model). Used to rehydrate the
// Prometheus counters from SQL on startup so dashboards stay consistent across
// app restarts.
type UsageMetricsRow struct {
	ProjectID      int64
	Model          string
	EventCount     int64
	CostMillicents int64
	TokensIn       int64
	TokensOut      int64
}

// AggregateForMetrics returns one row per (project_id, model) with all-time
// cumulative totals. Includes events from soft-deleted projects intentionally
// since cumulative queries should reflect history.
func (r *UsageRepo) AggregateForMetrics(ctx context.Context) ([]UsageMetricsRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			project_id,
			model,
			COUNT(*),
			COALESCE(SUM(cost_millicents), 0),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0)
		FROM usage_events
		GROUP BY project_id, model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UsageMetricsRow
	for rows.Next() {
		var row UsageMetricsRow
		if err := rows.Scan(&row.ProjectID, &row.Model, &row.EventCount, &row.CostMillicents, &row.TokensIn, &row.TokensOut); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}


func (r *UsageRepo) SumCostByDay(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_millicents), 0) FROM usage_events WHERE project_id = ? AND DATE(created_at) = DATE(?)`,
		projectID, date.Format("2006-01-02"),
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) SumCostByMonth(ctx context.Context, projectID int64, month time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_millicents), 0) FROM usage_events WHERE project_id = ? AND strftime('%Y-%m', created_at) = ?`,
		projectID, month.Format("2006-01"),
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) SumTokensSplitByDay(ctx context.Context, projectID int64, date time.Time) (int64, int64, error) {
	var in, out int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0)
		 FROM usage_events
		 WHERE project_id = ? AND DATE(created_at) = DATE(?)`,
		projectID, date.Format("2006-01-02"),
	).Scan(&in, &out)
	return in, out, err
}

func (r *UsageRepo) SumTokensSplitByMonth(ctx context.Context, projectID int64, month time.Time) (int64, int64, error) {
	var in, out int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0)
		 FROM usage_events
		 WHERE project_id = ? AND strftime('%Y-%m', created_at) = ?`,
		projectID, month.Format("2006-01"),
	).Scan(&in, &out)
	return in, out, err
}

func (r *UsageRepo) SumCostByProject(ctx context.Context, projectID int64) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_millicents), 0) FROM usage_events WHERE project_id = ?`,
		projectID,
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) SumUsageByRange(ctx context.Context, projectID int64, from, to time.Time) (*UsageAggregate, error) {
	var agg UsageAggregate
	err := r.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(SUM(cost_millicents), 0),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0),
			COUNT(*)
		 FROM usage_events
		 WHERE project_id = ?
		   AND created_at BETWEEN ? AND ?`,
		projectID, from.UTC().Format(sqliteTimeFormat), to.UTC().Format(sqliteTimeFormat),
	).Scan(&agg.CostMillicents, &agg.TokensIn, &agg.TokensOut, &agg.EventCount)
	if err != nil {
		return nil, err
	}
	agg.Tokens = agg.TokensIn + agg.TokensOut
	return &agg, nil
}

func (r *UsageRepo) SumUsageByRangeAllProjects(ctx context.Context, from, to time.Time) ([]ProjectUsageAggregate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			p.id,
			p.name,
			SUM(u.cost_millicents),
			SUM(u.tokens_in),
			SUM(u.tokens_out),
			COUNT(u.id)
		 FROM usage_events u
		 JOIN projects p ON p.id = u.project_id
		 WHERE u.created_at BETWEEN ? AND ?
		 GROUP BY p.id, p.name
		 ORDER BY SUM(u.cost_millicents) DESC`,
		from.UTC().Format(sqliteTimeFormat), to.UTC().Format(sqliteTimeFormat),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProjectUsageAggregate
	for rows.Next() {
		var p ProjectUsageAggregate
		if err := rows.Scan(&p.ProjectID, &p.ProjectName, &p.CostMillicents, &p.TokensIn, &p.TokensOut, &p.EventCount); err != nil {
			return nil, err
		}
		p.Tokens = p.TokensIn + p.TokensOut
		result = append(result, p)
	}
	return result, nil
}