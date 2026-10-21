# Changelog Spesifikasi Planner

Riwayat revisi folder `docs/APP_SPEC/PLANNER_SPEC`. Hasil validasi hanya dicatat setelah dijalankan.

## 2026-10-21 — Menu 01 (BE) diimplementasikan: koreksi spesifikasi dari hasil membangunnya

Backend menu 01 sudah menjadi kode ([dokumen](../../backend/planner/2026-10-21-planner-menu-01.md)). Membangunnya dan mengujinya terhadap PostgreSQL 16.2 menemukan hal berikut; semua ✔ terbukti dengan eksekusi.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Kunci sesi bawaan bertentangan**: 01 BE §4.1 langkah 2 menyuruh `scope_key` bawaan diturunkan dari nama dengan aturan sesi buatan pengguna, tetapi "Akad Nikah" menghasilkan `akad_nikah`, sedangkan 02, 03, 04, 06 dan data emas memakai `akad` | `SessionSeed` membawa `scope_key` eksplisit; 01 BE §4.1 dan kontrak bab 4 dikoreksi |
| ✔ 2 | **SQL menu 01 tidak bisa jalan tanpa tabel modul 02–06**: kartu Hub, `usage` sesi, pindah sesi, dan geser tugas membaca/mengubah `planner_tasks`, `planner_budget_*`, `planner_vendors`, `planner_shopping_items`, `planner_guests`, `planner_rundowns` | Migrasi 005–012 dirilis bersama (kontrak bab 9 sudah membolehkan); 006–012 disalin otomatis dari DDL spec modul. Catatan di kontrak bab 1.2 |
| ✔ 3 | **Kueri Hub memakai `now()`**, sehingga batas tengah malam tidak bisa diuji | Waktu menjadi parameter (`$3::timestamptz`); definisi fase tidak berubah. 01 BE §5.3–5.4 diberi catatan |
| ✔ 4 | pgx menolak argumen yang tidak dipakai pernyataan (`unused argument`) | Pernyataan pindah-sesi (§4.6) dijalankan dengan jumlah argumen masing-masing |
| ○ 5 | `db.Querier(ctx)` tidak mungkin tanpa pool global; `Overlay` Proyek Contoh tanpa antarmuka; hints vs warnings tidak konsisten (kontrak bab 6 menyebut `tasks_not_shifted` dan `sessions_dates_unchanged` "warning", 01 BE §4.3 menyebutnya "hint") | `db.Querier(ctx, fallback)`; antarmuka `SampleOverlay`; respons PATCH memisahkan `warnings[]` dan `hints[]` |
| ○ 6 | Batas `event_end_date` ≤ +30 hari hanya ditulis untuk POST; PATCH bisa melewatinya | Berlaku juga di PATCH |

Sinkronisasi dokumen (tindak lanjut hari yang sama): `01_DASHBOARD/BE.md` disesuaikan dengan kode, yaitu tag `binding` dihapus dari DTO, bab 3.4 baru (bentuk respons nyata), urutan pembuatan proyek, aturan PATCH, `SampleOverlay`, catatan pgx pada §4.6, parameter waktu pada kueri Hub dan filter `status` pada `view=lite`, serta bab 6 yang kini memetakan tiap butir uji ke nama tesnya (24 tes diverifikasi ada) dan menandai yang belum (uji emas, latensi). Kontrak bab 1.1, 1.1.1, 3, 4 dan `TEMPLATE_EVENT` bab 16 disesuaikan.

`01_DASHBOARD/FE.md` ditulis ulang sebagai acuan pembangunan oleh pengembang FE lain (belum ada kode FE). Dirinci: tipe TypeScript lengkap dan fixture, fungsi acuan `format`/`countdown` dengan vektor uji, spesifikasi BFF, aturan wizard dan edit per kolom, pemetaan bidang widget, tabel galat, aksesibilitas, rencana uji dan QA, urutan kerja. Dikoreksi karena tidak sesuai backend yang sudah jadi: baris Tamu kartu Hub (hanya pax), widget Vendor (tanpa daftar kategori dan tombol WA), peringatan hapus sesi (FE tak bisa menghitung pos yang bentrok), isi langkah 3 wizard (tanpa jumlah template yang tak punya sumber), dan sub-navigasi menu 02–06 yang belum ada. Terbukti: kode di dokumen di-`tsc` strict dan dijalankan sebagai `node:test`; interface dibandingkan dengan 136 objek JSON dari backend asli, nol selisih. Belum: apa pun yang membutuhkan browser (partial Astro, dialog, drag-drop). Celah yang butuh keputusan produk ada di FE.md bab 15.

