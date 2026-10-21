# Menu 03: Keuangan & Anggaran

> **Prinsip Perancangan:**
> 1. **Event-agnostic**: sesi, peran kontributor, dan template pos dibaca dari master sesi dan registry ([00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md)). Pernikahan memakai preset dan pos dari PDF; jenis lain memakai template persentase pagu ([TEMPLATE_EVENT bab 15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#15-pos-anggaran-template-non-pernikahan)).
> 2. **Kebebasan penuh**: template hanya titik awal. Pengguna menambah, mengubah, menghapus, dan mengurutkan pos kapan saja; tidak ada komponen terkunci.
> 3. **Template ≠ data contoh**: template membuat pos berestimasi 0 tanpa pembayaran. Angka pembayaran dan tabungan PDF hanya ada di Proyek Contoh ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)).
> 4. **Uang = rupiah utuh** (`BIGINT`/`int64`); semua angka efektif dibaca dari view `planner_budget_lines`.

---

## 1. Pemetaan ke Dokumen Acuan (Pernikahan)

Tiga sheet PDF digabung menjadi tiga tab berurutan sesuai alur nyata pengguna:

| Sheet PDF | Isi PDF | Tab |
|---|---|---|
| **Skenario Budget** (Hal. 4) | 6 paket (10/20/30/50/75/100 jt), masing-masing 10 pos, total = nama paket | **Tab 1 Simulasi & Alokasi** — rencana dan plafon |
| **Target Tabungan** (Hal. 7) | setoran bulanan Feb–Des 2026 untuk Calon Mempelai Pria dan Wanita; terkumpul Rp 47,5 jt dari target Rp 75 jt | **Tab 2 Kas & Tabungan** — sumber modal |
| **Anggaran Pernikahan** (Hal. 5–6) | pos per kegiatan: Lamaran (5), Akad (1), Resepsi (8); kolom Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa; subtotal kegiatan dan grafik | **Tab 3 Realisasi & Pembayaran** — kas keluar |

