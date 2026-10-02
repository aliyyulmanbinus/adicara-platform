# Docker

## Lokal

`docker-compose.yml` menjalankan PostgreSQL, migrator, Go API, Astro, dan Nginx. Akses aplikasi melalui port `8080`; PostgreSQL diekspos pada `5432` hanya untuk kebutuhan development lokal.

## Produksi

`deploy/docker-compose.prod.yml` memakai image frontend/backend dari GHCR. PostgreSQL tidak memiliki published port dan Nginx bind ke loopback VPS.

Frontend dan backend memakai multi-stage build serta user non-root pada runtime. Service migrator harus berhasil sebelum backend dijalankan.

## Validasi

Docker tidak tersedia pada lingkungan implementasi awal. Image build dan Compose runtime belum diverifikasi secara lokal; workflow CI menyediakan pemeriksaan tersebut ketika dijalankan di GitHub Actions.

