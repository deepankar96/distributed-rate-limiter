-- KEYS[1]: user bucket key
-- ARGV[1]: capacity (tokens), ARGV[2]: refill rate (tokens/second)
-- Returns 1 when a token was consumed, or 0 when denied.

local function isFinite(value)
    return value ~= nil and value == value
        and value ~= math.huge and value ~= -math.huge
end

if #KEYS ~= 1 or #ARGV ~= 2 then
    return redis.error_reply('expected one bucket key, capacity, and refill rate')
end

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refillRate = tonumber(ARGV[2])

if not isFinite(capacity) or capacity < 1 then
    return redis.error_reply('capacity must be finite and at least 1')
end
if not isFinite(refillRate) or refillRate <= 0 then
    return redis.error_reply('refill rate must be finite and positive')
end

-- Once an idle bucket would be full, deleting it preserves the allowance.
local ttlMs = math.max(1, math.ceil(capacity / refillRate * 1000))
if not isFinite(ttlMs) or ttlMs > 9007199254740991 then
    return redis.error_reply('bucket TTL exceeds the supported millisecond range')
end

local clock = redis.call('TIME')
local nowMs = tonumber(clock[1]) * 1000
    + math.floor(tonumber(clock[2]) / 1000)

local state = redis.call('HMGET', key, 'tokens', 'last_updated_ms')
local tokens = capacity
local lastUpdatedMs = nowMs

if state[1] or state[2] then
    tokens = tonumber(state[1])
    lastUpdatedMs = tonumber(state[2])
    -- Do not silently reset malformed state and grant a fresh allowance.
    if not isFinite(tokens) or tokens < 0
        or not isFinite(lastUpdatedMs) or lastUpdatedMs < 0 then
        return redis.error_reply('bucket state must contain valid tokens and last_updated_ms')
    end
end

-- A wall-clock adjustment must not move the stored timestamp backward.
nowMs = math.max(nowMs, lastUpdatedMs)
local elapsedSeconds = (nowMs - lastUpdatedMs) / 1000
local refillAmount = elapsedSeconds * refillRate
tokens = math.min(capacity, tokens + refillAmount)

local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

-- Save fractional tokens as text; integer-formatted strings preserve millisecond
-- timestamps and TTLs without scientific notation in Redis command arguments.
redis.call('HSET', key,
    'tokens', string.format('%.17g', tokens),
    'last_updated_ms', string.format('%.0f', nowMs))
redis.call('PEXPIRE', key, string.format('%.0f', ttlMs))

return allowed
