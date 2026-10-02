# Public Marketing Routes and SEO

Date: 2026-10-02

## Routes added

- `/tema`
- `/tema/editorial-ivory`
- `/tema/botanical-modern`
- `/tema/monochrome-luxe`
- `/fitur`
- `/harga`
- `/inspirasi`
- `/inspirasi/cara-membuat-undangan-digital`

Every route has a unique title, meta description, canonical path, one primary heading, semantic links, and server-rendered content. The theme catalog uses accurate `ItemList` structured data, theme detail pages use `BreadcrumbList`, and the guide uses `Article` data matching visible copy.

Private routes remain excluded: Astro sitemap filtering removes `/i/*` and `/auth/*`, while public invitation pages remain `noindex` unless their owner explicitly enables indexing.

The features page distinguishes working foundations from roadmap items. The pricing page explicitly says purchasing is unavailable, and no ratings, reviews, testimonials, discounts, or usage statistics are fabricated.
