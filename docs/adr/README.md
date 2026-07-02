# Architecture Decision Records

Index of significant design decisions for the Distributed Rate Limiter.

| ADR | Title | Status |
|-----|-------|--------|
| [001](001-production-core-vs-examples.md) | Production core vs example services | Accepted |
| [002](002-redis-lua-atomicity.md) | Redis Lua for atomic rate limiting | Accepted |
| [003](003-advisory-check-api.md) | Advisory check API with HTTP 429 on deny | Accepted |
| [004](004-opt-in-limiting-no-rule-allows.md) | No matching rule → allow | Accepted |
| [005](005-fail-closed-redis.md) | Redis unavailable → 503 fail-closed | Accepted |
| [006](006-read-only-config-api.md) | Read-only config API; writes via migrations | Accepted |
