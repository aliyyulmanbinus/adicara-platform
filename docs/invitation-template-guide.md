# Panduan Template Undangan

## Prinsip

Template adalah presentasi, bukan penyimpanan data atau business logic. Menambah template ke-4, ke-10, atau ke-50 tidak boleh memerlukan perubahan pada tabel undangan atau endpoint publik selama kontrak data tetap sama.

## Data contract

`frontend/src/invitation-templates/types.ts` mendefinisikan `InvitationData`, host, event, dan konfigurasi section. Adapter API di `frontend/src/lib/api/invitations.ts` mengubah bentuk JSON OpenAPI menjadi kontrak frontend.

## Struktur template

```text
frontend/src/invitation-templates/<key>/
├── Template.astro
├── theme.css
└── config.ts
```

Pratinjau statis ditempatkan di `frontend/public/images/`. Bila pipeline aset berkembang, preview WebP dapat diletakkan bersama template dan diimpor melalui Astro.

## Menambah template

1. Pilih key lowercase-kebab-case yang stabil.
2. Buat folder dan tiga file di atas.
3. Gunakan komponen di `components/invitation/sections`; jangan salin logika data atau interaksi.
4. Definisikan metadata yang memenuhi `InvitationTemplateDefinition`, termasuk section yang didukung dan versi.
5. Daftarkan config dan komponen pada `registry.ts`.
6. Tambahkan preview dengan dimensi eksplisit dan alt text yang bermakna.
7. Uji semua section pada lebar 360, 375, 390, 414, 430 piksel, tablet, dan desktop.
8. Jalankan check, test, dan build; dokumentasikan hasil aktual.

## Section

Section diaktifkan dan diurutkan oleh array `{ id, enabled, order }`. Theme merender urutan tersebut melalui komponen bersama. Theme tidak boleh memaksa urutan tetap.

## Performa

- Jangan menambahkan framework UI untuk presentasi statis.
- Optimalkan gambar dan isi atribut `width`/`height`.
- Lazy-load media di bawah fold.
- Audio harus dimulai dari gestur pengguna dan memiliki kontrol play/pause/mute.
- Hormati `prefers-reduced-motion`.

## Aksesibilitas

Pertahankan heading berurutan, fokus terlihat, kontras teks yang cukup, landmark semantik, tombol untuk aksi, serta tautan untuk navigasi. Konten utama tidak boleh bergantung pada animasi atau JavaScript.

