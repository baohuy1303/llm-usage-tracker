package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"llm-usage-tracker/internal/cache"
	"llm-usage-tracker/internal/store"
)

type UsageService struct {
	repo        *store.UsageRepo
	projectRepo *store.ProjectRepo
	modelRepo   *store.ModelRepo
	usageCache  *cache.UsageCache
}

func NewUsageService(repo *store.UsageRepo, projectRepo *store.ProjectRepo, modelRepo *store.ModelRepo, usageCache *cache.UsageCache) *UsageService {
	return &UsageService{repo: repo, projectRepo: projectRepo, modelRepo: modelRepo, usageCache: usageCache}
}

type DailyStats struct {
	CostCents int64 `json:"cost_cents"`
	Tokens    int64 `json:"tokens"`
	TokensIn  int64 `json:"tokens_in"`
	TokensOut int64 `json:"tokens_out"`
}

type MonthlyStats struct {
	CostCents int64 `json:"cost_cents"`
	Tokens    int64 `json:"tokens"`
	TokensIn  int64 `json:"tokens_in"`
	TokensOut int64 `json:"tokens_out"`
}

type RangeStats struct {
	From       string `json:"from"`
	To         string `json:"to"`
	CostCents  int64  `json:"cost_cents"`
	Tokens     int64  `json:"tokens"`
	TokensIn   int64  `json:"tokens_in"`
	TokensOut  int64  `json:"tokens_out"`
	EventCount int64  `json:"event_count"`
}

type ProjectSummaryRow struct {
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	CostCents   int64  `json:"cost_cents"`
	Tokens      int64  `json:"tokens"`
	TokensIn    int64  `json:"tokens_in"`
	TokensOut   int64  `json:"tokens_out"`
	EventCount  int64  `json:"event_count"`
}

type SummaryStats struct {
	From             string              `json:"from"`
	To               string              `json:"to"`
	TotalCostCents   int64               `json:"total_cost_cents"`
	TotalTokens      int64               `json:"total_tokens"`
	TotalTokensIn    int64               `json:"total_tokens_in"`
	TotalTokensOut   int64               `json:"total_tokens_out"`
	TotalEventCount  int64               `json:"total_event_count"`
	Projects         []ProjectSummaryRow `json:"projects"`
}

// BudgetWindow is the spend/budget/percent/flag for one window (daily, monthly, or total).
type BudgetWindow struct {
	SpentCents  int64   `json:"spent_cents"`
	BudgetCents int64   `json:"budget_cents"`
	Percent     float64 `json:"percent"`
	OverBudget  bool    `json:"over_budget"`
}

// BudgetStatus groups the three budget windows. Any window is nil when the
// project has no budget set for it.
type BudgetStatus struct {
	Daily   *BudgetWindow `json:"daily,omitempty"`
	Monthly *BudgetWindow `json:"monthly,omitempty"`
	Total   *BudgetWindow `json:"total,omitempty"`
}

// UsageResult is the response from AddUsage: the stored usage event plus
// the post-write budget status. BudgetStatus is nil if Redis was unavailable.
type UsageResult struct {
	*store.Usage
	BudgetStatus *BudgetStatus `json:"budget_status,omitempty"`
}

// cacheGet calls fn and returns (value, true) on a cache hit, or (zero, false) on miss or error.
// Hits, misses, and real errors are all logged so call sites don't have to.
func cacheGet[T any](fn func() (T, error), op string) (T, bool) {
	val, err := fn()
	if err == nil {
		slog.Debug("cache hit", "op", op)
		return val, true
	}
	if errors.Is(err, redis.Nil) {
		slog.Debug("cache miss", "op", op)
	} else {
		slog.Warn("redis read failed", "op", op, "err", err)
	}
	var zero T
	return zero, false
}

// budgetSentinel returns the int64 value for the Lua script: the budget if set,
// or -1 to signal "no enforcement for this window".
func budgetSentinel(b *int64) int64 {
	if b == nil {
		return -1
	}
	return *b
}

// percent computes spend/budget*100, rounded to 1 decimal place. Returns 0 if budget is 0.
func percent(spent, budget int64) float64 {
	if budget <= 0 {
		return 0
	}
	p := float64(spent) / float64(budget) * 100
	return float64(int64(p*10)) / 10
}

