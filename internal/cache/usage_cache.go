package cache

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	dailyTTL   = 48 * time.Hour
	monthlyTTL = 32 * 24 * time.Hour
)

//go:embed scripts/incr_usage.lua
var incrUsageSrc string

var incrUsageScript = redis.NewScript(incrUsageSrc)

type UsageCache struct {
	client *redis.Client
}

func NewUsageCache(client *redis.Client) *UsageCache {
	return &UsageCache{client: client}
}

// BudgetSnapshot is the raw result of the Lua script.
// DailyBudget / MonthlyBudget are -1 when the project has no budget set for that window.
type BudgetSnapshot struct {
	OverDaily     bool
	DailyCents    int64
	DailyBudget   int64
	OverMonthly   bool
	MonthlyCents  int64
	MonthlyBudget int64
}

func dailyCostKey(projectID int64, date time.Time) string {
	return fmt.Sprintf("usage:%d:%s", projectID, date.Format("2006-01-02"))
}

func monthlyCostKey(projectID int64, month time.Time) string {
	return fmt.Sprintf("usage:%d:%s", projectID, month.Format("2006-01"))
}

func dailyTokensKey(projectID int64, date time.Time) string {
	return fmt.Sprintf("tokens:%d:%s", projectID, date.Format("2006-01-02"))
}

// IncrUsageWithBudget atomically increments the daily/monthly cost and daily token
// counters, then checks daily and monthly spend against the project's budgets.
// Pass -1 for a budget value to skip enforcement for that window.
func (c *UsageCache) IncrUsageWithBudget(
	ctx context.Context,
	projectID, costCents, tokens, dailyBudget, monthlyBudget int64,
	at time.Time,
) (*BudgetSnapshot, error) {
	keys := []string{
		dailyCostKey(projectID, at),
		monthlyCostKey(projectID, at),
		dailyTokensKey(projectID, at),
	}
	args := []any{
		costCents,
		tokens,
		dailyBudget,
		monthlyBudget,
		int64(dailyTTL.Seconds()),
		int64(monthlyTTL.Seconds()),
	}

	raw, err := incrUsageScript.Run(ctx, c.client, keys, args...).Slice()
	if err != nil {
		return nil, err
	}
	if len(raw) != 6 {
		return nil, fmt.Errorf("unexpected lua result length: %d", len(raw))
	}

	int64At := func(i int) int64 {
		if v, ok := raw[i].(int64); ok {
			return v
		}
		return 0
	}

	return &BudgetSnapshot{
		OverDaily:     int64At(0) == 1,
		DailyCents:    int64At(1),
		DailyBudget:   int64At(2),
		OverMonthly:   int64At(3) == 1,
		MonthlyCents:  int64At(4),
		MonthlyBudget: int64At(5),
	}, nil
}

// GetDailyCost returns cached cost cents for a project on a given day.
// Returns (0, redis.Nil) on cache miss.
func (c *UsageCache) GetDailyCost(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	return c.client.Get(ctx, dailyCostKey(projectID, date)).Int64()
}

// GetMonthlyCost returns cached cost cents for a project in a given month.
// Returns (0, redis.Nil) on cache miss.
func (c *UsageCache) GetMonthlyCost(ctx context.Context, projectID int64, month time.Time) (int64, error) {
	return c.client.Get(ctx, monthlyCostKey(projectID, month)).Int64()
}

// GetDailyTokens returns cached total tokens for a project on a given day.
// Returns (0, redis.Nil) on cache miss.
func (c *UsageCache) GetDailyTokens(ctx context.Context, projectID int64, date time.Time) (int64, error) {
	return c.client.Get(ctx, dailyTokensKey(projectID, date)).Int64()
}

// DeleteUsageKeys removes all three counters for the given project and day.
// Used to invalidate a potentially partial write so reads fall back to SQL.
func (c *UsageCache) DeleteUsageKeys(ctx context.Context, projectID int64, at time.Time) error {
	return c.client.Del(ctx,
		dailyCostKey(projectID, at),
		monthlyCostKey(projectID, at),
		dailyTokensKey(projectID, at),
	).Err()
}
