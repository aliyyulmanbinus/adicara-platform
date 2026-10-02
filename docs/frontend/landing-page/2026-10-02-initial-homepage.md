# Initial Adicara Homepage

## Date

2026-10-02

## Status

Implemented

## Area

Homepage / Navigation / Footer / SEO

## Summary

Created the first original Adicara marketing homepage with a warm editorial visual system and Indonesian content.

## Objective

Establish a credible, mobile-first public presence without fabricated social proof or unnecessary client-side JavaScript.

## Previous Behavior

Not applicable — initial implementation.

## New Behavior

The homepage presents the product, template showcase, creation flow, feature explanation, full invitation preview, benefits, transparent pricing placeholder, inspiration preview, FAQ, and launch CTA. Navigation uses normal crawlable links and a native `details` mobile menu.

## Visual Changes

- Warm ivory, charcoal, stone, bronze, and forest design tokens.
- Editorial serif headings paired with system sans-serif body text.
- CSS-only hero invitation mockup and restrained borders/shadows.
- Responsive grids that collapse into a single-column mobile flow.

## Responsive Behavior

Styles are mobile-first and include a desktop breakpoint at 48rem plus navigation breakpoint at 52rem. Source review covers layouts intended for 360–430px widths, tablet, and desktop. Physical device and browser-matrix verification was not performed in the current environment.

## Components Affected

- `Header.astro`
- `Footer.astro`
- `BaseLayout.astro`
- `MarketingLayout.astro`
- `index.astro`

## Files Changed

- `frontend/src/pages/index.astro`
- `frontend/src/components/layout/Header.astro`
- `frontend/src/components/layout/Footer.astro`
- `frontend/src/layouts/BaseLayout.astro`
- `frontend/src/layouts/MarketingLayout.astro`
- `frontend/src/styles/tokens.css`
- `frontend/src/styles/global.css`
- `frontend/src/pages/robots.txt.ts`

## API Dependencies

None for the homepage.

## SEO Impact

Adds unique metadata, canonical URL, Organization JSON-LD, semantic sections, robots output, sitemap integration, and server-rendered Indonesian content.

## Accessibility Impact

Adds skip navigation, semantic landmarks, visible focus styles, real anchors for navigation, native disclosure controls, reduced-motion handling, heading structure, and descriptive image text.

## Performance Impact

No client framework or hydration is added. The hero is CSS-only; the below-fold preview SVG uses dimensions and lazy loading. No performance measurements were taken.

## Validation

- `npm run format:check` — passed.
- `npm run check` — passed with 0 errors, warnings, or hints across 22 files.
- `npm test` — 3 tests passed.
- `npm run build` with `PUBLIC_SITE_URL=https://example.com` — passed.
- Local HTTP smoke test — `/`, `/auth/masuk`, `/robots.txt`, and `/sitemap-index.xml` returned 200.
- Interactive browser/device visual QA — not verified because no browser surface was available in the current environment.

## Screenshots

Not provided.

## Notes

Pricing is deliberately presented as pending instead of showing invented amounts. Social-proof claims are omitted.
