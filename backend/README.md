# Adicara Backend

Backend API untuk platform Adicara. Saat ini hanya mencakup modul inti: **Auth**, **Profile**, dan **Health**. Arsitekturnya Module-Based Clean Architecture di atas Gin; lihat [docs/architecture.md](../docs/architecture.md).

## Persiapan Lokal

1. Jalankan PostgreSQL lokal (mis. DBngin atau Docker) dan buat database kosong bernama `adicara`.
2. Salin `.env.example` menjadi `.env`, lalu isi `DB_PASSWORD` dan `JWT_SECRET` (acak, panjang; di produksi nilai ini divalidasi saat start).
3. Jalankan:

   ```bash
   make run          # atau: go run ./cmd/api   → http://localhost:8080
   ```

   Migrasi dijalankan otomatis saat start (`MIGRATE_ON_START=true`).

## Perintah

| Perintah | Fungsi |
|---|---|
| `make run` | Menjalankan API. |
| `make test` | `go test ./...`. |
| `make tidy` | `go mod tidy`. |
| `make migrate-up` | Menjalankan migrasi yang belum ter-apply. |
| `make migrate-fresh` | **Destruktif.** Menghapus seluruh isi database di `.env`, lalu migrasi ulang. |

## Environment

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` / `HTTP_ADDR` | `:8080` | `PORT` didahulukan bila ada. |
| `DATABASE_URL` | dirakit dari `DB_*` | Bila diisi, menimpa `DB_*`. |
| `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`, `DB_SSLMODE` | `127.0.0.1`, `5432`, `postgres`, (kosong), `adicara`, `disable` | |
| `APP_ENV` | `development` | `development`, `dev`, `local`, `test`, atau kosong = development. Nilai lain (`production`, `staging`, salah ketik) mengaktifkan validasi `JWT_SECRET`. |
| `JWT_SECRET` | `change-me-in-dev` | **Wajib** (acak, ≥ 32 karakter, mis. `openssl rand -hex 32`) bila `APP_ENV` bukan development: backend menolak start dengan nilai kosong, bawaan, placeholder, atau pendek. Boleh bawaan di development. |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | `15m` / `168h` | Format durasi Go. |
| `MIGRATE_ON_START` | `true` | |
| `AUTH_RATE_LIMIT` | `10` | Percobaan per IP per menit untuk `/v1/auth/{register,login,refresh,logout}`; `0` menonaktifkan. |
| `TRUSTED_PROXIES` | loopback + jaringan privat | Proxy yang boleh mengirim `X-Real-IP` (CSV IP/CIDR). Nilai tidak valid menghentikan start. |
| `CORS_ORIGINS` | `http://localhost:5173` | Tidak diperlukan pada alur normal (browser tidak memanggil backend langsung). |

`PUBLIC_BASE_URL` dan `DB_CONNECTION` ada di `.env.example` tetapi tidak dibaca oleh kode.

## Endpoint

| Area | Path |
|---|---|
| Health | `GET /healthz` |
| Auth | `POST /v1/auth/register` · `/login` · `/refresh` · `/logout` · `/password` |
| Profile | `GET /v1/me` · `PATCH /v1/me` |

Kontrak lengkap dan satu-satunya yang berlaku: [`docs/openapi.yaml`](docs/openapi.yaml) (`../contracts/openapi.yaml` hanya backup versi lama). Ringkasan, format error, dan rate limit: [docs/api.md](../docs/api.md). Docker: `/adicara-api -healthcheck` dan `/adicara-migrate` ([docs/devops/docker.md](../docs/devops/docker.md)).

## Dokumentasi

Dokumentasi per modul ada di [docs/backend](../docs/backend/README.md).
