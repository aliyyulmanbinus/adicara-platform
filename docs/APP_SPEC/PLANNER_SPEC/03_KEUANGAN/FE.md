# Menu 03: Keuangan & Anggaran — Frontend Specification

Antarmuka tiga tab: **Simulasi & Alokasi Budget** → **Kas & Tabungan Acara** → **Realisasi Anggaran & Pembayaran**. Pola umum frontend (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens). Angka pada wireframe adalah **Proyek Contoh** (dataset PDF Hal. 4–7).

---

## 1. Route & File Path

| Item | Nilai |
|---|---|
| Halaman | `/dashboard/planner/[id]/keuangan` → `src/pages/dashboard/planner/[id]/keuangan/index.astro` |
| Tab (query) | `?tab=simulasi` (default) · `?tab=tabungan` · `?tab=realisasi` (alias `?tab=anggaran`) |
| Layout | `AppLayout.astro`; `nav_active = 'planner'`, sub-nav `keuangan` |
| Komponen | `src/components/planner/finance/`: `ScenarioSwitcher`, `SubEventSelector`, `BudgetPresetBar`, `BudgetDonutChart`, `ScenarioItemsTable`, `ScenarioComparisonModal`, `ApplyScenarioModal`, `ContributorCards`, `SavingsLedger`, `SavingsMatrix`, `SavingsTrendChart`, `CashflowBar`, `BudgetKpiCards`, `SessionMatrix`, `BudgetTable`, `PaymentDrawer`, `BudgetItemModal` |
| Uang | integer rupiah; `lib/planner/format.ts` memformat ke `Rp 74.150.000` dan mengurai input tanpa float |
| Aksi klien | BFF `/dashboard/api/planner/projects/:id/...` |

Chip sesi dan label peran kontributor berasal dari master sesi dan registry, bukan teks tetap. Proyek `archived`: semua kontrol ubah nonaktif.

```html
<nav class="financial-tabs">
  <a href="?tab=simulasi"  class="tab-item active"><i class="ph ph-calculator"></i> 1 Simulasi & Alokasi Budget</a>
  <a href="?tab=tabungan"  class="tab-item"><i class="ph ph-piggy-bank"></i> 2 Kas & Tabungan Acara</a>
  <a href="?tab=realisasi" class="tab-item"><i class="ph ph-receipt"></i> 3 Realisasi & Pembayaran</a>
</nav>
```

---

## 2. Tab 1: Simulasi & Alokasi Budget (PDF Hal. 4)

