# Menu 04: Vendor & Belanja Kebutuhan — Backend Specification

Spesifikasi backend modul Go `plannervendor` (`backend/internal/modules/plannervendor/`) untuk **direktori/pipeline vendor** (PDF Hal. 12) dan **daftar belanja pengadaan** (PDF Hal. 9–10). Aturan lintas modul (uang, master sesi, template vs data contoh, kode error) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

---

## 1. Skema Database

### 1.1 Vendor

Migrasi `006_000_planner_vendors.sql` (dijalankan sebelum tugas, anggaran, belanja, dan rundown karena mereka menunjuk vendor; sudah diuji di PostgreSQL 16):

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

- `category_key` (slug stabil) dipakai untuk pencocokan pagu dan hitungan "kategori terkunci"; `category_label` hanya tampilan, jadi kategori kustom aman.
- **Tabel ini tidak punya `budget_item_id`.** FK dua arah dengan `planner_budget_items.vendor_id` tidak bisa dimigrasikan dalam urutan mana pun dan bisa menyimpang; satu-satunya arah adalah `planner_budget_items.vendor_id` (unik per vendor).
- `price_delta` bernilai `NULL` bila belum ada pagu atau penawaran (agar vendor yang belum menawar tidak tampil "hemat" sebesar pagu penuh).
- Uang `BIGINT`; `formatted_wa_number` diisi usecase (bab 2.3).

### 1.2 Daftar belanja

Migrasi `009_000_planner_shopping.sql` (setelah `008`, karena menunjuk `planner_budget_items`):

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

- Ditambah **`quantity` dan `unit`** (agar "300 seminar kit" dapat dicatat). `estimated_price`/`actual_price` adalah **harga per satuan**; `estimated_total`/`actual_total` generated. Untuk data PDF `quantity = 1`, sehingga total sama dengan harga PDF.
- `budget_item_id` menentukan pos anggaran tempat total belanja digulung (bab 2.4); `purpose = 'souvenir'` menandai barang pembanding stok souvenir Menu 05.
- Template tidak menyalin tautan Shopee pribadi pembuat PDF. Total belanja digulung lewat view (bab 2.4), bukan dengan menulis ke kolom generated `paid_total` (PostgreSQL menolak: *column can only be updated to DEFAULT*).

---

## 2. Logika Bisnis

### 2.1 Pagu vendor dari skenario aktif
Registry menyimpan per kategori vendor daftar `budget_item_keys` (kunci pos anggaran/skenario yang menampungnya). Tabel pernikahan di bab 5.1. Saat vendor dibuat/diubah (`category_key`, `sub_event_scope`), pagu dicari:

```sql
SELECT si.item_key, si.cost_amount
FROM planner_budget_scenario_items si
JOIN planner_budget_scenarios s ON s.id = si.scenario_id AND s.is_active
WHERE s.project_id = $1 AND si.sub_event_scope = $2 AND si.item_key = ANY($3::text[]);
```

