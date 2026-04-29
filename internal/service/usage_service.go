package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"llm-usage-tracker/internal/cache"
	"llm-usage-tracker/internal/metrics"
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

// All money fields below are dollar floats for the API response. Internal
// storage and Prometheus metrics use millicents — see store.MillicentsToDollars.

type DailyStats struct {
	CostDollars float64 `json:"cost_dollars"`
	Tokens      int64   `json:"tokens"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
}

type MonthlyStats struct {
	CostDollars float64 `json:"cost_dollars"`
	Tokens      int64   `json:"tokens"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
}

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

const (
	eventsDefaultLimit = 30
	eventsMaxLimit     = 100
)

type EventsPage struct {
	Events     []store.Usage `json:"events"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// encodeCursor packs (created_at, id) into an opaque base64 string.
func encodeCursor(t time.Time, id int64) string {
	raw := fmt.Sprintf("%s|%d", t.UTC().Format("2006-01-02 15:04:05"), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (time.Time, int64, error) {
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, errors.New("invalid cursor format")
	}
	t, err := time.Parse("2006-01-02 15:04:05", parts[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor time: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor id: %w", err)
	}
	return t, id, nil
}

// BudgetWindow holds spend and budget for one window. SpentDollars/BudgetDollars
// are derived from the millicent values stored internally.
type BudgetWindow struct {
	SpentDollars  float64 `json:"spent_dollars"`
	BudgetDollars float64 `json:"budget_dollars"`
	Percent       float64 `json:"percent"`
	OverBudget    bool    `json:"over_budget"`
}

type BudgetStatus struct {
	Daily   *BudgetWindow `json:"daily,omitempty"`
	Monthly *BudgetWindow `json:"monthly,omitempty"`
	Total   *BudgetWindow `json:"total,omitempty"`
}

type UsageResult struct {
	*store.Usage
	BudgetStatus *BudgetStatus `json:"budget_status,omitempty"`
}

func cacheGet[T any](fn func() (T, error), op string) (T, bool) {
	val, err := fn()
	if err == nil {
		slog.Debug("cache hit", "op", op)
		metrics.CacheHitsTotal.WithLabelValues(op).Inc()
		return val, true
	}
	if errors.Is(err, redis.Nil) {
		slog.Debug("cache miss", "op", op)
		metrics.CacheMissesTotal.WithLabelValues(op).Inc()
	} else {
		slog.Warn("redis read failed", "op", op, "err", err)
		metrics.RedisErrorsTotal.WithLabelValues(op).Inc()
	}
	var zero T
	return zero, false
}

func budgetSentinel(b *int64) int64 {
	if b == nil {
		return -1
	}
	return *b
}

// percent computes spend/budget*100, rounded to 1 decimal. Returns 0 if budget is 0.
func percent(spent, budget int64) float64 {
	if budget <= 0 {
		return 0
	}
	p := float64(spent) / float64(budget) * 100
	return float64(int64(p*10)) / 10
}

// buildWindow takes spend and budget in millicents and produces a dollar-denominated window.
func buildWindow(spentMillicents, budgetMillicents int64) *BudgetWindow {
	return &BudgetWindow{
		SpentDollars:  store.MillicentsToDollars(spentMillicents),
		BudgetDollars: store.MillicentsToDollars(budgetMillicents),
		Percent:       percent(spentMillicents, budgetMillicents),
		OverBudget:    spentMillicents > budgetMillicents,
	}
}

func (s *UsageService) AddUsage(ctx context.Context, projectID int64, modelName string,
	tokensIn, tokensOut int64, latencyMs *int64, tag string) (*UsageResult, error) {

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

	// Pricing is cents per million tokens. We want millicents (= 1000x more precise
	// than cents). So: cost_millicents = (tokens * cents_per_million * 1000) / 1_000_000
	// = (tokens * cents_per_million) / 1000.
	costMillicents := (tokensIn*model.InputPerMillionCents + tokensOut*model.OutputPerMillionCents) / 1000

	usage := &store.Usage{
		ProjectID:      projectID,
		Model:          modelName,
		TokensIn:       tokensIn,
		TokensOut:      tokensOut,
		CostMillicents: costMillicents,
		LatencyMs:      latencyMs,
		Tag:            tag,
	}

	if err := s.repo.Create(ctx, usage); err != nil {
		return nil, err
	}

	pidLabel := strconv.FormatInt(projectID, 10)
	metrics.UsageEventsTotal.WithLabelValues(pidLabel, modelName).Inc()
	metrics.UsageCostMillicentsTotal.WithLabelValues(pidLabel, modelName).Add(float64(costMillicents))
	metrics.UsageTokensTotal.WithLabelValues(pidLabel, modelName, "in").Add(float64(tokensIn))
	metrics.UsageTokensTotal.WithLabelValues(pidLabel, modelName, "out").Add(float64(tokensOut))
	if latencyMs != nil {
		metrics.LLMCallDurationSeconds.WithLabelValues(modelName).Observe(float64(*latencyMs) / 1000)
	}

	result := &UsageResult{Usage: usage}

	if s.usageCache != nil {
		now := time.Now().UTC()
		snap, err := s.usageCache.IncrUsageWithBudget(
			ctx,
			projectID,
			costMillicents,
			tokensIn,
			tokensOut,
			budgetSentinel(project.DailyBudgetMillicents),
			budgetSentinel(project.MonthlyBudgetMillicents),
			now,
		)
		if err != nil {
			slog.Warn("redis incr failed, invalidating cache keys", "err", err, "project_id", projectID)
			metrics.RedisErrorsTotal.WithLabelValues("incr_usage_lua").Inc()
			if derr := s.usageCache.DeleteUsageKeys(ctx, projectID, now); derr != nil {
				slog.Error("redis key deletion failed, cache may be stale", "err", derr, "project_id", projectID)
				metrics.RedisErrorsTotal.WithLabelValues("delete_usage_keys").Inc()
			}
		} else {
			slog.Debug("cache write ok", "op", "incr_usage", "project_id", projectID, "over_daily", snap.OverDaily, "over_monthly", snap.OverMonthly)
			if snap.OverDaily {
				metrics.BudgetExceededTotal.WithLabelValues(pidLabel, "daily").Inc()
			}
			if snap.OverMonthly {
				metrics.BudgetExceededTotal.WithLabelValues(pidLabel, "monthly").Inc()
			}
			result.BudgetStatus = s.buildBudgetStatusFromSnapshot(ctx, project, snap)
		}
	}

	return result, nil
}

func (s *UsageService) buildBudgetStatusFromSnapshot(ctx context.Context, project *store.Project, snap *cache.BudgetSnapshot) *BudgetStatus {
	status := &BudgetStatus{}

	if project.DailyBudgetMillicents != nil {
		status.Daily = buildWindow(snap.DailyMillicents, *project.DailyBudgetMillicents)
	}
	if project.MonthlyBudgetMillicents != nil {
		status.Monthly = buildWindow(snap.MonthlyMillicents, *project.MonthlyBudgetMillicents)
	}
	if project.TotalBudgetMillicents != nil {
		totalSpent, err := s.repo.SumCostByProject(ctx, project.ID)
		if err != nil {
			slog.Warn("sum cost by project failed", "err", err, "project_id", project.ID)
		} else {
			status.Total = buildWindow(totalSpent, *project.TotalBudgetMillicents)
		}
	}

	return status
}

func (s *UsageService) ComputeBudgetStatus(ctx context.Context, project *store.Project) (*BudgetStatus, error) {
	if project.DailyBudgetMillicents == nil && project.MonthlyBudgetMillicents == nil && project.TotalBudgetMillicents == nil {
		return nil, nil
	}

	status := &BudgetStatus{}
	now := time.Now().UTC()

	if project.DailyBudgetMillicents != nil {
		spent, err := s.getDailyCost(ctx, project.ID, now)
		if err != nil {
			return nil, err
		}
		status.Daily = buildWindow(spent, *project.DailyBudgetMillicents)
	}

	if project.MonthlyBudgetMillicents != nil {
		spent, err := s.getMonthlyCost(ctx, project.ID, now)
		if err != nil {
			return nil, err
		}
		status.Monthly = buildWindow(spent, *project.MonthlyBudgetMillicents)
	}

	if project.TotalBudgetMillicents != nil {
		spent, err := s.repo.SumCostByProject(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		status.Total = buildWindow(spent, *project.TotalBudgetMillicents)
	}

	return status, nil
}

func (s *UsageService) getDailyCost(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetDailyCost(ctx, projectID, date) }, "daily_cost"); ok {
			return cost, nil
		}
	}
	return s.repo.SumCostByDay(ctx, projectID, date)
}

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
				CostDollars: store.MillicentsToDollars(cost),
				Tokens:      in + out,
				TokensIn:    in,
				TokensOut:   out,
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
		CostDollars: store.MillicentsToDollars(cost),
		Tokens:      in + out,
		TokensIn:    in,
		TokensOut:   out,
	}, nil
}

func (s *UsageService) GetMonthlyStats(ctx context.Context, projectID int64, month time.Time) (*MonthlyStats, error) {
	in, out, err := s.repo.SumTokensSplitByMonth(ctx, projectID, month)
	if err != nil {
		return nil, err
	}

	cost, err := s.getMonthlyCost(ctx, projectID, month)
	if err != nil {
		return nil, err
	}

	return &MonthlyStats{
		CostDollars: store.MillicentsToDollars(cost),
		Tokens:      in + out,
		TokensIn:    in,
		TokensOut:   out,
	}, nil
}

func (s *UsageService) GetProjectRangeStats(ctx context.Context, projectID int64, from, to time.Time) (*RangeStats, error) {
	agg, err := s.repo.SumUsageByRange(ctx, projectID, from, to)
	if err != nil {
		return nil, err
	}
	return &RangeStats{
		From:        from.UTC().Format(time.RFC3339),
		To:          to.UTC().Format(time.RFC3339),
		CostDollars: store.MillicentsToDollars(agg.CostMillicents),
		Tokens:      agg.Tokens,
		TokensIn:    agg.TokensIn,
		TokensOut:   agg.TokensOut,
		EventCount:  agg.EventCount,
	}, nil
}

func (s *UsageService) ListEvents(ctx context.Context, projectID *int64, from, to *time.Time, cursor string, limit int) (*EventsPage, error) {
	if limit <= 0 {
		limit = eventsDefaultLimit
	}
	if limit > eventsMaxLimit {
		limit = eventsMaxLimit
	}

	filter := store.ListEventsFilter{
		ProjectID: projectID,
		From:      from,
		To:        to,
		Limit:     limit + 1,
	}

	if cursor != "" {
		t, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, err
		}
		filter.AfterTime = &t
		filter.AfterID = &id
	}

	rows, err := s.repo.ListEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	page := &EventsPage{Events: rows}
	if len(rows) > limit {
		page.Events = rows[:limit]
		page.HasMore = true
		last := page.Events[len(page.Events)-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	if page.Events == nil {
		page.Events = []store.Usage{}
	}
	return page, nil
}

func (s *UsageService) GetAllProjectsSummary(ctx context.Context, from, to time.Time) (*SummaryStats, error) {
	rows, err := s.repo.SumUsageByRangeAllProjects(ctx, from, to)
	if err != nil {
		return nil, err
	}

	projects := make([]ProjectSummaryRow, 0, len(rows))
	var totalCostMillicents, totalTokens, totalTokensIn, totalTokensOut, totalCount int64
	for _, r := range rows {
		projects = append(projects, ProjectSummaryRow{
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
			CostDollars: store.MillicentsToDollars(r.CostMillicents),
			Tokens:      r.Tokens,
			TokensIn:    r.TokensIn,
			TokensOut:   r.TokensOut,
			EventCount:  r.EventCount,
		})
		totalCostMillicents += r.CostMillicents
		totalTokens += r.Tokens
		totalTokensIn += r.TokensIn
		totalTokensOut += r.TokensOut
		totalCount += r.EventCount
	}

	return &SummaryStats{
		From:             from.UTC().Format(time.RFC3339),
		To:               to.UTC().Format(time.RFC3339),
		TotalCostDollars: store.MillicentsToDollars(totalCostMillicents),
		TotalTokens:      totalTokens,
		TotalTokensIn:    totalTokensIn,
		TotalTokensOut:   totalTokensOut,
		TotalEventCount:  totalCount,
		Projects:         projects,
	}, nil
}
