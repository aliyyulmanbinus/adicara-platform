# Menu 04: Vendor & Belanja Kebutuhan

> **Prinsip Perancangan:**
> 1. **Event-agnostic**: kategori vendor, kategori belanja, pemetaan kategori ke pos anggaran, dan pos gulungan belanja dibaca dari registry jenis acara; sesi dari master sesi ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Pernikahan memakai data PDF (*Kontak Vendor* Hal. 12 dan *List Seserahan* Hal. 9–10); delapan jenis lain memakai kategori usulan Adicara ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)) dan tidak membawa barang belanja bawaan.
> 2. **Tanpa salinan angka**: pagu vendor dibaca langsung dari skenario aktif Menu 03, status bayar dari pos anggaran yang bertaut, dan angka pos hasil gulungan dihitung view `planner_budget_lines` dari daftar belanja. Tidak ada kolom yang menyimpan angka milik modul lain, jadi tidak ada yang bisa basi.
> 3. **Template ≠ data contoh**: template pernikahan membuat 37 barang seserahan berstatus `belum` tanpa harga riil dan tanpa tautan; tidak ada template vendor. Sepuluh vendor contoh, status barang, dan tautan pos hanya ada di Proyek Contoh ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).
> 4. **Uang = rupiah utuh** (`BIGINT`/`int64`); jumlah barang dua desimal dihitung dengan aritmetika bulat dan dibatasi sebelum menyentuh basis data; tidak ada persentase di respons ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)).
>
> **Status (2026-10-23):**
> - **Backend: sudah diimplementasikan** (modul `plannervendor`, 17 endpoint; [BE.md](BE.md), [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md)). Tabel sudah ada sejak migrasi `006_000_planner_vendors.sql` dan `009_000_planner_shopping.sql` (identik dengan DDL di [BE bab 1](BE.md#1-skema-database); tidak ada migrasi baru). Titik sambung di Menu 01–03 (`vendor_id` pada tugas dan pos, view rollup, widget `vendors`, applier `shopping`) kini terisi.
> - **Frontend: belum ada satu baris pun.** [FE.md](FE.md) adalah acuan pembangunannya; kontrak datanya sudah dicocokkan dengan backend yang berjalan (tipe, fixture, contoh respons, katalog galat).
> - **Audit spesifikasi** dilakukan dengan menjalankan SQL-nya terhadap PostgreSQL 16.2 dan backend Menu 01–03 yang sebenarnya, membaca ulang PDF Hal. 9–10 dan 12 dengan `pdfminer`, dan menjalankan kode acuan Go dan TypeScript ([BE bab 7.1](BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi), [FE bab 11](FE.md#11-pengujian)). Angka PDF dan DDL ternyata benar; yang salah adalah perilaku di sekitarnya: pagu yang disalin menjadi basi, `lock-deal` yang bisa menautkan dua vendor ke satu pos secara diam-diam, pos berharga satuan yang rusak, dan jenis non-pernikahan yang selalu mendapat pos ganda ([CHANGELOG spesifikasi](../CHANGELOG.md)).
> - **Temuan saat implementasi:** pos gulungan bawaan tiga jenis (peresmian, wisuda, seminar) juga pos khusus kategori vendor, sehingga pos cadangan `perlengkapan` dipakai bila pos bawaan sudah dipegang vendor ([BE bab 9 butir 22](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
> - **Hasil uji:** 334 tes lulus terhadap PostgreSQL, 37 mutasi kode (36 digagalkan, satu setara), respons HTTP asli sama dengan fixture FE, dan tabel sembilan jenis di bawah dijaga tes terhadap registry. Yang belum ada: frontend dan tampilan `.xlsx` di Excel.

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

| Sheet PDF | Isi PDF | Di Adicara |
|---|---|---|
| **Kontak Vendor** (Hal. 12) | *List Rencana Vendor* dan *Fix Kerjasama Vendor* untuk 10 kategori: Venue, Decoration, Wedding Organizer, MUA, Catering, Photography, Attire, Hand Bouquet, MC, Entertainment (ditulis "Entertaiment" di PDF); tujuh pertama berstatus FIX (kolom Nama Vendor, No Tlp, PIC, Action); isi baris hanya contoh dummy | **Tab 1 Direktori & Kontak Vendor**: pipeline `rencana → negosiasi → fix` dengan checklist 10 kategori (7 dari 10 terkunci pada PDF); data dummy PDF tidak disalin |
| **List Seserahan** (Hal. 9–10) | 37 barang, kolom Kebutuhan, Link Pembelian, Harga, Kategori (6), Status (Selesai/On Proses); estimasi biaya Rp 4.578.500 | **Tab 2 Belanja Pengadaan & Logistik**: 37 barang dengan nama, kategori, harga, dan status persis PDF; tautan pribadi pembuat PDF tidak disalin |

**Angka acuan** (dibaca ulang dengan `pdfminer`, dibandingkan secara terprogram dengan [BE bab 5.3](BE.md#53-daftar-belanja-37-item-seserahan-pdf-hal-910): nama, kategori, harga, dan status 37 barang sama): Perangkat Alat Solat 4 barang **597.300** · Make Up & Skin Care 14 barang **1.268.900** · Peralatan Mandi 8 barang **990.700** · Pakaian Dalam 4 barang **517.400** · Pakaian Luar 5 barang **820.200** · Kebutuhan Lain 2 barang **384.000** · total **4.578.500**; PDF menandai 16 barang selesai (Rp 2.063.800) dan 21 on proses (Rp 2.514.700). Tabel Fix Kerjasama PDF memuat tujuh baris (Venue sampai Attire).

Yang tidak ada di PDF dan ditambahkan Adicara: pagu vendor dari skenario aktif, perbandingan pagu untuk kategori yang berbagi satu pos, kunci kerja sama ke anggaran tanpa pos ganda, tautan WhatsApp, jumlah × harga satuan, penanda souvenir, gulungan total belanja ke pos anggaran, ekspor `.xlsx` kontak (tanpa harga) dan belanja.

```text
[Menu 03 skenario aktif] ──pagu (dibaca saat tabel dibuka)────────────> [Tab 1 Vendor] ──lock-deal──> [Menu 03 pos anggaran]
                                                                          │                                  ▲
[Menu 03 pos anggaran] ──status bayar (view planner_budget_lines)─────────┘                                  │ rollup_source = 'shopping'
                                                                                                              │ (angka dari view)
                                                                          [Tab 2 Belanja] ──link-to-budget────┘
```

---

## 2. Tiga Mekanisme yang Dijaga

Tiga hal di menu ini menyentuh data milik Menu 03; masing-masing punya aturan yang terbukti dengan menjalankannya ([BE bab 2](BE.md#2-logika-bisnis)):

- **Pagu tidak disalin.** Pagu vendor = pos skenario aktif yang cocok lewat `category_key → budget_item_keys`, dibaca setiap kali tabel dibuka. Menyalinnya ke kolom vendor membuatnya basi begitu pengguna menerapkan skenario lain (Menu 03 tidak pernah menulis ke tabel vendor). Kolom `budget_limit` hanya menyimpan **pagu manual** (0 = ikut skenario). Pencocokan mencoba sesi vendor dulu, lalu Umum (`all`), karena seluruh pos jenis non-pernikahan berada di `all`. Kategori yang berbagi satu pos (pernikahan: Venue + Decoration, MUA + Attire, MC + Entertainment) dibandingkan di tingkat kelompok: Rp 12 jt + Rp 8 jt terhadap pagu Rp 20 jt.
- **`lock-deal` tidak menggandakan pos dan tidak menimpa diam-diam.** Satu transaksi di bawah kunci proyek: tautkan ke pos khusus kategori itu bila ada, buat pos baru bila tidak, biarkan pos bersama beberapa kategori (hanya ditandai), dan jangan sentuh estimasi pos berharga satuan. Tanpa kunci proyek, dua vendor sekategori yang dikunci bersamaan sama-sama dilaporkan berhasil padahal tautan yang pertama hilang (dijalankan).
- **Gulungan belanja** menjadikan angka pos anggaran *dihitung* dari daftar belanja oleh view, bukan disalin: estimasi = Σ total estimasi barang yang menunjuk pos itu, terbayar = Σ total riil barang berstatus selesai. Barang baru ikut bila dibuat dengan tujuan gulungan; mencabut gulungan melepas semua barang dan mengembalikan pos ke angka tersimpannya. Pos yang sudah punya pembayaran meminta konfirmasi (pembayaran tersimpan tetapi diabaikan selama gulungan aktif). Proyek Contoh **tidak** menghidupkan gulungan agar angka PDF Hal. 5–6 tetap utuh (dengan gulungan total proyek 74.150.000 menjadi 75.228.500).

---

## 3. Struktur Sub-Fitur

### Tab 1: Direktori & Kontak Vendor (`?tab=vendor`)
- **Checklist kategori** dari registry (+ kategori kustom yang dipakai): tiap kategori berstatus *Terkunci* (ada vendor `fix`), *Berjalan* (ada kandidat), atau *Kosong*. KPI "n dari m kategori terkunci" menghitung kategori, bukan baris vendor, dan tidak menghitung kategori yang ditandai "tidak diperlukan" (`skipped_vendor_categories` pada proyek, diubah lewat `PATCH` proyek Menu 01). Vendor `batal` tidak dihitung di mana pun.
- **Filter** sesi (master sesi), status pipeline (`Rencana`, `Negosiasi`, `FIX`, `Batal`), dan kategori; tabel berisi kategori, vendor, PIC + tombol WA, pagu, penawaran/deal, selisih, status bayar, aksi.
- **Pagu dan selisih** ([BE §2.1](BE.md#21-pagu-vendor-dari-skenario-aktif)): selisih hijau (hemat) atau merah (melebihi pagu) hanya tampil bila pagu dan penawaran ada; di kelompok pagu selisih positif adalah *sisa pagu* selama masih ada anggota tanpa penawaran, baru *hemat* bila semuanya sudah ada penawaran. Pada Proyek Contoh pagu Catering adalah Rp 30.000.000 (skenario 75 Jt), jadi deal Rp 25.000.000 menghemat Rp 5.000.000; Fotografi pas Rp 3.000.000.
- **Kunci Kerja Sama (FIX)** membuka dialog harga final dengan opsi *Sinkronkan ke anggaran*; hasilnya dijelaskan (ditautkan, pos baru, pos bersama tidak diubah, pos berharga satuan tidak diubah). Status `fix` mensyaratkan harga deal > 0.
- **Status bayar** dibaca dari pos tertaut (Menu 03 **Tab 3**): `Belum DP` → `DP` → `Sebagian` → `Lunas`. Vendor pada pos bersama tidak bertaut satu per satu, jadi baris kelompok menampilkan status pos bersama itu.
- **WhatsApp 1-klik** dengan nomor ternormalisasi (`628…`) dan templat pesan; **ekspor** kontak vendor `fix` (`.xlsx`, nama, kategori, PIC, telepon; tanpa harga karena lembar ini untuk WO/panitia).

### Tab 2: Belanja Pengadaan & Logistik (`?tab=belanja`)
- **Checklist barang**: sesi, kategori, nama barang, **jumlah dan satuan**, harga satuan estimasi dan riil, alamat pembelian (teks bebas; URL `http/https` dirender sebagai tautan), status `belum → proses → selesai` (menandai selesai mengisi harga riil dengan estimasi bila kosong).
- **Perbandingan estimasi vs riil** per barang dan di footer; **KPI** total estimasi, realisasi (hanya barang selesai), selesai/proses.
- **Gulungan ke anggaran**: banner di atas tabel menawarkan "Hitung pos dari daftar belanja" pada pos bawaan jenis acara (pernikahan: Seserahan). Banner menyebut bila pos sudah punya pembayaran dan meminta konfirmasi ([BE §2.4](BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup)).
- **Souvenir**: barang bertanda souvenir yang selesai membentuk stok yang dibandingkan dengan serah terima di meja resepsionis (Menu 05).
- **Terapkan template**: pernikahan menyediakan 37 barang seserahan (`status belum`); jenis lain tidak punya barang bawaan, hanya kategori.
- **Ekspor** rekap `.xlsx`.

---

## 4. Penerapan 9 Jenis Acara

Satu tabel registry ([kontrak bab 3](../00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data)); kategori vendor dan pemetaannya ke pos: [TEMPLATE_EVENT bab 7–15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md). Pernikahan bersumber dari PDF; delapan lainnya adalah usulan Adicara.

| Jenis | Kategori vendor | Kelompok pagu (pos bersama) | Kategori belanja | Template belanja | Pos gulungan bawaan |
|---|:-:|---|:-:|:-:|---|
| 💍 Pernikahan | 10 | `venue_dekor`, `rias_busana`, `hiburan` | 6 | 37 barang | `seserahan` |
| 🕌 Khitanan | 5 | – | 3 | – | `undangan_souvenir` |
| 🍼 Aqiqah | 4 | – | 3 | – | `undangan_souvenir` |
| 🏢 Peresmian | 7 | `venue_dekor` | 3 | – | `prosesi_vip` |
| 🚀 Launching | 7 | `produksi_av` | 3 | – | `merchandise_kit` |
| 🎂 Ulang Tahun | 6 | `katering_kue` | 3 | – | `souvenir` |
| 🎓 Wisuda | 6 | – | 3 | – | `toga_atribut` |
| 🎤 Seminar | 6 | `av_streaming` | 4 | – | `kit_sertifikat` |
| ✨ Acara Lainnya | 5 | – | 3 | – | `perlengkapan` (dibuat bila belum ada) |

Pengguna selalu dapat menambah kategori sendiri: vendor memakai `category_key` yang diturunkan dari nama (huruf kecil, `_`), belanja memakai teks kategori. Semua pos skenario dan template jenis non-pernikahan berada di sesi Umum (`all`), sehingga pagu dan `lock-deal` mencoba sesi vendor lebih dulu lalu Umum; tanpa itu vendor yang ditempatkan di sesi nyata tidak mendapat pagu dan `lock-deal` membuat pos kedua di samping pos template (dijalankan pada kedelapan jenis). Pos gulungan bawaan non-pernikahan adalah usulan yang menutup [03 BE bab 9 butir 29](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) ([BE bab 9 butir 6](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 5. Integrasi Lintas Modul (mekanisme nyata)

Mekanisme lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung).

| Dari → ke | Mekanisme |
|---|---|
| 04 Vendor → 03 Pos | `lock-deal`: tautkan (`planner_budget_items.vendor_id`, unik per vendor) atau buat pos tanpa duplikasi; SQL langsung ke tabel 03 dalam transaksi yang sama (pengecualian keempat, [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)); pos bersama beberapa kategori dan pos berharga satuan tidak ditimpa |
| 03 Skenario → 04 Vendor | pagu dibaca dari skenario aktif lewat `category_key → budget_item_keys` setiap kali daftar dibuka; tidak disalin |
| 04 Belanja → 03 Pos | `planner_shopping_items.budget_item_id` + `planner_budget_items.rollup_source='shopping'` (SQL langsung, pengecualian keempat); angka dihitung view; pembayaran pos itu terkunci di Menu 03 (`rollup_locked`) |
| 04 Belanja ↔ 05 Tamu | `purpose='souvenir'` (stok = Σ jumlah barang selesai, dibulatkan ke bawah) vs `planner_guests.souvenir_received`; Menu 05 membaca stok dengan SQL ([BE §2.5](BE.md#25-stok-souvenir)) |
| 04 Vendor ← 02 Tugas, 06 Agenda | `planner_tasks.vendor_id`, `planner_rundowns.vendor_id` (FK komposit; hapus vendor mengosongkannya); Menu 02, 03, dan 06 membaca nama dan nomor WA lewat `JOIN` |
| 04 → 01 Dashboard | `SummaryProvider` `vendors` (`VendorSummary`: kategori terkunci, kandidat berjalan) |
| 01 Wizard → 04 | `TemplateApplier` `shopping` (37 barang pernikahan) di dalam transaksi pembuatan proyek; `SampleOverlay` `vendors` dan `shopping` mengisi Proyek Contoh setelah overlay Menu 02 dan 03 |
| 04 → 01 Proyek | "Tandai tidak diperlukan" menulis `skipped_vendor_categories` lewat `PATCH` proyek Menu 01 (mengganti seluruh daftar) |

---

## 6. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — daftar isi, route dan komponen, alur data dan penanganan galat, prasyarat dari Menu 01 dan 03, tipe TypeScript yang cocok dengan DTO Go, fixture ([fixtures/](fixtures/)), kode acuan yang sudah dijalankan (`vendor.ts`, `vendorApi.ts`, `vendorMessages.ts`), wireframe dua tab dengan angka Proyek Contoh, alur interaksi, pesan galat, responsif/aksesibilitas/keamanan, pengujian, urutan kerja, dan celah.
- ⚙️ [BE.md](BE.md) — status dan hasil audit, migrasi 006 dan 009 (sudah ada), logika bisnis dengan SQL yang dijalankan (pagu, `lock-deal`, `NormalizeWA`, gulungan, stok souvenir, template, ekspor, sambungan ke Menu 01), 17 endpoint dengan bentuk respons dan validasi, DTO Go yang dikompilasi, master data (10 kategori, 10 vendor contoh fiktif, 37 barang), pengujian ✔/✘, urutan pengerjaan, dan celah yang menunggu keputusan.
