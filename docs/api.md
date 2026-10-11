# API

Kontrak yang berlaku adalah [`backend/docs/openapi.yaml`](../backend/docs/openapi.yaml) (OpenAPI 3.0.3); inilah yang di-lint CI. [`contracts/openapi.yaml`](../contracts/openapi.yaml) hanya **backup** API versi sebelumnya: tidak sesuai backend dan tidak di-lint; lihat [Riwayat](#riwayat-endpoint-yang-sudah-dihapus). Ubah kontrak hanya di `backend/docs/openapi.yaml`.

## Endpoint saat ini

| Method dan path | Auth | Fungsi |
|---|---|---|
| `GET /healthz` | — | `200 {"status":"ok"}` jika PostgreSQL terjangkau, selain itu `503` dengan kode `database_unavailable`. Dipakai sebagai liveness **dan** readiness; tidak ada `/readyz`. |
| `POST /v1/auth/register` | — | Membuat akun (role customer). `201 {"data": user}`. Tidak mengembalikan token. |
| `POST /v1/auth/login` | — | Email **atau** username + password. `200 {"data": user, "tokens": {access_token, refresh_token, expires_in}}`. |
| `POST /v1/auth/refresh` | — | Body `{refresh_token}`. `200 {"tokens": ...}`. Token **dirotasi**: yang dikirim langsung hangus. Mengirim ulang token yang sudah dirotasi > 10 detik kemudian mencabut **semua** sesi pengguna. `401` juga untuk akun nonaktif. |
| `POST /v1/auth/logout` | — | Body `{refresh_token}`. `204`. Mencabut sesi refresh. |
| `POST /v1/auth/password` | Bearer | Body `{current_password, new_password}`. `204`. Mencabut **semua** sesi refresh dan membuat access token yang terbit sebelumnya `401`. |
| `GET /v1/me` | Bearer | `200 {"data": {"user": {id, email, username, name}}}`. |
| `PATCH /v1/me` | Bearer | Body `{name}` (wajib, tidak kosong). `200 {"data": {id, email, username, name}}`. |

Path backend tidak memakai prefix `/api`. Lewat Nginx (origin tunggal) path yang sama tersedia sebagai `/api/v1/...`: Nginx membuang `/api` sebelum meneruskan, jadi `/api/v1/auth/login` sampai ke backend sebagai `/v1/auth/login`. Hanya `/api/v1/*` yang diteruskan; `/healthz` tetap di root. Aturan input dan detail mekanisme ada di [Session & JWT Authentication](backend/authentication/session-authentication.md).

## Autentikasi

Access token adalah JWT (HS256, default 15 menit) yang dikirim sebagai `Authorization: Bearer <token>`. Selain tanda tangan, backend memeriksa database pada setiap request: akun harus ada dan aktif, dan token tidak boleh terbit sebelum penggantian password terakhir. Semua kegagalan itu `401 unauthorized` ([ADR-0003](adr/ADR-0003-access-token-database-check.md)). Refresh token juga JWT (default 7 hari) yang `jti`-nya menunjuk baris di tabel `refresh_sessions`. Backend **tidak** memakai cookie.

Browser tidak memanggil API ini langsung. Server Astro memanggilnya (`API_BASE_URL`) dan menyimpan token di cookie HttpOnly; browser hanya memanggil endpoint same-origin `POST /auth/login`, `/auth/register`, dan `/auth/logout`. Lihat [dokumentasi frontend](frontend/authentication/2026-10-04-session-cookies.md).

## Format error

```json
{
  "error": {
    "code": "email_taken",
    "message": "email already registered"
  }
}
```

`code` stabil dan dimaksudkan untuk dibaca mesin; `message` berbahasa Inggris dan dapat berubah, jadi klien sebaiknya memetakan `code` ke teksnya sendiri.

| Status | Kode |
|---|---|
| 400 | `invalid_json`, `invalid_input`, `invalid_email`, `invalid_username`, `weak_password`, `password_too_long` |
| 401 | `unauthorized` (kredensial/token salah atau tidak ada; juga akun nonaktif atau terhapus pada rute terlindungi dan refresh, serta token yang terbit sebelum ganti password) |
| 403 | `account_inactive` |
| 404 | `not_found` |
| 409 | `email_taken`, `username_taken` |
| 429 | `rate_limited` |
| 500 | `internal` |
| 503 | `database_unavailable` |

## Rate limit

`/v1/auth/{register,login,refresh,logout}` dibatasi `AUTH_RATE_LIMIT` percobaan per IP klien per menit (default 10). IP klien dibaca dari `X-Real-IP`, hanya jika koneksi berasal dari proxy di `TRUSTED_PROXIES`. Server perantara (Astro) wajib meneruskan header itu, jika tidak semua pengguna berbagi satu bucket. Nginx menambah batas sendiri di depan backend untuk `/auth/{login,register}` (form web) dan `/api/v1/auth/{login,register}` (5 per menit per IP klien, burst 10) dan menjawab `429` dengan body error API yang sama; IP klien di Nginx diambil dari `X-Forwarded-For` bila koneksi datang dari proxy privat/loopback ([Nginx](devops/nginx.md)).

## CORS

Backend mengizinkan origin di `CORS_ORIGINS` (default `http://localhost:5173`) dengan credentials. Pada alur normal CORS tidak terpakai karena browser tidak memanggil backend langsung.

## Riwayat: endpoint yang sudah dihapus

Versi sebelumnya (cookie session + CSRF, prefix `/api/v1`) menyediakan `/api/v1/public/invitations/{slug}`, `/api/v1/invitations` (CRUD, publish/unpublish), `/api/v1/invitations/{id}/guests`, dan RSVP publik. Semuanya dihapus bersama modulnya dan akan kembali saat modul undangan dibangun ulang. Frontend (`/i/[slug]`, form dan tombol di dashboard) masih memanggil endpoint tersebut, sehingga fitur itu belum berfungsi.
