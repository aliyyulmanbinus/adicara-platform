# Database

## Teknologi

PostgreSQL 18 digunakan di Compose. Skema dikelola dengan migrasi SQL eksplisit (goose) di `backend/internal/migrate/sql`. File di-embed ke binary API dan dijalankan saat start jika `MIGRATE_ON_START=true` (default). Tidak ada service migrasi terpisah di Compose.

> Migrasi dan alur auth diuji terhadap PostgreSQL 16 (lokal). Image PostgreSQL 18 di Compose belum diuji.

## Skema saat ini

### `m_role`

Referensi role. Baris bawaan: `1 = admin`, `2 = customer`. Kolom: `id` (smallint, PK), `name`, `created_at`, `updated_at`, `deleted_at` (soft delete). Nama unik tanpa memandang huruf besar-kecil di antara baris yang belum dihapus.

### `users`

| Kolom | Catatan |
|---|---|
| `id` | UUID, dibuat oleh aplikasi. PK. |
| `email` | Disimpan huruf kecil. Unik tanpa memandang huruf besar-kecil (`users_email_lower_idx`). |
| `username` | Disimpan huruf kecil. Unik tanpa memandang huruf besar-kecil (`users_username_lower_idx`). |
| `password_hash` | bcrypt cost 12. |
| `name` | Opsional; diubah lewat `PATCH /v1/me`. |
| `role_id` | FK ke `m_role`, default `2` (customer). |
| `status` | `active` atau `inactive` (CHECK), default `active`. |
| `password_changed_at` | `timestamptz`, nullable. Diisi saat password diganti; access token yang terbit sebelumnya ditolak. `NULL` = belum pernah diganti. |
| `created_at`, `updated_at` | `timestamptz`. |

### `refresh_sessions`

| Kolom | Catatan |
|---|---|
| `id` | UUID. Sama dengan klaim `jti` pada refresh token. Token mentah tidak disimpan. |
| `user_id` | FK ke `users`, `ON DELETE CASCADE`. |
| `expires_at` | Batas berlaku sesi. |
| `revoked_at` | Terisi saat sesi dipakai untuk refresh, logout, atau ganti password. |
| `rotated_at` | `timestamptz`, nullable. Terisi **hanya** saat sesi dihabiskan oleh rotasi (refresh); dipakai untuk mendeteksi replay token. |
| `created_at` | `timestamptz`. |

Indeks `refresh_sessions_user_idx` pada `(user_id, revoked_at)`. Baris yang melewati `expires_at` dihapus otomatis saat backend start dan setiap jam.

### Planner (`planner_*`)

Tabel milik menu 01, dipakai kodenya:

| Tabel | Catatan |
|---|---|
| `planner_projects` | Satu baris per acara; satu pengguna banyak proyek (`user_id` → `users`, `ON DELETE CASCADE`). `event_type` (9 nilai, `CHECK`), `title` (1–255, dipangkas, tanpa karakter kontrol), `event_date`, `event_end_date` (nullable), `prep_start_date`, `timezone` (IANA kanonik, dijaga `CHECK` dan pemicu), `target_budget`/`savings_target` (rupiah utuh `BIGINT`, 0…10¹³), `status` (`active`/`completed`/`archived`), `is_sample` (maks satu per pengguna, indeks unik parsial), `client_token` (unik per pengguna), `metadata` (`JSONB`, divalidasi usecase terhadap registry), `invitation_id` (tanpa FK sampai modul undangan ada). |
| `planner_event_sessions` | Master sesi bersama semua modul Planner: `scope_key` unik per proyek, nama unik per proyek tanpa membedakan huruf besar-kecil, satu baris sistem `all` per proyek. |

Tabel modul lain sudah dibuat; skemanya ada di `BE.md` modul masing-masing. Yang sudah punya kode: `planner_tasks` dan `planner_admin_docs` (menu 02), serta `planner_budget_scenarios`, `planner_budget_scenario_items`, `planner_budget_items`, `planner_contributors`, `planner_savings_entries`, dan view `planner_budget_lines` (menu 03; semua angka anggaran dibaca dari view, tidak dari kolom `paid_total`/`remaining_balance`). Menu 04 (`planner_vendors`, `planner_shopping_items`; tidak ada tabel atau migrasi baru): pagu vendor **tidak** disimpan (kolom `budget_limit` hanya pagu manual, `0` = ikut skenario aktif) dan kolom `planner_vendors.price_delta` hanya benar untuk pagu manual sehingga API tidak pernah mengirimnya; `lock-deal` dan gulungan belanja menulis `planner_budget_items` (`vendor_id`, `rollup_source`, baris baru). Menu 05 (`planner_guests`; tidak ada tabel atau migrasi baru): `qr_code_token` (128 bit, hex 32) adalah tiket sekaligus kredensial RSVP tamu; `pax_actual` memegang dua hal (jumlah yang menyatakan hadir lewat RSVP, lalu jumlah di pintu sesudah check-in); `invitation_id` tanpa FK sampai modul undangan ada, dan baris dengan `project_id IS NULL` milik modul undangan saja (tidak pernah dibaca atau ditulis Planner); stok souvenir dibaca dari `planner_shopping_items` (baca-saja). Constraint yang disarankan (`pax_actual BETWEEN 0 AND 20`, `is_checked_in = (checked_in_at IS NOT NULL)`) belum diterapkan: usecase menjaganya. Yang belum: `planner_rundowns`, `planner_playlists`. FK dari tabel-tabel itu ke `planner_event_sessions` bertipe `DEFERRABLE INITIALLY DEFERRED` ([alasan](backend/migrations/2026-10-21-planner-migrations-005-012.md#aturan-yang-dijaga-skema)).

goose menyimpan versi migrasi di tabel `goose_db_version`.

## Migrasi

- `001_000_m_role.sql` — referensi role
- `002_000_users.sql` — akun pengguna
- `003_000_refresh_sessions.sql` — sesi refresh token
- `004_000_auth_hardening.sql` — `users.password_changed_at` dan `refresh_sessions.rotated_at` (nullable; kompatibel dengan image sebelumnya). Detail: [migrasi 004](backend/migrations/2026-10-04-auth-hardening-columns.md)
- `005`–`012` — skema Planner (14 tabel + view). Detail: [migrasi 005–012](backend/migrations/2026-10-21-planner-migrations-005-012.md)

Prosedur manual, termasuk mereset database yang masih berskema lama: [schema-migrations](backend/migrations/schema-migrations.md). Ringkasnya, dari `backend/`: `make migrate-up` dan `make migrate-fresh` (**destruktif**).

Tabel undangan, tamu, dan RSVP dari versi sebelumnya sudah tidak dipakai dan akan kembali lewat migrasi baru saat modulnya dibangun ulang. Pada database yang masih berisi tabel-tabel itu, backend memindahkannya ke schema `legacy_<waktu>` (data tetap ada) sebelum migrasi; lihat [schema-migrations](backend/migrations/schema-migrations.md#database-dengan-skema-lama).

## Data pribadi

Skema saat ini menyimpan email, username, nama opsional, dan hash password, serta judul dan atribut acara di `planner_projects`. Tabel `planner_guests` (nama dan telepon tamu) sudah ada tetapi belum ada kode yang mengisinya. Kebijakan retensi dan penghapusan akun masih perlu ditetapkan sebelum produksi publik. Data tamu dan RSVP, bila modul undangan kembali, hanya boleh tersedia pada endpoint pemilik.
