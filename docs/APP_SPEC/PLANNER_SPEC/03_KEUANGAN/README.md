# Menu 03: Keuangan & Anggaran

> **Prinsip Perancangan:**
> 1. **Event-agnostic**: sesi, peran kontributor, dan template pos dibaca dari master sesi dan registry ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Pernikahan memakai preset dan pos dari PDF; jenis lain memakai template persentase pagu ([TEMPLATE_EVENT bab 15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#15-pos-anggaran-template-non-pernikahan)).
> 2. **Kebebasan penuh**: template hanya titik awal. Pengguna menambah, mengubah, menghapus, dan mengurutkan pos kapan saja; tidak ada komponen terkunci.
> 3. **Template ≠ data contoh**: template tidak pernah membawa pembayaran atau tabungan. Template pernikahan membuat 14 pos berestimasi 0; template jenis lain membuat pos berestimasi persentase dari pagu (0 bila pagu 0). Angka pembayaran dan tabungan PDF hanya ada di Proyek Contoh ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).
> 4. **Uang = rupiah utuh** (`BIGINT`/`int64`); semua angka efektif dibaca dari view `planner_budget_lines`; backend tidak mengirim persentase (klien menghitung, [kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)).
>
> **Status (2026-10-21):** **backend Menu 03 sudah diimplementasikan** (modul `plannerfinance`, 32 endpoint, provider `finance`, applier `budget`, overlay Proyek Contoh; [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-03.md), [BE](BE.md)); **frontend belum dibangun** (Menu 01 dan 02 punya backend juga). Widget Keuangan di dashboard terisi, wizard menawarkan template anggaran, dan Proyek Contoh memuat 14 pos PDF dengan pembayarannya, 22 setoran, dan tiga skenario (uji emas lulus). Spesifikasi diaudit dengan **menjalankan** SQL-nya terhadap PostgreSQL 16.2 sementara dan membaca ulang PDF Hal. 4–7 ([BE bab 7.1](BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi)), lalu kodenya diuji ([BE bab 7.2](BE.md#72-pengujian-pada-implementasi)). Blok DTO Go dan tipe TypeScript FE dicocokkan terhadap struct Go yang sebenarnya (29 pasangan tipe, 329 jalur, nol selisih) dan kode referensi FE lulus `tsc` strict dan 15 tes ([FE bab 12](FE.md#12-pengujian)). Yang belum ada: kode FE dan respons HTTP asli sebagai fixture ([fixtures/](fixtures/) masih dibangun dari basis data, bukan dari endpoint).

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

Tiga sheet PDF digabung menjadi tiga tab berurutan sesuai alur nyata pengguna:

| Sheet PDF | Isi PDF | Tab |
|---|---|---|
| **Skenario Budget** (Hal. 4) | 6 paket (10/20/30/50/75/100 jt), masing-masing 10 pos, total = nama paket | **Tab 1 Simulasi & Alokasi** — rencana dan plafon |
| **Target Tabungan** (Hal. 7) | setoran bulanan Feb–Des 2026 untuk Calon Mempelai Pria dan Wanita; terkumpul Rp 47,5 jt dari target Rp 75 jt | **Tab 2 Kas & Tabungan** — sumber modal |
| **Anggaran Pernikahan** (Hal. 5–6) | pos per kegiatan: Lamaran (5), Akad (1), Resepsi (8); kolom Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa; subtotal kegiatan dan grafik | **Tab 3 Realisasi & Pembayaran** — kas keluar |

**Angka acuan** (diuji; [BE bab 6](BE.md#6-master-data)): tagihan awal **Rp 74.150.000**; DP/Lunas 20.050.000; Termin 1 12.000.000; Termin 2 1.500.000; **dibayar Rp 33.550.000**; **sisa Rp 40.600.000** (45,2% terbayar). Lamaran 7.150.000 (dibayar 4.750.000, sisa 2.400.000); Akad 600.000 (lunas); Resepsi 66.400.000 (dibayar 28.200.000, sisa 38.200.000). Tabungan: Pria 30.000.000 + Wanita 17.500.000 = 47.500.000.

Yang tidak ada di PDF dan ditambahkan Adicara: multi-skenario (maks. 3) + perbandingan, peran kontributor di luar dua pihak, rekomendasi bulanan, milestone per sesi, Cashflow Bar, **status pelunasan per pos** (Belum/DP/Sebagian/Lunas), drawer pembayaran dengan bukti, pos berharga satuan, gulungan belanja, ekspor. Baris **"Subtotal Kategori"** pada PDF Hal. 5 (19.050.000 / 1.800.000 / 161.000.000 / 181.850.000) sengaja tidak dibawa: ia menjumlah enam baris yang tumpang-tindih (tagihan, DP, termin, dibayar, sisa) sehingga tidak bermakna.

```text
[Tab 1: Skenario Aktif] ──(estimasi per pos, kunci item_key)──┐
                                                              ├──> [Tab 3: Realisasi & Kontrol Kas]
[Tab 2: Kas & Tabungan] ──(total dana terkumpul)──────────────┘
```

| Tab | Peran | Pertanyaan |
|---|---|---|
| 1 Simulasi | Rencana dan plafon (*blueprint*) | "Berapa estimasi per pos?" |
| 2 Tabungan | Kas masuk | "Berapa uang yang terkumpul?" |
| 3 Realisasi | Kas keluar | "Bagaimana uang dibayarkan ke vendor?" |

---

## 2. Berpindah Skenario dengan Aman

Pengguna menyimpan paling banyak **3 skenario** (mencegah kelumpuhan keputusan; batas ditegakkan dengan kunci transaksi, terbukti: tanpa kunci dua permintaan bersamaan menghasilkan 4) dan berpindah kapan saja:

- **Smart Adjust (default)**: hanya `initial_estimate` pos yang cocok `(sesi, item_key)` diubah ke nilai skenario; pos baru ditambahkan; pos kustom dan pos yang tidak ada di skenario dibiarkan. DP/termin, tanggal, bukti, vendor, dan dokumen tertaut **tidak pernah disentuh**; `Sisa` dihitung ulang oleh view. Pratinjau perubahan ditampilkan sebelum konfirmasi. Menerapkan skenario yang sama dua kali tidak mengubah apa pun.
- **Reset bersih**: hapus semua pos dan pembayaran lalu salin skenario; wajib mengetik `RESET`; pratinjau menyebut berapa pos dan berapa rupiah pembayaran yang hilang.
- Opsi: *perbarui pagu proyek* (`target_budget` = total skenario; default aktif) dan *target tabungan ikut pagu*.
- Pos yang digulung dari daftar belanja (`rollup`) dan pos berharga satuan dilewati dan dilaporkan.

Satu sumber pagu: `planner_projects.target_budget` (diperbarui saat skenario diterapkan); total skenario adalah rencana, jumlah estimasi pos adalah kenyataan, dan selisihnya ("sisa plafon", boleh negatif = melebihi plafon) ditampilkan. Target tabungan = `savings_target` bila diisi, selain itu mengikuti `target_budget`; mengubahnya memakai `PATCH` proyek Menu 01 (`savings_target` / `clear_savings_target`), tidak ada endpoint target di menu ini. Skenario aktif hanya berubah lewat "Jadikan Skenario Aktif & Terapkan"; menghapus skenario aktif membuat proyek tanpa skenario aktif.

### Integrasi Tab 2 → Tab 3
Cashflow Bar di Tab 3: `Kas bebas = terkumpul (Tab 2) − terbayar (Tab 3)`; bila `sisa kewajiban > kas bebas`, peringatan defisit dan rekomendasi setoran. Data PDF: kas bebas 13.950.000, defisit 26.650.000. "Sisa kewajiban" menjumlah bagian positif per pos, jadi pos yang lebih bayar tidak menutupi pos lain yang kurang ([BE §3.5](BE.md#35-likuiditas)).

---

## 3. Struktur Tab

### Tab 1: Simulasi & Alokasi Budget (`?tab=simulasi`)
- **Pemilih skenario** (maks. 3; tombol tambah nonaktif bila penuh) dengan tanda ★ pada skenario aktif; aksi ganti nama, duplikasi, hapus.
- **Rangkaian acara yang disimulasikan**: kotak centang dari master sesi (pernikahan: Lamaran, Akad Nikah, Resepsi, + sesi kustom seperti Siraman/Ngunduh Mantu). Tanpa Lamaran cukup uncheck; sesi yang dicentang tetapi belum punya pos tampil kosong dengan ajakan menambah pos (template pos per sesi tersedia di Tab 3).
- **Preset siap pakai**: pernikahan memakai 6 paket PDF (10, 20, 30, 50, 75, 100 jt); jenis lain memakai template persentase. Input nominal bebas dan slider.
- **Donut** proporsi pos, **tabel pos per sesi** (tambah, ubah, hapus, kembalikan ke nilai preset), **bandingkan skenario** berdampingan, tombol **Jadikan Skenario Aktif & Terapkan**.

### Tab 2: Kas & Tabungan Acara (`?tab=tabungan`)
- **Pembukuan mandiri.** Adicara bukan lembaga keuangan, tidak menyediakan rekening/VA, dan tidak menampung dana; tab ini hanya catatan bersama. Kolom *catatan/lokasi simpanan* hanyalah pengingat pribadi.
- **KPI**: target, terkumpul, kekurangan, persentase capaian.
- **Rekomendasi bulanan** dan **milestone per sesi** dihitung dari tanggal sesi ([BE §3.6](BE.md#36-rekomendasi-tabungan-dan-milestone-bertahap)); sesi tanpa tanggal tidak membuat milestone.
- **Kontributor**: kartu ringkasan (nama, peran dari registry, total, porsi, jumlah setoran), tambah kontributor.
- **Dua tampilan setoran**: *Daftar* (tanggal, jumlah, penyetor, aksi; total di footer; filter per penyetor) dan *Matriks Bulanan* (bulan × kontributor, meniru PDF Hal. 7; bulan tanpa setoran tetap satu baris).
- **Grafik tren** saldo kumulatif menuju target dengan penanda tanggal sesi.

### Tab 3: Realisasi Anggaran & Pembayaran (`?tab=realisasi`)
- **Filter sesi** dari master sesi; **4 KPI** (total estimasi, terbayar, sisa, rasio pelunasan) dengan mikro-grafik; **Cashflow Bar**.
- **Matriks rekapitulasi sesi** meniru PDF Hal. 5–6 (baris: Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa; kolom: sesi + Total).
- **Tabel pos**: sesi, pos, vendor terkait, estimasi, DP, Termin 1, Termin 2, terbayar, sisa, status (`Belum`, `DP`, `Sebagian`, `Lunas`, `Lebih bayar`), aksi. Penanda sumber: *Dari vendor*, *Dari dokumen*, *Dihitung dari belanja* (pos `rollup`, terkunci untuk pembayaran).
- **Drawer pembayaran** (DP / Termin 1 / Termin 2 / Pelunasan Penuh; mengganti atau menghapus pembayaran yang salah catat), **harga satuan** (jumlah × harga untuk pos seperti katering), **terapkan template**, **ekspor `.xlsx`**.

---

## 4. Penerapan 9 Jenis Acara

Satu tabel registry; template yang tersedia menentukan apa yang ditawarkan wizard dan Tab 1. Pernikahan bersumber dari PDF; delapan lainnya adalah usulan Adicara ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)).

| Jenis | Peran kontributor | Preset skenario | Pos template | Estimasi pos template |
|---|---|---|:-:|---|
| 💍 Pernikahan | `groom` · `bride` · `family` · `sponsor` · `other` | 6 paket PDF `wedding_10jt` … `wedding_100jt` | 14 (Lamaran 5, Akad 1, Resepsi 8) | 0 |
| 🕌 Khitanan | `host` · `family` · `other` | `khitanan_pct` | 8 | persen × pagu |
| 🍼 Aqiqah | `host` · `family` · `other` | `aqiqah_pct` | 7 | idem |
| 🏢 Peresmian | `internal` · `sponsor` · `other` | `peresmian_pct` | 8 | idem |
| 🚀 Launching | `internal` · `sponsor` · `other` | `launching_pct` | 8 | idem |
| 🎂 Ulang Tahun | `host` · `family` · `other` | `birthday_pct` | 7 | idem |
| 🎓 Wisuda | `internal` · `family` · `sponsor` · `other` | `wisuda_pct` | 7 | idem |
| 🎤 Seminar | `internal` · `sponsor` · `other` | `seminar_pct` | 8 | idem |
| ✨ Acara Lainnya | `host` · `internal` · `sponsor` · `other` | `other_pct` | 6 | idem |

Template jenis non-pernikahan: pos di sesi `Umum` (`all`), nominal = `round(persen × pagu ÷ 100)` dibulatkan ke Rp 1.000, selisih pembulatan ke Dana darurat; preset `<jenis>_pct` memakai tabel persentase yang sama dan menolak total di bawah Rp 100.000. Tab 2 dan Tab 3 sama untuk semua jenis; hanya label peran dan nama sesi yang berbeda.

---

## 5. Integrasi Lintas Modul (mekanisme nyata)

Mekanisme lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

| Dari → ke | Mekanisme |
|---|---|
| 03 → 01 Dashboard | `SummaryProvider` `finance` (`FinanceSummary`: estimasi, terbayar, sisa, lebih bayar, tabungan, kas bebas, defisit, jumlah pos lunas/sebagian/belum); angka dibaca dari view yang sama dengan menu ini |
| 01 Wizard → 03 | `TemplateApplier` `budget` (14 pos pernikahan atau pos persentase) di dalam transaksi pembuatan proyek; `SampleOverlay` `finance` mengisi dataset PDF di Proyek Contoh |
| Sesi → 03 | master sesi: `sub_event_scope` pos dan item skenario (FK komposit); `active_sub_events` skenario dibersihkan saat sesi dihapus; `session_date` membentuk milestone tabungan |
| 02 Dokumen → 03 Pos | `planner_budget_items.admin_doc_id` + `POST …/admin-docs/:id/sync-budget` (SQL langsung ke tabel 03; pengecualian kontrak bab 1.1) |
| 04 Vendor → 03 Pos | `lock-deal`: menautkan atau membuat pos tanpa duplikasi (kunci `item_key`); pos bersama beberapa kategori tidak ditimpa |
| 03 Skenario → 04 Vendor | skenario aktif memberi pagu vendor lewat `category_key → budget_item_keys`; dibaca setiap kali daftar vendor dibuka (tidak disalin ke tabel vendor, jadi ikut berganti bersama skenario) |
| 04 Belanja → 03 Pos | `planner_shopping_items.budget_item_id` + `rollup_source='shopping'`; angka dihitung view, bukan salinan; pembayaran pos itu terkunci |
| 05 Tamu → 03 Pos | proyeksi porsi dari RSVP mengisi `quantity` pos berharga satuan atas aksi pengguna (tidak otomatis) |
| 03 → 01 Proyek | opsi Apply skenario menulis `target_budget` dan `savings_target` (SQL langsung; pengecualian ketiga, [BE bab 9 butir 4](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)); target tabungan lain lewat `PATCH` proyek |

Penerimaan angpau/amplop digital **tidak** masuk Tab Tabungan.

## 6. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — daftar isi, route dan komponen, alur data dan penanganan galat, tipe TypeScript yang cocok dengan DTO Go, fixture ([fixtures/](fixtures/)), kode acuan yang sudah dijalankan (`finance.ts`, `financeApi.ts`, `messages.ts`), wireframe tiga tab dengan angka PDF, alur interaksi, pesan galat, responsif/aksesibilitas/keamanan, pengujian, urutan kerja, dan celah.
- ⚙️ [BE.md](BE.md) — status dan hasil audit, migrasi 008 dan 010 (sudah ada), kunci `item_key`, logika bisnis dengan SQL yang dijalankan (Smart Adjust, pembayaran, likuiditas, milestone, matriks, template), endpoint (32) dengan bentuk respons dan validasi, DTO Go, master data (6 preset, 14 pos template, dataset PDF, tabungan 22 setoran, skenario contoh), pengujian ✔/✘, urutan pengerjaan, dan celah yang menunggu keputusan.
