# VPS Deployment

## Struktur yang diharapkan

Repository berada di `/opt/adicara`; file environment produksi dikelola langsung di VPS. Reverse proxy host mengirim trafik HTTPS ke `127.0.0.1:8080`.

## Persiapan awal

1. Install Docker Engine dan Compose plugin.
2. Buat user deployment non-root dan batasi SSH.
3. Checkout repository ke `/opt/adicara`.
4. Buat `.env` dengan `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `DATABASE_URL`, `JWT_SECRET`, `PUBLIC_SITE_URL`, `FRONTEND_IMAGE`, dan `BACKEND_IMAGE` (daftar lengkap variabel dan batasannya: [deployment](../deployment.md#environment)).
5. Login ke GHCR dengan token yang hanya memiliki izin package read.
6. Jalankan `docker compose -f deploy/docker-compose.prod.yml up -d`.
7. Periksa `curl --fail http://127.0.0.1:8080/healthz`.

## Setup awal Ubuntu di SumoPod

Login menggunakan SSH sebagai user `ubuntu`. Instal Docker Engine dan Compose plugin dari [repository resmi Docker untuk Ubuntu](https://docs.docker.com/engine/install/ubuntu/), lalu aktifkan service:

```bash
sudo systemctl enable --now docker
sudo usermod -aG docker ubuntu
```

Logout lalu login kembali agar keanggotaan grup berlaku. Verifikasi `docker info` dan `docker compose version` tanpa `sudo`. Grup Docker memberikan akses setara root; gunakan hanya untuk user deployment tepercaya.

Siapkan checkout baru pada VPS yang belum memiliki `/opt/adicara`:

```bash
sudo apt install -y git
sudo install -d -o ubuntu -g ubuntu /opt/adicara
git clone https://github.com/aliyyulmanbinus/adicara-platform.git /opt/adicara
cd /opt/adicara
umask 077
test -e .env || cp .env.example .env
chmod 600 .env
nano .env
```

Jika repository privat, autentikasikan Git menggunakan kredensial read-only; key GitHub Actions ke VPS tidak otomatis memberikan akses VPS ke GitHub.

Isi `JWT_SECRET` dengan nilai acak (`openssl rand -hex 32`); nilai placeholder dari `.env.example` ditolak backend, dan Compose tidak mau start bila variabel ini tidak ada. Isi `POSTGRES_DB` dan `POSTGRES_USER`, buat password database acak (misalnya dengan `openssl rand -hex 24`), kemudian gunakan password yang sama dalam `POSTGRES_PASSWORD` dan `DATABASE_URL`. Host database adalah `postgres`, bukan IP publik VPS. Tetapkan `PUBLIC_SITE_URL` ke domain HTTPS sebenarnya, sama dengan Actions variable. `FRONTEND_IMAGE` dan `BACKEND_IMAGE` di-export workflow ke tag commit yang diterbitkan.

Workflow memakai `--env-file /opt/adicara/.env` secara eksplisit. Setelah file siap, jalankan Deploy (atau push ke `main`). Pemeriksaan prasyarat menghentikan deployment jika Docker, Compose, checkout, atau `.env` belum siap. Workflow tidak melakukan provisioning VPS secara otomatis, tetapi **memperbarui checkout `/opt/adicara` ke commit yang di-deploy** pada setiap deploy; jangan mengedit file yang di-track di sana (deploy akan berhenti dan memberi tahu bila ada). Lihat [Update otomatis](../deployment.md#update-otomatis).

Port `8080` hanya bind ke loopback; setelah health check berhasil, siapkan reverse proxy HTTPS di depan `127.0.0.1:8080` agar website bisa diakses publik. Jangan memakai `docker compose down -v` untuk memperbaiki error karena dapat menghapus data PostgreSQL.

## Akses sementara melalui IP publik (HTTP)

Untuk VPS tanpa domain, tambahkan pengaturan berikut ke `/opt/adicara/.env`, dengan IP publik VPS sebenarnya:

```dotenv
PUBLIC_SITE_URL=http://43.173.6.155
APP_ENV=production
COOKIE_SECURE=false
NGINX_BIND_ADDRESS=0.0.0.0
NGINX_HTTP_PORT=80
```

Isi Actions variable `PUBLIC_SITE_URL` dengan URL yang sama. Commit/push perubahan proyek; workflow memperbarui Compose di VPS sendiri, tidak perlu `git pull` manual. Jangan membagikan `.env` atau private key.

Izinkan inbound TCP port `80` dari `0.0.0.0/0` melalui tab Firewall VPS di SumoPod. Pertahankan rule SSH yang sudah digunakan. Jika UFW aktif, tambahkan `sudo ufw allow 80/tcp`; tidak perlu mengaktifkan atau mereset firewall untuk langkah ini. Docker dapat melewati beberapa rule UFW untuk port yang dipublikasikan; gunakan firewall cloud untuk pembatasan akses publik.

Setelah deploy, periksa `curl --fail http://127.0.0.1/healthz`, lalu buka `http://43.173.6.155` di browser. Workflow health check mengambil port Nginx yang dipublikasikan secara otomatis dan memeriksa `/healthz` serta `/`. HTTP tidak mengenkripsi password dan cookie sesi; gunakan untuk pengujian sementara. `COOKIE_SECURE` dibaca oleh container **frontend** (yang menulis cookie sesi); dengan `false` cookie dikirim lewat HTTP biasa. Setelah domain dan HTTPS tersedia, aktifkan kembali `COOKIE_SECURE=true` dan gunakan URL HTTPS.

## Pemeriksaan service

Periksa log backend dengan `docker compose logs backend` (format teks Gin dan `log`, bukan JSON). Jangan menyalin log yang mungkin berisi data pribadi ke kanal publik.

Deployment ini belum dijalankan pada VPS dari lingkungan implementasi saat ini.
