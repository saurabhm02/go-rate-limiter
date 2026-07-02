# ADR 003: Advisory Check API with HTTP 429 on Deny

## Status

Accepted

## Context

Clients need a standard contract to ask "should I allow this request?" before doing work.

## Decision

- Expose `POST /v1/check` with JSON `{ "route", "cost" }`.
- Return **200** when allowed, **429** when denied (not 200 with `allowed: false`).
- Include `X-RateLimit-*` headers and `Retry-After` on deny.

## Consequences

- Proxies, SDKs, and retries understand 429 natively.
- Callers in `examples/` enforce the decision in their own process or gateway.
- Handler stays thin; `CheckService` owns business logic.
