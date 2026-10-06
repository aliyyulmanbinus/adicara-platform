# Editorial Ivory Template

## Date

2026-10-02

## Status

Implemented

## Area

Invitation Template / Public Invitation Route

## Summary

Added the first invitation template, a registry, a shared typed data model, shared sections, and an on-demand rendered `/i/[slug]` page.

## Objective

Prove that invitation data can move between presentations without duplicating backend data or template business logic.

## Previous Behavior

Not applicable — initial implementation.

## New Behavior

The frontend fetches a published invitation on the server, maps the API response to `InvitationData`, chooses a registered theme, and renders enabled sections in configured order. Missing invitations return 404 and API failures return 503.

## Visual Changes

- Editorial ivory palette with typography-led cover.
- Shared host, schedule, and closing sections.
- Mobile-first single column with two-column host/event lists on wider screens.

## Responsive Behavior

The template uses fluid type, intrinsic spacing, and a 42rem breakpoint. Physical verification at 360, 375, 390, 414, and 430px was not performed in the current environment.

## Components Affected

- Invitation registry and types
- Cover, Hosts, EventSchedule, Closing
- Editorial Ivory template and theme stylesheet
- Invitation layout and dynamic route

## Files Changed

- `frontend/src/invitation-templates/**`
- `frontend/src/components/invitation/sections/**`
- `frontend/src/lib/api/invitations.ts`
- `frontend/src/layouts/InvitationLayout.astro`
- `frontend/src/pages/i/[slug].astro`

## API Dependencies

`GET /api/v1/public/invitations/{slug}`. That endpoint no longer exists: the invitation module was removed from the backend (see `docs/api.md`), so `/i/[slug]` currently always returns 404 until the module is rebuilt.

## SEO Impact

Invitation pages are `noindex, nofollow` by default, excluded from sitemap, and receive a canonical URL. Indexing is enabled only when `allow_indexing` is true.

## Accessibility Impact

Core content is server-rendered; landmarks, headings, datetime values, and real map links are used. The template does not depend on motion or client JavaScript.

## Performance Impact

Adds no hydrated island. API data is fetched once during server rendering. No performance measurements were taken.

## Validation

- `npm run check` — passed with 0 diagnostics.
- `npm test` — source tests for the registry and privacy behavior passed.
- `npm run build` — passed and emitted a Node server entry plus static marketing routes.
- Local SSR smoke test without a backend — `/i/demo-undangan` returned the designed 503 response.
- End-to-end rendering with PostgreSQL data and interactive browser/device visual QA — not verified in the current environment.

## Screenshots

Not provided.

## Notes

Only Editorial Ivory is included in the initial vertical slice. Botanical Modern and Monochrome Luxe remain later Phase 3 work.
