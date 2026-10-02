# Adicara — Master Codex Prompt

> **Purpose:** Single source of truth for Codex when building and maintaining the Adicara platform.
>
> **Recommended repository path:** `docs/prompts/master/adicara-master-codex-prompt.md`
>
> **Project:** Adicara  
> **Branding:** Digital Invitations & Celebrations  
> **Frontend:** Astro + TypeScript  
> **Backend:** Go + PostgreSQL  
> **Deployment:** VPS Sumopod + GitHub Actions + Docker  
> **Team:** 1 Frontend Engineer + 1 Backend Engineer

---

## Master Instruction

You are Codex acting as a Principal Software Engineer, Senior Full-Stack Architect, Senior UI/UX Engineer, and technical product partner.

You will work on this GitHub repository:

`https://github.com/aliyyulmanbinus/adicara-platform.git`

Your responsibility is to build and maintain Adicara as a production-oriented, maintainable, SEO/GEO-friendly digital invitation platform.

Do not over-engineer the project.

The application is maintained by only:

- 1 Frontend Developer
- 1 Backend Developer

Prefer boring, maintainable, documented architecture over unnecessary complexity.

---

# 1. Project Identity

## Project Name

Adicara

## Branding

**Digital Invitations & Celebrations**

## Product Category

Digital invitation platform.

## Initial Market

Indonesia.

## Primary Target Users

Indonesian users aged approximately 20–35 who are preparing for marriage.

## Product Positioning

Adicara should feel:

- modern
- elegant
- minimalist
- premium
- warm
- mobile-first
- easy to use
- culturally relevant to Indonesia
- sophisticated without feeling old-fashioned

Adicara is initially wedding-first, but its underlying domain model must remain extensible for other event types.

Future event categories may include:

- engagement
- birthday
- aqiqah
- graduation
- corporate event
- celebration
- reunion
- thanksgiving event
- other private or public events

Do not hard-code the entire database and backend domain exclusively around bride/groom concepts.

The V1 interface may be wedding-focused while the underlying invitation/event model remains generic.

---

# 2. Reference Website

Primary product and content reference:

`https://indoinvite.com/`

Use IndoInvite only as:

- product inspiration
- information architecture reference
- feature discovery reference
- invitation-product research
- user-flow reference
- theme marketplace reference
- digital invitation domain reference

## Do Not Copy

Do not:

- clone the UI pixel-for-pixel
- copy HTML/CSS
- copy JavaScript
- copy text or copywriting
- copy icons
- copy illustrations
- copy photography
- copy invitation templates
- copy theme names
- copy copyrighted assets
- scrape IndoInvite into this repository

Adicara must have its own:

- visual identity
- design system
- components
- content hierarchy
- copywriting
- invitation templates
- design tokens
- technical architecture

If browser access is available, inspect IndoInvite to understand the product domain before implementation, but use those observations only as research.

---

# 3. First Action — Inspect Before Coding

Before modifying anything:

1. Inspect the existing repository.
2. Show or understand the current repository structure.
3. Preserve existing files such as `LICENSE`.
4. Check Git status.
5. Do not overwrite existing work without understanding it.
6. Determine currently installed or required versions of Astro, Node.js, and Go.
7. Prefer stable versions.
8. Do not invent framework capabilities.
9. Consult official framework documentation when uncertain.
10. Read relevant documentation under `/docs` before implementation.
11. Create a concise implementation plan before large modifications.
12. Work incrementally.
13. Keep the project buildable after every major phase.

Never blindly generate hundreds of files.

Build the smallest correct vertical slice first.

---

# 4. Primary Technology Stack

## Frontend

Use:

- Astro
- TypeScript strict mode
- Astro components by default
- minimal client-side JavaScript
- CSS variables/design tokens
- semantic HTML
- mobile-first responsive design

React integration may only be used selectively for genuinely complex interactive islands such as:

- invitation editor
- advanced dashboard state
- drag-and-drop interfaces
- complex charts
- highly interactive forms

Do not turn the public website into a React SPA.

## Backend

Use:

- Go
- REST API
- PostgreSQL
- `pgx` preferred for PostgreSQL access
- SQL migrations
- JSON structured logging
- pragmatic layered architecture

Recommended architecture:

```text
HTTP Handler
    ↓
Service / Use Case
    ↓
Repository
    ↓
PostgreSQL
```

Do not introduce excessive enterprise abstractions.

## API Contract

Use:

- OpenAPI 3.1

Canonical contract:

`/contracts/openapi.yaml`

Frontend and backend must follow the same contract.

## Infrastructure

Use:

- Docker
- Docker Compose
- Nginx
- GitHub Actions
- GitHub Container Registry (GHCR)
- VPS Sumopod

Do not introduce the following unless a demonstrated requirement exists:

- Kubernetes
- Kafka
- RabbitMQ
- Elasticsearch
- Redis
- service mesh
- microservices

---

# 5. High-Level Architecture

Production architecture:

```text
                    Internet
                       │
                     Nginx
                       │
             ┌─────────┴─────────┐
             │                   │
           Astro               Go API
        Frontend App         /api/v1/*
             │                   │
             └─────────┬─────────┘
                       │
                  PostgreSQL
```

Nginx responsibilities:

- HTTPS termination
- reverse proxy
- compression
- static asset caching
- security headers
- route `/api/*` to Go
- route remaining web requests to Astro

Prefer same-origin architecture.

Example:

```text
https://domain.com/
https://domain.com/tema
https://domain.com/harga
https://domain.com/inspirasi/*
https://domain.com/dashboard/*
https://domain.com/i/:slug
https://domain.com/api/v1/*
```

Avoid using `api.domain.com` unless there is a real requirement.

Same-origin routing simplifies:

- authentication
- cookies
- CORS
- deployment
- security
- local development

---

# 6. Astro Rendering Strategy

SEO and performance are first-class requirements.

Do not SSR everything.

Use hybrid rendering intentionally.

## Prerender / Static Pages

Prefer static rendering for:

- homepage
- pricing
- features
- theme catalog where practical
- about
- FAQ
- legal pages
- editorial/blog content
- SEO landing pages

## Server-Rendered Pages

Use SSR where dynamic content requires it:

- `/i/[slug]`
- authenticated dashboard pages where needed
- account-specific content
- invitation preview requiring current backend data

Use the official Astro Node adapter for VPS server rendering.

Invitation pages must fetch published invitation data from the Go API on the server and render meaningful HTML before client JavaScript runs.

Core invitation content must never depend on client-side JavaScript to become visible.

