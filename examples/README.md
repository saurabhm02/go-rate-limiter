# Examples

Demonstration microservices that integrate with the **production** rate limiter over HTTP only.

They do **not** import `internal/` — same contract a real service would use.

| Example | Port | Pattern | Status |
|---------|------|---------|--------|
| `payment-service` | 8081 | Advisory — calls `POST /v1/check` before handling | ✅ |
| `order-service` | 8082 | Advisory — per-route limits | ✅ |
| `gateway-middleware` | 8083 | Gateway — enforces 429 before proxying upstream | ✅ |

Shared client: [`ratelimitclient`](ratelimitclient/client.go)

## Run

```bash
make docker-up && make migrate
make examples-up
```

## Environment variables (examples)

| Variable | Description |
|----------|-------------|
| `RATE_LIMITER_URL` | Base URL, e.g. `http://ratelimit:8080` |
| `API_KEY` | Tenant API key (`X-API-Key` header) |
| `UPSTREAM_URL` | Gateway only — e.g. `http://payment-service:8081` |
