# Menu 05: Buku Tamu & Undangan Digital

> **Prinsip Perancangan:**
> 1. **Event-agnostic**: pihak (sisi) tamu, kategori relasi, dan sebutan "Tamu"/"Peserta" dibaca dari registry jenis acara; sesi dari master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Pernikahan memakai `Wedding_Planner.pdf` (*List Tamu*, Hal. 13); delapan jenis lain memakai sisi dan kategori usulan Adicara ([TEMPLATE_EVENT 4 dan 4.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori)) dan tidak membawa tamu bawaan.
> 2. **Mode Standalone adalah mode yang bisa dibangun sekarang.** Modul Undangan Digital belum ada di backend dan spesifikasinya masih kosong; integrasi dengannya dibatasi pada dua antarmuka Go kecil ([BE bab 5](BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital)). Standalone tidak kehilangan fitur logistik: kuota pax, meja, WhatsApp, proyeksi katering, meja resepsionis.
> 3. **Tanpa salinan angka dan tanpa menulis ke tabel modul lain**: seluruh penghitung (kartu, pil pihak, pil kategori, RSVP) dihitung dari baris tamu oleh satu fungsi yang dipakai bersama dashboard dan halaman; proyeksi porsi dibaca Menu 03 lewat endpoint (bukan ditulis ke tabel pos); satu-satunya akses lintas tabel adalah `SELECT` baca-saja atas stok souvenir di `planner_shopping_items` Menu 04, dan `TestSouvenirStockAgreesWithMenu04` menjaga hasilnya sama dengan KPI Menu 04 (Menu 04 menjumlah di Go, Menu 05 di SQL).
> 4. **Template ≠ data contoh**: wizard tidak membuat tamu. Dua puluh tamu fiktif ada hanya di Proyek Contoh ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).
> 5. **Data pribadi**: nama dan telepon tamu hanya terbaca oleh pemilik proyek; Planner tidak punya endpoint publik. RSVP publik masuk lewat modul undangan.
>
> **Status (2026-10-25):**
> - **Backend: sudah diimplementasikan** (2026-10-25; [BE.md](BE.md), [dokumen modul](../../../backend/planner/2026-10-25-planner-menu-05.md)): modul `plannerguest`, 19 endpoint, provider dan overlay `guests`, registry `GuestSides`/`GuestCategories`, `openapi.yaml` (tag `Planner Guest`). Tabel `planner_guests` sudah ada sejak migrasi `011_000_planner_guests.sql` (identik dengan DDL di [BE bab 1](BE.md#1-skema-database); tidak ada migrasi baru). Diuji terhadap PostgreSQL 16.2: 448 tes lulus di seluruh backend (114 baru), `-race`, cakupan 91,9%, 33 mutasi kode produksi (kunci proyek dan batas, SQL antrean dan KPI, cap RSVP, stok souvenir, urutan daftar, pembatas laju, batas badan impor, awalan rumus, zona jam masuk, dan lainnya) semuanya digagalkan tes; dua awalnya selamat (batas badan impor dan zona jam masuk, mesin uji kebetulan berzona WIB) dan menambah dua tes, empat tidak terkompilasi dan ditulis ulang ([BE bab 8.4](BE.md#84-hasil-implementasi-2026-10-25)); **respons HTTP asli sama dengan keempat fixture FE** dan setiap baris katalog galat FE bab 4.6 dihasilkan backend.
> - **Frontend: belum ada satu baris pun.** [FE.md](FE.md) adalah acuan pembangunannya; tipe datanya sudah dicocokkan dengan DTO Go yang dikompilasi (21 pasangan tipe, 125 jalur JSON, nol selisih).
> - **Audit spesifikasi** dilakukan dengan menjalankan SQL-nya terhadap PostgreSQL 16.2 dan backend Menu 01–04 yang sebenarnya, membaca ulang PDF Hal. 13 dengan `pdfminer`, dan menjalankan kode acuan Go dan TypeScript ([BE bab 8.1](BE.md#81-yang-sudah-dijalankan-pada-audit-spesifikasi), [FE bab 11](FE.md#11-pengujian)). DDL dan angka contoh ternyata benar; yang salah adalah janji dan perilaku di sekitarnya: ekspor PDF yang pustakanya belum disetujui, impor `multipart` yang ditolak BFF, tautan personal tanpa identitas tamu (RSVP publik tidak dapat dikaitkan), FK undangan yang membuat hapus undangan gagal, pembulatan porsi dengan `float64`, penghitung "belum hadir" yang salah bila tamu yang menolak tetap datang, persentase di respons, templat impor tanpa endpoint, dan penghitung kategori yang tidak per pihak seperti PDF. Satu temuan di luar folder ini: Nginx mengirim `Permissions-Policy: camera=()`, yang memblokir pemindai QR di produksi ([CHANGELOG spesifikasi](../CHANGELOG.md)).
> - **Hasil kode acuan** (sebelum modul dibangun): 43 tes Go dan 28 tes TypeScript lulus, 24 + 24 mutasi kode semuanya digagalkan, fixture dibangkitkan dari tabel contoh dan angkanya sama dengan SQL ([BE bab 8.3](BE.md#83-hasil-kode-acuan-2026-10-25)). Yang belum ada: frontend dan BFF, modul Undangan Digital, dan `.xlsx` yang dibuka di Excel/Numbers. **Prasyarat produksi yang belum dilakukan:** Nginx mengirim `Permissions-Policy: camera=()` (`deploy/nginx/adicara.conf`) sehingga pemindai QR tidak jalan sampai menjadi `camera=(self)`.

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

PDF Hal. 13 (dibaca ulang dengan `pdfminer`) hanya berisi **dua tabel berdampingan**, *LIST TAMU MEMPELAI LAKI-LAKI* dan *LIST TAMU MEMPELAI PEREMPUAN*, masing-masing dengan **enam penghitung kategori relasi** (Teman, Kolega, Keluarga, Tetangga, Teman Orang Tua, VIP) yang semuanya 0. Tidak ada nama tamu pada PDF.

| Elemen PDF | Di Adicara |
|---|---|
| Dua tabel Laki-Laki / Perempuan | `guest_side` = `pria` / `wanita` (label "Pihak Pria (CPP)" / "Pihak Wanita (CPW)" dari registry) |
| Enam penghitung **di setiap tabel** | enam kategori registry pernikahan (`teman`, `kolega`, `keluarga`, `tetangga`, `teman_ortu`, `vip`) sebagai pil penghitung interaktif **per pihak**: memilih "Pihak Pria" menampilkan penghitung tabel pria (sel `side_categories`, [BE §2.8](BE.md#28-ringkasan-penghitung-kategori-dan-pihak)); "Semua" menjumlahkannya |

Yang ditambahkan Adicara (PDF hanya catatan pasif): kuota pax, nomor meja, pesan WhatsApp, pelacakan RSVP, proyeksi porsi katering, meja resepsionis (QR, check-in, walk-in, souvenir), impor/ekspor. Penanganan jenis acara lain: pisah sisi dan kategori dari registry ([bab 5](#5-penerapan-9-jenis-acara)).

```text
[Menu 05 Tamu] ──GET catering-projection (klien)──> [Menu 03 modal pos] ──PUT pos (aksi pengguna)──> [Menu 03 pos katering]
[Menu 04 Belanja] ──stok souvenir (baca-saja)──> [Menu 05 Meja Resepsionis]
[Modul Undangan] ──ApplyRSVP(token)──> [Menu 05 tamu]      [Menu 05] ──InvitationLookup.PublicSlug──> tautan personal
```

---

## 2. Tiga Mode Layanan

Adicara menjual tiga layanan modular (Undangan Digital, Planner, Vendor/Gift) yang dapat dibeli sendiri atau digabung:

| Skenario | Mode Buku Tamu | Perilaku |
|---|---|---|
| **Bundling Undangan + Planner** | Terintegrasi (dual-key) | `planner_guests` dimiliki bersama; tamu yang ditambah di salah satu sisi muncul di sisi lain; tautan personal (`?g=<token>&to=<nama>`) dan RSVP publik aktif. *Belum dapat dibangun: menunggu modul undangan.* |
| **Hanya Planner** | **Standalone** | Undangan cetak atau pesan WhatsApp biasa (tanpa tautan); tetap ada kuota pax, meja, proyeksi katering, meja resepsionis; RSVP diubah manual; banner halus memberi tahu Undangan Digital belum tersedia. |
| **Hanya Undangan Digital** | Fokus undangan | Dikelola modul undangan; tamu tanpa `project_id`. Planner tidak menampilkan fitur logistik. |

Model data bersama: satu tabel `planner_guests` dengan kunci ganda opsional (`project_id`, `invitation_id`, minimal salah satu). Planner tidak pernah menulis ke tabel undangan, dan modul undangan menulis RSVP lewat `GuestRSVPWriter` berbasis token tiket; tidak ada FK silang sampai migrasi modul undangan menambahkannya, dengan aturan hapus yang dibuktikan di [BE bab 5.3](BE.md#53-migrasi-modul-undangan).

---

## 3. Empat Mekanisme yang Dijaga

Empat hal di menu ini punya aturan yang terbukti dengan menjalankannya ([BE bab 2](BE.md#2-logika-bisnis)):

- **Proyeksi katering menghitung hanya tamu yang hadir, dengan bilangan bulat.** Tamu `declined` tidak pernah masuk `confirmed_pax`; `maybe`/`unconfirmed` dilaporkan terpisah (`potential_pax`) dan tidak masuk rekomendasi. Rekomendasi = `ceil(confirmed × (100 + persen) / 100) + porsi`; versi `float64` menambah satu porsi pada 663 dari 303.101 kombinasi. Hasilnya tidak mengubah anggaran sendiri: modal pos Menu 03 mengisi `quantity` atas tindakan pengguna (dan dinonaktifkan pada 0 porsi karena Menu 03 menolak `quantity` 0).
- **Meja resepsionis: check-in atomik dan angka yang konsisten.** Check-in adalah satu `UPDATE … WHERE is_checked_in = FALSE` (dua tablet pada tamu yang sama: satu sukses, satu `409 already_checked_in` yang menampilkan jam masuk). Check-in **tidak mengubah RSVP**; tamu yang menolak tetapi datang dihitung hadir (`expected = bukan penolak ∨ hadir`), sehingga "Belum hadir" tidak pernah salah satu (aturan lama: 6 padahal 7) dan persentase tidak melewati 100%. RSVP tidak dapat diubah sesudah check-in.
- **Tautan personal adalah kredensial.** `?g=<token 128 bit>&to=<nama>`: RSVP publik dan tiket QR memakai token yang sama, tidak dapat ditebak, dan terikat ke undangan (`ApplyRSVP(invitationID, token)`). Versi sebelumnya hanya membawa nama, sehingga RSVP tidak dapat dikaitkan ke tamu tertentu.
- **Berkas impor tidak dipercaya.** Dikirim sebagai JSON base64 (BFF hanya meneruskan JSON); CSV dari Excel Indonesia (BOM, titik koma, Windows-1252) dibaca benar; `.xlsx` dibatasi ekstraksinya (unggahan 1 MiB yang mengklaim 300 MiB ditolak seketika); nomor telepon yang kehilangan `0` karena Excel ditolak, tidak ditebak; pratinjau `dry_run` tidak menulis apa pun; semua-atau-tidak sama sekali.

---

## 4. Struktur Sub-Fitur

### Tab 1: Daftar Tamu & Undangan (`?tab=tamu`)
- **4 kartu metrik**: total undangan dan pax; porsi per pihak (hanya jenis yang punya sisi); terkirim vs belum; respons RSVP — **empat status** (hadir, ragu, tidak hadir, belum merespons) yang jumlahnya selalu sama dengan total undangan. Semuanya seluruh proyek, bukan halaman.
- **Pemisah pihak** dari registry (pernikahan `[Semua] [Pihak Pria (CPP)] [Pihak Wanita (CPW)]`, seminar `[Semua] [Panitia/Internal] [Peserta Eksternal]`, tanpa sisi → disembunyikan) dan **penghitung kategori per pihak** (enam kategori PDF untuk pernikahan; kategori kustom hanya bila terpakai).
- **Nomor meja** (`VIP-01`, `FAM-02`, `T-04`) dan kuota pax per undangan.
- **WhatsApp 1-klik** (pesan dibentuk klien dari nomor ternormalisasi; dengan tautan personal bila undangan terhubung, tanpa tautan pada Standalone) dan status kirim yang diubah manual ("Tandai terkirim").
- **Impor CSV/Excel** dengan templat dari proyek (dropdown pihak, kategori, sesi), pratinjau (`dry_run`), deteksi duplikat, batas 5 MB/2.000 baris; **ekspor** `.xlsx`.
- **Pagination server** (100 per halaman); pencarian nama, meja, atau nomor; semua filter di URL.

### Tab 2: Meja Resepsionis & Check-in Hari-H (`?tab=checkin`)
- Antarmuka cepat dan ramah sentuh untuk tablet/laptop panitia: 4 kartu metrik (tamu check-in, pax fisik, souvenir diserahkan beserta stok, belum hadir), **pemindai QR** (token mentah atau tautan personal), pencarian instan, chip Semua / Sudah hadir / Belum hadir dan dropdown kategori, **check-in 1 ketukan** (jam otomatis, pax riil, souvenir) yang aman terhadap ketukan ganda, tombol *Batal*, **tamu walk-in**, ekspor rekap kehadiran `.xlsx`.
- **Segar lewat polling** (5 detik), bukan SSE/WebSocket; satu panggilan membawa KPI, hitungan filter, 10 check-in terbaru, dan satu halaman antrean (50).
- Batasan: satu check-in per tamu. Pemindai kamera membutuhkan HTTPS dan `Permissions-Policy: camera=(self)`; tanpa kamera tersedia jalur ketik nama/meja/kode.

---

## 5. Penerapan 9 Jenis Acara

Satu tabel registry ([kontrak bab 3](../00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data)); label sisi dan kategori: [TEMPLATE_EVENT 4.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#42-label-sisi-dan-kategori-tamu-milik-menu-05). Pernikahan bersumber dari PDF; delapan lainnya adalah usulan Adicara. Sesi kehadiran = master sesi proyek (pernikahan: Lamaran, Akad Nikah, Resepsi); jenis tanpa sisi tidak menampilkan pemisah pihak dan kolom "Pihak". Kata benda tamu dan judul menu mengikuti `labels` registry (seminar: "Peserta" dan "Peserta & RSVP"; kata teknis "undangan" menjadi "pendaftaran").

| Jenis | Sebutan | Sisi tamu | Kategori tamu |
|---|---|---|---|
| 💍 Pernikahan | Tamu | `pria`, `wanita` | `teman`, `kolega`, `keluarga`, `tetangga`, `teman_ortu`, `vip` |
| 🕌 Khitanan | Tamu | `keluarga_ayah`, `keluarga_ibu` | `keluarga`, `tetangga`, `teman`, `tokoh_agama`, `vip` |
| 🍼 Aqiqah | Tamu | `keluarga_ayah`, `keluarga_ibu` | `keluarga`, `tetangga`, `teman`, `tokoh_agama` |
| 🏢 Peresmian | Tamu | `internal`, `eksternal` | `pimpinan_vip`, `mitra`, `media`, `karyawan`, `warga_sekitar`, `umum` |
| 🚀 Launching | Tamu | `internal`, `eksternal` | `media`, `kreator`, `mitra`, `klien`, `vip`, `umum` |
| 🎂 Ulang Tahun | Tamu | – | `keluarga`, `teman_sekolah`, `teman_kuliah`, `komunitas`, `vip` |
| 🎓 Wisuda | Tamu | – | `wisudawan`, `orang_tua`, `dosen_pejabat`, `tamu_undangan`, `vip` |
| 🎤 Seminar | Peserta | `internal`, `eksternal` | `pembicara`, `peserta`, `sponsor`, `media`, `vip` |
| ✨ Acara Lainnya | Tamu | – | `umum`, `vip` |

Pengguna selalu dapat mengetik kategori sendiri (disimpan sebagai teks kustom) atau membiarkannya kosong (kunci cadangan `lainnya`, bukan kategori registry). Sisi tidak dapat dibuat pengguna: hanya dari registry.

---

## 6. Integrasi Lintas Modul (mekanisme nyata)

Mekanisme lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

| Dari → ke | Mekanisme |
|---|---|
| 05 → 03 Anggaran | `GET …/guests/catering-projection?buffer_percent=&buffer_pax=` dipanggil **klien** dari modal pos Menu 03; tombol "Gunakan proyeksi RSVP" mengisi `quantity` pos berharga satuan atas aksi pengguna lewat `PUT …/budget/items/:id` dengan semua isian (termasuk `vendor_id`). Tidak ada SQL antar modul ([BE §2.3](BE.md#23-proyeksi-porsi-katering-diperbaiki)) |
| 05 ↔ 04 Belanja | barang `purpose='souvenir'` (stok = `floor(Σ quantity)` barang selesai) vs `souvenir_received` (serah terima, satuan per undangan); Menu 05 membaca stok dengan SQL baca-saja ([BE §2.5](BE.md#25-meja-resepsionis-hari-h)) |
| 05 ← 02 Jadwal | tugas Adicara pernikahan `wedding.adicara_finalisasi_tamu` (H-45), `wedding.adicara_sebar_undangan_digital` (H-30), `wedding.adicara_followup_rsvp` (H-10); hanya pernikahan memilikinya (jenis lain memakai jadwal `general` tanpa tugas tamu khusus) |
| 05 → 01 Dashboard | `SummaryProvider` `guests` → `domain.GuestSummary` (sisi dengan label registry, terkirim, empat status RSVP); kartu Hub memakai `guest_pax_count` (pax diundang) |
| 01 → 05 Sesi | hapus sesi memindahkan tamu ke `move_to`; hitungan `usage.guests` per sesi |
| 05 ↔ Undangan Digital | `InvitationLookup` / `GuestRSVPWriter` berbasis token, tautan `?g=<token>`; FK undangan dan aturan hapus di [BE bab 5](BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital) |
| Amplop/angpau digital | **Tidak ada** di Planner; menunggu spesifikasi layanan Gift |

---

## 7. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — daftar isi, route dan komponen, alur data dan penanganan galat, prasyarat dari Menu 01 dan 04, tipe TypeScript yang cocok dengan DTO Go, fixture ([fixtures/](fixtures/)), kode acuan yang sudah dijalankan (`guest.ts`, `guestApi.ts`, `guestMessages.ts`, `guestProps.ts`), wireframe dua tab dengan angka Proyek Contoh, alur interaksi, pesan galat, responsif/aksesibilitas/keamanan (termasuk kamera dan Nginx), pengujian, urutan kerja, dan celah.
- ⚙️ [BE.md](BE.md) — status dan hasil audit, migrasi 011 (sudah ada), logika bisnis dengan SQL yang dijalankan (tautan personal, RSVP, proyeksi, WhatsApp, meja resepsionis, impor/ekspor/templat, pencarian, ringkasan), 19 endpoint dengan bentuk respons dan validasi, DTO Go yang dikompilasi, kontrak antarmuka undangan, 20 tamu contoh fiktif, sambungan ke Menu 01, pengujian ✔/○, urutan pengerjaan, dan celah yang menunggu keputusan.
