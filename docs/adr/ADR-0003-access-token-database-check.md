# ADR-0003: Access tokens are checked against the database

## Status

Accepted

## Date

2026-10-04

## Context

Access tokens were verified by signature only. A deactivated account, or a password that had just been changed because a device was lost, kept working until the token expired (default 15 minutes). A refresh token could also keep an inactive account signed in indefinitely, and replaying a rotated refresh token was rejected without revoking the thief's or the owner's other sessions.

## Decision

`BearerAuth` loads `status` and `password_changed_at` for the token's user on every protected request and rejects the token when the account is missing or inactive, or when the token was issued before the last password change. `Refresh` applies the same status check. A refresh token spent by rotation and presented again more than 10 seconds later revokes every session of that user; within 10 seconds it is only rejected. All of these rejections are `401 unauthorized`.

## Alternatives Considered

- Keep stateless access tokens and shorten the TTL: leaves a window of the TTL and multiplies refresh traffic against a rate-limited endpoint.
- A token version or denylist cache in memory: faster, but per-process state does not survive restarts or a second instance, and adds invalidation logic a two-person team must maintain. Reconsider only if the extra query becomes measurable.
- `403 account_inactive` for deactivated accounts on protected routes: more precise, but the frontend treats non-401 failures from `/v1/me` and refresh as "backend unavailable" and keeps the cookies, which would strand the user on an error page.
- Revoking all sessions on any replay without a grace period: simplest, but two browser tabs or two frontend instances racing on one refresh would sign the user out everywhere.

## Consequences

One primary-key query per protected request (two for `GET /v1/me`, which loads the user again). Measured only locally (600 authenticated `GET /v1/me`, 25 in parallel, API and PostgreSQL on one machine): all `200`, about 460 requests per second, p50 1.1 ms, p95 1.7 ms. That is not a production figure, but it shows the extra query is negligible at the current scale. Account deactivation and password change take effect on the next request. A leaked, replayed refresh token costs the owner a re-login, which is the intended outcome. The API depends on the database for authentication, which it already did for everything it serves. Details: `docs/backend/authentication/2026-10-04-auth-hardening.md`.
