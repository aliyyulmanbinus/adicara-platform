# Session & JWT Authentication

Modul `auth` memakai dua JWT (HS256, ditandatangani `JWT_SECRET`) yang dikirim di body JSON. Backend **tidak** mengatur cookie. Refresh token dilacak di database sehingga bisa dicabut.

## Mekanisme

1. **Register** — validasi input (lihat Aturan Input), password di-hash bcrypt cost 12, baris `users` dibuat (role customer, status `active`). Tidak ada token; klien memanggil login setelahnya.
2. **Login** — identifier (email atau username) dipangkas dan dikecilkan, dicari lewat `lower(email)` atau `lower(username)`, password dibandingkan dengan bcrypt. Identifier yang tidak ditemukan tetap membandingkan password dengan hash bcrypt tiruan (biaya sama), jadi waktu respons tidak membedakan akun yang ada dari yang tidak. Status `inactive` ditolak dengan `403 account_inactive` (dicek **setelah** password benar, jadi tidak membocorkan status akun ke penebak). Lalu diterbitkan:
   - **Access token** (`typ=access`, `sub` = id pengguna, umur `JWT_ACCESS_TTL`, default 15 menit) untuk rute terlindungi seperti `/v1/me`.
   - **Refresh token** (`typ=refresh`, `sub`, `jti` = id baris `refresh_sessions`, umur `JWT_REFRESH_TTL`, default 168 jam). Baris sesinya dibuat saat itu juga.
3. **Refresh** — refresh token diverifikasi, lalu sesinya "dikonsumsi" secara atomik (`UPDATE ... WHERE revoked_at IS NULL AND expires_at > now()`). Jika tidak ada baris yang berubah, hasilnya `401`; bila sesi itu pernah dihabiskan oleh rotasi **lebih dari 10 detik** lalu, token dianggap bocor dan **semua** sesi refresh pengguna dicabut (dalam 10 detik pertama hanya `401`, karena itu kemungkinan besar dua request klien yang berpacu). Jika berhasil, status akun diperiksa (nonaktif atau terhapus: `401`), lalu pasangan token **baru** (dan sesi baru) diterbitkan. Jadi setiap refresh token hanya bisa dipakai sekali. Rincian: [pengerasan autentikasi](2026-10-04-auth-hardening.md).
4. **Logout** — sesi refresh dicabut (`revoked_at` diisi; barisnya tidak dihapus). Sesi yang sudah dicabut atau hilang diperlakukan sebagai sukses. Refresh token yang tidak valid atau sudah kedaluwarsa menghasilkan `401`. Respons `204`.
5. **Ganti password** (`POST /v1/auth/password`, Bearer) — password lama diverifikasi (`401` jika salah), password baru divalidasi dengan aturan yang sama dengan register, lalu hash diganti dan **semua** sesi refresh milik pengguna dicabut dalam satu transaksi. Respons `204`.
6. **Rute terlindungi** — middleware `BearerAuth` memverifikasi tanda tangan dan `typ=access`, lalu **membaca database** (satu query PK): akun harus ada dan `active`, dan token tidak boleh terbit sebelum `users.password_changed_at`. Kegagalan apa pun adalah `401 unauthorized` (bukan `403`, supaya klien yang memperlakukan 401 sebagai "masuk lagi" berakhir di form login; lihat [ADR-0003](../../adr/ADR-0003-access-token-database-check.md)).

