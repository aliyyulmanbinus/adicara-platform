# Initial Invitation Schema

## Date

2026-10-02

## Status

Implemented

## Module

Database / Invitation

## Summary

Adds normalized tables for generic invitations, hosts, and scheduled events.

## Objective

Provide a minimal domain that supports weddings today without preventing other event types later.

## Previous Behavior

Not applicable — initial implementation.

## New Behavior

Invitations have UUID identifiers, validated public slugs, lifecycle status, template selection, and an opt-in indexing flag. Hosts and events are separate ordered child records.

## API Changes

Supports `GET /api/v1/public/invitations/{slug}`.

## Database Changes

- Creates `invitations` with unique slug and status checks.
- Creates `invitation_hosts` and `invitation_events` with cascading foreign keys.
- Adds status and relationship indexes.

## Migration

- Up: `backend/migrations/000001_create_invitation_domain.up.sql`
- Down: `backend/migrations/000001_create_invitation_domain.down.sql`

The up migration is additive. The down migration removes the three tables and all contained data.

## Configuration Changes

Requires PostgreSQL and `DATABASE_URL`.

## Security Considerations

UUIDs avoid sequential public identifiers. Invitation indexing defaults to false. Guest tokens and ownership data are not part of this migration.

## Performance Considerations

Unique slug lookup and child foreign-key lookups are indexed. Payload size is bounded only by the number of host/event rows; practical limits should be added with future write APIs.

## Files Changed

- `backend/migrations/000001_create_invitation_domain.up.sql`
- `backend/migrations/000001_create_invitation_domain.down.sql`
- `docs/database.md`

## Tests

Migration execution requires PostgreSQL or Docker. Neither runtime was installed in the current environment, so the up/down migration was not executed.

## Validation

SQL and Compose wiring were reviewed. The YAML files parse successfully. Runtime migration and rollback remain unverified in the current environment.

## Breaking Changes

None; initial schema.

## Rollback Notes

Back up data before running the down migration. It is intentionally destructive.

## Related Documentation

- `docs/database.md`
- `docs/backend/api/2026-10-02-health-and-public-invitation-api.md`
- `contracts/openapi.yaml`
