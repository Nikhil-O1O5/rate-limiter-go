-- Keys: KEYS[1] = rate limit key (e.g. rate_limit:user123:/hash)
-- Args: ARGV[1] = capacity, ARGV[2] = refill_rate (tokens/sec), ARGV[3] = now (unix ms), ARGV[4] = ttl (seconds)
--
-- Returns: { allowed (0|1), remaining_tokens, retry_after_ms }

local key          = KEYS[1]
local capacity     = tonumber(ARGV[1])
local refill_rate  = tonumber(ARGV[2])
local now          = tonumber(ARGV[3])
local ttl          = tonumber(ARGV[4])

local data = redis.call("HMGET", key, "tokens", "last_refill")
local tokens      = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

-- refill based on elapsed time
local elapsed = math.max(0, (now - last_refill) / 1000)
tokens = math.min(capacity, tokens + elapsed * refill_rate)

local allowed      = 0
local retry_after  = 0

if tokens >= 1 then
    tokens  = tokens - 1
    allowed = 1
else
    -- ms until one token is available
    retry_after = math.ceil((1 - tokens) / refill_rate * 1000)
end

redis.call("HSET", key, "tokens", tokens, "last_refill", now)
redis.call("EXPIRE", key, ttl)

return { allowed, math.floor(tokens), retry_after }
