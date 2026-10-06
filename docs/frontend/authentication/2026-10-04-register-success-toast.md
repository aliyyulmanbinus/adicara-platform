# Toast setelah register dan form daftar tanpa petunjuk username

## Date

2026-10-04

## Status

Implemented

## Area

Authentication / form daftar dan halaman masuk

## Summary

Setelah akun berhasil dibuat, pengguna sudah diarahkan ke halaman masuk (lihat [sesi autentikasi](2026-10-04-session-cookies.md)); sekarang halaman itu menampilkan toast "Akun berhasil dibuat. Silakan masuk." Teks petunjuk "3–30 karakter: huruf, angka, titik, atau garis bawah." di bawah kolom username dihapus.

## Objective

Halaman masuk yang muncul tanpa penjelasan membuat pengguna baru bertanya-tanya apakah akunnya sudah jadi. Petunjuk username dihapus atas permintaan produk.

## Previous Behavior

- Setelah register, halaman masuk tampil tanpa pesan apa pun.
- Form daftar menampilkan petunjuk username, dirujuk lewat `aria-describedby="username-help"`.

## New Behavior

- Saat `POST /auth/register` berhasil, form daftar menyimpan satu penanda sekali pakai di `sessionStorage` (kunci `adicara:toast`, nilai `registered`; hanya kunci yang dikenal, bukan teks) lalu mengikuti `redirect` ke `/auth/masuk`.
- Form masuk membaca dan langsung menghapus penanda itu saat halaman dimuat, lalu menampilkan toast. Memuat ulang halaman, atau membuka halaman masuk langsung, tidak menampilkan toast.
- Toast menghilang sendiri setelah 6 detik, atau saat tombol × ditekan.
- Register yang gagal (mis. email sudah terdaftar) tetap di halaman daftar dengan pesan error dan **tidak** meninggalkan penanda.
- Bila `sessionStorage` tidak tersedia, redirect tetap terjadi, hanya tanpa toast, tanpa error.
- Validasi bawaan username (`minlength=3`, `maxlength=30`, `pattern`) tetap ada; pesan `invalid_username` dari backend tetap menjelaskan aturannya bila dilanggar.

## Visual Changes

- Toast: kartu dengan ikon ✓, teks, dan tombol ×; latar `--color-surface`, garis kiri `--color-accent`, bayangan `--shadow-soft`, sudut `--radius-md`. Muncul dengan animasi geser-pudar 0,25 detik.
- Diletakkan `position: fixed` di bawah header sticky (`4.25rem + --space-3` dari atas), di tengah, lebar maksimum 28rem. Uji awal menunjukkan toast di posisi paling atas menutupi logo, tombol Menu, dan navigasi; karena itu dipindah ke bawah header.
- Teks petunjuk username dan `aria-describedby`-nya dihapus; petunjuk kata sandi tidak berubah.

## Responsive Behavior

Diperiksa di Chrome pada 360, 390, dan 1280 piksel: toast seluruhnya di dalam viewport, di bawah header, tanpa scroll horizontal (lebar 328 px pada 360, 347 px pada 390 dan 1280). Belum diperiksa pada 375, 414, 430, dan tablet.

## Components Affected

- `frontend/src/components/auth/AuthForm.astro` (satu-satunya file kode yang berubah; dipakai oleh `daftar.astro` dan `masuk.astro`)

## Files Changed

- `frontend/src/components/auth/AuthForm.astro`
- `frontend/tests/auth-wiring.test.mjs` (tes sumber baru)

## API Dependencies

None. Memakai respons `201 {"redirect":"/auth/masuk"}` yang sudah ada.

## SEO Impact

No direct SEO impact. Kedua halaman tetap `noindex`.

## Accessibility Impact

- Wadah toast adalah live region (`role="status"`, `aria-live="polite"`) yang sudah ada di DOM sebelum pesan ditambahkan, sehingga pembaca layar membacakannya. Ikon ✓ disembunyikan dari teknologi bantu (`aria-hidden`); tombol × berlabel "Tutup pemberitahuan".
- Tombol × dapat difokus dan menampilkan cincin fokus (outline solid 3 px). Karena toast diletakkan setelah form di DOM, urutan Tab-nya berada setelah kolom form; pesan tetap diumumkan lewat live region, dan toast hilang sendiri setelah 6 detik.
- `prefers-reduced-motion: reduce` mematikan animasi toast.
- Keberhasilan tidak disampaikan lewat warna saja: ada teks dan ikon ✓.

## Performance Impact

Sekitar 40 baris skrip klien tambahan di komponen yang sama, tanpa dependensi dan tanpa hidrasi baru. Tidak diukur.

## Validation

Dijalankan 2026-10-04:

- `npm run format:check`, `npm run check` (0 error, 0 warning, 0 hint), `npm test` (31 tes lolos).
- Mematikan penyimpanan penanda, atau mengembalikan petunjuk username, membuat tes sumber baru gagal.
- Chrome 154 (headless, dikendalikan `puppeteer-core`) terhadap build produksi di depan backend dan PostgreSQL sementara, 24 pemeriksaan lolos: petunjuk username hilang dan tidak ada `aria-describedby` menggantung; setelah register browser ada di `/auth/masuk` dengan toast berisi pesan yang benar; toast tidak menutupi header (390, 360, 1280 px); tombol × menghapus toast; toast masih ada pada detik ke-5 dan hilang pada detik ke-7; muat ulang tidak menampilkannya lagi; membuka halaman masuk langsung tidak menampilkannya; register ganda tetap di halaman daftar dan halaman masuk sesudahnya tanpa toast; dengan `sessionStorage` diblokir redirect tetap terjadi tanpa toast dan tanpa error; `prefers-reduced-motion` mematikan animasi; tidak ada error konsol.

**Belum diuji:** Safari dan Firefox, perangkat sentuh nyata, pembaca layar (hanya atribut ARIA yang diperiksa), dan lebar 375, 414, 430, serta tablet.

## Screenshots

Not provided.

## Notes

Pesan toast hanya satu ("registered"). Bila kelak butuh toast lain (mis. setelah ganti password), tambahkan kunci baru pada tabel `TOASTS` di skrip; kunci yang tidak dikenal diabaikan.
