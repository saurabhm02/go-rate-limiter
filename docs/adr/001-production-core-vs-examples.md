# ADR 001: Production Core vs Example Services

## Status

Accepted

## Context

The service must demonstrate both advisory and gateway integration patterns without polluting the production API surface.

## Decision

- **Core service** exposes only production APIs: `POST /v1/check`, read-only config, health, metrics.
- **Demonstrations** live in `examples/` (`payment-service`, `order-service`, `gateway-middleware`).
- Example apps integrate **only via HTTP** — no imports from `internal/`.

## Consequences

- Core binary stays deployable as a standalone rate limiter.
- Interview story is clear: "real service" vs "how clients use it."
- Example apps can be deleted or replaced without touching core.
