# Menu 02: Timeline & Tugas Persiapan

> **Prinsip Perancangan:** skema, endpoint, dan antarmuka bersifat **event-agnostic**; sesi, peran dokumen, nama fase, dan template dibaca dari registry dan master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Implementasi default dan template terlengkap difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Sheet *Timeline Planning* Hal. 3 dan *List Administrasi* Hal. 11). Delapan jenis acara lain memakai template `general` + tugas khusus yang didefinisikan di [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (usulan Adicara, bukan dari PDF).

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Timeline Planning** (Hal. 3) | 44 tugas berurutan, tanggal mulai/selesai, durasi hari, status (Selesai/Proses), Gantt mingguan | Tab 1 *Jadwal & Milestone*: tabel/Kanban/Gantt; 44 tugas disalin dengan nama, urutan, dan durasi PDF |
| **List Administrasi** (Hal. 11) | 15 berkas nikah (Selesai/Belum), kolom Nominal/Detail/Jumlah (kosong), catatan alur RT/RW → kelurahan → KUA | Tab 2 *Dokumen & Legalitas*: checklist 15 berkas, kolom biaya/jumlah menjadi saran Adicara yang bisa diubah, panduan 4 langkah |

Yang **tidak** ada di PDF dan ditambahkan Adicara (ditandai di antarmuka dengan badge "Tambahan Adicara"): kolom scope/kategori/fase tiap tugas, offset relatif terhadap Hari-H (PDF memakai tanggal absolut), status ketiga `belum`, 6 tugas penghubung ke Menu 05/06 (finalisasi tamu, sebar undangan, follow-up RSVP, technical meeting, gladi resik, Hari-H), serta nominal/jumlah dokumen.

---

## 2. Struktur Sub-Fitur (Tab)

### Tab 1: Jadwal & Milestone (`?tab=jadwal`)

- **Rangkaian acara (filter sesi)**: chip dibangkitkan dari **master sesi** proyek (`GET .../sessions`): `[Semua] [Umum] [Lamaran] [Akad] [Resepsi]` untuk pernikahan; `[Semua] [Umum] [Registrasi] [Sesi Pagi] …` untuk seminar. Tidak ada chip yang ditulis keras.
- **Kustomisasi penuh**: tambah, ubah (judul, tanggal, durasi, PIC, vendor terkait, catatan), hapus, urut ulang tugas; template hanyalah titik awal. Opsi *tambah tugas template yang kurang* (`append`), *ganti tugas template* (`replace_template`, tugas kustom aman), atau *ganti semuanya* (`replace_all`, dengan konfirmasi).
- **Generator jadwal**: preset diambil dari registry jenis acara (pernikahan: *Lengkap 44 tugas PDF*, *Esensial 22 tugas*, opsi *+ 6 tugas tambahan Adicara*; jenis lain: *Umum 15 tugas + tugas khusus jenisnya*). Tanggal dihitung mundur dari Hari-H; bila waktu persiapan lebih pendek dari horizon baku jadwal **dimampatkan otomatis** dan pengguna diberi tahu ([TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis)).
- **Tiga tampilan**: Tabel/List per fase (nama fase tersimpan di data), Kanban 3 kolom (drag-drop dengan penanda *overdue*), dan Gantt CSS Grid mingguan dengan penanda Hari Ini.
- **Tautan vendor**: tugas dapat menunjuk vendor di Menu 04 (mis. "Batas DP Venue").
- **Geser jadwal**: saat tanggal Hari-H diubah, pengguna ditawari menggeser tugas template yang belum selesai.
- **Ekspor `.xlsx`** menyerupai dokumen acuan, dengan kolom Rangkaian Acara.

### Tab 2: Dokumen & Legalitas (`?tab=dokumen`)

- **Checklist berkas**: pernikahan memuat 15 berkas PDF (nama persis); jenis lain memuat daftar perizinan dari registry. Filter peran dari registry (pernikahan: `[Semua] [CPP] [CPW] [Orang Tua] [Bersama]`; lainnya: `[Semua] [Panitia] [Lembaga/Instansi] [Umum]`).
- **Toggle** `belum` ↔ `selesai`, ubah biaya/jumlah/catatan, tambah, hapus, urut ulang, atau terapkan ulang template (`append` / `replace_template`).
- **Panduan prosedur** (callout): 4 langkah RT/RW → kelurahan (N1, N2, N4) → KUA asal → KUA tempat akad untuk pernikahan; 4 langkah perizinan untuk jenis lain. Teks berasal dari registry.
- **Sinkron biaya ke Anggaran (Menu 03)**: tombol "Kirim biaya ke anggaran" menautkan berkas ke pos yang sesuai (mis. Pendaftaran KUA → Biaya KUA di sesi Akad) atau membuat pos baru; idempoten dan tidak menimpa angka yang sudah diisi pengguna ([BE §4.5](BE.md#45-sinkron-biaya-dokumen-ke-anggaran)).

---

## 3. Integrasi (mekanisme nyata)

| Dari → ke | Mekanisme |
|---|---|
| 02 Dokumen → 03 Anggaran | `planner_budget_items.admin_doc_id` + `POST /admin-docs/:id/sync-budget` |
| 02 Tugas ↔ 04 Vendor | `planner_tasks.vendor_id` (FK komposit; tidak bisa menunjuk vendor proyek lain) |
| 02 Tugas ← sesi | `sub_event_scope` → master sesi (FK komposit) |
| 02 ← 01 Dashboard | provider `timeline` (total, selesai, terlambat, 3 tugas mendesak) dan `admin_docs` |
| 02 ↔ 05/06 | tugas Adicara `wedding.adicara_*` memberi target waktu kerja Menu 05/06 (H-45, H-30, H-10, H-7, H-1) |

---

## 4. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — View Switcher, chip sesi dinamis, Gantt CSS Grid, modal generator berbasis registry, daftar berkas, panduan, modal CRUD.
- ⚙️ [BE.md](BE.md) — migrasi 007 (`planner_tasks`, `planner_admin_docs`), endpoint, generator + penskalaan, master data 44 tugas + 6 tugas Adicara + 15 berkas.
