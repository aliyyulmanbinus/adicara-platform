# Menu 03: Keuangan & Anggaran — Backend Specification

Spesifikasi backend modul Go `plannerfinance` (`backend/internal/modules/plannerfinance/`) untuk tiga sub-modul: **Simulasi** (PDF Hal. 4), **Tabungan** (Hal. 7), dan **Realisasi** (Hal. 5–6), mengikuti alur $\text{Simulasi} \rightarrow \text{Tabungan} \rightarrow \text{Realisasi}$. Aturan lintas modul (uang `BIGINT`, master sesi, template vs data contoh, kode error) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md); titik sambung yang sudah disediakan Menu 01 (`SummaryProvider`, `TemplateApplier`, `SampleOverlay`) dijelaskan di [bab 3.12](#312-sambungan-ke-menu-01).

> **Status (2026-10-21):** modul `plannerfinance` **sudah diimplementasikan** di `backend/internal/modules/plannerfinance/` (`domain`, `repository`, `usecase`, `delivery/http`) dan terpasang di `internal/app/app.go`: 32 endpoint, provider `finance`, applier `budget`, overlay `finance`, dan registry jenis acara punya `contributor_roles`. Widget Keuangan di dashboard terisi, `GET /event-types` menawarkan `budget_lines`, dan Proyek Contoh berisi 14 pos PDF dengan pembayarannya, Arif dan Sinta dengan 22 setoran, dan tiga skenario (uji emas lulus). Tabel dan view (migrasi `008_000_planner_finance.sql` dan `010_000_planner_budget_lines_view.sql`) **identik** dengan blok SQL bab 1. Dokumen implementasi beserta penyimpangan dari spesifikasi: [docs/backend/planner/2026-10-21-planner-menu-03.md](../../../backend/planner/2026-10-21-planner-menu-03.md); kontrak API: `backend/docs/openapi.yaml` (tag `Planner Finance`). Frontend menu ini belum ada ([FE.md](FE.md)).
>
> Dokumen ini diaudit dengan **menjalankan** SQL dan algoritmanya sebelum kode ditulis (PostgreSQL 16.2 sementara, 12 migrasi nyata, data emas PDF; [bab 7.1](#71-yang-sudah-dijalankan-pada-audit-spesifikasi)), lalu kodenya diuji terhadap PostgreSQL 16.2 ([bab 7.2](#72-pengujian-pada-implementasi)). Blok DTO bab 5 adalah struct Go yang dipakai (disalin ke `domain/entity.go`) dan dibandingkan kunci demi kunci dengan tipe TypeScript FE (29 pasangan tipe, 329 jalur, nol selisih, juga terhadap struct yang sebenarnya) dan dengan skema `openapi.yaml` (31 skema, dijaga tes). Hal yang ditemukan saat membangunnya dicatat di [bab 9](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) butir 18–29. Yang **belum** dijalankan: frontend, Menu 04 (pos rollup dan vendor diuji dengan baris yang disisipkan tes), berkas `.xlsx` di Excel/Numbers/LibreOffice, image Docker, Nginx di depan rute baru, PostgreSQL 18 (versi Compose), beberapa replika API (pembatas laju ekspor di memori per proses).
>
> Pola dokumen mengikuti [01 BE](../01_DASHBOARD/BE.md) dan [02 BE](../02_TIMELINE_DAN_TUGAS/BE.md): DTO tanpa tag `binding` (validasi di usecase dengan pesan `<field>: <alasan>`), bentuk respons per endpoint ([bab 4.4](#44-bentuk-respons)), validasi dan batas ([bab 4.5](#45-validasi-dan-batas)), bab sambungan ke Menu 01 ([3.12](#312-sambungan-ke-menu-01)), pengujian dipetakan ke nama tes ([bab 7](#7-pengujian)), urutan pengerjaan ([bab 8](#8-urutan-pengerjaan-dan-definisi-selesai)), dan celah yang menunggu keputusan ([bab 9](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Audit ini mengoreksi versi sebelumnya (kesalahan, kontradiksi, dan bagian yang belum terdefinisi); daftarnya di [CHANGELOG spesifikasi](../CHANGELOG.md).

> **Acuan angka.** Seluruh angka bab 6 berasal dari PDF Hal. 4–7 (dibaca ulang dengan `pdfminer`) dan diuji terhadap basis data ([kontrak bab 10](../00_KONTRAK_BERSAMA/README.md#10-validasi-spesifikasi)): total dibayar **Rp 33.550.000**, sisa **Rp 40.600.000**, rasio **45,2%**; Lamaran belum lunas (dibayar 4.750.000, sisa 2.400.000); 14 pos memakai nama dan nilai PDF; kolom 10 pos × 6 paket berjumlah persis nama paketnya.

---

## 1. Skema Database

### 1.1 Tabel

Migrasi `008_000_planner_finance.sql` (sudah ada di `backend/internal/migrate/sql/`, dijalankan di PostgreSQL 16; teks di bawah **identik** dengan berkas itu). Ciri skema: uang `BIGINT`; preset sistem **tidak disimpan di tabel** (ada di kode, bab 6) sehingga `planner_budget_scenarios.project_id` selalu terisi; persentase dihitung saat dibaca; status pelunasan dihitung view dan tidak disimpan sehingga tidak bisa menyimpang; `active_sub_events` adalah `TEXT[]` berisi `scope_key`; FK komposit mencegah kontributor, vendor, atau dokumen proyek lain ditunjuk; `item_key`, `source`, `admin_doc_id`, `rollup_source`, dan penetapan harga per satuan mendukung integrasi.

```sql
-- +goose Up
CREATE TABLE planner_budget_scenarios (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    total_target BIGINT NOT NULL CHECK (total_target > 0),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    active_sub_events TEXT[] NOT NULL DEFAULT '{}',     -- scope_key yang ikut disimulasikan; kosong = semua sesi non-sistem
    preset_key TEXT,                                    -- asal salinan, mis. wedding_75jt; preset itu sendiri ada di kode
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

-- Paling banyak satu skenario aktif per proyek. Batas 3 skenario ditegakkan usecase
-- di dalam transaksi yang mengunci baris proyek (SELECT ... FOR UPDATE).
CREATE UNIQUE INDEX planner_budget_scenarios_one_active_idx ON planner_budget_scenarios (project_id) WHERE is_active;

CREATE TABLE planner_budget_scenario_items (
    id UUID PRIMARY KEY,
    scenario_id UUID NOT NULL,
    project_id UUID NOT NULL,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    item_key TEXT,                                      -- kunci pencocok Smart Adjust; NULL = pos kustom
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 150),
    cost_amount BIGINT NOT NULL DEFAULT 0 CHECK (cost_amount >= 0),
    sort_order INT NOT NULL DEFAULT 0,
    FOREIGN KEY (scenario_id, project_id)
        REFERENCES planner_budget_scenarios (id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_budget_scenario_items_scenario_idx ON planner_budget_scenario_items (scenario_id, sub_event_scope);
CREATE UNIQUE INDEX planner_budget_scenario_items_key_idx ON planner_budget_scenario_items (scenario_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL;

CREATE TABLE planner_contributors (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    role_label TEXT NOT NULL DEFAULT 'other',           -- divalidasi terhadap registry event type
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

CREATE TABLE planner_savings_entries (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    contributor_id UUID NOT NULL,
    entry_date DATE NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (contributor_id, project_id)
        REFERENCES planner_contributors (id, project_id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_savings_project_date_idx ON planner_savings_entries (project_id, entry_date, created_at);
CREATE INDEX planner_savings_contributor_idx ON planner_savings_entries (contributor_id);

CREATE TABLE planner_budget_items (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 150),
    item_key TEXT,                                      -- sama dengan planner_budget_scenario_items.item_key
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('template', 'scenario', 'vendor', 'admin_doc', 'manual')),
    vendor_id UUID,
    admin_doc_id UUID,
    rollup_source TEXT CHECK (rollup_source IN ('shopping')),  -- bila terisi, estimasi/terbayar dihitung dari daftar belanja
    pricing_mode TEXT NOT NULL DEFAULT 'lump' CHECK (pricing_mode IN ('lump', 'unit')),
    quantity NUMERIC(12, 2) CHECK (quantity > 0),
    unit_label TEXT,
    unit_price BIGINT CHECK (unit_price >= 0),
    initial_estimate BIGINT NOT NULL DEFAULT 0 CHECK (initial_estimate >= 0),

    dp_amount BIGINT NOT NULL DEFAULT 0 CHECK (dp_amount >= 0),
    dp_paid_at DATE,
    dp_receipt_url TEXT,
    term_1_amount BIGINT NOT NULL DEFAULT 0 CHECK (term_1_amount >= 0),
    term_1_paid_at DATE,
    term_1_receipt_url TEXT,
    term_2_amount BIGINT NOT NULL DEFAULT 0 CHECK (term_2_amount >= 0),
    term_2_paid_at DATE,
    term_2_receipt_url TEXT,

    paid_total BIGINT GENERATED ALWAYS AS (dp_amount + term_1_amount + term_2_amount) STORED,
    remaining_balance BIGINT GENERATED ALWAYS AS (initial_estimate - (dp_amount + term_1_amount + term_2_amount)) STORED,

    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    CHECK (pricing_mode = 'lump' OR (quantity IS NOT NULL AND unit_price IS NOT NULL)),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id),
    FOREIGN KEY (admin_doc_id, project_id)
        REFERENCES planner_admin_docs (id, project_id) ON DELETE SET NULL (admin_doc_id)
);

CREATE INDEX planner_budget_items_project_scope_idx ON planner_budget_items (project_id, sub_event_scope);
CREATE INDEX planner_budget_items_project_sort_idx ON planner_budget_items (project_id, sort_order);
CREATE UNIQUE INDEX planner_budget_items_key_idx ON planner_budget_items (project_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL;
CREATE UNIQUE INDEX planner_budget_items_vendor_idx ON planner_budget_items (vendor_id) WHERE vendor_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_budget_items;
DROP TABLE IF EXISTS planner_savings_entries;
DROP TABLE IF EXISTS planner_contributors;
DROP TABLE IF EXISTS planner_budget_scenario_items;
DROP TABLE IF EXISTS planner_budget_scenarios;
```

### 1.2 View baca `planner_budget_lines`

Semua angka anggaran (matriks, KPI, cashflow, dashboard, ekspor) dibaca dari satu view agar pos yang digulung dari daftar belanja ikut terhitung dan status pelunasan tidak pernah disimpan.

Migrasi `010_000_planner_budget_lines_view.sql` (dijalankan setelah `009_000_planner_shopping.sql`, karena view membaca `planner_shopping_items`; teks di bawah **identik** dengan berkas itu):

```sql
-- +goose Up
-- Model baca tunggal untuk semua angka anggaran (matriks, KPI, cashflow, dashboard).
-- Pos dengan rollup_source = 'shopping' mengambil estimasi dan terbayar dari
-- planner_shopping_items; lainnya memakai kolomnya sendiri. Status pelunasan
-- dihitung di sini, bukan disimpan, sehingga tidak bisa menyimpang.
CREATE VIEW planner_budget_lines AS
SELECT x.*,
       x.estimate - x.paid AS remaining,
       CASE
           WHEN x.paid = 0 THEN 'unpaid'
           WHEN x.estimate > 0 AND x.paid >= x.estimate THEN 'settled'
           WHEN x.rollup_source IS NULL AND x.dp_amount > 0 AND x.term_1_amount = 0 AND x.term_2_amount = 0 THEN 'dp_paid'
           ELSE 'partial'
       END AS payment_status
FROM (
    SELECT b.*,
           CASE WHEN b.rollup_source = 'shopping' THEN COALESCE(s.estimate_total, 0) ELSE b.initial_estimate END AS estimate,
           CASE WHEN b.rollup_source = 'shopping' THEN COALESCE(s.actual_total, 0) ELSE b.paid_total END AS paid
    FROM planner_budget_items b
    LEFT JOIN LATERAL (
        SELECT SUM(i.estimated_total)::bigint AS estimate_total,
               SUM(CASE WHEN i.status = 'selesai' THEN i.actual_total ELSE 0 END)::bigint AS actual_total
        FROM planner_shopping_items i
        WHERE i.budget_item_id = b.id
    ) s ON b.rollup_source = 'shopping'
) x;

-- +goose Down
DROP VIEW IF EXISTS planner_budget_lines;
```

Aturan status: `unpaid` bila `paid = 0` (termasuk pos berestimasi 0 yang belum diisi); `settled` bila `estimate > 0` dan `paid >= estimate`; `dp_paid` bila hanya DP terisi; selain itu `partial`. Pos berestimasi 0 **tidak** dihitung pada penghitung status KPI (belum dianggarkan). Kelebihan bayar menghasilkan `remaining` negatif dan ditandai klien sebagai "lebih bayar".

Tiga sifat view yang ditemukan dengan menjalankannya (bab 7.1) dan mengikat kode:

1. **Kunci baris lewat tabel, bukan view.** `SELECT … FROM planner_budget_lines … FOR UPDATE` ditolak PostgreSQL (`FOR UPDATE cannot be applied to the nullable side of an outer join`). Usecase mengunci baris dasar (`SELECT id FROM planner_budget_items WHERE id = $2 AND project_id = $1 FOR UPDATE`), lalu membaca view.
2. **Kolom view beku saat `CREATE VIEW`** (`b.*` dikembangkan sekali). Kolom yang kelak ditambahkan ke `planner_budget_items` tidak muncul di view; migrasi yang menambahkannya harus ikut `CREATE OR REPLACE VIEW`.
3. **`paid_total` dan `remaining_balance` bukan angka efektif untuk pos rollup.** Contoh yang dijalankan: pos `seserahan` dengan satu barang belanja selesai (estimasi 1.000.000, aktual 900.000): view memberi `estimate 1.000.000`, `paid 900.000`, `remaining 100.000`, sedangkan kolom tabel tetap `paid_total 0` dan `remaining_balance 3.500.000`. Aturan kode: **pembaca hanya memakai `estimate`, `paid`, `remaining`, `payment_status`** (DTO `BudgetLine` tidak mengirim kolom tabel itu); `paid_total` hanya dipakai view. Kolom `remaining_balance` tidak dibaca siapa pun ([bab 9, butir 13](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 2. Kunci Pencocok (`item_key`) dan Integrasi Pos

Pos dari skenario, template, vendor, dan dokumen harus saling mengenali agar tidak menggandakan pos (tanpa kunci, *Smart Adjust* tidak punya dasar pencocokan dan `lock-deal` vendor membuat pos bernama vendor yang bersaing dengan pos skenario):

- `item_key` (slug `^[a-z0-9_]{1,40}$`) sama pada `planner_budget_scenario_items` dan `planner_budget_items`; unik per `(sub_event_scope, item_key)` di tiap tabel (indeks parsial; pada skenario juga per `scenario_id`).
- Pos yang dibuat dari **template** (bab 6.2), **preset skenario** (6.1), dan **vendor** (04 BE §2.2) memakai kunci yang sama, jadi satu pos "Catering" resepsi tidak pernah muncul dua kali.
- Pos kustom pengguna boleh `item_key NULL`; ia tidak pernah ditimpa skenario. Formulir pos tidak memiliki isian kunci: pos buatan pengguna selalu tanpa kunci.
- Memindahkan pos ke sesi yang sudah punya kunci yang sama **menetralkan kunci** pos yang dipindah (menjadi `NULL`), sama dengan hapus sesi di [01 BE §4.6](../01_DASHBOARD/BE.md#46-sesi-acara-master-sesi) ([bab 3.10](#310-mengubah-memindahkan-dan-menghapus-pos)).

| Sumber (`source`) | Dibuat oleh | Boleh ditimpa skenario? |
|---|---|---|
| `template` | `apply-template` / wizard | estimasi saja (pembayaran tidak pernah disentuh) |
| `scenario` | Apply skenario (pos baru) | estimasi saja |
| `vendor` | `lock-deal` vendor (pos baru bila tak ada yang cocok) | estimasi saja |
| `admin_doc` | sinkron biaya dokumen (02 BE §4.5; pos baru dari sinkron tanpa `item_key`) | estimasi saja |
| `manual` | pengguna | tidak (tanpa `item_key`) |

---

## 3. Logika Bisnis

### 3.1 Batas 3 skenario (tanpa balapan)
`POST /scenarios`, `POST /scenarios/:id/clone` berjalan dalam transaksi yang mengunci baris proyek (`SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE`, `LockProject` seperti Menu 02), lalu menghitung skenario; bila ≥ 3 → `409 max_scenarios_reached`. Terbukti dengan dua sesi `psql` (sesi A membaca jumlah lalu menunggu 1,5 dtk sebelum menulis, sesi B mulai 0,4 dtk kemudian): **tanpa** kunci keduanya melihat 2 dan menulis, hasil **4 skenario**; **dengan** kunci B menunggu A dan melihat 3 (jadi menolak). Karena itu kunci ini bagian dari kontrak. Indeks `planner_budget_scenarios_one_active_idx` menjamin paling banyak satu skenario aktif.

Kunci proyek yang sama diambil oleh semua penulisan yang membaca-lalu-menulis banyak baris: `POST /scenarios`, `clone`, `load-preset`, `POST /scenarios/:id/items` (batas 100), `apply`, `POST /budget/items` (batas jumlah dan `max(sort_order)+1`), `reorder`, `apply-template`, `POST /savings`/`/contributors` (batas jumlah). `PUT` pos, pembayaran, dan hapus pembayaran mengunci **baris pos** (`FOR UPDATE` pada tabel dasar); penulisan lain (skenario, item skenario, kontributor, setoran) tidak membaca-lalu-menulis sehingga tidak perlu kunci. **Terbukti** pada kode (bab 7.2): enam `POST /scenarios` serentak pada dua skenario → tepat satu `201` (tiga putaran), delapan `POST /budget/items` pada 297 pos → tepat tiga, delapan pembayaran `full` serentak pada satu pos → tepat satu yang membayar; tanpa kunci baris pembayaran tes terakhir itu gagal.

Menghapus skenario yang aktif diizinkan dan **tidak mengaktifkan yang lain** (dijalankan: 0 aktif, item ikut terhapus oleh `CASCADE`); `budget_limit` vendor dari skenario ([04 BE §2.1](../04_VENDOR_DAN_BELANJA/BE.md#21-pagu-vendor-dari-skenario-aktif)) jatuh ke manual sampai pengguna menerapkan skenario lain. `clone` menyalin item, `active_sub_events`, `preset_key`, dan `total_target`, dengan `is_active = false`.

### 3.2 Menerapkan skenario (*Smart Adjust*)

`POST …/scenarios/apply-preview` (tanpa menulis) dan `POST …/scenarios/apply` memakai algoritma yang sama dalam satu transaksi dengan kunci proyek:

```text
scenario  := skenario terpilih
items     := item skenario dengan sub_event_scope ∈ active_sub_events ∪ {'all'}  (active_sub_events kosong = semua sesi)
untuk tiap item:
  bila item_key ≠ NULL:  pos := budget_items WHERE project, sub_event_scope, item_key
  bila item_key = NULL:  pos := budget_items WHERE project, sub_event_scope, lower(item_name) dan item_key IS NULL
                                (bila ada beberapa: yang sort_order terkecil)
  bila pos ada:
      rollup_source ≠ NULL  → LEWATI (laporkan reason "rollup")
      pricing_mode = 'unit'  → LEWATI (laporkan reason "unit_pricing"; harga per satuan diatur pengguna)
      nilai sama             → tidak dihitung "diperbarui" (idempoten)
      selain itu             → UPDATE initial_estimate = cost_amount   # DP/termin/vendor/dokumen TIDAK disentuh
  bila tidak ada            → INSERT pos (source='scenario', item_key, estimasi)
pos proyek yang tidak ada di skenario → tidak disentuh, tidak dihapus
aktifkan skenario ini         # DUA pernyataan: nonaktifkan yang lama dulu, baru aktifkan (lihat di bawah)
bila sync_target_budget   → planner_projects.target_budget = scenario.total_target
bila sync_target_savings  → planner_projects.savings_target = NULL   # ikut target_budget
```

SQL yang dijalankan (data emas pada proyek berisi 14 pos template berestimasi 0, satu DP 500.000 pada `fotografer`, satu pos kustom "Photobooth", dan skenario `wedding_75jt` dengan `active_sub_events = {akad, resepsi}`); `$1` = project_id, `$2` = scenario_id:

```sql
-- 0. kunci proyek
SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE;

-- 1. pasangan item skenario ↔ pos (dipakai apply-preview dan apply)
CREATE TEMP TABLE adj ON COMMIT DROP AS
WITH sc AS (SELECT active_sub_events FROM planner_budget_scenarios WHERE id = $2 AND project_id = $1),
it AS (
  SELECT i.* FROM planner_budget_scenario_items i, sc
  WHERE i.scenario_id = $2 AND i.project_id = $1
    AND (cardinality(sc.active_sub_events) = 0 OR i.sub_event_scope = 'all' OR i.sub_event_scope = ANY (sc.active_sub_events))
)
SELECT it.id AS item_id, it.sub_event_scope, it.item_key, it.item_name, it.category, it.cost_amount, it.sort_order,
       b.id AS budget_id, b.initial_estimate AS from_amount,
       CASE WHEN b.id IS NULL THEN 'create'
            WHEN b.rollup_source IS NOT NULL THEN 'skip_rollup'
            WHEN b.pricing_mode = 'unit' THEN 'skip_unit'
            WHEN b.initial_estimate = it.cost_amount THEN 'same'
            ELSE 'update' END AS action
FROM it LEFT JOIN LATERAL (
   SELECT * FROM planner_budget_items b
   WHERE b.project_id = $1 AND b.sub_event_scope = it.sub_event_scope
     AND ((it.item_key IS NOT NULL AND b.item_key = it.item_key)
       OR (it.item_key IS NULL AND b.item_key IS NULL AND lower(b.item_name) = lower(it.item_name)))
   ORDER BY b.sort_order, b.id LIMIT 1) b ON true;

-- 2. laporan: SELECT action, ... FROM adj  → updated / created / skipped;
--    untouched_count := pos proyek yang tidak punya pasangan:
SELECT count(*) FROM planner_budget_items b WHERE b.project_id = $1 AND NOT EXISTS (SELECT 1 FROM adj a WHERE a.budget_id = b.id);

-- 3. apply: perbarui estimasi, lalu sisipkan pos baru di urutan terakhir
UPDATE planner_budget_items b SET initial_estimate = a.cost_amount, updated_at = now() FROM adj a WHERE a.action = 'update' AND b.id = a.budget_id;
INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, initial_estimate, sort_order)
SELECT gen_random_uuid(), $1, a.sub_event_scope, a.category, a.item_name, a.item_key, 'scenario', a.cost_amount,
       (SELECT COALESCE(max(sort_order), 0) FROM planner_budget_items WHERE project_id = $1) + row_number() OVER (ORDER BY a.sort_order, a.item_id)
FROM adj a WHERE a.action = 'create';

-- 4. aktifkan: DUA pernyataan, urutan ini
UPDATE planner_budget_scenarios SET is_active = false, updated_at = now() WHERE project_id = $1 AND is_active;
UPDATE planner_budget_scenarios SET is_active = true,  updated_at = now() WHERE id = $2 AND project_id = $1;

-- 5. sinkron pagu (hanya bila diminta). Menulis planner_projects dari modul ini: pengecualian SQL ketiga (kontrak bab 1.1)
UPDATE planner_projects SET target_budget = $3, updated_at = now() WHERE id = $1;       -- sync_target_budget
UPDATE planner_projects SET savings_target = NULL, updated_at = now() WHERE id = $1;    -- sync_target_savings
```

Hasil yang dijalankan: `create 1, update 9`, `untouched 6` (5 pos Lamaran + "Photobooth"), total estimasi 76.500.000 (75.000.000 + pos kustom 1.500.000 → `unallocated` negatif, "melebihi plafon"), DP 500.000 pada `fotografer` tidak berubah, `source` pos yang diperbarui tetap `template`. **Menjalankan ulang pasangan yang sama → `same 10`** (idempoten, nol `update`, nol `create`).

**Pelajaran dari menjalankannya.** Versi awal spesifikasi menulis "aktifkan, nonaktifkan yang lain" yang wajar diterjemahkan satu `UPDATE … SET is_active = (id = $2)`. Dijalankan, **satu pernyataan itu gagal `23505` pada `planner_budget_scenarios_one_active_idx`** saat berpindah dari skenario 75 Jt ke 100 Jt (indeks unik non-ditunda diperiksa per baris; apakah lolos bergantung pada urutan fisik baris, jadi kadang berhasil dan kadang gagal: pindah-balik dengan pernyataan yang sama kebetulan lolos). Dua pernyataan berurutan (langkah 4) selalu berhasil.

`apply-preview` mengembalikan `ApplyReport` ([bab 5](#5-dto-go)) agar antarmuka menampilkan daftar perubahan sebelum konfirmasi; `apply` mengembalikan laporan yang sama untuk apa yang benar-benar dilakukan. `estimate_after` = Σ estimasi semua pos sesudah penerapan (dasar "sisa plafon"); `new_target_budget` ada bila `sync_target_budget`.

Mode `clean_overwrite` menghapus **semua** pos proyek (pembayaran ikut hilang; tautan belanja `budget_item_id` menjadi kosong, `vendor_id`/`admin_doc_id` hilang bersama barisnya) lalu menyalin item skenario yang `included`; wajib `confirm: "RESET"` (`400 invalid_input: confirm: required` bila kosong, `confirm: must be RESET` bila salah). Pratinjaunya mengisi `deleted {lines, paid_total}` supaya pengguna melihat kerugiannya (dijalankan: 16 pos dan Rp 500.000 pembayaran hilang, 10 pos berestimasi 75.000.000 dan terbayar 0 tersisa). Tanpa `confirm`, `apply-preview` untuk `clean_overwrite` tetap boleh (hanya membaca).

**Ketentuan yang ditetapkan saat implementasi** (spesifikasi diam): (1) `mode` kosong pada `apply` / `apply-preview` = `smart_adjust` (tidak merusak), `scenario_id` kosong → `400 invalid_input: scenario_id: required`; (2) dua item skenario **tanpa kunci** bernama sama (tanpa membedakan huruf besar-kecil) berpasangan dengan pos yang sama: item pertama memperbarui, **item kedua dibuat sebagai pos sendiri**, karena `UPDATE … FROM` dengan target ganda tidak terdefinisi; (3) pos yang dibuat `source = 'scenario'`, `sort_order` melanjutkan urutan proyek menurut urutan item skenario; (4) laporan dihitung lebih dulu oleh fungsi murni (`smartAdjustPlan`, `cleanOverwritePlan`) dari pasangan yang dibaca satu `SELECT`, jadi `apply-preview` dan `apply` memberi laporan yang sama (diuji) dan `estimate_after` adalah angka rencana yang dicocokkan dengan view setelahnya; (5) batas 300 pos diperiksa **sebelum** satu pun pos ditulis (`409 budget_item_limit_reached`, tidak ada yang berubah); (6) `compare` menghitung semua item kedua skenario, bukan hanya yang `included`.

Contoh nyata (diuji pada data bab 6.3, tanpa pos kustom): proyek dengan 14 pos template (estimasi 0) menerima preset `wedding_75jt` → 9 pos diperbarui (`akad/biaya_kua` 600.000; `resepsi/mahar`, `catering`, `rias_busana`, `venue_dekor`, `undangan_souvenir`, `dokumentasi`, `seserahan`, `hiburan`), 1 pos baru (`all/dana_darurat` 1.000.000), 5 pos Lamaran tak tersentuh; total estimasi 75.000.000.

### 3.3 Mencatat dan menghapus pembayaran
`PATCH …/budget/items/:item_id/payment` menulis **satu tahap** pada kolom pos (PDF punya tiga tahap tetap: DP/Lunas, Termin 1, Termin 2). Dalam satu transaksi, baris dasar dikunci (bab 1.2 butir 1):

- Tahap `dp | term_1 | term_2` **mengganti seluruh isi tahap itu** (nominal, `paid_at`, `receipt_url`; koreksi dimungkinkan). `amount` wajib dan > 0. `receipt_url` yang tidak dikirim mengosongkan bukti tahap itu (klien mengisi awal dengan nilai lama).
- Tahap `full` mengisi **sisa** (`remaining` dari view) ke tahap kosong pertama menurut urutan DP, Termin 1, Termin 2; `amount` pada permintaan **diabaikan** (server memakai sisa saat menulis, jadi klik ganda tidak menghasilkan dua pembayaran). Jawaban `400 invalid_stage` bila ketiga tahap terisi **atau** `remaining ≤ 0` (tidak ada yang dilunasi).
- Pos dengan `rollup_source` menolak pembayaran: `409 rollup_locked` (angkanya dihitung dari daftar belanja).
- Kelebihan bayar diizinkan (status tetap `settled` bila `estimate > 0`, `remaining` negatif).
- `paid_at` wajib (`YYYY-MM-DD`, dalam jendela tanggal bab 4.5). Pada Proyek Contoh `paid_at` kosong karena PDF tidak memuat tanggal pembayaran.

`DELETE …/budget/items/:item_id/payment/:stage` (`:stage` ∈ `dp | term_1 | term_2`) **mengosongkan** satu tahap (nominal 0, `paid_at` dan `receipt_url` `NULL`) dan mengembalikan `BudgetLine`. Versi awal spesifikasi tidak punya cara membatalkan pembayaran yang salah catat (nominal wajib > 0 dan tidak ada hapus). `rollup_locked` berlaku juga.

SQL yang dijalankan (`$1` project, `$2` pos; stage dipilih usecase dari daftar putih, bukan dari teks klien):

```sql
SELECT id FROM planner_budget_items WHERE id = $2 AND project_id = $1 FOR UPDATE;        -- bukan view (bab 1.2)
-- full: tahap kosong pertama menerima sisa (contoh term_1 saat DP terisi dan T1 kosong)
WITH cur AS (SELECT b.id, l.remaining, b.dp_amount, b.term_1_amount, b.term_2_amount
             FROM planner_budget_items b JOIN planner_budget_lines l ON l.id = b.id WHERE b.id = $2 AND b.project_id = $1)
UPDATE planner_budget_items b SET term_1_amount = c.remaining, term_1_paid_at = $3, term_1_receipt_url = $4, updated_at = now()
FROM cur c WHERE b.id = c.id AND c.remaining > 0 AND c.dp_amount > 0 AND c.term_1_amount = 0;
-- hapus tahap
UPDATE planner_budget_items SET term_1_amount = 0, term_1_paid_at = NULL, term_1_receipt_url = NULL, updated_at = now() WHERE id = $2 AND project_id = $1;
```

Hasil: pos `fotografer` (estimasi 1.000.000, DP 500.000) + `full` → `term_1 = 500.000`, `paid 1.000.000`, `remaining 0`, `settled`; dihapus kembali → `dp_paid`, `remaining 500.000`.

### 3.4 Pos berharga satuan
`pricing_mode = 'unit'`: usecase menghitung `initial_estimate = round(quantity × unit_price)` (estimasi dari klien diabaikan). Dipakai pos seperti Katering (porsi × harga/porsi); angka porsi dapat diisi dari proyeksi RSVP (05 BE §2.3) atas tindakan pengguna, tidak otomatis. Aturan yang ditemukan dengan menjalankannya:

- Perkalian dilakukan dengan aritmetika bilangan bulat atau `math/big` (`quantity` dibaca sebagai ratusan: `NUMERIC(12,2)` → `q100`), **bukan** `float64`, dan hasilnya diperiksa **sebelum** ditulis: `round(9999999999.99 × 10¹³)` menghasilkan `bigint out of range` di PostgreSQL (galat 500 bila tidak dijaga). Batas: `0,01 ≤ quantity ≤ 1.000.000` (dua desimal), `unit_price` 0…10¹³, hasil ≤ 10¹³, selain itu `400 invalid_input: unit_price: the estimate (quantity × unit_price) must be at most 10000000000000`.
- `pricing_mode = 'unit'` mewajibkan `quantity` dan `unit_price` (`required when pricing_mode is unit`; CHECK basis data sama).
- Pos `rollup` mengabaikan `initial_estimate`, `pricing_mode`, `quantity`, dan `unit_price` pada `PUT` (angkanya dari belanja); antarmuka menonaktifkan isiannya dan menampilkan angka efektif dari respons.

### 3.5 Likuiditas
```sql
-- $1 = project_id; dari view; sama dengan yang dipakai dashboard (01 BE §5.2)
SELECT COALESCE(sum(paid), 0) AS paid,
       COALESCE(sum(GREATEST(remaining, 0)), 0)  AS remaining,   -- kewajiban nyata: bagian positif per pos
       COALESCE(sum(GREATEST(-remaining, 0)), 0) AS overpaid     -- kelebihan bayar pada pos tertentu
FROM planner_budget_lines WHERE project_id = $1;
SELECT COALESCE(sum(amount), 0) AS collected FROM planner_savings_entries WHERE project_id = $1;
```
`available_cash = collected − paid`; `cash_deficit = max(remaining_debt − available_cash, 0)` (`has_deficit` tidak dikirim: `cash_deficit > 0`). `remaining` memakai bagian positif per pos karena kelebihan bayar pada satu pos tidak mengurangi kewajiban pos lain (selisih mentah `Σ estimate − Σ paid` menyembunyikan kewajiban: pos A lebih bayar Rp 5 jt dan pos B kurang Rp 10 jt = kewajiban nyata Rp 10 jt, bukan Rp 5 jt); `overpaid` ditampilkan terpisah. Angka yang sama dipakai widget dashboard ([01 BE §5](../01_DASHBOARD/BE.md#5-dashboard-get-v1plannerprojectsiddashboard)). Dijalankan pada data PDF: `47.500.000 − 33.550.000 = ` **13.950.000** kas bebas; sisa kewajiban 40.600.000 → defisit **26.650.000**, `overpaid 0`. Dengan satu pos diubah menjadi lebih bayar Rp 5.000.000 (estimasi 2.000.000, dibayar 7.000.000): selisih mentah `Σ estimate − Σ paid` = 35.600.000 tetapi `remaining_debt` tetap **40.600.000** dan `overpaid` 5.000.000. Angka yang sama pada kueri provider 01 BE §5.2 (dijalankan: `7 / 7 / 0` pos lunas / berjalan / belum).

### 3.6 Rekomendasi tabungan dan milestone bertahap
Target tabungan = `COALESCE(savings_target, target_budget)` (satu sumber di `planner_projects`). Menulisnya **tidak** punya endpoint Menu 03: pengguna memakai `PATCH /v1/planner/projects/:id` Menu 01 dengan `savings_target` atau `clear_savings_target: true` (kembali mengikuti pagu; sudah ada dan teruji di kode Menu 01). Versi awal spesifikasi punya `PATCH /savings/target {"target_savings": …}`: dua jalur menulis kolom yang sama dengan nama kolom berbeda, dan `null` pada struct Go tidak dapat dibedakan dari "tidak dikirim" (alasan Menu 01 memakai `clear_*`).

Waktu "sekarang" adalah **parameter** (`$3`, tanggal hari ini di zona proyek yang dihitung usecase, seperti kueri Hub dan provider Menu 02), bukan `now()` di SQL, agar uji emas deterministik.

- **Milestone** = sesi non-sistem yang punya `session_date`, diurut `session_date, sort_order`. `target_amount` = jumlah `estimate` pos pada sesi itu (dari view). Dana terkumpul dialokasikan berurutan (FIFO) ke milestone: `secured_amount = min(max(terkumpul − Σ target milestone sebelumnya, 0), target_amount)`; `is_secured = secured_amount = target_amount`. Sesi tanpa tanggal dan sesi sistem `all` tidak membentuk milestone; jumlah estimasinya dilaporkan sebagai `unscheduled_estimate`.
- **Rekomendasi bulanan**: `target_date` = `session_date` terakhir milik sesi non-sistem (`basis: "session"`), selain itu `event_date` (`basis: "event_date"`); `shortfall = max(target − collected, 0)`; `months_left = max(1, ceil((target_date − hari_ini) ÷ 30))`; `recommended_monthly = ceil(shortfall ÷ months_left ÷ 1000) × 1000`; `per_contributor = ceil(recommended_monthly ÷ jumlah kontributor ÷ 1000) × 1000` (dihilangkan bila belum ada kontributor; "aktif" berarti semua kontributor proyek). Tanggal tiap sub-acara berasal dari `planner_event_sessions.session_date`.

```sql
-- $1 = project_id, $3 = hari ini (DATE)
WITH col AS (SELECT COALESCE(sum(amount), 0)::bigint AS c FROM planner_savings_entries WHERE project_id = $1),
ms AS (
  SELECT s.scope_key, s.name, s.session_date, s.sort_order,
         COALESCE((SELECT sum(estimate) FROM planner_budget_lines l WHERE l.project_id = s.project_id AND l.sub_event_scope = s.scope_key), 0)::bigint AS target
  FROM planner_event_sessions s WHERE s.project_id = $1 AND NOT s.is_system AND s.session_date IS NOT NULL),
cum AS (SELECT ms.*, COALESCE(sum(target) OVER (ORDER BY session_date, sort_order ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING), 0)::bigint AS before FROM ms)
SELECT scope_key, name, session_date, target AS target_amount,
       LEAST(GREATEST(col.c - before, 0), target) AS secured_amount,
       LEAST(GREATEST(col.c - before, 0), target) = target AS is_secured
FROM cum, col ORDER BY session_date, sort_order;

-- unscheduled_estimate
SELECT COALESCE(sum(l.estimate), 0) FROM planner_budget_lines l
LEFT JOIN planner_event_sessions s ON s.project_id = l.project_id AND s.scope_key = l.sub_event_scope
WHERE l.project_id = $1 AND (s.session_date IS NULL OR s.is_system);

-- months_left (target_date dari ms/event_date; shortfall dari target − col.c)
SELECT GREATEST(1, ceil((DATE '2027-03-28' - $3::date) / 30.0))::int AS months_left;
```

Dijalankan pada data PDF (target 75.000.000, terkumpul 47.500.000, `target_date` 28 Mar 2027): Lamaran 7.150.000 terjamin penuh, Akad 600.000 terjamin penuh, Resepsi **39.750.000 dari 66.400.000 (59,9%)**; `unscheduled_estimate 0`. Rekomendasi: 27.500.000 ÷ 6 → **Rp 4.584.000/bulan** (2.292.000 per orang) pada hari ini = 4 Okt 2026 (175 hari → 6 bulan) **dan** pada 21 Okt 2026 (158 hari → 6 bulan); pada 27 Mar 2027 (1 hari) dan 2 Apr 2027 (lampau) `months_left` terjepit 1 sehingga 27.500.000. Angka 4.584.000 hanya stabil untuk hari ini ∈ (150, 180] hari sebelum Hari-H: uji emas wajib memasang jam tetap.

### 3.7 Integrasi dengan modul lain
| Arah | Mekanisme |
|---|---|
| 04 Vendor → pos | `lock-deal`: tautkan ke pos khusus yang cocok `(scope, item_key)` atau buat pos baru (pos bersama tidak ditimpa); unik `planner_budget_items_vendor_idx` (satu vendor, satu pos) |
| 04 Belanja → pos | `planner_shopping_items.budget_item_id` + `rollup_source='shopping'`; angka dari view |
| 02 Dokumen → pos | `admin_doc_id` (02 BE §4.5); sinkron membuat pos tanpa `item_key` bersumber `admin_doc` atau menautkan pos `(akad, biaya_kua)` yang sudah ada |
| 05 Tamu → pos katering | `pricing_mode='unit'` + tombol "Gunakan proyeksi RSVP" (muncul pada pos berharga satuan mana pun; pengguna memutuskan pos mana) |
| Skenario ← 04 | `budget_limit` vendor dicari dari skenario aktif lewat `category_key → budget_item_keys` (04 BE §2.1) |
| 03 → 01 Dashboard | `SummaryProvider` `finance` ([3.12](#312-sambungan-ke-menu-01)) |

Penerimaan dana dari tamu (angpau/amplop digital lewat gateway) **tidak** dicatat di Tab Tabungan: Adicara tidak menampung dana, dan layanan Gift belum punya spesifikasi ([kontrak bab 8 #13](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung)).

### 3.8 Matriks: sesi dan bulanan

**Matriks sesi** (`GET /budget/matrix`) dibangun dari view; kolom `dp`, `term_1`, `term_2` hanya menjumlah pos **non-rollup**, `rollup_paid` menjumlah `paid` pos rollup, jadi `paid = dp + term_1 + term_2 + rollup_paid`. Baris `remaining` = Σ bagian positif per pos dan `overpaid` = Σ bagian negatif, sehingga `estimate − paid = remaining − overpaid` selalu berlaku (diuji terhadap data emas). Urutan baris = urutan sesi (`sort_order`), sesi sistem `all` ("Umum") terakhir; hanya sesi yang punya pos. `total` = jumlah semua baris (`sub_event_scope: ""`).

```sql
-- $1 = project_id
WITH l AS (SELECT * FROM planner_budget_lines WHERE project_id = $1)
SELECT l.sub_event_scope,
       sum(l.estimate)::bigint AS estimate,
       COALESCE(sum(l.dp_amount)     FILTER (WHERE l.rollup_source IS NULL), 0)::bigint AS dp,
       COALESCE(sum(l.term_1_amount) FILTER (WHERE l.rollup_source IS NULL), 0)::bigint AS term_1,
       COALESCE(sum(l.term_2_amount) FILTER (WHERE l.rollup_source IS NULL), 0)::bigint AS term_2,
       COALESCE(sum(l.paid)          FILTER (WHERE l.rollup_source IS NOT NULL), 0)::bigint AS rollup_paid,
       sum(l.paid)::bigint AS paid, sum(GREATEST(l.remaining, 0))::bigint AS remaining, sum(GREATEST(-l.remaining, 0))::bigint AS overpaid,
       count(*) AS line_count
FROM l LEFT JOIN planner_event_sessions s ON s.project_id = l.project_id AND s.scope_key = l.sub_event_scope
GROUP BY l.sub_event_scope
ORDER BY min(CASE WHEN s.is_system THEN 1000000 ELSE s.sort_order END);
-- kpi.line_status: count(*) FILTER (WHERE estimate > 0 AND payment_status = 'settled' | IN ('dp_paid','partial') | = 'unpaid')
-- active_scenario: skenario is_active; unallocated = total_target − Σ estimate semua pos
```
Hasil pada data emas: Lamaran `7.150.000 / 3.250.000 / 1.000.000 / 500.000 / 4.750.000 / 2.400.000`, Akad `600.000 / 600.000 / 0 / 0 / 600.000 / 0`, Resepsi `66.400.000 / 16.200.000 / 11.000.000 / 1.000.000 / 28.200.000 / 38.200.000`, total `74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000`; `line_status {7, 7, 0}`; skenario aktif 75 Jt → `unallocated 850.000`. Baris "Subtotal Kategori" pada PDF Hal. 5 (19.050.000 / 1.800.000 / 161.000.000 / 181.850.000) **tidak** dibuat: ia menjumlah enam baris yang tumpang-tindih (tagihan, DP, termin, dibayar, sisa) sehingga tidak bermakna.

**Matriks bulanan tabungan** (`savings.matrix`): bulan **berurutan tanpa celah** dari bulan setoran pertama sampai terakhir (bulan tanpa setoran tetap satu baris berisi 0; dijalankan dengan menghapus Mei–Juni → kedua baris muncul 0), `by_contributor` memuat **setiap** kontributor proyek (0 bila kosong). Tanggal setoran bebas, jadi satu sel dapat menjumlah beberapa setoran.

```sql
-- $1 = project_id
WITH r AS (SELECT date_trunc('month', min(entry_date))::date a, date_trunc('month', max(entry_date))::date z FROM planner_savings_entries WHERE project_id = $1),
m AS (SELECT generate_series(r.a, r.z, interval '1 month')::date AS month FROM r)
SELECT to_char(m.month, 'YYYY-MM') AS month, c.id AS contributor_id, COALESCE(sum(e.amount), 0) AS amount
FROM m CROSS JOIN planner_contributors c
LEFT JOIN planner_savings_entries e ON e.contributor_id = c.id AND date_trunc('month', e.entry_date) = m.month
WHERE c.project_id = $1 GROUP BY m.month, c.id ORDER BY m.month, c.sort_order;
```
`GET /savings?contributor_id=` hanya menyaring `entries`; KPI, kartu, milestone, rekomendasi, dan matriks selalu seluruh proyek. `entries` urut `entry_date, created_at, id`.

### 3.9 Template pos dan `apply-template`

Dua pintu memakai kode yang sama: `TemplateApplier` `budget` (wizard, mode `append` di dalam transaksi pembuatan proyek) dan `POST /budget/apply-template` (pengguna). Data template di kode (`plannerfinance/domain/template_wedding.go`, `template_percent.go`, dibangkitkan dari tabel bab 6 dan dijaga tes yang membaca tabel itu lagi, seperti `TestTemplatesMatchSpec` Menu 02):

- **Pernikahan:** 14 pos bab 6.2, `source = 'template'`, estimasi **0**, nol pembayaran, tiap pos di sesinya (`lamaran`, `akad`, `resepsi`). Sesi yang sudah dihapus pengguna → pos jatuh ke `all` dengan warning `scope_fallback` (`count` = pos yang jatuh).
- **Jenis lain:** pos [TEMPLATE_EVENT bab 15.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#151-pos-dan-persentase), `sub_event_scope = all`, estimasi = `round(persen × target_budget ÷ 100)` dibulatkan ke Rp 1.000 (setengah naik) dengan selisih pembulatan ke `dana_darurat` (atau pos terakhir); `target_budget` di bawah Rp 100.000 (termasuk 0) → semua 0 (di atas batas itu selisih pembulatan tidak pernah negatif; di bawahnya wizard dengan pagu kecil melanggar `CHECK (initial_estimate >= 0)` dan menjadi 500, ditemukan saat implementasi); `category` pos dari kolom Kategori tabel itu (kolom yang ditambahkan karena `planner_budget_items.category` bertipe `NOT NULL`). **Estimasi 0 hanyalah sifat template pernikahan**; template persentase membawa estimasi acuan (selaras kontrak bab 5 "struktur + estimasi acuan"). Preset skenario `<jenis>_pct` memakai pembulatan yang sama atas `total_target`, dan **menolak `total_target < 100.000`** (`400 invalid_input: total_target: must be at least 100000`): dijalankan atas 8 jenis × semua kelipatan Rp 1.000 sampai 300.000 plus 1.234.567 / 33.333.333 / 75.000.000 / 10⁹, selisih pembulatan menjadi **negatif** (melanggar `CHECK cost_amount >= 0`) hanya untuk total ≤ Rp 19.000.

`POST /budget/apply-template` `{mode, sub_event_scope?, confirm?}` (`sub_event_scope` membatasi pada pos satu sesi, dipakai kotak centang sesi di Tab 1):

- **`append`** menambah pos template yang `(sesi, item_key)`-nya belum ada. Dijalankan: `INSERT … ON CONFLICT (project_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL DO NOTHING` memberi `created 2, skipped 1` bila dua dari tiga pos terhapus; idempoten.
- **`replace_template`** menghapus pos bersumber `template` **yang belum punya pembayaran** (`paid = 0` menurut view), lalu mengisi ulang; yang sudah dibayar dilewati dan dihitung `skipped`. `confirm: true` **wajib** bila ada pos yang akan dihapus yang membawa pekerjaan pengguna: estimasi efektif > 0 (mis. hasil Apply skenario; pos `template` yang diisi skenario tetap bersumber `template`), tautan vendor, tautan dokumen, atau tautan belanja/rollup; tanpa itu `400 invalid_input: confirm: required`. Pos itu kembali berestimasi 0 (atau persentase) dan tautannya hilang (belanja tetap hidup dengan `budget_item_id` kosong, dijalankan). Respons memuat warning `budget_link_lost` (`count` = pos bertaut yang dihapus).
- Batas 300 pos per proyek berlaku setelah penyisipan; semua-atau-tidak.

```sql
-- $1 = project_id: syarat replace_template (dijalankan pada 14 pos template setelah Apply 75 Jt, satu DP, satu rollup)
SELECT count(*) FILTER (WHERE l.paid = 0) AS deletable,
       count(*) FILTER (WHERE l.paid > 0) AS skipped_with_payment,
       count(*) FILTER (WHERE l.paid = 0 AND (l.estimate > 0 OR l.vendor_id IS NOT NULL OR l.admin_doc_id IS NOT NULL OR l.rollup_source IS NOT NULL
                          OR EXISTS (SELECT 1 FROM planner_shopping_items s WHERE s.budget_item_id = l.id))) AS at_risk
FROM planner_budget_lines l WHERE l.project_id = $1 AND l.source = 'template';
-- confirm wajib bila at_risk > 0. Hasil: deletable 13, skipped_with_payment 1, at_risk 9.
```

### 3.10 Mengubah, memindahkan, dan menghapus pos

`PUT …/budget/items/:item_id` **mengganti penuh** ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)): field opsional yang tidak dikirim dikosongkan, termasuk **`vendor_id` yang tidak dikirim melepas vendor** (formulir wajib mengirim semua isiannya, termasuk vendor yang sudah tertaut lewat `lock-deal`). Kolom yang tidak ada di body tidak berubah: `source`, `item_key`, `admin_doc_id`, `rollup_source`, pembayaran. Baris dikunci `FOR UPDATE`.

- **Vendor.** Menautkan vendor yang sudah tertaut ke pos lain melanggar `planner_budget_items_vendor_idx` (`23505`): usecase memetakan nama indeks itu ke `409 vendor_already_linked`; vendor proyek lain → `400 invalid_input: vendor_id: must be a vendor of this project`.
- **Pindah sesi.** Mengubah `sub_event_scope` ke sesi yang sudah punya `item_key` yang sama melanggar `planner_budget_items_key_idx` (**`23505` dijalankan**). Aturannya mengikuti [01 BE §4.6](../01_DASHBOARD/BE.md#46-sesi-acara-master-sesi): pos yang dipindah **menjadi `item_key = NULL`** (pos biasa) alih-alih gagal; data pembayaran utuh. Dijalankan: `Catering` Lamaran dipindah ke Resepsi menghasilkan dua pos Catering, satu berkunci dan satu tanpa kunci. FE tidak bisa menghitung bentrok itu, jadi memberi catatan tetap.
- **Hapus.** `DELETE …/budget/items/:item_id` → `204`. Pembayaran pos ikut hilang (antarmuka meminta konfirmasi bila `paid > 0`); `planner_shopping_items.budget_item_id` dan `planner_admin_docs` (lewat `budget_item_id` yang dihitung `LEFT JOIN`) kehilangan tautan, angkanya tetap; tidak ada penghapusan berantai ke vendor. Pos `rollup` boleh dihapus.
- **Urut ulang.** `POST …/budget/items/reorder` `{ids}`: daftar lengkap ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)), `sort_order = 1…n` dalam satu `UPDATE … FROM (VALUES …)`.

### 3.11 Ekspor `.xlsx`

`GET …/budget/export?format=xlsx` (selain itu `400 invalid_input: format`) menghasilkan buku kerja yang menyerupai PDF Hal. 5–7, dengan `excelize` (disetujui pemilik 2026-10-21 untuk Menu 02, dipakai bersama). **Diimplementasikan** menurut tata letak di bawah dan dibaca kembali dengan `excelize` oleh tes (sel angka, label, awalan `'`); **belum dibuka** di Excel, Numbers, atau LibreOffice. Baris "Dibayar dari belanja" (pos rollup) dan "Lebih bayar" di lembar `Rekap` hanya ditulis bila totalnya tidak nol.

| Lembar | Isi |
|---|---|
| `Rekap` | matriks sesi (baris: Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa; kolom: sesi + Total); angka sebagai bilangan, format `"Rp" #,##0`; skenario aktif dan sisa plafon di bawahnya |
| `Anggaran` | satu blok per sesi seperti PDF ("ANGGARAN LAMARAN", dst.): No, Kegiatan, Tagihan (Awal), DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan (Sisa), Status; baris Total per sesi; pos rollup diberi catatan "dihitung dari belanja" |
| `Tabungan` | matriks bulanan (bulan × kontributor + Jumlah), target, terkumpul, kekurangan (PDF Hal. 7) |

Teks pengguna (nama pos, kontributor, catatan) ditulis sebagai string dan diberi awalan `'` bila diawali `=`, `+`, `-`, `@` ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan) butir 5; tampak sebagai karakter, keputusan di 02 BE bab 8 butir 12). Nama berkas `keuangan-<slug>-<tanggal>.xlsx` dari `ginutil.Slug`; `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`; 10 ekspor/menit/pengguna (`429 rate_limited`, pembatas per proses seperti Menu 02); proyek `archived` tetap boleh mengekspor. Semua pos proyek, bukan hasil saringan sesi.

### 3.12 Sambungan ke Menu 01

`plannerfinance.Service` memenuhi tiga antarmuka dari `planner/domain` (`repository.go`, kode yang sudah ada), masing-masing dengan **satu** kunci (berbeda dengan Menu 02 yang punya dua):

| Antarmuka | Kunci | Isi |
|---|---|---|
| `SummaryProvider` | `finance` | `domain.FinanceSummary` (`planner/domain/summary.go`, **sudah ada**): `target_budget, total_estimate, total_paid, remaining, overpaid, savings_target, savings_collected, free_cash, cash_deficit, line_status{settled, partial, unpaid}` |
| `TemplateApplier` | `budget` | `Apply(ctx, projectID, ApplyRequest)` memakai `req.Project` (`EventType`, `TargetBudget`); `req.Mode` selalu `append` dari wizard; menghidupkan `templates_available.budget_lines` |
| `SampleOverlay` | `finance` | dataset PDF di Proyek Contoh (di bawah) |

Pemasangan di `internal/app/app.go` (menambah ke baris Menu 02, registry dan `TxManager` adalah instance yang sama):

```go
fin := financeusecase.New(financerepo.NewPostgresRepository(pool), plannerTx, registry, financeusecase.Options{})
plannerSvc.RegisterSummary(timelineSvc.TimelineProvider(), timelineSvc.AdminDocsProvider(), fin.Provider())
plannerSvc.RegisterTemplateAppliers(timelineSvc.TimelineApplier(), timelineSvc.AdminDocsApplier(), fin.Applier())
plannerSvc.RegisterSampleOverlays(timelineSvc.TimelineOverlay(), timelineSvc.AdminDocsOverlay(), fin.Overlay() /* , vendor, guest */)
financehttp.RegisterRoutes(ginEngine, idn, plannerSvc, fin)
```

Urutan overlay penting: overlay `finance` berjalan **sebelum** overlay vendor (Menu 04 menautkan `vendor_id` ke pos `catering` dan `dokumentasi` yang dibuat template anggaran, [04 BE bab 5.2](../04_VENDOR_DAN_BELANJA/BE.md)) dan sesudah overlay Menu 02 (tanggal sesi Lamaran memengaruhi milestone).

**Kueri provider.** Kueri `finance` di [01 BE §5.2](../01_DASHBOARD/BE.md#52-query-per-provider) sudah benar; dijalankan terhadap data emas: `estimate 74.150.000, paid 33.550.000, remaining 40.600.000, overpaid 0, settled 7, partial 7, unpaid 0`, terkumpul 47.500.000 → `free_cash 13.950.000`, `cash_deficit 26.650.000` (sama dengan fixture dashboard 01 FE).

**Registry.** Menu ini menambah `ContributorRoles []Option` ke `EventTypeSpec` (nilai: [TEMPLATE_EVENT bab 4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori), sembilan jenis; semuanya memuat `other`); `GET /event-types` mengirimnya sebagai `contributor_roles: [{key, label}]` (label Indonesia dari registry, mis. `groom` → "Mempelai Pria"). `role_label` divalidasi terhadapnya; bawaan `other`.

**Overlay Proyek Contoh** (hanya dari `POST /projects/sample` dan tes, dalam transaksi yang sama, setelah template anggaran diterapkan):
- **estimasi dan pembayaran** dataset bab 6.3 diterapkan pada 14 pos template menurut `(sesi, item_key)` (satu `UPDATE` per pos; template membawa estimasi 0, jadi estimasi PDF ikut ditulis); `paid_at` dan `receipt_url` kosong (PDF tidak memuatnya); `target_budget` sudah 75.000.000 dari pembuatan proyek;
- dua kontributor dan 22 setoran bab 6.4 (tanggal 1 tiap bulan Feb–Des 2026; pada contoh yang dibuat Okt 2026 setoran Nov–Des 2026 berdatang kemudian tetapi **dihitung** terkumpul, karena PDF mencatatnya begitu);
- tiga skenario bab 6.6 (Opsi Hemat 50 Jt, **Opsi Ideal 75 Jt (aktif)**, Opsi 100 Jt), `active_sub_events = {akad, resepsi}`.

Skenario aktif 75 Jt tidak diterapkan ke pos contoh (estimasi pos PDF, mis. Catering Resepsi 25.000.000, berbeda dari preset 30.000.000); jadi `unallocated = 75.000.000 − 74.150.000 = 850.000` seperti wireframe FE.

---

## 4. Endpoint

`/v1/planner/projects/:id/...`, dengan `BearerAuth` + `ProjectGuard` + `WriteGuard` (rantai middleware dan pemasangan di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)). Respons `{"data": ...}`, galat `{"error": {code, message}}`. Id anak di URL (`:scenario_id`, `:item_id`, `:contributor_id`, `:entry_id`) harus UUID kanonik 36 karakter, selain itu `404 not_found`. Rute statis (`/scenarios/compare`, `/scenarios/apply`, `/scenarios/apply-preview`, `/budget/items/reorder`, `/budget/matrix`, …) berdampingan dengan rute berparameter; Gin menyelesaikan segmen statis lebih dulu (terbukti untuk `/projects/summary` di Menu 01). Penulisan di proyek `archived` ditolak `409 project_read_only`; `GET` (termasuk ekspor) tetap boleh. Total **32 endpoint**.

### 4.1 Simulasi & Alokasi Budget

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/scenarios` | Skenario proyek (≤ 3, urut `total_target`) beserta item, dan `presets` = preset kode untuk jenis acara proyek (pernikahan: `wedding_10jt` … `wedding_100jt`, `kind: fixed`; jenis lain: `<jenis>_pct`, `kind: percent`, [TEMPLATE_EVENT bab 15.3](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#153-preset-skenario-non-pernikahan)). |
| `GET` | `/scenarios/:scenario_id` | Satu `Scenario` dengan item. |
| `POST` | `/scenarios` | `CreateScenarioRequest` → `201 ScenarioResult`. `409 max_scenarios_reached`. Dengan `preset_key` fixed: `total_target` boleh dikosongkan (diambil dari preset) atau harus sama; preset percent: `total_target` wajib. Item preset di sesi yang sudah tidak ada jatuh ke `all` (`scope_fallback`). |
| `PUT` | `/scenarios/:scenario_id` | `UpdateScenarioRequest`: ubah nama, target, sesi aktif (mengganti penuh). Mengubah `total_target` tidak menskala item. |
| `DELETE` | `/scenarios/:scenario_id` | `204`. Boleh pada skenario aktif (§3.1). |
| `POST` | `/scenarios/:scenario_id/clone` | `{"new_name": "..."}` → `201 ScenarioResult`; mengikuti batas 3. |
| `POST` | `/scenarios/:scenario_id/load-preset` | `LoadPresetRequest` → `ScenarioResult`: mengganti semua item skenario dengan item preset (tombol paket di Tab 1). `confirm: true` wajib bila skenario sudah punya item (`confirm: required`); `preset_key` yang bukan milik jenis acara → `400 invalid_template`. |
| `POST` | `/scenarios/:scenario_id/items` | `ScenarioItemRequest` → `201 ScenarioItem`. `409 scenario_item_limit_reached` (100 per skenario); `item_key` ganda di sesi yang sama → `409 conflict`. |
| `PUT` | `/scenarios/:scenario_id/items/:item_id` | `UpdateScenarioItemRequest` (nama, nominal, sesi); `item_key` tidak berubah (dikirim → `400 invalid_input: item_key`). Pindah sesi yang bentrok kunci → kunci dinetralkan (§3.10). |
| `DELETE` | `/scenarios/:scenario_id/items/:item_id` | `204`. |
| `POST` | `/scenarios/compare` | `CompareRequest {a, b}` → `ScenarioComparison`: dua skenario berdampingan, selisih `b − a` per `(sesi, item_key)` (pos tanpa kunci dipasangkan lewat `lower(item_name)`, aturan yang sama dengan §3.2); dua id sama → `400 invalid_input: b: must differ from a`. |
| `POST` | `/scenarios/apply-preview` | `ApplyScenarioRequest` → `ApplyReport`, tanpa menulis (§3.2). |
| `POST` | `/scenarios/apply` | `ApplyScenarioRequest` → `ApplyReport` (§3.2). |

### 4.2 Kas & Tabungan

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/contributors` | `Contributor[]` (urut `sort_order`) dengan total dan jumlah setoran. |
| `POST` | `/contributors` | `ContributorRequest` → `201 Contributor`. `role_label` divalidasi registry. `409 contributor_limit_reached` (20). |
| `PUT` | `/contributors/:contributor_id` | `ContributorRequest`; ubah nama/peran (mengganti penuh: `role_label` kosong = `other`). |
| `DELETE` | `/contributors/:contributor_id` | `204`; `409 contributor_in_use` bila punya setoran. Dijalankan: `DELETE` berhasil di pernyataan dan **gagal `23503` saat `COMMIT`** (FK `savings → contributors` ditunda, kontrak bab 4), jadi usecase memeriksa lebih dulu; sisa galat commit (setoran masuk di antara pemeriksaan dan hapus) dipetakan ke `409 conflict`. |
| `GET` | `/savings?contributor_id=` | `SavingsOverview`: KPI, kontributor, milestone, rekomendasi, **matriks bulanan**, setoran (§3.6, §3.8). |
| `POST` | `/savings` | `SavingsEntryRequest` → `201 SavingsEntry`. `409 savings_entry_limit_reached` (2.000). |
| `PUT` | `/savings/:entry_id` | `SavingsEntryRequest` → `SavingsEntry` (koreksi, boleh pindah kontributor). |
| `DELETE` | `/savings/:entry_id` | `204`. |

Tidak ada `PATCH /savings/target`: target tabungan diubah lewat `PATCH /v1/planner/projects/:id` Menu 01 (`savings_target` / `clear_savings_target`, [§3.6](#36-rekomendasi-tabungan-dan-milestone-bertahap)).

### 4.3 Realisasi

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/budget/cashflow-summary` | `CashflowSummary` (§3.5). |
| `GET` | `/budget/matrix` | `BudgetMatrix`: KPI + matriks sesi + skenario aktif (§3.8). |
| `GET` | `/budget/items?scope=` | `BudgetLineList`, baris `planner_budget_lines` urut `sort_order`. **Tanpa `scope` = semua sesi; `scope=all` = hanya pos Umum; `scope=<scope_key>` = satu sesi** (aturan yang sama dengan Menu 02; versi awal menulis "`scope` = scope_key atau `all`" sehingga `all` ambigu). |
| `POST` | `/budget/items` | `BudgetItemRequest` → `201 BudgetLine`. `409 budget_item_limit_reached` (300). |
| `PUT` | `/budget/items/:item_id` | `BudgetItemRequest` → `BudgetLine` (§3.10). `409 vendor_already_linked`. |
| `PATCH` | `/budget/items/:item_id/payment` | `PaymentRequest` → `BudgetLine` (§3.3). `400 invalid_stage`, `409 rollup_locked`. |
| `DELETE` | `/budget/items/:item_id/payment/:stage` | Mengosongkan satu tahap (`dp`, `term_1`, `term_2`) → `200 BudgetLine` (§3.3). |
| `DELETE` | `/budget/items/:item_id` | `204`; tautan dokumen/belanja menjadi kosong (§3.10). |
| `POST` | `/budget/items/reorder` | `{"ids": [...]}` lengkap → `BudgetLineList`. |
| `POST` | `/budget/apply-template` | `ApplyTemplateRequest` → `ApplyTemplateResult` (§3.9). |
| `GET` | `/budget/export?format=xlsx` | Berkas `.xlsx` (§3.11); 10 ekspor/menit/pengguna. |

### 4.4 Bentuk respons

| Endpoint | `data` |
|---|---|
| `GET /scenarios` | `ScenarioList` `{scenarios[], presets[]}` |
| `GET /scenarios/:id`, `PUT /scenarios/:id` | `Scenario` |
| `POST /scenarios`, `…/clone`, `…/load-preset` | `ScenarioResult` `{scenario, warnings[]}` (`201` untuk dua yang pertama) |
| `POST …/items` (`201`), `PUT …/items/:id` | `ScenarioItem` |
| `DELETE` (skenario, item skenario, kontributor, setoran, pos) | `204` |
| `POST /scenarios/compare` | `ScenarioComparison` |
| `POST /scenarios/apply-preview`, `/apply` | `ApplyReport` |
| `GET /contributors`, `POST` (`201`), `PUT` | `Contributor[]` / `Contributor` |
| `GET /savings` | `SavingsOverview` |
| `POST /savings` (`201`), `PUT /savings/:id` | `SavingsEntry` |
| `GET /budget/cashflow-summary` | `CashflowSummary` |
| `GET /budget/matrix` | `BudgetMatrix` |
| `GET /budget/items`, `POST …/reorder` | `BudgetLineList` `{items[]}` |
| `POST /budget/items` (`201`), `PUT`, `PATCH …/payment`, `DELETE …/payment/:stage` | `BudgetLine` |
| `POST /budget/apply-template` | `ApplyTemplateResult` `{created, skipped, deleted, warnings[]}` |
| `GET /budget/export` | berkas (bukan JSON): `Content-Disposition: attachment; filename="keuangan-<slug>-<tanggal>.xlsx"` ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)) |

Kunci opsional **dihilangkan** bila kosong (`omitempty`, tipe FE `?:`): `item_key`, `preset_key`, `preset_amount`, `vendor_*`, `admin_doc_id`, `rollup_source`, `quantity`, `unit_label`, `unit_price`, `*_paid_at`, `*_receipt_url`, `notes` (pos dan setoran), `per_contributor`, `deleted`, `new_target_budget`, `total_target` pada `PresetInfo` percent. `BudgetMatrix.active_scenario` adalah satu-satunya `null` yang dikirim. Semua daftar (`scenarios`, `items`, `presets`, `warnings`, `rows`, …) selalu array, bukan `null`. `remaining` pada pos dan `available_cash` boleh negatif. **Tidak ada persentase di respons mana pun**; klien menghitung dengan `percent`/`sharePercents` ([FE bab 5](FE.md#5-library-libplannerfinancets)), selaras kontrak bab 6 (pembagi nol).

### 4.5 Validasi dan batas

Semua galat berbentuk `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)); pesan memakai kosakata Menu 01 dan 02 ditambah alasan di bawah. Angka batas adalah usulan ([bab 9, butir 6](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)) dan dijaga usecase.

| Field | Aturan | Galat |
|---|---|---|
| `name`, `new_name` (skenario, kontributor) | dipangkas, 1–100 karakter (rune), tanpa karakter kontrol | `400 invalid_input` |
| `item_name` | dipangkas, 1–150 | `400 invalid_input` |
| `category` | 1–60 | `400 invalid_input` |
| `notes` | ≤ 2.000; `\n` boleh (`cleanNote` Menu 02) | `400 invalid_input` |
| `unit_label` | ≤ 30 | `400 invalid_input` |
| `receipt_url` | `http`/`https`, ≤ 500 karakter | `400 invalid_input: receipt_url: must be an http or https URL` |
| `sub_event_scope` | ada di master sesi (termasuk `all`); diperiksa sebelum menulis (FK-nya ditunda) | `400 invalid_scope` |
| `active_sub_events` | tiap elemen = `scope_key` sesi non-sistem milik proyek; duplikat dibuang | `400 invalid_scope` |
| `item_key` (POST item skenario) | `^[a-z0-9_]{1,40}$` | `400 invalid_input` |
| uang: `cost_amount`, `initial_estimate`, `unit_price`, `amount` | bilangan bulat 0…10¹³ | `400 invalid_input: <field>: must be between 0 and 10000000000000` |
| `total_target` | bilangan bulat 1…10¹³ | `must be between 1 and 10000000000000` |
| `total_target` + preset | fixed: dikosongkan atau sama dengan total preset; percent: ≥ 100.000 | `must match the preset total` / `must be at least 100000` |
| `preset_key` | termasuk `presets` jenis acara proyek | `400 invalid_template` |
| `pricing_mode` | `lump` atau `unit` | `must be one of lump, unit` |
| `quantity` | 0,01…1.000.000, dua desimal; wajib bersama `unit_price` bila `unit` | `must be between 0.01 and 1000000`; `required when pricing_mode is unit` |
| `quantity × unit_price` | hasil ≤ 10¹³ (aritmetika bulat, §3.4) | `unit_price: the estimate (quantity × unit_price) must be at most 10000000000000` |
| `payment_stage` | `dp`, `term_1`, `term_2`, `full` | `must be one of dp, term_1, term_2, full`; semua tahap terisi atau sisa ≤ 0 → `400 invalid_stage` |
| `amount` (pembayaran, setoran) | wajib dan > 0 (pembayaran `full`: diabaikan) | `required` / `must be greater than 0` |
| `paid_at`, `entry_date` | tanggal valid dalam jendela `Hari-H − 1095 hari` s/d akhir acara `+ 365 hari` (seperti Menu 02); tanggal mendatang diizinkan | `400 invalid_date` (`must be within the project schedule window`) |
| `role_label` | ∈ `contributor_roles` jenis acara; kosong = `other` | `must be one of the event type roles` |
| `contributor_id` | kontributor proyek ini | `must be a contributor of this project` |
| `vendor_id` | vendor proyek ini; belum tertaut ke pos lain | `must be a vendor of this project`; `409 vendor_already_linked` |
| `mode` | `smart_adjust`/`clean_overwrite` (apply); `append`/`replace_template` (apply-template) | `must be one of …` |
| `confirm` | bool (`load-preset`, `apply-template`) atau `"RESET"` (`clean_overwrite`); aturan di §3.2, §3.9, `load-preset` | `confirm: required` / `confirm: must be RESET` |
| `ids` (reorder) | tepat semua id proyek, tanpa ganda | `must list every id exactly once` |
| `a`, `b` (compare) | dua skenario berbeda milik proyek | `b: must differ from a`; `404 not_found` |
| Jumlah | ≤ 3 skenario; ≤ 100 pos per skenario; ≤ 300 pos anggaran; ≤ 20 kontributor; ≤ 2.000 setoran per proyek. Diperiksa di bawah kunci proyek, semua-atau-tidak (juga untuk `apply`, `load-preset`, `apply-template`) | `409 max_scenarios_reached`, `scenario_item_limit_reached`, `budget_item_limit_reached`, `contributor_limit_reached`, `savings_entry_limit_reached` |
| Frekuensi | ekspor 10/menit/pengguna ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)) | `429 rate_limited` |

Kode galat baru (ditambahkan ke tabel kontrak bab 6): `scenario_item_limit_reached`, `budget_item_limit_reached`, `contributor_limit_reached`, `savings_entry_limit_reached`. Kode lain sudah ada: `max_scenarios_reached`, `contributor_in_use`, `vendor_already_linked`, `rollup_locked`, `invalid_stage`, `invalid_scope`, `invalid_template`.

Alasan baku (Inggris, setelah `<field>: `) yang dipetakan FE ([FE bab 10](FE.md#10-pesan-galat-dan-teks)): kosakata Menu 01 (`required`, `must not be empty`, `must be at most N characters`, `must not contain control characters`, `must be a valid date in YYYY-MM-DD format`, `must be an integer`, `must be between A and B`, `must be at least N`) ditambah `must be an http or https URL`, `must be within the project schedule window`, `must be one of the event type roles`, `must be a contributor of this project`, `must be a vendor of this project`, `must match the preset total`, `must be RESET`, `must list every id exactly once`, `must be greater than 0`, `required when pricing_mode is unit`, `the estimate (quantity × unit_price) must be at most 10000000000000`, `must differ from a`, dan `must be one of …`.

Hanya field yang dikirim pada `PATCH` yang divalidasi (`PATCH …/payment`: semua field permintaan). `PUT` memvalidasi seluruh body.

Urutan pemeriksaan (ditetapkan saat implementasi): id anak yang bukan UUID kanonik atau milik proyek lain → `404` lebih dulu (pada `PUT /savings/:entry_id` sebelum `contributor_id` dalam body diperiksa); batas jumlah `409` diperiksa di dalam transaksi berkunci, sesudah validasi bentuk tetapi **sebelum** isian yang bergantung pada data proyek (`active_sub_events`) pada `POST /scenarios`; `apply`, `apply-preview`, dan `compare` menolak id kosong dengan `400 invalid_input: <field>: required`. `item_key` yang dikirim pada `PUT` item skenario → `400 invalid_input: item_key: cannot be changed` (struct tidak memuatnya, jadi handler memeriksa kunci mentah JSON). `GET …/budget/export` tanpa `format=xlsx` → `400 invalid_input: format: must be xlsx`.

---

## 5. DTO (Go)

Blok di bawah **sudah dikompilasi** (`go vet`, dengan stub `Notice` yang sama dengan `planner/domain`) dan dibandingkan kunci demi kunci dengan tipe TypeScript [FE bab 4.2](FE.md#42-tambahan-pada-srclibplannertypests): 29 pasangan tipe, 329 jalur JSON, nol selisih pada kunci, opsionalitas (`omitempty` ↔ `?:`), `null`, dan jenis nilai untuk tipe respons (tipe permintaan: kunci dan jenis saja). Pembanding diuji dengan memutasi tipe TS (enam mutasi, semuanya dilaporkan).

```go
package domain

import (
	"github.com/google/uuid"

	pdomain "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Konvensi (sama dengan Menu 01 dan 02): tag json snake_case; kunci opsional berupa pointer + omitempty (dihilangkan bila
// kosong, FE: `?:`); daftar selalu array (tidak pernah null); TANPA tag `binding` (validasi di usecase, pesan
// "<field>: <alasan>"); uang int64 rupiah utuh; tidak ada persentase yang dikirim (klien memakai percent/sharePercents).

// ----- Tab 1: skenario -----

type ScenarioItem struct {
	ID            uuid.UUID `json:"id"`
	SubEventScope string    `json:"sub_event_scope"`
	ItemKey       *string   `json:"item_key,omitempty"`
	Category      string    `json:"category"`
	ItemName      string    `json:"item_name"`
	CostAmount    int64     `json:"cost_amount"`
	Included      bool      `json:"included"`                // hasil hitung: sesi ikut disimulasikan (active_sub_events kosong, 'all', atau terdaftar)
	PresetAmount  *int64    `json:"preset_amount,omitempty"` // nilai di preset asal skenario (item_key cocok); dasar "kembalikan ke nilai preset"
}

type Scenario struct {
	ID              uuid.UUID      `json:"id"`
	Name            string         `json:"name"`
	TotalTarget     int64          `json:"total_target"`
	IsActive        bool           `json:"is_active"`
	ActiveSubEvents []string       `json:"active_sub_events"` // scope_key; kosong = semua sesi non-sistem
	PresetKey       *string        `json:"preset_key,omitempty"`
	Allocated       int64          `json:"allocated"` // Σ cost_amount item yang Included
	Items           []ScenarioItem `json:"items"`
}

// PresetInfo: preset ada di kode (bab 6). total_target hanya pada kind "fixed" (pernikahan); "percent" memakai total dari pengguna.
type PresetInfo struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"` // fixed | percent
	TotalTarget *int64 `json:"total_target,omitempty"`
}

type ScenarioList struct {
	Scenarios []Scenario   `json:"scenarios"`
	Presets   []PresetInfo `json:"presets"`
}

type ScenarioResult struct {
	Scenario Scenario         `json:"scenario"`
	Warnings []pdomain.Notice `json:"warnings"` // scope_fallback
}

type CreateScenarioRequest struct {
	Name            string   `json:"name"`
	TotalTarget     *int64   `json:"total_target"`
	PresetKey       *string  `json:"preset_key"`
	ActiveSubEvents []string `json:"active_sub_events"`
}

type UpdateScenarioRequest struct { // PUT: mengganti penuh
	Name            string   `json:"name"`
	TotalTarget     int64    `json:"total_target"`
	ActiveSubEvents []string `json:"active_sub_events"`
}

type CloneScenarioRequest struct {
	NewName string `json:"new_name"`
}

type LoadPresetRequest struct {
	PresetKey   string `json:"preset_key"`
	TotalTarget *int64 `json:"total_target"`
	Confirm     bool   `json:"confirm"` // wajib bila skenario sudah punya pos
}

type ScenarioItemRequest struct { // POST
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key"`
	Category      string  `json:"category"`
	ItemName      string  `json:"item_name"`
	CostAmount    int64   `json:"cost_amount"`
}

type UpdateScenarioItemRequest struct { // PUT: item_key tidak berubah (dikirim → 400 invalid_input: item_key)
	SubEventScope string `json:"sub_event_scope"`
	Category      string `json:"category"`
	ItemName      string `json:"item_name"`
	CostAmount    int64  `json:"cost_amount"`
}

type CompareRequest struct {
	A uuid.UUID `json:"a"`
	B uuid.UUID `json:"b"`
}

type CompareRow struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	AAmount       int64   `json:"a_amount"`
	BAmount       int64   `json:"b_amount"`
	Delta         int64   `json:"delta"` // b − a
}

type CompareSide struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TotalTarget int64     `json:"total_target"`
}

type CompareTotals struct {
	A     int64 `json:"a"`
	B     int64 `json:"b"`
	Delta int64 `json:"delta"`
}

type ScenarioComparison struct {
	A      CompareSide   `json:"a"`
	B      CompareSide   `json:"b"`
	Rows   []CompareRow  `json:"rows"`
	Totals CompareTotals `json:"totals"`
}

type ApplyScenarioRequest struct {
	ScenarioID        uuid.UUID `json:"scenario_id"`
	Mode              string    `json:"mode"` // smart_adjust | clean_overwrite
	SyncTargetBudget  bool      `json:"sync_target_budget"`
	SyncTargetSavings bool      `json:"sync_target_savings"` // savings_target := NULL (ikut pagu)
	Confirm           *string   `json:"confirm"`             // "RESET" wajib untuk clean_overwrite
}

type AppliedUpdate struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	From          int64   `json:"from"`
	To            int64   `json:"to"`
}

type AppliedCreate struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	Amount        int64   `json:"amount"`
}

type AppliedSkip struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	Reason        string  `json:"reason"` // rollup | unit_pricing
}

type AppliedDeleted struct {
	Lines     int   `json:"lines"`
	PaidTotal int64 `json:"paid_total"`
}

// ApplyReport: isi yang sama untuk apply-preview (tanpa menulis) dan apply (yang benar-benar dilakukan).
type ApplyReport struct {
	Mode            string          `json:"mode"`
	Updated         []AppliedUpdate `json:"updated"`
	Created         []AppliedCreate `json:"created"`
	Skipped         []AppliedSkip   `json:"skipped"`
	Deleted         *AppliedDeleted `json:"deleted,omitempty"` // hanya clean_overwrite
	UntouchedCount  int             `json:"untouched_count"`
	EstimateAfter   int64           `json:"estimate_after"`
	NewTargetBudget *int64          `json:"new_target_budget,omitempty"` // ada bila sync_target_budget
}

// ----- Tab 2: kas dan tabungan -----

type Contributor struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	RoleLabel  string    `json:"role_label"`
	Total      int64     `json:"total"`
	EntryCount int       `json:"entry_count"`
}

type ContributorRequest struct { // POST dan PUT
	Name      string  `json:"name"`
	RoleLabel *string `json:"role_label"` // bawaan "other"
}

type SavingsEntry struct {
	ID              uuid.UUID `json:"id"`
	ContributorID   uuid.UUID `json:"contributor_id"`
	ContributorName string    `json:"contributor_name"`
	RoleLabel       string    `json:"role_label"`
	EntryDate       string    `json:"entry_date"` // YYYY-MM-DD
	Amount          int64     `json:"amount"`
	Notes           *string   `json:"notes,omitempty"`
}

type SavingsEntryRequest struct { // POST dan PUT
	ContributorID uuid.UUID `json:"contributor_id"`
	EntryDate     string    `json:"entry_date"`
	Amount        int64     `json:"amount"`
	Notes         *string   `json:"notes"`
}

type Milestone struct {
	SubEventScope string `json:"sub_event_scope"`
	Name          string `json:"name"`
	SessionDate   string `json:"session_date"`
	TargetAmount  int64  `json:"target_amount"`
	SecuredAmount int64  `json:"secured_amount"`
	IsSecured     bool   `json:"is_secured"`
}

type SavingsMonth struct {
	Month         string           `json:"month"`          // "2026-02"
	ByContributor map[string]int64 `json:"by_contributor"` // kunci = id kontributor; semua kontributor selalu ada (0 bila kosong)
	Total         int64            `json:"total"`
}

type SavingsKPI struct {
	Target              int64 `json:"target"` // COALESCE(savings_target, target_budget)
	Collected           int64 `json:"collected"`
	Shortfall           int64 `json:"shortfall"`
	TargetFollowsBudget bool  `json:"target_follows_budget"` // savings_target IS NULL
}

type Recommendation struct {
	MonthsLeft         int    `json:"months_left"`
	RecommendedMonthly int64  `json:"recommended_monthly"`
	PerContributor     *int64 `json:"per_contributor,omitempty"` // dihilangkan bila belum ada kontributor
	TargetDate         string `json:"target_date"`
	Basis              string `json:"basis"` // session | event_date
}

type SavingsMatrix struct {
	Months []string       `json:"months"`
	Rows   []SavingsMonth `json:"rows"`
}

type SavingsOverview struct {
	KPI                 SavingsKPI     `json:"kpi"`
	Contributors        []Contributor  `json:"contributors"`
	Milestones          []Milestone    `json:"milestones"`
	UnscheduledEstimate int64          `json:"unscheduled_estimate"`
	Recommendation      Recommendation `json:"recommendation"`
	Matrix              SavingsMatrix  `json:"matrix"`
	Entries             []SavingsEntry `json:"entries"` // disaring ?contributor_id=; bagian lain selalu seluruh proyek
}

// ----- Tab 3: realisasi -----

// BudgetLine = satu baris planner_budget_lines (angka efektif). Kolom paid_total dan remaining_balance tabel TIDAK dikirim.
type BudgetLine struct {
	ID              uuid.UUID  `json:"id"`
	SubEventScope   string     `json:"sub_event_scope"`
	Category        string     `json:"category"`
	ItemName        string     `json:"item_name"`
	ItemKey         *string    `json:"item_key,omitempty"`
	Source          string     `json:"source"` // template | scenario | vendor | admin_doc | manual
	VendorID        *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName      *string    `json:"vendor_name,omitempty"`
	AdminDocID      *uuid.UUID `json:"admin_doc_id,omitempty"`
	RollupSource    *string    `json:"rollup_source,omitempty"` // "shopping": estimasi/terbayar dihitung dari belanja
	PricingMode     string     `json:"pricing_mode"`            // lump | unit
	Quantity        *float64   `json:"quantity,omitempty"`
	UnitLabel       *string    `json:"unit_label,omitempty"`
	UnitPrice       *int64     `json:"unit_price,omitempty"`
	Estimate        int64      `json:"estimate"` // efektif (dari belanja bila rollup)
	DPAmount        int64      `json:"dp_amount"`
	Term1Amount     int64      `json:"term_1_amount"`
	Term2Amount     int64      `json:"term_2_amount"`
	DPPaidAt        *string    `json:"dp_paid_at,omitempty"` // YYYY-MM-DD
	Term1PaidAt     *string    `json:"term_1_paid_at,omitempty"`
	Term2PaidAt     *string    `json:"term_2_paid_at,omitempty"`
	DPReceiptURL    *string    `json:"dp_receipt_url,omitempty"`
	Term1ReceiptURL *string    `json:"term_1_receipt_url,omitempty"`
	Term2ReceiptURL *string    `json:"term_2_receipt_url,omitempty"`
	Paid            int64      `json:"paid"`
	Remaining       int64      `json:"remaining"`      // < 0 = lebih bayar
	PaymentStatus   string     `json:"payment_status"` // unpaid | dp_paid | partial | settled
	SortOrder       int        `json:"sort_order"`
	Notes           *string    `json:"notes,omitempty"`
}

type BudgetLineList struct {
	Items []BudgetLine `json:"items"`
}

// BudgetItemRequest: POST dan PUT. PUT mengganti penuh: vendor_id yang tidak dikirim MELEPAS vendor.
type BudgetItemRequest struct {
	SubEventScope   string     `json:"sub_event_scope"`
	Category        string     `json:"category"`
	ItemName        string     `json:"item_name"`
	VendorID        *uuid.UUID `json:"vendor_id"`
	PricingMode     *string    `json:"pricing_mode"`     // bawaan lump
	InitialEstimate *int64     `json:"initial_estimate"` // diabaikan bila pricing_mode = unit atau pos rollup
	Quantity        *float64   `json:"quantity"`
	UnitLabel       *string    `json:"unit_label"`
	UnitPrice       *int64     `json:"unit_price"`
	Notes           *string    `json:"notes"`
}

type PaymentRequest struct {
	PaymentStage string  `json:"payment_stage"` // dp | term_1 | term_2 | full
	Amount       *int64  `json:"amount"`        // wajib dan > 0 kecuali full (diabaikan)
	PaidAt       string  `json:"paid_at"`       // YYYY-MM-DD
	ReceiptURL   *string `json:"receipt_url"`   // http/https; tidak dikirim = bukti tahap itu dikosongkan
}

type ApplyTemplateRequest struct {
	Mode          string  `json:"mode"` // append | replace_template
	SubEventScope *string `json:"sub_event_scope"`
	Confirm       bool    `json:"confirm"`
}

type ApplyTemplateResult struct {
	Created  int              `json:"created"`
	Skipped  int              `json:"skipped"`
	Deleted  int              `json:"deleted"`
	Warnings []pdomain.Notice `json:"warnings"`
}

type CashflowSummary struct {
	TotalSavingsCollected int64 `json:"total_savings_collected"`
	TotalPaid             int64 `json:"total_paid"`
	AvailableCash         int64 `json:"available_cash"` // collected − paid; boleh negatif
	RemainingDebt         int64 `json:"remaining_debt"` // Σ max(estimate − paid, 0)
	Overpaid              int64 `json:"overpaid"`       // Σ max(paid − estimate, 0)
	CashDeficit           int64 `json:"cash_deficit"`   // max(remaining_debt − available_cash, 0)
}

type MatrixRow struct {
	SubEventScope string `json:"sub_event_scope"` // "" pada baris total
	Estimate      int64  `json:"estimate"`
	DP            int64  `json:"dp"`
	Term1         int64  `json:"term_1"`
	Term2         int64  `json:"term_2"`
	RollupPaid    int64  `json:"rollup_paid"`
	Paid          int64  `json:"paid"` // dp + term_1 + term_2 + rollup_paid
	Remaining     int64  `json:"remaining"`
	Overpaid      int64  `json:"overpaid"`
	LineCount     int    `json:"line_count"`
}

type LineStatus struct { // sama dengan FinanceSummary.line_status (Menu 01)
	Settled int `json:"settled"`
	Partial int `json:"partial"`
	Unpaid  int `json:"unpaid"`
}

type MatrixKPI struct {
	TotalEstimate  int64      `json:"total_estimate"`
	TotalPaid      int64      `json:"total_paid"`
	TotalRemaining int64      `json:"total_remaining"`
	TotalOverpaid  int64      `json:"total_overpaid"`
	LineCount      int        `json:"line_count"`
	LineStatus     LineStatus `json:"line_status"` // hanya pos berestimasi > 0
}

type ActiveScenario struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TotalTarget int64     `json:"total_target"`
	Unallocated int64     `json:"unallocated"` // total_target − Σ estimasi semua pos; boleh negatif
}

type BudgetMatrix struct {
	KPI            MatrixKPI       `json:"kpi"`
	Rows           []MatrixRow     `json:"rows"` // satu per sesi yang punya pos, urut sesi, Umum terakhir
	Total          MatrixRow       `json:"total"`
	ActiveScenario *ActiveScenario `json:"active_scenario"` // null bila tidak ada skenario aktif
}
```

---

## 6. Master Data

### 6.1 Enam preset skenario (PDF Hal. 4)

Berada di kode (`plannerfinance/domain/template_wedding.go`), bukan basis data. `preset_key` = `wedding_10jt` … `wedding_100jt`. Setiap paket 10 pos; total tiap kolom sama dengan nama paket (diperiksa). Scope mengikuti letak pos pada PDF Hal. 5–6 (Mahar dan Seserahan berada pada daftar Resepsi PDF; Lamaran tidak ada dalam skenario Hal. 4 sehingga tidak punya pos preset). Seluruh 60 nilai dibaca ulang dari PDF dengan `pdfminer` dan sama.

| No | `item_key` | Pos (PDF) | Kategori | Sesi | 10 Jt | 20 Jt | 30 Jt | 50 Jt | 75 Jt | 100 Jt |
|:-:|---|---|---|:-:|--:|--:|--:|--:|--:|--:|
| 1 | `biaya_kua` | Biaya KUA | Legalitas | `akad` | 600.000 | 600.000 | 600.000 | 600.000 | 600.000 | 600.000 |
| 2 | `mahar` | Mahar & cincin pernikahan | Mahar | `resepsi` | 1.400.000 | 2.500.000 | 4.500.000 | 5.000.000 | 5.000.000 | 5.000.000 |
| 3 | `catering` | Catering | Catering | `resepsi` | 4.500.000 | 5.000.000 | 10.000.000 | 20.000.000 | 30.000.000 | 50.000.000 |
| 4 | `rias_busana` | Rias & baju pengantin | Busana & MUA | `resepsi` | 250.000 | 1.500.000 | 2.400.000 | 3.000.000 | 5.000.000 | 5.000.000 |
| 5 | `venue_dekor` | Sewa tempat/gedung & dekorasi | Venue & Dekorasi | `resepsi` | 2.000.000 | 4.500.000 | 4.500.000 | 10.000.000 | 20.000.000 | 24.000.000 |
| 6 | `undangan_souvenir` | Undangan & souvenir | Perlengkapan | `resepsi` | 150.000 | 800.000 | 1.500.000 | 2.500.000 | 3.500.000 | 3.500.000 |
| 7 | `dokumentasi` | Dokumentasi | Dokumentasi | `resepsi` | 0 | 2.000.000 | 2.000.000 | 2.500.000 | 3.000.000 | 3.000.000 |
| 8 | `seserahan` | Seserahan | Hantaran | `resepsi` | 800.000 | 1.600.000 | 2.000.000 | 3.000.000 | 3.500.000 | 3.500.000 |
| 9 | `hiburan` | Hiburan & pengisi acara | Hiburan | `resepsi` | 0 | 1.000.000 | 2.000.000 | 2.400.000 | 3.400.000 | 4.400.000 |
| 10 | `dana_darurat` | Dana darurat | Operasional | `all` | 300.000 | 500.000 | 500.000 | 1.000.000 | 1.000.000 | 1.000.000 |
| | | **Total** | | | **10.000.000** | **20.000.000** | **30.000.000** | **50.000.000** | **75.000.000** | **100.000.000** |

Catatan paket 10 Jt: PDF memberi label khusus ("Sewa Kebaya Simple" Rp 250.000 untuk baris Rias & busana, "Dekorasi Rumah" Rp 2.000.000, "Rias (Sendiri)" Rp 0, "Pengisi acara (keluarga sendiri)" Rp 0); baris "Rias (Sendiri)" digabung ke `rias_busana`, dan `dokumentasi` bernilai 0 karena tidak ada di paket itu. Jenis non-pernikahan memakai preset persentase `<jenis>_pct` ([TEMPLATE_EVENT bab 15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#15-pos-anggaran-template-non-pernikahan)).

### 6.2 Template 14 pos Realisasi (PDF Hal. 5–6)

`apply-template` / wizard membuat 14 pos ini (`source='template'`, estimasi **0**, pembayaran **0**). Nama pos persis PDF; jumlah 5 + 1 + 8.

| No | Sesi | `item_key` | Nama pos (PDF) | Kategori |
|:-:|:-:|---|---|---|
| 1 | `lamaran` | `catering` | Catering | Catering |
| 2 | `lamaran` | `gaun_kebaya` | Gaun/Kebaya | Busana & MUA |
| 3 | `lamaran` | `fotografer` | Fotografer | Dokumentasi |
| 4 | `lamaran` | `cincin` | Cincin | Perhiasan |
| 5 | `lamaran` | `dekorasi` | Dekorasi | Dekorasi |
| 6 | `akad` | `biaya_kua` | Biaya KUA | Legalitas |
| 7 | `resepsi` | `mahar` | Mahar | Mahar |
| 8 | `resepsi` | `catering` | Catering | Catering |
| 9 | `resepsi` | `rias_busana` | Rias & Baju Pengantin | Busana & MUA |
| 10 | `resepsi` | `venue_dekor` | Sewa Gedung & Dekorasi | Venue & Dekorasi |
| 11 | `resepsi` | `undangan_souvenir` | Undangan & souvenir | Perlengkapan |
| 12 | `resepsi` | `dokumentasi` | Dokumentasi | Dokumentasi |
| 13 | `resepsi` | `seserahan` | Seserahan | Hantaran |
| 14 | `resepsi` | `hiburan` | Hiburan & Pengisi acara | Hiburan |

Pos `all/dana_darurat` tidak ada di PDF Hal. 5–6; ia hanya muncul lewat preset skenario. Pos `seserahan` adalah target gulungan daftar belanja (04 BE §2.4). Pos `biaya_kua` (`akad`) adalah sasaran sinkron biaya dokumen KUA (02 BE §4.5).

### 6.3 Dataset Acuan PDF (hanya Proyek Contoh dan uji emas)

Nilai persis PDF Hal. 5–6 (dibaca ulang dengan `pdfminer`; semua 14 baris dan keempat subtotal sama). Ini **bukan** template; hanya dimuat oleh `POST /v1/planner/projects/sample` dan test. PDF tidak memuat tanggal pembayaran, jadi `*_paid_at` kosong.

| No | Sesi | `item_key` | Estimasi | DP/Lunas | Termin 1 | Termin 2 | Total Dibayar | Sisa | Status |
|:-:|:-:|---|--:|--:|--:|--:|--:|--:|:-:|
| 1 | lamaran | catering | 2.000.000 | 500.000 | 1.000.000 | 500.000 | 2.000.000 | 0 | settled |
| 2 | lamaran | gaun_kebaya | 750.000 | 750.000 | 0 | 0 | 750.000 | 0 | settled |
| 3 | lamaran | fotografer | 1.000.000 | 500.000 | 0 | 0 | 500.000 | 500.000 | dp_paid |
| 4 | lamaran | cincin | 2.500.000 | 600.000 | 0 | 0 | 600.000 | 1.900.000 | dp_paid |
| 5 | lamaran | dekorasi | 900.000 | 900.000 | 0 | 0 | 900.000 | 0 | settled |
| | | **Subtotal Lamaran** | **7.150.000** | **3.250.000** | **1.000.000** | **500.000** | **4.750.000** | **2.400.000** | |
| 6 | akad | biaya_kua | 600.000 | 600.000 | 0 | 0 | 600.000 | 0 | settled |
| | | **Subtotal Akad** | **600.000** | **600.000** | **0** | **0** | **600.000** | **0** | |
| 7 | resepsi | mahar | 3.000.000 | 3.000.000 | 0 | 0 | 3.000.000 | 0 | settled |
| 8 | resepsi | catering | 25.000.000 | 5.000.000 | 10.000.000 | 0 | 15.000.000 | 10.000.000 | partial |
| 9 | resepsi | rias_busana | 5.000.000 | 600.000 | 0 | 0 | 600.000 | 4.400.000 | dp_paid |
| 10 | resepsi | venue_dekor | 20.000.000 | 900.000 | 0 | 0 | 900.000 | 19.100.000 | dp_paid |
| 11 | resepsi | undangan_souvenir | 3.500.000 | 1.500.000 | 0 | 0 | 1.500.000 | 2.000.000 | dp_paid |
| 12 | resepsi | dokumentasi | 3.000.000 | 300.000 | 0 | 0 | 300.000 | 2.700.000 | dp_paid |
| 13 | resepsi | seserahan | 3.500.000 | 3.500.000 | 0 | 0 | 3.500.000 | 0 | settled |
| 14 | resepsi | hiburan | 3.400.000 | 1.400.000 | 1.000.000 | 1.000.000 | 3.400.000 | 0 | settled |
| | | **Subtotal Resepsi** | **66.400.000** | **16.200.000** | **11.000.000** | **1.000.000** | **28.200.000** | **38.200.000** | |
| | | **TOTAL** | **74.150.000** | **20.050.000** | **12.000.000** | **1.500.000** | **33.550.000** | **40.600.000** | |

Rasio pelunasan **45,2%** (33.550.000 ÷ 74.150.000 = 45,246%); status pos: 7 lunas, 6 hanya-DP, 1 sebagian (Catering Resepsi), 0 belum. Angka-angka ini direproduksi persis oleh view `planner_budget_lines` pada pengujian DDL dan pada audit ini (bab 7.1).

### 6.4 Tabungan: Kontributor dan Setoran (PDF Hal. 7)

PDF mencatat **setoran per bulan per pihak** (Feb–Des 2026). Kontributor contoh: **Arif (Pria)** `groom`, **Sinta (Wanita)** `bride` (hanya Proyek Contoh; proyek nyata mulai tanpa kontributor dan tanpa setoran). Setoran contoh memakai tanggal 1 tiap bulan (PDF hanya memberi bulan).

| Bulan 2026 | Calon Mempelai Pria | Calon Mempelai Wanita | Jumlah |
|---|--:|--:|--:|
| Februari | 2.500.000 | 500.000 | 3.000.000 |
| Maret | 5.000.000 | 2.000.000 | 7.000.000 |
| April | 1.500.000 | 1.000.000 | 2.500.000 |
| Mei | 3.000.000 | 500.000 | 3.500.000 |
| Juni | 1.500.000 | 1.000.000 | 2.500.000 |
| Juli | 2.500.000 | 1.500.000 | 4.000.000 |
| Agustus | 2.500.000 | 1.500.000 | 4.000.000 |
| September | 1.500.000 | 2.500.000 | 4.000.000 |
| Oktober | 2.000.000 | 1.000.000 | 3.000.000 |
| November | 2.000.000 | 2.000.000 | 4.000.000 |
| Desember | 6.000.000 | 4.000.000 | 10.000.000 |
| **Total** | **30.000.000** | **17.500.000** | **47.500.000** |

Target **75.000.000** (= pagu skenario 75 Jt), terkumpul **47.500.000** (63,3%), kekurangan 27.500.000. Persentase porsi: Pria 63,2% dan Wanita 36,8% **dari total terkumpul** (porsi dihitung dari total terkumpul, bukan dari target; 63,3% adalah capaian terhadap target, bukan porsi Arif).

### 6.5 Kontributor dan pos jenis lain
Peran kontributor per jenis acara: registry ([TEMPLATE_EVENT bab 4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori)). Template pos non-pernikahan: persentase ([TEMPLATE_EVENT bab 15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#15-pos-anggaran-template-non-pernikahan)); delapan jenis berjumlah 100% (diperiksa) dengan 8 / 7 / 8 / 8 / 7 / 7 / 8 / 6 pos (khitanan, aqiqah, peresmian, launching, ulang tahun, wisuda, seminar, lainnya). Template ini usulan Adicara, bukan dari PDF.

### 6.6 Skenario Proyek Contoh

Tiga skenario dari preset bab 6.1, dimuat overlay `finance` bersama dataset 6.3–6.4. Semuanya `active_sub_events = {akad, resepsi}`, 10 pos berkunci dari preset.

| Nama | `preset_key` | `total_target` | `is_active` |
|---|---|--:|:-:|
| Opsi Hemat 50 Jt | `wedding_50jt` | 50.000.000 | – |
| **Opsi Ideal 75 Jt** | `wedding_75jt` | 75.000.000 | ✔ |
| Opsi 100 Jt | `wedding_100jt` | 100.000.000 | – |

Versi awal spesifikasi menampilkan tiga skenario ini di wireframe FE tetapi tidak menetapkannya di dataset mana pun.

---

## 7. Pengujian

### 7.1 Yang sudah dijalankan pada audit spesifikasi

Dijalankan 2026-10-21 pada PostgreSQL 16.2 lokal, database sementara `planner_fin_scratch_test` (dibuat khusus dengan migrasi 001–012 dari berkas nyata, dihapus setelahnya; database `adicara` tidak disentuh), koneksi `psql`. Skrip ada di scratchpad sesi, bukan di repo. Ini adalah SQL dan algoritma yang dijalankan **sebelum** kode Go modul ditulis; hasil pada kodenya ada di bab 7.2.

| Butir | Hasil |
|---|---|
| Blok SQL bab 1 identik dengan migrasi 008 dan 010; keduanya naik di atas 001–012 | ✔ perbandingan teks; 14 tabel + 1 view `planner_*` |
| PDF Hal. 4, 5, 6, 7 dibaca ulang (`pdfminer`): 60 nilai preset, 14 pos, 4 subtotal, 22 setoran, target, terkumpul | ✔ sama dengan bab 6; baris "Subtotal Kategori" PDF Hal. 5 tak bermakna (bab 3.8) |
| View pada data emas: 74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000; status 7 / 6 / 1 / 0; per sesi sama dengan PDF | ✔ |
| Smart Adjust (3.2): `create 1, update 9`, tak tersentuh 6, DP utuh, ulang → `same 10` | ✔ |
| Aktivasi skenario dengan satu `UPDATE` | ✔ **gagal `23505`** pada pindah 75 → 100 Jt; dua pernyataan lolos |
| Batas 3 skenario, dua sesi berbalapan | ✔ tanpa kunci 4 skenario; dengan kunci B melihat 3 |
| Likuiditas 13.950.000 / 26.650.000; contoh lebih-bayar (selisih mentah 35.600.000 vs `remaining_debt` 40.600.000) | ✔ |
| Milestone FIFO (7.150.000 / 600.000 / 39.750.000 = 59,9%); rekomendasi 4.584.000 / 2.292.000 pada 4 dan 21 Okt 2026, 27.500.000 pada H-1 dan lampau | ✔ |
| Matriks bulanan tanpa celah (Mei–Juni dihapus → baris 0) | ✔ |
| Pembayaran: `full` ke tahap kosong pertama; hapus tahap; kunci `FOR UPDATE` pada view ditolak, pada tabel berhasil | ✔ |
| `append` dengan `ON CONFLICT … WHERE item_key IS NOT NULL DO NOTHING`: `created 2, skipped 1` | ✔ |
| Syarat `replace_template`: `deletable 13, skipped_with_payment 1, at_risk 9`; hapus pos rollup membiarkan belanja (`budget_item_id` kosong) | ✔ |
| `clean_overwrite`: 16 pos / Rp 500.000 terbayar hilang, 10 pos berestimasi 75.000.000 tersisa | ✔ |
| Pindah pos ke sesi yang sudah punya kuncinya: `23505`; netralisasi kunci berhasil | ✔ |
| Hapus skenario aktif: 0 aktif, item ikut terhapus | ✔ |
| Hapus kontributor bersetoran: lolos di pernyataan, gagal `23503` saat `COMMIT` | ✔ |
| `round(9999999999.99 × 10¹³)::bigint` | ✔ `bigint out of range` (bab 3.4) |
| Kolom baru tabel tidak muncul di view; `paid_total`/`remaining_balance` salah untuk pos rollup | ✔ (bab 1.2) |
| Pembulatan preset persentase: 8 jenis berjumlah 100; selisih negatif hanya untuk total ≤ Rp 19.000 | ✔ (bab 3.9) |
| Kueri provider `finance` 01 BE §5.2 pada data emas | ✔ 7 / 7 / 0; kas bebas 13.950.000 |
| Ketujuh blok SQL bab 3 (Smart Adjust, pembayaran, likuiditas, milestone, matriks sesi, matriks bulanan, syarat `replace_template`) dijalankan **apa adanya** (hanya `$n` diganti nilai) di dalam transaksi pada proyek uji | ✔ tanpa galat |
| Blok DTO bab 5: `gofmt`, `go vet`; kunci sama dengan tipe TS FE (29 pasangan, 329 jalur, nol selisih) | ✔ |
| Seluruh tautan relatif di folder PLANNER_SPEC (termasuk ke bab-bab dokumen ini) | ✔ nol rusak |

**Tidak dijalankan pada audit (sebelum kodenya ditulis; sebagian besar kini dicakup bab 7.2):** kode Go modul ini, jalur HTTP/Gin, `go test`, ekspor `.xlsx` (belum ada prototipe), konkurensi selain batas skenario (mis. dua pembayaran serentak pada pos yang sama), latensi kueri pada data 10× contoh, PostgreSQL 18, Docker, Nginx.

### 7.2 Pengujian pada implementasi

Berkas: `internal/modules/plannerfinance/{domain,usecase,delivery/http}/*_test.go` (83 tes) dan satu tes registry di `planner/domain`. Tes DB butuh `TEST_DATABASE_URL` ke database `…_test` (isinya dihapus), dijalankan `go test -p 1`; tanpa itu mereka di-skip. Dijalankan 2026-10-21 pada PostgreSQL 16.2 (database sementara yang dihapus setelahnya): **261 tes tingkat atas lulus** di seluruh backend, 0 gagal, 0 di-skip; `-race` pada `plannerfinance`, `planner`, `plannertimeline`, dan `app` lulus; cakupan modul 91,3% pernyataan; `go vet`, `gofmt -l`, dan `GOOS=linux go build ./cmd/api` bersih. Nama tes di bawah adalah nama yang sebenarnya (berbeda dari usulan semula).

| Butir | Tes |
|---|---|
| DDL bab 1 identik dengan migrasi 008/010; naik, turun, naik | ○ naik-turun-naik oleh tes migrasi Menu 01 (lulus); kesamaan teks blok bab 1 dengan berkas migrasi diperiksa sekali pada audit (bab 7.1), **tidak ada tesnya** |
| Data template sama dengan tabel spesifikasi: 6 paket × 10 pos, 14 pos, dataset PDF, 22 setoran, 3 skenario, 59 pos persentase | ✔ `TestTemplatesMatchSpec` (membaca BE bab 6 dan TEMPLATE_EVENT 15.1; di-skip bila `docs/` tidak ikut), `TestWeddingPresetsSumToPackage`, `TestPercentTemplatesAddUpTo100`, `TestSampleDataset` |
| Preset: paket berjumlah namanya; persentase berjumlah 100 dan tak pernah negatif untuk total ≥ 100.000 (semua kelipatan 1 rupiah sampai 400.000 × 8 jenis); total < 100.000 → 0 / ditolak | ✔ `TestPercentAmountsAddUpAndNeverGoNegative`, `TestPresets`, `TestPresetPlan`, `TestTemplateLines` |
| **Uji emas Proyek Contoh** (kontrak bab 9), jam tetap 4 Okt 2026 | ✔ `TestSampleProjectFinanceGolden` (matriks, per sesi, status 7/7/0, kas bebas, defisit, milestone, rekomendasi, 22 setoran, 3 skenario, `unallocated` 850.000, widget dashboard); dibuktikan gagal bila satu angka PDF diubah |
| Rekomendasi di berbagai jam (4 Okt, 21 Okt, H-150, H-151, H-30, H-31, H-1, Hari-H, H+5), tanpa kontributor, tanpa sesi bertanggal, tiga kontributor | ✔ `TestRecommendationAcrossTime`, `TestSavingsWithoutContributorsHasNoPerContributor` |
| Wizard: `budget_lines` membuat 14 pos / pos persentase; `templates_available` memuatnya; sesi hilang → `scope_fallback` | ✔ `TestBudgetTemplateScopeFallbackAndWizard`, `TestApplyTemplateForOtherEventTypes`, `TestEventTypesOfferBudgetLinesAndContributorRoles`, `TestFinanceWiring`; ○ atomisitas saat applier `budget` gagal tidak diuji sendiri (mekanisme Menu 01, diuji dengan applier jadwal) |
| Smart Adjust: 9 diperbarui, 1 dibuat, 5 tak tersentuh; pembayaran utuh; ulang → nol perubahan; rollup/unit dilewati; pos kustom bernama sama; `apply-preview` = `apply`; `sync_target_*` | ✔ `TestApplySmartAdjust`, `TestApplySkipsRollupAndUnitPricingAndPairsKeylessLines`, `TestApplySyncsTheTargets`, `TestSmartAdjustPlanDecidesPerPairedLine`, `TestSmartAdjustPlanIsIdempotent`, `TestSmartAdjustPlanDoesNotPairTwoItemsWithOneLine` |
| Aktivasi: berpindah antar skenario sepuluh kali tanpa `23505`; paling banyak satu aktif | ✔ `TestActivateScenarioSwitchesWithoutUniqueViolation` |
| `clean_overwrite`: tanpa/salah `confirm` ditolak, laporan `deleted`, batas 300 tidak menulis apa pun | ✔ `TestApplyCleanOverwriteNeedsReset`, `TestCleanOverwritePlan`, `TestApplyStaysUnderTheLineLimit` |
| Skenario ke-4 ditolak, klon, hapus yang aktif, dua permintaan bersamaan | ✔ `TestScenarioLimitCloneAndDeleteActive`, `TestTwoSimultaneousCreatesCannotPassTheScenarioLimit`, `TestCreateScenarioFromPresets`, `TestPercentPresetForOtherEventTypes`, `TestScenarioPresetFallsBackToTheGeneralSession`, `TestUpdateScenarioReplacesTheHeader` |
| `load-preset`, item skenario, perbandingan | ✔ `TestLoadPreset`, `TestScenarioItems`, `TestScenarioItemLimit`, `TestCompareScenarios`, `TestCompare`, `TestEnrichComputesIncludedAllocatedAndPresetAmount` |
| Pembayaran: tahap mengganti, `full` ke tahap kosong pertama, tahap penuh atau sisa ≤ 0 → `invalid_stage`, rollup → `rollup_locked`, lebih bayar, hapus tahap, `receipt_url`, klik ganda | ✔ `TestRecordPayment`, `TestPaymentsOfAllThreeStagesAndTheFullStageOrder`, `TestRollupLinesCannotBePaid`, `TestSimultaneousFullPaymentsPayOnce`, `TestFirstEmptyStage`, `TestCleanReceiptURL` |
| Pos satuan: `round(quantity × unit_price)`, estimasi klien diabaikan, hasil > 10¹³ ditolak tanpa 500 | ✔ `TestUnitPricing`, `TestPricing`, `TestUnitEstimate`, `TestQuantityX100` |
| Ubah pos: `vendor_already_linked`, vendor proyek lain, pindah sesi menetralkan kunci, `PUT` tanpa `vendor_id` melepas vendor, hapus melepas belanja, urut ulang | ✔ `TestUpdateBudgetItem`, `TestMovingALineNeutralisesTheKeyWhenTheSessionHasIt`, `TestDeleteAndReorderBudgetItems`, `TestCreateAndReadBudgetItems` |
| `apply-template`: `append` idempoten, `replace_template` melewati pos terbayar, `confirm` hanya bila ada pekerjaan, `budget_link_lost`; sinkron biaya Menu 02 menautkan pos `akad/biaya_kua` milik modul ini | ✔ `TestApplyTemplateAppend`, `TestApplyTemplateReplace`, `TestAdminDocSyncLinksTheTemplateLineOfThisModule` |
| Tabungan: milestone FIFO, matriks tanpa celah, `contributor_id` hanya menyaring `entries`, `contributor_in_use`, peran dari registry, batas | ✔ `TestContributors`, `TestContributorLimit`, `TestSavingsEntries`, `TestSavingsEntryLimit`, `TestSavingsMatrixHasNoGapsAndMilestonesFollowSessionDates`, `TestSavingsMatrixFoldsCellsIntoMonths`; peran: `TestContributorRoles` (`planner/domain`) |
| Matriks sesi (identitas `estimate − paid = remaining − overpaid`, `paid = dp + term_1 + term_2 + rollup_paid`, Umum terakhir, `active_scenario`) dan likuiditas (lebih bayar tak mengurangi kewajiban pos lain) | ✔ `TestBuildMatrix`, `TestCashflow`, `TestCashflowSummaryOverpaid`, `TestFinanceSummaryMatchesTheDashboardContract`, golden, `TestRollupLinesCannotBePaid` |
| Batas 300 pos / 100 item skenario / 20 kontributor / 2.000 setoran, termasuk di bawah balapan | ✔ `TestBudgetLineLimit`, `TestTwoSimultaneousCreatesCannotPassTheLineLimit`, `TestScenarioItemLimit`, `TestContributorLimit`, `TestSavingsEntryLimit` |
| Isolasi: pengguna B `404` di 32 rute proyek A dan pada id proyek A di proyek B; tanpa token `401`; arsip menolak tulis, `GET` dan ekspor tetap boleh; body rusak | ✔ `TestUserBGetsNotFoundOnEveryFinanceEndpoint`, `TestEveryFinanceRouteNeedsABearerToken`, `TestArchivedProjectIsReadOnlyButStillReadable`, `TestFinanceDocumentsAreRejectedForMalformedBodies` |
| Ekspor: tiga lembar, angka sebagai bilangan, sel berawalan `=`/`+`/`-`/`@` menjadi teks, 10/menit | ✔ `TestWorkbookOfTheSampleProject`, `TestWorkbookNeutralisesFormulasInUserText`, `TestExportParametersAndRateLimit`, `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow` |
| Kontrak: 32 rute terdaftar = 32 operasi `openapi.yaml`; skema = struct Go | ✔ `TestOpenAPIDocumentsEveryFinanceRoute`, `TestOpenAPISchemasMatchTheGoDTOs` (31 skema; dibuktikan gagal bila satu kunci diganti nama); `redocly lint` valid (3 peringatan lama); DTO ↔ tipe TS FE terhadap struct sebenarnya: 29 pasangan, 329 jalur, 0 selisih (skrip, bukan tes repo) |
| Universal: sembilan jenis acara menjalani alur yang sama (template wizard, preset + Apply, peran dari registry, setoran, pembayaran, cashflow, dashboard, ekspor, sinkron biaya dokumen) | ✔ `TestEveryEventTypeRunsTheWholeFinanceFlow` (sembilan subtes; gagal di semuanya bila pemasangan item skenario ke pos dirusak) |
| Mutasi | 23 mutasi kode produksi semuanya digagalkan tes (rincian di [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-03.md#pengujian)) |

**Tidak dijalankan:** frontend; Menu 04 (rollup, vendor, `lock-deal` diuji dengan baris yang disisipkan tes ke `planner_shopping_items` dan `planner_vendors`); `.xlsx` di Excel, Numbers, atau LibreOffice; pembayaran serentak dengan `apply` pada pos yang sama dan `PUT` pos serentak; latensi pada data 10× contoh; PostgreSQL 18, Docker, Nginx, beberapa replika.

---

## 8. Urutan pengerjaan dan definisi selesai

Posisi di urutan pembangunan ([kontrak bab 9](../00_KONTRAK_BERSAMA/README.md#9-urutan-pembangunan-dan-uji-penerimaan)): setelah `planner` (Menu 01) dan `plannertimeline` (Menu 02); tabel Menu 04 (vendor 006, belanja 009) cukup tabelnya, yang sudah ada. **Semua langkah selesai pada 2026-10-21.**

1. ✔ **Registry**: `ContributorRoles []Option` di `planner/domain/eventtype.go` dengan nilai [TEMPLATE_EVENT bab 4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori); `GET /event-types` mengirim `contributor_roles`.
2. ✔ **`plannerfinance`**: `domain` (entitas, DTO bab 5, template pernikahan + persentase dibangkitkan dari tabel spesifikasi), `repository` (SQL §3), `usecase` (skenario, kontributor/tabungan, pos/pembayaran, matriks/likuiditas, template; `LockProject`, `cleanNote`, validasi bab 4.5), `delivery/http` (32 rute, `ratelimit.go` salinan pola yang ada, `export.go`).
3. ✔ **Sambungan** di `internal/app/app.go` ([3.12](#312-sambungan-ke-menu-01)): provider `finance`, applier `budget`, overlay `finance` (sesudah overlay Menu 02, sebelum overlay vendor yang belum ada), rute.
4. ✔ **Dependensi**: tidak ada yang baru (`excelize` dari Menu 02). `github.com/goccy/go-yaml` (sudah tidak langsung) kini dipakai langsung oleh tes kontrak dan pindah ke blok `require` langsung di `go.mod`; `go.sum` tidak berubah.
5. ✔ **Dokumen**: `backend/docs/openapi.yaml` (tag `Planner Finance`); [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-03.md) dengan daftar penyimpangan; `docs/backend/README.md` dan `CHANGELOG.md`; `docs/api.md`, `docs/architecture.md`, `docs/database.md`, `backend/README.md`, `README.md`; entri [CHANGELOG spesifikasi](../CHANGELOG.md); kontrak bab 6 (kode galat baru) dan bab 1.1 (pengecualian SQL ketiga); TEMPLATE_EVENT 15.1 (kolom Kategori).
6. ✔ **Pengujian**: lihat bab 7.2.

**Selesai bila:** semua butir 7.2 lulus (`go test -p 1`, `-race`, `go vet`, build `linux`) ✔ kecuali dua butir bertanda ○ (kesamaan teks DDL dengan migrasi tidak punya tes; atomisitas kegagalan applier `budget` tidak diuji sendiri); uji emas Proyek Contoh sesuai kontrak bab 9 ✔; widget Keuangan di dashboard terisi ✔; `templates_available.budget_lines` benar ✔; openapi lolos lint tanpa peringatan baru ✔; dokumen implementasi menyebut apa yang tidak diuji ✔.

---

## 9. Celah dan keputusan yang menunggu pemilik produk

Butir 1–13 ditemukan saat mencocokkan spesifikasi dengan kode Menu 01 dan 02 yang sudah jadi, kontrak, PDF, dan SQL yang dijalankan (audit 2026-10-21); butir 14–17 adalah keputusan yang baru dapat dibuktikan setelah ada kode. Dokumen mengikuti pilihan bawaan yang tertulis; ubah bila produk menginginkan lain.

1. **Aktivasi skenario dua pernyataan** (bukan satu `UPDATE`): terbukti gagal `23505` (§3.2). Bukan keputusan produk.
2. **Pembayaran bisa dibatalkan** lewat `DELETE …/payment/:stage` (endpoint baru) dan `full` mengabaikan `amount`. Versi awal tidak punya cara membatalkan, dan `amount` pada `full` ambigu. Alternatif: `amount: 0` pada `PATCH` berarti hapus.
3. **Target tabungan lewat PATCH proyek Menu 01** (tanpa `PATCH /savings/target`). `FE` memakai `savings_target` / `clear_savings_target`, dan 01 FE `UpdateProjectBody` perlu dua kolom itu (backend sudah menerimanya; [FE bab 1.3](FE.md#13-perubahan-pada-file-yang-sudah-ada)).
4. **Pengecualian SQL ketiga:** modul ini menulis `planner_projects.target_budget` dan `savings_target` (opsi Apply). Alternatif: antarmuka Go di `planner/domain` yang diimplementasikan `planner`. Dicatat di kontrak bab 1.1 sebagai pengecualian; ubah bila Menu 01 lebih suka memiliki penulisan itu.
5. **Template pos non-pernikahan membawa estimasi persentase**, bukan 0 (hanya pernikahan 0). README versi awal menyebut "template membuat pos berestimasi 0" untuk semua jenis; kontrak bab 5 dan TEMPLATE_EVENT bab 15 menyatakan sebaliknya. Preset persentase menolak total < Rp 100.000 (selisih pembulatan negatif di bawah ±Rp 19.000).
6. **Angka batas adalah usulan:** 3 skenario (keputusan produk), 100 pos per skenario, 300 pos anggaran, 20 kontributor, 2.000 setoran, `quantity` 0,01…1.000.000, teks 100/150/60/30/500/2.000 karakter, jendela tanggal seperti Menu 02. Empat kode galat baru ditambahkan ke kontrak.
7. **Tanggal setoran mendatang diizinkan dan dihitung "terkumpul".** Proyek Contoh yang dibuat Okt 2026 memuat setoran Nov dan Des 2026 (PDF mencatatnya begitu). Bila produk ingin "terkumpul = sampai hari ini", milestone dan defisit berubah; perlu penanda "rencana" pada setoran.
8. **Proyeksi RSVP** ditawarkan pada pos berharga satuan mana pun, bukan hanya pos katering: registry tidak punya daftar kunci pos katering per jenis (kunci beragam: `catering`, `konsumsi`, `katering_box`, `katering_coffee_break`, `katering_kue`). Alternatif: menurunkan dari `budget_item_keys` kategori vendor `catering` (04 BE §2.1).
9. **Kategori pos tidak punya daftar di registry.** Dropdown kategori diisi kategori yang sudah dipakai proyek (pos dan item skenario) ditambah isian bebas; proyek kosong hanya punya isian bebas.
10. **Menghapus skenario aktif** membiarkan proyek tanpa skenario aktif (tidak ada pengaktifan otomatis); pagu vendor jatuh ke manual. Alternatif: mengaktifkan skenario lain yang tersisa.
11. **`PUT` pos mengganti penuh dan `vendor_id` yang tidak dikirim melepas vendor** (kontrak bab 6). Pos yang tertaut lewat `lock-deal` bisa terlepas bila formulir lupa mengirimnya. Alternatif: `PATCH` dengan `clear_vendor` seperti Menu 02.
12. **Pindah sesi menetralkan `item_key`** (menjadi pos biasa) tanpa warning. Bila produk ingin tahu, tambahkan notice `item_key_cleared`.
13. **`remaining_balance` pada `planner_budget_items` mubazir dan menyesatkan** (§1.2): tidak dibaca siapa pun, dan salah untuk pos rollup. Migrasi 008 boleh diedit di tempat (tahap pengembangan) untuk menghapusnya; kode tidak membacanya.
14. **Ekspor `.xlsx`** diimplementasikan menurut tata letak §3.11 (lembar `Rekap`, `Anggaran`, `Tabungan`; baris "Dibayar dari belanja" dan "Lebih bayar" pada Rekap hanya muncul bila tidak nol) dan dibaca kembali dengan `excelize` oleh tes, tetapi **belum dibuka di Excel, Numbers, atau LibreOffice**. Awalan `'` pada sel tampak sebagai karakter (02 BE bab 8 butir 12).
15. **Pembatas laju** kini salinan keempat (auth, planner, timeline, finance); ekstraksi ke `internal/ratelimit` tetap refaktor terpisah (02 BE bab 8 butir 13). Salinan finance memulai jam sapuan dari jamnya sendiri (`l.now`); salinan planner dan timeline memulainya dari `time.Now()`, sehingga tes mereka `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow` gagal setelah sekitar pukul 10.05 UTC 2026-10-21 (jam tetap tes sudah lewat dari jam dinding). Tes keduanya kini mengatur `lastSweep`; kodenya tidak diubah.
16. **Pos `rollup`** mengabaikan perubahan estimasi pada `PUT` secara diam-diam (angka efektif dari belanja); bila produk ingin penolakan eksplisit, gunakan `409 rollup_locked`.
17. **Kunci bersama milik Menu 04.** Pos bersama (mis. `venue_dekor`) tidak ditautkan ke satu vendor (04 BE §2.2); tampilan pos tidak menandai "mewakili beberapa vendor" kecuali lewat warning `shared_budget_item` pada lock-deal.

Butir 18–28 ditemukan **saat membangun kodenya** (2026-10-21); semuanya sudah mengikuti pilihan bawaan yang tertulis dan diuji. Butir 29 ditemukan pada audit kejelasan, universalitas, dan integrasi lintas menu sesudahnya (hari yang sama) dan **belum diputuskan**.

18. **Template persentase tidak punya kategori.** `planner_budget_items.category` `NOT NULL`, tetapi TEMPLATE_EVENT 15.1 hanya memuat kunci, nama, dan persen. Kolom **Kategori** ditambahkan (59 baris; memakai kategori pernikahan bila sepadan: Catering, Dokumentasi, Hiburan, Operasional, Perlengkapan, Busana & MUA, Legalitas, Venue & Dekorasi; baru: Medis, Dekorasi, Prosesi, Audio Visual, Publikasi, Honor). Nama kategori adalah usulan, belum ditinjau pemilik produk.
19. **Template persentase untuk `target_budget` kecil.** Di bawah Rp 100.000 semua estimasi 0 (§3.9); tanpa itu pagu Rp 50.000 membuat `dana_darurat` negatif dan pembuatan proyek menjadi 500.
20. **`load-preset` menulis `total_target` dan `preset_key` skenario** (paket tetap: total paketnya; persentase: total yang dikirim atau yang sudah ada). Spesifikasi hanya menyebut "mengganti semua item".
21. **Dua item tanpa kunci bernama sama** dalam satu skenario: yang kedua menjadi pos sendiri saat Smart Adjust (§3.2).
22. **Isian kosong**: `mode` kosong pada `apply` = `smart_adjust`; pada `apply-template` = `append`; `scenario_id`, `a`, `b` kosong → `400 required`.
23. **Urutan pemeriksaan**: 404 untuk id proyek lain mendahului validasi body; batas jumlah mendahului validasi `active_sub_events` (§4.5).
24. **Overlay mengisi estimasi** selain pembayaran: dataset PDF memuat estimasi 74.150.000 yang tidak dimiliki template (§3.12).
25. **Milestone bertarget 0 berstatus `is_secured: true`** (0 dari 0). Proyek baru yang sesinya bertanggal Hari-H menampilkan dua milestone tanpa dana (`GET /savings`). Pilihan bawaan: server mengirimnya apa adanya; **FE perlu memutuskan** menyembunyikan milestone bertarget 0 atau menampilkannya sebagai "belum ada anggaran".
26. **Pesan `409` kunci kembar** pada item skenario: `conflict` dengan pesan `item_key: already used in this session` (spesifikasi hanya `409 conflict`); vendor yang sudah tertaut → `409 vendor_already_linked`.
27. **`GET /contributors` mengirim array** (`data: Contributor[]`), bukan objek; `GET /budget/export` tanpa `format=xlsx` → `400 invalid_input: format`.
28. **Tes kontrak memakai `goccy/go-yaml` langsung** (pindah ke `require` langsung di `go.mod`); alternatifnya skrip di luar repo, yang tidak menjaga kontrak terus-menerus.
29. **Kunci gulungan belanja jenis non-pernikahan tidak ada di template.** [04 BE §2.4](../04_VENDOR_DAN_BELANJA/BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup) menyebut pos gulungan `perlengkapan` untuk jenis lain (pernikahan: `seserahan`, yang ada di 14 pos PDF), tetapi tidak satu pun dari 59 pos persentase ([TEMPLATE_EVENT 15.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#151-pos-dan-persentase)) berkunci `perlengkapan` (pos sejenis bernama `undangan_souvenir`, `souvenir`, `merchandise_kit`, `toga_atribut`, `kit_sertifikat`, `cetak`). Akibatnya `link-to-budget` pada jenis non-pernikahan selalu **membuat pos baru** di luar persentase pagu, sehingga total estimasi melebihi `target_budget` sebesar nilai belanja. Tidak merusak apa pun (04 belum dibangun, dan spesifikasi 04 memang menyebut "bila tidak ada, dibuat"), tetapi perlu keputusan sebelum Menu 04: (a) biarkan, (b) tiap jenis menunjuk satu pos 15.1 sebagai gulungan (mis. `undangan_souvenir`, `souvenir`, `merchandise_kit`) dan 04 membaca kunci itu dari registry, atau (c) tambah pos `perlengkapan` ke tabel persentase (mengubah angka tiap jenis).
