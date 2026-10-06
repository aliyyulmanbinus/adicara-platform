# Adicara

Adicara adalah platform undangan digital dan perayaan yang mobile-first untuk Indonesia. Repositori ini memuat website marketing dan katalog tema Astro, autentikasi serta dashboard awal, Go REST API, PostgreSQL, kontrak OpenAPI, serta fondasi Docker dan CI/CD.

## Menjalankan secara lokal

Ikuti [panduan menjalankan secara lokal](docs/local-development.md) untuk persiapan database, Docker Compose, atau Windows tanpa Docker. Panduan tersebut juga memuat URL login dan pratinjau template Batak Senja.

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
