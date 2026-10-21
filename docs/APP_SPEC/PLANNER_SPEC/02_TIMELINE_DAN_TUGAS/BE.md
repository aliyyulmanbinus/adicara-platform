# Menu 02: Timeline & Tugas Persiapan — Backend Specification

Spesifikasi backend modul Go `plannertimeline` (`backend/internal/modules/plannertimeline/`) untuk **jadwal/tugas persiapan** (PDF Hal. 3) dan **dokumen legalitas/perizinan** (PDF Hal. 11). Aturan lintas modul ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md); titik sambung yang sudah disediakan Menu 01 (`SummaryProvider`, `TemplateApplier`, `SampleOverlay`) dijelaskan di [bab 4.6](#46-sambungan-ke-menu-01).

> **Status (2026-10-21):** modul `plannertimeline` **sudah diimplementasikan** di `backend/internal/modules/plannertimeline/` (`domain`, `repository`, `usecase`, `delivery/http`) dan terpasang di `internal/app/app.go`: 17 endpoint, dua provider (`timeline`, `admin_docs`), dua applier, dua overlay, dan registry jenis acara punya `admin_subjects`, `admin_guide`, `AdminLabel`. Widget Timeline dan Legalitas di dashboard terisi, `GET /event-types` menawarkan `timeline_presets`, `include_adicara_tasks`, dan `admin_docs`, dan Proyek Contoh berisi 50 tugas dan 15 berkas (uji emas lulus). Dokumen implementasi beserta penyimpangan dari spesifikasi: [docs/backend/planner/2026-10-21-planner-menu-02.md](../../../backend/planner/2026-10-21-planner-menu-02.md); kontrak API: `backend/docs/openapi.yaml` (tag `Planner Timeline`). Frontend menu ini belum ada.
>
> Dokumen ini diaudit dengan **menjalankan** SQL dan algoritmanya sebelum kode ditulis, lalu kodenya diuji terhadap PostgreSQL 16.2 (hasil dan batasnya di [bab 6](#6-pengujian)). Yang **belum** dijalankan: image Docker, Nginx di depan rute baru, PostgreSQL 18 (versi Compose), beberapa replika API (pembatas laju ekspor di memori per proses), dan berkas `.xlsx` di Excel, Numbers, atau LibreOffice.
>
> Pola dokumen mengikuti [01 BE](../01_DASHBOARD/BE.md): DTO tanpa tag `binding` (validasi di usecase dengan pesan `<field>: <alasan>`), bentuk respons per endpoint ([bab 3.3](#33-bentuk-respons)), bab sambungan ke Menu 01, pengujian dipetakan ke nama tes, serta celah yang menunggu keputusan ([bab 8](#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 1. Skema Database

Migrasi `007_000_planner_timeline.sql` ada di `backend/internal/migrate/sql/` dan ikut naik, turun, naik lagi pada `TestPlannerMigrationsRollBackAndReapply`; kolom `vendor_id` mengarah ke `planner_vendors` dari migrasi 006. **Blok `sql` pertama di bawah adalah sumber salinan file migrasi itu** (006–012 disalin otomatis dari BE.md modulnya): ubah keduanya bersamaan. Blok `sql` lain di dokumen ini adalah kueri, bukan migrasi.

```sql
-- +goose Up
CREATE TABLE planner_tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL DEFAULT 'Umum',
    phase_order INT NOT NULL DEFAULT 1,
    phase_name TEXT NOT NULL DEFAULT '',                -- nama fase ikut tersimpan; tidak ada tabel fase
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    -- Konvensi PDF Hal. 3: durasi = selesai - mulai (1 Feb -> 2 Feb = "1 Hari"), minimal 1.
    duration_days INT GENERATED ALWAYS AS (GREATEST(end_date - start_date, 1)) STORED,
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'proses', 'selesai')),
    pic TEXT NOT NULL DEFAULT '',
    vendor_id UUID,
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    template_key TEXT,                                  -- mis. wedding.pertemuan_keluarga_1; dasar mode "append"
    origin TEXT NOT NULL DEFAULT 'custom' CHECK (origin IN ('pdf', 'adicara', 'custom')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id)
);

CREATE INDEX planner_tasks_project_sort_idx ON planner_tasks (project_id, sort_order);
CREATE INDEX planner_tasks_project_dates_idx ON planner_tasks (project_id, start_date, end_date);
CREATE UNIQUE INDEX planner_tasks_template_key_idx ON planner_tasks (project_id, template_key) WHERE template_key IS NOT NULL;

CREATE TABLE planner_admin_docs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    document_name TEXT NOT NULL CHECK (char_length(document_name) BETWEEN 1 AND 255),
    subject_role TEXT NOT NULL DEFAULT 'bersama',       -- divalidasi usecase terhadap registry event type
    quantity_detail TEXT NOT NULL DEFAULT '',
    nominal_cost BIGINT NOT NULL DEFAULT 0 CHECK (nominal_cost >= 0),
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'selesai')),
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    template_key TEXT,
    origin TEXT NOT NULL DEFAULT 'custom' CHECK (origin IN ('pdf', 'adicara', 'custom')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

CREATE INDEX planner_admin_docs_project_sort_idx ON planner_admin_docs (project_id, sort_order);
CREATE UNIQUE INDEX planner_admin_docs_template_key_idx ON planner_admin_docs (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_admin_docs;
DROP TABLE IF EXISTS planner_tasks;
```

Catatan desain:
- `duration_days` adalah **kolom generated** `GREATEST(end_date - start_date, 1)` mengikuti konvensi PDF Hal. 3 ("1 Feb → 2 Feb" = "1 Hari"); ia tidak bisa menyimpang dari tanggal. Balok Gantt tetap inklusif (3 kolom untuk 5→7 Feb, durasi 2): dua hal berbeda yang disengaja.
- `sub_event_scope` terhubung ke master sesi lewat FK komposit ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)); tidak ada konstanta Go untuk nilai scope karena nilainya berasal dari sesi proyek.
- `phase_name` ikut tersimpan pada tugas, jadi jenis acara non-pernikahan punya nama fase sendiri; `vendor_id`, `template_key`, `origin` mendukung tautan vendor dan mode *append*.
- Peran orang tua memakai `ortu_cpp`/`ortu_cpw` (PDF menulis CMP/CMW untuk makna yang sama). Status "tersinkron ke anggaran" diturunkan dari `planner_budget_items.admin_doc_id` (§4.5), bukan disimpan di dokumen. Uang `BIGINT`.
- Mode *append* mencocokkan `template_key` karena judul PDF berulang ("Pertemuan keluarga" ×5).
- **Waktu galat berbeda per FK** (terbukti dengan eksekusi): vendor milik proyek lain memicu `23503` `planner_tasks_vendor_id_project_id_fkey` **pada pernyataan itu**; sesi tak dikenal memicu `23503` `planner_tasks_project_id_sub_event_scope_fkey` **baru saat `COMMIT`** (FK ditunda). Karena itu usecase memeriksa vendor dan sesi lebih dulu (`400 invalid_input: vendor_id` dan `400 invalid_scope`), dan galat commit yang tersisa menjadi `409 conflict` ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)).
- `CHECK (end_date >= start_date)` tidak bernama (`planner_tasks_check`, `23514`), jadi usecase memvalidasi tanggal lebih dulu (`400 invalid_date`) dan tidak mengandalkan pesan basis data.
- `document_name` boleh 255 karakter, sedangkan `planner_budget_items.item_name` dibatasi 150 (`planner_budget_items_item_name_check`). Sinkron biaya memotong nama menjadi 150 karakter (§4.5); tanpa itu berkas bernama 255 karakter membuat sinkron gagal `23514` (terbukti dengan eksekusi).

---

## 2. Domain & DTO (Go)

Uang adalah `int64` rupiah utuh. Dalam kode, entitas domain memakai `time.Time` (00:00 UTC) untuk `DATE` dan `delivery/http/dto.go` memetakannya ke `"YYYY-MM-DD"` seperti Menu 01; blok di bawah menunjukkan **bentuk JSON**. Request **tanpa tag `binding`**: semua validasi di usecase supaya string kosong atau berspasi sampai ke sana dan dijawab `<field>: <alasan>` (tag `binding:"required"` menjawab `invalid_json` tanpa nama kolom; alasan lengkap di [01 BE bab 2](../01_DASHBOARD/BE.md#2-domain--dto-go)). Notifikasi memakai `domain.Notice` milik Menu 01 (`code`, `message`, `count`, `shift_days?`).

```go
package domain

type TaskStatus string // "belum" | "proses" | "selesai"
type DocStatus string  // "belum" | "selesai"
type Origin string     // "pdf" | "adicara" | "custom"

// ---------- Tugas ----------

type Task struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	SubEventScope string     `json:"sub_event_scope"` // scope_key dari master sesi; "all" = Umum
	Category      string     `json:"category"`
	PhaseOrder    int        `json:"phase_order"`
	PhaseName     string     `json:"phase_name"`
	StartDate     string     `json:"start_date"` // YYYY-MM-DD
	EndDate       string     `json:"end_date"`
	DurationDays  int        `json:"duration_days"` // kolom generated; klien tidak menghitung
	Status        TaskStatus `json:"status"`
	PIC           string     `json:"pic"`
	VendorID      *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName    *string    `json:"vendor_name,omitempty"` // JOIN planner_vendors
	VendorWA      *string    `json:"vendor_wa,omitempty"`   // planner_vendors.formatted_wa_number (62…); FE membentuk tautan wa.me
	SortOrder     int        `json:"sort_order"`
	Notes         string     `json:"notes"`
	TemplateKey   *string    `json:"template_key,omitempty"`
	Origin        Origin     `json:"origin"`
	Overdue       bool       `json:"overdue"` // status <> selesai dan end_date < hari ini (zona proyek); dihitung server
}

// Summary dihitung dari SELURUH tugas proyek, bukan dari hasil saringan query.
// Nama bidang sama dengan TimelineSummary milik widget dashboard (summary.go), ditambah NotStarted.
type TaskSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	NotStarted int `json:"not_started"`
	Overdue    int `json:"overdue"`
}

type TaskList struct {
	Tasks   []Task      `json:"tasks"` // selalu array
	Summary TaskSummary `json:"summary"`
}

type CreateTaskRequest struct {
	Title         string     `json:"title"`           // dipangkas; 1–255 karakter
	SubEventScope string     `json:"sub_event_scope"` // kosong = "all"; harus ada di master sesi (400 invalid_scope)
	Category      string     `json:"category"`        // kosong = "Umum"
	PhaseOrder    int        `json:"phase_order"`     // 0 = ikut tugas dengan sort_order terbesar; kosong semua = fase 1 (bab 4.2)
	PhaseName     string     `json:"phase_name"`
	StartDate     string     `json:"start_date"` // YYYY-MM-DD
	EndDate       string     `json:"end_date"`   // >= start_date, selain itu 400 invalid_date
	PIC           string     `json:"pic"`
	VendorID      *uuid.UUID `json:"vendor_id"` // harus vendor proyek ini (400 invalid_input: vendor_id)
	Notes         string     `json:"notes"`
}

// PATCH: hanya field yang dikirim yang berubah. Pointer + ClearVendor karena JSON null dan "tidak dikirim" harus dibedakan.
// Mengubah PhaseOrder TIDAK menyalin PhaseName (berbeda dari POST): kirim keduanya (bab 8, butir 20).
type UpdateTaskRequest struct {
	Title         *string     `json:"title"`
	SubEventScope *string     `json:"sub_event_scope"`
	Category      *string     `json:"category"`
	PhaseOrder    *int        `json:"phase_order"`
	PhaseName     *string     `json:"phase_name"`
	StartDate     *string     `json:"start_date"`
	EndDate       *string     `json:"end_date"`
	PIC           *string     `json:"pic"`
	Notes         *string     `json:"notes"`
	Status        *TaskStatus `json:"status"`
	VendorID      *uuid.UUID  `json:"vendor_id"`
	ClearVendor   bool        `json:"clear_vendor"` // bersama vendor_id -> 400 invalid_input
}

type UpdateTaskStatusRequest struct {
	Status TaskStatus `json:"status"`
}

type ReorderRequest struct {
	IDs []uuid.UUID `json:"ids"` // tugas atau dokumen; harus memuat SEMUA id proyek tepat sekali
}

type ShiftTasksRequest struct {
	Days int `json:"days"` // negatif = mundur; bukan 0; |days| <= 1826
}

type ShiftTasksResponse struct {
	Shifted  int      `json:"shifted"`
	Warnings []Notice `json:"warnings"` // tasks_overdue_after_shift
}

// Preset berasal dari templates_available.timeline_presets proyek (wedding_pdf_44 | wedding_essential_22 | general);
// preset yang tidak dimiliki jenis acara -> 400 invalid_template.
type GenerateTimelineRequest struct {
	Preset              string `json:"preset"`                // wajib (400 invalid_input: preset: required)
	IncludeAdicaraTasks bool   `json:"include_adicara_tasks"` // tugas wedding.adicara_* (§5.2); diabaikan untuk preset general
	Mode                string `json:"mode"`                  // append (bila kosong) | replace_template | replace_all
	Confirm             bool   `json:"confirm"`               // aturan di bab 4.1
}

// Dipakai juga oleh POST /admin-docs/apply-template. Created/Skipped sama dengan domain.ApplyResult (Menu 01).
type GenerateResult struct {
	Created  int      `json:"created"`
	Skipped  int      `json:"skipped"`
	Deleted  int      `json:"deleted"`
	Warnings []Notice `json:"warnings"` // selalu array: lead_time_compressed | lead_time_too_short | scope_fallback | budget_link_lost
}

// ---------- Dokumen ----------

type AdminDoc struct {
	ID             uuid.UUID  `json:"id"`
	DocumentName   string     `json:"document_name"`
	SubjectRole    string     `json:"subject_role"` // kunci admin_subjects dari registry jenis acara
	QuantityDetail string     `json:"quantity_detail"`
	NominalCost    int64      `json:"nominal_cost"` // rupiah utuh
	Status         DocStatus  `json:"status"`
	SortOrder      int        `json:"sort_order"`
	Notes          string     `json:"notes"`
	TemplateKey    *string    `json:"template_key,omitempty"`
	Origin         Origin     `json:"origin"`
	BudgetItemID   *uuid.UUID `json:"budget_item_id,omitempty"` // pos anggaran yang menunjuk dokumen ini; dihilangkan = belum tersinkron
}

type AdminDocList struct {
	Docs    []AdminDoc `json:"docs"` // selalu array
	Summary struct {
		Total     int `json:"total"`
		Completed int `json:"completed"`
	} `json:"summary"` // seluruh dokumen proyek, bukan hasil saringan
}

type CreateAdminDocRequest struct {
	DocumentName   string `json:"document_name"`
	SubjectRole    string `json:"subject_role"` // kosong = "bersama" (pernikahan) atau "umum"; harus ada di admin_subjects (400 invalid_input: subject_role)
	QuantityDetail string `json:"quantity_detail"`
	NominalCost    int64  `json:"nominal_cost"` // 0…10¹³
	Notes          string `json:"notes"`
}

type UpdateAdminDocRequest struct { // PATCH, semua pointer
	DocumentName   *string `json:"document_name"`
	SubjectRole    *string `json:"subject_role"`
	QuantityDetail *string `json:"quantity_detail"`
	NominalCost    *int64  `json:"nominal_cost"`
	Notes          *string `json:"notes"`
}

type UpdateAdminDocStatusRequest struct {
	Status DocStatus `json:"status"`
}

type ApplyDocsRequest struct {
	Mode    string `json:"mode"` // append (bila kosong) | replace_template
	Confirm bool   `json:"confirm"`
}

type SyncBudgetRequest struct {
	SubEventScope *string `json:"sub_event_scope"` // opsional; bab 4.5
}

type SyncBudgetResponse struct {
	BudgetItemID uuid.UUID `json:"budget_item_id"`
	Created      bool      `json:"created"`  // pos baru dibuat
	Linked       bool      `json:"linked"`   // pos yang sudah ada baru ditautkan
	Warnings     []Notice  `json:"warnings"` // estimate_differs
}
```

Registry (`planner/domain/eventtype.go`, sudah ada dari Menu 01) **bertambah tiga bidang** milik modul ini; nilainya per jenis ada di [TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02):

```go
type Option struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Guide struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"` // tepat 4 langkah
}

// EventTypeSpec bertambah:
//	AdminSubjects []Option `json:"admin_subjects"` // filter subjek dan dropdown peran dokumen
//	AdminGuide    Guide    `json:"admin_guide"`    // callout panduan
//	AdminLabel    string   `json:"-"`              // judul widget Legalitas; dipakai provider admin_docs, tidak dikirim ke klien
```

`GET /event-types` lalu mengirim `admin_subjects` dan `admin_guide` per jenis (menambah [01 BE §3.4](../01_DASHBOARD/BE.md#34-bentuk-respons-sebagaimana-diimplementasikan) tanpa mengubah bidang yang ada). Ini satu-satunya sumber peran dan panduan: respons `GET /admin-docs` tidak membawa salinannya ([kontrak bab 3](../00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data)).

---

## 3. Endpoint

Semua di bawah `BearerAuth` + `ProjectGuard` + `WriteGuard` (`/v1/planner/projects/:id/...`; rantai middleware dan pemasangan di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)). Respons `{"data": ...}`, galat `{"error": {code, message}}`. Id anak di URL (`:task_id`, `:doc_id`) harus UUID kanonik 36 karakter, selain itu `404 not_found`. Rute statis (`/tasks/reorder`, `/tasks/shift`, `/tasks/generate-defaults`, `/tasks/export-spreadsheet`, `/admin-docs/reorder`, `/admin-docs/apply-template`) berdampingan dengan `/tasks/:task_id` dan `/admin-docs/:doc_id`; Gin menyelesaikan segmen statis lebih dulu (terbukti untuk `/projects/summary` di Menu 01). Penulisan di proyek `archived` ditolak `409 project_read_only`; `GET` (termasuk ekspor) tetap boleh.

### 3.1 Jadwal & Milestone

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/tasks?status=&category=&scope=&phase=` | Daftar tugas urut `sort_order` + `summary`. `status` ∈ `belum\|proses\|selesai`; `category` persis; `scope` = `scope_key` sesi (**`scope=all` hanya tugas Umum**; tanpa parameter = semua sesi); `phase` = `phase_order` (bilangan bulat). Nilai tak valid → `400 invalid_input: <param>`; `scope` yang berbentuk benar tetapi tak dikenal memberi daftar kosong. `summary` selalu seluruh proyek (dibaca dengan kueri terpisah dari daftar, bukan satu snapshot). |
| `POST` | `/tasks` | `CreateTaskRequest` → `201`. `409 task_limit_reached` di atas batas (§3.4). |
| `PATCH` | `/tasks/:task_id` | `UpdateTaskRequest`. |
| `PATCH` | `/tasks/:task_id/status` | Toggle cepat (checkbox, Kanban); mengisi `status` saja. |
| `DELETE` | `/tasks/:task_id` | `204`. |
| `POST` | `/tasks/reorder` | `ReorderRequest`; harus memuat **semua** id proyek tepat sekali (`400 invalid_input: ids` bila ada yang asing, ganda, atau hilang). Menulis `sort_order = 1…n` menurut urutan daftar (§4.2). |
| `POST` | `/tasks/shift` | `ShiftTasksRequest`: geser tugas template yang belum selesai sebanyak `days` (aturan pemilihan tugas: [01 BE §4.3](../01_DASHBOARD/BE.md#43-ubah-tanggal-acara-dan-geser-jadwal)). Aksi manual dari layar Timeline, **tidak idempoten** (setiap panggilan menggeser lagi); perubahan tanggal Hari-H memakai `PATCH /projects/:id` dengan `shift_tasks` yang atomik, bukan endpoint ini. |
| `POST` | `/tasks/generate-defaults` | `GenerateTimelineRequest` → `GenerateResult`. |
| `GET` | `/tasks/export-spreadsheet` | `.xlsx` bergaya dokumen acuan (§4.4). |

### 3.2 Dokumen & Legalitas

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/admin-docs?subject=&status=` | Daftar berkas urut `sort_order` + `summary {total, completed}`. `subject` ∈ `admin_subjects` jenis acara; `status` ∈ `belum\|selesai`. Panduan dan daftar subjek **tidak** ada di sini, melainkan di registry (§2). |
| `POST` | `/admin-docs` | `CreateAdminDocRequest` → `201`. `409 admin_doc_limit_reached` di atas batas. |
| `PATCH` | `/admin-docs/:doc_id` | `UpdateAdminDocRequest`. |
| `PATCH` | `/admin-docs/:doc_id/status` | Toggle `belum` ↔ `selesai`. |
| `DELETE` | `/admin-docs/:doc_id` | `204`. Pos anggaran yang menunjuknya kehilangan tautan (`ON DELETE SET NULL`), angkanya tetap (terbukti). |
| `POST` | `/admin-docs/reorder` | `ReorderRequest`. |
| `POST` | `/admin-docs/apply-template` | `ApplyDocsRequest`: berkas standar jenis acara (pernikahan: 15 KUA; lainnya: perizinan, [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)). Jenis tanpa template dokumen (`other`) → `400 invalid_template`. |
| `POST` | `/admin-docs/:doc_id/sync-budget` | Salin biaya berkas ke pos anggaran (§4.5). |

### 3.3 Bentuk respons

| Endpoint | `data` |
|---|---|
| `GET /tasks` | `TaskList` `{tasks[], summary{total,completed,in_progress,not_started,overdue}}` |
| `POST /tasks` | `Task` (`201`) |
| `PATCH /tasks/:id`, `PATCH /tasks/:id/status` | `Task` |
| `DELETE /tasks/:id`, `DELETE /admin-docs/:id` | `204` |
| `POST /tasks/reorder` | `TaskList` urutan baru |
| `POST /tasks/shift` | `{shifted, warnings[]}` |
| `POST /tasks/generate-defaults`, `POST /admin-docs/apply-template` | `{created, skipped, deleted, warnings[]}` |
| `GET /tasks/export-spreadsheet` | berkas (bukan JSON): `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="jadwal-<slug>-<tanggal>.xlsx"` (slug ASCII, [kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)) |
| `GET /admin-docs`, `POST /admin-docs/reorder` | `AdminDocList` `{docs[], summary{total,completed}}` |
| `POST /admin-docs` | `AdminDoc` (`201`) |
| `PATCH /admin-docs/:id`, `PATCH /admin-docs/:id/status` | `AdminDoc` |
| `POST /admin-docs/:id/sync-budget` | `{budget_item_id, created, linked, warnings[]}`; `201` bila pos baru dibuat, `200` selainnya |

Kunci `vendor_id`, `vendor_name`, `vendor_wa`, `template_key`, `budget_item_id` **dihilangkan** bila kosong (`omitempty`); daftar (`tasks`, `docs`, `warnings`) selalu array, bukan `null`.

### 3.4 Validasi dan batas

Semua galat berbentuk `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)). Angka batas adalah usulan (bab 8, butir 3) dan dijaga usecase.

| Field | Aturan | Galat |
|---|---|---|
| `title`, `document_name` | dipangkas, 1–255 karakter (rune), tanpa karakter kontrol | `400 invalid_input` |
| `category`, `phase_name`, `pic`, `quantity_detail` | ≤ 60 / 100 / 100 / 100 karakter, tanpa karakter kontrol | `400 invalid_input` |
| `notes` | ≤ 2.000 karakter; `\n` boleh (`\r\n` dinormalkan menjadi `\n`), karakter kontrol lain ditolak. Perlu pembantu baru `cleanNote`: `cleanText` Menu 01 memakai `unicode.IsControl` sehingga menolak `\n` juga | `400 invalid_input` |
| `phase_order` | 1–20 | `400 invalid_input` |
| `sub_event_scope` | harus ada di master sesi proyek (diperiksa sebelum menulis; FK-nya ditunda) | `400 invalid_scope` |
| `start_date`, `end_date` | `YYYY-MM-DD` valid; `end ≥ start`; dalam jendela `Hari-H − 1095 hari` s/d `akhir acara + 365 hari` | `400 invalid_date` |
| `vendor_id` | vendor proyek ini | `400 invalid_input: vendor_id` |
| `status` | enum tugas atau dokumen | `400 invalid_input: status` |
| `subject_role` | ∈ `admin_subjects` jenis acara | `400 invalid_input: subject_role` |
| `nominal_cost` | 0…10¹³ | `400 invalid_input: nominal_cost` |
| `ids` (reorder) | tepat semua id proyek, tanpa ganda | `400 invalid_input: ids` |
| `days` (shift) | bukan 0, \|days\| ≤ 1826; setiap tugas yang digeser tetap dalam jendela tanggal | `400 invalid_input: days` / `invalid_date` |
| `confirm` | aturan di §4.1 dan §4.7 | `400 invalid_input: confirm` |
| Jumlah | ≤ 500 tugas dan ≤ 200 dokumen per proyek; berlaku juga untuk `generate-defaults` dan `apply-template` (jumlah setelah penyisipan, bukan hanya sebelum). Diperiksa di bawah kunci proyek (§4.2) | `409 task_limit_reached`, `409 admin_doc_limit_reached`; semua-atau-tidak, tidak ada baris yang tertulis |
| Frekuensi | ekspor 10/menit/pengguna ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)) | `429 rate_limited` |

