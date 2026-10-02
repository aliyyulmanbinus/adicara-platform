# GitHub Actions

## CI

`.github/workflows/ci.yml` berjalan pada pull request dan push ke `main`, serta dapat dipanggil workflow lain. Job memeriksa:

- format, Astro type-check, source tests, dan build frontend;
- gofmt, vet, test, dan build backend;
- lint OpenAPI;
- build kedua Docker image.

## CD

`.github/workflows/deploy.yml` memanggil CI, membangun image frontend/backend, memberi tag Git SHA dan `main`, lalu push ke GHCR. Deployment melalui SSH menggunakan environment `production`, menjalankan Compose, dan memeriksa `/readyz`.

Required secrets: `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, dan `VPS_PORT`. Required repository variable: `PUBLIC_SITE_URL`.

## Catatan keamanan

Gunakan deployment user non-root dengan akses minimum. Jangan memasukkan `.env` produksi atau private key ke repository. Pertimbangkan PAT read-only di VPS bila package privat tidak dapat ditarik dengan kredensial workflow sementara.

