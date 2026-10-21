# Menu 06: Rundown & Acara — Frontend Specification

Antarmuka **Susunan Rundown** menit demi menit dan **Audio, Playlist & Arahan Teknis**. Pola umum frontend (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens). Teks agenda pada wireframe adalah isi PDF Hal. 14; angka progres live adalah **ilustrasi hari-H**.

---

## 1. Route & File Path

| Item | Nilai |
|---|---|
| Halaman | `/dashboard/planner/[id]/rundown` → `src/pages/dashboard/planner/[id]/rundown/index.astro` |
| Tab | `?tab=rundown` (default) · `?tab=playlist` |
| Layout | `AppLayout.astro`; `nav_active = 'planner'`, sub-nav `rundown` |
| Komponen | `src/components/planner/rundown/`: `RundownKpi`, `SessionTabs`, `RundownTable`, `RundownTimeline`, `LiveStatusBadge`, `RundownModal`, `WaPreviewModal`, `AudioKpi`, `PlaylistTable`, `PlaylistModal` |
| Aksi klien | BFF `/dashboard/api/planner/projects/:id/...` |

Sesi (tab, dropdown, header) berasal dari `GET .../sessions` (master sesi, tanpa sesi sistem `all`). Menambah, mengubah, atau menghapus sesi dilakukan di **Edit Acara → Kelola Sesi** (Menu 01); tab rundown menyediakan tautan pintas ke sana, bukan editor sesi sendiri.

```html
<nav class="rundown-tabs">
  <a href="?tab=rundown"  class="tab-item active"><i class="ph ph-clock"></i> Susunan Rundown</a>
  <a href="?tab=playlist" class="tab-item"><i class="ph ph-music-notes"></i> Audio, Playlist & Arahan Teknis</a>
</nav>
```

---

## 2. Tab 1: Susunan Rundown

