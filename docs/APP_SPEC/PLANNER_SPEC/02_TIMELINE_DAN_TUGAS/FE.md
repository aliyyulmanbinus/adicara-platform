# Menu 02: Timeline & Tugas Persiapan — Frontend Specification

Antarmuka **Jadwal & Milestone** (tabel, Kanban, Gantt) dan **Dokumen & Legalitas**. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens).

---

## 1. Route & File Path

| Item | Nilai |
|---|---|
| Halaman | `/dashboard/planner/[id]/timeline` → `src/pages/dashboard/planner/[id]/timeline/index.astro` |
| Tab | `?tab=jadwal` (default) · `?tab=dokumen` |
| Layout | `AppLayout.astro`; `nav_active = 'planner'`, sub-nav `timeline` |
| Komponen | `src/components/planner/timeline/`: `TaskTable`, `KanbanBoard`, `GanttMatrix`, `TaskModal`, `GeneratorModal`, `DocChecklist`, `GuideCallout` |
| Data SSR | `GET .../tasks`, `GET .../sessions`, `GET .../admin-docs`, `GET /v1/planner/event-types` (registry) |
| Aksi klien | `fetch('/dashboard/api/planner/projects/:id/...')` (BFF) |

**Sumber chip dan label**: seluruh chip sesi berasal dari `GET .../sessions` (kecuali sesi sistem `all`, yang ditampilkan sebagai chip "Umum"); peran dokumen, nama preset generator, dan teks panduan berasal dari registry jenis acara proyek. Tidak ada daftar `Lamaran/Akad/Resepsi` atau `CPP/CPW` yang tertulis di komponen.

---

## 2. Header, Metrik, dan Navigasi Tab

- Progress bar dan counter: `Total`, `Selesai`, `Sedang Berjalan`, `Belum`, `Terlambat` (dari `summary` respons `GET /tasks`; `overdue` dihitung server di zona waktu proyek).
- Tombol: `⚡ Generate Jadwal Otomatis`, `+ Tambah Tugas`, `📥 Ekspor .xlsx`.
- Tab: `[★ Jadwal & Milestone]` `[Dokumen & Legalitas]`.
- Proyek `archived`: semua kontrol ubah dinonaktifkan dengan keterangan "Acara diarsipkan".
- Kode error (`invalid_scope`, `invalid_date`, `project_read_only`, …) dipetakan ke teks Indonesia oleh `lib/planner/messages.ts`.

---

## 3. Tab 1: Jadwal & Milestone

