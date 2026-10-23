# Planner menu 04: vendor dan belanja kebutuhan

Tanggal: 2026-10-23. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/04_VENDOR_DAN_BELANJA/BE.md](../../APP_SPEC/PLANNER_SPEC/04_VENDOR_DAN_BELANJA/BE.md), [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md), dan [TEMPLATE_EVENT](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner Vendor`). Dokumen menu lain yang disentuh modul ini: [menu 01](2026-10-21-planner-menu-01.md) (registry, widget, wizard, Proyek Contoh), [menu 02](2026-10-21-planner-menu-02.md) (`planner_tasks.vendor_id`), dan [menu 03](2026-10-21-planner-menu-03.md) (pos anggaran, view `planner_budget_lines`, skenario).

Modul `plannervendor` memiliki **pipeline vendor** (kandidat → negosiasi → fix → batal, dengan pagu tiap kategori dibaca dari skenario aktif) dan **daftar belanja** (barang, harga per satuan, status, dan gulungan ke satu pos anggaran). Seperti menu 02 dan 03 ia tidak diimpor oleh `planner`; ia mendaftarkan diri lewat `SummaryProvider` (`vendors`), `TemplateApplier` (`shopping`), dan dua `SampleOverlay` (`vendors`, `shopping`), sehingga widget Vendor di dashboard terisi, wizard menawarkan template belanja pernikahan (`templates_available.shopping_list`), dan Proyek Contoh memuat 10 vendor dan 37 barang seserahan PDF Hal. 9–10 dan 12.

## Yang ada

| Area | Isi |
|---|---|
| Vendor (8) | `GET/POST …/vendors`, `GET/PUT/DELETE …/vendors/:vendor_id`, `PATCH …/vendors/:vendor_id/status`, `POST …/vendors/:vendor_id/lock-deal`, `GET …/vendors/export-contacts?format=xlsx` |
| Belanja (9) | `GET/POST …/shopping-items`, `PUT/DELETE …/shopping-items/:item_id`, `PATCH …/shopping-items/:item_id/status`, `POST …/shopping-items/apply-template`, `…/link-to-budget`, `…/unlink-from-budget`, `GET …/shopping-items/export-shopping-list?format=xlsx` |
| Registry | `GET /event-types` memuat `vendor_categories` (`{key, label}`) dan `shopping_categories` per jenis acara; pos penampung kategori (`budget_item_keys`) dan pos gulungan bawaan (`ShoppingRollup`) tinggal di server |
| Wizard | applier `shopping` (37 barang seserahan; hanya pernikahan); `templates_available.shopping_list` menjadi `true` untuk pernikahan saja |
| Dashboard | provider `vendors` mengisi widget 3 (`VendorSummary`) dari bagian KPI fungsi yang sama dengan tab |
| Proyek Contoh | overlay `vendors`: 10 vendor fiktif (7 fix, 3 rencana/negosiasi), `vendor_id` pada pos `catering` dan `dokumentasi` tanpa mengubah estimasi, dan tugas "Visit Venue Pernikahan" ditautkan ke venue; overlay `shopping`: status 37 barang (16 selesai = Rp 2.063.800, 21 proses) |
| Utilitas bersama | `planner/domain.NormalizeWA` (nomor WhatsApp; dipakai juga oleh Menu 05 nanti) |

Total **17 endpoint**. Tidak ada migrasi baru: tabel sudah ada sejak `006_000_planner_vendors.sql` dan `009_000_planner_shopping.sql` (dirilis bersama 005–012, teksnya identik dengan blok SQL BE.md bab 1). Tidak ada perubahan frontend.

## Struktur kode

```text
internal/modules/plannervendor/
  domain/       entity.go (DTO bab 4), overview.go (BuildOverview, pagu dan kelompok pagu), category.go, shopping.go
                (LineTotal, KPI belanja, stok), repository.go (antarmuka, sentinel, tipe internal), template.go (37 barang),
                sample.go (dataset Proyek Contoh), export.go
  usecase/      service.go, validate.go, vendor.go, lockdeal.go, shopping.go, rollup.go, template.go, export.go,
                integration.go (provider, applier, dua overlay)
  repository/   postgres.go
  delivery/http router.go, handler.go, export.go (.xlsx), ratelimit.go
internal/modules/planner/domain/phone.go          NormalizeWA (+ vektor bersama dengan FE)
internal/modules/planner/domain/eventtype.go      + VendorCategories, ShoppingCategories, ShoppingRollup
internal/modules/planner/domain/vendorcatalog.go  data registry sembilan jenis
internal/app/app.go                               + pemasangan modul (provider, applier, overlay, rute)
```

Mengikuti menu 01–03: usecase tidak mengimpor `pgx`; transaksi dibawa lewat `context` oleh `TxManager` yang **sama** dengan milik `planner`, sehingga applier berjalan di dalam transaksi pembuatan proyek. DTO tanpa tag `binding`, validasi di usecase dengan pesan `<field>: <alasan>`, daftar tidak pernah `null`, tidak ada persentase di respons. Kode inti (`BuildOverview`, `LineTotal`, `NormalizeWA`, `CategoryKeyFromLabel`, blok DTO) diambil dari blok yang sudah dijalankan pada audit spesifikasi, dengan komentar diterjemahkan ke bahasa kode yang ada (Inggris).

Data (37 barang, 10 vendor contoh, kategori dan pos penampung sembilan jenis) dijaga tes yang membaca dokumen spesifikasi: `TestWeddingShoppingMatchesSpec`, `TestSampleVendorsMatchSpec`, `TestVendorCategoriesMatchTemplateEvent`. Mengubah salah satu tanpa yang lain menggagalkan tes.

## Perilaku yang perlu diketahui

- **Pagu tidak disimpan.** `GET /vendors` membaca skenario aktif, vendor, dan baris view setiap kali (`BuildOverview`); sesi vendor dicoba dulu lalu `all` (seluruh pos skenario dan template delapan jenis non-pernikahan ada di `all`). Pagu manual (`budget_limit > 0`) menang atas skenario dan mengeluarkan vendor dari kelompok. Pos skenario bernilai 0 (paket 10 Jt pernikahan tak punya dokumentasi) memberi `limit_source: scenario` dengan pagu 0 dan tanpa `price_delta`.
- **Kelompok pagu.** Kunci pos yang dipakai ≥ 2 kategori pada registry (pernikahan `venue_dekor`, `rias_busana`, `hiburan`; peresmian `venue_dekor`; launching `produksi_av`; ulang tahun `katering_kue`; seminar `av_streaming`) dibandingkan di `limit_groups`, bukan per vendor; status bayarnya dibaca dari pos bersama (`limit_groups[].line`).
- **`lock-deal`** di bawah kunci proyek (`SELECT … FOR UPDATE` pada `planner_projects`), berurutan: pos yang sudah bertaut ke vendor (estimasi disamakan dengan deal), pos bersama (hanya dilaporkan: `shared_item` + `shared_budget_item`, estimasi tidak diubah), pos khusus kategori yang belum dipegang vendor atau gulungan (sesi vendor lalu `all`; pembaruan bersyarat `vendor_id IS NULL`), selain itu pos baru (nama dipotong 150 karakter, `item_key` hanya bila belum dipakai di sesi itu). Pos berharga satuan tidak ditimpa (`unit_pricing`). Meninggalkan `fix` (`PUT` atau `PATCH …/status`) melepas tautan (`budget_link_released`); mengunci ulang menautkan pos yang sama.
- **Gulungan belanja.** Total tidak disalin: `link-to-budget` menyalakan `rollup_source = 'shopping'` dan menunjuk barang ke pos, view menjumlahkan. Pos berpembayaran tersimpan butuh `confirm` (`409 rollup_conflict`); pos berharga satuan atau bertaut vendor/dokumen ditolak; barang yang sudah menunjuk pos lain tidak dipindah; idempoten. `unlink-from-budget` melepas **semua** barang lalu mematikan gulungan. Barang baru ikut gulungan hanya bila `POST /shopping-items` membawa `budget_item_id` pos itu.
- **Harga dan total.** `quantity` 0,01–1.000.000 (dua desimal; dibaca sebagai ratusan), harga per satuan; `round(quantity × harga)` dihitung dengan aritmetika bulat dan ditolak di atas 10¹³ sebelum menulis (`400`, bukan 500 dari `bigint out of range`).
- **Status barang.** `PATCH …/status` ke `selesai` mengisi harga riil dengan estimasi bila 0; harga yang sudah ada tidak ditimpa dan kembali ke `proses`/`belum` tidak menghapusnya. Stok souvenir = `floor(Σ quantity)` barang `purpose = 'souvenir'` yang selesai (hilang bila tak ada barang souvenir, 0 bila ada tetapi belum selesai).
- **Template belanja.** `append` idempoten (`ON CONFLICT … DO NOTHING`); `replace_template` menghapus hanya barang bertanda template, wajib `confirm` bila ada yang memuat pekerjaan pengguna (status, harga riil, tautan, tanda souvenir), dan memberi `budget_link_lost` untuk barang yang digulung. Jenis selain pernikahan: `400 invalid_template`. Batas 1.000 barang diperiksa setelah penyisipan; melewatinya membatalkan seluruh transaksi.
- **Nomor WhatsApp.** `phone_number` disimpan apa adanya, `formatted_wa_number` (62…) diturunkan oleh `NormalizeWA`; hanya nomor Indonesia (`400 invalid_phone`). Tautan `wa.me` dibentuk klien.
- **Kunci proyek** dipegang oleh `POST /vendors`, `lock-deal`, `POST /shopping-items`, `apply-template`, `link-to-budget`, dan `unlink-from-budget`. Batas 200 vendor, 1.000 barang, dan 300 pos (batas Menu 03, berlaku juga untuk pos yang dibuat di sini) diperiksa di dalamnya.
- **Ekspor `.xlsx`** (`excelize`, sudah ada): `Kontak Vendor` (vendor `fix`, **tanpa harga, pagu, dan catatan**; kolom WhatsApp berisi tautan `https://wa.me/…`) dan `Daftar Belanja` (semua barang, baris TOTAL). Angka sebagai bilangan berformat `"Rp" #,##0`; teks pengguna berawalan `'` bila diawali `=`, `+`, `-`, `@`; `Cache-Control: no-store`; nama berkas dari slug judul dan tanggal; 10 permintaan per menit per pengguna untuk **kedua** ekspor bersama.
- **Proyek arsip** baca-saja (`409 project_read_only`) di semua 17 rute kecuali `GET` dan ekspor (diuji satu per satu).

## Dependensi

Tidak ada yang baru (`excelize` dari menu 02, `golang.org/x/text` sudah ada).

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Semuanya dicatat di [BE.md bab 9](../../APP_SPEC/PLANNER_SPEC/04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) (butir 22–28) dan [CHANGELOG spesifikasi](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **Pos gulungan bawaan bertabrakan dengan pos khusus vendor pada tiga jenis acara.** Pos bawaan peresmian (`prosesi_vip`), wisuda (`toga_atribut`), dan seminar (`kit_sertifikat`) juga merupakan pos khusus kategori vendor (`mc`, `toga`, `percetakan`) menurut TEMPLATE_EVENT 15.2. Setelah vendor itu dikunci, `lock-deal` menautkannya ke pos tersebut, dan tombol "Hitung pos dari daftar belanja" berakhir `400 must not be linked to a vendor or document`. Ditemukan oleh tes yang menjalankan alur penuh untuk sembilan jenis acara. **Perbaikan:** bila pos bawaan ada tetapi tidak dapat digulung (bertaut vendor atau dokumen, atau berharga satuan), pos bawaan diganti pos cadangan `perlengkapan` (Kategori Perlengkapan, sesi Umum, dibuat sekali bila belum ada); `default_target` memberitahukan pos yang akan dipakai sehingga antarmuka tidak berubah. Arah sebaliknya sudah benar: pos yang sedang menggulung tidak diambil `lock-deal` (vendor mendapat pos sendiri). Keputusan pemilik: apakah pemetaan 15.2 atau pos bawaan 15.4 yang sebaiknya diubah.
2. **Pos yang dibuat di sini menghormati batas 300 pos Menu 03.** Spesifikasi tidak menyebutnya; `lock-deal` yang perlu pos baru dan gulungan yang perlu membuat pos bawaan menjawab `409 budget_item_limit_reached` dan membatalkan seluruh panggilan (vendor tidak jadi `fix`). Dibuktikan di bawah balapan: dari delapan deal pada 296 pos tepat empat berhasil. Menu 02 (`sync-budget`) tidak memeriksa batas ini; tidak diubah.
3. **Id non-UUID pada body `link-to-budget`/`unlink-from-budget`** dijawab `400 invalid_json` (dekoder), bukan `404` seperti tertulis di tabel validasi; UUID yang benar tetapi bukan pos proyek ini `400 invalid_input: budget_item_id: must be a budget line of this project`. Sama dengan perlakuan `ids` pada `reorder` di menu 03.
4. **`estimate_differs` pada `PUT` vendor** hanya bila `agreed_price` berubah pada permintaan itu, pos masih bertaut, dan estimasi efektif pos berbeda dari harga baru; mengirim ulang harga yang sama tidak memberi peringatan.
5. **Satu pembatas laju untuk kedua ekspor** (10 permintaan per menit per pengguna, bukan 10 per rute), dan ia berjalan sebelum handler sehingga permintaan dengan `format` salah ikut terhitung. Salinan kelima pola `ratelimit.go`.
6. **`apply-template` dan `link-to-budget` menerima badan kosong** (semua kolomnya opsional); `PUT`/`POST` lain tetap `400 invalid_json` tanpa badan yang sah.
7. **Label kategori vendor** untuk kunci yang butir TEMPLATE_EVENT 7–14 tulis tanpa label (`venue`, `dekorasi`, `katering`, `mc`, …) adalah kata biasanya ("Venue", "Dekorasi", …); label yang tertulis dipakai apa adanya (mis. `medis` "Dokter/Klinik"). Dijaga `TestVendorCategoriesMatchTemplateEvent`.
8. **`BuildOverview` tidak lagi mengurutkan masukannya** (versi acuan mengurutkan irisan milik pemanggil di tempat); urutan tampil adalah urutan repository (`sort_order`, waktu dibuat, id). `ScenarioRow` mendapat `ItemName` (nama pos skenario dipakai `shared_item` bila pos anggaran belum ada).
9. **Blok kode ganda di BE.md** (dua pagar ```go berturut-turut pada kode `BuildOverview`) dibuang.

## Pengujian

Dijalankan 2026-10-23 pada PostgreSQL 16.2 lokal, database sementara `planner_vendor_impl_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 ./...` | 334 tes tingkat atas lulus, 0 gagal, 0 di-skip (261 sebelumnya + 73 baru: 49 HTTP, 15 domain, 7 usecase, 2 `NormalizeWA`) |
| `-race` pada `plannervendor/...` dan `planner/domain` | lulus, tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`) | 91,4% pernyataan |
| `go vet ./...`, `gofmt -l` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Kontrak | `TestOpenAPIDocumentsEveryVendorRoute`: 17 rute terdaftar = 17 operasi bertag `Planner Vendor`; `TestOpenAPISchemasMatchTheGoDTOs`: 26 skema sama dengan tag JSON struct Go (kunci, dan untuk respons kewajiban hadir) |
| Mutasi | 37 mutasi kode produksi (kunci proyek pada `lock-deal` dan pada tambah vendor, pelaporan pos bersama, penimpaan estimasi pos satuan di dua pernyataan, pemotongan nama 150, kunci pos baru, pelepasan tautan, fallback sesi Umum, ambang kelompok, total baris belanja, pengisian harga riil dan penimpaan harga ketikan, konfirmasi gulungan, pelepasan barang, pos cadangan dan syarat `rollable`, konfirmasi dan batas template, peringatan `budget_link_lost`, pembulatan dan syarat stok, `WriteGuard`, `ProjectGuard`, filter proyek pada daftar belanja, awalan rumus, ekspor semua status, `Cache-Control`, `+62 0812`, batas 200, hitungan batal pada widget, tautan tugas dan status Proyek Contoh, `fix` tanpa deal, normalisasi nomor) — 36 digagalkan tes. Satu selamat dan setara: pencarian vendor tanpa filter proyek, karena setiap penulisan memakai filter proyeknya sendiri (pertahanan berlapis). Awalnya selamat dan ditutup: kunci proyek pada `lock-deal` (pernyataan bersyarat `vendor_id IS NULL` menutup balapan tautan, jadi tes balapan awal tidak cukup; kini ada tes batas 300 pos di bawah balapan yang gagal tanpa kunci: tujuh dari delapan deal lolos pada 296 pos), hitungan `pipeline` dan keadaan kategori vendor batal, dan empat mutasi yang tidak terkompilasi atau tak bermakna (dibetulkan atau dibuang) |
| Universalitas | `TestEveryEventTypeRunsTheWholeVendorFlow`: sembilan jenis acara dari registry menjalani alur yang sama lewat router, PostgreSQL, dan wiring asli: template anggaran, preset pertama lalu Apply, satu vendor per kategori di sesi nyata, pagu dari skenario (kategori bertas pos mendapat pagu, yang tidak, tidak), kelompok pagu per jenis, `lock-deal` semua kategori tanpa pos ganda, widget dashboard, barang belanja, gulungan ke pos bawaan (atau cadangan), dan dua ekspor |
| Integrasi dengan Menu 03 | `TestMenu03LocksThePaymentsOfARolledUpLine`: setelah `link-to-budget` lewat endpoint asli, `PATCH …/payment` dan `DELETE …/payment/dp` pada pos itu dijawab `409 rollup_locked` oleh Menu 03, dan setelah `unlink-from-budget` pembayaran diterima lagi (sebelumnya kuncinya hanya diuji di Menu 03 dengan `rollup_source` yang diisi lewat SQL) |
| Dokumen ↔ registry | `TestReadmeNineTypeTableMatchesRegistry`: tabel "Penerapan 9 Jenis Acara" di README menu (jumlah kategori vendor, kelompok pagu, jumlah kategori belanja, template, pos gulungan) sama dengan registry untuk kesembilan jenis; diuji dapat gagal dengan mengubah satu angka README |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, pembatas laju, kontrak berkas, dan tes yang membaca dokumen spesifikasi lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh lewat seluruh tumpukan (7 dari 10 kategori, tetap 58.000.000, pipeline 4.300.000, tiga kelompok pas, Catering pagu 30.000.000 selisih +5.000.000, status bayar katering `partial` dan fotografi `dp_paid`, 37 barang 4.578.500 / 2.063.800 / 16 / 21, subtotal enam kategori, anggaran tetap 74.150.000, widget dashboard 10 / 7 / 7 / 3); gulungan pada Proyek Contoh (`rollup_conflict`, lalu `confirm`: 4.578.500 / 2.063.800 / sisa 2.514.700, total proyek 75.228.500 / 32.113.800); 300 pasangan `LineTotal` dari PostgreSQL tanpa selisih dan 26 vektor `NormalizeWA` yang sama dengan FE; delapan `lock-deal` serentak pada satu kategori → tepat satu mengambil pos khusus, tujuh membuat pos sendiri, semua deal masuk anggaran; isolasi di 17 rute dan id proyek lain; arsip baca-saja; ekspor dibaca ulang (kontak tanpa harga, nama berawalan `=` aman, 10 per menit); **respons HTTP asli sama dengan fixture FE** (`planner-vendors.json` dan `planner-shopping.json`: struktur, tipe, dan nilai tanpa selisih setelah id dinormalkan).

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- Frontend menu 04: tidak disentuh. Tipe TypeScript di [FE.md](../../APP_SPEC/PLANNER_SPEC/04_VENDOR_DAN_BELANJA/FE.md) sudah dicocokkan dengan blok DTO Go dan fixture-nya sama dengan respons HTTP asli, tetapi belum ada satu baris pun kode halaman.
- Berkas `.xlsx` dibaca ulang dengan `excelize` tetapi belum dibuka di Excel, Numbers, atau LibreOffice (tata letak, lebar kolom, tautan WhatsApp, format `"Rp"`). Tata letaknya rancangan implementasi (BE.md bab 2.7 hanya menyebut kolom).
- Menu 05 belum ada: stok souvenir hanya dibaca lewat `GET /shopping-items` (`kpi.souvenir_stock`); pembacaan SQL langsung oleh Menu 05 belum diuji. `NormalizeWA` belum dipakai Menu 05.
- Balapan selain yang disebut: `PUT` barang bersamaan dengan `link-to-budget`, `apply-template` bersamaan dengan gulungan, `PUT` vendor bersamaan dengan `lock-deal`.
- Image Docker, PostgreSQL 18 (versi Compose), Nginx di depan rute baru, beberapa replika API (pembatas laju ekspor per proses).
- Latensi `GET /vendors` (tiga pembacaan) dan `GET /shopping-items` pada data jauh di atas Proyek Contoh (200 vendor, 1.000 barang): tidak diukur.
