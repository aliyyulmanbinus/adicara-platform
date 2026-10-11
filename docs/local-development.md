# Menjalankan Adicara secara lokal

Panduan ini berlaku untuk anggota tim yang baru meng-clone repositori. Adicara terdiri dari frontend Astro, API Go, dan PostgreSQL. Pilih salah satu cara menjalankan seluruh stack di bawah.

## Prasyarat

- Git.
- Untuk cara Docker: Docker Engine/Desktop dengan Docker Compose.
- Untuk cara manual: Node.js 22.12+ dan npm 10+, Go 1.27+, serta PostgreSQL dengan `psql` (Compose memakai PostgreSQL 18; tes manual pernah memakai PostgreSQL 16).

Pastikan port yang dipakai belum digunakan proses lain. Perintah di bawah ditulis untuk PowerShell dari root repositori; di macOS/Linux ganti `Copy-Item` dengan `cp` dan `$env:NAMA = 'nilai'` dengan `export NAMA=nilai`.

| Cara           | Website                 | API                                               | PostgreSQL                                 |
| -------------- | ----------------------- | ------------------------------------------------- | ------------------------------------------ |
| Docker Compose | `http://localhost:8080` | Melalui `/api` pada port 8080; kesehatan `/healthz` | `localhost:5432`                           |
| Manual         | `http://127.0.0.1:4321` | `http://127.0.0.1:8080` (path `/v1/...`)          | `127.0.0.1:5432` atau port PostgreSQL Anda |

Migrasi database dijalankan **otomatis oleh API saat start** (goose, `MIGRATE_ON_START=true` secara bawaan). Tidak ada lagi berkas `*.up.sql` yang dijalankan manual dan tidak ada service migrasi terpisah di Compose.

## Pilihan A: Docker Compose

1. Dari root repositori, buat konfigurasi lokal:

   ```powershell
   Copy-Item .env.example .env
   notepad .env
   ```

2. Di `.env`, ganti `POSTGRES_PASSWORD` dan bagian password dalam `DATABASE_URL` dengan **nilai yang sama**. `DATABASE_URL` di Compose memakai host `postgres`, bukan `localhost`. Pilih password yang aman untuk URL, atau lakukan URL encoding pada karakter khusus. `JWT_SECRET` boleh dibiarkan untuk pengembangan lokal selama `APP_ENV=development`; untuk produksi wajib diganti (`openssl rand -hex 32`), karena backend menolak start dengan secret kosong, placeholder, atau kurang dari 32 karakter. File `.env` diabaikan Git; jangan commit kredensial.
3. Jalankan stack:

   ```powershell
   docker compose up --build
   ```

   Compose menyalakan PostgreSQL, lalu API (yang menjalankan migrasi saat start), frontend, dan Nginx. Pada terminal lain, periksa dengan `docker compose ps`.

4. Buka `http://localhost:8080`. Periksa API dan koneksi database melalui `http://localhost:8080/healthz`; respons `{"status":"ok"}` berarti API dapat mengakses PostgreSQL. Tidak ada `/readyz`.

Untuk menghentikan, tekan `Ctrl+C`, lalu jalankan `docker compose down` bila perlu. Volume database tetap tersimpan. Mengubah `POSTGRES_PASSWORD` di `.env` setelah volume database dibuat **tidak** otomatis mengganti password di database yang sudah ada.

> Alur Compose belum diverifikasi pada lingkungan yang menggabungkan branch `backend` ke `development` (Docker tidak tersedia); lihat [docker](devops/docker.md).

## Pilihan B: tanpa Docker

Cara ini menggunakan instalasi PostgreSQL lokal milik masing-masing anggota tim. Pastikan layanan PostgreSQL sudah berjalan. Contoh berikut memakai port `5432`; sesuaikan jika port Anda berbeda. Perintah `psql` mengasumsikan folder `bin` PostgreSQL ada di `PATH`; jika belum, panggil executable dengan path lengkap.

### 1. Buat database kosong

Masuk ke PostgreSQL menggunakan akun admin lokal Anda:

```powershell
psql -h 127.0.0.1 -p 5432 -U postgres -d postgres
```

Di prompt `psql`, jalankan SQL berikut. Ganti contoh password dengan password lokal Anda sendiri. Jika role atau database `adicara` sudah ada, gunakan yang sudah ada; jangan buat ulang.

```sql
CREATE ROLE adicara LOGIN PASSWORD 'password-lokal-anda';
CREATE DATABASE adicara OWNER adicara;
```

Keluar dari `psql` dengan `\q`. Cukup database kosong: tabel dibuat oleh API pada start pertama.

