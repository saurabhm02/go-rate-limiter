-- Sliding window counter (atomic hybrid approximation)
-- KEYS[1]: current window counter (string)
-- KEYS[2]: previous window counter (string)
-- KEYS[3]: stored window index (string)
-- ARGV[1]: limit (max requests)
-- ARGV[2]: window_seconds
-- ARGV[3]: now_seconds (unix)
-- ARGV[4]: cost
-- Returns: {allowed (1/0), remaining, reset_at_unix, retry_after_seconds}

local curr_key = KEYS[1]
local prev_key = KEYS[2]
local idx_key = KEYS[3]

local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

local curr_window = math.floor(now / window)
local stored_window = tonumber(redis.call('GET', idx_key))

if stored_window == nil or stored_window < curr_window then
  if stored_window ~= nil and stored_window == curr_window - 1 then
    local curr_val = tonumber(redis.call('GET', curr_key) or '0')
    redis.call('SET', prev_key, curr_val)
  else
    redis.call('SET', prev_key, 0)
  end
  redis.call('SET', curr_key, 0)
  redis.call('SET', idx_key, curr_window)
  redis.call('EXPIRE', prev_key, window * 2)
  redis.call('EXPIRE', curr_key, window * 2)
  redis.call('EXPIRE', idx_key, window * 2)
end

local prev_count = tonumber(redis.call('GET', prev_key) or '0')
local curr_count = tonumber(redis.call('GET', curr_key) or '0')
local window_start = curr_window * window
local elapsed = now - window_start
local weight = 1 - (elapsed / window)
local estimate = prev_count * weight + curr_count

if estimate + cost > limit then
  local retry_after = math.ceil(window - elapsed)
  local reset_at = window_start + window
  local remaining = math.max(0, math.floor(limit - estimate))
  return {0, remaining, reset_at, retry_after}
end

local new_curr = curr_count + cost
redis.call('SET', curr_key, new_curr)
redis.call('EXPIRE', curr_key, window * 2)
redis.call('EXPIRE', prev_key, window * 2)
redis.call('EXPIRE', idx_key, window * 2)

local remaining = math.max(0, math.floor(limit - (estimate + cost)))
local reset_at = window_start + window
return {1, remaining, reset_at, 0}
