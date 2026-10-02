# Shared Invitation Section Renderer

Date: 2026-10-02

`frontend/src/components/invitation/shared/SectionRenderer.astro` is the single mapping between section IDs and shared Astro components. It:

1. ignores disabled sections;
2. ignores unsupported IDs defensively;
3. sorts enabled sections by their numeric `order`;
4. renders each section with the same invitation data.

Theme wrappers provide only a theme root class and CSS import. They must not copy RSVP, guest, event, or content logic.
