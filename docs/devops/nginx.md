# Nginx

Nginx adalah pintu masuk same-origin (`deploy/nginx/adicara.conf`):

| Path | Tujuan | Catatan |
|---|---|---|
| `= /healthz` | backend | |
| `/api/v1/*` | backend | Prefix `/api` dibuang: `/api/v1/auth/login` → `/v1/auth/login`. Backend tidak punya prefix `/api`. Path `/api/...` lain (bukan `/api/v1/`) jatuh ke frontend. |
| `/_astro/*` | frontend | Cache immutable. |
| route lain | frontend | Termasuk `/auth/*` (login, register, logout) dan `/dashboard`. |

Konfigurasi menambahkan compression, body-size limit, rate limit, dan security headers dasar. Nginx mengirim `X-Real-IP $remote_addr` ke upstream; header ini **dibutuhkan** karena Astro meneruskannya ke backend untuk rate limit per IP klien (lihat [autentikasi backend](../backend/authentication/session-authentication.md#rate-limit)). CSP saat ini mengizinkan inline style/script karena Astro dapat menghasilkan markup tersebut; kebijakan perlu diperketat setelah inventaris asset produksi tersedia.

## Rate limit

| Zona | Batas | Berlaku untuk |
|---|---|---|
| `auth_attempts` | 5 per menit per IP klien, burst 10 | `POST /auth/login`, `/auth/register` (form web, lewat Astro) **dan** `/api/v1/auth/login`, `/api/v1/auth/register` |
| `api_general` | 30 per detik, burst 60 | `/api/v1/*` |
| `public_writes` | 10 per menit, burst 15 | `/api/v1/public/invitations/{slug}/rsvp` (endpoint belum ada; dipasang lebih dulu) |

Form web memanggil `/auth/*` milik Astro, bukan `/api`, sehingga batas untuk menebak password harus dipasang di jalur itu juga; sebelumnya zona ini hanya cocok dengan `/api/v1/auth/...`, yang tidak dipakai siapa pun. Backend punya limiter sendiri (default 10 per menit per IP untuk register, login, refresh, dan logout sekaligus) yang berjalan di belakang batas Nginx.

Penolakan memakai status `429` dan body yang sama dengan error API (`{"error":{"code":"rate_limited",...}}`, `Content-Type: application/json`), sehingga form login dan klien API menanganinya seperti `rate_limited` dari backend. Security headers tetap disertakan.

## IP klien

TLS diterminasi oleh proxy host Sumopod, jadi bagi Nginx `$remote_addr` adalah proxy itu, bukan pengunjung. Tanpa penanganan, batas per IP di Nginx **dan** di backend (yang membaca `X-Real-IP`) menjadi satu bucket untuk seluruh situs, dan 5 login per menit akan berlaku untuk semua pengguna sekaligus.

Konfigurasi memakai modul `realip`:

```nginx
set_real_ip_from 127.0.0.1; set_real_ip_from ::1;
set_real_ip_from 10.0.0.0/8; set_real_ip_from 172.16.0.0/12; set_real_ip_from 192.168.0.0/16; set_real_ip_from fc00::/7;
real_ip_header X-Forwarded-For;
real_ip_recursive on;
```

- `X-Forwarded-For` hanya dipercaya bila koneksi berasal dari alamat privat/loopback, yaitu proxy host yang menjangkau container lewat jaringan Docker. Pengunjung yang langsung mengakses Nginx (mis. lewat `http://IP-publik`) punya alamat publik, sehingga header yang mereka kirim **diabaikan**.
- `real_ip_recursive on` memilih alamat **paling kanan yang tidak tepercaya**, jadi klien tidak bisa menyisipkan alamat palsu di depan daftar.
- Bila proxy host tidak mengirim `X-Forwarded-For` (kebanyakan proxy mengirimnya), Nginx memakai alamat koneksi dan semua pengguna berbagi satu bucket seperti sebelumnya. Perlu dicek pada proxy Sumopod yang sebenarnya, dan bila perlu ganti `real_ip_header` (mis. `X-Real-IP`).
- Bila Nginx container menjadi terminator TLS langsung, tambahkan konfigurasi sertifikat dan redirect HTTP secara terpisah; blok `set_real_ip_from` tidak berbahaya tetapi tidak lagi diperlukan.

## Validasi

Dijalankan 2026-10-04 dengan binary nginx 1.25.4 (milik Herd, termasuk modul `realip`; **bukan** `nginx:1.29-alpine` dari Compose) menjalankan `adicara.conf` apa adanya, hanya alamat upstream dan port yang diganti, di depan backend Go sungguhan (mode production, PostgreSQL 16) dan stub frontend yang mencetak header yang diterimanya:

- `nginx -t` lolos.
- `GET /healthz` → `200`. `POST /api/v1/auth/login` dengan kredensial salah → `401` JSON dari backend (bukan `404`); register, login, dan `GET /api/v1/me` lewat `/api/v1` berhasil. `/api/unknown` diteruskan ke frontend, bukan backend.
- `X-Forwarded-For: 203.0.113.7` dari peer tepercaya → upstream menerima `X-Real-IP: 203.0.113.7`; `X-Forwarded-For: 6.6.6.6, 203.0.113.7` → tetap `203.0.113.7`. Dengan peer yang tidak tepercaya (varian konfigurasi tanpa `set_real_ip_from` loopback), `X-Forwarded-For: 9.9.9.9` diabaikan dan upstream menerima `127.0.0.1`.
- 14 `POST /auth/login` cepat dari satu IP klien: 11 lolos lalu `429` dengan body JSON dan header keamanan; IP klien lain di balik proxy yang sama tidak terdampak. `POST /api/v1/auth/login` serupa; log backend menunjukkan IP klien yang benar. `GET /auth/masuk` 14 kali tidak dibatasi.

**Belum diuji:** `nginx:1.29-alpine` di Docker, di belakang proxy Sumopod yang sebenarnya (apakah ia mengirim `X-Forwarded-For`), dan pengujian dari peer berstatus publik. Deploy menjalankan `nginx -t` sebelum reload dan akan gagal (tanpa memutus Nginx yang sedang berjalan) bila konfigurasi tidak valid.