Wireframe `01_DASHBOARD/FE.md` (bab 7.1, 9.1) diaudit terhadap kode BE, `AppHeader.astro` yang ada, fixture, dan aturan dokumen itu sendiri, lalu dibuat ulang. Terbukti dari respons backend asli: kartu Proyek Contoh hari ini `total_tasks 0`, `total_paid 0`, `guest_pax_count 0`, keenam widget `null`, sesi Lamaran tanpa tanggal. Wireframe lama menggambar keadaan akhir tanpa menyebutnya. Dikoreksi: sapaan pengguna yang tidak ada di header, baris "Peserta Belum ada peserta", tombol Lihat Proyek Contoh di samping kartu contoh, tombol "Arsip & Kenangan" tanpa halaman, widget Keuangan (donut, status baris, format kas), "Berjalan: n" pada Timeline, judul widget Berkas, sub-navigasi yang semuanya aktif. Ditambah wireframe keadaan hari ini dan empat keadaan widget. Tiga celah baru tercatat (FE.md bab 15, butir 9–11); `01_DASHBOARD/BE.md` mengoreksi klaim bahwa `admin_docs.label` berasal dari registry. `01_DASHBOARD/README.md` dan `FE.md` lalu disamakan dengan versi terbaru: bagian yang sudah usang diganti, bukan ditambah catatan (widget Vendor tanpa tautan WA, kartu Hub hanya pax, Project Switcher di bilah sekunder, pengecualian SQL lintas-modul, jendela tugas mendesak 7 hari, tanggal Lamaran sampel 10 Okt 2026 menurut BE §4.5). Wireframe Menu 03 masih menulis Lamaran 15 Okt 2026 (belum disentuh). Perubahan aturan yang ditetapkan sebagai keputusan sementara karena spec diam: filter Kategori memuat semua jenis di registry, kartu ajakan selalu tampil, kartu selesai menaut ke workspace.

`00_KONTRAK_BERSAMA/README.md` dan `README.md` folder ini disamakan dengan kode: prasyarat `time/tzdata` dan `DB_MAX_CONNS` ditulis sudah ada (sebelumnya "wajib ditambahkan"), daftar warning dipisah menjadi `warnings[]`, `hints[]`, dan warning milik applier modul, pengecualian SQL lintas-modul dicatat, "tidak ada batas jumlah proyek" diganti batas teknis 100, dan Project Switcher dipindah dari "navbar" ke bilah sekunder. Entri FLOWCHART dihapus dari `01_DASHBOARD/README.md` karena filenya tidak ada.

Status implementasi: hanya **menu 01 backend**. Frontend dan menu 02–06 masih spesifikasi; widget dashboard dan `templates_available` kosong sampai modulnya terdaftar. Uji emas PDF (kontrak bab 9) belum bisa dijalankan.

## 2026-10-21 — Audit menu 01 Dashboard: bug siklus hidup, runtime, dan celah universal

Audit ini mengeksekusi DDL, SQL, dan perilaku library yang dirujuk spec 01 (bukan hanya membacanya). Temuan yang terbukti dengan eksekusi ditandai ✔; yang berasal dari penalaran atau pembacaan kode ditandai ○.

