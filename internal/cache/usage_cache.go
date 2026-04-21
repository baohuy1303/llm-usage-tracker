package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	dailyTTL   = 48 * time.Hour
	monthlyTTL = 32 * 24 * time.Hour
)

type UsageCache struct {
	client *redis.Client
}

func NewUsageCache(client *redis.Client) *UsageCache {
	return &UsageCache{client: client}
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

// IncrUsage atomically increments all three counters for a usage event using a pipeline.
func (c *UsageCache) IncrUsage(ctx context.Context, projectID, costCents, tokens int64, at time.Time) error {
	dck := dailyCostKey(projectID, at)
	mck := monthlyCostKey(projectID, at)
	dtk := dailyTokensKey(projectID, at)

	pipe := c.client.Pipeline()
	pipe.IncrBy(ctx, dck, costCents)
	pipe.Expire(ctx, dck, dailyTTL)
	pipe.IncrBy(ctx, mck, costCents)
	pipe.Expire(ctx, mck, monthlyTTL)
	pipe.IncrBy(ctx, dtk, tokens)
	pipe.Expire(ctx, dtk, dailyTTL)
	_, err := pipe.Exec(ctx)
	return err
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
