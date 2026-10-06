# Sesi Autentikasi (cookie HttpOnly via server Astro)

Date: 2026-10-04

Menggantikan [Account Forms](2026-10-02-account-forms.md), yang menjelaskan backend lama (sesi cookie + CSRF + `/api/v1/...`). Backend sekarang mengembalikan JWT di body (`/v1/auth/*`, lihat [backend](../../backend/authentication/session-authentication.md)), sehingga frontend menyimpannya sendiri.

## Alur

Browser **tidak pernah** memanggil backend untuk auth. Ia hanya memanggil endpoint same-origin milik Astro; server Astro yang memanggil backend (`API_BASE_URL`) dan menyimpan token di cookie yang tidak bisa dibaca JavaScript halaman.

| Endpoint (POST, JSON) | Fungsi |
|---|---|
| `/auth/register` | Hanya membuat akun: **tidak** login dan tidak menulis cookie. Menjawab `201 {"redirect":"/auth/masuk"}`, dan form mengarahkan pengguna ke halaman masuk untuk masuk sendiri, tempat toast keberhasilan ditampilkan ([detail](2026-10-04-register-success-toast.md)). Error backend (mis. `email_taken`) tetap ditampilkan di form. |
| `/auth/login` | Menerima `identifier` (email **atau** username) dan `password`. |
| `/auth/logout` | Mencabut refresh token di backend (best effort), menghapus cookie. Selalu 204. |

Endpoint ini sengaja **tidak** di bawah `/api/`, karena nginx meneruskan `/api/` ke backend.

## Cookie

| Cookie | Isi | Umur |
|---|---|---|
| `adicara_at` | access token | sesuai `exp` token (15 menit) |
| `adicara_rt` | refresh token | sesuai `exp` token (7 hari) |

`HttpOnly`, `SameSite=Lax`, `Path=/`. Atribut `Secure` aktif jika env `COOKIE_SECURE=true` (default `true` di produksi, `false` di dev). Password tidak pernah disimpan di browser.

## Menjaga `/dashboard/**`

`src/middleware.ts` memverifikasi sesi untuk setiap rute `/dashboard`. Halaman di bawahnya **wajib** `export const prerender = false` — halaman yang di-prerender dilayani sebagai HTML statis tanpa melewati middleware. Hasil verifikasi tersedia di `Astro.locals.user` dan `Astro.locals.accessToken`.

Urutan `resolveSession` (`src/lib/auth/session.ts`):
1. Access token masih berlaku (>30 detik)? Pakai.
2. Jika tidak, tukar refresh token → pasangan baru, cookie diperbarui.
3. Panggil `/v1/me`. Jika backend menjawab 401 padahal token belum kedaluwarsa, coba refresh sekali.
4. Hasil: `ok`, `anonymous` (cookie dihapus, redirect ke `/auth/masuk`), atau `unavailable` (503, **cookie dibiarkan** — backend yang mati tidak boleh membuat pengguna keluar).

## Hal yang mudah salah

- **Rotasi refresh token.** Backend menghanguskan refresh token yang dipakai, dan mengirim ulang token yang sudah dirotasi lebih dari 10 detik kemudian mencabut semua sesi pengguna (jendela 10 detik sama dengan `REFRESH_REUSE_MS` di bawah). Beberapa request bersamaan dengan cookie kedaluwarsa yang sama (halaman + aset, dua tab) akan saling menggugurkan jika masing-masing me-refresh. `createCoalescer` (`src/lib/auth/refresh.ts`) menggabungkannya menjadi satu panggilan dan memberi hasil yang sama pada request susulan selama 10 detik. Bekerja per proses Node; jika frontend di-scale ke beberapa instance, perlu sticky session atau penyimpanan bersama.
- **Rate limit per IP.** Karena server Astro yang memanggil backend, ia meneruskan IP klien asli (`X-Real-IP` dari nginx) agar rate limit backend tidak jatuh ke satu bucket untuk semua pengguna.
- **CSRF.** Endpoint hanya menerima `Content-Type: application/json` (halaman lain tidak bisa mengirimnya lintas-situs tanpa preflight) ditambah cookie `SameSite=Lax`. Logout juga wajib mengirim JSON: POST tanpa body diperlakukan Astro sebagai form dan diperiksa `Origin`-nya, yang gagal di belakang proxy HTTPS.
- **Pesan error.** Backend mengirim kode (`email_taken`, dst.); teks Indonesia dipetakan di `src/lib/auth/messages.ts`, dan tes memastikan setiap kode error backend punya pesan.
- **Nginx.** `/auth/login` dan `/auth/register` dibatasi zona `auth_attempts` di Nginx (5/menit per IP klien, di atas limiter backend) dan menjawab `429` dengan body `{"error":{"code":"rate_limited",...}}`; `X-Real-IP` yang diteruskan sudah berisi IP klien sebenarnya ([Nginx](../../devops/nginx.md#ip-klien)).
- **Access token diperiksa ke database oleh backend.** Ganti password dan akun nonaktif membuat access token lama `401` seketika (frontend lalu mencoba refresh, gagal, dan mengarahkan ke login); logout hanya mencabut refresh token, jadi access token yang sudah terbit tetap valid sampai kedaluwarsa (maks 15 menit). Backend mengembalikan `401`, bukan `403`, untuk akun nonaktif justru agar alur ini berakhir di form login.

## Verifikasi

Register tidak lagi login otomatis (perubahan 2026-10-04): diuji pada build produksi (`node dist/server/entry.mjs`) di depan backend dan PostgreSQL 16 sementara. `POST /auth/register` menjawab `201 {"redirect":"/auth/masuk"}` tanpa header `Set-Cookie`; `/dashboard` tanpa cookie diarahkan ke `/auth/masuk`; login setelahnya menulis kedua cookie dan `/dashboard` menjawab `200`; pendaftaran ganda tetap `409 email_taken` dan password lemah `400`. Tes sumber otomatis (`auth-wiring`) menjaga agar cabang register tidak memanggil login atau menulis cookie; **belum** dicoba di browser (skrip form tidak berubah dan mengikuti `redirect` dari respons).

Diuji pada 2026-10-04 dengan PostgreSQL 16 lokal: `astro dev` + API Go, dan build produksi (`node dist/server/entry.mjs`) lewat curl; alur form di Chrome (daftar, dashboard, reload, refresh token otomatis, keluar, login dengan email dan username, pesan error, batas input). Tes otomatis: `npm test` (unit untuk token, penggabungan refresh, pemetaan pesan, dan pemeriksaan wiring). **Belum** diuji: Nginx, Docker, dan lebih dari satu instance frontend.

## Belum tersedia

Modul undangan di backend belum ada. Dashboard memanggil `/v1/invitations`; jawaban 404 diperlakukan sebagai daftar kosong. Form "Buat Undangan" serta tombol terbit/hapus masih memakai API lama dan belum berfungsi.
