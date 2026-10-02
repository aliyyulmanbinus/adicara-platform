# Theme Catalog and Registry

Date: 2026-10-02

## Outcome

The invitation registry now exposes three original themes:

- `editorial-ivory`
- `botanical-modern`
- `monochrome-luxe`

The public catalog is available at `/tema`. Each registry entry produces a statically generated detail route at `/tema/{key}` with a full server-rendered preview.

## Architecture

Theme packages contain only configuration, presentation CSS, and a thin `Template.astro` wrapper. `SectionRenderer.astro` owns section filtering, ordering, and shared component selection. Invitation content remains an `InvitationData` value and is never embedded in a theme.

This boundary means a future section implementation can be changed once and consumed by every registered theme. A missing template key still falls back to Editorial Ivory on the public invitation route.

## Preview assets

Each theme supplies an original SVG catalog preview under `frontend/public/images`. SVGs contain accessible titles and descriptions; catalog image elements also use descriptive alternative text.

## Validation

The registry is covered by the source-level test suite. Astro type checking and the production build validate all three theme components and generated detail paths.
