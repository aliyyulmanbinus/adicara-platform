# Dokumentasi Backend

Backend dibangun menggunakan Golang dengan pola **Module-Based Clean Architecture**.

## Implementasi

- [Health Check API](health/server-health.md)
- [Skema Database & Migrasi (Goose)](migrations/schema-migrations.md)
- [Migrasi 004: kolom pengerasan autentikasi](migrations/2026-10-04-auth-hardening-columns.md)
- [Session & JWT Authentication](authentication/session-authentication.md)
- [Pengerasan autentikasi (2026-10-04)](authentication/2026-10-04-auth-hardening.md)
- [Manajemen Profil User](profile/user-profile.md)
- [Planner menu 01: proyek, sesi, registry, dashboard](planner/2026-10-21-planner-menu-01.md)
- [Planner menu 02: timeline, tugas, dokumen legalitas](planner/2026-10-21-planner-menu-02.md)
- [Planner menu 03: keuangan dan anggaran](planner/2026-10-21-planner-menu-03.md)
- [Planner menu 04: vendor dan belanja kebutuhan](planner/2026-10-23-planner-menu-04.md)
- [Migrasi 005–012: skema Planner](migrations/2026-10-21-planner-migrations-005-012.md)
- [Changelog](CHANGELOG.md)

## Aturan

Setiap penambahan modul atau perubahan logika yang bermakna wajib didokumentasikan di dalam sub-folder modul yang bersangkutan dan dicatat pada file `CHANGELOG.md`.

Dokumentasi *technical contract* (seperti `openapi.yaml` atau `API.md`) berada di folder `/backend/docs/`.