Batas 500 dan 200 bukan beban kueri: pada 500 tugas (catatan 200 karakter) dan 200 dokumen, `GET /tasks` berukuran **370 kB** JSON mentah dan `GET /admin-docs` **114 kB**, ringkasan dan daftar berjalan 0,2 ms dan 0,15 ms (`EXPLAIN ANALYZE`, PostgreSQL 16.2). Kasus terburuk dengan setiap catatan 2.000 karakter sekitar 1,3 MB (dihitung, bukan diukur). Angka itu dipilih agar layar tanpa paginasi (tabel, Kanban, Gantt) tetap bisa digambar dan agar ekspor tidak membengkak.

Alasan baku (Inggris, setelah `<field>: `) yang dipetakan FE: memakai kosakata Menu 01 (`required`, `must not be empty`, `must be at most N characters`, `must not contain control characters`, `must be a valid date in YYYY-MM-DD format`, `must be an integer`, `must be between A and B`, `must be at least N`), ditambah `must not be before start_date` (pada `end_date`), `must be within the project schedule window` (pada `start_date`/`end_date`), `must be one of the event type subjects` (pada `subject_role`), `must be a vendor of this project` (pada `vendor_id`), `must list every id exactly once` (pada `ids`), `must not be zero` dan `must be at most 1826 in absolute value` (pada `days`), dan `required` (pada `confirm` bila konfirmasi diwajibkan).