### Bug yang akan terjadi saat dibangun (terbukti)
| # | Temuan | Perbaikan |
|---|---|---|
| ✔ 1 | **Hapus proyek/akun gagal** bila proyek punya skenario anggaran: FK sesi `RESTRICT` diperiksa sebelum item skenario (cascade proyek → skenario → item) terhapus. `NO ACTION` biasa juga gagal. Lolos uji sederhana karena anak langsung proyek kebetulan terhapus lebih dulu | Semua FK ke sesi dan ke kontributor `DEFERRABLE INITIALLY DEFERRED` (migrasi 005–011); diuji: hapus proyek/akun bersih, hapus sesi terpakai tetap ditolak (saat commit), rename `scope_key` tetap merambat |
| ✔ 2 | **Hapus sesi gagal/menyimpang**: pemindahan baris bentrok indeks unik `item_key` (`catering` ada di Lamaran dan Resepsi → `23505`); item skenario tidak ada di `usage` tetapi menahan hapus (`23503`); `active_sub_events` menyimpan sesi yang sudah hilang | Algoritma pindah-sesi (01 BE §4.6): `item_key` bentrok dinetralkan, item skenario ikut, array dibersihkan; total anggaran tak berubah |
| ✔ 3 | **Zona waktu gagal di produksi**: image runtime `alpine:3.23` tanpa zoneinfo, `time/tzdata` tidak diimpor → `LoadLocation("Asia/Jakarta")` error (diuji dengan emulasi); semua hitung mundur dan pembuatan proyek gagal | `import _ "time/tzdata"` wajib ([kontrak 1.5](00_KONTRAK_BERSAMA/README.md#15-prasyarat-runtime-dibaca-dari-kode-dan-dockerfile)) |
| ✔ 4 | **Satu zona waktu rusak menggagalkan seluruh Hub** pengguna itu (`time zone "Foo/Bar" not recognized`); PostgreSQL menerima `asia/jakarta`, `UTC+7` yang tidak sama dengan Go | `CHECK` bentuk kanonik + pemicu keberadaan di migrasi 005; `400 invalid_timezone` |
| ✔ 5 | **Panic di goroutine provider mematikan seluruh proses API** walau ada `gin.Recovery()` | `recover` per goroutine, paralelisme 3, batas waktu 3 dtk, batas laju ([kontrak bab 7](00_KONTRAK_BERSAMA/README.md#7-dashboard-pola-summaryprovider)) |
| ✔ 6 | **Proyek contoh ganda** (tanpa indeks unik; klik ganda) | `planner_projects_sample_idx`; `POST .../sample` idempoten (201/200) |
| ✔ 7 | **Judul `""`/`"   "` lolos** `binding:"omitempty,max=255"` pada `*string`; DB menerima judul berspasi ujung dan berbaris baru | `CHECK` judul + validasi usecase |
| ✔ 8 | `NextSession.StartTime string` tidak bisa memindai `NULL` (pgx); jam keluar `07:00:00` bukan `HH:MM` | `*string` + `to_char`; konvensi pgx di kontrak bab 6 |
| ✔ 9 | `civil.Date` dipakai spec tetapi bukan dependensi `go.mod` | `time.Time` 00:00 UTC |

### Celah logika (penalaran/pembacaan kode)
| # | Temuan | Perbaikan |
|---|---|---|
| ○ 10 | Pool koneksi bawaan `max(4, CPU)` dipakai bersama `BearerAuth`; dashboard 6 query serentak per 30 dtk bisa menahan login | `DB_MAX_CONNS`, paralelisme 3 |
| ○ 11 | Wizard menawarkan template untuk modul yang belum dibangun (urutan bangun: `planner` dulu) → 500 | `TemplateApplier` terdaftar; `templates_available`; `invalid_template` |
| ○ 12 | `Apply(ctx, tx, …)` membocorkan `pgx` ke usecase | `TxManager.InTx` membawa transaksi lewat `context` |
| ○ 13 | `PATCH metadata` menimpa seluruh objek: dua tab saling menghapus | JSON Merge Patch |
| ○ 14 | Geser jadwal "setelah disimpan" lewat panggilan relatif terpisah bisa dobel (klik ganda) | FE bertanya dulu; `PATCH` dengan `shift_tasks` atomik; `shiftable_tasks` |
| ○ 15 | Klik ganda/timeout-retry pada wizard membuat proyek ganda | `client_token` + indeks unik |
| ○ 16 | `WriteGuard` memblokir `DELETE` proyek arsip (harus dibuka dulu) | pengecualian `DELETE` dan `PATCH …/status` |
| ○ 17 | `:id` bukan UUID (mis. `summary`) → 500 dari pgx | `404` untuk id tak valid |
| ○ 18 | Pembagi nol (pagu 0, tugas 0, berkas 0) → `NaN%` | `percent(n, d)` → `null`, keadaan kosong |
| ○ 19 | Kelebihan bayar satu pos mengurangi kewajiban pos lain di widget keuangan | `Remaining` = Σ bagian positif; `Overpaid` terpisah (juga 03 BE §3.5) |
| ○ 20 | Kartu Hub tak punya `total_estimate` untuk "semua lunas"; "Acara Terdekat" tak terdefinisi; urutan non-aktif | `total_estimate`, aturan `NearestProject`, urutan baru |
| ○ 21 | Project Switcher memanggil kueri Hub (agregat) pada setiap navigasi; `PlannerLayout` dan `404.astro` tidak ada di spec/repo | `view=lite`, `PlannerLayout.astro`, `404.astro` |
| ○ 22 | Nama sesi ganda membuat impor tamu ambigu; slug kosong/terlalu panjang melanggar `CHECK` | indeks nama unik; algoritma `scope_key` |
| ○ 23 | Judul 409/400 tanpa nama kolom → formulir tak bisa menandai kolom | pesan `<field>: <alasan>` |
| ○ 24 | Uang tanpa batas atas (> 2⁵³ rusak di JS) | batas 0…10¹³ |
| ○ 25 | Jumlah proyek/pembuatan tanpa batas laju | `PLANNER_MAX_PROJECTS` (bawaan 100), 10/menit |

### Celah universal
| # | Temuan | Perbaikan |
|---|---|---|
| ○ 26 | Acara beberapa hari (seminar, konferensi, wisuda, adat) dianggap "telah berlangsung" pada hari kedua | `event_end_date` + `phase`/`day_number` dari server |
| ○ 27 | Kategori vendor yang tidak diperlukan membuat progres tak pernah 100% | `skipped_vendor_categories` |
| ○ 28 | "Tamu/undangan/Buku Tamu" tertulis tetap untuk seminar | `Labels` registry |
| ○ 29 | Sesi bawaan lahir tanpa tanggal sehingga milestone/WA butuh langkah manual | `SameDay`: bertanggal Hari-H, ikut pindah saat Hari-H diubah (diuji) |
| ○ 30 | Menambah jenis acara bisa membuat registry dan `CHECK(event_type)` menyimpang | prosedur + uji keselarasan (dijalankan) |
| ○ 31 | Hero berdasarkan jam perangkat | `server_time` untuk koreksi; fase dari server |

### Dikoreksi selama audit
Dugaan awal bahwa `DATE → string` gagal pada pgx **salah untuk mode yang dipakai backend** (`QueryExecModeSimpleProtocol`): di mode itu berhasil; yang tetap benar adalah `NULL → string` gagal dan `TIME` keluar `HH:MM:SS`. Dugaan bahwa `ON DELETE RESTRICT` menggagalkan hapus proyek pada semua kasus juga salah: gagal hanya pada cascade dua tingkat (skenario). Rute Gin statis berdampingan dengan `:id` (klaim 05 BE) terbukti benar.

### Divalidasi
Validator dokumen kini menjalankan 239 pemeriksaan (sebelumnya 202), termasuk SQL yang diambil langsung dari dokumen: DDL 005–012, hapus akun hingga 14 tabel kosong, constraint (judul, zona, uang, tanggal, ganda), fase acara pada batas tengah malam WIB, pindah-sesi pada sampel golden (total tetap 74.150.000), `lite.sql`, dan keselarasan registry ↔ `CHECK(event_type)`. Belum dijalankan: kode Go modul Planner (belum ada), image Docker sebenarnya (Docker tidak terpasang), pengukuran latensi.

### Perlu keputusan produk
1. **`PLANNER_MAX_PROJECTS` bawaan 100** (dan 10 pembuatan/menit): README awal menyebut "tanpa batas". Ini batas teknis yang bisa dinaikkan lewat env; jika ingin benar-benar tanpa batas, setel `0`.
2. `DB_MAX_CONNS` bawaan 10 perlu disesuaikan dengan kapasitas VPS dan pgBouncer.
3. Acara maksimal 30 hari (`event_end_date`) dan horizon Hari-H 5 tahun adalah angka usulan.

## 2026-10-20 — Revisi besar: selaras PDF, terintegrasi, universal, dan mengikuti arsitektur yang ada

Pemeriksaan terhadap `Wedding_Planner.pdf` dan arsitektur Adicara (Astro SSR, Go/Gin, PostgreSQL, goose) menemukan bahwa spesifikasi sebelumnya (a) menyimpang dari PDF pada beberapa angka dan isi, (b) mengklaim integrasi tanpa mekanisme dan DDL-nya tidak bisa dijalankan, serta (c) universal hanya pada tingkat kolom, sedangkan seed, DTO, dan teks FE khusus pernikahan.

### Dokumen baru
- `00_KONTRAK_BERSAMA/README.md` — kontrak lintas modul (arsitektur BE/FE yang ada, migrasi 005–012, isolasi, registry, master sesi, template vs data contoh, konvensi, kode error, matriks integrasi, uji emas).
- `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` — registry 9 jenis acara, penskalaan jadwal, template `general` dan non-pernikahan (usulan Adicara, belum ditinjau pemilik produk).

### Sesuai PDF
| Temuan | Perbaikan |
|---|---|
| 03: total dibayar Rp 24.550.000, sisa 49.600.000, rasio 33,1%, Lamaran "lunas" | Mengikuti PDF Hal. 5–6: **33.550.000 / 40.600.000 / 45,2%**; Lamaran dibayar 4.750.000, sisa 2.400.000 |
| 03: 14 pos "dari PDF" hanya cocok pada subtotal | Nama dan nilai 14 baris PDF apa adanya (Lamaran: Catering, Gaun/Kebaya, Fotografer, Cincin, Dekorasi; Resepsi: Mahar, Catering, Rias, Gedung & Dekorasi, Undangan, Dokumentasi, Seserahan, Hiburan) |
| 03: tabungan 3 setoran lump-sum; "target proyeksi" Dimas 47,5 jt (63,3%) | 22 setoran bulanan Feb–Des 2026; porsi 63,2% / 36,8% dari total terkumpul |
| 04: subtotal kategori belanja Make Up 1.488.900 dan Mandi 989.700 | 1.268.900 dan 990.700 (jumlah kategori = 4.578.500) |
| 04: kategori vendor tanpa MUA/Photography/Attire, dengan "Biaya KUA" sebagai vendor | 10 kategori PDF; 7 FIX = Venue, Decoration, WO, MUA, Catering, Photography, Attire |
| 06: 22 agenda mengubah susunan PDF dan memberi nama fiktif pada "PIC bawaan PDF" | 11 + 22 agenda persis PDF Hal. 14; PIC kosong tetap kosong; Akad 8 + Resepsi 14 |
| 05: "20 tamu acuan PDF Hal. 13" | PDF tidak memuat nama; dua puluh tamu dilabeli fiktif dan hanya ada di Proyek Contoh |
| 02: "disalin persis" padahal nominal/jumlah dokumen tidak ada di PDF | Nama dan urutan persis; nilai tambahan ditandai saran Adicara; durasi mengikuti konvensi PDF (selesai − mulai) |
| Tautan Shopee pribadi pembuat PDF ikut tersalin ke tiap proyek | Tidak disalin |

### Terintegrasi
| Temuan | Perbaikan |
|---|---|
| `sub_event_scope` teks bebas, berbeda per modul; tabel sesi 06 tidak dipakai modul lain | Master sesi `planner_event_sessions` + FK komposit di semua tabel; tanggal sesi tersimpan |
| FK sirkular `budget_items ↔ vendors`; 6 dari 15 tabel gagal dibuat | FK satu arah; migrasi 005–012 berurutan; seluruhnya dijalankan |
| `paid_total` (kolom generated) diisi oleh sinkron belanja | Rollup lewat view `planner_budget_lines`; tanpa penyalinan angka |
| Smart Adjust tanpa kunci pencocok; `lock-deal` menggandakan pos | `item_key` bersama di skenario, pos, vendor; link-or-create; pratinjau perubahan |
| Pagu vendor dicocokkan lewat nama kategori | `category_key → budget_item_keys` + `limit_groups` untuk pos gabungan; `lock-deal` tidak menimpa pos bersama (mis. Sewa Gedung & Dekorasi) |
| Target tabungan tanpa penyimpan; tiga sumber angka pagu | `planner_projects.savings_target` (null = ikut `target_budget`) |
| Klaim integrasi tanpa kolom (task↔vendor, stok souvenir, porsi katering, tugas H-45/H-30/H-10/H-7/H-1) | `planner_tasks.vendor_id`, `purpose='souvenir'`, `pricing_mode='unit'`, 6 tugas Adicara |
| 05 mereferensikan `digital_invitations` yang tidak ada; `project_id NOT NULL` vs `IS NULL`; `url_encode()` bukan fungsi PG | `invitation_id` tanpa FK, `project_id` nullable, tautan personal dihitung saat dibaca, kontrak antarmuka Go |
| Kueri katering menghitung penolak sebagai hadir | Hanya `attending` yang masuk `confirmed_pax`; `maybe`/`unconfirmed` terpisah |
| Dashboard: CTE tunggal tidak mengisi sebagian DTO; `maybe` hilang; vendor dihitung per baris | `SummaryProvider` per modul; empat status RSVP; kategori terkunci |
| Amplop digital via gateway bertentangan dengan prinsip "tidak menampung dana" | Dihapus dari Planner; menunggu spesifikasi Gift |
| SSE/WebSocket | Polling (arsitektur tidak memilikinya) |

### Universal
| Temuan | Perbaikan |
|---|---|
| Matriks "9 kategori" di README 04/05/06 bukan 9 jenis resmi; ikon bertentangan | Registry tunggal (`TEMPLATE_EVENT`) |
| Template hanya pernikahan; `essential_22`/`general_15` tak terdefinisi | Template `general` 15 tugas + tugas khusus 8 jenis; `wedding_essential_22` terdefinisi |
| Offset tetap H-180…H-4 | Penskalaan terhadap waktu persiapan nyata; tugas H-3…H-0 |
| `guest_side DEFAULT 'pria'`, `GroomPax/BridePax`, konstanta scope pernikahan, teks wizard tetap | Sisi/kategori/peran dari registry; DTO generik; wizard dari ketersediaan template |
| Belanja tanpa jumlah/satuan | `quantity`, `unit`, total generated |
| Template mencampur DP terbayar, setoran, vendor, tamu | Template (struktur + estimasi, nol pembayaran) dipisah dari Proyek Contoh |

### Mengikuti arsitektur yang ada
- DDL goose bergaya `002_000_users.sql` (`TEXT`, `UUID` dari aplikasi, `now()`), nomor 005–012.
- Modul Go `planner*` dengan lapisan `delivery/http → usecase → domain ← repository`, pemasangan di `app.go` termasuk `mux.Handle("/v1/planner/", …)`, `BearerAuth`, respons `{"data":…}` dan error `apierr`.
- Frontend: halaman SSR di `/dashboard/**`, data via `backendFetch`, aksi interaktif lewat BFF same-origin `/dashboard/api/planner/[...path]` (browser tidak memanggil Go), tanpa dependensi frontend baru.
- Uang `BIGINT`/`int64` (bukan `float64`); hitung mundur memakai zona waktu proyek (rumus lama selisih 1 hari pada 00:00–06:59 WIB).

### Divalidasi (PostgreSQL 16.2 lokal, DB sementara dihapus)
- DDL migrasi 005–012 dari dokumen BE: terapkan, `Down`, `Up` ulang; 14 tabel + 1 view.
- View `planner_budget_lines` mereproduksi total PDF **74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000** dan angka per sesi.
- Penolakan FK komposit (vendor/kontributor lintas proyek, sesi tak dikenal), `RESTRICT` hapus sesi, rename `scope_key` merambat, kolom generated (durasi tugas/agenda, selisih harga, total belanja).
- Kueri dashboard dan Hub pada bab 5 BE 01; hitung mundur Go dan SQL.
- Simulasi SQL algoritma yang didokumentasikan: Smart Adjust (preset 75 jt ke 14 pos template: 9 diperbarui, 1 dibuat, total 75.000.000; ulang tanpa pos baru; pembayaran tidak berubah), `lock-deal` (tertaut / pos baru / indeks unik vendor), dan rollup belanja. Ini simulasi terhadap DDL, bukan kode Go.
- Pemetaan jenis non-pernikahan: semua kategori vendor terpetakan ke `item_key` pos yang ada, persentase pos = 100, peran dokumen valid, jumlah agenda sesuai README 06.
- Parsing tabel dokumen: jumlah tugas/dokumen/belanja/agenda/lagu, subtotal, dan persentase template.

### Belum dijalankan
Kode Go modul Planner (belum ada), endpoint, frontend, ekspor `.xlsx`/PDF, pengukuran latensi dashboard, pemindai QR pada perangkat nyata.

### Perlu keputusan produk
1. Template delapan jenis non-pernikahan (isi, persentase pos, agenda) adalah usulan Adicara.
2. Penambahan dependensi Go: `excelize` dan satu pustaka PDF.
3. Model *dual-key* tamu (Planner + Undangan) dipertahankan; kontrak antarmuka perlu disepakati saat spesifikasi Undangan Digital ditulis.
4. Pembatasan akses Planner per paket (`entitlement_denied`) menunggu modul billing.
