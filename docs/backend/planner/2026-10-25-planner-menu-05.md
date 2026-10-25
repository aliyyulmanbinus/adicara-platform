# Planner menu 05: buku tamu, RSVP, dan meja resepsionis

Tanggal: 2026-10-25. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/05_BUKU_TAMU/BE.md](../../APP_SPEC/PLANNER_SPEC/05_BUKU_TAMU/BE.md), [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md), dan [TEMPLATE_EVENT](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (bab 4 dan 4.2). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner Guest`). Dokumen menu lain yang disentuh modul ini: [menu 01](2026-10-21-planner-menu-01.md) (registry, widget, wizard, Proyek Contoh, pemindahan tamu saat sesi dihapus), [menu 03](2026-10-21-planner-menu-03.md) (pos berharga satuan yang diisi proyeksi porsi, lewat klien), dan [menu 04](2026-10-23-planner-menu-04.md) (stok souvenir, `NormalizeWA`).

Modul `plannerguest` memiliki **daftar tamu** (sisi, kategori, sesi, meja, pax, nomor WhatsApp, status undangan, RSVP) dan **meja resepsionis hari-H** (pindai tiket, check-in, tamu tanpa undangan, souvenir, antrean), ditambah proyeksi porsi katering dan impor/ekspor daftar. Seperti menu 02–04 ia tidak diimpor oleh `planner`; ia mendaftarkan diri lewat `SummaryProvider` (`guests`) dan `SampleOverlay` (`guests`), sehingga widget Buku Tamu di dashboard terisi dan Proyek Contoh memuat 20 tamu. Ia **tidak punya template wizard** (wizard tidak membuat tamu) dan menyediakan `RSVPWriter()` untuk modul Undangan Digital kelak.

## Yang ada

| Area | Isi |
|---|---|
| Daftar (12) | `GET/POST …/guests`, `GET …/guests/summary`, `GET …/guests/catering-projection`, `GET …/guests/export?format=xlsx`, `GET …/guests/import-template?format=xlsx`, `POST …/guests/bulk-import`, `GET/PUT/DELETE …/guests/:guest_id`, `PATCH …/guests/:guest_id/sent-status`, `PATCH …/guests/:guest_id/rsvp` |
| Meja resepsionis (7) | `GET …/guests/reception-overview`, `GET …/guests/export-attendance?format=xlsx`, `POST …/guests/scan-qr`, `POST …/guests/walk-in`, `POST/DELETE …/guests/:guest_id/checkin`, `PATCH …/guests/:guest_id/souvenir` |
| Registry | `GET /event-types` memuat `guest_sides` dan `guest_categories` (`{key, label}`) per jenis acara; jenis tanpa sisi (ulang tahun, wisuda, lainnya) mengirim `[]`, bukan `null` |
| Dashboard | provider `guests` mengisi widget 4 (`GuestSummary`) dari `overview` yang sama dengan `GET /guests/summary`, sehingga angka dashboard dan halaman tidak dapat berbeda |
| Proyek Contoh | overlay `guests`: 20 tamu fiktif (PDF Hal. 13 tidak memuat nama), 43 pax, 14 undangan terkirim, RSVP 12 hadir / 2 ragu / 2 menolak / 4 belum, pax_actual = pax_count untuk yang hadir |
| Modul undangan | `Service.RSVPWriter()` memenuhi `planner/domain.GuestRSVPWriter` (`ApplyRSVP` berbasis token tiket dan terikat ke undangan); `Options.Invitations` (`InvitationLookup`) dan `Options.PublicBaseURL` mengisi `personalized_url`. Keduanya kosong di `app.go` sampai modul undangan ada |

Total **19 endpoint**. Tidak ada migrasi baru: tabel sudah ada sejak `011_000_planner_guests.sql` (dirilis bersama 005–012, teksnya identik dengan blok SQL BE.md bab 1; diperiksa dengan `diff`). Tidak ada perubahan frontend (BFF Planner belum ada di repo).

## Struktur kode

```text
internal/modules/plannerguest/
  domain/       entity.go (DTO bab 4; tipe Menu 01 lewat alias), spec.go (GuestSpec: ResolveSide, ResolveCategory),
                overview.go (AssembleOverview, ReceptionFilters), projection.go (RecommendedPortions, bilangan bulat),
                importer.go (DecodeCSV, MapColumns, BuildImportPlan), upload.go (DecodeUpload), repository.go (antarmuka,
                sentinel), export.go, sample.go (20 tamu contoh)
  usecase/      service.go, validate.go, guest.go, reception.go, import.go (impor, templat, dua ekspor), integration.go
                (provider, overlay, RSVPWriter)
  repository/   postgres.go
  delivery/http router.go, handler.go, export.go (.xlsx), ratelimit.go
  delivery/http/xlsx  xlsx.go (pembaca), template.go (templat impor)
internal/modules/planner/domain/invitation.go    RSVPStatus, InvitationLookup, GuestRSVPWriter, PersonalURL
internal/modules/planner/domain/eventtype.go     + GuestSides, GuestCategories
internal/modules/planner/domain/guestcatalog.go  data registry sembilan jenis (TEMPLATE_EVENT 4.2)
internal/app/app.go                              + pemasangan modul (provider, overlay, rute)
```

Mengikuti menu 01–04: usecase tidak mengimpor `pgx`; transaksi dibawa lewat `context` oleh `TxManager` yang **sama** dengan milik `planner`. DTO tanpa tag `binding`, validasi di usecase dengan pesan `<field>: <alasan>`, daftar tidak pernah `null`, tidak ada persentase di respons (klien membagi). Kode murni (`AssembleOverview`, `RecommendedPortions`, pembaca CSV/`.xlsx`, `BuildImportPlan`, templat) berasal dari modul acuan yang sudah dijalankan pada audit spesifikasi (dengan 43 tesnya), dengan komentar diterjemahkan ke bahasa kode yang ada (Inggris). Data registry dan contoh dijaga tes yang membaca dokumen spesifikasi: `TestEventTypesCarryTheGuestRegistry`, `TestSampleGuestsMatchSpec`, `TestReadmeNineTypeTableMatchesRegistry`. Mengubah salah satu tanpa yang lain menggagalkan tes.

## Perilaku yang perlu diketahui

- **Satu fungsi untuk semua angka.** `AssembleOverview` melayani provider dashboard, `GET /guests/summary`, dan `overview` pada `GET /guests`; ia selalu menghitung seluruh proyek, tidak terpengaruh filter, pencarian, atau halaman. `side_categories` mengembalikan tabel PDF Hal. 13 (enam penghitung **per pihak**).
- **Check-in satu `UPDATE … WHERE is_checked_in = FALSE`.** Dari dua ketukan serentak satu menang dan sisanya `409 already_checked_in` (diuji delapan serentak: satu 200, tujuh 409). Check-in **tidak mengubah RSVP**: tamu yang menolak tetapi datang tetap `declined` di daftar dan dihitung hadir di meja (`expected = bukan penolak ∨ hadir`, jadi "belum hadir" tidak pernah kurang dari yang sebenarnya di luar). RSVP tidak dapat diubah sesudah check-in (`409`); membatalkan check-in menghapus tanda hadir dan souvenir tetapi membiarkan `pax_actual`.
- **Antrean meja** satu paket per siklus polling 5 detik (KPI, jumlah filter, 10 check-in terbaru, satu halaman 50): `status=all` = tamu yang diharapkan, `in` = yang sudah hadir (terbaru dulu), `pending` = yang belum; **pencarian melebarkan antrean ke tamu yang menolak** (satu mungkin berdiri di pintu). Stok souvenir = `floor(Σ quantity)` barang `purpose = 'souvenir'` berstatus `selesai` (Menu 04, baca-saja); kunci hilang bila tak ada barang bertanda souvenir.
- **Pencarian = `strpos`, bukan `LIKE`:** `%` dan `_` berarti dirinya sendiri; mencari nama, nomor meja, nomor apa adanya, dan bentuk `62…`. Urutan daftar = urutan dibuat (`sort_order`), impor melanjutkan urutan berkas; antrean meja urut nama menurut kolasi basis data (diuji pada `en_US.UTF-8`; hasilnya sama dengan urutan byte pada data contoh).
- **RSVP.** `pax_actual` hanya berarti untuk `attending` (0 = ikut `pax_count`); status lain menyimpan 0. `rsvp_submitted_at` dicap untuk setiap jawaban baru, dihapus untuk `unconfirmed`, dan dibiarkan bila jawaban yang sama dikirim lagi. `ApplyRSVP` idempoten, tidak mengubah tamu yang sudah check-in (dan berhasil), dan mencari tamu dengan pasangan undangan + token (token undangan lain `pd.ErrNotFound`).
- **Proyeksi porsi** dengan aritmetika bulat: `(c·(100+p)+99)/100 + buffer_pax`; hanya tamu `attending` yang dihitung. Tidak ada yang disimpan dan tidak ada SQL ke Menu 03: klien mengisi pos berharga satuan lewat endpoint Menu 03.
- **Impor.** Berkas dikirim sebagai JSON `{filename, content_base64}` (BFF hanya meneruskan JSON); badan dibatasi 7,5 MiB hanya untuk rute ini (selain itu 64 KiB), berkas 5 MiB setelah di-decode, 2.000 baris. CSV: BOM dibuang, pemisah `,` `;` atau tab dideteksi, Windows-1252 didekode; `.xlsx`: `UnzipSizeLimit` 64 MiB, pembacaan berhenti setelah 2.001 baris berisi. Nomor telepon yang terbaca sebagai angka tanpa awalan ditolak (tidak ditebak). `dry_run` tidak menulis dan tidak mengunci; impor sungguhan satu transaksi di bawah kunci proyek, membaca ulang tamu yang ada di dalam kunci, semua-atau-tidak (`skip_invalid` menulis baris valid; `skip_duplicates` melewatkan duplikat nama + nomor), dan batal bila total > 5.000. Templat `.xlsx` memuat judul yang dikenali impor, dropdown dari registry dan master sesi (daftar di lembar Petunjuk), kolom telepon bertipe teks, dan tanpa baris contoh.
- **Batas.** 5.000 tamu per proyek (kunci proyek memegang tambah, walk-in, dan impor; diuji serentak), 20 pax, panjang teks 255/60/30/40/2.000, `scan-qr` 120/menit, impor 5/menit, tiga unduhan (daftar, rekap, templat) 10/menit bersama, semuanya per pengguna dan dihitung sebelum handler (permintaan yang gagal validasi ikut terhitung). Badan lebih dari 64 KiB pada rute selain impor: `400 invalid_json`; hanya impor yang menjawab `413 file_too_large`.
- **Ekspor `.xlsx`** (`excelize`, sudah ada): `Daftar Tamu` (semua tamu dengan Kode Tiket, baris TOTAL) dan `Kehadiran` (tamu yang diharapkan atau hadir, jam masuk `HH.MM` di zona proyek, baris TOTAL), `Cache-Control: no-store`, nama berkas dari slug judul dan tanggal, teks berawalan `=`, `+`, `-`, `@` diberi `'`. Hanya `.xlsx`; PDF menunggu persetujuan pustaka.
- **Tautan personal** `{PUBLIC_BASE_URL}/i/{slug}?g={token}&to={nama}` hanya ada bila modul undangan, URL dasar, dan proyek bertaut tersedia; pencarian slug dilakukan sekali per permintaan. Slug hilang (undangan dihapus) = tanpa tautan; galat lain dari pencarian tidak disembunyikan (500). Selama modul undangan belum ada, tidak ada tamu yang memilikinya.
- **Tamu milik undangan saja** (`project_id IS NULL`) tidak pernah terbaca, terhitung, atau tertulis oleh modul ini (diuji lewat daftar, pemindaian, proyeksi, KPI, `GET`, dan `DELETE` dengan id-nya).
- **Proyek arsip** baca-saja (`409 project_read_only`) di semua 19 rute kecuali `GET`, ekspor, dan templat (diuji satu per satu).

## Dependensi

Tidak ada yang baru (`excelize` dari menu 02, `golang.org/x/text` sudah ada di `go.mod`).

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Semuanya dicatat di [BE.md bab 10](../../APP_SPEC/PLANNER_SPEC/05_BUKU_TAMU/BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk) (butir 29–38) dan [CHANGELOG spesifikasi](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **Bentrok token tidak diulang.** Spesifikasi menyebut "dicoba ulang maksimal 3 kali", tetapi di dalam transaksi pernyataan yang gagal membatalkan transaksi PostgreSQL (SQLSTATE 25P02 untuk semua pernyataan berikutnya). Indeks unik menjawab `409 conflict`; mengulang permintaan memberi token baru. Token 128 bit: peluang bentrok tidak terukur.
2. **Cap `rsvp_submitted_at` dibiarkan untuk jawaban yang sama.** Spesifikasi: `now()` pada setiap status selain `unconfirmed`, tetapi `ApplyRSVP` harus idempoten "tanpa perubahan"; satu pernyataan SQL dipakai kedua jalur, jadi `PATCH …/rsvp` yang diulang juga tidak menggeser cap.
3. **`RSVPStatus` tinggal di `planner/domain`** (bersama `GuestRSVPWriter`, yang menamainya); `plannerguest/domain` memakainya lewat alias, begitu juga `Option`, `GuestSide`, dan `RSVPSummary` (bentuk JSON tetap satu).
4. **`filters.categories` pada paket meja** memuat kategori registry (juga yang 0) dan, bila dipakai tamu yang diharapkan, `lainnya` dan teks kustom (`custom: true`); fungsi acuan di fixture hanya mendaftar kategori registry.
5. **`Limits` dan `RegisterRoutesWithLimits`.** Pembatas laju dihitung sebelum handler, jadi tes yang memanggil impor atau unduhan puluhan kali memerlukan batas yang dapat dimatikan; `RegisterRoutes` memakai `DefaultLimits` (120 / 5 / 10), dan tes pembatas memakainya.
6. **Badan JSON lebih dari 64 KiB** pada rute selain impor dijawab `400 invalid_json` (sama dengan menu 02–04); versi pertama modul ini menjawab `413 file_too_large` untuk semua rute, ditangkap oleh `TestGuestEndpointsRejectMalformedBodies`.
7. **Pesan galat tiga kode** mengikuti contoh di katalog FE bab 4.6 (`the guest is already checked in`, `at most 5000 guests per project`, `file too large`); `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays` membaca tabel itu dan memaksa backend menghasilkan tiap pesan.
8. **Walk-in memakai jam aplikasi** (`Options.Now`) untuk `checked_in_at` dan `rsvp_submitted_at`, sedangkan check-in biasa memakai `now()` basis data; keduanya UTC. Tes menyuntikkan jam tetap.
9. **Constraint DDL yang disarankan tidak diterapkan** (`pax_actual BETWEEN 0 AND 20`, `is_checked_in = (checked_in_at IS NOT NULL)`): usecase menjaga keduanya dan migrasi 011 tidak diubah.
10. **`PUBLIC_BASE_URL` belum dibaca `internal/config`** dan `Options.Invitations` kosong di `app.go`, sesuai BE bab 5.1: keduanya ditambahkan bersamaan dengan modul undangan.

## Pengujian

Dijalankan 2026-10-25 pada PostgreSQL 16.2 lokal, database sementara `planner_guest_impl_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 ./...` | 448 tes tingkat atas lulus, 0 gagal, 0 di-skip (334 sebelumnya + 114 baru: 56 HTTP, 40 domain, 10 usecase, 7 `.xlsx`, 1 `PersonalURL`) |
| `-race` pada `plannerguest/...` dan `planner/...` | lulus, tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`) | 91,9% pernyataan |
| `go vet ./...`, `gofmt -l` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Kontrak | `TestOpenAPIDocumentsEveryGuestRoute`: 19 rute terdaftar = 19 operasi bertag `Planner Guest`; `TestOpenAPISchemasMatchTheGoDTOs`: 24 skema sama dengan tag JSON struct Go (kunci, dan untuk respons kewajiban hadir); `TestOpenAPIEventTypeHasTheGuestRegistry` |
| Mutasi | 33 mutasi kode produksi (kunci proyek dan batas, SQL antrean dan KPI, cap RSVP, stok souvenir, urutan daftar, pembatas laju, batas badan impor, awalan rumus, zona jam masuk, dan lainnya) semuanya digagalkan tes; dua awalnya selamat (batas badan impor dan zona jam masuk, mesin uji kebetulan berzona WIB) dan menambah dua tes, empat tidak terkompilasi dan ditulis ulang |
| Universalitas | `TestEveryEventTypeRunsTheWholeGuestFlow`: sembilan jenis acara dari registry menjalani alur yang sama lewat router, PostgreSQL, dan wiring asli: sisi dan kategori dari registry, sisi jenis lain ditolak, ringkasan menjumlah ke total, impor dengan label, check-in dan KPI, widget dashboard, templat dengan kata jenisnya (Tamu / Peserta, kolom Pihak hanya bila ada sisi), kedua ekspor, dan wizard yang tidak membuat tamu |
| Dokumen ↔ kode | `TestEventTypesCarryTheGuestRegistry` (registry = TEMPLATE_EVENT 4.2: kunci, label, urutan), `TestReadmeNineTypeTableMatchesRegistry` (tabel 9 jenis di README), `TestSampleGuestsMatchSpec` (20 tamu = tabel BE bab 6), `TestFixturesAreCurrent` |
| **Respons HTTP asli = fixture FE** | `TestResponsesMatchTheFEFixtures`: daftar tamu, proyeksi, pratinjau impor, dan paket meja resepsionis (11 tamu ilustrasi hadir pada waktu fixture, stok souvenir 40) sama dengan keempat fixture JSON di `PLANNER_SPEC/05_BUKU_TAMU/fixtures/` setelah id, tiket, dan waktu dinormalkan: nol selisih. `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays`: setiap baris katalog galat FE bab 4.6 dihasilkan backend dengan status, kode, dan pesan yang tertulis. Keduanya diuji dapat gagal (satu nilai fixture dan satu pesan katalog diubah sementara) |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, pembaca/templat `.xlsx`, pembatas laju, kontrak berkas, dan tes yang membaca dokumen spesifikasi lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh lewat seluruh tumpukan (20 tamu / 43 pax, pihak 10/22 dan 10/21, terkirim 14, RSVP 12/2/2/4 dan 27 pax, enam kategori dan tabel per pihak, proyeksi 27 / 12 / 4 → 34 porsi, widget dashboard, `guest_pax_count` 43 pada kartu Hub, `usage.guests` 16 / 4 / 0); ilustrasi hari-H (11 dari 18 tamu yang diharapkan, 7 di luar, 22 dari 39 pax, 11 souvenir; seorang penolak yang hadir menjadi 12/19 dengan "belum" tetap 7); 8 check-in serentak → satu menang; 12 tambah serentak pada 4.995 tamu → tepat lima berhasil (`sort_order` unik); dua impor serentak pada 4.990 tamu → satu 200, satu 409; limit 5.000 tepat (tamu ke-5.000 lolos, ke-5.001 `409`); impor CSV Excel (BOM, `;`, CRLF) dan `.xlsx` termasuk sel telepon numerik; templat dibaca balik oleh pembaca impor; ekspor dibaca ulang (nama berawalan `=` aman, jam masuk `11.14` di Asia/Jakarta); isolasi pengguna B di 19 rute; arsip baca-saja; `ApplyRSVP` lintas undangan.

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- **Frontend menu 05**: tidak disentuh. Tipe TypeScript di [FE.md](../../APP_SPEC/PLANNER_SPEC/05_BUKU_TAMU/FE.md) cocok dengan DTO Go dan fixture-nya sama dengan respons asli, tetapi belum ada satu baris pun kode halaman. **BFF Planner** belum ada di repo; saat dibuat ia perlu batas badan 7,5 MiB untuk `…/guests/bulk-import` saja.
- **Nginx memblokir kamera**: `deploy/nginx/adicara.conf` mengirim `Permissions-Policy: camera=()`, jadi pemindai QR frontend tidak akan jalan di produksi sampai menjadi `camera=(self)`. Tidak diubah.
- **Modul Undangan Digital** belum ada: `personalized_url`, `ApplyRSVP` dari halaman publik, dan FK `invitation_id` belum dapat dicoba di luar tes dengan pencarian tiruan.
- Berkas `.xlsx` (daftar, rekap, templat) dibaca ulang dengan `excelize` tetapi belum dibuka di Excel, Numbers, atau LibreOffice (tata letak, lebar kolom, dropdown, validasi `Pax`).
- Kinerja pada 5.000 tamu (pencarian `strpos` memindai proyek, satu paket meja per 5 detik dari beberapa tablet): tidak diukur; hanya jalur 600 tamu yang diuji.
- Urutan nama antrean mengikuti kolasi basis data: diuji pada `en_US.UTF-8` (PostgreSQL 16.2 lokal); image `postgres:18-alpine` (musl) belum dicoba.
- Image Docker, PostgreSQL 18 (versi Compose), Nginx di depan rute baru, beberapa replika API (pembatas laju per proses).
