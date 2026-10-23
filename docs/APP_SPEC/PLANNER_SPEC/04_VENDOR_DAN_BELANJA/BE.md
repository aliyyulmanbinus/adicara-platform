# Menu 04: Vendor & Belanja Kebutuhan — Backend Specification

Spesifikasi backend modul Go `plannervendor` (`backend/internal/modules/plannervendor/`) untuk **direktori/pipeline vendor** (PDF Hal. 12) dan **daftar belanja pengadaan** (PDF Hal. 9–10). Aturan lintas modul (uang `BIGINT`, master sesi, template vs data contoh, kode error) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md); titik sambung yang sudah disediakan Menu 01 (`SummaryProvider`, `TemplateApplier`, `SampleOverlay`) dijelaskan di [bab 6](#6-sambungan-ke-menu-01).

> **Status (2026-10-23):** modul `plannervendor` **sudah diimplementasikan** di `backend/internal/modules/plannervendor/` (17 endpoint, provider `vendors`, applier `shopping`, overlay `vendors` dan `shopping`, `NormalizeWA` di `planner/domain/phone.go`), mengikuti dokumen ini; hasil, penyimpangan, dan tes ada di [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md) dan [bab 7.3](#73-hasil-implementasi-2026-10-23). Tabelnya sudah ada sejak migrasi `006_000_planner_vendors.sql` dan `009_000_planner_shopping.sql` di `backend/internal/migrate/sql/` (**identik** dengan blok SQL bab 1; tidak ada migrasi baru). Menu 01–03 sudah memakai tabel itu: `vendor_id` pada tugas ([02 BE §4.3](../02_TIMELINE_DAN_TUGAS/BE.md#43-hubungan-vendor)) dan pos ([03 BE §3.10](../03_KEUANGAN/BE.md#310-mengubah-memindahkan-dan-menghapus-pos)), view `planner_budget_lines` dengan gulungan belanja, dan pemindahan vendor dan belanja saat sesi dihapus ([01 BE §4.6](../01_DASHBOARD/BE.md#46-sesi-acara-master-sesi)). Frontend menu ini belum ada ([FE.md](FE.md)); fixture-nya sudah dicocokkan dengan respons HTTP backend yang berjalan (nol selisih).
>
> Dokumen ini diaudit dengan **menjalankan** SQL dan algoritmanya sebelum kode ditulis: PostgreSQL 16.2 sementara dengan 12 migrasi nyata, **backend Menu 01–03 yang sebenarnya** (Proyek Contoh dan sembilan jenis acara dibuat lewat HTTP), data emas PDF dibaca ulang dengan `pdfminer` ([bab 7.1](#71-yang-sudah-dijalankan-pada-audit-spesifikasi)). Blok DTO bab 4 dikompilasi (`go vet`) dan dibandingkan kunci demi kunci dengan tipe TypeScript FE (26 pasangan tipe, 378 jalur, nol selisih); `NormalizeWA`, `LineTotal`, dan `BuildOverview` (bab 2) dijalankan dengan tes, dan `BuildOverview` diuji terhadap Proyek Contoh yang diekspor dari basis data (delapan dari sembilan mutasi digagalkan tes; satu setara). Hal yang ditemukan dicatat di [CHANGELOG spesifikasi](../CHANGELOG.md) dan [bab 9](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk). Yang belum dijalankan **pada audit** (kode modul ini, jalur HTTP, tata letak `.xlsx`, balapan selain `lock-deal`) dijalankan saat implementasi ([bab 7.3](#73-hasil-implementasi-2026-10-23)); yang masih belum: tampilan `.xlsx` di Excel/Numbers, PostgreSQL 18 (versi Compose), image Docker, dan Nginx.
>
> Pola dokumen mengikuti [01 BE](../01_DASHBOARD/BE.md), [02 BE](../02_TIMELINE_DAN_TUGAS/BE.md), dan [03 BE](../03_KEUANGAN/BE.md): DTO tanpa tag `binding` (validasi di usecase dengan pesan `<field>: <alasan>`), bentuk respons per endpoint ([bab 3.3](#33-bentuk-respons)), validasi dan batas ([bab 3.4](#34-validasi-dan-batas)), bab sambungan ke Menu 01 ([bab 6](#6-sambungan-ke-menu-01)), pengujian dipetakan ke nama tes ([bab 7](#7-pengujian)), urutan pengerjaan ([bab 8](#8-urutan-pengerjaan-dan-definisi-selesai)), dan celah yang menunggu keputusan ([bab 9](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Versi sebelumnya (373 baris) benar pada DDL dan angka PDF tetapi memuat perilaku yang gagal bila dijalankan; koreksinya ada di CHANGELOG.

> **Acuan angka.** Seluruh angka bab 5 berasal dari PDF Hal. 9–10 dan 12 (dibaca ulang dengan `pdfminer`) dan diuji terhadap basis data ([kontrak bab 10](../00_KONTRAK_BERSAMA/README.md#10-validasi-spesifikasi)): 37 barang, estimasi **Rp 4.578.500**, 16 selesai = **2.063.800**, 21 on proses = **2.514.700**; 10 kategori vendor, 7 berstatus FIX. Vendor contoh bukan dari PDF (baris PDF hanya dummy): nama, PIC, dan nomor adalah fiktif.

---

## 1. Skema Database

### 1.1 Vendor

Migrasi `006_000_planner_vendors.sql` (sudah ada di `backend/internal/migrate/sql/`, dijalankan sebelum tugas, anggaran, belanja, dan rundown karena mereka menunjuk vendor; teks di bawah **identik** dengan berkas itu):

```sql
-- +goose Up
CREATE TABLE planner_vendors (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category_key TEXT NOT NULL CHECK (category_key ~ '^[a-z0-9_]{1,40}$'),  -- slug stabil: venue, catering, mua, ...
    category_label TEXT NOT NULL,                       -- teks tampilan; bebas untuk kategori kustom
    vendor_name TEXT NOT NULL CHECK (char_length(vendor_name) BETWEEN 1 AND 255),
    pic_name TEXT,
    phone_number TEXT,                                  -- input pengguna
    formatted_wa_number TEXT,                           -- 62xxxxxxxxxx, diisi usecase
    portfolio_url TEXT,
    budget_limit BIGINT NOT NULL DEFAULT 0 CHECK (budget_limit >= 0),   -- pagu dari skenario aktif atau manual
    agreed_price BIGINT NOT NULL DEFAULT 0 CHECK (agreed_price >= 0),   -- 0 = belum ada penawaran
    -- Positif = hemat, negatif = melebihi pagu; NULL bila pagu atau penawaran belum ada.
    price_delta BIGINT GENERATED ALWAYS AS (
        CASE WHEN agreed_price > 0 AND budget_limit > 0 THEN budget_limit - agreed_price END
    ) STORED,
    status TEXT NOT NULL DEFAULT 'rencana' CHECK (status IN ('rencana', 'negosiasi', 'fix', 'batal')),
    notes TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_vendors_project_status_idx ON planner_vendors (project_id, status);
CREATE INDEX planner_vendors_project_category_idx ON planner_vendors (project_id, category_key);

-- +goose Down
DROP TABLE IF EXISTS planner_vendors;
```

- `category_key` (slug stabil) dipakai untuk pencocokan pagu dan hitungan "kategori terkunci"; `category_label` hanya tampilan, jadi kategori kustom aman. Kunci kategori registry ada di [bab 5.1](#51-kategori-vendor-pernikahan-pdf-hal-12) dan [TEMPLATE_EVENT bab 7–15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md); kunci kategori kustom diturunkan dari label oleh `CategoryKeyFromLabel` ([bab 2.8](#28-kategori-kpi-dan-ringkasan)).
- **Tabel ini tidak punya `budget_item_id`.** FK dua arah dengan `planner_budget_items.vendor_id` tidak bisa dimigrasikan dalam urutan mana pun dan bisa menyimpang; satu-satunya arah adalah `planner_budget_items.vendor_id` (indeks unik: satu vendor, satu pos).
- **`budget_limit` hanya menyimpan pagu manual; `0` berarti "ikut skenario aktif".** Versi awal menyalin pagu skenario ke kolom ini saat vendor dibuat atau diubah. Itu menjadi basi begitu pengguna menerapkan skenario lain, karena Menu 03 tidak pernah menulis ke tabel vendor (dijalankan: catering 30 Jt di skenario 75 Jt menjadi 50 Jt di skenario 100 Jt, sedangkan salinan tetap 30 Jt), dan `limit_source` tidak punya kolom. Pagu dari skenario kini dibaca setiap kali tabel dibuka ([bab 2.1](#21-pagu-vendor-dari-skenario-aktif)).
- **`price_delta` (kolom generated) hanya benar untuk pagu manual** (`budget_limit > 0`): ia tidak tahu pagu skenario. API tidak pernah mengirimnya mentah; selisih dihitung dari pagu efektif. Kolom ini mubazir seperti `remaining_balance` pada pos ([03 BE bab 9 butir 13](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)); membuangnya dari migrasi 006 diizinkan pada tahap pengembangan ([bab 9 butir 3](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- `agreed_price = 0` berarti belum ada penawaran; status `fix` mensyaratkan `agreed_price > 0` (diperiksa usecase, tidak ada `CHECK`).
- `formatted_wa_number` diisi usecase dari `phone_number` oleh `NormalizeWA` ([bab 2.3](#23-normalisasi-nomor-whatsapp)); `phone_number` disimpan apa adanya.
- FK `(project_id, sub_event_scope)` ke master sesi bertipe `DEFERRABLE INITIALLY DEFERRED` ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)): sesi tak dikenal baru galat saat `COMMIT` (dijalankan), jadi usecase memeriksa sesi lebih dulu (`400 invalid_scope`).
- Terbukti dengan eksekusi: `category_key` di luar `^[a-z0-9_]{1,40}$` ditolak, `vendor_name` kosong atau 300 karakter ditolak, `status` di luar empat nilai ditolak; selisih generated `2.000.000` / `NULL` / `NULL` / `−1.000.000` untuk (pagu 5 jt, deal 3 jt), (0, 3 jt), (5 jt, 0), (2 jt, 3 jt).

### 1.2 Daftar belanja

Migrasi `009_000_planner_shopping.sql` (sudah ada; setelah `008`, karena menunjuk `planner_budget_items`; teks di bawah **identik** dengan berkas itu):

```sql
-- +goose Up
CREATE TABLE planner_shopping_items (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 255),
    quantity NUMERIC(12, 2) NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit TEXT,                                          -- pcs, set, box, ... (opsional)
    estimated_price BIGINT NOT NULL DEFAULT 0 CHECK (estimated_price >= 0),  -- HARGA PER SATUAN
    actual_price BIGINT NOT NULL DEFAULT 0 CHECK (actual_price >= 0),        -- HARGA PER SATUAN, 0 = belum dibeli
    estimated_total BIGINT GENERATED ALWAYS AS (round(quantity * estimated_price)::bigint) STORED,
    actual_total BIGINT GENERATED ALWAYS AS (round(quantity * actual_price)::bigint) STORED,
    -- Positif = hemat, negatif = lebih mahal; NULL bila belum ada harga riil.
    price_delta BIGINT GENERATED ALWAYS AS (
        CASE WHEN actual_price > 0 THEN round(quantity * estimated_price)::bigint - round(quantity * actual_price)::bigint END
    ) STORED,
    purchase_location TEXT,                             -- teks bebas: URL marketplace atau nama toko/alamat/telepon
    vendor_id UUID,
    budget_item_id UUID,                                -- pos anggaran tempat total belanja ini digulung (rollup)
    purpose TEXT CHECK (purpose IN ('souvenir')),       -- 'souvenir' = pembanding stok vs serah terima di Menu 05
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'proses', 'selesai')),
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id),
    FOREIGN KEY (budget_item_id, project_id)
        REFERENCES planner_budget_items (id, project_id) ON DELETE SET NULL (budget_item_id)
);

CREATE INDEX planner_shopping_project_category_idx ON planner_shopping_items (project_id, category);
CREATE INDEX planner_shopping_project_status_idx ON planner_shopping_items (project_id, status);
CREATE INDEX planner_shopping_budget_item_idx ON planner_shopping_items (budget_item_id) WHERE budget_item_id IS NOT NULL;
CREATE UNIQUE INDEX planner_shopping_template_key_idx ON planner_shopping_items (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_shopping_items;
```

- **`quantity` dan `unit`**: agar "300 seminar kit" dapat dicatat. `estimated_price`/`actual_price` adalah **harga per satuan**; `estimated_total`/`actual_total` generated (`round(quantity × harga)`, setengah ke atas). Untuk data PDF `quantity = 1`, sehingga total sama dengan harga PDF. `quantity` bertipe `NUMERIC(12,2)`: `0,005` tersimpan sebagai `0,01`, `0` ditolak `CHECK`.
- **Kolom generated meluap.** `round(9999999999.99 × 10¹³)::bigint` menjawab `bigint out of range` saat `INSERT`/`UPDATE` (dijalankan; jadi galat 500 bila tidak dijaga, seperti pos berharga satuan di [03 BE §3.4](../03_KEUANGAN/BE.md#34-pos-berharga-satuan)). Usecase menghitung total dengan aritmetika bulat dan menolak hasil di atas 10¹³ sebelum menulis ([bab 2.9](#29-konkurensi-dan-batas), `LineTotal`).
- `price_delta` muncul begitu `actual_price > 0`, apa pun statusnya (ia tidak tahu status); KPI realisasi hanya menjumlah barang `selesai` (bab 2.4).
- `budget_item_id` menentukan pos anggaran tempat total belanja digulung (bab 2.4); FK komposit `ON DELETE SET NULL`: menghapus pos membiarkan barangnya (dijalankan di [03 BE §3.10](../03_KEUANGAN/BE.md#310-mengubah-memindahkan-dan-menghapus-pos)). `purpose = 'souvenir'` menandai barang pembanding stok souvenir Menu 05. `template_key` unik per proyek (`wedding.seserahan_<nama>`, bab 2.6).
- Total belanja digulung lewat view, bukan dengan menulis ke kolom generated `paid_total` (PostgreSQL menolak: *column can only be updated to DEFAULT*).
- Template tidak menyalin tautan Shopee pribadi pembuat PDF; `purchase_location` adalah teks bebas (URL marketplace atau nama toko, alamat, telepon).

### 1.3 Tabel dan view milik modul lain yang disentuh

Modul ini tidak mengimpor modul lain ([kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)); yang berikut adalah SQL, dengan aturan yang sudah ditetapkan Menu 01–03:

| Tabel / view | Dipakai untuk | Aturan yang harus dihormati |
|---|---|---|
| `planner_budget_scenarios`, `…_scenario_items` | pagu (baca) | hanya skenario `is_active`; pos yang ikut = sesinya ada di `active_sub_events`, atau `all`, atau `active_sub_events` kosong (aturan sama dengan Smart Adjust, 03 BE §3.2) |
| `planner_budget_items` | `lock-deal` (tulis `vendor_id`, `initial_estimate`, baris baru), gulungan (tulis `rollup_source`) | `planner_budget_items_vendor_idx` (unik `vendor_id`), `planner_budget_items_key_idx` (unik `(project_id, sub_event_scope, item_key)` bila `item_key` terisi), `item_name` ≤ 150, pos berharga satuan menghitung `initial_estimate = round(quantity × unit_price)` sendiri (03 BE §3.4) |
| `planner_budget_lines` (view) | status bayar vendor, pos bersama kelompok, angka gulungan | bacalah `estimate`, `paid`, `remaining`, `payment_status` saja; jangan `SELECT … FOR UPDATE` pada view (kunci tabel dasar) |
| `planner_projects` | `skipped_vendor_categories` (tulis lewat `PATCH` Menu 01), kunci proyek (`FOR UPDATE`) | modul ini tidak menulis kolom proyek lain |
| `planner_tasks`, `planner_rundowns` | `vendor_id` ke vendor ini | menghapus vendor mengosongkannya (`ON DELETE SET NULL`); baris tetap ada (dijalankan untuk tugas) |
| `planner_shopping_items` (dibaca Menu 05) | stok souvenir untuk meja resepsionis | Menu 05 hanya membaca barang `purpose = 'souvenir'` berstatus `selesai` (bab 2.5) |

Penulisan ke `planner_budget_items` oleh modul ini adalah **pengecualian keempat** atas aturan "tidak ada SQL lintas modul" ([kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)), sejenis sinkron biaya dokumen Menu 02: `lock-deal` dan gulungan harus berada di transaksi yang sama dengan penulisan vendor dan barang.

---

## 2. Logika Bisnis

### 2.1 Pagu vendor dari skenario aktif

Registry menyimpan per kategori vendor daftar `budget_item_keys` (kunci pos anggaran/skenario yang menampungnya): pernikahan di [bab 5.1](#51-kategori-vendor-pernikahan-pdf-hal-12), jenis lain di [TEMPLATE_EVENT bab 15.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#152-pemetaan-kategori-vendor--pos-budget_item_keys). Pagu **dibaca setiap kali daftar vendor dibuka** (bukan disimpan) dari tiga pembacaan: pos skenario aktif, daftar vendor, dan pos berkunci dari view. Fungsi murni `BuildOverview` (di bawah) menggabungkannya; `GET /vendors`, widget dashboard, dan hasil `PUT` memakainya.

Aturan per vendor (bukan `batal`), berurutan:

1. `budget_limit > 0` → **`manual`**: pagu = nilai itu. Vendor dikeluarkan dari kelompok pagu.
2. Selain itu cari pos skenario untuk kunci pertama `budget_item_keys` yang ada di skenario aktif: **sesi vendor lebih dulu, lalu Umum (`all`)**. Fallback ke `all` wajib: seluruh pos skenario dan template jenis non-pernikahan berada di `all` (dijalankan pada kedelapan jenis: 0 kategori mendapat pagu bila vendor ditempatkan di sesi nyata tanpa fallback, mis. Venue di sesi `registrasi` seminar).
3. Kunci yang dipakai **≥ 2 kategori pada registry** (statis, tidak bergantung vendor yang sudah ada) adalah **pos bersama**: vendor berstatus **`group`**, `budget_limit` 0, dan dibandingkan di `limit_groups`. Pernikahan: `venue_dekor` (Venue + Decoration), `rias_busana` (MUA + Attire), `hiburan` (MC + Entertainment); jenis lain satu kelompok: peresmian `venue_dekor`, launching `produksi_av`, ulang tahun `katering_kue`, seminar `av_streaming` (dihitung dari TEMPLATE_EVENT 15.2 dengan skrip; kedelapan daftar kategori bab 7–14 sama dengan 15.2). Pencocokan lewat nama kategori ("Venue & Dekor" ≠ "Venue") tidak dipakai karena hampir selalu menghasilkan pagu 0.
4. Kunci tunggal → **`scenario`**: pagu = `cost_amount` pos itu.
5. Selain itu (kategori tanpa pos: Wedding Organizer, Hand Bouquet; atau skenario aktif tidak memuatnya; atau tidak ada skenario aktif) → **`none`**.

`price_delta` (DTO) = pagu − penawaran, hanya untuk `manual` dan `scenario` bila keduanya > 0 (agar vendor yang belum menawar tidak tampil "hemat" sebesar pagu penuh). **Kelompok**: `{item_key, sub_event_scope, limit, total_agreed, delta, members, priced, vendor_ids, line}` dengan `total_agreed` = Σ `agreed_price` anggota yang bukan `batal` (penawaran **dan** deal), `delta = limit − total_agreed` hanya bila `priced ≥ 1`, dan `line` = pos anggaran bersama kelompok bila ada (sesi pos skenario, `item_key` sama, tidak bertaut ke vendor). `delta` positif dengan `priced < members` adalah **sisa pagu**, bukan hemat (antarmuka memilih kata dengan `groupVerdict`, FE bab 5). Vendor `batal` tidak pernah anggota kelompok dan tidak dihitung di KPI.

Pembacaan skenario (`$1` = project_id; dijalankan apa adanya pada Proyek Contoh: 10 baris, `catering` resepsi 30.000.000):

```sql
-- $1 = project_id. Pos skenario AKTIF yang ikut disimulasikan (aturan sesi sama dengan Smart Adjust, 03 BE §3.2).
SELECT si.sub_event_scope, si.item_key, si.cost_amount
FROM planner_budget_scenario_items si
JOIN planner_budget_scenarios s ON s.id = si.scenario_id AND s.is_active
WHERE s.project_id = $1 AND si.item_key IS NOT NULL
  AND (cardinality(s.active_sub_events) = 0 OR si.sub_event_scope = 'all' OR si.sub_event_scope = ANY (s.active_sub_events));
```

Pembacaan vendor dan pos (dijalankan apa adanya; `payment` = baris view yang `vendor_id`-nya vendor itu, `line` kelompok = baris berkunci tanpa vendor):

```sql
-- $1 = project_id. Daftar vendor beserta pos yang menunjuknya; baris tanpa pos mengisi payment_* dengan NULL.
SELECT v.id, v.sub_event_scope, v.category_key, v.category_label, v.vendor_name, v.pic_name, v.phone_number, v.formatted_wa_number,
       v.portfolio_url, v.budget_limit, v.agreed_price, v.status, v.notes, v.sort_order,
       l.id AS line_id, l.item_name, l.estimate, l.paid, l.remaining, l.payment_status
FROM planner_vendors v
LEFT JOIN planner_budget_lines l ON l.vendor_id = v.id AND l.project_id = v.project_id
WHERE v.project_id = $1
ORDER BY v.sort_order, v.created_at;

-- Pos bersama kelompok pagu (limit_groups[].line) dan semua pos berkunci, tanpa tautan vendor:
SELECT id, sub_event_scope, item_key, item_name, estimate, paid, remaining, payment_status, vendor_id
FROM planner_budget_lines WHERE project_id = $1 AND item_key IS NOT NULL;
```

Kode acuan (`planner/domain`; **sudah dijalankan**: tiga tes terhadap Proyek Contoh yang diekspor dari basis data backend asli, ditambah kasus batal, pagu manual, anggota tanpa penawaran, ganti skenario ke 100 Jt, tanpa skenario, kategori kustom dan dilewati, dan jenis non-pernikahan; sembilan mutasi kode dicoba: delapan menjatuhkan tes, satu setara, yaitu penjumlahan penawaran bernilai 0):

```go
// backend/internal/modules/plannervendor/domain/overview.go
package domain

import (
	"sort"

	"github.com/google/uuid"
)

// Kode acuan bab 2.1 dan 2.8: fungsi murni (tanpa basis data) yang membentuk VendorOverview dari lima pembacaan.
// Dijalankan terhadap Proyek Contoh dan sembilan jenis acara (overview_test.go).

const ScopeAll = "all"

// CategorySpec: satu kategori vendor di registry jenis acara. BudgetItemKeys kosong = tanpa pos (pagu manual).
type CategorySpec struct {
	Key, Label     string
	BudgetItemKeys []string
}

// VendorRow: kolom planner_vendors yang dipakai.
type VendorRow struct {
	ID                                        uuid.UUID
	SubEventScope, CategoryKey, CategoryLabel string
	VendorName                                string
	PICName, PhoneNumber, FormattedWANumber   *string
	PortfolioURL, Notes                       *string
	BudgetLimit, AgreedPrice                  int64
	Status                                    VendorStatus
	SortOrder                                 int
}

// ScenarioRow: pos skenario AKTIF yang ikut disimulasikan (planner_budget_scenario_items).
type ScenarioRow struct {
	SubEventScope, ItemKey string
	Cost                   int64
}

// LineRow: satu baris view planner_budget_lines.
type LineRow struct {
	ID                        uuid.UUID
	SubEventScope, ItemKey    string
	ItemName                  string
	VendorID                  *uuid.UUID
	Estimate, Paid, Remaining int64
	PaymentStatus             string
}

func (l LineRow) payment() *LinePayment {
	return &LinePayment{BudgetItemID: l.ID, ItemName: l.ItemName, Estimate: l.Estimate, Paid: l.Paid, Remaining: l.Remaining, PaymentStatus: l.PaymentStatus}
}

// SharedKeys: kunci pos yang dipakai ≥ 2 kategori pada registry. Statis (tidak bergantung vendor yang ada), supaya
// menambah vendor kedua tidak mengubah cara vendor pertama dibandingkan.
func SharedKeys(cats []CategorySpec) map[string]bool {
	n := map[string]int{}
	for _, c := range cats {
		if len(c.BudgetItemKeys) > 0 {
			n[c.BudgetItemKeys[0]]++
		}
	}
	out := map[string]bool{}
	for k, c := range n {
		if c >= 2 {
			out[k] = true
		}
	}

	return out
}

type scenarioKey struct{ scope, key string }

// BuildOverview menerapkan bab 2.1 (pagu), 2.8 (kategori dan KPI). `skipped` = planner_projects.skipped_vendor_categories.
func BuildOverview(cats []CategorySpec, skipped []string, vendors []VendorRow, scenario []ScenarioRow, lines []LineRow) VendorOverview {
	cost := map[scenarioKey]int64{}
	for _, r := range scenario {
		cost[scenarioKey{r.SubEventScope, r.ItemKey}] += r.Cost
	}
	byCat := map[string]CategorySpec{}
	for _, c := range cats {
		byCat[c.Key] = c
	}
	shared := SharedKeys(cats)
	payByVendor := map[uuid.UUID]LineRow{}
	lineByKey := map[scenarioKey]LineRow{}
	for _, l := range lines {
		if l.VendorID != nil {
			payByVendor[*l.VendorID] = l
		}
		if l.ItemKey != "" {
			if _, ok := lineByKey[scenarioKey{l.SubEventScope, l.ItemKey}]; !ok {
				lineByKey[scenarioKey{l.SubEventScope, l.ItemKey}] = l
			}
		}
	}

	// pagu per vendor: sesi vendor lebih dulu, lalu Umum ('all'); kunci pertama registry yang ada di skenario.
	resolve := func(v VendorRow) (scenarioKey, bool) {
		c, ok := byCat[v.CategoryKey]
		if !ok {
			return scenarioKey{}, false
		}
		for _, k := range c.BudgetItemKeys {
			for _, scope := range []string{v.SubEventScope, ScopeAll} {
				if _, ok := cost[scenarioKey{scope, k}]; ok {
					return scenarioKey{scope, k}, true
				}
			}
		}

		return scenarioKey{}, false
	}

	sort.SliceStable(vendors, func(i, j int) bool { return vendors[i].SortOrder < vendors[j].SortOrder })
	groupIdx := map[scenarioKey]*LimitGroup{}
	var groupOrder []scenarioKey
	out := VendorOverview{Vendors: []Vendor{}, LimitGroups: []LimitGroup{}, Categories: []CategoryState{}}

	for _, v := range vendors {
		dto := Vendor{
			ID: v.ID, SubEventScope: v.SubEventScope, CategoryKey: v.CategoryKey, CategoryLabel: v.CategoryLabel, VendorName: v.VendorName,
			PICName: v.PICName, PhoneNumber: v.PhoneNumber, FormattedWANumber: v.FormattedWANumber, PortfolioURL: v.PortfolioURL,
			AgreedPrice: v.AgreedPrice, Status: v.Status, Notes: v.Notes, SortOrder: v.SortOrder, LimitSource: LimitSourceNone,
		}
		if l, ok := payByVendor[v.ID]; ok {
			dto.Payment = l.payment()
		}
		switch sk, found := resolve(v); {
		case v.BudgetLimit > 0:
			dto.BudgetLimit, dto.LimitSource = v.BudgetLimit, LimitSourceManual
		case found && shared[sk.key]:
			g := sk.key
			dto.LimitSource, dto.LimitGroup = LimitSourceGroup, &g
			if v.Status != VendorCancelled {
				grp, ok := groupIdx[sk]
				if !ok {
					grp = &LimitGroup{ItemKey: sk.key, SubEventScope: sk.scope, Limit: cost[sk], VendorIDs: []uuid.UUID{}}
					if l, ok := lineByKey[sk]; ok && l.VendorID == nil {
						grp.Line = l.payment()
					}
					groupIdx[sk] = grp
					groupOrder = append(groupOrder, sk)
				}
				grp.Members++
				grp.VendorIDs = append(grp.VendorIDs, v.ID)
				if v.AgreedPrice > 0 {
					grp.Priced++
					grp.TotalAgreed += v.AgreedPrice
				}
			}
		case found:
			dto.BudgetLimit, dto.LimitSource = cost[sk], LimitSourceScenario
		}
		if (dto.LimitSource == LimitSourceManual || dto.LimitSource == LimitSourceScenario) && dto.BudgetLimit > 0 && v.AgreedPrice > 0 {
			d := dto.BudgetLimit - v.AgreedPrice
			dto.PriceDelta = &d
		}
		out.Vendors = append(out.Vendors, dto)
	}

	for _, sk := range groupOrder {
		g := groupIdx[sk]
		if g.Priced > 0 {
			d := g.Limit - g.TotalAgreed
			g.Delta = &d
			if d < 0 {
				out.KPI.GroupsOver++
			}
		}
		out.LimitGroups = append(out.LimitGroups, *g)
	}
	out.KPI.GroupsTotal = len(out.LimitGroups)

	// kategori: registry (urutan registry) lalu kustom yang dipakai (urutan vendor)
	skip := map[string]bool{}
	for _, k := range skipped {
		skip[k] = true
	}
	type tally struct{ fix, pipeline, active int }
	counts := map[string]*tally{}
	var customOrder []VendorRow
	seenCustom := map[string]bool{}
	for _, v := range vendors {
		if v.Status == VendorCancelled {
			continue
		}
		t := counts[v.CategoryKey]
		if t == nil {
			t = &tally{}
			counts[v.CategoryKey] = t
		}
		t.active++
		if v.Status == VendorFixed {
			t.fix++
		} else {
			t.pipeline++
		}
		if _, ok := byCat[v.CategoryKey]; !ok && !seenCustom[v.CategoryKey] {
			seenCustom[v.CategoryKey] = true
			customOrder = append(customOrder, v)
		}
		switch v.Status {
		case VendorFixed:
			out.KPI.FixedVendors++
			out.KPI.FixedTotal += v.AgreedPrice
		default:
			out.KPI.Pipeline++
			if v.AgreedPrice > 0 {
				out.KPI.PipelineOffers += v.AgreedPrice
			}
		}
	}
	add := func(key, label string, custom bool) {
		st := CategoryState{Key: key, Label: label, State: "empty", Custom: custom, Skipped: skip[key]}
		if t := counts[key]; t != nil {
			st.Vendors = t.active
			if t.fix > 0 {
				st.State = "fix"
			} else if t.pipeline > 0 {
				st.State = "pipeline"
			}
		}
		out.Categories = append(out.Categories, st)
		if st.Skipped {
			out.KPI.Skipped++

			return
		}
		out.KPI.CategoriesTotal++
		if st.State == "fix" {
			out.KPI.CategoriesFixed++
		}
	}
	for _, c := range cats {
		add(c.Key, c.Label, false)
	}
	for _, v := range customOrder {
		add(v.CategoryKey, v.CategoryLabel, true)
	}

	return out
}
```

**Kode di repositori** (`domain/overview.go`, `category.go`, `shopping.go`) adalah kode acuan ini dengan perubahan yang tidak mengubah perilaku: `BuildOverview` dipecah menjadi `buildLimits`, `joinGroup`, dan `buildCategories`, tidak mengurutkan irisan masukan, memakai `ResolveKey`/`ScenarioCosts` yang juga dipakai `SharedItemOf` (pos bersama pada `lock-deal`), dan `ScenarioRow` mendapat `ItemName`; `CategoryKeyFromLabel` dipisah dari `underscoreKey` (dipakai juga untuk `template_key` barang); `shopping.go` menambah `QuantityX100`. Tes domain menjalankan data Proyek Contoh yang sama dan semua kasus tepi di bawah ([bab 7.2](#72-pengujian-pada-implementasi)).

Hasil pada Proyek Contoh (skenario aktif 75 Jt, `active_sub_events = {akad, resepsi}`):

| Kategori | Sumber pagu | Pagu | Penawaran / deal | Selisih |
|---|:-:|--:|--:|--:|
| Venue, Decoration | `group` `venue_dekor` | 20.000.000 (kelompok) | 12.000.000 + 8.000.000 = 20.000.000 | 0 (pas) |
| MUA, Attire | `group` `rias_busana` | 5.000.000 | 3.500.000 + 1.500.000 = 5.000.000 | 0 (pas) |
| MC, Entertainment | `group` `hiburan` | 3.400.000 | 2.000.000 + 1.400.000 = 3.400.000 (negosiasi) | 0 (pas) |
| Catering | `scenario` `catering` | **30.000.000** | 25.000.000 | **+5.000.000** (hemat) |
| Photography | `scenario` `dokumentasi` | 3.000.000 | 3.000.000 | 0 (pas) |
| Wedding Organizer, Hand Bouquet | `none` | – | 5.000.000 / 900.000 | – |

Versi awal wireframe FE menulis pagu Catering 25.000.000 (selisih 0); pagu skenario 75 Jt untuk `catering` resepsi adalah 30.000.000 ([03 BE §6.1](../03_KEUANGAN/BE.md#61-enam-preset-skenario-pdf-hal-4), dan [03 BE bab 6](../03_KEUANGAN/BE.md#6-master-data) sendiri mencatat estimasi pos PDF 25 Jt berbeda dari preset).

### 2.2 Kunci kerja sama (`lock-deal`) tanpa pos ganda

`POST …/vendors/:vendor_id/lock-deal` `{agreed_price, sync_to_budget_items}` dalam satu transaksi **di bawah kunci proyek** ([bab 2.9](#29-konkurensi-dan-batas)):

```sql
-- $1 = project_id. Semua penulisan 04 yang menyentuh pos anggaran atau batas jumlah memegang kunci proyek (seperti Menu 02 dan 03).
SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE;
```

1. Validasi: vendor milik proyek (`404`), `agreed_price` 1…10¹³. `status = 'fix'`, `agreed_price` diisi (langkah 1 di bawah). Tanpa `sync_to_budget_items` selesai di sini (`budget_item_id` kosong).
2. Dengan `sync_to_budget_items`, berurutan:
   - **2a. Vendor sudah punya pos** (`planner_budget_items.vendor_id`) → perbarui `initial_estimate = agreed_price`, **kecuali pos berharga satuan** (`pricing_mode = 'unit'`): estimasinya tetap `round(quantity × unit_price)` dan hasilnya membawa warning **`unit_pricing`**. Menimpa estimasi pos satuan dengan deal membuatnya tidak sama dengan jumlah × harga (dijalankan: 25.000.000 tersimpan, 22.500.000 seharusnya) dan `PUT` pos berikutnya menimpanya kembali tanpa peringatan.
   - **2b. Kategori memetakan ke pos bersama** (kunci yang dipakai ≥ 2 kategori, bab 2.1) → **tidak ditautkan dan estimasi pos tidak diubah**: pos itu mewakili beberapa vendor sekaligus, sehingga menimpanya dengan harga satu vendor menurunkan rencana (Venue Rp 12 jt akan menggantikan pos Rp 20 jt sebelum Dekorasi dikunci). Respons memuat `shared_item: {item_key, item_name, budget_item_id, plan, total_agreed}` dan warning **`shared_budget_item`**; antarmuka menawarkan "Samakan estimasi pos dengan total deal kelompok" (aksi pengguna, `PUT /budget/items/:id` Menu 03).
   - **2c. Pos khusus** (kunci hanya dipakai kategori ini) yang belum bertaut vendor dan bukan pos gulungan, dengan `item_key ∈ budget_item_keys` pada **sesi vendor, lalu Umum** (`ORDER BY (sesi = sesi vendor) DESC, sort_order`) → tautkan (`vendor_id`) dan set `initial_estimate = agreed_price` (kecuali satuan, `unit_pricing`). Pembaruan bersyarat `vendor_id IS NULL`; nol baris berubah → lanjut ke 2d.
   - **2d. Tidak ada yang cocok** → buat pos baru (`source = 'vendor'`; `category` = label kategori dipotong 60, `item_name` = nama vendor **dipotong 150 karakter** karena `item_name` pos dibatasi 150 sedangkan nama vendor 255 (dijalankan: `23514`), `initial_estimate = agreed_price`, `item_key = category_key` bila kunci itu belum dipakai pada sesi tersebut, selain itu `NULL` agar indeks unik `(sesi, item_key)` tidak bentrok); `created: true`.
3. Vendor kedua pada kategori yang sama dengan pos khusus yang sudah bertaut (mis. katering B setelah katering A) mendapat pos baru dengan `item_key NULL` (2c tidak menemukan pos yang belum bertaut); tidak ada yang ditimpa.
4. **Meninggalkan `fix`** (`PUT` atau `PATCH …/status` dengan status bukan `fix`) melepas tautan: `UPDATE … SET vendor_id = NULL` pada pos vendor itu; pos tetap ada sebagai pos biasa (`source` tetap `vendor`) dan hasilnya membawa warning **`budget_link_released`** (`count` = pos yang dilepas, tidak ada warning bila 0). Mengunci ulang vendor itu menautkan kembali pos yang sama lewat 2c bila `item_key`-nya masih kunci kategori; pos yang `item_key`-nya `NULL` (kasus 3) tidak ditemukan lagi dan dibuat satu pos baru, jadi pengguna menghapus yang lama ([bab 9 butir 16](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

Langkah 1, 2a, dan 2c (dijalankan apa adanya; `$4` = `budget_item_keys`, `$5` = sesi vendor):

```sql
-- $1 = project_id, $2 = vendor_id, $3 = agreed_price, $4 = budget_item_keys kategori (text[]), $5 = sesi vendor.
-- Satu transaksi dengan kunci proyek (lock_lock). Langkah 2a dan 2c hanya membaca; 2c-tautkan atau 2d dijalankan sesudahnya.

-- 1. vendor menjadi fix
UPDATE planner_vendors SET status = 'fix', agreed_price = $3, updated_at = now() WHERE id = $2 AND project_id = $1;

-- 2a. vendor sudah punya pos? (ya → perbarui estimasi pos itu kecuali berharga satuan, selesai)
SELECT id, pricing_mode FROM planner_budget_items WHERE project_id = $1 AND vendor_id = $2;

-- 2c. pos khusus kategori ini yang belum bertaut (sesi vendor lebih dulu, lalu Umum)
SELECT id, pricing_mode FROM planner_budget_items
WHERE project_id = $1 AND item_key = ANY ($4::text[]) AND sub_event_scope IN ($5, 'all')
  AND vendor_id IS NULL AND rollup_source IS NULL
ORDER BY (sub_event_scope = $5) DESC, sort_order
LIMIT 1;
```

Sesudahnya tepat satu dari tiga pernyataan berikut (`$9` = id pos hasil 2c atau 2a):

```sql
-- $1 = project_id, $2 = vendor_id, $3 = agreed_price, $9 = id pos hasil 2c (atau 2a). Estimasi diganti deal kecuali pos berharga satuan
-- (warning unit_pricing). Syarat vendor_id IS NULL (pada 2c) menutup balapan; nol baris → lanjut ke 2d.
UPDATE planner_budget_items
SET vendor_id = $2, initial_estimate = CASE WHEN pricing_mode = 'unit' THEN initial_estimate ELSE $3 END, updated_at = now()
WHERE id = $9 AND project_id = $1 AND vendor_id IS NULL;

-- 2a (vendor sudah bertaut): perbarui estimasi pos itu kecuali berharga satuan (warning unit_pricing); angka bayar tidak disentuh.
UPDATE planner_budget_items
SET initial_estimate = CASE WHEN pricing_mode = 'unit' THEN initial_estimate ELSE $3 END, updated_at = now()
WHERE project_id = $1 AND vendor_id = $2;

-- $1 = project_id, $2 = vendor_id, $3 = agreed_price, $5 = sesi vendor, $6 = vendor_name, $7 = category_label, $8 = category_key.
-- 2d. pos baru: nama dipotong 150 karakter (batas item_name), kategori 60; item_key = category_key hanya bila belum dipakai pada sesi itu.
INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, vendor_id, initial_estimate, sort_order)
SELECT gen_random_uuid(), $1, $5, left($7, 60), left($6, 150),
       CASE WHEN NOT EXISTS (SELECT 1 FROM planner_budget_items WHERE project_id = $1 AND sub_event_scope = $5 AND item_key = $8) THEN $8 END,
       'vendor', $2, $3, COALESCE(max(sort_order), 0) + 1
FROM planner_budget_items WHERE project_id = $1;
```

Pelepasan (langkah 4):

```sql
-- $1 = project_id, $2 = vendor_id. Dijalankan bila status vendor fix berubah menjadi bukan fix (PUT atau PATCH …/status); jumlah baris yang berubah = count warning budget_link_released.
UPDATE planner_budget_items SET vendor_id = NULL, updated_at = now() WHERE project_id = $1 AND vendor_id = $2;
```

**Hasil khas** (data emas, reimplementasi acuan dalam SQL/plpgsql, lalu blok di atas dijalankan apa adanya): Katering Bu Minah (500 pax) ditautkan ke pos `catering` resepsi (estimasi 25.000.000, terbayar 15.000.000, `partial` tetap); Studio Cahaya ke `dokumentasi`; Sasana Kriya dan Pelaminan Mega Indah: pos bersama, tidak ditautkan; Harmony WO: pos baru `wo` berestimasi 5.000.000 (total proyek 74.150.000 → 79.150.000); kunci ulang katering dengan harga 24 jt → pos 24.000.000; nama vendor 255 karakter menghasilkan `item_name` 150 karakter. Respons ringkas:

| Situasi | `budget_item_id` | `created` | `shared_item` | `warnings` |
|---|---|:-:|---|---|
| tanpa `sync_to_budget_items` | – | false | – | – |
| vendor sudah bertaut | pos itu | false | – | `unit_pricing` bila pos satuan |
| pos khusus ditemukan | pos itu | false | – | `unit_pricing` bila pos satuan |
| pos bersama | – | false | ada | `shared_budget_item` |
| tidak ada yang cocok | pos baru | true | – | – |

Pembayaran vendor dibaca dari pos (`paid`, `remaining`, `payment_status` pada view), jadi **status bayar di tabel vendor selalu berasal dari Tab Realisasi (Menu 03 Tab 3)**; untuk pos bersama, status bayarnya ada di `limit_groups[].line`. Menerapkan ulang skenario ([03 BE §3.2](../03_KEUANGAN/BE.md#32-menerapkan-skenario-smart-adjust)) memperbarui estimasi pos yang bertaut vendor ke nilai skenario, jadi deal dan estimasi pos bisa berbeda lagi; Vendor memuat `payment.estimate` dan antarmuka menandai ketidaksamaan itu (lencana "Pos ≠ deal", warning `estimate_differs` pada `PUT`, [bab 9 butir 7](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

**Balapan (terbukti):** dua `lock-deal` serentak untuk dua vendor sekategori (pos khusus yang sama, belum bertaut) tanpa kunci proyek **sama-sama dijawab "ditautkan"**, padahal tautan yang pertama tertimpa (`vendor_id` berganti; indeks unik per vendor tidak keberatan karena nilainya berbeda), sehingga deal vendor itu tidak pernah masuk anggaran. Dengan `SELECT … FOR UPDATE` pada `planner_projects`: satu vendor ditautkan, yang lain mendapat pos baru.

### 2.3 Normalisasi nomor WhatsApp

Fungsi murni `NormalizeWA(input) (string, error)` di **`planner/domain/phone.go`** (bukan di `plannervendor`): Menu 05 memakainya untuk nomor tamu dan Menu 02–06 hanya boleh mengimpor `planner/domain`. Aturan lama ("`+62…` → `62…`, `0…` → `62…`, selain itu galat") menghasilkan nomor yang salah untuk salah ketik yang umum, dan tidak menyebut `.`, `00`, serta nomor kosong; aturan sekarang:

```go
// backend/internal/modules/planner/domain/phone.go
package domain

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid phone number")

// NormalizeWA mengubah nomor ponsel Indonesia isian pengguna menjadi bentuk wa.me (62…, hanya digit).
// Kosong (setelah dipangkas) → "" tanpa galat: nomor opsional. Aturan, berurutan:
//  1. buang spasi, '-', '.', '(' dan ')'; satu-satunya karakter lain yang boleh adalah '+' di awal; selain itu galat;
//  2. awalan "+" atau "00" dibuang (internasional); selain itu bila diawali "0" diganti "62" (nasional);
//  3. hasil harus diawali "62" (hanya nomor Indonesia); "0" sesudah "62" dibuang ("+62 0812…" salah ketik yang umum);
//  4. panjang akhir 10–15 digit.
//
// Nomor tanpa awalan ("8123456789") ditolak: tidak ada tebakan.
func NormalizeWA(input string) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(input) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return "", ErrInvalidPhone
		}
	}
	s := b.String()
	if s == "" {
		return "", nil
	}

	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	}
	if !strings.HasPrefix(s, "62") {
		return "", ErrInvalidPhone
	}
	s = "62" + strings.TrimPrefix(s[2:], "0")
	if len(s) < 10 || len(s) > 15 {
		return "", ErrInvalidPhone
	}

	return s, nil
}
```

Vektor bersama BE dan FE: [fixtures/planner-wa-vectors.json](fixtures/planner-wa-vectors.json) (26 kasus; `wa: null` = `400 invalid_phone`, `""` = kosong). Contoh: `081298765432` → `6281298765432`; `+62812-9876-5432` → `6281298765432`; `+62 0812 9876 5432` → `6281298765432`; `0062 812…` → `62812…`; `8129876543` (tanpa awalan), `+65 9123 4567` (bukan Indonesia), `0812abc`, dan `08123456` (terlalu pendek) → `invalid_phone`. Hanya nomor Indonesia yang diterima ([bab 9 butir 8](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)); nomor tetap ("021 5551234") lolos tetapi tidak tentu punya WhatsApp.

Nomor disimpan dua bentuk (`phone_number` apa adanya, `formatted_wa_number` ternormalisasi, NULL bila kosong). Tautan dibentuk klien: `https://wa.me/6281298765432?text=<url-encoded>` dengan teks *"Halo {PIC atau nama vendor}, saya ingin menanyakan perihal kerja sama untuk acara {judul acara} pada sesi {nama sesi}."* (klausa sesi dihilangkan untuk Umum; FE bab 5, `waMessage`/`waLink`).

### 2.4 Daftar belanja digulung ke anggaran (*rollup*)

Total belanja tidak disalin ke anggaran; pos anggaran dapat **dihitung dari daftar belanja** melalui view `planner_budget_lines` ([03 BE §1.2](../03_KEUANGAN/BE.md#12-view-baca-planner_budget_lines)): `estimate = Σ estimated_total` dan `paid = Σ actual_total` untuk barang berstatus `selesai`, atas barang yang `budget_item_id`-nya menunjuk pos itu. Status pelunasan pos ikut dihitung view (`partial` bila terbayar < estimasi).

**Pos bawaan** (`default_target` pada `GET /shopping-items`) = pos dengan `item_key` gulungan registry jenis acara pada sesi mana pun (`sort_order` terkecil): pernikahan `seserahan` (ada di 14 pos template, sesi `resepsi`); delapan jenis lain satu pos 15.1 yang dipilih ([TEMPLATE_EVENT 15.4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#154-pos-gulungan-belanja)); `other` tidak punya yang cocok sehingga `perlengkapan` dibuat bila belum ada (`source = 'manual'`, sesi Umum). Versi awal menyebut `perlengkapan` untuk semua jenis non-pernikahan, tetapi tidak satu pun dari 59 pos persentase berkunci itu, jadi gulungan selalu membuat pos baru di luar persentase pagu ([03 BE bab 9 butir 29](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk); keputusan bawaan di [bab 9 butir 6](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

**Pos cadangan.** Pos bawaan peresmian (`prosesi_vip`), wisuda (`toga_atribut`), dan seminar (`kit_sertifikat`) juga pos khusus kategori vendor (`mc`, `toga`, `percetakan`; TEMPLATE_EVENT 15.2), sehingga `lock-deal` menautkannya ke vendor dan gulungan ke pos itu ditolak (`must not be linked to a vendor or document`). Ditemukan saat implementasi oleh tes yang menjalankan alur penuh untuk sembilan jenis acara. Bila pos bawaan **ada tetapi tidak dapat digulung** (bertaut ke vendor atau dokumen, atau berharga satuan), tujuan bawaan (dan `default_target`) beralih ke pos cadangan `perlengkapan` (Kategori Perlengkapan, sesi Umum, `source = 'manual'`, dibuat sekali bila belum ada); `budget_item_id` yang dikirim klien tidak pernah dialihkan. Arah sebaliknya sudah benar: `lock-deal` melewati pos yang sedang menggulung (langkah 2c) sehingga vendor mendapat pos sendiri. Keputusan pemilik: [bab 9 butir 22](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk).

`POST …/shopping-items/link-to-budget` `{budget_item_id?, item_ids?, confirm?}` di bawah kunci proyek:

- **Pos tujuan**: `budget_item_id` bila dikirim (harus pos proyek ini, bukan berharga satuan, tidak bertaut ke vendor atau dokumen; alasan `must be a budget line of this project` / `must not be a unit-priced line` / `must not be linked to a vendor or document`), selain itu pos bawaan; bila belum ada, dibuat (`created: true`).
- **Barang**: `item_ids` bila ada (harus barang proyek ini: `must be items of this project`), selain itu semua barang yang **belum bertaut ke pos rollup lain** (`budget_item_id IS NULL` atau sudah menunjuk pos ini).
- Mengatur `rollup_source = 'shopping'` pada pos. Bila pos sudah punya pembayaran tersimpan (DP/termin) dan belum rollup → `409 rollup_conflict` kecuali `confirm: true` (pembayaran tersimpan tetapi diabaikan selama gulungan aktif; di Menu 03, `rollup_locked` menolak pembayaran baru dan Apply skenario melewati pos itu). Pos yang sudah rollup dan dikirim lagi: idempoten.
- `POST …/shopping-items/unlink-from-budget` `{budget_item_id}`: **melepas semua barang** (`budget_item_id = NULL`) lalu mencabut gulungan; pos kembali memakai angka tersimpannya. Versi awal hanya mencabut gulungan dan membiarkan barang menunjuk pos (dijalankan: 37 barang tetap menunjuk), yang membuat "semua barang yang belum bertaut" berikutnya ambigu.
- **Barang yang dibuat sesudahnya ikut** bila `POST /shopping-items` membawa `budget_item_id` pos rollup tujuan (antarmuka mengisinya dari banner); tanpa itu barang baru tidak masuk hitungan (dijalankan: estimasi pos tetap 4.578.500 setelah barang baru ditambahkan). `PUT` barang tidak mengubah tautan.

```sql
-- $1 = project_id, $2 = item_key bawaan jenis acara, $3 = budget_item_id (hasil langkah pertama atau dikirim klien), $4 = item_ids (uuid[])
-- Kunci proyek dulu (SELECT … FOR UPDATE, bab 2.2). Pos tujuan bawaan: kunci registry, sesi mana pun, sort_order terkecil.
SELECT id, sub_event_scope, item_name, pricing_mode, rollup_source, vendor_id, admin_doc_id,
       (dp_amount + term_1_amount + term_2_amount) > 0 AS has_payments
FROM planner_budget_items WHERE project_id = $1 AND item_key = $2 ORDER BY sort_order LIMIT 1;

-- aktifkan gulungan, lalu tautkan barang: semua yang belum bertaut ke pos rollup lain, atau hanya item_ids
UPDATE planner_budget_items SET rollup_source = 'shopping', updated_at = now() WHERE id = $3 AND project_id = $1;
UPDATE planner_shopping_items SET budget_item_id = $3, updated_at = now()
WHERE project_id = $1 AND (budget_item_id IS NULL OR budget_item_id = $3) AND ($4::uuid[] IS NULL OR id = ANY ($4::uuid[]));

-- angka efektif untuk respons
SELECT id, item_name, sub_event_scope, estimate, paid,
       (SELECT count(*) FROM planner_shopping_items WHERE budget_item_id = l.id) AS linked_items
FROM planner_budget_lines l WHERE id = $3 AND project_id = $1;
```

```sql
-- $1 = project_id, $3 = budget_item_id. Lepas semua barang, lalu cabut gulungan: pos kembali memakai angka tersimpannya.
UPDATE planner_shopping_items SET budget_item_id = NULL, updated_at = now() WHERE project_id = $1 AND budget_item_id = $3;
UPDATE planner_budget_items SET rollup_source = NULL, updated_at = now() WHERE id = $3 AND project_id = $1 AND rollup_source IS NOT NULL;
```

Dijalankan pada Proyek Contoh dengan pos `seserahan` yang sudah punya pembayaran PDF (estimasi 3.500.000, terbayar 3.500.000, `settled`): setelah menautkan 37 barang pos menjadi estimasi **4.578.500**, terbayar **2.063.800**, sisa **2.514.700**, `partial`; total proyek 74.150.000 → 75.228.500 dan terbayar 33.550.000 → 32.113.800. **Proyek Contoh tidak mengaktifkan gulungan** agar angka PDF Hal. 5–6 tetap utuh.

Status barang `selesai` dengan `actual_price = 0` otomatis mengisi `actual_price = estimated_price` (dapat diedit); harga riil yang sudah ada tidak ditimpa, dan kembali ke `proses`/`belum` tidak menghapusnya. Aturan ini dijalankan server (`PATCH …/status`) agar konsisten di semua klien:

```sql
-- $1 = project_id, $2 = item_id, $3 = status baru. Menandai selesai mengisi harga riil dengan estimasi bila 0; harga riil yang ada tidak ditimpa.
UPDATE planner_shopping_items
SET status = $3,
    actual_price = CASE WHEN $3 = 'selesai' AND actual_price = 0 THEN estimated_price ELSE actual_price END,
    updated_at = now()
WHERE id = $2 AND project_id = $1
RETURNING id, status, estimated_price, actual_price, actual_total;
```

### 2.5 Stok souvenir

Barang `purpose = 'souvenir'` berstatus `selesai` membentuk stok = `floor(Σ quantity)` (jumlah boleh berdesimal, stok bilangan bulat; dijalankan: 300 + 50,5 selesai dan 20 proses → 350). `souvenir_stock` pada KPI belanja hilang bila tidak ada barang bertanda souvenir sama sekali (0 bila ada tetapi belum ada yang selesai). Serah terima dari Menu 05 (`souvenir_received`). Menu 05 menampilkan `stok − diserahkan` ([05 BE §2.5](../05_BUKU_TAMU/BE.md#25-meja-resepsionis-hari-h)); ia membaca `planner_shopping_items` dengan SQL (baca-saja, [bab 6](#6-sambungan-ke-menu-01)). Proyek Contoh tidak punya barang souvenir (PDF Hal. 9–10 hanya seserahan), jadi stok kosong.

```sql
-- $1 = project_id. KPI belanja (selalu seluruh proyek) dan stok souvenir.
SELECT COALESCE(sum(estimated_total), 0)::bigint                                    AS estimated_total,
       COALESCE(sum(actual_total) FILTER (WHERE status = 'selesai'), 0)::bigint      AS actual_total,
       count(*) FILTER (WHERE status = 'selesai')                                    AS done,
       count(*) FILTER (WHERE status = 'proses')                                     AS in_progress,
       count(*) FILTER (WHERE status = 'belum')                                      AS not_started,
       count(*)                                                                      AS total,
       count(*) FILTER (WHERE purpose = 'souvenir')                                  AS souvenir_items,
       COALESCE(floor(sum(quantity) FILTER (WHERE purpose = 'souvenir' AND status = 'selesai')), 0)::bigint AS souvenir_stock
FROM planner_shopping_items WHERE project_id = $1;
```

### 2.6 Template

`POST …/shopping-items/apply-template` `{mode, confirm?}`: satu template per jenis acara, jadi tanpa `template_key` di permintaan. Pernikahan: 37 barang seserahan (bab 5.3); jenis lain `400 invalid_template` (hanya kategori di dropdown). `mode`:

- **`append`** (bawaan untuk applier wizard): lewati `template_key` yang sudah ada (`ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`; dijalankan: tiga barang dihapus pengguna → `created 3, skipped 1`). Barang template berstatus `belum`, `actual_price = 0`, `purchase_location` kosong, `quantity 1`, sesi `all`, `sort_order` melanjutkan.
- **`replace_template`**: menghapus semua barang ber-`template_key` lalu menyisipkan template segar; barang buatan pengguna aman. **`confirm: true` wajib bila ada barang yang dihapus yang membawa pekerjaan pengguna**: status bukan `belum`, harga riil, tautan pos/vendor, atau tanda souvenir; selain itu `400 invalid_input: confirm: required`. Hasil membawa warning `budget_link_lost` (`count` = barang terhapus yang tertaut ke pos gulungan; angka pos dihitung ulang). Versi awal tidak mengatur ini (02 dan 03 sudah).

`template_key` = `wedding.seserahan_` + nama barang huruf kecil bergaris-bawah (aturan `CategoryKeyFromLabel` tanpa pemotongan; 37 kunci unik, terpanjang 44 karakter, mis. `wedding.seserahan_dompet_tas_kecil_sling_bag`). Dampak pada Proyek Contoh (semua 37 barang bersumber template dan sudah berstatus): `removed 37, carry_work 37` → konfirmasi wajib.

```sql
-- $1 = project_id. Satu INSERT per barang template (contoh satu baris). created = baris yang masuk, skipped = template_key yang sudah ada.
INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, quantity, estimated_price, status, sort_order, template_key)
VALUES (gen_random_uuid(), $1, 'all', 'Perangkat Alat Solat', 'Mukena', 1, 194000, 'belum', 1, 'wedding.seserahan_mukena')
ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING
RETURNING id;

-- $1 = project_id. Dampak replace_template: semua barang ber-template_key dihapus; carry_work > 0 mewajibkan confirm: true.
SELECT count(*) AS removed,
       count(*) FILTER (WHERE status <> 'belum' OR actual_price > 0 OR budget_item_id IS NOT NULL OR vendor_id IS NOT NULL OR purpose IS NOT NULL) AS carry_work,
       count(*) FILTER (WHERE budget_item_id IS NOT NULL) AS rollup_links
FROM planner_shopping_items WHERE project_id = $1 AND template_key IS NOT NULL;
```

### 2.7 Ekspor

Dua berkas `.xlsx` (`excelize`, disetujui pemilik; kontrak bab 1.4): `GET …/vendors/export-contacts?format=xlsx` dan `GET …/shopping-items/export-shopping-list?format=xlsx`. **Hanya `xlsx`**: versi awal menjanjikan `pdf`, tetapi pustaka PDF belum disetujui (kontrak bab 1.4); tanpa `format=xlsx` → `400 invalid_input: format: must be xlsx` (seperti 03). 10 ekspor/menit/pengguna (`429 rate_limited`); nama berkas dari slug ASCII judul proyek + tanggal (`vendor-pernikahan-sinta-arif-2026-10-22.xlsx`, `belanja-…`); sel teks di-escape ([kontrak bab 2 butir 5](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan); awalan `'` tampak di sel, [02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)); `Cache-Control: no-store` (data pribadi). Tata letak berikut adalah rancangan (belum dibuat prototipe dan belum dibuka di Excel/Numbers/LibreOffice):

| Berkas | Lembar | Kolom | Baris |
|---|---|---|---|
| kontak vendor | `Kontak Vendor` | No · Sesi · Kategori · Vendor · PIC · Telepon · WhatsApp (tautan `https://wa.me/…`) | vendor `fix` menurut `sort_order`; **tanpa harga, pagu, dan catatan** karena lembar ini untuk WO/panitia |
| daftar belanja | `Daftar Belanja` | No · Sesi · Kategori · Barang · Jumlah · Satuan · Harga satuan estimasi · Total estimasi · Harga satuan riil · Total riil · Selisih · Alamat pembelian · Status · Catatan | semua barang menurut `sort_order`; baris `TOTAL` di bawah: Σ total estimasi, Σ total riil (barang selesai), jumlah selesai/proses/belum |

Angka ditulis sebagai angka (rupiah utuh ≤ 10¹³ tepat dalam `float64`), bukan teks; nama sesi dari master sesi (`all` = "Umum").

### 2.8 Kategori, KPI, dan ringkasan

- **Kategori** = kategori registry jenis acara (urutan registry) ∪ kategori kustom yang dipakai vendor yang bukan `batal` (urutan kemunculan). Tiap kategori: `state` `fix` (≥ 1 vendor fix) · `pipeline` (≥ 1 rencana/negosiasi, tanpa fix) · `empty`; `vendors` = jumlah bukan batal; `custom`; `skipped` (kuncinya ada di `planner_projects.skipped_vendor_categories`).
- **KPI** (`VendorKPI`, selalu seluruh proyek): `categories_total` (kategori tanpa yang dilewati), `categories_fixed` (di antaranya yang `fix`), `skipped`, `fixed_vendors`, `pipeline` (rencana + negosiasi), `fixed_total` (Σ `agreed_price` fix), `pipeline_offers` (Σ `agreed_price` > 0 vendor rencana dan negosiasi), `groups_total`, `groups_over` (`delta < 0`). Pada Proyek Contoh: 7 dari 10, 7, 3, 58.000.000, 4.300.000, 3, 0.
- **Kategori yang dilewati** hanyalah penanda yang mengeluarkan kategori dari hitungan; ia tidak melarang vendor ditambahkan (antarmuka menawarkannya hanya pada kategori kosong). Daftar diganti lewat `PATCH /projects/:id` `{skipped_vendor_categories}` (Menu 01: mengganti seluruh daftar, dideduplikasi, ≤ 50, tiap kunci `^[a-z0-9_]{1,40}$`).
- **`category_key` kustom**: pada `POST`/`PUT` vendor, `category_key` hanya boleh kosong atau kunci registry jenis acara proyek; kategori kustom dikirim sebagai `category_label` saja dan kuncinya diturunkan server (`CategoryKeyFromLabel`, di bawah; juga pada `PUT`, supaya tidak ada dua jalur). Label yang menghasilkan kunci registry (mis. "Venue") bergabung dengan kategori registry. `ginutil.Slug` tidak dipakai: ia menghasilkan tanda hubung dan nilai bawaan `proyek` untuk nama berkas, sedangkan kolom ini `^[a-z0-9_]{1,40}$`.
- **Provider `vendors`** (widget Menu 01) memakai fungsi yang sama (bagian KPI); kueri 01 BE §5.2 tetap sah sebagai pembacanya dan dijalankan terhadap data emas: `fixed_vendors 7, pipeline 3`, 10 kategori terpakai, 7 fix.

```sql
-- $1 = project_id. Provider vendors (01 BE §5.2, tidak berubah); CategoriesTotal/Fixed dihitung di Go dari dua larik ini.
SELECT count(*) FILTER (WHERE status = 'fix') AS fixed_vendors,
       count(*) FILTER (WHERE status IN ('rencana', 'negosiasi')) AS pipeline,
       COALESCE(array_agg(DISTINCT category_key) FILTER (WHERE status <> 'batal'), '{}') AS categories_in_use,
       COALESCE(array_agg(DISTINCT category_key) FILTER (WHERE status = 'fix'), '{}') AS categories_fixed
FROM planner_vendors WHERE project_id = $1;
```

Kode acuan penurunan kunci kategori kustom (**sudah dijalankan**: `Photobooth` → `photobooth`, `Dekorasi & Tenda` → `dekorasi_tenda`, `Café Étoile` → `cafe_etoile`, emoji atau aksara non-Latin → `kategori`, dipotong 40; aturan yang sama dengan `planner/domain.ScopeKeyFromName`):

```go
// backend/internal/modules/plannervendor/domain/category.go
package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxCategoryKeyLen adalah batas kolom (^[a-z0-9_]{1,40}$).
const MaxCategoryKeyLen = 40

// CategoryKeyFromLabel menurunkan category_key kategori kustom: huruf kecil ASCII (aksen dilipat), selain [a-z0-9] menjadi
// "_", rentetan dan ujung "_" dibuang, dipotong 40. Label tanpa huruf Latin atau angka (emoji, aksara lain) menjadi "kategori".
// Aturan sama dengan planner/domain.ScopeKeyFromName; label "Venue" menghasilkan kunci registry `venue`, jadi kategori kustom
// bernama sama dengan kategori registry bergabung dengannya.
func CategoryKeyFromLabel(label string) string {
	var b strings.Builder
	pendingSep := false
	for _, r := range norm.NFD.String(label) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingSep && b.Len() > 0 {
				b.WriteByte('_')
			}
			pendingSep = false
			b.WriteRune(r)
		} else {
			pendingSep = true
		}
	}

	key := b.String()
	if len(key) > MaxCategoryKeyLen {
		key = strings.TrimRight(key[:MaxCategoryKeyLen], "_")
	}
	if key == "" {
		return "kategori"
	}

	return key
}
```

### 2.9 Konkurensi dan batas

- **Kunci proyek** (`SELECT … FOR UPDATE` pada `planner_projects`, repository modul ini seperti Menu 02 dan 03) dipegang oleh: `POST /vendors`, `PUT /vendors` dan `PATCH …/status` (melepas tautan), `lock-deal`, `POST /shopping-items` (batas jumlah dan `sort_order = max + 1`), `apply-template`, `link-to-budget`, dan `unlink-from-budget`. Tanpa kunci, `lock-deal` serentak menimpa tautan diam-diam (bab 2.2) dan `max(sort_order) + 1` serta batas jumlah bocor (terbukti di Menu 02). `PUT`/`PATCH status`/`DELETE` barang tidak butuh kunci (satu baris).
- **Batas jumlah** (usulan; [bab 9 butir 1](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)): 200 vendor dan 1.000 barang per proyek, diperiksa di bawah kunci, semua-atau-tidak (`apply-template` dihitung setelah penyisipan): `409 vendor_limit_reached`, `409 shopping_item_limit_reached`. Daftar tidak berpaginasi, jadi batas ini juga batas tampilan.
- **Hapus**: `DELETE` vendor membiarkan pos (tautan `vendor_id` jadi `NULL`), tugas, belanja, dan agenda (`SET NULL`); `DELETE` barang tidak menyentuh pos (angka gulungan dihitung ulang view).
- **Nilai**: uang 0…10¹³; `quantity` 0,01…1.000.000 dua desimal; total baris `round(quantity × harga)` ≤ 10¹³, dihitung dengan aritmetika bulat sebelum menulis (`quantity` dibaca sebagai ratusan, `q100`). Kode acuan, **dijalankan terhadap 300 pasangan yang dihitung PostgreSQL 16.2** (`round(q100/100 × harga)`, termasuk tepi tepat 10¹³ dan 10¹³+1; 33 pasangan di atas batas) tanpa selisih:

```go
// backend/internal/modules/plannervendor/domain/shopping.go
package domain

import (
	"math"
	"math/bits"
)

// MaxMoney: batas semua nilai uang (kontrak bab 6): aman bagi Number JavaScript dan cukup untuk bigint.
const MaxMoney int64 = 10_000_000_000_000

// MaxQuantityX100 = 1.000.000,00: batas jumlah barang (dua desimal) dalam ratusan.
const MaxQuantityX100 int64 = 100_000_000

// LineTotal = round(quantity × price) tanpa float, dengan quantity dalam ratusan (NUMERIC(12,2) → q100). Pembulatan sama dengan
// kolom generated `round(quantity * price)::bigint` PostgreSQL (setengah ke atas untuk nilai positif). ok=false bila hasil
// melebihi MaxMoney: usecase menolak sebelum menulis, karena kolom generated menjawab `bigint out of range` (500).
func LineTotal(q100, price int64) (total int64, ok bool) {
	if q100 < 0 || price < 0 {
		return 0, false
	}
	hi, lo := bits.Mul64(uint64(q100), uint64(price))
	if hi != 0 || lo > ^uint64(0)-50 {
		return 0, false
	}
	t := (lo + 50) / 100
	if t > uint64(MaxMoney) {
		return 0, false
	}

	return int64(t), true
}

// ShoppingKPIOf menjumlahkan daftar belanja (bab 2.4): estimasi semua barang, realisasi hanya barang selesai.
func ShoppingKPIOf(items []ShoppingItem) ShoppingKPI {
	var k ShoppingKPI
	var stockX100 int64
	var hasSouvenir bool
	for _, it := range items {
		k.Total++
		k.EstimatedTotal += it.EstimatedTotal
		switch it.Status {
		case ShoppingDone:
			k.Done++
			k.ActualTotal += it.ActualTotal
		case ShoppingInProgress:
			k.InProgress++
		default:
			k.NotStarted++
		}
		if it.Purpose != nil && *it.Purpose == "souvenir" {
			hasSouvenir = true
			if it.Status == ShoppingDone {
				stockX100 += int64(math.Round(it.Quantity * 100)) // ratusan: tanpa penjumlahan float
			}
		}
	}
	if hasSouvenir {
		s := stockX100 / 100
		k.SouvenirStock = &s
	}

	return k
}
```

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, dengan `BearerAuth` + `ProjectGuard` + `WriteGuard` (rantai middleware dan pemasangan di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)). Respons `{"data": ...}`, galat `{"error": {code, message}}`. Id anak di URL (`:vendor_id`, `:item_id`) harus UUID kanonik 36 karakter, selain itu `404 not_found`. Rute statis (`/vendors/export-contacts`, `/shopping-items/apply-template`, …) berdampingan dengan rute berparameter; Gin menyelesaikan segmen statis lebih dulu (terbukti untuk `/projects/summary` di Menu 01). Penulisan di proyek `archived` ditolak `409 project_read_only`; `GET` (termasuk ekspor) tetap boleh. Total **17 endpoint** (8 vendor + 9 belanja). Mengganti kategori yang dilewati memakai `PATCH /projects/:id` Menu 01, bukan endpoint di sini.

### 3.1 Vendor

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/vendors?scope=&status=&category=` | `VendorOverview`: KPI, `categories`, `limit_groups` (selalu seluruh proyek) dan `vendors` (disaring). `scope` seperti Menu 02: tanpa parameter = semua sesi, `all` = hanya Umum; `status` salah satu dari empat nilai; `category` = `category_key`. Urut `sort_order`, lalu waktu dibuat. |
| `GET` | `/vendors/:vendor_id` | `Vendor` dengan `payment`. (Versi awal juga menjanjikan tugas dan agenda yang menunjuk vendor; tidak ada tampilan yang memakainya, jadi dibuang, [bab 9 butir 10](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk).) |
| `POST` | `/vendors` | Tambah kandidat. `201 VendorResult`. `409 vendor_limit_reached`. |
| `PUT` | `/vendors/:vendor_id` | Mengganti penuh ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)): `pic_name`, `phone_number`, `portfolio_url`, `notes` yang tidak dikirim dikosongkan; `budget_limit` 0 = ikut skenario. Mengubah `agreed_price` pada vendor yang bertaut ke pos dengan estimasi berbeda → warning `estimate_differs`; meninggalkan `fix` → `budget_link_released` (bab 2.2). Estimasi pos tidak pernah diubah di sini (hanya `lock-deal`). |
| `PATCH` | `/vendors/:vendor_id/status` | `{status}` untuk pipeline satu klik. `fix` hanya bila `agreed_price > 0` (`agreed_price: must be greater than 0 when status is fix`); aturan yang sama dengan `PUT`. |
| `POST` | `/vendors/:vendor_id/lock-deal` | `{agreed_price, sync_to_budget_items}` → `LockDealResult` (bab 2.2). |
| `DELETE` | `/vendors/:vendor_id` | `204`; rujukan di tugas/pos/belanja/agenda menjadi kosong (`SET NULL`), pos anggaran tetap. |
| `GET` | `/vendors/export-contacts?format=xlsx` | Kontak vendor `fix` untuk WO/panitia, tanpa harga (bab 2.7). |

### 3.2 Belanja

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/shopping-items?scope=&category=&status=` | `ShoppingOverview`: KPI, `rollups`, `default_target` (seluruh proyek) dan `items` (disaring). `category` = teks kategori persis. |
| `POST` | `/shopping-items` | Tambah barang; `budget_item_id` opsional (pos rollup tujuan). `201 ShoppingItem`. `409 shopping_item_limit_reached`. |
| `PUT` | `/shopping-items/:item_id` | Mengganti penuh, **kecuali** tautan rollup (`budget_item_id`), `template_key`, dan urutan, yang tidak berubah. `ShoppingItem`. |
| `PATCH` | `/shopping-items/:item_id/status` | `{status}`: `belum` → `proses` → `selesai`; `selesai` mengisi harga riil dengan estimasi bila 0 (bab 2.4). `ShoppingItem`. |
| `DELETE` | `/shopping-items/:item_id` | `204`. |
| `POST` | `/shopping-items/apply-template` | `{mode, confirm?}` (bab 2.6). |
| `POST` | `/shopping-items/link-to-budget` | bab 2.4. |
| `POST` | `/shopping-items/unlink-from-budget` | bab 2.4. |
| `GET` | `/shopping-items/export-shopping-list?format=xlsx` | Rekap belanja (bab 2.7). |

Tidak ada `reorder`: daftar urut `sort_order` (urutan dibuat; kolom "No" FE dihitung dari urutan itu) dan wireframe tidak menyediakan seret-lepas.

### 3.3 Bentuk respons

| Endpoint | `data` |
|---|---|
| `GET /vendors` | `VendorOverview` `{kpi, categories[], limit_groups[], vendors[]}` |
| `GET /vendors/:id` | `Vendor` |
| `POST /vendors` (`201`), `PUT`, `PATCH …/status` | `VendorResult` `{vendor, warnings[]}` |
| `POST …/lock-deal` | `LockDealResult` `{vendor, budget_item_id?, created, shared_item?, warnings[]}` |
| `GET /shopping-items` | `ShoppingOverview` `{kpi, rollups[], default_target, items[]}` |
| `POST /shopping-items` (`201`), `PUT`, `PATCH …/status` | `ShoppingItem` |
| `POST …/apply-template` | `ShoppingTemplateResult` `{created, skipped, deleted, warnings[]}` |
| `POST …/link-to-budget` | `LinkToBudgetResult` `{line, created, linked}` |
| `POST …/unlink-from-budget` | `UnlinkFromBudgetResult` `{unlinked}` |
| `DELETE` (vendor, barang) | `204` |
| `GET …/export-*` | berkas (bukan JSON): `Content-Disposition: attachment; filename="vendor-<slug>-<tanggal>.xlsx"` / `belanja-…` ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)) |

Kunci opsional **dihilangkan** bila kosong (`omitempty`, tipe FE `?:`): `pic_name`, `phone_number`, `formatted_wa_number`, `portfolio_url`, `limit_group`, `price_delta`, `notes`, `payment`, `delta` (kelompok), `line`, `budget_item_id` (pada `SharedItem`, `LockDealResult`, `RollupTarget`, `ShoppingItem`), `shared_item`, `unit`, `vendor_id`, `vendor_name`, `purpose`, `template_key`, `purchase_location`, `souvenir_stock`. Tidak ada `null` yang dikirim di menu ini. Semua daftar (`vendors`, `categories`, `limit_groups`, `vendor_ids`, `items`, `rollups`, `warnings`) selalu array, bukan `null`. `remaining` pada `LinePayment` boleh negatif (lebih bayar); `delta` kelompok dan `price_delta` boleh negatif (melebihi pagu atau lebih mahal). **Tidak ada persentase di respons mana pun**; klien menghitung dengan `percent` ([FE bab 5](FE.md#5-library-libplannervendorts)), selaras kontrak bab 6 (pembagi nol).

### 3.4 Validasi dan batas

Semua galat berbentuk `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)); pesan memakai kosakata Menu 01–03 ditambah alasan di bawah. Angka batas adalah usulan ([bab 9 butir 1](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)) dan dijaga usecase. Teks dipangkas (`trim`); panjang dihitung dalam rune; tanpa karakter kontrol (`notes` dan `purchase_location` boleh baris baru: `cleanNote` Menu 02).

| Field | Aturan | Galat |
|---|---|---|
| `sub_event_scope` | ada di master sesi (termasuk `all`); kosong = `all`; diperiksa sebelum menulis (FK-nya ditunda) | `400 invalid_scope` |
| `category_key` (vendor) | kosong atau kunci registry jenis acara proyek | `400 invalid_input: category_key: must be a registry category key` |
| `category_label` (vendor) | 1–60; wajib bila `category_key` kosong; diabaikan bila kunci registry | `required when category_key is empty` |
| `vendor_name` | 1–255 | `400 invalid_input` |
| `pic_name` | ≤ 255 | `400 invalid_input` |
| `phone_number` | ≤ 40; harus lolos `NormalizeWA`; kosong diizinkan | `400 invalid_phone` (`phone_number: must be an Indonesian phone number (08…, +62…)`) |
| `portfolio_url` | `http`/`https`, ≤ 500 | `portfolio_url: must be an http or https URL` |
| `budget_limit`, `agreed_price` | bilangan bulat 0…10¹³ | `must be between 0 and 10000000000000` |
| `status` (vendor) | `rencana` (bila kosong), `negosiasi`, `fix`, `batal`; `fix` ⇒ `agreed_price > 0` | `must be one of rencana, negosiasi, fix, batal`; `agreed_price: must be greater than 0 when status is fix` |
| `agreed_price` (`lock-deal`) | bilangan bulat 1…10¹³ | `must be between 1 and 10000000000000` |
| `notes` | ≤ 2.000 | `400 invalid_input` |
| `category` (belanja) | 1–60 | `400 invalid_input` |
| `item_name` | 1–255 | `400 invalid_input` |
| `quantity` | 0,01…1.000.000, dua desimal; kosong = 1 | `must be between 0.01 and 1000000` |
| `unit` | ≤ 30 | `400 invalid_input` |
| `estimated_price`, `actual_price` | bilangan bulat 0…10¹³ | `must be between 0 and 10000000000000` |
| `quantity × harga` | `round()` ≤ 10¹³ (aritmetika bulat, bab 2.9) untuk harga estimasi **dan** riil | `estimated_price: the total (quantity × price) must be at most 10000000000000` (idem `actual_price`) |
| `purchase_location` | ≤ 500; teks bebas (tidak diperiksa skema; FE yang memutuskan tautan) | `400 invalid_input` |
| `vendor_id` (belanja) | vendor proyek ini | `must be a vendor of this project` |
| `budget_item_id` (`POST /shopping-items`) | pos proyek ini yang sudah `rollup_source = 'shopping'` | `must be a shopping rollup line of this project` |
| `purpose` | kosong atau `souvenir` | `must be one of souvenir` |
| `status` (belanja) | `belum` (bila kosong), `proses`, `selesai` | `must be one of belum, proses, selesai` |
| `mode` (`apply-template`) | `append` (bawaan bila kosong) atau `replace_template` | `must be one of append, replace_template`; `400 invalid_template` bila jenis acara tak punya template belanja (`shopping_list: not available for this event type`) |
| `confirm` | bool; wajib `true` pada `replace_template` yang menghapus barang bekerja (bab 2.6) dan pada `link-to-budget` ke pos berpembayaran (bab 2.4) | `400 invalid_input: confirm: required` (template); `409 rollup_conflict` (gulungan) |
| `budget_item_id` (`link`/`unlink`) | pos proyek ini; untuk `link`: bukan berharga satuan dan tidak bertaut vendor/dokumen | `must be a budget line of this project` / `must not be a unit-priced line` / `must not be linked to a vendor or document`; `400 invalid_json` bila bukan UUID (dekoder; id pos yang benar bentuknya tetapi bukan milik proyek ini: `invalid_input`) |
| `item_ids` | barang proyek ini | `must be items of this project` |
| Query `scope`, `status`, `category` (daftar vendor), `scope`, `status` (daftar belanja) | `scope` dan `category` hanya diperiksa bentuknya (kunci huruf kecil/angka/`_`); kunci yang berbentuk benar tetapi tak dikenal memberi daftar kosong; `status` = enum di atas; kosong = tanpa filter (`scope=all` = hanya sesi Umum) | `400 invalid_input`: `scope: must be a session scope_key`, `status: must be one of …`, `category: must be a category key` |
| `format` (ekspor) | `xlsx` | `400 invalid_input: format: must be xlsx` |
| Jumlah | ≤ 200 vendor; ≤ 1.000 barang per proyek. Diperiksa di bawah kunci proyek, semua-atau-tidak | `409 vendor_limit_reached`, `409 shopping_item_limit_reached` |
| Frekuensi | ekspor 10/menit/pengguna ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)) | `429 rate_limited` |

Kode galat baru (ditambahkan ke tabel kontrak bab 6): `vendor_limit_reached`, `shopping_item_limit_reached`. Kode lain sudah ada: `invalid_phone`, `invalid_scope`, `invalid_template`, `rollup_conflict`, `conflict`, `project_read_only`. Warning baru (notice, kontrak bab 6): `unit_pricing`, `budget_link_released`; `shared_budget_item`, `estimate_differs`, dan `budget_link_lost` sudah terdaftar.

Alasan baku (Inggris, setelah `<field>: `) yang dipetakan FE ([FE bab 9](FE.md#9-pesan-galat-dan-teks)): kosakata Menu 01 (`required`, `must not be empty`, `must be at most N characters`, `must not contain control characters`, `must be an integer`, `must be between A and B`, `must be at least N`) dan Menu 03 (`must be a vendor of this project`, `must be an http or https URL`, `must be greater than 0`, `must be one of …`) ditambah yang di tabel: `must be a registry category key`, `required when category_key is empty`, `must be greater than 0 when status is fix`, `the total (quantity × price) must be at most N`, `must be a shopping rollup line of this project`, `must be items of this project`, `must be a budget line of this project`, `must not be a unit-priced line`, `must not be linked to a vendor or document`, `must be an Indonesian phone number (08…, +62…)`.

Urutan pemeriksaan: id anak yang bukan UUID kanonik atau milik proyek lain → `404` lebih dulu; kemudian bentuk dan isi body (berurutan menurut field di tabel); batas jumlah `409` diperiksa di dalam transaksi berkunci, sesudah validasi bentuk. `PATCH …/status` hanya memvalidasi `status`; `PUT` memvalidasi seluruh body.

---

## 4. DTO (Go)

Blok di bawah **sudah dikompilasi** (`go vet`, `gofmt`, dengan stub `Notice` yang sama dengan `planner/domain`) dan dibandingkan kunci demi kunci dengan tipe TypeScript [FE bab 4.2](FE.md#42-tambahan-pada-srclibplannertypests): **26 pasangan tipe, 378 jalur JSON, nol selisih** pada kunci, opsionalitas (`omitempty` ↔ `?:`), `null`, dan jenis nilai untuk tipe respons (tipe permintaan: kunci dan jenis saja). Pembanding diuji dengan memutasi tipe TS (enam mutasi, semuanya dilaporkan). Versi awal blok ini memakai tag `binding`, struct anonim tanpa tag JSON (kunci Go bawaan), `Quantity float64` wajib, dan KPI `TotalLimitFix` yang tidak terdefinisi; semuanya diganti.

```go
// backend/internal/modules/plannervendor/domain/entity.go
package domain

import (
	"github.com/google/uuid"

	pdomain "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Konvensi (sama dengan Menu 01–03): tag json snake_case; kunci opsional berupa pointer + omitempty (dihilangkan bila
// kosong, FE: `?:`); daftar selalu array (tidak pernah null); TANPA tag `binding` (validasi di usecase, pesan
// "<field>: <alasan>"); uang int64 rupiah utuh; tidak ada persentase yang dikirim (klien memakai percent/sharePercents).

// ----- Tab 1: vendor -----

type VendorStatus string

const (
	VendorPlanned     VendorStatus = "rencana"
	VendorNegotiating VendorStatus = "negosiasi"
	VendorFixed       VendorStatus = "fix"
	VendorCancelled   VendorStatus = "batal"
)

// Sumber pagu yang berlaku untuk satu vendor (bab 2.1).
const (
	LimitSourceManual   = "manual"   // planner_vendors.budget_limit > 0
	LimitSourceScenario = "scenario" // satu pos skenario aktif, tidak dibagi dengan kategori lain
	LimitSourceGroup    = "group"    // pos skenario yang dibagi beberapa kategori: dibandingkan di LimitGroup
	LimitSourceNone     = "none"     // tidak ada pagu (kategori tanpa pos, atau skenario aktif tidak memuatnya)
)

// LinePayment: angka efektif satu pos anggaran (view planner_budget_lines), dibaca dari pos yang bertaut ke vendor
// (planner_budget_items.vendor_id) atau, untuk LimitGroup.Line, dari pos bersama kelompok.
type LinePayment struct {
	BudgetItemID  uuid.UUID `json:"budget_item_id"`
	ItemName      string    `json:"item_name"`
	Estimate      int64     `json:"estimate"` // efektif (dari belanja bila pos rollup)
	Paid          int64     `json:"paid"`
	Remaining     int64     `json:"remaining"`      // < 0 = lebih bayar
	PaymentStatus string    `json:"payment_status"` // unpaid | dp_paid | partial | settled
}

type Vendor struct {
	ID                uuid.UUID    `json:"id"`
	SubEventScope     string       `json:"sub_event_scope"`
	CategoryKey       string       `json:"category_key"`
	CategoryLabel     string       `json:"category_label"`
	VendorName        string       `json:"vendor_name"`
	PICName           *string      `json:"pic_name,omitempty"`
	PhoneNumber       *string      `json:"phone_number,omitempty"`
	FormattedWANumber *string      `json:"formatted_wa_number,omitempty"` // 62…; tautan wa.me dibentuk klien (02 BE §4.3)
	PortfolioURL      *string      `json:"portfolio_url,omitempty"`
	BudgetLimit       int64        `json:"budget_limit"`          // pagu efektif; 0 bila LimitSource group atau none
	LimitSource       string       `json:"limit_source"`          // manual | scenario | group | none
	LimitGroup        *string      `json:"limit_group,omitempty"` // item_key kelompok bila LimitSource group
	AgreedPrice       int64        `json:"agreed_price"`          // 0 = belum ada penawaran
	PriceDelta        *int64       `json:"price_delta,omitempty"` // BudgetLimit − AgreedPrice (positif = hemat); hilang bila salah satunya 0 atau LimitSource group/none
	Status            VendorStatus `json:"status"`
	Notes             *string      `json:"notes,omitempty"`
	SortOrder         int          `json:"sort_order"`
	Payment           *LinePayment `json:"payment,omitempty"` // pos yang menunjuk vendor ini; hilang bila belum bertaut
}

// LimitGroup: perbandingan pagu satu pos skenario yang dipakai ≥ 2 kategori vendor (pernikahan: venue_dekor, rias_busana, hiburan).
type LimitGroup struct {
	ItemKey       string       `json:"item_key"`
	SubEventScope string       `json:"sub_event_scope"` // sesi pos skenario (bisa 'all')
	Limit         int64        `json:"limit"`
	TotalAgreed   int64        `json:"total_agreed"`    // Σ AgreedPrice anggota yang bukan batal (penawaran dan deal)
	Delta         *int64       `json:"delta,omitempty"` // Limit − TotalAgreed; hilang bila belum ada anggota yang punya penawaran
	Members       int          `json:"members"`
	Priced        int          `json:"priced"` // anggota dengan AgreedPrice > 0
	VendorIDs     []uuid.UUID  `json:"vendor_ids"`
	Line          *LinePayment `json:"line,omitempty"` // pos anggaran bersama kelompok bila ada
}

type CategoryState struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	State   string `json:"state"`   // fix | pipeline | empty
	Vendors int    `json:"vendors"` // yang bukan batal
	Custom  bool   `json:"custom"`  // bukan dari registry jenis acara
	Skipped bool   `json:"skipped"` // ada di planner_projects.skipped_vendor_categories
}

type VendorKPI struct {
	CategoriesTotal int   `json:"categories_total"` // kategori registry ∪ kustom yang dipakai, tanpa yang dilewati
	CategoriesFixed int   `json:"categories_fixed"` // di antaranya yang punya ≥ 1 vendor fix
	Skipped         int   `json:"skipped"`
	FixedVendors    int   `json:"fixed_vendors"`
	Pipeline        int   `json:"pipeline"` // rencana + negosiasi
	FixedTotal      int64 `json:"fixed_total"`
	PipelineOffers  int64 `json:"pipeline_offers"` // Σ AgreedPrice (> 0) vendor rencana dan negosiasi
	GroupsTotal     int   `json:"groups_total"`
	GroupsOver      int   `json:"groups_over"` // grup dengan Delta < 0
}

type VendorOverview struct {
	KPI         VendorKPI       `json:"kpi"`          // selalu seluruh proyek
	Categories  []CategoryState `json:"categories"`   // idem
	LimitGroups []LimitGroup    `json:"limit_groups"` // idem
	Vendors     []Vendor        `json:"vendors"`      // disaring ?scope=&status=&category=
}

// VendorRequest: POST dan PUT (PUT mengganti penuh). CategoryKey registry, atau kosong untuk kategori kustom (kunci diturunkan
// dari CategoryLabel oleh CategoryKeyFromLabel).
type VendorRequest struct {
	SubEventScope string       `json:"sub_event_scope"`
	CategoryKey   *string      `json:"category_key"`
	CategoryLabel *string      `json:"category_label"`
	VendorName    string       `json:"vendor_name"`
	PICName       *string      `json:"pic_name"`
	PhoneNumber   *string      `json:"phone_number"`
	PortfolioURL  *string      `json:"portfolio_url"`
	BudgetLimit   int64        `json:"budget_limit"` // 0 = ikut skenario aktif
	AgreedPrice   int64        `json:"agreed_price"`
	Status        VendorStatus `json:"status"`
	Notes         *string      `json:"notes"`
}

type VendorStatusRequest struct {
	Status VendorStatus `json:"status"`
}

// VendorResult: POST, PUT, dan PATCH …/status. Warnings: estimate_differs, budget_link_released.
type VendorResult struct {
	Vendor   Vendor           `json:"vendor"`
	Warnings []pdomain.Notice `json:"warnings"`
}

type LockDealRequest struct {
	AgreedPrice       int64 `json:"agreed_price"`
	SyncToBudgetItems bool  `json:"sync_to_budget_items"`
}

// SharedItem: pos yang dipakai bersama beberapa kategori; tidak ditautkan dan estimasinya tidak diubah (bab 2.2).
type SharedItem struct {
	ItemKey      string     `json:"item_key"`
	ItemName     string     `json:"item_name"`                // nama pos, atau nama pos skenario bila pos belum ada
	BudgetItemID *uuid.UUID `json:"budget_item_id,omitempty"` // hilang bila pos belum ada
	Plan         int64      `json:"plan"`                     // estimasi efektif pos, atau pagu skenario bila pos belum ada
	TotalAgreed  int64      `json:"total_agreed"`
}

// LockDealResult: Warnings: shared_budget_item, unit_pricing.
type LockDealResult struct {
	Vendor       Vendor           `json:"vendor"`
	BudgetItemID *uuid.UUID       `json:"budget_item_id,omitempty"` // pos yang ditautkan atau dibuat
	Created      bool             `json:"created"`                  // pos baru dibuat
	SharedItem   *SharedItem      `json:"shared_item,omitempty"`
	Warnings     []pdomain.Notice `json:"warnings"`
}

// ----- Tab 2: belanja -----

const (
	ShoppingNotStarted = "belum"
	ShoppingInProgress = "proses"
	ShoppingDone       = "selesai"
)

type ShoppingItem struct {
	ID               uuid.UUID  `json:"id"`
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         float64    `json:"quantity"` // dua desimal
	Unit             *string    `json:"unit,omitempty"`
	EstimatedPrice   int64      `json:"estimated_price"` // per satuan
	ActualPrice      int64      `json:"actual_price"`    // per satuan; 0 = belum dibeli
	EstimatedTotal   int64      `json:"estimated_total"` // round(Quantity × EstimatedPrice), dihitung basis data
	ActualTotal      int64      `json:"actual_total"`
	PriceDelta       *int64     `json:"price_delta,omitempty"` // EstimatedTotal − ActualTotal (positif = hemat); hilang bila ActualPrice 0
	PurchaseLocation *string    `json:"purchase_location,omitempty"`
	VendorID         *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName       *string    `json:"vendor_name,omitempty"`
	BudgetItemID     *uuid.UUID `json:"budget_item_id,omitempty"` // pos rollup tujuan; hilang bila tidak digulung
	Purpose          *string    `json:"purpose,omitempty"`        // souvenir
	Status           string     `json:"status"`                   // belum | proses | selesai
	SortOrder        int        `json:"sort_order"`
	TemplateKey      *string    `json:"template_key,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

type ShoppingKPI struct {
	EstimatedTotal int64  `json:"estimated_total"` // Σ EstimatedTotal semua barang
	ActualTotal    int64  `json:"actual_total"`    // Σ ActualTotal barang selesai
	Done           int    `json:"done"`
	InProgress     int    `json:"in_progress"`
	NotStarted     int    `json:"not_started"`
	Total          int    `json:"total"`
	SouvenirStock  *int64 `json:"souvenir_stock,omitempty"` // Σ Quantity (dibulatkan ke bawah) barang souvenir selesai; hilang bila tak ada barang souvenir
}

// RollupLine: pos anggaran yang dihitung dari daftar belanja.
type RollupLine struct {
	BudgetItemID  uuid.UUID `json:"budget_item_id"`
	ItemName      string    `json:"item_name"`
	SubEventScope string    `json:"sub_event_scope"`
	Estimate      int64     `json:"estimate"`
	Paid          int64     `json:"paid"`
	LinkedItems   int       `json:"linked_items"`
}

// RollupTarget: pos bawaan tombol "Hitung pos dari daftar ini" (kunci dari registry jenis acara, bab 2.4).
type RollupTarget struct {
	BudgetItemID  *uuid.UUID `json:"budget_item_id,omitempty"` // hilang bila pos belum ada: akan dibuat
	ItemKey       string     `json:"item_key"`
	ItemName      string     `json:"item_name"`
	SubEventScope string     `json:"sub_event_scope"`
	Rollup        bool       `json:"rollup"`       // sudah dihitung dari belanja
	HasPayments   bool       `json:"has_payments"` // pembayaran tersimpan > 0: mengaktifkan butuh confirm
	Estimate      int64      `json:"estimate"`
	Paid          int64      `json:"paid"`
}

type ShoppingOverview struct {
	KPI           ShoppingKPI    `json:"kpi"` // selalu seluruh proyek
	Rollups       []RollupLine   `json:"rollups"`
	DefaultTarget RollupTarget   `json:"default_target"`
	Items         []ShoppingItem `json:"items"` // disaring ?scope=&category=&status=
}

// ShoppingItemRequest: POST. BudgetItemID hanya di sini (harus pos rollup proyek ini); PUT tidak mengubah tautan.
type ShoppingItemRequest struct {
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         *float64   `json:"quantity"` // kosong = 1
	Unit             *string    `json:"unit"`
	EstimatedPrice   int64      `json:"estimated_price"`
	ActualPrice      int64      `json:"actual_price"`
	PurchaseLocation *string    `json:"purchase_location"`
	VendorID         *uuid.UUID `json:"vendor_id"`
	BudgetItemID     *uuid.UUID `json:"budget_item_id"`
	Purpose          *string    `json:"purpose"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes"`
}

// UpdateShoppingItemRequest: PUT (mengganti penuh, kecuali tautan rollup, template_key, dan urutan).
type UpdateShoppingItemRequest struct {
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         *float64   `json:"quantity"`
	Unit             *string    `json:"unit"`
	EstimatedPrice   int64      `json:"estimated_price"`
	ActualPrice      int64      `json:"actual_price"`
	PurchaseLocation *string    `json:"purchase_location"`
	VendorID         *uuid.UUID `json:"vendor_id"`
	Purpose          *string    `json:"purpose"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes"`
}

type ShoppingStatusRequest struct {
	Status string `json:"status"`
}

type ShoppingTemplateRequest struct {
	Mode    string `json:"mode"` // append | replace_template
	Confirm bool   `json:"confirm"`
}

// ShoppingTemplateResult: Warnings: budget_link_lost.
type ShoppingTemplateResult struct {
	Created  int              `json:"created"`
	Skipped  int              `json:"skipped"`
	Deleted  int              `json:"deleted"`
	Warnings []pdomain.Notice `json:"warnings"`
}

type LinkToBudgetRequest struct {
	BudgetItemID *uuid.UUID  `json:"budget_item_id"` // kosong = pos bawaan registry (RollupTarget)
	ItemIDs      []uuid.UUID `json:"item_ids"`       // kosong = semua barang yang belum bertaut ke pos rollup lain
	Confirm      bool        `json:"confirm"`
}

type LinkToBudgetResult struct {
	Line    RollupLine `json:"line"`
	Created bool       `json:"created"` // pos dibuat karena belum ada
	Linked  int        `json:"linked"`  // barang yang baru ditautkan
}

type UnlinkFromBudgetRequest struct {
	BudgetItemID uuid.UUID `json:"budget_item_id"`
}

type UnlinkFromBudgetResult struct {
	Unlinked int `json:"unlinked"`
}
```

---

## 5. Master Data

### 5.1 Kategori vendor pernikahan (PDF Hal. 12)

PDF memuat sepuluh kategori di "List Rencana Vendor" dan tujuh di "Fix Kerjasama Vendor" (Venue, Decoration, Wedding Organizer, MUA, Catering, Photography, Attire; baris data PDF hanya contoh dummy, mis. nama "Asep", telepon "1514"). Template **tidak membuat baris vendor**; sepuluh kategori ini adalah registry pernikahan dan menjadi checklist "kategori terkunci" (7 dari 10 pada PDF). Berada di kode (`plannervendor` menambahkan `VendorCategories` ke registry, bab 6), bukan basis data:

| `category_key` | Label (PDF) | `budget_item_keys` | Kelompok pagu |
|---|---|---|---|
| `venue` | Venue | `venue_dekor` | `venue_dekor` (dengan `dekorasi`) |
| `dekorasi` | Decoration | `venue_dekor` | `venue_dekor` |
| `wo` | Wedding Organizer | – | – |
| `mua` | MUA | `rias_busana` | `rias_busana` (dengan `attire`) |
| `catering` | Catering | `catering` | – |
| `fotografi` | Photography | `dokumentasi` | – |
| `attire` | Attire | `rias_busana` | `rias_busana` |
| `hand_bouquet` | Hand Bouquet | – | – |
| `mc` | MC | `hiburan` | `hiburan` (dengan `entertainment`) |
| `entertainment` | Entertainment (PDF: "Entertaiment") | `hiburan` | `hiburan` |

Setiap `budget_item_keys` adalah `item_key` pos pernikahan di [03 BE §6.1–6.2](../03_KEUANGAN/BE.md#6-master-data) (diperiksa dengan skrip). Biaya KUA bukan vendor melainkan pos legalitas di Menu 03.

**Kategori belanja pernikahan** (registry `ShoppingCategories`, enam dari PDF Hal. 9–10): Perangkat Alat Solat · Make Up & Skin Care · Peralatan Mandi · Pakaian Dalam · Pakaian Luar · Kebutuhan Lain.

### 5.2 Vendor Proyek Contoh (fiktif)

Dataset hanya untuk Proyek Contoh, ditulis oleh overlay `vendors` (bab 6). Nama, PIC, dan nomor telepon **fiktif** (nomor memakai pola `08120000xxxx`, tidak mengarah ke orang nyata; PIC dikosongkan, bukan diberi nama karangan). Semua `sub_event_scope = resepsi`, `budget_limit = 0` (pagu dibaca dari skenario, bab 2.1). Tujuh vendor `fix` sama dengan tujuh kategori PDF; tiga lainnya masih rencana/negosiasi. Kolom pagu dan selisih adalah hasil bab 2.1 terhadap skenario aktif 75 Jt (dijalankan), bukan data tersimpan.

| Kategori | Nama vendor | Telepon | Penawaran / deal | Status | Pagu (skenario 75 Jt) | Selisih | Tertaut pos |
|---|---|---|--:|:-:|--:|--:|---|
| venue | Sasana Kriya Ballroom | 081200000001 | 12.000.000 | fix | kelompok `venue_dekor` 20.000.000 | kelompok 0 | – (pos bersama) |
| dekorasi | Pelaminan Mega Indah | 081200000002 | 8.000.000 | fix | idem | idem | – (pos bersama) |
| wo | Harmony Wedding Organizer | 081200000003 | 5.000.000 | fix | – | – | – (tak ada pos) |
| mua | Sanggar Rias Ayu | 081200000004 | 3.500.000 | fix | kelompok `rias_busana` 5.000.000 | kelompok 0 | – (pos bersama) |
| catering | Katering Bu Minah (500 pax) | 081200000005 | 25.000.000 | fix | **30.000.000** | **+5.000.000** | `catering` (resepsi) |
| fotografi | Studio Cahaya | 081200000006 | 3.000.000 | fix | 3.000.000 | 0 | `dokumentasi` (resepsi) |
| attire | Butik Kebaya Anggun | 081200000007 | 1.500.000 | fix | idem `rias_busana` | idem | – (pos bersama) |
| hand_bouquet | Fleur Hand Bouquet | 081200000008 | 900.000 | rencana | – | – | – |
| mc | Kang Dedi Master MC | 081200000009 | 2.000.000 | negosiasi | kelompok `hiburan` 3.400.000 | kelompok 0 | – (pos bersama) |
| entertainment | Akustik Nada Kirana | 081200000010 | 1.400.000 | negosiasi | idem | idem | – (pos bersama) |

Kelompok pagu: `venue_dekor` 20.000.000 (Venue 12 + Dekorasi 8 = 20, selisih 0); `rias_busana` 5.000.000 (MUA 3,5 + Attire 1,5 = 5); `hiburan` 3.400.000 (MC 2,0 + Entertainment 1,4 = 3,4). Ketiga pos itu adalah pos bersama sehingga tidak ada vendor yang tertaut untuk kelompok tersebut; hanya `catering` dan `fotografi` (pos khusus `catering` dan `dokumentasi`) yang tertaut. Status bayar kelompok dibaca dari pos bersamanya: Sewa Gedung & Dekorasi estimasi 20.000.000, terbayar 900.000 (`dp_paid`); Rias & Baju Pengantin 5.000.000, 600.000 (`dp_paid`); Hiburan & Pengisi acara 3.400.000, 3.400.000 (`settled`). Kejanggalan data PDF yang dibawa apa adanya: pos Hiburan sudah lunas padahal MC dan Entertainment masih negosiasi (dua lembar PDF yang terpisah; tidak diubah).

Overlay **tidak** menjalankan `lock-deal` (Wedding Organizer akan membuat pos ke-15 senilai 5.000.000 dan total proyek 74.150.000 berubah); ia hanya menulis `vendor_id` pada pos `catering` dan `dokumentasi` tanpa mengubah estimasi (25.000.000 dan 3.000.000 memang sama dengan deal), sehingga total PDF tetap utuh. Satu tugas ditautkan ke vendor: `wedding.visit_venue_pernikahan` ("Visit Venue Pernikahan") → Sasana Kriya Ballroom ([02 BE §4.6](../02_TIMELINE_DAN_TUGAS/BE.md#46-sambungan-ke-menu-01)); metadata proyek contoh sudah memakai nama venue yang sama.

### 5.3 Daftar belanja: 37 item seserahan (PDF Hal. 9–10)

Nama barang, kategori, dan harga persis PDF (dibandingkan secara terprogram dengan pembacaan ulang `pdfminer`: 37 baris sama, urutan sama, 16 selesai dan 21 on proses); estimasi total **Rp 4.578.500**. Template memuat kolom *Harga acuan* dengan `status = belum`, `actual_price = 0`, tanpa tautan, `quantity = 1`, sesi `all`, `template_key = wedding.seserahan_<nama>` (bab 2.6). Kolom status berlaku untuk Proyek Contoh (PDF: Selesai/On Proses; `actual_price = harga acuan` untuk yang selesai): **16 selesai = Rp 2.063.800**, **21 proses = Rp 2.514.700**.

| No | Kategori | Barang (PDF) | Harga acuan (Rp) | Status Proyek Contoh |
|:-:|---|---|--:|:-:|
| 1 | Perangkat Alat Solat | Mukena | 194.000 | selesai |
| 2 | Perangkat Alat Solat | Al-Quran | 70.500 | proses |
| 3 | Perangkat Alat Solat | Sajadah Couple | 233.000 | selesai |
| 4 | Perangkat Alat Solat | Tasbih Digital | 99.800 | proses |
| 5 | Make Up & Skin Care | Cushion | 175.700 | selesai |
| 6 | Make Up & Skin Care | Cream Blush | 29.000 | proses |
| 7 | Make Up & Skin Care | Lipmatte | 79.000 | selesai |
| 8 | Make Up & Skin Care | Liptint | 75.400 | proses |
| 9 | Make Up & Skin Care | Loose Powder | 73.200 | selesai |
| 10 | Make Up & Skin Care | Mascara | 66.700 | proses |
| 11 | Make Up & Skin Care | Eyebrow | 61.300 | selesai |
| 12 | Make Up & Skin Care | Eyeliner | 34.900 | proses |
| 13 | Make Up & Skin Care | Micellar Water | 44.500 | selesai |
| 14 | Make Up & Skin Care | Facial Wash | 53.800 | proses |
| 15 | Make Up & Skin Care | Moisturizer | 201.600 | selesai |
| 16 | Make Up & Skin Care | Sunscreen | 162.800 | proses |
| 17 | Make Up & Skin Care | Serum | 95.400 | proses |
| 18 | Make Up & Skin Care | Night Cream | 115.600 | selesai |
| 19 | Peralatan Mandi | Sabun Mandi | 195.000 | proses |
| 20 | Peralatan Mandi | Shampo | 195.000 | selesai |
| 21 | Peralatan Mandi | Sikat Gigi | 67.000 | proses |
| 22 | Peralatan Mandi | Body Lotion | 40.700 | selesai |
| 23 | Peralatan Mandi | Body Scrub | 54.100 | proses |
| 24 | Peralatan Mandi | Body Serum | 65.300 | selesai |
| 25 | Peralatan Mandi | Sisir (Wet Brush) | 77.600 | proses |
| 26 | Peralatan Mandi | Handuk Couple | 296.000 | selesai |
| 27 | Pakaian Dalam | Bra | 152.500 | proses |
| 28 | Pakaian Dalam | Celana Dalam | 72.600 | selesai |
| 29 | Pakaian Dalam | Lingerie | 133.000 | proses |
| 30 | Pakaian Dalam | Baju Tidur | 159.300 | selesai |
| 31 | Pakaian Luar | Gamis/Baju Couple | 388.800 | proses |
| 32 | Pakaian Luar | Hijab/Khimar | 57.000 | selesai |
| 33 | Pakaian Luar | Sepatu/Flat Shoes | 116.900 | proses |
| 34 | Pakaian Luar | Dompet/Tas Kecil (Sling Bag) | 149.300 | proses |
| 35 | Pakaian Luar | Setelan Kebaya Muslimah | 108.200 | proses |
| 36 | Kebutuhan Lain | Seprai Bed Cover | 346.100 | proses |
| 37 | Kebutuhan Lain | Box Seserahan (10 Kotak) | 37.900 | proses |

Subtotal kategori: Perangkat Alat Solat 4 item **597.300** · Make Up & Skin Care 14 item **1.268.900** · Peralatan Mandi 8 item **990.700** · Pakaian Dalam 4 item **517.400** · Pakaian Luar 5 item **820.200** · Kebutuhan Lain 2 item **384.000** · total **4.578.500**.

PDF juga memuat kolom "Link Pembelian" berisi tautan pribadi pembuat dokumen; tautan itu **tidak** disalin ke template maupun Proyek Contoh (`purchase_location` kosong). Proyek Contoh tidak punya barang souvenir dan tidak mengaktifkan gulungan.

### 5.4 Jenis lain
Kategori vendor, kategori belanja, dan agenda non-pernikahan: [TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md); pemetaan kategori vendor ke pos: [bab 15.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#152-pemetaan-kategori-vendor--pos-budget_item_keys); pos gulungan belanja per jenis: [bab 15.4](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#154-pos-gulungan-belanja).

---

## 6. Sambungan ke Menu 01

`plannervendor.Service` memenuhi empat peran dari `planner/domain` (`repository.go`, kode yang sudah ada). Satu modul memiliki dua area (vendor dan belanja), tetapi tiap antarmuka punya satu `Key()`, jadi masing-masing diwakili tipe kecil yang membungkus `Service` (seperti Menu 02):

| Antarmuka | Kunci | Isi |
|---|---|---|
| `SummaryProvider` | `vendors` | `domain.VendorSummary` (`planner/domain/summary.go`, **sudah ada**): `categories_total, categories_fixed, skipped, fixed_vendors, pipeline`; dihitung oleh bagian KPI `BuildOverview` (kueri 01 BE §5.2 di bab 2.8) |
| `TemplateApplier` | `shopping` | `Apply(ctx, projectID, ApplyRequest)` memakai `req.Project` (`EventType`); `req.Mode` selalu `append` dari wizard; menghidupkan `templates_available.shopping_list` (hanya pernikahan memilikinya: `TemplateSet.ShoppingList`); membuat 37 barang |
| `SampleOverlay` | `vendors` | 10 vendor, `vendor_id` pada pos `catering` dan `dokumentasi`, dan tugas `wedding.visit_venue_pernikahan` (bab 5.2) |
| `SampleOverlay` | `shopping` | status 37 barang dan harga riil (bab 5.3) |

Pemasangan di `internal/app/app.go` (menambah ke baris Menu 02 dan 03, registry dan `TxManager` adalah instance yang sama):

```go
vendorSvc := vendorusecase.New(vendorrepo.NewPostgresRepository(pool), plannerTx, registry, vendorusecase.Options{})
plannerSvc.RegisterSummary(timelineSvc.TimelineProvider(), timelineSvc.AdminDocsProvider(), financeSvc.Provider(), vendorSvc.Provider())
plannerSvc.RegisterTemplateAppliers(timelineSvc.TimelineApplier(), timelineSvc.AdminDocsApplier(), financeSvc.Applier(), vendorSvc.Applier())
plannerSvc.RegisterSampleOverlays(timelineSvc.TimelineOverlay(), timelineSvc.AdminDocsOverlay(), financeSvc.Overlay(), vendorSvc.VendorsOverlay(), vendorSvc.ShoppingOverlay())
vendorhttp.RegisterRoutes(ginEngine, idn, plannerSvc, vendorSvc)
```

Urutan overlay penting (dijalankan dalam transaksi pembuatan Proyek Contoh sesudah semua template): `timeline → admin_docs → finance → vendors → shopping`. Overlay `finance` mengisi estimasi dan pembayaran 14 pos yang akan ditautkan overlay `vendors` ([03 BE §3.12](../03_KEUANGAN/BE.md#312-sambungan-ke-menu-01)); overlay `timeline` memberi tugas `wedding.visit_venue_pernikahan` ([02 BE §4.6](../02_TIMELINE_DAN_TUGAS/BE.md#46-sambungan-ke-menu-01)). Template berjalan `timeline → admin_docs → budget → shopping → …`: pos anggaran harus ada sebelum barang yang digulung ke dalamnya, dan wizard tidak membuat vendor. Kunci terdaftar dua kali atau tak dikenal membuat start gagal (panic), sengaja. `applier` mengembalikan `{created, skipped, warnings: []}`; `append` idempoten.

**Registry.** Menu ini menambah ke `EventTypeSpec` (nilai: [TEMPLATE_EVENT bab 7–15](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md), sembilan jenis):

```go
type VendorCategory struct {
	Key, Label     string
	BudgetItemKeys []string `json:"-"` // pos skenario/anggaran yang menampungnya (TEMPLATE_EVENT 15.2); kosong = tanpa pos
}

// ShoppingRollup: pos anggaran bawaan gulungan belanja (TEMPLATE_EVENT 15.4). Pos dicari menurut Key pada sesi mana pun;
// bila tidak ada, dibuat dengan Name dan Category (hanya `other`, yang tak punya pos 15.1 yang cocok).
type ShoppingRollup struct{ Key, Name, Category string }

// pada EventTypeSpec:
VendorCategories   []VendorCategory `json:"vendor_categories"`   // dikirim {key, label}
ShoppingCategories []string         `json:"shopping_categories"` // teks kategori belanja; disimpan apa adanya di planner_shopping_items.category
ShoppingRollup     ShoppingRollup   `json:"-"`
```

`GET /event-types` mengirim `vendor_categories: [{key, label}]` dan `shopping_categories: [...]` (FE bab 4.2, `EventTypeVendorInfo`). `ShoppingCategories` adalah `[]string` (bukan `[]Option` seperti tertulis di kontrak bab 3): kategori belanja disimpan sebagai teks, tidak punya kunci. Tes keselarasan `TestVendorCategoriesMatchTemplateEvent` (bab 7.2) membaca TEMPLATE_EVENT: kunci kategori = bab 7–14, `budget_item_keys` ∈ 15.1 jenis yang sama, kunci gulungan ∈ 15.1 (kecuali `other`).

**Pengecualian SQL.** Dicatat di kontrak bab 1.1 sebagai pengecualian keempat: `lock-deal` dan gulungan menulis `planner_budget_items`. Pembacaan lintas modul yang sudah ada dan kini punya pemilik: Menu 02 (nama dan WA vendor pada tugas), Menu 03 (`vendor_name` pada pos), dan Menu 06 (kontak PIC pada agenda) `JOIN planner_vendors`; Menu 05 membaca `planner_shopping_items` untuk stok souvenir. Semuanya baca-saja.

---

## 7. Pengujian

### 7.1 Yang sudah dijalankan pada audit spesifikasi

Dijalankan 2026-10-22 pada PostgreSQL 16.2 lokal, database sementara `planner_spec04_scratch_test` (dibuat khusus dengan migrasi 001–012 dari berkas nyata lewat `go run ./cmd/migrate up`, dihapus setelahnya; database `adicara` tidak disentuh), koneksi `psql`, dan **backend Menu 01–03 yang sebenarnya** pada port sementara (pengguna uji, `POST /projects/sample`, sembilan proyek dengan template anggaran dan skenario). Skrip dan modul Go/TS ada di scratchpad sesi, bukan di repo.

| Butir | Hasil |
|---|---|
| Blok SQL bab 1 identik dengan migrasi 006 dan 009; keduanya naik di atas 001–012 | ✔ perbandingan teks; 14 tabel + 1 view `planner_*` |
| PDF Hal. 9, 10, 12 dibaca ulang (`pdfminer`): 37 nama/kategori/harga/status, 6 subtotal, 16 selesai dan 21 proses, 10 kategori vendor dengan 7 FIX | ✔ sama dengan bab 5 (dibandingkan secara terprogram; nol selisih) |
| Constraint vendor (kunci kategori, nama kosong, status, sesi tak dikenal baru saat `COMMIT`, vendor lintas proyek) dan belanja (`quantity 0`, `purpose`, vendor lintas proyek); kolom generated | ✔ ditolak/terhitung seperti dirancang (bab 1) |
| `round(9999999999.99 × 10¹³)::bigint` pada belanja | ✔ `bigint out of range` (bab 1.2) |
| `LineTotal` (Go) dan `lineTotalX100` (TS) terhadap 300 pasangan yang dihitung PostgreSQL, termasuk tepi 10¹³ dan 10¹³+1 | ✔ nol selisih ([fixtures/planner-line-totals.json](fixtures/planner-line-totals.json)) |
| Pagu (bab 2.1) pada Proyek Contoh: skenario aktif 10 pos; Catering 30.000.000 (bukan 25.000.000 di wireframe lama), Photography 3.000.000, tiga kelompok pas | ✔ SQL dan `BuildOverview` |
| Jenis non-pernikahan: delapan proyek dibuat lewat HTTP dengan template anggaran dan preset `<jenis>_pct` diterapkan; seluruh pos skenario dan template berada di `all`; vendor di sesi nyata tanpa fallback: 0 pagu; `lock-deal` aturan lama menambah pos kedua `venue` di sesi `registrasi` seminar (total estimasi 120.000.000 dari pagu 100.000.000) | ✔ bab 2.1–2.2 |
| TEMPLATE_EVENT 15.2 sama dengan daftar kategori bab 7–14 untuk delapan jenis; kelompok pagu per jenis; kunci gulungan bab 15.4 ada di 15.1 (kecuali `other`) | ✔ skrip, nol selisih |
| `lock-deal`: pos khusus (catering, fotografi) tertaut; pos bersama (venue) tidak; pos baru (WO) menambah 5.000.000; nama vendor 255 karakter → `23514` pada aturan lama, 150 karakter pada aturan baru; pos berharga satuan: estimasi 25.000.000 ≠ 22.500.000 pada aturan lama | ✔ |
| Balapan `lock-deal` dua vendor sekategori: tanpa kunci proyek keduanya "ditautkan", yang pertama tertimpa; dengan `FOR UPDATE` pada `planner_projects`: satu tertaut, satu pos baru | ✔ dua sesi `psql` |
| Blok SQL bab 2.1–2.8 (`scenario_items`, `vendor_read`, `lock_deal` + `link`/`reprice`/`create`, `unlock`, `rollup`, `unlink`, `shopping_kpi`, `status_patch`, `template_append`, `template_impact`, `vendor_provider`) dijalankan **apa adanya** (hanya `$n` diganti nilai) dalam transaksi yang di-rollback | ✔ tanpa galat |
| Gulungan: 37 barang → pos `seserahan` estimasi 4.578.500, terbayar 2.063.800, sisa 2.514.700, `partial`; total proyek 75.228.500 / 32.113.800; barang baru tanpa tautan tidak ikut; pencabutan aturan lama meninggalkan 37 barang yang masih menunjuk pos | ✔ |
| KPI belanja 4.578.500 / 2.063.800 / 16 / 21 / 0 / 37; subtotal per kategori; stok souvenir `floor(300 + 50,5)` = 350 dan terencana 370,5; `selesai` mengisi harga riil hanya bila 0 | ✔ |
| Template: `append` dengan `ON CONFLICT … DO NOTHING` → `created 3, skipped 1`; dampak `replace_template` pada Proyek Contoh `removed 37, carry_work 37`; 37 `template_key` unik (maks 44 karakter) | ✔ |
| `DELETE` vendor: tugas, pos, dan barang yang menunjuknya tetap ada dengan `vendor_id` kosong | ✔ |
| Kueri provider `vendors` 01 BE §5.2 pada data emas | ✔ 7 / 3; 10 kategori terpakai, 7 fix |
| Blok DTO bab 4: `gofmt`, `go vet`; kunci sama dengan tipe TS FE (26 pasangan, 378 jalur, nol selisih); pembanding diuji dengan enam mutasi | ✔ |
| `NormalizeWA` terhadap 26 vektor bersama (Go dan TS); `CategoryKeyFromLabel`; `BuildOverview` (3 tes terhadap data Proyek Contoh yang diekspor + kasus tepi); mutasi: delapan dari sembilan menjatuhkan tes (satu setara) | ✔ |
| Seluruh tautan relatif di folder PLANNER_SPEC (termasuk ke bab-bab dokumen ini) | ✔ nol rusak |

**Tidak dijalankan pada audit:** kode modul `plannervendor` (belum ada), jalur HTTP dan Gin, `go test` modul ini, ekspor `.xlsx` (belum ada prototipe), balapan pada `link-to-budget`, `apply-template`, dan batas jumlah (kunci proyek yang sama; terbukti hanya untuk `lock-deal` dan, pada Menu 02, untuk batas jumlah), latensi kueri pada data 10× contoh, PostgreSQL 18, Docker, Nginx.

### 7.2 Pengujian pada implementasi

Nama tes di kode (semua ✔, dijalankan 2026-10-23; hasilnya di [bab 7.3](#73-hasil-implementasi-2026-10-23)). Tes DB butuh `TEST_DATABASE_URL` ke database `…_test`, dijalankan `go test -p 1`. Berkas ada di `backend/internal/modules/plannervendor/{domain,usecase,delivery/http}` dan `planner/domain` (`NormalizeWA`).

| Butir | Tes |
|---|---|
| Data template sama dengan tabel spesifikasi: 37 barang, kategori, harga, `template_key` unik; 10 vendor contoh; kategori vendor semua jenis | ✔ `TestWeddingShoppingMatchesSpec`, `TestSampleVendorsMatchSpec`, `TestVendorCategoriesMatchTemplateEvent` (membaca BE bab 5 dan TEMPLATE_EVENT 7–15), `TestEveryEventTypeHasItsSharedGroups` |
| `NormalizeWA` terhadap vektor bersama; `CategoryKeyFromLabel`; `LineTotal` terhadap vektor PostgreSQL; kuantitas dan KPI belanja | ✔ `TestNormalizeWAMatchesTheSharedVectors`, `TestNormalizeWAEdges`, `TestCategoryKeyFromLabel`, `TestLineTotalMatchesPostgres`, `TestLineTotalEdges`, `TestQuantityX100`, `TestShoppingKPIOf` |
| `BuildOverview`: Proyek Contoh, batal, pagu manual, anggota tanpa penawaran, kelompok melebihi pagu, ganti skenario, tanpa skenario, kategori kustom/dilewati, fallback sesi Umum, `SharedItemOf` | ✔ `TestSampleOverview`, `TestOverviewEdgeCases`, `TestNonWeddingScopeFallback`, `TestSharedItemOf`, `TestSharedKeysPerEventType` |
| Validasi murni usecase: kategori, status `fix`, kuantitas dan total, nomor, URL, filter | ✔ `TestResolveCategory`, `TestVendorStatusRules`, `TestShoppingWriteDefaultsAndLimits`, `TestCleanPhone`, `TestCleanURLAndText`, `TestVendorFilter` |
| Vendor: tambah/ubah/hapus, saring, `PATCH …/status`, setiap baris validasi bab 3.4, kategori kustom dan registry, batas 200 (termasuk balapan), peringatan `estimate_differs`/`budget_link_released`, hapus vendor | ✔ `TestVendorLifecycle`, `TestVendorValidation`, `TestVendorCategoriesAreRegistryOrCustom`, `TestVendorLimitOf200`, `TestVendorUpdateWarnings`, `TestDeleteVendorKeepsReferrers` |
| `lock-deal`: pos khusus, pos baru (nama 255 karakter, vendor kedua sekategori, kategori kustom), pos bersama, pos berharga satuan, kunci ulang, sesi nyata tanpa pos (fallback `all`), tanpa `sync`, validasi, batas 300 pos, setelah meninggalkan `fix` | ✔ `TestLockDealLinksTheDedicatedLine`, `TestLockDealMakesANewLine`, `TestLockDealLeavesASharedLineAlone`, `TestLockDealKeepsTheUnitPricedEstimate`, `TestLockDealFallsBackToTheGeneralSession`, `TestLockDealValidationAndTheLineLimit`, `TestLockDealAfterLeavingFix` |
| Balapan: delapan `lock-deal` serentak untuk vendor sekategori; tanpa pos khusus (kunci `item_key`); pada batas 300 pos | ✔ `TestConcurrentLockDealNeverLosesALink`, `TestConcurrentLockDealWithoutADedicatedLine`, `TestConcurrentLockDealRespectsTheLineLimit` |
| Belanja: tambah/ubah/hapus, saring, tepi `quantity`/total (10¹³ → `400`, bukan 500), `PATCH …/status` mengisi harga riil, stok souvenir, batas 1.000 | ✔ `TestShoppingLifecycle`, `TestShoppingValidation`, `TestShoppingOverflowIsA400`, `TestShoppingStatusFillsTheActualPrice`, `TestSouvenirStock`, `TestShoppingLimitOf1000` |
| Gulungan: konflik pada pos berpembayaran lalu `confirm`; idempoten; `item_ids` parsial; pos satuan/bertaut ditolak; barang baru dengan `budget_item_id` ikut hitungan; `unlink` melepas semua barang; pos bawaan dibuat bila belum ada; pos persentase delapan jenis; pos cadangan | ✔ `TestLinkToBudgetRollsTheListUpIntoTheDefaultLine`, `TestUnlinkReleasesEveryItem`, `TestLinkToBudgetNeedsConfirmWhenTheLineHoldsPayments`, `TestLinkToBudgetPartialItemsAndTheirRules`, `TestLinkToBudgetMakesTheDefaultLineWhenMissing`, `TestLinkToBudgetUsesALineOfThePercentTemplate`, `TestTheDefaultLineIsNotFoughtOverByAVendorAndTheList` |
| Template: wizard (37 barang, jenis lain ditolak dan proyek tidak terbentuk), `append` idempoten, `replace_template` dengan `confirm`/`budget_link_lost`, jenis tanpa template → `invalid_template` | ✔ `TestWizardCreatesTheShoppingTemplate`, `TestShoppingTemplateAppendIsIdempotentAndFillsGaps`, `TestShoppingTemplateReplaceNeedsConfirmWhenItCarriesWork`, `TestShoppingTemplateIsAskedForTheWeddingOnly` |
| Provider/applier/overlay terpasang; kunci ganda panik | ✔ `TestVendorWiring`, `TestEventTypesCarryTheVendorRegistry` |
| **Uji emas Proyek Contoh** (kontrak bab 9) | ✔ `TestSampleProjectVendorGolden`: 7 dari 10 kategori, fix 58.000.000, pipeline 4.300.000, tiga kelompok pas, Catering pagu 30.000.000 selisih +5.000.000, 37 barang 4.578.500 / 2.063.800 / 16 / 21, status bayar katering `partial` dan fotografi `dp_paid`, pos bersama dibaca di `limit_groups[].line`, gulungan tidak aktif (`default_target.has_payments = true`), tugas Visit Venue bertaut, anggaran tetap 74.150.000 |
| Sembilan jenis acara menjalani alur yang sama; tabel 9 jenis di [README](README.md#4-penerapan-9-jenis-acara) sama dengan registry | ✔ `TestEveryEventTypeRunsTheWholeVendorFlow`, `TestReadmeNineTypeTableMatchesRegistry` |
| Integrasi Menu 03: pos yang digulung menolak pembayaran (`409 rollup_locked`) dan menerimanya lagi setelah `unlink-from-budget` | ✔ `TestMenu03LocksThePaymentsOfARolledUpLine` |
| Isolasi: pengguna B `404` di 17 rute proyek A dan pada id proyek A di proyek B; tanpa token `401`; arsip menolak tulis, `GET` dan ekspor tetap boleh; body rusak | ✔ `TestUserBGetsNotFoundOnEveryVendorEndpoint`, `TestEveryVendorRouteNeedsABearerToken`, `TestArchivedProjectIsReadOnlyButStillReadable`, `TestVendorEndpointsRejectMalformedBodies` |
| Ekspor: dibaca ulang (`excelize`), kontak tanpa harga dan hanya `fix`, nama berawalan `=` aman, nama berkas dari slug, permintaan ke-11 `429`, tanpa `format` `400` | ✔ `TestContactSheetOfTheSampleProject`, `TestContactSheetEscapesFormulas`, `TestShoppingSheetOfTheSampleProject`, `TestExportNeedsAFormatAndIsRateLimited`, `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow`, `TestSafeCellPrefixesFormulas` |
| Kontrak OpenAPI: 17 operasi dan 26 skema = rute dan tag JSON struct | ✔ `TestOpenAPIDocumentsEveryVendorRoute`, `TestOpenAPISchemasMatchTheGoDTOs` |

### 7.3 Hasil implementasi (2026-10-23)

PostgreSQL 16.2 lokal, database sementara `planner_vendor_impl_test` (dihapus setelahnya), pgx mode simple-protocol. **334 tes tingkat atas lulus** (261 sebelumnya + 73 baru), `-race` lulus, cakupan modul 91,4%, `go vet` dan `gofmt` bersih, `redocly lint` valid (3 peringatan lama). **37 mutasi** kode produksi: 36 digagalkan tes, satu setara (pencarian vendor tanpa filter proyek, karena setiap penulisan memakai filter proyeknya sendiri). Respons HTTP asli Proyek Contoh **sama dengan** [fixtures/planner-vendors.json](fixtures/planner-vendors.json) dan [fixtures/planner-shopping.json](fixtures/planner-shopping.json) (struktur, tipe, dan nilai; nol selisih setelah id dinormalkan). Rincian, penyimpangan dari dokumen ini, dan yang belum diuji: [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md).

Temuan saat implementasi yang tidak terlihat pada audit dokumen (dicatat di [bab 9 butir 22–28](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)): tabrakan pos gulungan bawaan dengan pos khusus vendor pada tiga jenis acara; batas 300 pos Menu 03 berlaku untuk pos yang dibuat di sini; pernyataan bersyarat `vendor_id IS NULL` saja sudah menutup balapan tautan, sehingga yang dijaga kunci proyek di `lock-deal` hanyalah batas 300 pos (tanpa kunci tujuh dari delapan deal lolos pada 296 pos), kini dibuktikan tes tersendiri.

---

## 8. Urutan pengerjaan dan definisi selesai

Posisi di urutan pembangunan ([kontrak bab 9](../00_KONTRAK_BERSAMA/README.md#9-urutan-pembangunan-dan-uji-penerimaan)): setelah `planner` (Menu 01), `plannertimeline` (02), dan `plannerfinance` (03); tabel dan view yang dibutuhkan sudah ada. Vendor (04a) dan belanja (04b) berada di satu modul tetapi dapat dikirim bertahap.

1. **Domain**: DTO bab 4, `NormalizeWA` (di `planner/domain/phone.go`), `CategoryKeyFromLabel`, `LineTotal`, `BuildOverview`, registry (`VendorCategories`, `ShoppingCategories`, `ShoppingRollup`) beserta datanya untuk sembilan jenis, data template 37 barang; tes data dan registry.
2. **Repository** dan **usecase** vendor (CRUD, `PATCH status`, `lock-deal`, ringkasan) lalu belanja (CRUD, status, template, gulungan, KPI), dengan `LockProject`.
3. **Delivery**: 17 rute, DTO ke JSON, pemetaan galat, pembatas laju ekspor (salinan kelima pola `ratelimit.go`; ekstraksi dicatat di [02 BE bab 8 butir 13](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)), ekspor `.xlsx`.
4. **Sambungan** di `internal/app/app.go` ([bab 6](#6-sambungan-ke-menu-01)): provider `vendors`, applier `shopping`, overlay `vendors` dan `shopping`, rute.
5. **Kontrak dan dokumen**: operasi dan skema ke `backend/docs/openapi.yaml` (tag `Planner Vendor`); dokumen `docs/backend/planner/<tanggal>-planner-menu-04.md`; `docs/api.md`, `docs/database.md` (tidak ada tabel baru), `docs/backend/CHANGELOG.md`; kode galat dan warning baru di kontrak bab 6; entri CHANGELOG spesifikasi.
6. **Uji emas** Proyek Contoh dan seluruh tabel [bab 7.2](#72-pengujian-pada-implementasi); `-race`, `go vet`, `gofmt`, cakupan.

**Status: langkah 1–6 selesai 2026-10-23** (kecuali ekstraksi pembatas laju, yang tetap salinan kelima; dokumen modul ada di [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md)).

**Definisi selesai:** semua butir bab 7.2 lulus terhadap PostgreSQL; widget Vendor di dashboard terisi dan `templates_available.shopping_list` benar untuk pernikahan; Proyek Contoh memuat 10 vendor dan 37 barang dengan angka PDF; OpenAPI lolos lint; dokumen modul menyebut setiap penyimpangan dari spesifikasi ini.

---

## 9. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–03 (kode dan dokumen), kontrak, PDF, dan SQL yang dijalankan. Dokumen mengikuti pilihan yang tertulis; kolom "alternatif" adalah yang bisa dipilih bila produk tidak setuju.

1. **Batas jumlah** 200 vendor dan 1.000 barang per proyek, serta batas lain di bab 3.4 (panjang kolom, `quantity` ≤ 1.000.000), adalah angka usulan tanpa pengukuran; daftar tidak berpaginasi.
2. **Status `fix` mensyaratkan deal > 0 di semua jalur** (`POST`, `PUT`, `PATCH status`); sinkron ke anggaran hanya lewat `lock-deal`. Alternatif: `PUT`/`PATCH` ke `fix` otomatis menjalankan `lock-deal`.
3. **Kolom `planner_vendors.price_delta` mubazir** (hanya benar untuk pagu manual). Dibiarkan agar DDL tetap identik dengan migrasi 006; alternatif: membuangnya dari migrasi 006 (diizinkan pada tahap pengembangan).
4. **Pagu dibaca langsung dari skenario aktif, tidak disimpan**, sehingga `limit_source` tidak punya kolom. Alternatif: menyalin ke `budget_limit` dan membuat Apply skenario Menu 03 menulis ke tabel vendor (kopling baru dan pengecualian SQL kelima).
5. **Pagu dan `lock-deal` mencoba sesi vendor lalu Umum (`all`)**, karena template dan preset non-pernikahan berada di `all`. Alternatif: menaruh pos non-pernikahan di sesi nyata (mengubah 03 dan TEMPLATE_EVENT).
6. **Pos gulungan belanja per jenis acara** (registry `ShoppingRollup`, TEMPLATE_EVENT 15.4) menutup [03 BE bab 9 butir 29](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) dengan pilihan (b): khitanan dan aqiqah `undangan_souvenir`, peresmian `prosesi_vip`, launching `merchandise_kit`, ulang tahun `souvenir`, wisuda `toga_atribut`, seminar `kit_sertifikat`, lainnya `perlengkapan` dibuat bila belum ada. Pilihan pos adalah usulan Adicara. Menggulung belanja ke pos berpersentase **mengganti** estimasi persentasenya dengan jumlah belanja (aksi pengguna, ada konfirmasi bila berpembayaran). Alternatif: (a) selalu membuat pos baru di luar persentase pagu, atau (c) menambah pos `perlengkapan` ke tabel persentase (mengubah angka tiap jenis). Tiga dari pilihan itu bertabrakan dengan pemetaan vendor dan ditangani pos cadangan (butir 22).
7. **Menerapkan ulang skenario menimpa estimasi pos yang bertaut vendor** (perilaku kode Menu 03 saat ini): deal tersimpan di vendor, pos kembali ke nilai skenario. Dikompensasi dengan lencana "Pos ≠ deal" dan warning `estimate_differs`. Alternatif: Apply melewati pos bertaut vendor seperti rollup dan pos satuan (alasan baru di `ApplyReport.skipped`, perubahan kode 03).
8. **Hanya nomor Indonesia** yang diterima `NormalizeWA`; acara dengan vendor luar negeri tidak dapat menyimpan nomor `+65…`. Alternatif: menerima kode negara lain tanpa memaksa `62` (menyimpan nomor tanpa panjang yang diperiksa).
9. **Ekspor hanya `.xlsx`**: PDF menunggu persetujuan pustaka ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)). Ekspor kontak sengaja tanpa harga dan catatan karena lembar dikirim ke WO/panitia. Tata letak (dua lembar, bab 2.7) kini dibuat dan dibaca ulang dengan `excelize`, tetapi belum dibuka di Excel/Numbers; awalan `'` tampak di sel ([02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
10. **`GET /vendors/:id` hanya `Vendor`**: tugas dan agenda yang menunjuk vendor tidak diringkas karena tidak ada layar yang memakainya (Menu 02 dan 06 menampilkan nama vendor di baris mereka sendiri). Alternatif: menambah daftar `linked_tasks` bila ada layar detail.
11. **Label status `rencana` adalah "Rencana"**, bukan "Riset": PDF menyebutnya *List Rencana Vendor* dan tidak ada kata "riset" di enum atau dokumen lain. Chip pipeline memisahkan Rencana dan Negosiasi (versi awal menjumlah keduanya sebagai "Riset (3)").
12. **Stok souvenir** = `floor(Σ quantity)` barang selesai (jumlah boleh berdesimal); 0 bila ada barang souvenir tetapi belum selesai, hilang bila tak ada.
13. **Template belanja hanya pernikahan**, semua di sesi `all`; delapan jenis lain tidak punya barang bawaan (hanya kategori). Menambah template jenis lain = menambah data dan `TemplateSet.ShoppingList`.
14. **Tidak ada urut ulang manual** vendor dan barang (urut `sort_order` = urutan dibuat); wireframe tidak menyediakan seret-lepas.
15. **Kategori belanja adalah teks bebas** (tanpa kunci); dropdown berisi registry ∪ yang sudah dipakai; dua ejaan berbeda menjadi dua kategori.
16. **Melepas `fix` melepas tautan pos**; mengunci ulang menautkan pos yang sama hanya bila `item_key`-nya masih kunci kategori. Pos yang dibuat untuk vendor kedua sekategori (`item_key NULL`) tidak ditemukan lagi, sehingga kunci ulang membuat pos baru dan pengguna menghapus yang lama. Alternatif: mempertahankan tautan selama vendor `batal`/`rencana`.
17. **Kejanggalan data PDF di Proyek Contoh**: pos Hiburan sudah lunas padahal MC dan Entertainment masih negosiasi (bab 5.2). Dibiarkan sebagai angka PDF.
18. **Menu 05 membaca stok lewat SQL** (bab 6). Alternatif: antarmuka Go `SouvenirStock` di `planner/domain` yang diimplementasikan modul ini.
19. **`PUT` barang tidak mengubah tautan rollup**; hanya `POST` (dengan `budget_item_id`) dan endpoint link/unlink yang mengubahnya. Barang yang dibuat tanpa tujuan, saat gulungan aktif, tidak masuk hitungan sampai pengguna menautkannya (antarmuka mengisi tujuan dari banner).
20. **Kinerja belum diukur**: kueri daftar (satu baca vendor + satu baca skenario + satu baca view) tidak diuji pada data 10× contoh.
21. **Perubahan terhadap spesifikasi sebelumnya yang menyentuh dokumen lain** (semuanya dicatat di [CHANGELOG](../CHANGELOG.md)): kontrak (pengecualian SQL keempat, kode galat dan warning baru, `ShoppingCategories []string`, `ShoppingRollup`), TEMPLATE_EVENT bab 15.4 (baru), 03 BE butir 29, 02 FE bab 13 butir 2, dan wireframe Menu 04 (pagu Catering, status bayar kelompok).
22. **Pos gulungan bawaan bertabrakan dengan pos khusus vendor pada tiga jenis** (ditemukan saat implementasi, dijalankan oleh `TestEveryEventTypeRunsTheWholeVendorFlow`). Peresmian `prosesi_vip` (juga pos kategori `mc`), wisuda `toga_atribut` (kategori `toga`), seminar `kit_sertifikat` (kategori `percetakan`): begitu vendor itu dikunci dan tertaut, pos tidak lagi dapat digulung dan tombol "Hitung pos dari daftar belanja" berakhir `400 must not be linked to a vendor or document`. Bawaan sekarang: pos cadangan `perlengkapan` (bab 2.4), dibuat sekali di luar persentase pagu. Alternatif: (a) ubah TEMPLATE_EVENT 15.2 sehingga `mc`, `toga`, dan `percetakan` tidak punya pos khusus (vendor mereka membuat pos sendiri); (b) ubah pos bawaan 15.4 ke pos yang tak dipetakan ke vendor (peresmian `publikasi_media`, seminar `publikasi`; wisuda tidak punya pos yang sesuai); (c) mengizinkan pos bertaut vendor digulung (estimasi pos ≠ deal, sulit dijelaskan).
23. **Pos yang dibuat modul ini menghormati batas 300 pos Menu 03** (`409 budget_item_limit_reached`; seluruh `lock-deal` dibatalkan, termasuk status `fix`). Menu 02 (`sync-budget`) membuat pos tanpa memeriksa batas itu; tidak diubah di sini.
24. **Id non-UUID pada body `link-to-budget` dan `unlink-from-budget`** dijawab `400 invalid_json` (dekoder body), bukan `404`; UUID yang benar tetapi bukan pos proyek ini `400 invalid_input`. Selaras dengan `ids` pada `reorder` Menu 03.
25. **`estimate_differs` pada `PUT` vendor** hanya bila `agreed_price` berubah pada permintaan itu; mengirim ulang harga yang sama tidak memberi peringatan.
26. **Satu pembatas laju untuk kedua ekspor** (10 permintaan per menit per pengguna, dihitung sebelum handler sehingga permintaan dengan `format` salah ikut terhitung).
27. **Pos skenario bernilai 0** memberi `limit_source: scenario`, `budget_limit: 0`, dan tanpa `price_delta` (paket 10 Jt pernikahan tak punya dokumentasi). Kolom Pagu di [FE.md](FE.md) menampilkan "Rp 0 · skenario" untuk keadaan ini; apakah 0 sebaiknya dibaca "tidak dianggarkan" (dan penawaran di atasnya "melebihi") adalah keputusan produk/FE, bukan BE.
28. **Perubahan kecil saat implementasi:** `apply-template` dan `link-to-budget` menerima badan kosong; `BuildOverview` tidak mengubah urutan masukannya (urutan tampil = urutan repository); `ScenarioRow` mendapat `ItemName`; label kategori vendor untuk kunci yang TEMPLATE_EVENT tulis tanpa label adalah kata biasanya ("Venue", "Dekorasi", …); kunci proyek pada `lock-deal` bukan satu-satunya penutup balapan tautan (pembaruan bersyarat `vendor_id IS NULL` juga menutupnya); yang hanya ia jaga adalah batas 300 pos (tanpa kunci, tujuh dari delapan deal lolos pada 296 pos; dibuktikan `TestConcurrentLockDealRespectsTheLineLimit`). Balapan kunci `item_key` pada pos baru tidak berhasil dipicu tanpa kunci (satu `INSERT … WHERE NOT EXISTS`), jadi `TestConcurrentLockDealWithoutADedicatedLine` adalah tes regresi, bukan bukti kunci.
