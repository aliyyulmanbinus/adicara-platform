# Merge branch `backend` ke `development` (2026-10-11)

Merge `origin/backend` (`df59917`) ke `development` (`d008422`, yang sudah memuat desain Batak Senja dari `design-undangan`). Tujuannya: backend baru berjalan di lokal dan produksi bersama isi `main`.

## Aturan penyelesaian

Atas permintaan pemilik: bila ada konflik pada backend, **versi branch `backend` yang dipakai**; bila tidak ada konflik, kedua sisi digabung.

## Konflik dan hasilnya

| File | Konflik | Hasil |
| --- | --- | --- |
| `backend/cmd/api/main.go` | kedua sisi mengubah | versi `backend` (`app.Build`, Gin) |
| `backend/internal/{domain,handler,repository/postgres,service,server}/…` (7 file undangan) | `backend` menghapus, `development` mengubah | dihapus, mengikuti `backend` |
| `README.md`, `docs/architecture.md`, `docs/database.md` | kedua sisi mengubah | versi `backend` (mendeskripsikan backend baru) |
| `frontend/astro.config.mjs` | proxy dev `/api` | versi `backend` (`API_BASE_URL`, prefix `/api` dibuang agar cocok dengan route `/v1`) |

Tidak ada konflik teks pada `frontend/src/pages/dashboard/undangan/baru.astro`; kedua perubahan tergabung (`prerender = false` dari `backend`, form Batak Senja dari `development`).

## File tanpa konflik yang tetap dihapus

Empat file hanya ada di `development` sehingga git menggabungkannya tanpa konflik, tetapi tidak bisa hidup di backend baru:

- `backend/internal/handler/invitation_media.go` dan `backend/internal/repository/postgres/invitation_media_repository.go` mengimpor paket `internal/domain` yang sudah dihapus. Dengan keduanya, `go build ./...` gagal (`no required module provides package …/internal/domain`).
- `backend/migrations/000004_create_invitation_design.{up,down}.sql` mengubah tabel `invitations` yang tidak ada di skema goose baru, dan folder `backend/migrations` sudah dihapus oleh `backend`.

Keempatnya tetap tersedia di riwayat git (`d008422`) untuk dibangun ulang bersama modul undangan.

## Yang tergabung dari `development`

- Template **Batak Senja** (`frontend/src/invitation-templates/template-design-udangan/`, halaman `/template-design-udangan/pernikahan/batak-senja` dan `/tema/batak-senja`, tiga gambar `.webp`).
- Form input data undangan di `/dashboard/undangan/baru` dan klien `lib/api/invitations.ts`.
- `docs/local-development.md` (diperbarui, lihat [Menjalankan secara lokal](../local-development.md)).
- `deploy/nginx/adicara.conf`: `client_max_body_size` 8m (sebelumnya 2m, dinaikkan untuk unggah foto undangan). Endpoint unggah itu sudah tidak ada; pemilik memutuskan batas 8m **dipertahankan** (keputusan 2026-10-11) agar siap saat unggah foto kembali.

## Dampak fungsional

Modul undangan, tamu, RSVP, dan media foto **tidak ada** di backend baru. Akibatnya:

- `/dashboard/undangan/baru` (buat draft, unggah foto) dan halaman publik `/i/[slug]` memanggil `/api/v1/invitations…` dan `/api/v1/public/invitations…` yang tidak ada lagi: API menjawab `404` (`/v1/invitations` diuji), dan `/i/[slug]` menampilkan `503` "Layanan undangan sedang tidak tersedia" (bukan `404`).
- Halaman katalog dan pratinjau tema (termasuk Batak Senja) tetap berfungsi karena statis.
- Auth (daftar, masuk, refresh, keluar, ganti password), `/v1/me`, dan `/healthz` berfungsi.

## Validasi yang dijalankan

Dijalankan di macOS (Go 1.27.1, Node 22.17.1, PostgreSQL 16.2 lokal) setelah resolusi:

| Pemeriksaan | Hasil |
| --- | --- |
| `go build ./...`, `go vet ./...`, `gofmt -l .` | bersih |
| `go test ./...` | lulus semua paket |
| Tes integrasi Postgres (`TEST_DATABASE_URL`, database `*_test`): `internal/migrate`, `internal/modules/auth/repository` | 9 tes lulus dengan `-p 1`; **gagal bila dijalankan paralel** pada satu database karena dua paket saling mereset (bukan cacat merge) |
| Frontend: `npm test`, `astro check`, `npm run build`, `prettier --check .` | 31 tes lulus; check dan build berhasil; format bersih |
| Smoke test binary backend (`APP_ENV=production`, JWT secret acak, database scratch) | `/healthz` 200, migrasi 001–004 otomatis, register 201, login 200, `/v1/me` 200, refresh 200, refresh ulang dengan token lama 401, password salah 401, `/v1/invitations` 404 |
| Frontend (`node dist/server/entry.mjs`, port 18092) + backend (port 18090) berjalan bersama, database scratch, `curl` dengan cookie jar | `/auth/masuk`, `/auth/daftar` 200; `/dashboard` tanpa login 302 ke `/auth/masuk`; register 201; login 200 lalu `/dashboard` 200 dan menampilkan username; `/dashboard/undangan/baru`, `/template-design-udangan/pernikahan/batak-senja`, `/tema/batak-senja` 200; password salah 401; logout 204 lalu `/dashboard` 302. `/i/[slug]` menjawab **503** (API undangan tidak ada). Browser sungguhan tidak dipakai pada uji ini |
| `nginx -t` pada `deploy/nginx/adicara.conf` hasil merge (nginx 1.25.4, host upstream diganti ke 127.0.0.1) | syntax ok |
| Upgrade dari skema `main` penuh (migrasi 000001–000004 dari `d008422`, 8 tabel termasuk `invitation_media`) | start berhasil; 8 tabel dipindahkan ke `legacy_<waktu>` dengan data utuh; `public` berisi `goose_db_version`, `m_role`, `users`, `refresh_sessions`; `/healthz` 200 |