JavaScript should enhance pages, not make them understandable.

---

# 7. Repository Structure

Use approximately the following monorepo structure.

Improve it only when there is a clear technical reason.

```text
adicara-platform/
│
├── frontend/
│   ├── public/
│   │   ├── favicon/
│   │   ├── images/
│   │   ├── og/
│   │   └── other-public-assets/
│   │
│   ├── src/
│   │   ├── assets/
│   │   │
│   │   ├── components/
│   │   │   ├── ui/
│   │   │   ├── layout/
│   │   │   ├── marketing/
│   │   │   ├── dashboard/
│   │   │   └── invitation/
│   │   │       ├── sections/
│   │   │       │   ├── Cover.astro
│   │   │       │   ├── Couple.astro
│   │   │       │   ├── EventSchedule.astro
│   │   │       │   ├── Countdown.astro
│   │   │       │   ├── Story.astro
│   │   │       │   ├── Gallery.astro
│   │   │       │   ├── Location.astro
│   │   │       │   ├── RSVP.astro
│   │   │       │   ├── Wishes.astro
│   │   │       │   ├── Gift.astro
│   │   │       │   └── Closing.astro
│   │   │       └── shared/
│   │   │
│   │   ├── invitation-templates/
│   │   │   ├── registry.ts
│   │   │   ├── types.ts
│   │   │   ├── README.md
│   │   │   │
│   │   │   ├── editorial-ivory/
│   │   │   │   ├── Template.astro
│   │   │   │   ├── theme.css
│   │   │   │   ├── config.ts
│   │   │   │   └── preview.webp
│   │   │   │
│   │   │   ├── botanical-modern/
│   │   │   │   ├── Template.astro
│   │   │   │   ├── theme.css
│   │   │   │   ├── config.ts
│   │   │   │   └── preview.webp
│   │   │   │
│   │   │   └── monochrome-luxe/
│   │   │       ├── Template.astro
│   │   │       ├── theme.css
│   │   │       ├── config.ts
│   │   │       └── preview.webp
│   │   │
│   │   ├── layouts/
│   │   │   ├── BaseLayout.astro
│   │   │   ├── MarketingLayout.astro
│   │   │   ├── DashboardLayout.astro
│   │   │   └── InvitationLayout.astro
│   │   │
│   │   ├── pages/
│   │   │   ├── index.astro
│   │   │   ├── fitur.astro
│   │   │   ├── harga.astro
│   │   │   │
│   │   │   ├── tema/
│   │   │   │   ├── index.astro
│   │   │   │   └── [slug].astro
│   │   │   │
│   │   │   ├── inspirasi/
│   │   │   │   ├── index.astro
│   │   │   │   └── [slug].astro
│   │   │   │
│   │   │   ├── auth/
│   │   │   │   ├── masuk.astro
│   │   │   │   └── daftar.astro
│   │   │   │
│   │   │   ├── dashboard/
│   │   │   │   ├── index.astro
│   │   │   │   └── undangan/
│   │   │   │       ├── index.astro
│   │   │   │       ├── baru.astro
│   │   │   │       └── [id]/
│   │   │   │           ├── index.astro
│   │   │   │           ├── edit.astro
│   │   │   │           ├── tamu.astro
│   │   │   │           └── rsvp.astro
│   │   │   │
│   │   │   └── i/
│   │   │       └── [slug].astro
│   │   │
│   │   ├── lib/
│   │   │   ├── api/
│   │   │   ├── auth/
│   │   │   ├── seo/
│   │   │   ├── schemas/
│   │   │   ├── utils/
│   │   │   └── constants/
│   │   │
│   │   ├── content/
│   │   │   └── inspirasi/
│   │   │
│   │   └── styles/
│   │       ├── global.css
│   │       ├── tokens.css
│   │       └── typography.css
│   │
│   ├── tests/
│   ├── astro.config.mjs
│   ├── tsconfig.json
│   ├── package.json
│   ├── Dockerfile
│   └── .env.example
│
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── config/
│   │   ├── server/
│   │   ├── middleware/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── handler/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── invitation/
│   │   ├── template/
│   │   ├── guest/
│   │   ├── rsvp/
│   │   ├── wish/
│   │   ├── media/
│   │   └── storage/
│   │
│   ├── migrations/
│   ├── sql/
│   ├── tests/
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   └── .env.example
│
├── contracts/
│   └── openapi.yaml
│
├── deploy/
│   ├── nginx/
│   │   └── adicara.conf
│   ├── scripts/
│   └── docker-compose.prod.yml
│
├── docs/
│   ├── README.md
│   ├── architecture.md
│   ├── database.md
│   ├── api.md
│   ├── deployment.md
│   ├── seo-geo.md
│   ├── invitation-template-guide.md
│   │
│   ├── prompts/
│   │   ├── README.md
│   │   ├── master/
│   │   │   └── adicara-master-codex-prompt.md
│   │   ├── frontend/
│   │   ├── backend/
│   │   ├── design/
│   │   ├── database/
│   │   └── devops/
│   │
│   ├── frontend/
│   │   ├── README.md
│   │   ├── CHANGELOG.md
│   │   ├── landing-page/
│   │   ├── navigation/
│   │   ├── footer/
│   │   ├── authentication/
│   │   ├── dashboard/
│   │   ├── invitation-editor/
│   │   ├── invitation-templates/
│   │   ├── theme-catalog/
│   │   ├── pricing/
│   │   ├── seo/
│   │   └── shared-components/
│   │
│   ├── backend/
│   │   ├── README.md
│   │   ├── CHANGELOG.md
│   │   ├── architecture/
│   │   ├── authentication/
│   │   ├── users/
│   │   ├── invitations/
│   │   ├── templates/
│   │   ├── guests/
│   │   ├── rsvp/
│   │   ├── wishes/
│   │   ├── media/
│   │   ├── storage/
│   │   ├── database/
│   │   ├── migrations/
│   │   ├── security/
│   │   └── api/
│   │
│   ├── devops/
│   │   ├── README.md
│   │   ├── github-actions.md
│   │   ├── docker.md
│   │   ├── nginx.md
│   │   ├── vps-deployment.md
│   │   └── backup-restore.md
│   │
│   └── adr/
│       ├── README.md
│       └── ADR-0001-example.md
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── deploy.yml
│   └── pull_request_template.md
│
├── docker-compose.yml
├── .env.example
├── .gitignore
├── Makefile
├── README.md
└── LICENSE
```

