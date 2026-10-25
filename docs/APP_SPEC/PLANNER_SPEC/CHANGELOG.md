# Changelog Spesifikasi Planner

Riwayat revisi folder `docs/APP_SPEC/PLANNER_SPEC`. Hasil validasi hanya dicatat setelah dijalankan.

## 2026-10-25 (audit kejelasan) — README, FE.md, dan BE.md menu 05 dicek terhadap kode, jenis acara lain, dan menu 01–04

Permintaan: apakah README, FE.md, dan BE.md (dengan implementasinya) menu 05 sudah jelas, universal untuk jenis acara lain, dan saling terintegrasi. Hasil: **ya, dengan tiga koreksi kecil dan satu tes baru.** Yang dijalankan: seluruh suite backend pada database sementara (448 tes tingkat atas lulus, 0 gagal, 0 di-skip; `gofmt` dan `go vet` bersih; database dihapus setelahnya), 196 tautan dan anchor di tiga dokumen menu 05 (0 putus), perbandingan 9 blok Go di BE.md dengan berkas di repo, pencarian literal pernikahan di kode produksi (hanya `guestcatalog.go`, registry, dan `sample.go`, data Proyek Contoh yang memang pernikahan), kunci tugas Menu 02 (`wedding.adicara_*`) dengan templat di kode, dan nama kolom `recommended_portions` dengan Menu 03.

Koreksi: (1) README prinsip 3 menyebut "tanpa SQL antar modul" padahal stok souvenir dibaca dengan `SELECT` baca-saja atas `planner_shopping_items`; kalimatnya kini menyebut itu satu-satunya akses lintas tabel. (2) Kepala BE.md mengatakan penyesuaian kode "tercatat di butir 29–38", padahal butir itu hanya memuat perubahan perilaku; selisih bentuk kode (alias ke `planner/domain`, `TemplateInput` di paket `domain`, `SideLabel(*string)`, `ReceptionFilters`) kini disebut apa adanya. (3) Butir 29 menunjuk kalimat bab 2.1 yang sudah tidak ada. Tes baru `TestSouvenirStockAgreesWithMenu04`: Menu 04 menjumlah stok di Go (perseratus), Menu 05 di SQL (`numeric`); dua implementasi satu aturan itu kini dibandingkan pada pecahan yang jatuh tepat di bawah dan di atas bilangan bulat (hasilnya sama; tidak ada bug). Total tes 447 menjadi 448 (114 baru).

Yang diperiksa dan tidak berubah: 9 jenis acara menjalani alur yang sama lewat `TestEveryEventTypeRunsTheWholeGuestFlow`; label sisi dan kategori di TEMPLATE_EVENT 4.2 dibaca tes dan sama dengan registry; fixture FE dan katalog galat FE sama dengan respons backend; Hub (`guest_pax_count` 43), `usage.guests` per sesi, dan pemindahan tamu saat sesi dihapus dijalankan lewat Menu 01 yang asli. Yang tetap belum ada: FE dan BFF, modul Undangan Digital (`personalized_url` belum pernah muncul), perubahan Nginx `camera=(self)`, dan `.xlsx` yang dibuka di Excel/Numbers.

## 2026-10-25 (lanjutan) — BE menu 05 diimplementasikan: koreksi spesifikasi dari hasil membangunnya

Permintaan: apakah BE.md menu 05 sudah siap diimplementasikan, dan bila sudah, implementasikan sebagai kode mengikuti arsitektur yang ada. Hasil: **siap**; blok kode di BE.md (domain, impor, templat) sudah dikompilasi dan diuji pada audit, dan DDL 011 identik dengan migrasi, sehingga yang tersisa adalah repository, usecase, delivery, pemasangan, kontrak OpenAPI, dan tes. Tidak ada celah yang membatalkan rancangan; yang ditemukan saat membangun kecil dan tercatat di bawah. Modul `plannerguest` kini ada ([dokumen](../../backend/planner/2026-10-25-planner-menu-05.md)): 19 endpoint, provider `guests`, overlay `guests`, registry `GuestSides`/`GuestCategories`, `RSVPWriter()` untuk modul undangan, ekspor/templat `.xlsx`, dan 19 operasi + 24 skema di `openapi.yaml`. Dijalankan terhadap PostgreSQL 16.2 (database sementara `planner_guest_impl_test`, dihapus setelahnya): **448 tes tingkat atas lulus** (334 sebelumnya + 114 baru), `-race`, cakupan 91,9%, `redocly lint` valid, 33 mutasi kode produksi (kunci proyek dan batas, SQL antrean dan KPI, cap RSVP, stok souvenir, urutan daftar, pembatas laju, batas badan impor, awalan rumus, zona jam masuk, dan lainnya) semuanya digagalkan tes; dua awalnya selamat (batas badan impor dan zona jam masuk, mesin uji kebetulan berzona WIB) dan menambah dua tes, empat tidak terkompilasi dan ditulis ulang.

| # | Temuan saat membangun | Koreksi |
|---|---|---|
| ✔ 1 | **Tes jalur dokumen yang salah lulus tanpa menguji apa pun:** dua tes yang membaca dokumen (fixture FE, katalog galat) memakai jalur relatif satu tingkat terlalu dangkal, sehingga di-skip dan dilaporkan `ok` | jalur dibetulkan (enam tingkat dari `delivery/http`); kedua tes diperiksa dengan `-v` (tidak ada `SKIP`) dan diuji dapat gagal dengan mengubah satu nilai fixture dan satu pesan katalog |
| ✔ 2 | **Bentrok token tidak dapat diulang di dalam transaksi** (bab 2.1: "dicoba ulang maksimal 3 kali"): pernyataan yang gagal membatalkan transaksi PostgreSQL | `409 conflict`; mengulang permintaan memberi token baru (butir 29) |
| ✔ 3 | **`ApplyRSVP` harus idempoten tanpa perubahan, tetapi bab 2.2 menyuruh `rsvp_submitted_at = now()` pada setiap status** | jawaban yang sama membiarkan cap lama; satu pernyataan SQL untuk `PATCH …/rsvp` dan `ApplyRSVP` (butir 30) |
| ✔ 4 | **Badan JSON di atas 64 KiB dijawab `413 file_too_large` di semua rute** pada versi pertama modul (ditangkap oleh `TestGuestEndpointsRejectMalformedBodies`); spesifikasi: 413 hanya untuk impor | `400 invalid_json` kecuali impor (butir 33) |
| ✔ 5 | **Contoh pesan katalog galat FE bab 4.6 berbeda dari yang diucapkan backend** (`already_checked_in`, `guest_limit_reached`, `file_too_large`); tabel katalog tidak pernah dijalankan | backend mengikuti contoh; `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays` membaca tabel dan memaksa tiap pesan dihasilkan (butir 34) |
| ✔ 6 | **Fixture FE dibangun dari fungsi acuan, bukan dari respons HTTP** | respons asli daftar, proyeksi, pratinjau impor, dan paket meja resepsionis sama dengan keempat fixture (nol selisih setelah id, tiket, dan waktu dinormalkan) |
| ✔ 7 | **Paket meja: fungsi acuan fixture hanya mendaftar kategori registry di `filters`**, padahal FE menyaring "kategori kustom hanya bila ada" | `filters.categories` memuat registry (juga yang 0) lalu `lainnya` dan teks kustom yang dipakai tamu yang diharapkan (butir 31) |
| ✔ 8 | **`RSVPStatus` harus berada di `planner/domain`** supaya `GuestRSVPWriter` dapat menamainya tanpa siklus impor; `ErrNotFound` sudah ada di sana | `planner/domain/invitation.go`; tipe Menu 01 (`Option`, `GuestSide`, `RSVPSummary`) dipakai modul lewat alias (BE bab 5.1) |
| ✔ 9 | **Pembatas laju dihitung sebelum handler**, jadi tes impor (5/menit) dan unduhan (10/menit) yang memanggil puluhan kali tertahan `429` | `Limits` / `RegisterRoutesWithLimits`; `RegisterRoutes` memakai `DefaultLimits` (butir 32) |
| ○ 10 | **Pratinjau impor (`dry_run`) tidak memeriksa batas 5.000** dan tidak mengunci proyek | dicatat (butir 36); hanya impor sungguhan yang menjawab `409` |
| ○ 11 | **Urutan nama antrean mengikuti kolasi basis data**; diuji pada `en_US.UTF-8`, belum pada image `postgres:18-alpine` | dicatat (butir 37) |
| ○ 12 | **`PUBLIC_BASE_URL` belum dibaca kode apa pun** dan modul undangan belum ada, sehingga `personalized_url` tidak pernah dikirim | `Options.Invitations`/`PublicBaseURL` kosong di `app.go`; jalur tautan diuji dengan pencarian tiruan (butir 38) |

