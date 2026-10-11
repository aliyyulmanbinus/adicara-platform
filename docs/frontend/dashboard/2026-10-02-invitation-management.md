# Invitation Management Dashboard

> **Status (2026-10-04):** Only the access control part is current. The dashboard is now guarded by `src/middleware.ts` and loads the user from the new backend (see `authentication/2026-10-04-session-cookies.md`); the cookie forwarding and `X-CSRF-Token` header described below were part of the previous backend and are no longer used for sign-in. The invitation list, the creation form, and the publish/unpublish/delete actions call the previous API (`/api/v1/invitations`), which the backend no longer serves, so they do not work until the invitation module is rebuilt. The dashboard shows its empty state in the meantime.

Date: 2026-10-02

The first focused dashboard is available at `/dashboard`. Its server-rendered request forwards the incoming cookie to the internal API, redirects unauthenticated users to login, and displays only owner-scoped invitation data.

The creation route at `/dashboard/undangan/baru` uses a structured form rather than a freeform canvas. It creates a draft with basic information, two hosts, one event, a registered theme, and an explicit indexing preference. Indexing defaults to off.

Dashboard actions support publish, unpublish, and permanent deletion. State-changing fetches echo the CSRF cookie through `X-CSRF-Token`. Delete requires an explicit browser confirmation. Dashboard routes are `noindex`, disallowed in robots, and excluded from the sitemap.
