# GitHub Actions

## CI

`.github/workflows/ci.yml` berjalan pada pull request dan push ke `main`, serta dapat dipanggil workflow lain. Job memeriksa:

- format, Astro type-check, source tests, dan build frontend;
- gofmt, vet, test, dan build backend;
- lint OpenAPI;
- build kedua Docker image.

## CD

`.github/workflows/deploy.yml` memanggil CI, membangun image frontend/backend, memberi tag Git SHA dan `main`, lalu push ke GHCR. Deployment melalui SSH menggunakan environment `production`, menjalankan Compose, dan memeriksa `/readyz`.

Required secrets: `VPS_HOST`, `VPS_USER`, dan `VPS_SSH_KEY`. Optional secret: `VPS_PORT` (default `22`). Required repository variable untuk URL produksi: `PUBLIC_SITE_URL`.

### Konfigurasi SSH

Di repository GitHub, buka Settings > Secrets and variables > Actions > New repository secret, lalu isi:

- `VPS_HOST`: IP publik atau hostname VPS saja, tanpa `https://`, path, atau port.
- `VPS_USER`: username SSH di VPS.
- `VPS_SSH_KEY`: private key SSH lengkap, termasuk baris BEGIN dan END. Public key pasangannya harus tersedia di `authorized_keys` milik user VPS.
- `VPS_PORT`: port SSH jika bukan `22`.

Sebagai alternatif, simpan secret di Settings > Environments > production > Environment secrets. Job deploy memakai environment `production`; secret yang disimpan hanya di environment lain tidak tersedia untuk job ini. Secret tidak sama dengan Actions variable: workflow SSH membaca `secrets.VPS_HOST`, bukan `vars.VPS_HOST`.

Error `missing server host` berarti action menerima host kosong, sebelum mencoba koneksi SSH. Periksa nama secret, scope repository/environment, dan nilainya. Step `Validate SSH configuration` memeriksa kelengkapan tanpa mencetak nilai secret. Setelah mengisi secret, jalankan ulang job deploy yang gagal. Bila konfigurasi workflow berubah, commit dan push perubahan lalu jalankan workflow untuk commit baru.

Isi `PUBLIC_SITE_URL` di Settings > Secrets and variables > Actions > Variables dengan origin HTTPS website produksi, misalnya `https://example.com` (ganti dengan domain sebenarnya). Variable ini digunakan saat build untuk canonical URL, sitemap, dan robots.txt. Bila belum diisi, workflow memakai `http://localhost:8080` agar build tidak gagal karena URL kosong; URL cadangan ini bukan URL kanonis produksi. Setelah mengganti variable, jalankan workflow kembali untuk membangun ulang image.

## Catatan keamanan

Gunakan deployment user non-root dengan akses minimum. Jangan memasukkan `.env` produksi atau private key ke repository. Pertimbangkan PAT read-only di VPS bila package privat tidak dapat ditarik dengan kredensial workflow sementara.
