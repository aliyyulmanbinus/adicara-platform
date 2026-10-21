# Menu 06: Rundown & Acara

> **Prinsip Perancangan:** skema, endpoint, dan antarmuka **event-agnostic**. Sesi tidak ditulis keras: modul ini memakai **master sesi bersama** (`planner_event_sessions`, dimiliki Menu 01) yang juga menggerakkan filter di Menu 02–05 ([00_KONTRAK_BERSAMA bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)). Implementasi default difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Sheet *Rundown Acara* Hal. 14 dan *Playlist Pernikahan* Hal. 15–16); delapan jenis lain memakai agenda usulan Adicara ([TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)).

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Rundown Acara Lamaran** (Hal. 14 kiri) | 11 agenda 09.00 – selesai; kolom No, Waktu, Acara, PIC, Keterangan, Highlights | Sesi `lamaran`: 11 agenda dengan teks persis PDF; agenda 11 terbuka ("10.40 - selesai") |
| **Rundown Resepsi dan Pernikahan** (Hal. 14 kanan) | 22 agenda 07.00–13.00 (Akad, lalu "Resepsi 09.15"); kolom No, Waktu, Acara, PIC, Keterangan | Sesi `akad` (agenda 1–8) dan `resepsi` (9–22); total 33 agenda template |
| **Playlist Pernikahan** (Hal. 15–16) | 30 lagu (No, Judul, Penyanyi) | Tab Audio: 30 lagu; moment tag, format, dan status adalah tambahan Adicara |

Fakta PDF yang dijaga: beberapa agenda berjalan paralel (sesi makan, hiburan, salaman) dan sebagian tumpang tindih, sehingga **total durasi dihitung sebagai rentang** per sesi (Proyek Contoh: 100 + 120 + 225 = **445 menit**), bukan jumlah durasi. PIC yang kosong di PDF tetap kosong; template tidak memuat nama orang.

Yang ditambahkan Adicara: tracker live, tautan PIC ke vendor, ekspor WhatsApp dan PDF cue card, dua tampilan (tabel/timeline), kustomisasi sesi, tautan lagu ke agenda, catatan cue audio, moment tag adaptif.

---

## 2. Struktur Sub-Fitur

### Tab 1: Susunan Rundown (`?tab=rundown`)
- **Tabel mandiri per sesi** dengan penomoran tersendiri; berpindah sesi lewat tab sesi atau tampilan *stacked* semua sesi. Sesi dikelola di Edit Acara (Menu 01), bukan di sini; tab hanya mengambil daftar sesi.
- **4 kartu metrik**: total agenda, **total rentang waktu**, progres eksekusi live (selesai / berjalan / menunggu), PIC dan vendor unik.
- **Dua tampilan**: tabel profesional WO (No, Waktu, Durasi, Agenda, PIC/Vendor, Keterangan & Highlights, Status Live, Aksi) dan timeline kartu dengan drag handle.
- **Live Stage Manager**: `⚪ Belum` → `🟡 LIVE` → `🟢 Selesai`, mencatat jam mulai/selesai; beberapa agenda boleh live bersamaan. Pembaruan antarperangkat lewat polling 5 detik.
- **Vendor/PIC**: agenda dapat menunjuk vendor di Menu 04; mengklik namanya membuka WhatsApp yang sudah ternormalisasi.
- **Ekspor**: teks WhatsApp (per sesi atau semua) dan cue card PDF.

### Tab 2: Audio, Playlist & Arahan Teknis (`?tab=playlist`)
- Master playlist/sound cue yang adaptif untuk semua jenis acara (pernikahan: 30 lagu PDF; seminar: bumper, countdown, Indonesia Raya; peresmian: mars perusahaan; dll. — pengguna menambah sendiri; hanya pernikahan yang punya lagu bawaan).
- **4 metrik**: total koleksi, distribusi format (live band / digital / akustik / sound FX), momen terpetakan, kesiapan audio (`ready` / `needs_review` / `missing_file`).
- **Moment tag** dari registry jenis acara (satu set per jenis; pengguna boleh menambah tag).
- **Tautan lagu ↔ agenda** (`rundown_id`) agar Stage Manager dan soundman membaca agenda yang sama; **catatan cue** (titik mulai, fade, ducking, level volume); **Spotify/YouTube**; **Salin WA Soundman & Band** dan **PDF cue sheet**.

---

## 3. Penerapan 9 Jenis Acara

Sesi bawaan dan agenda template semua jenis ada di [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (bab 3 dan 7–14). Ringkasnya:

| Jenis | Sesi bawaan | Agenda template | Playlist bawaan |
|---|---|:-:|:-:|
| 💍 Pernikahan | Lamaran → Akad → Resepsi | 11 + 22 (PDF) | 30 lagu |
| 🕌 Khitanan | Prosesi Doa → Hiburan → Jamuan | 9 | – |
| 🍼 Aqiqah | Prosesi Aqiqah → Syukuran | 7 | – |
| 🏢 Peresmian | Pembukaan → Prosesi Peresmian → Ramah Tamah | 7 | – |
| 🚀 Launching | Opening → Peluncuran → Networking | 9 | – |
| 🎂 Ulang Tahun | Welcome → Acara Utama → After Party | 7 | – |
| 🎓 Wisuda | Prosesi Senat → Pelantikan → Foto Studio | 9 | – |
| 🎤 Seminar | Registrasi → Sesi Pagi → Coffee Break → Sesi Sore | 9 | – |
| ✨ Lainnya | Sesi 1 | 3 | – |


---

## 4. Integrasi (mekanisme nyata)

| Dari → ke | Mekanisme |
|---|---|
| 06 → 04 Vendor | `planner_rundowns.vendor_id` (FK komposit) → kontak WA darurat |
| 06 ← 01 Sesi | `session_id` → `planner_event_sessions` (master bersama; tanggal sesi dipakai header tab dan pesan WA) |
| 06 ← 02 Jadwal | tugas Adicara `wedding.adicara_technical_meeting` (H-7) dan `…gladi_resik` (H-1) |
| 06 → 01 Dashboard | provider `rundown` (jumlah sesi/agenda/lagu, sesi terdekat) |
| 06 ↔ 05 Tamu | tanggal dan jam sesi dipakai pesan undangan WhatsApp; tidak ada sinkronisasi dua arah lain |

## 5. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — KPI, sesi switcher dari master sesi, dua tampilan, tracker live, generator WA, tab audio, wireframe.
- ⚙️ [BE.md](BE.md) — migrasi 012, jam `TIME` dan agenda terbuka, rentang KPI, endpoint, master data PDF (11 + 22 agenda, 30 lagu).