**Dokumen yang berubah.** `05_BUKU_TAMU/BE.md` (status, bab 2.1, 2.2, 2.5, 5.1, 6, 8.1–8.4, 9, 10 butir 29–38), `FE.md` (status; bab 4.2–4.5, 11.5 langkah 6, 13 butir 1, 14), `README.md` (status), `00_KONTRAK_BERSAMA/README.md` (status, bab 9, bab 10), `README.md` folder (status). Di luar PLANNER_SPEC: `backend/docs/openapi.yaml`, `docs/backend/planner/2026-10-25-planner-menu-05.md` (baru), `docs/backend/README.md` dan `CHANGELOG.md`, `docs/api.md`, `docs/architecture.md`, `docs/database.md`, `backend/README.md`, `README.md`.

**Terbukti pada kode, bukan hanya pada SQL audit:** Proyek Contoh lewat seluruh tumpukan (20 tamu / 43 pax, pihak 10/22 dan 10/21, terkirim 14, RSVP 12/2/2/4 dan 27 pax, enam kategori dan tabel per pihak, proyeksi 27 / 12 / 4 → 34, `guest_pax_count` 43, `usage.guests` 16 / 4 / 0); ilustrasi hari-H (11 dari 18 diharapkan, 22 dari 39 pax, 11 souvenir; penolak yang hadir menjadi 12/19 dengan "belum" tetap 7); delapan check-in serentak → satu menang; 12 tambah serentak pada 4.995 tamu → lima berhasil; dua impor serentak pada 4.990 tamu → satu 409; limit 5.000 tepat; CSV Excel dan `.xlsx` (sel telepon numerik ditolak); templat dibaca balik; ekspor dibaca ulang; alur penuh untuk sembilan jenis acara; isolasi pengguna di 19 rute; 19 rute terdaftar = 19 operasi dan 24 skema = tag JSON struct.

**Belum dijalankan:** frontend dan BFF (belum ada; batas badan 7,5 MiB untuk `…/guests/bulk-import` ikut saat BFF dibuat); modul Undangan Digital; `.xlsx` di Excel/Numbers/LibreOffice; kinerja pada 5.000 tamu; image Docker, Nginx (termasuk `Permissions-Policy: camera=()` yang memblokir pemindai QR), PostgreSQL 18, beberapa replika.

**Keputusan produk yang menunggu** (pilihan bawaan sudah dipakai): BE bab 10 butir 2 (satuan souvenir), 3 (`pax_actual` satu kolom), 7 (token pada tautan personal sebagai kredensial), 8 (pustaka QR), 29–38, serta butir 1–28 yang sudah ada.

## 2026-10-25 — Spesifikasi menu 05 (Buku Tamu) ditulis ulang mengikuti pola 01–04, setelah diaudit dengan menjalankannya

