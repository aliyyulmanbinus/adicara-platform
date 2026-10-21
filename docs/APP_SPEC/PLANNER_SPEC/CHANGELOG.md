# Changelog Spesifikasi Planner

Riwayat revisi folder `docs/APP_SPEC/PLANNER_SPEC`. Hasil validasi hanya dicatat setelah dijalankan.

## 2026-10-21 (keputusan) — `excelize` disetujui

Pemilik produk menyetujui dependensi `github.com/xuri/excelize/v2` v2.11.0 untuk ekspor `.xlsx` Menu 02 (CSV tidak dipilih). Status "menunggu persetujuan/dapat dicabut" diganti "disetujui" di 02 BE (§4.4, bab 8 butir 6 dan penutup bab 8, daftar keputusan di akhir), kontrak bab 1.4, dan catatan implementasi. Pustaka PDF di kontrak bab 1.4 tetap belum disetujui. Entri sebelumnya di berkas ini yang menyebut `excelize` sebagai "perlu keputusan" atau "menunggu persetujuan" adalah catatan sejarah dan tidak diubah. Tidak ada perubahan kode.

## 2026-10-21 (lanjutan) — FE menu 02: dicocokkan dengan backend asli dan dirinci

Permintaan: apakah `02_TIMELINE_DAN_TUGAS/FE.md` sudah sesuai, dan bila belum, perbaiki dan detailkan sebagai acuan developer lain. Backend menu 02 dijalankan terhadap PostgreSQL 16.2 sementara (`planner_fe02_scratch_test`, dihapus setelahnya): 120 panggilan ke semua 17 endpoint menghasilkan 119 respons (821 objek JSON, 33 jenis galat) yang dibandingkan dengan tipe di `FE.md` lewat parser atas blok kode dokumen. **Tipe bab 4.2 sudah benar: nol selisih.** Yang ternyata kurang atau salah ada pada perilaku di sekitar tipe, bukan pada bentuk datanya. Semua ✔ terbukti dengan eksekusi.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Aturan konfirmasi `replace_all` salah.** FE.md menyuruh meminta konfirmasi hanya bila ada kemajuan; server meminta konfirmasi bila proyek punya tugas satu pun (`replace_all` pada 18 tugas "belum" → `400 confirm: required`) | `generatorImpact`/`docsImpact` (kode, diuji: keputusan lokal = status server pada keadaan nyata); 02 FE bab 5, 7.5 |
| ✔ 2 | **Label `general` "Jadwal umum (15 tugas)" salah** di 01 FE bab 8.4 dan 02 FE: seminar dan enam jenis lain mendapat **18** tugas, hanya `other` 15 | "Jadwal standar jenis acara" (tanpa angka, sesuai aturan 01 FE bab 8.4 sendiri); 02 FE bab 13 butir 3 |
| ✔ 3 | **`PATCH` `phase_order` tidak menyalin `phase_name`** (POST menyalin): tugas pindah ke fase 4 membawa nama fase 2 | FE mengirim keduanya (`phaseNameOf`); dicatat sebagai butir 20 di 02 BE bab 8 (perbaikan server opsional) |
| ✔ 4 | **Alasan galat tanpa teks**: `must not be after end_date`, `must keep every shifted task within the project schedule window`, `must be greater than 0`, `cannot be combined with vendor_id`, `must be one of …`, `must be a session scope_key` | `timelineFieldMessage` mencakup semua alasan di `fixtures/planner-errors.json` (49 galat), diuji |
| ✔ 5 | **Angka dan permintaan yang dijawab `invalid_json` tanpa nama kolom**: `nominal_cost` desimal atau string, UUID bukan UUID, body kosong tanpa `{}` | aturan permintaan 02 FE bab 4.1; `timelineApi` selalu mengirim `{}` |
| ✔ 6 | **`sort_order` bukan nomor tampilan** (mulai dari 19 pada proyek yang punya tugas kustom setelah `replace_template`, berlubang setelah hapus) | `taskNumbers` untuk kolom "No" |
| ✔ 7 | **Wireframe berkas** menampilkan baris yang belum dikirim sebagai "✓ Di anggaran" | diperbaiki (nomor 13 `[Kirim]`; keadaan sesudah kirim dijelaskan terpisah) |
| ✔ 8 | **Mekanisme render dan pembaruan sesudah aksi tidak ditetapkan**; fixture hanya tiga baris contoh | satu sumber render + partial (bab 2.4, 7.6), lima berkas fixture asli di `02_TIMELINE_DAN_TUGAS/fixtures/` |
| ○ 9 | Ekspor: tanpa `Cache-Control` dan tanpa `Retry-After` pada `429`; sesi sistem bernama "Umum / Semua Sesi" | BFF menambah `no-store`; chip memakai teks tetap "Umum"; dicatat di 02 FE bab 13 |

