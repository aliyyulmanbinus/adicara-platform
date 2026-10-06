# Migrasi 004: kolom pengerasan autentikasi

## Date

2026-10-04

## Status

Implemented

## Module

Database / Authentication

## Summary

`backend/internal/migrate/sql/004_000_auth_hardening.sql` menambah dua kolom nullable. Alasan dan perilaku yang memakainya ada di [Pengerasan Autentikasi](../authentication/2026-10-04-auth-hardening.md).

## Database Changes

| Tabel | Kolom | Tipe | Diisi oleh |
|---|---|---|---|
| `users` | `password_changed_at` | `TIMESTAMPTZ NULL` | `ChangePassword`, dengan jam aplikasi. Access token yang `iat`-nya lebih awal (per detik) ditolak. |
| `refresh_sessions` | `rotated_at` | `TIMESTAMPTZ NULL` | `ConsumeRefreshSession` (rotasi). Tidak diisi oleh logout atau ganti password, supaya replay token yang dicabut dengan cara itu tidak dianggap pencurian. |

Tanpa indeks atau foreign key baru.

## Migration

- **Kompatibel ke belakang:** kolom nullable tanpa default yang tidak dirujuk kode image sebelumnya (semua `INSERT`/`SELECT` lama menyebut kolomnya secara eksplisit), jadi memundurkan image setelah migrasi aman. Ini memenuhi syarat di [deployment](../../deployment.md#rollback): migrasi harus kompatibel dengan versi aplikasi sebelumnya.
- **Tidak destruktif**, dan `ADD COLUMN` nullable tanpa default tidak menulis ulang tabel.
- **Reversible:** `Down` menghapus kedua kolom; baris tidak tersentuh.
- Baris yang sudah ada tetap `NULL`: pengguna lama tidak punya batas waktu token, dan sesi lama tidak memicu deteksi replay.

## Tests

Dijalankan 2026-10-04 terhadap PostgreSQL 16.2 dengan database sementara berakhiran `_test` (`TEST_DATABASE_URL`):

- `TestAuthHardeningMigrationKeepsExistingData` (`internal/migrate/migrate_test.go`): database pada versi 3 berisi satu pengguna dan dua sesi di-upgrade ke 4; baris utuh dan kolom baru `NULL`; `goose down` menghapus kedua kolom tanpa menghilangkan baris; `Up` ulang berhasil.
- Tes migrasi yang sudah ada (arsip skema lama, pemulihan setengah termigrasi, database kosong, data tetap utuh) lolos dengan versi terbaru 4.
- Start API sungguhan mencatat versi goose 4 dan kedua kolom ada.
- Simulasi deploy pertama produksi dari skema lama asli (migrasi `000001`–`000003` dari riwayat git + `schema_migrations`): skema lama diarsipkan, migrasi 1–4 berjalan, start kedua tidak mengarsip ulang.
- Salinan `pg_dump` database dev pada versi 3 (1 pengguna, 2 sesi) di-upgrade ke 4 oleh kode baru: baris pengguna identik, kolom baru `NULL`.

**Belum diuji:** pada PostgreSQL 18 (image Compose) dan pada database produksi.

## Breaking Changes

None.

## Rollback Notes

Memundurkan image tidak memerlukan `goose down`. Bila kolom ingin dihapus, jalankan `down` satu langkah, tetapi mundurkan image terlebih dahulu: image baru membaca kedua kolom, sehingga login, refresh, dan semua rute terlindungi akan error (`500`) bila kolomnya sudah tidak ada.

## Related Documentation

- [Pengerasan Autentikasi](../authentication/2026-10-04-auth-hardening.md)
- [Skema Database & Migrasi](schema-migrations.md)
- [Database](../../database.md)
