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
| `GET /v1/planner/event-types` | Bearer | Registry sembilan jenis acara + `templates_available`. |
| `GET/POST /v1/planner/projects` | Bearer | Daftar kartu Hub (`?status=`, `?view=lite` untuk project switcher) dan pembuatan proyek (atomik, `client_token`, 10/menit/pengguna). |
| `GET /v1/planner/projects/summary` · `POST /v1/planner/projects/sample` | Bearer | Metrik portofolio; membuat Proyek Contoh (idempoten). |
| `GET/PATCH/DELETE /v1/planner/projects/:id` · `PATCH .../status` | Bearer | Detail (+ `shiftable_tasks`), ubah (geser jadwal opsional), hapus (`?confirm=<judul>`), ubah status. Proyek bukan milik pemanggil dan id non-UUID: `404`. Proyek `archived` baca-saja kecuali status dan hapus (`409 project_read_only`). |
| `GET /v1/planner/projects/:id/dashboard` | Bearer | Enam widget; 60/menit/pengguna. Widget `null` bila modulnya belum ada. |
| `GET/POST .../sessions` · `POST .../sessions/reorder` · `PUT/DELETE .../sessions/:session_id` | Bearer | Master sesi bersama. |
| `GET/POST /v1/planner/projects/:id/tasks` · `PATCH/DELETE .../tasks/:task_id` · `PATCH .../tasks/:task_id/status` | Bearer | Menu 02: tugas persiapan (saring `status`, `category`, `scope`, `phase`; `summary` seluruh proyek). Maks 500 per proyek. |
| `POST .../tasks/reorder` · `POST .../tasks/shift` · `POST .../tasks/generate-defaults` | Bearer | Urut ulang (daftar lengkap), geser manual tugas template yang belum selesai, dan buat jadwal dari template (`append`, `replace_template`, `replace_all` + `confirm`). |
| `GET .../tasks/export-spreadsheet` | Bearer | `.xlsx` semua tugas + Gantt mingguan; 10/menit/pengguna. |
| `GET/POST .../admin-docs` · `PATCH/DELETE .../admin-docs/:doc_id` · `PATCH .../admin-docs/:doc_id/status` · `POST .../admin-docs/reorder` | Bearer | Menu 02: dokumen legalitas/perizinan. Maks 200 per proyek. |
| `POST .../admin-docs/apply-template` · `POST .../admin-docs/:doc_id/sync-budget` | Bearer | Berkas standar jenis acara; salin biaya berkas ke pos anggaran (idempoten, `201` bila pos baru). |

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

`code` stabil dan dimaksudkan untuk dibaca mesin; `message` berbahasa Inggris dan dapat berubah, jadi klien sebaiknya memetakan `code` ke teksnya sendiri. Pada galat validasi Planner (`invalid_input`, `invalid_metadata`, `invalid_date`, `invalid_timezone`) `message` berbentuk `<field>: <alasan>` agar formulir dapat menandai kolomnya. `error.details` hanya ada bila klien butuh konteks (mis. `session_in_use` memuat `usage`).

| Status | Kode |
|---|---|
| 400 | `invalid_json`, `invalid_input`, `invalid_email`, `invalid_username`, `weak_password`, `password_too_long`, `invalid_metadata`, `invalid_scope`, `invalid_date`, `invalid_timezone`, `invalid_template` (Planner) |
| 401 | `unauthorized` (kredensial/token salah atau tidak ada; juga akun nonaktif atau terhapus pada rute terlindungi dan refresh, serta token yang terbit sebelum ganti password) |
| 403 | `account_inactive` |
| 404 | `not_found` |
| 409 | `email_taken`, `username_taken`, `project_read_only`, `project_limit_reached`, `session_in_use`, `session_name_taken`, `session_limit_reached`, `task_limit_reached`, `admin_doc_limit_reached`, `conflict` (Planner) |
| 429 | `rate_limited` |
| 500 | `internal` |
| 503 | `database_unavailable` |

## Rate limit

`/v1/auth/{register,login,refresh,logout}` dibatasi `AUTH_RATE_LIMIT` percobaan per IP klien per menit (default 10). IP klien dibaca dari `X-Real-IP`, hanya jika koneksi berasal dari proxy di `TRUSTED_PROXIES`. Server perantara (Astro) wajib meneruskan header itu, jika tidak semua pengguna berbagi satu bucket. Nginx menambah batas sendiri di depan backend untuk `/auth/{login,register}` (form web) dan `/api/v1/auth/{login,register}` (5 per menit per IP klien, burst 10) dan menjawab `429` dengan body error API yang sama; IP klien di Nginx diambil dari `X-Forwarded-For` bila koneksi datang dari proxy privat/loopback ([Nginx](devops/nginx.md)).

Planner membatasi per **pengguna** (bukan IP): `POST /v1/planner/projects` 10 per menit, `GET .../dashboard` 60 per menit, dan `GET .../tasks/export-spreadsheet` 10 per menit, dihitung di memori tiap proses.

## CORS

Backend mengizinkan origin di `CORS_ORIGINS` (default `http://localhost:5173`) dengan credentials. Pada alur normal CORS tidak terpakai karena browser tidak memanggil backend langsung.

## Riwayat: endpoint yang sudah dihapus

Versi sebelumnya (cookie session + CSRF, prefix `/api/v1`) menyediakan `/api/v1/public/invitations/{slug}`, `/api/v1/invitations` (CRUD, publish/unpublish), `/api/v1/invitations/{id}/guests`, dan RSVP publik. Semuanya dihapus bersama modulnya dan akan kembali saat modul undangan dibangun ulang. Frontend (`/i/[slug]`, form dan tombol di dashboard) masih memanggil endpoint tersebut, sehingga fitur itu belum berfungsi.