Tiga sumber diperiksa sama persis untuk 50 tugas Proyek Contoh (tabel BE §5 → fixture yang dibangun dari tabel → respons HTTP backend). Nilai widget dashboard yang dihasilkan sama dengan fixture 01 FE.

**Ditambahkan ke `FE.md`** (692 → ±1.340 baris): kontrak props, state URL, katalog endpoint, contoh respons asli, katalog galat, `timelineApi.ts`, alur interaksi A–M dengan atribut data, salinan teks, `messages.ts` yang diuji, lingkungan lokal, dan riwayat. Kode di dokumen (`types`, `timeline`, `timelineApi`, `messages`, `props`) diekstrak langsung dari dokumen: `tsc` strict bersih, 19 tes lulus (9 dari 9 mutasi menjatuhkan tes), dan ke-17 fungsi `timelineApi` dijalankan terhadap backend asli.

**Dokumen yang berubah.** `02_TIMELINE_DAN_TUGAS/FE.md` (hampir seluruhnya) dan `fixtures/` (baru, lima berkas); `02_TIMELINE_DAN_TUGAS/BE.md` (bab 8 butir 20); `01_DASHBOARD/FE.md` (label `general` di bab 8.4); `README.md` menu 02 (status).

**Belum dijalankan:** frontend itu sendiri (halaman, partial Astro, dialog, seret-lepas, Gantt tergambar, unduhan) tidak ada; bentuk `vendor_*` pada tugas, `sync-budget` dengan `linked: true`, dan `409 conflict` tidak teramati pada backend yang berjalan; `.xlsx` belum dibuka di aplikasi spreadsheet.

## 2026-10-21 — Menu 02 (BE) diimplementasikan: koreksi spesifikasi dari hasil membangunnya

