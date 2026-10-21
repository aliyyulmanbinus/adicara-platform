# Menu 01: Dashboard & Hub Acara Multi-Event

> **Prinsip Perancangan:**
> 1. **Multi-Planner (1 User : N Acara)**: satu akun dapat membuat, mengelola, dan mengarsipkan banyak acara tanpa batas (mis. Pernikahan dan Seminar bersamaan, atau menyimpan riwayat acara yang sudah selesai).
> 2. **Universal (Event-Agnostic)**: ikon, label, sesi, peran, dan template dibaca dari **registry event type** ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data)) untuk 9 jenis acara; tidak ada daftar jenis acara kedua di dokumen mana pun.
> 3. **Fokus Pernikahan**: master data default dan studi kasus terlengkap mengikuti **Wedding_Planner.pdf** (Sheet *Dashboard*, Hal. 1).
> 4. **Aturan lintas modul** (master sesi, template vs data contoh, uang/zona waktu, kode error, matriks integrasi) hanya ditulis di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).
>
> **Status (2026-10-21):** backend Menu 01 sudah diimplementasikan dan diuji ([BE.md](BE.md)); frontend belum dibangun ([FE.md](FE.md) adalah acuannya). Modul 02–06 belum ada, jadi keenam widget Workspace saat ini tampil "Segera hadir".

---

## 1. Pemetaan ke Dokumen Acuan (PDF Hal. 1)

Halaman pertama PDF berisi judul besar, nama pasangan, tanggal acara, dan hitung mundur:

