# PostgreSQL Backup dan Restore

## Backup

Jalankan backup terjadwal dari host atau job terisolasi:

```sh
docker compose -f deploy/docker-compose.prod.yml exec -T postgres \
  pg_dump --format=custom --no-owner --username="$POSTGRES_USER" "$POSTGRES_DB" \
  > "adicara-$(date +%Y%m%d-%H%M%S).dump"
```

Simpan backup terenkripsi di lokasi terpisah dari VPS. Rekomendasi awal: backup harian, retensi 7 harian + 4 mingguan, lalu sesuaikan setelah kebutuhan bisnis dan kebijakan privasi ditetapkan.

## Restore

1. Hentikan trafik tulis.
2. Verifikasi checksum dan tanggal backup.
3. Buat database kosong atau pastikan target dapat diganti.
4. Restore:

   ```sh
   pg_restore --clean --if-exists --no-owner --dbname="$DATABASE_URL" adicara.dump
   ```

5. Jalankan migrasi menuju versi aplikasi target.
6. Periksa `/readyz` dan lakukan smoke test undangan.

## Verifikasi

Backup belum dianggap terbukti hanya karena `pg_dump` berhasil. Jadwalkan restore drill berkala ke database terisolasi dan catat hasil, durasi, serta checksum. Prosedur ini belum diuji pada environment produksi.

