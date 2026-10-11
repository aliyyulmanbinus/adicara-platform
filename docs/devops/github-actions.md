# GitHub Actions

## CI

`.github/workflows/ci.yml` berjalan pada pull request dan push ke `main`, serta dapat dipanggil workflow lain. Job memeriksa:

- format, Astro type-check, tes (unit dan source), dan build frontend. Skrip tes memakai `node --experimental-strip-types`, tersedia di Node 22.6+ (CI memakai Node 24);
- gofmt, vet, test, dan build backend;
- lint OpenAPI (Redocly) pada `backend/docs/openapi.yaml`, kontrak yang berlaku. `contracts/openapi.yaml` adalah backup versi lama dan tidak di-lint;
- build kedua Docker image.

Pemeriksaan frontend dan backend di atas sudah dijalankan secara lokal dan lolos. Lint OpenAPI sudah dijalankan secara lokal (`npx @redocly/cli lint backend/docs/openapi.yaml`: valid, exit code 0, 3 warning: lisensi, server `localhost`, `/healthz` tanpa respons 4XX) tetapi belum di GitHub Actions. Build image belum pernah dijalankan dari lingkungan implementasi.

## CD

`.github/workflows/deploy.yml` berjalan pada push ke `main` (dan manual lewat *Run workflow*). Alurnya: CI → build dan push image frontend/backend (tag Git SHA dan `main`) → deploy lewat SSH ke VPS dengan environment `production`. Hanya satu deploy berjalan pada satu waktu (`concurrency: production`).

Skrip di VPS, berurutan:

| Langkah | Bila gagal |
|---|---|
| Prasyarat: Docker, Compose, `/opt/adicara`, `.env` | Berhenti; tidak ada yang diubah. |
| Cek file ter-track di `/opt/adicara` bersih, lalu `git fetch` + `git reset --hard <sha>` | Berhenti dengan petunjuk; tidak ada yang diubah. File ter-track yang diedit di server **tidak ditimpa diam-diam**. |
| Login GHCR, `compose config`, `compose pull frontend backend` | Berhenti; container yang berjalan tidak disentuh. Hanya image aplikasi yang ditarik, supaya database tidak ikut di-recreate saat ada rilis minor postgres. |
| `compose up -d --remove-orphans` | Lihat "Bila deploy gagal". `--remove-orphans` membersihkan container `migrate` dari versi Compose lama. |
| `nginx -t` lalu `nginx -s reload` (diulang hingga 5 kali) | Lihat "Bila deploy gagal". Diperlukan karena file konfigurasi di-mount: mengubahnya tidak membuat container di-recreate. |
| Health check: `/healthz` (backend lewat Nginx) lalu `/` (frontend), dengan retry | Lihat "Bila deploy gagal". |
| Sukses: catat SHA ke `.deploy-last-good`, `docker image prune` untuk image tak terpakai lebih dari 7 hari | — |

**Bila deploy gagal** (mulai dari `up`): `compose ps` dan 80 baris terakhir log backend/frontend/nginx dicetak ke log job; bila ada versi baik sebelumnya, stack dikembalikan ke image itu dan dicek sehat lagi. Job tetap **gagal** supaya kegagalannya terlihat, walau situs sudah pulih.

Skrip ini diuji dengan `docker`, `git`, dan `curl` tiruan (sukses, versi rusak dengan dan tanpa versi sebelumnya, git kotor, `fetch` gagal, `pull` gagal, `nginx -t` gagal), serta shellcheck (mode POSIX) dan actionlint. **Belum** dijalankan terhadap Docker/VPS sungguhan.

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
