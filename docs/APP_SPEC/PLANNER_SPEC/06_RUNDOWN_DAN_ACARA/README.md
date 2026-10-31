# Menu 06: Rundown & Acara

> **Prinsip Perancangan:**
> 1. **Event-agnostic**: skema, endpoint, dan antarmuka tidak menulis keras sesi. Modul ini memakai **master sesi bersama** (`planner_event_sessions`, dimiliki Menu 01) yang juga menggerakkan filter di Menu 02–05 ([00_KONTRAK_BERSAMA bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)). Implementasi default difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Hal. 14–16); delapan jenis lain memakai agenda usulan Adicara ([TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)).
> 2. **Total waktu adalah rentang, bukan jumlah durasi.** Agenda paralel (sesi makan, hiburan, salaman) dan yang tumpang tindih tidak boleh dijumlahkan: Proyek Contoh = 100 + 120 + 225 = **445 menit**, sedangkan jumlah durasi 699. Jam dihitung sebagai menit, bukan aritmetika `time` ([BE bab 2.1](BE.md#21-waktu-durasi-dan-rentang)).
> 3. **Template ≠ data contoh**: wizard dan Proyek Contoh hanya memuat agenda dan lagu template (33 agenda, 30 lagu; semua `pending`, tanpa vendor dan tautan). Tidak ada overlay; angka hari-H di wireframe FE adalah ilustrasi ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).
> 4. **Tidak menulis ke tabel modul lain**: modul ini hanya membaca sesi (Menu 01) dan vendor (Menu 04). Tautan agenda↔vendor dan lagu↔agenda dijaga FK komposit `(…, project_id)`.
> 5. **Cetak lewat peramban, bukan PDF**: pustaka PDF belum disetujui ([kontrak 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)); versi awal menjanjikan ekspor PDF untuk cue card dan cue sheet, kini dihapus.
>
> **Status (2026-10-31):**
> - **Backend: sudah diimplementasikan (2026-10-31).** Modul `plannerrundown` (`backend/internal/modules/plannerrundown/`): 17 endpoint, provider `rundown`, applier `rundown` dan `playlist`, `replace_template` di tempat dengan `confirm`, dua teks WhatsApp, `openapi.yaml` (tag `Planner Rundown`). Migrasi `012_000_planner_rundown.sql` (identik dengan DDL di [BE bab 1](BE.md#1-skema-database)) mendapat dua indeks unik `template_key`; Menu 01 mendapat `ModeReplaceTemplate`, `Confirm`, dan `moment_tags`. Rangkuman, penyimpangan, dan hasil uji: [BE bab 7.3](BE.md#73-hasil-implementasi) dan [docs/backend/planner/2026-10-31-planner-menu-06.md](../../../backend/planner/2026-10-31-planner-menu-06.md). Respons HTTP asli **sama dengan fixture** di [fixtures/](fixtures/) (dua dikoreksi saat dicocokkan).
> - **Frontend: belum ada satu baris pun.** [FE.md](FE.md) adalah acuan pembangunannya; tipe datanya sudah dicocokkan dengan DTO Go yang dikompilasi (23 pasangan tipe, 118 jalur JSON, nol selisih).
> - **Audit spesifikasi** (2026-10-31) dilakukan dengan menjalankan SQL-nya terhadap PostgreSQL 16.2 (database sementara), membaca ulang PDF Hal. 14–16 dengan `pdfminer` dan membandingkannya secara terprogram dengan tabel master data, mengompilasi DTO Go, dan menjalankan kode acuan TypeScript ([BE bab 7.1](BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi), [FE bab 11](FE.md#11-pengujian)). Yang benar: DDL, 33 agenda dan 30 lagu (nol selisih dengan PDF), rentang 445 menit, 17 PIC unik. Yang salah atau kurang: DTO Go tidak dapat dikompilasi; rumus rentang salah untuk acara lewat tengah malam; ekspor PDF dijanjikan tanpa pustaka; `replace_template` memutus tautan lagu↔agenda diam-diam; migrasi 012 tanpa indeks unik `template_key` (pola `append` Menu 02–04 gagal); batas, bentuk respons, dan konkurensi tidak diatur; durasi di wireframe FE salah enam dari delapan baris; layout FE keliru (`AppLayout`, bukan `PlannerLayout`).
> - **Hasil kode acuan FE** (sebelum frontend dibangun): 19 tes `node:test` lulus, 36 dari 36 mutasi kode digagalkan, `tsc` strict bersih; lima fixture di [fixtures/](fixtures/) dibangkitkan dari data SQL audit dan cocok dengan DTO Go (dan, sejak modul dibangun, dengan respons HTTP asli). Yang belum ada: frontend dan BFF serta cetak di peramban.

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Rundown Acara Lamaran** (Hal. 14 kiri) | 11 agenda 09.00 – selesai; kolom No, Waktu, Acara, PIC, Keterangan, Highlights | Sesi `lamaran`: 11 agenda dengan teks persis PDF; agenda 11 terbuka ("10.40 - selesai") |
| **Rundown Resepsi dan Pernikahan** (Hal. 14 kanan) | 22 agenda 07.00–13.00 (Akad, lalu "Resepsi 09.15"); kolom No, Waktu, Acara, PIC, Keterangan | Sesi `akad` (agenda 1–8) dan `resepsi` (9–22, bernomor 1–14 di Adicara); total 33 agenda template |
| **Daftar lagu** (Hal. 15–16; judul tabel "LIST LAGU PERNIKAHAN", menu samping "PLAYLIST PERNIKAHAN") | 30 lagu (No, Judul, Penyanyi) | Tab Audio: 30 lagu; moment tag, format, dan status adalah tambahan Adicara |

Fakta PDF yang dijaga: beberapa agenda berjalan paralel (sesi makan, hiburan, salaman) dan sebagian tumpang tindih, sehingga **total durasi dihitung sebagai rentang** per sesi (Proyek Contoh: 100 + 120 + 225 = **445 menit**), bukan jumlah durasi. PIC yang kosong di PDF tetap kosong; template tidak memuat nama orang. Durasi baris adalah selisih jam (07.21–07.25 = 4 menit). Teks agenda sama dengan PDF kecuali enam nama agenda Resepsi yang berhuruf kecil di PDF (kini berhuruf awal kapital), "Iring -iring" → "Iring-iring", dan agenda 3 yang dipecah menjadi agenda + PIC "Qori" ([BE bab 5](BE.md#5-master-data)); dua ejaan penyanyi dikoreksi.

Yang ditambahkan Adicara: tracker live, tautan PIC ke vendor, ekspor teks WhatsApp dan cetak cue card lewat peramban, dua tampilan (tabel/timeline), kustomisasi sesi, tautan lagu ke agenda, catatan cue audio, moment tag adaptif.

---

## 2. Empat Mekanisme yang Dijaga

Empat hal di menu ini punya aturan yang terbukti dengan menjalankannya ([BE bab 2](BE.md#2-logika-bisnis)):

- **Rentang waktu dihitung dalam menit.** Per sesi: akhir mutlak terlambat − mulai paling awal; agenda terbuka dihitung dari jam mulainya; `ends_next_day` menambah 1.440 menit. Acara 22.00–23.00 dan 23.30–00.30 berentang 150 menit (rumus lama: 1.500 atau 60). Agenda yang *mulai* sesudah tengah malam belum dapat dimodelkan ([BE bab 9 butir 3](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Live Stage Manager: satu pernyataan untuk semua status.** `PATCH …/status` mengisi atau mengosongkan `started_at`/`completed_at` sehingga status dan cap tidak berselisih (basis data sendiri menerima `completed` tanpa `completed_at`); beberapa agenda boleh LIVE bersamaan; pembaruan antarperangkat lewat polling 5 detik, bukan SSE.
- **Template aman diulang.** `append` idempoten lewat indeks unik `(project_id, template_key)` (kini ada di migrasi 012); `replace_template` mengembalikan agenda template yang belum berjalan ke isi template **di tempat** dan mempertahankan `id`, vendor, serta tautan lagu; agenda berjalan, selesai, dan buatan pengguna tidak disentuh; butuh `confirm`.
- **Hapus agenda melepas tautan lagu, bukan menghapus lagu.** `rundown_id` menjadi kosong (FK `SET NULL`); hapus sesi menghapus agendanya (butuh `delete_agendas`) dan melepas semua tautannya.

---

## 3. Struktur Sub-Fitur

### Tab 1: Susunan Rundown (`?tab=rundown`)
- **Tabel mandiri per sesi** dengan penomoran tersendiri; berpindah sesi lewat tab sesi atau tampilan *stacked* semua sesi. Sesi dikelola di Edit Acara (Menu 01), bukan di sini; tab hanya mengambil daftar sesi.
- **4 kartu metrik**: total agenda, **total rentang waktu**, progres eksekusi live (selesai / berjalan / menunggu), PIC dan vendor unik.
- **Dua tampilan**: tabel profesional WO (No, Waktu, Durasi, Agenda, PIC/Vendor, Keterangan & Highlights, Status Live, Aksi) dan timeline kartu dengan drag handle (dan tombol naik/turun).
- **Live Stage Manager**: `⚪ Belum` → `🟡 LIVE` → `🟢 Selesai`, mencatat jam mulai/selesai; beberapa agenda boleh live bersamaan. Pembaruan antarperangkat lewat polling 5 detik (30 detik bila acara belum berlangsung).
- **Vendor/PIC**: agenda dapat menunjuk vendor di Menu 04; mengklik namanya membuka WhatsApp yang sudah ternormalisasi.
- **Ekspor**: teks WhatsApp (per sesi atau semua) yang disusun server, dan cetak cue card lewat peramban (tanpa PDF).

### Tab 2: Audio, Playlist & Arahan Teknis (`?tab=playlist`)
- Master playlist/sound cue yang adaptif untuk semua jenis acara (pernikahan: 30 lagu PDF; seminar: bumper, countdown, Indonesia Raya; peresmian: mars perusahaan; dll. — pengguna menambah sendiri; hanya pernikahan yang punya lagu bawaan).
- **4 metrik**: total koleksi, distribusi format (live band / digital / akustik / sound FX), momen terpetakan, kesiapan audio (`ready` / `needs_review` / `missing_file`).
- **Moment tag** dari registry jenis acara (satu set per jenis; pengguna boleh menambah tag); pencarian judul/artis/cue di klien.
- **Tautan lagu ↔ agenda** (`rundown_id`) agar Stage Manager dan soundman membaca agenda yang sama; **catatan cue** (titik mulai, fade, ducking, level volume); **Spotify/YouTube**; **Salin WA Soundman & Band** dan **cetak cue sheet** lewat peramban.

---

## 4. Penerapan 9 Jenis Acara

Sesi bawaan dan agenda template semua jenis ada di [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (bab 3 dan 7–14). Jumlah agenda di bawah dihitung dari tabel itu dan sama dengan registry kode untuk kunci sesinya (25 kunci pada sembilan jenis). Kunci template non-pernikahan `<kunci jenis>.agenda_<NN>` ([BE bab 2.6](BE.md#26-template)).

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

Moment tag audio per jenis: [TEMPLATE_EVENT bab 6](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#6-moment-tag-audio-satu-set-per-jenis-sumber-tunggal-untuk-06). Pada delapan jenis tanpa lagu bawaan, `apply-template` lagu dijawab `400 invalid_template` dan menu template lagu tidak tampil.

---

## 5. Integrasi (mekanisme nyata)

Mekanisme lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

| Dari → ke | Mekanisme |
|---|---|
| 06 → 04 Vendor | `planner_rundowns.vendor_id` (FK komposit) → kontak WA darurat; hanya dibaca (`LEFT JOIN`), vendor yang dihapus mengosongkan tautan |
| 06 ← 01 Sesi | `session_id` → `planner_event_sessions` (master bersama; tanggal sesi dipakai header tab dan pesan WA); hapus sesi dengan agenda butuh `delete_agendas` (`409 session_in_use` + `usage.agendas`) |
| 06 ← 02 Jadwal | tugas "Technical meeting" (H-7), "Gladi resik" (H-1), dan "Hari-H acara" ada di kode untuk **semua jenis**: `wedding.adicara_technical_meeting`, `wedding.adicara_gladi_resik`, `wedding.adicara_hari_h` (pernikahan) dan `general.technical_meeting_vendor_dan_panitia`, `general.gladi_resik`, `general.hari_h_acara` (delapan jenis lain); tidak ada sinkronisasi dua arah |
| 06 → 01 Dashboard | provider `rundown` → `RundownSummary` (jumlah sesi, agenda, selesai, berjalan, lagu; agenda yang sedang berjalan dan sesi terdekat); SQL-nya sudah dijalankan pada audit |
| 06, 05 ← 01 Sesi | Menu 05 membentuk pesan undangan WhatsApp dari tanggal, jam, dan lokasi **sesi master** (Menu 01), bukan dari agenda Menu 06. **Tidak ada tautan langsung 06 ↔ 05** (tidak ada kolom, endpoint, atau SQL yang menghubungkan keduanya); keduanya hanya sama-sama membaca sesi |

---

## 6. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — daftar isi, route dan komponen, alur data dan penanganan galat (partial untuk polling), prasyarat dari Menu 01 dan 04, tipe TypeScript yang cocok dengan DTO Go, fixture ([fixtures/](fixtures/)), kode acuan yang sudah dijalankan (`rundown.ts`, `rundownApi.ts`, `rundownMessages.ts`), wireframe dua tab dengan angka Proyek Contoh, alur interaksi, pesan galat, responsif/aksesibilitas/keamanan, pengujian, urutan kerja, dan celah.
- ⚙️ [BE.md](BE.md) — status dan hasil audit, migrasi 012 (identik) dan indeks yang disarankan, logika bisnis dengan SQL yang dijalankan (rentang, PIC unik, status, WhatsApp, urut ulang, template, KPI), 17 endpoint dengan bentuk respons dan validasi, DTO Go yang dikompilasi, master data PDF (11 + 22 agenda, 30 lagu), sambungan ke Menu 01, pengujian ✔/○, urutan pengerjaan, dan celah yang menunggu keputusan.
