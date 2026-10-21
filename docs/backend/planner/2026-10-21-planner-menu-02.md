# Planner menu 02: timeline, tugas, dan dokumen legalitas

Tanggal: 2026-10-21. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/BE.md](../../APP_SPEC/PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/BE.md), [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md), dan [TEMPLATE_EVENT](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner Timeline`). Dokumen menu 01 yang menjadi induknya: [planner-menu-01](2026-10-21-planner-menu-01.md).

Modul `plannertimeline` memiliki **jadwal/tugas persiapan** (`planner_tasks`) dan **dokumen legalitas/perizinan** (`planner_admin_docs`). Ia tidak diimpor oleh modul `planner`; ia mendaftarkan diri ke sana lewat titik sambung yang disediakan menu 01 (`SummaryProvider`, `TemplateApplier`, `SampleOverlay`), sehingga widget Timeline dan Legalitas di dashboard terisi, wizard bisa meminta template jadwal dan berkas, dan Proyek Contoh berisi data PDF.

## Yang ada

| Area | Isi |
|---|---|
| Tugas | `GET/POST /projects/:id/tasks` (saring `status`, `category`, `scope`, `phase`; ringkasan seluruh proyek), `PATCH/DELETE …/tasks/:task_id`, `PATCH …/status`, `POST …/tasks/reorder`, `POST …/tasks/shift`, `POST …/tasks/generate-defaults` |
| Ekspor | `GET …/tasks/export-spreadsheet`: `.xlsx` berisi semua tugas dan matriks Gantt mingguan, 10/menit/pengguna |
| Dokumen | `GET/POST …/admin-docs`, `PATCH/DELETE …/admin-docs/:doc_id`, `PATCH …/status`, `POST …/admin-docs/reorder`, `POST …/admin-docs/apply-template`, `POST …/admin-docs/:doc_id/sync-budget` |
| Registry | `GET /event-types` kini memuat `admin_subjects` dan `admin_guide` per jenis acara (`AdminLabel` hanya dipakai server) |
| Wizard | applier `timeline` (preset `wedding_pdf_44`, `wedding_essential_22`, `general`, tugas Adicara) dan `admin_docs`; `templates_available` memuatnya |
| Dashboard | provider `timeline` dan `admin_docs` mengisi widget 1 dan 5 |
| Proyek Contoh | overlay `timeline` (status PDF 15/29/6, sesi Lamaran bertanggal tugas Lamaran) dan `admin_docs` (9 berkas selesai) |

Tidak ada migrasi baru: tabel ada sejak `007_000_planner_timeline.sql` (dirilis bersama 005–012). Tidak ada perubahan frontend.

## Struktur kode

```text
internal/modules/plannertimeline/
  domain/       entity.go (tugas, dokumen, permintaan), plan.go (penskalaan jadwal), template.go + template_wedding.go +
                template_general.go (data template), repository.go (antarmuka dan galat)
  usecase/      service.go, validate.go, task.go, generate.go, doc.go, budget.go, integration.go (provider, applier, overlay)
  repository/   postgres.go
  delivery/http router.go, handler.go, dto.go, export.go (.xlsx), ratelimit.go
internal/ginutil/slug.go          Slug untuk nama berkas unduhan
internal/modules/planner/domain/eventtype.go   + Option, Guide, AdminSubjects, AdminGuide, AdminLabel
```

Mengikuti menu 01: usecase tidak mengimpor `pgx`; transaksi dibawa lewat `context` oleh `TxManager` yang **sama** dengan milik `planner` (satu instance dibuat di `app.go`), sehingga applier berjalan di dalam transaksi pembuatan proyek. Galat repository memakai sentinel `planner/domain` (`ErrNotFound`, `ErrInvalidScope`, `ErrConflict`) ditambah `ErrInvalidVendor`.

`data template` (`template_wedding.go`, `template_general.go`) dibangkitkan dari tabel spesifikasi dengan skrip sekali jalan, bukan diketik; `TestTemplatesMatchSpec` membaca tabel itu lagi dan membandingkannya, jadi mengubah salah satu tanpa yang lain menggagalkan tes.

## Perilaku yang perlu diketahui

- **Satu modul, dua kunci.** `SummaryProvider`, `TemplateApplier`, dan `SampleOverlay` masing-masing punya satu `Key()`, sedangkan modul ini mengisi `timeline` dan `admin_docs`; masing-masing diwakili tipe kecil yang membungkus `Service` (`TimelineProvider()`, `AdminDocsProvider()`, `TimelineApplier()`, dan seterusnya).
- **Penskalaan jadwal.** Offset template (hari sebelum Hari-H pada horizon `R`) diskalakan ke waktu yang tersisa: `k = LeadTimeDays/R`, dikalikan `avail/LeadTimeDays` bila waktu kurang, dibulatkan setengah naik, dan tidak ada tugas sebelum `max(prep_start_date, hari ini)` di zona proyek. Kurang dari 3 hari: semua tugas di `[hari ini, Hari-H]` (`lead_time_too_short`). Proyek Contoh (dibuat 4 Okt 2026, Hari-H 28 Mar 2027) menjaga durasi PDF dan menaruh Lamaran pada **15 Okt 2026**.
- **Mode.** `append` menyisipkan tugas yang `template_key`-nya belum ada (idempoten, `ON CONFLICT DO NOTHING`); `replace_template` menghapus tugas ber-`template_key` lalu menyisipkan yang baru, tugas kustom tetap; `replace_all` menghapus semuanya. `confirm` wajib bila ada kemajuan yang hilang. Satu panggilan semua-atau-tidak, termasuk penghapusannya.
- **Kunci proyek.** `POST /tasks`, `POST /admin-docs`, `PATCH` tugas dan dokumen, `generate-defaults`, `apply-template`, `reorder`, dan `shift` mengambil `SELECT … FOR UPDATE` pada baris proyek di awal transaksi (`LockProject`). Hitungan batas, `max(sort_order)+1`, dan pemeriksaan "daftar lengkap" tidak berbalapan: enam `POST /tasks` serentak pada 499 tugas menghasilkan tepat satu `201`.
- **Sinkron biaya** (`POST …/admin-docs/:doc_id/sync-budget`) menulis ke `planner_budget_items` milik Menu 03 dengan SQL langsung, pengecualian kedua di kontrak bab 1.1. Baris dokumen dikunci `FOR UPDATE` karena tabel anggaran tidak punya indeks unik pada `admin_doc_id`: 8 permintaan serentak membuat satu pos. Idempoten, tidak menimpa estimasi yang terisi (`estimate_differs`), nama dipotong 150 karakter, hanya satu kali (mengubah biaya dokumen sesudahnya tidak mengubah pos).
- **Validasi** `<field>: <alasan>` seperti menu 01; batas 500 tugas, 200 berkas, teks 255/60/100/2.000 karakter, jendela tanggal `Hari-H − 1095 hari` s/d `akhir acara + 365 hari`. `notes` boleh berisi baris baru (pembantu `cleanNote`; `cleanText` menu 01 menolaknya).
- **Ekspor.** Kolom No, Tugas, Rangkaian Acara, Mulai, Selesai, Durasi, Status, lalu satu kolom per pekan (Senin–Minggu) yang terisi menurut status; semua tugas proyek, bukan hasil saringan. Teks pengguna ditulis sebagai string (tidak pernah formula) dan, sesuai kontrak, diberi awalan `'` bila diawali `=`, `+`, `-`, `@`; awalan itu ditulis sebagai karakter dan karena itu **tampak** di sel.
- **Proyek arsip** baca-saja (`409 project_read_only`); `GET` dan ekspor tetap boleh.

## Dependensi baru

`github.com/xuri/excelize/v2 v2.11.0` untuk ekspor. `go.mod` bertambah satu modul langsung dan lima tidak langsung (`richardlehane/mscfb`, `richardlehane/msoleps`, `tiendc/go-deepcopy`, `xuri/efp`, `xuri/nfp`); `golang.org/x/crypto`, `x/net`, dan `x/text` sudah lebih baru di backend dan tidak turun. Ditambahkan atas permintaan pemilik untuk segera mengimplementasikan (kontrak bab 1.4 menyebutnya "disetujui saat implementasi") dan **disetujui pemilik pada 2026-10-21**; hanya paket `delivery/http` yang mengimpornya.

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Yang menyangkut spesifikasi sudah dikoreksi di [BE.md](../../APP_SPEC/PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk) (butir 15–19) dan [CHANGELOG spesifikasi](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **`PATCH` memakai kunci proyek.** Spesifikasi awal: penulisan satu baris tidak butuh kunci. Karena `PATCH` membaca, menggabung, lalu menulis semua kolom, tanpa kunci dua tab yang mengubah kolom berbeda saling menimpa.
2. **`preset` wajib, `mode` bawaan `append`** pada `generate-defaults`; tidak disebut sebelumnya.
3. **Satu peringatan jadwal per panggilan:** `lead_time_too_short` menggantikan `lead_time_compressed`; `count` = tugas yang dibuat; tanpa tugas baru tidak ada peringatan.
4. **`subject_role` kosong** = `bersama` (pernikahan) atau `umum` (jenis lain); catatan tabel §5.3 disimpan sebagai `notes` berkas template.
5. **Template di `plannertimeline/domain`,** bukan `planner/template/` seperti tertulis di TEMPLATE_EVENT bab 16: data milik modul ini dan tidak dipakai modul lain.
6. **Tugas template non-pernikahan diurutkan kronologis** (offset mulai menurun), bukan 15 tugas umum lalu tiga tugas khusus; tidak ditentukan sebelumnya (BE.md §5.5).
7. **Offset Seminar "Undang dan konfirmasi pembicara"** H-70…H-56 melampaui `R = 60`; diubah menjadi H-58…H-50 (TEMPLATE_EVENT bab 13) dan dijaga `TestEveryTemplateOffsetIsWithinItsHorizon`.
8. **`ginutil.Slug`** melipat aksen; spesifikasi hanya menyebut "selain `[a-z0-9]` menjadi `-`". Judul proyek tidak bisa berisi karakter kontrol (CHECK di basis data).
9. **Awalan `'` tampak di sel Excel** (`excelize` tidak pernah menulis formula dari teks dan tidak punya atribut `quotePrefix`); keputusan mempertahankan atau mencabutnya ada di BE.md bab 8 butir 12.
10. **Pembatas laju** disalin dari `planner/delivery/http/ratelimit.go` (tipe itu tidak diekspor); modul ketiga yang memakai pola yang sama (auth, planner, timeline). Mengekstraknya ke paket bersama adalah refaktor terpisah yang menyentuh menu 01.

## Pengujian

Dijalankan 2026-10-21 pada PostgreSQL 16.2 lokal, database sementara `planner_tl_scratch_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 ./...` | 177 tes tingkat atas lulus, 0 gagal, 0 di-skip (127 sebelumnya + 50 milik modul ini) |
| `-race` pada `plannertimeline/...` dan `ginutil` | lulus |
| Cakupan modul (`-coverpkg`) | 87,8% pernyataan |
| `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./cmd/api` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru; 17 rute = 17 operasi |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, dan ekspor lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh (50 tugas 15/29/6, tiga berdurasi 2 hari, Lamaran 15 Okt, 15 berkas 9/6, widget `50/15/29/0` dan `9/15`); data template identik dengan tabel spesifikasi; semua offset di `0…R`; wizard atomik (penolakan tidak meninggalkan proyek); `append` dua kali tidak menggandakan, `confirm` hanya bila ada kemajuan; sinkron biaya delapan kasus dan klik ganda; batas 500 tugas di bawah balapan; `overdue` memakai zona proyek (Jakarta vs Jayapura pada instan yang sama) dan angka audit 4 Okt, 1 Nov, 4 Nov; isolasi di 17 endpoint; ekspor `.xlsx` dibaca kembali (tanpa elemen `<f>`, 50 × 26 sel Gantt sama dengan perhitungan hari demi hari).

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- Frontend menu 02 (halaman Timeline & Tugas, BFF, unduhan): tidak disentuh. Tipe TypeScript di [FE.md](../../APP_SPEC/PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/FE.md) sudah dibandingkan dengan respons backend ini (120 panggilan, 821 objek, nol selisih; [FE bab 11.5](../../APP_SPEC/PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/FE.md#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang)); yang belum teramati: bentuk `vendor_*` pada tugas (menunggu Menu 04) dan `sync-budget` dengan `linked: true` (menunggu Menu 03). Satu perilaku yang ditemukan dicatat di [BE bab 8, butir 20](../../APP_SPEC/PLANNER_SPEC/02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk): `PATCH` tidak menyalin `phase_name` saat `phase_order` berubah.
- Berkas `.xlsx` belum dibuka di Excel, Numbers, atau LibreOffice.
- Image Docker, PostgreSQL 18 (versi Compose), dan Nginx di depan rute baru (lokasi `/api/v1/` meneruskan semua path ke backend; hanya dibaca).
- Beberapa replika API: pembatas laju ekspor dihitung per proses.
- Latensi `GET /tasks` pada 500 tugas lewat HTTP dan dashboard dengan dua provider baru: SQL-nya diukur 0,2 ms, jalur HTTP penuh tidak.
- Menu 03 belum ada: sinkron biaya diuji terhadap tabel `planner_budget_items` yang dibuat migrasi 008, dengan baris `biaya_kua` yang disisipkan tes, bukan yang akan dibuat template anggaran.
