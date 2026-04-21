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
	usageCache  *cache.UsageCache
}

func NewUsageService(repo *store.UsageRepo, projectRepo *store.ProjectRepo, usageCache *cache.UsageCache) *UsageService {
	return &UsageService{repo: repo, projectRepo: projectRepo, usageCache: usageCache}
}

type DailyStats struct {
	CostCents int64 `json:"cost_cents"`
	Tokens    int64 `json:"tokens"`
}

type MonthlyStats struct {
	CostCents int64 `json:"cost_cents"`
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

func (s *UsageService) AddUsage(ctx context.Context, projectID int64, model string,
	tokensIn, tokensOut, costCents, latencyMs int64, tag string) (*store.Usage, error) {

	usage := &store.Usage{
		ProjectID: projectID,
		Model:     model,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostCents: costCents,
		LatencyMs: latencyMs,
		Tag:       tag,
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := s.repo.Create(ctx, usage); err != nil {
		return nil, err
	}

	if s.usageCache != nil {
		now := time.Now().UTC()
		if err := s.usageCache.IncrUsage(ctx, projectID, costCents, tokensIn+tokensOut, now); err != nil {
			slog.Warn("redis incr failed, invalidating cache keys", "err", err, "project_id", projectID)
			if derr := s.usageCache.DeleteUsageKeys(ctx, projectID, now); derr != nil {
				slog.Error("redis key deletion failed, cache may be stale", "err", derr, "project_id", projectID)
			}
		} else {
			slog.Debug("cache write ok", "op", "incr_usage", "project_id", projectID)
		}
	}

	return usage, nil
}

func (s *UsageService) GetDailyStats(ctx context.Context, projectID int64, date time.Time) (*DailyStats, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetDailyCost(ctx, projectID, date) }, "daily_cost"); ok {
			tokens, _ := cacheGet(func() (int64, error) { return s.usageCache.GetDailyTokens(ctx, projectID, date) }, "daily_tokens")
			return &DailyStats{CostCents: cost, Tokens: tokens}, nil
		}
	}

	cost, err := s.repo.SumCostByDay(ctx, projectID, date)
	if err != nil {
		return nil, err
	}
	tokens, err := s.repo.SumTokensByDay(ctx, projectID, date)
	if err != nil {
		return nil, err
	}
	return &DailyStats{CostCents: cost, Tokens: tokens}, nil
}

func (s *UsageService) GetMonthlyStats(ctx context.Context, projectID int64, month time.Time) (*MonthlyStats, error) {
	if s.usageCache != nil {
		if cost, ok := cacheGet(func() (int64, error) { return s.usageCache.GetMonthlyCost(ctx, projectID, month) }, "monthly_cost"); ok {
			return &MonthlyStats{CostCents: cost}, nil
		}
	}

	cost, err := s.repo.SumCostByMonth(ctx, projectID, month)
	if err != nil {
		return nil, err
	}
	return &MonthlyStats{CostCents: cost}, nil
}
