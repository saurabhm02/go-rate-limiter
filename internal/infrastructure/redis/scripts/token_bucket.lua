-- Token bucket rate limiter (atomic)
-- KEYS[1]: hash key storing tokens and last_refill_ms
-- ARGV[1]: capacity (max tokens)
-- ARGV[2]: refill_rate (tokens per second)
-- ARGV[3]: now_ms (current time in milliseconds)
-- ARGV[4]: cost (tokens to consume)
-- Returns: {allowed (1/0), remaining, reset_at_unix, retry_after_seconds}

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last_refill_ms')
local tokens = tonumber(data[1])
local last_refill_ms = tonumber(data[2])

if tokens == nil then
  tokens = capacity
  last_refill_ms = now_ms
else
  local elapsed_ms = math.max(0, now_ms - last_refill_ms)
  local refill = (elapsed_ms / 1000.0) * refill_rate
  tokens = math.min(capacity, tokens + refill)
  last_refill_ms = now_ms
end

local allowed = 0
local retry_after = 0

if tokens >= cost then
  allowed = 1
  tokens = tokens - cost
else
  if refill_rate > 0 then
    retry_after = math.ceil((cost - tokens) / refill_rate)
  end
end

redis.call('HSET', key, 'tokens', tokens, 'last_refill_ms', last_refill_ms)

if refill_rate > 0 then
  redis.call('EXPIRE', key, math.ceil(capacity / refill_rate) + 60)
end

local remaining = math.floor(tokens)
local reset_at = math.floor(now_ms / 1000)
if refill_rate > 0 and tokens < capacity then
  local deficit = capacity - tokens
  reset_at = math.floor(now_ms / 1000) + math.ceil(deficit / refill_rate)
end

return {allowed, remaining, reset_at, retry_after}
