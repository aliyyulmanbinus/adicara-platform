# Owner-scoped Invitation CRUD

Date: 2026-10-02

Authenticated owners can list, create, read, update, delete, publish, and unpublish invitations under `/api/v1/invitations`.

Every repository query includes the authenticated user ID. A resource owned by another user therefore returns the same 404 response as a missing resource. UUID validation happens before repository access, and public sequential identifiers are not introduced.

Create and update operations validate:

- lowercase event type and slug formats;
- title and field length limits;
- a registered template key;
- one to ten hosts;
- one to twenty events;
- non-zero start times and end times after starts;
- HTTP(S)-only map URLs.

Child host and event replacement runs in the same transaction as invitation creation or update. New invitations always start as `draft`. Publishing revalidates the complete stored aggregate before changing status to `published`; unpublishing returns it to `draft`.
