-- Atomic usage increment + budget check.
--
-- KEYS[1] = usage:{pid}:{YYYY-MM-DD}    daily cost counter (string, INCRBY)
-- KEYS[2] = usage:{pid}:{YYYY-MM}       monthly cost counter (string, INCRBY)
-- KEYS[3] = tokens:{pid}:{YYYY-MM-DD}   daily tokens counter (hash with "in"/"out" fields)
--
-- ARGV[1] = cost_cents to add
-- ARGV[2] = tokens_in to add
-- ARGV[3] = tokens_out to add
-- ARGV[4] = daily budget cents   (-1 if project has no daily budget)
-- ARGV[5] = monthly budget cents (-1 if project has no monthly budget)
-- ARGV[6] = daily key TTL seconds
-- ARGV[7] = monthly key TTL seconds
--
-- Returns: {over_daily, new_daily, daily_budget, over_monthly, new_monthly, monthly_budget}
-- A -1 budget means "no enforcement" and over_* will be 0.

local cost           = tonumber(ARGV[1])
local daily_budget   = tonumber(ARGV[4])
local monthly_budget = tonumber(ARGV[5])

local new_daily = redis.call('INCRBY', KEYS[1], cost)
redis.call('EXPIRE', KEYS[1], ARGV[6])

local new_monthly = redis.call('INCRBY', KEYS[2], cost)
redis.call('EXPIRE', KEYS[2], ARGV[7])

redis.call('HINCRBY', KEYS[3], 'in',  ARGV[2])
redis.call('HINCRBY', KEYS[3], 'out', ARGV[3])
redis.call('EXPIRE',  KEYS[3], ARGV[6])

local over_daily = 0
if daily_budget >= 0 and new_daily > daily_budget then over_daily = 1 end

local over_monthly = 0
if monthly_budget >= 0 and new_monthly > monthly_budget then over_monthly = 1 end

return {over_daily, new_daily, daily_budget, over_monthly, new_monthly, monthly_budget}