Konfigurasi: `JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `APP_ENV` (lihat [backend/README.md](../../../backend/README.md)). Bila `APP_ENV` bukan nilai development, backend menolak start dengan `JWT_SECRET` kosong, bawaan, placeholder, atau kurang dari 32 karakter.

## Struktur Data
- **`users`** — identitas pengguna; `password_changed_at` menandai batas waktu access token.
- **`refresh_sessions`** — satu baris per refresh token yang pernah diterbitkan: `id` (= `jti`), `user_id`, `expires_at`, `revoked_at`, `rotated_at` (terisi hanya bila dihabiskan oleh rotasi).

Skema lengkap: [database](../../database.md).

## Aturan Input
| Field | Aturan | Kode error |
|---|---|---|
| `email` | Format alamat email valid, domain mengandung titik, maks 254 karakter. Disimpan huruf kecil. | `invalid_email` |
| `username` | 3–30 karakter, hanya `a-z 0-9 . _` (huruf besar otomatis dikecilkan). | `invalid_username` |
| `password` | Min 8 karakter, wajib ada huruf dan angka. | `weak_password` |
| `password` | Maks 72 **byte** (batas bcrypt). | `password_too_long` |

Login menerima email **atau** username (`email` atau `username` pada body; `email` dipakai bila keduanya ada). Kesalahan kredensial selalu `401 unauthorized` tanpa membedakan akun tidak ada dari password salah. Daftar lengkap kode error: [API](../../api.md#format-error).

## Konsumen Frontend
Frontend Astro memanggil backend dari sisi server dan menyimpan token di cookie HttpOnly — lihat [Sesi autentikasi (frontend)](../../frontend/authentication/2026-10-04-session-cookies.md). Hal yang harus dipegang klien mana pun:
- **Pesan error** berupa kode di `error.code`; teks `error.message` berbahasa Inggris dan bisa berubah, jadi klien sebaiknya memetakan kodenya sendiri.
- **Logout tidak mencabut access token.** Logout mencabut refresh token saja; access token yang sudah terbit tetap valid sampai kedaluwarsa (default 15 menit). Sebaliknya, **ganti password** dan **menonaktifkan akun** langsung membuat access token lama `401`.
- **Refresh token dirotasi.** Memakai token yang sama dua kali menghasilkan 401, jadi request paralel yang sama-sama butuh refresh harus digabung menjadi satu panggilan. Mengirim ulang token yang sudah dirotasi lebih dari 10 detik kemudian mencabut semua sesi pengguna.
- Server perantara harus meneruskan IP klien asli di header `X-Real-IP`; jika tidak, semua pengguna berbagi satu bucket rate limit (IP server perantara).

## Rate Limit
`POST /v1/auth/{register,login,refresh,logout}` dibatasi `AUTH_RATE_LIMIT` percobaan per IP per menit (default 10, `0` menonaktifkan). Melewati batas menghasilkan `429 rate_limited`. `POST /v1/auth/password` tidak dibatasi.

IP klien ditentukan dari header `X-Real-IP`, **hanya jika** koneksi datang dari proxy yang terdaftar di `TRUSTED_PROXIES` (CSV IP/CIDR; default loopback + jaringan privat `10/8`, `172.16/12`, `192.168/16`, `fc00::/7`). Dari peer lain header diabaikan, sehingga klien yang menembak langsung tidak bisa menghindari limit dengan memalsukan header. `X-Forwarded-For` tidak dipakai. Nginx mengirim `X-Real-IP $remote_addr` (sudah ada di `deploy/nginx/adicara.conf`); jika ada proxy lain di depan nginx, nginx perlu `real_ip` agar `$remote_addr` berisi IP klien sebenarnya.

## Pembersihan Sesi
Setiap refresh merotasi sesi (baris lama di-revoke, baris baru dibuat). Baris yang sudah lewat `expires_at` dihapus otomatis saat backend start dan setiap jam.

## Batasan yang Diketahui
- **Logout tidak mencabut access token yang sudah terbit**; ia berlaku sampai kedaluwarsa (default 15 menit). Hanya ganti password dan penonaktifan akun yang memotongnya seketika.
- **Satu query database per request terlindungi** (lihat [ADR-0003](../../adr/ADR-0003-access-token-database-check.md)); diukur hanya di lokal (600 request `GET /v1/me`, 25 paralel: ±460 req/s, p50 1,1 ms, p95 1,7 ms), bukan di produksi. Bila menjadi beban, cache singkat adalah langkah berikutnya, dengan jendela pencabutan sebesar TTL cache.
- **Token yang terbit pada detik yang sama dengan ganti password diterima** (`iat` beresolusi detik), supaya pengguna yang langsung login ulang tidak terkunci.
- **Deteksi replay tidak mencakup sesi yang dicabut lewat logout/ganti password**, dan tidak berlaku untuk sesi sebelum migrasi 004 (kolom `rotated_at` kosong).
- **Rate limiter berada di memori satu proses.** Dengan lebih dari satu instance backend, batasnya per instance.
