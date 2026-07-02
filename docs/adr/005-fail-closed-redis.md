# ADR 005: Redis Unavailable → 503 Fail-Closed

## Status

Accepted

## Context

When the rate limit backend is down, the service cannot safely decide allow vs deny.

## Decision

- Redis errors on check → **503 Service Unavailable**.
- Readiness probe includes Redis ping → pod removed from load balancer.

## Alternatives considered

| Option | Rejected because |
|--------|------------------|
| Fail-open (allow) | Unlimited traffic during outage — unacceptable for abuse protection |
| Fail-closed as 429 | Misleading; client might retry forever |

## Consequences

- Short Redis blips block checks rather than bypass limits.
- Operators must monitor Redis HA (Sentinel/managed Redis in production).
