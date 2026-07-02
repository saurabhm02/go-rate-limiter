# ADR 006: Read-Only Config API

## Status

Accepted

## Context

Operators need visibility into tenants and rules. Runtime writes add complexity (validation, audit, cache invalidation).

## Decision

- `GET /v1/config/tenants` and `GET /v1/config/tenants/{id}/rules` are **read-only**.
- All writes go through **SQL migrations and seeds**.

## Consequences

- Git-reviewed configuration changes.
- No admin API attack surface in V1.
- Rule cache TTL (30s) bounds staleness after migration deploy.
