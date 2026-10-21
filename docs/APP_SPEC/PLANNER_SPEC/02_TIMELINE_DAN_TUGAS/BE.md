# Menu 02: Timeline & Tugas Persiapan — Backend Specification

Spesifikasi backend modul Go `plannertimeline` (`backend/internal/modules/plannertimeline/`) untuk **jadwal/tugas persiapan** (PDF Hal. 3) dan **dokumen legalitas/perizinan** (PDF Hal. 11). Aturan lintas modul ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

---

## 1. Skema Database

Migrasi `007_000_planner_timeline.sql` (sudah dijalankan di PostgreSQL 16; kolom `vendor_id` mengarah ke `planner_vendors` dari migrasi 006):

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
- `duration_days` adalah **kolom generated** `GREATEST(end_date - start_date, 1)` mengikuti konvensi PDF Hal. 3 ("1 Feb → 2 Feb" = "1 Hari"); ia tidak bisa menyimpang dari tanggal.
- `sub_event_scope` terhubung ke master sesi lewat FK komposit ([kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)); tidak ada konstanta Go untuk nilai scope karena nilainya berasal dari sesi proyek.
- `phase_name` ikut tersimpan pada tugas, jadi jenis acara non-pernikahan punya nama fase sendiri; `vendor_id`, `template_key`, `origin` mendukung tautan vendor dan mode *append*.
- Peran orang tua memakai `ortu_cpp`/`ortu_cpw` (PDF menulis CMP/CMW untuk makna yang sama). Status "tersinkron ke anggaran" diturunkan dari `planner_budget_items.admin_doc_id` (§4.5), bukan disimpan di dokumen. Uang `BIGINT`.
- Mode *append* mencocokkan `template_key` karena judul PDF berulang ("Pertemuan keluarga" ×5).

---

## 2. Domain & DTO (Go)

```go
package domain

type TaskStatus string // "belum" | "proses" | "selesai"

type Task struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	SubEventScope string     `json:"sub_event_scope"` // scope_key dari master sesi; "all" = umum
	Category      string     `json:"category"`
	PhaseOrder    int        `json:"phase_order"`
	PhaseName     string     `json:"phase_name"`
	StartDate     string     `json:"start_date"` // YYYY-MM-DD
	EndDate       string     `json:"end_date"`
	DurationDays  int        `json:"duration_days"`
	Status        TaskStatus `json:"status"`
	PIC           string     `json:"pic"`
	VendorID      *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName    *string    `json:"vendor_name,omitempty"` // JOIN planner_vendors
	SortOrder     int        `json:"sort_order"`
	Notes         string     `json:"notes"`
	TemplateKey   *string    `json:"template_key,omitempty"`
	Origin        string     `json:"origin"` // "pdf" | "adicara" | "custom"
	Overdue       bool       `json:"overdue"` // status <> selesai dan end_date < hari ini (zona proyek); dihitung server
}

type CreateTaskRequest struct {
	Title         string     `json:"title" binding:"required,max=255"`
	SubEventScope string     `json:"sub_event_scope"` // default "all"; harus ada di master sesi (400 invalid_scope)
	Category      string     `json:"category"`
	PhaseOrder    int        `json:"phase_order"`
	PhaseName     string     `json:"phase_name"`
	StartDate     string     `json:"start_date" binding:"required"`
	EndDate       string     `json:"end_date" binding:"required"` // >= start_date, selain itu 400 invalid_date
	PIC           string     `json:"pic"`
	VendorID      *uuid.UUID `json:"vendor_id"`
	Notes         string     `json:"notes"`
}

type UpdateTaskRequest struct { // semua opsional (PATCH)
	Title, SubEventScope, Category, PhaseName, StartDate, EndDate, PIC, Notes *string
	PhaseOrder *int
	Status     *TaskStatus
	VendorID   *uuid.UUID
	ClearVendor bool `json:"clear_vendor"`
}

type UpdateTaskStatusRequest struct {
	Status TaskStatus `json:"status" binding:"required,oneof=belum proses selesai"`
}

type ReorderRequest struct { IDs []uuid.UUID `json:"ids" binding:"required"` } // tugas atau dokumen

type ShiftTasksRequest struct {
	Days int `json:"days" binding:"required"` // negatif = mundur
}

type GenerateTimelineRequest struct {
	Preset              string `json:"preset" binding:"required"` // kunci dari registry: wedding_pdf_44 | wedding_essential_22 | general
	IncludeAdicaraTasks bool   `json:"include_adicara_tasks"`      // tugas wedding.adicara_* (§5.2); diabaikan untuk preset general
	Mode                string `json:"mode" binding:"required,oneof=append replace_template replace_all"`
	Confirm             bool   `json:"confirm"`                    // wajib true untuk replace_all bila sudah ada tugas
}

type GenerateTimelineResponse struct {
	Created, Skipped int
	Warnings         []Warning `json:"warnings"` // lead_time_compressed | lead_time_too_short
}

type AdminDoc struct {
	ID             uuid.UUID `json:"id"`
	DocumentName   string    `json:"document_name"`
	SubjectRole    string    `json:"subject_role"` // kunci dari registry jenis acara
	QuantityDetail string    `json:"quantity_detail"`
	NominalCost    int64     `json:"nominal_cost"` // rupiah utuh
	Status         string    `json:"status"`       // "belum" | "selesai"
	SortOrder      int       `json:"sort_order"`
	Notes          string    `json:"notes"`
	TemplateKey    *string   `json:"template_key,omitempty"`
	Origin         string    `json:"origin"`
	BudgetItemID   *uuid.UUID `json:"budget_item_id,omitempty"` // pos anggaran yang menunjuk dokumen ini; nil = belum disinkronkan
}

type CreateAdminDocRequest struct {
	DocumentName   string `json:"document_name" binding:"required,max=255"`
	SubjectRole    string `json:"subject_role" binding:"required"` // divalidasi terhadap registry (400 invalid_input)
	QuantityDetail string `json:"quantity_detail"`
	NominalCost    int64  `json:"nominal_cost" binding:"gte=0"`
	Notes          string `json:"notes"`
}
// UpdateAdminDocRequest: field yang sama, semuanya pointer.
// UpdateAdminDocStatusRequest: {"status": "belum" | "selesai"}

type ApplyTemplateRequest struct {
	Mode    string `json:"mode" binding:"required,oneof=append replace_template"`
	Confirm bool   `json:"confirm"`
}
```