`docker-compose.yml`, `deploy/docker-compose.prod.yml`, `.github/workflows/*`, `backend/Dockerfile`, dan `.env.example` identik dengan branch `backend` (hanya `adicara.conf` yang berbeda satu baris).

## Pemeriksaan Docker (tanpa Docker)

Build dan Compose ditiru secara native; hasilnya di [Docker](docker.md#validasi). Temuan: tidak ada `.dockerignore`, sehingga `COPY . .` mengirim `backend/.env` dan `frontend/node_modules` milik host ke build. `backend/.dockerignore` dan `frontend/.dockerignore` ditambahkan; konteks backend turun dari 36,7 MB ke 0,1 MB dan frontend dari 270,6 MB ke 1,1 MB tanpa membuang file yang dilacak git.

## Hasil deploy produksi (2026-10-11)

Dicatat dari log container dan sesi `psql` yang ditunjukkan pemilik, serta laporan pemilik. Tidak diamati langsung oleh penulis catatan ini.

- Sebelum deploy dijalankan ulang, checkout di VPS sudah di `9d0a4f2` tetapi container masih memakai image lama (`d008422…`, aktif 5–7 hari) dan `/opt/adicara/.env` belum memiliki `JWT_SECRET`. Itu konsisten dengan deploy pertama yang berhenti di `compose config` tanpa mengganti apa pun.
- Setelah `JWT_SECRET` diisi dan deploy dijalankan ulang, log backend menampilkan `migrate: found the pre-goose schema; moved 8 tables to schema "legacy_20261011_132532"`, migrasi 001–004 `OK`, `goose: successfully migrated database to version: 4`, route `/healthz`, `/v1/auth/*`, dan `/v1/me` terdaftar, lalu `adicara-api listening on :8080`. `/healthz` menjawab 200 setiap 10 detik (healthcheck container). Frontend Astro `listening` di port 4321.
- `\dt+` pada schema `public`: `goose_db_version`, `m_role`, `refresh_sessions` (0 bytes: belum ada login saat itu), dan `users`. Tabel lama berada di `legacy_20261011_132532`.
- Pemilik melaporkan daftar dan login berhasil di `https://adicara.id`.
- Backend berjalan dalam mode debug Gin (log: `Running in "debug" mode`); `GIN_MODE=release` belum diset di Compose produksi. Tidak memengaruhi fungsi.

Terkonfirmasi setelah deploy: (1) container nginx memakai konfigurasi terbaru (dilaporkan pemilik; keluaran `diff` tidak dilihat). Risikonya: `git reset --hard` mengganti `deploy/nginx/adicara.conf` dengan inode baru, dan bind mount satu file di Docker dapat tetap melihat file lama. (2) Proxy Sumopod mengirim `X-Forwarded-For`: baris log akses nginx untuk `GET /auth/masuk` menampilkan IP publik pengunjung (bukan alamat proxy atau container) dan nilai `X-Forwarded-For` yang sama; alamatnya sengaja tidak dicatat di sini. Belum diamati: IP klien pada log backend untuk `POST /v1/auth/login`. Backup database sebelum deploy tidak dibuat: pemilik memutuskan tidak perlu karena proyek masih di awal pengembangan (keputusan 2026-10-11).

## Belum diverifikasi

- Docker tidak tersedia secara lokal: `docker compose up` untuk `docker-compose.yml` (dev) belum dijalankan. Image yang dibangun CI dan Compose produksi sudah berjalan di VPS (lihat di atas). `redocly lint` dan job `docker` CI tidak dijalankan lokal.
- Rate limit backend per IP klien di produksi (log backend untuk `POST /v1/auth/login` belum diperiksa).
- Panduan PowerShell di `docs/local-development.md` belum dijalankan di Windows.

## Tindakan operator untuk deploy ini

1. `JWT_SECRET` di `/opt/adicara/.env` di VPS (`openssl rand -hex 32`); tanpanya deploy berhenti di `compose config` dan versi lama tetap melayani ([deployment](../deployment.md#batasan-yang-diketahui)). Sudah diisi sebelum deploy dijalankan ulang.
2. Cadangkan database produksi (tidak dilakukan; keputusan pemilik 2026-10-11, proyek masih awal pengembangan). Deploy pertama memindahkan semua tabel lama (akun, undangan, tamu, RSVP, foto) ke schema `legacy_<waktu>`. Data tetap ada tetapi **tidak dilayani**, akun lama tidak dapat login (pengguna mendaftar ulang), dan rollback image tidak memulihkan skema. [Backup dan restore](backup-restore.md).
3. Pahami bahwa fitur undangan yang kini aktif di `main` akan hilang dari produksi sampai modul undangan dibangun ulang.