Permintaan: apakah BE.md, FE.md, dan README menu 05 sudah sesuai, mengikuti pola menu 01–04. Hasil: **belum** (sama seperti menu 03 dan 04): DDL migrasi 011 identik dengan BE.md dan angka contoh (20 tamu, 43 pax, RSVP 12/2/2/4) benar, tetapi dokumen berukuran 327 / 125 / 71 baris tanpa daftar isi, status dan audit, kontrak data FE, bentuk respons, validasi, sambungan ke Menu 01, urutan pengerjaan, atau celah, dan memuat janji yang gagal bila dijalankan. Diaudit dengan **menjalankan**: PostgreSQL 16.2 sementara dengan 12 migrasi nyata, backend Menu 01–04 yang sebenarnya (Proyek Contoh dan sembilan jenis acara lewat HTTP), PDF Hal. 13 dibaca ulang dengan `pdfminer`, Gin 1.12 untuk ke-19 rute, `excelize` v2.11.0 untuk impor/templat, modul acuan Go dan TypeScript yang dikompilasi bersama kode 01–04 yang diekstrak dari dokumennya. Hasil: BE.md 1.858 baris (19 endpoint), FE.md 1.447 baris (14 bab), README 117 baris, lima fixture baru.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Ekspor `xlsx\|pdf`** untuk daftar tamu dan rekap kehadiran, padahal satu pustaka PDF belum disetujui ([kontrak bab 1.4](00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)) dan Menu 02–04 hanya `.xlsx` | hanya `.xlsx` |
| ✔ 2 | **Impor `multipart/form-data` 5 MB tidak dapat melewati BFF**: non-GET wajib `application/json` dan badan dibatasi 64 KiB; `checkOrigin` Astro memeriksa POST berbentuk form di balik proxy TLS (01 FE bab 3 dan 6) | badan JSON `{filename, content_base64}` (≈ 6,7 MiB), batas badan BFF 7,5 MiB hanya untuk `…/guests/bulk-import` |
| ✔ 3 | **Tautan personal hanya `?to=<nama>`** padahal `GuestRSVPWriter.ApplyRSVP(invitationID, guestID, …)` butuh id tamu yang tidak dimiliki halaman publik; mencocokkan nama rapuh dan dapat dipalsukan | `?g=<token 128 bit>&to=<nama>`; `ApplyRSVP` berbasis token dan terikat ke undangan (`PersonalURL`, diuji) |
| ✔ 4 | **FK undangan `ON DELETE SET NULL` (§5.3) membuat hapus undangan gagal** bila ia masih punya tamu-saja-undangan: `invitation_id` menjadi `NULL` pada baris tanpa `project_id` dan `CHECK` menolaknya; kasus "undangan dulu" juga gagal pada `COMMIT` bila `sub_event_scope` tamu tidak ada di proyek (FK komposit ditunda) | modul undangan menghapus tamu-saja-undangan lebih dulu (dijalankan); `CASCADE` ditolak (menghapus tamu milik proyek); kasus B mereset sesi ke `all` |
| ✔ 5 | **`ceil` `float64` menambah porsi**: `200 × 1,1` = `220,00000000000003` → 221; pada tamu 0–3.000 × persen 0–100 berbeda dari jawaban eksak pada **663 dari 303.101** kombinasi | `buffer_percent` bilangan bulat 0–100 dan `RecommendedPortions` dengan aritmetika bulat (tes: tidak pernah berbeda dari ceil eksak) |
| ✔ 6 | **"Belum hadir" = total tanpa penolak − sudah hadir salah bila tamu yang menolak tetap datang** (12 hadir dengan satu penolak: 6, seharusnya 7; kehadiran bisa melewati 100%) | `expected_guests` = bukan penolak ∨ hadir; `pending = expected − checked_in`; check-in tidak mengubah RSVP; penolak yang belum datang hanya muncul bila dicari |
| ✔ 7 | `ReceptionOverview.AttendancePercent` melanggar kontrak bab 6 (tanpa `*_pct` di respons) | dibuang; klien memakai `percent` |
| ✔ 8 | **"Unduh templat .xlsx" di FE tanpa endpoint** | `GET …/guests/import-template`: judul tebal, tanpa baris contoh, dropdown pihak/kategori/sesi dari lembar Petunjuk, kolom telepon bertipe teks; dibaca balik pembaca impor |
| ✔ 9 | **CSV dari Excel Indonesia (BOM UTF-8 + titik koma) terbaca satu kolom** oleh `encoding/csv`; CSV Windows bukan UTF-8; Excel membuang `0` di depan nomor angka (`81200000101`) | `DecodeCSV` (BOM, deteksi pemisah, Windows-1252); nomor angka ditolak (tidak ditebak, sama dengan `NormalizeWA`) |
| ✔ 10 | **`.xlsx` adalah zip**: unggahan ≈ 1 MiB yang mengklaim 300 MiB XML diekstrak ke direktori sementara (bawaan `excelize` 16 GB) | `UnzipSizeLimit` 64 MiB (ditolak seketika), `defer Close`, berhenti membaca setelah 2.001 baris |
| ✔ 11 | **PDF Hal. 13 punya enam penghitung di setiap tabel pihak**; spesifikasi hanya satu set gabungan sehingga memilih pihak tidak mengubah hitungan | `side_categories` pada `GuestOverview`; pil kategori mengikuti pihak (diuji: 3/2/2/1/1/1 dan 3/1/2/1/2/1) |
| ✔ 12 | **Pesan WhatsApp dibentuk server** (`whatsapp_url` per tamu: ≈ 600 byte × 100 tamu, nama hari dan bulan Indonesia di Go), berbeda dari Menu 04 (klien); pada Standalone `whatsapp_url = null` dengan "tautan eksternal pengguna" tanpa kolom penyimpan, jadi mode utama yang bisa dibangun tidak punya WhatsApp | klien membentuk pesan dari `formatted_wa_number` (`guestWaLink`); Standalone mengirim pesan tanpa tautan ("balas pesan ini") |
| ✔ 13 | `GuestOverview.Sides[].Key string` bertentangan dengan `GuestSide.Key *string` Menu 01 yang sudah ada di kode (pihak belum ditentukan = `null`) | memakai tipe Menu 01 apa adanya |
| ✔ 14 | `reception-overview` memuat **seluruh antrean** setiap 5 detik (sampai ribuan baris) | satu paket: KPI, hitungan filter, 10 check-in terbaru, satu halaman antrean (50, maks 200) |
| ✔ 15 | Check-in: `reception_notes` tanpa kolom; `pax_actual` tanpa batas atas (DDL menerima 500) sedangkan `pax_count` ≤ 20 → walk-in 21 pax = `23514` (500); batal check-in tidak menyebut apa yang direset | `reception_notes` dibuang; pax 1–20 di usecase; batal mereset `is_checked_in`, `checked_in_at`, `souvenir_received` (dijalankan) |
| ✔ 16 | `409 already_checked_in` "beserta jam check-in sebelumnya", padahal galat hanya `{code, message}` | klien mengambil ulang `GET …/guests/:id` |
| ✔ 17 | Tidak ada batas jumlah tamu dan kode galatnya | 5.000 tamu di bawah kunci proyek, `409 guest_limit_reached` (kode baru di kontrak bab 6) |
| ✔ 18 | Pencarian teks tanpa aturan: `ILIKE '%'||q||'%'` mencocokkan semua tamu untuk `q = %` | `strpos(lower(…))`, tanpa wildcard (dijalankan: 0 vs 20 baris) |
| ✔ 19 | `TEMPLATE_EVENT` hanya memuat **kunci** sisi dan kategori tamu; label (mis. "Teman Orang Tua") tidak ditetapkan | TEMPLATE_EVENT 4.2 baru; tes membaca dokumen; tabel 9 jenis di README dijaga tes (`TestReadmeNineTypeTableMatchesTemplateEvent`) |
| ✔ 20 | Stok souvenir vs `souvenir_received`: satuan (per undangan atau per orang) tidak didefinisikan | per undangan ([BE bab 10 butir 2](05_BUKU_TAMU/BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)); stok `floor(Σ quantity)` (dijalankan pada backend Menu 04: 350) |
| ✔ 21 | FE: `AppLayout`/`nav_active` (desain 01 FE memakai `PlannerLayout`); tombol *Kirim WA bergilir* dan *Aktifkan Undangan Digital Adicara* tanpa spesifikasi atau modul; "kotak kuning" tanpa token; chip VIP/Keluarga khusus pernikahan | `PlannerLayout`; tombol dibuang (banner Standalone tanpa aksi); token `--sage-*`; dropdown kategori dari registry |
| ✔ 22 | **Nginx mengirim `Permissions-Policy: camera=()`** (`deploy/nginx/adicara.conf` baris 51), yang memblokir `getUserMedia` di produksi sehingga pemindai QR tidak dapat berfungsi | prasyarat T0 FE: `camera=(self)`; konfigurasi **tidak diubah** di sini |
| ✔ 23 | `PUBLIC_BASE_URL` ada di `.env.example` tetapi tidak dibaca kode (`backend/README.md`) | dicatat sebagai prasyarat modul undangan ([BE bab 5.1](05_BUKU_TAMU/BE.md#51-antarmuka-go-dimiliki-plannerdomain)) |
| ✔ 24 | Planner tidak dapat membuat gambar QR (butuh pustaka; "tanpa dependensi baru") sehingga di Standalone pemindaian tak punya tiket | ekspor memuat kolom *Kode Tiket* untuk alat luar; keputusan pemilik: setujui pustaka QR |
| ✔ 25 | `/i/[slug]` memeriksa `guest` (bukan `to`) untuk `noindex` | dicatat untuk modul undangan ([BE bab 10 butir 27](05_BUKU_TAMU/BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)) |
| ○ 26 | Constraint DDL yang disarankan (`pax_actual BETWEEN 0 AND 20`, `is_checked_in = (checked_in_at IS NOT NULL)`) | belum diterapkan di migrasi 011 (boleh diubah di tempat; usecase menjaminnya) |

**Yang dijalankan:** blok SQL BE bab 2 dan 5 apa adanya dalam transaksi di-rollback (daftar + filter, ringkasan, proyeksi, scan, check-in dengan dua sesi `psql`, batal, walk-in, KPI dengan penolak yang hadir, antrean, RSVP, kunci proyek, `INSERT` multi-baris, stok souvenir, kasus B/C/hapus undangan); backend Menu 01 (hapus sesi memindahkan tamu, `guest_pax_count`, cascade), 03 (`PUT` pos berharga satuan 34 porsi → 200, tanpa `vendor_id` melepas vendor, `quantity` 0 → 400), 04 (stok souvenir 350) yang sebenarnya; modul acuan Go (43 tes, cakupan `domain` 93,5% dan `xlsx` 90,4%, **24 mutasi kode semuanya digagalkan** setelah empat tes diperkuat) dan TypeScript (28 tes, `tsc` strict bersih bersama 01–04, **24 mutasi semuanya digagalkan**); DTO Go dibandingkan dengan tipe TS (21 pasangan, 125 jalur, **nol selisih**; pembanding diuji dengan enam mutasi); blok kode di BE.md/FE.md diekstrak kembali dan sama persis dengan berkas yang dikompilasi.

**Tidak berubah:** tidak ada perubahan kode backend, frontend, migrasi, OpenAPI, atau Nginx; database `adicara` tidak disentuh (database sementara `planner_spec05_scratch_test` dihapus). **Belum diuji:** modul `plannerguest` (belum ada), jalur HTTP menu ini, frontend, balapan impor dan batas 5.000, `.xlsx`/templat yang dibuka di Excel/Numbers, kamera dan `BarcodeDetector` pada perangkat, kinerja pada 5.000 tamu, PostgreSQL 18, Docker, dan Nginx.

**Dokumen yang berubah.** `05_BUKU_TAMU/BE.md`, `FE.md`, `README.md` (ditulis ulang) dan `05_BUKU_TAMU/fixtures/` (baru: `planner-guests.json`, `planner-reception.json`, `planner-catering-projection.json`, `planner-import-dryrun.json`, `planner-import-sample.csv`); `00_KONTRAK_BERSAMA/README.md` (status, BFF butir 6, bab 1.4, bab 6 paginasi dan kode galat `guest_limit_reached`, matriks #8 dan #11, bab 9 baris Tamu, bab 10 audit); `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` (bab 4.2 baru); `01_DASHBOARD/FE.md` (BFF: batas badan jalur impor); `04_VENDOR_DAN_BELANJA/FE.md` (bab 2.1: sumber serah terima souvenir); `README.md` (status).

## 2026-10-23 (lanjutan) — FE.md, BE.md, dan README menu 04 dicocokkan dengan kode dan antar-menu

Permintaan: apakah FE.md, BE.md (dengan implementasinya), dan README menu 04 sudah jelas, universal untuk jenis acara lain, dan saling terintegrasi. Diperiksa dengan **menjalankan**, bukan membaca ulang: tipe TypeScript FE dibandingkan dengan struct Go di kode (26 pasangan, 378 jalur, nol selisih; lima blok kode FE sama persis dengan yang dikompilasi, `tsc` strict bersih, 19 tes lulus), 17 rute router = tabel BE bab 3 = katalog FE bab 4.4, contoh respons FE bab 4.5 dan katalog galat bab 4.6 dijalankan terhadap backend asli (41 kasus galat), pustaka `vendor.ts` dijalankan terhadap respons asli kesembilan jenis acara (548 pemeriksaan), tabel README dibandingkan dengan registry, dan SQL Menu 06 yang membaca vendor dijalankan terhadap skema asli.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | Dua teks contoh di FE bab 4.6 tidak sama dengan server: `invalid_template` (`template:` seharusnya `shopping_list:`) dan `rollup_conflict` (kalimat lain) | FE bab 4.6 disalin dari respons asli |
| ✔ 2 | FE bab 4.4 menulis `404` pada `unlinkFromBudget` (server: `400 invalid_input` untuk pos yang bukan milik proyek) dan tidak menyebut `400 invalid_json` untuk id yang bukan UUID di body `link`/`unlink`; filter daftar `scope`/`category` juga dapat `400` dan tidak tercatat di BE bab 3.4 | FE bab 4.4 dan BE bab 3.4 (baris filter query) dikoreksi |
| ✔ 3 | Pernyataan basi: FE bab 11.4 "belum bisa dijalankan sampai backend ada", FE bab 11.5 "cuplikan respons tulis belum disalin dari respons asli", dan "belum teramati karena Menu 04 belum ada" di 02 FE (bab 4 dan 11.5, bab 13 butir 1) serta 03 FE (bab 11.5) | diganti hasil pengamatan 2026-10-23 (`vendor_id`/`vendor_name`/`vendor_wa` pada tugas, `vendor_name` pada pos, `rollup_source: "shopping"`) |
| ✔ 4 | 05 BE bab 2.5 menulis stok souvenir = Σ `quantity`, sedangkan 04 BE §2.5 dan kodenya `floor(Σ quantity)` | 05 BE disamakan |
| ✔ 5 | Kunci Menu 03 atas pembayaran pos yang digulung (`rollup_locked`) hanya diuji di Menu 03 dengan `rollup_source` yang diisi lewat SQL; Menu 04 tidak membuktikan bahwa `link-to-budget` benar-benar menghidupkannya | tes `TestMenu03LocksThePaymentsOfARolledUpLine` |
| ✔ 6 | Tabel 9 jenis acara di README menu tidak dijaga tes terhadap registry (hanya BE.md dan TEMPLATE_EVENT yang dijaga) | tes `TestReadmeNineTypeTableMatchesRegistry`; paragraf Status README diperpendek menjadi butir |

Tidak berubah: tidak ada perubahan perilaku backend; dua tes ditambahkan sehingga total menjadi **334 tes tingkat atas** (73 baru untuk menu ini), `-race` dan cakupan modul 91,4% dijalankan ulang sesudahnya (tanpa `DATA RACE`). Belum diuji: frontend (belum ada), `.xlsx` di Excel/Numbers, `409 conflict` (tabrakan transaksi), `5xx`.

## 2026-10-23 — BE menu 04 diimplementasikan: koreksi spesifikasi dari hasil membangunnya

Permintaan: apakah BE.md menu 04 sudah siap diimplementasikan, dan bila sudah, implementasikan sebagai kode mengikuti arsitektur yang ada. Hasil: **siap, dengan satu celah yang baru terlihat saat semua jenis acara dijalankan** (pos gulungan bawaan bertabrakan dengan pos khusus vendor, di bawah); modul `plannervendor` kini ada ([dokumen](../../backend/planner/2026-10-23-planner-menu-04.md)): 17 endpoint, provider `vendors`, applier `shopping`, dua overlay Proyek Contoh, registry `vendor_categories`/`shopping_categories`/`ShoppingRollup`, `NormalizeWA` di `planner/domain`, dua ekspor `.xlsx`, dan 17 operasi + 26 skema di `openapi.yaml`. Dijalankan terhadap PostgreSQL 16.2 (database sementara `planner_vendor_impl_test`, dihapus setelahnya): **334 tes tingkat atas lulus** (261 sebelumnya + 73 baru), `-race`, cakupan 91,4%, `redocly lint` valid, 37 mutasi kode produksi (36 digagalkan, satu setara).

| # | Temuan saat membangun | Koreksi |
|---|---|---|
| ✔ 1 | **Pos gulungan bawaan peresmian (`prosesi_vip`), wisuda (`toga_atribut`), dan seminar (`kit_sertifikat`) juga pos khusus kategori vendor** (`mc`, `toga`, `percetakan`; TEMPLATE_EVENT 15.2). Setelah vendor dikunci dan tertaut, "Hitung pos dari daftar belanja" berakhir `400 must not be linked to a vendor or document`. Tidak terlihat pada audit (tabel 15.2 dan 15.4 benar sendiri-sendiri); terlihat oleh tes yang menjalankan alur penuh untuk sembilan jenis | pos cadangan `perlengkapan` bila pos bawaan ada tetapi tidak dapat digulung (BE §2.4, butir 22; TEMPLATE_EVENT 15.4); `default_target` melaporkan pos yang akan dipakai, jadi FE tidak berubah. Arah sebaliknya (rollup lebih dulu) sudah benar. Keputusan pemilik: ubah 15.2 atau 15.4 |
| ✔ 2 | **Pos yang dibuat `lock-deal` dan gulungan tidak memeriksa batas 300 pos Menu 03** (spesifikasi diam) | `409 budget_item_limit_reached`, seluruh panggilan dibatalkan; terbukti di bawah balapan (butir 23). Menu 02 `sync-budget` tidak memeriksa batas itu (○ 11) |
| ✔ 3 | **Tabel validasi menyebut `404` untuk id pos non-UUID pada body**; dekoder JSON menjawab `400 invalid_json` (sama dengan `ids` pada `reorder` Menu 03) | tabel BE §3.4 dan butir 24 |
| ✔ 4 | **Kunci proyek bukan satu-satunya penutup balapan `lock-deal`:** pembaruan bersyarat `vendor_id IS NULL` sudah mencegah tautan hilang; tes balapan awal (delapan vendor sekategori) tidak menggagalkan mutasi "tanpa kunci" | tiga tes balapan: tautan, pos baru tanpa pos khusus (tes regresi: tanpa kunci pun tidak gagal), dan batas 300 pos (tanpa kunci tujuh dari delapan deal lolos pada 296 pos, tes gagal); butir 28 |
| ✔ 5 | **Kode acuan `BuildOverview` di BE.md mengurutkan irisan masukan di tempat** dan blok kodenya memakai dua pagar ```go berturut-turut | tidak mengurutkan (urutan repository); pagar diperbaiki |
| ✔ 6 | Satu mutasi awalnya selamat karena tes tidak memeriksa hitungan `pipeline` dan keadaan kategori ketika satu-satunya vendor dibatalkan | tes menambahkan kedua pemeriksaan |
| ✔ 7 | **Label kategori vendor** untuk kunci yang butir TEMPLATE_EVENT 7–14 tulis tanpa label tidak ditetapkan | kata biasanya ("Venue", "Dekorasi", …); dijaga `TestVendorCategoriesMatchTemplateEvent` yang membaca dokumen (butir 28) |
| ✔ 8 | **Batas ekspor 10/menit** tidak menyebut apakah per rute dan apakah permintaan yang gagal `format` terhitung | satu pembatas untuk kedua ekspor, dihitung sebelum handler (butir 26) |
| ✔ 9 | Tata letak `.xlsx` hanya rancangan di BE §2.7 | dua lembar dibuat dan dibaca ulang dengan `excelize` (judul, header, baris TOTAL, angka sebagai bilangan, tautan `wa.me`, awalan `'`); belum dibuka di Excel/Numbers |
| ✔ 10 | **Fixture FE dibangun dari basis data dan fungsi acuan**, bukan dari respons HTTP | respons asli `GET /vendors` dan `GET /shopping-items` pada Proyek Contoh dibandingkan dengan fixture: struktur, tipe, dan nilai sama (nol selisih, id dinormalkan); struct Go di kode identik dengan blok BE bab 4 (26 struct) |
| ○ 11 | **Menu 02 `sync-budget` membuat pos tanpa memeriksa batas 300 pos** (sudah ada sebelum menu 04) | tidak diubah; dicatat (butir 23) |
| ○ 12 | **Pos skenario bernilai 0** (paket 10 Jt tak punya dokumentasi) memberi `limit_source: scenario` dengan pagu 0 dan tanpa `price_delta`; FE menampilkan "Rp 0 · skenario" | keputusan produk/FE: baca 0 sebagai "tidak dianggarkan"? (butir 27) |

**Dokumen yang berubah.** `04_VENDOR_DAN_BELANJA/BE.md` (status, §2.4 pos cadangan, §3.4, bab 7.2 nama tes dan 7.3 hasil, bab 8, bab 9 butir 6, 9, 22–28; kode `BuildOverview` tanpa pagar ganda), `README.md` dan `FE.md` (status; FE bab 4.1, 4.5, 8.2, 11.5, 12.1, 14 butir 1), `00_KONTRAK_BERSAMA/README.md` (status, uji emas bab 9, bab 10), `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` (15.4 pos cadangan), `03_KEUANGAN/BE.md` (bab 9 butir 29), `README.md` (status). Di luar PLANNER_SPEC: `backend/docs/openapi.yaml`, `docs/backend/planner/2026-10-23-planner-menu-04.md` (baru), `docs/backend/README.md` dan `CHANGELOG.md`, `docs/api.md`, `docs/architecture.md`, `docs/database.md`, `backend/README.md`, `README.md`.

**Terbukti pada kode, bukan hanya pada SQL audit:** Proyek Contoh lewat seluruh tumpukan (7 dari 10 kategori, fix 58.000.000, pipeline 4.300.000, tiga kelompok pas, Catering 30.000.000 dengan selisih +5.000.000, 37 barang 4.578.500 / 2.063.800 / 16 / 21, anggaran tetap 74.150.000); gulungan pada pos berpembayaran (`rollup_conflict`, lalu 4.578.500 / 2.063.800 / 2.514.700 dan total proyek 75.228.500 / 32.113.800); alur penuh untuk sembilan jenis acara tanpa pos ganda; delapan `lock-deal` serentak; 300 pasangan `LineTotal` dari PostgreSQL dan 26 vektor `NormalizeWA`; 17 rute terdaftar = 17 operasi dan 26 skema = tag JSON struct; `redocly lint` valid (3 peringatan lama).

**Belum dijalankan:** frontend; Menu 05 (membaca stok souvenir dan memakai `NormalizeWA`); `.xlsx` di Excel/Numbers/LibreOffice; cuplikan respons tulis FE bab 4.5 sebagai respons asli; image Docker, Nginx, PostgreSQL 18, beberapa replika; latensi pada 200 vendor dan 1.000 barang.

**Keputusan produk yang menunggu** (pilihan bawaan sudah dipakai): BE bab 9 butir 22 (tabrakan pos gulungan dengan pos vendor), 27 (pagu 0), serta butir 1–21 yang sudah ada.

## 2026-10-22 — Menu 04: README, BE, dan FE disamakan dengan pola menu 01–03 (audit dengan menjalankan SQL dan backend Menu 01–03)

Permintaan: apakah `04_VENDOR_DAN_BELANJA/{README,BE,FE}.md` sudah sesuai, dengan mengikuti pola menu 01 sampai 03. Hasil: **belum**. DDL dan angka PDF benar (blok SQL bab 1 identik dengan migrasi 006 dan 009; 37 barang, subtotal, 16/21, 10 kategori, 7 FIX sama dengan PDF Hal. 9–10 dan 12 menurut pembandingan terprogram), tetapi dokumen masih berbentuk spesifikasi awal: README tanpa status dan tabel jenis acara, BE (373 baris) tanpa bentuk respons, tabel validasi, bab sambungan ke Menu 01, dan DTO yang tidak mengikuti pola implementasi, FE (122 baris) hanya wireframe tanpa kontrak data, dan beberapa klaim yang gagal bila dijalankan atau bertentangan dengan Menu 01–03 yang sudah menjadi kode. Audit dilakukan dengan **menjalankan**: PostgreSQL 16.2 sementara (`planner_spec04_scratch_test`, migrasi 001–012 nyata lewat `go run ./cmd/migrate up`, dihapus setelahnya; database `adicara` tidak disentuh), **backend Menu 01–03 yang sebenarnya** pada port sementara (Proyek Contoh dan sembilan jenis acara dibuat lewat HTTP, preset skenario diterapkan), PDF Hal. 9–10 dan 12 dibaca ulang dengan `pdfminer`, serta kode referensi Go dan TypeScript di dokumen. ✔ = terbukti dengan eksekusi, ○ = dari pembacaan atau penalaran.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Wireframe FE menulis pagu Catering 25.000.000 (selisih 0)**; aturan BE terhadap skenario aktif 75 Jt (10 pos, `catering` resepsi) menghasilkan **30.000.000**, jadi deal 25.000.000 menghemat 5.000.000 ([03 BE bab 6](03_KEUANGAN/BE.md#6-master-data) sendiri mencatat estimasi pos 25 Jt berbeda dari preset 30 Jt) | wireframe, tabel 04 BE §5.2 (kolom pagu dan selisih), fixture, dan tes (`pagu: catering 30 Jt…`) |
| ✔ 2 | **Status bayar "DP 0,9 jt" pada baris Venue mustahil**: `venue_dekor` adalah pos bersama yang tidak bertaut ke vendor (BE §5.2 sendiri), jadi `payment` kosong | `limit_groups[].line` (angka view pos bersama) dan baris ringkasan kelompok di FE |
| ✔ 3 | **Pagu yang disalin ke `budget_limit` basi**: Menu 03 tidak pernah menulis ke tabel vendor (catering 30 Jt di skenario 75 Jt menjadi 50 Jt di 100 Jt, salinan tetap 30 Jt); `limit_source` di DTO tidak punya kolom; `price_delta` generated tidak tahu pagu skenario | pagu dibaca setiap kali daftar dibuka; kolom hanya pagu manual (0 = ikut skenario); selisih dihitung dari pagu efektif; kolom `price_delta` dicatat mubazir (BE §1.1, butir 3, 4) |
| ✔ 4 | **Jenis non-pernikahan tidak pernah mendapat pagu dan mendapat pos ganda**: seluruh pos skenario dan template delapan jenis berada di sesi `all` (dijalankan lewat HTTP pada kedelapan jenis); vendor di sesi nyata → 0 kategori mendapat pagu; `lock-deal` aturan lama membuat pos `venue` kedua di `registrasi` (total estimasi seminar 120.000.000 dari pagu 100.000.000) | pencocokan sesi vendor lebih dulu, lalu `all` (BE §2.1, §2.2; TEMPLATE_EVENT 15 intro) |
| ✔ 5 | **`lock-deal` berbalapan**: dua vendor sekategori dikunci serentak, keduanya dijawab "ditautkan pos khusus" padahal tautan yang pertama tertimpa (indeks unik per vendor tidak keberatan); dengan `SELECT … FOR UPDATE` pada proyek: satu tertaut, satu pos baru | kunci proyek + pembaruan bersyarat `vendor_id IS NULL` (BE §2.2, §2.9) |
| ✔ 6 | **`lock-deal` pada pos berharga satuan** menyimpan estimasi 25.000.000 padahal jumlah × harga = 22.500.000; `PUT` pos berikutnya menimpanya kembali (aturan [03 BE §3.4](03_KEUANGAN/BE.md#34-pos-berharga-satuan)) | pos satuan hanya ditautkan, estimasi tetap, warning `unit_pricing` |
| ✔ 7 | **Pos baru dari nama vendor 255 karakter** melanggar `item_name` ≤ 150 (`23514`; temuan serupa di audit 02 #2) | `left(nama, 150)` dan kategori ≤ 60 |
| ✔ 8 | **Gulungan belanja**: barang yang dibuat sesudah gulungan aktif tidak ikut (DTO tanpa `budget_item_id`; estimasi pos tetap 4.578.500); mencabut gulungan meninggalkan 37 barang yang masih menunjuk pos, sehingga "semua barang yang belum bertaut" ambigu | `budget_item_id` opsional pada `POST /shopping-items`; `PUT` tidak mengubah tautan; `unlink` melepas semua barang; `default_target` memberi pos bawaan dan `has_payments` |
| ✔ 9 | **`round(quantity × harga)` meluap**: `9999999999.99 × 10¹³` → `bigint out of range` pada kolom generated (galat 500 bila tak dijaga) | batas `quantity` 0,01…1.000.000 dan total ≤ 10¹³, aritmetika bulat (`LineTotal`); 300 pasangan dibandingkan dengan PostgreSQL, nol selisih |
| ✔ 10 | **`NormalizeWA` salah untuk salah ketik umum**: `+62 0812…` menjadi `620812…`; titik, awalan `00`, dan nomor kosong tidak diatur; fungsinya dipakai Menu 05 tetapi disebut milik 04, padahal modul 02–06 hanya boleh mengimpor `planner/domain` | aturan baru, `planner/domain/phone.go`, 26 vektor bersama Go dan TS ([fixtures/planner-wa-vectors.json](04_VENDOR_DAN_BELANJA/fixtures/planner-wa-vectors.json)) |
| ✔ 11 | **Kunci kategori kustom**: tidak jelas siapa yang membuatnya; `ginutil.Slug` (calon yang jelas) menghasilkan tanda hubung dan `proyek`, bukan `^[a-z0-9_]{1,40}$` | `CategoryKeyFromLabel` (aturan `ScopeKeyFromName`), klien mengirim label untuk kategori kustom, juga pada `PUT` |
| ○ 12 | DTO: tag `binding`, struct anonim tanpa tag JSON (kunci Go bawaan), `Quantity float64` wajib, KPI `TotalLimitFix` tak terdefinisi, bentuk respons dan tabel validasi tidak ada | BE bab 3.3, 3.4, 4 ditulis ulang; blok Go dikompilasi; 26 pasangan tipe, 378 jalur, nol selisih terhadap tipe TS (pembanding diuji dengan enam mutasi) |
| ○ 13 | Ekspor `format=xlsx\|pdf` dijanjikan, padahal pustaka PDF belum disetujui ([kontrak bab 1.4](00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)); lembar kontak untuk WO/panitia tidak boleh memuat harga | hanya `xlsx` (`400` tanpa `format`); kontak tanpa harga dan catatan; tata letak dirancang (BE §2.7) |
| ○ 14 | **Pos gulungan non-pernikahan** `perlengkapan` tidak ada di 59 pos persentase ([03 BE bab 9 butir 29](03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)) | registry `ShoppingRollup` per jenis + [TEMPLATE_EVENT 15.4](00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#154-pos-gulungan-belanja) (kunci diperiksa terhadap 15.1 dengan skrip; `other` membuat `perlengkapan`); keputusan pemilik tetap menunggu |
| ○ 15 | SQL lintas modul tak tercatat: `lock-deal` dan gulungan menulis `planner_budget_items`; Menu 02, 03, 06 `JOIN planner_vendors`; Menu 05 membaca belanja | pengecualian keempat di kontrak bab 1.1; [04 BE §1.3](04_VENDOR_DAN_BELANJA/BE.md#13-tabel-dan-view-milik-modul-lain-yang-disentuh) |
| ○ 16 | Tanpa batas jumlah dan kode galat padahal daftar tidak berpaginasi; `phone`/URL/catatan tanpa panjang | 200 vendor, 1.000 barang, panjang kolom (BE §3.4); kode `vendor_limit_reached`, `shopping_item_limit_reached` |
| ○ 17 | `replace_template` belanja tanpa aturan `confirm` (02 dan 03 punya); dijalankan: pada Proyek Contoh 37 dari 37 barang membawa pekerjaan | BE §2.6 (`confirm: true` bila status/harga riil/tautan/souvenir) |
| ○ 18 | Status `fix` tanpa deal, tanpa `PATCH` status, melepas `fix` tak diatur | `fix` ⇒ `agreed_price > 0`; `PATCH …/status`; melepas tautan dengan warning `budget_link_released` (BE §2.2, §3.1) |
| ○ 19 | `GET /vendors/:id` menjanjikan tugas dan agenda yang menunjuk vendor, tanpa layar yang memakainya | dibuang (BE bab 9 butir 10) |
| ○ 20 | Apply skenario Menu 03 (sudah menjadi kode) menimpa estimasi pos yang bertaut vendor | lencana "Pos ≠ deal" dan warning `estimate_differs`; keputusan di BE bab 9 butir 7 |
| ○ 21 | FE: `AppLayout`/`nav_active` (seharusnya `PlannerLayout`); chip "Riset (3)" menjumlah Rencana dan Negosiasi; portofolio "URL/IG"; kolom Alamat bergambar tautan placeholder; "0,9 jt" tidak dihasilkan `formatRupiahShort`; komponen tersirat tak terdaftar; tanpa kontrak data, tipe, fixture, kode acuan | FE ditulis ulang mengikuti 03 FE (14 bab) |
| ○ 22 | [02 FE bab 13 butir 2](02_TIMELINE_DAN_TUGAS/FE.md#13-celah-dan-keputusan-yang-menunggu-pemilik-produk) dan 03 FE §2.1: bentuk `GET …/vendors` belum ditetapkan | ditetapkan (`VendorOverview`); kedua dokumen merujuknya |
| ○ 23 | README: tanpa status, tabel 9 jenis acara, dan integrasi bermekanisme tingkat kolom; tidak menyebut tiga mekanisme yang menyentuh Menu 03 | README ditulis ulang (58 → ±105 baris) |

Tidak berubah karena **sudah benar**: DDL migrasi 006 dan 009 (identik dengan blok BE); 37 nama, kategori, harga, dan status barang; subtotal kategori; 10 kategori vendor dan 7 FIX; daftar kategori vendor TEMPLATE_EVENT bab 7–14 sama dengan tabel 15.2 untuk delapan jenis dan setiap `budget_item_keys` ada di 15.1; `DELETE` vendor mengosongkan rujukan di tugas, pos, dan belanja; kueri provider `vendors` 01 BE §5.2 (7 / 3, 10 kategori terpakai, 7 fix).

**Ditambahkan.** `README.md` (status, tiga mekanisme yang dijaga, tabel 9 jenis acara, tabel integrasi). `BE.md` (373 → ±1.470 baris): status dan hasil audit, bab 1.3 (tabel milik modul lain), pagu dan kelompok pagu dengan kode acuan `BuildOverview`, `lock-deal` dengan SQL yang dijalankan, `NormalizeWA`, gulungan, stok souvenir, template, ekspor, kategori/KPI, konkurensi dan batas, 17 endpoint dengan bentuk respons dan tabel validasi, DTO Go yang dikompilasi, master data (vendor contoh dengan kolom pagu), bab 6 sambungan ke Menu 01 (provider, applier, dua overlay, registry), pengujian ✔/✘ dengan nama tes usulan, urutan pengerjaan, dan 21 celah. `FE.md` (122 → ±1.400 baris): seluruh struktur 03 FE (route dan file, kontrak props, alur data dan galat, prasyarat, kontrak data, library, pembungkus, dua tab, pesan galat, aksesibilitas, pengujian, urutan, celah, riwayat). `fixtures/` (baru, empat berkas). Kontrak: pengecualian SQL keempat, dua kode galat 409, warning `unit_pricing` dan `budget_link_released`, `ShoppingCategories []string` dan `ShoppingRollup`, `purchase_location` teks bebas, baris matriks 4–7, audit bab 10. TEMPLATE_EVENT bab 15.4 (baru) dan catatan sesi di bab 15.2. Kode di FE.md dan BE.md disisipkan langsung dari berkas yang dijalankan.

**Terbukti dengan eksekusi.** Blok SQL bab 2 (`scenario_items`, `vendor_read`, `lock_deal` + `link`/`reprice`/`create`, `unlock`, `rollup`, `unlink`, `shopping_kpi`, `status_patch`, `template_append`, `template_impact`, `vendor_provider`) dijalankan apa adanya (hanya `$n` diganti nilai) dalam transaksi yang di-rollback; gulungan 37 barang → 4.578.500 / 2.063.800 / 2.514.700 (total proyek 75.228.500 / 32.113.800); kueri provider `vendors` 7 / 3. Go (`vendorref`, tujuh tes tingkat atas): DTO `gofmt` + `go vet`; `NormalizeWA` 26 vektor; `CategoryKeyFromLabel`; `LineTotal` 300 vektor PostgreSQL; `BuildOverview` terhadap Proyek Contoh yang diekspor dari basis data, ditambah kasus batal, pagu manual, anggota tanpa penawaran, ganti skenario, tanpa skenario, kategori kustom/dilewati, dan fallback sesi Umum; 8 dari 9 mutasi digagalkan tes (satu setara). TypeScript: `tsc` strict (`noUncheckedIndexedAccess`, konfigurasi Astro repo) bersih bersama kode 01–03 yang diekstrak dari dokumennya; **19 tes `node:test` lulus**; **26 dari 27 mutasi kode** menjatuhkan tes (satu setara); ke-16 pembungkus `vendorApi` diuji metode, path, dan body; kode diekstrak ulang dari BE.md dan FE.md yang jadi dan dijalankan lagi (lulus). DTO Go ↔ tipe TS: 26 pasangan, 378 jalur, **0 selisih**, enam mutasi tipe semuanya dilaporkan. Seluruh tautan relatif di folder ini (666 tautan) diperiksa: nol rusak.

**Dokumen yang berubah.** `04_VENDOR_DAN_BELANJA/{README,BE,FE}.md` dan `fixtures/` (baru); `00_KONTRAK_BERSAMA/README.md` (status, bab 1.1, 2, 3, 6, 8, 9, 10), `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` (bab 15 intro, 15.4); `03_KEUANGAN/BE.md` (bab 3.1 baris skenario aktif, bab 3.12 baris integrasi, bab 9 butir 29), `03_KEUANGAN/README.md` dan `FE.md` (baris integrasi dan §2.1); `02_TIMELINE_DAN_TUGAS/FE.md` (bab 13 butir 2); `05_BUKU_TAMU/BE.md` (lokasi `NormalizeWA`); `01_DASHBOARD/BE.md` (§5.2 catatan provider `vendors`); `README.md` folder ini (status); entri ini. Tidak ada perubahan kode atau migrasi.

**Belum dijalankan.** Kode Go modul `plannervendor` (belum ada), jalur HTTP dan respons asli (fixture bukan respons endpoint), tata letak `.xlsx` (belum ada prototipe), balapan selain `lock-deal` (`link-to-budget`, `apply-template`, batas jumlah berbagi kunci proyek yang sama), latensi daftar pada data 10× contoh, kode frontend dan seluruh yang bergantung Astro atau peramban, PostgreSQL 18, image Docker, Nginx.

**Keputusan yang menunggu pemilik produk** (pilihan bawaan sudah ditulis; [04 BE bab 9](04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk), [04 FE bab 13](04_VENDOR_DAN_BELANJA/FE.md#13-celah-dan-keputusan-yang-menunggu-pemilik-produk)): batas 200 vendor dan 1.000 barang; `fix` mensyaratkan deal; kolom `price_delta` mubazir; pagu dibaca langsung (tidak disimpan); fallback pagu ke sesi Umum; **pos gulungan per jenis acara** (menutup 03 butir 29, usulan Adicara); Apply skenario menimpa estimasi pos tertaut vendor; hanya nomor Indonesia; ekspor hanya `.xlsx` dan kontak tanpa harga; label "Rencana" menggantikan "Riset"; menu 05 membaca stok lewat SQL.

## 2026-10-21 (lanjutan) — Audit kejelasan, universalitas, dan integrasi Menu 03

Permintaan: apakah README, BE (dengan implementasinya), dan FE menu 03 sudah jelas, universal untuk jenis acara lain, dan saling terintegrasi. Dijalankan, bukan hanya dibaca: tes baru `TestEveryEventTypeRunsTheWholeFinanceFlow` membawa **sembilan jenis acara** dari registry lewat alur yang sama (template wizard, preset + Apply, kontributor dengan peran jenis itu, setoran, pembayaran, cashflow, dashboard, ekspor, sinkron biaya dokumen Menu 02) terhadap router, PostgreSQL 16.2, dan wiring asli; lulus di sembilan jenis dan gagal di sembilan bila pemasangan item skenario ke pos dirusak. Suite penuh: 261 tes tingkat atas lulus, 0 gagal. Kode produksi tidak berubah. Dicocokkan juga: peran kontributor README = registry; 59 pos persentase; setiap `budget_item_keys` TEMPLATE_EVENT 15.2 ada di 15.1 untuk jenis yang sama (diperiksa dengan skrip, nol selisih).

**Satu ketidakcocokan lintas menu, dicatat sebagai keputusan** ([03 BE bab 9 butir 29](03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)): 04 BE §2.4 menyebut pos gulungan `perlengkapan` untuk jenis non-pernikahan, tetapi tidak ada pos berkunci itu di TEMPLATE_EVENT 15.1.

## 2026-10-21 (lanjutan) — BE menu 03 diimplementasikan: koreksi spesifikasi dari hasil membangunnya

Permintaan: apakah BE.md menu 03 sudah siap diimplementasikan, dan bila sudah, implementasikan sebagai kode mengikuti arsitektur yang ada. Hasil: **siap, dengan satu celah** (kategori pos persentase, di bawah); modul `plannerfinance` kini ada ([dokumen](../../backend/planner/2026-10-21-planner-menu-03.md)): 32 endpoint, provider `finance`, applier `budget`, overlay Proyek Contoh, registry `contributor_roles`, ekspor `.xlsx`, dan 32 operasi + 31 skema di `openapi.yaml`. Dijalankan terhadap PostgreSQL 16.2 sementara: 261 tes tingkat atas lulus (0 gagal, 0 di-skip), `-race` pada paket planner, plannertimeline, plannerfinance, dan `app` lulus, cakupan modul 91,3%, 23 mutasi kode produksi semuanya digagalkan tes, uji emas PDF Hal. 4–7 lulus lewat seluruh tumpukan.

| # | Temuan saat membangun | Koreksi |
|---|---|---|
| ✔ 1 | **Template persentase tidak punya kategori:** `planner_budget_items.category` `NOT NULL`, tabel TEMPLATE_EVENT 15.1 hanya memuat kunci, nama, persen | kolom **Kategori** ditambahkan (59 baris) di TEMPLATE_EVENT 15.1 dan kode; dijaga `TestTemplatesMatchSpec`; nama kategori adalah usulan (BE bab 9 butir 18) |
| ✔ 2 | **Template persentase untuk `target_budget` kecil menjadi 500:** hanya preset yang menolak total < Rp 100.000; wizard dengan pagu Rp 50.000 membuat `dana_darurat` negatif dan melanggar `CHECK` | di bawah Rp 100.000 semua estimasi 0 (BE §3.9, butir 19) |
| ✔ 3 | `load-preset` tidak menyebut nasib `total_target` dan `preset_key` skenario (skenario 20 Jt berisi paket 30 Jt) | keduanya ditulis (butir 20) |
| ✔ 4 | Dua item skenario tanpa kunci bernama sama berpasangan dengan pos yang sama (`UPDATE … FROM` dengan target ganda tidak terdefinisi) | yang kedua dibuat sebagai pos sendiri (§3.2, butir 21) |
| ✔ 5 | `mode` kosong, `scenario_id`/`a`/`b` kosong, urutan 404 dan 400, dan kewajiban `format=xlsx` tidak ditetapkan | ditetapkan dan diuji (§4.5, butir 22–23, 27) |
| ✔ 6 | Overlay Proyek Contoh hanya menyebut pembayaran, padahal dataset PDF memuat estimasi yang tidak dimiliki template | overlay menulis estimasi dan pembayaran (§3.12, butir 24) |
| ○ 7 | **Milestone bertarget 0 berstatus `is_secured: true`** (0 dari 0); proyek baru menampilkan dua milestone tanpa dana | dicatat sebagai keputusan FE (butir 25; 03 FE bab 14 butir 15) |
| ✔ 8 | **Tes pembatas laju menu 01 dan 02 bergantung pada jam dinding:** `newUserLimiter` memulai jam sapuan dari `time.Now()`, jam tetap tes (10.00 UTC 2026-10-21) sudah lewat, sehingga `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow` gagal setelah sekitar pukul 10.05 UTC (ditemukan pada baseline, sebelum kode menu 03 disentuh) | tesnya mengatur `lastSweep` (satu baris di tiap modul); salinan keempat di finance memulai jam dari `l.now` dan diuji di tiga tahun berbeda (butir 15) |
| ✔ 9 | Tes pembayaran serentak: delapan `full` pada satu pos dijawab sukses semua bila baris tidak dikunci; terkunci: satu sukses, tujuh `invalid_stage` | kunci baris (`FOR UPDATE` pada tabel dasar) terbukti diperlukan (§3.1) |
| ✔ 10 | Dua mutasi awalnya selamat: satu karena skrip memilih tes sasaran yang salah, satu **celah tes sungguhan** (filter `rollup_source IS NULL` pada kolom `dp` matriks tidak terjaga) | tes rollup menambahkan pembayaran lama sebelum rollup dan memeriksa `paid = dp + term_1 + term_2 + rollup_paid` |
| ✔ 11 | Daftar 7.2 usulan nama tes; yang dua baris tidak dibuat sebagai tes: kesamaan teks DDL dengan migrasi, dan atomisitas kegagalan applier `budget` | ditandai ○ di BE bab 7.2 dan 8; DDL naik-turun-naik dijaga tes migrasi Menu 01 |

**Dokumen yang berubah.** `03_KEUANGAN/BE.md` (status, §3.1, §3.2, §3.9, §3.11, §3.12, §4.5, bab 7.2, 8, 9 butir 14, 15, 18–28), `03_KEUANGAN/README.md` dan `FE.md` (status; FE bab 14 butir 1, 15, 16), `00_KONTRAK_BERSAMA/README.md` (status, matriks baris 2, bab 9 dan 10), `00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md` (15.1 kolom Kategori), `README.md` (status). Di luar PLANNER_SPEC: `backend/docs/openapi.yaml`, `docs/backend/planner/2026-10-21-planner-menu-03.md` (baru), `docs/backend/README.md` dan `CHANGELOG.md`, `docs/api.md`, `docs/architecture.md`, `docs/database.md`, `docs/backend/migrations/…-005-012.md`, `backend/README.md`, `README.md`.

**Terbukti pada kode, bukan hanya pada SQL audit:** DTO Go ↔ tipe TypeScript FE terhadap struct sebenarnya (29 pasangan, 329 jalur, 0 selisih); `redocly lint` valid (3 peringatan lama); 32 rute terdaftar = 32 operasi dan 31 skema = tag JSON struct (tes repo).

**Belum dijalankan:** frontend; Menu 04 (rollup dan vendor diuji dengan baris yang disisipkan tes); `.xlsx` di Excel/Numbers/LibreOffice; respons HTTP asli sebagai fixture FE; image Docker, Nginx, PostgreSQL 18, beberapa replika; latensi pada data jauh di atas Proyek Contoh.

**Keputusan produk yang menunggu** (pilihan bawaan sudah dipakai): BE bab 9 butir 18 (nama kategori template persentase), 19 (estimasi 0 di bawah Rp 100.000), 20 (`load-preset` mengubah total), 25 (milestone bertarget 0), serta butir 1–17 yang sudah ada.

## 2026-10-21 (lanjutan) — Menu 03: README, BE, dan FE disamakan dengan pola menu 01 dan 02 (audit dengan menjalankan SQL)

Permintaan: apakah `03_KEUANGAN/{README,BE,FE}.md` sudah sesuai, dengan mengikuti pola menu 01 dan 02. Hasil: **belum**. Angka dan DDL benar (blok SQL bab 1 identik dengan migrasi 008 dan 010; 60 nilai preset, 14 pos, 4 subtotal, dan 22 setoran sama dengan PDF Hal. 4–7 menurut pembandingan terprogram), tetapi dokumen masih berbentuk spesifikasi awal: tanpa status, DTO tidak mengikuti pola implementasi, banyak endpoint tanpa bentuk respons, tidak ada bab sambungan ke Menu 01, FE sepertiga dari panjang 02 FE tanpa kontrak data, dan beberapa klaim bertentangan dengan kode atau dokumen lain. Audit dilakukan dengan **menjalankan** SQL dan algoritmanya terhadap PostgreSQL 16.2 sementara (`planner_fin_scratch_test`, migrasi 001–012 dari berkas nyata, dihapus setelahnya; database `adicara` tidak disentuh), mengompilasi blok DTO Go, dan menjalankan kode referensi FE. ✔ = terbukti dengan eksekusi, ○ = dari pembacaan.

| # | Temuan | Koreksi |
|---|---|---|
| ✔ 1 | **Aktivasi skenario "nonaktifkan yang lain" sebagai satu `UPDATE` gagal `23505`** pada `planner_budget_scenarios_one_active_idx` saat pindah 75 → 100 Jt (indeks unik diperiksa per baris; pindah-balik kebetulan lolos, jadi hasilnya bergantung urutan baris) | dua pernyataan berurutan (03 BE §3.2) |
| ✔ 2 | Batas 3 skenario: dua sesi berbalapan **tanpa kunci menghasilkan 4 skenario**; dengan kunci proyek sesi kedua melihat 3 (sebelumnya hanya klaim) | terbukti; daftar penulisan yang memakai kunci proyek ditambahkan (§3.1) |
| ✔ 3 | **Pembayaran yang salah catat tidak bisa dibatalkan** (nominal wajib > 0, tidak ada hapus); `amount` pada tahap `full` ambigu | `DELETE …/payment/:stage`; `full` mengabaikan `amount`; `invalid_stage` juga bila sisa ≤ 0 (§3.3) |
| ✔ 4 | `SELECT … FROM planner_budget_lines … FOR UPDATE` ditolak (`FOR UPDATE cannot be applied to the nullable side of an outer join`) | kunci baris tabel dasar lalu baca view (§1.2) |
| ✔ 5 | Kolom view beku saat `CREATE VIEW`; `paid_total` dan `remaining_balance` salah untuk pos rollup (belanja selesai 900.000: view `paid 900.000`, tabel `paid_total 0`, `remaining_balance 3.500.000`) | aturan baca (hanya `estimate/paid/remaining/payment_status`), `remaining_balance` dicatat mubazir (§1.2, bab 9 butir 13) |
| ✔ 6 | Memindahkan pos ke sesi yang sudah punya `item_key` yang sama melanggar `planner_budget_items_key_idx` (`23505`); spesifikasi diam | kunci pos yang dipindah dinetralkan seperti 01 BE §4.6 (§3.10) |
| ✔ 7 | Pos satuan: `round(9999999999.99 × 10¹³)::bigint` → `bigint out of range` (galat 500 bila tidak dijaga); perhitungan `float64` di usecase | aritmetika bulat, batas `quantity` dan hasil ≤ 10¹³ sebelum menulis (§3.4, §4.5) |
| ✔ 8 | `apply-template` punya `confirm` tanpa aturan; `replace_template` menghapus estimasi hasil Apply skenario dan tautan | syarat dijalankan (13 dapat dihapus, 1 dilewati karena dibayar, 9 berisiko); `templateImpact` di FE (§3.9) |
| ✔ 9 | `clean_overwrite`: pratinjau tidak menyebut kerugian | `deleted {lines, paid_total}` (dijalankan: 16 pos, Rp 500.000) (§3.2) |
| ✔ 10 | Hapus skenario aktif tak terdefinisi | boleh; tidak ada yang aktif, item ikut terhapus (§3.1) |
| ✔ 11 | README: "template membuat pos berestimasi 0" untuk semua jenis; kontrak bab 5 dan TEMPLATE_EVENT bab 15: jenis lain persentase × pagu | prinsip 3 README, §3.9, tabel 9 jenis |
| ✔ 12 | Pembulatan preset persentase (Rp 1.000, selisih ke dana darurat) menghasilkan selisih **negatif** untuk total ≤ Rp 19.000 (melanggar `CHECK cost_amount >= 0`) | preset persentase menolak total < Rp 100.000 (§3.9) |
| ○ 13 | `PATCH /savings/target {"target_savings"}` menggandakan `PATCH /projects/:id` Menu 01 (`savings_target` + `clear_savings_target`, sudah di kode `UpdateProjectRequest`) dengan nama kolom berbeda, dan `null` pada struct Go tidak terbedakan dari "tidak dikirim" | endpoint dihapus; FE memakai PATCH proyek; 01 FE `UpdateProjectBody` perlu dua kolom (FE bab 1.3) |
| ✔ 14 | DTO: tag `binding` (pola 01 dan 02 memvalidasi di usecase dengan `<field>: <alasan>`); banyak struct tanpa tag `json` (kunci Go bawaan `DPAmount`); persentase dikirim (kontrak bab 6: backend tidak mengirim `*_pct`); `LineStatus.InProgress` padahal widget Menu 01 yang sudah di kode memakai `partial`; sebagian besar endpoint tanpa tipe permintaan atau respons | bab 4.4, 4.5, dan 5 ditulis ulang; blok Go dikompilasi (`go vet`) dan dibandingkan dengan tipe TS: 29 pasangan, 329 jalur, 0 selisih; pembanding diuji dengan enam mutasi |
| ✔ 15 | FE membutuhkan data yang tidak ada di DTO: "Skenario aktif / sisa plafon" (tidak ada sumbernya), daftar preset, "paket cepat", "kembalikan ke nilai preset" | `BudgetMatrix.active_scenario`, `PresetInfo`, `ScenarioItem.preset_amount`, `included`, `allocated`, `POST …/load-preset` |
| ✔ 16 | `GET /budget/items?scope=` — "scope_key atau `all`" ambigu; Menu 02 memakai: tanpa parameter = semua, `all` = hanya Umum | disamakan dengan Menu 02 (§4.3) |
| ✔ 17 | Dataset Proyek Contoh tidak memuat skenario (wireframe FE menampilkan tiga) dan tidak mencatat bahwa PDF tak punya tanggal bayar; urutan overlay terhadap vendor tak disebut | §6.3, §6.6, §3.12 |
| ○ 18 | Tidak ada bab sambungan ke Menu 01 (provider `finance`, applier `budget`, overlay, `ContributorRoles` di registry, wiring `app.go`, pengecualian SQL ketiga) | §3.12; kontrak bab 1.1 |
| ✔ 19 | Rekomendasi tabungan bergantung jam; angka 4.584.000 hanya benar bila sisa waktu 151–180 hari | jam menjadi parameter; dijalankan pada 4 Okt, 21 Okt 2026, H-1, dan lampau (§3.6) |
| ✔ 20 | FE: memakai `AppLayout`/`nav_active` padahal semua halaman `[id]/**` memakai `PlannerLayout`; `format.ts` disebut mengurai rupiah padahal 01 FE hanya memformat; aturan "persen berjumlah 100" bertentangan dengan wireframe (Tab 1 menjumlah 100,1%, Tab 3 99,9%); komponen yang tersirat tak terdaftar; sel Matriks Bulanan "bisa diedit" tak terdefinisi bila satu bulan memuat beberapa setoran | FE ditulis ulang: `PlannerLayout`, `parseRupiah`, `sharePercents` (angka wireframe dikoreksi: Seserahan 4,6%, Resepsi 89,6%), daftar komponen lengkap, `cellAction` |
| ✔ 21 | README dan FE menjanjikan "mencentang sesi menawarkan pos template" pada skenario, tanpa endpoint atau daftar template di FE | dibuang; template per sesi di Tab 3 (`apply-template` dengan `sub_event_scope`); dicatat sebagai celah (FE bab 14 butir 3) |
| ✔ 22 | PDF Hal. 5 punya baris "Subtotal Kategori" (181.850.000) yang menjumlah enam baris tumpang-tindih; tidak tercatat mengapa tidak dibawa | dicatat (README §1, BE §3.8) |
| ○ 23 | Baris status di kontrak dan README folder ini masih menyebut hanya menu 01 yang sudah menjadi kode (menu 02 sudah) | diperbarui |

**Ditambahkan.** `README.md`: status, tabel 9 jenis acara, tabel integrasi dengan mekanisme. `BE.md` (538 → ±1.200 baris): status dan hasil audit, sifat view, SQL yang dijalankan (Smart Adjust, pembayaran, likuiditas, milestone, rekomendasi, matriks sesi dan bulanan, template), bab 3.8–3.12 (matriks, template, ubah/pindah/hapus pos, ekspor `.xlsx`, sambungan ke Menu 01), 32 endpoint dengan bentuk respons dan tabel validasi, DTO Go yang dikompilasi, skenario Proyek Contoh, pengujian ✔/✘ dengan nama tes usulan, urutan pengerjaan, dan 17 celah. `FE.md` (207 → ±1.500 baris): seluruh struktur 02 FE (route dan file, alur data dan galat, kontrak data, library, pembungkus, tiga tab, pesan galat, aksesibilitas, pengujian, urutan, celah, riwayat). `fixtures/` (baru, lima berkas dibangun dari basis data). Kode di FE.md (`types`, `finance`, `financeApi`, `messages`, `props`) dan DTO di BE.md disisipkan langsung dari berkas yang dijalankan.

**Terbukti dengan eksekusi.** Tujuh blok SQL bab 3 BE dijalankan apa adanya (hanya `$n` diganti nilai); view pada data emas (74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000, status 7 / 6 / 1 / 0); Smart Adjust (9 diperbarui, 1 dibuat, ulang → tidak ada perubahan); kas bebas 13.950.000 dan defisit 26.650.000; milestone (Resepsi 39.750.000 = 59,9%); rekomendasi 4.584.000 / 2.292.000; matriks bulanan tanpa celah; kueri provider `finance` 01 BE §5.2 (7 / 7 / 0). FE: `tsc` strict (`noUncheckedIndexedAccess`) bersih, **15 tes `node:test` lulus**, **19 dari 20 mutasi kode menjatuhkan tes** (satu mutan setara), ke-32 pembungkus `financeApi` diuji metode, path, dan body. Seluruh tautan relatif di folder ini diperiksa: nol rusak.

**Dokumen yang berubah.** `03_KEUANGAN/{README,BE,FE}.md` dan `fixtures/` (baru); `00_KONTRAK_BERSAMA/README.md` (status, pengecualian SQL ketiga, empat kode galat 409, audit menu 03); `README.md` folder ini (status); entri ini. Tidak ada perubahan kode atau migrasi.

**Belum dijalankan.** Kode Go modul `plannerfinance` (belum ada), jalur HTTP dan respons asli (fixture bukan respons endpoint), ekspor `.xlsx` (tata letak adalah rancangan, belum ada prototipe), konkurensi selain batas skenario, kode frontend dan seluruh yang bergantung Astro atau peramban, latensi, image Docker, PostgreSQL 18, Nginx.

**Keputusan yang menunggu pemilik produk** (pilihan bawaan sudah ditulis; [03 BE bab 9](03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk), [03 FE bab 14](03_KEUANGAN/FE.md#14-celah-dan-keputusan-yang-menunggu-pemilik-produk)): target tabungan lewat PATCH proyek; pengecualian SQL ketiga; estimasi persentase pada template non-pernikahan; angka batas (100 / 300 / 20 / 2.000); setoran mendatang dihitung terkumpul; proyeksi RSVP pada pos satuan mana pun; `PUT` pos yang melepas vendor bila `vendor_id` tak dikirim; template pos per sesi di Tab 1.

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
