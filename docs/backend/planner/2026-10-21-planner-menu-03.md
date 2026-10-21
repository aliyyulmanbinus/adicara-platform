# Planner menu 03: keuangan dan anggaran

Tanggal: 2026-10-21. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/03_KEUANGAN/BE.md](../../APP_SPEC/PLANNER_SPEC/03_KEUANGAN/BE.md), [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md), dan [TEMPLATE_EVENT](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner Finance`). Dokumen menu induknya: [planner-menu-01](2026-10-21-planner-menu-01.md); menu 02 yang menulis ke tabel anggaran: [planner-menu-02](2026-10-21-planner-menu-02.md).

Modul `plannerfinance` memiliki **simulasi anggaran** (skenario), **tabungan** (kontributor dan setoran), dan **realisasi** (pos anggaran, pembayaran per tahap, matriks, arus kas, ekspor). Seperti menu 02 ia tidak diimpor oleh `planner`; ia mendaftarkan diri lewat `SummaryProvider` (`finance`), `TemplateApplier` (`budget`), dan `SampleOverlay`, sehingga widget Keuangan di dashboard terisi, wizard menawarkan template anggaran, dan Proyek Contoh memuat data PDF Hal. 4–7.

## Yang ada

| Area | Isi |
|---|---|
| Skenario (13) | `GET/POST …/scenarios`, `GET/PUT/DELETE …/scenarios/:scenario_id`, `POST …/clone`, `POST …/load-preset`, `POST …/items`, `PUT/DELETE …/items/:item_id`, `POST …/scenarios/compare`, `POST …/scenarios/apply-preview`, `POST …/scenarios/apply` |
| Tabungan (8) | `GET/POST …/contributors`, `PUT/DELETE …/contributors/:contributor_id`, `GET/POST …/savings`, `PUT/DELETE …/savings/:entry_id` |
| Realisasi (11) | `GET …/budget/cashflow-summary`, `…/budget/matrix`, `…/budget/items?scope=`, `POST …/budget/items`, `PUT/DELETE …/budget/items/:item_id`, `POST …/items/reorder`, `PATCH …/items/:item_id/payment`, `DELETE …/payment/:stage`, `POST …/budget/apply-template`, `GET …/budget/export?format=xlsx` |
| Registry | `GET /event-types` memuat `contributor_roles` per jenis acara (`groom`, `bride`, `host`, `internal`, `family`, `sponsor`, `other`, label Indonesia) |
| Wizard | applier `budget` (14 pos pernikahan berestimasi 0; delapan jenis lain berisi pos persentase dari `target_budget`); `templates_available.budget_lines` menjadi `true` |
| Dashboard | provider `finance` mengisi widget 2 (`FinanceSummary`, kueri 01 BE §5.2) |
| Proyek Contoh | overlay `finance`: estimasi dan pembayaran 14 pos PDF, Arif dan Sinta dengan 22 setoran, tiga skenario (Opsi Ideal 75 Jt aktif) |

Total **32 endpoint**. Tidak ada migrasi baru: tabel dan view ada sejak `008_000_planner_finance.sql` dan `010_000_planner_budget_lines_view.sql` (dirilis bersama 005–012, teksnya identik dengan blok SQL di BE.md bab 1). Tidak ada perubahan frontend.

## Struktur kode

```text
internal/modules/plannerfinance/
  domain/       entity.go (DTO bab 5), repository.go (antarmuka, sentinel, tipe internal), export.go,
                money.go (aritmetika uang), template.go (preset, template, pembulatan), template_wedding.go +
                template_percent.go + sample.go (data, dibangkitkan dari tabel spesifikasi)
  usecase/      service.go, validate.go, scenario.go, apply.go (Smart Adjust), savings.go, budget.go
                (pos, pembayaran, matriks, template, ekspor), integration.go (provider, applier, overlay)
  repository/   postgres.go
  delivery/http router.go, handler.go, export.go (.xlsx), ratelimit.go
internal/modules/planner/domain/eventtype.go   + ContributorRoles
internal/app/app.go                            + pemasangan modul (provider, applier, overlay, rute)
```

Mengikuti menu 01 dan 02: usecase tidak mengimpor `pgx`; transaksi dibawa lewat `context` oleh `TxManager` yang **sama** dengan milik `planner`, sehingga applier berjalan di dalam transaksi pembuatan proyek. DTO tanpa tag `binding`, validasi di usecase dengan pesan `<field>: <alasan>`, daftar tidak pernah `null`, tidak ada persentase di respons. Semua angka anggaran dibaca dari view `planner_budget_lines`, tidak pernah dari kolom tabel (`paid_total` dan `remaining_balance` salah untuk pos rollup; keduanya tidak dikirim).

Data template (`template_wedding.go`, `template_percent.go`, `sample.go`) dibangkitkan dari tabel BE.md 6.1–6.4 dan TEMPLATE_EVENT 15.1 dengan skrip sekali jalan; `TestTemplatesMatchSpec` membaca tabel itu lagi dan membandingkannya, jadi mengubah salah satu tanpa yang lain menggagalkan tes.

## Perilaku yang perlu diketahui

- **Tiga peran, satu kunci masing-masing** (`finance` untuk widget, `budget` untuk template, `finance` untuk overlay). Tiap antarmuka menu 01 punya satu `Key()`, jadi tiap peran adalah tipe kecil yang membungkus `Service` (`Provider()`, `Applier()`, `Overlay()`), seperti menu 02 tetapi tanpa kunci kedua.
- **Kunci.** Penulisan yang membaca lalu menulis banyak baris mengambil `SELECT … FOR UPDATE` pada baris proyek (`LockProject`): buat/klon skenario, `load-preset`, tambah item skenario, `apply`, tambah pos, urut ulang, `apply-template`, tambah kontributor dan setoran. Penulisan satu pos (`PUT`, pembayaran, hapus pembayaran) mengunci **baris pos** pada tabel dasar, karena view tidak bisa dikunci. Dibuktikan dengan balapan: enam `POST /scenarios` serentak pada dua skenario menghasilkan tepat satu `201` (tiga putaran), delapan `POST /budget/items` pada 297 pos menghasilkan tepat tiga, dan delapan pembayaran `full` serentak pada satu pos membayar **sekali** (tiga putaran); tanpa `LockBudgetItem` tes itu gagal (mutasi).
- **Smart Adjust** (`apply`, `apply-preview`): pasangan item skenario ↔ pos dicari satu `SELECT` (kunci `(sesi, item_key)`, tanpa kunci lewat `lower(item_name)`); pos rollup dan berharga satuan dilewati; nilai yang sama tidak dihitung diperbarui (idempoten); pembayaran, vendor, dan dokumen tidak disentuh. Laporan dibangun sebelum menulis oleh fungsi murni, jadi `apply-preview` dan `apply` memberi laporan yang sama (diuji). Aktivasi skenario **dua pernyataan** (nonaktifkan lalu aktifkan): satu `UPDATE … SET is_active = (id = $x)` gagal `23505` pada indeks unik parsial.
- **`clean_overwrite`** menghapus semua pos proyek lalu menyalin item yang `included`; wajib `confirm: "RESET"` (tidak untuk `apply-preview`, yang mengisi `deleted {lines, paid_total}`).
- **Pembayaran.** `dp`, `term_1`, `term_2` mengganti seluruh isi tahap (koreksi), `full` mengisi sisa ke tahap kosong pertama dan mengabaikan `amount`, `DELETE …/payment/:stage` membatalkan, pos rollup menolak (`409 rollup_locked`), kelebihan bayar boleh. `receipt_url` hanya `http`/`https`, ≤ 500 karakter.
- **Harga satuan.** `estimate = round(quantity × unit_price)` dengan `math/big` (kuantitas dalam ratusan), diperiksa ≤ 10¹³ **sebelum** ditulis: hasil `999.999,99 × 10¹³` menjadi `400`, bukan 500 dari `bigint out of range`.
- **Template anggaran.** Pernikahan: 14 pos estimasi 0 per sesi. Jenis lain: pos persentase di sesi `all`, `round(persen × target_budget ÷ 100)` ke Rp 1.000 (setengah naik), selisih ke `dana_darurat`; `target_budget` di bawah Rp 100.000 memberi semua 0. `replace_template` menghapus hanya pos `template` tanpa pembayaran, `confirm` wajib bila ada yang memuat pekerjaan pengguna, dan memberi peringatan `budget_link_lost`.
- **Tabungan.** Target = `COALESCE(savings_target, target_budget)` (ditulis lewat `PATCH /projects/:id` Menu 01). Milestone = sesi non-sistem bertanggal, dana dialokasikan berurutan; rekomendasi bulanan memakai jam sebagai parameter (`4.584.000` per bulan pada 4 dan 21 Okt 2026, diuji di keduanya dan pada H-150, H-151, H-30, H-31, H-1, Hari-H, H+5).
- **Sinkron pagu.** `apply` dengan `sync_target_budget` / `sync_target_savings` menulis `planner_projects.target_budget` dan `savings_target` dengan SQL langsung (pengecualian ketiga di kontrak bab 1.1).
- **Ekspor `.xlsx`** (`excelize`, sudah ada dari menu 02): lembar `Rekap`, `Anggaran`, `Tabungan`; angka sebagai bilangan berformat `"Rp" #,##0`; teks pengguna berawalan `'` bila diawali `=`, `+`, `-`, `@` (tampak di sel, seperti menu 02); 10 ekspor per menit per pengguna; proyek arsip boleh mengekspor.
- **Proyek arsip** baca-saja (`409 project_read_only`) di semua 32 rute kecuali `GET` dan ekspor (diuji satu per satu).

## Dependensi

Tidak ada yang baru. `github.com/goccy/go-yaml v1.19.2` sudah ada di graf modul sebagai dependensi tidak langsung; ia kini dipakai langsung oleh tes kontrak (`contract_test.go`), sehingga `go mod tidy` memindahkannya ke blok `require` langsung. `go.sum` tidak berubah dan tidak ada kode produksi yang mengimpornya.

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Semuanya sudah dikoreksi di [BE.md](../../APP_SPEC/PLANNER_SPEC/03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) (butir 18–28) dan [CHANGELOG spesifikasi](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **Template persentase tidak punya kategori.** `planner_budget_items.category` bertipe `NOT NULL` tetapi tabel TEMPLATE_EVENT 15.1 hanya memuat kunci, nama, dan persen. Ditambahkan kolom **Kategori** (59 baris) di tabel itu dan di kode; dijaga `TestTemplatesMatchSpec`.
2. **Template persentase untuk `target_budget` kecil.** Spesifikasi hanya menolak preset di bawah Rp 100.000; wizard dengan pagu Rp 50.000 akan membuat selisih pembulatan negatif dan melanggar `CHECK (initial_estimate >= 0)` (500). Template kini memberi semua estimasi 0 di bawah Rp 100.000.
3. **`load-preset` menulis `total_target` dan `preset_key`** skenario (fixed: total paket; percent: total yang dikirim atau yang sudah ada). Spesifikasi hanya menyebut "mengganti semua item"; tanpa ini skenario 20 Jt berisi paket 30 Jt.
4. **Dua item tanpa kunci bernama sama** berpasangan dengan pos yang sama; item kedua dibuat sebagai pos sendiri (`UPDATE … FROM` dengan target ganda tidak terdefinisi).
5. **`mode` kosong**: `apply` = `smart_adjust` (tidak merusak), `apply-template` = `append`. `scenario_id`, `a`, dan `b` yang kosong `400 required` (bukan 404).
6. **404 mendahului 400** pada `PUT /savings/:id` (id proyek lain tidak boleh terbaca sebagai "kontributor tidak valid"); batas tiga skenario diperiksa sebelum sesi aktif divalidasi.
7. **Overlay mengisi estimasi, bukan hanya pembayaran:** dataset PDF memuat estimasi 74.150.000 yang tidak dimiliki template (estimasi 0).
8. **Milestone bertarget 0 berstatus `is_secured: true`** (0 dari 0). Proyek baru yang sesinya bertanggal Hari-H menampilkan dua milestone tanpa dana; apakah antarmuka menyembunyikannya adalah keputusan FE.
9. **Pembatas laju disalin untuk keempat kalinya,** dengan satu perbaikan: jam sapuan dimulai dari jam pembatas sendiri (`l.now`), bukan `time.Now()`. Salinan di `planner` dan `plannertimeline` memakai `time.Now()` sehingga tes `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow` **gagal setelah sekitar pukul 10.05 UTC 2026-10-21** (jam tetap tes sudah lewat dari jam dinding). Tes itu kini mengatur `l.lastSweep = now`; kode kedua modul itu tidak diubah.
10. **Kunci duplikat pada item skenario** → `409 conflict` dengan pesan `item_key: already used in this session` (spesifikasi: `409 conflict` saja); vendor yang sudah tertaut → `409 vendor_already_linked`.

## Pengujian

Dijalankan 2026-10-21 pada PostgreSQL 16.2 lokal, database sementara `planner_fin_impl_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 ./...` | 261 tes tingkat atas lulus, 0 gagal, 0 di-skip (177 sebelumnya + 83 milik modul ini + 1 tes registry peran) |
| `-race` pada `plannerfinance/...`, `planner/...`, `plannertimeline/...`, `internal/app` (kode akhir) | lulus, tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`) | 91,3% pernyataan |
| `go vet ./...`, `GOOS=linux go build ./cmd/api`, `gofmt -l` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Kontrak | `TestOpenAPIDocumentsEveryFinanceRoute`: 32 rute terdaftar = 32 operasi bertag `Planner Finance`; `TestOpenAPISchemasMatchTheGoDTOs`: 31 skema sama dengan tag JSON struct Go (kunci, dan untuk respons kewajiban hadir); dibuktikan gagal bila satu kunci diganti nama |
| Mutasi | 23 mutasi kode produksi (urutan aktivasi, kunci proyek, `GREATEST` pada sisa, pos rollup dibayar atau dianggap pembaruan, jumlah `full`, kunci baris pembayaran, netralisasi kunci, skema URL, pembulatan harga satuan dan persen, filter matriks, konfirmasi `replace_template`, batas 300, jendela milestone, pembulatan bulan, urutan 404, `item_key` pada PUT, `dp_amount` saat Smart Adjust, `RESET`, skenario aktif contoh, hapus kontributor bersetoran, awalan rumus) semuanya digagalkan oleh tes. Awalnya dua selamat: satu karena skrip saya memilih tes sasaran yang salah (dan satu mutasi tidak terkompilasi), satu **celah tes sungguhan** (filter `rollup_source IS NULL` pada kolom `dp` matriks tidak terjaga) yang ditutup dengan pembayaran lama sebelum rollup dan pengecekan `paid = dp + term_1 + term_2 + rollup_paid` |
| Universalitas | `TestEveryEventTypeRunsTheWholeFinanceFlow`: sembilan jenis acara dari registry menjalani alur yang sama lewat router, PostgreSQL, dan wiring asli: template wizard, preset pertama lalu Apply (preset persentase di atas template persentase tidak membuat pos baru), kontributor dengan peran jenis itu (peran lain ditolak), setoran, KPI tabungan, pembayaran, cashflow, widget dashboard, ekspor, dan sinkron biaya dokumen Menu 02. Dibuktikan gagal bila pemasangan item skenario ke pos dirusak (sembilan jenis gagal) |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, pembatas laju, dan kontrak berkas lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh dari PDF Hal. 4–7 lewat seluruh tumpukan (matriks 74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000, status 7/7/0, per sesi, kas bebas 13.950.000, defisit 26.650.000, milestone 7.150.000 / 600.000 / 39.750.000 dari 66.400.000, rekomendasi 4.584.000 dan 2.292.000, 22 setoran 30.000.000 / 17.500.000, tiga skenario, `unallocated` 850.000, widget dashboard); data template dan dataset identik dengan tabel spesifikasi; semua total persen jenis lain dari Rp 100.000 sampai Rp 400.000 tanpa selisih negatif; Smart Adjust 9 diperbarui / 1 dibuat / 5 tak tersentuh / total 75.000.000, ulang → nol perubahan, sama dengan `apply-preview`; berpindah antar tiga skenario sepuluh kali tanpa `23505`; `clean_overwrite` (15 pos, Rp 500.000 hilang, sisa 10 pos); batas 3 / 100 / 300 / 20 / 2.000 di bawah balapan; isolasi di 32 rute dan id proyek lain; arsip baca-saja; ekspor `.xlsx` dibaca kembali (angka sebagai bilangan, tanpa sel berawalan `=`).

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- Frontend menu 03: tidak disentuh. Tipe TypeScript di [FE.md](../../APP_SPEC/PLANNER_SPEC/03_KEUANGAN/FE.md) dan fixture-nya masih dibangun dari DDL dan SQL spesifikasi, belum diganti dengan salinan respons HTTP backend ini.
- Menu 04 belum ada: pos rollup (`rollup_source = 'shopping'`), tautan vendor (`vendor_id`), dan `lock-deal` diuji dengan baris yang disisipkan tes ke `planner_shopping_items` dan `planner_vendors`, bukan lewat modulnya. Overlay `finance` harus tetap berjalan **sebelum** overlay vendor.
- Berkas `.xlsx` dibaca ulang dengan `excelize` tetapi belum dibuka di Excel, Numbers, atau LibreOffice (tata letak, lebar kolom, format `"Rp"`).
- Kegagalan applier `budget` yang menggagalkan pembuatan proyek (atomisitas) tidak diuji sendiri; mekanismenya milik menu 01 dan sudah diuji dengan applier jadwal.
- Pembayaran serentak dengan `apply` pada pos yang sama; `PUT` pos serentak.
- Image Docker, PostgreSQL 18 (versi Compose), Nginx di depan rute baru (lokasi `/api/v1/` meneruskan semua path; hanya dibaca), beberapa replika API (pembatas laju ekspor per proses).
- Latensi `GET /savings`, `/budget/matrix`, dan dashboard pada data jauh di atas Proyek Contoh; kueri dijalankan pada audit spesifikasi dengan data PDF saja.
