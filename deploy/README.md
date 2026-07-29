# Deploying the rate limiter

Target: `https://gorate.ixgo.in`

| Piece      | Where                                    |
|------------|------------------------------------------|
| Go service | Render (Docker web service, free plan)   |
| Postgres   | Neon (`ap-southeast-1`)                  |
| Redis      | Aiven for Valkey                         |
| DNS / TLS  | Cloudflare (`ixgo.in`)                   |

Render hosts only the service. Postgres and Redis are managed elsewhere and
reach the app as two secrets, so [`render.yaml`](../render.yaml) declares one
resource and contains nothing sensitive.

## Why Render

All of Railway, Fly.io and Render can run this. Render wins on reproducibility —
the service is declared in a file in this repo, so redoing the deploy is
"point a Blueprint at the repo", not a sequence of dashboard clicks you have to
remember. Railway is cheaper but databases are added by hand in the dashboard;
Fly's Postgres is one you operate yourself.

Since Postgres and Redis ended up on Neon and Aiven anyway, the remaining reason
to stay on Render is the free Docker web service with HTTPS on a custom domain.

## Secrets

Set these on the Render service. Both are `sync: false` in `render.yaml`, so
Render prompts once and stores them encrypted — they are never in the repo.

| Key | Value |
|-----|-------|
| `DATABASE_URL` | Neon connection string, including `?sslmode=require` |
| `REDIS_URL` | Aiven Valkey URI, `rediss://default:<password>@<host>:<port>` |

Things that bite here:

- **The scheme must be `rediss://`, not `redis://`.** Aiven only speaks TLS, and
  its console shows the URI without a scheme — you add it.
- **Use the Valkey service's port, not another service's.** One Aiven project
  shares a hostname across services on different ports. Pointing this at the
  Kafka port gives a clean TLS handshake followed by an immediate EOF on every
  command, because Kafka speaks a binary protocol and silently drops RESP. It
  looks exactly like an auth failure and is not one.
- **No custom CA is needed.** Aiven Valkey serves a Let's Encrypt wildcard for
  `*.e.aivencloud.com`, which the system trust store already accepts. The
  per-project CA Aiven offers for download applies to other services (Kafka).

Tenant API keys are not here — only their SHA-256 hashes, in Postgres, put there
by `migrations/002_admitdesk_tenant.up.sql`. The raw key lives in AdmitDesk's
own secret store.

## Migrations

`cmd/migrate` records every applied file in a `schema_migrations` table and runs
each one in a transaction, so it is safe to run on every deploy.

Render does not offer a pre-deploy command on free instances, so apply them
yourself after any schema change:

```bash
DATABASE_URL='postgresql://…neon.tech/neondb?sslmode=require' make migrate-prod
```

`migrate-prod` never applies the dev seed; `make migrate` (local) does, via
`SEED=dev`. Keep it that way — the demo key `rl_demo_abc123xyz` is published in
the README and must not exist in production.

On a paid plan, add `preDeployCommand: /migrate` to `render.yaml` and this
becomes automatic.

Schema and the AdmitDesk tenant are **already applied to Neon**. Verified: the
only tenant is `admitdesk`, and `rl_demo_abc123xyz` gets a 401.

## Deploying

The repo is public, so Render can deploy it without connecting a Git provider.

1. **Push.** Render builds from GitHub, not from your laptop. This is the step
   that gates everything else.

2. Render → **New** → **Web Service** → **Public Git Repository** →
   `https://github.com/saurabhm02/go-rate-limiter`.

   (Connecting GitHub instead is what enables auto-deploy on push; the
   public-repo path needs a manual redeploy each time.)

3. Service settings:

   | Field | Value |
   |---|---|
   | Language / Runtime | **Docker** |
   | Dockerfile Path | `deploy/docker/Dockerfile` |
   | Docker Build Context Directory | `.` |
   | Region | Singapore |
   | Instance Type | Free |
   | Health Check Path | `/health/ready` |

   Render's create-service API silently ignores a nested Dockerfile path, so if
   you script this, set it and then verify it stuck — a wrong path fails the
   build with `open Dockerfile: no such file or directory`.

4. Add `DATABASE_URL` and `REDIS_URL` from the table above, plus
   `LOG_LEVEL=info` and `MIGRATIONS_DIR=/app/migrations`. Create the service.

   First build is ~3–5 min. `/health/ready` returning `{"status":"ready"}` means
   both Postgres and Redis answered.

5. Smoke test:

   ```bash
   curl https://gorate.onrender.com/health/ready
   ```

## Custom domain

Render → service → Settings → Custom Domains → add `gorate.ixgo.in`. Render
shows a CNAME target.

Cloudflare → `ixgo.in` → DNS → add:

| Type  | Name     | Target                | Proxy |
|-------|----------|-----------------------|-------|
| CNAME | `gorate` | `gorate.onrender.com` | **DNS only** at first |

Leave it grey-clouded until Render marks the domain verified and its certificate
issued — Render's ACME challenge cannot complete through Cloudflare's proxy.
Then, to put Cloudflare in front, switch to **Proxied** and set SSL/TLS mode to
**Full (strict)**. Anything less leaves the Cloudflare→Render hop
unauthenticated.

If you proxy, add a Cache Rule: hostname equals `gorate.ixgo.in` → **Bypass
cache**. A cached `/v1/check` response would hand out stale allow decisions.
Cloudflare does not cache `POST` by default; the rule stops someone enabling
aggressive caching later.

## Rolling back

Render → service → **Events** → *Rollback*. Migrations are not rolled back with
it; `*.down.sql` files exist but are applied deliberately.

## Cost

| Resource | Plan | USD/mo |
|---|---|---|
| Render web service | Free | 0 |
| Neon Postgres | Free | 0 |
| Aiven Valkey | Free / Hobbyist | 0 |
| Cloudflare DNS | Free | 0 |
| **Total** | | **0** |

The free Render instance sleeps after ~15 min idle, and the next request pays a
30–60 s cold start. For a rate limiter on AdmitDesk's request path that is the
whole cost question: Render Starter is **$7/mo**, removes the spin-down, and
unlocks `preDeployCommand` so migrations stop being a manual step.

Neon's free tier suspends idle compute too, but wakes in about a second, and the
rule cache (`RULE_CACHE_TTL`, 30 s) absorbs most reads anyway.

## Local equivalent

```bash
make docker-up     # postgres + redis + migrations + API
```

Runs migrations itself; the API waits on them. If 5432/6379/8080 are taken —
they are, by `goadmitdesk-db-1` — override:

```bash
POSTGRES_PORT=5433 make docker-up
```

`HTTP_PORT`, `REDIS_PORT`, `PROMETHEUS_PORT` work the same way. Pass the same
override to any later `docker compose` command against this stack, or Compose
tries to recreate Postgres on the default port.
