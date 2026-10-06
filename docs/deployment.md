# Deployment

## Topologi

Produksi memakai empat service: Nginx, frontend Astro, backend Go, dan PostgreSQL. Tidak ada service migrasi terpisah: backend menjalankan migrasi sendiri saat start (`MIGRATE_ON_START`, default `true`) setelah PostgreSQL healthy, dan frontend baru start setelah `/healthz` backend sehat. PostgreSQL hanya terhubung ke network privat Compose.

Nginx pada `deploy/docker-compose.prod.yml` bind ke `127.0.0.1:8080`; HTTPS publik diasumsikan diterminasi oleh reverse proxy/host Sumopod di depan stack. Sesuaikan binding hanya bila model jaringan VPS berbeda.

## Persiapan VPS

1. Buat pengguna deployment non-root.
2. Checkout repository ke `/opt/adicara`.
3. Buat file environment produksi di VPS (`/opt/adicara/.env`); jangan commit. Variabel yang dipakai ada di [Environment](#environment).
4. Pastikan Docker dapat menarik package GHCR milik repository.
5. Atur reverse proxy HTTPS host ke `127.0.0.1:8080`.

File migrasi SQL sudah ada di dalam image backend; tidak ada yang perlu disalin ke VPS.

## Environment

| Variabel | Dibaca oleh | Catatan |
|---|---|---|
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | postgres | |
| `DATABASE_URL` | backend | DSN lengkap; host database adalah `postgres`. |
| `PUBLIC_SITE_URL` | frontend | Origin HTTPS publik; dipakai saat build dan runtime. |
| `JWT_SECRET` | **backend** | **Wajib.** Acak, minimal 32 karakter: `openssl rand -hex 32`. Compose menolak start tanpa nilai ini, dan backend menolak nilai kosong, bawaan, placeholder (`change-me…`, `your-secret…`), atau pendek. Mengganti nilainya membuat semua pengguna login ulang. |
| `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `AUTH_RATE_LIMIT`, `TRUSTED_PROXIES`, `MIGRATE_ON_START` | backend | Opsional. Kosong/tidak diisi = default backend ([backend/README.md](../backend/README.md#environment)). |
| `CORS_ORIGINS` | backend | Opsional. Default: nilai `PUBLIC_SITE_URL`. |
| `COOKIE_SECURE` | **frontend** | Atribut `Secure` pada cookie sesi. Default `true` di produksi; atur `false` hanya untuk uji lewat HTTP. |
| `FRONTEND_IMAGE`, `BACKEND_IMAGE` | Compose | Di-export workflow ke tag commit. |
| `NGINX_BIND_ADDRESS`, `NGINX_HTTP_PORT` | Compose | Default `127.0.0.1` dan `8080`. |
| `APP_ENV` | backend | Compose produksi mengunci backend ke `production` (nilai di `.env` tidak berpengaruh pada backend). Nilai selain development mengaktifkan validasi `JWT_SECRET`. |
| `LOG_LEVEL` | — | Diteruskan Compose tetapi tidak dibaca aplikasi saat ini. |

## Batasan yang diketahui

- **Deploy pertama di atas database versi lama** (tabel `users` dengan `display_name`, `invitations`, ...): backend memindahkan tabel lama ke schema `legacy_<waktu>` lalu membuat skema baru; tidak ada data yang dihapus dan deploy tidak gagal. Pengguna lama harus mendaftar ulang, dan versi aplikasi lama tidak bisa lagi berjalan di database itu. Detail: [Database dengan Skema Lama](backend/migrations/schema-migrations.md#database-dengan-skema-lama). Rollback otomatis (di bawah) tidak mengembalikan skema.
- **Akses lewat HTTP (tanpa HTTPS)**: cookie sesi memakai atribut `Secure` kecuali `COOKIE_SECURE=false` diisi di `/opt/adicara/.env`. Tanpa itu, login tampak berhasil tetapi browser langsung membuang cookie dan Anda kembali ke halaman masuk.
- **Rate limit per IP klien bergantung pada header `X-Forwarded-For` dari proxy host.** Nginx memercayainya hanya bila koneksi datang dari alamat privat/loopback (`set_real_ip_from`). Bila proxy host tidak mengirim `X-Forwarded-For`, atau berada di alamat publik, semua pengguna berbagi satu bucket (default backend 10 percobaan/menit untuk register, login, refresh, dan logout sekaligus; 5/menit di Nginx untuk login dan register). Belum diuji di belakang proxy Sumopod yang sebenarnya; lihat [Nginx](devops/nginx.md#ip-klien).
- **Deploy pertama setelah pengerasan auth (2026-10-04) membutuhkan `JWT_SECRET` di `/opt/adicara/.env`.** Tanpanya deploy berhenti di langkah `compose config` (tidak ada yang diubah; versi lama tetap melayani). Tambahkan di VPS: `echo "JWT_SECRET=$(openssl rand -hex 32)" >> /opt/adicara/.env`. Versi sebelumnya berjalan dengan secret bawaan yang diketahui publik, jadi semua token yang beredar tidak berlaku setelah ini dan pengguna login ulang satu kali. Migrasi 004 ikut berjalan; aman untuk rollback ([detail](backend/migrations/2026-10-04-auth-hardening-columns.md)).
- **Cadangkan database sebelum deploy pertama backend yang ditulis ulang.** Produksi masih menjalankan backend lama; deploy pertama memindahkan skema lama ke `legacy_…` lalu menerapkan migrasi 1–4. Rollback otomatis hanya mengganti image dan tidak memulihkan skema, sehingga aplikasi lama tidak bisa berjalan lagi di database itu. Jalur ini sudah disimulasikan dengan skema lama asli dari riwayat git ([detail](backend/authentication/2026-10-04-auth-hardening.md#tests)), bukan dijalankan di produksi. Prosedur backup: [backup-restore](devops/backup-restore.md).

## GitHub Actions

Workflow deploy membutuhkan:

- `VPS_HOST`
- `VPS_USER`
- `VPS_SSH_KEY`
- `VPS_PORT`
- repository variable `PUBLIC_SITE_URL`

## Update otomatis

Setiap push ke `main` menjalankan: CI → build dan push image (tag SHA) → deploy ke VPS. Deploy hanya berjalan bila CI lolos. Langkah di VPS (rincian dan penanganan gagal: [GitHub Actions](devops/github-actions.md#cd)):

1. Menyamakan checkout `/opt/adicara` dengan commit yang di-deploy, supaya perubahan `docker-compose.prod.yml` dan konfigurasi Nginx ikut terpasang. Berhenti dengan pesan jelas bila file yang di-track diedit di server.
2. Menarik image aplikasi (bukan postgres/nginx). Gagal di sini tidak mengubah apa pun yang sedang berjalan.
3. `docker compose up -d`; backend menjalankan migrasi sendiri. Config Nginx diterapkan ulang (`nginx -t` lalu reload).
4. Memeriksa `/healthz` (backend lewat Nginx) dan `/` (frontend). Hanya bila keduanya menjawab, deploy dianggap sukses dan SHA-nya dicatat sebagai versi terakhir yang baik.
5. Bila gagal: log service ditampilkan dan, jika ada versi baik sebelumnya, stack dikembalikan ke versi itu. Job tetap berstatus gagal.

Prasyarat tambahan di VPS: `git` harus bisa `fetch` dari GitHub (repository saat ini publik; bila dijadikan privat, beri kredensial read-only), dan `/opt/adicara` dikelola oleh deploy, jadi jangan mengedit file yang di-track di sana (`.env` aman: tidak di-track).

## Rollback

- **Otomatis**, bila deploy gagal dan sudah pernah ada deploy sukses sebelumnya (dicatat di `/opt/adicara/.deploy-last-good`). Deploy pertama setelah fitur ini ada tidak punya versi pembanding.
- **Manual**: tetapkan `FRONTEND_IMAGE` dan `BACKEND_IMAGE` ke SHA image sebelumnya, lalu jalankan ulang `docker compose up -d`.

Rollback hanya mengganti image. Backend hanya menjalankan migrasi *up* saat start: memundurkan image tidak memundurkan skema, jadi migrasi harus kompatibel dengan versi aplikasi sebelumnya, dan migrasi destruktif tidak boleh digabung dengan deployment yang memerlukan rollback sederhana.

Lihat [panduan VPS](devops/vps-deployment.md) dan [backup/restore](devops/backup-restore.md).

