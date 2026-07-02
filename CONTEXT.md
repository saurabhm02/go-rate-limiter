# Domain Glossary

Ubiquitous language for the Distributed Rate Limiter.  
**This file defines business terms only** — no implementation, storage, or API details.

---

## Core Entities

### Tenant
An organization or customer account that uses the rate limiting service.  
Each tenant has its own API keys and rate limit rules.  
Tenants are isolated from one another — one tenant's traffic never affects another's limits.

### API Key
A secret credential that identifies a tenant on every request.  
Clients send it via the `X-API-Key` header.  
A tenant may have multiple API keys (e.g. production vs staging).  
Keys can be active or revoked.

### Rule
A configuration that defines how requests are rate limited for a tenant.  
A rule specifies:
- Which routes it applies to (see Route Pattern)
- Which algorithm to use (see Algorithm)
- The limit parameters (capacity, rate, window size)

Rules can be enabled or disabled.  
Multiple rules may exist per tenant, scoped to different routes.

### Route Pattern
A path or pattern that a rule applies to.  
Examples:
- Exact: `/v1/check`
- Prefix: `/api/payments*` (matches `/api/payments` and sub-paths)
- Default: `*` (tenant-wide fallback)

### Algorithm
The rate limiting strategy a rule uses.  
Two algorithms are supported:
- **Token Bucket** — allows bursts up to a capacity; tokens refill at a steady rate
- **Sliding Window** — limits requests over a rolling time window

### Rate Limit Decision
The outcome of evaluating whether a request is allowed.  
Contains:
- Whether the request is **allowed** or **denied**
- The **limit** (maximum allowed)
- **Remaining** capacity in the current window/bucket
- **Reset** time — when the limit fully recovers

### Cost
How many units a single request consumes from the limit.  
Default is 1.  
A heavy operation (e.g. batch export) might cost 5, consuming 5 tokens or window slots.

---

## Integration Modes

### Advisory Check
A client calls the rate limiter to ask "should I allow this request?"  
The client receives a decision and enforces it in their own service.  
Useful when the rate limiter cannot sit in front of all traffic.

### Gateway Mode
A separate middleware or proxy service calls the rate limiter and enforces the decision before traffic reaches upstream APIs.  
Denied requests receive `429 Too Many Requests` immediately.  
In this project, gateway mode is demonstrated by `examples/gateway-middleware` — not by the core service.

---

## Relationships

```
Tenant
  ├── has many API Keys
  └── has many Rules
        └── applies to Route Patterns
              └── evaluated by Algorithm
                    └── produces Rate Limit Decision
```

---

## Status Values

| Entity   | Statuses              |
|----------|-----------------------|
| Tenant   | active, suspended     |
| API Key  | active, revoked       |
| Rule     | enabled, disabled     |

A **suspended** tenant cannot make requests.  
A **revoked** API key is rejected.  
A **disabled** rule is skipped during resolution.

---

## Rule Resolution (conceptual)

When a request arrives for a tenant and route:
1. Find enabled rules for that tenant
2. Prefer exact route match over prefix match
3. Prefer longer (more specific) prefix over shorter
4. Fall back to tenant default rule (`*`)
5. If no rule matches → allow (explicit opt-in limiting)

---

## Terms We Do Not Use

| Avoid | Use instead | Why |
|-------|-------------|-----|
| Client | Tenant | "Client" is ambiguous (HTTP client vs customer) |
| User | Tenant | We rate-limit organizations, not individual users (V1) |
| Quota | Rule / Limit | "Quota" implies billing; we enforce rate limits |
| Throttle | Rate limit | Throttle often means slow-down; we allow or deny |