- Bila kategori hanya satu pos dan pos itu tidak dipakai kategori vendor lain pada sesi yang sama → `budget_limit = cost_amount` (`limit_source: "scenario"`).
- Bila beberapa kategori berbagi satu pos (Venue + Decoration → `venue_dekor`; MUA + Attire → `rias_busana`; MC + Entertainment → `hiburan`, meniru pos gabungan PDF), `budget_limit` vendor dibiarkan `0` dan respons menyertakan **`limit_groups`**: `[{item_key, sub_event_scope, limit, total_agreed, delta}]` sehingga perbandingan dilakukan di tingkat pos (mis. Venue Rp 12 jt + Dekorasi Rp 8 jt terhadap pagu Rp 20 jt). Pencocokan lewat nama kategori ("Venue & Dekor" ≠ "Venue") tidak dipakai karena hampir selalu menghasilkan pagu 0.
- Kategori tanpa pos (Wedding Organizer, Hand Bouquet) → `budget_limit` manual. Pengguna selalu boleh menimpa pagu (`limit_source: "manual"`).
- Jenis non-pernikahan: `item_key` pos = slug nama pos; tiap kategori vendor menyebut `budget_item_keys` eksplisit di registry ([TEMPLATE_EVENT bab 15.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#152-pemetaan-kategori-vendor--pos-budget_item_keys)); kategori kustom tanpa pemetaan memakai pagu manual.

### 2.2 Kunci kerja sama (`lock-deal`) tanpa pos ganda
`POST .../vendors/:vendor_id/lock-deal` `{agreed_price, sync_to_budget_items}` dalam satu transaksi:

1. `status = 'fix'`, `agreed_price` diisi (harus > 0).
2. Bila `sync_to_budget_items`:
   - Vendor sudah punya pos (`planner_budget_items.vendor_id`) → perbarui `initial_estimate = agreed_price`.
   - Kategori vendor memetakan ke **pos bersama** (kunci yang dipakai ≥ 2 kategori pada registry, mis. `venue_dekor` oleh Venue + Decoration) → **tidak ditautkan dan estimasi pos tidak diubah**: pos itu mewakili beberapa vendor sekaligus, sehingga menimpanya dengan harga satu vendor menurunkan rencana (Venue Rp 12 jt akan menggantikan pos Rp 20 jt sebelum Dekorasi dikunci). Respons memuat `shared_item: {item_key, plan, total_agreed}` dan warning `shared_budget_item`; antarmuka menawarkan "Samakan estimasi pos dengan total deal kelompok" (aksi pengguna, `PUT /budget/items/:id`).
   - Kategori punya pos **khusus** (kunci hanya dipakai kategori itu) yang belum bertaut vendor dengan `(sub_event_scope, item_key ∈ budget_item_keys)` → tautkan (`vendor_id`) dan set `initial_estimate = agreed_price`.
   - Tidak ada yang cocok → buat pos baru (`source='vendor'`, kategori dan nama dari vendor, `initial_estimate = agreed_price`, `item_key = category_key` bila kunci itu belum dipakai pada sesi tersebut, selain itu `NULL` agar indeks unik `(sesi, item_key)` tidak bentrok).
3. Vendor kedua pada kategori yang sama dengan pos khusus yang sudah bertaut (mis. katering B setelah katering A) mendapat pos baru dengan `item_key NULL`; tidak ada yang ditimpa.
4. Batal kunci (`status` kembali bukan `fix`) tidak menghapus pos; tautan `vendor_id` dilepas dan pos menjadi pos biasa.

Respons: `{vendor, budget_item_id, warnings}`. Pembayaran vendor dibaca dari pos (`paid`, `remaining`, `payment_status` pada view), jadi **status bayar di tabel vendor selalu berasal dari Tab Realisasi (Menu 03 Tab 3)**.

### 2.3 Normalisasi nomor WhatsApp
Fungsi murni `NormalizeWA(input) (string, error)`: buang spasi, `-`, `()`; `+62…` → `62…`; `0…` → `62…`; `62…` dipertahankan; selain itu (mis. diawali `8` atau huruf) → `400 invalid_phone`. Panjang akhir 10–15 digit. Contoh: `081298765432` → `6281298765432`; `+62812-9876-5432` → `6281298765432`. Tautan: `https://wa.me/6281298765432?text=<url-encoded>`; teks bawaan: *"Halo {PIC}, saya ingin menanyakan perihal kerja sama untuk acara {judul acara} pada sesi {nama sesi}…"*. Nomor disimpan dua bentuk (`phone_number` apa adanya, `formatted_wa_number` ternormalisasi).

### 2.4 Daftar belanja digulung ke anggaran (*rollup*)
Total belanja tidak disalin ke anggaran; pos anggaran dapat **dihitung dari daftar belanja** melalui view `planner_budget_lines` ([03 BE §1.2](../03_KEUANGAN/BE.md#12-view-baca-planner_budget_lines)): `estimate = Σ estimated_total`, `paid = Σ actual_total` untuk barang berstatus `selesai`, atas barang yang `budget_item_id`-nya menunjuk pos itu.

`POST .../shopping-items/link-to-budget` `{budget_item_id?, item_ids?, confirm?}`:
- Target pos: `budget_item_id` bila ada; selain itu pos dengan `item_key` rollup registry (pernikahan: `seserahan`; jenis lain: `perlengkapan`) pada sesi apa pun; bila tidak ada, dibuat (`source='manual'`).
- Barang: `item_ids` bila ada, selain itu semua barang yang belum bertaut.
- Mengatur `rollup_source = 'shopping'` pada pos. Bila pos sudah punya pembayaran tercatat (DP/termin) → `409 rollup_conflict` kecuali `confirm: true` (pembayaran tersimpan tetapi diabaikan selama rollup aktif). Selama rollup, pembayaran manual pada pos ditolak (`rollup_locked`).
- `POST .../shopping-items/unlink-from-budget` `{budget_item_id}`: mencabut rollup; pos kembali memakai angka tersimpannya.

Status barang `selesai` dengan `actual_price = 0` otomatis mengisi `actual_price = estimated_price` (dapat diedit); aturan ini dijalankan server agar konsisten di semua klien. **Proyek Contoh tidak mengaktifkan rollup** agar angka PDF Hal. 5–6 tetap utuh.

### 2.5 Stok souvenir
Barang `purpose = 'souvenir'` berstatus `selesai` membentuk stok (`Σ quantity`); serah terima dari Menu 05 (`souvenir_received`). Menu 05 menampilkan `stok − diserahkan` ([05 BE §2.5](../05_BUKU_TAMU/BE.md#25-meja-resepsionis-hari-h)).

### 2.6 Template
`POST .../shopping-items/apply-template` `{template_key, mode}`: pernikahan `seserahan_wedding` (37 item, bab 5.3), `mode` `append` (lewati `template_key` yang ada) atau `replace_template`. Barang template berstatus `belum`, `actual_price = 0`, `purchase_location` kosong, sesi `all`. Jenis lain: tidak ada barang template (hanya kategori di dropdown).

### 2.7 Ekspor
Daftar kontak darurat vendor `fix` dan daftar belanja dalam `.xlsx`/PDF; sel di-escape; 10 ekspor/menit/pengguna.

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, `BearerAuth` + `ProjectGuard` + `WriteGuard`.

### 3.1 Vendor
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/vendors?scope=&status=&category=` | KPI + `categories` (checklist registry ∪ kustom: `fix`/`pipeline`/`empty`) + `limit_groups` + vendor beserta `payment` dari view. |
| `GET` | `/vendors/:vendor_id` | Detail + pos tertaut + tugas/agenda yang menunjuk vendor. |
| `POST` | `/vendors` | Tambah kandidat. |
| `PUT` | `/vendors/:vendor_id` | Ubah data, kontak, penawaran, status. |
| `POST` | `/vendors/:vendor_id/lock-deal` | §2.2. |
| `DELETE` | `/vendors/:vendor_id` | `204`; rujukan di tugas/pos/belanja/agenda menjadi kosong (`SET NULL`), pos anggaran tetap. |
| `GET` | `/vendors/export-contacts?format=xlsx\|pdf` | Kontak vendor `fix` untuk WO/panitia. |

### 3.2 Belanja
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/shopping-items?category=&status=&scope=` | KPI + barang. |
| `POST` | `/shopping-items` | Tambah barang. |
| `PUT` | `/shopping-items/:item_id` | Ubah barang. |
| `PATCH` | `/shopping-items/:item_id/status` | Toggle `belum` → `proses` → `selesai`. |
| `DELETE` | `/shopping-items/:item_id` | `204`. |
| `POST` | `/shopping-items/apply-template` | §2.6. |
| `POST` | `/shopping-items/link-to-budget` | §2.4. |
| `POST` | `/shopping-items/unlink-from-budget` | §2.4. |
| `GET` | `/shopping-items/export-shopping-list?format=xlsx\|pdf` | Rekap belanja. |

---

## 4. DTO (Go)

```go
package domain

type VendorStatus string // "rencana" | "negosiasi" | "fix" | "batal"

type CreateVendorRequest struct {
	SubEventScope   string  `json:"sub_event_scope" binding:"required"`
	CategoryKey     string  `json:"category_key" binding:"required"`  // dari registry atau slug kustom
	CategoryLabel   string  `json:"category_label"`                   // wajib untuk kategori kustom
	VendorName      string  `json:"vendor_name" binding:"required,max=255"`
	PICName         *string `json:"pic_name"`
	PhoneNumber     *string `json:"phone_number"`                     // 400 invalid_phone bila tak bisa dinormalisasi
	PortfolioURL    *string `json:"portfolio_url"`                    // http/https
	BudgetLimit     *int64  `json:"budget_limit"`                     // nil = hitung dari skenario (bab 2.1)
	AgreedPrice     int64   `json:"agreed_price" binding:"gte=0"`
	Status          string  `json:"status" binding:"required,oneof=rencana negosiasi fix batal"`
	Notes           *string `json:"notes"`
}
// UpdateVendorRequest: field yang sama.

type LockDealRequest struct {
	AgreedPrice       int64 `json:"agreed_price" binding:"required,gt=0"`
	SyncToBudgetItems bool  `json:"sync_to_budget_items"`
}

type VendorOverview struct {
	KPI struct {
		FixedVendors, PipelineVendors int
		CategoriesTotal, CategoriesFixed int    // kategori, bukan baris vendor
		TotalAgreedFix, TotalLimitFix  int64
	} `json:"kpi"`
	Categories  []struct{ Key, Label, State string; Vendors int } `json:"categories"` // State: fix | pipeline | empty
	LimitGroups []struct{ ItemKey, SubEventScope string; Limit, TotalAgreed, Delta int64 } `json:"limit_groups"`
	Vendors     []Vendor `json:"vendors"`
}

type Vendor struct {
	ID                uuid.UUID `json:"id"`
	SubEventScope     string    `json:"sub_event_scope"`
	CategoryKey, CategoryLabel, VendorName string
	PICName, PhoneNumber, FormattedWANumber, PortfolioURL *string
	BudgetLimit       int64     `json:"budget_limit"`
	LimitSource       string    `json:"limit_source"` // scenario | manual | group
	AgreedPrice       int64     `json:"agreed_price"`
	PriceDelta        *int64    `json:"price_delta"`  // nil bila belum ada pagu/penawaran
	Status            VendorStatus `json:"status"`
	BudgetItemID      *uuid.UUID `json:"budget_item_id,omitempty"` // dari planner_budget_items.vendor_id
	Payment           *struct{ Estimate, Paid, Remaining int64; Status string } `json:"payment,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
}

type CreateShoppingItemRequest struct {
	SubEventScope    string     `json:"sub_event_scope" binding:"required"`
	Category         string     `json:"category" binding:"required"`
	ItemName         string     `json:"item_name" binding:"required,max=255"`
	Quantity         float64    `json:"quantity" binding:"omitempty,gt=0"` // default 1
	Unit             *string    `json:"unit"`
	EstimatedPrice   int64      `json:"estimated_price" binding:"gte=0"`   // per satuan
	ActualPrice      int64      `json:"actual_price" binding:"gte=0"`      // per satuan
	PurchaseLocation *string    `json:"purchase_location"`                 // teks bebas; URL dirender sebagai tautan
	VendorID         *uuid.UUID `json:"vendor_id"`
	Purpose          *string    `json:"purpose" binding:"omitempty,oneof=souvenir"`
	Status           string     `json:"status" binding:"required,oneof=belum proses selesai"`
	Notes            *string    `json:"notes"`
}
// UpdateShoppingItemRequest: field yang sama.

type ShoppingOverview struct {
	KPI struct {
		EstimatedTotal, ActualTotal int64 // actual hanya barang selesai
		Done, InProgress, NotStarted, Total int
	} `json:"kpi"`
	Items []ShoppingItem `json:"items"`
}
```

---

## 5. Master Data

### 5.1 Kategori vendor pernikahan (PDF Hal. 12)

PDF memuat sepuluh kategori di "List Rencana Vendor" dan tujuh di "Fix Kerjasama Vendor" (Venue, Decoration, Wedding Organizer, MUA, Catering, Photography, Attire; baris data PDF hanya contoh dummy). Template **tidak membuat baris vendor**; sepuluh kategori ini menjadi checklist "kategori terkunci" (7 dari 10 pada PDF):

| `category_key` | Label (PDF) | `budget_item_keys` |
|---|---|---|
| `venue` | Venue | `venue_dekor` |
| `dekorasi` | Decoration | `venue_dekor` |
| `wo` | Wedding Organizer | – |
| `mua` | MUA | `rias_busana` |
| `catering` | Catering | `catering` |
| `fotografi` | Photography | `dokumentasi` |
| `attire` | Attire | `rias_busana` |
| `hand_bouquet` | Hand Bouquet | – |
| `mc` | MC | `hiburan` |
| `entertainment` | Entertainment | `hiburan` |

Biaya KUA bukan vendor melainkan pos legalitas di Menu 03.

### 5.2 Vendor Proyek Contoh (fiktif)

Dataset hanya untuk Proyek Contoh. Nama, PIC, dan nomor telepon **fiktif** (nomor memakai pola `08120000xxxx`, tidak mengarah ke orang nyata). Semua `sub_event_scope = resepsi`. Tujuh vendor `fix` sama dengan tujuh kategori PDF; tiga lainnya masih riset/negosiasi.

| Kategori | Nama vendor | Telepon | Penawaran / deal | Status | Tertaut pos |
|---|---|---|--:|:-:|---|
| venue | Sasana Kriya Ballroom | 081200000001 | 12.000.000 | fix | (pos bersama `venue_dekor`) |
| dekorasi | Pelaminan Mega Indah | 081200000002 | 8.000.000 | fix | (pos bersama `venue_dekor`) |
| wo | Harmony Wedding Organizer | 081200000003 | 5.000.000 | fix | – |
| mua | Sanggar Rias Ayu | 081200000004 | 3.500.000 | fix | (pos bersama `rias_busana`) |
| catering | Katering Bu Minah (500 pax) | 081200000005 | 25.000.000 | fix | `catering` |
| fotografi | Studio Cahaya | 081200000006 | 3.000.000 | fix | `dokumentasi` |
| attire | Butik Kebaya Anggun | 081200000007 | 1.500.000 | fix | (pos bersama `rias_busana`) |
| hand_bouquet | Fleur Hand Bouquet | 081200000008 | 900.000 | rencana | – |
| mc | Kang Dedi Master MC | 081200000009 | 2.000.000 | negosiasi | (pos bersama `hiburan`) |
| entertainment | Akustik Nada Kirana | 081200000010 | 1.400.000 | negosiasi | (pos bersama `hiburan`) |

Kelompok pagu: `venue_dekor` 20.000.000 (Venue 12 + Dekorasi 8 = 20, selisih 0); `rias_busana` 5.000.000 (MUA 3,5 + Attire 1,5 = 5); `hiburan` 3.400.000 (MC 2,0 + Entertainment 1,4 = 3,4). Ketiga pos itu adalah pos bersama sehingga tidak ada vendor yang tertaut ke `planner_budget_items.vendor_id` untuk kelompok tersebut pada dataset; hanya `catering` dan `fotografi` (pos khusus `catering` dan `dokumentasi`) yang tertaut.

### 5.3 Daftar Belanja: 37 Item Seserahan (PDF Hal. 9–10)

Nama barang, kategori, dan harga persis PDF; 37 item, estimasi total **Rp 4.578.500**. Template memuat kolom *Harga acuan* dengan `status = belum`, `actual_price = 0`, tanpa tautan, `quantity = 1`, sesi `all`. Kolom status berlaku untuk Proyek Contoh (PDF: Selesai/On Proses; `actual_price = harga acuan` untuk yang selesai): **16 selesai = Rp 2.063.800**, **21 proses = Rp 2.514.700**.

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

PDF juga memuat kolom "Link Pembelian" berisi tautan pribadi pembuat dokumen; tautan itu **tidak** disalin ke template maupun Proyek Contoh.

### 5.4 Jenis lain
Kategori vendor dan kategori belanja non-pernikahan: [TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md).

---

## 6. Pengujian yang Dibutuhkan
- `NormalizeWA`: tabel kasus (`08…`, `+62…`, `62…`, spasi/tanda hubung, huruf, terlalu pendek).
- `lock-deal`: pos khusus tertaut tanpa pos ganda; pos bersama tidak ditautkan dan estimasinya tidak berubah (`shared_budget_item`); vendor kedua pada kategori yang sama mendapat pos baru; vendor tidak bisa menaut pos proyek lain; hapus vendor melepas tautan.
- Pagu: kelompok `venue_dekor` menghasilkan `limit_groups` dan `budget_limit = 0`.
- Rollup: aktifkan pada pos berpembayaran → `rollup_conflict`; dengan `confirm` → `rollup_locked` untuk pembayaran; angka view = `Σ estimated_total` / `Σ actual_total` barang selesai.
- Uji emas PDF: 37 item, 4.578.500, 16 selesai = 2.063.800, 21 proses = 2.514.700; kategori vendor 7 dari 10.
