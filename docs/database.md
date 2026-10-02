# Database

## Teknologi

PostgreSQL 18 digunakan di Compose. Perubahan skema selalu memakai migrasi SQL eksplisit di `backend/migrations`; aplikasi tidak melakukan auto-migration saat start.

## Skema saat ini

### `invitations`

Menyimpan identitas event generik, slug publik, judul, template, status (`draft`, `published`, `archived`), dan pengaturan indexing. Model tidak dikunci ke konsep mempelai agar jenis acara lain dapat ditambahkan.

### `invitation_hosts`

Menyimpan orang atau host yang ditampilkan pada undangan. Urutan dikontrol dengan `sort_order`.

### `invitation_events`

Menyimpan satu atau lebih agenda beserta waktu, timezone, venue, alamat, dan tautan peta opsional.

### `users` dan `sessions`

Menyimpan akun, hash bcrypt, hash token session, hash token CSRF, dan masa berlaku. Token session mentah tidak disimpan di database.

### `guests` dan `rsvps`

Menyimpan tamu per undangan, token publik acak, status RSVP, jumlah kehadiran, dan pesan opsional. Satu tamu memiliki paling banyak satu baris RSVP yang diperbarui secara idempoten.

## Relasi dan indeks

- `invitation_hosts.invitation_id` dan `invitation_events.invitation_id` memakai foreign key dengan `ON DELETE CASCADE`.
- `invitations.slug` unik dan tervalidasi lowercase-kebab-case.
- Indeks tersedia pada status dan seluruh foreign key yang dipakai untuk lookup.
- Semua query dashboard dibatasi oleh `invitations.user_id`; token tamu memiliki indeks unik.

## Migrasi

- `000001_create_invitation_domain.up.sql`
- `000001_create_invitation_domain.down.sql`
- `000002_create_auth_and_ownership.up.sql` / `.down.sql`
- `000003_create_guests_and_rsvps.up.sql` / `.down.sql`

Local Compose menjalankan `migrate/migrate` sebelum API. Produksi juga memiliki service migrasi eksplisit yang harus selesai sebelum backend start.

## Data pribadi

Skema menyimpan akun, token tamu, dan RSVP. Nomor telepon serta catatan tamu bersifat opsional dan hanya tersedia pada endpoint pemilik. Kebijakan retensi dan penghapusan akun masih perlu ditetapkan sebelum produksi publik. Hadiah, media, dan wishes belum ditambahkan.
