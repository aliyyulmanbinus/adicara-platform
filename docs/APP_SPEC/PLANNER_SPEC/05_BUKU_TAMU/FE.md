# Menu 05: Buku Tamu & Undangan Digital — Frontend Specification

Antarmuka **Daftar Tamu & Undangan** dan **Meja Resepsionis & Check-in Hari-H**. Pola umum frontend (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens). Wireframe Tab 1 memakai **Proyek Contoh** (20 tamu fiktif); wireframe Tab 2 adalah **ilustrasi hari-H** (Proyek Contoh belum berada di hari-H).

---

## 1. Route & File Path

| Item | Nilai |
|---|---|
| Halaman | `/dashboard/planner/[id]/tamu` → `src/pages/dashboard/planner/[id]/tamu/index.astro` |
| Tab | `?tab=tamu` (default) · `?tab=checkin` |
| Layout | `AppLayout.astro`; `nav_active = 'planner'`, sub-nav `tamu` |
| Komponen | `src/components/planner/guest/`: `GuestKpi`, `SideSwitcher`, `CategoryCounters`, `GuestTable`, `GuestModal`, `GuestImportModal`, `ReceptionConsole`, `QrScanModal`, `QuickCheckinSheet`, `WalkInModal` |
| Aksi klien | BFF `/dashboard/api/planner/projects/:id/...` |