Backend menu 02 sudah menjadi kode ([dokumen](../../backend/planner/2026-10-21-planner-menu-02.md)): 17 endpoint, provider, applier, dan overlay terpasang di menu 01, ekspor `.xlsx`, dan `excelize` v2.11.0 ditambahkan atas permintaan pemilik. Membangunnya dan mengujinya terhadap PostgreSQL 16.2 menemukan hal berikut; semua ✔ terbukti dengan eksekusi.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **`PATCH` tidak aman tanpa kunci**: ia membaca, menggabung, lalu menulis semua kolom, jadi dua tab yang mengubah kolom berbeda saling menimpa; 02 BE §4.2 menyatakan penulisan satu baris tidak butuh kunci | `PATCH` tugas dan dokumen memegang kunci proyek; status, hapus, dan sinkron biaya tidak (02 BE §4.2, bab 8 butir 15) |
| ✔ 2 | **Kontrak `generate-defaults` kurang**: `preset` kosong dan `mode` kosong tidak ditetapkan, sedangkan wizard memang punya kasus "tugas Adicara saja" tanpa preset dasar | endpoint: `preset` wajib, `mode` bawaan `append`; applier menerima preset kosong untuk pernikahan (02 BE §2, §4.1, bab 8 butir 16) |
| ✔ 3 | **`Avail < 3` memenuhi dua peringatan sekaligus** (`lead_time_too_short` dan `lead_time_compressed`), dan `count` tidak ditetapkan | satu peringatan; `too_short` menggantikan `compressed`; `count` = tugas yang dibuat; tanpa tugas baru tanpa peringatan (02 BE §4.1) |
| ✔ 4 | **`subject_role` kosong = `bersama` tidak berlaku untuk delapan jenis lain**: subjek mereka `panitia`, `lembaga`, `umum` | bawaan `bersama` (pernikahan) atau `umum` (02 BE §2, bab 8 butir 18) |
| ✔ 5 | **Sinkron KUA di dua kasus tepi**: sesi `akad` bisa sudah dihapus pengguna, dan pos tunggu `biaya_kua` bisa sudah dipakai berkas lain; 02 BE §4.5 tidak menyebut kasusnya | sesi hilang → `all`; pos yang sudah bertaut tidak diambil alih, berkas KUA mendapat pos baru di `akad` (tes `TestSyncBudget` kasus G) |
| ✔ 6 | **Batas jumlah, `max(sort_order)+1`, dan "daftar lengkap" berbalapan** (tanpa kunci: 52 tugas dari batas 51, dua pos untuk satu berkas) sudah terbukti pada audit spesifikasi; kini terbukti pada kode | enam `POST /tasks` serentak pada 499 tugas → tepat satu `201`; delapan `sync-budget` serentak → satu pos |
| ○ 7 | Pembatas laju per pengguna tidak diekspor dari paket `planner`; Menu 02 menjadi salinan ketiga | disalin; ekstraksi dicatat di 02 BE bab 8 butir 13 |

Terbukti dengan kode: data template (44 + 6 + 15 + 15 umum + 21 + 22) sama dengan tabel spesifikasi (`TestTemplatesMatchSpec` membaca tabelnya); uji emas Proyek Contoh untuk baris **Timeline** dan **Dokumen** kontrak bab 9 (44 tugas PDF, 15 berkas 9/6, tiga tugas berdurasi 2 hari, Lamaran 15 Okt 2026, widget Timeline `50/15/29/0` dan Legalitas `9/15`); wizard atomik dengan applier sungguhan; isolasi di 17 endpoint; ekspor `.xlsx` dibaca kembali.

**Dokumen yang berubah.** `02_TIMELINE_DAN_TUGAS/BE.md` (status, bab 6 dengan nama tes yang sungguh ada dan hasil, bab 7 selesai, bab 8 butir 15–19); kontrak bab 9–10 (status uji); `README.md` dan `FE.md` menu ini (status: backend ada, frontend belum).

**Status implementasi:** menu 01 dan 02 backend. Frontend dan menu 03–06 masih spesifikasi; widget keuangan, vendor, tamu, dan rundown serta template-nya masih kosong. Uji emas baris Anggaran, Tabungan, Skenario, Seserahan, Vendor, Tamu, Rundown, dan Playlist belum bisa dijalankan.

**Belum dijalankan:** image Docker, PostgreSQL 18, Nginx di depan rute baru, beberapa replika API, `.xlsx` di Excel/Numbers/LibreOffice, dan tipe TypeScript `FE.md` terhadap respons sungguhan.

## 2026-10-21 (lanjutan) — BE menu 02: celah yang tersisa ditutup dengan eksekusi

