# Menu 04: Vendor & Belanja Kebutuhan — Frontend Specification

Antarmuka **Direktori & Kontak Vendor** dan **Belanja Pengadaan & Logistik**. Pola umum frontend (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens). Angka wireframe adalah **Proyek Contoh** (PDF Hal. 9–10 dan 12; nama vendor fiktif).

---

## 1. Route & File Path

| Item | Nilai |
|---|---|
| Halaman | `/dashboard/planner/[id]/vendor` → `src/pages/dashboard/planner/[id]/vendor/index.astro` |
| Tab | `?tab=vendor` (default) · `?tab=belanja` |
| Layout | `AppLayout.astro`; `nav_active = 'planner'`, sub-nav `vendor` |
| Komponen | `src/components/planner/vendor/`: `VendorKpi`, `CategoryChecklist`, `VendorTable`, `VendorModal`, `LockDealModal`, `ShoppingTable`, `ShoppingItemModal`, `RollupBanner` |
| Aksi klien | BFF `/dashboard/api/planner/projects/:id/...` |

Chip sesi, daftar kategori vendor, dan kategori belanja berasal dari master sesi dan registry jenis acara proyek. Proyek `archived`: kontrol ubah nonaktif.

```html
<nav class="vendor-tabs">
  <a href="?tab=vendor"  class="tab-item active"><i class="ph ph-handshake"></i> Direktori & Kontak Vendor</a>
  <a href="?tab=belanja" class="tab-item"><i class="ph ph-shopping-bag"></i> Belanja Pengadaan & Logistik</a>
</nav>
```

---

## 2. Tab 1: Direktori & Kontak Vendor