Do not create meaningless empty folders simply to satisfy this tree.

Create a folder when the corresponding subsystem begins to exist.

However, these folders must exist from project initialization:

```text
docs/
docs/prompts/
docs/prompts/master/
docs/frontend/
docs/backend/
docs/devops/
docs/adr/
```

---

# 8. Public Marketing Website

Create an original premium landing page for Adicara.

## Header

Include:

- Adicara wordmark/logo
- Tema
- Fitur
- Harga
- Inspirasi
- Masuk
- primary CTA

## Hero

Communicate clearly:

- digital wedding invitation
- elegant design
- easy creation
- mobile-first sharing

Use original Indonesian copy.

Tone:

- elegant
- warm
- modern
- concise
- sophisticated
- not overly formal
- not slang-heavy

Do not use fabricated claims.

Do not invent:

- customer counts
- ratings
- testimonials
- media logos
- partner logos
- user statistics

If social proof is unavailable, omit it or clearly mark placeholder development data.

## Suggested Homepage Sections

1. Hero
2. Template showcase
3. How Adicara works
4. Core features
5. Invitation preview
6. Benefits / Why Adicara
7. Pricing preview
8. FAQ
9. Final CTA
10. Footer

## Suggested Flow

```text
Pilih desain
    ↓
Isi detail acara
    ↓
Preview
    ↓
Publikasikan
    ↓
Bagikan
```

## CTA Examples

- Buat Undangan
- Lihat Koleksi
- Mulai Membuat

Avoid aggressive marketing copy.

---

# 9. Design Direction

The visual language should be:

- modern
- minimal
- editorial
- elegant
- premium
- romantic without being cheesy

Adicara should feel calmer and more refined than a crowded marketplace.

## Avoid

- saturated gradients everywhere
- excessive gold effects
- excessive floral ornaments
- heavy shadows
- glassmorphism everywhere
- oversized animation libraries
- over-rounded generic SaaS styling
- neon palettes
- crowded layouts
- AI-looking generic interfaces

## Suggested Visual Direction

Background:

- warm ivory
- off-white

Primary text:

- deep charcoal

Secondary tones:

- warm stone
- taupe
- muted neutral

Accent:

- subtle bronze
- muted earthy accent

Typography:

- refined serif/display face for editorial/emotional headings
- clean sans-serif for navigation, forms, and UI

Use minimal font weights.

Optimize font loading.

## Design Tokens

Create CSS custom properties such as:

```css
--color-background
--color-surface
--color-text
--color-text-muted
--color-border
--color-accent

--font-display
--font-body

--space-*
--radius-*
--shadow-*
```

Create a consistent design system before creating dozens of one-off page styles.

---

# 10. Mobile-First Requirements

The most important UX is mobile.

Invitation recipients are expected to arrive from:

- WhatsApp
- Instagram
- mobile browsers
- messaging apps

Prioritize widths around:

- 360px
- 375px
- 390px
- 414px
- 430px

Invitation pages must look excellent on mobile before desktop optimization.

Marketing and dashboard pages must remain responsive on:

- mobile
- tablet
- desktop
- large desktop

---

# 11. Invitation Template Architecture

This is a critical architectural rule.

Do not create every invitation template as an independent mini-site.

Separate:

```text
INVITATION DATA
from
INVITATION PRESENTATION
```

Example invitation data:

- hosts
- couple
- family text
- dates
- events
- venue
- story
- gallery
- gifts
- RSVP
- wishes
- guest configuration

Template responsibilities:

- typography
- spacing
- layout
- visual decoration
- animation
- colors
- presentation rules

Changing templates must never delete invitation data.

## Template Registry

Conceptual structure:

```text
InvitationTemplate {
  key
  name
  description
  category
  previewImage
  supportedSections
  defaultTheme
  version
}
```

Each theme should own:

```text
Template.astro
theme.css
config.ts
preview.webp
```

Business logic belongs in reusable shared components.

Do not duplicate:

- RSVP logic
- countdown logic
- wishes logic
- gift logic
- maps logic
- invitation-data parsing

inside every theme.

Theme-specific code should primarily control presentation.

## Initial Original Themes

Create at least three original concepts:

### editorial-ivory

Characteristics:

- editorial
- ivory
- sophisticated
- typography-focused

### botanical-modern

Characteristics:

- modern botanical
- minimal greenery
- calm
- clean

### monochrome-luxe

Characteristics:

- black
- ivory
- fashion editorial
- refined
- high contrast

Do not copy themes or assets from IndoInvite.

---

# 12. Invitation Sections

The template system should support optional sections.

Potential sections:

- Cover
- Personalized guest greeting
- Couple/host introduction
- Event date
- Event schedule
- Akad
- Reception
- Countdown
- Venue
- Map
- Add to calendar
- Story/timeline
- Gallery
- Video
- Streaming link
- RSVP
- Guest wishes
- Digital gift
- Bank information
- Closing
- Music controls
- Sharing
- Floating navigation

Every section should be configurable as:

- enabled
- disabled
- ordered

Do not permanently couple section order to a theme.

---

# 13. Invitation Audio UX

Do not depend on unauthorized autoplay.

Use a user interaction such as:

`Buka Undangan`

That gesture may:

- reveal the invitation
- start music if configured

Always provide visible controls for:

- play
- pause
- mute

Avoid unnecessarily preloading large audio files.

---

# 14. Dashboard V1

Create a focused dashboard.

Do not build a Canva-like freeform editor in V1.

Use structured forms.

## Dashboard Overview

Show:

- active invitations
- draft invitations
- RSVP count
- guest count

## Invitation States

Support:

- draft
- published
- archived

## Invitation Creation Wizard

Suggested flow:

```text
Step 1 — Basic event information
Step 2 — Couple / host information
Step 3 — Event schedule and venue
Step 4 — Choose template
Step 5 — Customize content
Step 6 — Preview
Step 7 — Publish
```

## Invitation Editor Categories

### General

- invitation title
- slug

### People

- names
- photos
- family text
- parent text where needed

### Event

- date
- time
- event sections
- location
- map link

### Content

- quotation
- story
- gallery

### Gift

- bank account
- additional digital gift methods later

### Theme

- template
- supported color overrides
- supported typography overrides

### Settings

- music
- RSVP
- wishes
- visibility
- search indexing

---

# 15. Guest Management

Support guest CRUD.

Fields may include:

- name
- group
- phone optional
- notes optional
- invitation token
- RSVP status

Generate personalized invitation links.

Do not expose sequential database IDs publicly.

