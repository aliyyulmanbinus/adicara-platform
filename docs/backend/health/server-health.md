# Health Check API

Modul `health` menyediakan endpoint untuk monitoring dan healthcheck Docker. Rute berada di root (`/healthz`), bukan di bawah `/v1`.

## `GET /healthz`

Melakukan ping ke PostgreSQL dan berfungsi sebagai liveness **dan** readiness (tidak ada `/readyz`).

- Database merespons: `200 {"status":"ok"}`.
- Database tidak terjangkau: `503 {"error":{"code":"database_unavailable","message":"cannot reach Postgres"}}`.

## Healthcheck Docker

`/adicara-api -healthcheck` memanggil endpoint ini di listener lokal (`PORT`/`HTTP_ADDR`) dan keluar dengan kode 0 hanya jika menjawab `200`; selain itu kode 1 (koneksi ditolak, atau `503`). Ketiga kasus sudah diuji secara lokal.
