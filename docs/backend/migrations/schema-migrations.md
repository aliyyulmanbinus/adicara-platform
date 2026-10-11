# Skema Database & Migrasi (Goose)

Manajemen skema database menggunakan `goose`, sebuah pustaka migrasi untuk Golang yang berbasis pada instruksi SQL *up* dan *down*.

## Cara Kerja
Semua file `.sql` disimpan dalam `internal/migrate/sql/`. File-file tersebut di-*embed* langsung ke dalam *binary* aplikasi menggunakan `//go:embed`. Jika variabel `MIGRATE_ON_START=true` diatur dalam file `.env`, sistem akan secara otomatis mengeksekusi file SQL migrasi yang belum berjalan (*Up*) ketika *backend* pertama kali dinyalakan.

## Skema Saat Ini
1. **`001_000_m_role.sql`**: Membuat tabel referensi statis `m_role` (memiliki nilai bawaan `1: admin`, `2: customer`).
2. **`002_000_users.sql`**: Tabel utama untuk menyimpan identitas pengguna, memiliki relasi referensial ke `m_role`.
3. **`003_000_refresh_sessions.sql`**: Tabel penyimpan token sesi (*refresh token*), memiliki relasi *Foreign Key* (dengan set hapus *cascade*) ke tabel `users`.
4. **`004_000_auth_hardening.sql`**: Menambah `users.password_changed_at` dan `refresh_sessions.rotated_at` (keduanya nullable; kompatibel dengan image sebelumnya). Detail: [2026-10-04-auth-hardening-columns.md](2026-10-04-auth-hardening-columns.md).

## Menjalankan Migrasi Manual
Selain otomatis lewat `MIGRATE_ON_START`, tersedia perintah `cmd/migrate` (juga ada di image Docker sebagai `/adicara-migrate`):

| Perintah | Fungsi |
|---|---|
| `make migrate-up` / `go run ./cmd/migrate up` | Menjalankan migrasi yang belum ter-apply. Aman diulang. |
| `make migrate-fresh` / `go run ./cmd/migrate fresh -yes` | **Destruktif.** `DROP SCHEMA public CASCADE`, membuat ulang, lalu menjalankan semua migrasi. Tanpa `-yes` perintah menolak berjalan. |

Target database dibaca dari `.env` (atau `DATABASE_URL`) dan ditampilkan (tanpa kredensial) sebelum dihapus.

## Database dengan Skema Lama

Sebelum rewrite ke goose, backend memakai `golang-migrate` dengan tabel `users` (kolom `display_name`), `sessions`, `invitations`, `guests`, `rsvps`, dst. Tabel-tabel itu tidak dipakai backend baru, tetapi `users` bentrok dengan migrasi `002_000_users.sql`.

**Backend menanganinya sendiri saat start (`migrate up`)**: bila `public.users` masih punya kolom `display_name`, seluruh tabel di `public` **dipindahkan** (bukan dihapus) ke schema baru `legacy_<waktu UTC>`, lalu migrasi berjalan seperti biasa pada schema `public` yang bersih. Dalam satu transaksi, dan hanya sekali: start berikutnya tidak menemukan `display_name` lagi sehingga tidak mengarsip ulang. Log-nya:

```text
migrate: found the pre-goose schema; moved 7 tables to schema "legacy_20261004_153814" (data kept, run DROP SCHEMA legacy_20261004_153814 CASCADE when no longer needed)
```

Konsekuensinya:

- **Tidak ada data yang hilang dan deploy tidak gagal.** Akun lama tidak ikut pindah ke tabel `users` yang baru (skemanya berbeda: `display_name` vs `username`), jadi pengguna lama harus mendaftar ulang. Data lamanya tetap bisa dibaca, mis. `SELECT * FROM legacy_20261004_153814.users`.
- Arsip itu memuat **hash password dan data pribadi** pengguna lama. Hapus bila tidak diperlukan: `DROP SCHEMA legacy_<waktu> CASCADE;`.
- Kondisi setengah termigrasi juga ditangani: versi sebelumnya yang gagal start meninggalkan `m_role` dan `goose_db_version` di `public`; keduanya ikut diarsipkan.
- Aplikasi versi lama tidak bisa berjalan lagi di database itu (tabelnya sudah dipindahkan). Kembali ke versi lama berarti memulihkan dari backup ([backup-restore](../../devops/backup-restore.md)) atau memindahkan tabel kembali dengan `ALTER TABLE legacy_<waktu>.<tabel> SET SCHEMA public`.

`migrate fresh -yes` berbeda: ia **menghapus** semuanya tanpa arsip, dan hanya untuk database yang tidak perlu dipertahankan.

### Pengujian

Perilaku di atas diuji terhadap PostgreSQL 16 dengan skema lama asli (dari riwayat git) berisi data: tes otomatis di `internal/migrate/migrate_test.go` dan start API sungguhan dua kali berturut-turut. Tes itu membutuhkan database dan **menghapus** schema `public`-nya, jadi hanya berjalan bila `TEST_DATABASE_URL` menunjuk database yang namanya berakhiran `_test` (selain itu di-skip; CI tidak memiliki database):

```bash
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_migrate_test?sslmode=disable' go test ./internal/migrate/
```
