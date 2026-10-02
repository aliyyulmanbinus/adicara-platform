# Guests and RSVP Schema

Date: 2026-10-02

Migration `000003_create_guests_and_rsvps` adds normalized guest and RSVP tables with invitation foreign keys and cascade deletion. Guest public tokens are unique. RSVP status and attendee counts use database checks, and `guest_id` is unique in `rsvps` so repeated submissions update the same response.

Indexes cover invitation guest lists, invitation/status aggregates, public token lookup, and invitation RSVP reporting. Runtime migration validation remains pending because Docker and PostgreSQL executables are unavailable in the current environment.