### 2.1 Wireframe (Proyek Contoh)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────────┐ ┌─────────────────────┐ ┌────────────────────┐ ┌────────────────────┐│
│ │ 🤝 Kategori Terkunci│ │ 💰 Deal Vendor FIX │ │ ⏳ Kandidat Berjalan│ │ 🎯 Kelompok Pagu   ││
│ │ 7 dari 10  [███████░░░] 70%│ Rp 58.000.000  │ │ 3 vendor           │ │ 3 kelompok, 0 over ││
│ │ Belum: Hand Bouquet, MC,   │ (7 vendor FIX) │ │ Penawaran Rp 4,3 Jt│ │                    ││
│ │ Entertainment              │                │ │                    │ │                    ││
│ └────────────────────┘ └─────────────────────┘ └────────────────────┘ └────────────────────┘│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Kategori: ✅Venue ✅Decoration ✅WO ✅MUA ✅Catering ✅Photography ✅Attire ⏳Hand Bouquet ⏳MC ⏳Entertainment│
│ Sesi: [Semua] [Umum] [Lamaran] [Akad] [Resepsi]   Pipeline: [Semua (10)] [FIX (7)] [Riset (3)] │
│ [ + Tambah Vendor ]                                     [ 📥 Unduh Kontak Panitia & WO ]      │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Sesi    Kategori  Vendor              PIC / WA          Pagu       Deal        Status bayar   │
│ Resepsi Venue     Sasana Kriya        0812 0000 0001    ┐ kelompok 12.000.000  🟡 DP 0,9 jt   │
│ Resepsi Decoration Pelaminan Mega     0812 0000 0002    │ venue_dekor 8.000.000 –            │
│                                                         └ 20.000.000 = 20.000.000 (pas)       │
│ Resepsi WO        Harmony WO          0812 0000 0003    –          5.000.000   –              │
│ Resepsi MUA       Sanggar Rias Ayu    0812 0000 0004    ┐ rias_busana 3.500.000 –             │
│ Resepsi Attire    Butik Kebaya Anggun 0812 0000 0007    ┘ 5.000.000 = 5.000.000 (pas)         │
│ Resepsi Catering  Katering Bu Minah   0812 0000 0005    25.000.000 25.000.000  🟡 15 jt/sisa 10 jt│
│ Resepsi Photography Studio Cahaya     0812 0000 0006    3.000.000  3.000.000   🟡 DP 0,3 jt   │
│ Resepsi Hand Bouquet Fleur            0812 0000 0008    –          900.000     [Riset]        │
│ Resepsi MC        Kang Dedi           0812 0000 0009    ┐ hiburan   2.000.000    [Negosiasi] 🔒FIX│
│ Resepsi Entertainment Akustik Nada    0812 0000 0010    ┘ 3.400.000 1.400.000   [Negosiasi] 🔒FIX│
│ Tiap baris: [💬 WA] [✏️] [🗑️]                                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Komponen
1. **`VendorKpi`**: "Kategori terkunci" = `CategoriesFixed / CategoriesTotal` (kategori, bukan baris); total deal vendor `fix`; kandidat berjalan; jumlah kelompok pagu dan yang melebihi pagu.
2. **`CategoryChecklist`**: chip per kategori dengan status ✅ terkunci / ⏳ berjalan / kosong; klik memfilter tabel; kategori kosong menawarkan "Tambah vendor" dan **"Tandai tidak diperlukan"** (mis. acara tanpa Entertainment). Kategori yang dilewati keluar dari hitungan "x dari y terkunci" (progres bisa mencapai 100%) dan tampil redup dengan "Diperlukan lagi"; disimpan lewat `PATCH /v1/planner/projects/:id` field `skipped_vendor_categories` ([01 BE](../01_DASHBOARD/BE.md#2-domain--dto-go)).
3. **`VendorTable`**: kolom dan perilaku:
   - **Pagu / selisih**: untuk kategori yang berbagi pos, baris-baris vendor digabung visual (braket) dan perbandingan ditampilkan di tingkat kelompok dari `limit_groups`; selisih hijau = hemat, merah = melebihi pagu; **kosong bila pagu atau penawaran belum ada** (bukan "hemat penuh").
   - **PIC & WA**: tombol `💬 WA` membuka `https://wa.me/628…?text=` dengan templat *"Halo {PIC}, saya ingin menanyakan perihal kerja sama untuk acara {judul} pada sesi {sesi}…"*.
   - **Status bayar**: dari pos tertaut (`Belum DP` / `DP Rp X` / `Rp X dari Y` / `Lunas`); strip "–" bila belum ada pos.
   - Vendor `negosiasi`/`rencana` menampilkan tombol **🔒 FIX**.
4. **`LockDealModal`**: harga deal final dan opsi *Sinkronkan ke pos anggaran* (default aktif). Hasilnya ditampilkan: "Ditautkan ke pos Catering Resepsi" atau "Pos baru dibuat". Untuk kategori yang berbagi pos (Venue/Decoration, MUA/Attire, MC/Entertainment) estimasi pos tidak diubah; ditampilkan catatan kuning (`shared_budget_item`): "Pos 'Sewa Gedung & Dekorasi' (Rp 20 jt) dipakai bersama; total deal kelompok Rp 12 jt" dengan tombol **Samakan estimasi pos dengan total deal** dan tautan ke Tab Realisasi.
5. **`VendorModal`**: sesi (master sesi), kategori (registry + kustom), nama, portofolio (URL/IG), PIC, nomor ponsel (`08…`/`+62…` divalidasi; `invalid_phone` → pesan), pagu (terisi otomatis dari skenario, dapat ditimpa), penawaran/deal, status (`Riset`, `Negosiasi`, `Fix`, `Batal`), catatan.
6. **Ekspor kontak**: hanya vendor `fix` (nama, PIC, telepon) untuk lembar WO/panitia.

---

## 3. Tab 2: Belanja Pengadaan & Logistik

### 3.1 Wireframe (37 barang PDF; Proyek Contoh)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌──────────────────┐ ┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐       │
│ │ 🎯 Total Estimasi│ │ 💰 Realisasi      │ │ 🟢 Selesai        │ │ 🟡 On Proses      │       │
│ │ Rp 4.578.500     │ │ Rp 2.063.800      │ │ 16 dari 37        │ │ 21 dari 37        │       │
│ │ 37 barang        │ │ (barang selesai)  │ │ [████░░░░░] 43,2% │ │ [█████░░░░] 56,8% │       │
│ └──────────────────┘ └───────────────────┘ └───────────────────┘ └───────────────────┘       │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 🧺 Pos anggaran "Seserahan" [belum dihitung dari belanja]      [ Hitung pos dari daftar ini ]│
│ Sesi: [Semua] …   Status: [Semua (37)] [Selesai (16)] [On Proses (21)] [Belum (0)]           │
│ Kategori: [Semua] [Alat Solat 4] [Make Up & Skin Care 14] [Mandi 8] [Dalam 4] [Luar 5] [Lain 2]│
│ [ + Tambah Barang ]  [ Terapkan Template ▾ ]  [ 📥 Unduh (.xlsx/.pdf) ]                      │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Kategori           Barang              Jml  Satuan  Harga satuan  Riil     Alamat   Status │
│  1 Perangkat Alat Solat Mukena             1   –        194.000     194.000  🔗 …     Selesai │
│  2 Perangkat Alat Solat Al-Quran           1   –         70.500           0  🔗 …     On Proses│
│  5 Make Up & Skin Care  Cushion            1   –        175.700     175.700  🔗 …     Selesai │
│ 37 Kebutuhan Lain       Box Seserahan (10 Kotak) 1 –     37.900           0  🔗 …     On Proses│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ TOTAL 37 barang                                          4.578.500   2.063.800                │
│  • 16 selesai                                            2.063.800   2.063.800                │
│  • 21 on proses                                          2.514.700           0                │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Tautan pada kolom *Alamat* pada Proyek Contoh adalah placeholder; PDF asli berisi tautan pribadi pembuatnya dan tidak disalin.

### 3.2 Komponen
1. **KPI**: total estimasi (Σ `estimated_total`), realisasi (Σ `actual_total` barang `selesai`), selesai, proses.
2. **Kolom Alamat Pembelian** — teks bebas satu kolom. Bila diawali `http://`/`https://` ditampilkan sebagai tautan (`target="_blank" rel="noopener noreferrer"`, tampil hostname, mis. `🔗 shopee.co.id/…`); selain itu ditampilkan sebagai teks (nama toko, alamat, telepon). Skema lain (`javascript:` dll.) tidak pernah dijadikan tautan.
3. **Kolom Jumlah/Satuan** dan harga **per satuan**; total baris = jumlah × harga (dihitung server). Mengubah status ke `Selesai` dengan harga riil kosong mengisi riil = estimasi (dapat diedit).
4. **Toggle status inline** `⚪ Belum → 🟡 On Proses → 🟢 Selesai`.
5. **`RollupBanner`**: menampilkan pos anggaran target. Tombol "Hitung pos dari daftar ini" memanggil `link-to-budget`; bila pos sudah punya pembayaran, dialog konfirmasi menjelaskan bahwa pembayaran tersimpan namun diabaikan selama rollup aktif. Setelah aktif, banner berubah menjadi "✓ Pos 'Seserahan' dihitung dari daftar ini: Rp 4.578.500 / terbayar Rp 2.063.800" dengan tombol "Lepas".
6. **Souvenir**: kotak centang "Barang ini souvenir tamu" (`purpose`); banner kecil menunjukkan stok vs diserahkan (dari Menu 05).
7. **`ShoppingItemModal`**: sesi, kategori (registry + kustom), nama barang, jumlah + satuan, harga estimasi/riil per satuan, vendor rekanan (opsional), alamat pembelian, status, catatan, tanda souvenir.
8. **Terapkan Template**: pernikahan "37 barang seserahan (PDF)" dengan `Tambah yang kurang` / `Ganti template`; jenis lain menampilkan "Belum ada template barang untuk jenis acara ini" (kategori tetap tersedia di dropdown).

---

## 4. Responsif
- < 768px: KPI grid 2×2; vendor menjadi *Vendor Contact Cards* (badge sesi/kategori, pagu vs deal, tombol WA besar); belanja menjadi *Shopping Checklist Cards* (checkbox selesai, harga, tombol buka tautan).

## 5. Pengujian Frontend
`node:test`: validasi/normalisasi nomor WA yang sama dengan backend (tabel kasus bersama), penentuan tautan aman vs teks, pengelompokan `limit_groups`, perhitungan total baris (jumlah × harga tanpa float), pemetaan kode error (`invalid_phone`, `rollup_conflict`, `rollup_locked`, `vendor_already_linked`).