### 2.1 Wireframe (sesi Akad Nikah dari PDF; KPI ilustrasi hari-H)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────┐ ┌────────────────────┐ ┌────────────────────────┐ ┌──────────────────┐   │
│ │ 📋 Total Agenda│ │ ⏳ Total Rentang   │ │ 🎯 Progres Live        │ │ 👥 PIC & Vendor  │   │
│ │ 33 · 3 sesi    │ │ 7 jam 25 menit     │ │ 15 selesai · 1 LIVE ·  │ │ 17 unik (teks    │   │
│ │                │ │ (100+120+225 mnt)  │ │ 17 menunggu [████░] 45%│ │ PIC template PDF)│   │
│ └────────────────┘ └────────────────────┘ └────────────────────────┘ └──────────────────┘   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Sesi: [💍 Lamaran (11)] [★ 📜 Akad Nikah (8)] [🏛️ Resepsi (14)]   Tampilan: (•) Satu sesi ( ) Semua│
│ Mode: [📋 Tabel] [⏱️ Timeline]            📅 Minggu, 28 Maret 2027 · 07.00 – 09.00 · 8 agenda   │
│ [ + Tambah Agenda ]  [ 💬 Salin WA Sesi Ini ]  [ 💬 Salin WA Semua ]  [ 📥 Cetak PDF Cue Card ] │
│ ✏️ Kelola sesi di Edit Acara (Menu 01)                                                         │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Waktu          Dur  Agenda                    PIC / Vendor      Keterangan            Live  │
│  1 07.00 – 07.20  20m  Persiapan                 CMP & CMW         Penghulu datang …     🟢    │
│  2 07.21 – 07.25   5m  Pembukaan acara           MC                Pembukaan akad nikah  🟢    │
│  3 07.26 – 07.35  10m  Pembacaan ayat suci       Qori              Pembacaan ayat-ayat…  🟢    │
│  4 07.36 – 07.45  10m  Khutbah                   Ustadz/Kyai       Kutbah pernikahan …   🟢    │
│  5 07.46 – 07.50   5m  Ijab Kabul ⭐             Penghulu          Dipimpin penghulu …   🟡 LIVE│
│  6 07.51 – 08.00  10m  Doa bersama               Kyai/penghulu     Doa dan buku nikah    ⚪    │
│  7 08.01 – 08.10  10m  Sesi foto                 Fotografer        Foto & maskawin       ⚪    │
│  8 08.11 – 09.00  49m  Selesai akad              MC                Menuju resepsi        ⚪    │
│  Rekap sesi: 8 agenda · rentang 07.00 – 09.00 (2 j 00 m) · 4 selesai, 1 LIVE, 3 menunggu       │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 Agenda terbuka tampil "10.40 – selesai" dan durasi "–" (mis. Ramah Tamah pada sesi Lamaran).
```

Durasi baris 8 = 49 menit (08.11–09.00), sesuai PDF.

### 2.2 Perilaku

1. **Tabel mandiri per sesi** dengan penomoran dari 1 pada tiap sesi. Mode *Satu sesi* menampilkan tab terpilih; mode *Semua* menumpuk tabel semua sesi (untuk mencetak "Master Event Rundown Book"). Header sesi menampilkan tanggal, rentang jam, lokasi, dan jumlah agenda dari master sesi.
2. **Dua tampilan**: *Tabel* (kolom di atas) dan *Timeline* (kartu vertikal dengan penanda waktu, garis alur, dan drag handle; jeda antar agenda terlihat).
3. **KPI**: *Total rentang* = jumlah rentang per sesi (agenda paralel tidak dijumlahkan; [BE §2.1](BE.md#21-waktu-dan-durasi)), progres live dari status agenda, PIC unik = teks PIC tak kosong ∪ vendor tertaut. Kartu menggunakan `GET .../rundowns/summary` dan diperbarui tiap 5 detik pada hari-H.
4. **Live Stage Manager**: badge status 1-ketukan `⚪ Belum Mulai → 🟡 LIVE ON STAGE → 🟢 Selesai` (denyut kuning saat live). Beberapa agenda boleh LIVE bersamaan (sesi makan, hiburan, salaman). Mengetuk badge selesai membukanya kembali (`pending`) hanya lewat menu aksi, supaya tidak tertukar saat panik.
5. **PIC / Vendor**: teks PIC bebas atau pilih vendor dari Menu 04; agenda bervendor menampilkan nama vendor dan tombol `💬 WA` (nomor ternormalisasi). Agenda yang punya lagu menampilkan badge `🎵 N lagu` (dari `linked_songs`) yang membuka Tab Audio terfilter.
6. **Agenda terbuka**: waktu "10.40 – selesai" (atau hanya jam mulai) dan durasi "–".
7. **Pesan WhatsApp** (modal pratinjau, dapat disalin; teks disusun server, netral jenis acara):

```text
📋 *RUNDOWN ACARA AKAD NIKAH*
💍 *Pernikahan Sinta & Arif*
🗓 *Minggu, 28 Maret 2027*
🏛 *Sasana Kriya Ballroom, Jakarta Selatan*
────────────────────────
⏰ *07.00 - 07.20* (20m) : Persiapan
   👤 PIC: CMP & CMW
   📝 Penghulu datang dan melakukan pemeriksaan kelengkapan persyaratan administrasi, …

⏰ *07.46 - 07.50* (5m) : Ijab Kabul
   👤 PIC: Penghulu
   📝 Ijab kabul yang dipimpin oleh penghulu dan dilakukan oleh wali nikah.
