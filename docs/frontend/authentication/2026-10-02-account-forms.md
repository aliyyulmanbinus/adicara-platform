# Account Forms

Date: 2026-10-02

`/auth/masuk` and `/auth/daftar` now provide accessible account forms connected to the same-origin API. Both routes remain `noindex` and are excluded from the sitemap and robots crawling.

The forms use browser validation, send JSON with same-origin credentials, show errors through an ARIA live region, and redirect successful sessions to `/dashboard`. Passwords are never persisted in browser storage. Session authentication remains in the HttpOnly cookie; the readable CSRF cookie is used only as a request header for state changes.
