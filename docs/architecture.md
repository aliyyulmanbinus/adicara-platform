# Arsitektur Adicara

## Ringkasan

Adicara memakai monorepo dengan tiga batas utama:

```text
Browser → Nginx → Astro
              └→ Go API → PostgreSQL
```

Nginx menyediakan origin tunggal. Permintaan `/api/*`, `/healthz`, dan `/readyz` diarahkan ke Go; permintaan lain diarahkan ke Astro.

## Frontend

Astro memakai mode static-first. Homepage dan halaman marketing diprerender. Route `/i/[slug]` memakai on-demand rendering dan mengambil undangan yang sudah dipublikasikan dari Go API sebelum menghasilkan HTML. JavaScript browser bukan syarat untuk membaca detail inti.

Template undangan menerima kontrak `InvitationData`. Registry menghubungkan `template_key` ke komponen presentasi. Bagian umum seperti cover dan jadwal tidak menyimpan data sendiri.

## Backend

Alur dependensi backend:

```text
HTTP handler → service → repository → PostgreSQL
```

`net/http` menangani routing. `pgxpool` mengelola koneksi. Handler tidak mengandung SQL dan repository tidak menentukan status HTTP.

## Batas fase

Autentikasi session, dashboard awal, CRUD undangan, tamu, RSVP, serta input teks/foto untuk template Batak Senja sudah tersedia. Foto disimpan di PostgreSQL dan hanya dapat diakses publik setelah undangan diterbitkan. Wishes, transaksi hadiah, editor visual lanjutan, reporting lengkap, serta pembayaran belum tersedia. Nginx menerapkan rate limit umum API dan limit lebih ketat untuk autentikasi serta RSVP publik.

## Keputusan

- [ADR-0001: monorepo Astro, Go, PostgreSQL dengan same-origin](adr/ADR-0001-foundation-architecture.md)
- [ADR-0002: pemisahan data dan presentasi template](adr/ADR-0002-template-data-separation.md)
