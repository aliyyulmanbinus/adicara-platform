# Guests and RSVP

Date: 2026-10-02

Owners can list, add, replace, and delete guests through `/api/v1/invitations/{id}/guests`. The repository joins every operation through an invitation owned by the authenticated user.

Guest creation generates a 256-bit random Base64URL token. Public links can use this opaque token instead of database IDs. Owner responses include the token so a future guest-management UI can construct `/i/{slug}?guest={token}`; personalized URLs are always rendered `noindex` and are never included in the sitemap.

`POST /api/v1/public/invitations/{slug}/rsvp` accepts that token, validates the published invitation, and performs an idempotent upsert for one RSVP per guest. Attendee counts are limited to 20, absent responses are normalized to zero attendees, and messages are limited to 500 characters. Nginx applies a dedicated public-write rate limit.
