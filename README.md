# Adicara

Adicara adalah platform undangan digital dan perayaan yang mobile-first untuk Indonesia. Repositori ini memuat website marketing dan katalog tema Astro, autentikasi serta dashboard awal, Go REST API, PostgreSQL, kontrak OpenAPI, serta fondasi Docker dan CI/CD.

## Prasyarat

- Node.js 22.12 atau lebih baru (Node 24 direkomendasikan)
- npm 10 atau lebih baru
- Go 1.27 atau lebih baru
- Docker Engine + Docker Compose untuk menjalankan seluruh stack

## Menjalankan seluruh stack

1. Salin `.env.example` menjadi `.env` dan ganti password lokal.
2. Jalankan:

   ```sh
   docker compose up --build
   ```

3. Buka `http://localhost:8080`.
4. Periksa API melalui `http://localhost:8080/healthz` dan `http://localhost:8080/readyz`.

PostgreSQL lokal tersedia pada port `5432`. Jangan gunakan nilai `.env.example` untuk produksi.

## Menjalankan tanpa Docker

Jalankan PostgreSQL dan migrasi terlebih dahulu, lalu:

```sh
cd backend
cp .env.example .env
go run ./cmd/api
```

Pada terminal lain:

```sh
cd frontend
cp .env.example .env
npm ci
npm run dev
```

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

- `frontend/` — Astro, design system, halaman marketing, dan template undangan.
- `backend/` — Go API, domain, service, repository PostgreSQL, dan migrasi.
- `contracts/openapi.yaml` — kontrak API kanonis.
- `deploy/` — Nginx dan Compose produksi.
- `docs/` — arsitektur, operasi, dan catatan perubahan.

Lihat [indeks dokumentasi](docs/README.md) untuk detail keputusan dan batasan fase ini.
