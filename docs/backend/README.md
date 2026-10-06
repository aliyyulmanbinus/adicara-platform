# Dokumentasi Backend

Backend dibangun menggunakan Golang dengan pola **Module-Based Clean Architecture**.

## Implementasi

- [Health Check API](health/server-health.md)
- [Skema Database & Migrasi (Goose)](migrations/schema-migrations.md)
- [Migrasi 004: kolom pengerasan autentikasi](migrations/2026-10-04-auth-hardening-columns.md)
- [Session & JWT Authentication](authentication/session-authentication.md)
- [Pengerasan autentikasi (2026-10-04)](authentication/2026-10-04-auth-hardening.md)
- [Manajemen Profil User](profile/user-profile.md)
- [Changelog](CHANGELOG.md)

## Aturan

Setiap penambahan modul atau perubahan logika yang bermakna wajib didokumentasikan di dalam sub-folder modul yang bersangkutan dan dicatat pada file `CHANGELOG.md`.

Dokumentasi *technical contract* (seperti `openapi.yaml` atau `API.md`) berada di folder `/backend/docs/`.