Permintaan: perbaiki kekurangan audit sebelumnya dan fokus ke `02_TIMELINE_DAN_TUGAS/BE.md`. Butir yang tadinya hanya penalaran (○) dijalankan, dan yang belum pernah dijalankan (ekspor, balapan) dibuat dan diperiksa. Database sementara baru (`planner_spec02_scratch_test`, migrasi 001–012, 50 tugas dan 16 berkas) dihapus setelahnya; prototipe ekspor ada di modul Go sementara di luar repo.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 17 | **Satu tugas template melanggar horizon**: Seminar "Undang dan konfirmasi pembicara" H-70…H-56 di atas `R = 60`; penskalaan menaruhnya sebelum awal jadwal sehingga selalu dijepit dan durasinya menyusut. Skrip atas seluruh template (44 + 6 pernikahan, 15 umum, 21 khusus): hanya ini yang melanggar; tidak ada kunci `template_key` ganda (maks. 58 karakter) | TEMPLATE_EVENT: H-58…H-50 (setelah panitia inti, selesai tepat sebelum registrasi H-50); aturan "offset di `0…R`" ditambahkan di bab 1; `TestEveryTemplateOffsetIsWithinItsHorizon` |
| ✔ 18 | **Awalan `'` yang diwajibkan kontrak tampak di sel**: `excelize` v2.9.0 dan v2.11.0 menyimpan teks berawalan `=` sebagai string bersama (`t="s"`, tanpa `<f>`; dibaca ulang dengan parser XML Python), jadi sel tidak pernah menjadi formula, dan `'` ditulis sebagai karakter biasa (tidak ada `quotePrefix` pada `Style`) | kontrak bab 2 butir 5 dan 02 BE §4.4 mencatat perilaku; keputusan (pertahankan atau cabut aturan) di 02 BE bab 8 butir 12 |
| ✔ 19 | **`notes` boleh baris baru, tetapi `cleanText` Menu 01 menolak semua karakter kontrol** (`unicode.IsControl('\n')` benar) | pembantu baru `cleanNote` (02 BE §3.4) |
| ✔ 20 | **Klik ganda sinkron biaya**: dua sesi berbalapan. Dengan `FOR UPDATE` pada baris dokumen: 1 pos; tanpa kunci (kontrol): 2 pos | kunci menjadi bagian kontrak (02 BE §4.5) |
| ✔ 21 | **Batas 500 tugas bocor tanpa kunci**: dua `POST /tasks` serentak pada batas 51 dengan 50 tugas: dengan `LockProject` 51 tugas (satu `task_limit_reached`), tanpa kunci 52. Generator dan `apply-template` juga bisa melewati batas dalam satu panggilan | semua penulisan massal mengambil `LockProject`; generator memeriksa jumlah **setelah** penyisipan, semua-atau-tidak (02 BE §4.1, §4.2, §3.4) |
| ✔ 22 | **Urut ulang tak lengkap**: pernyataan §4.2 persis. Daftar lengkap → 50 nilai 1…50; kurang satu, ganda, atau berisi id tugas proyek lain → 49 baris berubah ≠ 50 dan tugas proyek lain tidak tersentuh | tabel hasil di 02 BE §4.2 |
| ✔ 23 | **Urutan `sort_order` template non-pernikahan tidak ditentukan**: tiga tugas khusus akan menumpuk di akhir meski tanggalnya di tengah | kronologis (offset mulai menurun, seri menurut urutan template); diperiksa untuk tujuh jenis: urutan 15 tugas umum tetap, fase tak pernah turun (02 BE §5.5) |
| ✔ 24 | **Ekspor hanya satu paragraf**: kolom, format sel, aturan Gantt tidak ditentukan | tabel kolom, aturan Gantt mingguan yang sama dengan `buildGantt` FE, `safeCell`, nama berkas; prototipe dibangun dan dibaca kembali: 51 baris, 4 judul berbahaya jadi teks, tanggal sungguhan, 50 × 26 sel Gantt cocok dengan perhitungan hari demi hari (nol selisih, 61 sel terisi) |
| ✔ 25 | **Angka batas tanpa dasar**: pada 500 tugas (catatan 200 karakter) respons 370 kB, pada 200 dokumen 114 kB; ringkasan 0,2 ms dan daftar 0,15 ms (`EXPLAIN ANALYZE`) | angka menjadi batas tampilan, bukan kinerja (02 BE §3.4, bab 8 butir 3) |
| ✔ 26 | **Syarat `confirm` belum berupa kueri**: tugas ber-`template_key` dengan `status <> 'belum'` atau `vendor_id`: **44** pada Proyek Contoh, **0** bila semua `belum`; dokumen: `finished 9, linked 1` dengan berkas KUA tertaut 600.000; setelah hapus template pos tetap ada dengan angka 600.000 dan tanpa tautan | kueri ditulis di 02 BE §4.1 dan §4.7 |
| ○ 27 | **Pembatas laju per pengguna tidak diekspor** dari paket `planner`; Menu 02 menjadi salinan ketiga | pilihan bawaan menyalin; ekstraksi dicatat di 02 BE bab 8 butir 13 |

