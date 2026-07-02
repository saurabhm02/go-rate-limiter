# ADR 004: No Matching Rule → Allow (Opt-In Limiting)

## Status

Accepted

## Context

Tenants may define rules for only some routes. Undefined routes should not be implicitly blocked.

## Decision

If rule resolution finds **no enabled match**, return `allowed: true` without calling Redis.

## Consequences

- Safer rollout: add rules incrementally without breaking unlisted routes.
- Explicit opt-in to limiting per route pattern.
- Callers must not assume every route is limited.
