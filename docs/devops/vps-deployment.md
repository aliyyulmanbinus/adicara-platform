# VPS Deployment

## Struktur yang diharapkan

Repository berada di `/opt/adicara`; file environment produksi dikelola langsung di VPS. Reverse proxy host mengirim trafik HTTPS ke `127.0.0.1:8080`.

## Persiapan awal

1. Install Docker Engine dan Compose plugin.
2. Buat user deployment non-root dan batasi SSH.
3. Checkout repository ke `/opt/adicara`.
4. Buat `.env` dengan `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `DATABASE_URL`, `PUBLIC_SITE_URL`, `FRONTEND_IMAGE`, dan `BACKEND_IMAGE`.
5. Login ke GHCR dengan token yang hanya memiliki izin package read.
6. Jalankan `docker compose -f deploy/docker-compose.prod.yml up -d`.
7. Periksa `curl --fail http://127.0.0.1:8080/readyz`.

## Operasi

Periksa log terstruktur backend dengan `docker compose logs backend`. Jangan menyalin log yang mungkin berisi data pribadi ke kanal publik.

Deployment ini belum dijalankan pada VPS dari lingkungan implementasi saat ini.

