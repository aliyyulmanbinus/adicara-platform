# Invitation Management Dashboard

Date: 2026-10-02

The first focused dashboard is available at `/dashboard`. Its server-rendered request forwards the incoming cookie to the internal API, redirects unauthenticated users to login, and displays only owner-scoped invitation data.

The creation route at `/dashboard/undangan/baru` uses a structured form rather than a freeform canvas. It creates a draft with basic information, two hosts, one event, a registered theme, and an explicit indexing preference. Indexing defaults to off.

Dashboard actions support publish, unpublish, and permanent deletion. State-changing fetches echo the CSRF cookie through `X-CSRF-Token`. Delete requires an explicit browser confirmation. Dashboard routes are `noindex`, disallowed in robots, and excluded from the sitemap.
