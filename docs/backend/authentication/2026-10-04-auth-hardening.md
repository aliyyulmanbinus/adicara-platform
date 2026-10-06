# Pengerasan Autentikasi

## Date

2026-10-04

## Status

Updated

## Module

Authentication (dan konfigurasi start backend)

## Summary

Menutup empat batasan auth yang sebelumnya tercatat di [session-authentication.md](session-authentication.md): `JWT_SECRET` bawaan tidak divalidasi, waktu respons login membocorkan identifier yang terdaftar, replay refresh token tidak mencabut sesi lain, dan access token tidak memeriksa database.

## Objective

Produksi tidak boleh berjalan dengan secret yang bisa ditebak, dan akun yang dinonaktifkan atau password yang diganti harus berlaku seketika, bukan setelah access token kedaluwarsa.

## Previous Behavior

- `JWT_SECRET` bawaan `change-me-in-dev`; Compose tidak meneruskan variabelnya, jadi produksi memakai nilai itu.
- Identifier tidak terdaftar ditolak langsung (tanpa bcrypt, ±1 ms); password salah memakan ±250 ms.
- Refresh token yang sudah dirotasi hanya ditolak `401`.
- `BearerAuth` hanya memverifikasi tanda tangan; akun `inactive` dan password yang baru diganti tetap bisa memakai access token sampai kedaluwarsa. `Refresh` tidak memeriksa status akun, jadi akun nonaktif bisa memperbarui sesi tanpa batas.

## New Behavior

### 1. Validasi `JWT_SECRET` saat start

`config.Load` menolak secret lemah bila `APP_ENV` **bukan** nilai development (`development`, `dev`, `local`, `test`, atau kosong). Nilai lain (`production`, `staging`, dan salah ketik seperti `prod`) diperlakukan sebagai produksi, supaya salah ketik gagal tertutup. Ditolak: kosong, sama dengan bawaan, mengandung `change-me`, berawalan `your-secret` (placeholder di `.env.example`), atau kurang dari 32 karakter. Backend keluar dengan kode 1 dan pesan yang tidak memuat nilai secret:

```text
JWT_SECRET (APP_ENV=production): is unset or still a placeholder; set a random value, e.g. `openssl rand -hex 32`
```

Pada development secret bawaan tetap diterima agar `go run ./cmd/api` jalan di checkout baru.

### 2. Waktu respons login

Bila identifier tidak ditemukan, `Login` membandingkan password dengan hash bcrypt tiruan berbiaya sama (cost 12) sebelum menjawab `401`. Hash dibuat sekali saat `usecase.New`.

### 3. Deteksi replay refresh token

Migrasi 004 menambah `refresh_sessions.rotated_at`, diisi hanya saat sesi dihabiskan oleh **rotasi** (bukan logout atau ganti password). Bila refresh token ditolak (sesi sudah dihabiskan), `Refresh` memanggil `RevokeUserSessionsOnReuse`:

| Keadaan sesi yang dikirim | Hasil |
|---|---|
| Dirotasi ≤ 10 detik lalu | `401`, tidak ada yang dicabut (hampir pasti dua request klien yang berpacu). |
| Dirotasi > 10 detik lalu | `401` **dan semua** refresh session pengguna dicabut. Pemilik harus login ulang. |
| Dicabut lewat logout / ganti password, kedaluwarsa, atau tidak dikenal | `401`, tidak ada yang dicabut. |

Perbandingan 10 detik memakai jam database di sisi SQL (`rotated_at` juga dari `now()`), sehingga tidak bergantung pada selisih jam aplikasi dan database. Angka 10 detik sengaja sama dengan jendela penggabungan refresh di frontend (`REFRESH_REUSE_MS`).

### 4. Access token diperiksa ke database

`BearerAuth` kini memanggil `Service.Authenticate` (menggantikan `ParseAccess`): setelah tanda tangan valid, satu query `SELECT status, password_changed_at FROM users WHERE id=$1`. Ditolak dengan `401` bila akun tidak ada, tidak `active`, atau token terbit **sebelum** `password_changed_at` (`iat` dibandingkan per detik; token yang terbit di detik yang sama dengan perubahan diterima agar pengguna yang langsung login ulang tidak terkunci). `ChangePassword` mengisi `password_changed_at` dengan jam aplikasi, jam yang sama yang mengisi `iat`.

`Refresh` juga memeriksa status akun setelah menghabiskan sesi.

**Kenapa `401`, bukan `403 account_inactive`, untuk akun nonaktif di rute terlindungi dan refresh.** Frontend memperlakukan `401` sebagai "masuk lagi" (hapus cookie, arahkan ke login), sedangkan status lain dari `/v1/me` atau refresh dianggap backend bermasalah (`503`, cookie dipertahankan). Dengan `403`, pengguna nonaktif akan terjebak di halaman `503`. Penjelasan `account_inactive` tetap muncul di form login (`POST /v1/auth/login` → `403`).

