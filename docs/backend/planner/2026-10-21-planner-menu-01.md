# Planner menu 01: proyek, sesi, registry, dan dashboard

Tanggal: 2026-10-21. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/01_DASHBOARD/BE.md](../../APP_SPEC/PLANNER_SPEC/01_DASHBOARD/BE.md) dan [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner`).

Modul `planner` adalah fondasi enam menu Planner: ia memiliki proyek (`planner_projects`), master sesi bersama (`planner_event_sessions`), registry sembilan jenis acara, dan agregator dashboard. Modul 02–06 (timeline, keuangan, vendor, tamu, rundown) dibangun sesudahnya (2026-10-21 sampai 2026-10-31; dokumen masing-masing ada di folder ini); menu 01 sengaja dibangun tanpa mengimpor mereka dan menyediakan titik sambungnya (bagian [Menyambungkan modul lain](#menyambungkan-modul-lain)).

## Yang ada

| Area | Isi |
|---|---|
| Migrasi | `005_000_planner_core.sql` (proyek, sesi, pemicu zona waktu) dan `006`–`012` (tabel modul 02–06, lihat [alasannya](#migrasi-005012)) |
| `GET /v1/planner/event-types` | Registry 9 jenis acara + `templates_available`; `Cache-Control: public, max-age=3600` |
| Proyek | daftar kartu Hub / `view=lite` (project switcher), `summary`, buat (atomik, `client_token`), `sample`, detail (+ `shiftable_tasks`), ubah (PATCH, geser jadwal), status, hapus (`?confirm=`) |
| Sesi | daftar (+ `usage`), buat, ganti penuh (PUT), urut ulang, hapus dengan pemindahan data |
| Dashboard | `GET /projects/:id/dashboard`, enam widget lewat `SummaryProvider` |
| Pengaman | `ProjectGuard` (404 untuk proyek orang lain dan id non-UUID), `WriteGuard` (arsip baca-saja), batas 10 pembuatan/menit dan 60 dashboard/menit per pengguna |

Tidak ada perubahan frontend. `PUT /projects/:id/invitation` sengaja **tidak terdaftar** sampai modul Undangan Digital ada (diuji: menjawab 404).

## Struktur kode

```text
internal/modules/planner/
  domain/       entitas, registry (eventtype.go), fase acara (phase.go), kunci sesi (scopekey.go),
                antarmuka Repository / TxManager / SummaryProvider / TemplateApplier / SampleOverlay
  usecase/      project.go, session.go, dashboard.go, sample.go, validate.go, metadata.go
  repository/   postgres.go (SQL + TxManager)
  delivery/http handler.go, router.go, guard.go, ratelimit.go, dto.go
internal/db     Connect(ctx, url, maxConns), Queryable, WithTx, TxFrom, Querier
```

Usecase tidak mengimpor `pgx`. Transaksi dibawa lewat `context`: `TxManager.InTx` menaruh `pgx.Tx` di context dan repository mengambilnya dengan `db.Querier(ctx, pool)`. `InTx` bersarang bergabung ke transaksi luar. Galat PostgreSQL yang relevan diterjemahkan di repository ke galat domain (`ErrSessionNameTaken`, `ErrScopeKeyTaken`, `ErrInvalidTimezone`, `ErrConflict`, `ErrInvalidScope`); kegagalan `COMMIT` karena FK yang ditunda (`23503`) menjadi `ErrConflict` → `409 conflict`.

## Perilaku yang perlu diketahui

- **Pembuatan proyek atomik.** Baris proyek, sesi bawaan (selalu, termasuk sesi sistem `all`), dan template yang diminta ditulis dalam satu transaksi. Template berjalan berurutan `timeline → admin_docs → budget → shopping → rundown → playlist` (pos anggaran harus ada sebelum belanja yang digulung ke dalamnya). Satu kegagalan membatalkan semuanya (diuji dengan applier palsu yang gagal di tengah).
- **`client_token`.** Pengiriman ulang dengan token yang sama menjawab `200 {created:false}` dengan proyek asli dan tidak menjalankan template lagi, juga saat pengguna sudah di batas proyek (token dicek sebelum batas). Delapan permintaan serentak menghasilkan tepat satu `201` dan tujuh `200` (diuji, begitu juga untuk `POST /projects/sample`).
- **Hari ini dan fase dihitung di zona waktu proyek.** `ComputePhase` (Go) dan kueri Hub (SQL) memakai satu definisi; tes membandingkan keduanya di 5 zona × acara 1 dan 3 hari pada setiap 30 menit selama 6 hari (> 1.000 perbandingan, nol selisih). Pada 06:47 WIB (23:47 UTC hari sebelumnya) `prep_start_date` tetap tanggal WIB.
- **Ubah tanggal.** `event_end_date` ikut bergeser (durasi tetap) kecuali dikirim; sesi yang bertanggal Hari-H lama ikut pindah; tugas template yang belum selesai hanya bergeser bila `shift_tasks: true`, dalam transaksi yang sama (PATCH yang gagal validasi tidak menggeser apa pun, diuji). Notifikasi: `warnings[]` (`tasks_overdue_after_shift`) dan `hints[]` (`tasks_not_shifted` dengan `shift_days`, `sessions_dates_unchanged`).
- **`metadata` = JSON Merge Patch** terhadap kolom registry jenis acara: kunci asing `400 invalid_metadata: <kunci>: unknown key…`, kolom wajib harus tetap ada setelah penggabungan, `null` menghapus.
- **Hapus sesi.** Di bawah `SELECT … FOR UPDATE` pada proyek: semua baris yang menunjuk sesi dipindah ke `move_to` (default `all`), `active_sub_events` skenario dibersihkan, `item_key` yang bentrok dinetralkan menjadi `NULL` (mis. `catering` di Lamaran dan Resepsi), total anggaran tetap. Sesi dengan agenda rundown butuh `delete_agendas=true`, selain itu `409 session_in_use` dengan `error.details.usage`.
- **Dashboard.** Provider berjalan paralel (maks 3) dalam batas 3 detik; yang gagal, panic, atau lambat hanya mengubah widgetnya menjadi `{"error":"internal"|"timeout"}`. Provider yang mengabaikan `ctx` tidak menahan respons melewati batas. Widget tanpa provider `null`.
- **Batas.** Judul 1–255 karakter tanpa karakter kontrol; uang 0…10¹³; Hari-H tidak lebih awal dari hari ini (zona proyek) dan paling jauh 1.826 hari; `event_end_date` ≤ Hari-H + 30 hari; ≤ 20 sesi non-sistem; body JSON ≤ 64 KiB; `PLANNER_MAX_PROJECTS` (default 100) proyek nyata per pengguna.

## Menyambungkan modul lain

Dilakukan di `internal/app/app.go` setelah `plannerSvc` dibuat (komentar di sana menandai tempatnya):

```go
plannerSvc.RegisterSummary(timelineSvc, financeSvc, vendorSvc, guestSvc, rundownSvc) // widget dashboard
plannerSvc.RegisterTemplateAppliers(timelineSvc, financeSvc, vendorSvc, rundownSvc)  // template wizard
plannerSvc.RegisterSampleOverlays(financeSvc, guestSvc, ...)                          // dataset PDF di proyek contoh
```

- `SummaryProvider.Key()` ∈ `timeline | finance | vendors | guests | admin_docs | rundown`; hasilnya memakai tipe di `domain/summary.go` (JSON `snake_case`, diuji).
- `TemplateApplier.Key()` ∈ `timeline | admin_docs | budget | shopping | rundown | playlist`; `Apply` berjalan di dalam transaksi pembuatan.
- Kunci tak dikenal atau terdaftar dua kali membuat **start gagal (panic)**: itu kesalahan perakitan, bukan widget kosong.
- Selama sebuah applier belum terdaftar, templatenya **tidak ditawarkan** `GET /event-types` dan diminta lewat `POST /projects` menjadi `400 invalid_template` (bukan 500).

## Migrasi 005–012

`005` adalah milik menu 01. `006`–`012` (vendor, timeline, keuangan, belanja, view `planner_budget_lines`, tamu, rundown) ikut dirilis karena **SQL menu 01 sendiri membaca dan mengubah tabel-tabel itu**: agregat kartu Hub (`planner_budget_lines`, `planner_tasks`, `planner_guests`), `usage` sesi (7 tabel), pemindahan data saat hapus sesi, dan penggeseran tugas template. Kontrak bersama (bab 9) memang membolehkan migrasi 005–012 dirilis sekaligus dengan tabel kosong untuk modul yang belum dibangun.

DDL 006–012 **disalin otomatis** dari blok ```sql di `BE.md` masing-masing modul (bukan diketik ulang), jadi sama persis dengan yang sudah diuji di spesifikasi. Tidak ada kode Go untuk modul 02–06.

Akibat deploy: `MIGRATE_ON_START=true` di produksi, jadi merge ke `main` membuat 14 tabel dan 1 view baru di database produksi. Semuanya tambahan (tidak menyentuh `users`, `refresh_sessions`, `m_role`), dan `Down` 005–012 diuji (lihat bawah). Mengikuti [catatan dev-stage](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md): sebelum rilis pertama, migrasi ini boleh diedit di tempat.

## Konfigurasi baru

| Variabel | Default | Keterangan |
|---|---|---|
| `DB_MAX_CONNS` | `10` | Ukuran pool (1–1000). `db.Connect` kini menerima ukuran pool. |
| `PLANNER_MAX_PROJECTS` | `100` | Batas proyek nyata per pengguna. |

Keduanya diteruskan Compose (dev dan prod, kosong = default), tercantum di `.env.example`, `backend/.env.example`, [backend/README.md](../../../backend/README.md#environment), dan [deployment](../../deployment.md#environment). `cmd/api` kini mengimpor `time/tzdata` (image runtime `alpine` tidak punya zoneinfo; tanpa itu setiap `LoadLocation("Asia/Jakarta")` gagal). `go list -deps ./cmd/api` memuat `time/tzdata`; image Docker sebenarnya belum dibangun (Docker tidak ada).

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Setiap butir ditemukan saat membangun atau mengujinya; yang menyangkut spesifikasi sudah dikoreksi di [PLANNER_SPEC/CHANGELOG.md](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **Kunci sesi bawaan eksplisit.** Spesifikasi 01 BE §4.1 menyuruh `scope_key` bawaan dibuat dari nama lewat aturan sesi buatan pengguna, tetapi aturan itu menghasilkan `akad_nikah` untuk "Akad Nikah", sedangkan seluruh spesifikasi lain (`02`, `03`, `04`, `06`, data emas) memakai `akad`. `SessionSeed` kini membawa `ScopeKey` sendiri; aturan penurunan hanya untuk sesi buatan pengguna.
2. **`TemplateSet`/registry dipersempit** ke yang dibutuhkan menu 01 (kunci, label, ikon, horizon, sesi, kolom metadata, `Labels`, template yang dimiliki). Kategori vendor/belanja, sisi dan kategori tamu, peran kontributor, subjek dokumen, panduan birokrasi, dan *moment tag* ditambahkan oleh modulnya masing-masing (datanya ada di TEMPLATE_EVENT; panduan 4 langkah non-pernikahan **tidak** ada teksnya di spesifikasi).
3. **Antarmuka `SampleOverlay` baru.** Spesifikasi menyebut "overlay dataset PDF" tanpa antarmuka. Sampai modul 02–06 ada, Proyek Contoh hanya berisi proyek, sesi, dan metadata (tanpa tugas, anggaran, tamu); endpoint, idempotensi, dan balapan sudah benar.
4. **Waktu sebagai parameter.** Kueri Hub memakai `$3::timestamptz` (jam dari usecase) alih-alih `now()`, supaya batas tengah malam bisa diuji deterministik. Semantiknya sama.
5. **`db.Querier(ctx, fallback)`** menerima pool sebagai argumen (spesifikasi menulis `db.Querier(ctx)`, yang tidak mungkin tanpa pool global). `TxManager` ada di `planner/repository`, bukan `internal/db`, agar pemetaan galat commit PostgreSQL tidak bocor ke usecase.
6. **`warnings` dan `hints` dipisah** pada respons PATCH seperti 01 BE §4.3; kontrak bersama bab 6 menyebut ketiganya "warnings".
7. **Batas 30 hari `event_end_date` juga berlaku di PATCH** (spesifikasi hanya menuliskannya untuk POST); tanpa itu bisa dilewati lewat PATCH.
8. **`status` ikut menyaring `view=lite`** (SQL ringan di spesifikasi tidak punya filter itu). Tanpa parameter, perilakunya sama.
9. **`client_token` dicek sebelum batas proyek**, supaya percobaan ulang saat sudah di batas tetap mendapat proyeknya.
10. **Ringkasan portofolio mengecualikan contoh dari ketiga hitungan** (aktif, selesai, arsip); spesifikasi hanya menyebut aktif secara eksplisit.
11. **Tambahan kecil yang spesifikasi diam soal itu:** `apierr.Error.Details` (untuk `session_in_use`), batas body 64 KiB, `skipped_vendor_categories` divalidasi `^[a-z0-9_]{1,40}$` dan maks 50, teks metadata opsional yang kosong dianggap tidak dikirim, karakter kontrol ditolak di semua teks, batas 1.000.000 untuk kolom jumlah (`graduate_count`, `expected_participants`), id URL hanya UUID kanonik 36 karakter (bentuk `urn:uuid:`/`{…}` yang diterima `uuid.Parse` ditolak 404).

## Pengujian

Dijalankan 2026-10-21 pada PostgreSQL 16.2 lokal, database sementara `planner_scratch_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| Seluruh suite `go test -p 1 -count=1 ./...` | 127 tes tingkat atas lulus, 0 gagal, 0 skip (termasuk tes auth dan migrate yang sudah ada) |
| `-race` pada `planner/...`, `db`, `config` | lulus |
| Tanpa `TEST_DATABASE_URL` (mode CI) | lulus; 33 tes DB planner di-skip (tes DB auth dan migrate juga, seperti sebelumnya) |
| Cakupan (`-cover`) | delivery 89,7%, domain 88,4%, repository 90,1%, usecase 89,9%, db 85,2% |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Build `GOOS=linux CGO_ENABLED=0` | berhasil |

Yang dibuktikan (pilihan): fase SQL = Go di batas tengah malam; pindah sesi dengan `item_key` bentrok menjaga total (46.050.000 sebelum dan sesudah); hapus proyek dengan skenario, kontributor, belanja, dan rundown tanpa baris yatim, juga lewat hapus akun; hapus sesi yang masih dipakai ditolak saat commit; pemicu zona waktu menolak `Asia/Nowhere`; `CHECK(event_type)` = kunci registry (tes membaca `pg_get_constraintdef`); `Down` 005–012 menghapus semua objek termasuk fungsi pemicu dan `Up` ulang bersih; isolasi: pengguna B mendapat 404 di setiap endpoint proyek/sesi milik A; rute statis `/projects/summary` dan `/projects/sample` berdampingan dengan `/projects/:id`; arsip menolak tulis (`project_read_only`) kecuali ubah status dan hapus; `POST` pembuatan dibatasi 10/menit dan per pengguna.

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- Frontend Planner (Hub, wizard, workspace, BFF `/dashboard/api/planner/*`): tidak disentuh.
- Modul 02–06 beserta provider, applier, dan overlay-nya; karena itu widget dashboard `null`, `templates_available` kosong, dan Proyek Contoh belum memenuhi tabel uji emas (anggaran 74.150.000 dst.). Uji emas dijalankan saat modulnya dibangun.
- Latensi: target p95 dashboard < 100 ms **belum diukur** (tidak ada provider untuk diukur) dan kueri Hub belum diukur pada data 10× proyek contoh.
- Image Docker, PostgreSQL 18 (versi Compose), Nginx di depan `/api/v1/planner/*` (hanya dibaca: lokasi `/api/v1/` meneruskan semua path ke backend), dan rilis ke produksi.
- Pembatas laju disimpan di memori per proses; dengan beberapa replika API setiap replika menghitung sendiri.
- `Cache-Control: public` pada `GET /event-types` ikut ke respons yang butuh token; isinya statis dan tidak pribadi, tetapi cache bersama di depan backend perlu memperhatikannya.
