# Menu 05: Buku Tamu & Undangan Digital

> **Prinsip Perancangan:** skema, endpoint, dan antarmuka **event-agnostic**; sisi tamu, kategori relasi, dan sesi dibaca dari registry dan master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Implementasi default difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Sheet *List Tamu*, Hal. 13).
> **Mode Standalone adalah mode utama yang bisa dibangun sekarang.** Modul Undangan Digital belum ada di backend dan spesifikasinya masih kosong; integrasi dengannya dibatasi pada dua antarmuka Go kecil ([BE bab 5](BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital)).

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

PDF Hal. 13 hanya berisi **dua tabel berdampingan**, *List Tamu Mempelai Laki-Laki* dan *List Tamu Mempelai Perempuan*, masing-masing dengan **enam penghitung kategori relasi** (Teman, Kolega, Keluarga, Tetangga, Teman Orang Tua, VIP; semuanya 0 pada PDF). Tidak ada nama tamu pada PDF.

| Elemen PDF | Di Adicara |
|---|---|
| Dua tabel Laki-Laki / Perempuan | `guest_side` = `pria` / `wanita` (label "Pihak Pria (CPP)" / "Pihak Wanita (CPW)" dari registry) |
| Enam penghitung kategori | enam kategori registry pernikahan (`teman`, `kolega`, `keluarga`, `tetangga`, `teman_ortu`, `vip`) sebagai pil penghitung interaktif |

Yang ditambahkan Adicara (PDF hanya catatan pasif): kuota pax, nomor meja, tautan undangan personal dan WhatsApp, pelacakan RSVP, proyeksi porsi katering, meja resepsionis (QR, check-in, souvenir), impor/ekspor. Penanganan jenis acara lain: pisah sisi dan kategori dari registry ([TEMPLATE_EVENT bab 4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori)).

---

## 2. Tiga Mode Layanan

Adicara menjual tiga layanan modular (Undangan Digital, Planner, Vendor/Gift) yang dapat dibeli sendiri atau digabung:

| Skenario | Mode Buku Tamu | Perilaku |
|---|---|---|
| **Bundling Undangan + Planner** | Terintegrasi (dual-key) | `planner_guests` dimiliki bersama; tamu yang ditambah di salah satu sisi muncul di sisi lain; tautan personal dan RSVP publik aktif. *Belum dapat dibangun: menunggu modul undangan.* |
| **Hanya Planner** | **Standalone** | Undangan cetak atau tautan eksternal; tetap ada kuota pax, meja, proyeksi katering, meja resepsionis; RSVP diubah manual; banner halus menawarkan Undangan Digital. |
| **Hanya Undangan Digital** | Fokus undangan | Dikelola modul undangan; tamu tanpa `project_id`. Planner tidak menampilkan fitur logistik. |

Model data bersama: satu tabel `planner_guests` dengan kunci ganda opsional (`project_id`, `invitation_id`, minimal salah satu). Planner tidak pernah menulis ke tabel undangan, dan modul undangan menulis RSVP lewat `GuestRSVPWriter`; tidak ada FK silang sampai migrasi modul undangan menambahkannya ([BE bab 5](BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital)).

---

## 3. Struktur Sub-Fitur

### Tab 1: Daftar Tamu & Undangan (`?tab=tamu`)
- **4 kartu metrik**: total undangan dan pax; porsi per pihak/sisi (hanya jenis yang punya sisi); distribusi undangan terkirim vs draf; respons RSVP — **empat status** (hadir, tidak hadir, ragu, belum merespons) yang jumlahnya selalu sama dengan total undangan.
- **Pemisah sisi** (dari registry; pernikahan `[Semua] [Pihak Pria] [Pihak Wanita]`, seminar `[Semua] [Panitia/Internal] [Peserta Eksternal]`, tanpa sisi → disembunyikan) dan **penghitung kategori** (6 kategori PDF untuk pernikahan).
- **Nomor meja** (`VIP-01`, `FAM-02`, `T-04`) dan kuota pax per undangan.
- **Tautan personal + WhatsApp 1-klik** (bila undangan terhubung) dengan templat pesan netral jenis acara; **Standalone** menampilkan tautan eksternal dan status kirim manual.
- **Impor CSV/Excel** dengan pratinjau (`dry_run`), deteksi duplikat, dan batas 5 MB/2.000 baris; **ekspor**.
- **Pagination server** (default 100/halaman) dengan penghitung yang dihitung atas seluruh data.

### Tab 2: Meja Resepsionis & Check-in Hari-H (`?tab=checkin`)
- Antarmuka cepat dan ramah sentuh untuk tablet/laptop panitia: 4 kartu metrik (tamu check-in, pax fisik, souvenir diserahkan beserta stok, belum hadir), **pemindai QR** (< 1 dtk), pencarian instan, filter (Semua / Sudah hadir / Belum hadir / VIP), **check-in 1 ketukan** (jam otomatis, pax riil, souvenir) yang aman terhadap ketukan ganda (`already_checked_in` menampilkan jam masuk sebelumnya), tombol *Batal*, **tamu walk-in**, ekspor rekap kehadiran.
- **Segar lewat polling** (5 detik), bukan SSE/WebSocket.
- Batasan: satu check-in per tamu.

---

## 4. Penerapan 9 Jenis Acara

Sisi dan kategori per jenis acara ada di registry (mis. khitanan/aqiqah: Keluarga Ayah/Ibu; peresmian dan launching: Internal/Eksternal; seminar: Panitia/Peserta; ulang tahun, wisuda, lainnya: tanpa sisi). Sesi kehadiran = master sesi proyek (pernikahan: Lamaran, Akad, Resepsi).

---

## 5. Integrasi (mekanisme nyata)

| Dari → ke | Mekanisme |
|---|---|
| 05 → 03 Anggaran | `GET /guests/catering-projection` (hanya tamu hadir; `maybe`/`unconfirmed` terpisah); tombol "Gunakan proyeksi RSVP" mengisi `quantity` pos katering berharga satuan atas aksi pengguna |
| 05 ↔ 04 Belanja | barang `purpose='souvenir'` (stok) vs `souvenir_received` (serah terima) |
| 05 ← 02 Jadwal | tugas Adicara `wedding.adicara_finalisasi_tamu` (H-45), `…sebar_undangan_digital` (H-30), `…followup_rsvp` (H-10) |
| 05 → 01 Dashboard | provider `guests` (sisi, kirim, empat status RSVP) |
| 05 ↔ Undangan Digital | `InvitationLookup` / `GuestRSVPWriter` (BE bab 5) |
| Amplop/angpau digital | **Tidak ada** di Planner; menunggu spesifikasi layanan Gift |

## 6. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — KPI tamu dan RSVP, pemisah sisi dinamis, penghitung kategori, tabel, modal, impor, meja resepsionis, wireframe dengan data contoh.
- ⚙️ [BE.md](BE.md) — migrasi 011, proyeksi katering yang diperbaiki, check-in atomik, impor, kontrak antarmuka undangan, 20 tamu contoh fiktif.
