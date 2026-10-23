# Menu 01: Dashboard Utama & Hub Acara — Backend Specification

> **Status (2026-10-21):** sudah diimplementasikan dan diuji ([dokumen](../../../backend/planner/2026-10-21-planner-menu-01.md), kontrak [`openapi.yaml`](../../../../backend/docs/openapi.yaml)). Penyimpangan dan koreksi yang ditemukan saat membangunnya ada di sana dan di [CHANGELOG](../CHANGELOG.md). Kode, bukan dokumen ini, yang berlaku bila berbeda.

Spesifikasi backend modul Go `planner` (`backend/internal/modules/planner/`, lapisan `delivery/http → usecase → domain ← repository`) untuk **proyek acara, master sesi, registry event type, dan dashboard**. Aturan yang berlaku untuk semua modul Planner (posisi di arsitektur, migrasi, isolasi data, template vs data contoh, konvensi uang/waktu/pgx, kode error, prasyarat runtime) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md) dan tidak diulang di sini.

---

## 1. Skema Database

Migrasi `backend/internal/migrate/sql/005_000_planner_core.sql` (format goose seperti `002_000_users.sql`; blok ini sudah dijalankan di PostgreSQL 16, lihat [kontrak bab 10](../00_KONTRAK_BERSAMA/README.md#10-validasi-spesifikasi)):

```sql
-- +goose Up
CREATE TABLE planner_projects (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'wedding', 'khitanan', 'aqiqah', 'peresmian', 'launching',
        'birthday', 'wisuda', 'seminar', 'other'
    )),
    title TEXT NOT NULL CHECK (
        char_length(title) BETWEEN 1 AND 255 AND title = btrim(title) AND title !~ '[[:cntrl:]]'
    ),
    event_date DATE NOT NULL,                           -- Hari-H (hari pertama bila acara beberapa hari)
    event_end_date DATE,                                -- NULL = acara satu hari
    prep_start_date DATE NOT NULL,                      -- diisi usecase (tanggal hari ini di zona waktu proyek)
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta' CHECK (timezone ~
        '^(UTC|(Africa|America|Antarctica|Arctic|Asia|Atlantic|Australia|Europe|Indian|Pacific)/[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)?)$'),
    target_budget BIGINT NOT NULL DEFAULT 0 CHECK (target_budget BETWEEN 0 AND 10000000000000),  -- rupiah utuh, maks 10^13
    savings_target BIGINT CHECK (savings_target BETWEEN 0 AND 10000000000000),                   -- NULL = ikut target_budget
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
    is_sample BOOLEAN NOT NULL DEFAULT FALSE,           -- proyek contoh dari dataset PDF
    skipped_vendor_categories TEXT[] NOT NULL DEFAULT '{}',  -- kategori vendor yang ditandai "tidak diperlukan"
    invitation_id UUID,                                 -- undangan digital terhubung (bundling); TANPA FK: tabel undangan belum ada
    client_token TEXT CHECK (char_length(client_token) BETWEEN 8 AND 64),  -- pengaman kirim-ganda dari wizard
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,        -- divalidasi usecase terhadap registry event type
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (prep_start_date <= event_date),
    CHECK (event_end_date IS NULL OR event_end_date >= event_date)
);

CREATE INDEX planner_projects_user_status_idx ON planner_projects (user_id, status);
CREATE INDEX planner_projects_user_date_idx ON planner_projects (user_id, event_date);
CREATE UNIQUE INDEX planner_projects_invitation_idx ON planner_projects (invitation_id) WHERE invitation_id IS NOT NULL;
CREATE UNIQUE INDEX planner_projects_sample_idx ON planner_projects (user_id) WHERE is_sample;
CREATE UNIQUE INDEX planner_projects_client_token_idx ON planner_projects (user_id, client_token) WHERE client_token IS NOT NULL;

-- Nama zona waktu harus dikenal PostgreSQL; satu nilai salah akan menggagalkan seluruh kueri Hub pengguna itu.
-- Bentuk kanonik (huruf besar-kecil, awalan wilayah) dijaga CHECK di atas; keberadaannya dijaga pemicu ini.
-- +goose StatementBegin
CREATE FUNCTION planner_check_timezone() RETURNS trigger AS $$
BEGIN
    PERFORM now() AT TIME ZONE NEW.timezone;
    RETURN NEW;
EXCEPTION WHEN invalid_parameter_value THEN
    RAISE EXCEPTION 'unknown time zone: %', NEW.timezone USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER planner_projects_timezone_trg
    BEFORE INSERT OR UPDATE OF timezone ON planner_projects
    FOR EACH ROW EXECUTE FUNCTION planner_check_timezone();

-- Master sesi/sub-acara per proyek. SATU-SATUNYA sumber nilai sub_event_scope
-- untuk modul 02-05; modul 06 memakai id-nya langsung.
CREATE TABLE planner_event_sessions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    scope_key TEXT NOT NULL CHECK (scope_key ~ '^[a-z0-9_]{1,40}$'),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100 AND name = btrim(name)),
    icon TEXT NOT NULL DEFAULT '📋' CHECK (char_length(icon) BETWEEN 1 AND 8),
    is_system BOOLEAN NOT NULL DEFAULT FALSE,           -- TRUE hanya untuk baris 'all'
    sort_order INT NOT NULL DEFAULT 0,
    session_date DATE,
    start_time TIME,
    end_time TIME,
    location_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, scope_key),
    UNIQUE (id, project_id),
    CHECK ((scope_key = 'all') = is_system)
);

-- Nama sesi unik per proyek: impor tamu dan ekspor memetakan sesi lewat nama.
CREATE UNIQUE INDEX planner_event_sessions_name_idx ON planner_event_sessions (project_id, lower(name));

-- +goose Down
DROP TABLE IF EXISTS planner_event_sessions;
DROP TABLE IF EXISTS planner_projects;
DROP FUNCTION IF EXISTS planner_check_timezone();
```

Catatan desain (setiap butir menutup celah yang diuji, bukan asumsi):

- **Konvensi:** `TEXT` (bukan `VARCHAR`); ID dari aplikasi (tanpa `gen_random_uuid()`, seperti `users`); uang `BIGINT` rupiah utuh dengan batas 0…10¹³ (aman bagi `Number` JavaScript).
- **`title`** tersimpan sudah dipangkas dan tanpa karakter kontrol (`CHECK`). Tag `binding:"omitempty,max=255"` pada `*string` **tidak** menolak `""` atau `"   "` (diuji), jadi usecase memangkas lalu memvalidasi sendiri; `confirm` pada hapus proyek dibandingkan dengan judul tersimpan, sehingga pengguna selalu bisa mengetiknya.
- **`timezone`** harus nama IANA kanonik. `CHECK` menjaga bentuk (awalan wilayah, huruf besar-kecil) dan pemicu `planner_check_timezone` menjaga keberadaannya. Alasannya: satu nilai rusak menggagalkan seluruh kueri Hub pengguna itu (`time zone "Foo/Bar" not recognized`), dan PostgreSQL menerima `asia/jakarta`, `ASIA/JAKARTA`, `UTC+7`, `Etc/GMT-7` yang ditolak Go atau tidak kanonik. Usecase tetap memanggil `time.LoadLocation` agar Go dan PostgreSQL sama-sama setuju (prasyarat tzdata: [kontrak bab 1.5](../00_KONTRAK_BERSAMA/README.md#15-prasyarat-runtime-dibaca-dari-kode-dan-dockerfile)).
- **`event_end_date`** untuk acara beberapa hari (seminar, konferensi, wisuda, upacara adat); `NULL` = satu hari. Tanpa kolom ini acara dua hari dianggap "telah berlangsung" pada hari kedua.
- **`is_sample` + `planner_projects_sample_idx`**: paling banyak satu proyek contoh per pengguna, dijamin basis data, bukan hanya usecase (dua klik bersamaan tidak membuat dua proyek).
- **`client_token`**: token yang dibuat wizard (UUID) saat dibuka; mengirim ulang permintaan yang sama (klik ganda, percobaan ulang setelah timeout) tidak membuat proyek kedua (`planner_projects_client_token_idx`).
- **`skipped_vendor_categories`**: kategori vendor yang ditandai "tidak diperlukan" (mis. tanpa Entertainment); keluar dari hitungan "x dari y kategori terkunci" sehingga progres bisa mencapai 100%.
- **`invitation_id`** tanpa FK sampai modul Undangan Digital ada ([05 BE bab 5](../05_BUKU_TAMU/BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital)).
- **Sesi:** nama unik per proyek (impor/ekspor memetakan sesi lewat nama), ikon 1–8 karakter. FK dari tabel modul lain ke sesi bertipe `DEFERRABLE INITIALLY DEFERRED` ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)): tanpa itu menghapus proyek yang punya skenario anggaran gagal.
- `prep_start_date` tanpa `DEFAULT CURRENT_DATE` (default itu memakai zona waktu server); tanpa indeks `event_type` (kardinalitas rendah, tidak dipakai filter). `CHECK` pada `event_type` harus selaras dengan registry ([kontrak bab 3](../00_KONTRAK_BERSAMA/README.md#3-registry-event-type-di-kode-bukan-basis-data)).

### 1.1 Kolom `metadata`

Kunci yang diizinkan per jenis acara didefinisikan di registry ([TEMPLATE_EVENT bab 5](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#5-kolom-metadata-proyek-planner_projectsmetadata)). Contoh pernikahan (urutan nama mengikuti PDF Hal. 1 "Nama Cewe & Nama Pria"):

```json
{
  "hero_title": "Bismillah Nikah",
  "bride_name": "Sinta Maharani",
  "groom_name": "Arif Pratama",
  "bride_parents": "Bpk. Bambang & Ibu Sri",
  "groom_parents": "Bpk. Hendra & Ibu Rina",
  "venue_name": "Sasana Kriya Ballroom",
  "city": "Jakarta Selatan"
}
```

Nilai teks dipangkas; teks opsional yang kosong setelah dipangkas dianggap tidak dikirim (kuncinya dibuang), karakter kontrol ditolak, panjang dihitung dalam karakter (rune), bilangan harus bulat dan dalam rentang `Min`/`Max` kolomnya (kolom jumlah seperti `graduate_count` dan `expected_participants` dibatasi 1.000.000). Usecase menolak kunci yang tidak ada di registry jenis acara proyek (`400 invalid_metadata: <kunci>`), memeriksa tipe dan panjang, dan memastikan kolom wajib jenis itu tetap terisi **setelah** penggabungan. `PATCH` memakai **JSON Merge Patch** ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)): kunci yang ada ditimpa, `null` menghapus, kunci yang tidak disebut dibiarkan. Dua tab yang mengedit kolom berbeda tidak saling menghapus.

---

## 2. Domain & DTO (Go)

Uang adalah `int64` rupiah utuh. Tanggal kalender (`DATE`) dipetakan ke `time.Time` pukul 00:00 UTC dan di JSON `"YYYY-MM-DD"` (bukan `civil.Date`, yang bukan dependensi di `go.mod`).

```go
package domain

type EventType string // "wedding" | "khitanan" | "aqiqah" | "peresmian" | "launching" | "birthday" | "wisuda" | "seminar" | "other"

type ProjectStatus string // "active" | "completed" | "archived"

type EventPhase string // "upcoming" | "ongoing" | "finished"

// PhaseInfo dihitung SQL/Go di zona waktu proyek (bab 4.2); FE tidak menghitung ulang dari jam perangkat.
type PhaseInfo struct {
	DaysRemaining int        `json:"days_remaining"` // event_date − hari ini; 0 atau negatif sejak Hari-H
	Phase         EventPhase `json:"phase"`          // upcoming: sebelum event_date; ongoing: event_date…akhir; finished: sesudahnya
	DayNumber     int        `json:"day_number"`     // hari ke-N saat ongoing, selain itu 0
	TotalDays     int        `json:"total_days"`     // 1 untuk acara satu hari
}

type Project struct {
	ID                      uuid.UUID
	UserID                  uuid.UUID
	EventType               EventType
	Title                   string
	EventDate               time.Time  // 00:00 UTC
	EventEndDate            *time.Time // nil = satu hari
	PrepStartDate           time.Time
	Timezone                string
	TargetBudget            int64
	SavingsTarget           *int64
	Status                  ProjectStatus
	IsSample                bool
	SkippedVendorCategories []string
	InvitationID            *uuid.UUID // nil = Mode Standalone (05 BE bab 5)
	Metadata                map[string]any
	CreatedAt, UpdatedAt    time.Time
}

// ---------- Request ----------
// Tanpa tag `binding`: semua validasi di usecase, supaya string kosong/berspasi sampai ke sana dan dijawab
// `<field>: <alasan>` (tag `binding:"required"` menjawab `invalid_json` tanpa nama kolom).

type InitProjectRequest struct {
	EventType     EventType      `json:"event_type"`
	Title         string         `json:"title"`      // dipangkas dan divalidasi usecase (bab 4.1)
	EventDate     string         `json:"event_date"` // YYYY-MM-DD
	EventEndDate  *string        `json:"event_end_date"`                // acara beberapa hari
	PrepStartDate *string        `json:"prep_start_date"`               // default: hari ini di zona waktu proyek
	Timezone      *string        `json:"timezone"`                      // IANA kanonik, default "Asia/Jakarta"
	TargetBudget  int64          `json:"target_budget"`
	Metadata      map[string]any `json:"metadata"`
	Templates     InitTemplates  `json:"templates"`
	ClientToken   *string        `json:"client_token"`                  // 8–64 karakter; dibuat wizard saat dibuka
}

// Hanya yang diminta yang dibuat. Sesi bawaan SELALU dibuat, tidak bergantung pada field ini.
// Template yang tidak dimiliki jenis acara, atau yang modulnya belum punya TemplateApplier -> 400 invalid_template.
type InitTemplates struct {
	TimelinePreset      *string `json:"timeline_preset"`       // mis. "wedding_pdf_44" | "wedding_essential_22" | "general"; null = tidak dibuat
	IncludeAdicaraTasks bool    `json:"include_adicara_tasks"` // tambahkan tugas wedding.adicara_* (02 BE §5.2)
	AdminDocs           bool    `json:"admin_docs"`
	BudgetLines         bool    `json:"budget_lines"`  // pos anggaran template (nominal 0 / persentase pagu)
	ShoppingList        bool    `json:"shopping_list"` // hanya wedding (37 item seserahan)
	Rundown             bool    `json:"rundown"`       // agenda template per sesi
	Playlist            bool    `json:"playlist"`
}

// PATCH: hanya field yang dikirim yang berubah. Pointer + flag Clear* karena JSON null dan "tidak dikirim" harus dibedakan.
type UpdateProjectRequest struct {
	Title                   *string        `json:"title"`
	EventDate               *string        `json:"event_date"`
	EventEndDate            *string        `json:"event_end_date"`
	ClearEventEndDate       bool           `json:"clear_event_end_date"`
	PrepStartDate           *string        `json:"prep_start_date"`
	Timezone                *string        `json:"timezone"`
	TargetBudget            *int64         `json:"target_budget"`
	SavingsTarget           *int64         `json:"savings_target"`
	ClearSavingsTarget      bool           `json:"clear_savings_target"` // kembali mengikuti target_budget
	SkippedVendorCategories *[]string      `json:"skipped_vendor_categories"`
	Metadata                map[string]any `json:"metadata"`             // JSON Merge Patch (null menghapus kunci)
	ShiftTasks              bool           `json:"shift_tasks"`          // geser tugas template ikut perubahan event_date (bab 4.3)
}

type ChangeProjectStatusRequest struct {
	Status ProjectStatus `json:"status"`
}

// ---------- Sesi (master bersama, kontrak bab 4) ----------

type CreateSessionRequest struct {
	Name         string  `json:"name"` // dipangkas; 1–100 karakter; tanpa karakter kontrol
	Icon         *string `json:"icon"`                    // 1–8 karakter; default "📋"
	SessionDate  *string `json:"session_date"`            // YYYY-MM-DD
	StartTime    *string `json:"start_time"`              // "HH:MM" atau "HH.MM"
	EndTime      *string `json:"end_time"`
	LocationName *string `json:"location_name"`
}
// scope_key dibuat server (kontrak bab 4). PUT memakai bentuk yang sama dan MENGGANTI PENUH:
// field opsional yang tidak dikirim dikosongkan (formulir selalu mengirim semuanya). scope_key tidak pernah berubah.
type UpdateSessionRequest = CreateSessionRequest

type ReorderSessionsRequest struct {
	SessionIDs []uuid.UUID `json:"session_ids"` // harus memuat SEMUA sesi non-sistem tepat sekali
}

type SessionBrief struct { // dipakai dashboard dan switcher; tanpa usage
	ID           uuid.UUID `json:"id"`
	ScopeKey     string    `json:"scope_key"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon"`
	IsSystem     bool      `json:"is_system"`
	SortOrder    int       `json:"sort_order"`
	SessionDate  *string   `json:"session_date,omitempty"`
	StartTime    *string   `json:"start_time,omitempty"` // "HH:MM" (to_char), bukan "07:00:00"
	EndTime      *string   `json:"end_time,omitempty"`
	LocationName *string   `json:"location_name,omitempty"`
}

type SessionResponse struct { // GET /sessions saja
	SessionBrief
	Usage struct { // jumlah baris yang menunjuk sesi ini; dipakai dialog hapus
		Tasks, Vendors, BudgetItems, ScenarioItems, ShoppingItems, Guests, Agendas int
	} `json:"usage"`
}

// ---------- Hub ----------

type ProjectCardResponse struct {
	ID             uuid.UUID      `json:"id"`
	EventType      EventType      `json:"event_type"`
	Title          string         `json:"title"`
	EventDate      string         `json:"event_date"`
	EventEndDate   *string        `json:"event_end_date,omitempty"`
	PhaseInfo
	Status         ProjectStatus  `json:"status"`
	IsSample       bool           `json:"is_sample"`
	TargetBudget   int64          `json:"target_budget"`
	TotalEstimate  int64          `json:"total_estimate"` // lunas = TotalEstimate > 0 && TotalPaid >= TotalEstimate
	TotalPaid      int64          `json:"total_paid"`
	TotalTasks     int            `json:"total_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	GuestPaxCount  int            `json:"guest_pax_count"`
	Metadata       map[string]any `json:"metadata"`
}

type ProjectLiteResponse struct { // GET /projects?view=lite — Project Switcher; tanpa agregat
	ID           uuid.UUID     `json:"id"`
	EventType    EventType     `json:"event_type"`
	Title        string        `json:"title"`
	EventDate    string        `json:"event_date"`
	EventEndDate *string       `json:"event_end_date,omitempty"`
	Status       ProjectStatus `json:"status"`
	IsSample     bool          `json:"is_sample"`
	DaysRemaining int          `json:"days_remaining"`
	Phase        EventPhase    `json:"phase"`
}

type PortfolioSummaryResponse struct {
	ActiveCount          int                   `json:"active_count"`           // tanpa proyek contoh
	CompletedCount       int                   `json:"completed_count"`
	ArchivedCount        int                   `json:"archived_count"`
	CombinedTargetBudget int64                 `json:"combined_target_budget"` // proyek aktif, tanpa contoh
	NearestProject       *ProjectCardResponse  `json:"nearest_project,omitempty"`
	Projects             []ProjectCardResponse `json:"projects"` // semua, termasuk contoh; tab, kategori, dan pencarian disaring klien
}
```

`NearestProject` ("Acara Terdekat"): dari `Projects` pilih yang `status = active`, bukan `is_sample`, dan `phase ≠ finished`, dengan `event_date` paling awal (acara yang sedang berlangsung otomatis menang atas yang belum mulai); `null` bila tidak ada. Jumlah pada tab status dihitung klien dari `Projects` (tab "Aktif (2)" menghitung proyek contoh, kartu "Acara Aktif 1" tidak: itu disengaja dan diberi catatan di FE).

Persentase progres tidak dikirim; klien memakai `percent(n, d)` dengan aturan pembagi nol ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)).

---

## 3. Endpoint

Semua di bawah `BearerAuth`. Endpoint dengan `:id` juga lewat `ProjectGuard` + `WriteGuard` ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)). Respons `{"data": ...}`.

