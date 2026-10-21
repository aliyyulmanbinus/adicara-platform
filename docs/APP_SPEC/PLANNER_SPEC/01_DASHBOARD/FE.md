# Menu 01: Dashboard Utama & Hub Acara — Frontend Specification

Acuan pembangunan antarmuka **Planner Hub**, **Wizard Buat Acara**, dan **Workspace Dashboard** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: seluruh kontrak data, aturan tampilan, algoritma, kasus galat, dan pengujian ada di sini.

> **Status (2026-10-21):** backend menu 01 **sudah berjalan dan teruji** ([dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-01.md), kontrak mesin [`openapi.yaml`](../../../../backend/docs/openapi.yaml), bentuk respons [BE.md §3.4](BE.md#34-bentuk-respons-sebagaimana-diimplementasikan)). Frontend belum ada satu baris pun. Semua yang ada di dokumen ini dapat dibangun dan diuji **hari ini** terhadap backend asli, dengan satu catatan penting: modul Planner 02–06 belum dibangun, jadi **keenam widget dashboard bernilai `null`** ("Segera hadir") dan `templates_available` kosong (langkah template wizard otomatis dilewati). Keadaan terisi/kosong/galat widget diuji lewat [fixture](#44-fixture-dashboard-lengkap), bukan lewat backend.
>
> Dokumen ini **belum dijalankan di browser** (belum ada kodenya). Angka, vektor uji, dan fungsi acuan di bab 5 sudah dijalankan di Node 22; klaim teknis yang bergantung pada Astro/browser ditandai *(belum diuji di browser)*.

Pola umum Planner (BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); dokumen ini memuat yang khas Menu 01 dan **rinciannya**.

> Angka pada wireframe berasal dari **Proyek Contoh** (dataset PDF): Sinta & Arif, 28 Maret 2027, H-175 (hari ini 4 Okt 2026), pagu Rp 75.000.000, terbayar Rp 33.550.000. Wireframe bab 7.1 dan 9.1 menggambar **keadaan akhir**, yaitu setelah modul 02–06 ada dan mengisi data. **Keadaan hari ini** terhadap backend asli (semua widget `null`, kartu tanpa tugas/pembayaran) digambar terpisah di 7.1 dan 9.1. Wireframe adalah tata letak, bukan desain visual akhir: warna, jarak, dan tipografi memakai token yang sudah ada (bab 12.1). Emoji pada wireframe untuk **jenis acara dan sesi** memang emoji dari API; ikon **kontrol dan widget** memakai Phosphor (tabel 9.4) dan tidak digambar.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat per halaman](#2-alur-data-dan-penanganan-galat-per-halaman)
3. [Prasyarat: yang sudah ada di repo](#3-prasyarat-yang-sudah-ada-di-repo)
4. [Kontrak data: tipe TypeScript dan fixture](#4-kontrak-data-tipe-typescript-dan-fixture)
5. [Library `lib/planner/*`](#5-library-libplanner)
6. [BFF `/dashboard/api/planner/*`](#6-bff-dashboardapiplanner)
7. [Planner Hub](#7-planner-hub-dashboardplanner)
8. [Wizard Buat Acara](#8-wizard-buat-acara-dashboardplannerbaru)
9. [Workspace Dashboard](#9-workspace-dashboard-dashboardplannerid)
10. [Project Switcher dan PlannerLayout](#10-project-switcher-dan-plannerlayout)
11. [Pesan galat](#11-pesan-galat)
12. [Gaya, responsif, aksesibilitas, keamanan](#12-gaya-responsif-aksesibilitas-keamanan)
13. [Pengujian](#13-pengujian)
14. [Urutan pengerjaan dan definisi selesai](#14-urutan-pengerjaan-dan-definisi-selesai)
15. [Celah dan keputusan yang menunggu pemilik produk](#15-celah-dan-keputusan-yang-menunggu-pemilik-produk)

---

## 1. Route, file, dan perubahan pada kode yang sudah ada

### 1.1 File baru

| Halaman / berkas | Route | File | Fungsi |
|---|---|---|---|
| Planner Hub | `/dashboard/planner` | `src/pages/dashboard/planner/index.astro` | portofolio, metrik, filter, "+ Buat Acara", "Lihat Proyek Contoh" |
| Wizard Acara Baru | `/dashboard/planner/baru` | `src/pages/dashboard/planner/baru.astro` | wizard 3 langkah |
| Workspace Dashboard | `/dashboard/planner/[id]` | `src/pages/dashboard/planner/[id]/index.astro` | hero + 6 widget; membungkus komponen yang sama dengan partial |
| Partial dashboard | `/dashboard/planner/[id]/widgets` | `src/pages/dashboard/planner/[id]/widgets.astro` (`export const partial = true`) | HTML hero + grid widget saja; dipakai polling 30 detik dan tombol "muat ulang" (bab 9.5) |
| BFF Planner | `/dashboard/api/planner/[...path]` | `src/pages/dashboard/api/planner/[...path].ts` | proxy same-origin untuk aksi interaktif (bab 6) |
| Layout | – | `src/layouts/PlannerLayout.astro` | membungkus `AppLayout` untuk semua halaman `[id]/**`: bilah sekunder berisi Project Switcher dan sub-navigasi (bab 10) |
| Halaman 404 | `/404` | `src/pages/404.astro` | tujuan `Astro.rewrite('/404')` untuk proyek yang tidak ditemukan; belum ada di repo |
| Komponen | – | `src/components/planner/dashboard/*.astro` | lihat 1.2 |
| Library | – | `src/lib/planner/{types,format,countdown,messages,eventTypes,client,server}.ts` | lihat bab 5 |
| Tes | – | `tests/planner-*.test.mjs` | bab 13 |

### 1.2 Komponen

| Komponen | Props utama | Dipakai di |
|---|---|---|
| `PortfolioMetrics` | `summary: PortfolioSummary`, `types: EventTypeInfo[]` | Hub |
| `ProjectFilters` | `counts`, `types` | Hub (tab status, kategori, pencarian; skrip klien) |
| `ProjectCard` | `card: ProjectCard`, `type: EventTypeInfo \| undefined` | Hub |
| `ProjectMenu` | `card` | Hub (menu `•••`: edit, tandai selesai, arsipkan, hapus) |
| `ConfirmDeleteDialog` | `projectId`, `title` | Hub dan Workspace |
| `EventWizard` | `types: EventTypeInfo[]`, `initialType?: EventTypeKey` | wizard |
| `HeroCountdown` | `project: DashboardProject`, `serverTime: string`, `type` | Workspace dan partial |
| `EditEventDialog` | `projectId`, `type: EventTypeInfo` | Workspace dan Hub |
| `WidgetCard` | `title`, `icon`, `href?`, `state: 'filled' \| 'empty' \| 'soon' \| 'error'`, slot | enam widget |
| `widgets/{Timeline,Finance,Vendor,Guest,AdminDoc,Rundown}Widget.astro` | `data`, `projectId`, `type` | Workspace dan partial |
| `SessionManager` | `projectId`, `type` | modal dari `EditEventDialog` |
| `ProjectSwitcher` | `items: ProjectLite[]`, `current`, `subpath` | `PlannerLayout` |

### 1.3 Perubahan pada file yang sudah ada (kecil, sengaja)

| File | Perubahan | Mengapa |
|---|---|---|
| `src/components/layout/AppHeader.astro` | tambah satu tautan `<a href="/dashboard/planner">Planner</a>` di samping "Undangan" | navigasi |
| `src/lib/api/backend.ts` | (1) tipe `method` ditambah `'PUT'`; (2) `BackendError` ditambah `details?: Record<string, unknown>` | sesi memakai `PUT`; `session_in_use` membawa `error.details.usage`. Tanpa ini `tsc` menolak dan `details` tak terbaca bertipe |
| (tidak diubah) | `AppLayout.astro`, `middleware.ts`, `tokens.css`, `global.css`, `env.d.ts` | `PlannerLayout` menyusun `AppLayout`; middleware sudah menjaga `/dashboard/**` |

`astro check` (bagian dari `npm run build`) harus lolos: TypeScript `strict` dan `noUncheckedIndexedAccess` aktif, jadi akses array/objek berindeks menghasilkan `T | undefined`.

---

## 2. Alur data dan penanganan galat per halaman

```text
SSR (baca)    : halaman → backendFetch('/v1/planner/…', { accessToken: Astro.locals.accessToken }) → HTML
Interaksi     : <script> klien → plannerFetch('projects/:id/…') → BFF /dashboard/api/planner/… → Go API
Pembaruan     : <script> klien → fetch('/dashboard/planner/:id/widgets') (HTML partial) → ganti isi DOM
Registry      : lib/planner/eventTypes.ts → GET /v1/planner/event-types (cache memori proses 1 jam)
```

`backendFetch` mengembalikan seluruh payload di `result.data`, jadi isi respons ada di **`result.data.data`** (amplop `{ "data": … }`). Data yang tidak berhasil: `result.data === null` dan `result.error = { code, message, details? }`.

### 2.1 Pemuatan per halaman

| Halaman | Panggilan (paralel dengan `Promise.all`) | Catatan |
|---|---|---|
| Hub | `GET projects/summary` + registry | satu kueri Hub; tidak memanggil `?view=lite` |
| Wizard | registry | tanpa data proyek |
| Workspace `[id]` | `GET projects/:id/dashboard` + registry + (di `PlannerLayout`) `GET projects?status=all&view=lite` | **tidak** memanggil `GET projects/:id`; blok `project` ada di respons dashboard |
| Partial | `GET projects/:id/dashboard` + registry | tanpa daftar proyek |

Validasi `Astro.params.id` sebelum memanggil backend: harus cocok `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` (huruf kecil); selain itu langsung `Astro.rewrite('/404')` tanpa memanggil backend.

### 2.2 Hasil backend → tindakan halaman

| Hasil | Hub / Wizard | Workspace `[id]` | Partial `widgets` |
|---|---|---|---|
| `2xx` | render | render | render (status 200) |
| `401` | `Astro.redirect('/auth/masuk')` | sama | status 401 (klien mengarahkan ke login) |
| `404 not_found` | – | `Astro.rewrite('/404')`; tidak membedakan "bukan milik Anda", "tidak ada", dan id bukan UUID | status 404 (klien berhenti polling, tampilkan banner "Acara tidak ditemukan") |
| `429 rate_limited` (hanya dashboard) | – | render kerangka halaman + banner "Terlalu banyak permintaan, coba lagi sebentar" | status 429 (klien mundur 60 detik) |
| `5xx` atau `BackendUnavailableError` | banner "Planner sedang tidak tersedia", grid kosong, halaman tetap terbuka | kerangka halaman + banner + tombol "Muat ulang"; **bukan 404** | status 503 (klien mundur 60 detik, isi lama dipertahankan) |
| Registry gagal | pakai cache lama bila ada; bila tidak ada, label jenis jatuh ke `key`, ikon `✨`, filter kategori disembunyikan; wizard menampilkan banner dan menonaktifkan tombol lanjut | idem | idem |

Aturan umum: setiap halaman `export const prerender = false`; middleware sudah menambah `Cache-Control: private, no-store`. Teks galat ke pengguna selalu dari `lib/planner/messages.ts`, tidak pernah `error.message` mentah (Inggris dan bisa berubah).

---

## 3. Prasyarat: yang sudah ada di repo

Dipakai apa adanya, tidak dibangun ulang:

| Kebutuhan | Yang ada |
|---|---|
| Sesi dan token | `src/middleware.ts` menjaga `/dashboard/**`, mengisi `Astro.locals.user` dan `Astro.locals.accessToken` (refresh otomatis). Bila belum login: redirect `302` ke `/auth/masuk`. Bila backend tak terjangkau: `503` teks biasa |
| Panggilan ke Go | `backendFetch(path, { method, body, accessToken, clientIP })` di `src/lib/api/backend.ts` (timeout 8 dtk; melempar `BackendUnavailableError` bila jaringan gagal). Path backend asli (`/v1/…`), bukan `/api/v1/…` |
| IP klien | `clientIPOf(context)` di `src/lib/auth/credentials.ts`; teruskan sebagai `clientIP` agar batas laju per pengguna benar |
| Layout | `AppLayout` (`title`, `description`; sudah `noindex`; memuat `AppHeader` + `<main id="main-content">`) di atas `BaseLayout` |
| Gaya | `src/styles/tokens.css` (token `--sage-*`, `--color-*`, `--space-1…9`, `--radius-*`, `--font-display`, `--font-body`, `--shadow-soft`, `--transition`), kelas global di `global.css`: `.container`, `.section`, `.eyebrow`, `.lede`, `.button`, `.button--secondary`, `.visually-hidden`, `.skip-link`. Fredoka (judul) dan Nunito Sans (isi) |
| Ikon antarmuka | Phosphor duotone, sudah diimpor `BaseLayout`: `<i class="ph-duotone ph-timer"></i>`. Nama yang sudah dicek ada: `timer wallet coins handshake users-three scroll file-text clipboard-text list-checks calendar-check piggy-bank music-notes caret-down plus dots-three archive trash pencil-simple check-circle warning-circle clock-countdown flag-checkered x arrow-right chat-circle-text`. **Ikon jenis acara dan sesi adalah emoji dari API**, bukan Phosphor |
| Pola skrip klien | `<script>` TypeScript per komponen, tanpa framework (lihat `AuthForm.astro`, `AppHeader.astro`) |
| Konfigurasi | `output: 'static'`, adapter Node standalone, `security.checkOrigin: true`. Halaman server **wajib** `export const prerender = false`. Tidak ada dependensi baru |

> **Jebakan `checkOrigin`** (`security.checkOrigin: true`): menurut komentar di `AppHeader.astro`, POST tanpa body dianggap form dan diperiksa origin-nya, yang gagal di balik proxy yang menghentikan TLS. Karena itu **setiap** `fetch` non-GET dari klien mengirim `Content-Type: application/json` dan body JSON (`'{}'` bila tidak ada isi), seperti tombol Keluar di `AppHeader.astro`. Bagi `DELETE` yang tidak butuh body, BFF mengabaikannya.

---

## 4. Kontrak data: tipe TypeScript dan fixture

### 4.1 Aturan JSON yang dikirim backend

- Amplop sukses `{ "data": … }`; galat `{ "error": { "code", "message", "details?" } }`. `204` tanpa body.
- **Tanggal kalender** (`event_date`, `session_date`, …) adalah string `"YYYY-MM-DD"` **tanpa zona**. Jangan `new Date("2027-03-28")` lalu memformat di zona lokal (bergeser sehari di zona negatif). Selalu format dengan `timeZone: 'UTC'` (bab 5.1). **Waktu kejadian** (`server_time`, `created_at`) adalah RFC 3339.
- **Jam** `"HH:MM"` (24 jam, tanpa detik). Input `type="time"` sudah berformat itu.
- **Uang** bilangan bulat rupiah (`number`, aman hingga 10¹³).
- **`null` vs tidak ada:** `project.*` mengirim `null` untuk kosong; `ProjectCard.event_end_date`, `ProjectLite.event_end_date`, `SessionBrief.{session_date,start_time,end_time,location_name}` **dihilangkan** bila kosong; `nearest_project` dihilangkan bila tidak ada. Tipe di bawah mencerminkan itu (`?:` = bisa tidak ada, `| null` = dikirim `null`).
- `metadata` hanya berisi kunci milik jenis acara (registry); nilainya `string` atau `number` (integer).

**Sudah dicocokkan dengan backend asli (2026-10-21):** interface di 4.2 dibandingkan otomatis dengan **136 objek JSON** dari backend yang berjalan (registry 9 jenis beserta sesi/kolom/template, buat `201` dan kirim ulang `200`, contoh, kartu Hub, versi `lite`, ringkasan, detail, PATCH, daftar sesi, buat sesi dengan semua kolom dan dengan nama saja, dashboard, ubah status, tiga galat): **nol selisih** kunci, opsionalitas (`?:`), dan `null`. Pembandingan memakai parser sederhana terhadap blok kode dokumen ini, bukan tinjauan manual. Yang **tidak** bisa dibuktikan secara langsung karena modul 02–06 belum ada: isi `applied`/`ApplyResult` dan `Notice` bernilai (tugas yang digeser); bentuknya dicek lewat tes Go (`delivery/http`, `usecase`) bukan lewat panggilan nyata. Isi widget (`TimelineSummary` dst.) adalah kontrak [BE §5.1](BE.md#51-response) yang dipakai fixture, bukan keluaran backend saat ini.

### 4.2 `src/lib/planner/types.ts`

Hanya sintaks yang dapat dihapus (tanpa `enum`), supaya bisa diimpor tes `node --experimental-strip-types`.

```ts
export type EventTypeKey =
  | 'wedding' | 'khitanan' | 'aqiqah' | 'peresmian' | 'launching'
  | 'birthday' | 'wisuda' | 'seminar' | 'other';
export type ProjectStatus = 'active' | 'completed' | 'archived';
export type EventPhase = 'upcoming' | 'ongoing' | 'finished';
export type Metadata = Record<string, string | number>;

export interface Envelope<T> { data: T }
export interface ApiError { code: string; message: string; details?: Record<string, unknown> }

export interface PhaseInfo {
  days_remaining: number; // event_date − hari ini (zona proyek); 0 atau negatif sejak Hari-H
  phase: EventPhase;
  day_number: number;     // hari ke-N saat ongoing, selain itu 0
  total_days: number;     // 1 untuk acara satu hari
}

// ---------- registry: GET /v1/planner/event-types ----------
export interface MetadataField {
  key: string; label: string;
  type: 'string' | 'int' | 'date';
  max_len?: number;                 // string: karakter (code point)
  min?: number; max?: number;       // int
  required: boolean;
}
export interface SessionSeed { scope_key: string; name: string; icon: string; same_day: boolean }
export interface TemplatesAvailable {
  timeline_presets: string[];       // [] bila modul timeline belum ada
  include_adicara_tasks: boolean;
  admin_docs: boolean; budget_lines: boolean; shopping_list: boolean; rundown: boolean; playlist: boolean;
}
export interface EventTypeInfo {
  key: EventTypeKey; label: string; icon: string; lead_time_days: number;
  sessions: SessionSeed[];
  metadata_fields: MetadataField[]; // urutan registry: kolom khas jenis dulu, lalu hero_title, venue_name, city
  labels: { guest: string; guest_menu: string }; // 'Tamu' / 'Buku Tamu & RSVP' (seminar: 'Peserta' / 'Peserta & RSVP')
  templates_available: TemplatesAvailable;
}

// ---------- proyek ----------
export interface Project extends PhaseInfo {
  id: string; event_type: EventTypeKey; title: string;
  event_date: string; event_end_date: string | null; prep_start_date: string; timezone: string;
  target_budget: number; savings_target: number | null;
  status: ProjectStatus; is_sample: boolean;
  skipped_vendor_categories: string[]; invitation_id: string | null;
  metadata: Metadata; created_at: string; updated_at: string;
}
export interface ProjectCard extends PhaseInfo {
  id: string; event_type: EventTypeKey; title: string;
  event_date: string; event_end_date?: string;
  status: ProjectStatus; is_sample: boolean; target_budget: number;
  total_estimate: number; total_paid: number;
  total_tasks: number; completed_tasks: number;
  guest_pax_count: number;          // jumlah pax; TIDAK ada jumlah undangan di kartu
  metadata: Metadata;
}
export interface ProjectLite {
  id: string; event_type: EventTypeKey; title: string;
  event_date: string; event_end_date?: string;
  status: ProjectStatus; is_sample: boolean;
  days_remaining: number; phase: EventPhase;
}
export interface PortfolioSummary {
  active_count: number; completed_count: number; archived_count: number; // tanpa proyek contoh
  combined_target_budget: number;   // proyek aktif, tanpa contoh
  nearest_project?: ProjectCard;
  projects: ProjectCard[];          // semua, termasuk contoh
}

export interface InitTemplates {
  timeline_preset?: string | null; include_adicara_tasks?: boolean;
  admin_docs?: boolean; budget_lines?: boolean; shopping_list?: boolean; rundown?: boolean; playlist?: boolean;
}
export interface CreateProjectBody {
  event_type: EventTypeKey; title: string; event_date: string;
  event_end_date?: string; prep_start_date?: string; timezone?: string;
  target_budget: number; metadata: Metadata; templates: InitTemplates; client_token: string;
}
export interface UpdateProjectBody {
  title?: string; event_date?: string;
  event_end_date?: string; clear_event_end_date?: boolean;
  target_budget?: number;
  metadata?: Record<string, string | number | null>; // JSON Merge Patch: null menghapus kunci
  shift_tasks?: boolean;
}
export interface Notice { code: string; message: string; count: number; shift_days?: number }
export interface ApplyResult { created: number; skipped: number; warnings: Notice[] }
export interface CreateProjectResult { project: Project; created: boolean; applied?: Record<string, ApplyResult> }
export interface UpdateProjectResult { project: Project; warnings: Notice[]; hints: Notice[] }
export interface ProjectDetail { project: Project; shiftable_tasks: number }

// ---------- sesi ----------
export interface SessionBrief {
  id: string; scope_key: string; name: string; icon: string;
  is_system: boolean; sort_order: number;
  session_date?: string; start_time?: string; end_time?: string; location_name?: string;
}
export interface SessionUsage {
  tasks: number; vendors: number; budget_items: number; scenario_items: number;
  shopping_items: number; guests: number; agendas: number;
}
export interface SessionWithUsage extends SessionBrief { usage: SessionUsage }
export interface SessionBody {      // POST dan PUT (PUT mengganti penuh)
  name: string; icon?: string; session_date?: string; start_time?: string; end_time?: string; location_name?: string;
}

// ---------- dashboard: GET /v1/planner/projects/:id/dashboard ----------
export interface DashboardProject extends PhaseInfo {
  id: string; event_type: EventTypeKey; title: string;
  event_date: string; event_end_date: string | null;
  status: ProjectStatus; is_sample: boolean;
  invitation_linked: boolean; target_budget: number; savings_target: number | null;
  timezone: string; metadata: Metadata;
}
export interface UrgentTask { id: string; title: string; due_date: string; days_left: number } // negatif = terlambat
export interface TimelineSummary { total: number; completed: number; in_progress: number; overdue: number; urgent: UrgentTask[] }
export interface FinanceSummary {
  target_budget: number; total_estimate: number; total_paid: number;
  remaining: number; overpaid: number;
  savings_target: number; savings_collected: number;
  free_cash: number; cash_deficit: number;
  line_status: { settled: number; partial: number; unpaid: number };
}
export interface VendorSummary { categories_total: number; categories_fixed: number; skipped: number; fixed_vendors: number; pipeline: number }
export interface GuestSummary {
  invitations: number; total_pax: number;
  sides: { key: string | null; label: string; invitations: number; pax: number }[];
  sent: number;
  rsvp: { attending: number; declined: number; maybe: number; unconfirmed: number; attending_pax: number };
}
export interface AdminDocSummary { label: string; completed: number; total: number; pending: string[] } // pending maks 5
export interface RundownSummary {
  sessions: number; agendas: number; completed: number; in_progress: number; songs: number;
  current_agenda: { activity: string; session: string } | null;
  next_session: { name: string; date: string; start_time: string | null } | null;
}
export interface WidgetError { error: 'internal' | 'timeout' }
export type Widget<T> = T | WidgetError | null;   // null = modulnya belum ada
export interface Dashboard {
  server_time: string;              // RFC 3339, UTC
  project: DashboardProject;
  sessions: SessionBrief[];         // tanpa sesi sistem 'all', tanpa usage
  timeline: Widget<TimelineSummary>; finance: Widget<FinanceSummary>; vendors: Widget<VendorSummary>;
  guests: Widget<GuestSummary>; admin_docs: Widget<AdminDocSummary>; rundown: Widget<RundownSummary>;
}
export const isWidgetError = (w: unknown): w is WidgetError =>
  typeof w === 'object' && w !== null && 'error' in w;
```

Tipe widget dimiliki modul 02–06 dan **dapat bertambah bidang**; baca hanya bidang di atas dan abaikan sisanya. Provider seharusnya mengirim `[]`, bukan `null`, untuk daftar kosong, tetapi tampilan selalu bertahan: `data.urgent ?? []`, `data.pending ?? []`, `data.sides ?? []`.

### 4.3 Bentuk respons per endpoint

Tabel lengkap ada di [BE.md §3.4](BE.md#34-bentuk-respons-sebagaimana-diimplementasikan). Yang penting untuk FE:

| Panggilan | Sukses | Perhatikan |
|---|---|---|
| `POST projects` | `201` `{ project, created: true, applied }` atau `200` `{ project, created: false }` | `200` juga **sukses**: kirim ulang dengan `client_token` yang sama (bab 8.5) |
| `POST projects/sample` | `201` baru / `200` sudah ada | keduanya sukses → buka workspace |
| `PATCH projects/:id` | `200` `{ project, warnings[], hints[] }` | `warnings` dan `hints` selalu array |
| `PATCH projects/:id/status` | `200` `{ project }` | lolos pada proyek `archived` |
| `DELETE projects/:id?confirm=` | `204` | lolos pada proyek `archived` |
| `GET projects/:id/sessions` | `200` array `SessionWithUsage` termasuk sesi sistem `all` | saring `is_system` untuk editor |
| `POST projects/:id/sessions/reorder` | `200` array `SessionBrief` semua sesi | |
| `DELETE …/sessions/:sid?move_to=&delete_agendas=` | `204` | `409 session_in_use` membawa `error.details.usage` |

### 4.4 Fixture dashboard lengkap

Simpan sebagai `tests/fixtures/planner-dashboard.json` dan gunakan untuk menguji **keempat keadaan** tiap widget tanpa menunggu backend (ubah satu widget menjadi `null`, `{"error":"timeout"}`, atau nol-semua). Angka = Proyek Contoh PDF:

```json
{
  "server_time": "2026-10-04T03:00:00Z",
  "project": {
    "id": "6347c545-f38c-4ed2-975d-6b5ddf6915f4", "event_type": "wedding",
    "title": "Contoh: Pernikahan Sinta & Arif", "event_date": "2027-03-28", "event_end_date": null,
    "days_remaining": 175, "phase": "upcoming", "day_number": 0, "total_days": 1,
    "status": "active", "is_sample": true, "invitation_linked": false,
    "target_budget": 75000000, "savings_target": null, "timezone": "Asia/Jakarta",
    "metadata": { "hero_title": "Bismillah Nikah", "bride_name": "Sinta Maharani", "groom_name": "Arif Pratama",
                  "venue_name": "Sasana Kriya Ballroom", "city": "Jakarta Selatan" }
  },
  "sessions": [
    { "id": "11111111-1111-4111-8111-111111111111", "scope_key": "lamaran", "name": "Lamaran", "icon": "💍", "is_system": false, "sort_order": 1, "session_date": "2026-10-10" },
    { "id": "22222222-2222-4222-8222-222222222222", "scope_key": "akad", "name": "Akad Nikah", "icon": "📜", "is_system": false, "sort_order": 2, "session_date": "2027-03-28" },
    { "id": "33333333-3333-4333-8333-333333333333", "scope_key": "resepsi", "name": "Resepsi", "icon": "🏛️", "is_system": false, "sort_order": 3, "session_date": "2027-03-28" }
  ],
  "timeline": { "total": 50, "completed": 15, "in_progress": 2, "overdue": 0, "urgent": [] },
  "finance": {
    "target_budget": 75000000, "total_estimate": 74150000, "total_paid": 33550000, "remaining": 40600000, "overpaid": 0,
    "savings_target": 75000000, "savings_collected": 47500000, "free_cash": 13950000, "cash_deficit": 26650000,
    "line_status": { "settled": 3, "partial": 4, "unpaid": 7 }
  },
  "vendors": { "categories_total": 10, "categories_fixed": 7, "skipped": 0, "fixed_vendors": 7, "pipeline": 3 },
  "guests": {
    "invitations": 20, "total_pax": 43,
    "sides": [ { "key": "pria", "label": "Pihak Pria", "invitations": 10, "pax": 22 }, { "key": "wanita", "label": "Pihak Wanita", "invitations": 10, "pax": 21 } ],
    "sent": 14, "rsvp": { "attending": 12, "declined": 2, "maybe": 2, "unconfirmed": 4, "attending_pax": 27 }
  },
  "admin_docs": { "label": "Kelengkapan Berkas KUA", "completed": 9, "total": 15,
                  "pending": ["FC KTP Ortu CMP", "FC KTP Ortu CMW", "Pas Foto 2x3", "Pas Foto 4x6", "Surat ket. belum nikah"] },
  "rundown": { "sessions": 3, "agendas": 33, "completed": 0, "in_progress": 0, "songs": 30, "current_agenda": null,
               "next_session": { "name": "Lamaran", "date": "2026-10-10", "start_time": null } }
}
```

Bagian ini **bukan** jawaban backend saat ini (di backend, semua widget `null`); ia hanya bentuk kontrak yang dijanjikan [BE.md §5.1](BE.md#51-response). Tambah satu fixture dengan semua widget `null` (keadaan nyata sekarang).

---

## 5. Library `lib/planner/*`

Aturan: modul **murni** (`types`, `format`, `countdown`, `messages`) tidak mengimpor apa pun dengan alias `@/` dan tidak memakai `enum`/parameter property, supaya tes `node --experimental-strip-types` (seperti `tests/auth-messages.test.mjs`) dapat mengimpornya langsung. Modul yang butuh Astro atau `backendFetch` (`eventTypes.ts`, `server.ts`) menerima ketergantungannya sebagai argumen atau dipisah agar bagian murninya teruji.

### 5.1 `format.ts` (acuan sudah dijalankan; vektor uji di 13.1)

```ts
const idDate = new Intl.DateTimeFormat('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' });
const idDateShort = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' });
/** "2027-03-28" → "Minggu, 28 Maret 2027". timeZone UTC supaya tanggal kalender tidak bergeser. */
export const formatDate = (d: string) => idDate.format(new Date(`${d}T00:00:00Z`));
/** "2026-10-15" → "15 Okt 2026" */
export const formatDateShort = (d: string) => idDateShort.format(new Date(`${d}T00:00:00Z`));

const grouped = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 });
/** 74150000 → "Rp 74.150.000"; negatif → "-Rp 1.500" */
export const formatRupiah = (n: number) => (n < 0 ? '-' : '') + 'Rp ' + grouped.format(Math.abs(n));

const dec = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 });
/** Singkat untuk kartu: 33550000 → "Rp 33,55 Jt"; 950000 → "Rp 950 Rb"; 1500000000 → "Rp 1,5 M"; 10^13 → "Rp 10 T"; <1000 → "Rp 999" */
export function formatRupiahShort(n: number): string {
  const abs = Math.abs(n), sign = n < 0 ? '-' : '';
  for (const [size, label] of [[1e12, 'T'], [1e9, 'M'], [1e6, 'Jt'], [1e3, 'Rb']] as const) {
    if (abs >= size) return `${sign}Rp ${dec.format(abs / size)} ${label}`;
  }
  return `${sign}Rp ${grouped.format(abs)}`;
}

/** Pembagi nol → null (tampilkan keadaan kosong, bukan 0% atau NaN%). */
export const percent = (n: number, d: number): number | null => (d <= 0 ? null : n / d);
/** 0.452 → "45,2%" dengan digits=1; 0.3 → "30%" dengan digits=0 (default); null → "–"; 1.2 → "120%" */
export function formatPercent(p: number | null, digits = 0): string {
  if (p === null) return '–';
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: digits }).format(p * 100) + '%';
}
/** Lebar batang progres: dijepit 0–100 walau teks bisa >100%. */
export const barWidth = (p: number | null) => (p === null ? 0 : Math.min(100, Math.max(0, p * 100)));

/** Tanggal kalender (YYYY-MM-DD) saat ini di zona tz. 'en-CA' memformat ISO. */
export const todayIn = (tz: string, nowMs = Date.now()) =>
  new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(nowMs));
export function addDays(date: string, n: number): string {
  const d = new Date(`${date}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}
export const daysBetween = (from: string, to: string) =>
  Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / 86400000);
export function isKnownZone(tz: string): boolean {
  try { new Intl.DateTimeFormat('en', { timeZone: tz }); return true; } catch { return false; }
}
/** Bentuk yang diterima backend (CHECK): nama IANA kanonik. 'asia/jakarta', 'WIB', 'UTC+7', 'Etc/GMT-7' ditolak. */
export const ZONE_SHAPE = /^(UTC|(Africa|America|Antarctica|Arctic|Asia|Atlantic|Australia|Europe|Indian|Pacific)\/[A-Za-z0-9_+-]+(\/[A-Za-z0-9_+-]+)?)$/;
/** Panjang dalam karakter (code point) seperti backend; 'str.length' (UTF-16) salah untuk emoji. */
export const charCount = (s: string) => Array.from(s).length;
/** Pesan backend "<field>: <alasan>" → { field, reason }; null bila tak berbentuk itu. */
export function parseFieldError(message: string | undefined) {
  const m = /^([a-z0-9_]+): (.+)$/.exec(message ?? '');
  return m ? { field: m[1] as string, reason: m[2] as string } : null;
}
```

Label status/fase dan "H-n" (dipakai Hub, switcher, hero):

| Kondisi (`status`, `phase`) | Lencana | Teks waktu |
|---|---|---|
| `active`, `upcoming` | AKTIF | `⏳ {days_remaining} hari menuju Hari-H` (singkat: `H-{days_remaining}`) |
| `active`, `ongoing`, `total_days = 1` | BERLANGSUNG | `Hari-H telah tiba` |
| `active`, `ongoing`, `total_days > 1` | BERLANGSUNG | `Hari ke-{day_number} dari {total_days}` |
| `active`, `finished` | AKTIF | `Acara sudah lewat` + tombol **Tandai selesai** |
| `completed` (fase apa pun) | SELESAI | tanggal acara |
| `archived` (fase apa pun) | ARSIP | tanggal acara |
| `is_sample` | tambahan lencana CONTOH | – |

### 5.2 `countdown.ts` (acuan sudah dijalankan)

```ts
function tzOffsetMs(tz: string, t: number): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: tz, hourCycle: 'h23', year: 'numeric', month: 'numeric', day: 'numeric',
    hour: 'numeric', minute: 'numeric', second: 'numeric',
  }).formatToParts(new Date(t));
  const v = Object.fromEntries(parts.filter((p) => p.type !== 'literal').map((p) => [p.type, Number(p.value)]));
  return Date.UTC(v.year!, v.month! - 1, v.day!, v.hour!, v.minute!, v.second!) - Math.floor(t / 1000) * 1000;
}
/** Epoch ms pukul 00:00 pada `date` di zona `tz` (mis. 2027-03-28 di Asia/Jakarta = 2027-03-27T17:00:00Z). */
export function zonedMidnight(date: string, tz: string): number {
  const [y, m, d] = date.split('-').map(Number) as [number, number, number];
  const wall = Date.UTC(y, m - 1, d);
  let t = wall - tzOffsetMs(tz, wall);
  t = wall - tzOffsetMs(tz, t); // lintasan kedua: offset pada instan sebenarnya (DST)
  return t;
}
export function countdown(targetMs: number, nowMs: number) {
  const s = Math.max(0, Math.floor((targetMs - nowMs) / 1000));
  return { days: Math.floor(s / 86400), hours: Math.floor((s % 86400) / 3600), minutes: Math.floor((s % 3600) / 60), seconds: s % 60 };
}
```

Aturan pemakaian (bab 9.2 merinci): jam yang dipakai adalah `Date.now() + skew`, dengan `skew = Date.parse(server_time) − Date.now()` diambil **sekali saat respons diterima** (bukan jam perangkat mentah). Keterbatasan: `zonedMidnight` tidak menangani tengah malam yang tidak ada (zona yang DST-nya melompat tepat pukul 00:00, mis. beberapa zona Amerika Selatan/Timur Tengah); hasilnya bergeser satu jam, dan layar dikoreksi saat server mengganti `phase`. Untuk zona Indonesia tidak ada DST.

### 5.3 `eventTypes.ts`

- `getEventTypes(fetcher): Promise<EventTypeInfo[]>` — cache modul-level `{ value, expiresAt }` TTL 1 jam; permintaan serentak berbagi satu `Promise` (hindari badai ke backend saat cache kosong); bila gagal dan ada cache lama, kembalikan yang lama; bila tidak ada, lempar.
- `typeOf(types, key)` → `EventTypeInfo | undefined`; konsumen memakai `fallbackType(key)` = `{ label: key, icon: '✨', labels: { guest: 'Tamu', guest_menu: 'Buku Tamu & RSVP' }, metadata_fields: [], … }`.
- Cache hidup per proses Node; restart server mengosongkannya. Hasil registry statis per rilis backend (`Cache-Control: public, max-age=3600`).
- **Dilarang** menulis daftar jenis acara, ikon, label, sesi, atau kolom metadata di komponen. Tes sumber (13.3) memeriksanya.

### 5.4 `messages.ts`

Tabel kode → teks ada di [bab 11](#11-pesan-galat). API:

```ts
export function plannerErrorMessage(status: number, code: string | undefined): string
export function fieldErrorMessage(field: string, reason: string, ctx?: { max?: number; min?: number }): string
export const TEMPLATE_LABELS: Record<string, string>      // kunci template/preset → teks wizard (bab 8.4)
export const WARNING_LABELS: Record<string, string>       // kode warning/hint → teks
export const USAGE_LABELS: (guest: string) => Record<keyof SessionUsage, string>
```

### 5.5 `client.ts` (khusus browser)

`plannerFetch(path, { method = 'GET', body, query })` memanggil `/dashboard/api/planner/${path}` dengan `credentials: 'same-origin'`, dan **selalu** mengirim `Content-Type: application/json` + body (`'{}'` bila kosong) untuk metode non-GET. Mengembalikan `{ ok, status, data, error }` dan menangani tiga kasus yang `fetch` mentah tidak menangani:

1. **Sesi habis:** middleware membalas `302` ke `/auth/masuk` untuk BFF juga, dan `fetch` mengikuti redirect itu lalu menerima HTML login dengan status `200`. Jika `response.redirected` atau `Content-Type` bukan JSON/`204` → `window.location.assign('/auth/masuk')` lalu kembalikan `{ ok: false, status: 401 }`.
2. **Backend/BFF mati:** `fetch` melempar atau status `503` → `{ ok: false, status: 503, error: { code: 'service_unavailable' } }`.
3. **`204`:** tanpa body → `{ ok: true, status: 204, data: null }`.

### 5.6 `server.ts`

Pembantu SSR yang dipakai ketiga halaman: `loadRegistry(context)`, `isUuid(id)`, `toPageError(result)` (menerapkan tabel 2.2). Boleh memakai alias `@/`; tidak diuji dengan `node:test`.

---

## 6. BFF `/dashboard/api/planner/*`

File `src/pages/dashboard/api/planner/[...path].ts`; `export const prerender = false`; mengekspor `GET`, `POST`, `PATCH`, `PUT`, `DELETE`. Berada di bawah `/dashboard/` sehingga middleware menjaganya (token sudah segar). Mengikuti pola `src/pages/auth/*.ts` dan memakai ulang `json()`/`clientIPOf()` dari `credentials.ts`.

Langkah untuk setiap permintaan:

1. `path = context.params.path` (mis. `projects/6347…/sessions`). Tolak `404 not_found` bila kosong, tak cocok `^[a-z0-9/_-]+$`, memuat `..` atau `//`, atau diawali/diakhiri `/`. (UUID memakai huruf kecil, angka, dan tanda hubung, jadi lolos.)
2. Metode selain `GET` wajib `Content-Type: application/json`, selain itu `415 unsupported_media_type`. Body dibatasi 64 KiB (`400 invalid_json` bila lebih atau bukan JSON object); `DELETE` tidak meneruskan body.
3. Query string diteruskan dengan membangun ulang lewat `URLSearchParams` (di-encode ulang), bukan menyambung teks. Mis. `?confirm=Pernikahan%20Sinta%20%26%20Arif`.
4. `backendFetch('/v1/planner/' + path + query, { method, body, accessToken, clientIP })`. Karena `backendFetch` hanya mendukung `GET|POST|PATCH|DELETE`, **tambahkan `'PUT'`** (bab 1.3).
5. Jawaban: status backend diteruskan apa adanya. Sukses → `json(status, result.data)` (atau `204` tanpa body). Galat → `json(status, { error: result.error })` (termasuk `details`). `BackendUnavailableError` → `503 { error: { code: 'service_unavailable' } }`.
6. Header jawaban: `Content-Type: application/json`, `Cache-Control: no-store` (sudah dilakukan `json()`).

Yang **tidak** boleh dilakukan BFF: memanggil path di luar `/v1/planner/`, meneruskan header klien selain yang disebut, atau menyimpan token di respons. Browser tidak pernah memanggil Go langsung (token ada di cookie HttpOnly).

---

## 7. Planner Hub (`/dashboard/planner`)

### 7.1 Wireframe

**Keadaan akhir** (modul 02–06 sudah ada; kartu pertama = Proyek Contoh, dua kartu lain ilustrasi):

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Adicara                                                           Undangan  Planner  Keluar │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ RINGKASAN PORTOFOLIO (tanpa Proyek Contoh)                                                  │
│ ┌──────────────────┐ ┌──────────────────┐ ┌────────────────────┐ ┌────────────────────────┐ │
│ │ Acara Aktif      │ │ Acara Selesai    │ │ Total Pagu Aktif   │ │ Acara Terdekat         │ │
│ │ 1                │ │ 1                │ │ Rp 75.000.000      │ │ 🎤 Tech Summit — H-223 │ │
│ └──────────────────┘ └──────────────────┘ └────────────────────┘ └────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ [Semua (3)] [Aktif (2)] [Selesai (1)] [Arsip (0)]   Kategori: [Semua ▼]   🔍 [ cari … ]     │
│                                                                       [ + Buat Acara Baru ] │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ 💍 PERNIKAHAN            [CONTOH] [AKTIF] │ │ 🎤 SEMINAR                        [AKTIF] │ │
│ │ Contoh: Pernikahan Sinta & Arif           │ │ Tech Summit Indonesia 2027                │ │
│ │ Minggu, 28 Maret 2027                     │ │ Sabtu, 15 Mei 2027                        │ │
│ │ ⏳ 175 hari menuju Hari-H                 │ │ ⏳ 223 hari menuju Hari-H                 │ │
│ │ Tugas   15/50  [███░░░░░░░] 30%           │ │ Tugas   2/18   [█░░░░░░░░░] 11%           │ │
│ │ Bayar   Rp 33,55 Jt / pagu Rp 75 Jt       │ │ Bayar   Rp 0 / pagu Rp 75 Jt              │ │
│ │ Tamu    43 pax                            │ │ Belum ada peserta                         │ │
│ │ [ Masuk Workspace ➜ ]             [ ••• ] │ │ [ Masuk Workspace ➜ ]             [ ••• ] │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ 🍼 AQIQAH                       [SELESAI] │ │ ➕ BUAT ACARA BARU                        │ │
│ │ Aqiqah Muhammad Rayyan                    │ │ Pernikahan, Khitanan, Aqiqah, Peresmian,  │ │
│ │ Sabtu, 14 Februari 2026                   │ │ Launching, Ulang Tahun, Wisuda, Seminar,  │ │
│ │ Terbayar Rp 18,45 Jt — semua lunas        │ │ atau Acara Lainnya                        │ │
│ │                                           │ │                                           │ │
│ │                                           │ │                                           │ │
│ │                                           │ │                                           │ │
│ │ [ Lihat Ringkasan ➜ ]             [ ••• ] │ │ [ + Mulai Buat Acara ]                    │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

Catatan:

- **Header**: hanya tautan **Planner** yang ditambahkan di samping "Undangan" (bab 1.3). Header yang ada tidak punya sapaan nama pengguna dan FE tidak menambahkannya.
- **Baris Tamu/Peserta** hanya `guest_pax_count`. Kartu tidak punya jumlah undangan, jadi hanya pax yang dapat ditampilkan (bab 15, butir 2). Bila 0, seluruh baris diganti "Belum ada peserta" (tanpa label di depannya).
- **Tombol "Lihat Proyek Contoh" tidak digambar** karena kartu contoh sudah ada. Tombol itu hanya tampil bila `projects` belum memuat proyek `is_sample` (akun baru, atau contoh sudah dihapus).
- **Tombol utama kartu**: `active` → "Masuk Workspace ➜"; `completed`/`archived` → "Lihat Ringkasan ➜". Keduanya menuju `/dashboard/planner/{id}` (bab 15, butir 9).
- **Kartu terakhir "Buat Acara Baru"** selalu tampil di akhir grid (juga saat filter aktif, tidak ikut hitungan tab). Daftar jenis pada kartu dirangkai dari `label` registry (dipisah koma, "atau" sebelum yang terakhir), bukan teks tetap.
- Angka "15/50" dan "Rp 33,55 Jt" pada kartu baru muncul setelah Menu 02 dan 03 mengisi datanya; "43 pax" setelah Menu 05.

**Keadaan hari ini** (backend asli, modul 02–06 belum ada). Hanya baris yang bergantung modul lain yang berubah; sisanya sama dengan di atas. Respons asli: `total_tasks: 0`, `total_paid: 0`, `target_budget: 75000000`, `guest_pax_count: 0`.

```text
┌───────────────────────────────────────────┐
│ 💍 PERNIKAHAN            [CONTOH] [AKTIF] │
│ Contoh: Pernikahan Sinta & Arif           │
│ Minggu, 28 Maret 2027                     │
│ ⏳ 175 hari menuju Hari-H                 │
│ Tugas   Belum ada tugas                   │
│ Bayar   Rp 0 / pagu Rp 75 Jt              │
│ Belum ada tamu                            │
│ [ Masuk Workspace ➜ ]             [ ••• ] │
└───────────────────────────────────────────┘
```

### 7.2 Pemetaan data → tampilan

Data: `GET projects/summary` (`PortfolioSummary`) dan registry. Semua persentase lewat `percent(n, d)`; **tidak pernah** menampilkan `NaN%`/`Infinity`.

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu "Acara Aktif" | `active_count` | catatan kecil "tanpa proyek contoh" |
| Kartu "Acara Selesai" | `completed_count` | idem |
| Kartu "Total Pagu Aktif" | `combined_target_budget` | `formatRupiah`; 0 → "Rp 0" |
| Kartu "Acara Terdekat" | `nearest_project` | `{icon} {title} — {waktu}`: `upcoming` → `H-{days_remaining}`; `ongoing` → "Sedang berlangsung"; tidak ada → "Belum ada acara aktif". Tautan ke workspace-nya |
| Jumlah pada tab | dihitung klien dari `projects` | **menghitung proyek contoh**, sedangkan 4 kartu di atas tidak (disengaja; kartu diberi catatan) |
| Ikon + label jenis | registry `icon`, `label` | tidak ada daftar tertulis |
| Judul | `title` | teks biasa (bukan HTML) |
| Tanggal | `formatDate(event_date)`; bila `event_end_date`: "28 Mar – 30 Mar 2027" (`formatDateShort`) | |
| Baris waktu | tabel 5.1 | |
| **Tugas** | `completed_tasks`/`total_tasks` | `total_tasks = 0` → "Belum ada tugas" (tanpa batang); selain itu "15/50" + batang + `formatPercent(p)` |
| **Bayar** | `total_paid`, `target_budget` | `target_budget = 0` → "Pagu belum diisi"; selain itu "Rp {short(paid)} / pagu Rp {short(target)}" |
| **Tamu / Peserta** | `guest_pax_count`, `labels.guest` | 0 → seluruh baris menjadi "Belum ada {labels.guest.toLowerCase()}" (tanpa label di depannya); selain itu "{labels.guest} {n} pax" |
| Kartu selesai: lunas | `total_estimate > 0 && total_paid >= total_estimate` | "Terbayar {formatRupiahShort(total_paid)} — semua lunas"; `total_estimate > 0` tetapi belum lunas: "Terbayar Rp X dari estimasi Rp Y"; `total_estimate = 0` dan `target_budget > 0`: "Belum ada rincian biaya"; keduanya 0: tidak ada baris |
| Tombol utama kartu | `status` | `active` → "Masuk Workspace ➜"; `completed`/`archived` → "Lihat Ringkasan ➜"; keduanya ke `/dashboard/planner/{id}` |
| Filter **Kategori** | registry | "Semua" + satu opsi per jenis di registry (`{icon} {label}`), bukan hanya jenis yang dipakai proyek; nilai di URL = `key` (`?jenis=`) |
| Kartu ajakan "Buat Acara Baru" | registry | elemen terakhir grid, selalu tampil (bukan hasil filter, tidak ikut hitungan tab); teks daftar jenis dirangkai dari `label` registry; tombol → `/dashboard/planner/baru` |
| Tombol "Lihat Proyek Contoh" | `projects` | tampil hanya bila tidak ada proyek `is_sample` di `projects`; hilang begitu contoh dibuat, muncul lagi bila contoh dihapus |

### 7.3 Interaksi

- **Tab status, filter kategori, pencarian** bekerja di klien atas `projects` yang sudah termuat. Pencarian tidak membedakan huruf, mencocokkan `title` dan semua nilai teks `metadata` (nama mempelai, penyelenggara, dll.), ditunda 150 ms. Status filter disimpan di URL (`?status=active&jenis=wedding&q=sinta`) lewat `history.replaceState` agar muat ulang/berbagi tautan mempertahankannya. Urutan kartu = urutan server (aktif dulu, terdekat dulu; lalu selesai; lalu arsip, terbaru dulu); jangan diurutkan ulang.
- Hasil filter kosong: "Tidak ada acara yang cocok" + tombol "Atur ulang filter". Pengguna tanpa proyek: **empty state** "Mulai Rencanakan Momen Bersejarah Anda bersama Adicara" dengan `+ Rencanakan Acara Pertama Anda` (→ `/dashboard/planner/baru`) dan `Lihat Proyek Contoh`.
- **"Lihat Proyek Contoh"** (tampil hanya bila belum ada kartu contoh, tabel 7.2): `POST projects/sample` (tombol nonaktif selama permintaan); `201` atau `200` → redirect ke `/dashboard/planner/{project.id}`. Sampai modul 02–06 ada, contoh hanya berisi sesi dan metadata, jadi semua widgetnya "Segera hadir" (wajar, bukan bug). Kartu contoh berlencana CONTOH dan tidak ikut kartu metrik. Contoh yang sudah lewat tampak terlambat; hapus lalu buat ulang.
- **Menu `•••`** (satu `ProjectMenu` per kartu; dapat dioperasikan dengan papan ketik; ditutup dengan `Esc`):

  | Aksi | Tampil bila | Panggilan | Setelah sukses |
  |---|---|---|---|
  | Edit | bukan `archived` | membuka `EditEventDialog` (memuat `GET projects/:id`) | muat ulang halaman |
  | Tandai selesai | `status = active` | `PATCH projects/:id/status {"status":"completed"}` | muat ulang |
  | Aktifkan kembali | `completed` atau `archived` | `PATCH …/status {"status":"active"}` | muat ulang |
  | Arsipkan | `active` atau `completed` | `PATCH …/status {"status":"archived"}` | muat ulang |
  | Hapus | selalu (juga `archived`) | `DELETE projects/:id?confirm=<judul>` | `204` → hapus kartu dari DOM, perbarui hitungan; bila kosong tampilkan empty state |

- **Dialog Hapus** (`<dialog>` + `showModal()`): teks "Menghapus acara ini juga menghapus semua jadwal, anggaran, vendor, tamu, dan rundown di dalamnya. Tindakan ini tidak dapat dibatalkan." Pengguna **mengetik judul proyek**; tombol "Hapus Permanen" aktif hanya bila isian **sama persis** dengan `title` (peka huruf besar-kecil, tanpa memangkas). Dikirim sebagai `?confirm=` lewat `URLSearchParams` (di-encode). `400 invalid_input: confirm` hanya muncul bila judul berubah di tab lain → "Judul acara berubah. Muat ulang halaman."
- Proyek `archived`: lencana ARSIP; tombol edit dan aksi tulis di dalamnya nonaktif (backend menjawab `409 project_read_only`); hanya "Aktifkan kembali" dan "Hapus".
- Batas jumlah proyek: `409 project_limit_reached` dari wizard tampil sebagai pesan di tombol "Buat Acara Baru" (bab 11).
- Galat memuat (bab 2.2): banner di atas grid, tidak menutup halaman.

---

## 8. Wizard Buat Acara (`/dashboard/planner/baru`)

Tiga langkah dengan transisi halus; seluruh isi tiap langkah bergantung pada **jenis terpilih dari registry**. Wizard memerlukan JavaScript; tanpa JS tampilkan `<noscript>` "Aktifkan JavaScript untuk membuat acara". Parameter `?jenis=<key>` melompati langkah 1 (opsional).

### 8.1 Status dan `client_token`

Satu objek status di modul skrip: `{ step, type, title, eventDate, multiDay, endDate, prepStart, timezone, budget, metadata, templates, submitting, token }`. `token` dibuat **sekali saat halaman dibuka**: `crypto.randomUUID()`; bila tidak tersedia (halaman HTTP non-localhost tidak menyediakannya) pakai cadangan 32 karakter heksadesimal dari `crypto.getRandomValues`. Panjang harus 8–64 karakter. Token dikirim pada **setiap** percobaan kirim akhir dan tidak berubah antar-percobaan. Tombol "Simpan" nonaktif selama permintaan berjalan. Memuat ulang halaman membuat token baru (disengaja).

### 8.2 Langkah 1: Jenis acara

Sembilan kartu dari registry (ikon, `label`, satu kalimat deskripsi); satu dapat dipilih (`role="radiogroup"`, panah papan ketik). Teks deskripsi singkat per jenis boleh berupa kamus statis **dikunci oleh `key`** di `messages.ts` (bukan daftar jenis di komponen) dengan cadangan kosong. Pernikahan diberi tanda "Paling lengkap: berbasis template Wedding Planner".

### 8.3 Langkah 2: Detail

| Kolom | Kontrol | Aturan klien (cermin aturan server) | Dikirim sebagai |
|---|---|---|---|
| Judul\* | teks | dipangkas; 1–255 karakter (`charCount`); tanpa karakter kontrol/baris baru | `title` |
| Tanggal Hari-H\* | `type="date"` | `min` = `todayIn(timezone)`; `max` = `addDays(today, 1826)` (paling jauh 5 tahun); diperbarui saat zona berubah | `event_date` |
| ☐ Acara berlangsung lebih dari satu hari | kotak centang | membuka kolom berikut | – |
| Tanggal selesai | `type="date"` | `min` = Hari-H; `max` = `addDays(Hari-H, 30)`; saat Hari-H berubah dan tanggal selesai di luar rentang → kosongkan dan beri tahu | `event_end_date` (hanya bila dicentang) |
| Target anggaran | teks angka (`inputmode="numeric"`) | boleh kosong → 0; 0…10¹³; tampilkan berformat ribuan saat blur, simpan angka bulat | `target_budget` (angka; kosong = `0`) |
| Atribut subjek | dari `metadata_fields` (bab 8.3.1) | wajib bila `required` | `metadata` |
| **Lanjutan** (terlipat): Tanggal mulai persiapan | `type="date"` | default hari ini di zona proyek; `max` = Hari-H | `prep_start_date` (kirim hanya bila diubah dari default) |
| **Lanjutan**: Zona waktu | pilihan WIB `Asia/Jakarta`, WITA `Asia/Makassar`, WIT `Asia/Jayapura`, UTC, "Lainnya…" (ketik nama IANA) | harus cocok `ZONE_SHAPE` **dan** `isKnownZone`; selain itu "Zona waktu tidak dikenal." | `timezone` (kirim hanya bila bukan `Asia/Jakarta`) |

#### 8.3.1 Atribut subjek dinamis

Dibangkitkan dari `type.metadata_fields` **dalam urutan registry** (kolom khas jenis dulu, lalu `hero_title`, `venue_name`, `city`). Kolom `required` di bagian utama ditandai \*; `hero_title`, `venue_name`, `city` di subbagian "Tampilan hero & lokasi" (opsional di semua jenis).

| `type` | Kontrol | Atribut | Nilai kosong |
|---|---|---|---|
| `string` | `<input type="text" maxlength=max_len>` | batas = `max_len` karakter | **jangan kirim kuncinya** |
| `int` | `<input type="number" step="1" min max>` | `min`/`max` dari field; bilangan bulat | **jangan kirim kuncinya** (kirim string kosong untuk `int` ditolak `400`) |
| `date` | `<input type="date">` | – | jangan kirim kuncinya |

Nilai `int` dikirim sebagai **angka JSON**, bukan string. Contoh jenis: pernikahan → Nama Mempelai Wanita\*, Nama Mempelai Pria\*, Orang Tua kedua pihak; khitanan → Nama Anak\*, Nama Orang Tua; aqiqah → Nama Bayi\*, Tanggal Lahir (`date`), Nama Orang Tua, Jumlah Kambing (`int` 0–50); seminar → Nama Penyelenggara\*, Pembicara Utama, Perkiraan Peserta (`int`); dst. Itu hanya contoh keluaran registry; **jangan ditulis di komponen**.

Saran judul (opsional, nilai tambah kecil): bila `title` belum pernah diketik pengguna dan kolom subjek wajib terisi, isi otomatis (pernikahan: `Pernikahan {bride_name} & {groom_name}`; jenis lain: `{label} {nilai kolom wajib pertama}`).

#### 8.3.2 Galat server di formulir

Galat berbentuk `<field>: <alasan>` (`parseFieldError`) dipetakan ke kolom: `title`, `event_date`, `event_end_date`, `prep_start_date`, `timezone`, `target_budget`, `client_token` (selain kolom pengguna → galat umum), atau kunci metadata (mis. `invalid_metadata` + `bride_name: required` → kolom `bride_name`). Kolom diberi `aria-invalid="true"` dan pesan terhubung lewat `aria-describedby`; fokus dipindah ke kolom pertama yang salah (dan langkah wizard dibuka ke langkah yang memuatnya). Galat tanpa kolom tampil di atas formulir (`role="alert"`). Teks Indonesia dari `fieldErrorMessage` (bab 11.2).

### 8.4 Langkah 3: Template

Daftar centang **dibangkitkan dari `type.templates_available`** (template yang dimiliki jenis **dan** yang modulnya sudah punya applier di backend); yang tidak tersedia **tidak ditampilkan** (bukan disabled). Bila **semuanya kosong** (`timeline_presets = []` dan semua boolean `false`; keadaan backend sekarang), langkah ini dilewati dan tombol akhir langsung "Simpan & Buka Workspace".

| Bidang `templates_available` | Kontrol | Dikirim di `templates` |
|---|---|---|
| `timeline_presets` (tidak kosong) | centang "Jadwal persiapan" + pilihan preset (radio bila >1) | `timeline_preset: "<kunci>"`; tanpa centang: tidak dikirim |
| `include_adicara_tasks` | centang bawahan "Tambahkan tugas Adicara" | `include_adicara_tasks: true` |
| `admin_docs` | centang | `admin_docs: true` |
| `budget_lines` | centang | `budget_lines: true` |
| `shopping_list` | centang | `shopping_list: true` |
| `rundown` | centang | `rundown: true` |
| `playlist` | centang | `playlist: true` |

Teks centang dari kamus `TEMPLATE_LABELS` yang dikunci oleh **kunci template/preset** (bukan jenis acara): `wedding_pdf_44` → "Lengkap (44 tugas dari Wedding Planner)", `wedding_essential_22` → "Esensial (22 tugas)", `general` → "Jadwal umum (15 tugas)", `admin_docs` → "Berkas administrasi / perizinan", `budget_lines` → "Pos anggaran (nominal 0 atau persentase pagu)", `shopping_list` → "Daftar belanja", `rundown` → "Rundown per sesi", `playlist` → "Playlist". Jumlah persis di luar nama preset **tidak** ditulis (berbeda per jenis dan belum ada sumbernya di API). Semua centang awal aktif **kecuali** preset opsional; teks bantu: "Template hanya struktur dan perkiraan yang bisa diubah semua, tanpa pembayaran, vendor, atau tamu." Sesi bawaan selalu dibuat (tidak ada centangnya).

### 8.5 Pengiriman dan hasil

`POST projects` lewat `plannerFetch('projects', { method: 'POST', body })`. Contoh isi (kunci opsional yang kosong **tidak dikirim**):

```json
{
  "event_type": "wedding",
  "title": "Pernikahan Sinta & Arif",
  "event_date": "2027-03-28",
  "target_budget": 75000000,
  "metadata": { "bride_name": "Sinta Maharani", "groom_name": "Arif Pratama", "hero_title": "Bismillah Nikah" },
  "templates": {},
  "client_token": "0b8c9e5e-4a4e-4c0a-9e44-6c2f5d1b7a10"
}
```

- `201` **dan** `200` (`created: false`) sama-sama sukses → lanjut ke langkah berikut. `200` terjadi pada klik ganda/kirim ulang setelah timeout; jangan tampilkan galat.
- Bila `created` dan `applied` memuat `warnings` apa pun, tampilkan antarmuka ringkasan sebelum redirect: per kunci `applied` → "{label}: {created} dibuat" (+ "{skipped} dilewati" bila >0) dan catatan dari `WARNING_LABELS` (`lead_time_compressed`: "Jadwal dimampatkan karena waktu persiapan lebih pendek dari standar."; `lead_time_too_short`: "Waktu persiapan sangat singkat; semua tugas dijepit ke rentang hari ini sampai Hari-H."; kode lain: "Ada catatan pada {label}."), dengan tombol "Buka Workspace". Tanpa warnings: redirect langsung ke `/dashboard/planner/{project.id}`. `warnings` adalah catatan, **bukan** galat.
- Galat: `400` → bab 8.3.2; `409 project_limit_reached` → pesan di atas formulir dan tombol Simpan tetap bisa dicoba (pengguna menghapus acara lain lalu kembali); `429 rate_limited` (10 pembuatan/menit) → "Terlalu banyak permintaan..."; `400 invalid_template` → muat ulang registry dan kembali ke langkah 3 (daftar template berubah); `503`/jaringan → "Layanan sedang tidak dapat dijangkau" dan token **dipertahankan** sehingga mencoba lagi aman.
- Setelah sukses, hapus status wizard.

---

## 9. Workspace Dashboard (`/dashboard/planner/[id]`)

### 9.1 Wireframe

**Keadaan akhir** (modul 02–06 sudah ada; Proyek Contoh). Daftar Project Switcher digambar terbuka; default-nya tertutup (hanya tombol). Tautan sub-navigasi aktif karena semua menu sudah ada:

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Adicara                                                           Undangan  Planner  Keluar │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ]  [Dashboard] Timeline Keuangan Vendor Tamu Rundown │
│   ├ 💍 Contoh: Pernikahan Sinta & Arif (Contoh · Aktif · H-175)  ✓ Sedang dibuka            │
│   ├ 🎤 Tech Summit Indonesia 2027 (Aktif · H-223)                                           │
│   ├ 🍼 Aqiqah Muhammad Rayyan (Selesai)                                                     │
│   └ ➕ Rencanakan acara lain…                                                               │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ BISMILLAH NIKAH                                                              [ Edit Acara ] │
│ SINTA MAHARANI & ARIF PRATAMA                                                      [CONTOH] │
│ 📅 Minggu, 28 Maret 2027   📍 Sasana Kriya Ballroom, Jakarta Selatan                        │
│ ⏳ 175 HARI menuju Hari-H                                                                   │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ TIMELINE & TUGAS                 [Buka ➜] │ │ KEUANGAN & TABUNGAN              [Buka ➜] │ │
│ │ 15 dari 50 selesai                        │ │ [donut: terbayar | sisa]                  │ │
│ │ [███░░░░░░░] 30%                          │ │ Estimasi Rp 74.150.000                    │ │
│ │ Berjalan: 2   Terlambat: 0                │ │ Terbayar Rp 33.550.000 (45,2%)            │ │
│ │ Tugas mendesak (7 hari ke depan):         │ │ Sisa tagihan Rp 40.600.000                │ │
│ │   Tidak ada tugas mendesak                │ │ Tabungan Rp 47.500.000 / Rp 75.000.000    │ │
│ │                                           │ │ (!) Kas bebas Rp 13.950.000 · kurang      │ │
│ │                                           │ │     Rp 26.650.000                         │ │
│ │                                           │ │ Lunas 3 · Sebagian 4 · Belum 7            │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ VENDOR REKANAN                   [Buka ➜] │ │ BUKU TAMU & RSVP                 [Buka ➜] │ │
│ │ 7 dari 10 kategori terkunci               │ │ 20 Tamu · 43 pax                          │ │
│ │ [███████░░░] 70%                          │ │ Pihak Pria 10 · Pihak Wanita 10           │ │
│ │ Kandidat berjalan: 3                      │ │ Terkirim: 14                              │ │
│ │                                           │ │ Hadir 12 (27 pax) · Ragu 2                │ │
│ │                                           │ │ Tidak 2 · Belum 4                         │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ KELENGKAPAN BERKAS KUA           [Buka ➜] │ │ RUNDOWN HARI-H                   [Buka ➜] │ │
│ │ 9 dari 15 berkas                          │ │ 3 sesi · 33 agenda · 30 lagu              │ │
│ │ [██████░░░░] 60%                          │ │ Sesi terdekat: 💍 Lamaran — 10 Okt 2026   │ │
│ │ Menunggu: FC KTP Ortu CMP, FC KTP Ortu    │ │                                           │ │
│ │ CMW, Pas Foto 2x3, Pas Foto 4x6, Surat    │ │                                           │ │
│ │ ket. belum nikah (+1 lainnya)             │ │                                           │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

**Keadaan hari ini** (backend asli, `ready: true` hanya untuk Dashboard, semua widget `null`). Sub-navigasi menu yang belum ada tampil nonaktif (tanda kurung = abu-abu, bukan tautan); judul switcher dipotong dengan ellipsis CSS bila tidak muat; widget tanpa tautan Buka:

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Adicara                                                           Undangan  Planner  Keluar │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ [ 💍 Contoh: Pernikahan… ▼ ]    [Dashboard] (Timeline) (Keuangan) (Vendor) (Tamu) (Rundown) │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ BISMILLAH NIKAH                                                              [ Edit Acara ] │
│ SINTA MAHARANI & ARIF PRATAMA                                                      [CONTOH] │
│ 📅 Minggu, 28 Maret 2027   📍 Sasana Kriya Ballroom, Jakarta Selatan                        │
│ ⏳ 175 HARI menuju Hari-H                                                                   │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ TIMELINE & TUGAS                          │ │ KEUANGAN & TABUNGAN                       │ │
│ │ Segera hadir                              │ │ Segera hadir                              │ │
│ │                                           │ │                                           │ │
│ │                                           │ │                                           │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ VENDOR REKANAN                            │ │ BUKU TAMU & RSVP                          │ │
│ │ Segera hadir                              │ │ Segera hadir                              │ │
│ │                                           │ │                                           │ │
│ │                                           │ │                                           │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
│ ┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐ │
│ │ BERKAS & PERIZINAN                        │ │ RUNDOWN HARI-H                            │ │
│ │ Segera hadir                              │ │ Segera hadir                              │ │
│ │                                           │ │                                           │ │
│ │                                           │ │                                           │ │
│ └───────────────────────────────────────────┘ └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

Empat keadaan sebuah widget (contoh Vendor; aturannya di tabel 9.4):

```text
┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐
│ VENDOR REKANAN                   [Buka ➜] │ │ VENDOR REKANAN                   [Buka ➜] │
│ 7 dari 10 kategori terkunci               │ │ Belum ada kategori vendor                 │
│ [███████░░░] 70%                          │ │                                           │
│ Kandidat berjalan: 3                      │ │                                           │
│                                           │ │                                           │
│                                           │ │                                           │
└───────────────────────────────────────────┘ └───────────────────────────────────────────┘
┌───────────────────────────────────────────┐ ┌───────────────────────────────────────────┐
│ VENDOR REKANAN                            │ │ VENDOR REKANAN                   [Buka ➜] │
│ Segera hadir                              │ │ Data widget belum dapat dimuat            │
│                                           │ │ [Muat ulang]                              │
│                                           │ │                                           │
│                                           │ │                                           │
│                                           │ │                                           │
└───────────────────────────────────────────┘ └───────────────────────────────────────────┘
```

Catatan:

- **Hero**: kapital berasal dari CSS `text-transform` pada judul dan nama (datanya tetap "Bismillah Nikah", "Sinta Maharani"). Lencana CONTOH di baris nama. Ikon 📅/📍 hanya ilustrasi (bab 9.2).
- **Timeline**: "Berjalan: n" hanya tampil bila `in_progress > 0` (Proyek Contoh punya 2), "Terlambat: n" selalu. Daftar "Tugas mendesak" kosong menampilkan "Tidak ada tugas mendesak". Subjudul daftar "Tugas mendesak (7 hari ke depan)": jendela dan urutannya ditetapkan [BE §5.1](BE.md#51-response) (belum selesai, tenggat ≤ hari ini + 7, yang sudah lewat ikut).
- **Keuangan**: donut SVG (bukan teks) di atas daftar angka; ketiga angka kas memakai `formatRupiah` penuh, bukan singkatan.
- **Tamu**: "20 **T**amu" memakai `labels.guest` apa adanya (seminar: "Peserta").
- **Berkas**: judul = `admin_docs.label`; pada `soon`/`error` datanya tidak ada, jadi judul tetap "Berkas & Perizinan" (tabel 9.4; bab 15, butir 11).
- **Rundown**: "Sesi terdekat: Lamaran — 10 Okt 2026" butuh sesi Lamaran bertanggal. [BE §4.5](BE.md#45-proyek-contoh): overlay sampel mengisinya dengan tanggal mulai tugas `wedding.lamaran` (H-169), jadi baru ada setelah Menu 02 terdaftar. Hari ini Lamaran tanpa tanggal (jenis ini biasanya beda hari, sehingga tidak ikut Hari-H) dan baris itu tidak tampil (bab 15, butir 10).

### 9.2 Hero dan hitung mundur (`HeroCountdown`)

Sumber: blok `project` dari dashboard (SSR dan partial). **Fase datang dari server**, bukan dihitung dari jam perangkat.

| Elemen | Aturan |
|---|---|
| Judul besar | `metadata.hero_title` bila ada (huruf kapital lewat CSS `text-transform`, bukan data) |
| Nama subjek | pernikahan: `bride_name` `&` `groom_name` (urutan PDF: wanita dulu); jenis lain: `title`, subjudul = nilai kolom wajib pertama dari registry (mis. `organizer_name`). Tampil kapital lewat CSS |
| Tanggal | `formatDate(event_date)`; bila `event_end_date`: "Sabtu, 15 Mei – Minggu, 16 Mei 2027" |
| Lokasi | `venue_name` dan/atau `city` (gabung dengan ", "); sembunyikan bila keduanya kosong |
| Penanda | lencana CONTOH bila `is_sample`; lencana status bila bukan `active` |
| Tombol | **Edit Acara** (nonaktif bila `archived`, dengan teks bantu "Acara diarsipkan") |

Mode tampilan menurut `phase`:

| `phase` | Tampilan |
|---|---|
| `upcoming`, `days_remaining > 7` | "**{n}** HARI menuju Hari-H" (statis) |
| `upcoming`, `days_remaining` 1…7 | `Hari : Jam : Menit : Detik` berdetak tiap detik |
| `ongoing`, `total_days = 1` | "🎉 Hari-H telah tiba" |
| `ongoing`, `total_days > 1` | "Hari ke-{day_number} dari {total_days}" |
| `finished` | "🏁 Acara telah berlangsung pada {formatDate(event_date)}" + ajakan **Tandai selesai** bila `status = active` (`PATCH …/status`) |

Algoritma detik (`upcoming`, 1…7 hari): saat skrip dimuat, hitung `skew = Date.parse(server_time) − Date.now()`; `target = zonedMidnight(event_date, timezone)` (bab 5.2); tiap detik `countdown(target, Date.now() + skew)` dan tulis ke empat sel. Perangkat dengan jam salah menampilkan hitungan yang sama dengan server berkat `skew`. Saat sisa mencapai 0 sementara `phase` masih `upcoming`: ambil partial segera; bila partial masih `upcoming`, coba lagi setelah 5 detik, maksimal 3 kali, lalu ikuti polling biasa. **Jangan** mengubah fase secara lokal.

Aksesibilitas hitung mundur: wadah `role="timer"` dengan `aria-live="off"` (detik tidak diumumkan); sediakan teks `visually-hidden` statis "{n} hari lagi menuju Hari-H" yang diperbarui hanya saat hari berubah; hormati `prefers-reduced-motion` (tanpa animasi angka; pembaruan teks tetap jalan). Mobile: grid 2×2 untuk keempat sel.

Fungsi `initHeroCountdown(root)` harus **idempoten** dan dipanggil lagi setelah partial menggantikan DOM (hentikan interval lama sebelum memulai yang baru).

### 9.3 Edit Acara (`EditEventDialog`)

Dibuka dari hero atau menu `•••` di Hub. Alur:

1. Buka `<dialog>`; panggil `GET projects/:id` (BFF) → `{ project, shiftable_tasks }` (tampilkan loading; galat → pesan dan tombol coba lagi). Isi formulir dari `project`.
2. Kolom: **Judul**, **Tanggal Hari-H**, ☐ multi-hari + **Tanggal selesai**, **Target anggaran**, atribut subjek (sama dengan wizard 8.3.1, nilai awal dari `project.metadata`), tombol **Kelola Sesi** (membuka `SessionManager`). *Tidak* mengedit zona waktu atau tanggal mulai persiapan di UI ini (bab 15, butir 4).
3. Simpan hanya **yang berubah** (lacak "kotor" per kolom). Tidak ada perubahan → tutup tanpa permintaan. Pembentukan `PATCH`:
   - `title`, `target_budget`: kirim bila berubah.
   - `event_date`: kirim bila berubah. Bila **tanggal selesai tidak disentuh**, jangan kirim `event_end_date`: server menggeser tanggal selesai mengikuti Hari-H (durasi tetap); tampilkan pratinjau "Tanggal selesai ikut bergeser menjadi {tanggal}". Bila pengguna mengubahnya, kirim `event_end_date`. Bila kotak multi-hari **dimatikan** dan sebelumnya ada tanggal selesai: `clear_event_end_date: true` (jangan kirim bersama `event_end_date`).
   - `metadata`: **JSON Merge Patch** — hanya kunci yang berubah; kunci yang dikosongkan → `null`; kolom `int` sebagai angka; tidak mengirim kunci yang tak berubah (dua tab yang mengedit kolom berbeda tidak saling menghapus). Kunci wajib tidak boleh dikosongkan di klien.
4. **Mengubah tanggal Hari-H** dengan `shiftable_tasks > 0`: **sebelum** mengirim, tampilkan konfirmasi "Geser {n} tugas template yang belum selesai ikut {maju|mundur} {|selisih|} hari?" (`selisih = daysBetween(lama, baru)`; tombol **Ya, geser** → `shift_tasks: true`; **Tidak, biarkan** → `shift_tasks: false`/tidak dikirim; **Batal**). Keputusan dikirim dalam **satu** `PATCH` atomik (penggeseran dan perubahan tanggal satu transaksi; klik ganda tidak menggeser dua kali). Dengan `shiftable_tasks = 0` atau tanggal tidak berubah, jangan tanya dan jangan kirim `shift_tasks`.
5. Respons `{ project, warnings, hints }`: simpan, tutup dialog, muat ulang partial. Tampilkan catatan hasil (toast/status `aria-live="polite"`):
   - `warnings` `tasks_overdue_after_shift` → "{count} tugas kini melewati tenggat setelah digeser."
   - `hints` `sessions_dates_unchanged` → "{count} sesi bertanggal lain tidak ikut berubah." (informasi)
   - `hints` `tasks_not_shifted` → tidak ditampilkan oleh FE resmi (pengguna sudah ditanya).
6. Galat: `<field>: <alasan>` ke kolom (bab 8.3.2). `invalid_date` yang menyebut `prep_start_date` (mis. Hari-H digeser ke sebelum tanggal mulai persiapan) → tampil di kolom Hari-H: "Tanggal acara tidak boleh sebelum tanggal mulai persiapan ({prep_start_date})". `409 project_read_only` (diarsipkan di tab lain) → pesan dan nonaktifkan tombol. `409 conflict` → "Data berubah di tempat lain. Muat ulang lalu coba lagi." `404` → arahkan ke Hub.

Tanggal lampau **boleh** diisi pada edit (koreksi data), tetapi `min`-nya tidak dipasang; wizard tetap melarang.

### 9.4 Widget

Enam `WidgetCard` dengan empat keadaan:

| Keadaan | Kondisi | Tampilan |
|---|---|---|
| `soon` | nilai `null` (modul belum ada) | judul + "Segera hadir"; **tanpa** tautan Buka |
| `error` | `isWidgetError(w)` | "Data widget belum dapat dimuat" (`timeout`: "…terlalu lama dimuat") + tombol **Muat ulang** yang memperbarui seluruh partial (hanya widget itu yang diganti bila tombolnya diklik) |
| `empty` | ada data tetapi penyebut 0 (aturan per widget) | pesan ajakan, **tanpa persen** |
| `filled` | selain itu | isi di bawah |

Tombol **Buka ➜** hanya tampil bila widget bukan `null` (modulnya sudah dibangun, berarti halaman menunya ada); pada `error` tombol Buka tetap tampil bersama **Muat ulang**, karena halaman menu masih dapat dibuka. Rute: Timeline `/dashboard/planner/{id}/timeline`, Keuangan `/keuangan`, Vendor `/vendor`, Tamu `/tamu`, Legalitas `/timeline?tab=dokumen`, Rundown `/rundown`.

| Widget | Judul | Ikon | Bidang yang dibaca → tampilan | Keadaan kosong |
|---|---|---|---|---|
| **Timeline & Tugas** (`timeline`) | Timeline & Tugas | `ph-timer` | `completed`/`total` → "{completed} dari {total} selesai" + batang + `formatPercent`; `in_progress` → "Berjalan: n" (sembunyikan bila 0); `overdue` → "Terlambat: n" (gaya peringatan bila >0); `urgent[]` (maks 3; belum selesai, tenggat ≤ hari ini + 7, yang sudah lewat ikut, urut tenggat) di bawah subjudul "Tugas mendesak (7 hari ke depan)" → baris `{title} · {formatDateShort(due_date)} · {label}` dengan label dari `days_left`: `<0` "Terlambat {n} hari", `0` "Hari ini", `1` "Besok", selain itu "{n} hari lagi"; daftar kosong → "Tidak ada tugas mendesak" | `total = 0` → "Belum ada tugas." + tautan "Buat jadwal otomatis" |
| **Keuangan & Tabungan** (`finance`) | Keuangan & Tabungan | `ph-wallet` | donut SVG dua segmen: terbayar = `total_estimate − remaining`, sisa = `remaining` (jumlahnya = `total_estimate`); teks: "Estimasi {rupiah}", "Terbayar {rupiah} ({formatPercent(percent(total_paid,total_estimate),1)})", "Sisa tagihan {rupiah(remaining)}"; bila `overpaid > 0`: "Ada kelebihan bayar {rupiah}"; bila `total_estimate > target_budget > 0`: "Estimasi melebihi pagu {rupiah}"; baris tabungan "Tabungan {rupiah(savings_collected)} / {rupiah(savings_target)}"; **badge kas**: `cash_deficit = 0` → hijau "Kas aman"; selain itu peringatan "Kas bebas {rupiah(free_cash)} · kurang {rupiah(cash_deficit)}"; `line_status` → "Lunas {settled} · Sebagian {partial} · Belum {unpaid}" | `total_estimate = 0` → bagian anggaran "Belum ada anggaran" (bagian tabungan tetap bila `savings_target > 0 \|\| savings_collected > 0`); `target_budget = 0` → "Pagu belum diisi" |
| **Vendor Rekanan** (`vendors`) | Vendor Rekanan | `ph-handshake` | `categories_fixed`/`categories_total` → "{fixed} dari {total} kategori terkunci" + batang + persen; `pipeline` → "Kandidat berjalan: n"; `skipped > 0` → "{skipped} kategori dilewati" | `categories_total = 0` → "Belum ada kategori vendor" |
| **Buku Tamu & RSVP** (`guests`) | `labels.guest_menu` jenis acara | `ph-users-three` | "{invitations} {labels.guest} · {total_pax} pax"; per sisi `sides[]`: baris `{label} {invitations}` (sisi `key = null` → "Belum ditentukan"; jenis tanpa sisi: satu baris tunggal atau disembunyikan); "Terkirim: {sent}"; RSVP: hadir `attending` ({attending_pax} pax), ragu `maybe`, tidak `declined`, belum `unconfirmed` (empat angka selalu berjumlah `invitations`; tampilkan apa adanya, jangan "dibetulkan") | `invitations = 0` → "Belum ada {labels.guest.toLowerCase()}" |
| **Legalitas / Perizinan** (`admin_docs`) | `data.label` (diisi provider Menu 02; contoh pernikahan: "Kelengkapan Berkas KUA"). Pada `soon`/`error`/label kosong, datanya tidak ada, jadi pakai judul tetap "Berkas & Perizinan" | `ph-scroll` | "{completed} dari {total} berkas" + batang + persen; `pending[]` (maks 5) → "Menunggu: a, b, c"; bila `total − completed > pending.length` tambah "(+{selisih} lainnya)" | `total = 0` → "Belum ada berkas" + tombol "Tambah" (→ `/timeline?tab=dokumen`) |
| **Rundown Hari-H** (`rundown`) | Rundown Hari-H | `ph-clipboard-text` | "{sessions} sesi · {agendas} agenda · {songs} lagu"; `agendas > 0` dan `completed > 0` → batang `percent(completed, agendas)`; `current_agenda` ≠ `null` → "Sedang berjalan: {activity} ({session})" dengan titik berdenyut (diam bila `prefers-reduced-motion`); `next_session` → "Sesi terdekat: {ikon} {name} — {formatDateShort(date)}{, start_time}" — ikon dicari dari `dashboard.sessions` dengan `name` sama, tanpa ikon bila tak ketemu | `sessions = 0` → "Belum ada sesi" + tombol "Kelola sesi" (membuka `SessionManager`); `agendas = 0` → "Belum ada agenda" |

Nilai negatif atau tak terduga: tampilkan apa adanya setelah format, kecuali pembagi nol (selalu lewat `percent`). Semua string dari API dirender sebagai **teks** (Astro meng-escape; di skrip klien pakai `textContent`, bukan `innerHTML`).

### 9.5 Pembaruan berkala (polling) lewat partial

Satu sumber render: `widgets.astro` (`export const partial = true`, `prerender = false`) menghasilkan **hero + grid widget** dari komponen yang sama dengan `[id]/index.astro`; ia memanggil `GET projects/:id/dashboard` (batas 60/menit/pengguna). Skrip halaman:

- Interval = 30 detik ± 5 detik acak. **Hanya saat tab terlihat** (`document.visibilityState === 'visible'`); berhenti saat tersembunyi dan, bila sudah lewat dari 30 detik sejak pembaruan terakhir, langsung mengambil saat terlihat kembali.
- `fetch('/dashboard/planner/{id}/widgets', { credentials: 'same-origin', headers: { Accept: 'text/html' } })`. Sukses: ganti isi `[data-planner-live]` (hero + grid) dengan HTML yang diterima, lalu panggil `initHeroCountdown` lagi. Pertahankan fokus: bila elemen fokus ada di dalam area yang diganti, kembalikan fokus ke elemen ber-`id`/`data-focus-key` yang sama.
- Galat `5xx`/`429`/jaringan: **jangan** mengosongkan isi; mundur ke 60 detik, tampilkan indikator kecil "Pembaruan tertunda" (`aria-live="polite"`); sukses berikutnya menghapusnya dan mengembalikan 30 detik. `401`/pengalihan ke halaman login: `location.assign('/auth/masuk')`. `404`: berhenti polling, banner "Acara tidak ditemukan" + tautan ke Hub.
- Jangan polling saat dialog terbuka (agar isian tidak hilang); lanjutkan setelah ditutup.
- *(Belum diuji di browser: `export const partial = true` didukung kode Astro 7.3.5 (`mod.partial` di `fetch-state`), tetapi belum dijalankan dari halaman nyata. Bila ternyata bermasalah, alternatifnya JSON lewat BFF `GET dashboard` + fungsi render klien; itu menggandakan logika render sehingga sengaja tidak dipilih.)*

### 9.6 Kelola Sesi (`SessionManager`, modal dari Edit Acara)

Daftar dari `GET projects/:id/sessions` (BFF); sesi sistem `all` (`is_system`) **tidak ditampilkan**. Tiap baris: pegangan urut, ikon (teks emoji, maks 8 karakter), nama, tanggal, jam mulai/selesai (`type="time"`), lokasi; tombol Simpan per baris, tombol Hapus.

| Kolom | Aturan klien (cermin server) | Kirim |
|---|---|---|
| Nama\* | dipangkas; 1–100 karakter (`charCount`); tanpa karakter kontrol; **unik di proyek tanpa membedakan huruf besar-kecil** (server `409 session_name_taken` → tampil di bawah kolom) | `name` |
| Ikon | 1–8 karakter (`charCount`, bukan `.length`); kosong → server memakai 📋 | `icon` |
| Tanggal | `type="date"`; bebas (boleh di luar rentang acara) | `session_date` |
| Jam | `type="time"` → `"HH:MM"`; kosong = tidak ada | `start_time`, `end_time` |
| Lokasi | maks 150 karakter | `location_name` |

- **`PUT` mengganti penuh**: formulir selalu mengirim **semua** kolom; kolom kosong cukup tidak dikirim/dikirim kosong (server mengosongkannya). `scope_key` tidak pernah dikirim dan tidak ditampilkan sebagai bisa diubah.
- **Tambah sesi**: `POST` (`201`); `409 session_limit_reached` ("Maksimal 20 sesi per acara."). Nol sesi diperbolehkan.
- **Urutkan**: seret-lepas (HTML5 DnD) **dan** tombol naik/turun untuk papan ketik. Setelah lepas, `POST sessions/reorder` dengan `session_ids` = **semua** sesi non-sistem dalam urutan baru, tepat sekali tiap sesi. Pembaruan optimistis; bila galat (`400 invalid_input` karena daftar berubah di tab lain, atau jaringan), kembalikan urutan lama dan muat ulang daftar.
- **Hapus**: buka dialog setelah menyegarkan `GET …/sessions` untuk `usage` terbaru. Isi dialog:
  - Teks pemakaian dari `USAGE_LABELS`, hanya yang >0: tugas, vendor, pos anggaran, item skenario, item belanja, `{labels.guest.toLowerCase()}`, agenda rundown. Semua 0 → "Sesi ini belum dipakai."
  - **Pindahkan ke**: `<select>` berisi "Umum / Semua Sesi" (`all`, default) dan semua sesi lain kecuali yang dihapus; nilai = `scope_key`.
  - Bila `usage.agendas > 0`: kotak centang wajib "Hapus juga {n} agenda rundown di sesi ini" — tanpa dicentang, tombol Hapus nonaktif (server menolak `409 session_in_use`).
  - Catatan tetap: "Pos anggaran dengan kunci yang sama di sesi tujuan digabung sebagai pos biasa. Total anggaran tidak berubah." (FE **tidak** bisa menghitung pos mana yang bentrok; datanya milik Menu 03.)
  - Kirim `DELETE …/sessions/{id}?move_to={scope_key}&delete_agendas={true|false}`; `204` → hapus baris, perbarui partial. `409 session_in_use` (agenda bertambah di tab lain): perbarui `usage` dari `error.details.usage` dan minta pengguna meninjau. `400 invalid_scope`: muat ulang daftar. `409 conflict`: "Data berubah di tempat lain. Muat ulang lalu coba lagi."
- Perubahan sesi memuat ulang partial dashboard (daftar sesi di widget Rundown dan ikon).

---

## 10. Project Switcher dan PlannerLayout

`PlannerLayout.astro` membungkus `AppLayout` (tanpa mengubahnya) dan menambah **bilah sekunder** di awal slot: kiri Project Switcher, kanan sub-navigasi enam menu. Props: `{ title, description, projectId, current: { id, icon, title, event_type }, active: 'dashboard' | 'timeline' | 'keuangan' | 'vendor' | 'tamu' | 'rundown' }`. Layout memuat daftar proyek **ringan** sekali per permintaan: `GET projects?status=all&view=lite` (tanpa agregat; `ProjectLite[]`). Halaman menu tidak memuat daftar proyek sendiri.

- **Sub-navigasi:** konstanta `MENUS = [{ key, label, path, ready }]`. Hanya menu dengan `ready: true` yang berupa tautan; sisanya tampil nonaktif ("Segera"), karena halaman 02–06 belum ada. Saat ini hanya `dashboard` yang `ready`. Setiap menu baru yang dibangun mengubah satu flag ini. Menu aktif memakai `aria-current="page"`. Label menu **Tamu** = `labels.guest` jenis proyek yang sedang dibuka (seminar: "Peserta"), diambil dari registry lewat `current.event_type`; label lain tetap. Layout memakai registry (cache 1 jam, bab 5.3) untuk ikon dan label ini.
- **Switcher:** tombol disclosure (`aria-haspopup="listbox"`, `aria-expanded`), daftar:
  1. proyek yang sedang dibuka ditandai "✓ Sedang dibuka";
  2. proyek aktif lain (urut seperti API) dengan tanda waktu;
  3. proyek selesai/arsip;
  4. `➕ Rencanakan acara lain…` → `/dashboard/planner/baru`.
  Format item: `{icon registry} {title}` + keterangan: `active`/`upcoming` → "Aktif · H-{days_remaining}"; `active`/`ongoing` → "Berlangsung"; `active`/`finished` → "Aktif · sudah lewat"; `completed` → "Selesai"; `archived` → "Arsip"; `is_sample` → awalan "Contoh · ". Daftar bisa dioperasikan dengan panah/Home/End/Enter/Esc.
- **Berpindah** menuju URL yang sama dengan `id` baru dan menu yang sedang dibuka: `/dashboard/planner/{id_tujuan}{subpath}` (`subpath` = bagian setelah `/{id}` pada `Astro.url.pathname`, tanpa query; mis. `/timeline`). Bila menu itu belum `ready` untuk tujuan, jatuhkan ke dashboard.
- Bila daftar gagal dimuat, bilah menampilkan hanya proyek yang sedang dibuka (dari `current`) tanpa daftar.
- Mobile (< 768px): daftar menjadi *bottom sheet* (`<dialog>`); sub-navigasi bisa digulir horizontal.

---

## 11. Pesan galat

### 11.1 Kode → teks (`plannerErrorMessage(status, code)`)

| Kode / kondisi | Teks Indonesia | Muncul pada |
|---|---|---|
| `invalid_json` | "Permintaan tidak valid. Muat ulang halaman lalu coba lagi." | semua |
| `unsupported_media_type` (BFF) | idem | semua (bug klien) |
| `invalid_input`, `invalid_date`, `invalid_timezone`, `invalid_metadata` | **per kolom** (11.2); tanpa kolom: "Periksa kembali data Anda." | wizard, edit, sesi |
| `invalid_timezone` | "Zona waktu tidak dikenal." | wizard, edit |
| `invalid_template` | "Template yang dipilih belum tersedia. Muat ulang halaman." | wizard |
| `invalid_scope` | "Sesi tujuan tidak valid. Muat ulang daftar sesi." | hapus sesi |
| `unauthorized` / pengalihan login / `401` | tanpa teks: arahkan ke `/auth/masuk` | semua |
| `not_found` | proyek: "Acara tidak ditemukan." (halaman: `/404`); sesi: "Sesi tidak ditemukan. Muat ulang daftar." | semua |
| `project_read_only` | "Acara ini diarsipkan. Buka arsip untuk mengubahnya." | semua penulisan |
| `project_limit_reached` | "Jumlah acara sudah mencapai batas. Hapus acara yang tidak dipakai lalu coba lagi." | wizard, contoh |
| `session_name_taken` | "Nama sesi sudah dipakai." | sesi |
| `session_limit_reached` | "Maksimal 20 sesi per acara." | sesi |
| `session_in_use` | "Sesi masih punya agenda rundown. Centang untuk menghapus agendanya juga." | hapus sesi |
| `conflict` | "Data berubah di tempat lain. Muat ulang lalu coba lagi." | semua penulisan |
| `rate_limited` / `429` | "Terlalu banyak permintaan. Tunggu sebentar lalu coba lagi." | pembuatan, dashboard |
| `service_unavailable`, `503`, `5xx`, `internal`, jaringan | "Layanan sedang tidak dapat dijangkau. Silakan coba lagi sebentar lagi." | semua |
| tak dikenal | "Permintaan belum berhasil. Periksa kembali data Anda." | semua |

### 11.2 Alasan per kolom (`fieldErrorMessage(field, reason, ctx)`)

Format backend `<field>: <alasan>` memakai alasan Inggris yang stabil dalam praktik tetapi **tidak dijamin**; terjemahan di sini adalah pemetaan pola dengan cadangan per kode.

| Pola alasan (regex) | Teks (`{label}` = label kolom) |
|---|---|
| `^must not be empty$` / `^required$` | "{label} wajib diisi." |
| `^must be at most (\d+) characters$` | "{label} maksimal $1 karakter." |
| `^must not contain control characters$` | "{label} tidak boleh berisi baris baru atau karakter kontrol." |
| `^must be a valid date in YYYY-MM-DD format$` / `^must be a date in YYYY-MM-DD format$` | "{label} bukan tanggal yang valid." |
| `^must not be before today$` | "Tanggal acara tidak boleh sebelum hari ini." |
| `^must be within 5 years from today$` | "Tanggal acara paling jauh 5 tahun dari hari ini." |
| `^must not be after event_date$` | "Tanggal mulai persiapan tidak boleh setelah tanggal acara." |
| `^must not be before event_date$` | "Tanggal selesai tidak boleh sebelum tanggal acara." |
| `^must be within 30 days of event_date$` | "Acara paling lama 30 hari." |
| `^must be between (\d+) and (\d+)$` | "{label} harus antara $1 dan $2." |
| `^must be at least (\d+)$` / `^must be at most (\d+)$` | "{label} minimal $1." / "{label} maksimal $1." |
| `^must be an integer$` / `^must be a string$` | "{label} tidak valid." (kesalahan klien: kontrol yang salah) |
| `^unknown key for this event type$` | "Kolom {label} tidak berlaku untuk jenis acara ini." (kesalahan klien) |
| lainnya | "{label} tidak valid." |

`{label}` diambil dari `metadata_fields[].label` untuk kunci metadata, atau kamus tetap: `title` "Judul", `event_date` "Tanggal acara", `event_end_date` "Tanggal selesai", `prep_start_date` "Tanggal mulai persiapan", `timezone` "Zona waktu", `target_budget` "Target anggaran", `name` "Nama sesi", `icon` "Ikon", `session_date` "Tanggal sesi", `start_time` "Jam mulai", `end_time` "Jam selesai", `location_name` "Lokasi", `confirm` "Konfirmasi".

---

## 12. Gaya, responsif, aksesibilitas, keamanan

### 12.1 Gaya

Hanya token yang ada (bab 3). Kartu: `--color-surface`, batas `--color-border`, `--radius-lg`, bayangan `--shadow-soft`; judul `--font-display`; isi `--font-body`. Lencana status memakai pola `.status` di `dashboard/index.astro` (pill `--radius-pill`, huruf kapital kecil berjarak). Warna peringatan dan galat belum ada sebagai token; gunakan nilai yang sudah dipakai di kode ( `#8e2f25` untuk bahaya, `#285238` di atas `#dce8df` untuk sukses) dan **jangan** menambah token baru di dokumen ini tanpa persetujuan. Setiap angka status diberi teks, bukan hanya warna.

### 12.2 Responsif

- `< 768px`: kartu Hub satu kolom; metrik portofolio 2×2; hero hitung mundur grid 2×2 saat mode `Hari:Jam:Menit:Detik`; enam widget satu kolom dengan tombol `Buka` besar; switcher *bottom sheet*; dialog penuh layar.
- `768–1100px`: kartu Hub dan widget dua kolom.
- Target sentuh minimal 44×44 px untuk tombol, `•••`, dan pegangan seret.
- Tidak ada gulir horizontal pada lebar 360 px (uji dengan judul 255 karakter dan nama sesi 100 karakter: gunakan `overflow-wrap: anywhere`).

### 12.3 Aksesibilitas

- Dialog: elemen `<dialog>` dengan `showModal()` (menjebak fokus, `Esc` menutup); fokus awal ke kolom pertama atau judul; fokus dikembalikan ke pemicu saat ditutup.
- Formulir: setiap kontrol punya `<label>`; kolom salah `aria-invalid="true"` + `aria-describedby` ke pesan; ringkasan galat `role="alert"`.
- Status asinkron (menyimpan, pembaruan tertunda, hasil PATCH): wilayah `aria-live="polite"`; hitung mundur detik tidak diumumkan (9.2).
- Tab status Hub: `role="tablist"` + `aria-selected`; panah kiri/kanan. Switcher dan `•••`: pola disclosure/menu dengan papan ketik.
- Drag-drop selalu punya padanan papan ketik (tombol naik/turun).
- Kontras dan indikator fokus terlihat; hormati `prefers-reduced-motion` (tanpa animasi detik, tanpa denyut).

### 12.4 Keamanan

- **Tidak ada HTML dari data:** judul, nama, `metadata`, nama sesi, `pending[]` dirender sebagai teks. Skrip klien memakai `textContent`/`createElement`, bukan `innerHTML` untuk string dari API. Satu-satunya `innerHTML` yang dibenarkan adalah mengganti isi dengan **HTML partial dari server kita sendiri** (9.5).
- Path ke halaman dibangun hanya dari `id` yang sudah divalidasi UUID; query lewat `URLSearchParams`.
- CSRF: BFF menolak non-GET tanpa `application/json` (cross-site tidak dapat mengirim JSON tanpa preflight yang tidak pernah diberikan). Token tetap di cookie HttpOnly; skrip tidak pernah melihatnya.
- Tidak ada `localStorage` untuk data proyek. `sessionStorage` boleh untuk draf wizard (opsional, bukan persyaratan).

---

## 13. Pengujian

Gunakan pola yang sudah ada: `node:test` di `frontend/tests/*.test.mjs`, dijalankan `npm test` (`node --experimental-strip-types --test tests/*.test.mjs`), mengimpor `.ts` lewat jalur relatif.

### 13.1 Unit: `tests/planner-format.test.mjs`

Vektor di bawah sudah dijalankan sebagai `node:test` terhadap kode di bab 5 yang diekstrak langsung dari dokumen ini (lulus; `tsc` strict + `noUncheckedIndexedAccess` juga bersih untuk `types.ts`, `format.ts`, `countdown.ts`, dan fixture dengan `satisfies Dashboard`).

| Fungsi | Masukan → keluaran |
|---|---|
| `formatDate` | `2027-03-28` → `Minggu, 28 Maret 2027`; `2027-05-15` → `Sabtu, 15 Mei 2027` |
| `formatDateShort` | `2026-10-15` → `15 Okt 2026` |
| `formatRupiah` | `74150000` → `Rp 74.150.000`; `0` → `Rp 0`; `-1500` → `-Rp 1.500` |
| `formatRupiahShort` | `33550000` → `Rp 33,55 Jt`; `75000000` → `Rp 75 Jt`; `18450000` → `Rp 18,45 Jt`; `26650000` → `Rp 26,65 Jt`; `950000` → `Rp 950 Rb`; `999` → `Rp 999`; `1500000000` → `Rp 1,5 M`; `10000000000000` → `Rp 10 T`; `-13950000` → `-Rp 13,95 Jt` |
| `percent`/`formatPercent` | `(33550000,74150000)`, 1 digit → `45,2%`; `(15,50)` → `30%`; `(2,18)` → `11%`; `(0,0)` → `–`; `(5,0)` → `–`; `(12,10)` → `120%` |
| `todayIn` (now = `2026-10-20T23:47:00Z`) | `Asia/Jakarta` → `2026-10-21`; `UTC` → `2026-10-20`; `Pacific/Auckland` → `2026-10-21`; `America/New_York` → `2026-10-20` |
| `addDays` | `(2026-10-21, 1826)` → `2031-10-21`; `(2027-03-28, 30)` → `2027-04-27`; `(2027-03-28, -7)` → `2027-03-21`; `(2027-01-31, 1)` → `2027-02-01` |
| `daysBetween` | `(2026-10-21, 2027-03-28)` → `158`; `(2027-03-28, 2027-03-21)` → `-7` |
| `isKnownZone` | `Asia/Jakarta` → true; `Asia/Nowhere` → false |
| `ZONE_SHAPE` | `America/Argentina/Buenos_Aires`, `UTC` → cocok; `asia/jakarta`, `WIB`, `Etc/GMT-7` → tidak |
| `parseFieldError` | `bride_name: required` → `{field:'bride_name', reason:'required'}`; `event_date: must not be before today` → field `event_date`; `internal server error` → `null` |
| `charCount` | `"🏛️"` → 2 (`.length` = 3); `"🎉"` → 1 (`.length` = 2); `"é"` → 1 |

### 13.2 Unit: `tests/planner-countdown.test.mjs`

| Kasus | Hasil |
|---|---|
| `zonedMidnight('2027-03-28','Asia/Jakarta')` | `2027-03-27T17:00:00.000Z` |
| `…'UTC'` | `2027-03-28T00:00:00.000Z` |
| `…'Asia/Jayapura'` | `2027-03-27T15:00:00.000Z` |
| `…'Asia/Kolkata'` | `2027-03-27T18:30:00.000Z` |
| `…'Pacific/Auckland'` | `2027-03-27T11:00:00.000Z` |
| `('2027-03-13'/'2027-03-14'/'2027-03-15', 'America/New_York')` (DST mulai 14 Mar) | `…13T05:00Z`, `…14T05:00Z`, `…15T04:00Z` |
| `countdown(target WIB, 2027-03-24T08:15:30Z)` | `{days:3, hours:8, minutes:44, seconds:30}` |
| `countdown(target, target+5000)` (lewat) | semua 0 (tidak negatif) |
| skew | server `12:00:00Z`, perangkat `12:00:30Z` → `skew = −30000`; hasil memakai `Date.now()+skew` |

### 13.3 Tes sumber (pola `tests/auth-wiring.test.mjs`)

`tests/planner-wiring.test.mjs` membaca berkas dan memeriksa:

- Semua halaman `pages/dashboard/planner/**` dan BFF memuat `export const prerender = false`; `widgets.astro` memuat `export const partial = true`.
- BFF mengekspor `GET`, `POST`, `PATCH`, `PUT`, `DELETE`; memuat pemeriksaan `application/json` dan regex path `^[a-z0-9/_-]+$`; tidak mengandung `/v1/auth`.
- Tidak ada komponen/halaman di `components/planner` dan `pages/dashboard/planner` yang memuat daftar jenis acara tertulis: tidak boleh mengandung literal `'khitanan'`, `'aqiqah'`, `'peresmian'`, `'launching'`, `'wisuda'` (hanya `types.ts` yang boleh mendaftar kunci).
- Tidak ada `innerHTML =` selain di berkas yang menukar partial.
- `AppHeader.astro` memuat `/dashboard/planner`.

### 13.4 Uji komponen dengan fixture (manual atau skrip)

Render tiap widget dengan: data fixture penuh; semua angka 0 (keadaan kosong); `null` ("Segera hadir"); `{"error":"timeout"}`; `{"error":"internal"}`; `urgent` berisi `days_left` −3, 0, 1, 5; `overpaid > 0`; `cash_deficit = 0`; `sides` berisi `key: null`; seminar (`labels.guest = 'Peserta'`); `pending` 5 nama dengan `total − completed = 6`; `current_agenda` terisi.

### 13.5 QA manual (checklist, terhadap backend asli)

Jalankan backend dan frontend lokal (bab 14.1), lalu:

1. Akun baru tanpa acara → empty state; "Lihat Proyek Contoh" membuat contoh, tombol nonaktif selama permintaan, klik kedua kali tetap satu proyek.
2. Wizard pernikahan lengkap → redirect ke workspace; judul berspasi ujung tersimpan terpangkas; hero menampilkan `hero_title`, nama, tanggal Indonesia, hitung mundur hari.
3. Wizard dengan judul kosong, tanggal lampau, tanggal selesai > 30 hari, zona `asia/jakarta`, `Asia/Nowhere`, atribut wajib kosong → tiap galat muncul di kolom yang benar dengan teks Indonesia.
4. Klik ganda "Simpan" dan kirim ulang setelah matikan jaringan sebentar → satu proyek saja.
5. Seminar (`labels.guest = Peserta`): kartu Hub, widget Tamu, dan teks kosong memakai "Peserta".
6. Hub dengan 3+ proyek: tab status, kategori, pencarian, URL filter bertahan saat muat ulang; kartu metrik tidak menghitung contoh; "Acara Terdekat" benar.
7. Edit judul/pagu/atribut; hapus satu atribut opsional (kunci terhapus); ubah Hari-H pada acara multi-hari → tanggal selesai ikut bergeser; matikan multi-hari → tanggal selesai terhapus. Sesi yang bertanggal Hari-H ikut pindah (cek di Kelola Sesi).
8. Ubah Hari-H dengan `shiftable_tasks > 0` (butuh modul timeline; sampai ada, **belum dapat diuji manual**, uji lewat respons tiruan).
9. Kelola Sesi: tambah, nama ganda (409), ikon 9 karakter, ubah, urut (seret dan papan ketik), hapus sesi tak terpakai, hapus dengan pindah ke sesi lain, batas 20 sesi.
10. Arsipkan → semua tombol tulis nonaktif; coba tulis dari tab lain → `project_read_only`; hapus proyek arsip tanpa membuka arsip; dialog hapus memerlukan judul persis (coba huruf kecil semua → tombol tetap nonaktif).
11. Dua tab: ubah di satu tab, tab lain mendapat pembaruan ≤ ~35 detik; tab tersembunyi tidak polling (cek tab Network); matikan backend → "Pembaruan tertunda", isi lama tetap; nyalakan lagi → pulih.
12. Hapus proyek di tab A saat tab B terbuka → polling B mendapat 404 → banner.
13. Habiskan sesi (hapus cookie/biarkan kedaluwarsa) lalu klik aksi → diarahkan ke login, bukan layar kosong atau JSON.
14. Project Switcher: pindah antarproyek dari workspace; dari proyek contoh ke proyek lain; "Rencanakan acara lain".
15. Hero: ubah tanggal acara menjadi 3 hari lagi → mode `Hari:Jam:Menit:Detik` berdetak; ubah jam perangkat ±1 jam → angka tetap sama berkat `skew`; ubah hari menjadi hari ini → "Hari-H telah tiba"; kemarin → "Acara telah berlangsung" + Tandai selesai.
16. Mobile 360 px: tidak ada gulir horizontal; switcher bottom sheet; dialog muat; sentuh nyaman.
17. Papan ketik saja: seluruh alur Hub, wizard, edit, sesi dapat dijalankan; fokus terlihat dan kembali ke pemicu setelah dialog ditutup.
18. VoiceOver/NVDA ringkas: lencana, hasil PATCH, dan galat terbaca; detik hitung mundur tidak dibacakan terus-menerus.
19. Tombol "Lihat Proyek Contoh" hilang begitu kartu contoh ada dan muncul lagi setelah contoh dihapus; kartu contoh hari ini menampilkan "Belum ada tugas", "Rp 0 / pagu Rp 75 Jt", "Belum ada tamu" (cocok dengan wireframe "keadaan hari ini"); kartu selesai menaut ke workspace dengan tombol "Lihat Ringkasan".

---

## 14. Urutan pengerjaan dan definisi selesai

### 14.1 Menjalankan lingkungan lokal

```bash
# Backend (Postgres lokal sudah berjalan; migrasi otomatis saat start)
cd backend && make run                          # :8080  (butuh backend/.env; lihat backend/README.md)

# Frontend
cd frontend && npm install
API_BASE_URL=http://localhost:8080 npm run dev  # :4321; daftar akun di /auth/daftar lalu masuk di /auth/masuk
```

Data uji: buka `/dashboard/planner` dan tekan "Lihat Proyek Contoh" (atau `POST /v1/planner/projects/sample` dengan token). Jangan memakai database produksi; gunakan database lokal/sementara.

### 14.2 Tahapan

| # | Tahap | Isi | Selesai bila |
|---|---|---|---|
| M0 | Fondasi | `types.ts`, `format.ts`, `countdown.ts`, `messages.ts`, tes 13.1–13.2; perubahan kecil pada `backend.ts` dan `AppHeader.astro`; `404.astro` | `npm test` hijau |
| M1 | BFF | `[...path].ts` + `client.ts` (`plannerFetch`) | uji manual: `GET /dashboard/api/planner/event-types` membalas JSON; tanpa login terarah ke masuk; non-GET tanpa JSON → 415; path `..` → 404 |
| M2 | Hub | registry, `PortfolioMetrics`, `ProjectCard`, filter, empty state, contoh, menu `•••`, dialog hapus | QA 1, 6, 10 lulus |
| M3 | Wizard | langkah 1–3 (langkah 3 otomatis terlewati), `client_token`, galat per kolom | QA 2–5 lulus |
| M4 | Workspace | `PlannerLayout`, switcher, hero + hitung mundur, enam `WidgetCard` (semua `soon` terhadap backend asli; uji lain dengan fixture) | QA 14–15 lulus; 13.4 lulus |
| M5 | Edit + Sesi | `EditEventDialog`, `SessionManager` | QA 7, 9 lulus |
| M6 | Partial + polling | `widgets.astro`, skrip polling, banner/indikator | QA 11–13 lulus |
| M7 | Kualitas | aksesibilitas, responsif, `astro check`, `npm run build`, `format:check` | QA 16–18 lulus; build bersih |

**Definisi selesai (seluruh Menu 01 FE):** semua tahap lulus; `npm test`, `npm run check`, `npm run build`, `npm run format:check` bersih; tidak ada daftar jenis acara tertulis (tes 13.3); dokumen ini diperbarui bila perilaku berubah, termasuk `docs/frontend/planner/` dan `ROUTES.md` (rute baru), sesuai aturan repo bahwa kode dan dokumen satu paket.

---

## 15. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan backend yang sudah jadi. Dokumen ini sudah mengikuti perilaku backend; butir berikut perlu keputusan bila perilakunya ingin berbeda.

1. **Widget Vendor hanya angka.** Daftar kategori yang belum terkunci, nama kandidat, dan tombol WhatsApp vendor tidak dapat ditampilkan: `VendorSummary` ([BE §5.1](BE.md#51-response)) hanya membawa `categories_total/fixed`, `skipped`, `fixed_vendors`, `pipeline`. Bila ingin lebih, kontrak provider Menu 04 perlu menambah bidang (mis. daftar kategori terbuka dan kontak vendor utama); FE tidak dapat menurunkannya dari angka.
2. **Kartu Hub tanpa jumlah undangan.** `ProjectCardResponse` hanya punya `guest_pax_count`. "20 tamu · 43 pax" membutuhkan bidang `guest_count` (perubahan BE kecil di kueri Hub dan DTO).
3. **`next_session` tanpa ikon.** Ikon dicari dari `dashboard.sessions` berdasarkan nama (unik per proyek); cukup untuk sekarang.
4. **Zona waktu dan tanggal mulai persiapan** dapat diubah oleh API (`PATCH`) tetapi sengaja tidak ada di dialog Edit. Mengubah zona memengaruhi arti "hari ini" untuk seluruh proyek; keputusan produk apakah dibuka (di bawah "Lanjutan").
5. **Tidak ada pratinjau tabrakan `item_key`** saat menghapus sesi: FE hanya memberi catatan umum (9.6) karena datanya milik Menu 03.
6. **Menu 02–06 belum ada**, sehingga tombol **Buka** dan sub-navigasi hanya aktif untuk menu yang sudah `ready`, dan Proyek Contoh tampak hampir kosong. Uji emas PDF di dashboard (angka 74.150.000 dst.) baru dapat dijalankan setelah modul-modulnya terdaftar.
7. **Partial Astro** (9.5) belum dijalankan end-to-end; bila bermasalah, gunakan JSON lewat BFF `GET …/dashboard` (sudah tersedia dan dibatasi 60/menit).
8. **Deskripsi singkat jenis acara** (8.2) belum punya sumber di registry; kamus statis berkunci `key` adalah solusi sementara. Bila ingin disimpan di backend, tambahkan ke `EventTypeSpec`.
9. **Kartu selesai tidak punya tujuan "Arsip & Kenangan".** Halaman itu tidak ada di spec mana pun. Sementara, kartu `completed`/`archived` memakai "Lihat Ringkasan ➜" ke workspace yang sama. Bila produk ingin tampilan arsip/kenangan tersendiri, itu menu baru.
10. **Tanggal sesi Lamaran pada Proyek Contoh berbeda antar dokumen.** [BE §4.5](BE.md#45-proyek-contoh): overlay sampel mengisinya dengan tanggal mulai tugas `wedding.lamaran` (H-169), yaitu Hari-H − 169 hari: 10 Okt 2026 untuk contoh yang dibuat 4 Okt 2026. Wireframe dan fixture bab ini memakai 10 Okt 2026; wireframe Menu 03 menulis 15 Okt 2026, dan salah satunya perlu disamakan (di luar cakupan Menu 01). Sampai Menu 02 dan overlay-nya ada, Lamaran tidak bertanggal dan baris "Sesi terdekat" tidak tampil.
11. **Sumber `admin_docs.label`.** Registry hanya punya `labels.guest` dan `labels.guest_menu`; label widget Berkas diisi provider Menu 02 ([BE §5.1](BE.md#51-response)). Spec Menu 02 belum menyebut nilainya per jenis acara; FE membaca apa adanya dan memakai "Berkas & Perizinan" saat datanya belum ada.

---

## Riwayat perubahan dokumen ini

**2026-10-21:** dirinci untuk pengembang FE lain dan disesuaikan dengan backend yang sudah diimplementasikan; wireframe Hub dan Workspace dibuat ulang (keadaan akhir dan keadaan hari ini). Rincian temuannya ada di [CHANGELOG spesifikasi](../CHANGELOG.md).