**Dokumen yang berubah.** `02_TIMELINE_DAN_TUGAS/BE.md` (status, §3.4, §4.1, §4.2, §4.4 ditulis ulang, §4.5, §4.7, §5.5, §6, §7, bab 8 butir 3, 6, 12–14), `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` (offset Seminar, aturan offset di bab 1), `00_KONTRAK_BERSAMA/README.md` (bab 2 butir 5 dan bab 10). FE menu 02 tidak diubah.

**Belum dijalankan:** modul `plannertimeline` di repo (usecase, handler, pemetaan galat), isolasi antar pengguna pada endpoint, uji emas Proyek Contoh, pembatas laju ekspor, dan berkas `.xlsx` **belum dibuka di Excel, Numbers, atau LibreOffice** (tampilan akhir awalan `'` disimpulkan dari isi berkas). Dependensi `excelize` di repo menunggu persetujuan; hanya dipakai di modul percobaan.

**Perlu keputusan produk:** 02 BE bab 8 butir 6 (dependensi `excelize`) dan butir 8 (isi template dan teks panduan non-pernikahan); butir 12–14 baru (awalan `'`, salinan `userLimiter`, offset Seminar).

## 2026-10-21 — Audit menu 02 Timeline & Tugas: disamakan dengan pola menu 01

