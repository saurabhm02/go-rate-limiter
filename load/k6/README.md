# k6 Load Tests

Load tests for `POST /v1/check` using [Grafana k6](https://k6.io/).

## Prerequisites

- Stack running: `make docker-up && make migrate`
- k6 installed **or** Docker (Makefile uses `grafana/k6` image)

## Scripts

| Script | Purpose |
|--------|---------|
| `check_smoke.js` | 5 VUs × 10s — quick sanity check |
| `check_load.js` | 1000 req/s × 30s — sustained load (`constant-arrival-rate`) |

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | Rate limiter base URL |
| `API_KEY` | `rl_demo_abc123xyz` | Tenant API key |
| `RATE` | `1000` | Arrival rate for load script (req/s) |
| `DURATION` | `30s` | Load test duration |

## Run

```bash
make loadtest-smoke
make loadtest
```

## Interpreting results

- **http_req_duration p(99)** — tail latency; stretch goal **< 5ms** on a warm local Docker stack; **< 100ms** is the Makefile threshold for laptops.
- **http_req_failed** — should stay near 0 (401/503 count as failures).
- **429 responses** — expected at 1000 RPS against `/v1/check` (seed limit 100/min); proves deny path is fast too.

Example PromQL (after load test):

```promql
sum(rate(ratelimit_checks_total[1m])) by (result)
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))
```
