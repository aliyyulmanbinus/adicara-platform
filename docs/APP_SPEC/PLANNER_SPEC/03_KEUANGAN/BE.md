# Menu 03: Keuangan & Anggaran — Backend Specification

Spesifikasi backend modul Go `plannerfinance` (`backend/internal/modules/plannerfinance/`) untuk tiga sub-modul: **Simulasi** (PDF Hal. 4), **Tabungan** (Hal. 7), dan **Realisasi** (Hal. 5–6), mengikuti alur $\text{Simulasi} \rightarrow \text{Tabungan} \rightarrow \text{Realisasi}$. Aturan lintas modul (uang `BIGINT`, master sesi, template vs data contoh, kode error) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

> **Acuan angka.** Seluruh angka bab 6 berasal dari PDF Hal. 4–7 dan diuji terhadap basis data ([kontrak bab 10](../00_KONTRAK_BERSAMA/README.md#10-validasi-spesifikasi)): total dibayar **Rp 33.550.000**, sisa **Rp 40.600.000**, rasio **45,2%**; Lamaran belum lunas (dibayar 4.750.000, sisa 2.400.000); 14 pos memakai nama dan nilai PDF.

---

## 1. Skema Database

### 1.1 Tabel

Migrasi `008_000_planner_finance.sql` (sudah dijalankan di PostgreSQL 16). Ciri skema: uang `BIGINT`; preset sistem **tidak disimpan di tabel** (ada di kode, bab 6) sehingga `planner_budget_scenarios.project_id` selalu terisi; persentase dihitung saat dibaca; status pelunasan dihitung view dan tidak disimpan sehingga tidak bisa menyimpang; `active_sub_events` adalah `TEXT[]` berisi `scope_key`; FK komposit mencegah kontributor, vendor, atau dokumen proyek lain ditunjuk; `item_key`, `source`, `admin_doc_id`, `rollup_source`, dan penetapan harga per satuan mendukung integrasi.

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

Migrasi `010_000_planner_budget_lines_view.sql` (dijalankan setelah `009_000_planner_shopping.sql`, karena view membaca `planner_shopping_items`):

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

---

## 2. Kunci Pencocok (`item_key`) dan Integrasi Pos

Pos dari skenario, template, vendor, dan dokumen harus saling mengenali agar tidak menggandakan pos (tanpa kunci, *Smart Adjust* tidak punya dasar pencocokan dan `lock-deal` vendor membuat pos bernama vendor yang bersaing dengan pos skenario):

- `item_key` (slug) sama pada `planner_budget_scenario_items` dan `planner_budget_items`; unik per `(sub_event_scope, item_key)` di tiap tabel (indeks parsial).
- Pos yang dibuat dari **template** (bab 6.2), **preset skenario** (6.1), dan **vendor** (04 BE §2.2) memakai kunci yang sama, jadi satu pos "Catering" resepsi tidak pernah muncul dua kali.
- Pos kustom pengguna boleh `item_key NULL`; ia tidak pernah ditimpa skenario.

| Sumber (`source`) | Dibuat oleh | Boleh ditimpa skenario? |
|---|---|---|
| `template` | `apply-template` / wizard | estimasi saja (pembayaran tidak pernah disentuh) |
| `scenario` | Apply skenario (pos baru) | estimasi saja |
| `vendor` | `lock-deal` vendor (pos baru bila tak ada yang cocok) | estimasi saja |
| `admin_doc` | sinkron biaya dokumen (02 BE §4.5) | estimasi saja |
| `manual` | pengguna | tidak (tanpa `item_key`) |

---

## 3. Logika Bisnis

### 3.1 Batas 3 skenario (tanpa balapan)
`CreateScenario`/`Clone` berjalan dalam transaksi yang mengunci baris proyek (`SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE`), lalu menghitung skenario; bila ≥ 3 → `409 max_scenarios_reached`. Tanpa kunci, dua permintaan bersamaan sama-sama melihat 2 dan membuat skenario ke-4. Indeks `planner_budget_scenarios_one_active_idx` menjamin paling banyak satu skenario aktif.

### 3.2 Menerapkan skenario (*Smart Adjust*)

`POST .../scenarios/apply-preview` (tanpa menulis) dan `POST .../scenarios/apply` memakai algoritma yang sama dalam satu transaksi dengan kunci proyek:

```text
scenario  := skenario terpilih ; aktifkan, nonaktifkan yang lain
items     := item skenario dengan sub_event_scope ∈ active_sub_events ∪ {'all'}  (active_sub_events kosong = semua sesi)
untuk tiap item:
  bila item_key ≠ NULL:  pos := budget_items WHERE project, sub_event_scope, item_key
  bila item_key = NULL:  pos := budget_items WHERE project, sub_event_scope, lower(item_name) dan item_key IS NULL
  bila pos ada:
      rollup_source ≠ NULL  → LEWATI (laporkan "dihitung dari belanja")
      pricing_mode = 'unit'  → LEWATI (laporkan; harga per satuan diatur pengguna)
      selain itu             → UPDATE initial_estimate = cost_amount   # DP/termin/vendor/dokumen TIDAK disentuh
  bila tidak ada            → INSERT pos (source='scenario', item_key, estimasi)
pos proyek yang tidak ada di skenario → tidak disentuh, tidak dihapus
bila sync_target_budget   → planner_projects.target_budget = scenario.total_target
bila sync_target_savings  → planner_projects.savings_target = NULL   # ikut target_budget
```

`apply-preview` mengembalikan `{updated: [{scope, item_key, from, to}], created: [...], skipped: [{reason}], untouched_count, new_target_budget}` agar antarmuka menampilkan daftar perubahan sebelum konfirmasi. Mode `clean_overwrite` menghapus **semua** pos proyek (pembayaran ikut hilang) lalu menyalin skenario; wajib `confirm: "RESET"`.

Contoh nyata (diuji pada data bab 6.3): proyek dengan 14 pos template (estimasi 0) menerima preset `wedding_75jt` → 9 pos diperbarui (`akad/biaya_kua` 600.000; `resepsi/mahar`, `catering`, `rias_busana`, `venue_dekor`, `undangan_souvenir`, `dokumentasi`, `seserahan`, `hiburan`), 1 pos baru (`all/dana_darurat` 1.000.000), 5 pos Lamaran tak tersentuh; total estimasi 75.000.000.

### 3.3 Mencatat pembayaran
`PATCH .../budget/items/:item_id/payment` menulis **satu tahap** pada kolom pos (PDF punya tiga tahap tetap: DP/Lunas, Termin 1, Termin 2). Tahap `dp | term_1 | term_2` mengganti nilai tahap itu (koreksi dimungkinkan); tahap `full` mengisi sisa (`remaining`) ke tahap kosong pertama (urutan DP, T1, T2), dan bila ketiganya terisi → `400 invalid_stage`. Pos dengan `rollup_source` menolak pembayaran: `409 rollup_locked` (angkanya dihitung dari daftar belanja). Kelebihan bayar diizinkan (status tetap `settled`, `remaining` negatif).

### 3.4 Pos berharga satuan
`pricing_mode = 'unit'`: usecase menghitung `initial_estimate = round(quantity × unit_price)` (estimasi dari klien diabaikan). Dipakai pos seperti Katering (porsi × harga/porsi); angka porsi dapat diisi dari proyeksi RSVP (05 BE §2.3) atas tindakan pengguna, tidak otomatis.

### 3.5 Likuiditas
```sql
-- dari view; sama dengan yang dipakai dashboard
SELECT COALESCE(sum(paid), 0) AS paid,
       COALESCE(sum(GREATEST(remaining, 0)), 0)  AS remaining,   -- kewajiban nyata: bagian positif per pos
       COALESCE(sum(GREATEST(-remaining, 0)), 0) AS overpaid     -- kelebihan bayar pada pos tertentu
FROM planner_budget_lines WHERE project_id = $1;
SELECT COALESCE(sum(amount), 0) AS collected FROM planner_savings_entries WHERE project_id = $1;
```
`free_cash = collected − paid`; `cash_deficit = max(remaining − free_cash, 0)`; `has_deficit = cash_deficit > 0`. `remaining` memakai bagian positif per pos karena kelebihan bayar pada satu pos tidak mengurangi kewajiban pos lain (selisih mentah `Σ estimate − Σ paid` menyembunyikan kewajiban: pos A lebih bayar Rp 5 jt dan pos B kurang Rp 10 jt = kewajiban nyata Rp 10 jt, bukan Rp 5 jt); `overpaid` ditampilkan terpisah. Angka yang sama dipakai widget dashboard ([01 BE §5](../01_DASHBOARD/BE.md#5-dashboard-get-v1plannerprojectsiddashboard)). Dengan data PDF `overpaid = 0`: 47.500.000 − 33.550.000 = **13.950.000** kas bebas; sisa kewajiban 40.600.000 → defisit **26.650.000**.

### 3.6 Rekomendasi tabungan dan milestone bertahap
Target tabungan = `COALESCE(savings_target, target_budget)` (satu sumber di `planner_projects`; `PATCH /savings/target` menulis `savings_target`, `null` mengembalikan ke ikut pagu).

- **Milestone** = sesi non-sistem yang punya `session_date`, diurut `session_date, sort_order`. `target_amount` = jumlah `estimate` pos pada sesi itu (dari view). Dana terkumpul dialokasikan berurutan (FIFO) ke milestone: `secured_amount = min(sisa dana, target_amount)`; `is_secured = secured_amount = target_amount`. Sesi tanpa tanggal tidak membentuk milestone (dilaporkan sebagai `unscheduled_estimate`).
- **Rekomendasi bulanan**: `shortfall = max(target − collected, 0)`; `months_left = max(1, ceil(hari_ke_sesi_terakhir_bertanggal ÷ 30))` (dengan fallback `event_date`); `recommended_monthly = ceil(shortfall ÷ months_left ÷ 1000) × 1000`; per orang dibagi jumlah kontributor aktif. Dengan data PDF: 27.500.000 ÷ 6 → **Rp 4.584.000/bulan** (Rp 2.292.000 per orang). Tanggal tiap sub-acara berasal dari `planner_event_sessions.session_date`.

### 3.7 Integrasi dengan modul lain
| Arah | Mekanisme |
|---|---|
| 04 Vendor → pos | `lock-deal`: tautkan ke pos khusus yang cocok `(scope, item_key)` atau buat pos baru (pos bersama tidak ditimpa); unik `planner_budget_items_vendor_idx` (satu vendor, satu pos) |
| 04 Belanja → pos | `planner_shopping_items.budget_item_id` + `rollup_source='shopping'`; angka dari view |
| 02 Dokumen → pos | `admin_doc_id` (02 BE §4.5) |
| 05 Tamu → pos katering | `pricing_mode='unit'` + tombol "Gunakan proyeksi RSVP" |
| Skenario ← 04 | `budget_limit` vendor dicari dari skenario aktif lewat `category_key → budget_item_keys` (04 BE §2.1) |

Penerimaan dana dari tamu (angpau/amplop digital lewat gateway) **tidak** dicatat di Tab Tabungan: Adicara tidak menampung dana, dan layanan Gift belum punya spesifikasi ([kontrak bab 8 #13](../00_KONTRAK_BERSAMA/README.md#8-matriks-integrasi-apa-yang-benar-benar-tersambung)).

---

## 4. Endpoint

`/v1/planner/projects/:id/...`, dengan `BearerAuth` + `ProjectGuard` + `WriteGuard`.

### 4.1 Simulasi & Alokasi Budget

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/scenarios` | Skenario proyek (≤ 3) beserta item; `presets` = preset kode untuk jenis acara proyek (pernikahan: `wedding_10jt` … `wedding_100jt`; jenis lain: `<jenis>_pct`, [TEMPLATE_EVENT bab 15.3](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#153-preset-skenario-non-pernikahan)). |
| `GET` | `/scenarios/:scenario_id` | Detail + item per sesi + `percentage_share` hasil hitung. |
| `POST` | `/scenarios` | `CreateScenarioRequest` (nama, `total_target`, `preset_key` opsional, `active_sub_events`). `409 max_scenarios_reached`. |
| `PUT` | `/scenarios/:scenario_id` | Ubah nama, target, sesi aktif. |
| `DELETE` | `/scenarios/:scenario_id` | Hapus. |
| `POST` | `/scenarios/:scenario_id/clone` | `{"new_name": "..."}`; mengikuti batas 3. |
| `POST` | `/scenarios/:scenario_id/items` | Tambah pos (`sub_event_scope`, `item_key?`, `category`, `item_name`, `cost_amount`). |
| `PUT` | `/scenarios/:scenario_id/items/:item_id` | Ubah nama, nominal, sesi. |
| `DELETE` | `/scenarios/:scenario_id/items/:item_id` | Hapus pos. |
| `POST` | `/scenarios/compare` | Dua skenario berdampingan, selisih per `(scope, item_key)`. |
| `POST` | `/scenarios/apply-preview` | Pratinjau §3.2 tanpa menulis. |
| `POST` | `/scenarios/apply` | Terapkan (§3.2). |

### 4.2 Kas & Tabungan

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/contributors` | Kontributor + total + persentase. |
| `POST` | `/contributors` | `name`, `role_label` (divalidasi registry). |
| `PUT` | `/contributors/:contributor_id` | Ubah nama/peran. |
| `DELETE` | `/contributors/:contributor_id` | `409 contributor_in_use` bila punya setoran (usecase memeriksa lebih dulu; FK `savings → contributors` ditunda sampai commit sehingga hapus proyek tidak bergantung pada urutan penghapusan anak). |
| `GET` | `/savings?contributor_id=` | Ringkasan: KPI, kartu kontributor, milestone, rekomendasi, **matriks bulanan**, riwayat. |
| `POST` | `/savings` | Catat setoran. |
| `PUT` | `/savings/:entry_id` | Koreksi setoran. |
| `DELETE` | `/savings/:entry_id` | Hapus setoran. |
| `PATCH` | `/savings/target` | `{"target_savings": 75000000}` atau `{"target_savings": null}` (ikut pagu). |

### 4.3 Realisasi

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/budget/cashflow-summary` | §3.5. |
| `GET` | `/budget/matrix` | KPI + matriks per sesi (kolom Tagihan Awal, DP, Termin 1, Termin 2, Dibayar, Sisa). |
| `GET` | `/budget/items?scope=` | Baris `planner_budget_lines`. `scope` = `scope_key` atau `all`. |
| `POST` | `/budget/items` | Tambah pos. |
| `PUT` | `/budget/items/:item_id` | Ubah pos. Menautkan vendor yang sudah bertaut ke pos lain → `409 vendor_already_linked`. |
| `PATCH` | `/budget/items/:item_id/payment` | §3.3. |
| `DELETE` | `/budget/items/:item_id` | Hapus pos (tautan dokumen/belanja menjadi kosong). |
| `POST` | `/budget/items/reorder` | `ids` lengkap. |
| `POST` | `/budget/apply-template` | `{"mode": "append" \| "replace_template", "confirm": bool}`. `append` menambah pos template yang `item_key`-nya belum ada. `replace_template` menghapus pos bersumber `template` **yang belum punya pembayaran**, lalu mengisi ulang; pos berpembayaran dilewati dan dilaporkan. |
| `GET` | `/budget/export?format=xlsx` | Rekap menyerupai PDF Hal. 5–6; sel di-escape; 10 ekspor/menit/pengguna. |

---

## 5. DTO (Go)

```go
package domain

type Scenario struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	TotalTarget     int64     `json:"total_target"`
	IsActive        bool      `json:"is_active"`
	ActiveSubEvents []string  `json:"active_sub_events"` // scope_key; kosong = semua sesi non-sistem
	PresetKey       *string   `json:"preset_key,omitempty"`
	Items           []ScenarioItem `json:"items,omitempty"`
}

type ScenarioItem struct {
	ID              uuid.UUID `json:"id"`
	SubEventScope   string    `json:"sub_event_scope"`
	ItemKey         *string   `json:"item_key,omitempty"`
	Category        string    `json:"category"`
	ItemName        string    `json:"item_name"`
	CostAmount      int64     `json:"cost_amount"`
	PercentageShare float64   `json:"percentage_share"` // dihitung: cost / total_target × 100, hanya untuk tampilan
}

type ApplyScenarioRequest struct {
	ScenarioID        uuid.UUID `json:"scenario_id" binding:"required"`
	Mode              string    `json:"mode" binding:"required,oneof=smart_adjust clean_overwrite"`
	SyncTargetBudget  bool      `json:"sync_target_budget"`  // default true di klien
	SyncTargetSavings bool      `json:"sync_target_savings"` // savings_target := NULL (ikut pagu)
	Confirm           string    `json:"confirm"`             // "RESET" wajib untuk clean_overwrite
}

// BudgetLine = satu baris planner_budget_lines (angka efektif).
type BudgetLine struct {
	ID              uuid.UUID  `json:"id"`
	SubEventScope   string     `json:"sub_event_scope"`
	Category        string     `json:"category"`
	ItemName        string     `json:"item_name"`
	ItemKey         *string    `json:"item_key,omitempty"`
	Source          string     `json:"source"`
	VendorID        *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName      *string    `json:"vendor_name,omitempty"`
	AdminDocID      *uuid.UUID `json:"admin_doc_id,omitempty"`
	RollupSource    *string    `json:"rollup_source,omitempty"` // "shopping": estimasi/terbayar dihitung dari belanja
	PricingMode     string     `json:"pricing_mode"`            // "lump" | "unit"
	Quantity        *float64   `json:"quantity,omitempty"`
	UnitLabel       *string    `json:"unit_label,omitempty"`
	UnitPrice       *int64     `json:"unit_price,omitempty"`
	Estimate        int64      `json:"estimate"`
	DPAmount, Term1Amount, Term2Amount int64
	DPPaidAt, Term1PaidAt, Term2PaidAt *string                 // YYYY-MM-DD
	DPReceiptURL, Term1ReceiptURL, Term2ReceiptURL *string
	Paid            int64      `json:"paid"`
	Remaining       int64      `json:"remaining"`
	PaymentStatus   string     `json:"payment_status"` // unpaid | dp_paid | partial | settled
	SortOrder       int        `json:"sort_order"`
	Notes           *string    `json:"notes,omitempty"`
}

type CreateBudgetItemRequest struct {
	SubEventScope  string     `json:"sub_event_scope" binding:"required"`
	Category       string     `json:"category" binding:"required"`
	ItemName       string     `json:"item_name" binding:"required,max=150"`
	VendorID       *uuid.UUID `json:"vendor_id"`
	PricingMode    string     `json:"pricing_mode" binding:"omitempty,oneof=lump unit"`
	InitialEstimate int64     `json:"initial_estimate" binding:"gte=0"` // diabaikan bila pricing_mode = unit
	Quantity       *float64   `json:"quantity"`
	UnitLabel      *string    `json:"unit_label"`
	UnitPrice      *int64     `json:"unit_price"`
	Notes          *string    `json:"notes"`
}
// UpdateBudgetItemRequest: field yang sama.

type RecordPaymentRequest struct {
	PaymentStage string  `json:"payment_stage" binding:"required,oneof=dp term_1 term_2 full"`
	Amount       int64   `json:"amount" binding:"required,gt=0"`
	PaidAt       string  `json:"paid_at" binding:"required"` // YYYY-MM-DD
	ReceiptURL   *string `json:"receipt_url"`                // http/https
}

type CashflowSummary struct {
	TotalSavingsCollected int64 `json:"total_savings_collected"`
	TotalPaid             int64 `json:"total_paid"`
	AvailableCash         int64 `json:"available_cash"`  // collected - paid; boleh negatif
	RemainingDebt         int64 `json:"remaining_debt"` // Σ max(estimate − paid, 0)
	Overpaid              int64 `json:"overpaid"`       // Σ max(paid − estimate, 0)
	CashDeficit           int64 `json:"cash_deficit"`
	HasDeficit            bool  `json:"has_deficit"`
}

type BudgetMatrix struct {
	KPI struct {
		TotalEstimate, TotalPaid, TotalRemaining int64
		SettlementRatio                          float64 // paid / estimate × 100, satu desimal
		LineStatus struct{ Settled, InProgress, Unpaid int } // pos berestimasi > 0
	} `json:"kpi"`
	Rows []struct { // satu per sesi + baris Total
		SubEventScope                                          string
		Estimate, DP, Term1, Term2, RollupPaid, Paid, Remaining int64 // Paid = DP + Term1 + Term2 + RollupPaid
		LineCount                                              int
	} `json:"rows"`
}

type SavingsOverview struct {
	KPI struct { Target, Collected, Shortfall int64; AchievementPercent float64 } `json:"kpi"`
	Contributors []struct { ID uuid.UUID; Name, RoleLabel string; Total int64; Percentage float64; EntryCount int } `json:"contributors"`
	Milestones   []struct { SubEventScope, Name, Date string; TargetAmount, SecuredAmount int64; IsSecured bool } `json:"milestones"`
	Recommendation struct { MonthsLeft int; RecommendedMonthly, PerContributor int64; Note string } `json:"recommendation"`
	Matrix  struct { // padanan PDF Hal. 7: bulan × kontributor
		Months []string                // "2026-02"
		Rows   []struct{ Month string; ByContributor map[string]int64; Total int64 }
	} `json:"matrix"`
	Entries []struct { ID, ContributorID uuid.UUID; ContributorName, RoleLabel, EntryDate string; Amount int64; Notes *string } `json:"entries"`
	Total   int64 `json:"total"`
}
```

---

## 6. Master Data

### 6.1 Enam preset skenario (PDF Hal. 4)

Berada di kode (`planner/template/budget_wedding.go`), bukan basis data. `preset_key` = `wedding_10jt` … `wedding_100jt`. Setiap paket 10 pos; total tiap kolom sama dengan nama paket (diperiksa). Scope mengikuti letak pos pada PDF Hal. 5–6 (Mahar dan Seserahan berada pada daftar Resepsi PDF; Lamaran tidak ada dalam skenario Hal. 4 sehingga tidak punya pos preset).

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

Pos `all/dana_darurat` tidak ada di PDF Hal. 5–6; ia hanya muncul lewat preset skenario. Pos `seserahan` adalah target gulungan daftar belanja (04 BE §2.4).

### 6.3 Dataset Acuan PDF (hanya Proyek Contoh dan uji emas)

Nilai persis PDF Hal. 5–6. Ini **bukan** template; hanya dimuat oleh `POST /v1/planner/projects/sample` dan test.

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

Rasio pelunasan **45,2%**; status pos: 7 lunas, 6 hanya-DP, 1 sebagian (Catering Resepsi), 0 belum. Angka-angka ini direproduksi persis oleh view `planner_budget_lines` pada pengujian DDL.

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
Peran kontributor per jenis acara: registry ([TEMPLATE_EVENT bab 4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori)). Template pos non-pernikahan: persentase (TEMPLATE_EVENT bab 15).

---

## 7. Pengujian yang Dibutuhkan
- **Uji emas PDF** (Proyek Contoh): total dan subtotal bab 6.3, rasio 45,2%, kas bebas 13.950.000, defisit 26.650.000, rekomendasi 4.584.000/bulan, milestone Lamaran dan Akad aman, Resepsi 39.750.000 (59,9%).
- Apply `wedding_75jt` ke 14 pos template: 9 diperbarui, 1 dibuat, total 75.000.000; ulangi → tidak ada perubahan nilai dan tidak ada pos baru (idempoten; pos hanya dihitung "diperbarui" bila nilainya berubah); pembayaran dan `vendor_id` tidak berubah.
- Skenario ke-4 ditolak, juga saat dua permintaan bersamaan.
- Pembayaran: `full` mengisi sisa; tahap penuh → `invalid_stage`; pos `rollup` → `rollup_locked`.
- Isolasi: kontributor/vendor/dokumen proyek lain tidak bisa ditautkan (FK komposit).
- `clean_overwrite` tanpa `confirm: "RESET"` ditolak.
