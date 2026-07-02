# ADR 002: Redis Lua for Atomic Rate Limiting

## Status

Accepted

## Context

Distributed rate limiting requires read-modify-write on counters. Multiple app instances must not race and over-allow traffic.

## Decision

- Store counters in **Redis**.
- Execute token bucket and sliding window logic in **Lua scripts** via `EVALSHA`.
- One script invocation per check — atomic on the Redis server.

## Alternatives considered

| Option | Rejected because |
|--------|------------------|
| In-memory per instance | Not distributed; limits multiply by replica count |
| Postgres row locks | Higher latency; poor fit for per-request hot path |
| Redis WATCH/MULTI | More round-trips; harder to reason about than Lua |

## Consequences

- Redis is a hard dependency for the check path.
- Lua scripts are versioned in `internal/infrastructure/redis/scripts/` and embedded at build time.
- miniredis used in unit tests; real Redis in integration/load tests.
