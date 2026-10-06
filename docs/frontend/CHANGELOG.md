# Frontend Changelog

## 2026-10-04

### Authentication

- Auth disesuaikan dengan backend baru (JWT di body, `/v1/auth/*`). Server Astro menjadi perantara dan menyimpan token di cookie HttpOnly; endpoint `/auth/login`, `/auth/register`, `/auth/logout`; middleware menjaga `/dashboard/**` dan me-refresh token otomatis. Documentation: `authentication/2026-10-04-session-cookies.md`.
- Halaman masuk menampilkan toast "Akun berhasil dibuat. Silakan masuk." setelah register berhasil, dan petunjuk username "3–30 karakter…" dihapus dari form daftar. Documentation: `authentication/2026-10-04-register-success-toast.md`.
- Register tidak lagi login otomatis: setelah akun dibuat, pengguna diarahkan ke `/auth/masuk` dan masuk sendiri (tanpa cookie sesi dari endpoint register). Documentation: `authentication/2026-10-04-session-cookies.md`.
- Form daftar memakai `username` (3–30 karakter) dan password 8–72 karakter sesuai aturan backend; form masuk menerima email atau username. Pesan error dipetakan dari kode error backend.
- Bug yang diperbaiki: login tidak pernah membuat sesi (token diabaikan), dashboard memanggil `/api/v1/me` yang tidak ada, tombol Keluar tidak berfungsi (body `refresh_token` tidak dikirim), password 73–128 karakter menghasilkan HTTP 500, halaman editor tidak dijaga karena di-prerender.
- Belum berfungsi: pembuatan/pengelolaan undangan (menunggu modul undangan di backend).

## 2026-10-02

### Landing Page

- Added the initial mobile-first Adicara marketing homepage and SEO foundation. Documentation: `landing-page/2026-10-02-initial-homepage.md`.

### Invitation Templates

- Added the data-driven Editorial Ivory theme and server-rendered public invitation route. Documentation: `invitation-templates/2026-10-02-editorial-ivory-template.md`.
- Added Botanical Modern and Monochrome Luxe, a shared section renderer, and statically generated theme detail pages. Documentation: `theme-catalog/2026-10-02-theme-catalog-and-registry.md`.

### Marketing Routes

- Added the public theme catalog, features, transparent pricing status, inspiration index, and first original guide. Documentation: `seo/2026-10-02-public-marketing-routes.md`.

### Authentication and Dashboard

- Replaced the authentication placeholder with registration and login forms connected to the same-origin API. Documentation: `authentication/2026-10-02-account-forms.md`.
- Added an owner dashboard, structured invitation creation form, and publish/unpublish/delete controls. Documentation: `dashboard/2026-10-02-invitation-management.md`.
