# Registry Event Type dan Template per Jenis Acara

Data untuk `EventTypeSpec` ([README bab 3](README.md#3-registry-event-type-di-kode-bukan-basis-data)) dan template bawaan sembilan jenis acara Adicara.

> **Sumber data.** Pernikahan (💍) bersumber dari `Wedding_Planner.pdf` dan tabelnya ada di dokumen modul masing-masing (02 BE §5, 03 BE §6, 04 BE §5, 06 BE §5). Delapan jenis lainnya (🕌 🍼 🏢 🚀 🎂 🎓 🎤 ✨) **tidak ada di PDF**: isi di bawah adalah **usulan Adicara** (ditandai `origin: adicara`) yang perlu ditinjau pemilik produk sebelum dirilis. Template hanya struktur dan estimasi; tidak ada pembayaran, vendor, atau tamu (lihat [README bab 5](README.md#5-template-vs-data-contoh)).

---

## 1. Aturan Template Jadwal (berlaku untuk semua jenis)

Setiap tugas template menyimpan **offset hari terhadap Hari-H** pada horizon acuan `R`: pernikahan `R = 180` (offset di 02 BE §5), jenis lain memakai template `general` dengan `R = 60` (bab 2). Generator ("Generate Jadwal Otomatis", `POST .../tasks/generate-defaults`):

```text
avail = event_date − max(prep_start_date, hari_ini)          # dalam hari, zona waktu proyek
k0    = LeadTimeDays(jenis) / R                              # skala horizon baku jenis acara
k     = k0                       bila avail ≥ LeadTimeDays    # cukup waktu: pakai horizon baku
k     = k0 × avail / LeadTimeDays bila avail < LeadTimeDays  # waktu pendek: mampatkan proporsional + warning lead_time_compressed
start = event_date − round(offset_mulai × k) ; end = event_date − round(offset_selesai × k)
start = max(start, max(prep_start_date, hari_ini))           # tidak ada tugas di masa lalu
bila avail < 3: semua tugas dijepit ke [hari_ini, event_date] + warning lead_time_too_short
```

Contoh terhitung: pernikahan dengan `avail = 175` (Proyek Contoh): `k = 175/180`, tugas 1 (H-180) jatuh pada hari ini. Aqiqah dengan `avail = 10`: `k = (21/60) × 10/21 ≈ 0,167`; "Gladi resik" (H-1…H-0) menjadi H-0…H-0 (durasi minimal 1 hari). `round()` = bilangan bulat terdekat, setengah naik (`floor(x + 0,5)`); bukan pembulatan bankir. Durasi tugas selalu `max(selesai − mulai, 1)` (konvensi PDF Hal. 3).

Setiap offset berada di rentang `0…R` (tugas paling awal tepat di H-`R`, yang oleh penskalaan jatuh pada awal horizon baku `LeadTimeDays`); offset di atas `R` tidak diizinkan karena akan jatuh sebelum horizon dan hanya dijepit ke hari ini tanpa arti. Aturan ini dijaga tes `TestEveryTemplateOffsetIsWithinItsHorizon` ([02 BE §6](../02_TIMELINE_DAN_TUGAS/BE.md#6-pengujian)); pemeriksaan skrip atas seluruh template menemukan satu pelanggaran (Seminar "Undang dan konfirmasi pembicara" H-70…H-56, kini H-58…H-50: setelah panitia inti terbentuk, selesai tepat sebelum registrasi dibuka).

Penskalaan diperlukan karena offset tetap H-180…H-4 akan menjatuhkan tanggal tugas ke masa lalu untuk acara berjarak pendek; template juga memuat tugas H-3…H-0 (gladi resik, Hari-H).

---

## 2. Template `general` (15 tugas, `R = 60`) untuk Jenis Non-Pernikahan

`template_key = general.<slug>`. Scope semua `all` kecuali dicatat. Dipakai untuk 8 jenis non-pernikahan; tiap jenis menambah tugas khusus (bab 6–13). Kolom *E* = termasuk pada pratinjau ringkas.

| No | Tugas | Kategori | Fase | Offset (R=60) |
|:-:|---|---|---|:-:|
| 1 | Tetapkan tujuan, tanggal, dan panitia inti | Perencanaan | 1 Perencanaan | H-60 s/d H-58 |
| 2 | Tetapkan anggaran total | Keuangan | 1 Perencanaan | H-58 s/d H-56 |
| 3 | Survei dan pilih venue | Venue | 1 Perencanaan | H-56 s/d H-50 |
| 4 | Susun daftar tamu/peserta awal | Tamu | 1 Perencanaan | H-52 s/d H-48 |
| 5 | Susun konsep dan rundown awal | Acara | 1 Perencanaan | H-48 s/d H-44 |
| 6 | Riset dan bandingkan vendor | Vendor | 2 Persiapan | H-46 s/d H-38 |
| 7 | Urus perizinan dan administrasi | Administrasi | 2 Persiapan | H-44 s/d H-30 |
| 8 | Kunci vendor utama dan bayar DP | Vendor | 2 Persiapan | H-38 s/d H-34 |
| 9 | Siapkan undangan dan publikasi | Undangan | 3 Pelaksanaan | H-34 s/d H-28 |
| 10 | Sebarkan undangan | Undangan | 3 Pelaksanaan | H-28 s/d H-26 |
| 11 | Belanja perlengkapan dan logistik | Logistik | 3 Pelaksanaan | H-26 s/d H-14 |
| 12 | Follow-up konfirmasi kehadiran (RSVP) | Tamu | 3 Pelaksanaan | H-14 s/d H-10 |
| 13 | Technical meeting vendor dan panitia | Panitia | 4 Pekan Terakhir | H-7 s/d H-6 |
| 14 | Gladi resik | Acara | 4 Pekan Terakhir | H-1 s/d H-0 |
| 15 | Hari-H acara | Acara | 4 Pekan Terakhir | H-0 s/d H-0 |

---

## 3. Ringkasan Registry (sembilan jenis)

| Key | Label | Ikon | `LeadTimeDays` | Sesi bawaan (`scope_key`: nama) |
|---|---|:-:|:-:|---|
| `wedding` | Pernikahan | 💍 | 180 | `lamaran`: 💍 Lamaran · `akad`: 📜 Akad Nikah · `resepsi`: 🏛️ Resepsi |
| `khitanan` | Khitanan | 🕌 | 45 | `prosesi_doa`: 🕌 Prosesi Doa · `hiburan`: 🎉 Hiburan · `jamuan`: 🍽️ Jamuan |
| `aqiqah` | Aqiqah | 🍼 | 21 | `prosesi_aqiqah`: 🕌 Prosesi Aqiqah · `syukuran`: 🍽️ Syukuran |
| `peresmian` | Peresmian | 🏢 | 60 | `pembukaan`: 🎭 Pembukaan · `prosesi_peresmian`: ✂️ Prosesi Peresmian · `ramah_tamah`: 🍽️ Ramah Tamah |
| `launching` | Launching | 🚀 | 90 | `opening`: 🎬 Opening · `peluncuran`: 🚀 Peluncuran · `networking`: 🤝 Networking |
| `birthday` | Ulang Tahun | 🎂 | 45 | `welcome`: 🎉 Welcome · `acara_utama`: 🎂 Acara Utama · `after_party`: 🎶 After Party |
| `wisuda` | Wisuda | 🎓 | 60 | `prosesi_senat`: 📜 Prosesi Senat · `pelantikan`: 🎓 Pelantikan · `foto_studio`: 📸 Foto Studio |
| `seminar` | Seminar | 🎤 | 75 | `registrasi`: 📝 Registrasi · `sesi_pagi`: 🎤 Sesi Pagi · `coffee_break`: ☕ Coffee Break · `sesi_sore`: 🎤 Sesi Sore |
| `other` | Acara Lainnya | ✨ | 60 | `sesi_1`: 📋 Sesi 1 |

Setiap proyek juga mendapat sesi sistem `all` ("Umum / Semua Sesi").

**Tanggal sesi bawaan.** Setiap sesi bawaan lahir dengan `session_date = event_date` (`SessionSeed.SameDay = true`) kecuali `lamaran` pada pernikahan, yang tanpa tanggal sampai pengguna mengisinya (lamaran biasanya berbeda hari). Dengan begitu milestone tabungan, header rundown, dan pesan WhatsApp berfungsi tanpa langkah tambahan. Saat Hari-H diubah, sesi yang bertanggal Hari-H lama ikut pindah ke Hari-H baru; sesi bertanggal lain tidak diubah ([01 BE §4.3](../01_DASHBOARD/BE.md#43-ubah-tanggal-acara-dan-geser-jadwal)).

### 3.1 Sebutan antarmuka (`Labels`)

Hanya seminar yang berbeda dari bawaan; jenis lain memakai bawaan.

| Key | `Labels.Guest` | `Labels.GuestMenu` |
|---|---|---|
| semua kecuali `seminar` | Tamu | Buku Tamu & RSVP |
| `seminar` | Peserta | Peserta & RSVP |

## 4. Peran, Sisi Tamu, dan Kategori

| Key | Sisi tamu (`guest_side`) | Kategori tamu | Peran kontributor | Subjek dokumen |
|---|---|---|---|---|
| `wedding` | `pria` Pihak Pria (CPP) · `wanita` Pihak Wanita (CPW) | `teman` · `kolega` · `keluarga` · `tetangga` · `teman_ortu` · `vip` *(6 kategori PDF Hal. 13)* | `groom` · `bride` · `family` · `sponsor` · `other` | `cpp` · `cpw` · `ortu_cpp` · `ortu_cpw` · `bersama` |
| `khitanan` | `keluarga_ayah` Keluarga Ayah · `keluarga_ibu` Keluarga Ibu | `keluarga` · `tetangga` · `teman` · `tokoh_agama` · `vip` | `host` · `family` · `other` | `panitia` · `lembaga` · `umum` |
| `aqiqah` | `keluarga_ayah` · `keluarga_ibu` | `keluarga` · `tetangga` · `teman` · `tokoh_agama` | `host` · `family` · `other` | `panitia` · `lembaga` · `umum` |
| `peresmian` | `internal` Tuan Rumah/Internal · `eksternal` Undangan Eksternal | `pimpinan_vip` · `mitra` · `media` · `karyawan` · `warga_sekitar` · `umum` | `internal` · `sponsor` · `other` | `panitia` · `lembaga` · `umum` |
| `launching` | `internal` · `eksternal` | `media` · `kreator` · `mitra` · `klien` · `vip` · `umum` | `internal` · `sponsor` · `other` | `panitia` · `lembaga` · `umum` |
| `birthday` | *(kosong: tanpa pemisahan pihak)* | `keluarga` · `teman_sekolah` · `teman_kuliah` · `komunitas` · `vip` | `host` · `family` · `other` | `panitia` · `lembaga` · `umum` |
| `wisuda` | *(kosong)* | `wisudawan` · `orang_tua` · `dosen_pejabat` · `tamu_undangan` · `vip` | `internal` · `family` · `sponsor` · `other` | `panitia` · `lembaga` · `umum` |
| `seminar` | `internal` Panitia/Internal · `eksternal` Peserta Eksternal | `pembicara` · `peserta` · `sponsor` · `media` · `vip` | `internal` · `sponsor` · `other` | `panitia` · `lembaga` · `umum` |
| `other` | *(kosong)* | `umum` · `vip` | `host` · `internal` · `sponsor` · `other` | `panitia` · `lembaga` · `umum` |

Label tampilan Indonesia berasal dari registry (mis. `groom` → "Mempelai Pria", `ortu_cpp` → "Orang Tua CPP"). Istilah PDF "CMP/CMW" disamakan dengan CPP/CPW; nama dokumen PDF tetap disalin apa adanya.

### 4.1 Subjek dokumen, label widget, dan panduan (milik Menu 02)

Tiga bidang registry yang ditambahkan modul Timeline & Tugas ([02 BE bab 2](../02_TIMELINE_DAN_TUGAS/BE.md#2-domain--dto-go)): `AdminSubjects` (kunci dan label subjek dokumen; dikirim ke klien), `AdminGuide` (judul dan 4 langkah; dikirim ke klien), dan `AdminLabel` (judul widget Legalitas di dashboard; hanya dipakai provider `admin_docs`, tidak dikirim).

| Key | `AdminSubjects` (`key` → label) | `AdminLabel` | `AdminGuide.Title` |
|---|---|---|---|
| `wedding` | `cpp` CPP · `cpw` CPW · `ortu_cpp` Orang Tua CPP · `ortu_cpw` Orang Tua CPW · `bersama` Bersama | Kelengkapan Berkas KUA | Panduan Pengurusan Berkas Nikah |
| delapan jenis lain | `panitia` Panitia · `lembaga` Lembaga/Instansi · `umum` Umum | Kelengkapan Perizinan | Panduan Perizinan Acara |

Langkah panduan pernikahan ada di [02 BE §5.4](../02_TIMELINE_DAN_TUGAS/BE.md#54-panduan-birokrasi-pdf-hal-11) (catatan PDF Hal. 11). Langkah panduan **delapan jenis lain** adalah usulan Adicara, bukan dari PDF, dan belum ditinjau pemilik produk; syarat, biaya, dan instansi berbeda per daerah sehingga teksnya mengarahkan pengguna bertanya ke kantor setempat:

1. **RT/RW**: sampaikan rencana acara dan minta surat pengantar atau persetujuan lingkungan, terutama bila acara memakai jalan, tenda, atau pengeras suara.
2. **Kelurahan dan Polsek**: sampaikan pemberitahuan acara dengan membawa surat pengantar RT/RW dan fotokopi KTP penanggung jawab; tanyakan apakah acara Anda memerlukan izin keramaian.
3. **Izin keramaian (bila diperlukan)**: untuk acara yang ramai, memakai jalan umum, atau menghadirkan hiburan dan musik, ajukan ke Polsek setempat dengan rincian acara; tanyakan syarat dan biaya yang berlaku.
4. **Pengelola venue**: pastikan tanggal, jam, kapasitas, aturan lokasi (suara, parkir, dekorasi), dan kontrak atau surat izin penggunaan tempat secara tertulis.

Tugas khusus jenis pada bab 7–14 diisi kunci, kategori, dan fasenya dengan aturan [02 BE §5.5](../02_TIMELINE_DAN_TUGAS/BE.md#55-template-jenis-non-pernikahan-usulan-adicara).

### 4.2 Label sisi dan kategori tamu (milik Menu 05)

Tabel di atas hanya memuat kunci; label tampilan yang dikirim `GET /event-types` sebagai `guest_sides` dan `guest_categories` ([05 BE bab 7](../05_BUKU_TAMU/BE.md#7-sambungan-ke-menu-01)) ada di sini (format `kunci=Label`, dipisah `;`; tes keselarasan membaca baris ini). Label pernikahan mengikuti PDF Hal. 13 ("Teman Orang Tua"); label jenis lain adalah usulan Adicara, dan kata yang tabel bab 4 tulis tanpa label ("Internal"/"Eksternal" pada launching) memakai kata biasanya. Kunci cadangan `lainnya` (label "Lainnya") **tidak** ada di registry: ia dipakai bila kategori tidak diisi dan hanya muncul di penghitung bila ada tamu yang memakainya ([05 BE §2.8](../05_BUKU_TAMU/BE.md#28-ringkasan-penghitung-kategori-dan-pihak)).

| Key | Sisi (`guest_side`) | Kategori |
|---|---|---|
| `wedding` | `pria=Pihak Pria (CPP)`; `wanita=Pihak Wanita (CPW)` | `teman=Teman`; `kolega=Kolega`; `keluarga=Keluarga`; `tetangga=Tetangga`; `teman_ortu=Teman Orang Tua`; `vip=VIP` |
| `khitanan` | `keluarga_ayah=Keluarga Ayah`; `keluarga_ibu=Keluarga Ibu` | `keluarga=Keluarga`; `tetangga=Tetangga`; `teman=Teman`; `tokoh_agama=Tokoh Agama`; `vip=VIP` |
| `aqiqah` | `keluarga_ayah=Keluarga Ayah`; `keluarga_ibu=Keluarga Ibu` | `keluarga=Keluarga`; `tetangga=Tetangga`; `teman=Teman`; `tokoh_agama=Tokoh Agama` |
| `peresmian` | `internal=Tuan Rumah/Internal`; `eksternal=Undangan Eksternal` | `pimpinan_vip=Pimpinan/VIP`; `mitra=Mitra`; `media=Media`; `karyawan=Karyawan`; `warga_sekitar=Warga Sekitar`; `umum=Umum` |
| `launching` | `internal=Internal`; `eksternal=Eksternal` | `media=Media`; `kreator=Kreator`; `mitra=Mitra`; `klien=Klien`; `vip=VIP`; `umum=Umum` |
| `birthday` | – | `keluarga=Keluarga`; `teman_sekolah=Teman Sekolah`; `teman_kuliah=Teman Kuliah`; `komunitas=Komunitas`; `vip=VIP` |
| `wisuda` | – | `wisudawan=Wisudawan`; `orang_tua=Orang Tua`; `dosen_pejabat=Dosen/Pejabat`; `tamu_undangan=Tamu Undangan`; `vip=VIP` |
| `seminar` | `internal=Panitia/Internal`; `eksternal=Peserta Eksternal` | `pembicara=Pembicara`; `peserta=Peserta`; `sponsor=Sponsor`; `media=Media`; `vip=VIP` |
| `other` | – | `umum=Umum`; `vip=VIP` |

## 5. Kolom Metadata Proyek (`planner_projects.metadata`)

Semua jenis juga menerima `hero_title` (≤ 60 karakter; judul besar hero, mis. "Bismillah Nikah" pada PDF Hal. 1), `venue_name` (≤ 150), dan `city` (≤ 80). Kunci lain ditolak `400 invalid_metadata`.

| Key | Kolom khusus (tipe, wajib?) |
|---|---|
| `wedding` | `bride_name`\* · `groom_name`\* · `bride_parents` · `groom_parents` *(string ≤ 150; urutan nama mengikuti PDF "Nama Cewe & Nama Pria")* |
| `khitanan` | `child_name`\* · `parent_names` |
| `aqiqah` | `baby_name`\* · `birth_date` (date) · `parent_names` · `goat_count` (int 0–50) |
| `peresmian` | `organization_name`\* · `vip_name` |
| `launching` | `brand_name`\* · `product_name` · `target_media` |
| `birthday` | `celebrant_name`\* · `age` (int 1–120) · `theme` |
| `wisuda` | `institution_name`\* · `faculty` · `graduate_count` (int ≥ 0) |
| `seminar` | `organizer_name`\* · `keynote_speaker` · `expected_participants` (int ≥ 0) |
| `other` | `organizer_name` |

\* wajib. Hero: pernikahan menampilkan `bride_name & groom_name`; jenis lain menampilkan `title` dan kolom wajibnya sebagai subjudul.

## 6. Moment Tag Audio (satu set per jenis; sumber tunggal untuk 06)

| Key | Moment tag |
|---|---|
| `wedding` | Akad Khidmat · Kirab / Entrance · Prosesi Adat · Santap & Ramah Tamah · Games & Lempar Bunga · Penutup |
| `khitanan` | Sholawat & Doa · Hiburan Ceria · Jamuan Santai |
| `aqiqah` | Sholawat & Doa · Syukuran |
| `peresmian` | Mars Perusahaan · Prosesi Peresmian · Networking Santai |
| `launching` | Countdown & Reveal · Applause & Fanfare · Networking Ambience |
| `birthday` | Welcome Music · Happy Birthday Moment · Party / Dance · Chill / Closing |
| `wisuda` | Paduan Suara / Hymne · Prosesi Masuk · Pelantikan · Lagu Perpisahan |
| `seminar` | Opening Bumper · Background Presentasi · Jeda Istirahat · Closing Jingle |
| `other` | Pembukaan · Acara Utama · Penutup |

---

## 7. 🕌 Khitanan (`khitanan`)

- **Tugas tambahan** (di atas `general`, offset R=60): Booking dokter/tenaga medis H-40…H-38 · Pesan konsumsi syukuran H-20…H-18 · Siapkan busana khitan dan perlengkapan doa H-18…H-14.
- **Dokumen** (`template_key` = `khitanan.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Surat izin lingkungan RT/RW (opsional) (`lembaga`) · Konfirmasi dokter/klinik (`panitia`) · Pemesanan tenda/kursi (`panitia`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `katering` Katering · `medis` Dokter/Klinik · `dekorasi` Dekorasi/Tenda · `dokumentasi` Dokumentasi · `hiburan` Hadroh/Hiburan.
- **Kategori belanja:** Busana dan perlengkapan · Souvenir · Konsumsi.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| prosesi_doa | 08.00–08.30 | Persiapan dan penyambutan keluarga | Panitia |
| prosesi_doa | 08.30–08.40 | Pembukaan dan sambutan keluarga | MC |
| prosesi_doa | 08.40–09.00 | Pembacaan Al-Qur'an dan shalawat | Qari/Ustadz |
| prosesi_doa | 09.00–09.15 | Doa dan tausiyah singkat | Ustadz |
| hiburan | 09.15–10.00 | Hadroh / hiburan | Pengisi acara |
| hiburan | 10.00–10.15 | Games anak | MC |
| jamuan | 10.15–11.30 | Santap bersama | Katering |
| jamuan | 11.30–11.45 | Foto bersama keluarga | Fotografer |
| jamuan | 11.45–12.00 | Penutupan | MC |

## 8. 🍼 Aqiqah (`aqiqah`)

- **Tugas tambahan:** Tentukan hari pelaksanaan dan ustadz H-18…H-16 · Pesan kambing dan katering aqiqah H-30…H-26 · Siapkan dokumen kelahiran dan nama H-40…H-36.
- **Dokumen** (`template_key` = `aqiqah.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Akta/surat keterangan lahir (`umum`) · Surat izin lingkungan (opsional) (`lembaga`) · Konfirmasi rumah potong/penyedia kambing (`panitia`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `kambing` Penyedia Kambing · `katering` Katering/Box Nasi · `dekorasi` Dekorasi · `dokumentasi` Dokumentasi.
- **Kategori belanja:** Box dan kemasan · Souvenir · Perlengkapan acara.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| prosesi_aqiqah | 09.00–09.15 | Pembukaan dan penyambutan | MC |
| prosesi_aqiqah | 09.15–09.30 | Pembacaan Al-Qur'an dan shalawat | Qari/Ustadz |
| prosesi_aqiqah | 09.30–09.45 | Pencukuran rambut dan pemberian nama | Ustadz dan orang tua |
| prosesi_aqiqah | 09.45–10.00 | Doa aqiqah dan tausiyah | Ustadz |
| syukuran | 10.00–11.30 | Santap bersama dan ramah tamah | Katering |
| syukuran | 11.30–11.45 | Pembagian box/nasi aqiqah | Panitia |
| syukuran | 11.45–12.00 | Foto bersama dan penutupan | Fotografer / MC |

## 9. 🏢 Peresmian (`peresmian`)

- **Tugas tambahan:** Koordinasi izin lingkungan/pengelola gedung H-44…H-36 · Siapkan prosesi gunting pita dan daftar VIP H-30…H-24 · Undang media dan mitra H-28…H-20.
- **Dokumen** (`template_key` = `peresmian.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Izin keramaian/lingkungan (RT/RW, kelurahan, Polsek) (`lembaga`) · Izin pengelola gedung/venue (`lembaga`) · Izin usaha (NIB/SIUP, bila relevan) (`lembaga`) · Surat pemberitahuan ke Dinas terkait (`lembaga`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `katering` · `av` Audio Visual · `dekorasi` · `dokumentasi` · `mc` · `keamanan` Keamanan.
- **Kategori belanja:** Perlengkapan prosesi (pita, gunting, plakat) · Cetak (spanduk, backdrop) · Souvenir.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| pembukaan | 08.00–08.10 | Registrasi tamu | Panitia |
| pembukaan | 08.10–08.15 | Pembukaan MC | MC |
| pembukaan | 08.15–08.30 | Sambutan pimpinan | Pimpinan |
| prosesi_peresmian | 08.30–08.40 | Gunting pita / peresmian | VIP dan pimpinan |
| prosesi_peresmian | 08.40–09.00 | Tur fasilitas / showcase | Pemandu |
| ramah_tamah | 09.00–10.00 | Ramah tamah dan jamuan | Katering |
| ramah_tamah | 10.00–10.10 | Doa dan penutup | MC |

## 10. 🚀 Launching (`launching`)

- **Tugas tambahan:** Finalisasi materi produk dan keynote H-40…H-30 · Undang media/pers dan kreator H-30…H-20 · Uji teknis demo dan AV H-6…H-4.
- **Dokumen** (`template_key` = `launching.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Izin venue/pengelola (`lembaga`) · Perjanjian media partner (`panitia`) · Izin keramaian (bila massal) (`lembaga`) · Lisensi musik/hak cipta (LMKN) bila memutar musik (`lembaga`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `av` · `pr` PR/Publikasi · `katering` · `talent` Talent/MC · `dokumentasi` · `produksi_konten` Produksi Konten.
- **Kategori belanja:** Press kit · Merchandise · Cetak (backdrop, standee).
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| opening | 13.00–13.30 | Registrasi dan welcome drink | Panitia |
| opening | 13.30–13.35 | Pembukaan MC | MC |
| opening | 13.35–13.50 | Sambutan founder/CEO | Founder |
| peluncuran | 13.50–14.10 | Presentasi brand dan visi | Presenter |
| peluncuran | 14.10–14.20 | Video teaser | Tim AV |
| peluncuran | 14.20–14.30 | Product reveal | Founder |
| peluncuran | 14.30–15.00 | Demo produk dan tanya jawab | Tim produk |
| networking | 15.00–15.30 | Sesi pers dan wawancara | PR |
| networking | 15.30–17.00 | Networking dan jamuan | Katering |

## 11. 🎂 Ulang Tahun (`birthday`)

- **Tugas tambahan:** Tentukan tema dan dresscode H-52…H-46 · Pesan kue dan dekorasi H-30…H-24 · Siapkan playlist dan games H-14…H-10.
- **Dokumen** (`template_key` = `birthday.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Izin venue/pengelola (bila di luar rumah) (`lembaga`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `katering` · `kue` Kue · `dekorasi` · `hiburan` · `dokumentasi`.
- **Kategori belanja:** Dekorasi dan balon · Souvenir/goodie bag · Aksesori photobooth.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| welcome | 17.00–17.30 | Registrasi dan sambutan tamu | Panitia |
| welcome | 17.30–17.35 | Pembukaan MC | MC |
| acara_utama | 17.35–17.50 | Sambutan orang tua/panitia | Keluarga |
| acara_utama | 17.50–18.00 | Tiup lilin dan potong kue | Yang berulang tahun |
| acara_utama | 18.00–18.30 | Games dan hiburan interaktif | MC |
| acara_utama | 18.30–19.30 | Santap dan ramah tamah | Tamu |
| after_party | 19.30–21.00 | Musik, karaoke, atau DJ set | DJ / pengisi acara |

## 12. 🎓 Wisuda (`wisuda`)

- **Tugas tambahan:** Koordinasi senat/fakultas H-56…H-48 · Alokasi toga dan tiket keluarga H-34…H-26 · Gladi prosesi H-3…H-2.
- **Dokumen** (`template_key` = `wisuda.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Izin penggunaan gedung/auditorium (`lembaga`) · Daftar wisudawan dari akademik (`panitia`) · Izin dokumentasi (`umum`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `katering` · `dokumentasi` · `audio` · `percetakan` Percetakan · `toga` Penyedia Toga.
- **Kategori belanja:** Toga dan atribut · Cetak · Bunga/hadiah.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| prosesi_senat | 08.00–08.30 | Registrasi wisudawan dan tamu | Panitia |
| prosesi_senat | 08.30–08.45 | Prosesi masuk senat | Protokol |
| prosesi_senat | 08.45–08.50 | Pembukaan dan Indonesia Raya | MC / paduan suara |
| prosesi_senat | 08.50–09.10 | Laporan rektor/dekan | Pimpinan |
| pelantikan | 09.10–09.20 | Pengukuhan lulusan | Rektor |
| pelantikan | 09.20–10.30 | Pemanggilan dan penyematan | Protokol |
| pelantikan | 10.30–10.45 | Pidato wisudawan terbaik | Wisudawan |
| pelantikan | 10.45–11.00 | Doa, penutup, Gaudeamus Igitur | MC / paduan suara |
| foto_studio | 11.00–14.00 | Foto studio dan keluarga | Fotografer |

## 13. 🎤 Seminar (`seminar`)

- **Tugas tambahan:** Undang dan konfirmasi pembicara H-58…H-50 · Buka registrasi peserta H-50…H-48 · Siapkan materi, sertifikat, dan kit H-24…H-12.
- **Dokumen** (`template_key` = `seminar.<judul bergaris-bawah>`, biaya 0, jumlah "1 berkas"): Kontrak venue (`lembaga`) · Perjanjian pembicara (`panitia`) · Izin keramaian (bila > 500 orang) (`lembaga`) · Lisensi musik (LMKN) bila relevan (`lembaga`).
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `katering` · `av` · `streaming` Live Streaming · `dokumentasi` · `percetakan`.
- **Kategori belanja:** Seminar kit · ID card dan lanyard · Plakat/sertifikat · Cetak (banner, backdrop).
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| registrasi | 07.30–08.00 | Registrasi dan welcome kit | Panitia registrasi |
| registrasi | 08.00–08.05 | Indonesia Raya | MC dan hadirin |
| registrasi | 08.05–08.15 | Sambutan pembuka | Ketua panitia |
| sesi_pagi | 08.15–09.00 | Keynote speech | Keynote speaker |
| sesi_pagi | 09.00–10.00 | Panel diskusi | Moderator dan panelis |
| sesi_pagi | 10.00–10.15 | Tanya jawab | Moderator |
| coffee_break | 10.15–10.45 | Coffee break dan networking | Katering |
| sesi_sore | 10.45–12.00 | Workshop / breakout session | Fasilitator |
| sesi_sore | 12.00–12.10 | Penutupan dan foto bersama | MC |

## 14. ✨ Acara Lainnya (`other`)

- **Tugas tambahan:** tidak ada (hanya `general`).
- **Dokumen:** tidak ada template; pengguna menambah sendiri.
- **Pos anggaran:** bab 15.1 (persentase dan `item_key`); pemetaan kategori vendor → pos: bab 15.2.
- **Kategori vendor:** `venue` · `katering` · `dekorasi` · `hiburan` · `dokumentasi`.
- **Kategori belanja:** Perlengkapan · Cetak · Konsumsi.
- **Agenda bawaan:**

| Sesi | Waktu | Agenda | PIC |
|---|---|---|---|
| sesi_1 | 09.00–09.15 | Pembukaan | MC |
| sesi_1 | 09.15–11.00 | Acara utama | Penyelenggara |
| sesi_1 | 11.00–11.15 | Penutupan | MC |

---

## 15. Pos Anggaran Template Non-Pernikahan

Template pos anggaran jenis non-pernikahan berupa **persentase dari `target_budget`** (jumlah tiap jenis = 100). Nominal = `round(persen × target_budget / 100)` dibulatkan ke Rp 1.000; selisih pembulatan dibebankan ke pos `dana_darurat` (atau pos terakhir bila jenisnya tidak punya pos itu). Setiap pos berstatus `source = 'template'`, `sub_event_scope = all`, dan `item_key` serta `category` seperti tabel di bawah (kolom Kategori ditambahkan saat Menu 03 diimplementasikan: `planner_budget_items.category` bertipe `NOT NULL`, sedangkan versi awal tabel tidak memuatnya). Bila `target_budget = 0`, nominal 0 dan pengguna mengisi sendiri. Pernikahan memakai 6 preset PDF (03 BE §6.1) dan 14 pos PDF (03 BE §6.2), bukan persentase.

### 15.1 Pos dan persentase

| Jenis | `item_key` | Nama pos | Kategori | % pagu |
|---|---|---|---|--:|
| `khitanan` | `konsumsi` | Konsumsi/katering | Catering | 40 |
| `khitanan` | `medis` | Medis (dokter, perlengkapan) | Medis | 15 |
| `khitanan` | `dekorasi_tenda` | Dekorasi dan tenda | Dekorasi | 12 |
| `khitanan` | `busana_mua` | Busana dan MUA | Busana & MUA | 8 |
| `khitanan` | `dokumentasi` | Dokumentasi | Dokumentasi | 8 |
| `khitanan` | `undangan_souvenir` | Undangan dan souvenir | Perlengkapan | 7 |
| `khitanan` | `hiburan` | Pengajian/hiburan | Hiburan | 5 |
| `khitanan` | `dana_darurat` | Dana darurat | Operasional | 5 |
| `aqiqah` | `kambing` | Kambing dan pengolahan | Catering | 45 |
| `aqiqah` | `katering_box` | Katering/box nasi | Catering | 25 |
| `aqiqah` | `dekorasi_tempat` | Dekorasi dan tempat | Venue & Dekorasi | 8 |
| `aqiqah` | `undangan_souvenir` | Undangan dan souvenir | Perlengkapan | 8 |
| `aqiqah` | `dokumentasi` | Dokumentasi | Dokumentasi | 5 |
| `aqiqah` | `ustadz_doa` | Ustadz/doa | Prosesi | 4 |
| `aqiqah` | `dana_darurat` | Dana darurat | Operasional | 5 |
| `peresmian` | `venue_dekor` | Venue dan dekorasi | Venue & Dekorasi | 20 |
| `peresmian` | `katering` | Katering | Catering | 20 |
| `peresmian` | `av_panggung` | Audio visual dan panggung | Audio Visual | 15 |
| `peresmian` | `publikasi_media` | Publikasi dan media | Publikasi | 15 |
| `peresmian` | `prosesi_vip` | Prosesi peresmian dan VIP | Prosesi | 10 |
| `peresmian` | `dokumentasi` | Dokumentasi | Dokumentasi | 8 |
| `peresmian` | `perizinan_keamanan` | Perizinan dan keamanan | Legalitas | 5 |
| `peresmian` | `dana_darurat` | Dana darurat | Operasional | 7 |
| `launching` | `produksi_av` | Produksi konten dan AV | Audio Visual | 25 |
| `launching` | `venue` | Venue | Venue & Dekorasi | 20 |
| `launching` | `publikasi_pr` | Publikasi/PR | Publikasi | 18 |
| `launching` | `katering` | Katering | Catering | 12 |
| `launching` | `talent_mc` | Talent dan MC | Hiburan | 8 |
| `launching` | `dokumentasi` | Dokumentasi | Dokumentasi | 7 |
| `launching` | `merchandise_kit` | Merchandise/kit | Perlengkapan | 5 |
| `launching` | `dana_darurat` | Dana darurat | Operasional | 5 |
| `birthday` | `katering_kue` | Katering dan kue | Catering | 30 |
| `birthday` | `venue` | Venue | Venue & Dekorasi | 20 |
| `birthday` | `dekorasi` | Dekorasi | Venue & Dekorasi | 15 |
| `birthday` | `hiburan` | Hiburan (DJ/MC) | Hiburan | 15 |
| `birthday` | `dokumentasi` | Dokumentasi | Dokumentasi | 8 |
| `birthday` | `souvenir` | Souvenir | Perlengkapan | 7 |
| `birthday` | `dana_darurat` | Dana darurat | Operasional | 5 |
| `wisuda` | `venue_dekor` | Venue dan dekorasi | Venue & Dekorasi | 25 |
| `wisuda` | `katering` | Katering | Catering | 25 |
| `wisuda` | `dokumentasi` | Dokumentasi | Dokumentasi | 15 |
| `wisuda` | `toga_atribut` | Toga dan atribut | Perlengkapan | 10 |
| `wisuda` | `audio_paduan_suara` | Audio dan paduan suara | Audio Visual | 10 |
| `wisuda` | `cetak` | Cetak (buku, sertifikat) | Perlengkapan | 8 |
| `wisuda` | `dana_darurat` | Dana darurat | Operasional | 7 |
| `seminar` | `venue` | Venue | Venue & Dekorasi | 25 |
| `seminar` | `katering_coffee_break` | Katering dan coffee break | Catering | 20 |
| `seminar` | `honor_narasumber` | Honor narasumber | Honor | 15 |
| `seminar` | `av_streaming` | AV dan streaming | Audio Visual | 12 |
| `seminar` | `kit_sertifikat` | Kit dan sertifikat | Perlengkapan | 8 |
| `seminar` | `publikasi` | Publikasi | Publikasi | 8 |
| `seminar` | `dokumentasi` | Dokumentasi | Dokumentasi | 5 |
| `seminar` | `dana_darurat` | Dana darurat | Operasional | 7 |
| `other` | `venue` | Venue | Venue & Dekorasi | 30 |
| `other` | `katering` | Katering | Catering | 30 |
| `other` | `dekorasi` | Dekorasi | Venue & Dekorasi | 10 |
| `other` | `hiburan` | Hiburan | Hiburan | 10 |
| `other` | `dokumentasi` | Dokumentasi | Dokumentasi | 8 |
| `other` | `dana_darurat` | Dana darurat | Operasional | 12 |

### 15.2 Pemetaan kategori vendor → pos (`budget_item_keys`)

Dipakai untuk pagu vendor dan `lock-deal` ([04 BE §2.1–2.2](../04_VENDOR_DAN_BELANJA/BE.md#21-pagu-vendor-dari-skenario-aktif)). Semua pos 15.1 dan semua preset `<jenis>_pct` berada di sesi Umum (`all`), jadi pencocokan mencoba sesi vendor lebih dulu lalu `all` (tanpa itu vendor yang ditempatkan di sesi nyata tidak mendapat pagu dan `lock-deal` membuat pos kedua). Setiap `category_key` pada bagian "Kategori vendor" tiap jenis ada di sini, dan setiap `budget_item_keys` adalah `item_key` dari 15.1 untuk jenis yang sama; kategori yang berbagi satu pos (mis. `av` dan `streaming` → `av_streaming`) dibandingkan di tingkat kelompok pagu. Pemetaan pernikahan ada di 04 BE §5.1.

| Jenis | `category_key` vendor | `budget_item_keys` |
|---|---|---|
| `khitanan` | `katering` | `konsumsi` |
| `khitanan` | `medis` | `medis` |
| `khitanan` | `dekorasi` | `dekorasi_tenda` |
| `khitanan` | `dokumentasi` | `dokumentasi` |
| `khitanan` | `hiburan` | `hiburan` |
| `aqiqah` | `kambing` | `kambing` |
| `aqiqah` | `katering` | `katering_box` |
| `aqiqah` | `dekorasi` | `dekorasi_tempat` |
| `aqiqah` | `dokumentasi` | `dokumentasi` |
| `peresmian` | `venue` | `venue_dekor` |
| `peresmian` | `katering` | `katering` |
| `peresmian` | `av` | `av_panggung` |
| `peresmian` | `dekorasi` | `venue_dekor` |
| `peresmian` | `dokumentasi` | `dokumentasi` |
| `peresmian` | `mc` | `prosesi_vip` |
| `peresmian` | `keamanan` | `perizinan_keamanan` |
| `launching` | `venue` | `venue` |
| `launching` | `av` | `produksi_av` |
| `launching` | `pr` | `publikasi_pr` |
| `launching` | `katering` | `katering` |
| `launching` | `talent` | `talent_mc` |
| `launching` | `dokumentasi` | `dokumentasi` |
| `launching` | `produksi_konten` | `produksi_av` |
| `birthday` | `venue` | `venue` |
| `birthday` | `katering` | `katering_kue` |
| `birthday` | `kue` | `katering_kue` |
| `birthday` | `dekorasi` | `dekorasi` |
| `birthday` | `hiburan` | `hiburan` |
| `birthday` | `dokumentasi` | `dokumentasi` |
| `wisuda` | `venue` | `venue_dekor` |
| `wisuda` | `katering` | `katering` |
| `wisuda` | `dokumentasi` | `dokumentasi` |
| `wisuda` | `audio` | `audio_paduan_suara` |
| `wisuda` | `percetakan` | `cetak` |
| `wisuda` | `toga` | `toga_atribut` |
| `seminar` | `venue` | `venue` |
| `seminar` | `katering` | `katering_coffee_break` |
| `seminar` | `av` | `av_streaming` |
| `seminar` | `streaming` | `av_streaming` |
| `seminar` | `dokumentasi` | `dokumentasi` |
| `seminar` | `percetakan` | `kit_sertifikat` |
| `other` | `venue` | `venue` |
| `other` | `katering` | `katering` |
| `other` | `dekorasi` | `dekorasi` |
| `other` | `hiburan` | `hiburan` |
| `other` | `dokumentasi` | `dokumentasi` |

### 15.3 Preset skenario non-pernikahan

Tab Simulasi (03) memakai tabel 15.1 yang sama sebagai preset `<jenis>_pct` (mis. `seminar_pct`): `POST /scenarios {"preset_key": "seminar_pct", "total_target": N}` membuat item skenario dengan `item_key` dan nominal persentase. Dengan begitu skenario, pos Realisasi, dan vendor memakai kunci yang sama seperti pernikahan.

### 15.4 Pos gulungan belanja

Daftar belanja (Menu 04) dapat digulung ke satu pos anggaran: angka pos dihitung view `planner_budget_lines` dari barang yang menunjuknya ([04 BE §2.4](../04_VENDOR_DAN_BELANJA/BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup)). Registry menyimpan pos bawaannya per jenis (`ShoppingRollup{Key, Name, Category}`); tombol "Hitung pos dari daftar belanja" mencari pos berkunci itu pada sesi mana pun dan membuatnya bila belum ada. Pernikahan: pos `seserahan` ada di 14 pos PDF ([03 BE §6.2](../03_KEUANGAN/BE.md#62-template-14-pos-realisasi-pdf-hal-56)). Delapan jenis lain: satu pos 15.1 yang paling dekat dengan kategori belanja jenis itu (pilihan Adicara; usulan sampai ditinjau pemilik produk, [04 BE bab 9 butir 6](../04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). `other` tidak punya pos 15.1 yang cocok, jadi `perlengkapan` dibuat (`source = 'manual'`, sesi Umum) bila belum ada. Menggulung mengganti estimasi persentase pos itu dengan jumlah belanja.

| Jenis | `item_key` | Nama pos | Kategori | Ada di 15.1 |
|---|---|---|---|:-:|
| `wedding` | `seserahan` | Seserahan | Hantaran | ✔ (14 pos PDF) |
| `khitanan` | `undangan_souvenir` | Undangan dan souvenir | Perlengkapan | ✔ |
| `aqiqah` | `undangan_souvenir` | Undangan dan souvenir | Perlengkapan | ✔ |
| `peresmian` | `prosesi_vip` | Prosesi peresmian dan VIP | Prosesi | ✔ |
| `launching` | `merchandise_kit` | Merchandise/kit | Perlengkapan | ✔ |
| `birthday` | `souvenir` | Souvenir | Perlengkapan | ✔ |
| `wisuda` | `toga_atribut` | Toga dan atribut | Perlengkapan | ✔ |
| `seminar` | `kit_sertifikat` | Kit dan sertifikat | Perlengkapan | ✔ |
| `other` | `perlengkapan` | Perlengkapan | Perlengkapan | – (dibuat bila belum ada) |

Kunci, nama, dan kategori pos non-pernikahan di atas sama dengan 15.1 (dibandingkan dengan skrip; `TestVendorCategoriesMatchTemplateEvent` menjaganya).

**Pos cadangan.** Tiga pos di atas juga merupakan pos khusus kategori vendor menurut 15.2: peresmian `prosesi_vip` (kategori `mc`), wisuda `toga_atribut` (`toga`), seminar `kit_sertifikat` (`percetakan`). Begitu vendor itu dikunci, `lock-deal` menautkannya ke pos tersebut, dan pos yang bertaut vendor tidak dapat digulung. Ditemukan saat implementasi Menu 04 (2026-10-23); bila pos bawaan ada tetapi tidak dapat digulung, daftar belanja digulung ke pos cadangan `perlengkapan` (Kategori Perlengkapan, sesi Umum, dibuat sekali di luar persentase pagu), seperti `other` ([04 BE bab 2.4](../04_VENDOR_DAN_BELANJA/BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup), [bab 9 butir 22](../04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Alternatifnya mengubah 15.2 atau 15.4 dan menunggu keputusan pemilik produk.

## 16. Pustaka Template di Kode

Template ini hidup di kode Go, bukan basis data, mengikuti pola goose (SQL di-embed) dan registry template frontend (ADR-0002):

```text
backend/internal/modules/planner/template/
  (registry)           # sudah ada di planner/domain/eventtype.go (dipakai semua modul), bukan di paket template
  timeline_wedding.go  # 44 tugas PDF + 6 tugas Adicara (02 BE §5)
  timeline_general.go  # bab 2 + tugas tambahan per jenis
  docs_*.go            # 15 dokumen KUA; daftar perizinan per jenis
  budget_wedding.go    # 6 preset + 14 pos PDF (03 BE §6)
  budget_percent.go    # bab 15
  vendor_shopping.go   # kategori vendor; 37 item seserahan (04 BE §5)
  rundown_*.go         # 11 + 22 agenda PDF; agenda bab 7–14 (06 BE §5)
  playlist_wedding.go  # 30 lagu PDF
```

Tiap applier (`TemplateApplier.Apply(ctx, projectID, req)`, transaksi dibawa `ctx`) idempoten untuk `append` (melewati `template_key` yang sudah ada) dan mengembalikan `{created, skipped, warnings}`.