`02_TIMELINE_DAN_TUGAS/{README,BE,FE}.md` dibandingkan dengan pola yang dihasilkan audit menu 01 (BE disamakan dengan kode, FE sebagai acuan bangun, README dengan status), dengan kode menu 01 yang sudah jadi (`planner/domain/repository.go`, `summary.go`, `eventtype.go`, migrasi 007), kontrak, PDF, dan dokumen menu lain. ✔ = terbukti dengan eksekusi; ○ = dari pembacaan atau penalaran.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Tanggal sesi Lamaran Proyek Contoh bertentangan antar dokumen.** 01 FE menulis 10 Okt 2026 (Hari-H − 169 tanpa penskalaan), 03 FE menulis 15 Okt. Algoritma generator mengalikan offset dengan `175/180` untuk contoh yang dibuat 4 Okt 2026 (kode Go acuan lulus tes): tugas `wedding.lamaran` jatuh pada **15 Okt 2026** | 01 FE (fixture, wireframe, teks) memakai 15 Okt; 02 BE §4.6 mencatatnya; butir 10 di 01 FE bab 15 ditutup |
| ✔ 2 | **Sinkron biaya gagal untuk berkas bernama > 150 karakter**: `planner_admin_docs.document_name` boleh 255, `planner_budget_items.item_name` dibatasi 150 (`23514`) | `left(nama, 150)` pada pos baru (02 BE §4.5) |
| ✔ 3 | **Galat FK tugas datang pada waktu berbeda**: vendor lintas proyek `23503` pada pernyataan (`planner_tasks_vendor_id_project_id_fkey`), sesi tak dikenal baru saat `COMMIT` (FK ditunda); `end_date < start_date` memicu CHECK tak bernama `planner_tasks_check` | usecase memeriksa vendor, sesi, dan tanggal lebih dulu dan memetakan galatnya (02 BE bab 1, §3.4) |
| ✔ 4 | **Durasi PDF hanya terjaga tanpa pemampatan**: pada `Avail` 175 dan 180 semua 44 durasi sama dengan PDF; pada `Avail` 120 sampai 10 tiga tugas 2 hari (3, 4, 12) menjadi 1 hari | dicatat di 02 BE §4.1 |
| ✔ 5 | **Status Proyek Contoh**: PDF Hal. 3 memuat 15 *Selesai* dan 29 *Proses* (tidak ada *Belum*), jadi widget Timeline contoh `in_progress = 29`, bukan 2 seperti fixture 01 FE | 02 BE §4.6 menetapkan overlay; fixture dan teks 01 FE disamakan |
| ✔ 6 | **Sinkron biaya tidak aman terhadap klik ganda**: `planner_budget_items` tidak punya indeks unik pada `admin_doc_id`, jadi dua permintaan bisa membuat dua pos | baris dokumen dikunci `FOR UPDATE` dalam transaksi (02 BE §4.5; kasus berbalapan dijalankan pada entri lanjutan di atas: 1 pos dengan kunci, 2 tanpa) |
| ○ 7 | **Batas modul tidak tercatat**: sinkron biaya (Menu 02) menulis ke tabel milik Menu 03 | dicatat sebagai pengecualian SQL kedua di kontrak bab 1.1 dan bab 8 (alternatif antarmuka Go disebut di 02 BE bab 8) |
| ○ 8 | **`GET /tasks?scope=all` ambigu**: "semua sesi" atau "tugas Umum" (`all` juga `scope_key` sesi sistem) | `scope=all` hanya tugas Umum; tanpa parameter = semua |
| ○ 9 | **BE tidak menjelaskan sambungan ke kode menu 01**: satu modul dengan dua kunci (`timeline`, `admin_docs`) padahal tiap antarmuka punya satu `Key()`; `ApplyMode` hanya `append`; bidang registry yang harus ditambah; urutan overlay | 02 BE §4.6 dan §2 |
| ○ 10 | **DTO tidak mengikuti pola menu 01**: tag `binding:"required"` (menjawab `invalid_json` tanpa nama kolom), sintaks Go tak valid (`Title, SubEventScope … *string`), `Created, Skipped int` tanpa tag JSON, tipe `Warning` tak terdefinisi, bentuk respons per endpoint tidak ada | DTO ditulis ulang tanpa `binding`, `domain.Notice`, bab 3.3 bentuk respons |
| ○ 11 | **`admin_docs.label` tidak didefinisikan** (01 FE bab 15 butir 11) dan **teks panduan jenis non-pernikahan dirujuk ke TEMPLATE_EVENT yang tidak memuatnya** (dicatat juga di dokumen implementasi menu 01); filter subjek wireframe 4 chip padahal registry 5 subjek (`ortu_cpp`, `ortu_cpw` terpisah) | `AdminLabel`, `AdminSubjects`, `AdminGuide` + teks panduan di TEMPLATE_EVENT bab 4.1; README/FE memakai 6 chip |
| ○ 12 | **FE bertentangan dengan 01 FE**: `AppLayout` (seharusnya `PlannerLayout`), nama preset "dari registry" (registry hanya membawa kunci; 01 FE sudah memutuskan kamus `TEMPLATE_LABELS`), pemuatan data tanpa `GET projects/:id` (status arsip, zona waktu), tombol "Kirim ke anggaran" tak konsisten di wireframe, contoh "Batas DP Venue" bukan tugas PDF, ekspor `.xlsx` butuh `backendFetchRaw` yang tak ada di rencana BFF | FE ditulis ulang; README memakai "Visit Venue Pernikahan" |
| ○ 13 | **`replace_template` menghapus kemajuan dan tautan tanpa konfirmasi** (hanya `replace_all` yang meminta) | `confirm` wajib bila ada kemajuan (02 BE §4.1, §4.7) |
| ○ 14 | **Tanpa batas jumlah dan jendela tanggal** sementara daftar tanpa paginasi dan Gantt menggambar rentang tanggal | 500 tugas, 200 berkas, jendela `Hari-H − 1095` s/d `akhir acara + 365` (angka usulan); kode `task_limit_reached`, `admin_doc_limit_reached` di kontrak (ukuran pada batas diukur di entri lanjutan, #25) |
| ○ 15 | Kunci tugas 43 (`wedding.final_fitting`) tidak mengikuti aturan "judul huruf kecil bergaris-bawah"; kunci dokumen juga dikurasi | aturan dijelaskan (kurung dibuang; kunci di tabel adalah konstanta) |
| ○ 16 | Nama kunci ringkasan `selesai/proses/belum` berbeda dari `TimelineSummary` (`completed/in_progress`) | `completed/in_progress/not_started/overdue` |

Tidak berubah karena **sudah benar**: DDL migrasi 007 identik dengan blok SQL 02 BE (perbandingan teks); 44 nama, urutan, durasi, dan status serta 15 nama berkas dan statusnya (9/6) dan 4 langkah panduan cocok dengan PDF (nama tugas 43 hanya berbeda karakter privat pada font PDF); tanggal pada wireframe Tab Jadwal cocok dengan generator.

**Dokumen yang berubah.** `02_TIMELINE_DAN_TUGAS/BE.md` (status, DTO, bentuk respons, validasi, SQL sinkron, generator dengan kode acuan, sambungan ke menu 01, template non-pernikahan, tabel pengujian dengan status, urutan kerja, celah); `FE.md` ditulis ulang mengikuti 01 FE (routing, alur data, prasyarat, tipe dan fixture, library yang dijalankan, wireframe, galat, aksesibilitas, tes, QA, urutan kerja, celah); `README.md` (status, tabel 9 jenis, integrasi); `00_KONTRAK_BERSAMA/README.md` (pengecualian SQL, bidang registry `AdminLabel`, kode galat dan warning baru, validasi); `TEMPLATE_EVENT.md` bab 4.1 (subjek dokumen, label widget, panduan perizinan usulan); `01_DASHBOARD/FE.md` (butir 10 dan 11, fixture, wireframe) dan `01_DASHBOARD/BE.md` §5.2 (catatan waktu sebagai parameter).

**Divalidasi** (PostgreSQL 16.2 lokal, database sementara `planner_spec02_scratch_test`, sudah dihapus; database `adicara` tidak disentuh): migrasi 001–012 naik; 50 tugas dan 15 berkas Proyek Contoh dimasukkan dari tabel 02 BE; kueri provider dan ringkasan (`total 50, completed 15, in_progress 29, not_started 6, overdue 0` pada 4 Okt 2026; `overdue 1` pada 4 Nov; `urgent` tugas 16–18 pada 1 Nov), `admin_docs` (9/15 dan lima nama pertama), urut ulang (50 tugas, 50 nilai berbeda), geser (35 tugas), `append` idempoten, dan blok SQL sinkron biaya §4.5 dijalankan **persis seperti tertulis** per pernyataan, ditambah delapan kasus perilaku. Kode Go acuan `Plan`: lima tes lulus. Kode TypeScript di 02 FE (tipe, `timeline.ts`) diekstrak dari dokumen: `tsc` strict dengan `noUncheckedIndexedAccess` bersih dan 10 tes `node:test` lulus terhadap fixture 50 tugas. Nama, durasi, status, dan panduan dibandingkan dengan PDF memakai `pdfminer`.

**Belum dijalankan:** modul Go `plannertimeline`, endpoint, uji emas Proyek Contoh, apa pun di browser (Astro, dialog, seret-lepas, Gantt tergambar), dan perbandingan tipe FE dengan respons backend asli (backend belum ada).

**Perlu keputusan produk:** 02 BE bab 8 (label preset, judul widget Legalitas, batas dan jendela tanggal, `confirm` pada `replace_template`, batas modul sinkron biaya, dependensi `excelize`, status "proses" untuk 29 tugas contoh, panduan perizinan usulan, sinkron satu arah, ringkasan tidak per sesi) dan 02 FE bab 13.

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
