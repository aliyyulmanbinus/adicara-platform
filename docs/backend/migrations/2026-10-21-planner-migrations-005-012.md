# Migrasi 005–012: skema Planner

Tanggal: 2026-10-21. Modul: [Planner menu 01](../planner/2026-10-21-planner-menu-01.md).

| File | Isi | Pemilik |
|---|---|---|
| `005_000_planner_core.sql` | `planner_projects`, `planner_event_sessions`, fungsi dan pemicu `planner_check_timezone` | menu 01 (dipakai kodenya sekarang) |
| `006_000_planner_vendors.sql` | `planner_vendors` | menu 04 |
| `007_000_planner_timeline.sql` | `planner_tasks`, `planner_admin_docs` | menu 02 |
| `008_000_planner_finance.sql` | skenario anggaran, item skenario, kontributor, setoran, `planner_budget_items` | menu 03 |
| `009_000_planner_shopping.sql` | `planner_shopping_items` | menu 04 |
| `010_000_planner_budget_lines_view.sql` | view `planner_budget_lines` | menu 03 |
| `011_000_planner_guests.sql` | `planner_guests` | menu 05 |
| `012_000_planner_rundown.sql` | `planner_rundowns`, `planner_playlists` | menu 06 |

Total 14 tabel dan 1 view. Skema per kolom ada di blok ```sql pada `BE.md` modul masing-masing ([urutan dan alasan](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md#12-migrasi-goose-backendinternalmigratesql)); file migrasi **disalin otomatis** dari blok itu, bukan diketik ulang.

## Mengapa 006–012 ikut

Per 2026-10-31 semua tabelnya sudah punya kode: `planner` (menu 01; tabel 005), `plannertimeline` (menu 02; 007), `plannerfinance` (menu 03; 008 dan view 010), `plannervendor` (menu 04; 006 dan 009), `plannerguest` (menu 05; 011), dan `plannerrundown` (menu 06; 012). Saat 006–012 pertama dirilis, hanya tiga modul pertama yang ada. SQL menu 01 sendiri juga menyentuh tabel menu lain: kartu Hub menjumlah `planner_budget_lines`, `planner_tasks`, dan `planner_guests`; `usage` sesi menghitung tujuh tabel; hapus sesi memindahkan baris di enam tabel dan membersihkan `planner_budget_scenarios`; ubah tanggal menggeser `planner_tasks`. Tanpa tabelnya, fitur-fitur itu gagal pada kueri pertama. Kontrak bersama (bab 9) membolehkan 005–012 dirilis sekaligus dengan tabel kosong untuk modul yang belum dibangun.

## Aturan yang dijaga skema

- Semua tabel anak menunjuk `planner_projects(id)` dengan `ON DELETE CASCADE`; kepemilikan diperiksa lewat proyek (tabel anak tidak punya `user_id`). Menghapus pengguna ikut menghapus proyeknya.
- FK ke master sesi bertipe `DEFERRABLE INITIALLY DEFERRED`. Tanpa itu menghapus proyek yang punya skenario anggaran gagal (cascade proyek → skenario → item skenario memeriksa FK sesi sebelum item terhapus); dengan itu hapus proyek selalu berhasil dan menghapus sesi yang masih dipakai ditolak **saat commit** (`23503`, dipetakan ke `409 conflict`).
- FK komposit `(x_id, project_id)` mencegah baris proyek A menunjuk vendor, kontributor, pos, atau dokumen proyek B.
- `planner_projects.timezone`: `CHECK` menjaga bentuk kanonik, pemicu menjaga keberadaannya di PostgreSQL (`check_violation` "unknown time zone" → `400 invalid_timezone`).
- `CHECK(event_type)` harus selaras dengan registry di `domain/eventtype.go`; tes membandingkan keduanya. Menambah jenis acara berarti migrasi `ALTER … DROP CONSTRAINT … ADD CONSTRAINT`.

## Diuji

PostgreSQL 16.2 lokal, database sementara:

- `migrate fresh -yes` menerapkan 001–012 berurutan dan bersih lewat goose (simple protocol, blok `StatementBegin/End` untuk fungsi pemicu).
- `TestPlannerMigrationsRollBackAndReapply`: `Up` → 15 objek planner (14 tabel + view); sisipkan akun dan proyek; `DownTo(4)` → nol objek planner, fungsi pemicu hilang, akun utuh; `Up` ulang bersih.
- `TestAuthHardeningMigrationKeepsExistingData` (sudah ada) kini memakai `DownTo(3)`, sehingga ikut menjalankan seluruh `Down` 005–012 di atas data akun.
- Tes repository membuktikan constraint (judul, zona, uang, tanggal, contoh ganda, nama sesi ganda, `client_token`), siklus hidup hapus (proyek berisi skenario/kontributor/belanja/rundown, hapus akun), dan algoritma pindah sesi.

**Diubah di tempat (2026-10-31, menu 06):** `012_000_planner_rundown.sql` menambah `planner_rundowns_template_key_idx` dan `planner_playlists_template_key_idx` (unik parsial `(project_id, template_key) WHERE template_key IS NOT NULL`), seperti tugas, dokumen, dan belanja. Tanpa keduanya `INSERT … ON CONFLICT (project_id, template_key) … DO NOTHING` ditolak PostgreSQL dan kunci yang sama diterima dua kali. Migrasi belum dirilis ke `main`, jadi diubah di tempat (tahap pengembangan); database yang sudah menjalankan 012 lama harus `migrate fresh`, atau menambahkan keduanya dengan tangan (`CREATE UNIQUE INDEX planner_rundowns_template_key_idx ON planner_rundowns (project_id, template_key) WHERE template_key IS NOT NULL;` dan yang setara untuk `planner_playlists`). Tes: `TestRundownTemplateKeyIndexesMakeAppendIdempotent`.

**Belum diuji:** PostgreSQL 18 (versi di Compose), dan migrasi ini di atas database produksi sungguhan. Merge ke `main` menjalankannya di sana karena `MIGRATE_ON_START=true`; semuanya tabel baru sehingga tidak menyentuh data auth, dan rollback aplikasi ke image sebelumnya aman (kolom dan tabel lama tidak berubah).
