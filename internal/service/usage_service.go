package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"llm-usage-tracker/internal/cache"
	"llm-usage-tracker/internal/store"
)

type UsageService struct {
	repo       *store.UsageRepo
	usageCache *cache.UsageCache
}

func NewUsageService(repo *store.UsageRepo, usageCache *cache.UsageCache) *UsageService {
	return &UsageService{repo: repo, usageCache: usageCache}
}

type DailyStats struct {
	CostCents int64 `json:"cost_cents"`
	Tokens    int64 `json:"tokens"`
}

type MonthlyStats struct {
	CostCents int64 `json:"cost_cents"`
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

	if err := s.repo.Create(ctx, usage); err != nil {
		return nil, err
	}

	if s.usageCache != nil {
		now := time.Now().UTC()
		if err := s.usageCache.IncrUsage(ctx, projectID, costCents, tokensIn+tokensOut, now); err != nil {
			slog.Warn("redis cache update failed", "err", err, "project_id", projectID)
		}
	}

	return usage, nil
}

func (s *UsageService) GetDailyStats(ctx context.Context, projectID int64, date time.Time) (*DailyStats, error) {
	if s.usageCache != nil {
		cost, err := s.usageCache.GetDailyCost(ctx, projectID, date)
		if err != nil && !errors.Is(err, redis.Nil) {
			slog.Warn("redis get daily cost failed", "err", err)
		} else if err == nil {
			tokens, terr := s.usageCache.GetDailyTokens(ctx, projectID, date)
			if terr != nil && !errors.Is(terr, redis.Nil) {
				slog.Warn("redis get daily tokens failed", "err", terr)
			}
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
		cost, err := s.usageCache.GetMonthlyCost(ctx, projectID, month)
		if err != nil && !errors.Is(err, redis.Nil) {
			slog.Warn("redis get monthly cost failed", "err", err)
		} else if err == nil {
			return &MonthlyStats{CostCents: cost}, nil
		}
	}

	cost, err := s.repo.SumCostByMonth(ctx, projectID, month)
	if err != nil {
		return nil, err
	}
	return &MonthlyStats{CostCents: cost}, nil
}
