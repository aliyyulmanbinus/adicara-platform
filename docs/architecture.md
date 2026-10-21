# Arsitektur Adicara

## Ringkasan

Adicara memakai monorepo dengan tiga batas utama:

```text
Browser → Nginx → Astro ──(server-side, /v1/*)──→ Go API → PostgreSQL
                    │
                    └ halaman marketing statis, /auth/* dan /dashboard on-demand
```

Nginx menyediakan origin tunggal. Permintaan `/healthz` dan `/api/v1/*` diarahkan ke Go (prefix `/api` dibuang, karena backend melayani `/v1/*`); permintaan lain diarahkan ke Astro. Browser **tidak** memanggil Go API langsung untuk autentikasi: server Astro yang melakukannya, sehingga token tersimpan di cookie HttpOnly.

Rute `/api/v1/*` disediakan untuk klien browser di masa depan (mis. RSVP publik); autentikasi saat ini lewat Astro (`/auth/*`). Detail routing, rate limit, dan resolusi IP klien: [Nginx](devops/nginx.md).

## Frontend

Astro memakai mode static-first. Homepage dan halaman marketing diprerender. Yang dirender on-demand:

- `/auth/login`, `/auth/register`, `/auth/logout` — endpoint POST same-origin yang memanggil Go API dan mengatur cookie sesi.
- `/dashboard/**` — dijaga `src/middleware.ts`, yang memverifikasi sesi dan me-refresh token otomatis.
- `/i/[slug]` — mengambil undangan terbit dari API. Saat ini selalu 404 karena modul undangan belum ada di backend.

Template undangan menerima kontrak `InvitationData`. Registry menghubungkan `template_key` ke komponen presentasi. Bagian umum seperti cover dan jadwal tidak menyimpan data sendiri.

## Backend

Backend memakai Gin dengan pola Module-Based Clean Architecture. Setiap modul berada di `backend/internal/modules/<nama>/`:

```text
delivery/http  →  usecase  →  domain  ←  repository
(handler, DTO,    (aturan     (entity,    (SQL, pgx)
 router)           bisnis)     interface)
```

Handler tidak mengandung SQL dan repository tidak menentukan status HTTP. `domain` mendefinisikan interface repository; `repository` mengimplementasikannya.

| Paket | Peran |
|---|---|
| `cmd/api` | Entrypoint server; flag `-healthcheck` untuk Docker. |
| `cmd/migrate` | `up` dan `fresh -yes` (destruktif). |
| `internal/app` | Composition root: merangkai config, database, modul, dan pembersihan sesi berkala. |
| `internal/config` | Membaca environment (`.env` untuk lokal). |
| `internal/db` | Pool `pgxpool` (ukuran dari `DB_MAX_CONNS`) dan pembawa transaksi lewat `context` (`WithTx`, `Querier`). |
| `internal/migrate` | goose; file SQL di-embed ke binary. |
| `internal/ginserver` | Engine Gin, CORS, dan proxy tepercaya. |
| `internal/apierr`, `internal/ginutil` | Error API bertipe dan serialisasinya ke JSON. |
| `internal/modules/{auth,profile,health}` | Modul fungsional. |
| `internal/modules/planner` | Planner menu 01: proyek, master sesi, registry jenis acara, dashboard. Modul Planner lain mendaftar ke dalamnya lewat `SummaryProvider`, `TemplateApplier`, dan `SampleOverlay`. [Dokumen](backend/planner/2026-10-21-planner-menu-01.md). |
| `internal/modules/plannertimeline` | Planner menu 02: tugas/jadwal persiapan dan dokumen legalitas (penskalaan jadwal dari template, ekspor `.xlsx`, sinkron biaya ke anggaran). Vendor, tamu, dan rundown belum dibangun. [Dokumen](backend/planner/2026-10-21-planner-menu-02.md). |
| `internal/modules/plannerfinance` | Planner menu 03: skenario anggaran (Smart Adjust), kontributor dan setoran tabungan, pos anggaran dengan pembayaran per tahap, matriks, arus kas, template pos, dan ekspor `.xlsx`. Semua angka anggaran dibaca dari view `planner_budget_lines`. [Dokumen](backend/planner/2026-10-21-planner-menu-03.md). |

Modul `profile` memakai usecase `auth` untuk membaca dan mengubah pengguna.

## Batas fase

Tersedia: autentikasi (register, login, refresh, logout, ganti password), profil (`/v1/me`), health check, dan Planner menu 01–03 (proyek, sesi, dashboard, tugas, dokumen legalitas, skenario anggaran, tabungan, realisasi pembayaran; widget vendor, tamu, dan rundown serta template-nya terisi saat menu 04–06 dibangun). Modul undangan, tamu, dan RSVP dihapus dari backend dan menunggu dibangun ulang. Wishes, upload media, hadiah, editor lanjutan, reporting, dan pembayaran belum tersedia.

## Keputusan

- [ADR-0001: monorepo Astro, Go, PostgreSQL dengan same-origin](adr/ADR-0001-foundation-architecture.md)
- [ADR-0002: pemisahan data dan presentasi template](adr/ADR-0002-template-data-separation.md)
