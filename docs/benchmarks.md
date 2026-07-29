# Benchmarks

Real numbers from `make benchmark` / `make loadtest-smoke`, not estimates. Reproduce with:

```bash
make docker-up && make migrate
make benchmark          # allow-path, tiered RATE
make loadtest-smoke     # deny-path sanity check against /v1/check
```

## Environment

- MacBook (Apple Silicon, arm64), 8 CPUs / 4GB allotted to Docker Desktop
- Full local stack via `docker compose` (`deploy/docker/docker-compose.yml`): single `ratelimit` API container, Postgres, Redis, Prometheus — no host resource isolation between them
- k6 driven from a separate `grafana/k6` container hitting the API over `host.docker.internal`
- Not a dedicated benchmark rig — treat as a lower bound, not a ceiling. See [Caveats](#caveats).

## Allow-path throughput

`POST /v1/check` against a seeded high-limit route (`/bench`, 1,000,000/min sliding window — see `migrations/seeds/dev_seed.sql`) so the demo tenant's normal low limits don't dominate the result. 15s sustained load per tier via k6 `constant-arrival-rate`.

| Target RPS | Achieved RPS | p50 | p90 | p95 | max | Errors |
|---|---|---|---|---|---|---|
| 500 | 500 | 0.99ms | 1.19ms | 1.30ms | 27.4ms | 0% |
| 1,000 | 1,000 | 0.75ms | 0.91ms | 1.09ms | 17.3ms | 0% |
| 2,000 | 2,000 | 0.82ms | 1.09ms | 1.33ms | 33.1ms | 0% |
| 5,000 | 4,755 | 12.31ms | 55.0ms | 70.6ms | 190.8ms | 31.9% |

Holds sub-1.5ms p95 cleanly up to 2,000 req/s on this hardware. At 5,000 req/s the stack falls behind (dropped iterations, rising tail latency, failures) — see caveats below on what's actually being measured there before treating that as a hard ceiling.

## Deny-path latency

Same load pattern (1,000 req/s, 15s) but against the real `/v1/check` demo rule (100/min sliding window), so ~99% of requests are legitimate 429s. Confirms rejecting a request costs about the same as allowing one — no extra round trip on the deny path.

| Metric | Value |
|---|---|
| p50 | 0.77ms |
| p90 | 1.05ms |
| p95 | 1.50ms |
| Denied | 14,901 / 15,001 (99.3%, expected — proves fast reject) |

## Caveats

- Single API container, single Postgres, single Redis, all on one laptop competing for the same 8 CPUs as the k6 load generator and Docker Desktop's VM overhead — not a production topology.
- The 5,000 req/s tier's degradation isn't yet isolated to API vs. Redis vs. Postgres vs. k6-client-side saturation; profiling that (e.g. via `/metrics` + `docker stats` during the run) is the natural next step, not done here.
- No horizontal scale-out tested — one `ratelimit` replica. Multiple replicas sharing one Redis is the next thing to benchmark if this becomes a "does it scale" claim rather than a "is it fast" claim.
