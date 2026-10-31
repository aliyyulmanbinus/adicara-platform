# Spesifikasi Fitur Adicara Event Planner (Multi-Event & Multi-Project)

Cetak biru teknis layanan **Planner** pada platform **Adicara**, platform acara modular dengan tiga layanan: **Undangan Digital**, **Planner**, dan **Vendor/Gift**. Dokumen ini memodernisasi `Wedding_Planner.pdf` (spreadsheet 11 sheet) menjadi aplikasi web di atas arsitektur yang sudah ada: **Astro (SSR) di frontend, Go (Gin, Module-Based Clean Architecture) di backend, PostgreSQL** ([docs/architecture.md](../../architecture.md)).

> **Status (2026-10-31):** **backend menu 01, 02, 03, 04, 05, dan 06 sudah diimplementasikan** (modul `planner`, `plannertimeline`, `plannerfinance`, `plannervendor`, `plannerguest`, dan `plannerrundown`, migrasi 005–012; dokumen [menu 01](../../backend/planner/2026-10-21-planner-menu-01.md), [menu 02](../../backend/planner/2026-10-21-planner-menu-02.md), [menu 03](../../backend/planner/2026-10-21-planner-menu-03.md), [menu 04](../../backend/planner/2026-10-23-planner-menu-04.md), [menu 05](../../backend/planner/2026-10-25-planner-menu-05.md), dan [menu 06](../../backend/planner/2026-10-31-planner-menu-06.md)). Seluruh frontend Planner masih spesifikasi. Spesifikasi Menu 05 dan 06 diaudit dan ditulis ulang lebih dulu (BE.md, FE.md, README, fixture), lalu dibangun dan dicocokkan: respons HTTP asli sama dengan fixture FE (pada menu 06 dua fixture dikoreksi saat dicocokkan). Fokus pembangunan saat ini adalah Planner; berikutnya adalah BFF dan frontend. Yang sudah diuji pada spesifikasi: DDL migrasi 005–012 dan angka PDF pada basis data ([kontrak bab 10](00_KONTRAK_BERSAMA/README.md#10-validasi-spesifikasi)).

**Pernikahan adalah jenis acara paling lengkap dan menjadi acuan** (data dari PDF). Delapan jenis lain memakai kerangka yang sama dengan template usulan Adicara yang ditandai jelas.

---

## 1. Dokumen dalam Folder Ini

| Dokumen | Isi |
|---|---|
| **[00_KONTRAK_BERSAMA](00_KONTRAK_BERSAMA/README.md)** | **Baca dulu.** Posisi di arsitektur BE/FE yang ada, urutan migrasi, isolasi data, registry event type, master sesi bersama, template vs data contoh, konvensi (uang, waktu, polling), kode error, matriks integrasi, urutan pembangunan, uji emas PDF |
| [00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md](00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) | Registry 9 jenis acara, algoritma penskalaan jadwal, template `general`, template non-pernikahan |
| [01_DASHBOARD](01_DASHBOARD/README.md) | Planner Hub, Workspace, proyek, master sesi, dashboard |
| [02_TIMELINE_DAN_TUGAS](02_TIMELINE_DAN_TUGAS/README.md) | Jadwal/Gantt dan dokumen legalitas |
| [03_KEUANGAN](03_KEUANGAN/README.md) | Simulasi, tabungan, realisasi |
| [04_VENDOR_DAN_BELANJA](04_VENDOR_DAN_BELANJA/README.md) | Vendor dan belanja pengadaan (pagu, kunci kerja sama, gulungan ke anggaran) |
| [05_BUKU_TAMU](05_BUKU_TAMU/README.md) | Tamu, RSVP, meja resepsionis |
| [06_RUNDOWN_DAN_ACARA](06_RUNDOWN_DAN_ACARA/README.md) | Rundown dan playlist |
| [CHANGELOG.md](CHANGELOG.md) | Riwayat revisi spesifikasi |
| `Wedding_Planner.pdf` | Dokumen acuan (16 halaman) |

Tiap folder modul memiliki `README.md` (pemetaan ke PDF, fitur, integrasi), `FE.md`, dan `BE.md`.

---

## 2. Sembilan Jenis Acara

Satu sumber: registry di kode ([kontrak bab 3](00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data), data di [TEMPLATE_EVENT](00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#3-ringkasan-registry-sembilan-jenis)).

| Key | Jenis | Ikon | Catatan |
|---|---|:-:|---|
| `wedding` | Pernikahan | 💍 | acuan PDF; paling lengkap |
| `khitanan` | Khitanan | 🕌 | |
| `aqiqah` | Aqiqah | 🍼 | |
| `peresmian` | Peresmian (kantor/gedung/toko) | 🏢 | |
| `launching` | Launching (produk/brand) | 🚀 | |
| `birthday` | Ulang Tahun | 🎂 | |
| `wisuda` | Wisuda | 🎓 | |
| `seminar` | Seminar / workshop / konferensi | 🎤 | |
| `other` | Acara Lainnya | ✨ | |

### 2.1 Satu Akun, Banyak Planner

```text
AKUN PENGGUNA (users.id)
  ├─ PLANNER 1: 💍 "Pernikahan Sinta & Arif" (28 Mar 2027)  [aktif]
  ├─ PLANNER 2: 🍼 "Aqiqah Muhammad Rayyan" (14 Feb 2026)   [selesai]
  └─ PLANNER 3: 🎤 "Tech Summit 2027" (15 Mei 2027)         [aktif]
Tiap planner punya enam menu dan data terisolasi sendiri.
```

Tidak ada batas produk jumlah proyek per akun; ada batas teknis 100 proyek nyata per pengguna (`PLANNER_MAX_PROJECTS`, `409 project_limit_reached`; Proyek Contoh tidak dihitung). Isolasi ditegakkan di tiga lapis: middleware `ProjectGuard` (`404` untuk proyek bukan milik pemanggil), filter `project_id` di setiap query, dan foreign key komposit di basis data ([kontrak bab 2](00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)). Tabel anak tidak memiliki `user_id`; kepemilikan selalu diperiksa lewat proyek.

### 2.2 Tiga Skenario Pengguna

1. **Pengguna baru (0 proyek)** — `/dashboard/planner` menampilkan *empty state* "Mulai Rencanakan Momen Bersejarah Anda bersama Adicara" dengan tombol `+ Rencanakan Acara Pertama Anda` dan `Lihat Proyek Contoh`. Wizard 3 langkah (`/dashboard/planner/baru`): pilih satu dari 9 jenis → judul, tanggal Hari-H, pagu, atribut jenis itu → pilih template yang **tersedia untuk jenis tersebut**. Setelah `Simpan & Buka Workspace`, pengguna diarahkan ke `/dashboard/planner/{id}`.
2. **Menambah planner setelah acara lama selesai** — kartu lama berbadge "Selesai" tetap bisa dibuka sebagai riwayat; `+ Buat Acara Baru` membuat proyek baru yang bersih (data tidak bercampur); Project Switcher di bilah sekunder (di bawah header) berpindah antara acara aktif dan riwayat.
3. **Planner paralel** — acara baru dapat dibuat dari Hub atau dari Project Switcher; kartu masing-masing menampilkan hitung mundur sendiri; berpindah ruang kerja satu klik.

---

## 3. Navigasi dan Routing Frontend

Semua halaman berada di `/dashboard/**` sehingga otomatis dijaga `src/middleware.ts`. Halaman membaca data server-side; aksi interaktif lewat BFF same-origin (browser tidak memanggil Go).

```text
frontend/src/pages/dashboard/
├── planner/
│   ├── index.astro                  Planner Hub
│   ├── baru.astro                   Wizard Acara Baru
│   └── [id]/
│       ├── index.astro              Menu 01 Dashboard Acara
│       ├── widgets.astro            Partial HTML hero + widget (polling 30 detik)
│       ├── timeline/index.astro     Menu 02 Jadwal & Dokumen
│       ├── keuangan/index.astro     Menu 03 Simulasi, Tabungan, Realisasi
│       ├── vendor/index.astro       Menu 04 Vendor & Belanja
│       ├── tamu/index.astro         Menu 05 Buku Tamu & Meja Resepsionis
│       └── rundown/index.astro      Menu 06 Rundown & Playlist
└── api/planner/[...path].ts         BFF → /v1/planner/* (aturan: kontrak bab 1.3)
```

Project Switcher dan sub-navigasi enam menu ada di bilah sekunder (`PlannerLayout`) di bawah header semua halaman `[id]/**`; menu yang belum dibangun tampil nonaktif. Sub-tab memakai query `?tab=…` (tiap menu mendokumentasikan nilainya).

---

## 4. Enam Menu

| No | Menu | Sheet PDF | Frontend | Backend |
|:-:|---|---|---|---|
| 01 | **Dashboard & Hub Acara** | Dashboard (Hal. 1) | [FE](01_DASHBOARD/FE.md) | [BE](01_DASHBOARD/BE.md) |
| 02 | **Timeline & Tugas** | Timeline Planning (Hal. 3) + List Administrasi (Hal. 11) | [FE](02_TIMELINE_DAN_TUGAS/FE.md) | [BE](02_TIMELINE_DAN_TUGAS/BE.md) |
| 03 | **Keuangan & Anggaran** | Skenario Budget (Hal. 4) + Target Tabungan (Hal. 7) + Anggaran Pernikahan (Hal. 5–6) | [FE](03_KEUANGAN/FE.md) | [BE](03_KEUANGAN/BE.md) |
| 04 | **Vendor & Belanja** | Kontak Vendor (Hal. 12) + List Seserahan (Hal. 9–10) | [FE](04_VENDOR_DAN_BELANJA/FE.md) | [BE](04_VENDOR_DAN_BELANJA/BE.md) |
| 05 | **Buku Tamu & Undangan** | List Tamu (Hal. 13) | [FE](05_BUKU_TAMU/FE.md) | [BE](05_BUKU_TAMU/BE.md) |
| 06 | **Rundown & Acara** | Rundown Acara (Hal. 14) + Playlist Pernikahan (Hal. 15–16) | [FE](06_RUNDOWN_DAN_ACARA/FE.md) | [BE](06_RUNDOWN_DAN_ACARA/BE.md) |

PDF halaman 2 (catatan "akses view only, buat salinan") dan halaman 8 (kosong) tidak menghasilkan fitur.

---

## 5. Basis Data

Skema tidak ditulis ulang di sini. DDL yang berlaku (14 tabel + 1 view) ada di dokumen BE tiap modul dan telah dijalankan di PostgreSQL 16; urutan migrasi dan ketergantungan antartabel: [kontrak bab 1.2](00_KONTRAK_BERSAMA/README.md#12-migrasi-goose-backendinternalmigratesql). Ringkas:

- `planner_projects` (1 pengguna : N proyek) dan `planner_event_sessions` (master sesi bersama) di migrasi 005.
- Semua tabel anak menunjuk `project_id` dengan `ON DELETE CASCADE`, dan menunjuk antarmereka lewat FK komposit `(id, project_id)`.
- Uang `BIGINT` rupiah utuh; waktu `DATE`/`TIME`/`TIMESTAMPTZ`; PK UUID dari aplikasi; `TEXT` dan `now()` seperti migrasi 001–004.

## 6. Standar Teknis

- **Frontend**: Astro SSR (`@astrojs/node`), TypeScript strict, CSS token Adicara (`tokens.css`), Fredoka dan Nunito Sans, Phosphor Icons; tanpa dependensi baru (grafik SVG, Gantt CSS Grid, drag-drop native).
- **Backend**: Go, Gin, `pgxpool`, goose, modul `planner`, `plannertimeline`, `plannerfinance`, `plannervendor`, `plannerguest`, `plannerrundown` dengan lapisan `delivery/http → usecase → domain ← repository`; respons `{"data": …}`, error `{"error": {code, message}}`.
- **Autentikasi**: JWT Bearer yang sudah ada; token tersimpan di cookie HttpOnly pada Astro dan dipakai server-side atau lewat BFF.
- **Dokumentasi saat implementasi**: tambahkan endpoint ke `backend/docs/openapi.yaml`, dokumen modul di `docs/backend/planner/` dan `docs/frontend/planner/`, serta entri CHANGELOG.
