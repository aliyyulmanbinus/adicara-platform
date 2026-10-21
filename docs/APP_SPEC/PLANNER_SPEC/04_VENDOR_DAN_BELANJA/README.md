# Menu 04: Vendor & Belanja Kebutuhan

> **Prinsip Perancangan:** skema, endpoint, dan antarmuka **event-agnostic**; kategori vendor, kategori belanja, dan sesi dibaca dari registry dan master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Implementasi default difokuskan pada **Pernikahan** berdasarkan `Wedding_Planner.pdf` (Sheet *Kontak Vendor* Hal. 12 dan *List Seserahan* Hal. 9–10). Jenis acara lain memakai kategori dari registry ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)); tidak ada daftar belanja bawaan selain pernikahan.

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Kontak Vendor** (Hal. 12) | *List Rencana Vendor* dan *Fix Kerjasama Vendor* untuk 10 kategori: Venue, Decoration, Wedding Organizer, MUA, Catering, Photography, Attire, Hand Bouquet, MC, Entertainment; 7 berstatus FIX (kolom Nama Vendor, No Tlp, PIC, Action); data baris hanya contoh dummy | **Tab 1 Direktori & Kontak Vendor**: pipeline `rencana → negosiasi → fix` dengan checklist 10 kategori (7 dari 10 terkunci pada PDF); data dummy PDF tidak disalin |
| **List Seserahan** (Hal. 9–10) | 37 barang, kolom Kebutuhan, Link Pembelian, Harga, Kategori (6), Status (Selesai/On Proses); estimasi biaya Rp 4.578.500 | **Tab 2 Belanja Pengadaan & Logistik**: 37 barang dengan nama, kategori, dan harga persis PDF; tautan pribadi pembuat PDF tidak disalin |

Subtotal kategori (diuji): Perangkat Alat Solat 4 item **597.300** · Make Up & Skin Care 14 item **1.268.900** · Peralatan Mandi 8 item **990.700** · Pakaian Dalam 4 item **517.400** · Pakaian Luar 5 item **820.200** · Kebutuhan Lain 2 item **384.000** · total **4.578.500**; PDF menandai 16 barang selesai (Rp 2.063.800) dan 21 on proses (Rp 2.514.700).

Tambahan Adicara: pagu vendor dari skenario, kunci kerja sama ke anggaran tanpa pos ganda, tautan WhatsApp, jumlah × harga satuan, penanda souvenir, gulungan total belanja ke pos anggaran, ekspor.

---

## 2. Struktur Sub-Fitur

### Tab 1: Direktori & Kontak Vendor (`?tab=vendor`)
- **Checklist kategori** dari registry (+ kategori kustom yang dipakai): tiap kategori berstatus *Terkunci* (ada vendor `fix`), *Berjalan* (ada kandidat), atau *Kosong*. KPI "n dari m kategori terkunci" menghitung kategori, bukan baris vendor, dan tidak menghitung kategori yang ditandai "tidak diperlukan" (`skipped_vendor_categories` pada proyek).
- **Filter sesi** (master sesi), status pipeline, dan kategori; tabel berisi sesi, kategori, vendor + portofolio, PIC + tombol WA, pagu, deal, selisih, status bayar, aksi.
- **Pagu dari skenario aktif** ([BE §2.1](BE.md#21-pagu-vendor-dari-skenario-aktif)): kategori yang berbagi satu pos anggaran (pernikahan: Venue + Decoration; MUA + Attire; MC + Entertainment) dibandingkan di tingkat kelompok pagu (mis. Rp 12 jt + Rp 8 jt terhadap pagu Rp 20 jt); selisih hijau (hemat) atau merah (melebihi pagu) hanya tampil bila pagu dan penawaran ada.
- **Kunci Kerja Sama (FIX)** membuka dialog harga final dengan opsi *Sinkronkan ke anggaran*: menautkan pos khusus yang cocok atau membuat pos baru, tanpa menggandakan; pos yang dipakai bersama beberapa kategori (mis. Sewa Gedung & Dekorasi) tidak ditimpa dan hanya ditandai, dengan tombol untuk menyamakan estimasi dengan total deal kelompok ([BE §2.2](BE.md#22-kunci-kerja-sama-lock-deal-tanpa-pos-ganda)).
- **Status bayar** dibaca dari pos anggaran (Menu 03 **Tab 3**): `Belum DP` → `DP / sebagian` → `Lunas`.
- **WhatsApp 1-klik** dengan nomor ternormalisasi (`628…`) dan templat pesan; **ekspor** kontak vendor `fix` (`.xlsx`/PDF) untuk WO/panitia.

### Tab 2: Belanja Pengadaan & Logistik (`?tab=belanja`)
- **Checklist barang**: sesi, kategori, nama barang, **jumlah dan satuan**, harga satuan estimasi dan riil, alamat pembelian (teks bebas; URL dirender sebagai tautan `http/https`), status `belum → proses → selesai` (menandai selesai mengisi harga riil dengan estimasi bila kosong).
- **Perbandingan estimasi vs riil** per barang dan di footer; **KPI** total estimasi, realisasi (hanya barang selesai), selesai/proses.
- **Gulungan ke anggaran (*rollup*)**: tombol "Hitung pos anggaran dari daftar belanja" menandai pos (pernikahan: Seserahan) sehingga estimasi dan terbayarnya otomatis dari tabel ini lewat view; bukan salinan angka. Bila pos sudah punya pembayaran, pengguna diminta konfirmasi ([BE §2.4](BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup)).
- **Souvenir**: barang bertanda souvenir membentuk stok yang dibandingkan dengan serah terima di meja resepsionis (Menu 05).
- **Terapkan template**: pernikahan menyediakan 37 barang seserahan (`status belum`); jenis lain tidak punya barang bawaan, hanya kategori.
- **Ekspor** rekap `.xlsx`/PDF.

---

## 3. Penerapan 9 Jenis Acara

Kategori vendor dan belanja per jenis berasal dari registry ([TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)): pernikahan (10 kategori vendor PDF; 6 kategori belanja PDF), khitanan, aqiqah, peresmian, launching, ulang tahun, wisuda, seminar, dan acara lainnya. Pengguna selalu dapat menambah kategori sendiri (`category_key` berupa slug; `category_label` tampilan).

## 4. Integrasi (mekanisme nyata)

| Dari → ke | Mekanisme |
|---|---|
| 04 Vendor → 03 Anggaran | `lock-deal`: tautkan/buat pos; `planner_budget_items.vendor_id` unik per vendor |
| 03 Skenario → 04 Vendor | `budget_limit` dari skenario aktif lewat `category_key → budget_item_keys` |
| 04 Belanja → 03 Anggaran | `shopping_items.budget_item_id` + `budget_items.rollup_source='shopping'` + view |
| 04 Belanja ↔ 05 Tamu | `purpose='souvenir'` vs `souvenir_received` |
| 04 Vendor ← 02 Tugas, 06 Agenda | `planner_tasks.vendor_id`, `planner_rundowns.vendor_id` (kontak WA darurat) |
| 04 → 01 Dashboard | provider `vendors` (kategori terkunci, kandidat berjalan) |

## 5. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — KPI dan checklist kategori, tabel vendor, modal vendor, tabel belanja dengan jumlah/satuan/rollup, wireframe dengan data PDF.
- ⚙️ [BE.md](BE.md) — migrasi 006 dan 009, pagu dan `lock-deal` tanpa duplikasi, `NormalizeWA`, rollup, endpoint, master data (10 kategori, 10 vendor contoh fiktif, 37 barang).
