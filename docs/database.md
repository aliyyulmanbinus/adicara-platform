# Database

## Teknologi

PostgreSQL 18 digunakan di Compose. Perubahan skema selalu memakai migrasi SQL eksplisit di `backend/migrations`; aplikasi tidak melakukan auto-migration saat start.

## Skema saat ini

### `invitations`

Menyimpan identitas event generik, slug publik, judul, template, status (`draft`, `published`, `archived`), pengaturan indexing, dan `design_data` JSONB untuk teks kustom template. Model tidak dikunci ke konsep mempelai agar jenis acara lain dapat ditambahkan.

### `invitation_media`

Menyimpan dua potret mempelai dan maksimal lima foto galeri per undangan sebagai `BYTEA`. Tipe dibatasi pada JPEG, PNG, dan WebP; API membatasi ukuran 5 MB per foto. Baris media ikut dihapus saat undangan dihapus. URL foto publik hanya dilayani ketika status undangan `published`.

### `invitation_hosts`

Menyimpan orang atau host yang ditampilkan pada undangan. Urutan dikontrol dengan `sort_order`.

### `invitation_events`

Menyimpan satu atau lebih agenda beserta waktu, timezone, venue, alamat, dan tautan peta opsional.

### `users` dan `sessions`

Menyimpan akun, hash bcrypt, hash token session, hash token CSRF, dan masa berlaku. Token session mentah tidak disimpan di database.

### `guests` dan `rsvps`

Menyimpan tamu per undangan, token publik acak, status RSVP, jumlah kehadiran, dan pesan opsional. Satu tamu memiliki paling banyak satu baris RSVP yang diperbarui secara idempoten.

## Relasi dan indeks

- `invitation_hosts.invitation_id`, `invitation_events.invitation_id`, dan `invitation_media.invitation_id` memakai foreign key dengan `ON DELETE CASCADE`.
- `invitations.slug` unik dan tervalidasi lowercase-kebab-case.
- Indeks tersedia pada status dan seluruh foreign key yang dipakai untuk lookup.
- Semua query dashboard dibatasi oleh `invitations.user_id`; token tamu memiliki indeks unik.

## Migrasi

- `000001_create_invitation_domain.up.sql`
- `000001_create_invitation_domain.down.sql`
- `000002_create_auth_and_ownership.up.sql` / `.down.sql`
- `000003_create_guests_and_rsvps.up.sql` / `.down.sql`
- `000004_create_invitation_design.up.sql` / `.down.sql`

Local Compose menjalankan `migrate/migrate` sebelum API. Produksi juga memiliki service migrasi eksplisit yang harus selesai sebelum backend start.

## Data pribadi

Skema menyimpan akun, token tamu, RSVP, foto undangan, dan informasi hadiah yang dipilih pemilik untuk ditampilkan secara publik. Nomor telepon serta catatan tamu bersifat opsional dan hanya tersedia pada endpoint pemilik. Kebijakan retensi dan penghapusan akun masih perlu ditetapkan sebelum produksi publik. Transaksi hadiah dan wishes belum ditambahkan.
