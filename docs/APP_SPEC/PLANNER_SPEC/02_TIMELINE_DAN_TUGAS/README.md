# Menu 02: Timeline & Tugas Persiapan

> **Prinsip Perancangan:** skema, endpoint, dan antarmuka bersifat **event-agnostic**; sesi, peran dokumen, nama fase, panduan, dan template dibaca dari registry dan master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Implementasi default dan template terlengkap difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Sheet *Timeline Planning* Hal. 3 dan *List Administrasi* Hal. 11). Delapan jenis acara lain memakai template `general` + tugas khusus yang didefinisikan di [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (usulan Adicara, bukan dari PDF).
>
> **Status (2026-10-21):** **backend sudah diimplementasikan** (modul `plannertimeline`, 17 endpoint, terpasang di menu 01; [BE.md](BE.md) dan [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-02.md)): widget Timeline dan Legalitas terisi, wizard menawarkan preset jadwal dan berkas, dan Proyek Contoh berisi 50 tugas dan 15 berkas. **Frontend belum dibangun** ([FE.md](FE.md) adalah acuannya; tipenya sudah dicocokkan dengan respons backend ini: 821 objek, nol selisih, dan fixture di [fixtures/](fixtures/) adalah salinan respons asli). Backend diuji terhadap PostgreSQL 16.2 (uji emas Proyek Contoh lulus, [BE bab 6](BE.md#6-pengujian)).

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Timeline Planning** (Hal. 3) | 44 tugas berurutan, tanggal mulai/selesai, durasi hari, status (15 Selesai, 29 Proses), Gantt mingguan | Tab 1 *Jadwal & Milestone*: tabel/Kanban/Gantt; 44 tugas disalin dengan nama, urutan, dan durasi PDF |
| **List Administrasi** (Hal. 11) | 15 berkas nikah (9 Selesai, 6 Belum), kolom Nominal/Detail/Jumlah (kosong), catatan alur RT/RW → kelurahan → KUA | Tab 2 *Dokumen & Legalitas*: checklist 15 berkas, kolom biaya/jumlah menjadi saran Adicara yang bisa diubah, panduan 4 langkah |

Yang **tidak** ada di PDF dan ditambahkan Adicara (ditandai di antarmuka dengan badge "Tambahan Adicara"): kolom scope/kategori/fase tiap tugas, offset relatif terhadap Hari-H (PDF memakai tanggal absolut), status ketiga `belum` (PDF hanya Selesai/Proses), 6 tugas penghubung ke Menu 05/06 (finalisasi tamu, sebar undangan, follow-up RSVP, technical meeting, gladi resik, Hari-H), serta nominal/jumlah dokumen.

---

## 2. Struktur Sub-Fitur (Tab)

### Tab 1: Jadwal & Milestone (`?tab=jadwal`)

- **Rangkaian acara (filter sesi)**: chip dibangkitkan dari **master sesi** proyek (`GET .../sessions`): `[Semua] [Umum] [Lamaran] [Akad] [Resepsi]` untuk pernikahan; `[Semua] [Umum] [Registrasi] [Sesi Pagi] …` untuk seminar. Tidak ada chip yang ditulis keras. `Semua` menampilkan semua sesi, `Umum` hanya tugas yang tidak terikat sesi.
- **Kustomisasi penuh**: tambah, ubah (judul, tanggal, durasi, PIC, vendor terkait, catatan), hapus, urut ulang tugas; template hanyalah titik awal. Opsi *tambah tugas template yang kurang* (`append`), *ganti tugas template* (`replace_template`, tugas kustom aman), atau *ganti semuanya* (`replace_all`); kedua mode ganti meminta konfirmasi: `replace_template` bila ada tugas template yang sudah berjalan atau bervendor, `replace_all` bila proyek punya tugas satu pun.
- **Generator jadwal**: preset diambil dari registry jenis acara (pernikahan: *Lengkap 44 tugas PDF*, *Esensial 22 tugas*, opsi *+ 6 tugas tambahan Adicara*; jenis lain: *Umum 15 tugas + tugas khusus jenisnya*). Tanggal dihitung mundur dari Hari-H; bila waktu persiapan lebih pendek dari horizon baku jadwal **dimampatkan otomatis** dan pengguna diberi tahu ([TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis), [BE §4.1](BE.md#41-generator-jadwal)).
- **Tiga tampilan**: Tabel/List per fase (nama fase tersimpan di data), Kanban 3 kolom (drag-drop dengan penanda *overdue*), dan Gantt CSS Grid mingguan dengan penanda Hari Ini.
- **Tautan vendor**: tugas dapat menunjuk vendor di Menu 04 (mis. "Visit Venue Pernikahan"), lengkap dengan nama vendor dan tombol WhatsApp.
- **Geser jadwal**: saat tanggal Hari-H diubah di Menu 01, pengguna ditanya apakah tugas template yang belum selesai ikut digeser (atomik dengan perubahan tanggal); di layar ini ada pula geser manual.
- **Ekspor `.xlsx`** menyerupai dokumen acuan, dengan kolom Rangkaian Acara.

### Tab 2: Dokumen & Legalitas (`?tab=dokumen`)

- **Checklist berkas**: pernikahan memuat 15 berkas PDF (nama persis); jenis lain memuat daftar perizinan dari registry (jenis *Acara Lainnya* tanpa template: pengguna menambah sendiri). Filter subjek dari registry (pernikahan: `[Semua] [CPP] [CPW] [Orang Tua CPP] [Orang Tua CPW] [Bersama]`; lainnya: `[Semua] [Panitia] [Lembaga/Instansi] [Umum]`).
- **Toggle** `belum` ↔ `selesai`, ubah biaya/jumlah/catatan, tambah, hapus, urut ulang, atau terapkan ulang template (`append` / `replace_template`).
- **Panduan prosedur** (callout): 4 langkah RT/RW → kelurahan (N1, N2, N4) → KUA asal → KUA tempat akad untuk pernikahan; 4 langkah perizinan untuk jenis lain. Teks berasal dari registry.
- **Sinkron biaya ke Anggaran (Menu 03)**: tombol "Kirim ke anggaran" menautkan berkas ke pos yang sesuai (mis. Pendaftaran KUA → Biaya KUA di sesi Akad) atau membuat pos baru; idempoten dan tidak menimpa angka yang sudah diisi pengguna ([BE §4.5](BE.md#45-sinkron-biaya-dokumen-ke-anggaran)). Sifatnya sekali kirim: mengubah biaya berkas sesudahnya tidak mengubah pos.

---

## 3. Penerapan 9 Jenis Acara

Satu tabel registry; template yang tersedia menentukan apa yang ditawarkan wizard dan generator. Pernikahan bersumber dari PDF; delapan lainnya adalah usulan Adicara ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)).

| Jenis | Horizon baku | Preset jadwal | Tugas | Dokumen (template) | Subjek dokumen | Judul widget Legalitas |
|---|:-:|---|:-:|:-:|---|---|
| 💍 Pernikahan | 180 hari | `wedding_pdf_44` · `wedding_essential_22` (+ 6 Adicara) | 44 (22) + 6 | 15 berkas KUA | CPP · CPW · Orang Tua CPP · Orang Tua CPW · Bersama | Kelengkapan Berkas KUA |
| 🕌 Khitanan | 45 | `general` | 18 | 3 | Panitia · Lembaga/Instansi · Umum | Kelengkapan Perizinan |
| 🍼 Aqiqah | 21 | `general` | 18 | 3 | idem | idem |
| 🏢 Peresmian | 60 | `general` | 18 | 4 | idem | idem |
| 🚀 Launching | 90 | `general` | 18 | 4 | idem | idem |
| 🎂 Ulang Tahun | 45 | `general` | 18 | 1 | idem | idem |
| 🎓 Wisuda | 60 | `general` | 18 | 3 | idem | idem |
| 🎤 Seminar | 75 | `general` | 18 | 4 | idem | idem |
| ✨ Acara Lainnya | 60 | `general` | 15 | – (tambah sendiri) | idem | idem |

Tugas non-pernikahan = 15 tugas umum + 3 tugas khusus jenisnya (Acara Lainnya: tanpa tugas khusus). Aturan pengisian kunci, kategori, dan fase tugas khusus: [BE §5.5](BE.md#55-template-jenis-non-pernikahan-usulan-adicara).

---

## 4. Integrasi (mekanisme nyata)

| Dari → ke | Mekanisme |
|---|---|
| 02 Dokumen → 03 Anggaran | `planner_budget_items.admin_doc_id` + `POST /admin-docs/:id/sync-budget` (SQL langsung ke tabel 03; pengecualian yang dicatat di kontrak bab 1.1) |
| 02 Tugas ↔ 04 Vendor | `planner_tasks.vendor_id` (FK komposit; tidak bisa menunjuk vendor proyek lain) |
| 02 Tugas ← sesi | `sub_event_scope` → master sesi (FK komposit) |
| 02 → 01 Dashboard | dua `SummaryProvider`: `timeline` (total, selesai, berjalan, terlambat, 3 tugas mendesak) dan `admin_docs` (judul, selesai/total, 5 berkas pertama yang belum) |
| 01 Wizard → 02 | dua `TemplateApplier` (`timeline`, `admin_docs`) dan dua `SampleOverlay` (status PDF, tanggal sesi Lamaran) |
| 02 ↔ 05/06 | tugas Adicara `wedding.adicara_*` memberi target waktu kerja Menu 05/06 (H-45, H-30, H-10, H-7, H-1, dan Hari-H) |

Rincian sambungan ke Menu 01: [BE §4.6](BE.md#46-sambungan-ke-menu-01). Matriks lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

---

## 5. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — View Switcher, chip sesi dinamis, Gantt CSS Grid, modal generator berbasis registry, daftar berkas, panduan, modal CRUD; juga tipe TypeScript, katalog endpoint dan galat dengan contoh respons asli, kode acuan yang sudah dijalankan (`timeline.ts`, `timelineApi.ts`, `messages.ts`), alur interaksi, QA, dan urutan kerja. Fixture: [fixtures/](fixtures/).
- ⚙️ [BE.md](BE.md) — migrasi 007 (`planner_tasks`, `planner_admin_docs`), DTO dan bentuk respons, endpoint, generator + penskalaan (kode acuan), sinkron biaya, sambungan ke Menu 01, master data 44 tugas + 6 tugas Adicara + 15 berkas, pengujian, dan celah yang menunggu keputusan.
