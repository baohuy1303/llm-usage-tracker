-- Atomic usage increment + budget check.
--
-- KEYS[1] = usage:{pid}:{YYYY-MM-DD}    daily cost counter
-- KEYS[2] = usage:{pid}:{YYYY-MM}       monthly cost counter
-- KEYS[3] = tokens:{pid}:{YYYY-MM-DD}   daily tokens counter
--
-- ARGV[1] = cost_cents to add
-- ARGV[2] = tokens to add
-- ARGV[3] = daily budget cents   (-1 if project has no daily budget)
-- ARGV[4] = monthly budget cents (-1 if project has no monthly budget)
-- ARGV[5] = daily key TTL seconds
-- ARGV[6] = monthly key TTL seconds
--
-- Returns: {over_daily, new_daily, daily_budget, over_monthly, new_monthly, monthly_budget}
-- A -1 budget means "no enforcement" and over_* will be 0.

local cost           = tonumber(ARGV[1])
local tokens         = tonumber(ARGV[2])
local daily_budget   = tonumber(ARGV[3])
local monthly_budget = tonumber(ARGV[4])

local new_daily = redis.call('INCRBY', KEYS[1], cost)
redis.call('EXPIRE', KEYS[1], ARGV[5])

local new_monthly = redis.call('INCRBY', KEYS[2], cost)
redis.call('EXPIRE', KEYS[2], ARGV[6])

redis.call('INCRBY', KEYS[3], tokens)
redis.call('EXPIRE', KEYS[3], ARGV[5])

local over_daily = 0
if daily_budget >= 0 and new_daily > daily_budget then over_daily = 1 end

local over_monthly = 0
if monthly_budget >= 0 and new_monthly > monthly_budget then over_monthly = 1 end

return {over_daily, new_daily, daily_budget, over_monthly, new_monthly, monthly_budget}