func buildWindow(spent, budget int64) *BudgetWindow {
	return &BudgetWindow{
		SpentCents:  spent,
		BudgetCents: budget,
		Percent:     percent(spent, budget),
		OverBudget:  spent > budget,
	}
}

func (s *UsageService) AddUsage(ctx context.Context, projectID int64, modelName string,
	tokensIn, tokensOut, latencyMs int64, tag string) (*UsageResult, error) {

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	model, err := s.modelRepo.GetByName(ctx, modelName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrModelNotFound
		}
		return nil, err
	}

	costCents := (tokensIn*model.InputPerMillionCents + tokensOut*model.OutputPerMillionCents) / 1_000_000

	usage := &store.Usage{
		ProjectID: projectID,
		Model:     modelName,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostCents: costCents,
		LatencyMs: latencyMs,
		Tag:       tag,
	}

	if err := s.repo.Create(ctx, usage); err != nil {
		return nil, err
	}

	result := &UsageResult{Usage: usage}

	if s.usageCache != nil {
		now := time.Now().UTC()
		snap, err := s.usageCache.IncrUsageWithBudget(
			ctx,
			projectID,
			costCents,
			tokensIn,
			tokensOut,
			budgetSentinel(project.DailyBudgetCents),
			budgetSentinel(project.MonthlyBudgetCents),
			now,
		)
		if err != nil {
			slog.Warn("redis incr failed, invalidating cache keys", "err", err, "project_id", projectID)
			if derr := s.usageCache.DeleteUsageKeys(ctx, projectID, now); derr != nil {
				slog.Error("redis key deletion failed, cache may be stale", "err", derr, "project_id", projectID)
			}
		} else {
			slog.Debug("cache write ok", "op", "incr_usage", "project_id", projectID, "over_daily", snap.OverDaily, "over_monthly", snap.OverMonthly)
			result.BudgetStatus = s.buildBudgetStatusFromSnapshot(ctx, project, snap)
		}
	}

	return result, nil
}

// buildBudgetStatusFromSnapshot builds a BudgetStatus using the Lua script result
// for daily/monthly (zero extra round-trips) and SQL for total (always authoritative).
func (s *UsageService) buildBudgetStatusFromSnapshot(ctx context.Context, project *store.Project, snap *cache.BudgetSnapshot) *BudgetStatus {
	status := &BudgetStatus{}

	if project.DailyBudgetCents != nil {
		status.Daily = buildWindow(snap.DailyCents, *project.DailyBudgetCents)
	}
	if project.MonthlyBudgetCents != nil {
		status.Monthly = buildWindow(snap.MonthlyCents, *project.MonthlyBudgetCents)
	}
	if project.TotalBudgetCents != nil {
		totalSpent, err := s.repo.SumCostByProject(ctx, project.ID)
		if err != nil {
			slog.Warn("sum cost by project failed", "err", err, "project_id", project.ID)
		} else {
			status.Total = buildWindow(totalSpent, *project.TotalBudgetCents)
		}
	}

	return status
}

// ComputeBudgetStatus computes the current budget status for a project without
// writing anything. Used by GET /projects/{id}. Reads daily/monthly from cache
// (SQL fallback on miss) and total always from SQL.
func (s *UsageService) ComputeBudgetStatus(ctx context.Context, project *store.Project) (*BudgetStatus, error) {
	if project.DailyBudgetCents == nil && project.MonthlyBudgetCents == nil && project.TotalBudgetCents == nil {
		return nil, nil
	}

	status := &BudgetStatus{}
	now := time.Now().UTC()

	if project.DailyBudgetCents != nil {
		spent, err := s.getDailyCost(ctx, project.ID, now)
		if err != nil {
			return nil, err
		}
		status.Daily = buildWindow(spent, *project.DailyBudgetCents)
	}

	if project.MonthlyBudgetCents != nil {
		spent, err := s.getMonthlyCost(ctx, project.ID, now)
		if err != nil {
			return nil, err
		}
		status.Monthly = buildWindow(spent, *project.MonthlyBudgetCents)
	}

	if project.TotalBudgetCents != nil {
		spent, err := s.repo.SumCostByProject(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		status.Total = buildWindow(spent, *project.TotalBudgetCents)
	}

	return status, nil
}

// getDailyCost tries Redis then falls back to SQL.
func (s *UsageService) getDailyCost(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetDailyCost(ctx, projectID, date) }, "daily_cost"); ok {
			return cost, nil
		}
	}
	return s.repo.SumCostByDay(ctx, projectID, date)
}