---

## 3. Endpoint

Semua di bawah `BearerAuth` + `ProjectGuard` + `WriteGuard` (`/v1/planner/projects/:id/...`).

### 3.1 Jadwal & Milestone

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/tasks?status=&category=&scope=&phase=` | Daftar tugas (urut `sort_order`) + `summary {total, selesai, proses, belum, overdue}`. `scope` = `scope_key` atau `all` (semua). |
| `POST` | `/tasks` | `CreateTaskRequest` → `201`. |
| `PATCH` | `/tasks/:task_id` | `UpdateTaskRequest`. |
| `PATCH` | `/tasks/:task_id/status` | Toggle cepat (checkbox, Kanban); mengisi `status` saja. |
| `DELETE` | `/tasks/:task_id` | `204`. |
| `POST` | `/tasks/reorder` | `ReorderRequest`; harus memuat **semua** id proyek (`400 invalid_input` bila ada yang asing/hilang). |
| `POST` | `/tasks/shift` | `ShiftTasksRequest`: geser tugas template yang belum selesai sebanyak `days` (aturan pemilihan tugas: [01 BE §4.3](../01_DASHBOARD/BE.md#43-ubah-tanggal-acara-dan-geser-jadwal)). Aksi manual dari layar Timeline, **tidak idempoten** (setiap panggilan menggeser lagi); perubahan tanggal Hari-H memakai `PATCH /projects/:id` dengan `shift_tasks` yang atomik, bukan endpoint ini. |
| `POST` | `/tasks/generate-defaults` | `GenerateTimelineRequest` → `GenerateTimelineResponse`. |
| `GET` | `/tasks/export-spreadsheet` | `.xlsx` bergaya dokumen acuan (§4.4). |

### 3.2 Dokumen & Legalitas

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/admin-docs?subject=&status=` | Daftar berkas + `summary {total, selesai}` + `guide` (judul + 4 langkah dari registry). |
| `POST` | `/admin-docs` | `CreateAdminDocRequest`. |
| `PATCH` | `/admin-docs/:doc_id` | Ubah field. |
| `PATCH` | `/admin-docs/:doc_id/status` | Toggle `belum` ↔ `selesai`. |
| `DELETE` | `/admin-docs/:doc_id` | `204`. Pos anggaran yang menunjuknya kehilangan tautan (`ON DELETE SET NULL`), angkanya tetap. |
| `POST` | `/admin-docs/reorder` | `ReorderRequest`. |
| `POST` | `/admin-docs/apply-template` | `ApplyTemplateRequest`: berkas standar jenis acara (pernikahan: 15 KUA; lainnya: perizinan, [TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)). |
| `POST` | `/admin-docs/:doc_id/sync-budget` | Salin biaya berkas ke pos anggaran (§4.5). |

