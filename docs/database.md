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

goose menyimpan versi migrasi di tabel `goose_db_version`.

## Migrasi

- `001_000_m_role.sql` — referensi role
- `002_000_users.sql` — akun pengguna
- `003_000_refresh_sessions.sql` — sesi refresh token
- `004_000_auth_hardening.sql` — `users.password_changed_at` dan `refresh_sessions.rotated_at` (nullable; kompatibel dengan image sebelumnya). Detail: [migrasi 004](backend/migrations/2026-10-04-auth-hardening-columns.md)

Prosedur manual, termasuk mereset database yang masih berskema lama: [schema-migrations](backend/migrations/schema-migrations.md). Ringkasnya, dari `backend/`: `make migrate-up` dan `make migrate-fresh` (**destruktif**).

Tabel undangan, tamu, dan RSVP dari versi sebelumnya sudah tidak dipakai dan akan kembali lewat migrasi baru saat modulnya dibangun ulang. Pada database yang masih berisi tabel-tabel itu, backend memindahkannya ke schema `legacy_<waktu>` (data tetap ada) sebelum migrasi; lihat [schema-migrations](backend/migrations/schema-migrations.md#database-dengan-skema-lama).

## Data pribadi

Skema saat ini menyimpan email, username, nama opsional, dan hash password. Tidak ada nomor telepon atau data tamu. Kebijakan retensi dan penghapusan akun masih perlu ditetapkan sebelum produksi publik. Data tamu dan RSVP, bila modul undangan kembali, hanya boleh tersedia pada endpoint pemilik.
