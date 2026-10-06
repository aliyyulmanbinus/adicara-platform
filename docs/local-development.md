# Menjalankan Adicara secara lokal

Panduan ini berlaku untuk anggota tim yang baru meng-clone repositori. Adicara terdiri dari frontend Astro, API Go, dan PostgreSQL. Pilih salah satu cara menjalankan seluruh stack di bawah.

## Prasyarat

- Git.
- Untuk cara Docker: Docker Engine/Desktop dengan Docker Compose.
- Untuk cara manual: Node.js 22.12+ dan npm 10+, Go 1.27+, serta PostgreSQL dengan `psql` (PostgreSQL 17 pernah dipakai secara lokal; Compose memakai PostgreSQL 18).

Gunakan PowerShell dari root repositori. Pastikan port yang dipakai belum digunakan proses lain.

| Cara           | Website                 | API                                         | PostgreSQL                                 |
| -------------- | ----------------------- | ------------------------------------------- | ------------------------------------------ |
| Docker Compose | `http://localhost:8080` | Melalui `/api` dan `/readyz` pada port 8080 | `localhost:5432`                           |
| Manual         | `http://127.0.0.1:4321` | `http://127.0.0.1:8080`                     | `127.0.0.1:5432` atau port PostgreSQL Anda |

## Pilihan A: Docker Compose

1. Dari root repositori, buat konfigurasi lokal:

   ```powershell
   Copy-Item .env.example .env
   notepad .env
   ```

2. Di `.env`, ganti `POSTGRES_PASSWORD` dan bagian password dalam `DATABASE_URL` dengan **nilai yang sama**. `DATABASE_URL` di Compose memakai host `postgres`, bukan `localhost`. Pilih password yang aman untuk URL, atau lakukan URL encoding pada karakter khusus. File `.env` diabaikan Git; jangan commit kredensial.
3. Jalankan stack:

   ```powershell
   docker compose up --build
   ```

   Compose menyalakan PostgreSQL, menjalankan migrasi SQL, lalu menyalakan API, frontend, dan Nginx. Pada terminal lain, periksa dengan `docker compose ps`.

4. Buka `http://localhost:8080`. Periksa koneksi database melalui `http://localhost:8080/readyz`; respons `{"status":"ready"}` berarti API dapat mengakses PostgreSQL.

Untuk menghentikan, tekan `Ctrl+C`, lalu jalankan `docker compose down` bila perlu. Volume database tetap tersimpan. Mengubah `POSTGRES_PASSWORD` di `.env` setelah volume database dibuat **tidak** otomatis mengganti password di database yang sudah ada.

## Pilihan B: Windows tanpa Docker

Cara ini menggunakan instalasi PostgreSQL lokal milik masing-masing anggota tim. Pastikan layanan PostgreSQL sudah berjalan. Contoh berikut memakai port PostgreSQL `5432`; sesuaikan semua perintah jika port Anda berbeda. Perintah `psql` mengasumsikan folder `bin` PostgreSQL ada di `PATH`. Jika belum, panggil executable dengan path lengkap, misalnya `C:\Program Files\PostgreSQL\17\bin\psql.exe`.

### 1. Buat database lokal

Masuk ke PostgreSQL menggunakan akun admin lokal Anda:

```powershell
psql -h 127.0.0.1 -p 5432 -U postgres -d postgres
```

Di prompt `psql`, jalankan SQL berikut. Ganti contoh password dengan password lokal Anda sendiri. Jika role atau database `adicara` sudah ada, gunakan yang sudah ada; jangan buat ulang.

```sql
CREATE ROLE adicara LOGIN PASSWORD 'password-lokal-anda';
CREATE DATABASE adicara OWNER adicara;
```

Keluar dari `psql` dengan `\q`.

### 2. Jalankan migrasi pada database baru

Dari root repositori, jalankan berkas `*.up.sql` **sekali**, sesuai urutan nama. Migrasi ini tidak dirancang untuk dijalankan ulang pada database yang sudah berisi skema tersebut.

```powershell
$env:PGPASSWORD = 'password-lokal-anda'
try {
    Get-ChildItem .\backend\migrations\*.up.sql | Sort-Object Name | ForEach-Object {
        psql -X -v ON_ERROR_STOP=1 -h 127.0.0.1 -p 5432 -U adicara -d adicara -f $_.FullName
        if ($LASTEXITCODE -ne 0) { throw "Migrasi gagal: $($_.Name)" }
    }
} finally {
    Remove-Item Env:PGPASSWORD
}
```

### 3. Jalankan API (terminal PowerShell baru)

```powershell
cd backend
$env:DATABASE_URL = 'postgres://adicara:password-lokal-anda@127.0.0.1:5432/adicara?sslmode=disable'
$env:PORT = '8080'
$env:COOKIE_SECURE = 'false'
go run ./cmd/api
```

Ganti password dan port sesuai database Anda. Jika password mengandung karakter khusus URL, lakukan URL encoding pada bagian password di `DATABASE_URL`. Program Go membaca `DATABASE_URL` dari lingkungan proses; menyalin `backend/.env.example` menjadi `.env` saja tidak akan memuat nilainya secara otomatis. Biarkan terminal ini terbuka. Periksa `http://127.0.0.1:8080/readyz` sebelum lanjut.

### 4. Jalankan frontend (terminal PowerShell lain)

```powershell
cd frontend
npm ci
$env:API_BASE_URL = 'http://127.0.0.1:8080'
$env:PUBLIC_SITE_URL = 'http://127.0.0.1:4321'
npm run dev -- --host 127.0.0.1
```

`npm ci` cukup dijalankan saat pertama kali menyiapkan proyek atau setelah `package-lock.json` berubah. Frontend development mem-proxy permintaan `/api` ke port `8080`, sehingga form login bekerja pada alamat frontend. Biarkan terminal ini terbuka, lalu buka `http://127.0.0.1:4321`.

Untuk menghentikan cara manual, tekan `Ctrl+C` pada terminal API dan frontend. Database lokal tetap tersimpan di PostgreSQL.

## Mencoba akun dan template

1. Buat akun pribadi di `/auth/daftar`, lalu masuk melalui `/auth/masuk`.
2. Buka `/dashboard/undangan/baru` dan pilih **Batak Senja — Pernikahan** pada pilihan tema.
3. Untuk melihat desain contoh tanpa akun atau API, buka `/template-design-udangan/pernikahan/batak-senja` pada alamat website yang sesuai dengan cara menjalankan di atas. Pratinjau katalog juga ada di `/tema/batak-senja`.

Setiap anggota tim memiliki database lokal sendiri. Akun demo dan undangan yang pernah dibuat di komputer pengembang tidak ikut ter-clone atau ter-push, karena data lokal di `.data/` diabaikan Git. Buat akun sendiri setelah database lokal siap.

## Jika tidak berjalan

- **`docker` tidak dikenali:** gunakan cara manual atau pasang Docker Desktop terlebih dahulu.
- **Port 5432 atau 8080 sudah dipakai:** cari layanan yang menggunakannya; sesuaikan port PostgreSQL pada semua perintah manual bila perlu. Compose memakai port tetap `5432` dan `8080` dalam `docker-compose.yml`.
- **API menampilkan `DATABASE_URL is required`:** set variabel itu di terminal yang sama sebelum `go run`.
- **`/readyz` gagal:** periksa apakah PostgreSQL berjalan, password dan port benar, serta migrasi sudah dijalankan.
- **Halaman tampil tetapi login gagal dijangkau:** pastikan API hidup pada port `8080` dan frontend dijalankan melalui alamat yang tercantum di atas.