**Dinamis, bukan tertulis:** pemisah sisi, label sisi, kategori relasi, dan dropdown sesi berasal dari registry dan master sesi. Jenis acara tanpa sisi (ulang tahun, wisuda, lainnya) tidak menampilkan pemisah sisi dan kolom "Pihak". Judul menu/tab dan kata benda "Tamu" (seminar: "Peserta") berasal dari `labels` registry ([TEMPLATE_EVENT bab 3.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#31-sebutan-antarmuka-labels)); teks "undangan" di dokumen ini adalah istilah teknis (kolom `invitation_*`) dan diganti "pendaftaran" pada tampilan seminar.

```html
<nav class="guest-tabs">
  <a href="?tab=tamu"    class="tab-item active"><i class="ph ph-users-three"></i> Daftar Tamu & Undangan</a>
  <a href="?tab=checkin" class="tab-item"><i class="ph ph-qr-code"></i> Meja Resepsionis & Check-in</a>
</nav>
```

---

## 2. Tab 1: Daftar Tamu & Undangan

### 2.1 Wireframe (Proyek Contoh; mode Standalone)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────┐ ┌─────────────────────┐ ┌────────────────┐ ┌──────────────────────────┐   │
│ │ 👥 Undangan    │ │ ⚖️ Per Pihak        │ │ 📤 Terkirim    │ │ ✉️ RSVP                  │   │
│ │ 20 · 43 pax    │ │ Pria 10 (22 pax)    │ │ 14 (70%)       │ │ 🟢 Hadir 12 (27 pax)     │   │
│ │                │ │ Wanita 10 (21 pax)  │ │ 6 belum        │ │ 🟡 Ragu 2 🔴 Tidak 2 ⚪ 4 │   │
│ └────────────────┘ └─────────────────────┘ └────────────────┘ └──────────────────────────┘   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Pihak: [Semua (20)] [🤵 Pihak Pria (10)] [👰 Pihak Wanita (10)]                               │
│ Kategori: [Semua] [Teman 6] [Kolega 3] [Keluarga 4] [Tetangga 2] [Teman Ortu 3] [VIP 2]       │
│ Sesi: [Semua ▼]  RSVP: [Semua ▼]  🔍 [ nama, no meja, no WA … ]                                │
│ [ + Tambah Tamu ]  [ 📥 Impor Excel/CSV ]  [ 📤 Ekspor ]  [ 💬 Kirim WA bergilir ]            │
│ ℹ️ Mode Standalone: undangan cetak/tautan eksternal. [Aktifkan Undangan Digital Adicara]       │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Nama                      Pihak  Kategori  Sesi    Meja   Pax Undangan        RSVP      Aksi│
│  1 Bpk. Prof. Bambang S.     Pria   VIP       Resepsi VIP-01  2  ✓ Terkirim     🟢 Hadir   ✏️🗑️│
│  2 Raditya Pratama & Pasangan Pria  Teman     Resepsi T-04    2  ✓ Terkirim     🟢 Hadir   ✏️🗑️│
│  3 Bpk. Hendro & Keluarga    Pria   Keluarga  Umum    FAM-01  4  ✓ Terkirim     🟢 Hadir   ✏️🗑️│
│  4 Satria Wibawa (Kolega)    Pria   Kolega    Resepsi T-10    2  ✓ Terkirim     🟡 Ragu    ✏️🗑️│
│  5 Bpk. Joko Susilo (RT 04)  Pria   Tetangga  Resepsi T-08    2  ✓ Terkirim     🔴 Tidak   ✏️🗑️│
│ … halaman 1 dari 1 (100 per halaman)                                                          │
│ TOTAL 20 tamu                                       43 pax    14 terkirim      12 hadir (27 pax)│
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Komponen

1. **`GuestKpi`** — empat kartu dari `GuestOverview` (dihitung atas seluruh data, bukan halaman). RSVP selalu empat status dan jumlahnya sama dengan total undangan.
2. **`SideSwitcher`** — dari `overview.sides`; disembunyikan bila registry jenis acara tidak punya sisi. **`CategoryCounters`** — pil berpenghitung dari `overview.categories` (pernikahan: 6 kategori PDF) dan berfungsi sebagai filter cepat.
3. **`GuestTable`** — kolom Pihak/Kategori/Sesi dari registry; tombol *Salin Link* dan *Kirim WA* hanya bila `personalized_url` ada (undangan terhubung); selain itu kolom Undangan menampilkan opsi "Undangan cetak / tautan eksternal" dan status kirim manual. Pada Standalone, RSVP diubah lewat dropdown per baris (`PATCH .../rsvp`).
4. **Pesan WhatsApp** — templat netral jenis acara yang disusun backend ([BE §2.4](BE.md#24-pesan-whatsapp-undangan)); contoh:

```text
Yth. Bpk. Prof. Bambang S.,

Tanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i pada Pernikahan Sinta & Arif:
📅 Minggu, 28 Maret 2027
📍 Resepsi
🏛️ Sasana Kriya Ballroom, Jakarta Selatan

Detail acara dan konfirmasi kehadiran (RSVP): https://adicara.id/i/sinta-arif?to=Bpk.+Prof.+Bambang+S.

Terima kasih.
```
   Baris tanpa data (jam, venue) dihilangkan. Status undangan berubah `Terkirim` hanya setelah pengguna menekan "Tandai terkirim" (membuka WhatsApp tidak otomatis mengubah status).
5. **`GuestModal`** — nama\*; pihak (radio dari registry; "Belum ditentukan" untuk jenis tanpa sisi/opsional); kategori (dropdown registry + kustom); sesi (master sesi); meja; pax (1–20); ponsel (`invalid_phone` → pesan); catatan (alergi, kursi roda).
6. **`GuestImportModal`** — unduh templat `.xlsx` (`Nama Tamu | Pihak | Kategori | Sesi | No Meja | Pax | No WhatsApp | Catatan`), unggah, lalu **pratinjau `dry_run`**: baris valid, baris galat (nomor baris + alasan), dan duplikat. Konfirmasi mengimpor semua atau, bila dicentang "lewati baris galat", hanya yang valid. Batas 5 MB/2.000 baris.
7. **Pagination** — 100 baris per halaman dengan `limit`/`offset`; pencarian dan filter dikirim ke server.
8. **Polling** — `GET .../guests/summary` tiap 30 detik saat tab aktif memperbarui kartu dan penghitung; tabel diperbarui bila pengguna menekan "Muat ulang" (agar baris tidak melompat).

### 2.3 Standalone vs Terintegrasi
- **Terintegrasi** (undangan terhubung): kolom Undangan menampilkan tautan aktif; status RSVP dari tamu muncul otomatis lewat polling.
- **Standalone**: banner tipis menawarkan Undangan Digital; tidak ada tautan personal. Perilaku ini sudah lengkap dan tidak bergantung pada modul undangan.

---

## 3. Tab 2: Meja Resepsionis & Check-in Hari-H

### 3.1 Wireframe (ilustrasi hari-H)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌──────────────────┐ ┌──────────────────┐ ┌───────────────────────┐ ┌────────────────────┐   │
│ │ 👥 Tamu Check-in │ │ 🚶 Pax di Lokasi │ │ 🎁 Souvenir           │ │ ⏳ Belum Hadir     │   │
│ │ 11 dari 18       │ │ 22 dari 39 pax   │ │ 11 diserahkan         │ │ 7 tamu             │   │
│ │ [██████░░░] 61%  │ │                  │ │ stok tersisa 29       │ │ (tanpa yang menolak)│  │
│ └──────────────────┘ └──────────────────┘ └───────────────────────┘ └────────────────────┘   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│  [ 📷 SCAN QR TIKET ]                      [ + TAMU WALK-IN ]                                 │
│  🔎 [ ketik nama, meja (VIP-01), atau no WA … ]      Segar otomatis tiap 5 detik · [Muat ulang]│
│  [ Semua (18) ] [ Sudah Hadir (11) ] [ Belum Hadir (7) ] [ VIP (2) ] [ Keluarga (4) ]         │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Jam    Nama                      Pihak  Kategori  Meja    Pax  Souvenir   Status   Aksi        │
│ 11.14  Bpk. Prof. Bambang S.     Pria   VIP       VIP-01   2   ✓          Hadir    [Batal]    │
│ 11.22  Raditya Pratama & Pasangan Pria  Teman     T-04     2   ✓          Hadir    [Batal]    │
│ --:--  Anisa Rahmawati           Wanita Teman     T-01     2   –          Belum    [ ✓ Check-in ]│
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Komponen
1. **Empat kartu** dari `reception-overview`; "belum hadir" tidak menghitung tamu `declined`. Kartu souvenir menampilkan stok hanya bila ada barang bertanda souvenir di Menu 04.
2. **`QrScanModal`** — `navigator.mediaDevices.getUserMedia` + `BarcodeDetector` bila tersedia (tanpa dependensi baru); bila peramban tidak mendukung, jatuh ke input manual token/nama dan pesan "Pemindai tidak didukung di perangkat ini". Bunyi *beep* saat berhasil; token proyek lain/salah → "Tiket tidak dikenali".
3. **`QuickCheckinSheet`** — nama (huruf besar), badge kategori/pihak, kotak kuning nomor meja ("MEJA VIP-01"), stepper pax riil (default `pax_count`), centang souvenir, catatan, tombol besar **KONFIRMASI HADIR & ARAHKAN**. Respons `409 already_checked_in` menampilkan "Sudah masuk pukul 11.14" dengan tombol *Batalkan check-in* dan *Tutup* (tidak ada check-in ganda).
4. **`WalkInModal`** — nama\*, pihak/kategori (opsional), pax datang, meja, souvenir; tamu langsung berstatus hadir.
5. **Ekspor rekap kehadiran** `.xlsx`/PDF.
6. **Sentuh**: tombol Scan dan Walk-in tinggi 56 px, target sentuh minimum 48 px, baris longgar, umpan balik audio.

---

## 4. Responsif
- < 768px: KPI grid 2×2; tabel menjadi *Guest Cards* (nama + badge, sesi/meja, status RSVP, tombol Salin Link / Kirim WA / Tandai terkirim sesuai mode); meja resepsionis dioptimalkan tablet landscape 1024 px.

## 5. Pengujian Frontend
`node:test`: normalisasi nomor WA (tabel kasus sama dengan backend), pembangkitan pil sisi/kategori dari registry termasuk kasus tanpa sisi, parse berkas impor di sisi klien hanya untuk pratinjau nama kolom (validasi sebenarnya di server), pemetaan kode error (`already_checked_in`, `invalid_phone`, `file_too_large`, `invalid_file`).
