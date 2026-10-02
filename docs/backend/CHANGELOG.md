# Backend Changelog

## 2026-10-02

### API Foundation

- Added liveness, readiness, and published invitation endpoints with structured JSON logging. Documentation: `api/2026-10-02-health-and-public-invitation-api.md`.

### Database

- Added the initial generic invitation, host, and event schema. Documentation: `migrations/2026-10-02-initial-invitation-schema.md`.

### Authentication

- Added bcrypt password storage, hashed opaque sessions, CSRF protection, registration, login, logout, and current-user endpoints. Documentation: `authentication/2026-10-02-session-authentication.md`.

### Invitation Management

- Added owner-scoped invitation CRUD and publish/unpublish endpoints. Documentation: `invitations/2026-10-02-owner-scoped-crud.md`.
- Added users, sessions, and invitation ownership migration. Documentation: `migrations/2026-10-02-auth-and-ownership-schema.md`.

### Guests and RSVP

- Added owner-scoped guest CRUD, secure random guest tokens, and idempotent public RSVP submission. Documentation: `invitations/2026-10-02-guests-and-rsvp.md`.
- Added guest and RSVP tables. Documentation: `migrations/2026-10-02-guests-and-rsvp-schema.md`.
