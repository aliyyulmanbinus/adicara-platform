# Kontrak Bersama Planner (Lintas Modul 01–06)

Dokumen ini adalah **satu-satunya tempat** aturan yang berlaku untuk semua modul Planner: posisi di arsitektur Adicara yang sudah ada, master sesi bersama, kebijakan template vs data contoh, konvensi data, kode error, dan matriks integrasi. README/FE/BE tiap modul merujuk ke sini dan tidak mengulang aturan ini.

> **Status:** per 2026-10-23 **backend menu 01, 02, 03, dan 04** sudah menjadi kode (modul `planner`, migrasi 005–012, `plannertimeline`, `plannerfinance`, dan `plannervendor`; dokumen [menu 01](../../../backend/planner/2026-10-21-planner-menu-01.md), [menu 02](../../../backend/planner/2026-10-21-planner-menu-02.md), [menu 03](../../../backend/planner/2026-10-21-planner-menu-03.md), dan [menu 04](../../../backend/planner/2026-10-23-planner-menu-04.md)). Menu 05–06 dan seluruh frontend `/dashboard/planner` masih berupa spesifikasi ([bab 10](#10-validasi-spesifikasi)). Tidak ada klaim "sudah berjalan" di folder ini; hasil validasi yang tercatat di sini hanya yang benar-benar dijalankan (lihat [Validasi](#10-validasi-spesifikasi)).

Data per jenis acara (registry, template non-pernikahan) ada di [TEMPLATE_EVENT.md](TEMPLATE_EVENT.md). Perubahan terhadap versi spesifikasi sebelumnya dicatat di [../CHANGELOG.md](../CHANGELOG.md).

---

## 1. Posisi di Arsitektur Adicara

Planner **tidak membuat arsitektur baru**. Ia memakai pola yang sudah ada ([docs/architecture.md](../../../architecture.md), [docs/api.md](../../../api.md), [docs/database.md](../../../database.md)):

```text
Browser → Nginx → Astro (SSR, /dashboard/**) ──(server-side, Bearer, /v1/planner/*)──→ Go API → PostgreSQL
                     └ /dashboard/api/planner/** = BFF same-origin untuk aksi interaktif browser
```

### 1.1 Backend (Go, Gin, Module-Based Clean Architecture)

Enam modul fungsional baru di `backend/internal/modules/`, masing-masing dengan lapisan `delivery/http → usecase → domain ← repository` seperti `auth` dan `profile`:

| Modul Go | Menu | Isi |
|---|---|---|
| `planner` | 01 | proyek, **master sesi**, registry event type, dashboard (agregator), `ProjectGuard`, proyek contoh |
| `plannertimeline` | 02 | tugas/jadwal dan dokumen legalitas |
| `plannerfinance` | 03 | skenario, kontributor, tabungan, pos anggaran, view `planner_budget_lines` |
| `plannervendor` | 04 | vendor dan daftar belanja |
| `plannerguest` | 05 | tamu, RSVP, check-in |
| `plannerrundown` | 06 | agenda dan playlist |

Aturan ketergantungan: modul 02–06 boleh mengimpor `planner/domain` dan `planner/delivery/http` (guard); `planner` **tidak** mengimpor modul lain. Dashboard mengumpulkan ringkasan lewat antarmuka (bab 7), bukan lewat import. Pengecualian yang disengaja adalah SQL, bukan import: kartu Hub, hitungan `usage` sesi, pemindahan data saat menghapus sesi, dan penggeseran tugas template membaca atau mengubah tabel milik modul 02–06 (`planner_tasks`, `planner_budget_*`, dst.). Pengecualian kedua yang disengaja: sinkron biaya dokumen (Menu 02, `POST …/admin-docs/:id/sync-budget`) menulis ke `planner_budget_items` milik Menu 03 dengan SQL langsung ([02 BE §4.5](../02_TIMELINE_DAN_TUGAS/BE.md#45-sinkron-biaya-dokumen-ke-anggaran)). Pengecualian ketiga: *Apply* skenario Menu 03 menulis `planner_projects.target_budget` dan `savings_target` dengan SQL langsung dalam transaksi yang sama dengan penulisan pos ([03 BE §3.2](../03_KEUANGAN/BE.md#32-menerapkan-skenario-smart-adjust)). Pengecualian keempat: `lock-deal` vendor dan gulungan belanja (Menu 04) menulis `planner_budget_items` milik Menu 03 (`vendor_id`, `initial_estimate`, baris baru, `rollup_source`) dengan SQL langsung dalam transaksi yang sama dengan penulisan vendor atau barang ([04 BE §1.3](../04_VENDOR_DAN_BELANJA/BE.md#13-tabel-dan-view-milik-modul-lain-yang-disentuh)). Pembacaan lintas modul yang sudah ada: Menu 02, 03, dan 06 `JOIN planner_vendors`, dan Menu 05 membaca `planner_shopping_items` untuk stok souvenir. Karena itu migrasi 005–012 dirilis bersama dan tabel modul yang belum dibangun hanya kosong.

Pemasangan di `internal/app/app.go` (komposisi akar yang sudah ada):

```go
plannerSvc := plannerusecase.New(plannerrepo.NewPostgresRepository(pool), plannerrepo.NewTxManager(pool), domain.DefaultRegistry() /* bab 3 */, plannerusecase.Options{MaxProjects: cfg.PlannerMaxProjects})
plannerhttp.RegisterRoutes(ginEngine, idn, plannerSvc)
timelinehttp.RegisterRoutes(ginEngine, idn, plannerSvc, timelineSvc)   // dst. untuk finance, vendor, guest, rundown
plannerSvc.RegisterSummary(timelineSvc, financeSvc, vendorSvc, guestSvc, rundownSvc)             // widget dashboard (bab 7)
plannerSvc.RegisterTemplateAppliers(timelineSvc, financeSvc, vendorSvc, rundownSvc)              // template wizard (bab 1.1.1)
plannerSvc.RegisterSampleOverlays(financeSvc, vendorSvc, guestSvc /* … */)                       // dataset PDF di Proyek Contoh (01 BE §4.5)

// app.go memakai http.ServeMux eksplisit; tanpa baris ini semua /v1/planner/* menjawab 404.
mux.Handle("/v1/planner/", ginEngine)
```

Setiap router modul memakai rantai middleware yang sama (pola `profile`):

```go
g := r.Group("/v1/planner/projects/:id")
g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(plannerSvc), plannerhttp.WriteGuard())
```

#### 1.1.1 Transaksi dan template lintas modul

Membuat proyek menyentuh banyak modul (sesi, jadwal, dokumen, pos, belanja, agenda, playlist) dan harus atomik. Usecase tidak boleh mengimpor `pgx`, jadi transaksi dibawa lewat `context`:

```go
// planner/domain
type TxManager interface {
    // InTx menjalankan fn dalam satu transaksi. ctx yang diteruskan membawa transaksi;
    // repository memakainya lewat db.Querier(ctx, pool) (pool bila tidak ada transaksi).
    InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Diimplementasikan tiap modul yang punya template; didaftarkan seperti SummaryProvider.
type TemplateApplier interface {
    Key() string // "timeline" | "admin_docs" | "budget" | "shopping" | "rundown" | "playlist"
    Apply(ctx context.Context, projectID uuid.UUID, req ApplyRequest) (ApplyResult, error) // dipanggil di dalam InTx
}
```

`internal/db` mendapat `WithTx`, `TxFrom`, `Querier`, dan antarmuka `Queryable` (diimplementasikan pool dan transaksi); kode auth yang ada tidak berubah. `TxManager` diimplementasikan di `planner/repository` (bukan `internal/db`) agar pemetaan galat commit PostgreSQL (`23503`/`23505`/serialisasi → `ErrConflict`) tidak bocor ke usecase; `InTx` bersarang bergabung ke transaksi luar. Template yang applier-nya belum terdaftar (modul belum dibangun) **tidak ditawarkan**: `GET /event-types` hanya mengisi `templates_available` dengan template yang punya applier, dan `POST /projects` meminta template lain dijawab `400 invalid_template`. Pada tahap pertama pembangunan (hanya modul `planner`) wizard otomatis melewati langkah template; tidak ada galat 500 karena modul belum ada.

- **Respons sukses** `{"data": ...}`; **error** `{"error": {"code","message"}}` lewat `ginutil.Error` + `apierr` (`message` Inggris, `code` stabil). Tidak ada envelope baru.
- **ID** dibuat aplikasi (`uuid.New()`), kolom `UUID PRIMARY KEY` tanpa default basis data, sama seperti `users`.
- Akses Planner sebagai layanan modular berbayar disiapkan lewat `apierr.Entitlement` (402 `entitlement_denied`). Sampai modul billing ada, pemeriksaannya no-op; kebijakan paket di luar lingkup dokumen ini.
- Backend **tidak** memakai WebSocket/SSE (lihat arsitektur). Data "live" memakai polling (bab 6).

### 1.2 Migrasi (goose, `backend/internal/migrate/sql/`)

Format file mengikuti `002_000_users.sql`: `-- +goose Up/Down`, `UUID PRIMARY KEY`, `TEXT` (bukan `VARCHAR`), `TIMESTAMPTZ NOT NULL DEFAULT now()`, `CHECK` untuk enum, nama indeks `<tabel>_<kolom>_idx`. Nomor berurutan menurut **ketergantungan foreign key**, tanpa siklus:

| File | Isi | Dokumen |
|---|---|---|
| `005_000_planner_core.sql` | `planner_projects`, `planner_event_sessions` | [01 BE](../01_DASHBOARD/BE.md#1-skema-database) |
| `006_000_planner_vendors.sql` | `planner_vendors` | [04 BE](../04_VENDOR_DAN_BELANJA/BE.md#1-skema-database) |
| `007_000_planner_timeline.sql` | `planner_tasks`, `planner_admin_docs` | [02 BE](../02_TIMELINE_DAN_TUGAS/BE.md#1-skema-database) |
| `008_000_planner_finance.sql` | skenario, kontributor, tabungan, `planner_budget_items` | [03 BE](../03_KEUANGAN/BE.md#1-skema-database) |
| `009_000_planner_shopping.sql` | `planner_shopping_items` | [04 BE](../04_VENDOR_DAN_BELANJA/BE.md#1-skema-database) |
| `010_000_planner_budget_lines_view.sql` | view `planner_budget_lines` | [03 BE](../03_KEUANGAN/BE.md#12-view-baca-planner_budget_lines) |
| `011_000_planner_guests.sql` | `planner_guests` | [05 BE](../05_BUKU_TAMU/BE.md#1-skema-database) |
| `012_000_planner_rundown.sql` | `planner_rundowns`, `planner_playlists` | [06 BE](../06_RUNDOWN_DAN_ACARA/BE.md#1-skema-database) |

Ketergantungan satu arah: `vendors ← tasks, budget_items, shopping_items, rundowns`; `admin_docs ← budget_items`; `budget_items ← shopping_items`. Tidak ada FK dua arah (siklus `budget_items ↔ vendors` tidak bisa dimigrasikan dalam urutan mana pun).

Semua DDL pada dokumen BE telah dijalankan di PostgreSQL 16 (bab 10). Migrasi 005–012 sudah ada sebagai file di `backend/internal/migrate/sql/` (006–012 disalin otomatis dari blok DDL spec modul masing-masing, karena SQL menu 01 membaca tabel-tabel itu). FK komposit dengan `ON DELETE SET NULL (kolom)` membutuhkan PostgreSQL ≥ 15 (Compose memakai 18). Migrasi 005 juga membuat fungsi/pemicu `planner_check_timezone` (blok `-- +goose StatementBegin/End`); `Down` menghapusnya.

### 1.3 Frontend (Astro SSR, TypeScript, CSS tokens)

- **Halaman** `src/pages/dashboard/planner/**` dirender on-demand dan otomatis dijaga `src/middleware.ts` (semua `/dashboard/**`). Data dibaca server-side dengan `backendFetch(path, { accessToken })` dari `Astro.locals`, sama seperti `dashboard/index.astro`. `backendFetch` mengembalikan seluruh payload, jadi data ada di `result.data.data`.
- **Browser tidak memanggil Go.** Token ada di cookie HttpOnly; `/api/v1/*` di Nginx tidak membawa Bearer. Aksi interaktif (centang tugas, drag-drop Kanban, catat pembayaran, check-in, ekspor) memanggil **BFF same-origin** `src/pages/dashboard/api/planner/[...path].ts` (URL `/dashboard/api/planner/...`), yang meneruskan ke `/v1/planner/...` dengan Bearer dan `X-Real-IP`. Aturan BFF mengikuti `src/pages/auth/*.ts`:
  1. berada di bawah `/dashboard/` sehingga dijaga middleware;
  2. method selain `GET` wajib `Content-Type: application/json` (selain itu 415) sebagai pertahanan CSRF;
  3. path diteruskan hanya jika cocok `^[a-z0-9/_-]+$` dan tidak memuat `..` (tidak bisa dipakai memanggil endpoint non-planner); query string diteruskan apa adanya (di-encode ulang, bukan disambung sebagai teks);
  4. respons file (`.xlsx`, `.pdf`) diteruskan sebagai stream beserta `Content-Disposition`; untuk itu ditambahkan varian `backendFetchRaw` di `lib/api/backend.ts` (tanpa parse JSON, batas waktu 30 dtk);
  5. status dan body error backend diteruskan apa adanya.
- **Layout bersama:** semua halaman `[id]/**` memakai `src/layouts/PlannerLayout.astro` (membungkus `AppLayout`). Ia memuat daftar proyek **ringan** sekali per permintaan (`GET /v1/planner/projects?status=all&view=lite`, tanpa agregat) untuk Project Switcher dan menggambar sub-navigasi enam menu di bilah sekunder di bawah header; hanya menu yang sudah dibangun berupa tautan, sisanya nonaktif ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)). Halaman menu tidak memuat daftar proyek sendiri. Halaman `src/pages/404.astro` perlu ditambahkan (belum ada di repo) karena `Astro.rewrite('/404')` dipakai untuk proyek yang tidak ditemukan.
- **Struktur file:** `src/components/planner/<menu>/*.astro`, `src/lib/planner/{types,format,countdown,messages,eventTypes,client,server}.ts`. `messages.ts` memetakan `error.code` ke teks Indonesia (seperti `lib/auth/messages.ts`). Interaksi klien memakai `<script>` TypeScript vanilla per komponen (seperti `AuthForm.astro`); **tanpa dependensi frontend baru**: grafik SVG murni, Gantt CSS Grid, drag-drop HTML5 native.
- **Gaya:** hanya token yang ada di `tokens.css` (`--sage-*`, `--color-*`, `--space-*`, `--radius-*`, `--font-display` Fredoka, `--font-body` Nunito Sans). Navigasi `AppHeader.astro` menambah tautan "Planner" di samping "Undangan".
- **Pembaruan berkala (polling):** layar yang diperbarui berkala (dashboard 30 dtk, daftar tamu, meja resepsionis, tracker rundown) sebaiknya memakai **halaman partial Astro** (`export const partial = true`) yang merender HTML dari komponen yang sama dengan halaman penuh, lalu skrip menukar isi DOM; dengan begitu logika render hanya ada satu. Polling berhenti saat tab tersembunyi, mundur setelah galat, dan tidak menimpa isian yang sedang dibuka. Contoh dan aturan rinci: [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial). BFF JSON tetap tersedia sebagai alternatif.
- **Uji:** fungsi murni (format rupiah, normalisasi nomor WA, parse jam `07.46`↔`07:46`, hitung hitung-mundur) diuji dengan `node:test` di `frontend/tests/*.test.mjs`.

### 1.4 Kontrak API dan dokumentasi

- Saat sebuah modul diimplementasikan, endpoint-nya **ditambahkan ke `backend/docs/openapi.yaml`** (kontrak yang di-lint CI). Spesifikasi di folder ini bukan pengganti kontrak itu; ia sumber desain sampai kontrak ditulis.
- Setiap modul menambah dokumen di `docs/backend/planner/` dan `docs/frontend/planner/` serta entri `CHANGELOG.md`, sesuai aturan [docs/backend/README.md](../../../backend/README.md).
- Dependensi Go baru yang dibutuhkan (disetujui saat implementasi, bukan sekarang): `github.com/xuri/excelize/v2` (ekspor/impor `.xlsx`; **disetujui pemilik 2026-10-21** bersama Menu 02, v2.11.0) dan satu pustaka PDF (cue card, rekap; belum disetujui). Pembaca CSV memakai `encoding/csv` (stdlib).
- Nginx: `client_max_body_size 8m` sudah berlaku; impor tamu dibatasi lebih ketat di aplikasi (5 MB, 2.000 baris). Endpoint ekspor/impor dibatasi per pengguna (ekspor 10/menit, impor 5/menit) dengan pola `ratelimit.go` modul auth. RSVP publik milik modul undangan memakai zona `public_writes` yang sudah ada.

### 1.5 Prasyarat runtime (dibaca dari kode dan Dockerfile)

1. **Data zona waktu.** Image runtime `backend/Dockerfile` adalah `alpine:3.23` tanpa paket `tzdata`, sehingga `time.LoadLocation("Asia/Jakarta")` gagal bila zoneinfo tidak ikut di biner. Diuji dengan emulasi (akses ke zoneinfo dan `GOROOT` diblokir pada macOS: `Asia/Jakarta` → error, dengan `-tags timetzdata` → berhasil); image sebenarnya belum diuji karena Docker tidak terpasang. Karena pembuatan proyek, validasi zona, dan hitung mundur semuanya memakai `LoadLocation`, `cmd/api/main.go` mengimpor `_ "time/tzdata"` (±450 KB; **sudah ditambahkan**). Jangan menambah paket OS sebagai satu-satunya pengaman.
2. **Pool koneksi.** `db.Connect(ctx, url, maxConns)` menerima ukuran pool dari `DB_MAX_CONNS` (bawaan 10, rentang 1–1000; ada di `internal/config`, Compose, dan `.env.example`); tanpa itu bawaan pgxpool = `max(4, jumlah CPU)`. `BearerAuth` memakai satu koneksi per request ([ADR-0003](../../../adr/ADR-0003-access-token-database-check.md)), sehingga dashboard yang menyebar ke banyak query bisa menghabiskan pool dan menahan login semua pengguna; karena itu paralelisme dashboard dibatasi 3 (bab 7).
3. **Mode protokol pgx.** `db.Connect` memakai `QueryExecModeSimpleProtocol` (agar cocok dengan pgBouncer). Hasil kueri datang dalam format teks; konsekuensi pemindaian ada di bab 6.

---

## 2. Isolasi Data dan Keamanan

1. **`ProjectGuard`** (paket `plannerhttp`): memuat proyek dari `:id` **dan** `user_id` pemanggil; proyek tidak ada, milik orang lain, maupun `:id` yang bukan UUID valid (mis. `summary`) sama-sama `404 not_found` (tidak membocorkan keberadaan; string yang bukan UUID tidak pernah diteruskan ke kueri). Aturan 404 untuk id yang tak valid berlaku juga untuk semua id anak di URL. Hasilnya disimpan di konteks (`CurrentProject(c)`).
2. **`WriteGuard`**: proyek berstatus `archived` bersifat baca-saja; method selain `GET` menjawab `409 project_read_only` (kecuali `PATCH .../status` untuk membuka arsip dan `DELETE /v1/planner/projects/:id`: menghapus proyek arsip tidak perlu membuka arsipnya dulu). Pengecualian dicocokkan dengan pola rute (`c.FullPath()` + method), bukan akhiran URL. Proyek `completed` tetap bisa diedit (pelunasan susulan).
3. **Pertahanan berlapis:** setiap query repository tetap memfilter `project_id`, dan id anak dari URL selalu dicari dengan `WHERE id = $x AND project_id = $y`. Di basis data, **FK komposit** `(x_id, project_id)` mencegah baris proyek A menunjuk vendor, kontributor, pos, atau dokumen proyek B (terverifikasi, bab 10).
4. Tautan dari pengguna (`receipt_url`, `portfolio_url`, `spotify_url`, `youtube_url`) divalidasi skema `http`/`https` saat disimpan dan dirender dengan `rel="noopener noreferrer"`. `purchase_location` (Menu 04) adalah teks bebas (URL marketplace atau nama toko dan alamat): server hanya membatasi panjangnya, dan klien menjadikannya tautan hanya bila `safeHref` menerimanya (`http`/`https` yang sah).
5. Sel ekspor `.xlsx` yang diawali `=`, `+`, `-`, `@` diberi awalan `'` (mencegah formula injection dari nama tamu/vendor). Teks pengguna ditulis dengan `SetCellStr`. Prototipe Menu 02 menunjukkan `excelize` menyimpan string berawalan `=` sebagai string bersama (tidak pernah formula), jadi awalan ini adalah pertahanan berlapis dan **tampak sebagai karakter `'`** di sel ([02 BE §4.4](../02_TIMELINE_DAN_TUGAS/BE.md#44-ekspor-xlsx), keputusan di 02 BE bab 8 butir 12).
6. Nama berkas ekspor (`Content-Disposition`) dibuat dari slug ASCII judul proyek + tanggal (`pernikahan-sinta-arif-2026-10-21.xlsx`), tidak pernah dari judul mentah, supaya judul dengan emoji, tanda kutip, atau baris baru tidak merusak header.
7. Data tamu (nama, telepon) bersifat pribadi: hanya pemilik proyek yang bisa membacanya; tidak ada endpoint publik di modul Planner. Kebijakan retensi/penghapusan akun mengikuti [docs/database.md](../../../database.md#data-pribadi).

---

## 3. Registry Event Type (di kode, bukan basis data)

Sembilan jenis acara Adicara didefinisikan **sekali** di `planner/domain/eventtype.go` dan menjadi satu-satunya sumber untuk ikon, label, sesi bawaan, peran, kategori, kolom metadata, dan ketersediaan template. Tidak boleh ada daftar jenis acara kedua di README atau FE modul.

```go
type Option struct{ Key, Label string }

// Implementasi menu 01 (planner/domain/eventtype.go) baru memuat: Key, Label, Icon, LeadTimeDays, Sessions
// (SessionSeed dengan ScopeKey eksplisit), MetadataFields, Labels, dan Templates. Bidang yang dimiliki modul lain
// (kategori vendor/belanja, sisi dan kategori tamu, peran, subjek dokumen, panduan, moment tag) ditambahkan
// modul itu saat dibangun; datanya ada di TEMPLATE_EVENT.
type EventTypeSpec struct {
    Key, Label, Icon   string
    LeadTimeDays       int            // horizon baku template jadwal
    Sessions           []SessionSeed  // sesi bawaan (selalu di-seed); SameDay=true -> session_date = event_date
    GuestSides         []Option       // kosong = tanpa pemisahan pihak
    GuestCategories    []Option
    ContributorRoles   []Option
    AdminSubjects      []Option
    AdminGuide         Guide          // judul + 4 langkah panduan birokrasi/perizinan (dikirim ke klien)
    AdminLabel         string         // judul widget Legalitas di dashboard (hanya provider admin_docs; tidak dikirim). Nilai: TEMPLATE_EVENT bab 4.1
    MetadataFields     []FieldSpec    // kunci, label, tipe, panjang maks, wajib
    VendorCategories   []VendorCategory // key, label, budget_item_keys (04 BE bab 6)
    ShoppingCategories []string         // teks kategori belanja; disimpan apa adanya (04 BE bab 6)
    ShoppingRollup     ShoppingRollup   // pos bawaan gulungan belanja: Key, Name, Category (TEMPLATE_EVENT 15.4)
    MomentTags         []string
    Templates          TemplateSet    // preset jadwal, dokumen, pos anggaran, belanja, agenda, playlist yang dimiliki jenis ini
    Labels             Labels         // sebutan untuk teks antarmuka yang bergantung jenis
}

type Labels struct {
    Guest     string // "Tamu" (seminar: "Peserta"): kata benda di widget ("20 Tamu · 43 pax") dan kartu Hub ("Tamu 43 pax")
    GuestMenu string // judul menu/widget 05: "Buku Tamu & RSVP" (seminar: "Peserta & RSVP")
}
```

`GET /v1/planner/event-types` mengembalikan registry ini (`Cache-Control: public, max-age=3600`). Respons memuat `templates_available` (template yang dimiliki jenis **dan** punya `TemplateApplier` terdaftar, bab 1.1.1); wizard hanya menawarkan itu. Frontend mengambilnya server-side dan menyimpannya dalam memori proses selama 1 jam; **tidak ada** daftar peran, sesi, atau kategori yang ditulis keras di komponen FE. Nilai registry untuk kesembilan jenis: [TEMPLATE_EVENT.md](TEMPLATE_EVENT.md).

Dampaknya pada modul:
- Chip filter sesi, dropdown peran dokumen, sisi tamu, peran kontributor, kategori vendor/belanja, preset di modal generator, dan teks wizard langkah 3 **dibangkitkan dari registry** untuk jenis acara proyek.
- `metadata` proyek divalidasi terhadap `MetadataFields`: kunci tak dikenal ditolak `400 invalid_metadata`, tipe dan panjang dicek.
- Ikon tunggal per jenis (💍 🕌 🍼 🏢 🚀 🎂 🎓 🎤 ✨); `GetEventTypeIcon` di 06 dihapus.
- Kata "tamu", "undangan", dan judul menu 05 pada kartu/widget/sub-navigasi memakai `Labels` jenis acara (seminar → "Peserta"), bukan teks tetap.

**Menambah jenis acara baru** (supaya registry dan basis data tidak menyimpang): (1) entri registry; (2) migrasi `ALTER TABLE planner_projects DROP CONSTRAINT … ADD CONSTRAINT … CHECK (event_type IN (…))` karena kolom `event_type` memakai `CHECK`; (3) template bila ada; (4) uji otomatis yang membaca `pg_get_constraintdef` untuk `event_type` dan membandingkannya dengan kunci registry (gagal bila beda).

---

## 4. Master Sesi Bersama (`planner_event_sessions`)

Wedding punya Lamaran/Akad/Resepsi, seminar punya Registrasi/Sesi Pagi/Coffee Break, dan seterusnya. **Satu tabel** menjadi master sesi untuk semua modul; tanpa master bersama, `sub_event_scope` berupa teks bebas bisa berbeda-beda per modul (mis. `resepsi` vs `resepsi_1`).

| Aturan | Keterangan |
|---|---|
| Seed | Dibuat **tanpa syarat** saat proyek dibuat (satu transaksi): sesi bawaan jenis acara + satu baris sistem `all` ("Umum / Semua Sesi", tidak tampil di tab rundown, tidak bisa diubah-hapus). Tidak lagi bergantung pada centang rundown. `scope_key` sesi bawaan tertulis **eksplisit** di registry (`SessionSeed.ScopeKey`: `lamaran`, `akad`, `resepsi`, …), tidak diturunkan dari nama: aturan penurunan di baris berikut akan menjadikan "Akad Nikah" `akad_nikah`. |
| Kunci | `scope_key` (`^[a-z0-9_]{1,40}$`) unik per proyek, dibuat server saat sesi dibuat: nama → ASCII huruf kecil (aksen dilipat), selain `[a-z0-9]` jadi `_`, `_` berulang dan di ujung dibuang; hasil kosong (nama emoji/aksara non-Latin) → `sesi`; `all` (dicadangkan) → `all_sesi`; dipotong 36 karakter; bentrok → akhiran `_2`, `_3`, … (maksimal `_999`, sehingga panjang ≤ 40). Bentrok balapan (`23505` pada `scope_key`) dicoba ulang maksimal 3 kali. Nama tampilan boleh diganti kapan saja tanpa memutus data. |
| Nama | Unik per proyek tanpa membedakan huruf besar-kecil (indeks `planner_event_sessions_name_idx`); nama ganda → `409 session_name_taken`. Impor dan ekspor memetakan sesi lewat nama sehingga tidak boleh ambigu. |
| Batas | Maksimal 20 sesi non-sistem per proyek (`409 session_limit_reached`). Nol sesi diperbolehkan (jenis `other` yang sesinya dihapus): menu yang bergantung sesi menampilkan ajakan menambah sesi. |
| Penautan | Tabel `planner_tasks`, `planner_vendors`, `planner_budget_scenario_items`, `planner_budget_items`, `planner_shopping_items`, `planner_guests` menyimpan `sub_event_scope TEXT` dengan **FK komposit** `(project_id, sub_event_scope) → planner_event_sessions(project_id, scope_key)` bertipe `ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED`. `planner_rundowns` memakai `session_id`. **Mengapa `DEFERRABLE`:** dengan `ON DELETE RESTRICT` (dan juga `NO ACTION` biasa) menghapus proyek yang punya skenario anggaran gagal, karena cascade proyek → skenario → item skenario dijalankan sebagai sub-statement terpisah sehingga pemeriksaan FK sesi terjadi sebelum item skenario terhapus (diuji; tidak terjadi pada anak yang berhubungan langsung dengan proyek, sehingga mudah lolos uji sederhana). FK yang ditunda diperiksa saat `COMMIT`: hapus proyek/akun selalu berhasil, sedangkan menghapus sesi yang masih dipakai tetap ditolak. Konsekuensinya galat `23503` muncul pada `tx.Commit()`, bukan pada pernyataan yang bersalah, sehingga usecase memeriksa lebih dulu (scope ada, sesi tidak dipakai) dan memetakan sisa galat commit ke `409 conflict`. FK `savings → contributors` diperlakukan sama. |
| Ubah kunci | `UPDATE scope_key` merambat otomatis ke semua tabel (terverifikasi). Antarmuka tidak menawarkannya; hanya nama yang bisa diubah. |
| Hapus sesi | `DELETE .../sessions/:session_id?move_to=<scope_key>`: dalam satu transaksi (baris proyek dikunci `SELECT … FOR UPDATE` agar tidak berbalapan dengan pembuatan/penghapusan sesi lain) usecase memindahkan semua baris yang menunjuk sesi itu ke `move_to` (default `all`; harus ada dan bukan sesi yang dihapus): **tugas, vendor, pos anggaran, item skenario, belanja, tamu**, membersihkan `active_sub_events`, lalu menghapus. Pos anggaran dan item skenario yang `item_key`-nya akan bentrok dengan baris di sesi tujuan (mis. `catering` ada di Lamaran dan Resepsi) dinetralkan menjadi `item_key = NULL` (menjadi pos biasa) alih-alih gagal dengan `23505`. Algoritma SQL lengkap dan hasil ujinya ada di [01 BE §4.6](../01_DASHBOARD/BE.md#46-sesi-acara-master-sesi). Jika sesi punya agenda rundown, wajib `delete_agendas=true`, selain itu `409 session_in_use` beserta jumlah per modul. |
| Skenario | `planner_budget_scenarios.active_sub_events TEXT[]` berisi `scope_key` (tanpa FK, sehingga **harus** dibersihkan usecase saat sesi dihapus: dibuang, dan `move_to` ditambahkan bila bukan `all`); kosong berarti semua sesi non-sistem. Tidak ada default pernikahan yang ditulis keras di kolom. |
| Tanggal | Sesi punya `session_date` dan `start_time`/`end_time` (opsional). Sesi bawaan lahir bertanggal Hari-H (`SameDay`; `lamaran` pernikahan dikecualikan) dan ikut pindah saat Hari-H diubah. Dipakai milestone tabungan (03), pesan WhatsApp (05, 06), dan header tab rundown. `planner_projects.event_date` tetap Hari-H utama (untuk wedding: tanggal akad/resepsi); acara beberapa hari memakai `event_end_date` (sesi boleh bertanggal di antara keduanya). |

CRUD sesi ada di [01 BE](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi). Modul 06 hanya menambahkan agenda di atas sesi.

---

## 5. Template vs Data Contoh

Seed "bawaan" yang berisi DP yang sudah dibayar, setoran Rp 47,5 jt, vendor fiktif, atau tamu yang sudah check-in akan membuat proyek nyata yang baru dibuat langsung tampil 33% terbayar. Karena itu keduanya dipisah tegas:

| | **Template** | **Data contoh (Proyek Contoh)** |
|---|---|---|
| Isi | struktur + estimasi acuan; **nol** pembayaran, nol setoran, nol vendor/tamu | dataset PDF lengkap (pembayaran, tabungan, vendor, tamu, status) |
| Lokasi | kode Go (`//go:embed`/struct), per jenis acara, kunci `template_key` | kode yang sama + overlay dataset |
| Dibuat | hanya atas pilihan pengguna (wizard langkah 3 atau tombol "Terapkan template") | `POST /v1/planner/projects/sample` |
| Efek pada proyek nyata | tidak pernah menimpa data pengguna: mode `append` (lewati `template_key` yang sudah ada) atau `replace` dengan konfirmasi | proyek terpisah bertanda `is_sample = true`, badge "Contoh", boleh dihapus |

- **`template_key`** (mis. `wedding.pertemuan_keluarga_1`) menjadi dasar mode `append`/`reset`. Pencocokan berdasarkan judul rapuh karena PDF punya judul berulang ("Pertemuan keluarga" ×5).
- **`origin`** pada tugas dan dokumen: `pdf` (disalin dari PDF), `adicara` (tambahan kami), `custom` (buatan pengguna). Tampilan menandai tambahan Adicara agar jelas mana yang dari acuan.
- Template menyalin **nama dan harga acuan PDF** tetapi tidak menyalin tautan produk pribadi pembuat PDF, nama orang, atau nomor telepon.
- **Proyek Contoh:** wedding "Contoh: Pernikahan Sinta & Arif", `event_date = hari ini + 175 hari` (meniru "Towards the Day: 175 Days" di PDF), `hero_title = "Bismillah Nikah"`, semua template yang appliernya terdaftar diterapkan lalu overlay dataset PDF (selama modul 02–06 belum ada, contoh hanya berisi proyek, sesi, dan metadata). Maksimal satu per pengguna (indeks unik parsial `planner_projects_sample_idx`; `POST .../sample` idempoten: `201` bila baru dibuat, `200` dengan proyek yang sudah ada, termasuk saat dua klik berbalapan), dikeluarkan dari metrik portofolio (total pagu, jumlah acara aktif), dan menjadi **fixture uji emas** (bab 9).
- Template non-pernikahan **tidak berasal dari PDF**; isinya usulan Adicara dan ditandai begitu di [TEMPLATE_EVENT.md](TEMPLATE_EVENT.md).

---

## 6. Konvensi Data

| Topik | Aturan |
|---|---|
| **Uang** | Rupiah utuh: `BIGINT` di DB, `int64` di Go, **bilangan bulat** di JSON. Tidak ada `float64`/`NUMERIC(…,2)` untuk uang. Perkalian (jumlah × harga satuan) dibulatkan `round()` ke rupiah. Persentase dihitung saat dibaca (`cost/total_target`), tidak disimpan. Batas tiap nilai 0…10¹³ (aman bagi `Number` JavaScript dan jumlah ribuan baris); diperiksa usecase (`400 invalid_input`) dan `CHECK` pada kolom proyek. |
| **Tanggal & zona waktu** | `DATE` untuk tanggal acara/jadwal, `TIME` untuk jam, `TIMESTAMPTZ` untuk kejadian. Proyek punya `timezone` IANA (default `Asia/Jakarta`); "hari ini" dan hitung mundur dihitung di zona itu: SQL `event_date - (now() AT TIME ZONE tz)::date` atau Go `DaysUntil` (01 BE). Jam di JSON `"HH:MM"` (SQL `to_char(kolom,'HH24:MI')`; pgx mode teks mengembalikan `07:00:00`); input FE `07.46` dinormalisasi ke `07:46`. Zona waktu hanya nama IANA kanonik (`Asia/Jakarta`, `UTC`); `asia/jakarta`, `UTC+7`, `Local`, dan singkatan seperti `WIB` ditolak (PostgreSQL menerima sebagian dari itu sedangkan Go menolaknya, dan sebaliknya: keduanya harus setuju). |
| **Pemindaian pgx** | Backend memakai `QueryExecModeSimpleProtocol` (hasil berformat teks). Aturan yang diuji pada mode itu: kolom `NULL` ke `string` gagal (`cannot scan NULL into *string`) → pakai `*string`/`pgtype`; `sum(bigint)` (numerik) terbaca ke `int64`; `DATE` ke `string` hanya berhasil karena mode teks → mapping tetap `time.Time` (00:00 UTC) lalu diformat `2006-01-02`, jangan mengandalkan mode; `TIME` memakai `to_char`. Tipe tanggal domain adalah `time.Time`, bukan `civil.Date` (bukan dependensi `go.mod`). |
| **Pembagi nol** | `percent(n, d) = d <= 0 ? null : n/d`. `null` ditampilkan "–" tanpa batang progres, dan kartu menampilkan keadaan kosong ("Belum ada tugas", "Pagu belum diisi", "Belum ada berkas", "Belum ada tamu"). Berlaku untuk progres tugas, terbayar/pagu, berkas, kategori vendor, dan RSVP; backend tidak mengirim `*_pct`. |
| **Pembaruan sebagian** | `PATCH` mengubah hanya field yang dikirim. `metadata` memakai JSON Merge Patch: kunci yang ada ditimpa, `null` menghapus, kunci yang tidak disebut dibiarkan (dua tab yang mengedit kolom berbeda tidak saling menimpa). `PUT` mengganti penuh; field opsional yang tidak dikirim dikosongkan, sehingga formulir harus mengirim semua field. |
| **Pesan validasi** | Untuk `invalid_input`, `invalid_metadata`, `invalid_date`, dan `invalid_timezone`, `message` berbentuk `<field>: <alasan>` (Inggris) supaya formulir dapat menandai kolom yang salah; `code` tetap satu-satunya yang dipetakan ke teks Indonesia. |
| **Urutan** | Endpoint `reorder` menerima daftar id lengkap; usecase memeriksa semuanya milik proyek dan menulis `sort_order = indeks` dalam satu `UPDATE ... FROM (VALUES ...)`. |
| **Paginasi** | Daftar kecil (tugas, pos, vendor, agenda) tanpa paginasi. Daftar tamu: `limit` (default 100, maks 500) dan `offset`; KPI/counter dihitung terpisah dari halaman. |
| **Enum** | `CHECK` di DB + konstanta Go bertipe string. Kunci berasal dari registry (sesi, peran, kategori); nilai kustom diperbolehkan untuk kategori, tidak untuk status. |
| **Polling (pengganti SSE)** | Dashboard 30 dtk, daftar tamu 30 dtk, meja resepsionis 5 dtk, tracker live rundown 5 dtk; hanya saat tab aktif (`document.visibilityState`). Endpoint ringan khusus: `.../guests/summary`, `.../rundowns/summary`. |
| **Hapus proyek** | `DELETE` menghapus semua data anak (CASCADE) dan butuh `?confirm=<judul proyek>` dari klien (cegah salah tekan). |

### Kode error Planner

Menambah daftar di [docs/api.md](../../../api.md#format-error); kode lama (`invalid_json`, `invalid_input`, `unauthorized`, `not_found`, `rate_limited`, `internal`) tetap.

| Status | Kode | Kapan |
|---|---|---|
| 400 | `invalid_metadata`, `invalid_scope`, `invalid_date`, `invalid_timezone`, `invalid_template`, `invalid_phone`, `invalid_file`, `invalid_stage` | validasi masukan; `invalid_scope` juga untuk pelanggaran FK sesi (pg `23503` pada `*_sub_event_scope_fkey`) |
| 404 | `not_found` | proyek/anak tidak ada **atau bukan milik pemanggil** |
| 409 | `project_read_only`, `project_limit_reached`, `max_scenarios_reached`, `session_in_use`, `session_name_taken`, `session_limit_reached`, `task_limit_reached`, `admin_doc_limit_reached`, `scenario_item_limit_reached`, `budget_item_limit_reached`, `contributor_limit_reached`, `savings_entry_limit_reached`, `vendor_limit_reached`, `shopping_item_limit_reached`, `conflict`, `contributor_in_use`, `already_checked_in`, `vendor_already_linked`, `rollup_locked`, `rollup_conflict` | konflik status/aturan bisnis. `vendor_already_linked`: vendor sudah tertaut ke pos lain (indeks unik). `rollup_locked`: pembayaran manual pada pos yang dihitung dari belanja. `rollup_conflict`: mengaktifkan rollup pada pos yang sudah punya pembayaran tanpa `confirm`. `conflict`: galat `23503`/`23505`/serialisasi yang muncul saat commit (FK ditunda) pada permintaan yang berbalapan; aman diulang. `project_limit_reached`: lebih dari `PLANNER_MAX_PROJECTS` proyek (bawaan 100; batas teknis, bukan batas produk). `task_limit_reached` / `admin_doc_limit_reached`: lebih dari 500 tugas / 200 berkas per proyek (angka usulan, [02 BE §3.4](../02_TIMELINE_DAN_TUGAS/BE.md#34-validasi-dan-batas)). `scenario_item_limit_reached` / `budget_item_limit_reached` / `contributor_limit_reached` / `savings_entry_limit_reached`: lebih dari 100 pos per skenario / 300 pos anggaran / 20 kontributor / 2.000 setoran per proyek (angka usulan, [03 BE §4.5](../03_KEUANGAN/BE.md#45-validasi-dan-batas)). `vendor_limit_reached` / `shopping_item_limit_reached`: lebih dari 200 vendor / 1.000 barang belanja per proyek (angka usulan, [04 BE §3.4](../04_VENDOR_DAN_BELANJA/BE.md#34-validasi-dan-batas)) |
| 413 | `file_too_large` | impor melebihi batas aplikasi |
| 402 | `entitlement_denied` | disiapkan untuk gating layanan |

**Notice** (bukan error; objek `{"code","message","count"}`, ditambah `shift_days` pada `tasks_not_shifted`). Pada `PATCH /projects/:id` ada dua daftar: `warnings[]` (`tasks_overdue_after_shift`) dan `hints[]` yang bersifat informasi (`tasks_not_shifted`, `sessions_dates_unchanged`). Pada `POST /projects`, setiap applier mengisi `applied.<kunci template>.warnings[]` dengan kodenya sendiri: `lead_time_compressed`, `lead_time_too_short`, `scope_fallback`, `budget_link_lost`, `estimate_differs` (02), `scope_missing` (06), `shared_budget_item` dan `unit_pricing` (04, hasil `lock-deal`). Menu 04 juga memakai `estimate_differs` dan `budget_link_released` (hasil `PUT`/`PATCH` vendor) serta `budget_link_lost` (hasil `apply-template` belanja).

---

## 7. Dashboard: Pola `SummaryProvider`

Satu query CTE yang menggabungkan semua tabel modul untuk widget tidak dipakai: ia mengikat dashboard ke tabel modul lain dan latensinya tidak pernah diukur (kueri Hub dan sesi di bab 1.1 adalah pengecualian yang terbatas). Yang dipakai:

```go
// planner/domain
type SummaryProvider interface {
    Key() string                                                     // "timeline" | "finance" | "vendors" | "guests" | "admin_docs" | "rundown"
    Summarize(ctx context.Context, p Project) (any, error)
}
```

Usecase dashboard menjalankan provider terdaftar paralel dengan aturan berikut (tiap aturan menutup satu kegagalan nyata):

1. **`defer recover()` di dalam setiap goroutine provider.** `gin.Recovery()` hanya melindungi goroutine handler; panic di goroutine anak mematikan seluruh proses API (diuji: proses keluar dengan panic walau ada `gin.Recovery()`). Provider yang panic → widget `{"error": "internal"}` dan panic dicatat.
2. **Paralelisme dibatasi 3** (`errgroup.SetLimit(3)`), karena pool koneksi kecil dan dipakai bersama `BearerAuth` (bab 1.5). Enam query serentak per pengguna per 30 detik akan membuat antrean pool.
3. **Batas waktu** `context.WithTimeout(ctx, 3*time.Second)` untuk seluruh dashboard; provider yang melewatinya → `{"error": "timeout"}`. Pembatalan konteks klien (tab ditutup) membatalkan kueri yang berjalan.
4. **Batas laju** `GET .../dashboard` 60 permintaan/menit/pengguna (pola `ratelimit.go`). Polling FE 30 detik ditambah jitter ±5 detik, berhenti saat tab tersembunyi, dan mundur ke 60 detik setelah galat.

Provider yang gagal menghasilkan `{"error": "<code>"}` pada widgetnya saja; widget lain tetap tampil. Modul yang belum dibangun tidak terdaftar, sehingga widgetnya `null` dan FE menampilkan kartu "Segera hadir". Target latensi p95 < 100 ms pada data 10 kali ukuran proyek contoh **belum diukur**; diukur saat implementasi dan dicatat di CHANGELOG.

---

## 8. Matriks Integrasi (apa yang benar-benar tersambung)

Setiap baris punya mekanisme konkret di skema/endpoint. Klaim integrasi tanpa mekanisme tidak boleh ada di README modul.

| # | Dari → Ke | Mekanisme | Dokumen |
|---|---|---|---|
| 1 | Semua modul → 01 Dashboard | `SummaryProvider` per modul (bab 7) | 01 BE §5 |
| 2 | 02 Dokumen → 03 Anggaran | `planner_budget_items.admin_doc_id`; `POST .../admin-docs/:id/sync-budget` (SQL langsung ke tabel 03, pengecualian bab 1.1); status "tersinkron" = ada pos yang menunjuk dokumen. Diuji terhadap pos `akad/biaya_kua` buatan modul 03 (`TestAdminDocSyncLinksTheTemplateLineOfThisModule`) | 02 BE §4.5 |
| 3 | 02 Tugas ↔ 04 Vendor | `planner_tasks.vendor_id` (FK komposit) | 02 BE §1 |
| 4 | 04 Vendor → 03 Anggaran | `lock-deal` di bawah kunci proyek: tautkan ke pos khusus via kunci kategori (sesi vendor, lalu Umum), atau buat pos baru; **tidak ada pos ganda**; pos yang dipakai bersama beberapa kategori dan pos berharga satuan tidak ditimpa | 04 BE §2.2 |
| 5 | 03 Skenario → 04 Vendor | pagu dibaca lewat `category_key → budget_item_keys` pada skenario aktif setiap kali daftar vendor dibuka (tidak disalin; kolom `budget_limit` hanya pagu manual) | 04 BE §2.1 |
| 6 | 04 Belanja → 03 Anggaran | `planner_shopping_items.budget_item_id` + `planner_budget_items.rollup_source='shopping'`; angka dihitung view, bukan salinan; pos bawaan per jenis acara dari registry (`ShoppingRollup`, TEMPLATE_EVENT 15.4) | 04 BE §2.4, 03 BE §1.2 |
| 7 | 04 Belanja ↔ 05 Tamu | `planner_shopping_items.purpose='souvenir'` (stok = `floor(Σ quantity)` barang selesai, dibaca Menu 05 dengan SQL) vs `planner_guests.souvenir_received` (serah terima) | 04 BE §2.5, 05 BE §2.5 |
| 8 | 05 Tamu → 03 Anggaran | proyeksi porsi dari RSVP mengisi `quantity` pos katering (`pricing_mode='unit'`) atas tindakan pengguna | 05 BE §2.3 |
| 9 | 06 Agenda → 04 Vendor | `planner_rundowns.vendor_id` → kontak WA | 06 BE §2.2 |
| 10 | 05/06 → 02 Jadwal | tugas tambahan Adicara bertanda `template_key` (`wedding.adicara_*`): finalisasi tamu H-45, sebar undangan H-30, follow-up RSVP H-10, technical meeting H-7, gladi resik H-1, Hari-H | 02 BE §5.2 |
| 11 | 05 Tamu ↔ Undangan Digital | `planner_guests.invitation_id` (dual-key); antarmuka Go `InvitationLookup`/`GuestRSVPWriter`; FK ditambah migrasi modul undangan. **Spec undangan masih kosong**, jadi mode Standalone adalah satu-satunya yang bisa dibangun sekarang | 05 BE §5 |
| 12 | Sesi → 02/03/04/05/06 | master sesi (bab 4) | 01 BE §3.3 |
| 12b | Wizard → semua modul | `TemplateApplier` per modul di dalam `TxManager.InTx` (bab 1.1.1); template tanpa applier tidak ditawarkan | 01 BE §4.1 |
| 13 | Planner ↔ layanan Gift | **tidak ada integrasi** sampai spec Vendor/Gift ditulis. Planner tidak menerima atau mencatat dana dari gateway (selaras prinsip pembukuan mandiri Tab Tabungan) | — |

---

## 9. Urutan Pembangunan dan Uji Penerimaan

**Urutan:** (1) `planner` core: migrasi 005, proyek, sesi, registry, guard, BFF FE → (2) 04a vendor (006) → (3) 02 timeline (007) → (4) 03 finance (008, 010 setelah 009) → (5) 04b belanja (009) → (6) 05 tamu (011) → (7) 06 rundown (012). Dashboard menampilkan widget sesuai provider yang sudah terdaftar. Migrasi 005–012 boleh dirilis sekaligus; modul yang belum dibangun hanya berarti tabel kosong.

**Uji emas dari Wedding_Planner.pdf** (dijalankan terhadap Proyek Contoh dengan `TEST_DATABASE_URL`, `go test -p 1`):

| Area | Nilai yang harus muncul | Sumber PDF |
|---|---|---|
| Anggaran | estimasi 74.150.000; DP 20.050.000; Termin 1 12.000.000; Termin 2 1.500.000; dibayar 33.550.000; sisa 40.600.000; rasio 45,2% | Hal. 5–6 |
| Per sesi | Lamaran 7.150.000 / 4.750.000 / 2.400.000; Akad 600.000 / 600.000 / 0; Resepsi 66.400.000 / 28.200.000 / 38.200.000 | Hal. 5–6 |
| Tabungan | target 75.000.000; terkumpul 47.500.000 (Pria 30.000.000, Wanita 17.500.000); 22 setoran Feb–Des 2026 | Hal. 7 |
| Skenario | 6 preset berjumlah 10/20/30/50/75/100 jt, masing-masing 10 pos | Hal. 4 |
| Seserahan | 37 item; estimasi 4.578.500; selesai 16 item = 2.063.800; on proses 21 item = 2.514.700 | Hal. 9–10 |
| Dokumen | 15 berkas; 9 selesai, 6 belum | Hal. 11 |
| Vendor | 10 kategori; 7 FIX; pagu skenario 75 Jt: Catering 30.000.000 (deal 25.000.000, hemat 5.000.000), Photography 3.000.000, tiga kelompok pas | Hal. 12 |
| Tamu | 2 pihak × 6 kategori | Hal. 13 |
| Rundown | Lamaran 11 agenda; Akad 8 + Resepsi 14 = 22 | Hal. 14 |
| Playlist | 30 lagu | Hal. 15–16 |
| Timeline | 44 tugas; durasi = selesai − mulai ("1 Feb→2 Feb" = 1 hari) | Hal. 3 |
| Dashboard | hitung mundur = 175 hari pada Proyek Contoh | Hal. 1 |
| Siklus hidup | `DELETE` proyek/akun yang berisi skenario, kontributor, belanja, rundown terhapus bersih (FK ditunda); pindah-sesi Lamaran → Resepsi menetralkan 1 `item_key` bentrok (`catering`), total tetap 74.150.000, `active_sub_events` bersih | audit 01 |

Angka anggaran, tabungan, dan durasi di atas sudah diuji pada DDL (bab 10); baris **Timeline** dan **Dokumen** (juga widget Timeline dan Legalitas) dijalankan pada kode menu 02 oleh `TestSampleProjectGolden`; baris **Anggaran**, **Per sesi**, **Tabungan**, dan **Skenario** (juga widget Keuangan) dijalankan pada kode menu 03 oleh `TestSampleProjectFinanceGolden`; baris **Seserahan** dan **Vendor** (juga widget Vendor) dijalankan pada kode menu 04 oleh `TestSampleProjectVendorGolden`; sisanya diuji saat modulnya diimplementasikan.

---

## 10. Validasi Spesifikasi

Yang sudah dijalankan pada 2026-10-20 dan 2026-10-21 (PostgreSQL 16.2 lokal, database sementara yang dihapus setelahnya; database `adicara` tidak disentuh):

- Migrasi 005–012 (blok `-- +goose Up` dari dokumen BE) diterapkan berurutan tanpa error, `Down` lalu `Up` ulang bersih (14 tabel + 1 view).
- 14 pos anggaran PDF Hal. 5–6 dimasukkan: view `planner_budget_lines` menghasilkan **74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000** (45,2%) dan angka per sesi persis seperti PDF.
- Ditolak seperti dirancang: sesi tak dikenal, vendor lintas proyek, kontributor lintas proyek, hapus sesi yang masih dipakai, dua baris `all`, tamu tanpa proyek dan undangan, agenda berakhir sebelum mulai. Diizinkan: tamu milik undangan saja; ganti `scope_key` merambat ke 8 pos.
- Kolom generated: durasi tugas (1/2/1), selisih harga vendor `NULL` saat belum ada penawaran, total belanja `300 × 45.000`, durasi agenda (20 mnt; `NULL` untuk "selesai"; 120 mnt lewat tengah malam).
- Hitung mundur: Go `DaysUntil` dan SQL sama-sama 173 hari pada 6 Okt 2026 untuk jam 00:00–23:00 WIB (rumus lama: 174 pada 00:00–06:59).

Audit menu 01 (2026-10-21), selain yang di atas:
- Hapus proyek/akun dengan skenario anggaran **gagal** pada DDL awal (FK sesi `RESTRICT` maupun `NO ACTION`), **berhasil** dengan `DEFERRABLE INITIALLY DEFERRED`; hapus sesi yang masih dipakai tetap ditolak (saat commit); ganti `scope_key` tetap merambat.
- Constraint baru: judul kosong/berspasi ujung/berisi baris baru, zona waktu non-kanonik atau tak dikenal, uang > 10¹³, `event_end_date` < `event_date`, proyek contoh ganda, nama sesi ganda, ikon terlalu panjang semuanya ditolak; `client_token` yang sama tidak membuat proyek kedua.
- Algoritma pindah-sesi (01 BE §4.6) pada data golden dengan bentrok `item_key`; fase acara (`upcoming/ongoing/finished`) pada batas tengah malam WIB untuk acara 1 dan 3 hari serta zona `UTC`; urutan Hub; kueri Hub dan versi ringan.
- Go: rute Gin statis berdampingan dengan `:id` (`/projects/summary`, `/guests/summary`, `/guests/export`, `/rundowns/summary`) terselesaikan benar; `{"title":""}`/`"   "` lolos `binding:"omitempty,max=255"` pada pointer; `NULL` → `string` gagal pada pgx mode teks; panic di goroutine provider mematikan proses walau ada `gin.Recovery()`; `LoadLocation` gagal tanpa zoneinfo.

Implementasi menu 01 (2026-10-21, PostgreSQL 16.2, database sementara): migrasi 005–012 naik, turun ke 4, dan naik lagi; kode Go, endpoint, isolasi antar pengguna, pindah sesi, hapus proyek dengan skenario, fase di batas tengah malam (SQL = Go), balapan `sample` dan `client_token`, dan registry ↔ `CHECK(event_type)` diuji otomatis (127 tes tingkat atas lulus, `-race` lulus); detail di [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-01.md#pengujian).

Audit menu 02 (2026-10-21, PostgreSQL 16.2, database sementara yang dihapus setelahnya): migrasi 001–012 naik; 50 tugas dan 15 berkas Proyek Contoh dimasukkan dari tabel 02 BE; kueri provider `timeline`/`admin_docs`, ringkasan `GET /tasks`, urut ulang, geser manual, `append` idempoten, dan semua langkah sinkron biaya (dijalankan persis seperti tertulis di 02 BE §4.5) menghasilkan angka yang tertulis di sana; nama galat FK tugas dicatat (vendor galat pada pernyataan, sesi baru saat `COMMIT`); nama, durasi, dan status 44 tugas serta 15 berkas dibandingkan dengan PDF; algoritma penskalaan jadwal (kode Go acuan) lulus lima tes. Temuan yang mengubah dokumen lain: tanggal sesi Lamaran Proyek Contoh adalah **15 Okt 2026** (bukan 10 Okt), karena offset H-169 diskalakan `175/180`.

Implementasi menu 02 (2026-10-21, backend): modul `plannertimeline` dengan 17 endpoint, ekspor `.xlsx`, dan sambungan ke menu 01 diuji terhadap PostgreSQL 16.2 (177 tes tingkat atas lulus, `-race` pada paket baru); rinciannya di [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-02.md) dan [02 BE bab 6](../02_TIMELINE_DAN_TUGAS/BE.md#6-pengujian).

Audit menu 03 (2026-10-21, PostgreSQL 16.2, database sementara yang dihapus setelahnya): migrasi 001–012 naik; 14 pos, pembayaran, 22 setoran, dan tiga skenario Proyek Contoh dimasukkan dari tabel 03 BE; PDF Hal. 4–7 dibaca ulang dengan `pdfminer` dan dibandingkan **secara terprogram** (60 nilai preset, 14 pos, 4 subtotal, 22 setoran sama); semua blok SQL logika bisnis 03 BE bab 3 dijalankan apa adanya; hasil khas: aktivasi skenario dengan satu `UPDATE` gagal `23505` (harus dua pernyataan), batas 3 skenario tanpa kunci menghasilkan 4, `FOR UPDATE` pada view ditolak, perkalian jumlah × harga satuan dapat melampaui `bigint`, pindah pos ke sesi yang sudah punya kuncinya melanggar indeks unik; rincian di [03 BE bab 7.1](../03_KEUANGAN/BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi).

Implementasi menu 03 (2026-10-21, backend): modul `plannerfinance` dengan 32 endpoint, template pos, ekspor `.xlsx`, dan sambungan ke menu 01 (provider `finance`, applier `budget`, overlay Proyek Contoh) diuji terhadap PostgreSQL 16.2 (261 tes tingkat atas lulus, 0 gagal, 0 di-skip; `-race` pada paket planner dan `app`; cakupan modul 91,3%; 23 mutasi kode produksi semuanya digagalkan tes). Uji emas Proyek Contoh dari PDF Hal. 4–7 lulus lewat seluruh tumpukan (74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000, status 7/7/0, milestone, rekomendasi 4.584.000, 22 setoran, tiga skenario, `unallocated` 850.000, widget dashboard). Hasil khas yang tidak terlihat pada audit SQL: delapan pembayaran `full` serentak pada satu pos semuanya dijawab sukses bila baris tidak dikunci (dengan kunci: satu sukses, tujuh `invalid_stage`), wizard dengan pagu kecil menjadi 500 pada template persentase (selisih pembulatan negatif), template persentase tidak punya kategori untuk kolom `NOT NULL`; rincian di [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-03.md) dan [03 BE bab 7.2](../03_KEUANGAN/BE.md#72-pengujian-pada-implementasi). Tes pembatas laju milik menu 01 dan 02 ditemukan bergantung pada jam dinding (gagal setelah sekitar pukul 10.05 UTC 2026-10-21) dan diperbaiki di tesnya.

Audit menu 04 (2026-10-22, PostgreSQL 16.2, database sementara yang dihapus setelahnya, **backend Menu 01–03 yang sebenarnya** untuk Proyek Contoh dan sembilan jenis acara): migrasi 001–012 naik lewat `go run ./cmd/migrate up`; DDL 006 dan 009 identik dengan BE; PDF Hal. 9–10 dan 12 dibaca ulang dengan `pdfminer` (37 barang, subtotal, 16/21, 10 kategori, 7 FIX; nol selisih); hasil khas: pagu Catering Proyek Contoh adalah 30.000.000 (skenario 75 Jt), bukan 25.000.000; pagu yang disalin ke kolom vendor menjadi basi saat skenario lain diterapkan; seluruh pos skenario dan template jenis non-pernikahan berada di sesi `all` sehingga `lock-deal` aturan lama membuat pos kedua (total estimasi 120.000.000 dari pagu 100.000.000 pada seminar); dua `lock-deal` serentak tanpa kunci proyek sama-sama "berhasil" padahal tautan yang pertama tertimpa; `lock-deal` pada pos berharga satuan membuat estimasi berbeda dari jumlah × harga; nama vendor 255 karakter melanggar `item_name` ≤ 150 pada pos baru; barang yang dibuat sesudah gulungan aktif tidak ikut dan pencabutan meninggalkan 37 barang yang menunjuk pos; `round(quantity × harga)` melebihi `bigint`; rincian di [04 BE bab 7.1](../04_VENDOR_DAN_BELANJA/BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi).

Implementasi menu 04 (2026-10-23, backend): modul `plannervendor` dengan 17 endpoint, `NormalizeWA` bersama (`planner/domain`), provider `vendors`, applier `shopping`, dua overlay Proyek Contoh, dan ekspor `.xlsx` diuji terhadap PostgreSQL 16.2 (334 tes tingkat atas lulus, 0 gagal, 0 di-skip; `-race`; cakupan modul 91,4%; 37 mutasi kode produksi, 36 digagalkan dan satu setara). Uji emas Proyek Contoh lulus lewat seluruh tumpukan (7 dari 10 kategori, fix 58.000.000, pipeline 4.300.000, Catering pagu 30.000.000 selisih +5.000.000, 37 barang 4.578.500 / 2.063.800 / 16 / 21); gulungan ke anggaran, delapan `lock-deal` serentak, dan alur penuh untuk sembilan jenis acara diuji. Temuan: pos gulungan bawaan peresmian, wisuda, dan seminar juga pos khusus kategori vendor (pos cadangan `perlengkapan`, [04 BE bab 9 butir 22](../04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Respons HTTP asli sama dengan fixture FE. Rinciannya di [dokumen implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md) dan [04 BE bab 7.3](../04_VENDOR_DAN_BELANJA/BE.md#73-hasil-implementasi-2026-10-23).

Belum dijalankan: kode modul 05–06, frontend, ekspor PDF, ekspor `.xlsx` yang dibuka di Excel/Numbers/LibreOffice, pengukuran latensi dashboard dengan data besar, image Docker sebenarnya (Docker tidak terpasang), uji emas PDF baris menu 04–06 sebagai tes Go (angka Menu 03 sudah menjadi tes Go: `TestSampleProjectFinanceGolden`).