Hanya field yang dikirim pada `PATCH` yang divalidasi: mengubah status tugas yang tanggalnya di luar jendela (mis. setelah Hari-H dimajukan) tetap berhasil.

---

## 4. Logika Bisnis

### 4.1 Generator jadwal

Algoritma penskalaan offset (horizon baku per jenis, pemampatan saat waktu pendek, jepit ke hari ini) ada di [TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis) dan berlaku untuk **semua** preset. Kode acuannya (paket `plannertimeline/domain`; dijalankan, lima tes lulus, lihat [bab 6](#6-pengujian)):

```go
// Plan adalah penskalaan jadwal untuk satu proyek. Semua tanggal DATE = time.Time 00:00 UTC.
type Plan struct {
	EventDate time.Time
	Floor     time.Time // max(prep_start_date, hari ini di zona proyek): tidak ada tugas sebelum ini
	K         float64   // pengali offset
	Avail     int       // EventDate − Floor, dalam hari
	TooShort  bool      // Avail < 3: semua tugas dijepit ke [Floor, EventDate] (warning lead_time_too_short)
	Squeezed  bool      // Avail < leadTimeDays (warning lead_time_compressed)
}

// leadTimeDays = LeadTimeDays jenis acara (registry); horizon = R template (180 pernikahan, 60 general).
// ok=false bila EventDate < Floor: tidak ada jadwal untuk masa lalu (400 invalid_date: event_date).
func NewPlan(eventDate, prepStart, today time.Time, leadTimeDays, horizon int) (p Plan, ok bool) {
	floor := prepStart
	if today.After(floor) {
		floor = today
	}
	avail := int(eventDate.Sub(floor).Hours() / 24)
	if avail < 0 {
		return Plan{}, false
	}
	k := float64(leadTimeDays) / float64(horizon)
	if avail < leadTimeDays {
		k *= float64(avail) / float64(leadTimeDays)
	}
	return Plan{EventDate: eventDate, Floor: floor, K: k, Avail: avail, TooShort: avail < 3, Squeezed: avail < leadTimeDays}, true
}

// round membulatkan setengah naik (bukan pembulatan bankir): floor(x + 0,5).
func round(x float64) int { return int(math.Floor(x + 0.5)) }

// At mengubah offset template (H-startOffset s/d H-endOffset) menjadi tanggal.
func (p Plan) At(startOffset, endOffset int) (start, end time.Time) {
	day := func(n int) time.Time { return p.EventDate.AddDate(0, 0, -n) }
	start, end = day(round(float64(startOffset)*p.K)), day(round(float64(endOffset)*p.K))
	if p.TooShort {
		return p.Floor, p.EventDate
	}
	if start.Before(p.Floor) {
		start = p.Floor
	}
	if end.Before(start) {
		end = start
	}
	return start, end
}
```

"Hari ini" dan `Floor` dihitung di zona waktu proyek ([01 BE §4.2](../01_DASHBOARD/BE.md#42-hitung-mundur-dan-fase-acara)); `Plan` menerima tanggal yang sudah dihitung, jadi tidak memanggil `LoadLocation`. Nilai terhitung (dijalankan): Proyek Contoh (`Avail = 175`, `K = 175/180`) menghasilkan tugas 1 pada hari ini (4 Okt 2026), tugas 3 pada 8–10 Okt, tugas 9 pada 19–20 Okt, tugas 16 pada 2–3 Nov, **tugas 7 "Lamaran" pada 15–16 Okt**, dan tugas Adicara "Finalisasi daftar tamu" pada 12–13 Feb 2027. Pada `Avail = 175` dan 180 **durasi 44 tugas sama dengan PDF**; pada waktu yang lebih pendek tugas 2 hari (3, 4, 12) menjadi 1 hari karena dimampatkan (dihitung untuk `Avail` 120 sampai 10).

Preset dan isinya:

| Preset | Isi | `R` |
|---|---|:-:|
| `wedding_pdf_44` | 44 tugas §5.1 (+ 6 tugas §5.2 bila `include_adicara_tasks`) | 180 |
| `wedding_essential_22` | 22 tugas bertanda E di §5.1 (+ 6 tugas Adicara) | 180 |
| `general` | 15 tugas [TEMPLATE_EVENT bab 2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#2-template-general-15-tugas-r--60-untuk-jenis-non-pernikahan) + tugas khusus jenis (§5.5) | 60 |

Mode:

| Mode | Efek | `confirm` |
|---|---|---|
| `append` | Menyisipkan tugas template yang `template_key`-nya belum ada di proyek; tugas lain tidak disentuh. Idempoten: `INSERT … ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`, `created` = baris yang masuk, `skipped` = sisanya (dijalankan: pemanggilan kedua menyisipkan 0 baris). | tidak perlu |
| `replace_template` | Menghapus tugas yang punya `template_key` lalu menyisipkan template segar; tugas kustom dipertahankan. | wajib `true` bila ada tugas ber-`template_key` yang `status <> 'belum'` atau punya `vendor_id` (kemajuan akan hilang); selain itu `400 invalid_input: confirm` |
| `replace_all` | Menghapus **semua** tugas lalu menyisipkan template. | wajib `true` bila proyek sudah punya tugas |

`deleted` pada respons adalah jumlah tugas yang dihapus (0 untuk `append`). Tugas baru berstatus `belum` dan diberi `sort_order` berurutan mulai `max(sort_order) + 1` menurut urutan template (pernikahan: tugas 1…44, lalu tugas Adicara; jenis lain: kronologis, §5.5); pada proyek baru hasilnya 1…50 untuk pernikahan dan 1…18 untuk jenis lain, dan tugas kustom yang tersisa pada mode `replace_*` berada di depan tugas template baru. Tampilan per fase menyusun menurut `phase_order` lalu `sort_order`. Scope tiap tugas template dipetakan ke sesi proyek lewat `scope_key`; jika sesi itu sudah dihapus pengguna, tugas jatuh ke `all` dan respons memuat `scope_fallback`.

**Urutan dan kunci.** Di dalam satu transaksi: `LockProject` (§4.2) → hitung jumlah → hapus (hanya `replace_*`) → sisipkan. Batas 500 diperiksa terhadap jumlah **setelah** penyisipan (`sisa + created`); bila lebih, `409 task_limit_reached` dan tidak ada yang tertulis (rollback). Syarat `confirm` dihitung dengan kueri ini (dijalankan pada Proyek Contoh: **44**, jadi `replace_template` meminta `confirm`; setelah semua status dikembalikan ke `belum`: **0**, tidak meminta):

```sql
-- replace_template: $1 = project_id; confirm wajib bila hasilnya > 0
SELECT count(*) FROM planner_tasks
WHERE project_id = $1 AND template_key IS NOT NULL AND (status <> 'belum' OR vendor_id IS NOT NULL);
-- replace_all: confirm wajib bila SELECT count(*) FROM planner_tasks WHERE project_id = $1 hasilnya > 0
```

Warning (objek `Notice`; `message` Inggris, `count` = jumlah tugas terkait):

| Kode | Kapan |
|---|---|
| `lead_time_compressed` | `Plan.Squeezed` dan bukan `TooShort`: waktu persiapan lebih pendek dari `LeadTimeDays` jenis acara |
| `lead_time_too_short` | `Plan.TooShort`: kurang dari 3 hari (menggantikan `lead_time_compressed`, tidak muncul bersamanya) |
| `scope_fallback` | sesi template sudah dihapus pengguna; tugas dipindah ke `all` |

`count` = jumlah tugas yang **dibuat** oleh panggilan itu (untuk `scope_fallback`: yang jatuh ke `all`); tanpa tugas baru (`created = 0`) tidak ada peringatan tanggal.

### 4.2 Kustomisasi bebas

Tidak ada tugas baca-saja. Tugas hasil template berperilaku sama dengan tugas kustom (ubah, hapus, geser, urut ulang); `template_key` hanya dipakai generator dan penggeseran.

- **Tambah tugas** (`POST /tasks`): `sort_order = max + 1`. `phase_order = 0` berarti mengikuti `(phase_order, phase_name)` tugas dengan `sort_order` terbesar; proyek tanpa tugas memakai fase 1 tanpa nama. `phase_order > 0` tanpa `phase_name` menyalin nama dari tugas lain pada fase itu bila ada. Pengecekan batas jumlah (500) dan pembuatan berjalan di bawah kunci proyek (`LockProject` Menu 01: `SELECT … FOR UPDATE` pada baris `planner_projects`) supaya klik ganda tidak melewati batas. **Kunci yang sama diambil di awal transaksi** oleh `POST /tasks`, `POST /admin-docs`, `PATCH` tugas dan dokumen, `generate-defaults`, `apply-template`, `reorder`, dan `shift`, sehingga hitungan jumlah, `max(sort_order) + 1`, dan pemeriksaan "daftar lengkap" tidak berbalapan. `PATCH` tugas atau dokumen (baca, gabung, tulis) juga mengambil kunci itu supaya dua tab yang mengubah kolom berbeda tidak saling menimpa; `PATCH …/status`, `DELETE`, dan sinkron biaya (yang mengunci baris dokumen) tidak membutuhkannya. Dijalankan dengan dua sesi PostgreSQL yang berbalapan pada proyek berisi 50 tugas dengan batas 51: **dengan kunci** satu permintaan membuat tugas dan yang lain menjawab `task_limit_reached` (51 tugas); **tanpa kunci** (kontrol) keduanya lolos (52 tugas). Kunci tidak membuat buntu dengan pemeriksaan FK `KEY SHARE` dari `INSERT` di transaksi yang sama.
- **Urut ulang**: pola yang sama dengan `ReorderSessions` Menu 01 dan sudah teruji di sana:
  ```sql
  UPDATE planner_tasks t SET sort_order = v.ord, updated_at = now()   -- $3 = 1…n menurut urutan daftar
  FROM unnest($2::text[]::uuid[], $3::int[]) AS v(id, ord)
  WHERE t.project_id = $1 AND t.id = v.id;
  ```
  Usecase membandingkan jumlah baris yang berubah dan `count(*)` tugas proyek dengan `len(ids)` dalam transaksi yang sama; beda → rollback `400 invalid_input: ids`. Berlaku sama untuk `planner_admin_docs`. Bentuk ini dijalankan pada audit terhadap 50 tugas, dengan pernyataan persis seperti di atas (dibungkus CTE hanya agar jumlah baris terlihat):

| Masukan | Baris berubah | Hasil |
|---|:-:|---|
| daftar lengkap, urutan dibalik | 50 | 50 nilai `sort_order` berbeda, 1…50 |
| kurang satu id | 49 | ≠ 50 → rollback `400 invalid_input: ids` |
| satu id diganti id tugas **proyek lain** | 49 | tugas proyek lain tidak tersentuh (`sort_order` tetap 77); ≠ 50 → ditolak |
| satu id ganda (panjang tetap 50) | 49 | ≠ 50 → ditolak (Go juga menolak id ganda lebih dulu) |
| teks bukan UUID | galat `22P02` | tidak terjadi lewat API: JSON `uuid.UUID` gagal di-decode lebih dulu → `400 invalid_json` |
- **Status**: `PATCH …/status` dan `PATCH` dengan `status` setara; tidak ada aturan peralihan (boleh `selesai` → `belum`).
- **Overdue** selalu dihitung server: `status <> 'selesai' AND end_date < (waktu_sekarang AT TIME ZONE zona_proyek)::date`; waktu sekarang menjadi parameter seperti kueri Hub ([01 BE §5.3](../01_DASHBOARD/BE.md#53-query-hub)) agar batas tengah malam bisa diuji.

### 4.3 Hubungan vendor

`vendor_id` pada tugas memungkinkan milestone seperti "Visit Venue Pernikahan" tertaut ke vendor di Menu 04; tampilan menampilkan nama vendor dan tombol WA (dari `vendor_wa`, nomor yang sudah dinormalisasi Menu 04). Vendor tidak boleh dari proyek lain: usecase memeriksa `SELECT 1 FROM planner_vendors WHERE id = $1 AND project_id = $2` lebih dulu (`400 invalid_input: vendor_id`), dan FK komposit menjadi pengaman kedua (galat `23503` pada `planner_tasks_vendor_id_project_id_fkey` dipetakan ke galat yang sama). Menghapus vendor di Menu 04 mengosongkan `vendor_id` (`ON DELETE SET NULL (vendor_id)`), tugas tetap ada.

### 4.4 Ekspor `.xlsx`

Pustaka `excelize` **v2.11.0** (terbaru di proxy Go per 2026-10-21; dependensi baru, disetujui pemilik 2026-10-21, [kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)). `go mod tidy` pada modul percobaan menambah satu modul langsung dan lima tidak langsung (`richardlehane/mscfb`, `richardlehane/msoleps`, `tiendc/go-deepcopy`, `xuri/efp`, `xuri/nfp`); `golang.org/x/crypto`, `x/net`, dan `x/text` sudah lebih baru di backend (v0.56.0, v0.57.0, v0.41.0) sehingga tidak turun, dan syarat Go 1.25 terpenuhi (backend 1.26). Ekspor memuat **semua** tugas proyek (bukan hasil saringan) urut `sort_order`, dalam satu sheet bernama `Timeline`:

| Kolom | Isi | Format sel |
|---|---|---|
| A No | nomor baris 1…n (bukan nilai `sort_order`) | angka |
| B Tugas | `title` | teks lewat `safeCell` |
| C Rangkaian Acara | nama sesi untuk `sub_event_scope` | teks lewat `safeCell` |
| D Mulai, E Selesai | `start_date`, `end_date` | tanggal sungguhan (bukan teks), `dd mmm yyyy` |
| F Durasi (hari) | `duration_days` | angka |
| G Status | "Belum", "Proses", "Selesai" | teks |
| H … | matriks Gantt mingguan | sel berwarna tanpa isi |

Matriks Gantt memakai aturan yang sama dengan `buildGantt` FE ([FE bab 5](./FE.md#5-library-libplannertimelinets)): kolom pertama adalah Senin pada pekan `min(start_date)`, kolom terakhir Senin pada pekan `max(end_date)`; header tiap kolom adalah tanggal Senin (`dd/mm`, teks tegak lurus, lebar 3). Sel pekan terisi bila rentang **inklusif** `[mulai, selesai]` beririsan dengan Senin–Minggu pekan itu; warna mengikuti status: selesai `#526E4D`, proses `#97B394`, belum `#D1DFCE` (sage-600/400/200 di `frontend/src/styles/tokens.css`). Baris 1 berlatar `#526E4D` dengan teks putih tebal; kolom A–B dan baris 1 dibekukan.

```go
// safeCell: kontrak bab 2 butir 5. Hanya teks dari pengguna yang melewatinya (judul, nama sesi).
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@", rune(s[0])) {
		return "'" + s
	}
	return s
}

// mondayOf: Senin pada pekan ISO tanggal d (00:00 UTC).
func mondayOf(d time.Time) time.Time { return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7)) }

// sel (tugas t, pekan dengan Senin mon) terisi bila:
filled := !t.Start.After(mon.AddDate(0, 0, 6)) && !t.End.Before(mon)
```

Teks pengguna ditulis dengan `SetCellStr`. **Dibangun dan dibaca kembali** (prototipe di modul Go sementara di luar repo, 50 tugas Proyek Contoh; `excelize` v2.9.0 dan v2.11.0 menghasilkan hal yang sama):
- 51 baris, header A–G sesuai tabel; empat judul berbahaya (`=cmd|' /C calc'!A0`, `+SUM(1+1)`, `-2+3`, `@SUM(A1)`) keluar sebagai teks berawalan `'`, bertipe string bersama, `GetCellFormula` kosong, dan **tidak ada elemen `<f>` di worksheet mana pun**; judul biasa tidak diberi awalan.
- Sel D2 adalah tanggal sungguhan (nomor seri 46299 = 4 Okt 2026).
- Header minggu `28/09` sampai `22/03` = 26 kolom (sama dengan 26 minggu pada vektor uji FE); isi 50 × 26 sel dicocokkan dengan perhitungan hari demi hari (algoritma berbeda dari irisan interval di atas): nol selisih, 61 sel terisi. Tugas 1 (Minggu 4 Okt – Senin 5 Okt) mengisi dua kolom.
- Berkas 9,3 kB.
- **`excelize` tidak pernah menulis string berawalan `=` sebagai formula** (`SetCellValue` maupun `SetCellStr` menyimpan string bersama, `t="s"`), jadi sel bertipe teks dan tidak dievaluasi saat dibuka. Ini dibaca ulang dengan parser XML pustaka standar Python (bukan `excelize`): sel `t="s"`, tanpa `<f>`, isi `'=cmd|…` apa adanya; **berkas belum dibuka di Excel, Numbers, atau LibreOffice**, jadi tampilan akhirnya disimpulkan dari isi berkas, bukan dilihat. Konsekuensinya awalan `'` yang diwajibkan kontrak ditulis sebagai **karakter biasa** dan tampak di sel (v2.11 tidak punya atribut `quotePrefix` pada `Style`). Aturan kontrak tetap diikuti sebagai pertahanan berlapis (sel disalin ke CSV atau diedit lalu dievaluasi ulang oleh Excel); akibatnya hanya terlihat pada judul yang memang berawalan `=`, `+`, `-`, `@` (bab 8, butir 12).

Nama berkas: `jadwal-<slug>-<tanggal>.xlsx`; `<slug>` = judul proyek huruf kecil, selain `[a-z0-9]` menjadi `-`, berulang dilebur, tepi dibuang, maksimal 40 karakter, kosong → `proyek`; `<tanggal>` = tanggal ekspor di zona proyek. Backend belum punya pembantu slug; Menu 03–06 butuh yang sama, jadi letakkan di `internal/ginutil/`. Pembatas 10 ekspor/menit/pengguna: `userLimiter` Menu 01 tidak diekspor, jadi modul ini menyalin pola `ratelimit.go` (kontrak bab 1.4) dan ekstraksi ke paket bersama ditunda (bab 8, butir 13). Header respons di §3.3.

### 4.5 Sinkron biaya dokumen ke anggaran

`POST /admin-docs/:doc_id/sync-budget` (body opsional `{"sub_event_scope": "akad"}`; default dari pemetaan `template_key`: `wedding.pendaftaran_kua` → sesi `akad`, `item_key biaya_kua`; selain itu sesi `all` tanpa `item_key`). Syarat `nominal_cost > 0` (selain itu `400 invalid_input: nominal_cost`); `sub_event_scope` yang diminta harus ada di master sesi (`400 invalid_scope`). Idempoten dan **tidak menimpa angka pengguna**.

**Batas modul.** Endpoint ini milik Menu 02 tetapi menulis ke `planner_budget_items` milik Menu 03. Selaras dengan pengecualian SQL yang sudah ada di Menu 01 ([kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)), ia memakai SQL langsung pada tabel itu (migrasi 008 sudah dirilis, jadi bisa dibangun dan diuji sekarang), dan pengecualian ini dicatat di kontrak. Alternatifnya (antarmuka Go yang diimplementasikan Menu 03) tidak dipilih karena Menu 03 belum ada (bab 8, butir 5).

Satu transaksi; baris dokumen dikunci `FOR UPDATE` sehingga klik ganda diserialkan (tabel `planner_budget_items` tidak punya indeks unik pada `admin_doc_id`, jadi tanpa kunci ini dua permintaan bisa membuat dua pos):

```sql
-- 0. kunci dokumen dan baca biayanya: $1 = project_id, $2 = doc_id
SELECT document_name, nominal_cost, template_key FROM planner_admin_docs WHERE id = $2 AND project_id = $1 FOR UPDATE;

-- 1. sudah tertaut?  -> 200 {created:false, linked:false, budget_item_id}, tanpa perubahan
SELECT id FROM planner_budget_items WHERE project_id = $1 AND admin_doc_id = $2 LIMIT 1;

-- 2. pos dengan (scope, item_key) cocok yang belum bertaut dokumen: $3 = scope, $4 = item_key, $5 = nominal_cost
SELECT id, initial_estimate FROM planner_budget_items
WHERE project_id = $1 AND sub_event_scope = $3 AND item_key = $4 AND admin_doc_id IS NULL FOR UPDATE;
UPDATE planner_budget_items
SET admin_doc_id = $2, initial_estimate = CASE WHEN initial_estimate = 0 THEN $5 ELSE initial_estimate END, updated_at = now()
WHERE id = $6;          -- $6 = id hasil SELECT; -> 200 {created:false, linked:true}

-- 3. selain itu pos baru (item_key NULL, jadi tidak bisa bentrok dengan indeks unik): $7 = uuid baru, $8 = nama dokumen
INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, admin_doc_id, initial_estimate, sort_order)
VALUES ($7, $1, $3, 'Legalitas', left($8, 150), 'admin_doc', $2, $5,
        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_budget_items WHERE project_id = $1));   -- -> 201 {created:true}
```

Hasil yang dijalankan (delapan kasus, database sementara): (A) pos `biaya_kua` berestimasi 0 tertaut dan terisi 600.000; (B) panggilan kedua tidak mengubah apa pun dan tetap satu pos; (C) "Materai 10.000" tanpa pemetaan menghasilkan pos baru kategori "Legalitas" sumber `admin_doc` di sesi `all`; (D) pos berestimasi 500.000 vs biaya 600.000 tertaut tanpa mengubah angka dan memunculkan `estimate_differs`; (E) nama 255 karakter terpotong menjadi 150 (tanpa `left()`, `INSERT` gagal `23514`); (F) menghapus dokumen mengosongkan `admin_doc_id`, pos dan angkanya tetap; (G) pos ber-`item_key` yang sudah bertaut ke dokumen lain tidak ditimpa, dokumen kedua mendapat pos baru; (H) biaya 0 ditolak.

**Klik ganda, dua sesi berbalapan** (dijalankan; sesi A memeriksa tautan lalu menunggu 1,5 detik sebelum menulis, sesi B mulai 0,4 detik kemudian dan menulis langsung): **dengan** `FOR UPDATE` pada baris dokumen, B menunggu A selesai, melihat tautan A, dan menjawab "sudah tertaut": **satu** pos untuk dokumen itu. **Tanpa** kunci (kontrol), keduanya menulis: **dua** pos. Karena itu kunci ini bagian dari kontrak, bukan optimasi.

Warning `estimate_differs` (`count` = 1) bila pos yang ditautkan sudah berestimasi > 0 dan berbeda dari `nominal_cost`; angka tidak diubah. **Sinkron tidak berlangsung terus-menerus**: mengubah `nominal_cost` dokumen setelah tersinkron tidak mengubah pos; tombol berubah menjadi badge "✓ Di anggaran" dan angka diubah di Menu 03.

Status "tersinkron" di UI = ada baris `planner_budget_items` dengan `admin_doc_id` = dokumen itu (`AdminDoc.budget_item_id`, diisi `LEFT JOIN` saat membaca).

### 4.6 Sambungan ke Menu 01

`plannertimeline.Service` memenuhi tiga antarmuka dari `planner/domain` (`repository.go`, kode yang sudah ada). Satu modul memiliki **dua kunci** (`timeline` dan `admin_docs`) padahal tiap antarmuka punya satu `Key()`, jadi masing-masing diwakili tipe kecil yang membungkus `Service`:

| Antarmuka | Kunci | Isi |
|---|---|---|
| `SummaryProvider` | `timeline` | `domain.TimelineSummary` (`summary.go`): `total, completed, in_progress, overdue, urgent[≤3]` |
| `SummaryProvider` | `admin_docs` | `domain.AdminDocSummary`: `label, completed, total, pending[≤5]` |
| `TemplateApplier` | `timeline` | `Apply(ctx, projectID, ApplyRequest)` memakai `req.Project`, `req.TimelinePreset`, `req.IncludeAdicaraTasks`; `req.Mode` selalu `append` dari wizard |
| `TemplateApplier` | `admin_docs` | dokumen standar jenis acara; mengabaikan preset |
| `SampleOverlay` | `timeline` dan `admin_docs` | status PDF dan tanggal sesi Lamaran (di bawah) |

Pemasangan di `internal/app/app.go` (menggantikan tanda di komentar yang sudah ada; registry adalah instance yang sama dengan yang diberikan ke `plannerusecase.New`, karena `LeadTimeDays` dan `admin_subjects` dibaca darinya):

```go
tl := plannertimeline.New(repo, txManager, registry /* dst. */)
timelinehttp.RegisterRoutes(ginEngine, idn, plannerSvc, tl)
plannerSvc.RegisterSummary(tl.TimelineProvider(), tl.AdminDocsProvider() /* , finance, … */)
plannerSvc.RegisterTemplateAppliers(tl.TimelineApplier(), tl.AdminDocsApplier() /* , … */)
plannerSvc.RegisterSampleOverlays(tl.TimelineOverlay(), tl.AdminDocsOverlay() /* , … */)
```

Kunci terdaftar dua kali atau tak dikenal membuat start gagal (panic), sengaja ([dokumen implementasi Menu 01](../../../backend/planner/2026-10-21-planner-menu-01.md#menyambungkan-modul-lain)). Applier berjalan berurutan `timeline → admin_docs → …` di dalam transaksi pembuatan proyek. `ApplyMode` di `planner/domain` hanya punya `append` (cukup untuk wizard); mode `replace_template` dan `replace_all` hanya lewat endpoint modul ini dengan tipe request sendiri, jadi `planner/domain` tidak perlu berubah. Setelah terdaftar, `GET /event-types` otomatis mengisi `templates_available.timeline_presets`, `include_adicara_tasks`, dan `admin_docs` (`other` tidak punya template dokumen: tetap `false`).

**Kueri provider.** Waktu sekarang menjadi parameter (`$3`, `timestamptz` dari usecase) seperti kueri Hub, bukan `now()`; kueri `timeline` dan `admin_docs` di [01 BE §5.2](../01_DASHBOARD/BE.md#52-query-per-provider) adalah bentuk awalnya. Dijalankan terhadap 50 tugas dan 15 berkas Proyek Contoh: pada 4 Okt 2026 hasilnya `total 50, completed 15, in_progress 29, overdue 0, urgent []`; pada 4 Nov 2026 `overdue 1`; pada 1 Nov 2026 `urgent` memuat tugas 16 (`days_left 2`), 17 (4), 18 (6); berkas `completed 9, total 15` dengan `pending` lima nama pertama menurut `sort_order`.

```sql
-- timeline: $1 = project_id, $2 = zona waktu proyek, $3 = waktu sekarang
SELECT count(*) AS total,
       count(*) FILTER (WHERE status = 'selesai') AS completed,
       count(*) FILTER (WHERE status = 'proses')  AS in_progress,
       count(*) FILTER (WHERE status = 'belum')   AS not_started,   -- hanya untuk GET /tasks
       count(*) FILTER (WHERE status <> 'selesai' AND end_date < ($3::timestamptz AT TIME ZONE $2)::date) AS overdue
FROM planner_tasks WHERE project_id = $1;

SELECT id, title, end_date, end_date - ($3::timestamptz AT TIME ZONE $2)::date AS days_left
FROM planner_tasks
WHERE project_id = $1 AND status <> 'selesai' AND end_date <= ($3::timestamptz AT TIME ZONE $2)::date + 7
ORDER BY end_date, sort_order LIMIT 3;
```

Kueri `admin_docs` tidak berubah dari 01 BE §5.2. `AdminDocSummary.Label` diisi `EventTypeSpec.AdminLabel` jenis proyek (pernikahan "Kelengkapan Berkas KUA", jenis lain "Kelengkapan Perizinan", [TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02)); ini menutup [01 FE butir 11](../01_DASHBOARD/FE.md#15-celah-dan-keputusan-yang-menunggu-pemilik-produk).

**Overlay Proyek Contoh** (hanya dari `POST /projects/sample` dan tes, dalam transaksi yang sama, setelah template diterapkan):
- Status tugas mengikuti PDF Hal. 3: tugas 1–15 `selesai`, 16–44 `proses`, enam tugas Adicara `belum` (15 / 29 / 6; PDF tidak punya status `belum`).
- Status berkas mengikuti PDF Hal. 11: `selesai` untuk nomor 1, 2, 3, 6, 7, 8, 10, 14, 15; `belum` untuk 4, 5, 9, 11, 12, 13.
- `UPDATE planner_event_sessions SET session_date = (SELECT start_date FROM planner_tasks WHERE project_id = $1 AND template_key = 'wedding.lamaran') WHERE project_id = $1 AND scope_key = 'lamaran'`. Pada contoh yang dibuat 4 Okt 2026 tanggalnya **15 Okt 2026**; angka 10 Okt yang pernah tertulis di FE Menu 01 memakai offset H-169 tanpa penskalaan dan bertentangan dengan §4.1.
- Tautan tugas → vendor contoh (mis. "Visit Venue Pernikahan" → Sasana Kriya Ballroom) dibuat oleh overlay Menu 04, pemilik data vendor, yang didaftarkan **setelah** overlay ini.

### 4.7 Dokumen: terapkan template

`POST /admin-docs/apply-template` memakai mode `append` (menyisipkan berkas yang `template_key`-nya belum ada, idempoten sama seperti tugas) atau `replace_template` (menghapus berkas ber-`template_key` lalu menyisipkan template segar; berkas kustom dipertahankan). `replace_template` wajib `confirm: true` bila ada berkas template yang `status = 'selesai'` atau sudah bertaut ke anggaran (statusnya kembali `belum`); respons memuat warning `budget_link_lost` (`count` = berkas bertaut yang dihapus) karena pos anggaran kehilangan tautannya, angkanya tetap. Berkas template baru berstatus `belum` dan, untuk pernikahan, membawa jumlah dan biaya saran §5.3 (kolom itu bukan dari PDF dan boleh diubah). Kunci proyek, batas 200 berkas setelah penyisipan, dan pemeriksaan `confirm` mengikuti aturan tugas di §4.1. Kueri syaratnya (dijalankan pada Proyek Contoh dengan berkas KUA tertaut ke anggaran 600.000: `finished 9, linked 1`, jadi `confirm` diminta):

```sql
-- $1 = project_id
SELECT count(*) FILTER (WHERE d.status = 'selesai') AS finished,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM planner_budget_items b WHERE b.admin_doc_id = d.id)) AS linked
FROM planner_admin_docs d WHERE d.project_id = $1 AND d.template_key IS NOT NULL;
-- confirm wajib bila finished + linked > 0; warning budget_link_lost.count = linked
```

Setelah `DELETE … WHERE template_key IS NOT NULL` pada contoh itu, pos anggaran masih ada dengan angka 600.000 dan `admin_doc_id` kosong (dijalankan), sesuai peringatan `budget_link_lost`.

---

## 5. Master Data Template

### 5.1 Jadwal: 44 Tugas PDF Hal. 3

Nama tugas, urutan, dan durasi (1–2 hari) disalin dari sheet *Timeline Planning*. **Yang bukan dari PDF**: kolom Scope/Kategori/Fase dan offset Hari-H (PDF hanya memuat tanggal absolut Feb–Apr 2026 yang tidak bisa dipakai ulang untuk Hari-H 28 Mar 2027; offset dibuat agar urutan dan durasi PDF tetap, dengan celah menjelang H-45). Nama fase: 1 *Persiapan Awal & Lamaran* (tugas 1–7) · 2 *Administrasi & Booking Vendor Inti* (8–15) · 3 *Detail Acara & Perlengkapan* (16–35) · 4 *Finalisasi & Menuju Hari-H* (36–44). Semua `origin = 'pdf'`, `R = 180`. Kolom **E** menandai preset `wedding_essential_22`; `wedding_pdf_44` memuat semuanya. `template_key` = `wedding.` + judul berhuruf kecil bergaris-bawah, ditambah `_n` bila judul berulang; tanda kurung dan tambahannya dibuang (tugas 43 "Final Fitting (H-7 acara)" menjadi `wedding.final_fitting`). Kunci yang tertulis di tabel adalah nilai yang berlaku: ia disimpan sebagai konstanta di kode, tidak dihitung ulang dari judul. `sort_order` = nomor tugas (tugas Adicara 45–50).

| No | `template_key` | Nama Tugas (PDF) | Scope | Kategori | Fase | Durasi | Offset Hari-H | E |
|:-:|---|---|:-:|---|:-:|:-:|:-:|:-:|
| 1 | `wedding.pertemuan_keluarga_1` | Pertemuan keluarga | `lamaran` | Keluarga | 1 | 1 hari | H-180 s/d H-179 |  |
| 2 | `wedding.pilih_tanggal_lamaran` | Pilih tanggal lamaran | `lamaran` | Acara | 1 | 1 hari | H-178 s/d H-177 |  |
| 3 | `wedding.tetapkan_anggaran_lamaran` | Tetapkan anggaran lamaran | `lamaran` | Keuangan | 1 | 2 hari | H-176 s/d H-174 |  |
| 4 | `wedding.draf_daftar_tamu_lamaran` | Draf daftar tamu lamaran | `lamaran` | Tamu | 1 | 2 hari | H-173 s/d H-171 |  |
| 5 | `wedding.book_vendor_untuk_lamaran` | Book vendor untuk lamaran | `lamaran` | Vendor | 1 | 1 hari | H-173 s/d H-172 |  |
| 6 | `wedding.persiapkan_cincin_dan_kebaya_lamaran` | Persiapkan cincin dan kebaya lamaran | `lamaran` | Busana | 1 | 1 hari | H-171 s/d H-170 |  |
| 7 | `wedding.lamaran` | Lamaran | `lamaran` | Acara | 1 | 1 hari | H-169 s/d H-168 |  |
| 8 | `wedding.penentuan_tanggal_pernikahan` | Penentuan tanggal pernikahan | `all` | Acara | 2 | 1 hari | H-167 s/d H-166 | ✓ |
| 9 | `wedding.siapkan_pendaftaran_pernikahan` | Siapkan pendaftaran pernikahan | `akad` | Administrasi | 2 | 1 hari | H-165 s/d H-164 | ✓ |
| 10 | `wedding.buat_list_seserahan` | Buat list seserahan | `all` | Logistik | 2 | 1 hari | H-163 s/d H-162 | ✓ |
| 11 | `wedding.tetapkan_anggaran_pernikahan` | Tetapkan anggaran pernikahan | `all` | Keuangan | 2 | 1 hari | H-161 s/d H-160 | ✓ |
| 12 | `wedding.draf_daftar_tamu_pernikahan` | Draf daftar tamu pernikahan | `resepsi` | Tamu | 2 | 2 hari | H-159 s/d H-157 | ✓ |
| 13 | `wedding.pertemuan_keluarga_2` | Pertemuan keluarga | `all` | Keluarga | 2 | 1 hari | H-156 s/d H-155 | ✓ |
| 14 | `wedding.book_vendor_untuk_pernikahan` | Book Vendor untuk pernikahan | `resepsi` | Vendor | 2 | 1 hari | H-154 s/d H-153 | ✓ |
| 15 | `wedding.buat_list_administrasi` | Buat list administrasi | `akad` | Administrasi | 2 | 1 hari | H-152 s/d H-151 | ✓ |
| 16 | `wedding.finalisasi_seserahan` | Finalisasi Seserahan | `all` | Logistik | 3 | 1 hari | H-150 s/d H-149 | ✓ |
| 17 | `wedding.pertemuan_keluarga_3` | Pertemuan keluarga | `all` | Keluarga | 3 | 1 hari | H-148 s/d H-147 |  |
| 18 | `wedding.fix_budgeting` | Fix budgeting | `all` | Keuangan | 3 | 1 hari | H-146 s/d H-145 | ✓ |
| 19 | `wedding.fix_tamu_undangan` | Fix Tamu undangan | `resepsi` | Tamu | 3 | 1 hari | H-144 s/d H-143 | ✓ |
| 20 | `wedding.moodboard_wedding_dream` | Moodboard wedding dream | `all` | Kreatif | 3 | 1 hari | H-142 s/d H-141 |  |
| 21 | `wedding.siapkan_undangan_fisik_dan_digital` | Siapkan undangan fisik dan digital | `all` | Undangan | 3 | 1 hari | H-140 s/d H-139 | ✓ |
| 22 | `wedding.souvenir` | Souvenir | `resepsi` | Logistik | 3 | 1 hari | H-138 s/d H-137 |  |
| 23 | `wedding.photobooth` | Photobooth | `resepsi` | Vendor | 3 | 1 hari | H-136 s/d H-135 |  |
| 24 | `wedding.pertemuan_keluarga_4` | Pertemuan keluarga | `all` | Keluarga | 3 | 1 hari | H-134 s/d H-133 |  |
| 25 | `wedding.visit_venue_pernikahan` | Visit Venue Pernikahan | `resepsi` | Vendor | 3 | 1 hari | H-132 s/d H-131 | ✓ |
| 26 | `wedding.info_tanggal_ke_bridesmaid_grooms` | Info tanggal ke bridesmaid & grooms | `all` | Panitia | 3 | 1 hari | H-130 s/d H-129 |  |
| 27 | `wedding.meeting_1_dengan_wo` | Meeting 1 dengan WO | `all` | WO | 3 | 1 hari | H-128 s/d H-127 |  |
| 28 | `wedding.prepare_berkas_kua` | Prepare berkas KUA | `akad` | Administrasi | 3 | 1 hari | H-126 s/d H-125 | ✓ |
| 29 | `wedding.pick_lock_semua_vendor_pernikahan` | Pick & Lock semua vendor pernikahan | `all` | Vendor | 3 | 1 hari | H-124 s/d H-123 | ✓ |
| 30 | `wedding.prewedding` | Prewedding | `all` | Dokumentasi | 3 | 1 hari | H-122 s/d H-121 |  |
| 31 | `wedding.finalisasi_urusan_dengan_kua` | Finalisasi urusan dengan KUA | `akad` | Administrasi | 3 | 1 hari | H-90 s/d H-89 | ✓ |
| 32 | `wedding.finalisasi_moodboard_wedding_yang_diinginkan` | Finalisasi moodboard wedding yang diinginkan | `all` | Kreatif | 3 | 1 hari | H-85 s/d H-84 |  |
| 33 | `wedding.souvenir_undangan_photobooth` | Souvenir, undangan , Photobooth | `resepsi` | Logistik | 3 | 1 hari | H-75 s/d H-74 |  |
| 34 | `wedding.research_book_honeymoon_destination` | Research & book honeymoon destination | `all` | Liburan | 3 | 1 hari | H-65 s/d H-64 |  |
| 35 | `wedding.meeting_2_dengan_wo` | Meeting 2 dengan WO | `all` | WO | 3 | 1 hari | H-60 s/d H-59 | ✓ |
| 36 | `wedding.pertemuan_keluarga_5` | Pertemuan keluarga | `all` | Keluarga | 4 | 1 hari | H-45 s/d H-44 |  |
| 37 | `wedding.penerima_tamu` | Penerima tamu | `resepsi` | Panitia | 4 | 1 hari | H-35 s/d H-34 |  |
| 38 | `wedding.fitting_attire` | Fitting Attire | `all` | Busana | 4 | 1 hari | H-30 s/d H-29 | ✓ |
| 39 | `wedding.bridal_shower` | Bridal Shower | `all` | Acara | 4 | 1 hari | H-25 s/d H-24 |  |
| 40 | `wedding.finalisasi_honeymoon_trip` | Finalisasi Honeymoon Trip | `all` | Liburan | 4 | 1 hari | H-20 s/d H-19 |  |
| 41 | `wedding.finalisasi_payment_ekspektasi_hasil_dari_tiap_vendor` | Finalisasi payment + ekspektasi hasil dari tiap vendor | `all` | Keuangan | 4 | 1 hari | H-14 s/d H-13 | ✓ |
| 42 | `wedding.meeting_3_dengan_wo` | Meeting 3 dengan WO | `all` | WO | 4 | 1 hari | H-10 s/d H-9 | ✓ |
| 43 | `wedding.final_fitting` | Final Fitting (H-7 acara) | `all` | Busana | 4 | 1 hari | H-7 s/d H-6 | ✓ |
| 44 | `wedding.kirim_undangan` | Kirim undangan | `all` | Undangan | 4 | 1 hari | H-5 s/d H-4 | ✓ |

Pada Proyek Contoh (`Avail = 175`) hasil penskalaan menjaga durasi PDF; tanggal contoh dan status sampel ada di [bab 4.6](#46-sambungan-ke-menu-01). Nama, urutan, dan durasi 44 tugas dibandingkan dengan PDF Hal. 3 (pdfminer): 43 nama identik, tugas 43 hanya berbeda karena font PDF memakai karakter privat untuk tanda kurung dan hubung; 3 tugas berdurasi 2 hari (3, 4, 12) dan 41 berdurasi 1 hari; status PDF 15 *Selesai* dan 29 *Proses*.

### 5.2 Tugas Tambahan Adicara (`origin = 'adicara'`, Fase 4)

PDF berhenti di H-4 dan tidak memuat tugas yang menjadi dasar integrasi modul 05 dan 06 (finalisasi tamu, sebar undangan, follow-up RSVP, technical meeting, gladi resik). Enam tugas ini opsional lewat `include_adicara_tasks` dan ditandai "Tambahan Adicara" di antarmuka. Fase 4 (*Finalisasi & Menuju Hari-H*), `sort_order` 45–50, status template `belum`.

| `template_key` | Tugas | Scope | Kategori | Offset | Terhubung ke |
|---|---|:-:|---|:-:|---|
| `wedding.adicara_finalisasi_tamu` | Finalisasi daftar tamu dan kuota pax | `resepsi` | Tamu | H-45 s/d H-44 | Menu 05 |
| `wedding.adicara_sebar_undangan_digital` | Sebar undangan digital via WhatsApp | `all` | Undangan | H-30 s/d H-29 | Menu 05 |
| `wedding.adicara_followup_rsvp` | Follow-up RSVP tamu | `resepsi` | Tamu | H-10 s/d H-9 | Menu 05 → proyeksi katering (Menu 03) |
| `wedding.adicara_technical_meeting` | Technical meeting vendor dan panitia | `all` | Panitia | H-7 s/d H-6 | Menu 04, 06 |
| `wedding.adicara_gladi_resik` | Gladi resik | `all` | Acara | H-1 s/d H-0 | Menu 06 |
| `wedding.adicara_hari_h` | Hari-H acara | `all` | Acara | H-0 s/d H-0 | Menu 06 |

### 5.3 Dokumen: 15 Berkas PDF Hal. 11

Urutan dan nama berkas persis PDF (dibandingkan dengan Hal. 11; `template_key` di tabel adalah konstanta, tidak diturunkan mekanis dari judul; `sort_order` = nomor). Kolom *Peran*, *Jumlah*, dan *Biaya* **tidak ada di PDF** (kolom Nominal/Detail/Jumlah pada PDF kosong): itu saran Adicara (`origin` tetap `pdf` untuk nama, nilai tambahan boleh diubah). `status` template selalu `belum`; Proyek Contoh memakai status PDF (selesai: 1, 2, 3, 6, 7, 8, 10, 14, 15; belum: 4, 5, 9, 11, 12, 13).

| No | `template_key` | Dokumen Pernikahan (PDF) | Peran | Jumlah | Biaya (saran) | Catatan |
|:-:|---|---|:-:|:-:|:-:|---|
| 1 | `wedding.pendaftaran_kua` | Pendaftaran KUA | `bersama` | 1 berkas | 600.000 | PNBP nikah di luar kantor/hari libur; Rp 0 di kantor KUA jam kerja |
| 2 | `wedding.fc_cpp` | FC CPP ( KTP, Ijazah, Akte, KK) | `cpp` | 2 lembar | 0 | Fotokopi legalitas calon pengantin pria |
| 3 | `wedding.fc_cpw` | FC CPW ( KTP, Ijazah, Akte, KK) | `cpw` | 2 lembar | 0 | Fotokopi legalitas calon pengantin wanita |
| 4 | `wedding.fc_ktp_ortu_cmp` | FC KTP Ortu CMP | `ortu_cpp` | 2 lembar | 0 | KTP ayah dan ibu pihak pria |
| 5 | `wedding.fc_ktp_ortu_cmw` | FC KTP Ortu CMW | `ortu_cpw` | 2 lembar | 0 | KTP ayah dan ibu pihak wanita |
| 6 | `wedding.surat_ket_sehat_puskesmas` | Surat Ket. Sehat puskesmas | `bersama` | 1 berkas | 0 | Pemeriksaan kesehatan Puskesmas |
| 7 | `wedding.vaksin_tetanus` | Vaksin Tetanus | `cpw` | 1 lembar | 0 | Surat keterangan imunisasi TT calon pengantin wanita |
| 8 | `wedding.surat_pengantar_nikah_kelurahan` | Surat pengantar nikah dari kelurahan | `bersama` | 1 berkas | 0 | Formulir N1, N2, N4 yang ditandatangani lurah/kepala desa |
| 9 | `wedding.pas_foto_2x3` | Pas Foto Background Biru 2x3 (3pcs) | `bersama` | 3 pcs | 15.000 | – |
| 10 | `wedding.pas_foto_3x4` | Pas Foto Background Biru 3x4 (3pcs) | `bersama` | 3 pcs | 15.000 | – |
| 11 | `wedding.pas_foto_4x6` | Pas Foto Background Biru 4x6 (3pcs) | `bersama` | 3 pcs | 15.000 | – |
| 12 | `wedding.surat_ket_belum_nikah` | Surat ket. belum nikah dr setempat | `bersama` | 1 lembar | 0 | Dari kelurahan masing-masing |
| 13 | `wedding.materai_10000` | Materai 10.000 | `bersama` | 2 pcs | 20.000 | Berkas pernyataan |
| 14 | `wedding.mahar_untuk_pengajuan_kua` | Mahar (utk pengajuan KUA) | `bersama` | 1 rincian | 0 | Rincian mas kawin untuk akta |
| 15 | `wedding.lain_lain` | Lain Lain | `bersama` | 1 berkas | 0 | Mis. rekomendasi numpang nikah dari KUA asal |

### 5.4 Panduan Birokrasi (PDF Hal. 11)

Disimpan di registry (`AdminGuide`, judul "Panduan Pengurusan Berkas Nikah"), bukan di komponen FE, dan dikirim lewat `GET /event-types` (§2). Isi mengikuti catatan PDF; formulir yang disebut PDF adalah **N1, N2, dan N4**.

1. **RT & RW**: minta surat pengantar dengan membawa fotokopi KTP kamu dan pasangan serta fotokopi Kartu Keluarga.
2. **Kelurahan**: bawa surat pengantar untuk mengisi formulir N1, N2, N4 dan surat keterangan belum menikah; syarat pasfoto 3×4 dan 2×3 masing-masing 2 lembar, fotokopi KTP CPW dan CPP 2 lembar, fotokopi KK CPP dan CPW 2 lembar, dan surat pengantar RT/RW.
3. **KUA asal (numpang nikah)**: minta surat rekomendasi numpang nikah di tempat tujuan; siapkan fotokopi KTP CPP dan CPW, fotokopi KK, surat pengantar kelurahan (N1, N2, N4), pasfoto 2×3 dan 3×4 masing-masing 2 lembar.
4. **KUA tempat akad**: serahkan surat rekomendasi pindah nikah dari KUA asal, fotokopi KTP dan KK, foto berwarna 2×3 dan 3×4 masing-masing 2 lembar, fotokopi ijazah terakhir, dan fotokopi akta kelahiran.

Jenis acara lain: panduan 4 langkah perizinan ada di [TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02) (usulan Adicara, bukan dari PDF); jadwal dan dokumen non-pernikahan ada di bab 5.5.

### 5.5 Template jenis non-pernikahan (usulan Adicara)

Nama tugas, offset, dan daftar dokumen ada di [TEMPLATE_EVENT bab 2 dan 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#2-template-general-15-tugas-r--60-untuk-jenis-non-pernikahan); semuanya **bukan dari PDF** dan ber-`origin = 'adicara'`. Kolom yang tidak ditulis di sana diisi dengan aturan ini:

| Kolom | Aturan |
|---|---|
| `template_key` | `general.<slug>` untuk 15 tugas umum; `<event_type>.<slug>` untuk tugas khusus jenis dan dokumen (mis. `seminar.undang_dan_konfirmasi_pembicara`). Slug = judul huruf kecil, selain `[a-z0-9]` menjadi `_`, berulang dilebur. Seperti kunci pernikahan, nilai yang jadi acuan adalah konstanta di kode. |
| `sub_event_scope` | `all` |
| `category`, `phase_order`, `phase_name` | tugas umum: kolom Kategori dan Fase di TEMPLATE_EVENT bab 2. Tugas khusus: kategori `Umum`; fase menurut offset **mulai** pada `R = 60`: ≥ 47 → 1 *Perencanaan*, 35–46 → 2 *Persiapan*, 8–34 → 3 *Pelaksanaan*, ≤ 7 → 4 *Pekan Terakhir*. Aturan ini menghasilkan fase yang sama untuk 15 tugas umum (diperiksa dengan skrip, nol selisih), jadi satu aturan untuk semuanya. |
| `subject_role` dokumen | huruf dalam kurung di TEMPLATE_EVENT (`panitia`, `lembaga`, `umum`) |
| `quantity_detail`, `nominal_cost` dokumen | "1 berkas", Rp 0 |
| `sort_order` tugas | kronologis: offset mulai terbesar lebih dulu, seri menurut urutan template (tugas umum lebih dulu). Tiga tugas khusus disisipkan di posisi kronologisnya (Seminar: "Undang dan konfirmasi pembicara" di antara tugas umum 2 dan 3), bukan ditumpuk di akhir. Diperiksa dengan skrip untuk tujuh jenis: urutan 15 tugas umum tidak berubah, `phase_order` tidak pernah turun sepanjang `sort_order`, dan offset mulai tidak pernah naik. Pernikahan tetap urutan PDF (§5.1) dengan enam tugas Adicara di belakang |

**Offset harus berada di `0…R`** ([TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis)). Diperiksa dengan skrip atas seluruh template: 44 + 6 tugas pernikahan (0…180, offset mulai tak pernah naik menurut nomor, durasi PDF = `max(mulai − selesai, 1)`), 15 tugas umum, dan 21 tugas khusus. Ditemukan **satu pelanggaran**: Seminar "Undang dan konfirmasi pembicara" H-70…H-56 (di atas `R = 60`, sehingga selalu dijepit ke awal jadwal dan durasinya menyusut). Diperbaiki di TEMPLATE_EVENT menjadi H-58…H-50 (setelah panitia inti terbentuk, selesai tepat sebelum registrasi H-50 dibuka). Dijaga `TestEveryTemplateOffsetIsWithinItsHorizon`.

Jumlah per jenis: **18 tugas** (15 + 3 khusus) untuk khitanan, aqiqah, peresmian, launching, ulang tahun, wisuda, dan seminar; **15 tugas** untuk `other`. Dokumen: khitanan 3, aqiqah 3, peresmian 4, launching 4, ulang tahun 1, wisuda 3, seminar 4, `other` tidak ada template (pengguna menambah sendiri).

---

## 6. Pengujian

Status per 2026-10-21. Berkas: `internal/modules/plannertimeline/{domain,usecase,delivery/http}/*_test.go` dan `internal/ginutil/slug_test.go`. Tes DB butuh `TEST_DATABASE_URL` ke database `…_test` (isinya **dihapus**) dan di-skip tanpa itu; jalankan dengan `go test -p 1`. Semua baris di bawah ✔ dijalankan terhadap PostgreSQL 16.2 dengan pgx mode simple-protocol seperti produksi.

| Butir | Tes |
|---|---|
| DDL bab 1 identik dengan migrasi 007; naik, turun, naik | perbandingan teks blok `sql` pertama dengan file migrasi (diulang saat implementasi); `TestPlannerMigrationsRollBackAndReapply` Menu 01 (005–012) |
| Data template sama dengan teks spesifikasi: 44 tugas PDF (nama, kunci, scope, kategori, fase, offset, tanda E), 6 tugas Adicara, 15 berkas, 15 tugas umum, 21 tugas dan 22 berkas khusus jenis | `TestTemplatesMatchSpec` (membaca tabel BE.md dan TEMPLATE_EVENT; di-skip bila folder `docs/` tidak ikut) |
| Bentuk template: 44/22/50/28/6 tugas per preset; durasi PDF (tiga tugas 2 hari) dan urutan; semua offset di `0…R`; 18/15 tugas per jenis, kronologis, fase tak pernah turun, kunci unik dan mengikuti aturan slug; 15/3/3/4/4/1/3/4/0 berkas, subjek ada di registry; setiap jenis punya subjek, label, dan panduan 4 langkah | `TestWeddingSchedules`, `TestWeddingTemplateKeepsPDFDurationsAndOrder`, `TestEveryTemplateOffsetIsWithinItsHorizon`, `TestGeneralSchedulePerType`, `TestSeminarSpeakerTaskSitsRightAfterTheCommitteeTasks`, `TestDocTemplates`, `TestEveryRegistryTypeHasASubjectListGuideAndLabel` |
| Penskalaan: Proyek Contoh (`Avail = 175`) menaruh tugas 1 pada hari ini dan Lamaran pada 15 Okt; horizon baku; `aqiqah` `Avail = 10`; `Avail < 3` dijepit dan hari lampau ditolak; `prep_start_date` lebih akhir menang; durasi PDF utuh pada `Avail` 175/180 dan tiga tugas 2 hari menyusut pada 120 | `TestSampleProjectScalesBy175Over180`, `TestEnoughTimeUsesTheStandardHorizon`, `TestAqiqahWithTenDaysCollapsesGladiResikToHariH`, `TestTooShortClampsEverythingAndPastEventIsRejected`, `TestPrepStartLaterThanTodayWins`, `TestPDFDurationsSurviveScalingOnlyWithoutCompression` |
| **Uji emas Proyek Contoh** (kontrak bab 9): 50 tugas (15 selesai, 29 proses, 6 belum Adicara; tiga berdurasi 2 hari), tanggal contoh, scope, Lamaran = 15 Okt 2026, 15 berkas (9 selesai), widget Timeline `50/15/29/0` dan Legalitas `9/15` dengan lima nama pertama, pembuatan ulang tidak menambah apa pun | `TestSampleProjectGolden` |
| Wizard atomik dengan applier sungguhan: preset essential + Adicara + berkas; Adicara saja; seminar `general` (18 tugas, 4 berkas); `other` tanpa berkas dan preset keluarga lain ditolak tanpa meninggalkan proyek; Hari-H hari ini → `lead_time_too_short` | `TestWizardAppliesTimelineAndDocumentsAtomically`, `TestEventTypesOffersTheTemplatesOnceTheModuleIsRegistered` |
| Generator: `append` dua kali, `replace_template` mempertahankan tugas kustom dan butuh `confirm` hanya bila ada kemajuan, `replace_all` selalu, penolakan tidak mengubah apa pun, `scope_fallback`, jenis lain, hari lampau | `TestGenerateModes` |
| Tugas: tambah (default, fase mengikuti), ubah sebagian, status, hapus, saring (`scope=all` = Umum), semua galat validasi dengan pesan `<field>: <alasan>`, tepi jendela tanggal, vendor proyek lain, `clear_vendor`, PATCH hanya memvalidasi yang dikirim | `TestTaskLifecycle`, `TestTaskValidation`, `TestCleanNoteAllowsLineBreaksButNoOtherControlCharacters`, `TestTaskQueryFilter`, `TestScheduleWindow`, `TestParseMode` |
| Urut ulang: daftar lengkap menulis 1…n; kurang, ganda, id proyek lain, kosong, kelebihan ditolak tanpa mengubah; tugas proyek lain tidak tersentuh | `TestReorderTasks`, `TestCheckIDs` |
| Geser manual: hanya tugas ber-`template_key` yang belum selesai (35 dari 50), tidak idempoten, `tasks_overdue_after_shift`, jendela tanggal, `days` 0 dan > 1826 | `TestManualShiftMovesOnlyUnfinishedTemplateTasks` |
| Ringkasan dan widget: `overdue` memakai zona proyek (Jakarta vs Jayapura pada instan yang sama); `urgent` ≤ 3 dalam 7 hari dan yang lewat ikut; angka audit pada 4 Okt, 1 Nov, 4 Nov 2026; batas tengah malam 23:59 vs 00:00 | `TestOverdueIsComputedByTheServerInTheProjectZone`, `TestTimelineWidgetAcrossTime` |
| Dokumen: tambah, ubah, status, hapus, saring, subjek per jenis (bawaan `bersama` / `umum`), urut ulang, `apply-template` (`append`, `replace_template` + `confirm` + `budget_link_lost`, `replace_all` ditolak, `other` → `invalid_template`) | `TestDocumentLifecycleAndValidation`, `TestApplyDocumentTemplate`, `TestDefaultSubject` |
| Sinkron biaya: kasus A–H (§4.5), tautan ke dokumen lain tidak ditimpa, sesi eksplisit; **klik ganda (8 permintaan serentak) membuat satu pos** | `TestSyncBudget`, `TestSyncBudgetDoubleClickMakesOneLine` |
| Batas: 500 tugas (tugas ke-501 → `task_limit_reached`; generator melewati batas tidak menulis apa pun; `replace_all` memberi ruang), 200 berkas; **enam `POST /tasks` serentak pada 499 tugas → tepat satu `201`**, `sort_order` berbeda semua | `TestTaskLimit`, `TestDocumentLimit`, `TestTwoSimultaneousCreatesCannotPassTheLimit` |
| Galat FK di tempat munculnya: vendor proyek lain pada pernyataan, sesi tak dikenal baru saat `COMMIT` (`ErrConflict`), `template_key` ganda, id proyek lain → `ErrNotFound` | `TestForeignKeyErrorsAreMappedWhereTheyAppear` |
| Isolasi: pengguna B mendapat `404` di 17 endpoint proyek A, id tugas/dokumen proyek A tidak dikenal di proyek B, id bukan UUID kanonik `404`; tanpa token `401`; arsip menolak tulis (`project_read_only`), `GET` dan ekspor tetap boleh | `TestUserBGetsNotFoundOnEveryEndpointOfUserAsProject`, `TestEveryRouteNeedsABearerToken`, `TestArchivedProjectIsReadOnlyButStillReadableAndExportable` |
| Ekspor: judul `=cmd\|…`, `+`, `-`, `@` menjadi teks (tanpa elemen `<f>`), tanggal sungguhan, 26 pekan, 50 × 26 sel Gantt sama dengan perhitungan hari demi hari (61 terisi), tanpa tugas hanya header, sesi tak dikenal memakai kuncinya, nama berkas dari slug, 10/menit/pengguna | `TestWorkbookOfTheSampleProject`, `TestWorkbookWithoutTasksHasOnlyTheHeader`, `TestWorkbookKeepsASessionNameThatIsNoLongerKnown`, `TestSafeCellPrefixesFormulaStarters`, `TestMondayOf`, `TestExportSpreadsheet`, `TestExportIsRateLimitedPerUser`, `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow`, `TestSlug` (`internal/ginutil`) |

**Hasil** (2026-10-21, PostgreSQL 16.2 lokal, database sementara `planner_tl_scratch_test` yang dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh): `go test -p 1 -count=1 ./...` lulus seluruhnya, **177 tes tingkat atas** (127 sebelumnya + 50 milik modul ini), 0 gagal, 0 di-skip; `-race` pada `plannertimeline/...` dan `ginutil` lulus; cakupan pernyataan modul **87,8%** (`-coverpkg`); `GOOS=linux CGO_ENABLED=0 go build ./cmd/api` dan `go vet ./...` bersih; `npx @redocly/cli lint backend/docs/openapi.yaml`: valid dengan 3 peringatan lama (lisensi, server `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru; 17 rute terdaftar = 17 operasi di `openapi.yaml` (dicocokkan dengan skrip).

**Tidak diuji:** image Docker dan PostgreSQL 18; Nginx di depan rute baru; beberapa replika API (pembatas ekspor dihitung per proses); `.xlsx` yang dibuka di Excel, Numbers, atau LibreOffice (isinya dibaca kembali dengan `excelize` dan parser XML, bukan dilihat); latensi `GET /tasks` pada 500 tugas lewat HTTP (hanya diukur di SQL: 0,2 ms); frontend.

---

## 7. Urutan pengerjaan dan definisi selesai

Posisi di urutan pembangunan: setelah `planner` (Menu 01) dan sebelum `plannerfinance`; Menu 04 (vendor) cukup tabelnya, yang sudah ada. **Semua langkah selesai pada 2026-10-21.**

1. ✔ **Registry**: `Option`, `Guide`, `AdminSubjects`, `AdminGuide`, `AdminLabel` di `planner/domain/eventtype.go` beserta nilai [TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02); `GET /event-types` mengirim `admin_subjects` dan `admin_guide` (bukan `AdminLabel`).
2. ✔ **`plannertimeline`**: `domain` (entitas, `Plan`, template 44 + 6 + 15 + `general` + 8 jenis; data dibangkitkan dari tabel spesifikasi), `repository`, `usecase`, `delivery/http` (17 rute, `ratelimit.go` salinan pola `userLimiter`, `export.go`). Pembantu baru: `cleanNote` di usecase dan `ginutil.Slug`.
3. ✔ **Sambungan** di `internal/app/app.go`: satu `Registry` dan satu `TxManager` dipakai bersama; provider, applier, overlay, dan rute terdaftar. Widget 1 dan 5 di dashboard tidak lagi `null`.
4. ✔ **Dependensi**: `github.com/xuri/excelize/v2 v2.11.0` ditambahkan ke `go.mod` (satu modul langsung dan lima tidak langsung; modul backend lain tidak turun versi) atas permintaan pemilik untuk segera mengimplementasikan, dan **disetujui pemilik pada 2026-10-21**.
5. ✔ **Dokumen**: `backend/docs/openapi.yaml`; [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-02.md) dengan daftar penyimpangan; `docs/backend/README.md` dan `CHANGELOG.md`; `docs/api.md`, `docs/architecture.md`, `backend/README.md`, `README.md`; entri [CHANGELOG spesifikasi](../CHANGELOG.md).
6. ✔ **Pengujian**: lihat bab 6.

**Selesai bila:** ✔ semua butir bab 6 lulus; ✔ widget Timeline/Legalitas Proyek Contoh sesuai uji emas; ✔ `templates_available` memuat preset dan `admin_docs`; ✔ openapi lolos lint tanpa peringatan baru; ✔ dokumen implementasi menyebut apa yang tidak diuji.

---

## 8. Celah dan keputusan yang menunggu pemilik produk

Butir 1–14 ditemukan saat mencocokkan spesifikasi dengan kode Menu 01 yang sudah jadi, kontrak, PDF, dan SQL yang dijalankan; butir 15–19 saat membangun modulnya; butir 20 saat mencocokkan FE dengan responsnya. Kode dan dokumen mengikuti pilihan yang tertulis; ubah bila produk menginginkan lain. **Sudah diputuskan pemilik:** butir 6 (dependensi `excelize` disetujui 2026-10-21). **Yang masih memerlukan keputusan pemilik:** butir 8 (isi template dan teks panduan non-pernikahan adalah usulan Adicara, bukan dari PDF), butir 12 (awalan `'` yang tampak di sel). Butir lain punya pilihan bawaan yang sudah berjalan.

1. **Label preset generator.** Registry hanya membawa kunci preset (`templates_available.timeline_presets`), bukan label, dan FE Menu 01 sudah memutuskan kamus `TEMPLATE_LABELS` berkunci preset. Menu 02 memakai kamus yang sama (bukan label dari registry seperti tertulis di versi lama README/FE). Bila ingin label di backend, tambahkan ke `TemplateSet`.
2. **Judul widget Legalitas** (`AdminLabel`): pernikahan "Kelengkapan Berkas KUA" (cocok dengan fixture 01 FE), jenis lain "Kelengkapan Perizinan" (README 01 menyebut "perizinan dan kontrak venue"; satu judul ringkas dipilih).
3. **Batas dan jendela tanggal** (500 tugas, 200 dokumen, `Hari-H − 1095` s/d `akhir acara + 365` hari, teks 60–2.000 karakter) adalah angka usulan, mengikuti sikap Menu 01 (mis. batas 20 sesi dan 30 hari). Tanpa batas, daftar tanpa paginasi dan Gantt bisa membengkak. Pengukuran (§3.4): 500 tugas = 370 kB dan 200 dokumen = 114 kB, kueri 0,2 ms, jadi angka ini adalah batas tampilan, bukan batas kinerja basis data, dan aman dinaikkan dari sisi PostgreSQL. Kode galat baru `task_limit_reached` dan `admin_doc_limit_reached` ditambahkan di kontrak.
4. **`confirm` pada `replace_template`.** Spesifikasi lama hanya mewajibkannya untuk `replace_all`, padahal `replace_template` menghapus tugas selesai dan tautan vendor tanpa peringatan. Kini wajib bila ada kemajuan yang hilang (§4.1, §4.7).
5. **Batas modul sinkron biaya.** SQL langsung ke `planner_budget_items` (dicatat sebagai pengecualian di kontrak) dipilih karena Menu 03 belum ada; bila Menu 03 lebih suka memiliki penulisan itu, ganti dengan antarmuka Go di `planner/domain` yang didaftarkan seperti `SummaryProvider`. Perilaku HTTP tidak berubah.
6. **Dependensi `excelize`** ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi) menyebutnya "disetujui saat implementasi"): ditambahkan ke `go.mod` bersama kode ekspor atas permintaan pemilik untuk mengimplementasikan segera, lalu **disetujui pemilik pada 2026-10-21** (keputusan: tetap memakai `excelize`, bukan CSV). Yang sudah diketahui: v2.11.0 dipakai dalam prototipe (§4.4), tambahan `go.mod` satu modul langsung + lima tidak langsung, tanpa penurunan versi modul backend. Alternatif tanpa dependensi yang praktis hanya CSV (`encoding/csv`), tanpa warna Gantt.
7. **Status "proses" untuk 29 tugas pada Proyek Contoh** setia pada PDF, tetapi secara isi janggal untuk tugas yang baru mulai Nov 2026 sampai Apr 2027. Dipertahankan karena PDF dijadikan data emas; widget dashboard Proyek Contoh menampilkan "Berjalan: 29".
8. **Tugas khusus jenis non-pernikahan** memakai kategori `Umum` dan fase dari offset (§5.5); isi dan panduan perizinan ([TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02)) adalah usulan Adicara yang belum ditinjau pemilik produk.
9. **Sinkron biaya satu arah dan satu kali.** Mengubah biaya dokumen setelah tersinkron tidak memperbarui pos anggaran, dan melepas tautan tidak ada kecuali menghapus dokumen. Bila produk ingin sinkron dua arah atau tombol "lepas", itu perluasan endpoint.
10. **`summary` pada `GET /tasks` selalu seluruh proyek**, jadi penyaring sesi di layar tidak mengubah progres di header. Bila ingin progres per sesi, hitung di klien dari `tasks` atau tambahkan parameter.
11. **Nama kunci ringkasan.** Spesifikasi lama memakai `selesai/proses/belum` sebagai kunci JSON (`summary {total, selesai, proses, belum, overdue}`); diganti `completed/in_progress/not_started` agar sama dengan `TimelineSummary` Menu 01. Nilai status (`belum`/`proses`/`selesai`) tetap karena itu data PDF.
12. **Awalan `'` tampak di sel Excel.** Kontrak bab 2 butir 5 mewajibkan awalan `'` pada sel berawalan `=`, `+`, `-`, `@`. Prototipe menunjukkan `excelize` selalu menulis teks pengguna sebagai string (tidak pernah formula), jadi sel tidak dieksekusi saat dibuka, dan awalan `'` yang ditulis sebagai karakter akan **terlihat** (mis. `'-5 hari lagi`). Dokumen mengikuti kontrak (pertahanan berlapis terhadap CSV atau pengeditan ulang). Alternatif: tidak memberi awalan karena tipe sel sudah string; ubah kontrak bab 2 butir 5 bila produk menganggap tanda kutip terlihat itu mengganggu.
13. **Salinan `userLimiter`.** Pembatas laju per pengguna kini ada di modul auth (per IP) dan Menu 01 (per pengguna, tidak diekspor); Menu 02 akan menjadi salinan ketiga, dan Menu 03–06 menyusul untuk ekspor/impor. Pilihan bawaan: menyalin (diff kecil, sesuai kontrak bab 1.4). Alternatif: mengekstrak ke `internal/ratelimit` dengan mengubah Menu 01 dan auth, lebih tepat dikerjakan satu kali sebelum Menu 03.
14. **Offset Seminar diubah.** "Undang dan konfirmasi pembicara" H-70…H-56 melampaui `R = 60` dan selalu terjepit; kini H-58…H-50 di TEMPLATE_EVENT bab 13. Ini isi template (usulan Adicara), jadi pemilik produk boleh menggantinya asal tetap di `0…R`.
15. **`PATCH` tugas dan dokumen memakai kunci proyek.** Spesifikasi awal menganggap penulisan satu baris tidak butuh kunci; `PATCH` menulis semua kolom yang bisa diubah, jadi tanpa kunci dua tab yang mengubah kolom berbeda saling menimpa. Biayanya satu `SELECT … FOR UPDATE` per `PATCH`; `PATCH …/status` dan `DELETE` tetap tanpa kunci.
16. **`preset` wajib dan `mode` bawaan `append`** pada `generate-defaults` (`400 invalid_input: preset: required`); spesifikasi awal tidak menyebut keduanya. Hanya untuk pernikahan, `include_adicara_tasks` tanpa preset dasar (di wizard) menghasilkan enam tugas Adicara saja.
17. **Satu peringatan jadwal per panggilan:** `lead_time_too_short` menggantikan `lead_time_compressed` (secara teknis `Avail < 3` juga memampatkan), `count` = tugas yang dibuat, dan tanpa tugas baru tidak ada peringatan. Spesifikasi awal tidak menetapkan `count`.
18. **`subject_role` kosong** pada tambah berkas = `bersama` (pernikahan) atau `umum` (delapan jenis lain; `bersama` tidak ada di subjek mereka). Kolom Catatan tabel §5.3 disimpan sebagai `notes` berkas template pernikahan.
19. **Nama berkas ekspor memakai `ginutil.Slug`** yang melipat aksen (`Café` → `cafe`), sedikit melampaui aturan "selain `[a-z0-9]` menjadi `-`"; judul proyek di basis data tidak boleh berisi karakter kontrol (CHECK), jadi baris baru di judul tidak mungkin terjadi.
20. **`PATCH` tugas tidak menyalin `phase_name` saat `phase_order` berubah** (`POST` menyalin nama dari tugas lain pada fase itu, §4.2). Teramati pada backend yang berjalan: `PATCH {"phase_order": 4}` menjawab `200` dengan `phase_order 4` tetapi `phase_name` fase lama, jadi satu tugas bisa membawa nama fase yang salah. [FE bab 4.1](FE.md#41-aturan-json-yang-dikirim-backend) menghindarinya dengan selalu mengirim `phase_order` dan `phase_name` berpasangan, sehingga tidak ada kerugian sekarang. Bila produk ingin server yang menyalin (konsisten dengan `POST`), perubahannya kecil: pada `UpdateTask`, bila `phase_order` dikirim tanpa `phase_name`, isi nama dari tugas lain pada fase itu; tambahkan satu tes dan satu kalimat di §4.2.