## API Changes

Tidak ada endpoint, request, atau kode error baru. Perubahan perilaku (sudah tercermin di `backend/docs/openapi.yaml`):

| Endpoint | Perubahan |
|---|---|
| `POST /v1/auth/refresh` | `401` juga untuk akun nonaktif/terhapus; replay lewat 10 detik mencabut semua sesi pengguna. |
| `GET/PATCH /v1/me`, `POST /v1/auth/password` | `401` juga untuk akun nonaktif/terhapus dan token yang terbit sebelum ganti password. |
| `POST /v1/auth/login` | Waktu respons identifier tidak dikenal kini sama dengan password salah. |

## Database Changes

Migrasi `backend/internal/migrate/sql/004_000_auth_hardening.sql`:

- `users.password_changed_at TIMESTAMPTZ NULL`
- `refresh_sessions.rotated_at TIMESTAMPTZ NULL`

Tanpa indeks baru: kueri memakai PK `users.id`, PK `refresh_sessions.id`, dan indeks `refresh_sessions_user_idx (user_id, revoked_at)` yang sudah ada. Detail: [migrasi](../migrations/2026-10-04-auth-hardening-columns.md).

## Migration

`004_000_auth_hardening.sql`: **kompatibel ke belakang** (kolom nullable yang tidak dibaca image sebelumnya, jadi rollback image aman), **tidak destruktif**, **reversible** (`Down` menghapus kedua kolom). Baris yang sudah ada tetap `NULL`: sesi lama tidak memicu deteksi replay dan pengguna lama tidak punya batas waktu token.

## Configuration Changes

| Variabel | Perubahan |
|---|---|
| `APP_ENV` | **Sekarang dibaca backend.** Nilai selain development mengaktifkan validasi `JWT_SECRET`. Default `development`. |
| `JWT_SECRET` | **Wajib** (acak, ≥ 32 karakter) bila `APP_ENV` bukan development. Compose produksi menolak start tanpa nilai ini (`${JWT_SECRET:?...}`). |