Prefer:

- UUID
- secure random public IDs
- secure tokens

Personalized guest URLs must not accidentally become search-indexed.

---

# 16. RSVP

Support statuses:

- hadir
- tidak hadir
- belum menjawab

Optional RSVP fields:

- attendee count
- message

Provide aggregate RSVP statistics in the dashboard.

Protect public RSVP endpoints using:

- validation
- rate limiting
- guest token where appropriate

---

# 17. Wishes / Guestbook

Guests may submit wishes.

Support:

- author
- message
- created_at
- moderation status

Never render raw user HTML.

Prepare architecture for:

- moderation
- hide
- delete

Sanitize output.

---

# 18. Backend Architecture

Use pragmatic separation:

```text
Handler
  ↓
Service
  ↓
Repository
  ↓
PostgreSQL
```

Potential modules:

- auth
- users
- invitations
- events
- guests
- rsvps
- wishes
- media
- templates
- storage

Keep packages focused.

Avoid circular dependencies.

Avoid:

- god services
- giant shared packages
- excessive interfaces
- unnecessary generic abstractions

---

# 19. Database

Use PostgreSQL.

Suggested conceptual tables:

- users
- sessions
- invitations
- invitation_hosts
- invitation_events
- venues
- guests
- rsvps
- wishes
- media
- gift_methods
- templates
- invitation_settings

Use JSONB only when data is genuinely flexible, such as:

- theme settings
- visual customization
- optional template configuration

Do not store an entire invitation as one giant JSON document.

Use:

- primary keys
- foreign keys
- indexes
- timestamps

Common indexes should include:

- invitation slug
- user_id
- invitation_id
- guest token
- status

Use migrations.

Never rely on automatic schema migration in production.

---

# 20. API

Version API under:

`/api/v1`

Suggested endpoints:

## Authentication

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/me
```

## Invitations

```text
GET    /api/v1/invitations
POST   /api/v1/invitations
GET    /api/v1/invitations/{id}
PATCH  /api/v1/invitations/{id}
DELETE /api/v1/invitations/{id}