────────────────────────
(Dibuat otomatis melalui Adicara Event Planner)
```
   Tombol "Salin WA Sesi Ini" memakai `session_id`; "Salin WA Semua" tanpa filter. Tanggal diambil dari `session_date` sesi (bila ada) atau tanggal Hari-H.
8. **Cetak PDF Cue Card**: A5/A4, huruf besar, kontras tinggi, per sesi (butuh dependensi PDF backend; sampai tersedia tombol disembunyikan dan halaman memakai `window.print()` dengan stylesheet cetak).

### 2.3 `RundownModal`
- **Sesi** (dropdown dari master sesi, tanpa `all`), **jam mulai / jam selesai** (input teks yang menerima `07.46` atau `07:46`; jam selesai boleh kosong = terbuka; centang "berakhir hari berikutnya"), durasi ditampilkan dari respons server (tidak diisi manual).
- Nama agenda\*, **PIC** (teks bebas atau pilih vendor Menu 04), keterangan, highlights (satu butir per baris, disimpan sebagai daftar "- …"), status (Belum / Live / Selesai).
- Galat `invalid_input` (selesai sebelum mulai tanpa centang hari berikutnya) ditampilkan di bawah field jam.
- Tambah agenda dari template: menu "Terapkan Template" dengan pilihan *Tambah yang belum ada* / *Ganti template yang belum berjalan* (menampilkan jumlah agenda live/selesai yang dipertahankan) dan warning `scope_missing` bila ada sesi template yang sudah dihapus.

---

## 3. Tab 2: Audio, Playlist & Arahan Teknis

### 3.1 Wireframe (30 lagu PDF; kolom tag/format/kesiapan adalah tambahan Adicara)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌────────────────────────┐      │
│ │ 🎵 Koleksi     │ │ 🎸 Format        │ │ 🏷️ Momen         │ │ ⚡ Kesiapan             │      │
│ │ 30 lagu        │ │ 30 Digital       │ │ 6 momen terpeta- │ │ 0 siap · 30 perlu       │      │
│ │ (template PDF) │ │ (belum diatur)   │ │ kan              │ │ ditinjau                │      │
│ └────────────────┘ └──────────────────┘ └──────────────────┘ └────────────────────────┘      │
│ 🔍 [ judul / artis / cue … ]  Momen:[Semua ▼] Format:[Semua ▼] Status:[Semua ▼]               │
│ [ + Tambah Audio ] [ 💬 Salin WA Soundman & Band ] [ 📥 Unduh Cue Sheet PDF ] [Terapkan Template ▾]│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Judul · Penyanyi                  Momen                 Format    Cue (arahan soundman)  Agenda        Status│
│  1 Akad · Payung Teduh               Akad Khidmat          Digital   –                      –             ⚠ Tinjau│
│  2 Janji Suci · Yovie & Nuno         Akad Khidmat          Digital   –                      –             ⚠ Tinjau│
│  5 Marry You · Maroon 5              Kirab / Entrance      Digital   –                      –             ⚠ Tinjau│
│ 15 Sugar · Maroon 5                  Santap & Ramah Tamah  Digital   –                      –             ⚠ Tinjau│
│ 28 Shallow · Lady Gaga               Games & Lempar Bunga  Digital   –                      –             ⚠ Tinjau│
│ … 25 lagu lainnya                                                                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Pada template semua lagu `digital_playback` + `needs_review` + tanpa tautan agenda, karena PDF hanya memuat judul dan penyanyi. Setelah pengguna menautkan lagu ke agenda dan mengisi cue, kolom Agenda menampilkan mis. "Iring-iring (09.36–09.40)" dan KPI berubah.

### 3.2 Komponen
1. **`AudioKpi`** dari `PlaylistOverview`: total, sebaran format, momen terpetakan (tag berbeda yang punya lagu), kesiapan (`ready` / `needs_review` / `missing_file`).
2. **Moment tag** dari registry jenis acara (pernikahan: 6 tag; seminar, ulang tahun, dll. tag mereka sendiri) ditambah tag kustom. Daftar tag tertulis di [TEMPLATE_EVENT bab 6](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#6-moment-tag-audio-satu-set-per-jenis-sumber-tunggal-untuk-06); tidak ada daftar kedua di FE.
3. **Tautan agenda**: dropdown seluruh agenda proyek dikelompokkan per sesi; menautkan menambah badge `🎵 N lagu` di Tab 1.
4. **Catatan cue**: textarea (titik mulai, fade in/out, ducking saat MC berbicara, level volume, titik berhenti). **Preview**: tombol Spotify/YouTube (hanya `http/https`, `rel="noopener noreferrer"`).
5. **`PlaylistModal`**: judul\*, artis\*, momen (dropdown + ketik baru), format (Live Band / Akustik / Digital Playback / Sound FX), tautan agenda, Spotify, YouTube, catatan cue, status (Siap Diputar / Review / File Belum Ada).
6. **Salin WA Soundman & Band**: teks terstruktur per momen/agenda (contoh: `📜 AKAD NIKAH (07.00–09.00)` → `• Akad - Payung Teduh (Format: Digital Playback)` → `↳ Cue: …`) disusun server berdasarkan tautan agenda dan tag; **Unduh Cue Sheet PDF** untuk meja mixer (tombol tersembunyi sampai dependensi PDF ada).

---

## 4. Responsif
- < 768px: KPI grid 2×2; tabel agenda menjadi *Rundown Cue Cards* (badge waktu tebal di kiri atas, nama agenda besar, badge live 1-ketukan, tombol `💬 WA PIC`); Tab Audio menjadi kartu per lagu dengan tombol preview besar.

## 5. Pengujian Frontend
`node:test`: parse `07.46`/`07:46`/`7.46` → `07:46` dan penolakan nilai tak valid, format durasi "7 jam 25 menit", tampilan agenda terbuka, pengelompokan agenda per sesi, penyusunan teks tombol salin (kecuali yang disusun server), pemetaan kode error (`invalid_input`, `scope_missing`, `session_in_use`).
