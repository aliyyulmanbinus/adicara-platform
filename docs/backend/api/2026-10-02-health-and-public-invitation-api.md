# Health and Public Invitation API

## Date

2026-10-02

## Status

Implemented

## Module

Operations / Invitation

## Summary

Created the initial Go service with health checks, structured request logs, consistent errors, and public published-invitation reads.

## Objective

Complete the smallest backend slice needed for deployment readiness and server-rendered invitation pages.

## Previous Behavior

Not applicable — initial implementation.

## New Behavior

The process exposes liveness independently of PostgreSQL, readiness backed by a bounded database ping, and an invitation endpoint that returns only `published` records. Slugs are validated before database access.

## API Changes

- `GET /healthz`: public, returns 200 when the process responds.
- `GET /readyz`: public, returns 200 when PostgreSQL responds and 503 otherwise.
- `GET /api/v1/public/invitations/{slug}`: public read, returns 200, 400, 404, or 500.

The full request/response contract is in `contracts/openapi.yaml`.

## Database Changes

Reads `invitations`, `invitation_hosts`, and `invitation_events`. See the related migration document.

## Migration

`backend/migrations/000001_create_invitation_domain.up.sql`; backward-compatible for a new database, non-destructive on apply, reversible through the matching down migration (which is destructive to created data).

## Configuration Changes

- `PORT` (default `8080`)
- `DATABASE_URL` (required)
- `LOG_LEVEL` (`info` by default; `debug` supported)

## Security Considerations

Uses parameterized queries, strict slug validation, generic internal-error output, request IDs, timeouts, and route-pattern logging instead of personal slug values. This slice has no authentication or write/upload endpoints.

## Performance Considerations

Each successful invitation lookup executes three bounded queries: invitation, hosts, and events. Slug and foreign-key indexes support those queries. There is no N+1 loop.

## Files Changed

- `backend/cmd/api/main.go`
- `backend/internal/config/config.go`
- `backend/internal/domain/invitation.go`
- `backend/internal/handler/**`
- `backend/internal/middleware/http.go`
- `backend/internal/repository/postgres/invitation_repository.go`
- `backend/internal/server/server.go`
- `backend/internal/service/**`
- `contracts/openapi.yaml`

## Tests

Unit tests cover liveness/readiness behavior and service slug validation/delegation.

## Validation

- `go vet ./...` — passed.
- `go test -count=1 ./...` — passed.
- `go build -o bin/adicara-api.exe ./cmd/api` — passed.
- `npx --yes @redocly/cli@latest lint contracts/openapi.yaml` — passed.
- Local smoke test with PostgreSQL intentionally unavailable: `/healthz` returned 200, `/readyz` returned 503, invalid slug returned 400, and a valid lookup returned 500 without leaking a database error.

## Breaking Changes

None; this is the initial contract.

## Rollback Notes

Revert the application image. Apply the down migration only if all invitation data may be safely removed.

## Related Documentation

- `docs/api.md`
- `docs/database.md`
- `docs/backend/migrations/2026-10-02-initial-invitation-schema.md`
- `docs/frontend/invitation-templates/2026-10-02-editorial-ivory-template.md`
