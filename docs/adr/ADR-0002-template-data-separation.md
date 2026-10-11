# ADR-0002: Separate Invitation Data from Template Presentation

## Status

Accepted

## Date

2026-10-02

## Context

Adicara must eventually support many themes while users retain names, schedules, venues, guests, RSVP, and wishes when switching designs.

## Decision

Store event data in normalized backend entities. Map API responses into a shared frontend `InvitationData` contract. Register templates by stable key and compose each from shared section components ordered by configuration.

## Alternatives Considered

- Store each invitation as template-specific JSON: rejected because switching themes would be fragile and querying core data would be difficult.
- Duplicate all section code per theme: rejected because RSVP, schedule, map, and accessibility behavior would drift.
- Freeform canvas schema in V1: rejected because it is too complex for the MVP and small team.

## Consequences

New themes can focus on CSS and composition. Shared section changes affect all compatible themes, so component contracts require care. Theme-specific capabilities must be declared explicitly in registry metadata.

## Update (2026-10-04)

The backend entities this decision refers to (invitations, hosts, events, guests, RSVPs) currently do not exist because the invitation module was removed from the backend. The decision stands for the rebuilt module; the frontend `InvitationData` contract and template registry are unchanged.