### 2. Jalankan API (terminal baru)

```powershell
cd backend
Copy-Item .env.example .env
notepad .env
```

Isi di `backend/.env`: `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`, `DB_HOST`, `DB_PORT` (atau satu `DATABASE_URL` penuh, yang menimpa `DB_*`), dan `JWT_SECRET`. Program membaca `.env` dari direktori tempat perintah dijalankan, jadi jalankan dari `backend/`. Password dengan karakter khusus URL harus di-URL-encode bila memakai `DATABASE_URL`.

```powershell
go run ./cmd/api
```

Biarkan terminal ini terbuka. Log awal menampilkan `OK 001_000_m_role.sql` sampai `OK 004_000_auth_hardening.sql`. Periksa `http://127.0.0.1:8080/healthz` sebelum lanjut. Di macOS/Linux, `make run` setara.

Perintah database manual dari `backend/`: `make migrate-up`, dan `make migrate-fresh` (**destruktif**: menghapus seluruh isi database lalu migrasi ulang). Detail: [schema-migrations](backend/migrations/schema-migrations.md).

### 3. Jalankan frontend (terminal lain)

```powershell
cd frontend
Copy-Item .env.example .env
npm ci
npm run dev -- --host 127.0.0.1
```

`frontend/.env.example` sudah berisi `API_BASE_URL=http://localhost:8080`. `npm ci` cukup dijalankan saat pertama kali menyiapkan proyek atau setelah `package-lock.json` berubah. Server Astro memanggil API dari sisi server, sehingga CORS tidak diperlukan. Proxy `/api` pada server dev membuang prefix `/api` sebelum meneruskan ke API (backend melayani `/v1/...`, bukan `/api/v1/...`; di Compose dan produksi Nginx yang melakukannya). Buka `http://127.0.0.1:4321`.

Untuk menghentikan cara manual, tekan `Ctrl+C` pada terminal API dan frontend. Database lokal tetap tersimpan di PostgreSQL.

## Mencoba akun dan template

1. Buat akun pribadi di `/auth/daftar`, lalu masuk melalui `/auth/masuk`. Setelah register, Anda diarahkan ke halaman masuk dan harus login sendiri.
2. Untuk melihat desain contoh tanpa akun atau API, buka `/template-design-udangan/pernikahan/batak-senja` pada alamat website yang sesuai dengan cara menjalankan di atas. Pratinjau katalog juga ada di `/tema/batak-senja`.
3. **Belum berfungsi:** form `/dashboard/undangan/baru` (termasuk pilihan **Batak Senja — Pernikahan**, unggah foto, dan teks kustom) serta halaman publik `/i/[slug]`. Keduanya memanggil API undangan yang dihapus dari backend baru dan menunggu modul undangan dibangun ulang; permintaan form kini mendapat `404`, dan `/i/[slug]` menampilkan `503`. Lihat [catatan merge](devops/2026-10-11-merge-backend-into-development.md).

Setiap anggota tim memiliki database lokal sendiri. Akun yang pernah dibuat di komputer pengembang tidak ikut ter-clone atau ter-push. Buat akun sendiri setelah database lokal siap.

## Jika tidak berjalan

- **`docker` tidak dikenali:** gunakan cara manual atau pasang Docker Desktop terlebih dahulu.
- **Port 5432 atau 8080 sudah dipakai:** cari layanan yang menggunakannya; sesuaikan port PostgreSQL pada konfigurasi manual bila perlu. Compose memakai port tetap `5432` dan `8080` dalam `docker-compose.yml`.
- **API keluar dengan `JWT_SECRET ...` saat start:** `APP_ENV` bukan nilai development (`development`, `dev`, `local`, `test`) sementara `JWT_SECRET` kosong, placeholder, atau pendek. Untuk lokal, biarkan `APP_ENV=development`; untuk produksi, isi `openssl rand -hex 32`.
- **API gagal terhubung ke database:** periksa `DB_*` atau `DATABASE_URL` di `backend/.env`, bahwa PostgreSQL berjalan, serta password dan port benar. Pada Compose, `DATABASE_URL` harus memakai host `postgres`.
- **Database lama (tabel `users` dengan `display_name`):** API memindahkan tabel lama ke schema `legacy_<waktu>` sebelum migrasi; data tidak dihapus. Lihat [schema-migrations](backend/migrations/schema-migrations.md#database-dengan-skema-lama).
- **`/readyz` tidak ditemukan:** memang tidak ada; gunakan `/healthz`.
- **Halaman tampil tetapi login gagal dijangkau:** pastikan API hidup pada port `8080` dan `API_BASE_URL` di `frontend/.env` menunjuk ke sana.