### 3.1 Registry

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/v1/planner/event-types` | Registry 9 jenis acara ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)) + `templates_available`. Statis per versi rilis, `Cache-Control: public, max-age=3600`. |

### 3.2 Proyek

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/v1/planner/projects?status=active\|completed\|archived\|all&view=lite` | Tanpa `view`: kartu Hub (bab 5.3). `view=lite`: hanya `ProjectLiteResponse` untuk Project Switcher (tanpa agregat; dipanggil `PlannerLayout` pada setiap navigasi). `status` berlaku untuk kedua bentuk; `view` lain → `400 invalid_input: view`. |
| `GET` | `/v1/planner/projects/summary` | Metrik portofolio + kartu (proyek contoh tidak dihitung pada angka). |
| `POST` | `/v1/planner/projects` | `InitProjectRequest` → `201 {project, applied, created: true}`. Atomik (bab 4.1). Dengan `client_token` yang sudah dipakai → `200 {project, created: false}` tanpa menjalankan template. |
| `POST` | `/v1/planner/projects/sample` | Membuat Proyek Contoh (bab 4.5). Idempoten: `201 {project, created: true}` bila baru, `200 {project, created: false}` bila sudah ada. |
| `GET` | `/v1/planner/projects/:id` | Detail proyek + `shiftable_tasks` (jumlah tugas ber-`template_key` yang belum selesai; dipakai dialog geser jadwal). |
| `PATCH` | `/v1/planner/projects/:id` | `UpdateProjectRequest`. Respons menyertakan `warnings[]` dan `hints[]` (bab 4.3). |
| `PATCH` | `/v1/planner/projects/:id/status` | `ChangeProjectStatusRequest`. Lolos `WriteGuard` pada proyek `archived`. |
| `DELETE` | `/v1/planner/projects/:id?confirm=<judul>` | Hapus permanen (CASCADE). `confirm` harus sama persis dengan judul tersimpan, selain itu `400 invalid_input: confirm`. `204`. Lolos `WriteGuard` pada proyek `archived`. |
| `GET` | `/v1/planner/projects/:id/dashboard` | Agregasi widget (bab 5). 60 permintaan/menit/pengguna. |
| `PUT` | `/v1/planner/projects/:id/invitation` | Menautkan/melepas undangan digital (`{"invitation_id": uuid \| null}`). **Tidak terdaftar** sampai modul Undangan Digital ada ([05 BE bab 5](../05_BUKU_TAMU/BE.md#5-kontrak-antarmuka-dengan-modul-undangan-digital)). |

### 3.3 Sesi Acara (master sesi)

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/v1/planner/projects/:id/sessions` | Semua sesi (termasuk `all` dengan `is_system=true`) beserta `usage`. Dipakai FE untuk semua chip filter dan dropdown sesi. |
| `POST` | `/v1/planner/projects/:id/sessions` | `CreateSessionRequest` → `201`. `409 session_name_taken` / `session_limit_reached`. |
| `PUT` | `/v1/planner/projects/:id/sessions/:session_id` | Ganti penuh nama/ikon/tanggal/jam/lokasi. Sesi sistem `all` tidak bisa diubah (`400 invalid_input`). |
| `POST` | `/v1/planner/projects/:id/sessions/reorder` | `ReorderSessionsRequest`. |
| `DELETE` | `/v1/planner/projects/:id/sessions/:session_id?move_to=<scope_key>&delete_agendas=<bool>` | Algoritma di bab 4.6. Sesi sistem tidak bisa dihapus. |

### 3.4 Bentuk respons (sebagaimana diimplementasikan)

Semua sukses dibungkus `{"data": …}`; galat `{"error": {code, message, details?}}`. `details` hanya ada di `session_in_use` (`{"usage": {…}}`). Kontrak mesin: [`openapi.yaml`](../../../../backend/docs/openapi.yaml).

| Endpoint | `data` |
|---|---|
| `GET /event-types` | array jenis acara: `key, label, icon, lead_time_days, sessions[{scope_key,name,icon,same_day}], metadata_fields[{key,label,type,max_len?,min?,max?,required}], labels{guest,guest_menu}, templates_available{timeline_presets[], include_adicara_tasks, admin_docs, budget_lines, shopping_list, rundown, playlist}`. Kolom milik modul lain (kategori vendor/belanja, sisi tamu, peran, subjek dokumen, panduan, moment tag) ditambahkan modulnya saat dibangun. |
| `GET /projects` | array `ProjectCardResponse`, atau `ProjectLiteResponse` bila `view=lite` |
| `GET /projects/summary` | `PortfolioSummaryResponse`. Ketiga hitungan (aktif, selesai, arsip) tidak menyertakan proyek contoh; kartunya tetap ada di `projects` |
| `POST /projects`, `POST /projects/sample` | `{project, created, applied?}` (`201`; `200` bila token sudah dipakai atau contoh sudah ada, tanpa `applied`) |
| `GET /projects/:id` | `{project, shiftable_tasks}` |
| `PATCH /projects/:id` | `{project, warnings[], hints[]}` |
| `PATCH /projects/:id/status` | `{project}` |
| `DELETE /projects/:id` | `204` |
| `project` | `id, event_type, title, event_date, event_end_date, prep_start_date, timezone, days_remaining, phase, day_number, total_days, target_budget, savings_target, status, is_sample, skipped_vendor_categories, invitation_id, metadata, created_at, updated_at` |
| `GET /sessions` | array `SessionBrief` + `usage{tasks,vendors,budget_items,scenario_items,shopping_items,guests,agendas}` |
| `POST /sessions`, `PUT /sessions/:id` | `SessionBrief` (`201`/`200`) |
| `POST /sessions/reorder` | array `SessionBrief` semua sesi (termasuk `all`) dengan urutan baru |
| `DELETE /sessions/:id` | `204` |
| `GET /dashboard` | `WorkspaceDashboardResponse` (bab 5.1) |

Id di URL harus UUID kanonik 36 karakter; bentuk lain yang diterima `uuid.Parse` (`urn:uuid:…`, `{…}`, tanpa tanda hubung) dijawab `404` seperti id tak dikenal.

---

## 4. Logika Bisnis

### 4.1 Pembuatan proyek (`POST /v1/planner/projects`)

**Validasi** (urut; galat memakai `message` `<field>: <alasan>`):

1. `event_type` ada di registry.
2. `title`: `strings.TrimSpace`, 1–255 karakter (rune), tanpa karakter kontrol (`400 invalid_input: title`).
3. `timezone` (bawaan `Asia/Jakarta`): sesuai bentuk di `CHECK`, `time.LoadLocation` berhasil; galat pemicu basis data (`check_violation` "unknown time zone") dipetakan ke `400 invalid_timezone`.
4. Tanggal (`400 invalid_date: <field>`): `event_date` format `YYYY-MM-DD`, tidak lebih awal dari hari ini di zona proyek dan paling jauh 1.826 hari (5 tahun) ke depan; `prep_start_date` (bawaan hari ini) ≤ `event_date`; `event_end_date` opsional, ≥ `event_date` dan ≤ `event_date + 30` hari.
5. `target_budget` 0…10¹³.
6. `metadata` terhadap registry ([bab 1.1](#11-kolom-metadata)); `templates` terhadap `templates_available`.
7. `client_token` (bila ada) 8–64 karakter.
8. Batas: jumlah proyek pengguna (tanpa proyek contoh) < `PLANNER_MAX_PROJECTS` (bawaan 100) → `409 project_limit_reached`; 10 pembuatan/menit/pengguna (pola `ratelimit.go`).

**Transaksi** (`TxManager.InTx`; gagal di mana pun berarti tidak ada yang tersimpan, [kontrak bab 1.1.1](../00_KONTRAK_BERSAMA/README.md#111-transaksi-dan-template-lintas-modul)):

1. Bila ada `client_token`: `INSERT … ON CONFLICT (user_id, client_token) WHERE client_token IS NOT NULL DO NOTHING RETURNING id`; tidak ada baris → ambil proyek lama dan jawab `200 created:false` (template tidak dijalankan lagi).
2. **Seed sesi bawaan jenis acara + sesi sistem `all`** (tanpa syarat). Sesi `SameDay` lahir dengan `session_date = event_date` ([TEMPLATE_EVENT bab 3](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#3-ringkasan-registry-sembilan-jenis)); `scope_key` bawaan **ditulis eksplisit di registry** (`SessionSeed.ScopeKey`), bukan diturunkan dari nama: aturan penurunan akan menghasilkan `akad_nikah` untuk "Akad Nikah" padahal seluruh spesifikasi memakai `akad`. Aturan penurunan hanya untuk sesi buatan pengguna.
3. Untuk tiap template yang diminta, panggil `TemplateApplier.Apply(ctx, projectID, req)` modul terkait (mode `append`): jadwal (02), dokumen (02), pos anggaran (03), daftar belanja (04), agenda (06), playlist (06). Template **tidak** membuat pembayaran, vendor, tamu, atau setoran.
4. `201 {"data": {"project": …, "created": true, "applied": {"timeline": {"created": 50, "skipped": 0, "warnings": []}, …}}}`. `warnings` memuat mis. `lead_time_compressed` ([TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis)).

Urutan nyata di kode (`usecase/project.go`): setelah validasi 1–7, bila `client_token` sudah dipakai pengguna itu, jawaban `200 created:false` diberikan **sebelum** batas proyek diperiksa, sehingga percobaan ulang saat pengguna sudah di batas tetap mendapat proyeknya. Dua permintaan serentak dengan token yang sama diputuskan indeks unik: yang kalah mendapat proyek pemenang (`200`). Template dijalankan berurutan `timeline → admin_docs → budget → shopping → rundown → playlist` (pos anggaran harus ada sebelum belanja yang digulung ke dalamnya). `warnings` setiap hasil template selalu array (`[]`, bukan `null`).

Wizard FE langkah 3 hanya menampilkan template yang tersedia untuk jenis terpilih (`templates_available`), dan melewati langkah itu bila kosong.

### 4.2 Hitung mundur dan fase acara

Memotong hari dengan `time.Now().Truncate(24*time.Hour)` memakai batas UTC sehingga pada 00:00–06:59 WIB hasilnya lebih satu hari (diuji: 174 vs 173 pada 6 Okt 2026); rumus itu tidak dipakai. Yang dipakai menghitung tanggal kalender di zona proyek:

```go
// DaysUntil mengembalikan selisih hari kalender dari "hari ini" (zona tz) ke eventDate (DATE, 00:00 UTC).
func DaysUntil(now time.Time, tz string, eventDate time.Time) (int, error) {
	loc, err := time.LoadLocation(tz) // butuh data zona di runtime (kontrak bab 1.5)
	if err != nil {
		return 0, err
	}
	y, m, d := now.In(loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	ey, em, ed := eventDate.Date()
	event := time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC)
	return int(event.Sub(today).Hours() / 24), nil
}
```

**Fase** (`PhaseInfo`) dihitung di SQL (Hub, lihat bab 5.4) dan Go (dashboard) dengan satu definisi, `today` = tanggal di zona proyek, `end = COALESCE(event_end_date, event_date)`:

| Kondisi | `phase` | `day_number` | Tampilan hero |
|---|---|---|---|
| `today < event_date` | `upcoming` | 0 | `days_remaining` hari menuju Hari-H; ≤ 7 hari: `Hari : Jam : Menit : Detik` |
| `event_date ≤ today ≤ end` | `ongoing` | `today − event_date + 1` | acara 1 hari: "Hari-H telah tiba"; beberapa hari: "Hari ke-N dari M" |
| `today > end` | `finished` | 0 | "Acara telah berlangsung pada [tanggal]"; bila `status = active`, tawarkan "Tandai selesai" |

Diuji pada batas tengah malam WIB: 23:59 hari sebelumnya → `upcoming`; 00:00 Hari-H → `ongoing 1/1`; 23:59 Hari-H → masih `ongoing`; 00:00 hari berikutnya → `finished`; acara 3 hari: hari ke-2 → `ongoing 2/3`, hari terakhir 23:59 → `ongoing 3/3`, 00:00 berikutnya → `finished`; zona `UTC` pada 17:00Z masih `upcoming` (zona proyek yang menentukan, bukan server atau perangkat).

Dashboard menyertakan `server_time`; hitung mundur detik di FE dikoreksi dengan selisih `server_time − jam perangkat` sehingga perangkat dengan jam salah tidak menampilkan hari yang berbeda dari server.

### 4.3 Ubah tanggal acara dan geser jadwal

Tugas template menyimpan tanggal absolut, jadi mengganti `event_date` tidak otomatis menggeser jadwal. Aturan:

- `delta = event_date_baru − event_date_lama` (hari). Bila `event_end_date` tidak ikut dikirim, ia digeser `delta` juga (durasi acara tetap); bila dikirim dipakai apa adanya; `clear_event_end_date` menghapusnya.
- **FE bertanya sebelum menyimpan.** `GET /projects/:id` memberi `shiftable_tasks`; bila > 0 dialog "Geser N tugas template ikut X hari?" muncul **sebelum** `PATCH`, dan pilihan pengguna dikirim sebagai `shift_tasks`. Penggeseran berjalan di transaksi yang sama dengan perubahan tanggal (atomik): tidak ada keadaan setengah jadi, dan klik ganda tidak menggeser dua kali (versi "geser setelah disimpan" lewat panggilan terpisah bersifat relatif sehingga bisa dobel).
- `shift_tasks = true`: `UPDATE planner_tasks SET start_date = start_date + $delta, end_date = end_date + $delta WHERE project_id = $1 AND status <> 'selesai' AND template_key IS NOT NULL`. Tugas selesai dan tugas kustom tidak disentuh. Tidak ada pemampatan; bila ada tugas yang kini lewat tenggat respons memuat warning `tasks_overdue_after_shift` (`count`).
- `shift_tasks = false` dan ada tugas template: `hints: [{"code": "tasks_not_shifted", "count": n, "shift_days": delta}]` untuk klien API; FE resmi tidak memakainya karena sudah bertanya.
- Sesi yang `session_date`-nya sama dengan Hari-H lama ikut pindah ke Hari-H baru dalam transaksi yang sama (`UPDATE planner_event_sessions SET session_date = $baru WHERE project_id = $1 AND session_date = $lama`; diuji); sesi bertanggal lain (mis. Lamaran, Ngunduh Mantu) tidak diubah dan respons memuat hint `sessions_dates_unchanged` (`count`).
- Validasi `PATCH`: `prep_start_date ≤ event_date`, `event_end_date ≥ event_date`, dan `event_end_date ≤ event_date + 30` hari (batas yang sama dengan pembuatan, supaya tidak bisa dilewati lewat PATCH) → `400 invalid_date`; tanggal lampau diperbolehkan untuk koreksi data. `clear_event_end_date` bersama `event_end_date`, atau `clear_savings_target` bersama `savings_target`, → `400 invalid_input`. PATCH berjalan di bawah `SELECT … FOR UPDATE` pada baris proyek, jadi dua tab tidak saling menimpa. Notifikasi dipisah di respons: `warnings[]` (`tasks_overdue_after_shift`) dan `hints[]` (`tasks_not_shifted`, `sessions_dates_unchanged`).
- `POST /projects/:id/tasks/shift` (02 BE §3.1) adalah aksi manual dari layar Timeline dan **tidak idempoten** (setiap panggilan menggeser); menu 01 tidak memakainya.

### 4.4 Status proyek

`active → completed → archived` dan kembali, tanpa syarat tambahan. `archived` baca-saja kecuali `PATCH .../status` dan `DELETE` ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)). `completed` tetap bisa diedit. Tidak ada perubahan status otomatis; proyek `active` yang `phase = finished` hanya diberi ajakan "Tandai selesai".

### 4.5 Proyek Contoh

`POST /v1/planner/projects/sample` (idempoten per pengguna; dilindungi `planner_projects_sample_idx`): bila sudah ada → `200` dengan proyek itu. Selain itu membuat wedding "Contoh: Pernikahan Sinta & Arif" (`is_sample=true`, `event_date = hari ini + 175`, `target_budget = 75.000.000`, metadata PDF dengan `hero_title: "Bismillah Nikah"`), menerapkan semua template wedding (`wedding_pdf_44` + tugas Adicara, 15 dokumen, 14 pos anggaran, 37 belanja, rundown 11 + 22, 30 lagu), lalu overlay dataset PDF: tanggal sesi (Lamaran = tanggal mulai tugas `wedding.lamaran`, Akad dan Resepsi = `event_date`), pembayaran dan tabungan (03 BE §6.3–6.4), status tugas/dokumen/belanja (02 BE, 04 BE), vendor contoh (04 BE §5.2), tamu contoh (05 BE §6). Overlay hanya dijalankan oleh endpoint ini dan test; tidak ada tombol "reset ke contoh" pada proyek nyata. Implementasi: `SampleOverlay` (antarmuka baru di `planner/domain`, `ApplySample(ctx, project)`) didaftarkan modul lewat `RegisterSampleOverlays` dan dijalankan di dalam transaksi yang sama; template yang dipakai hanyalah yang punya applier terdaftar. **Selama modul 02–06 belum dibangun, Proyek Contoh hanya berisi proyek, sesi, dan metadata** (tanpa tugas, anggaran, vendor, tamu). Pembuatan dua kali berbalapan menghasilkan satu proyek (`201` + `200`, diuji). Contoh bersifat statis terhadap hari pembuatannya: setelah 175 hari ia tampak "telah berlangsung" dengan tugas terlambat; pengguna menghapusnya dan membuat ulang dari Hub.

### 4.6 Sesi acara (master sesi)

**Buat** (`POST`): memangkas dan memvalidasi `name` (1–100, tanpa karakter kontrol), memeriksa batas 20 sesi non-sistem (`session_limit_reached`), membuat `scope_key` sesuai [kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions), `sort_order = max + 1`. Pelanggaran indeks nama → `409 session_name_taken`; pelanggaran `scope_key` karena balapan → coba ulang (maksimal 3 kali).

**Urut ulang** (`reorder`): daftar harus memuat semua sesi non-sistem proyek tepat sekali (`400 invalid_input` bila ada yang asing, ganda, atau hilang); `sort_order` = 1…n; sesi `all` tetap 0.

**Hapus** (`DELETE`), satu transaksi:

1. `SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE` (menserialkan operasi sesi dengan pembuatan baris yang menunjuk sesi).
2. Tolak sesi sistem (`400`), `move_to` tidak ada atau sama dengan sesi yang dihapus (`400 invalid_scope`).
3. Hitung agenda rundown di sesi itu; ada dan `delete_agendas ≠ true` → `409 session_in_use` beserta `usage` (tidak ada yang diubah).
4. Jalankan pemindahan di bawah, lalu hapus. Agenda ikut terhapus oleh `ON DELETE CASCADE` bila `delete_agendas = true`.
5. `COMMIT` (FK ditunda diperiksa di sini; galat `23503` → `409 conflict`, aman diulang).

```sql
-- session-move: $1 = project_id, $2 = scope_key sesi yang dihapus, $3 = scope_key tujuan (satu transaksi)
UPDATE planner_tasks          SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2;
UPDATE planner_vendors        SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2;
UPDATE planner_shopping_items SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2;
UPDATE planner_guests         SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2;
UPDATE planner_budget_items b
SET sub_event_scope = $3, updated_at = now(),
    item_key = CASE WHEN b.item_key IS NOT NULL AND EXISTS (
                 SELECT 1 FROM planner_budget_items o
                 WHERE o.project_id = b.project_id AND o.sub_event_scope = $3 AND o.item_key = b.item_key)
               THEN NULL ELSE b.item_key END
WHERE b.project_id = $1 AND b.sub_event_scope = $2;
UPDATE planner_budget_scenario_items s
SET sub_event_scope = $3,
    item_key = CASE WHEN s.item_key IS NOT NULL AND EXISTS (
                 SELECT 1 FROM planner_budget_scenario_items o
                 WHERE o.scenario_id = s.scenario_id AND o.sub_event_scope = $3 AND o.item_key = s.item_key)
               THEN NULL ELSE s.item_key END
WHERE s.project_id = $1 AND s.sub_event_scope = $2;
UPDATE planner_budget_scenarios
SET active_sub_events = array_remove(active_sub_events, $2)
        || CASE WHEN $3 <> 'all' AND NOT ($3 = ANY(active_sub_events)) THEN ARRAY[$3] ELSE ARRAY[]::text[] END,
    updated_at = now()
WHERE project_id = $1 AND $2 = ANY(active_sub_events);
DELETE FROM planner_event_sessions WHERE project_id = $1 AND scope_key = $2 AND NOT is_system;
```

Di kode, setiap pernyataan di atas dijalankan sebagai satu `Exec` dengan argumen yang benar-benar dipakainya: pgx (mode simple protocol) menolak argumen berlebih dengan `unused argument`, jadi `DELETE` hanya menerima `$1` dan `$2`. Seluruhnya berjalan dalam satu transaksi di bawah kunci proyek.

Mengapa bukan satu `UPDATE` sederhana (ketiganya diuji pada data golden): (a) pos anggaran Lamaran dan Resepsi sama-sama ber-`item_key = 'catering'`, sehingga memindahkannya begitu saja melanggar indeks unik `(project_id, sub_event_scope, item_key)` (`23505`); pos bentrok menjadi `item_key = NULL`; (b) item skenario juga menunjuk sesi tetapi tidak termasuk dalam hitungan `usage` versi awal sehingga penghapusan gagal pada FK; (c) `active_sub_events` adalah array tanpa FK dan akan menyimpan sesi yang sudah tidak ada. Total estimasi dan pembayaran tidak berubah oleh pemindahan (diuji: Rp 74.150.000 sebelum dan sesudah).

---

## 5. Dashboard (`GET /v1/planner/projects/:id/dashboard`)

Enam widget, tiap widget disuplai satu `SummaryProvider` ([kontrak bab 7](../00_KONTRAK_BERSAMA/README.md#7-dashboard-pola-summaryprovider): `recover` per goroutine, paralelisme 3, batas waktu 3 detik). Susunan enam widget disamakan di README, FE, dan BE:

| # | Widget | Provider (modul) | Menu |
|---|---|---|---|
| 1 | Timeline & Tugas | `timeline` (`plannertimeline`) | 02 Tab 1 |
| 2 | Keuangan & Tabungan | `finance` (`plannerfinance`) | 03 |
| 3 | Vendor Rekanan | `vendors` (`plannervendor`) | 04 Tab 1 |
| 4 | Buku Tamu & RSVP | `guests` (`plannerguest`) | 05 |
| 5 | Legalitas / Perizinan | `admin_docs` (`plannertimeline`) | 02 Tab 2 |
| 6 | Rundown Hari-H | `rundown` (`plannerrundown`) | 06 |

### 5.1 Response

```go
type WorkspaceDashboardResponse struct {
	ServerTime time.Time `json:"server_time"` // untuk mengoreksi jam perangkat pada hitung mundur detik

	Project struct {
		ID               uuid.UUID
		EventType        EventType
		Title            string
		EventDate        string
		EventEndDate     *string
		PhaseInfo                       // days_remaining, phase, day_number, total_days
		Status           ProjectStatus
		IsSample         bool
		InvitationLinked bool           // false = Mode Standalone
		TargetBudget     int64
		SavingsTarget    *int64
		Timezone         string
		Metadata         map[string]any
	} `json:"project"`

	Sessions []SessionBrief `json:"sessions"` // tanpa 'all', tanpa usage (usage 7 hitungan/sesi terlalu berat untuk polling 30 detik)

	// Setiap widget bernilai null bila providernya belum terdaftar, atau {"error": "<code>"} bila gagal (internal | timeout).
	Timeline  *TimelineSummary  `json:"timeline"`
	Finance   *FinanceSummary   `json:"finance"`
	Vendors   *VendorSummary    `json:"vendors"`
	Guests    *GuestSummary     `json:"guests"`
	AdminDocs *AdminDocSummary  `json:"admin_docs"`
	Rundown   *RundownSummary   `json:"rundown"`
}

type TimelineSummary struct {
	Total, Completed, InProgress, Overdue int
	Urgent []struct { // paling banyak 3: belum selesai, tenggat ≤ hari ini + 7 (termasuk yang sudah lewat, DaysLeft negatif), urut tenggat
		ID uuid.UUID; Title, DueDate string; DaysLeft int
	}
}

type FinanceSummary struct {
	TargetBudget     int64 // planner_projects.target_budget
	TotalEstimate    int64 // Σ planner_budget_lines.estimate
	TotalPaid        int64
	Remaining        int64 // Σ max(estimate − paid, 0): kewajiban nyata
	Overpaid         int64 // Σ max(paid − estimate, 0): kelebihan bayar pada pos tertentu
	SavingsTarget    int64 // COALESCE(savings_target, target_budget)
	SavingsCollected int64
	FreeCash         int64 // SavingsCollected − TotalPaid; negatif = kas kurang
	CashDeficit      int64 // max(Remaining − FreeCash, 0); > 0 = peringatan defisit
	LineStatus       struct{ Settled, Partial, Unpaid int } // hanya pos dengan estimate > 0
}

type VendorSummary struct {
	CategoriesTotal int // |kategori registry ∪ kustom yang dipakai (status <> batal)| dikurangi yang ditandai tidak diperlukan
	CategoriesFixed int // kategori (di luar yang dilewati) yang punya ≥ 1 vendor fix
	Skipped         int // jumlah kategori ditandai "tidak diperlukan"
	FixedVendors    int
	Pipeline        int // rencana + negosiasi
}

type GuestSummary struct {
	Invitations int
	TotalPax    int
	Sides       []struct{ Key *string; Label string; Invitations, Pax int } // dari data; Key nil = belum ditentukan
	Sent        int
	RSVP        struct{ Attending, Declined, Maybe, Unconfirmed, AttendingPax int } // empat status, jumlahnya = Invitations
}

type AdminDocSummary struct {
	Label     string   // diisi provider Menu 02, bukan registry (registry hanya punya Labels.Guest/GuestMenu); mis. "Kelengkapan Berkas KUA" untuk wedding
	Completed, Total int // Total 0 (mis. jenis "other") -> kartu kosong, tanpa persen
	Pending   []string // maks 5 nama berkas yang belum
}

type RundownSummary struct {
	Sessions, Agendas, Completed, InProgress, Songs int
	CurrentAgenda *struct{ Activity, Session string } // agenda dengan status in_progress tertua (hari-H)
	NextSession   *struct{ Name, Date string; StartTime *string } // StartTime nil bila jam belum diisi
}
```

Tipe ringkasan widget (`TimelineSummary` … `RundownSummary`) ada di `planner/domain/summary.go` dengan JSON `snake_case` (`in_progress`, `days_left`, `cash_deficit`, `current_agenda`, …) dan dikembalikan provider modul masing-masing; menu 01 hanya menyalinnya ke respons (`any`) dan tidak membaca isinya. Widget tanpa provider `null`; provider gagal/panic → `{"error":"internal"}`, terlalu lambat → `{"error":"timeout"}`. `server_time` selalu UTC. Batas waktu dashboard 3 detik dan paralelisme 3 dijaga di `usecase/dashboard.go`; provider yang mengabaikan `ctx` tidak menahan respons melewati batas.

Catatan: kelebihan bayar pada satu pos tidak boleh mengurangi kewajiban pos lain, sehingga `Remaining` memakai bagian positif per pos (jumlah mentah `Σ estimate − Σ paid` akan menyembunyikan kewajiban saat ada pos yang dibayar lebih; pada data PDF `Overpaid = 0` dan keduanya sama, 40.600.000). `NextSession.StartTime` dan `SessionBrief` memakai pointer karena `NULL` tidak dapat dipindai ke `string` (diuji).

### 5.2 Query per provider

`$1` = `project_id`, `$2` = zona waktu proyek. "Hari ini" selalu `(now() AT TIME ZONE $2)::date`. Kueri provider ini milik modul 02–06 dan **belum diimplementasikan** (hanya kueri Hub di bab 5.3–5.4 yang sudah menjadi kode). Modul yang mengimplementasikannya memakai waktu sebagai parameter (`$3::timestamptz`) seperti kueri Hub, bukan `now()`, supaya batas tengah malam bisa diuji; bentuk itu untuk `timeline` dan `admin_docs` ada di [02 BE §4.6](../02_TIMELINE_DAN_TUGAS/BE.md#46-sambungan-ke-menu-01) (dijalankan terhadap data Proyek Contoh).

**timeline**
```sql
SELECT count(*) AS total,
       count(*) FILTER (WHERE status = 'selesai') AS completed,
       count(*) FILTER (WHERE status = 'proses')  AS in_progress,
       count(*) FILTER (WHERE status <> 'selesai' AND end_date < (now() AT TIME ZONE $2)::date) AS overdue
FROM planner_tasks WHERE project_id = $1;

SELECT id, title, end_date, end_date - (now() AT TIME ZONE $2)::date AS days_left
FROM planner_tasks
WHERE project_id = $1 AND status <> 'selesai' AND end_date <= (now() AT TIME ZONE $2)::date + 7
ORDER BY end_date, sort_order LIMIT 3;
```

**finance**
```sql
SELECT COALESCE(sum(estimate), 0) AS estimate, COALESCE(sum(paid), 0) AS paid,
       COALESCE(sum(GREATEST(remaining, 0)), 0)  AS remaining,
       COALESCE(sum(GREATEST(-remaining, 0)), 0) AS overpaid,
       count(*) FILTER (WHERE estimate > 0 AND payment_status = 'settled') AS settled,
       count(*) FILTER (WHERE estimate > 0 AND payment_status IN ('dp_paid', 'partial')) AS partial,
       count(*) FILTER (WHERE estimate > 0 AND payment_status = 'unpaid') AS unpaid
FROM planner_budget_lines WHERE project_id = $1;

SELECT COALESCE(sum(amount), 0) FROM planner_savings_entries WHERE project_id = $1;
```

**vendors** (`CategoriesTotal`/`CategoriesFixed` dihitung di Go dari `categories_in_use`, `categories_fixed`, registry, dan `project.SkippedVendorCategories`; kueri dijalankan terhadap data emas Menu 04: `fixed_vendors 7, pipeline 3`, 10 kategori terpakai, 7 fix. Menu 04 menghitung KPI yang sama dengan `BuildOverview`, [04 BE §2.8](../04_VENDOR_DAN_BELANJA/BE.md#28-kategori-kpi-dan-ringkasan); vendor `batal` tidak dihitung di mana pun). **Diimplementasikan 2026-10-23:** provider `vendors` memakai `BuildOverview` itu (bukan kueri di atas), jadi widget menghitung persis yang dihitung tab; pada Proyek Contoh hasilnya 10 / 7 / 0 / 7 / 3, diuji `TestSampleProjectVendorGolden`)
```sql
SELECT count(*) FILTER (WHERE status = 'fix') AS fixed_vendors,
       count(*) FILTER (WHERE status IN ('rencana', 'negosiasi')) AS pipeline,
       COALESCE(array_agg(DISTINCT category_key) FILTER (WHERE status <> 'batal'), '{}') AS categories_in_use,
       COALESCE(array_agg(DISTINCT category_key) FILTER (WHERE status = 'fix'), '{}') AS categories_fixed
FROM planner_vendors WHERE project_id = $1;
```

**guests**
```sql
SELECT count(*) AS invitations, COALESCE(sum(pax_count), 0) AS total_pax,
       count(*) FILTER (WHERE invitation_sent_status <> 'unsent') AS sent,
       count(*) FILTER (WHERE rsvp_status = 'attending')   AS attending,
       count(*) FILTER (WHERE rsvp_status = 'declined')    AS declined,
       count(*) FILTER (WHERE rsvp_status = 'maybe')       AS maybe,
       count(*) FILTER (WHERE rsvp_status = 'unconfirmed') AS unconfirmed,
       COALESCE(sum(CASE WHEN rsvp_status = 'attending' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0) AS attending_pax
FROM planner_guests WHERE project_id = $1;

SELECT guest_side, count(*) AS invitations, sum(pax_count) AS pax
FROM planner_guests WHERE project_id = $1 GROUP BY guest_side ORDER BY guest_side NULLS LAST;
```

**admin_docs**
```sql
SELECT count(*) AS total, count(*) FILTER (WHERE status = 'selesai') AS completed,
       COALESCE((SELECT array_agg(document_name ORDER BY sort_order) FROM
                  (SELECT document_name, sort_order FROM planner_admin_docs
                   WHERE project_id = $1 AND status = 'belum' ORDER BY sort_order LIMIT 5) x), '{}') AS pending
FROM planner_admin_docs WHERE project_id = $1;
```

**rundown**
```sql
SELECT (SELECT count(*) FROM planner_event_sessions WHERE project_id = $1 AND NOT is_system) AS sessions,
       count(*) AS agendas,
       count(*) FILTER (WHERE status = 'completed')   AS completed,
       count(*) FILTER (WHERE status = 'in_progress') AS in_progress,
       (SELECT count(*) FROM planner_playlists WHERE project_id = $1) AS songs
FROM planner_rundowns WHERE project_id = $1;

SELECT r.activity_name, s.name AS session_name
FROM planner_rundowns r JOIN planner_event_sessions s ON s.id = r.session_id
WHERE r.project_id = $1 AND r.status = 'in_progress'
ORDER BY r.started_at NULLS LAST, r.sort_order LIMIT 1;

SELECT name, to_char(session_date, 'YYYY-MM-DD') AS session_date, to_char(start_time, 'HH24:MI') AS start_time
FROM planner_event_sessions
WHERE project_id = $1 AND NOT is_system AND session_date >= (now() AT TIME ZONE $2)::date
ORDER BY session_date, start_time NULLS LAST, sort_order LIMIT 1;
```

### 5.3 Query Hub

Satu kueri; `phase`/`day_number` dihitung di SQL dengan definisi bab 4.2. Parameter: `$1` = id pengguna, `$2` = `status` (`all` atau satu status), `$3` = waktu sekarang dari usecase. Waktu dikirim sebagai parameter, bukan `now()`, supaya batas tengah malam dapat diuji deterministik; hasilnya sama dengan `now()` di produksi. Tes membandingkan kolom `phase`/`days_remaining`/`day_number`/`total_days` dengan `ComputePhase` (Go) tiap 30 menit pada 5 zona waktu. Urutan: aktif dulu (tenggat terdekat lebih dulu), lalu selesai, lalu arsip (terbaru lebih dulu).

```sql
SELECT p.id, p.event_type, p.title, p.event_date, p.event_end_date, p.status, p.is_sample,
       p.target_budget, p.metadata,
       d.days_remaining, d.phase, d.day_number, d.total_days,
       COALESCE(f.estimate, 0) AS total_estimate, COALESCE(f.paid, 0) AS total_paid,
       COALESCE(t.total, 0) AS total_tasks, COALESCE(t.completed, 0) AS completed_tasks,
       COALESCE(g.pax, 0) AS guest_pax
FROM planner_projects p
CROSS JOIN LATERAL (SELECT ($3::timestamptz AT TIME ZONE p.timezone)::date AS today) n
CROSS JOIN LATERAL (SELECT p.event_date - n.today AS days_remaining,
       CASE WHEN n.today < p.event_date THEN 'upcoming'
            WHEN n.today <= COALESCE(p.event_end_date, p.event_date) THEN 'ongoing'
            ELSE 'finished' END AS phase,
       CASE WHEN n.today BETWEEN p.event_date AND COALESCE(p.event_end_date, p.event_date)
            THEN n.today - p.event_date + 1 ELSE 0 END AS day_number,
       COALESCE(p.event_end_date, p.event_date) - p.event_date + 1 AS total_days) d
LEFT JOIN LATERAL (SELECT sum(estimate)::bigint AS estimate, sum(paid)::bigint AS paid
                   FROM planner_budget_lines WHERE project_id = p.id) f ON true
LEFT JOIN LATERAL (SELECT count(*) AS total, count(*) FILTER (WHERE status = 'selesai') AS completed
                   FROM planner_tasks WHERE project_id = p.id) t ON true
LEFT JOIN LATERAL (SELECT sum(pax_count)::bigint AS pax FROM planner_guests WHERE project_id = p.id) g ON true
WHERE p.user_id = $1 AND ($2 = 'all' OR p.status = $2)
ORDER BY CASE p.status WHEN 'active' THEN 0 WHEN 'completed' THEN 1 ELSE 2 END,
         CASE WHEN p.status = 'active' THEN p.event_date END ASC,
         p.event_date DESC, p.id
```

### 5.4 Query ringan (Project Switcher)

`GET /v1/planner/projects?view=lite` tidak menghitung agregat sehingga aman dipanggil pada setiap navigasi antarmenu. Parameter sama dengan bab 5.3; `status` ikut menyaring (tanpa parameter = `all`):

```sql
SELECT p.id, p.event_type, p.title, p.event_date, p.event_end_date, p.status, p.is_sample,
       d.days_remaining, d.phase
FROM planner_projects p
CROSS JOIN LATERAL (SELECT ($3::timestamptz AT TIME ZONE p.timezone)::date AS today) n
CROSS JOIN LATERAL (SELECT p.event_date - n.today AS days_remaining,
       CASE WHEN n.today < p.event_date THEN 'upcoming'
            WHEN n.today <= COALESCE(p.event_end_date, p.event_date) THEN 'ongoing'
            ELSE 'finished' END AS phase) d
WHERE p.user_id = $1 AND ($2 = 'all' OR p.status = $2)
ORDER BY CASE p.status WHEN 'active' THEN 0 WHEN 'completed' THEN 1 ELSE 2 END,
         CASE WHEN p.status = 'active' THEN p.event_date END ASC,
         p.event_date DESC, p.id
```

---

## 6. Pengujian

Status per 2026-10-21 (PostgreSQL 16.2, `go test -p 1`; tes DB butuh `TEST_DATABASE_URL` ke database `…_test` dan di-skip tanpa itu). Berkas: `internal/modules/planner/{domain,usecase,repository,delivery/http}/*_test.go`, `internal/migrate/migrate_test.go`, `internal/db/db_test.go`.

| Butir | Status | Tes |
|---|---|---|
| Siklus hidup: hapus proyek/akun berisi skenario, kontributor, belanja, rundown; hapus sesi terpakai ditolak saat commit | ✔ otomatis | `TestDeleteProjectWithEverythingAttachedSucceeds`, `TestDeletingTheUserDeletesTheirProjects`, `TestDeletingASessionThatIsStillUsedFailsAtCommit` |
| Constraint: judul, zona (`asia/jakarta`, `UTC+7`, `Local`, `Etc/GMT-7`, `Asia/Nowhere` lewat pemicu), uang > 10¹³, tanggal, contoh ganda, nama sesi ganda, `client_token` berulang | ✔ otomatis | `TestProjectConstraintsRejectBadRows`, `TestUnknownTimezoneIsCaughtByTheTrigger`, `TestSampleIsUniquePerUserAndNotCounted`, `TestSessionsInsertListUpdateReorder` |
| Pindah sesi: bentrok `item_key`, item skenario, `active_sub_events`, total tak berubah, pindah ke `all`, agenda ikut terhapus | ✔ otomatis | `TestMoveSessionDataKeepsTotalsAndNeutralisesKeyCollisions`, `TestMoveSessionDataToAllAndRemovesAgendas`, `TestSessionUsageCountsEveryKindOfRow` |
| Fase acara pada batas tengah malam (1 dan 3 hari, zona proyek ≠ server); SQL Hub = Go | ✔ otomatis | `TestComputePhase…`, `TestHubPhaseMatchesComputePhaseAtEveryHalfHour` (>1.000 perbandingan, 5 zona) |
| `DaysUntil` dan kueri Hub: 00:00, 06:59, 07:00, 23:59 WIB sama | ✔ otomatis | `TestDaysUntilIsStableAcrossTheWIBDay`, `TestHubAtMidnightBoundaryInJakarta` |
| Pembuatan atomik; template tanpa applier → `invalid_template` | ✔ sebagian | Rollback dibuktikan dengan applier palsu (`TestCreateProjectIsAtomicWhenATemplateFails`, TxManager palsu) dan pada PostgreSQL nyata (`TestTxManagerRollsBackJoinsAndCommits`). **Belum** dengan applier modul sungguhan (belum ada) |
| Isolasi: pengguna B mendapat `404` di setiap endpoint proyek/sesi A; id non-UUID → `404`; sesi proyek lain tak bisa dipakai | ✔ otomatis | `TestIsolationBetweenUsersOnEveryEndpoint` |
| Provider: panic tidak mematikan proses, provider lambat → `timeout`, paralelisme ≤ 3 | ✔ otomatis | `TestDashboardMergesProvidersAndKeepsGoingWhenOneFails`, `…SlowProviderTimesOut…`, `…AtMostThreeProvidersAtOnce`, `TestDashboardOverHTTPWithProviders` |
| `POST …/sample` berbalapan → satu proyek (`201` + `200`) | ✔ otomatis | `TestSampleRacesProduceOneProject` (8 permintaan serentak), `TestCreateSampleIsIdempotent…` |
| Registry ↔ `CHECK(event_type)` tidak menyimpang | ✔ otomatis | `TestEventTypeCheckMatchesTheRegistry` (membaca `pg_get_constraintdef`) |
| Migrasi 005–012 naik, turun, naik | ✔ otomatis | `TestPlannerMigrationsRollBackAndReapply` |
| Proyek Contoh memenuhi tabel uji emas ([kontrak bab 9](../00_KONTRAK_BERSAMA/README.md#9-urutan-pembangunan-dan-uji-penerimaan)) | ✘ belum | Menunggu modul 02–06 (data anggaran, tabungan, tamu, rundown) |
| Latensi dashboard p95 < 100 ms | ✘ belum diukur | Menunggu provider |

Tambahan di luar daftar awal: ubah tanggal menggeser sesi dan tugas template secara atomik dan PATCH yang gagal validasi tidak menggeser apa pun (`TestShiftTasksOverHTTPIsAtomicWithTheDateChange`); pembatas 10/menit per pengguna; `client_token` serentak; arsip baca-saja kecuali status dan hapus; `PUT …/invitation` belum terdaftar (404).