### 2.1 Wireframe (preset 75 Jt aktif; Lamaran tidak disimulasikan)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ Keuangan & Anggaran                                      [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ]│
│ [★ 1 Simulasi]   [2 Kas & Tabungan]   [3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Skenario (maks. 3):  [ Opsi Hemat 50 Jt ]  [ ★ Opsi Ideal 75 Jt — AKTIF ]  [ Opsi 100 Jt ] [+]│
│ Rangkaian yang disimulasikan:  [ ] 💍 Lamaran   [✓] 📜 Akad Nikah   [✓] 🏛️ Resepsi   [+ Sesi] │
│ Paket cepat: [10 Jt] [20 Jt] [30 Jt] [50 Jt] [★ 75 Jt] [100 Jt]   Target: [ Rp 75.000.000 ] ━━●━│
├────────────────────────────────────┬─────────────────────────────────────────────────────────┤
│ 🍩 PROPORSI                        │ RINCIAN POS                                             │
│      Catering 40,0%                │ 📜 AKAD NIKAH                              Rp 600.000   │
│      Venue & Dekor 26,7%           │   Biaya KUA                       600.000   0,8%  ✏️ 🗑️ │
│      Mahar 6,7% · Rias 6,7% …      │ 🏛️ RESEPSI                              Rp 73.400.000   │
│                                    │   Mahar & cincin pernikahan     5.000.000   6,7%  ✏️ 🗑️ │
│ Terdistribusi Rp 75.000.000        │   Catering                     30.000.000  40,0%  ✏️ 🗑️ │
│ dari Rp 75.000.000 (sisa Rp 0)     │   Rias & baju pengantin         5.000.000   6,7%  ✏️ 🗑️ │
│                                    │   Sewa tempat & dekorasi       20.000.000  26,7%  ✏️ 🗑️ │
│                                    │   Undangan & souvenir           3.500.000   4,7%  ✏️ 🗑️ │
│                                    │   Dokumentasi                   3.000.000   4,0%  ✏️ 🗑️ │
│                                    │   Seserahan                     3.500.000   4,7%  ✏️ 🗑️ │
│                                    │   Hiburan & pengisi acara       3.400.000   4,5%  ✏️ 🗑️ │
│                                    │ 🛡️ UMUM                                    Rp 1.000.000 │
│                                    │   Dana darurat                  1.000.000   1,3%  ✏️ 🗑️ │
│                                    │ [+ Tambah pos di sesi ini]                              │
├────────────────────────────────────┴─────────────────────────────────────────────────────────┤
│ [⚖️ Bandingkan Skenario]                        [🚀 Jadikan Skenario Aktif & Terapkan ke Tab 3] │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Persentase dihitung terhadap `total_target` skenario (bukan terhadap subtotal sesi).

### 2.2 Komponen

1. **`ScenarioSwitcher`**: maksimal 3 skenario; `[+]` nonaktif dengan tooltip "Maksimal 3 skenario. Hapus salah satu untuk membuat yang baru" saat penuh (juga bila server menjawab `max_scenarios_reached`). Menu per skenario: ganti nama, duplikasi (nonaktif bila penuh), hapus.
2. **`SubEventSelector`**: kotak centang dari master sesi (tanpa sesi `all`) + `[+ Sesi]` (membuat sesi lewat `POST .../sessions`). Mencentang sesi yang belum punya pos menawarkan "Tambahkan pos template sesi ini (estimasi 0)" (pernikahan Lamaran: 5 pos). Uncheck menyembunyikan pos sesi itu dari skenario; plafon yang bebas ditawarkan "Alokasikan ke Resepsi".
3. **`BudgetPresetBar`**: tombol paket dari registry (pernikahan: 6 paket PDF; jenis lain: persentase pagu), input rupiah bebas, slider. Memilih paket mengisi pos skenario aktif dari preset, dengan konfirmasi bila pos sudah diedit.
4. **`BudgetDonutChart`**: SVG murni, warna dari token, tooltip nominal dan persen.
5. **`ScenarioItemsTable`**: dikelompokkan per sesi (accordion) dengan subtotal; ubah nama (inline), nominal, hapus, "kembalikan ke nilai preset".
6. **`ScenarioComparisonModal`**: dua skenario berdampingan, selisih per `(sesi, item_key)` hijau/merah.
7. **`ApplyScenarioModal`** — memanggil `POST .../scenarios/apply-preview` dan menampilkan hasilnya **sebelum** konfirmasi:

```text
Terapkan "Opsi Ideal 75 Jt" ke Realisasi?
  ✎ 9 pos diperbarui (estimasi saja)     Catering Resepsi 0 → 30.000.000, …
  ＋ 1 pos baru                          Dana darurat 1.000.000
  — 5 pos tidak tersentuh                 pos Lamaran, pos kustom
  ⚠ 0 pos dilewati                        (dihitung dari belanja / harga satuan)
Pembayaran, tanggal, bukti, dan vendor tidak diubah.
[x] Samakan pagu acara menjadi Rp 75.000.000     [x] Target tabungan ikut pagu
(•) Smart Adjust (disarankan)   ( ) Reset bersih — ketik RESET untuk menghapus semua pos & pembayaran
                                                               [ Batal ]  [ Terapkan ]
```

---

## 3. Tab 2: Kas & Tabungan Acara (PDF Hal. 7)

> **Batasan:** Adicara bukan bank/escrow/dompet digital, tidak punya rekening penampung/VA, dan tidak menampung dana. Tab ini hanya buku catatan bersama; spanduk batasan ini tampil di atas tab. Angpau/amplop digital dari tamu **tidak** dicatat di sini.

### 3.1 Wireframe (Proyek Contoh)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [1 Simulasi]   [★ 2 Kas & Tabungan]   [3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌──────────────────┐                │
│ │ 🎯 Target      │ │ 💰 Terkumpul   │ │ ⏳ Kekurangan  │ │ 📈 Capaian       │                │
│ │ Rp 75.000.000  │ │ Rp 47.500.000  │ │ Rp 27.500.000  │ │ 63,3%  [━━━●───] │                │
│ │ ikut pagu ✏️   │ │ Arif 63% · Sinta 37% │ Rp 4,58 Jt/bln │ │                  │                │
│ └────────────────┘ └────────────────┘ └────────────────┘ └──────────────────┘                │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 💡 Disarankan menabung Rp 4.584.000/bulan (Rp 2.292.000/orang) selama 6 bulan menuju Hari-H.   │
│    ✅ Lamaran (15 Okt 2026) Rp 7.150.000 aman · ✅ Akad (28 Mar 2027) Rp 600.000 aman          │
│    🟡 Resepsi (28 Mar 2027) Rp 66.400.000: terkumpul Rp 39.750.000 (59,9%)                    │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 👥 KONTRIBUTOR                                                    [ + Tambah Kontributor ]    │
│ ┌──────────────────────────────────────┐ ┌─────────────────────────────────────────────────┐ │
│ │ Arif (Mempelai Pria)                 │ │ Sinta (Mempelai Wanita)                         │ │
│ │ Rp 30.000.000 · 63,2% · 11 setoran   │ │ Rp 17.500.000 · 36,8% · 11 setoran              │ │
│ └──────────────────────────────────────┘ └─────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Tampilan: (•) Matriks Bulanan  ( ) Daftar Setoran                [ + Catat Uang Masuk 💵 ]    │
│ ┌───────────┬───────────────┬───────────────┬───────────────┐                                │
│ │ Bulan     │ Arif (Pria)   │ Sinta (Wanita)│ Jumlah        │   Grafik tren saldo kumulatif   │
│ │ Februari  │     2.500.000 │       500.000 │     3.000.000 │   menuju garis target 75 jt     │
│ │ Maret     │     5.000.000 │     2.000.000 │     7.000.000 │   dengan penanda sesi:          │
│ │ April     │     1.500.000 │     1.000.000 │     2.500.000 │   ▲ Lamaran  ▲ Akad/Resepsi     │
│ │ …         │               │               │               │                                 │
│ │ Desember  │     6.000.000 │     4.000.000 │    10.000.000 │                                 │
│ ├───────────┼───────────────┼───────────────┼───────────────┤                                 │
│ │ TOTAL     │    30.000.000 │    17.500.000 │    47.500.000 │                                 │
│ └───────────┴───────────────┴───────────────┴───────────────┘                                 │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Komponen
1. **KPI**: target (default ikut pagu; ikon pensil untuk menimpa → `PATCH /savings/target`, kosongkan untuk kembali ikut pagu), terkumpul, kekurangan, capaian.
2. **Rekomendasi & milestone**: dari `SavingsOverview.recommendation` dan `milestones` (hanya sesi bertanggal; sesi tanpa tanggal memunculkan ajakan "Isi tanggal sesi untuk target bertahap").
3. **`ContributorCards`**: nama, peran dari registry, total, porsi, jumlah setoran; tambah/ubah/hapus (hapus ditolak bila punya setoran: `contributor_in_use`).
4. **Dua tampilan setoran**: *Matriks Bulanan* (`SavingsMatrix`, meniru PDF; sel bisa diedit untuk mengubah setoran bulan itu) dan *Daftar* (`SavingsLedger`: filter chip per penyetor, kolom Tanggal, Uang Masuk, Penyetor, Aksi, footer TOTAL). Tanggal setoran bebas (tidak harus tanggal 1).
5. **`SavingsEntryModal`**: tanggal (default hari ini), nominal (integer rupiah), penyetor (dropdown kontributor + `[+ Daftarkan baru]`), catatan pribadi/lokasi simpanan.
6. **Proyek nyata mulai kosong** (tanpa kontributor dan setoran). Tidak ada tombol "reset ke contoh"; yang tersedia hanya hapus per setoran.
7. **`SavingsTrendChart`**: garis saldo kumulatif, garis target, penanda vertikal pada `session_date` tiap sesi.

---

## 4. Tab 3: Realisasi Anggaran & Pembayaran (PDF Hal. 5–6)

### 4.1 Wireframe (dataset PDF)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [1 Simulasi]   [2 Kas & Tabungan]   [★ 3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 🌊 Kas terkumpul Rp 47.500.000 │ Terbayar Rp 33.550.000 │ Kas bebas Rp 13.950.000             │
│ ⚠ Sisa kewajiban Rp 40.600.000 melebihi kas bebas: butuh tambahan kas Rp 26.650.000           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐ ┌──────────────────┐ ┌───────────────────┐ ┌───────────────────┐         │
│ │ Total Estimasi  │ │ Total Terbayar   │ │ Sisa Tagihan      │ │ Rasio Pelunasan   │         │
│ │ Rp 74.150.000   │ │ Rp 33.550.000    │ │ Rp 40.600.000     │ │ 45,2%  (ring)     │         │
│ │ Lam 9,6% Akad   │ │ DP 20,05 Jt      │ │ 🟢 7 lunas        │ │ Skenario aktif:   │         │
│ │ 0,8% Res 89,5%  │ │ T1 12 Jt T2 1,5  │ │ 🟡 7 berjalan     │ │ 75 Jt (sisa       │         │
│ │                 │ │                  │ │ 🔴 0 belum        │ │ plafon Rp 850 rb) │         │
│ └─────────────────┘ └──────────────────┘ └───────────────────┘ └───────────────────┘         │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Sesi: [Semua (14)] [💍 Lamaran (5)] [📜 Akad (1)] [🏛️ Resepsi (8)]                             │
│ [+ Catat Pembayaran] [+ Tambah Pos] [📥 Ekspor .xlsx] [Terapkan Template ▾]                   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ MATRIKS SESI           │ Lamaran      │ Akad       │ Resepsi       │ Total                    │
│ Tagihan Awal           │    7.150.000 │    600.000 │    66.400.000 │    74.150.000            │
│ DP / Lunas             │    3.250.000 │    600.000 │    16.200.000 │    20.050.000            │
│ Termin 1               │    1.000.000 │          0 │    11.000.000 │    12.000.000            │
│ Termin 2               │      500.000 │          0 │     1.000.000 │     1.500.000            │
│ Total Dibayar          │    4.750.000 │    600.000 │    28.200.000 │    33.550.000            │
│ Tagihan Sisa           │    2.400.000 │          0 │    38.200.000 │    40.600.000            │
│ Status                 │ 🟡 66,4%     │ ✅ LUNAS   │ 🟡 42,5%      │ 🟣 45,2%                 │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Sesi    Pos                    Estimasi      DP      Termin 1  Termin 2   Dibayar     Sisa  Status│
│  1 Lamaran Catering              2.000.000   500.000 1.000.000   500.000 2.000.000        0  LUNAS │
│  2 Lamaran Gaun/Kebaya             750.000   750.000         0         0   750.000        0  LUNAS │
│  3 Lamaran Fotografer            1.000.000   500.000         0         0   500.000  500.000  DP    │
│  4 Lamaran Cincin                2.500.000   600.000         0         0   600.000 1.900.000 DP    │
│  5 Lamaran Dekorasi                900.000   900.000         0         0   900.000        0  LUNAS │
│  6 Akad    Biaya KUA               600.000   600.000         0         0   600.000        0  LUNAS │
│  7 Resepsi Mahar                 3.000.000 3.000.000         0         0 3.000.000        0  LUNAS │
│  8 Resepsi Catering             25.000.000 5.000.000 10.000.000        0 15.000.000 10.000.000 SEBAGIAN│
│  9 Resepsi Rias & Baju Pengantin 5.000.000   600.000         0         0   600.000 4.400.000  DP    │
│ 10 Resepsi Sewa Gedung & Dekor. 20.000.000   900.000         0         0   900.000 19.100.000 DP    │
│ 11 Resepsi Undangan & souvenir   3.500.000 1.500.000         0         0 1.500.000 2.000.000  DP    │
│ 12 Resepsi Dokumentasi           3.000.000   300.000         0         0   300.000 2.700.000  DP    │
│ 13 Resepsi Seserahan 🧺 belanja  3.500.000 3.500.000         0         0 3.500.000        0  LUNAS │
│ 14 Resepsi Hiburan & Pengisi     3.400.000 1.400.000 1.000.000 1.000.000 3.400.000        0  LUNAS │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 🧺 = "Dihitung dari Belanja" bila pos digulung dari daftar belanja (Menu 04); kolom bayarnya terkunci.
```

### 4.2 Komponen

1. **`CashflowBar`**: `kas bebas = terkumpul − terbayar`; peringatan defisit memakai `cash_deficit` dari server.
2. **`BudgetKpiCards`**: mikro-grafik (proporsi sesi, rincian tahap bayar, pill status — hanya pos berestimasi > 0, ring rasio).
3. **`SessionMatrix`**: kolom dibangkitkan dari sesi yang punya pos + Total; status per sesi = `paid/estimate`.
4. **`BudgetTable`**: tabel 14+ kolom; pada layar sempit menjadi kartu. Penanda sumber (*Dari vendor*, *Dari dokumen*, *Dihitung dari belanja*). Pos `rollup` menampilkan angka dari server dan tombol bayar nonaktif (`rollup_locked`). Baris `payment_status`: `unpaid` Belum, `dp_paid` DP, `partial` Sebagian, `settled` Lunas, `remaining < 0` Lebih bayar.
5. **`PaymentDrawer`**: pilih pos (searchable), tahap (`DP` / `Termin 1` / `Termin 2` / `Pelunasan Penuh`), nominal dengan tombol "Bayar sisa (Rp X)", tanggal, tautan bukti (http/https), pratinjau sisa sebelum/sesudah. Memilih tahap yang sudah terisi menampilkan "Mengganti Rp X" (koreksi).
6. **`BudgetItemModal`**: sesi (dropdown master sesi), kategori (dropdown registry + kustom), nama pos (teks bebas), mode harga (*Total* / *Jumlah × harga satuan* dengan tombol "Gunakan proyeksi RSVP" untuk pos katering), vendor terkait (opsional), catatan.
7. **Terapkan Template**: `Tambah pos yang belum ada` / `Ganti pos template yang belum dibayar` (menampilkan pos berpembayaran yang dilewati).
8. **Ekspor**: `.xlsx` meniru PDF Hal. 5–6 (matriks sesi + tabel pos).

---

## 5. Responsif
- < 768px: KPI grid 2×2; matriks sesi *scroll* horizontal dengan kolom kiri sticky; tabel pos menjadi *Financial Expense Cards* berprogress bar; drawer pembayaran penuh layar; matriks bulanan Tab 2 menjadi daftar per bulan.

## 6. Pengujian Frontend
`node:test`: format/parse rupiah tanpa float, pembagian persen (satu desimal, jumlah tampil 100), penentuan label status pos, penyusunan kolom matriks sesi dari data, pemetaan kode error (`max_scenarios_reached`, `rollup_locked`, `invalid_stage`, `contributor_in_use`) ke pesan.
