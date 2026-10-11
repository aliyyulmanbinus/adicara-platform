# ADR-0001: Astro, Go, PostgreSQL Monorepo with Same-Origin Routing

## Status

Accepted

## Date

2026-10-02

## Context

Adicara needs SEO-friendly marketing pages, server-rendered invitations, a transactional API, and a deployment model maintainable by one frontend and one backend engineer.

## Decision

Use a monorepo with Astro for static-first frontend rendering, Go for a REST API, PostgreSQL for durable relational data, and Nginx for same-origin routing. Deploy separate frontend and backend images through Docker Compose on one VPS.

## Alternatives Considered

- A client-only SPA: rejected because core public content must be server-visible and low-JavaScript.
- One full-stack JavaScript service: simpler language count, but does not match the chosen Go backend ownership and contract boundary.
- Microservices/Kubernetes: rejected because operational cost is unjustified for the team and MVP.
- Separate API domain: rejected because it adds CORS, cookie, and deployment complexity without a current need.

## Consequences

Marketing routes remain fast and static while invitation routes can be current. The OpenAPI contract is an explicit shared boundary. The team operates two application runtimes, but avoids a larger distributed system. Nginx and Compose become production-critical configuration.

## Update (2026-10-04)

Same-origin routing still holds. Since this decision the backend was rewritten as Gin with a Module-Based Clean Architecture and JWT authentication (no cookies from the API), and the invitation, guest, and RSVP modules were removed pending a rebuild. The browser no longer calls the Go API for authentication: the Astro server does, and writes the session cookies. See `docs/architecture.md`.

## Update (2026-10-04, routing)

Nginx forwards `/api/v1/*` to the Go API and drops the `/api` prefix (the API serves `/v1/*`); `/healthz` stays at the root. The browser still signs in through Astro (`/auth/*`), so rate limits on credential endpoints are applied to both `/auth/{login,register}` and `/api/v1/auth/{login,register}`. Nginx also resolves the real client address from `X-Forwarded-For` when the peer is a private or loopback address, because TLS is terminated by a proxy on the host. See `docs/devops/nginx.md`.
