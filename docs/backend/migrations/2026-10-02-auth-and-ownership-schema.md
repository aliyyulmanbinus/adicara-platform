# Authentication and Ownership Schema

Date: 2026-10-02

Migration `000002_create_auth_and_ownership` adds:

- `users` with case-normalized unique email, display name, and password hash;
- `sessions` with token hash, CSRF token hash, expiry, and owner index;
- nullable `invitations.user_id` for backward compatibility with pre-existing public data;
- owner and owner/status indexes for dashboard queries.

Invitations created through the authenticated API always receive `user_id`. Deleting a user cascades into their sessions and owned invitations. The migration has a matching down migration and must be applied by the migration job, never through application auto-migration.

Runtime migration validation remains pending on a machine with Docker or PostgreSQL because neither executable is available in the current environment.