**Angka acuan** (diuji; [BE bab 6](BE.md#6-master-data)): tagihan awal **Rp 74.150.000**; DP/Lunas 20.050.000; Termin 1 12.000.000; Termin 2 1.500.000; **dibayar Rp 33.550.000**; **sisa Rp 40.600.000** (45,2% terbayar). Lamaran 7.150.000 (dibayar 4.750.000, sisa 2.400.000); Akad 600.000 (lunas); Resepsi 66.400.000 (dibayar 28.200.000, sisa 38.200.000). Tabungan: Pria 30.000.000 + Wanita 17.500.000 = 47.500.000.

Yang tidak ada di PDF dan ditambahkan Adicara: multi-skenario (maks. 3) + perbandingan, peran kontributor di luar dua pihak, rekomendasi bulanan, milestone per sesi, Cashflow Bar, drawer pembayaran dengan bukti, pos berharga satuan, gulungan belanja, ekspor.

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

Pengguna menyimpan paling banyak **3 skenario** (mencegah kelumpuhan keputusan; batas ditegakkan dengan kunci transaksi) dan berpindah kapan saja:

- **Smart Adjust (default)**: hanya `initial_estimate` pos yang cocok `(sesi, item_key)` diubah ke nilai skenario; pos baru ditambahkan; pos kustom dan pos yang tidak ada di skenario dibiarkan. DP/termin, tanggal, bukti, vendor, dan dokumen tertaut **tidak pernah disentuh**; `Sisa` dihitung ulang oleh view. Pratinjau perubahan ditampilkan sebelum konfirmasi.
- **Reset bersih**: hapus semua pos dan pembayaran lalu salin skenario; wajib mengetik `RESET`.
- Opsi: *perbarui pagu proyek* (`target_budget` = total skenario; default aktif) dan *target tabungan ikut pagu*.
- Pos yang digulung dari daftar belanja (`rollup`) dan pos berharga satuan dilewati dan dilaporkan.

Satu sumber pagu: `planner_projects.target_budget` (diperbarui saat skenario diterapkan); total skenario adalah rencana, jumlah estimasi pos adalah kenyataan, dan selisihnya ("sisa plafon") ditampilkan. Target tabungan = `savings_target` bila diisi, selain itu mengikuti `target_budget`.

### Integrasi Tab 2 → Tab 3
Cashflow Bar di Tab 3: `Kas bebas = terkumpul (Tab 2) − terbayar (Tab 3)`; bila `sisa kewajiban > kas bebas`, peringatan defisit dan rekomendasi setoran. Data PDF: kas bebas 13.950.000, defisit 26.650.000.

---

## 3. Struktur Tab

### Tab 1: Simulasi & Alokasi Budget (`?tab=simulasi`)
- **Pemilih skenario** (maks. 3; tombol tambah nonaktif bila penuh) dengan tanda ★ pada skenario aktif; aksi ganti nama, duplikasi, hapus.
- **Rangkaian acara yang disimulasikan**: kotak centang dari master sesi (pernikahan: Lamaran, Akad Nikah, Resepsi, + sesi kustom seperti Siraman/Ngunduh Mantu). Tanpa Lamaran cukup uncheck; mencentang sesi menawarkan pos template sesi itu (estimasi 0).
- **Preset siap pakai**: pernikahan memakai 6 paket PDF (10, 20, 30, 50, 75, 100 jt); jenis lain memakai template persentase. Input nominal bebas dan slider.
- **Donut** proporsi pos, **tabel pos per sesi** (tambah, ubah, hapus, kembalikan ke nilai preset), **bandingkan skenario** berdampingan, tombol **Jadikan Skenario Aktif & Terapkan**.

### Tab 2: Kas & Tabungan Acara (`?tab=tabungan`)
- **Pembukuan mandiri.** Adicara bukan lembaga keuangan, tidak menyediakan rekening/VA, dan tidak menampung dana; tab ini hanya catatan bersama. Kolom *catatan/lokasi simpanan* hanyalah pengingat pribadi.
- **KPI**: target, terkumpul, kekurangan, persentase capaian.
- **Rekomendasi bulanan** dan **milestone per sesi** dihitung dari tanggal sesi ([BE §3.6](BE.md#36-rekomendasi-tabungan-dan-milestone-bertahap)); sesi tanpa tanggal tidak membuat milestone.
- **Kontributor**: kartu ringkasan (nama, peran dari registry, total, porsi, jumlah setoran), tambah kontributor.
- **Dua tampilan setoran**: *Daftar* (tanggal, jumlah, penyetor, aksi; total di footer; filter per penyetor) dan *Matriks Bulanan* (bulan × kontributor, meniru PDF Hal. 7).
- **Grafik tren** saldo kumulatif menuju target dengan penanda tanggal sesi.

### Tab 3: Realisasi Anggaran & Pembayaran (`?tab=realisasi`)
- **Filter sesi** dari master sesi; **4 KPI** (total estimasi, terbayar, sisa, rasio pelunasan) dengan mikro-grafik; **Cashflow Bar**.
- **Matriks rekapitulasi sesi** meniru PDF Hal. 5–6 (baris: Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa; kolom: sesi + Total).
- **Tabel pos**: sesi, pos, vendor terkait, estimasi, DP, Termin 1, Termin 2, terbayar, sisa, status (`Belum`, `DP`, `Sebagian`, `Lunas`), aksi. Penanda sumber: *Dari vendor*, *Dari dokumen*, *Dihitung dari belanja* (pos `rollup`, terkunci untuk pembayaran).
- **Drawer pembayaran** (DP / Termin 1 / Termin 2 / Pelunasan), **harga satuan** (jumlah × harga untuk pos seperti katering), **terapkan template**, **ekspor `.xlsx`**.

---

## 4. Integrasi Lintas Modul

Mekanisme lengkap: [kontrak bab 8](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung). Ringkasnya: vendor `fix` menautkan atau membuat pos tanpa duplikasi; daftar belanja digulung ke pos lewat view; biaya dokumen disalin ke pos tanpa menimpa; proyeksi RSVP mengisi jumlah porsi pos katering atas aksi pengguna; skenario aktif memberi plafon `budget_limit` vendor. Penerimaan angpau/amplop digital **tidak** masuk Tab Tabungan.

## 5. Dokumen Spesifikasi Teknis
- 🎨 [FE.md](FE.md) — tab bar, wireframe tiga tab dengan angka PDF, pemilih skenario, modal pratinjau terapkan skenario, Cashflow Bar, matriks bulanan, drawer pembayaran, responsif.
- ⚙️ [BE.md](BE.md) — migrasi 008 dan 010 (view), algoritma Smart Adjust, kunci `item_key`, endpoint, DTO `int64`, master data (6 preset, 14 pos template, dataset PDF, tabungan 22 setoran).
