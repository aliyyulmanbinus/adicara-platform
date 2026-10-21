# Docker

## Lokal

`docker-compose.yml` menjalankan PostgreSQL, Go API, Astro, dan Nginx. Akses aplikasi melalui port `8080`; PostgreSQL diekspos pada `5432` hanya untuk kebutuhan development lokal.

Urutan start: PostgreSQL healthy → backend (menjalankan migrasi, lalu melayani) → frontend (menunggu backend healthy) → Nginx.

## Produksi

`deploy/docker-compose.prod.yml` memakai image frontend/backend dari GHCR. PostgreSQL tidak memiliki published port dan Nginx bind ke loopback VPS.

Frontend dan backend memakai multi-stage build serta user non-root pada runtime.

## Image backend

Berisi dua binary:

- `/adicara-api` — server. `/adicara-api -healthcheck` memanggil `/healthz` di listener lokal dan keluar dengan kode 0 hanya jika menjawab `200` (artinya API hidup **dan** PostgreSQL terjangkau). Dipakai sebagai `healthcheck` Compose.
- `/adicara-migrate` — `up` dan `fresh -yes` (destruktif). Dijalankan manual, misalnya `make migrate-up` atau `make migrate-fresh` dari root repository, yang memanggil `docker compose run --rm --no-deps --entrypoint /adicara-migrate backend ...`.

## Build context dan `.dockerignore`

Kedua Dockerfile memakai `COPY . .`, jadi apa pun di `backend/` dan `frontend/` ikut terkirim ke build. `backend/.dockerignore` dan `frontend/.dockerignore` (ditambahkan 2026-10-11) mengecualikan `.env*` (kecuali `.env.example`), `node_modules`, `dist`, `.astro`, binary `adicara-api`, dan file OS. Tanpanya, `docker compose up --build` di komputer pengembang mengirim `backend/.env` (berisi password database) ke lapisan build, dan `frontend/node_modules` milik host (paket macOS/Windows) bercampur dengan hasil `npm ci` di Alpine. Checkout CI yang bersih tidak terpengaruh. Diukur dengan pencocok pola yang sama dengan Docker (`moby/patternmatcher`) pada pohon kerja lokal:

| Konteks | Tanpa `.dockerignore` | Dengan `.dockerignore` |
| --- | --- | --- |
| `backend` | 44 file, 36,7 MB; termasuk `.env` dan binary `adicara-api` | 42 file, 0,1 MB; tidak ada rahasia/artefak |
| `frontend` | 13.657 file, 270,6 MB; termasuk `node_modules` dan `dist` | 90 file, 1,1 MB |

Setiap file yang dilacak git tetap ada di konteks (backend 41 + `.dockerignore`, frontend 89 + `.dockerignore`), termasuk `internal/migrate/sql/*.sql` yang di-embed lewat `go:embed`.

## Validasi

Docker tidak tersedia pada lingkungan implementasi. Yang sudah dijalankan: build kedua binary, flag `-healthcheck` (sukses, tanpa server, dan saat PostgreSQL tidak terjangkau), dan validasi struktur YAML kedua file Compose. Kedua file Compose juga dirender dengan library `compose-go` v2.16.1 (basis Docker Compose) di bawah beberapa isi `.env`: produksi tanpa `JWT_SECRET` ditolak dengan pesan yang jelas, dengan `JWT_SECRET` backend menerima env yang diharapkan (variabel opsional kosong, `CORS_ORIGINS` = `PUBLIC_SITE_URL`), dan override terbaca. Pemeriksaan ulang setelah merge `backend` ke `development` (2026-10-11, tetap tanpa Docker): langkah build `backend/Dockerfile` ditiru dari salinan bersih isi index git (`CGO_ENABLED=0`, `GOOS=linux`, `GOARCH` amd64 dan arm64; `cmd/api` 25 MB dan `cmd/migrate` 9,9 MB untuk amd64) dan `go mod tidy` tidak mengubah `go.mod`/`go.sum`. Binary dari salinan itu menjalankan 4 migrasi dari SQL yang ter-embed, `-healthcheck` keluar dengan kode 1 saat server mati dan 0 saat hidup, dan `APP_ENV=production` dengan `JWT_SECRET` placeholder ditolak. Langkah `frontend/Dockerfile` ditiru (`npm ci`, `PUBLIC_SITE_URL=https://example.com npm run build`, `npm prune --omit=dev`; Node 22.17.1, bukan 24 seperti di image); server dari `dist` + `node_modules` hasil prune melayani `/`, `/auth/masuk`, `/tema/batak-senja` (200), `/dashboard` tanpa login (302), dan canonical memakai build arg. Render `compose-go` v2.16.1 ulang untuk Compose dev (tanpa env dan dengan `.env.example`) dan produksi (tanpa `JWT_SECRET` ditolak, dengan `JWT_SECRET` benar, override bind/port terbaca). Tag image (`golang:1.27-alpine`, `alpine:3.23`, `node:24-alpine`, `postgres:18-alpine`, `nginx:1.29-alpine`) tidak berubah dari `main` (hanya service `migrate/migrate` yang dihapus); keberadaannya tidak bisa dicek langsung karena registry Docker tidak terjangkau dari lingkungan ini.

Update 2026-10-11: image yang dibangun CI (Alpine, Node 24) dan Compose produksi sudah berjalan di VPS: backend dan frontend start, migrasi terapan, `/healthz` 200 ([catatan](2026-10-11-merge-backend-into-development.md#hasil-deploy-produksi-2026-10-11)). **Masih belum** dijalankan: `docker compose up` untuk `docker-compose.yml` lokal, perilaku `.dockerignore` oleh Docker sungguhan (CI memakai checkout bersih), serta perintah `make migrate-*` lewat Docker. Workflow CI menyediakan build image ketika dijalankan di GitHub Actions.
