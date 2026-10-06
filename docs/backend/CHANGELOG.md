# Changelog Backend

Semua perubahan arsitektural dan fitur signifikan pada backend dicatat di sini.

## 2026-10-04 (pengerasan auth dan produksi)
- **Security**: `JWT_SECRET` divalidasi saat start. Bila `APP_ENV` bukan development, backend menolak secret kosong, bawaan, placeholder, atau kurang dari 32 karakter. `APP_ENV` kini dibaca backend. Documentation: `authentication/2026-10-04-auth-hardening.md`.
- **Security**: Waktu respons login tidak lagi membedakan identifier yang tidak terdaftar (bcrypt tiruan; sebelumnya ±1 ms vs ±250 ms).
- **Security**: Replay refresh token yang sudah dirotasi lebih dari 10 detik mencabut semua sesi pengguna (kolom `refresh_sessions.rotated_at`); dalam 10 detik hanya `401`. Logout dan ganti password tidak memicunya.
- **Security**: `BearerAuth` kini memeriksa database (akun aktif, token tidak terbit sebelum `users.password_changed_at`); `Refresh` memeriksa status akun. Akun nonaktif dan ganti password berlaku seketika; semua penolakan `401`. `ParseAccess` diganti `Authenticate`. [ADR-0003](../adr/ADR-0003-access-token-database-check.md).
- **Migration**: `004_000_auth_hardening.sql` (dua kolom nullable, kompatibel dengan image sebelumnya). Documentation: `migrations/2026-10-04-auth-hardening-columns.md`.
- **Fix (deploy)**: Compose (prod dan lokal) kini meneruskan env backend (`JWT_SECRET`, TTL, `AUTH_RATE_LIMIT`, `TRUSTED_PROXIES`, `MIGRATE_ON_START`, `CORS_ORIGINS`); sebelumnya produksi berjalan dengan `JWT_SECRET=change-me-in-dev`. Compose produksi menolak start tanpa `JWT_SECRET`. **Aksi operator:** isi `JWT_SECRET` di `/opt/adicara/.env` sebelum deploy berikutnya ([deployment](../deployment.md#batasan-yang-diketahui)).
- **Fix (deploy)**: Nginx: `/api/v1/*` kini sampai ke backend (prefix `/api` dibuang; sebelumnya `404`); limiter `auth_attempts` dipasang pada `/auth/{login,register}` yang dipakai form web (sebelumnya hanya pada path `/api/v1/auth/...` yang tidak dipakai); `real_ip` dari `X-Forwarded-For` bila peer privat/loopback agar rate limit tidak jatuh ke satu bucket di belakang proxy host; penolakan `429` memakai body error API. [Nginx](../devops/nginx.md).
- **Docs**: Master prompt (`docs/prompts/master/adicara-master-codex-prompt.md`) diselaraskan dengan kode: arsitektur modul Gin, kontrak `backend/docs/openapi.yaml` (OpenAPI 3.0.3), `/v1` vs `/api/v1` di Nginx, autentikasi JWT dengan cookie di sisi Astro, hanya `/healthz`, migrasi goose di `backend/internal/migrate/sql/`, target Makefile, dan struktur docs backend. Bagian modul yang belum ada ditandai planned.
- **Docs**: Kontrak yang berlaku adalah `backend/docs/openapi.yaml` (CI me-lint file ini, bukan lagi `contracts/openapi.yaml`, yang menjadi backup). OpenAPI diperbarui: server `/api`, semantik `401`, deteksi replay.

## 2026-10-04 (lanjutan)
- **Fix**: Registrasi tidak lagi menyisipkan baris ke `subscriptions`/`m_plan` (tabelnya belum ada di migrasi, sehingga setiap `POST /v1/auth/register` gagal 500). Penyediaan langganan ditunda sampai modul billing ada.
- **Fix**: Rate limiter `/v1/auth/*` kini memakai IP klien dari `X-Real-IP` yang dikirim proxy tepercaya (`TRUSTED_PROXIES`), bukan IP koneksi. Sebelumnya, di belakang nginx semua pengguna berbagi satu bucket. `X-Forwarded-For` sengaja diabaikan karena bisa dipalsukan klien.
- **Fix**: Refresh session yang kedaluwarsa kini dibersihkan otomatis (saat start, lalu tiap jam). Sebelumnya `PurgeExpiredSessions` tidak pernah dipanggil.
- **Chore**: Menghapus paket tak terpakai `internal/httputil` dan `internal/authctx`; memformat ulang `internal/config/config.go` (CI mewajibkan `gofmt` bersih).
- **Fix**: Validasi input auth dilengkapi: format email (`invalid_email`, maks 254), panjang username 3–30, dan panjang password maks 72 byte (`password_too_long`). Sebelumnya password 73+ byte menghasilkan error mentah bcrypt → HTTP 500, dan email apa pun diterima.
- **Fix**: Flag `-healthcheck` pada `cmd/api` dipulihkan (dipakai healthcheck Docker; tanpa itu container backend tidak pernah healthy).
- **Chore**: Service `migrate` (golang-migrate, folder `backend/migrations` yang sudah dihapus) dihapus dari Compose dev/prod; migrasi dijalankan oleh API saat start.
- **Docs**: `docs/openapi.yaml` diselaraskan dengan perilaku sebenarnya (kode status logout/password 204, body refresh/logout, daftar kode error, aturan validasi).
- **Docs**: Dokumentasi diselaraskan dengan kode: `session-authentication.md` (refresh token adalah JWT yang dicabut, bukan dihapus; ganti password; batasan yang diketahui), `server-health.md` (bentuk respons `/healthz` yang sebenarnya), `user-profile.md`, `docs/api.md`, `docs/architecture.md`, `docs/database.md`, serta dokumen deployment/DevOps (tidak ada service migrasi; `/healthz` menggantikan `/readyz`).
- **Fix**: `COOKIE_SECURE` di Compose dipindahkan ke service **frontend** (yang menulis cookie sesi); sebelumnya terpasang di backend yang tidak membacanya.
- **Feature**: Start di atas database yang masih berskema lama (pra-goose) tidak lagi gagal: tabel lama dipindahkan ke schema `legacy_<waktu>` (data tetap ada) sebelum migrasi. Sebelumnya API keluar dengan kode 1 (`relation "users" already exists`) dan restart terus. Diuji dengan tes otomatis (`TEST_DATABASE_URL`, nama DB berakhiran `_test`) dan start API sungguhan di atas skema lama berisi data.
- **Fix (deploy)**: Auto-update saat push ke `main` diperbaiki: health check memakai `/healthz` (`/readyz` tidak ada di backend), checkout VPS disamakan dengan commit yang di-deploy, hanya image aplikasi yang ditarik, config Nginx diterapkan ulang, `nginx -t` sebelum reload, dan bila gagal log dicetak serta versi baik sebelumnya dipulihkan. Lokasi `/readyz` dihapus dari Nginx. Detail: `docs/devops/github-actions.md`.
- **Feature**: Perintah `cmd/migrate` (`up`, `fresh -yes`) untuk menjalankan migrasi di luar proses API dan mereset database yang masih berisi skema lama. Lihat [schema-migrations.md](migrations/schema-migrations.md).

## 2026-10-04
- **Refactor**: Menghapus seluruh modul usang (`billing`, `catalog`, `gift`, `invitations`, `planner`, dsb) dari *codebase* utama.
- **Refactor**: Menyederhanakan struktur arsitektur menggunakan *Module-Based Clean Architecture* yang diadaptasi penuh dari *project* `ketuk`.
- **Feature**: Menerapkan modul inti secara murni: `auth`, `profile`, dan `health`.
- **Feature**: Mengintegrasikan migrasi database otomatis menggunakan `goose` ke dalam siklus *startup* aplikasi.
- **Docs**: Menulis ulang dokumentasi API menjadi format OpenAPI (Swagger 3.0.3).
- **Docs**: Merekonstruksi dokumentasi arsitektur internal backend yang lebih merinci per modul.
