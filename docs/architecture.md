# Architecture

Distributed Rate Limiter — production core with example integrations in `examples/`.

## System context

```mermaid
flowchart TB
    subgraph clients [Clients]
        Pay[payment-service :8081]
        Order[order-service :8082]
        GW[gateway-middleware :8083]
        Svc[Any microservice]
    end

    subgraph core [Rate Limiter :8080]
        direction TB
        HTTP[HTTP API]
        App[Application layer]
        HTTP --> App
    end

    subgraph stores [Data stores]
        PG[(PostgreSQL<br/>tenants, rules, keys)]
        Redis[(Redis<br/>Lua counters)]
    end

    subgraph ops [Operations]
        Prom[Prometheus :9090]
        K6[k6 load tests]
    end

    Pay -->|POST /v1/check| HTTP
    Order -->|POST /v1/check| HTTP
    GW -->|POST /v1/check| HTTP
    Svc -->|POST /v1/check| HTTP
    GW -.->|proxy if allowed| Pay
    App --> PG
    App --> Redis
    Prom -->|scrape /metrics| HTTP
    K6 -->|load test| HTTP
```

## Request flow (check)

```mermaid
sequenceDiagram
    participant C as Client
    participant A as Auth middleware
    participant H as Check handler
    participant S as CheckService
    participant PG as PostgreSQL
    participant R as Redis Lua

    C->>A: POST /v1/check + X-API-Key
    A->>H: tenant ID in context
    H->>S: Check(tenant, route, cost)
    S->>PG: rules (cached)
    S->>R: EVALSHA
    R-->>S: allowed, remaining, reset
    S-->>H: Decision
    H-->>C: 200 or 429 + headers
```

## Clean Architecture layers

| Layer | Path | Responsibility |
|-------|------|----------------|
| Entry | `cmd/server/`, `cmd/migrate/` | Wiring, lifecycle |
| Transport | `internal/transport/http/` | Routes, auth, logging, metrics |
| Observability | `internal/observability/` | slog + Prometheus |
| Application | `internal/application/` | Auth, check, rule resolution |
| Domain | `internal/domain/` | Entities, ports, errors |
| Infrastructure | `internal/infrastructure/` | Redis, Postgres, cache |

Dependency rule: **domain has no outward dependencies**. Infrastructure implements ports.

## Core API surface

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `POST /v1/check` | Yes | Advisory rate limit check |
| `GET /v1/config/tenants` | Yes | List tenants (read-only) |
| `GET /v1/config/tenants/{id}/rules` | Yes | List rules (read-only) |
| `GET /health/live` | No | Liveness |
| `GET /health/ready` | No | PG + Redis readiness |
| `GET /metrics` | No | Prometheus scrape |

## Algorithms

| Algorithm | Use case | Redis keys |
|-----------|----------|------------|
| Token bucket | Burst traffic (payments) | Single key per tenant+route hash |
| Sliding window | Smooth rate (orders, defaults) | curr + prev + index keys |

## Testing strategy

```
        ┌─────────┐
        │  k6     │  load/k6/
        ├─────────┤
        │ Concurr │  test/concurrency/ (-race)
        ├─────────┤
        │ Integr. │  test/integration/ (testcontainers)
        ├─────────┤
        │  Unit   │  test/unit/
        └─────────┘
```

## Deployment

| Profile | Command | Notes |
|---------|---------|-------|
| Development | `make docker-up` | Exposes PG/Redis ports |
| Production | `make docker-prod-up` | Internal DB/Redis, migrate job |
| Examples | `make examples-up` | Full demo stack |

## ADRs

Architecture decisions: [docs/adr/](adr/)

## Related docs

- [CONTEXT.md](../CONTEXT.md) — domain glossary
- [docs/api/openapi.yaml](api/openapi.yaml) — REST contract
- [docs/interview-guide.md](interview-guide.md) — interview prep