| Elemen PDF | Di Adicara |
|---|---|
| **"BISMILLAH NIKAH"** (judul besar) | `metadata.hero_title` (opsional, semua jenis acara) |
| **"Nama Cewe & Nama Pria"** | pernikahan: `bride_name & groom_name` (urutan mengikuti PDF); jenis lain: `title` + kolom wajib jenisnya (mis. penyelenggara seminar) |
| **"Weeding Date: 28-Mar-2027"** | `event_date` |
| **"Towards the Day: 175 Days"** | `days_remaining`, dihitung di zona waktu proyek ([BE §4.2](BE.md#42-hitung-mundur-dan-fase-acara)) |
| Menu 11 sheet (Dashboard … Playlist Pernikahan) | enam menu Planner; sheet Skenario Budget, Anggaran, dan Target Tabungan digabung di Menu 03 |

Contoh di seluruh dokumen Planner memakai satu data konsisten dari PDF: **Sinta & Arif**, **28 Maret 2027** (Minggu), pagu skenario **Rp 75 juta**, dan **Proyek Contoh** dengan hitung mundur 175 hari ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).

### Dua Tingkat

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│                             TINGKAT 1: PLANNER HUB                               │
│                         URL: /dashboard/planner                                  │
│ • Portofolio seluruh acara milik pengguna (semua jenis)                          │
│ • Filter status (Aktif, Selesai, Arsip) dan kategori (9 jenis dari registry)     │
│ • "+ Buat Acara Baru" (wizard 3 langkah); "Lihat Proyek Contoh" bila belum ada   │
└────────────────────────────────────────┬─────────────────────────────────────────┘
                 ┌───────────────────────┴───────────────────────┐
                 ▼                                               ▼
┌──────────────────────────────────┐            ┌──────────────────────────────────┐
│  TINGKAT 2: WORKSPACE PROYEK A   │            │  TINGKAT 2: WORKSPACE PROYEK B   │
│ /dashboard/planner/[id_a]        │            │ /dashboard/planner/[id_b]        │
│ 💍 Pernikahan Sinta & Arif       │            │ 🎤 Tech Summit 2027              │
│ • Hero + hitung mundur Hari-H    │            │ • Hero + hitung mundur Hari-H    │
│ • Switcher + sub-navigasi menu   │            │ • Switcher + sub-navigasi menu   │
│ • 6 widget ringkasan Menu 02–06  │            │ • 6 widget ringkasan Menu 02–06  │
└──────────────────────────────────┘            └──────────────────────────────────┘
```

---

## 2. Struktur Sub-Fitur

### Area 1: Planner Hub (`/dashboard/planner`)
- **4 kartu metrik portofolio**: acara aktif, acara selesai, total pagu proyek aktif, acara terdekat (nama + hitung mundur). Proyek Contoh tidak ikut dihitung.
- **Tab status** `[Semua] [Aktif] [Selesai] [Arsip]` dan **filter kategori** (sembilan jenis dari registry, mis. Pernikahan, Khitanan, Aqiqah, Peresmian, Launching, Ulang Tahun, Wisuda, Seminar, Acara Lainnya). Hitungan pada tab menyertakan Proyek Contoh; empat kartu metrik tidak.
- **Kartu proyek**: ikon + jenis, judul, tanggal Hari-H (dan tanggal selesai bila beberapa hari), hitung mundur atau fase ("Sedang berlangsung", "Selesai"), progres tugas, pagu vs terbayar, jumlah pax (label "Tamu"/"Peserta" mengikuti jenis acara; kartu tidak memuat jumlah undangan), tombol utama (`Masuk Workspace`; proyek selesai/arsip: `Lihat Ringkasan`), menu opsi (Edit, Tandai selesai, Arsipkan, Hapus). Kartu Proyek Contoh berbadge "Contoh". Semua persentase mengikuti aturan pembagi nol: nilai kosong ditampilkan sebagai keadaan kosong, bukan 0%.
- **Kartu ajakan** "Buat Acara Baru" di akhir grid. Tombol **Lihat Proyek Contoh** hanya tampil selama belum ada Proyek Contoh.
- **Wizard Buat Acara** (3 langkah, `/dashboard/planner/baru`): jenis acara → judul, tanggal (opsional: tanggal selesai untuk acara beberapa hari), pagu, zona waktu, atribut subjek jenis itu → pilih template yang tersedia untuk jenis tersebut ([BE §4.1](BE.md#41-pembuatan-proyek-post-v1plannerprojects)). Pengiriman ganda tidak membuat proyek kedua (`client_token`).
- **Empty state** untuk pengguna tanpa acara: ajakan membuat acara pertama atau membuka Proyek Contoh.

### Area 2: Workspace (`/dashboard/planner/[id]`)
- **Project Switcher** di bilah sekunder di bawah header (kiri), dengan **sub-navigasi enam menu** di kanannya (menu yang belum dibangun tampil nonaktif): lompat antar acara tanpa kembali ke Hub, tombol "Buat acara lain", daftar aktif lalu selesai/arsip.
- **Hero Event Card**: identitas subjek (pernikahan: `bride_name & groom_name`, tanggal, venue; jenis lain: judul + penyelenggara/subjek dan venue), hitung mundur hari (≤ 7 hari berganti `Hari : Jam : Menit : Detik`), dan **fase** dari server: sedang berlangsung ("Hari-H telah tiba" atau "Hari ke-N dari M" untuk acara beberapa hari) atau "Acara Telah Berlangsung". Tombol **Edit Acara** (judul, tanggal Hari-H/selesai, pagu, atribut, **Kelola Sesi**).
- **Edit tanggal** menanyakan dulu apakah tugas template yang belum selesai ikut digeser, lalu menyimpan keduanya dalam satu langkah atomik ([BE §4.3](BE.md#43-ubah-tanggal-acara-dan-geser-jadwal)); sesi yang bertanggal Hari-H ikut pindah.
- **Kelola Sesi**: tambah, ubah nama/ikon/tanggal/jam, urutkan, hapus dengan pemindahan data ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)). Satu daftar sesi ini menggerakkan chip filter di Menu 02–06.
- **Enam widget ringkasan** (susunan yang sama di README, FE, dan BE):

| # | Widget | Isi |
|---|---|---|
| 1 | **Timeline & Tugas** (Menu 02) | selesai dari total dengan persentase, jumlah berjalan dan terlambat, hingga 3 tugas mendesak (belum selesai, tenggat dalam 7 hari ke depan; yang sudah lewat ikut) |
| 2 | **Keuangan & Tabungan** (Menu 03) | donut estimasi vs terbayar vs sisa; tabungan terkumpul vs target; kas bebas / defisit; jumlah pos lunas, sebagian, belum |
| 3 | **Vendor Rekanan** (Menu 04) | kategori terkunci dari total kategori (mis. 7 dari 10; kategori "tidak diperlukan" tidak dihitung), kandidat berjalan, kategori yang dilewati (bila ada). Hanya angka: tanpa daftar vendor atau tautan WA |
| 4 | **Buku Tamu & RSVP** (Menu 05; judul mengikuti jenis acara, seminar: "Peserta & RSVP") | total tamu dan pax, per pihak/sisi, terkirim, empat status RSVP (hadir, tidak, ragu, belum) |
| 5 | **Legalitas / Perizinan** (Menu 02 Tab 2) | pernikahan: kelengkapan 15 berkas KUA; lainnya: perizinan dan kontrak venue; judul widget diisi provider Menu 02 |
| 6 | **Rundown Hari-H** (Menu 06) | jumlah sesi, agenda, lagu; sesi terdekat; agenda yang sedang berjalan pada hari-H |

Widget yang modulnya belum tersedia tampil sebagai kartu "Segera hadir" tanpa tombol Buka; widget yang kosong menampilkan ajakan tanpa persen; widget yang query-nya gagal atau melewati batas waktu menampilkan galat lokal tanpa merusak widget lain, dan satu provider yang panic tidak mematikan server ([BE bab 5](BE.md#5-dashboard-get-v1plannerprojectsiddashboard)).

Perubahan keadaan acara tidak otomatis: proyek yang tanggalnya sudah lewat tetap `active` sampai pengguna menekan "Tandai selesai". Acara beberapa hari memakai `event_end_date`; Hari-H tetap hari pertama.

---

## 3. Penerapan 9 Jenis Acara

Satu tabel registry; template yang tersedia menentukan apa yang ditawarkan wizard. ✅ = template ada. Pernikahan bersumber dari PDF; delapan lainnya adalah usulan Adicara ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)).

| Jenis | Ikon | Contoh judul | Subjek (metadata) | Jadwal | Dokumen | Pos anggaran | Belanja | Agenda | Playlist | Widget legalitas |
|---|:-:|---|---|:-:|:-:|:-:|:-:|:-:|:-:|---|
| Pernikahan | 💍 | Pernikahan Sinta & Arif | bride_name, groom_name, orang tua | 44 + tambahan | 15 KUA | 6 preset + 14 pos | 37 item | 11 + 22 | 30 lagu | Berkas KUA |
| Khitanan | 🕌 | Khitanan Muhammad Rayyan | child_name, parent_names | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Aqiqah | 🍼 | Aqiqah Rayyan | baby_name, birth_date, goat_count | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Peresmian | 🏢 | Grand Opening Kantor Adicara | organization_name, vip_name | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Launching | 🚀 | Peluncuran Adicara 2.0 | brand_name, product_name | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Ulang Tahun | 🎂 | Sweet 17 Keysha | celebrant_name, age, theme | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Wisuda | 🎓 | Wisuda Angkatan 2027 | institution_name, faculty | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Seminar | 🎤 | Tech Summit Indonesia 2027 | organizer_name, keynote_speaker | ✅ | ✅ | ✅ (%) | kategori | ✅ | – | Perizinan |
| Lainnya | ✨ | Reuni Akbar Alumni | organizer_name | ✅ (`general`) | – | ✅ (%) | kategori | ✅ | – | Perizinan |

Template daftar belanja dan playlist hanya ada untuk pernikahan. Jenis lain mendapat **kategori belanja** (dropdown dari registry) tanpa baris bawaan, dan Tab Audio kosong dengan *moment tag* dari registry.

---

## 4. Integrasi (mekanisme nyata)

Dashboard mengumpulkan ringkasan lewat `SummaryProvider` yang didaftarkan tiap modul ([kontrak bab 7](../00_KONTRAK_BERSAMA/README.md#7-dashboard-pola-summaryprovider)); ringkasan widget hanya dikumpulkan lewat provider. Pengecualian: kartu Hub, `usage` sesi, pemindahan data saat menghapus sesi, dan penggeseran tugas memakai SQL langsung pada tabel milik modul 02–06, sehingga migrasi 005–012 membuat tabel-tabel itu lebih dulu. Wizard menerapkan template lewat `TemplateApplier` tiap modul di dalam satu transaksi, dan hanya menawarkan template yang modulnya sudah ada. Matriks integrasi lengkap dan mekanismenya: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

```text
 MENU 01 Dashboard  ◄── SummaryProvider ──  02 Timeline │ 03 Keuangan │ 04 Vendor │ 05 Tamu │ 06 Rundown
        │
        └── memiliki: planner_projects, planner_event_sessions (master sesi untuk semua modul)
```

---

## 5. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — Planner Hub, wizard dinamis, Workspace Dashboard, Project Switcher, enam widget, Kelola Sesi, pola SSR + BFF; juga tipe TypeScript dan fixture, wireframe (keadaan akhir dan keadaan hari ini), fungsi acuan, tabel galat, QA, dan urutan kerja.
- ⚙️ [BE.md](BE.md) — skema `planner_projects` dan `planner_event_sessions` (migrasi 005; 006–012 membuat tabel Menu 02–06 lebih dulu), endpoint proyek/sesi/registry/dashboard, `SummaryProvider`, Proyek Contoh.