---

## 4. Logika Bisnis

### 4.1 Generator jadwal

Algoritma penskalaan offset (horizon baku per jenis, pemampatan saat waktu pendek, jepit ke hari ini) ada di [TEMPLATE_EVENT bab 1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#1-aturan-template-jadwal-berlaku-untuk-semua-jenis) dan berlaku untuk **semua** preset. Mode:

| Mode | Efek |
|---|---|
| `append` | Menyisipkan tugas template yang `template_key`-nya belum ada di proyek; tugas lain tidak disentuh. |
| `replace_template` | Menghapus tugas yang punya `template_key` lalu menyisipkan template segar; tugas kustom dipertahankan. |
| `replace_all` | Menghapus **semua** tugas lalu menyisipkan template; wajib `confirm: true` jika proyek sudah punya tugas. |

Scope tiap tugas template dipetakan ke sesi proyek lewat `scope_key`; jika sesi itu sudah dihapus pengguna, tugas jatuh ke `all` dan respons memuat warning `scope_fallback` (jumlah tugas terdampak). Semua tugas baru berstatus `belum`.

### 4.2 Kustomisasi bebas
Tidak ada tugas baca-saja. Tugas hasil template berperilaku sama dengan tugas kustom (ubah, hapus, geser, urut ulang); `template_key` hanya dipakai generator.

### 4.3 Hubungan vendor
`vendor_id` pada tugas memungkinkan milestone seperti "Batas DP Venue" tertaut ke vendor di Menu 04; tampilan menampilkan nama vendor dan tombol WA. Vendor tidak boleh dari proyek lain (dijamin FK komposit).

### 4.4 Ekspor `.xlsx`
Pustaka `excelize` (dependensi baru, [kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)): header berwarna tema, kolom No, Tugas, Rangkaian Acara (nama sesi), Mulai, Selesai, Durasi, Status, serta matriks Gantt mingguan. Sel diawali `=`, `+`, `-`, `@` diberi awalan `'`. Dibatasi 10 ekspor/menit/pengguna.

### 4.5 Sinkron biaya dokumen ke anggaran
`POST /admin-docs/:doc_id/sync-budget` (body opsional `{"sub_event_scope": "akad"}`; default dari pemetaan `template_key`, misal `wedding.pendaftaran_kua` → sesi `akad`, `item_key biaya_kua`; selain itu `all`). Syarat `nominal_cost > 0` (selain itu `400 invalid_input`). Idempoten dan **tidak menimpa angka pengguna**:

1. Dokumen sudah punya pos yang menunjuknya → `200 {created:false, budget_item_id}`.
2. Ada pos dengan `(sub_event_scope, item_key)` cocok yang belum bertaut dokumen → tautkan (`admin_doc_id`); bila estimasinya 0, isi dengan `nominal_cost`, bila beda tampilkan warning `estimate_differs` tanpa mengubah angka.
3. Selain itu buat pos baru (`source='admin_doc'`, kategori "Legalitas", `item_name` = nama berkas, `initial_estimate` = biaya).

Status "tersinkron" di UI = ada baris `planner_budget_items` dengan `admin_doc_id` = dokumen itu.

---

## 5. Master Data Template Pernikahan

### 5.1 Jadwal: 44 Tugas PDF Hal. 3

Nama tugas, urutan, dan durasi (1–2 hari) disalin dari sheet *Timeline Planning*. **Yang bukan dari PDF**: kolom Scope/Kategori/Fase dan offset Hari-H (PDF hanya memuat tanggal absolut Feb–Apr 2026 yang tidak bisa dipakai ulang untuk Hari-H 28 Mar 2027; offset dibuat agar urutan dan durasi PDF tetap, dengan celah menjelang H-45). Nama fase: 1 *Persiapan Awal & Lamaran* (tugas 1–7) · 2 *Administrasi & Booking Vendor Inti* (8–15) · 3 *Detail Acara & Perlengkapan* (16–35) · 4 *Finalisasi & Menuju Hari-H* (36–44). Semua `origin = 'pdf'`, `R = 180`. Kolom **E** menandai preset `wedding_essential_22`; `wedding_pdf_44` memuat semuanya. `template_key` = `wedding.` + judul berhuruf kecil bergaris-bawah, ditambah `_n` bila judul berulang.

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

### 5.2 Tugas Tambahan Adicara (`origin = 'adicara'`, Fase 4)

PDF berhenti di H-4 dan tidak memuat tugas yang menjadi dasar integrasi modul 05 dan 06 (finalisasi tamu, sebar undangan, follow-up RSVP, technical meeting, gladi resik). Enam tugas ini opsional lewat `include_adicara_tasks` dan ditandai "Tambahan Adicara" di antarmuka.

| `template_key` | Tugas | Scope | Kategori | Offset | Terhubung ke |
|---|---|:-:|---|:-:|---|
| `wedding.adicara_finalisasi_tamu` | Finalisasi daftar tamu dan kuota pax | `resepsi` | Tamu | H-45 s/d H-44 | Menu 05 |
| `wedding.adicara_sebar_undangan_digital` | Sebar undangan digital via WhatsApp | `all` | Undangan | H-30 s/d H-29 | Menu 05 |
| `wedding.adicara_followup_rsvp` | Follow-up RSVP tamu | `resepsi` | Tamu | H-10 s/d H-9 | Menu 05 → proyeksi katering (Menu 03) |
| `wedding.adicara_technical_meeting` | Technical meeting vendor dan panitia | `all` | Panitia | H-7 s/d H-6 | Menu 04, 06 |
| `wedding.adicara_gladi_resik` | Gladi resik | `all` | Acara | H-1 s/d H-0 | Menu 06 |
| `wedding.adicara_hari_h` | Hari-H acara | `all` | Acara | H-0 s/d H-0 | Menu 06 |

### 5.3 Dokumen: 15 Berkas PDF Hal. 11

Urutan dan nama berkas persis PDF. Kolom *Peran*, *Jumlah*, dan *Biaya* **tidak ada di PDF** (kolom Nominal/Detail/Jumlah pada PDF kosong): itu saran Adicara (`origin` tetap `pdf` untuk nama, nilai tambahan boleh diubah). `status` template selalu `belum`; Proyek Contoh memakai status PDF (selesai: 1, 2, 3, 6, 7, 8, 10, 14, 15; belum: 4, 5, 9, 11, 12, 13).

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

Disimpan di registry (`AdminGuide`), bukan di komponen FE. Isi mengikuti catatan PDF; formulir yang disebut PDF adalah **N1, N2, dan N4**.

1. **RT & RW**: minta surat pengantar dengan membawa fotokopi KTP kamu dan pasangan serta fotokopi Kartu Keluarga.
2. **Kelurahan**: bawa surat pengantar untuk mengisi formulir N1, N2, N4 dan surat keterangan belum menikah; syarat pasfoto 3×4 dan 2×3 masing-masing 2 lembar, fotokopi KTP CPW dan CPP 2 lembar, fotokopi KK CPP dan CPW 2 lembar, dan surat pengantar RT/RW.
3. **KUA asal (numpang nikah)**: minta surat rekomendasi numpang nikah di tempat tujuan; siapkan fotokopi KTP CPP dan CPW, fotokopi KK, surat pengantar kelurahan (N1, N2, N4), pasfoto 2×3 dan 3×4 masing-masing 2 lembar.
4. **KUA tempat akad**: serahkan surat rekomendasi pindah nikah dari KUA asal, fotokopi KTP dan KK, foto berwarna 2×3 dan 3×4 masing-masing 2 lembar, fotokopi ijazah terakhir, dan fotokopi akta kelahiran.

Jenis acara lain: panduan 4 langkah perizinan dari registry ([TEMPLATE_EVENT](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md)); jadwal dan dokumen non-pernikahan didefinisikan di sana.

---

## 6. Pengujian yang Dibutuhkan
- Generator: Proyek Contoh (`avail = 175`) menghasilkan tugas 1 pada hari ini; `avail = 10` pada jenis `aqiqah` memampatkan semua tugas ke dalam 10 hari tanpa tugas sebelum hari ini; `append` dua kali tidak menggandakan; `replace_template` mempertahankan tugas kustom.
- Durasi: 1→2 Feb = 1, 5→7 Feb = 2, tanggal sama = 1; `end_date < start_date` ditolak.
- Scope tak dikenal → `400 invalid_scope`; vendor proyek lain → ditolak.
- `sync-budget` idempoten dan tidak menimpa estimasi yang sudah terisi.
- Ekspor: sel `=cmd|...` keluar sebagai teks.