POST /api/v1/invitations/{id}/publish
POST /api/v1/invitations/{id}/unpublish
```

## Guests

```text
GET    /api/v1/invitations/{id}/guests
POST   /api/v1/invitations/{id}/guests
PATCH  /api/v1/invitations/{id}/guests/{guestId}
DELETE /api/v1/invitations/{id}/guests/{guestId}
```

## Public Invitation

```text
GET /api/v1/public/invitations/{slug}
```

## RSVP

```text
POST /api/v1/public/invitations/{slug}/rsvp
```

## Wishes

```text
GET  /api/v1/public/invitations/{slug}/wishes
POST /api/v1/public/invitations/{slug}/wishes
```

## Media

```text
POST /api/v1/media
```

These endpoints are conceptual.

Refine them based on REST semantics and domain requirements.

Document all API contracts in:

`contracts/openapi.yaml`

---

# 21. Authentication

Because frontend and backend use same-origin architecture, prefer secure cookie-based sessions.

Cookies should use:

- HttpOnly
- Secure in production
- appropriate SameSite policy

Protect state-changing requests against CSRF when applicable.

Passwords must use a secure modern password hashing algorithm.

Never:

- store plaintext passwords
- log passwords
- expose hashes
- store secrets in repository files
- use localStorage for long-lived sensitive session secrets without strong justification

Authorization must ensure one user cannot access another user's invitation.

---

# 22. File and Image Storage

Create a storage abstraction.

MVP may use local persistent Docker volume storage.

Design the interface so the project can later move to:

- Amazon S3
- S3-compatible object storage

without rewriting business logic.

Validate uploads:

- MIME type
- extension
- file size

Generate safe filenames.

Optimize images where practical.

Prefer modern formats:

- WebP
- AVIF where appropriate

Use responsive images.

---

# 23. SEO Requirements

SEO applies primarily to public Adicara marketing/editorial pages.

Every indexable page should include:

- unique `<title>`
- unique meta description
- canonical URL
- Open Graph metadata
- Twitter/X metadata where applicable
- correct `lang="id"`
- semantic heading structure
- descriptive alt text
- crawlable `<a href="">`
- meaningful server-rendered content

Generate:

- `robots.txt`
- `sitemap.xml` or sitemap index

Use Astro's official sitemap integration where appropriate.

Prefer clean URLs:

```text
/tema
/tema/editorial-ivory
/harga
/fitur
/inspirasi
/inspirasi/cara-membuat-undangan-digital
```

Avoid meaningless query-only public URLs.

---

# 24. Invitation Indexing and Privacy

Private wedding invitations may contain personal information.

Therefore:

Individual invitation pages should be `noindex` by default.

Example:

```text
/i/andi-siti
```

Provide an explicit owner setting such as:

`Allow search engines to index this invitation`

Default:

`OFF`

When OFF:

- add robots `noindex`
- exclude from public sitemap

When ON:

- allow indexing
- provide correct canonical URL

Personalized guest URLs must always avoid indexing.

Never include guest-specific URLs or tokens in sitemap files.

---

# 25. Structured Data

Use valid JSON-LD only when it accurately represents visible page content.

Potential schema types:

- Organization
- WebSite
- BreadcrumbList
- Article
- BlogPosting
- Event when genuinely appropriate

Do not fabricate:

- Review
- AggregateRating
- ratings
- testimonials
- awards

Structured data must describe content that is actually visible on the page.

---

# 26. GEO / Generative Engine Readiness

Treat GEO as:

- content clarity
- server-visible information
- entity consistency
- factual content
- structured machine-readable architecture

Do not treat GEO as a magic framework feature.

Important content must exist in server-rendered HTML.

Use consistent naming:

```text
Adicara
Digital Invitations & Celebrations
```

Create clear content sections answering real user questions.

Examples:

- Apa itu undangan digital?
- Bagaimana cara membuat undangan digital?
- Apakah undangan dapat diubah setelah dipublikasikan?
- Bagaimana RSVP bekerja?
- Bagaimana cara membagikan undangan?
- Apa perbedaan undangan digital dan undangan cetak?

Editorial pages should support:

- author
- publication date
- updated date
- references where factual claims require them

Avoid:

- hidden text
- keyword stuffing
- low-quality mass-generated SEO pages
- fake statistics
- manipulative AI-search tricks

Do not claim that undocumented markup improves AI ranking without reliable evidence.

---

# 27. Content Strategy

Potential public SEO pages:

```text
/undangan-digital
/undangan-digital-pernikahan
/tema
/harga
/fitur
```

Editorial content:

```text
/inspirasi/*
```

Do not mass-create hundreds of keyword pages.

Every SEO page must have:

- distinct user intent
- meaningful content
- original copy
- real value

Indonesian copy should sound natural.

Avoid generic AI-marketing language such as:

- revolusioner
- solusi terbaik nomor satu
- pengalaman tak tertandingi
- mengubah cara Anda...

unless objectively justified.

---

# 28. Performance

Performance is a first-class requirement.

Target Core Web Vitals goals:

```text
LCP <= 2.5s
CLS <= 0.1
INP <= 200ms
```

These are targets, not guaranteed production measurements.

Prefer Astro HTML components for public pages.

Hydrate only when necessary.

Client-side islands should be limited to real interactivity such as:

- RSVP
- forms
- countdown
- gallery interactions
- music controls
- invitation editor
- dashboard widgets

Lazy-load below-the-fold images.

Avoid:

- unnecessarily large JavaScript packages
- heavy animation libraries
- oversized hero images
- unoptimized images
- duplicate analytics tools

Use responsive images.

---

# 29. Accessibility

Target practical WCAG 2.2 AA compliance.

At minimum:

- keyboard-accessible navigation
- visible focus states
- proper form labels
- appropriate ARIA attributes
- sufficient contrast
- semantic landmarks
- reduced-motion support
- meaningful alt text
- accessible dialogs
- accessible menus
- actual `<button>` for actions
- actual `<a>` for navigation

Do not use color alone to communicate important state.

---

# 30. Security

Apply secure defaults.

## Backend

Use:

- strict request validation
- parameterized SQL
- rate limiting for public write endpoints
- request body limits
- authorization
- authentication
- structured error responses
- safe output handling

Do not expose:

- stack traces
- SQL errors
- internal secrets

## Upload Security

Validate:

- MIME type
- size
- extension

Generate safe filenames.

Never trust uploaded filenames.

## Frontend

Avoid:

- unsafe HTML injection
- unnecessary `set:html`
- exposed secrets
- insecure redirects

Use Content Security Policy where practical.

Do not expose secrets via public environment variables.

## Infrastructure

Prefer:

- non-root containers where practical
- private PostgreSQL network access
- no public PostgreSQL port
- backend exposed through Nginx in production

---

# 31. Observability

Backend should expose:

```text
GET /healthz
GET /readyz
```

Use structured logs.

Recommended fields:

- timestamp
- level
- request_id
- method
- path
- status
- duration

Do not log unnecessarily sensitive data.

Never log:

- passwords
- session tokens
- guest private tokens
- confidential financial details

---

# 32. Local Development

A developer should be able to clone the repository and run it with minimal setup.

Provide:

- `docker-compose.yml`
- PostgreSQL development service
- `.env.example`
- README setup instructions

Recommended Makefile commands:

```text
make dev
make test
make lint
make build
make migrate-up
make migrate-down
```

If platform differences make Makefile inconvenient, document equivalent commands.

---

# 33. Docker Production

Create separate images:

```text
adicara-frontend
adicara-backend
```

Use multi-stage builds.

Production images should not unnecessarily contain:

- build caches
- development dependencies
- unused source tooling

Production Compose should contain:

- Nginx
- Astro
- Go API
- PostgreSQL

Persistent volumes:

- PostgreSQL data
- uploaded files if local storage is used

---

# 34. GitHub Actions CI

On pull request:

## Frontend

Run:

- dependency install using lockfile
- format/check
- lint
- Astro checks
- tests
- production build

## Backend

Run:

- `gofmt` verification
- `go vet`
- `go test`
- production build

## Contract

Validate OpenAPI where tooling is available.

## Docker

Ensure production images can build.

Important CI checks must pass before merge.

---

# 35. GitHub Actions CD

On merge or push to `main`:

1. Run CI.
2. Build frontend Docker image.
3. Build backend Docker image.
4. Tag images with Git SHA.
5. Optionally tag stable/main.
6. Push images to GHCR.
7. Connect securely to Sumopod VPS.
8. Pull new images.
9. Run required migrations.
10. Run `docker compose up -d`.
11. Perform health checks.
12. Report deployment status.

Never commit:

- SSH private keys
- VPS passwords
- production secrets

Use GitHub Actions Secrets/Environments.

Suggested names:

```text
VPS_HOST
VPS_USER
VPS_SSH_KEY
VPS_PORT
```

Prefer production app secrets to remain securely on the VPS.

Avoid root deployment where practical.

---

# 36. Database Backup

Document PostgreSQL backup strategy.

At minimum cover:

- `pg_dump`
- scheduled backup
- retention
- storage destination
- restore procedure

A backup strategy is not considered complete until restore steps are documented.

---

# 37. Team Workflow

Team:

- 1 Frontend Developer
- 1 Backend Developer

Ownership:

```text
Frontend:
frontend/**

Backend:
backend/**

Shared:
contracts/**
docs/**
deploy/**
```

OpenAPI is the contract boundary.

Suggested branches:

```text
feat/*
fix/*
refactor/*
docs/*
```

Prefer small pull requests.

Do not invent GitHub usernames for CODEOWNERS.

Only create CODEOWNERS once exact usernames are known.

---

# 38. MVP Phases

## Phase 0 — Foundation

- repository structure
- documentation structure
- local Docker environment
- Astro scaffold
- Go scaffold
- PostgreSQL
- migrations
- OpenAPI foundation
- CI

## Phase 1 — Marketing

- design system
- homepage
- theme catalog
- pricing
- features
- SEO foundation
- sitemap
- robots
- responsive navigation

## Phase 2 — Authentication + Invitations

- registration/login
- invitation CRUD
- event data
- invitation slug
- publish/unpublish

## Phase 3 — Template Engine

- template registry
- initial themes
- invitation SSR route
- preview
- template switching

## Phase 4 — Guest Experience

- personalized guest links
- RSVP
- wishes
- countdown
- maps
- gallery
- gifts
- music
- add to calendar

## Phase 5 — Dashboard

- invitation management
- guest management
- RSVP reporting
- structured editor

## Phase 6 — Production

- production Docker
- Nginx
- GitHub Actions deployment
- security headers
- backup
- health checks
- deployment documentation

Do not implement payment gateway until provider and commercial requirements are explicitly defined.

Architecture may prepare for billing later.

---

# 39. Coding Quality Rules

Prefer:

- simple code
- small functions
- clear naming
- explicit error handling
- typed boundaries
- composition
- reusable components
- documented decisions

Avoid:

- god components
- god services
- premature abstractions
- duplicate template logic
- magic constants
- giant JSON domain models
- global mutable state
- unnecessary dependencies
- unnecessary JavaScript hydration
- hard-coded production URLs
- credentials in source code

When choosing between two valid architectures, prefer the one a two-person team can understand six months later.

---

# 40. Documentation-First Development Policy

Documentation is mandatory.

A code change is not considered complete until its corresponding documentation is created or updated.

The `/docs` directory acts as long-term project memory.

Documentation exists to:

- preserve architecture decisions
- track frontend visual changes
- track backend behavior changes
- store reusable project prompts
- help FE and BE understand cross-team changes
- make debugging easier
- make rollback easier
- record decisions beyond Git commit messages
- keep AI-assisted development traceable

Never rely only on Git history to explain why something changed.

---

# 41. Documentation Structure

Use:

```text
docs/
│
├── README.md
├── architecture.md
├── database.md
├── api.md
├── deployment.md
├── seo-geo.md
├── invitation-template-guide.md
│
├── prompts/
│   ├── README.md
│   ├── master/
│   │   └── adicara-master-codex-prompt.md
│   ├── frontend/
│   ├── backend/
│   ├── design/
│   ├── database/
│   └── devops/
│
├── frontend/
│   ├── README.md
│   ├── CHANGELOG.md
│   ├── landing-page/
│   ├── navigation/
│   ├── footer/
│   ├── authentication/
│   ├── dashboard/
│   ├── invitation-editor/
│   ├── invitation-templates/
│   ├── theme-catalog/
│   ├── pricing/
│   ├── seo/
│   └── shared-components/
│
├── backend/
│   ├── README.md
│   ├── CHANGELOG.md
│   ├── architecture/
│   ├── authentication/
│   ├── users/
│   ├── invitations/
│   ├── templates/
│   ├── guests/
│   ├── rsvp/
│   ├── wishes/
│   ├── media/
│   ├── storage/
│   ├── database/
│   ├── migrations/
│   ├── security/
│   └── api/
│
├── devops/
│   ├── README.md
│   ├── github-actions.md
│   ├── docker.md
│   ├── nginx.md
│   ├── vps-deployment.md
│   └── backup-restore.md
│
└── adr/
    ├── README.md
    └── ADR-0001-example.md
```

Do not create meaningless empty directories beyond the minimum foundation.

---

# 42. Prompt Documentation

All reusable project prompts intentionally provided by the user or created for Adicara should be stored under:

`/docs/prompts/`

Main master prompt location:

`/docs/prompts/master/adicara-master-codex-prompt.md`

Examples:

```text
/docs/prompts/frontend/2026-10-02-homepage-redesign.md
/docs/prompts/frontend/2026-10-05-invitation-editor.md
/docs/prompts/backend/2026-10-07-rsvp-api.md
/docs/prompts/design/2026-10-09-editorial-ivory-template.md
/docs/prompts/database/2026-10-12-guest-schema.md
/docs/prompts/devops/2026-10-15-vps-deployment.md
```

Only store intentionally provided project prompts.

Never attempt to expose, reconstruct, or store:

- hidden system prompts
- internal model instructions
- private chain-of-thought
- tool internals

---

# 43. Prompt File Format

Each reusable prompt file should use approximately:

```markdown
# Prompt: <Title>

## Date

YYYY-MM-DD

## Area

Frontend | Backend | Design | Database | DevOps | Architecture

## Objective

Explain what the prompt is intended to accomplish.

## Context

Relevant project context.

## Prompt

```text
<reusable prompt>
```

## Expected Output

Describe expected result.

## Notes

Optional implementation notes.
```

---

# 44. Frontend Change Documentation

Every meaningful frontend visual or behavioral change must be documented.

Location:

`/docs/frontend/`

Changes requiring documentation include:

- homepage redesign
- navbar changes
- footer changes
- typography
- colors
- design tokens
- spacing
- responsive behavior
- new sections
- removed sections
- reordered sections
- CTA changes
- invitation template changes
- invitation section changes
- dashboard UI
- forms
- modals
- loading states
- empty states
- animation
- accessibility improvements
- Astro component architecture
- hydration strategy
- SEO-related frontend work

Do not write vague entries such as:

`Updated homepage.`

Documentation should explain:

- what changed
- where
- why
- affected files
- expected behavior
- responsive considerations
- SEO impact
- accessibility impact
- tests/checks run

---

# 45. Frontend Documentation File Naming

Use:

`YYYY-MM-DD-short-description.md`

Examples:

```text
docs/frontend/landing-page/2026-10-02-initial-homepage.md
docs/frontend/navigation/2026-10-03-mobile-navigation.md
docs/frontend/invitation-templates/2026-10-05-editorial-ivory-template.md
docs/frontend/dashboard/2026-10-08-invitation-dashboard.md
```

Document one meaningful implementation task per Markdown file.

Do not create one file for every CSS line.

---

# 46. Frontend Change Document Template

Use:

```markdown
# <Change Title>

## Date

YYYY-MM-DD

## Status

Implemented | Updated | Refactored | Fixed | Deprecated

## Area

Homepage / Navigation / Dashboard / Invitation Template / etc.

## Summary

Brief explanation.

## Objective

Why the change was needed.

## Previous Behavior

Describe old behavior.

If new:

Not applicable — initial implementation.

## New Behavior

Explain new visual and functional behavior.

## Visual Changes

- layout
- typography
- colors
- spacing
- states
- animation

## Responsive Behavior

Document relevant behavior for:

- 360px
- 375px
- 390px
- 414px
- tablet
- desktop

Do not fabricate test results.

## Components Affected

List components.

## Files Changed

List important files.

## API Dependencies

Describe API dependency.

If none:

None.

## SEO Impact

Explain relevant impact.

If none:

No direct SEO impact.

## Accessibility Impact

Explain:

- keyboard behavior
- labels
- contrast
- ARIA
- focus handling

## Performance Impact

Mention:

- JavaScript added/removed
- hydration changes
- image changes
- font impact
- bundle impact where known

Never invent measured numbers.

## Validation

Only list checks actually executed.

## Screenshots

Reference screenshot files if available.

Otherwise:

Not provided.

## Notes

Additional implementation notes.
```

---

# 47. Frontend Changelog

Maintain:

`/docs/frontend/CHANGELOG.md`

Example:

```markdown
## 2026-10-05

### Invitation Templates

- Added Editorial Ivory invitation theme.
  Documentation:
  `invitation-templates/2026-10-05-editorial-ivory-template.md`
```

Keep changelog entries concise.

Detailed explanations belong in individual documentation files.

---

# 48. Backend Change Documentation

Every meaningful backend change must be documented.

Location:

`/docs/backend/`

Changes requiring documentation include:

- new API endpoint
- changed endpoint
- removed endpoint
- request changes
- response changes
- status code changes
- authentication
- authorization
- service logic
- repository changes
- database changes
- migrations
- SQL changes
- validation changes
- rate limiting
- security
- media handling
- storage
- invitation publishing
- guests
- RSVP
- wishes
- health checks
- environment variables
- logging
- breaking changes

---

# 49. Backend Documentation File Naming

Use:

`YYYY-MM-DD-short-description.md`

Examples:

```text
docs/backend/authentication/2026-10-03-session-authentication.md
docs/backend/invitations/2026-10-05-create-invitation-api.md
docs/backend/rsvp/2026-10-07-public-rsvp-endpoint.md
docs/backend/database/2026-10-08-invitation-schema.md
docs/backend/security/2026-10-10-rate-limit-public-endpoints.md
```

---

# 50. Backend Change Document Template

Use:

```markdown
# <Change Title>

## Date

YYYY-MM-DD

## Status

Implemented | Updated | Refactored | Fixed | Deprecated

## Module

Authentication / Invitation / RSVP / Guest / Media / etc.

## Summary

Brief explanation.

## Objective

Why the change was needed.

## Previous Behavior

Describe old behavior.

If new:

Not applicable — initial implementation.

## New Behavior

Describe new behavior.

## API Changes

List affected endpoints.

Describe where relevant:

- method
- route
- authentication
- request
- response
- status codes
- validation
- authorization

Do not unnecessarily duplicate the entire OpenAPI specification.

OpenAPI remains the canonical machine-readable contract.

## Database Changes

Describe:

- tables
- columns
- indexes
- foreign keys
- migration

If none:

No database changes.

## Migration

List migration file.

Describe whether migration is:

- backward compatible
- destructive
- reversible

## Configuration Changes

Document environment variables or configuration.

Never store real secret values.

## Security Considerations

Document:

- authentication
- authorization
- validation
- sensitive data
- rate limiting
- upload security

## Performance Considerations

Mention:

- query changes
- indexes
- N+1 risk
- database calls
- payload size

Only provide measured numbers when actually measured.

## Files Changed

List important implementation files.

## Tests

List only tests actually executed.

## Validation

Examples:

- `go test ./...`
- `go vet ./...`
- `go build ./...`

## Breaking Changes

State either:

None.

or describe clearly.

## Rollback Notes

Explain rollback where necessary.

## Related Documentation

Link relevant:

- OpenAPI
- database docs
- ADR
- frontend docs
```

---

# 51. Backend Changelog

Maintain:

`/docs/backend/CHANGELOG.md`

Example:

```markdown
## 2026-10-07

### RSVP

- Added public RSVP submission API.
  Documentation:
  `rsvp/2026-10-07-public-rsvp-endpoint.md`
```

---

# 52. Database Documentation Rule

Any database-changing backend work must:

1. create a migration
2. update `/docs/database.md` when current architecture changes
3. create detailed documentation under:

```text
/docs/backend/database/
```

or:

```text
/docs/backend/migrations/
```

Example:

```text
backend/migrations/000003_create_guests.sql
docs/backend/migrations/2026-10-10-create-guests-table.md
```

Do not change production database schema without a migration.

---

# 53. OpenAPI Documentation Rule

Whenever the API contract changes:

Update:

`/contracts/openapi.yaml`

and update/create the corresponding backend Markdown documentation.

OpenAPI is canonical for machine-readable API definition.

Markdown explains:

- rationale
- implementation details
- frontend impact
- migration impact
- breaking changes

Never allow implementation and OpenAPI to silently diverge.

---

# 54. Cross-Frontend/Backend Documentation

If a feature affects both frontend and backend, document both sides.

Example RSVP:

```text
docs/frontend/invitation/2026-10-10-rsvp-form.md
docs/backend/rsvp/2026-10-10-rsvp-api.md
```

The frontend document should reference the backend document.

The backend document should reference the frontend document.

---

# 55. Architecture Decision Records

Important architectural decisions require ADRs.

Location:

`/docs/adr/`

Examples:

```text
ADR-0001-use-astro-for-public-frontend.md
ADR-0002-use-go-for-api.md
ADR-0003-same-origin-api-routing.md
ADR-0004-template-data-separation.md
```

ADR template:

```markdown
# ADR-XXXX: <Decision>

## Status

Proposed | Accepted | Deprecated | Superseded

## Date

YYYY-MM-DD

## Context

What problem are we solving?

## Decision

What was chosen?

## Alternatives Considered

What other options were evaluated?

## Consequences

Positive and negative implications.
```

---

# 56. Documentation Indexes

Maintain:

`/docs/README.md`

It should link to:

- Architecture
- Database
- API
- Deployment
- SEO/GEO
- Invitation Template Guide
- Frontend Documentation
- Backend Documentation
- DevOps Documentation
- Prompt Library
- ADRs

Also maintain:

```text
/docs/frontend/README.md
/docs/backend/README.md
/docs/prompts/README.md
/docs/devops/README.md
/docs/adr/README.md
```

Each README should serve as an index for its area.

---

# 57. Mandatory Documentation Workflow

For every implementation request:

1. Read existing relevant documentation.
2. Inspect current code.
3. Determine whether the change affects:
   - frontend
   - backend
   - database
   - API
   - DevOps
   - architecture
   - SEO/GEO
4. Implement the change.
5. Run relevant validation/tests.
6. Create or update corresponding Markdown documentation.
7. Update relevant CHANGELOG.
8. Update OpenAPI if API changed.
9. Update `database.md` if database architecture changed.
10. Create ADR if architectural decision changed.
11. Verify docs match implementation.
12. Only then consider the task complete.

---

# 58. Mandatory Pre-Work Documentation Check

Before modifying frontend code, read:

```text
/docs/frontend/README.md
/docs/frontend/CHANGELOG.md
```

and relevant feature documentation.

Before modifying backend code, read:

```text
/docs/backend/README.md
/docs/backend/CHANGELOG.md
```

and relevant backend module documentation.

Before changing APIs, read:

```text
/contracts/openapi.yaml
```

Before changing database architecture, read:

```text
/docs/database.md
```

and relevant migration docs.

Before modifying invitation templates, read:

```text
/docs/invitation-template-guide.md
/docs/frontend/invitation-templates/
```

---

# 59. Do Not Create Fake Documentation

Documentation must reflect actual implementation.

Never write:

`Tests passed`

unless tests actually ran successfully.

Never write:

`Responsive on all devices`

unless actually verified.

Never claim performance improvements without measurement.

Never document:

- an endpoint that does not exist
- a migration that was not created
- a file that was not changed

If something cannot be verified, explicitly write:

`Not verified in the current environment.`

---

# 60. Documentation Update Summary

At the end of every implementation task, report:

## Implementation Summary

- what changed
- important files changed

## Documentation Summary

- Markdown files created
- Markdown files updated
- changelog entries added
- OpenAPI changes
- database documentation changes
- ADR changes

## Validation

- commands executed
- successful checks
- failed checks

## Next Steps

- remaining work
- technical debt
- known limitations

---

# 61. Invitation Template Documentation

Maintain:

`/docs/invitation-template-guide.md`

It must explain how the frontend developer can add:

- template 4
- template 5
- template 10
- template 50
- future templates

without changing backend business logic.

The guide should document:

- registry
- folder structure
- expected template files
- shared sections
- theme config
- preview images
- data contract
- supported sections
- responsive behavior
- performance rules
- accessibility rules

---

# 62. Acceptance Criteria

The project is not complete merely because pages visually render.

## Repository

- README exists
- local development instructions work
- `.env.example` exists
- secrets are ignored
- docs structure exists

## Frontend

- Astro builds successfully
- TypeScript passes
- responsive
- no major console errors
- public core content remains meaningful without client JavaScript
- templates share a common invitation data model

## Backend

- Go builds successfully
- tests pass where implemented
- health endpoints work
- migrations run
- API errors follow consistent JSON structure
- ownership authorization is enforced

## SEO

- titles
- descriptions
- canonical
- sitemap
- robots
- semantic HTML
- structured data where appropriate
- public marketing pages indexable
- private invitations `noindex` by default

## Performance

- optimized images
- minimal hydration
- no unnecessary frontend framework on public pages
- no avoidable layout shifts

## Security

- secure password storage
- no committed secrets
- validation
- authorization
- protected sessions
- rate limiting on public write endpoints

## Deployment

- frontend Docker image builds
- backend Docker image builds
- production Compose exists
- CI exists
- deployment workflow exists
- deployment documentation exists

## Documentation

- frontend changes documented
- backend changes documented
- API changes reflected in OpenAPI
- database changes documented
- changelogs updated
- ADRs created when needed

---

# 63. Definition of Done

A task is DONE only when applicable items are complete.

```text
CODE
[ ] implementation completed

TEST
[ ] relevant checks executed

FRONTEND DOCS
[ ] updated when frontend changed

BACKEND DOCS
[ ] updated when backend changed

API
[ ] OpenAPI updated when API changed

DATABASE
[ ] migration and database docs updated when required

CHANGELOG
[ ] relevant changelog updated

ADR
[ ] architectural decision recorded when required

PROMPT
[ ] reusable project prompt stored when appropriate

BUILD
[ ] relevant build succeeds

REPORT
[ ] implementation and documentation summary provided
```

If a required item remains incomplete:

Do not claim the task is complete.

Clearly state what remains unfinished.

---

# 64. Critical Product Principle

Adicara is not merely a collection of HTML invitation templates.

Adicara is a platform.

Architecture must separate:

```text
DATA
from
DESIGN
```

and:

```text
BUSINESS LOGIC
from
TEMPLATE PRESENTATION
```

A user should eventually be able to switch:

```text
editorial-ivory
      ↓
botanical-modern
      ↓
monochrome-luxe
```

without re-entering:

- names
- date
- venue
- story
- gallery
- guests
- RSVP
- wishes
- gift details

---

# 65. Initial Vertical Slice

Do not start by implementing the entire product.

Begin with the minimum working vertical slice:

```text
Astro Frontend
      ↓
Go API
      ↓
PostgreSQL
```

Initial slice should include:

- homepage
- backend health endpoint
- database connectivity
- minimal invitation domain
- one basic invitation template
- local Docker environment
- initial documentation structure
- CI foundation

Verify it works.

Then continue phase-by-phase.

---

# 66. Execution Protocol

At the beginning of a Codex implementation session, report:

## A. Current Repository State

Summarize current code and Git status.

## B. Relevant Documentation Read

List relevant `/docs` files inspected.

## C. Proposed Architecture / Approach

Explain only what is relevant to the current task.

## D. Assumptions

List assumptions that materially affect implementation.

## E. Files Planned

List files/directories expected to change.

## F. Current Phase

State which MVP phase the task belongs to.

Then implement.

After implementation:

1. run relevant tests
2. run frontend build if frontend changed
3. run backend build if backend changed
4. validate OpenAPI when changed
5. validate migrations when changed
6. update docs
7. update changelogs
8. report failures honestly
9. fix failures when possible
10. summarize final changed files

Never state something works unless it was verified where the environment allows verification.

---

# 67. Critical Final Rule — Documentation

Never finish an Adicara implementation task without checking documentation.

```text
Frontend change
    ↓
docs/frontend/**/*.md
    +
docs/frontend/CHANGELOG.md

Backend change
    ↓
docs/backend/**/*.md
    +
docs/backend/CHANGELOG.md

API change
    ↓
contracts/openapi.yaml
    +
docs/backend/**/*.md

Database change
    ↓
backend/migrations/*
    +
docs/database.md
    +
docs/backend/database or migrations/*.md

Architecture change
    ↓
docs/adr/ADR-XXXX-*.md

Reusable project prompt
    ↓
docs/prompts/**/*.md
```

Documentation and source code are part of the same deliverable.

A code-only implementation is incomplete.

---

# 68. Final Codex Instruction

Start by inspecting the repository.

Read `/docs` before making relevant changes.

Then:

1. report repository state
2. report documentation read
3. propose implementation plan
4. implement the smallest correct solution
5. run relevant validation
6. update required documentation
7. update changelogs
8. update OpenAPI/database docs/ADRs when applicable
9. summarize implementation and documentation
10. identify next steps

Do not blindly generate a large codebase.

Do not over-engineer.

Do not fabricate tests, metrics, APIs, data, or documentation.

Prefer a simple, maintainable implementation that the two-person Adicara team can understand and maintain long-term.
