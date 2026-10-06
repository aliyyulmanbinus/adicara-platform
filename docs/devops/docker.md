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

## Validasi

Docker tidak tersedia pada lingkungan implementasi. Yang sudah dijalankan: build kedua binary, flag `-healthcheck` (sukses, tanpa server, dan saat PostgreSQL tidak terjangkau), dan validasi struktur YAML kedua file Compose. Kedua file Compose juga dirender dengan library `compose-go` v2.16.1 (basis Docker Compose) di bawah beberapa isi `.env`: produksi tanpa `JWT_SECRET` ditolak dengan pesan yang jelas, dengan `JWT_SECRET` backend menerima env yang diharapkan (variabel opsional kosong, `CORS_ORIGINS` = `PUBLIC_SITE_URL`), dan override terbaca. **Belum** dijalankan: `docker compose config`/`up`, build image, serta perintah `make migrate-*` lewat Docker. Workflow CI menyediakan build image ketika dijalankan di GitHub Actions.