// getMonthlyCost tries Redis then falls back to SQL.
func (s *UsageService) getMonthlyCost(ctx context.Context, projectID int64, month time.Time) (int64, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetMonthlyCost(ctx, projectID, month) }, "monthly_cost"); ok {
			return cost, nil
		}
	}
	return s.repo.SumCostByMonth(ctx, projectID, month)
}

func (s *UsageService) GetDailyStats(ctx context.Context, projectID int64, date time.Time) (*DailyStats, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetDailyCost(ctx, projectID, date) }, "daily_cost"); ok {
			in, out, err := s.usageCache.GetDailyTokensSplit(ctx, projectID, date)
			if err != nil && !errors.Is(err, redis.Nil) {
				slog.Warn("redis read failed", "op", "daily_tokens_split", "err", err)
			}
			return &DailyStats{
				CostCents: cost,
				Tokens:    in + out,
				TokensIn:  in,
				TokensOut: out,
			}, nil
		}
	}

	cost, err := s.repo.SumCostByDay(ctx, projectID, date)
	if err != nil {
		return nil, err
	}
	in, out, err := s.repo.SumTokensSplitByDay(ctx, projectID, date)
	if err != nil {
		return nil, err
	}
	return &DailyStats{
		CostCents: cost,
		Tokens:    in + out,
		TokensIn:  in,
		TokensOut: out,
	}, nil
}

func (s *UsageService) GetMonthlyStats(ctx context.Context, projectID int64, month time.Time) (*MonthlyStats, error) {
	// Monthly tokens are not cached (not a hot path) — always SQL.
	in, out, err := s.repo.SumTokensSplitByMonth(ctx, projectID, month)
	if err != nil {
		return nil, err
	}

	cost, err := s.getMonthlyCost(ctx, projectID, month)
	if err != nil {
		return nil, err
	}

	return &MonthlyStats{
		CostCents: cost,
		Tokens:    in + out,
		TokensIn:  in,
		TokensOut: out,
	}, nil
}

func (s *UsageService) GetProjectRangeStats(ctx context.Context, projectID int64, from, to time.Time) (*RangeStats, error) {
	agg, err := s.repo.SumUsageByRange(ctx, projectID, from, to)
	if err != nil {
		return nil, err
	}
	return &RangeStats{
		From:       from.UTC().Format(time.RFC3339),
		To:         to.UTC().Format(time.RFC3339),
		CostCents:  agg.CostCents,
		Tokens:     agg.Tokens,
		TokensIn:   agg.TokensIn,
		TokensOut:  agg.TokensOut,
		EventCount: agg.EventCount,
	}, nil
}

func (s *UsageService) GetAllProjectsSummary(ctx context.Context, from, to time.Time) (*SummaryStats, error) {
	rows, err := s.repo.SumUsageByRangeAllProjects(ctx, from, to)
	if err != nil {
		return nil, err
	}

	projects := make([]ProjectSummaryRow, 0, len(rows))
	var totalCost, totalTokens, totalTokensIn, totalTokensOut, totalCount int64
	for _, r := range rows {
		projects = append(projects, ProjectSummaryRow{
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
			CostCents:   r.CostCents,
			Tokens:      r.Tokens,
			TokensIn:    r.TokensIn,
			TokensOut:   r.TokensOut,
			EventCount:  r.EventCount,
		})
		totalCost += r.CostCents
		totalTokens += r.Tokens
		totalTokensIn += r.TokensIn
		totalTokensOut += r.TokensOut
		totalCount += r.EventCount
	}

	return &SummaryStats{
		From:            from.UTC().Format(time.RFC3339),
		To:              to.UTC().Format(time.RFC3339),
		TotalCostCents:  totalCost,
		TotalTokens:     totalTokens,
		TotalTokensIn:   totalTokensIn,
		TotalTokensOut:  totalTokensOut,
		TotalEventCount: totalCount,
		Projects:        projects,
	}, nil
}