### 3.1 Wireframe (Proyek Contoh: pernikahan, H-175; nama dan jadwal dari template)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ADICARA PLANNER > Timeline & Tugas                          [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ] │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ [★ Jadwal & Milestone]   [Dokumen & Legalitas]                                               │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Progres [███░░░░░░░░░░░░░░░░░] 30% (15/50)   Selesai 15 · Berjalan 29 · Belum 6 · Terlambat 0│
│ [⚡ Generate Jadwal]   [+ Tambah Tugas]   [📥 Ekspor .xlsx]                                  │
│ Sesi: [Semua] [Umum] [💍 Lamaran] [📜 Akad Nikah] [🏛️ Resepsi]                                │
│ Tampilan: [📋 Tabel] [🗂️ Kanban] [📊 Gantt]                                                   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ▼ FASE 1 — Persiapan Awal & Lamaran (7)                                                       │
│ ☑  1 Pertemuan keluarga            [Lamaran]  Keluarga   04 Okt – 05 Okt   1 hari  Selesai ⋮ │
│ ☑  3 Tetapkan anggaran lamaran     [Lamaran]  Keuangan   08 Okt – 10 Okt   2 hari  Selesai ⋮ │
│ ▼ FASE 2 — Administrasi & Booking Vendor Inti (8)                                             │
│ ☑  9 Siapkan pendaftaran pernikahan [Akad]    Administrasi 19 Okt – 20 Okt 1 hari  Selesai ⋮ │
│ ▼ FASE 3 — Detail Acara & Perlengkapan (20)                                                   │
│ ☐ 16 Finalisasi Seserahan          [Umum]     Logistik   02 Nov – 03 Nov   1 hari  Proses  ⋮ │
│ ☐ 25 Visit Venue Pernikahan  🤝 Sasana Kriya [Resepsi] Vendor 20 Nov – 21 Nov  1 hari  Proses ⋮│
│ ▼ FASE 4 — Finalisasi & Menuju Hari-H (15)                                                    │
│ ☐ A1 Finalisasi daftar tamu & kuota pax ✦ [Resepsi] Tamu     12 Feb – 13 Feb  1 hari  Belum  ⋮ │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 ✦ = Tambahan Adicara (bukan dari PDF)
```

### 3.2 Kontrol

1. **Filter sesi**: mengklik chip menyaring seluruh tampilan tanpa muat ulang; `Semua` menampilkan semua sesi, `Umum` hanya `sub_event_scope = all`.
2. **View Switcher**: Tabel, Kanban, Gantt. Pilihan terakhir diingat di `sessionStorage` (dibungkus try/catch seperti `AuthForm`), tidak di server.

### 3.3 Mode tampilan

**Tabel** — dikelompokkan per `phase_order` dengan judul dari `phase_name` (accordion). Kolom: checkbox cepat (`belum`↔`selesai`), No dan judul + badge sesi + badge "Tambahan Adicara" bila `origin = adicara` + tautan vendor (nama vendor dan tombol WA bila `vendor_id`), kategori, jadwal (klik membuka datepicker inline), durasi (hasil server), PIC, status dropdown, menu aksi (Edit, Hapus). Pengurutan ulang dengan drag handle memanggil `POST /tasks/reorder` dengan **daftar id lengkap**.

**Kanban** — kolom `Belum`, `Sedang Berjalan`, `Selesai`; kartu memuat badge sesi/kategori, judul, tanggal target, badge merah *Overdue* (dari `overdue` server), inisial PIC. Drag-drop HTML5 native dengan *optimistic update* lalu `PATCH /tasks/:id/status`; bila gagal, kartu dikembalikan dan toast galat ditampilkan.

**Gantt** — CSS Grid tanpa pustaka. Kolom kiri *sticky* (No, tugas, durasi, status); kanan header minggu ("Minggu 1 · 4 Okt 2026"), tanggal, dan hari (Sen…Mgg). Balok dari `start_date` hingga `end_date` **inklusif** (hijau selesai, oranye proses, kontur abu belum). Garis vertikal "Hari Ini" berdasarkan tanggal di zona waktu proyek.

### 3.4 Modal

**Tambah/Edit Tugas** — judul\*; sesi (dropdown dari master sesi + "Umum"); kategori (dropdown + kustom); fase (nomor dan nama, default dari tugas di sekitarnya); mulai/selesai (durasi dihitung ulang dari respons, tidak diisi manual); PIC; **Vendor terkait** (dropdown vendor proyek, opsional); catatan.

**Generator Jadwal Otomatis** — daftar *preset* berasal dari registry jenis acara:

| Jenis | Pilihan preset |
|---|---|
| Pernikahan | *Lengkap 44 tugas (PDF Hal. 3)* · *Esensial 22 tugas (persiapan 3–6 bulan)*; centang *+ 6 tugas tambahan Adicara* (default aktif) |
| Jenis lain | *Umum 15 tugas + tugas khusus [jenis]* |

Mode: ⚪ *Tambah yang belum ada* (`append`, default) · ⚪ *Ganti tugas template, simpan tugas saya* (`replace_template`) · ⚪ *Ganti semuanya* (`replace_all`, harus mencentang konfirmasi dan menampilkan jumlah tugas yang akan hilang). Setelah dijalankan modal menampilkan `created/skipped` dan `warnings` (mis. "Hari-H tinggal 40 hari, jadwal dimampatkan", "2 tugas dipindah ke Umum karena sesinya sudah dihapus").

**Konfirmasi hapus** — untuk tugas dan berkas.

**Geser jadwal** — dialog dari Dashboard/Edit Acara saat Hari-H berubah: "Geser N tugas yang belum selesai sebesar +X hari?" → `POST /tasks/shift`.

---

## 4. Tab 2: Dokumen & Legalitas

### 4.1 Wireframe (pernikahan, status dari PDF Hal. 11)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [Jadwal & Milestone]   [★ Dokumen & Legalitas]                       Kelengkapan: 9 / 15 (60%)│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Subjek: [Semua] [CPP] [CPW] [Orang Tua] [Bersama]        [+ Tambah Berkas] [Terapkan Template▼]│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│    No Dokumen                          Subjek   Jumlah     Biaya        Catatan        Aksi    │
│ ☑   1 Pendaftaran KUA                  Bersama  1 berkas   Rp 600.000   [➡️ Kirim ke anggaran] ✏️🗑️│
│ ☑   2 FC CPP ( KTP, Ijazah, Akte, KK)  CPP      2 lembar   Rp 0                           ✏️🗑️│
│ ☐   4 FC KTP Ortu CMP                  Ortu CPP 2 lembar   Rp 0                           ✏️🗑️│
│ ☐   9 Pas Foto Background Biru 2x3     Bersama  3 pcs      Rp 15.000                      ✏️🗑️│
│ ☐  13 Materai 10.000                   Bersama  2 pcs      Rp 20.000    [➡️ Kirim ke anggaran]✏️🗑️│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 💡 Panduan Pengurusan Berkas Nikah                                                            │
│  1 RT & RW · 2 Kelurahan (N1, N2, N4) · 3 KUA asal (numpang nikah) · 4 KUA tempat akad        │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Kontrol dan perilaku
- **Filter subjek** dari registry (`AdminSubjects` jenis acara), bukan tertulis.
- **Kolom Jumlah/Biaya** berasal dari saran Adicara; ditandai ikon info "bukan dari dokumen acuan" dan bebas diubah.
- **Kirim ke anggaran**: tombol muncul bila `nominal_cost > 0`. Setelah tertaut ia berubah menjadi badge "✓ Di anggaran" (dari `budget_item_id`); warning `estimate_differs` ditampilkan sebagai catatan, bukan error.
- **Terapkan Template** menawarkan `Tambah yang kurang` / `Ganti template`.
- **Panduan** (`GuideCallout`): komponen callout dengan `var(--sage-50)`/`var(--sage-200)`; judul dan 4 langkah berasal dari registry. Pernikahan menampilkan teks catatan PDF Hal. 11 (formulir **N1, N2, dan N4**); jenis lain menampilkan 4 langkah perizinan (izin lingkungan RT/RW → pemberitahuan kelurahan/Polsek → izin keramaian bila perlu → konfirmasi pengelola venue).

### 4.3 Modal
Tambah/Edit Dokumen: nama\*, subjek (dropdown registry), jumlah/rangkap, biaya (Rupiah, integer), catatan. Hapus berkas: dialog konfirmasi.

---

## 5. Responsif
- < 768px: tabel menjadi kartu tugas dengan checkbox besar; Kanban menjadi satu kolom per status dengan tab; Gantt menampilkan 4 minggu dengan scroll horizontal dan kolom kiri sticky; checklist dokumen menjadi kartu.

## 6. Pengujian Frontend
`node:test` untuk fungsi murni: pengelompokan fase, penentuan posisi balok Gantt dari `start_date`/`end_date`/zona waktu, penandaan *overdue* tidak dihitung klien (dipakai nilai server), dan pemetaan kode error → pesan.
