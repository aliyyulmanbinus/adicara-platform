# Deployment

## Topologi

Produksi memakai empat service utama: Nginx, frontend Astro, backend Go, dan PostgreSQL. Service migrasi berjalan sebagai one-shot job. PostgreSQL hanya terhubung ke network privat Compose.

Nginx pada `deploy/docker-compose.prod.yml` bind ke `127.0.0.1:8080`; HTTPS publik diasumsikan diterminasi oleh reverse proxy/host Sumopod di depan stack. Sesuaikan binding hanya bila model jaringan VPS berbeda.

## Persiapan VPS

1. Buat pengguna deployment non-root.
2. Checkout repository ke `/opt/adicara`.
3. Salin migrasi ke lokasi yang dipasang oleh production Compose atau pertahankan struktur repository.
4. Buat file environment produksi di VPS; jangan commit.
5. Pastikan Docker dapat menarik package GHCR milik repository.
6. Atur reverse proxy HTTPS host ke `127.0.0.1:8080`.

## GitHub Actions

Workflow deploy membutuhkan:

- `VPS_HOST`
- `VPS_USER`
- `VPS_SSH_KEY`
- `VPS_PORT`
- repository variable `PUBLIC_SITE_URL`

Workflow membangun dua image, memberi tag SHA Git, melakukan push ke GHCR, menjalankan Compose melalui SSH, lalu memeriksa `/readyz`.

## Rollback

Tetapkan `FRONTEND_IMAGE` dan `BACKEND_IMAGE` ke SHA image sebelumnya, lalu jalankan ulang `docker compose up -d`. Migrasi destruktif tidak boleh digabung dengan deployment yang memerlukan rollback aplikasi sederhana.

Lihat [panduan VPS](devops/vps-deployment.md) dan [backup/restore](devops/backup-restore.md).

