# Adicara

Adicara adalah platform undangan digital dan perayaan yang mobile-first untuk Indonesia. Repositori ini memuat website marketing dan katalog tema Astro, autentikasi serta dashboard awal, Go REST API, PostgreSQL, kontrak OpenAPI, serta fondasi Docker dan CI/CD.

## Prasyarat

- Node.js 22.12 atau lebih baru (Node 24 direkomendasikan)
- npm 10 atau lebih baru
- Go 1.27 atau lebih baru
- Docker Engine + Docker Compose untuk menjalankan seluruh stack

## Menjalankan seluruh stack

1. Salin `.env.example` menjadi `.env` dan ganti password lokal. `JWT_SECRET` boleh dibiarkan untuk pengembangan lokal; untuk produksi wajib diganti dengan `openssl rand -hex 32` (backend menolak start bila tidak).
2. Jalankan:

   ```sh
   docker compose up --build
   ```

3. Buka `http://localhost:8080`.
4. Periksa API melalui `http://localhost:8080/healthz` (memeriksa koneksi PostgreSQL; tidak ada `/readyz`).

PostgreSQL lokal tersedia pada port `5432`. Migrasi dijalankan otomatis oleh API saat start. Jangan gunakan nilai `.env.example` untuk produksi.

> Alur Compose belum diverifikasi pada lingkungan implementasi (Docker tidak tersedia); lihat [docs/devops/docker.md](docs/devops/docker.md).

## Menjalankan tanpa Docker

Jalankan PostgreSQL lokal dan buat database kosong bernama `adicara`. Migrasi dijalankan otomatis oleh API saat start (`MIGRATE_ON_START=true`).

```sh
cd backend
cp .env.example .env   # isi DB_PASSWORD dan JWT_SECRET
make run               # atau: go run ./cmd/api  → http://localhost:8080
```

Pada terminal lain:

```sh
cd frontend
cp .env.example .env   # API_BASE_URL=http://localhost:8080
npm ci
npm run dev            # → http://localhost:4321
```

Buka `http://localhost:4321`. Frontend memanggil API dari sisi server, jadi CORS tidak diperlukan. Detail alur login: [docs/frontend/authentication](docs/frontend/authentication/2026-10-04-session-cookies.md).

Perintah database manual (dari `backend/`): `make migrate-up`, dan `make migrate-fresh` (**destruktif**: menghapus seluruh isi database lalu migrasi ulang).

PowerShell tidak menyediakan `cp` secara bawaan sebagai perintah lintas platform; gunakan `Copy-Item .env.example .env`.

## Validasi

```sh
cd frontend
npm run format:check
npm run check
npm test
npm run build

cd ../backend
gofmt -l .
go vet ./...
go test ./...
go build ./cmd/api
```

## Struktur utama

- `frontend/` — Astro, design system, halaman marketing, template undangan, dan alur login (endpoint `/auth/*`, middleware `/dashboard`).
- `backend/` — Go API (Gin, Module-Based Clean Architecture: `auth`, `profile`, `health`), migrasi goose, dan kontrak aktif di `backend/docs/openapi.yaml`.
- `contracts/openapi.yaml` — **backup** kontrak API versi lama (cookie session + undangan); tidak sesuai backend dan tidak di-lint CI. Kontrak yang berlaku dan di-lint: `backend/docs/openapi.yaml`.
- `deploy/` — Nginx dan Compose produksi.
- `docs/` — arsitektur, operasi, dan catatan perubahan.

## Status saat ini

Berfungsi end-to-end (diuji dengan PostgreSQL 16 dan Chrome): registrasi, login (email atau username), sesi dengan refresh token otomatis, logout, dan perlindungan `/dashboard`.

Belum berfungsi: pembuatan dan pengelolaan undangan, tamu, RSVP, serta halaman publik `/i/[slug]`. Modul undangan dihapus dari backend dan menunggu dibangun ulang; halaman frontend-nya masih memanggil API lama.

Lihat [indeks dokumentasi](docs/README.md) untuk detail keputusan dan batasan fase ini.