Variabel opsional (`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `AUTH_RATE_LIMIT`, `TRUSTED_PROXIES`, `MIGRATE_ON_START`, `CORS_ORIGINS`) kini diteruskan Compose; nilai kosong berarti default backend. Lihat [deployment](../../deployment.md#environment).

**Dampak pada produksi yang sedang berjalan:** `/opt/adicara/.env` harus diisi `JWT_SECRET` **sebelum** deploy berikutnya, dan mengganti secret membuat semua token yang beredar tidak berlaku (semua pengguna login ulang satu kali).

## Security Considerations

- Secret tidak pernah ada di pesan error atau log.
- Deteksi replay hanya aktif untuk sesi yang dirotasi, sehingga logout di satu perangkat tidak menjatuhkan perangkat lain; jendela 10 detik mencegah dua tab/instance frontend yang berpacu mengeluarkan seluruh pengguna.
- Respons `401` seragam untuk semua penolakan token, tanpa membedakan akun nonaktif dari token buruk.
- Pencabutan semua sesi saat replay memakai `user_id` dari refresh token yang tanda tangannya valid **dan** harus cocok dengan pemilik sesi (`WHERE id=$1 AND user_id=$2`), jadi satu pengguna tidak bisa mencabut sesi pengguna lain.
- Sisa batasan: lihat [Batasan yang Diketahui](session-authentication.md#batasan-yang-diketahui).

## Performance Considerations

Setiap permintaan terlindungi menambah **satu query PK** (`users`). Untuk `GET /v1/me` itu berarti dua query (middleware + handler). Pada skala saat ini (satu VPS) dapat diabaikan. Diukur hanya di lokal (macOS, API dan PostgreSQL 16.2 di mesin yang sama, bukan produksi): 600 request `GET /v1/me` dengan token valid, 25 paralel, semuanya `200`, ±460 req/s, latensi p50 1,1 ms, p95 1,7 ms, maks 3,2 ms. Bila kelak menjadi beban, cache singkat per-proses atas `AuthState` adalah langkah berikutnya, dengan konsekuensi jendela pencabutan sebesar TTL cache. Jalur gagal refresh menambah satu `UPDATE` tambahan; login identifier tidak dikenal kini memakan ±250 ms CPU (disengaja), dan tetap dibatasi rate limiter.

## Files Changed

- `backend/internal/config/config.go`, `config_test.go`
- `backend/internal/migrate/sql/004_000_auth_hardening.sql`, `migrate_test.go`
- `backend/internal/modules/auth/domain/{entity,repository}.go`
- `backend/internal/modules/auth/repository/postgres.go`, `postgres_test.go`
- `backend/internal/modules/auth/usecase/service.go`, `hardening_test.go`
- `backend/internal/modules/auth/delivery/http/handler.go`
- `backend/docs/openapi.yaml`, `backend/.env.example`

## Tests

Dijalankan 2026-10-04 (macOS, Go 1.27.1, PostgreSQL 16.2 lokal):

- `gofmt -l .` (bersih), `go vet ./...`, `go build ./cmd/api ./cmd/migrate`.
- `go test ./...` tanpa database: lolos (tes integrasi di-skip, seperti CI).
- `TEST_DATABASE_URL=…_test go test -p 1 ./...` dengan database sementara: lolos, termasuk tes SQL repository, tes migrasi dengan data pada versi 3, rollback `Down`, dan re-apply.
- Uji mutasi: mematikan masing-masing perbaikan (perbandingan bcrypt tiruan, pemanggilan deteksi replay, pemeriksaan status/`iat`, kondisi `rotated_at` pada SQL) membuat tes yang bersangkutan gagal.
- Uji end-to-end `curl` terhadap API produksi-mode dan database nyata: start ditolak untuk secret kosong/bawaan/placeholder/pendek/`APP_ENV=prod`, diterima untuk secret kuat dan untuk development; refresh dalam jendela 10 detik tidak mencabut apa pun; replay setelah 11 detik mencabut sesi hidup; replay token logout tidak mencabut perangkat lain; token lama `401` seketika setelah ganti password sementara login baru di detik yang sama `200`; akun `inactive` mendapat `401` di `/v1/me` dan refresh dan `403 account_inactive` di login; waktu login identifier tidak dikenal 0,254 s vs password salah 0,254 s. Sebagai pembanding, build tanpa perbaikan (dibuat sementara dari kode yang sama dengan perbandingan bcrypt tiruan dimatikan) menjawab identifier tidak dikenal dalam 0,0008–0,0017 s vs 0,254–0,261 s untuk password salah.

- **Deploy pertama dari skema lama (simulasi produksi):** database sementara dengan migrasi lama asli dari riwayat git (`000001`–`000003`) plus tabel `schema_migrations` golang-migrate dan satu pengguna lama, lalu backend baru dijalankan dengan `APP_ENV=production`. Log mencatat 8 tabel dipindahkan ke schema `legacy_…` (data pengguna lama utuh), migrasi 1–4 berjalan, dan API hidup. Start kedua: `no migrations to run, current version: 4` dan tidak mengarsip ulang.
- **Upgrade database dev:** `pg_dump` salinan database dev (versi 3, 1 pengguna, 2 sesi) dijalankan dengan kode baru dan `.env` dev: hanya migrasi 004 yang diterapkan, baris pengguna identik (sidik jari md5 sama), kolom baru `NULL`, pendaftaran dan login baru berhasil.
- **Refresh serentak:** 6 refresh dengan token yang sama dikirim bersamaan: 1× `200` dan 5× `401`; token pemenang tetap bisa di-refresh sesudahnya (tidak ada pencabutan massal di dalam jendela 10 detik).
- Build `linux/amd64` dan `linux/arm64` dengan `CGO_ENABLED=0` (flag yang sama dengan Dockerfile) sukses.

**Tidak diverifikasi:** perilaku di browser dengan semantik `401` baru (alur frontend dibaca dari kode, tidak dijalankan), dan beban/latensi akibat query tambahan di produksi (hanya diukur di lokal, lihat Performance Considerations).

## Breaking Changes

Untuk operator: deploy produksi **gagal sejak langkah `compose config`** bila `JWT_SECRET` belum ada di `/opt/adicara/.env` (tidak ada yang diubah; situs tetap melayani versi lama), dan backend menolak start bila nilainya lemah. Untuk klien API: tidak ada perubahan bentuk; hanya lebih banyak kasus `401`.

## Rollback Notes

Image backend lama berjalan di atas skema 004 tanpa masalah (kolom tambahan diabaikan). Rollback juga mengembalikan perilaku lama, termasuk secret bawaan bila `JWT_SECRET` tidak diteruskan. Menghapus kolom (`goose down`) aman dan tidak menyentuh baris.

## Related Documentation

- [Session & JWT Authentication](session-authentication.md)
- [Migrasi 004](../migrations/2026-10-04-auth-hardening-columns.md)
- [ADR-0003: access token diperiksa ke database](../../adr/ADR-0003-access-token-database-check.md)
- [OpenAPI](../../../backend/docs/openapi.yaml)
- [Database](../../database.md)
- [Sesi autentikasi di frontend](../../frontend/authentication/2026-10-04-session-cookies.md)
