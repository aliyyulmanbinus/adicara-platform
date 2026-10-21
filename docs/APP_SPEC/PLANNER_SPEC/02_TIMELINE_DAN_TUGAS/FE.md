# Menu 02: Timeline & Tugas Persiapan — Frontend Specification

Acuan pembangunan antarmuka **Jadwal & Milestone** (tabel, Kanban, Gantt) dan **Dokumen & Legalitas** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: kontrak data, aturan tampilan, algoritma, alur interaksi, kasus galat, dan pengujian ada di sini. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); fondasi yang dibangun Menu 01 (tipe, `format.ts`, `messages.ts`, `client.ts`, BFF, `PlannerLayout`) **dipakai apa adanya** dan dirujuk ke [01 FE](../01_DASHBOARD/FE.md), bukan diulang.

> **Status (2026-10-21):** frontend Planner belum ada satu baris pun. Backend menu ini **sudah ada** ([BE.md](BE.md), [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-02.md)) dan **kontrak di bab 4 sudah dicocokkan dengan responsnya**: 120 panggilan ke semua 17 endpoint (termasuk 33 jenis galat) menghasilkan 119 respons dan 821 objek JSON yang dibandingkan dengan tipe bab 4.2 oleh parser terhadap blok kode dokumen ini: **nol selisih** kunci, opsionalitas (`?:`), `null`, dan nilai enum ([bab 11.5](#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang)). Fixture ([bab 4.3](#43-fixture)) adalah salinan respons asli itu, bukan buatan tangan, dan sama persis dengan fixture yang dibangun dari tabel BE §5 (tiga sumber setuju: tabel spesifikasi, kode Go, respons HTTP). Menu ini bergantung pada Menu 01 FE (M0–M1 dan `PlannerLayout`, [bab 3](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-fe)).
>
> Kode di bab 4.2, 5, 6, dan 9.4 **sudah dijalankan**: `tsc` strict (dengan `noUncheckedIndexedAccess`, konfigurasi Astro yang dipakai repo) bersih; 19 tes `node:test` ([bab 11](#11-pengujian)) lulus terhadap fixture asli dan menolak 9 dari 9 mutasi kode yang dicoba; ke-17 pembungkus `timelineApi` dijalankan terhadap backend asli dan keputusan `generatorImpact`/`docsImpact` sama dengan keputusan server. Yang bergantung pada Astro atau browser (halaman, partial, dialog, seret-lepas, Gantt tergambar, unduhan) belum diuji dan ditandai *(belum diuji di browser)*.

> Angka pada wireframe berasal dari **Proyek Contoh** (dataset PDF) yang dibuat 4 Okt 2026: Sinta & Arif, 28 Maret 2027, H-175; 50 tugas (44 PDF + 6 Adicara) dengan 15 selesai, 29 berjalan, 6 belum; 15 berkas dengan 9 selesai. Tanggal tugas adalah hasil penskalaan `175/180` ([BE §4.1](BE.md#41-generator-jadwal)), jadi bergeser bila contoh dibuat pada hari lain (fixture sudah digeser ke 4 Okt 2026). Wireframe adalah tata letak, bukan desain visual akhir: warna, jarak, dan tipografi memakai token yang sudah ada (bab 10.1). Emoji pada wireframe untuk **sesi** adalah emoji dari API; ikon **kontrol** memakai Phosphor dan tidak digambar.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat](#2-alur-data-dan-penanganan-galat)
3. [Prasyarat yang harus sudah ada dari Menu 01 FE](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-fe)
4. [Kontrak data: tipe, fixture, contoh respons, katalog galat](#4-kontrak-data-tipe-fixture-contoh-respons-katalog-galat)
5. [Library `lib/planner/timeline.ts`](#5-library-libplannertimelinets)
6. [Pembungkus panggilan, BFF, dan unduhan](#6-pembungkus-panggilan-bff-dan-unduhan)
7. [Tab 1: Jadwal & Milestone](#7-tab-1-jadwal--milestone)
8. [Tab 2: Dokumen & Legalitas](#8-tab-2-dokumen--legalitas)
9. [Pesan galat dan teks](#9-pesan-galat-dan-teks)
10. [Gaya, responsif, aksesibilitas, keamanan](#10-gaya-responsif-aksesibilitas-keamanan)
11. [Pengujian](#11-pengujian)
12. [Urutan pengerjaan dan definisi selesai](#12-urutan-pengerjaan-dan-definisi-selesai)
13. [Celah dan keputusan yang menunggu pemilik produk](#13-celah-dan-keputusan-yang-menunggu-pemilik-produk)
14. [Riwayat perubahan dokumen ini](#14-riwayat-perubahan-dokumen-ini)

---

## 1. Route, file, dan perubahan pada kode yang sudah ada

### 1.1 File baru

| Halaman / berkas | Route | File | Fungsi |
|---|---|---|---|
| Timeline & Tugas | `/dashboard/planner/[id]/timeline` | `src/pages/dashboard/planner/[id]/timeline/index.astro` | dua tab lewat `?tab=jadwal` (default) dan `?tab=dokumen`; nilai lain jatuh ke `jadwal`. Tab dipilih server-side; bagian tab lain tidak dimuat. Parameter lain di bab 2.3 |
| Partial | `/dashboard/planner/[id]/timeline/live` | `src/pages/dashboard/planner/[id]/timeline/live.astro` (`export const partial = true`) | HTML area yang diperbarui setelah aksi tulis (bab 2.4); menerima `?tab=` yang sama dengan halaman. Satu sumber render: memakai komponen yang sama dengan `index.astro` |
| Komponen | – | `src/components/planner/timeline/*.astro` | lihat 1.2 dan 1.4 |
| Library | – | `src/lib/planner/timeline.ts` (murni, bab 5), `timelineApi.ts` (pembungkus panggilan, bab 6), `timelineClient.ts` (skrip klien halaman, bab 7.6) | |
| Tes + fixture | – | `tests/planner-timeline.test.mjs`, `tests/fixtures/planner-{tasks,docs,sessions,project,errors}.json` | bab 4.3 dan 11 |

Tidak ada file BFF baru: semua panggilan lewat `/dashboard/api/planner/[...path].ts` dari Menu 01 (bab 6). Halaman memakai `PlannerLayout` (bilah sekunder berisi Project Switcher dan sub-navigasi), bukan `AppLayout` langsung.

### 1.2 Komponen

Props lengkap (TypeScript) di bab 1.4. Kolom "Klien" menyebut perilaku yang dipasang skrip halaman; alurnya di bab 7.6 dan 8.3.

| Komponen | Props utama | Dipakai di | Klien |
|---|---|---|---|
| `TimelineHeader` | `summary`, `project`, `canGenerate`, `exportHref` | progres, lima penghitung, tombol Generate / Tambah / Ekspor | membuka modal; penghitung dihitung ulang dengan `summarize` saat aksi optimistik |
| `ScopeChips` | `sessions`, `value`, `counts` | chip sesi | menyaring baris tanpa muat ulang (`hidden`) + `?scope=` |
| `ViewSwitcher` | `value` | Tab 1 | mengganti tampilan + `sessionStorage` |
| `TaskTable` | `groups`, `numbers`, `sessions`, `readOnly` | tampilan Tabel | centang, dropdown status, tanggal inline, menu ⋮, Naik/Turun, seret |
| `KanbanBoard` | `columns`, `sessions`, `readOnly` | tampilan Kanban | seret-lepas antar kolom, "Pindahkan ke…" |
| `GanttMatrix` | `gantt`, `tasks`, `numbers`, `today` | tampilan Gantt | hanya baca; klik baris membuka `TaskModal` |
| `TaskModal` | `sessions`, `categories`, `defaultPhase`, `vendorsEnabled` | tambah/ubah tugas | `<dialog>`, validasi kenyamanan, pemetaan galat ke kolom |
| `GeneratorModal` | `type`, `project`, `taskCount` | Generate Jadwal Otomatis | dampak mode (`generatorImpact`), konfirmasi, hasil |
| `ShiftDialog` | `project` | geser jadwal manual | satu aksi tidak idempoten |
| `DocChecklist` | `list`, `subjects`, `readOnly`, `canApplyTemplate` | Tab 2 | centang, Kirim ke anggaran, menu Terapkan Template |
| `DocModal` | `subjects`, `defaultSubject` | tambah/ubah berkas | `<dialog>` |
| `GuideCallout` | `guide` | panduan prosedur | – |
| `ConfirmDialog` | `title`, `message`, `confirmLabel`, `danger` | hapus tugas/berkas | satu `<dialog>` dipakai bersama |
| `GeneratorResult` | `result`, `kind` | hasil generator dan Terapkan Template | menampilkan `created/skipped/deleted` dan `warnings` |

### 1.3 Perubahan pada file yang sudah ada

| File | Perubahan | Mengapa |
|---|---|---|
| `src/layouts/PlannerLayout.astro` | `MENUS`: `timeline.ready = true` | satu bendera; menu lain tidak berubah ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) |
| `src/lib/planner/types.ts` | tambah blok bab 4.2; `EventTypeInfo` menjadi `extends EventTypeAdminInfo` | kontrak menu ini |
| `src/lib/planner/messages.ts` | tambah blok bab 9.4 (kamus status, teks peringatan, alasan per kolom, galat berkonteks) | teks galat |
| `src/lib/api/backend.ts` dan BFF | `backendFetchRaw` dan aliran berkas untuk `…/tasks/export-spreadsheet` | satu-satunya respons non-JSON; sudah diwajibkan [kontrak bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens) butir 4 tetapi tidak ada di 01 FE bab 6 |
| `ROUTES.md` | baris halaman `/dashboard/planner/[id]/timeline` dan partial `…/timeline/live` | file itu minta diperbarui bersama setiap halaman baru |
| `docs/frontend/planner/` | dokumen bertanggal dan entri `CHANGELOG.md` | aturan repo: kode dan dokumen satu paket |
| (tidak diubah) | `AppLayout.astro`, `middleware.ts`, `tokens.css`, `global.css` | |

`astro check` (bagian dari `npm run build`) harus lolos; TypeScript `strict` dan `noUncheckedIndexedAccess` aktif (akses `array[i]` bertipe `T | undefined`).

### 1.4 Kontrak props

Disalin ke `interface Props` tiap komponen (atau diimpor dari satu `props.ts`); **sudah dikompilasi** `tsc` strict bersama bab 4.2 dan 5. `PhaseGroup`, `Gantt` dari bab 5; `numbers` = `Object.fromEntries(taskNumbers(semuaTugas))` (objek biasa karena props SSR tidak membawa `Map`).

```ts
import type { Gantt, PhaseGroup } from './timeline.ts';
import type {
  AdminGuide, AdminSubject, DocList, EventTypeInfo, GenerateResult, SessionBrief, Task, TaskStatus, TaskSummary,
} from './types.ts';

export type TimelineTab = 'jadwal' | 'dokumen';
export type ViewMode = 'tabel' | 'kanban' | 'gantt';

/** Konteks proyek yang dibaca halaman dari GET projects/:id dan diteruskan ke komponen (bukan dibaca ulang). */
export interface TimelineProject {
  id: string; timezone: string; eventDate: string; archived: boolean; shiftableTasks: number;
}

export interface TimelineHeaderProps { summary: TaskSummary; project: TimelineProject; canGenerate: boolean; exportHref: string }
export interface ScopeChipsProps { sessions: SessionBrief[]; value: string; counts: Record<string, number> }
export interface ViewSwitcherProps { value: ViewMode }
export interface TaskTableProps { groups: PhaseGroup[]; numbers: Record<string, number>; sessions: SessionBrief[]; readOnly: boolean }
export interface KanbanBoardProps { columns: Record<TaskStatus, Task[]>; sessions: SessionBrief[]; readOnly: boolean }
export interface GanttMatrixProps { gantt: Gantt; tasks: Task[]; numbers: Record<string, number>; today: string }
export interface TaskModalProps { sessions: SessionBrief[]; categories: string[]; defaultPhase: { order: number; name: string }; vendorsEnabled: boolean }
export interface GeneratorModalProps { type: EventTypeInfo; project: TimelineProject; taskCount: number }
export interface ShiftDialogProps { project: TimelineProject }
export interface DocChecklistProps { list: DocList; subjects: AdminSubject[]; readOnly: boolean; canApplyTemplate: boolean }
export interface DocModalProps { subjects: AdminSubject[]; defaultSubject: string }
export interface GuideCalloutProps { guide: AdminGuide }
export interface ConfirmDialogProps { title: string; message: string; confirmLabel: string; danger?: boolean }
export interface GeneratorResultProps { result: GenerateResult; kind: 'tasks' | 'docs' }
```

`TimelineProject` dibentuk satu kali di frontmatter halaman: `archived = detail.project.status === 'archived'`, `shiftableTasks = detail.shiftable_tasks`, `timezone`, `eventDate = detail.project.event_date`. `readOnly` pada komponen = `project.archived`.
---

## 2. Alur data dan penanganan galat

```text
SSR (baca)    : halaman → backendFetch('/v1/planner/…', { accessToken }) → HTML
Interaksi     : <script> klien → timelineApi(plannerFetch, id) → BFF /dashboard/api/planner/… → Go API
Pembaruan     : sesudah aksi tulis → fetch('/dashboard/planner/:id/timeline/live?tab=…') → HTML partial → ganti [data-timeline-live]
Unduhan       : <a href="/dashboard/api/planner/projects/:id/tasks/export-spreadsheet"> → BFF (aliran berkas)
Registry      : lib/planner/eventTypes.ts → GET /v1/planner/event-types (cache memori proses 1 jam)
```

`backendFetch` mengembalikan seluruh payload di `result.data`, jadi isi respons ada di `result.data.data`.

### 2.1 Pemuatan per tab

| Tab | Panggilan (paralel dengan `Promise.all`) | Catatan |
|---|---|---|
| `jadwal` | `GET projects/:id/tasks` + `GET projects/:id/sessions` + `GET projects/:id` + registry | `GET projects/:id` (`ProjectDetail`) memberi `project.status` (arsip), `project.timezone` (untuk "hari ini" pada Gantt), `project.event_date`, dan `shiftable_tasks`. `GET sessions` mengembalikan `SessionWithUsage[]` **termasuk sesi sistem `all`** (nama server "Umum / Semua Sesi", `sort_order 0`); chip memakai teks tetap "Umum" untuk `all`, bukan `name` (bab 7.2). Daftar vendor untuk dropdown **dimuat saat modal Tambah/Ubah dibuka** (`GET projects/:id/vendors`, [04 BE §3.1](../04_VENDOR_DAN_BELANJA/BE.md#31-vendor)); bila `404`/galat atau Menu 04 belum ada, kolom Vendor disembunyikan. Tidak ada `GET` satu tugas (rute itu `404`): tugas tunggal selalu datang dari respons tulis atau dari daftar |
| `dokumen` | `GET projects/:id/admin-docs` + `GET projects/:id` + registry | peran dan panduan dari registry (`admin_subjects`, `admin_guide`); respons `admin-docs` tidak membawa salinannya |

Validasi `Astro.params.id` seperti 01 FE bab 2.1 (UUID huruf kecil, selain itu `Astro.rewrite('/404')` tanpa memanggil backend).

### 2.2 Hasil backend → tindakan halaman

| Hasil | Tindakan |
|---|---|
| `2xx` | render |
| `401` | `Astro.redirect('/auth/masuk')` |
| `404 not_found` pada proyek | `Astro.rewrite('/404')` (tidak membedakan "bukan milik Anda" dan "tidak ada") |
| `5xx` atau `BackendUnavailableError` | kerangka halaman + banner "Timeline sedang tidak tersedia" + tombol "Muat ulang"; **bukan 404** |
| `GET tasks` gagal tetapi proyek berhasil | tab tetap tampil dengan banner galat lokal pada daftar; Tab dokumen tidak ikut gagal |
| Registry gagal | pakai cache lama; bila tidak ada, filter subjek dan panduan disembunyikan dan dropdown peran diganti isian teks tidak aktif (tombol simpan dokumen nonaktif) |

Aturan umum: `export const prerender = false`; teks galat ke pengguna selalu dari `lib/planner/messages.ts`, tidak pernah `error.message` mentah (pesan server berbahasa Inggris). Proyek `archived`: semua kontrol ubah dinonaktifkan dengan keterangan "Acara diarsipkan" (`GET` dan ekspor tetap berjalan; terbukti: tulis → `409 project_read_only`, baca dan ekspor → `200`); tetap tangani `409 project_read_only` bila status berubah di tab lain.

Kerangka frontmatter `timeline/index.astro` *(sketsa, belum dijalankan)*:

```ts
export const prerender = false;
const id = Astro.params.id ?? '';
if (!isUuid(id)) return Astro.rewrite('/404');
const tab = Astro.url.searchParams.get('tab') === 'dokumen' ? 'dokumen' : 'jadwal';
const accessToken = Astro.locals.accessToken;                     // diisi middleware (token sudah segar)
const [detail, registry] = await Promise.all([
  backendFetch<Envelope<ProjectDetail>>(`/v1/planner/projects/${id}`, { accessToken }),
  loadRegistry(Astro),
]);
// … toPageError(detail) menerapkan tabel 2.2; selanjutnya per tab:
const [list, sessions] = await Promise.all([
  backendFetch<Envelope<TaskList>>(`/v1/planner/projects/${id}/tasks`, { accessToken }),
  backendFetch<Envelope<SessionWithUsage[]>>(`/v1/planner/projects/${id}/sessions`, { accessToken }),
]);
const known = new Set(sessions.data.data.map((s) => s.scope_key));
const counts = countByScope(list.data.data.tasks, known);            // angka pada chip
const numbers = Object.fromEntries(taskNumbers(list.data.data.tasks));
```

### 2.3 State di URL dan di peramban

| Penyimpan | Kunci | Nilai | Ditulis oleh | Dibaca oleh |
|---|---|---|---|---|
| URL | `tab` | `jadwal` (default) · `dokumen` | tautan tab (navigasi penuh) | SSR: bagian yang dimuat |
| URL | `scope` | `scope_key` sesi atau `all` (Umum); tak ada = Semua | klien, `history.replaceState` | SSR menandai chip aktif dan memberi `hidden` pada baris lain; nilai yang tidak dikenal diperlakukan sebagai Semua |
| URL | `subject` | kunci `admin_subjects`; tak ada = Semua | klien, `history.replaceState` | SSR (Tab dokumen), aturan sama |
| `sessionStorage` | `planner.timeline.view` | `tabel` · `kanban` · `gantt` | `ViewSwitcher` | skrip klien saat dimuat; default `tabel`; nilai lain diabaikan. Dibungkus try/catch seperti `AuthForm` |

Tidak ada `localStorage`; tidak ada data proyek di penyimpan peramban.

### 2.4 Satu sumber render dan pembaruan sesudah aksi

Mengikuti pola partial [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial): **tidak ada logika render ganda di klien**.

1. SSR merender area `[data-timeline-live]` (penghitung, ketiga tampilan, atau daftar berkas) dari komponen yang sama dengan `live.astro`. **Ketiga tampilan dirender sekaligus**; yang tidak aktif diberi `hidden`. Setiap baris, kartu, dan balok membawa atribut data ([tabel 7.6.1](#761-atribut-data-yang-dibaca-skrip-klien)), sehingga memilih chip sesi, memilih tampilan, dan menyaring subjek hanyalah mengubah atribut `hidden` di klien, tanpa memanggil server.
2. Aksi tulis berhasil → klien memanggil `GET /dashboard/planner/{id}/timeline/live?tab={tab}` (`Accept: text/html`), mengganti isi `[data-timeline-live]`, lalu menerapkan ulang chip, tampilan, dan filter aktif dari URL/`sessionStorage`. Fokus dipertahankan lewat `data-focus-key` (elemen fokus sebelum penggantian difokuskan kembali bila kuncinya masih ada).
3. Aksi yang bisa berbalik (centang, dropdown status, Kanban) lebih dulu mengubah tampilan secara **optimistik** (bab 7.6) agar terasa seketika; hasil partial menjadi kebenaran akhir (penanda `overdue`, nomor, penghitung, fase).
4. Partial gagal (`5xx`, jaringan): jangan kosongkan area; tampilkan banner kecil "Daftar belum diperbarui" + tombol "Muat ulang" (`location.reload()`), `aria-live="polite"`. `401` atau pengalihan ke login: `location.assign('/auth/masuk')`.
5. Tab lain yang mengubah data tidak diketahui sampai aksi berikutnya atau muat ulang (tidak ada polling di menu ini; dashboard Menu 01 yang polling). Galat yang menandakan data usang (`not_found`, `invalid_scope`, `invalid_input: ids`) selalu ditutup dengan memuat ulang area lewat partial.

*(Belum diuji di browser, sama seperti 01: `export const partial = true` didukung kode Astro 7.3.5 tetapi belum dijalankan dari halaman nyata. Bila bermasalah, alternatif yang sah adalah `location.reload()` sesudah tiap aksi tulis: lebih kasar, tetapi satu-satunya logika render tetap SSR.)*

---

## 3. Prasyarat yang harus sudah ada dari Menu 01 FE

**Belum ada satu pun di repo** (frontend Planner belum dibangun). Menu ini mengimpor, tidak membangun ulang:

| Kebutuhan | Berasal dari |
|---|---|
| `types.ts` (`Notice`, `SessionBrief`, `EventTypeInfo`, `Project`, `ApplyResult`) | 01 FE bab 4.2 |
| `format.ts` (`formatDate`, `formatDateShort`, `formatRupiah`, `percent`, `formatPercent`, `barWidth`, `todayIn`, `addDays`, `daysBetween`, `charCount`, `parseFieldError`) | 01 FE bab 5.1 |
| `messages.ts` (`plannerErrorMessage`, `fieldErrorMessage`, `TEMPLATE_LABELS`, `WARNING_LABELS`) | 01 FE bab 5.4 dan 11 |
| `client.ts` (`plannerFetch`: tangani sesi habis, `204`, backend mati) dan `server.ts` (`loadRegistry`, `isUuid`) | 01 FE bab 5.5–5.6 |
| BFF `[...path].ts` dan perubahan `backend.ts` (`PUT`, `details`) | 01 FE bab 6 dan 1.3 |
| `PlannerLayout.astro` (Project Switcher, sub-navigasi `MENUS`) dan `404.astro` | 01 FE bab 10 dan 1.1 |
| Pola `<dialog>`, `sessionStorage` dibungkus try/catch, `textContent` untuk string API | 01 FE bab 12 |

Karena itu urutan kerja ([bab 12](#12-urutan-pengerjaan-dan-definisi-selesai)) dimulai dari M0, M1, dan `PlannerLayout` di Menu 01 FE.

---

## 4. Kontrak data: tipe, fixture, contoh respons, katalog galat

### 4.1 Aturan JSON yang dikirim backend

Selain aturan umum [01 FE bab 4.1](../01_DASHBOARD/FE.md#41-aturan-json-yang-dikirim-backend) (amplop `{ "data": … }`, tanggal `"YYYY-MM-DD"` tanpa zona, uang bilangan bulat). Semua butir di bawah **diamati pada backend asli**, bukan disalin dari BE.md.

**Bentuk respons**

- **Kunci dihilangkan bila kosong** (`omitempty`): `vendor_id`, `vendor_name`, `vendor_wa`, `template_key` pada tugas, dan `template_key`, `budget_item_id` pada berkas. Tipe memakai `?:`; jangan memeriksa `=== null` (tidak pernah `null`).
- **Daftar selalu array** (`tasks`, `docs`, `warnings`), tidak pernah `null`, termasuk saat kosong.
- **Teks bisa kosong tetapi tidak pernah hilang**: `pic`, `notes`, `phase_name`, `quantity_detail`. `category` tidak pernah kosong (kosong disimpan sebagai `Umum`).
- **Nilai status memakai kata PDF** (`belum` | `proses` | `selesai`; berkas hanya `belum` | `selesai`); kunci JSON berbahasa Inggris. Karena `proses` ditampilkan "Sedang Berjalan", jangan menampilkan nilainya mentah (`TASK_STATUS_LABELS`).
- **`duration_days` dan `overdue` adalah hasil server.** Durasi = `max(selesai − mulai, 1)` (konvensi PDF) sedangkan balok Gantt inklusif; keduanya sengaja berbeda (bab 5). `overdue` dihitung di zona waktu proyek; klien tidak membandingkan tanggal.
- **`sort_order` bukan nomor tampilan.** Ia dijamin hanya untuk urutan: bisa berlubang (hapus tugas) atau tidak dimulai dari 1 (pada proyek yang punya tugas kustom, `replace_*` menyisipkan template baru **setelah** tugas kustom; hasil asli: daftar dimulai dari `19`). Kolom "No" memakai `taskNumbers` (bab 5).
- `summary` pada `GET tasks` dan `GET admin-docs` adalah **seluruh proyek**, bukan hasil saringan (terbukti: `?status=belum` tetap `total: 50`).
- **`message` galat dan `warnings[].message` berbahasa Inggris** dan hanya untuk log. Tampilkan teks dari `messages.ts` berdasarkan `code` (+ kolom/alasan untuk `invalid_*`).
- Tidak ada `GET` satu tugas atau satu berkas; rute itu `404`.

**Permintaan**

- **Bilangan bulat murni.** `nominal_cost: 1.5` atau `"5000"` dijawab `400 invalid_json` (tanpa nama kolom). Ubah isian teks menjadi bilangan bulat (hanya digit) sebelum mengirim; jangan kirim string. Tipe lain yang salah (mis. `title: 5`, UUID bukan UUID) juga `invalid_json`.
- **Kunci yang tidak dikenal diabaikan tanpa galat** (typo `titel` lolos diam-diam): andalkan tipe TypeScript, bukan server.
- **Teks dipangkas** (`trim`) oleh server; panjang dihitung dalam *code point* (255 emoji diterima, 256 ditolak): pakai `charCount` (01 FE), bukan `.length`. Tab dan karakter kontrol ditolak; hanya `notes` boleh memuat `\n` (`\r\n` menjadi `\n`).
- **Bawaan bila kosong, berlaku di POST dan PATCH:** `sub_event_scope ""` → `all`; `category ""` → `Umum`; `subject_role` tidak dikirim → `bersama` (pernikahan) atau `umum` (jenis lain). Khusus POST: `phase_order` tidak dikirim atau `0` → fase tugas dengan `sort_order` terbesar (proyek kosong: fase 1 tanpa nama); `phase_order > 0` tanpa `phase_name` → nama disalin dari tugas lain di fase itu.
- **PATCH hanya mengubah kunci yang dikirim**, dan ada dua jebakan: (1) `vendor_id: null` **tidak** melepas vendor; pakai `clear_vendor: true` (bersama `vendor_id` → `400 invalid_input: clear_vendor`); (2) mengubah `phase_order` **tidak** menyalin `phase_name` (berbeda dari POST): tugas yang pindah ke fase 4 tetap membawa nama fase lamanya. **Kirim keduanya** (`phase_name` dari `phaseNameOf(tasks, order)` atau yang diketik pengguna).
- **Batas** (diukur persis: nilai tepat diterima, lebih satu ditolak): `LIMITS` di bab 5. Jendela tanggal tugas: `Hari-H − 1095` hari s/d akhir acara `+ 365` hari (`event_end_date` bila ada, selain itu Hari-H; di luar itu `invalid_date: … must be within the project schedule window`; batas persis diterima, satu hari lebih ditolak).
- **Idempotensi dan percobaan ulang.** Aman diulang: semua `GET`, `PATCH` (hasil sama), `generate-defaults` mode `append` (kedua kali `created: 0`), `sync-budget` (kedua kali tidak membuat pos kedua; klik ganda sudah diserialkan server), `reorder` dengan daftar yang sama. **Tidak** aman diulang: `POST tasks` dan `POST admin-docs` (membuat baris lagi), `shift` (menggeser lagi), mode `replace_*`. `DELETE` kedua kali menjawab `404`: perlakukan sebagai berhasil (baris sudah hilang) dan segarkan.

### 4.2 Tambahan pada `src/lib/planner/types.ts`

Hanya sintaks yang dapat dihapus (tanpa `enum`), seperti 01 FE. `Notice`, `SessionBrief`, `ApiError` berasal dari blok 01. **Ubah satu baris di blok 01:** `export interface EventTypeInfo {` menjadi `export interface EventTypeInfo extends EventTypeAdminInfo {`; tanpanya tab Dokumen tidak dapat dirender (bab 2.2). `GET /event-types` kini mengirim kedua bidang untuk **semua sembilan jenis**: pernikahan lima subjek (`cpp`, `cpw`, `ortu_cpp`, `ortu_cpw`, `bersama`), delapan jenis lain tiga subjek (`panitia`, `lembaga`, `umum`, juga `other`), dan `admin_guide.steps` selalu tepat 4 langkah.

```ts
// ---------- Menu 02: tugas, dokumen, registry (ditambahkan ke akhir types.ts) ----------
export type TaskStatus = 'belum' | 'proses' | 'selesai';
export type DocStatus = 'belum' | 'selesai';
export type Origin = 'pdf' | 'adicara' | 'custom';

export interface Task {
  id: string; title: string;
  sub_event_scope: string;          // scope_key sesi; 'all' = Umum
  category: string;                 // '' tidak pernah dikirim: kosong disimpan sebagai 'Umum'
  phase_order: number; phase_name: string;  // phase_name bisa '' (fase tanpa nama)
  start_date: string; end_date: string;   // YYYY-MM-DD; end_date inklusif pada Gantt
  duration_days: number;            // hasil server = max(end − start, 1); jangan dihitung klien
  status: TaskStatus;
  pic: string;
  vendor_id?: string; vendor_name?: string; vendor_wa?: string; // dihilangkan bila tanpa vendor
  sort_order: number;               // bisa berlubang atau tidak mulai dari 1: jangan dipakai sebagai nomor tampilan
  notes: string;                    // boleh berisi \n
  template_key?: string; origin: Origin;
  overdue: boolean;                 // hitungan server (zona proyek); klien tidak menghitung ulang
}
export interface TaskSummary { total: number; completed: number; in_progress: number; not_started: number; overdue: number }
export interface TaskList { tasks: Task[]; summary: TaskSummary }   // summary = seluruh proyek, bukan hasil saringan

export interface AdminDoc {
  id: string; document_name: string;
  subject_role: string;             // kunci dari registry admin_subjects
  quantity_detail: string; nominal_cost: number; // rupiah utuh
  status: DocStatus; sort_order: number; notes: string;
  template_key?: string; origin: Origin;
  budget_item_id?: string;          // ada = sudah tersinkron ke anggaran (Menu 03)
}
export interface DocList { docs: AdminDoc[]; summary: { total: number; completed: number } }

// Tambahan registry. Ubah deklarasi 01 menjadi `export interface EventTypeInfo extends EventTypeAdminInfo { … }`.
export interface AdminSubject { key: string; label: string }
export interface AdminGuide { title: string; steps: string[] }   // tepat 4 langkah
export interface EventTypeAdminInfo { admin_subjects: AdminSubject[]; admin_guide: AdminGuide }

export type GenerateMode = 'append' | 'replace_template' | 'replace_all';
export interface GenerateBody { preset: string; include_adicara_tasks: boolean; mode: GenerateMode; confirm?: boolean }
export interface GenerateResult { created: number; skipped: number; deleted: number; warnings: Notice[] }
export interface ApplyDocsBody { mode: 'append' | 'replace_template'; confirm?: boolean }
export interface ShiftBody { days: number }
export interface ShiftResult { shifted: number; warnings: Notice[] }
export interface SyncBudgetBody { sub_event_scope?: string }
export interface SyncBudgetResult { budget_item_id: string; created: boolean; linked: boolean; warnings: Notice[] }

export interface TaskBody {          // POST. Bawaan server bila kosong: sub_event_scope 'all', category 'Umum', phase_order 0 = ikut tugas terakhir
  title: string; sub_event_scope?: string; category?: string; phase_order?: number; phase_name?: string;
  start_date: string; end_date: string; pic?: string; vendor_id?: string; notes?: string;
}
export interface TaskPatch extends Partial<TaskBody> { status?: TaskStatus; clear_vendor?: boolean } // hanya kunci yang dikirim berubah
export interface DocBody { document_name: string; subject_role?: string; quantity_detail?: string; nominal_cost?: number; notes?: string }
export type DocPatch = Partial<DocBody>;
```

`GenerateBody.preset` wajib dan `mode` boleh dikosongkan di server (bawaan `append`), tetapi UI **selalu mengirim semuanya**. `TaskBody.start_date`/`end_date` wajib (tanggal kosong → `400 invalid_date`).

**Hasil pencocokan dengan respons asli** (parser terhadap blok kode di atas dan blok 01, objek per tipe dari 120 panggilan):

| Tipe | Objek | Tipe | Objek | Tipe | Objek |
|---|:-:|---|:-:|---|:-:|
| `Task` | 416 | `TaskList` / `TaskSummary` | 15 / 15 | `GenerateResult` | 12 |
| `AdminDoc` | 126 | `DocList` | 10 | `ShiftResult` | 3 |
| `SyncBudgetResult` | 4 | `Notice` | 6 | `ApiError` | 45 |
| `EventTypeInfo` (+ `AdminSubject` 29, `AdminGuide` 9) | 9 | `SessionWithUsage` | 4 | `ProjectDetail`, `Dashboard` (widget `timeline`, `admin_docs`) | 1, 1 |

Hasil: **0 selisih** (kunci berlebih, kunci wajib hilang, `null` pada tipe non-null, enum di luar daftar). Pemeriksa sudah diuji dengan lima mutasi tipe (bab 11.5) dan melaporkan semuanya. Cara mengulang: [bab 11.5](#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang).

### 4.3 Fixture

Lima berkas di [`fixtures/`](fixtures/) (folder ini) disalin ke `frontend/tests/fixtures/`. Semuanya **respons asli** dari backend yang berjalan untuk Proyek Contoh, hanya diubah dua hal: (1) setiap tanggal digeser mundur sehingga contoh tampak dibuat **2026-10-04** (Hari-H 2027-03-28, `Avail = 175`, sesuai wireframe), (2) `id` diganti id sintetis yang tetap supaya tes deterministik.

| Berkas | Isi | Bentuk |
|---|---|---|
| `planner-tasks.json` | 50 tugas: `sort_order` 1–50; 1–15 `selesai`, 16–44 `proses`, 45–50 `belum`; `overdue` semua `false`; ringkasan `{50, 15, 29, 6, 0}`; fase 7 / 8 / 20 / 15 | `TaskList` |
| `planner-docs.json` | 15 berkas; `selesai` untuk nomor 1, 2, 3, 6, 7, 8, 10, 14, 15; ringkasan `{15, 9}`; tidak ada yang tertaut anggaran | `DocList` |
| `planner-sessions.json` | sesi sistem `all` + Lamaran (15 Okt 2026) + Akad Nikah + Resepsi (28 Mar 2027), dengan `usage.tasks` 29 / 7 / 4 / 10 | `SessionWithUsage[]` |
| `planner-project.json` | `GET projects/:id` Proyek Contoh dengan `shiftable_tasks: 35` (50 tugas dikurangi 15 selesai) | `ProjectDetail` |
| `planner-errors.json` | 49 galat yang pernah diterima (`status`, `code`, `message`, `source`) | `{ status; code; message; source }[]` |

Contoh baris (dari `planner-tasks.json`):

```json
{ "id": "00000000-0000-4000-8000-000000000016", "title": "Finalisasi Seserahan", "sub_event_scope": "all", "category": "Logistik",
  "phase_order": 3, "phase_name": "Detail Acara & Perlengkapan", "start_date": "2026-11-02", "end_date": "2026-11-03", "duration_days": 1,
  "status": "proses", "pic": "", "sort_order": 16, "notes": "", "template_key": "wedding.finalisasi_seserahan", "origin": "pdf", "overdue": false }
```

Tiga sumber diperiksa sama persis untuk 50 tugas (judul, sesi, kategori, fase, tanggal, durasi, status, kunci, asal): tabel [BE §5](BE.md#5-master-data-template) → fixture yang dibangun dari tabel itu lewat `Plan` → respons HTTP backend. Nilai widget dashboard yang dihasilkan sama dengan fixture 01 FE (`timeline` `50/15/29/0`, lima berkas `pending` yang sama).

**Variasi yang wajib diuji** (turunkan dari fixture di dalam tes, jangan membuat berkas baru): daftar kosong (`tasks: []`); semua `overdue: true`; satu tugas dengan `vendor_id`/`vendor_name`/`vendor_wa` (valid dan tidak valid); `sub_event_scope` yang tidak ada di `sessions` (data usang: tampilkan sebagai "Umum"); judul 255 karakter dan nama berkas 255 karakter; satu berkas dibuat tertaut (`budget_item_id` terisi); jenis seminar (tanpa kotak Adicara; 18 tugas `origin: "adicara"` dengan `template_key` berawalan `general.` atau `seminar.`; subjek `panitia`/`lembaga`/`umum`); jenis `other` (tanpa template berkas); proyek `archived`.

### 4.4 Katalog endpoint

Semua di bawah `projects/:id/` (`:id` = UUID proyek) lewat BFF; semua penulisan juga mungkin menjawab `401`, `404 not_found` (proyek), `409 project_read_only`, dan `5xx`/jaringan, yang tidak diulang di kolom Galat. Nama fungsi = `timelineApi` (bab 6). Deskripsi lengkap: [BE §3](BE.md#3-endpoint).

| Aksi UI | Fungsi | Metode dan path | Body | Sukses | Galat khusus |
|---|---|---|---|---|---|
| Muat tugas | `listTasks` | `GET tasks` | – | `200 TaskList` | `invalid_input` (`status`/`scope`/`phase` bila memakai parameter; UI tidak memakainya) |
| Tambah tugas | `createTask` | `POST tasks` | `TaskBody` | `201 Task` | `invalid_input`, `invalid_date`, `invalid_scope`, `invalid_json`, `409 task_limit_reached` |
| Ubah tugas | `updateTask` | `PATCH tasks/:tid` | `TaskPatch` | `200 Task` | seperti POST + `invalid_input: clear_vendor`; `404 not_found` (tugas) |
| Centang / Kanban / dropdown | `setTaskStatus` | `PATCH tasks/:tid/status` | `{ status }` | `200 Task` | `invalid_input: status`; `404` (tugas) |
| Hapus tugas | `deleteTask` | `DELETE tasks/:tid` | – | `204` | `404` (tugas) |
| Urut ulang | `reorderTasks` | `POST tasks/reorder` | `{ ids }` (**semua** id) | `200 TaskList` (urutan baru, `sort_order` 1…n) | `invalid_input: ids` |
| Geser manual | `shiftTasks` | `POST tasks/shift` | `{ days }` | `200 ShiftResult` | `invalid_input: days` (nol, > 1826), `invalid_date: days` (tugas keluar jendela) |
| Generate jadwal | `generate` | `POST tasks/generate-defaults` | `GenerateBody` | `200 GenerateResult` | `invalid_input` (`preset`, `mode`, `confirm`), `invalid_template`, `invalid_date: event_date` (Hari-H lampau), `409 task_limit_reached` |
| Ekspor | `exportHref` | `GET tasks/export-spreadsheet` | – | `200` berkas `.xlsx` | `429 rate_limited` (10/menit) |
| Muat berkas | `listDocs` | `GET admin-docs` | – | `200 DocList` | – |
| Tambah berkas | `createDoc` | `POST admin-docs` | `DocBody` | `201 AdminDoc` | `invalid_input`, `invalid_json`, `409 admin_doc_limit_reached` |
| Ubah berkas | `updateDoc` | `PATCH admin-docs/:did` | `DocPatch` | `200 AdminDoc` | seperti POST; `404` (berkas) |
| Centang berkas | `setDocStatus` | `PATCH admin-docs/:did/status` | `{ status }` | `200 AdminDoc` | `invalid_input: status`; `404` (berkas) |
| Hapus berkas | `deleteDoc` | `DELETE admin-docs/:did` | – | `204` | `404` (berkas) |
| Urut ulang berkas | `reorderDocs` | `POST admin-docs/reorder` | `{ ids }` | `200 DocList` | `invalid_input: ids` |
| Terapkan template | `applyDocTemplate` | `POST admin-docs/apply-template` | `ApplyDocsBody` | `200 GenerateResult` | `invalid_input: mode`/`confirm`, `invalid_template` (`other`), `409 admin_doc_limit_reached` |
| Kirim ke anggaran | `syncBudget` | `POST admin-docs/:did/sync-budget` | `SyncBudgetBody` (boleh `{}`) | `201` pos baru · `200` selainnya; `SyncBudgetResult` | `invalid_input: nominal_cost` (biaya 0), `invalid_scope`; `404` (berkas) |

Tiga hasil `sync-budget` yang semuanya sukses: `201 {created: true}` (pos baru), `200 {created: false, linked: true}` (pos yang ada ditautkan; baru mungkin setelah Menu 03 ada), `200 {created: false, linked: false}` (sudah tertaut sebelumnya). Pakai `ok` dan `budget_item_id`, jangan membedakan 200 dari 201 untuk menampilkan badge.

### 4.5 Contoh permintaan dan respons asli

Dicuplik dari respons nyata (tanggal digeser ke baseline 4 Okt 2026, id dipendekkan seperlunya); respons aslinya sudah dilewatkan parser tipe. Satu pengecualian ditandai di tempatnya.

**Tambah tugas dengan isian minimal** (`POST tasks`, `201`). Perhatikan fase bawaan (fase tugas terakhir), `category` dan `sub_event_scope` bawaan, `origin: "custom"`, dan tidak ada `template_key`:

```json
// permintaan
{ "title": "Tugas minimal", "start_date": "2026-10-15", "end_date": "2026-10-15" }
// respons
{ "data": { "id": "5434…", "title": "Tugas minimal", "sub_event_scope": "all", "category": "Umum", "phase_order": 4,
  "phase_name": "Finalisasi & Menuju Hari-H", "start_date": "2026-10-15", "end_date": "2026-10-15", "duration_days": 1,
  "status": "belum", "pic": "", "sort_order": 51, "notes": "", "origin": "custom", "overdue": false } }
```

**Tugas dengan vendor** (kunci vendor muncul hanya bila ada): `…, "vendor_id": "…", "vendor_name": "Sasana Kriya Ballroom", "vendor_wa": "6281234567890", …`. *Bentuk dari [BE §2](BE.md#2-domain--dto-go) dan tes Go; belum teramati pada backend yang berjalan karena Menu 04 belum ada, jadi tipe `vendor_*` baru terbukti oleh tes Go.*

**Ubah status** (`PATCH tasks/:tid/status` `{"status":"selesai"}`) mengembalikan `Task` utuh dengan `overdue` yang sudah dihitung ulang (`false` bila selesai). **Ubah fase**: kirim keduanya, `{"phase_order": 3, "phase_name": "Detail Acara & Perlengkapan"}`.

**Geser manual** (`POST tasks/shift` `{"days": -40}`):

```json
{ "data": { "shifted": 35, "warnings": [ { "code": "tasks_overdue_after_shift", "message": "some shifted tasks are now past their due date", "count": 2 } ] } }
```

**Generate** (`POST tasks/generate-defaults`). Tanpa kemajuan yang hilang: `append` → `{"created": 0, "skipped": 50, "deleted": 0, "warnings": []}`. Waktu persiapan lebih pendek dari standar (contoh dibuat 175 hari sebelum Hari-H, standar 180): `replace_template` dengan `confirm: true` →

```json
{ "data": { "created": 22, "skipped": 0, "deleted": 50, "warnings": [
  { "code": "lead_time_compressed", "message": "less time than the standard lead time: the schedule was compressed", "count": 22 } ] } }
```

`count` = tugas yang **dibuat** panggilan itu. Waktu < 3 hari mengganti peringatan itu dengan `lead_time_too_short` (setiap tugas diberi rentang hari ini sampai Hari-H; Hari-H hari ini → semuanya satu hari, `duration_days: 1`); keduanya tidak muncul bersamaan, dan tanpa tugas baru tidak ada peringatan tanggal. Sesi template yang sudah dihapus: `scope_fallback`.

**Terapkan template berkas** (`replace_template`, `confirm: true`) saat berkas KUA sudah tertaut ke anggaran:

```json
{ "data": { "created": 15, "skipped": 0, "deleted": 15, "warnings": [
  { "code": "budget_link_lost", "message": "budget lines that pointed at the replaced documents keep their amounts but lost the link", "count": 1 } ] } }
```

**Kirim ke anggaran** (`POST admin-docs/:did/sync-budget` `{}`): pertama `201 {"data": {"budget_item_id": "2d75…", "created": true, "linked": false, "warnings": []}}`; kedua `200 {"data": {"budget_item_id": "2d75…", "created": false, "linked": false, "warnings": []}}` (id sama). Daftar berkas berikutnya memuat `"budget_item_id": "2d75…"` pada baris itu.

**Berkas baru tanpa subjek** (jenis pernikahan): `POST admin-docs` `{"document_name": "Dokumen minimal"}` → `201`, `subject_role: "bersama"`, `quantity_detail: ""`, `nominal_cost: 0`, `status: "belum"`, `sort_order: 16`, `origin: "custom"`.

**Daftar yang disaring** tetap membawa ringkasan seluruh proyek: `GET tasks?status=belum` → `{"tasks": [ …6 tugas… ], "summary": {"total": 50, "completed": 15, "in_progress": 29, "not_started": 6, "overdue": 0}}`.

**Unduhan** (`GET tasks/export-spreadsheet`): `200`, `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="jadwal-contoh-pernikahan-sinta-arif-2026-10-21.xlsx"`, `Content-Length` ada, **tanpa** `Cache-Control`. Pembatasan: respons ke-11 dalam semenit → `429 {"error":{"code":"rate_limited","message":"too many requests, try again later"}}`, tanpa header `Retry-After`.

### 4.6 Katalog galat

Bentuk: `{ "error": { "code": "…", "message": "…" } }` (+ `details` pada beberapa kode Menu 01). Untuk `invalid_input`/`invalid_date`/`invalid_metadata`, `message` = `<kolom>: <alasan>` dan dipetakan ke teks per kolom (bab 9.2); kode lain dipetakan per kode (bab 9.1). Daftar penuh yang pernah diterima ada di `fixtures/planner-errors.json`; tes 11.1 memastikan **setiap** alasan di sana punya teks Indonesia.

| Status | Kode | Contoh `message` | Penyebab dan penanganan UI |
|:-:|---|---|---|
| 400 | `invalid_input` | `title: must not be empty` · `phase_order: must be between 1 and 20` · `ids: must list every id exactly once` · `confirm: required` | pesan di bawah kolom (bab 9.2); `confirm: required` bukan galat tampil, melainkan perintah menampilkan kotak konfirmasi (bab 7.5); `ids` → muat ulang area |
| 400 | `invalid_date` | `end_date: must not be before start_date` · `start_date: must not be after end_date` · `…: must be within the project schedule window` · `days: must keep every shifted task within the project schedule window` · `event_date: must not be before today` | di bawah kolom; `event_date` (generator pada Hari-H lampau) → pesan khusus bab 7.5 |
| 400 | `invalid_scope` | `sub_event_scope: unknown session` | sesi dihapus di tab lain → muat ulang area (teks per kode) |
| 400 | `invalid_template` | `preset: not available for this event type` · `admin_docs: not available for this event type` | registry usang → muat ulang halaman (teks per kode) |
| 400 | `invalid_json` | `invalid JSON body` | kesalahan klien (tipe salah, body kosong tanpa `{}`): teks umum 01 + log |
| 401 | `unauthorized` | `missing bearer token` | `plannerFetch` mengalihkan ke login |
| 404 | `not_found` | `project not found` · `task not found` · `document not found` | dibedakan oleh **aksi** (`where`), bukan oleh pesan; tugas/berkas hilang → "…tidak ditemukan. Muat ulang daftar."; setelah `DELETE` kedua dianggap sukses |
| 409 | `project_read_only` | `the project is archived; restore it to make changes` | proyek diarsipkan di tab lain → nonaktifkan semua kontrol ubah, teks 01 |
| 409 | `task_limit_reached` / `admin_doc_limit_reached` | `at most 500 tasks per project` / `at most 200 documents per project` | banner di modal; tidak ada baris yang tertulis |
| 409 | `conflict` | `the project was changed concurrently, try again` | jarang (tabrakan kunci); teks 01, aman dicoba ulang |
| 429 | `rate_limited` | `too many requests, try again later` | hanya ekspor di menu ini: "Terlalu banyak ekspor…" |
| 5xx / jaringan | `internal`, `service_unavailable` | – | teks umum 01; jangan mengosongkan layar |

---

## 5. Library `lib/planner/timeline.ts`

Modul **murni** (aturan sama dengan 01 FE bab 5): tanpa alias `@/`, tanpa `enum`, dan mengimpor tetangganya dengan akhiran `.ts` (`allowImportingTsExtensions` aktif di konfigurasi Astro) sehingga bisa diimpor `node --experimental-strip-types`. **Sudah dijalankan**: `tsc` strict bersih, 19 tes lulus ([bab 11.1](#111-unit-testsplanner-timelinetestmjs)), dan setiap mutasi yang dicoba pada fungsi keputusan (`generatorImpact`, `docsImpact`, `taskNumbers`, `moveByOne`, `scopeOf`, `withStatus`, `whatsappUrl`) membuat tes gagal. Kode di bawah adalah isi berkas apa adanya.

```ts
import { addDays, daysBetween } from './format.ts';
import type { AdminDoc, GenerateMode, Task, TaskStatus, TaskSummary } from './types.ts';

/** Batas yang dijaga server. Diukur pada batas persis: nilai tepat diterima, lebih satu ditolak (karakter = code point). */
export const LIMITS = {
  title: 255, documentName: 255, category: 60, phaseName: 100, pic: 100, quantityDetail: 100, notes: 2000,
  phaseOrder: { min: 1, max: 20 },
  maxCost: 10_000_000_000_000,
  maxShiftDays: 1826,
  maxTasks: 500, maxDocs: 200,
} as const;

/** Nilai chip "Semua" (bukan scope_key; scope_key hanya [a-z0-9_]). */
export const SCOPE_ANY = '*';

/** Sesi tugas untuk tampilan: sesi yang tidak ada di `known` (data usang) dianggap 'all' (Umum). Tanpa `known`, apa adanya. */
export const scopeOf = (t: Task, known?: ReadonlySet<string>): string =>
  known === undefined || t.sub_event_scope === 'all' || known.has(t.sub_event_scope) ? t.sub_event_scope : 'all';

/** Saring menurut chip sesi. SCOPE_ANY = semua; 'all' = hanya tugas Umum. Tidak memanggil server. */
export const filterByScope = (tasks: Task[], scope: string, known?: ReadonlySet<string>): Task[] =>
  scope === SCOPE_ANY ? tasks : tasks.filter((t) => scopeOf(t, known) === scope);

/** Angka pada chip: jumlah per sesi dari daftar yang sedang dimuat (bukan `usage.tasks` server), SCOPE_ANY = total. */
export function countByScope(tasks: Task[], known?: ReadonlySet<string>): Record<string, number> {
  const out: Record<string, number> = { [SCOPE_ANY]: tasks.length };
  for (const t of tasks) {
    const k = scopeOf(t, known);
    out[k] = (out[k] ?? 0) + 1;
  }
  return out;
}

/** Kolom "No": urutan 1-based pada daftar LENGKAP menurut sort_order. sort_order bisa berlubang (hapus, tugas kustom), jadi jangan dipakai mentah. */
export function taskNumbers(all: Task[]): Map<string, number> {
  const sorted = [...all].sort((a, b) => a.sort_order - b.sort_order);
  return new Map(sorted.map((t, i) => [t.id, i + 1]));
}

export interface PhaseGroup { order: number; name: string; tasks: Task[] }

/** Kelompok per phase_order (naik); isi urut sort_order; nama fase dari tugas pertama kelompok yang bernama. */
export function groupByPhase(tasks: Task[]): PhaseGroup[] {
  const byOrder = new Map<number, Task[]>();
  for (const t of tasks) {
    const list = byOrder.get(t.phase_order);
    if (list) list.push(t); else byOrder.set(t.phase_order, [t]);
  }
  return [...byOrder.entries()]
    .sort(([a], [b]) => a - b)
    .map(([order, list]) => {
      const sorted = [...list].sort((a, b) => a.sort_order - b.sort_order);
      return { order, name: sorted.find((t) => t.phase_name !== '')?.phase_name ?? '', tasks: sorted };
    });
}

/** Nama fase yang sudah dipakai pada nomor fase itu ('' bila belum ada). PATCH tidak menyalin nama saat phase_order berubah: kirim keduanya. */
export const phaseNameOf = (tasks: Task[], order: number): string =>
  groupByPhase(tasks).find((g) => g.order === order)?.name ?? '';

/** Fase bawaan untuk tugas baru = fase tugas dengan sort_order terbesar (sama dengan bawaan backend); proyek kosong: fase 1 tanpa nama. */
export function defaultPhase(tasks: Task[]): { order: number; name: string } {
  let last: Task | undefined;
  for (const t of tasks) if (last === undefined || t.sort_order > last.sort_order) last = t;
  return last ? { order: last.phase_order, name: last.phase_name } : { order: 1, name: '' };
}

/** Isi dropdown kategori: kategori yang sudah dipakai proyek; "Umum" selalu ada dan pertama, sisanya urut abjad. */
export function categoriesOf(tasks: Task[]): string[] {
  const rest = [...new Set(tasks.map((t) => t.category))].filter((c) => c !== '' && c !== 'Umum');
  return ['Umum', ...rest.sort((a, b) => a.localeCompare(b, 'id'))];
}

export function kanbanColumns(tasks: Task[]): Record<TaskStatus, Task[]> {
  const cols: Record<TaskStatus, Task[]> = { belum: [], proses: [], selesai: [] };
  for (const t of [...tasks].sort((a, b) => a.sort_order - b.sort_order)) cols[t.status].push(t);
  return cols;
}

/** Penghitung header dari daftar lokal setelah perubahan; `overdue` memakai penanda server pada tiap baris (klien tidak membandingkan tanggal). */
export const summarize = (tasks: Task[]): TaskSummary => ({
  total: tasks.length,
  completed: tasks.filter((t) => t.status === 'selesai').length,
  in_progress: tasks.filter((t) => t.status === 'proses').length,
  not_started: tasks.filter((t) => t.status === 'belum').length,
  overdue: tasks.filter((t) => t.overdue).length,
});

/** Checkbox cepat: selesai ↔ belum (status "proses" hanya lewat dropdown atau Kanban). */
export const toggledStatus = (s: TaskStatus): TaskStatus => (s === 'selesai' ? 'belum' : 'selesai');

/** Ganti satu baris dengan Task dari respons server (posisi tetap). */
export const replaceTask = (tasks: Task[], next: Task): Task[] => tasks.map((t) => (t.id === next.id ? next : t));
export const removeTask = (tasks: Task[], id: string): Task[] => tasks.filter((t) => t.id !== id);

/**
 * Pembaruan optimistik status: hanya status (dan `overdue` menjadi false saat selesai). Penanda terlambat yang benar
 * datang dari Task respons PATCH, yang menggantikan baris ini lewat replaceTask; bila gagal, pulihkan salinan lama.
 */
export const withStatus = (tasks: Task[], id: string, status: TaskStatus): Task[] =>
  tasks.map((t) => (t.id === id ? { ...t, status, overdue: status === 'selesai' ? false : t.overdue } : t));

/**
 * Seret-lepas pada tampilan yang tersaring/dikelompokkan tetap harus mengirim DAFTAR ID LENGKAP ke
 * POST /tasks/reorder. Pindahkan `movedId` tepat sebelum `beforeId` (null = ke akhir) pada daftar lengkap.
 */
export function moveBefore(allIds: string[], movedId: string, beforeId: string | null): string[] {
  const rest = allIds.filter((id) => id !== movedId);
  if (!allIds.includes(movedId)) return allIds;
  if (beforeId === null) return [...rest, movedId];
  const at = rest.indexOf(beforeId);
  if (at < 0) return allIds;
  rest.splice(at, 0, movedId);
  return rest;
}

/**
 * Tombol Naik/Turun (padanan papan ketik seret-lepas): tukar dengan tetangga di kelompok yang sama (`groupIds` urut
 * sort_order). Mengembalikan daftar LENGKAP untuk POST /tasks/reorder, atau null bila sudah di ujung / bukan anggota.
 */
export function moveByOne(allIds: string[], groupIds: string[], id: string, dir: -1 | 1): string[] | null {
  const at = groupIds.indexOf(id);
  const neighbor = groupIds[at + dir];
  if (at < 0 || neighbor === undefined) return null;
  return dir < 0 ? moveBefore(allIds, id, neighbor) : moveBefore(allIds, neighbor, id);
}

export interface Impact {
  removed: number;     // baris yang akan dihapus
  atRisk: number;      // di antaranya yang punya kemajuan (status bukan 'belum', atau tertaut vendor)
  needsConfirm: boolean; // aturan server: kirim confirm: true, selain itu 400 invalid_input: confirm
}

/**
 * Dampak mode Generate pada daftar lokal, mengikuti aturan server (BE §4.1): replace_template minta konfirmasi bila ada
 * tugas ber-template_key dengan kemajuan; replace_all minta konfirmasi bila proyek punya tugas SATU PUN (walau semuanya "belum").
 */
export function generatorImpact(tasks: Task[], mode: GenerateMode): Impact {
  if (mode === 'append') return { removed: 0, atRisk: 0, needsConfirm: false };
  const pool = mode === 'replace_all' ? tasks : tasks.filter((t) => t.template_key !== undefined);
  const atRisk = pool.filter((t) => t.status !== 'belum' || t.vendor_id !== undefined).length;
  return { removed: pool.length, atRisk, needsConfirm: mode === 'replace_all' ? pool.length > 0 : atRisk > 0 };
}

export interface DocImpact { removed: number; finished: number; linked: number; needsConfirm: boolean }

/** Dampak "Ganti template" pada berkas (BE §4.7): konfirmasi bila ada berkas template yang selesai atau tertaut ke anggaran. */
export function docsImpact(docs: AdminDoc[], mode: 'append' | 'replace_template'): DocImpact {
  if (mode === 'append') return { removed: 0, finished: 0, linked: 0, needsConfirm: false };
  const pool = docs.filter((d) => d.template_key !== undefined);
  const finished = pool.filter((d) => d.status === 'selesai').length;
  const linked = pool.filter((d) => d.budget_item_id !== undefined).length;
  return { removed: pool.length, finished, linked, needsConfirm: finished + linked > 0 };
}

/** Tautan WhatsApp hanya dari nomor ternormalisasi backend (62…); selain itu null (tombol tidak digambar). */
export const whatsappUrl = (wa?: string): string | null =>
  wa !== undefined && /^62\d{6,15}$/.test(wa) ? `https://wa.me/${wa}` : null;

export interface GanttBar { id: string; col: number; span: number; status: TaskStatus }
export interface Gantt {
  rangeStart: string;               // Senin
  days: number;                     // kelipatan 7
  weeks: { number: number; start: string }[];
  bars: GanttBar[];                 // col = hari sejak rangeStart (0-based), span = hari INKLUSIF
  todayCol: number | null;          // null bila hari ini di luar rentang
}

/** Senin pada minggu yang memuat `date` (minggu dimulai Senin seperti PDF Hal. 3). */
export function mondayOf(date: string): string {
  const dow = new Date(`${date}T00:00:00Z`).getUTCDay(); // 0 = Minggu
  return addDays(date, -((dow + 6) % 7));
}

export function buildGantt(tasks: Task[], today: string): Gantt | null {
  if (tasks.length === 0) return null;
  const first = tasks.reduce((m, t) => (t.start_date < m ? t.start_date : m), tasks[0]!.start_date);
  const last = tasks.reduce((m, t) => (t.end_date > m ? t.end_date : m), tasks[0]!.end_date);
  const rangeStart = mondayOf(first);
  const days = Math.ceil((daysBetween(rangeStart, last) + 1) / 7) * 7;
  const todayCol = daysBetween(rangeStart, today);
  return {
    rangeStart,
    days,
    weeks: Array.from({ length: days / 7 }, (_, i) => ({ number: i + 1, start: addDays(rangeStart, i * 7) })),
    bars: tasks.map((t) => ({
      id: t.id,
      col: daysBetween(rangeStart, t.start_date),
      span: daysBetween(t.start_date, t.end_date) + 1,
      status: t.status,
    })),
    todayCol: todayCol >= 0 && todayCol < days ? todayCol : null,
  };
}

export const isSynced = (d: AdminDoc): boolean => d.budget_item_id !== undefined;
/** Tombol "Kirim ke anggaran" tampil bila ada biaya dan belum tertaut. */
export const canSyncBudget = (d: AdminDoc): boolean => d.nominal_cost > 0 && !isSynced(d);
/** SCOPE_ANY = semua subjek; selain itu cocokkan subject_role persis. */
export const filterDocs = (docs: AdminDoc[], subject: string): AdminDoc[] =>
  subject === SCOPE_ANY ? docs : docs.filter((d) => d.subject_role === subject);
/** Ganti satu berkas dengan AdminDoc dari respons (posisi tetap). */
export const replaceDoc = (docs: AdminDoc[], next: AdminDoc): AdminDoc[] => docs.map((d) => (d.id === next.id ? next : d));
```

Aturan pemakaian:

- **Saring di klien, tanpa render ulang.** Chip sesi dan penyaring subjek tidak memanggil server (`GET tasks?scope=` ada di API tetapi tidak dipakai UI). SSR memakai `filterByScope`/`countByScope` untuk keadaan awal dari `?scope=` dan angka chip; sesudahnya klien hanya mengubah `hidden` (bab 2.4). `SCOPE_ANY` = "Semua", `'all'` = "Umum". Basis data menjamin setiap tugas menunjuk sesi yang ada dan menghapus sesi memindahkan tugasnya ke `all`; `sub_event_scope` tak dikenal hanya muncul pada data usang (sesi dihapus di tab lain) dan `scopeOf(task, known)` memperlakukannya sebagai `all`, jadi **selalu berikan `known`** (kunci dari `GET sessions`) pada `filterByScope`/`countByScope`.
- **Angka chip dari daftar, bukan dari `usage.tasks`.** `countByScope` dihitung dari daftar yang sama dengan yang digambar, sehingga angka dan baris tidak pernah berbeda (hasil pada fixture: Semua 50, Umum 29, Lamaran 7, Akad 4, Resepsi 10).
- **Kolom "No" = `taskNumbers`.** Posisi 1-based pada daftar *lengkap* menurut `sort_order`, tetap sama saat daftar disaring atau dikelompokkan; jangan menampilkan `sort_order` mentah (bab 4.1).
- **Reorder selalu daftar lengkap.** Seret-lepas di tampilan yang tersaring atau dikelompokkan menghitung `moveBefore(semuaId, idYangDipindah, idTujuan)` dengan `semuaId` = id **seluruh** tugas menurut `sort_order` (bukan subset yang tampak), lalu `POST tasks/reorder`; sukses → ganti daftar lokal dengan `TaskList` respons. Tombol Naik/Turun memakai `moveByOne(semuaId, idKelompok, id, ±1)` (null = sudah di ujung: nonaktifkan tombolnya). Seret-lepas pada tampilan Tabel **dibatasi di dalam satu kelompok fase**: urutan tidak mengubah `phase_order`, jadi melepas di fase lain akan tampak tidak berpindah; pindah fase lewat modal Ubah.
- **Fase.** `defaultPhase(tasks)` mengisi modal Tambah (sama dengan bawaan backend). Saat pengguna mengubah nomor fase pada modal Ubah, kirim `phase_order` **dan** `phase_name` (`phaseNameOf(tasks, nomorBaru)`, atau nama yang diketik): PATCH tidak menyalin nama (bab 4.1).
- **Optimistik.** `withStatus` hanya mengubah status (dan `overdue` menjadi `false` bila selesai); `Task` dari respons PATCH, lewat `replaceTask`, menggantikan baris itu dan membawa `overdue` yang benar. Simpan salinan baris sebelum mengubah untuk pemulihan bila gagal.
- **Dampak Generate/Terapkan Template.** `generatorImpact` dan `docsImpact` mereplikasi aturan server (bab 4.4, [BE §4.1](BE.md#41-generator-jadwal)): `replace_template` meminta `confirm` bila ada tugas ber-`template_key` yang sudah berjalan atau bervendor; **`replace_all` meminta `confirm` bila proyek punya tugas satu pun, walau semuanya "belum"**; berkas: bila ada berkas template yang selesai atau tertaut anggaran. Terbukti terhadap server pada keadaan nyata (tes integrasi: keputusan lokal = status 400/200 server). Tetap tangani `400 invalid_input: confirm` (daftar lokal usang).
- **Hari ini** = `todayIn(project.timezone)` saat render; hanya dipakai untuk garis "Hari Ini" pada Gantt dan label relatif. Penanda terlambat selalu dari `task.overdue`.
- **Tanggal** ditampilkan dengan `formatDateShort` (01 FE); jangan `new Date('2026-10-04')` tanpa `timeZone: 'UTC'`.
- **Durasi vs balok.** `duration_days` dari server ditampilkan di kolom Durasi; lebar balok = `span` (inklusif). Tugas 8 → 10 Okt: durasi "2 hari", balok 3 kolom. Itu konvensi PDF, bukan galat.
- **Kategori.** `categoriesOf` mengisi dropdown ("Umum" selalu ada dan pertama; sisanya kategori yang sudah dipakai proyek, urut abjad Indonesia) dan "Lainnya…" membuka isian bebas (maks `LIMITS.category`).
- **WhatsApp.** `whatsappUrl(task.vendor_wa)`; `null` → jangan menggambar tombol. Tautan dibuka dengan `target="_blank" rel="noopener noreferrer"`.
- **Batas.** `LIMITS` dipakai untuk `maxlength`/`min`/`max` pada isian dan penghitung karakter (`charCount`); server tetap yang menentukan.

---

## 6. Pembungkus panggilan, BFF, dan unduhan

Tidak ada file BFF baru. Pemanggilan klien memakai `plannerFetch('projects/:id/…')` (01 FE bab 5.5) dengan path yang lolos aturan BFF `^[a-z0-9/_-]+$` (`generate-defaults`, `export-spreadsheet`, `apply-template`, `sync-budget`, `admin-docs` semuanya lolos). Semua non-GET mengirim `Content-Type: application/json` dengan body (`'{}'` bila kosong; body kosong tanpa `{}` dijawab `400 invalid_json`), `DELETE` tanpa body ditangani `plannerFetch` ([01 FE bab 3](../01_DASHBOARD/FE.md#3-prasyarat-yang-sudah-ada-di-repo), jebakan `checkOrigin`).

### 6.1 `timelineApi.ts`

Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan sebagai argumen sehingga modul bisa diuji tanpa peramban; `PlannerResult` adalah bentuk hasil `plannerFetch` yang dijelaskan 01 FE bab 5.5 (tidak melempar). Setiap id di path diperiksa UUID huruf kecil dan melempar `Error` bila tidak (kesalahan pemrogram, bukan permintaan ke server). **Sudah dijalankan**: tes unit memeriksa path, metode, dan body ke-17 fungsi, dan semuanya dipanggil terhadap backend asli (status sukses, hasil sesuai tipe, galat sesuai katalog).

```ts
import type {
  AdminDoc, ApiError, ApplyDocsBody, DocBody, DocList, DocPatch, DocStatus, GenerateBody, GenerateResult,
  ShiftResult, SyncBudgetBody, SyncBudgetResult, Task, TaskBody, TaskList, TaskPatch, TaskStatus,
} from './types.ts';

/** Bentuk hasil `plannerFetch` (01 FE bab 5.5): tidak pernah melempar; sesi habis sudah dialihkan di dalamnya. */
export interface PlannerResult<T> { ok: boolean; status: number; data: T | null; error: ApiError | null }
export type PlannerFetch = <T>(
  path: string,
  init?: { method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'; body?: unknown; query?: Record<string, string> },
) => Promise<PlannerResult<T>>;

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
/** Path dibangun hanya dari UUID huruf kecil; selain itu kesalahan pemrogram (lempar), bukan permintaan ke server. */
const uuid = (v: string): string => {
  if (!UUID.test(v)) throw new Error(`bukan UUID: ${v}`);
  return v;
};

/** Satu fungsi per aksi di UI. Semua non-GET mengirim body (plannerFetch mengisi '{}' bila undefined). */
export function timelineApi(call: PlannerFetch, projectId: string) {
  const p = `projects/${uuid(projectId)}`;
  const t = (id: string) => `${p}/tasks/${uuid(id)}`;
  const d = (id: string) => `${p}/admin-docs/${uuid(id)}`;
  return {
    // ----- tugas -----
    listTasks: () => call<TaskList>(`${p}/tasks`),
    createTask: (body: TaskBody) => call<Task>(`${p}/tasks`, { method: 'POST', body }),
    updateTask: (id: string, body: TaskPatch) => call<Task>(t(id), { method: 'PATCH', body }),
    setTaskStatus: (id: string, status: TaskStatus) => call<Task>(`${t(id)}/status`, { method: 'PATCH', body: { status } }),
    deleteTask: (id: string) => call<null>(t(id), { method: 'DELETE' }),
    /** `ids` = SEMUA id tugas proyek menurut urutan baru (moveBefore / moveByOne). */
    reorderTasks: (ids: string[]) => call<TaskList>(`${p}/tasks/reorder`, { method: 'POST', body: { ids } }),
    shiftTasks: (days: number) => call<ShiftResult>(`${p}/tasks/shift`, { method: 'POST', body: { days } }),
    generate: (body: GenerateBody) => call<GenerateResult>(`${p}/tasks/generate-defaults`, { method: 'POST', body }),
    // ----- berkas -----
    listDocs: () => call<DocList>(`${p}/admin-docs`),
    createDoc: (body: DocBody) => call<AdminDoc>(`${p}/admin-docs`, { method: 'POST', body }),
    updateDoc: (id: string, body: DocPatch) => call<AdminDoc>(d(id), { method: 'PATCH', body }),
    setDocStatus: (id: string, status: DocStatus) => call<AdminDoc>(`${d(id)}/status`, { method: 'PATCH', body: { status } }),
    deleteDoc: (id: string) => call<null>(d(id), { method: 'DELETE' }),
    reorderDocs: (ids: string[]) => call<DocList>(`${p}/admin-docs/reorder`, { method: 'POST', body: { ids } }),
    applyDocTemplate: (body: ApplyDocsBody) => call<GenerateResult>(`${p}/admin-docs/apply-template`, { method: 'POST', body }),
    /** 201 = pos baru, 200 = sudah tertaut atau pos yang ada ditautkan; keduanya sukses (pakai `ok`). */
    syncBudget: (id: string, body: SyncBudgetBody = {}) => call<SyncBudgetResult>(`${d(id)}/sync-budget`, { method: 'POST', body }),
  };
}

/** Tujuan tautan unduhan .xlsx (GET biasa lewat BFF; bukan plannerFetch karena responsnya berkas). */
export const exportHref = (projectId: string): string => `/dashboard/api/planner/projects/${uuid(projectId)}/tasks/export-spreadsheet`;
```

Pemakaian di skrip klien:

```ts
const api = timelineApi(plannerFetch, projectId);
const res = await api.setTaskStatus(taskId, toggledStatus(task.status));
if (!res.ok) { rollback(); toast(timelineErrorMessage(res.status, res.error?.code, 'task') ?? plannerErrorMessage(res.status, res.error?.code)); return; }
tasks = replaceTask(tasks, res.data!);          // res.data bertipe Task | null; null hanya untuk 204
```

### 6.2 Unduhan `.xlsx`

Tautan biasa `<a href={exportHref(project.id)} download>`; `GET` tidak butuh `Content-Type`, cookie sesi ikut. BFF meneruskan berkas sebagai aliran beserta `Content-Disposition` dari backend (nama berkas dibuat server dari slug ASCII judul proyek: `jadwal-<slug>-<tanggal>.xlsx`, tanggal = hari ini di zona proyek). Respons asli membawa `Content-Length` tetapi **tidak** `Cache-Control`, jadi BFF harus menambahkan `Cache-Control: no-store` (berkas berisi data pribadi). Bila sesi habis, middleware membalas pengalihan ke halaman login (bukan berkas): tautan dibuka di tab yang sama sehingga pengguna sampai di login; itu diterima. `429 rate_limited` (10/menit, tanpa `Retry-After`) tampil sebagai halaman JSON dari BFF; untuk teks yang ramah, ganti tautan dengan tombol yang memanggil `fetch` lalu menyimpan Blob *(belum diuji di browser; pilih saat membangun, bab 13)*. Isi berkas tidak diuji di Excel/Numbers/LibreOffice ([BE bab 8, butir 12](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 7. Tab 1: Jadwal & Milestone

### 7.1 Wireframe

**Keadaan akhir** (Proyek Contoh; tampilan Tabel). Baris 4 Okt 2026 = hari pembuatan contoh. Sampai halaman ini dibangun, sub-navigasi "Timeline" nonaktif (`ready: false`), lihat [01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout). Semua baris di bawah sama dengan `fixtures/planner-tasks.json`:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ Adicara                                                            Undangan  Planner  Keluar │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ]  Dashboard [Timeline] Keuangan Vendor Tamu Rundown  │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ [★ Jadwal & Milestone]   [Dokumen & Legalitas]                                               │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Progres [███░░░░░░░░░░░░░░░░░] 30% (15/50)   Selesai 15 · Berjalan 29 · Belum 6 · Terlambat 0│
│ [⚡ Generate Jadwal]   [+ Tambah Tugas]   [📥 Ekspor .xlsx]                                  │
│ Sesi: [Semua] [Umum] [💍 Lamaran] [📜 Akad Nikah] [🏛️ Resepsi]                                │
│ Tampilan: [📋 Tabel] [🗂️ Kanban] [📊 Gantt]                                                   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ▼ FASE 1 — Persiapan Awal & Lamaran (7)                                                       │
│ ☑   1 Pertemuan keluarga            [Lamaran]  Keluarga   4 Okt – 5 Okt     1 hari  Selesai ⋮│
│ ☑   3 Tetapkan anggaran lamaran     [Lamaran]  Keuangan   8 Okt – 10 Okt    2 hari  Selesai ⋮│
│ ▼ FASE 2 — Administrasi & Booking Vendor Inti (8)                                             │
│ ☑   9 Siapkan pendaftaran pernikahan [Akad]    Administrasi 19 Okt – 20 Okt 1 hari  Selesai ⋮│
│ ▼ FASE 3 — Detail Acara & Perlengkapan (20)                                                   │
│ ☐  16 Finalisasi Seserahan          [Umum]     Logistik   2 Nov – 3 Nov     1 hari  Berjalan ⋮│
│ ☐  25 Visit Venue Pernikahan  🤝 Sasana Kriya [Resepsi] Vendor 20 Nov – 21 Nov 1 hari Berjalan ⋮│
│ ▼ FASE 4 — Finalisasi & Menuju Hari-H (15)                                                    │
│ ☐  45 Finalisasi daftar tamu & kuota pax ✦ [Resepsi] Tamu  12 Feb – 13 Feb 2027 1 hari Belum ⋮│
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 ✦ = Tambahan Adicara (bukan dari PDF).  "No" = urutan dalam daftar lengkap, bukan nilai mentah `sort_order`.
```

**Kanban** (kolom `Belum` · `Sedang Berjalan` · `Selesai`; kartu menampilkan badge sesi dan kategori, judul, tanggal, badge merah *Terlambat* dari `overdue`, inisial PIC):

```text
┌─ Belum (6) ───────────┐ ┌─ Sedang Berjalan (29) ─┐ ┌─ Selesai (15) ────────┐
│ [Resepsi] Tamu ✦      │ │ [Umum] Logistik        │ │ [Lamaran] Keluarga    │
│ Finalisasi daftar…    │ │ Finalisasi Seserahan   │ │ Pertemuan keluarga    │
│ 12 Feb – 13 Feb       │ │ 2 Nov – 3 Nov          │ │ 4 Okt – 5 Okt         │
└───────────────────────┘ └────────────────────────┘ └───────────────────────┘
```

**Gantt** (minggu mulai Senin; balok inklusif; garis Hari Ini; kolom kiri *sticky*):

```text
┌────┬──────────────────────┬─────┬──────────┬─ Minggu 1 · 28 Sep ─┬─ Minggu 2 · 5 Okt ──┐
│ No │ Tugas                │ Dur │ Status   │ S  S  R  K  J  S  M │ S  S  R  K  J  S  M │
│  1 │ Pertemuan keluarga   │ 1 h │ Selesai  │                  ███│███                  │
│  2 │ Pilih tgl lamaran    │ 1 h │ Selesai  │                   ┊ │   ██████            │
│  3 │ Tetapkan anggaran…   │ 2 h │ Selesai  │                   ┊ │         █████████   │
│  4 │ Draf daftar tamu…    │ 2 h │ Selesai  │                   ┊ │                  ███│
│  5 │ Book vendor lamaran  │ 1 h │ Selesai  │                   ┊ │                  ███│
└────┴──────────────────────┴─────┴──────────┴─────────────────────┴─────────────────────┘
  ┊ = garis "Hari ini" (Minggu, 4 Okt); balok inklusif: 8–10 Okt = 3 kolom, durasi 2 hari
```

**Keadaan kosong** (proyek tanpa tugas; ringkasan `total = 0`, tanpa persen):

```text
│ Progres: Belum ada jadwal                                                                    │
│            Belum ada tugas.  [⚡ Buat jadwal otomatis]   [+ Tambah Tugas]                     │
```

### 7.2 Pemetaan data → tampilan

| Elemen | Sumber | Tampilan | Keadaan kosong |
|---|---|---|---|
| Progres | `percent(completed, total)` dari `summary` | batang (`barWidth`) + `formatPercent` + "(selesai/total)" | `total = 0` → "Belum ada jadwal", tanpa batang dan persen |
| Selesai · Berjalan · Belum · Terlambat | `completed` · `in_progress` · `not_started` · `overdue` | empat angka; "Terlambat" bergaya peringatan bila > 0, dengan teks (bukan hanya warna) | selalu tampil, angka 0 |
| Chip sesi | `GET sessions`: "Semua" (`SCOPE_ANY`), "Umum" (sesi sistem `all`; **teks tetap "Umum"**, bukan `name` server "Umum / Semua Sesi"), sesi lain urut `sort_order` | emoji `icon` + `name` + angka dari `countByScope` | nol sesi non-sistem → hanya "Semua" dan "Umum" |
| Nomor | `taskNumbers` (bab 5) | kolom "No" | – |
| Baris tugas | `Task` | bab 7.4 | – |
| Badge "Tambahan Adicara" | `origin = 'adicara'` | ✦ + teks pada `title` atribut | – |
| Vendor | `vendor_name` (+ `vendor_wa`) | chip nama; tombol WhatsApp hanya bila `whatsappUrl(vendor_wa)` bukan `null` | tanpa vendor → tidak ada chip |
| Terlambat | `task.overdue` | badge "Terlambat" pada baris/kartu | – |
| Nama fase | `groupByPhase` | "FASE {order} — {name} ({jumlah})"; nama kosong → "Fase {order}" | – |

Daftar kosong tetapi ada chip aktif → "Tidak ada tugas di sesi ini."

### 7.3 Kontrol

1. **Chip sesi** menyaring Tabel, Kanban, dan Gantt tanpa muat ulang (mengubah `hidden`, bab 2.4); pilihan disimpan di URL `?scope=<scope_key>` lewat `history.replaceState` (muat ulang mempertahankannya; "Semua" tidak ditulis). Kelompok fase yang semua barisnya tersembunyi ikut disembunyikan; angka di judul kelompok tetap jumlah seluruh kelompok. Ringkasan di header tetap seluruh proyek ([BE bab 8, butir 10](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
2. **View Switcher**: Tabel, Kanban, Gantt. Pilihan terakhir diingat di `sessionStorage` (bab 2.3), tidak di server; default Tabel. Ketiga tampilan sudah ada di DOM; memilih hanya menukar `hidden` dan `aria-pressed`.
3. **Tombol**: `⚡ Generate Jadwal`, `+ Tambah Tugas`, `📥 Ekspor .xlsx`, dan di menu "⋯" `Geser jadwal…`. Pada proyek `archived`, tiga yang pertama nonaktif (ekspor tetap aktif).

### 7.4 Mode tampilan

**Tabel** *(belum diuji di browser)*: dikelompokkan dengan `groupByPhase` (accordion `<details open>`, judul sesuai tabel 7.2). Kolom: checkbox cepat (`toggledStatus`), No (`taskNumbers`), judul + badge sesi + ✦ + vendor + *Terlambat*, kategori, jadwal (klik membuka dua isian tanggal; ubah → `PATCH`), durasi (dari server), PIC, dropdown status (`TASK_STATUS_LABELS`), menu ⋮ (Ubah, Hapus, Naik, Turun). Pegangan seret untuk urut ulang hanya di dalam satu kelompok fase (bab 5); tombol Naik/Turun adalah padanan papan ketik dan memakai `moveByOne`.

**Kanban** *(belum diuji di browser)*: tiga kolom dari `kanbanColumns`. Seret-lepas HTML5 native dengan *optimistic update*, lalu `PATCH …/status`; bila gagal, kartu dikembalikan dan toast galat tampil (teks dari `messages.ts`). Padanan papan ketik: menu kartu "Pindahkan ke…" (Belum / Sedang Berjalan / Selesai). Klik kartu membuka `TaskModal`. Kartu menampilkan badge sesi dan kategori, judul, tanggal, badge *Terlambat* dari `overdue`, inisial PIC.

**Gantt** *(belum diuji di browser)*: CSS Grid tanpa pustaka, dari `buildGantt(tasks, today)`. Kolom kiri *sticky* (No, tugas, durasi, status); kanan header minggu "Minggu {n} · {formatDateShort(start)}" dan hari (Sen…Mgg). Balok `grid-column: {col + 1} / span {span}`: terisi untuk `selesai`, terisi lebih muda untuk `proses`, kontur untuk `belum`; setiap balok punya `aria-label` ("Finalisasi Seserahan, 2–3 Nov, berjalan"). Garis vertikal "Hari Ini" pada kolom `todayCol` (hilang bila `null`); legenda menyebut tanggalnya sebagai teks. Gantt **hanya baca**: mengubah jadwal lewat modal atau tabel. Rentang lebar digulir horizontal di dalam wadahnya; `buildGantt` mengembalikan `null` untuk daftar kosong (tampilkan keadaan kosong). Rentang dihitung dari semua tugas, bukan dari hasil saringan chip (baris yang tidak cocok disembunyikan, kolom tetap).

Setiap perubahan dan pembaruan memakai alur bab 7.6 (optimistik untuk yang berbalik, lalu partial menjadi kebenaran akhir).

### 7.5 Modal dan dialog

**Tambah / Ubah Tugas** (`TaskModal`, `<dialog>`):

| Kolom | Kontrol | Aturan klien (kenyamanan; server yang menentukan) | Kirim |
|---|---|---|---|
| Judul\* | teks | wajib setelah `trim`; ≤ `LIMITS.title` (`charCount`); tanpa Enter/tab | `title` |
| Sesi | dropdown "Umum" + sesi non-sistem proyek | bawaan = chip sesi yang sedang aktif bila bukan Semua/Umum, selain itu Umum | `sub_event_scope` (`all` untuk Umum) |
| Kategori | dropdown dari `categoriesOf` + "Lainnya…" (isian teks) | ≤ `LIMITS.category`; bawaan "Umum" | `category` |
| Fase | angka `LIMITS.phaseOrder` + nama | bawaan `defaultPhase(tasks)`; mengubah nomor mengisi nama dari `phaseNameOf` kecuali pengguna sudah mengetik nama; ≤ `LIMITS.phaseName` | `phase_order` **dan** `phase_name` (selalu berpasangan) |
| Mulai\* · Selesai\* | `type="date"` | selesai ≥ mulai; teks bantu "Durasi dihitung server" (tampilkan `duration_days` dari respons, bukan hitungan klien) | `start_date`, `end_date` |
| PIC | teks | ≤ `LIMITS.pic` | `pic` |
| Vendor | dropdown vendor proyek, opsional; disembunyikan bila daftar vendor tidak tersedia | "Tanpa vendor" saat Ubah mengirim `clear_vendor: true` (bukan `vendor_id: null`); saat Tambah tidak mengirim apa pun | `vendor_id` · `clear_vendor` |
| Catatan | textarea | ≤ `LIMITS.notes`, penghitung karakter; baris baru boleh | `notes` |

Saat **Ubah**, kirim hanya kolom yang berubah (server menggabung di bawah kunci proyek, jadi dua tab yang mengubah kolom berbeda tidak saling menimpa); pasangan fase dikirim bersama bila salah satunya berubah. Galat `<field>: <alasan>` dipetakan ke kolom dengan `parseFieldError` dan teks Indonesia dari bab 9 (`aria-invalid`, `aria-describedby`, fokus ke kolom pertama yang salah); galat tanpa kolom (`task_limit_reached`, `invalid_scope`, `5xx`) tampil sebagai banner di dalam modal. Tombol Simpan nonaktif selama permintaan (`POST` tidak idempoten). Bila `GET vendors` tidak tersedia, kolom Vendor tidak digambar dan tugas yang sudah bervendor tetap menampilkan chip-nya dari `vendor_name`.

**Generator Jadwal Otomatis** (`GeneratorModal`):

- Daftar preset dari `type.templates_available.timeline_presets` (kosong → tombol Generate disembunyikan). Teks dari `TEMPLATE_LABELS` (01 FE bab 8.4, kamus yang sama dengan wizard): `wedding_pdf_44` "Lengkap (44 tugas dari Wedding Planner)", `wedding_essential_22` "Esensial (22 tugas)", `general` "Jadwal standar jenis acara" (**tanpa angka**: 18 tugas untuk tujuh jenis, 15 untuk `other`; bab 13 butir 3). Tidak ada label dari registry ([BE bab 8, butir 1](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- Centang "Tambahkan tugas Adicara" hanya bila `templates_available.include_adicara_tasks` (aktif secara bawaan; hanya pernikahan, enam tugas).
- Mode: ⚪ *Tambah yang belum ada* (`append`, default; aman diulang) · ⚪ *Ganti tugas template, simpan tugas saya* (`replace_template`) · ⚪ *Ganti semuanya* (`replace_all`). Setelah memilih mode, tampilkan hasil `generatorImpact(tasks, mode)` dari daftar lokal: "{removed} tugas akan dihapus" dan, bila `atRisk > 0`, "{atRisk} di antaranya sudah berjalan, selesai, atau bervendor". Bila `needsConfirm`, kotak "Saya mengerti tugas ini akan hilang" wajib dicentang sebelum tombol Kirim aktif dan menghasilkan `confirm: true`. **Aturannya berbeda per mode**: `replace_template` hanya bila ada kemajuan; `replace_all` **selalu** bila proyek punya tugas (walau semua "belum"). Bila server tetap menjawab `400 invalid_input: confirm` karena daftar lokal usang, tampilkan kotak itu lalu kirim ulang.
- Hasil (`GeneratorResult`): "Dibuat {created}, dilewati {skipped}{, dihapus {deleted}}" ditambah tiap `warnings` lewat `warningText(code, count)` (bab 9.4), lalu tombol "Tutup" menjalankan `refresh()`. `warnings` adalah catatan, bukan galat. `created: 0` bukan galat ("Semua tugas template sudah ada").
- Acara yang tanggalnya sudah lewat → `400 invalid_date: event_date: must not be before today` → "Tanggal acara sudah lewat; ubah tanggal acara dulu." Batas 500 tugas → `409 task_limit_reached`, tidak ada yang tertulis (semua-atau-tidak).

**Geser jadwal** (`ShiftDialog`, dari menu "⋯"): isian `days` (bilangan bulat ≠ 0, \|days\| ≤ `LIMITS.maxShiftDays`; negatif = mundur) dan teks "Geser {n} tugas template yang belum selesai {maju|mundur} {|days|} hari." dengan `n` = `shiftable_tasks` dari `GET projects/:id` (0 → tombol nonaktif, "Tidak ada tugas template yang belum selesai"). Hasil: "{shifted} tugas digeser" ditambah peringatan `tasks_overdue_after_shift`. Galat khusus: `invalid_date: days` (sebagian tugas keluar dari jendela jadwal; tidak ada yang berubah). Aksi ini **tidak idempoten**; tombol dinonaktifkan selama permintaan. Dialog yang muncul saat Hari-H diubah milik Menu 01 ([01 FE bab 9.3](../01_DASHBOARD/FE.md#93-edit-acara-editeventdialog)) dan memakai `PATCH projects/:id` dengan `shift_tasks`, bukan dialog ini.

**Konfirmasi hapus** (`ConfirmDialog`): untuk tugas dan berkas; menyebut judulnya. Satu `<dialog>` dipakai bersama.

### 7.6 Alur interaksi dan skrip klien

`timelineClient.ts` adalah satu-satunya skrip klien halaman (diimpor oleh `<script>` di `index.astro`). Ia **tidak merender** daftar: ia membaca DOM yang dirender server, memasang satu set penangan *delegasi* pada `[data-timeline-live]` (sehingga tidak perlu dipasang ulang setelah partial mengganti isinya), dan memakai fungsi murni bab 5 untuk keputusan. Semua string dari API ditulis dengan `textContent`/`createElement`, tidak pernah `innerHTML`.

**Data yang dibaca klien.** Komponen yang sama menanam satu blok JSON di dalam `[data-timeline-live]` (ikut diganti oleh partial), supaya fungsi murni bekerja pada `Task[]` yang segar:

```html
<script type="application/json" data-timeline-data>{"tasks":[…],"sessions":[…],"project":{"id":"…","timezone":"Asia/Jakarta","archived":false,"shiftableTasks":35}}</script>
```

Server menulisnya dengan `JSON.stringify(data).replace(/</g, '\\u003c')` (mencegah `</script>` dari judul/catatan memutus blok). Klien membacanya dengan `JSON.parse(el.textContent)` setiap kali ia membutuhkan keadaan terkini; tidak ada salinan jangka panjang di variabel global.

#### 7.6.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Isi |
|---|---|---|
| wadah `[data-timeline-live]` | `data-tab` | `jadwal` · `dokumen` |
| baris Tabel, kartu Kanban, baris Gantt | `data-task-id`, `data-scope` (`scopeOf(task, known)`, jadi sesi usang sudah `all`), `data-status`, `data-phase`, `data-focus-key` (`task-{id}`) | satu tugas; tiga tampilan membawa atribut yang sama |
| kelompok fase (`<details>`) | `data-phase-group` | `phase_order` |
| chip sesi / subjek | `data-scope-chip` / `data-subject-chip` + `aria-pressed` | `SCOPE_ANY` atau kunci |
| tombol tampilan dan panelnya | `data-view-btn` / `data-view-panel` (+ `hidden`) | `tabel` · `kanban` · `gantt` |
| penghitung header | `data-count` | `total` · `completed` · `in_progress` · `not_started` · `overdue`; `data-progress` pada batang |
| kontrol di baris | `data-action` | `toggle` · `status` · `edit` · `delete` · `up` · `down` · `move` · `dates` |
| baris berkas | `data-doc-id`, `data-subject`, `data-status`, `data-focus-key` (`doc-{id}`) | satu berkas |
| kontrol berkas | `data-action` | `toggle` · `sync` · `edit` · `delete` |

Satu permintaan per baris: selama `aria-busy="true"` pada baris, klik berikutnya pada baris itu diabaikan (mencegah balapan dan dua `POST`).

#### 7.6.2 Alur per aksi

Tanda ◆ = berubah secara optimistik. Pada semua alur, "galat" berarti `!res.ok`; teks dari `timelineErrorMessage(status, code, where) ?? plannerErrorMessage(status, code)` (bab 9), ditampilkan sebagai toast (`aria-live="polite"`) kecuali disebut lain. Galat yang menandakan data usang (`not_found`, `invalid_scope`, `invalid_input: ids`) selalu diakhiri `refresh()` (bab 2.4).

| # | Aksi | Langkah |
|:-:|---|---|
| A | **Ubah status** ◆ (checkbox, dropdown, "Pindahkan ke…", seret Kanban) | 1. `prev` = daftar sekarang; `next = withStatus(prev, id, status)`. 2. Ubah DOM: checkbox, lencana (`TASK_STATUS_BADGES`), `data-status`, kartu Kanban pindah kolom, penghitung header dari `summarize(next)`; baris `aria-busy`. 3. `api.setTaskStatus`. 4. Sukses: `replaceTask(next, res.data)` (membawa `overdue` yang benar), lalu `refresh()`. 5. Galat: kembalikan DOM ke `prev` (status lama, kartu ke kolom asal), toast; `refresh()` bila data usang. Dropdown status dan seret Kanban mengirim nilai persis; checkbox mengirim `toggledStatus` |
| B | **Ubah tanggal di Tabel** | Klik sel jadwal → dua `type="date"`; Enter/blur mengirim `updateTask(id, {start_date, end_date})` (keduanya, agar tidak pernah ada `start > end` sementara). Tidak optimistik (durasi dan terlambat ditentukan server): sel `aria-busy`. Sukses → `refresh()`. `invalid_date` → pesan di bawah sel, nilai lama dikembalikan; Esc membatalkan |
| C | **Urut ulang** ◆ (seret / Naik / Turun) | 1. `ids` = `moveBefore(semuaId, id, idTujuan)` atau `moveByOne(semuaId, idKelompok, id, ±1)` (`null` → tombol sudah nonaktif). 2. Pindahkan baris di DOM. 3. `api.reorderTasks(ids)`. 4. Sukses: `refresh()` (nomor dihitung ulang). 5. Galat: kembalikan posisi; `invalid_input: ids` → `refresh()` |
| D | **Tambah / Ubah tugas** | 1. Buka `TaskModal` (isian bawaan bab 7.5). 2. Validasi kenyamanan; Simpan `aria-busy`. 3. `createTask` / `updateTask` (hanya kolom yang berubah). 4. Sukses: tutup modal, `refresh()`, toast "Tugas ditambahkan." / "Perubahan disimpan.", fokus ke baris itu (`data-focus-key`) bila masih tampil, selain itu ke tombol pemicu. 5. `400` berkolom → di bawah kolom; `task_limit_reached`, `invalid_scope`, `5xx` → banner di modal, modal tetap terbuka |
| E | **Hapus tugas** | `ConfirmDialog` ("Hapus tugas?") → `api.deleteTask`. `204` **atau `404`** → `refresh()` + toast "Tugas dihapus." (404 berarti sudah terhapus di tempat lain). Galat lain → toast, baris tetap |
| F | **Generate** | 1. Buka `GeneratorModal`; preset pertama dan `append` terpilih. 2. Tiap ganti mode: `generatorImpact(tasks, mode)` → teks dampak dan kotak konfirmasi (bab 7.5). 3. Kirim `api.generate({preset, include_adicara_tasks, mode, confirm})` — `confirm: true` hanya bila `needsConfirm`. 4. `400 invalid_input: confirm` → tampilkan kotak konfirmasi, jangan tutup. 5. Sukses: tampilkan `GeneratorResult`; "Tutup" → `refresh()`. 6. `invalid_date: event_date`, `task_limit_reached` → banner; `invalid_template` → "Muat ulang halaman" |
| G | **Geser jadwal** | Dialog → `api.shiftTasks(days)`; tombol nonaktif selama permintaan; sukses → hasil + `refresh()`. Tidak pernah diulang otomatis |
| H | **Ekspor** | Tautan biasa (bab 6.2); tidak ada alur klien |
| I | **Chip sesi, tampilan, subjek** | Hanya DOM: ubah `hidden` pada baris menurut `data-scope`, sembunyikan kelompok fase kosong, `history.replaceState` untuk `scope`/`subject`, `sessionStorage` untuk tampilan. Tidak memanggil server |

Pembaruan sesudah aksi (`refresh()`): `fetch('/dashboard/planner/{id}/timeline/live?tab={tab}', { credentials: 'same-origin', headers: { Accept: 'text/html' } })` → ganti isi `[data-timeline-live]` → terapkan ulang chip/tampilan/subjek → pulihkan fokus (`data-focus-key`). Permintaan `refresh()` yang masih berjalan dibatalkan (`AbortController`) bila aksi lain mulai, supaya respons lama tidak menimpa yang baru.

#### 7.6.3 Papan ketik

| Kontrol | Tombol |
|---|---|
| Chip sesi/subjek, tombol tampilan | `Tab` ke grup, `Enter`/`Spasi` memilih (`aria-pressed`); panah kiri/kanan berpindah antar chip |
| Centang tugas | `Spasi` |
| Menu ⋮ / "Pindahkan ke…" | `Enter` membuka, panah atas/bawah memilih, `Esc` menutup dan mengembalikan fokus |
| Urut ulang | tombol Naik/Turun di menu ⋮ (padanan seret-lepas); fokus tetap pada baris yang sama sesudah `refresh()` (`data-focus-key`) |
| Dialog | `Esc` menutup; fokus awal pada kolom pertama (atau tombol aman pada konfirmasi hapus); fokus kembali ke pemicu |
| Gantt | dilewati: Tabel memuat data yang sama |

### 7.7 Salinan teks tampilan

Teks tetap di komponen dan `messages.ts` (bukan dari API). Teks yang berasal dari API (judul, kategori, PIC, nama vendor/berkas, langkah panduan, nama dan ikon sesi) selalu ditulis sebagai teks.

| Area | Teks |
|---|---|
| Tab | "Jadwal & Milestone" · "Dokumen & Legalitas" |
| Progres | "Progres" · "{completed}/{total}" · "Belum ada jadwal" · penghitung "Selesai", "Berjalan", "Belum", "Terlambat" |
| Tombol | "Generate Jadwal" · "Tambah Tugas" · "Ekspor .xlsx" · "Geser jadwal…" · "Buat jadwal otomatis" · "Muat ulang" |
| Chip dan tampilan | "Semua" · "Umum" · "Tabel" · "Kanban" · "Gantt" |
| Kolom Tabel | "No" · "Tugas" · "Kategori" · "Jadwal" · "Durasi" · "PIC" · "Status"; durasi "{n} hari"; jadwal "{formatDateShort(mulai)} – {formatDateShort(selesai)}" |
| Status | `TASK_STATUS_LABELS` ("Belum", "Sedang Berjalan", "Selesai"); lencana `TASK_STATUS_BADGES` ("Belum", "Berjalan", "Selesai") |
| Lencana | "Terlambat" · ✦ dengan `title` "Tambahan Adicara (bukan dari dokumen acuan)" |
| Menu ⋮ | "Ubah" · "Hapus" · "Naik" · "Turun" · "Pindahkan ke…" |
| Kosong | "Belum ada tugas." · "Tidak ada tugas di sesi ini." |
| Banner | "Acara diarsipkan" · "Timeline sedang tidak tersedia" · "Daftar belum diperbarui" |
| Toast sukses | "Tugas ditambahkan." · "Perubahan disimpan." · "Tugas dihapus." · "Urutan disimpan." · "{shifted} tugas digeser." |
| Dialog hapus | judul "Hapus tugas?" · isi "“{judul}” akan dihapus. Tindakan ini tidak dapat dibatalkan." · tombol "Hapus", "Batal" |
| Generator | judul "Generate Jadwal Otomatis" · mode "Tambah yang belum ada", "Ganti tugas template, simpan tugas saya", "Ganti semuanya" · dampak "{removed} tugas akan dihapus" / "{atRisk} di antaranya sudah berjalan, selesai, atau bervendor" · kotak "Saya mengerti tugas ini akan hilang" · "Semua tugas template sudah ada" (bila `created = 0`) |
| Geser | "Geser {n} tugas template yang belum selesai {maju\|mundur} {\|days\|} hari." · "Tidak ada tugas template yang belum selesai" · hasil "{shifted} tugas digeser." |
| Gantt | "Minggu {n} · {tanggal}" · legenda "Hari ini: {tanggal}" · `aria-label` "{judul}, {mulai}–{selesai}, {status}" |

---

## 8. Tab 2: Dokumen & Legalitas

### 8.1 Wireframe

Pernikahan, status dari PDF Hal. 11 (Proyek Contoh; sama dengan `fixtures/planner-docs.json`). Enam chip subjek karena registry memuat lima subjek:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [Jadwal & Milestone]   [★ Dokumen & Legalitas]                       Kelengkapan: 9 / 15 (60%)│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Subjek: [Semua] [CPP] [CPW] [Orang Tua CPP] [Orang Tua CPW] [Bersama]                        │
│ [+ Tambah Berkas]   [Terapkan Template ▼]                                                    │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│    No Dokumen                          Subjek   Jumlah     Biaya        Anggaran       Aksi   │
│ ☑   1 Pendaftaran KUA                  Bersama  1 berkas   Rp 600.000   [➡️ Kirim]      ✏️ 🗑️ │
│ ☑   2 FC CPP ( KTP, Ijazah, Akte, KK)  CPP      2 lembar   Rp 0                         ✏️ 🗑️ │
│ ☐   4 FC KTP Ortu CMP                  Ortu CPP 2 lembar   Rp 0                         ✏️ 🗑️ │
│ ☐   9 Pas Foto Background Biru 2x3…    Bersama  3 pcs      Rp 15.000    [➡️ Kirim]      ✏️ 🗑️ │
│ ☐  13 Materai 10.000                   Bersama  2 pcs      Rp 20.000    [➡️ Kirim]      ✏️ 🗑️ │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 💡 Panduan Pengurusan Berkas Nikah                                                            │
│  1 RT & RW · 2 Kelurahan (N1, N2, N4) · 3 KUA asal (numpang nikah) · 4 KUA tempat akad        │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 Setelah "Kirim" pada nomor 13: kolom Anggaran menjadi  ✓ Di anggaran  dan tombol hilang (tidak ada "lepas").
```

Gambar menyingkat nama panjang; di UI nama tampil penuh (terpotong dengan elipsis CSS, penuh pada `title`). Jenis lain: chip `[Semua] [Panitia] [Lembaga/Instansi] [Umum]` (label dari registry) dan callout "Panduan Perizinan Acara" (empat langkah). Jenis `other` tanpa template berkas: daftar kosong dengan ajakan "Belum ada berkas. [+ Tambah Berkas]" dan tanpa menu Terapkan Template (`templates_available.admin_docs = false`; pengguna tetap boleh menambah berkas sendiri).

### 8.2 Kontrol dan perilaku

- **Header**: `Kelengkapan: {completed} / {total} ({formatPercent(percent(completed,total))})`; `total = 0` → "Belum ada berkas" tanpa persen. Angka dari `summary` (seluruh proyek, tidak berubah oleh filter subjek).
- **Filter subjek**: `[Semua]` + satu chip per `type.admin_subjects[]` (label dari registry, bukan tertulis) lewat `filterDocs`; disimpan di URL `?subject=`; nilai tak dikenal = Semua. Disaring dengan `hidden` seperti chip sesi (bab 2.4), jadi tanpa panggilan server.
- **Kolom Jumlah dan Biaya** adalah saran Adicara (bukan dari PDF); ditandai ikon info "bukan dari dokumen acuan" (`title`) dan bebas diubah. Nilai uang `formatRupiah`. Nama berkas: `LIMITS.documentName`.
- **Centang** `belum` ↔ `selesai` lewat `PATCH …/status` (optimistik, kembali bila gagal; alur J).
- **Kirim ke anggaran**: tombol tampil bila `canSyncBudget(doc)` (biaya > 0 dan belum tertaut). Sukses (`ok`, `201` pos baru atau `200` ditautkan/sudah tertaut) → baris berubah menjadi badge "✓ Di anggaran" dari `budget_item_id` pada respons; warning `estimate_differs` ditampilkan sebagai catatan di baris ("Estimasi di anggaran berbeda dari biaya berkas; angka di anggaran tidak diubah"), bukan galat. Tombol menjadi badge, tidak ada "lepas" ([BE bab 8, butir 9](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Mengubah biaya berkas yang sudah di anggaran tidak mengubah pos; FE tidak tahu angka pos, jadi cukup teks statis pada badge: "Angka di anggaran tidak ikut berubah". Sesi tujuan pos dipilih server (`akad` untuk Pendaftaran KUA, selain itu `all`); UI tidak mengirim `sub_event_scope`. Klik ganda aman (server menyerialkan), tetapi tombol tetap `aria-busy` selama permintaan.
- **Terapkan Template** (menu): "Tambah yang kurang" (`append`) dan "Ganti template" (`replace_template`). Untuk "Ganti template", `docsImpact(docs, 'replace_template')` memberi jumlah berkas yang dihapus dan, bila `needsConfirm` (ada berkas template yang selesai atau tertaut anggaran), kotak "Saya mengerti status berkas ini akan kembali ke Belum" wajib dicentang → `confirm: true`; bila ada yang tertaut, tambahkan "Pos di anggaran tetap ada; hanya tautannya yang lepas". Berkas kustom tidak ikut dihapus. Tidak ditampilkan bila `!type.templates_available.admin_docs`. Hasil `created/skipped/deleted` dan `warnings` (`budget_link_lost`) ditampilkan dengan `GeneratorResult` (`kind: 'docs'`); lalu `refresh()`. Mode `replace_all` **tidak ada** untuk berkas (`400 invalid_input: mode`).
- **Panduan** (`GuideCallout`): `var(--sage-50)` dengan batas `var(--sage-200)`; judul dan 4 langkah dari `type.admin_guide`, dirender sebagai daftar bernomor teks (langkah berbentuk "Lokasi: penjelasan", panjang hingga beberapa kalimat). Pernikahan menampilkan teks catatan PDF Hal. 11 (formulir **N1, N2, dan N4**); jenis lain menampilkan 4 langkah perizinan ([TEMPLATE_EVENT bab 4.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#41-subjek-dokumen-label-widget-dan-panduan-milik-menu-02)). Tidak ada teks panduan di komponen.

### 8.3 Modal dan alur

**Tambah / Ubah Berkas** (`DocModal`): nama\* (≤ `LIMITS.documentName`); subjek (dropdown `admin_subjects`, bawaan `bersama` bila ada, selain itu `umum`; subjek yang tidak ada di registry dijawab `400 invalid_input: subject_role`); jumlah / rangkap (≤ `LIMITS.quantityDetail`); biaya (Rupiah, **bilangan bulat** ≥ 0 dan ≤ `LIMITS.maxCost`, `inputmode="numeric"`; kirim sebagai angka, bukan string, bab 4.1); catatan (≤ `LIMITS.notes`). Saat Ubah kirim hanya kolom yang berubah.

| # | Aksi | Langkah |
|:-:|---|---|
| J | **Centang berkas** ◆ | `prev` = salinan; ubah centang dan `data-status`; header `Kelengkapan` dihitung ulang dari baris; `api.setDocStatus`; sukses → `replaceDoc` + `refresh()`; galat → kembalikan, toast |
| K | **Kirim ke anggaran** | tombol `aria-busy` → `api.syncBudget(id)` → `ok`: badge "✓ Di anggaran" + catatan `estimate_differs` bila ada, lalu `refresh()`; `invalid_input: nominal_cost` (biaya diubah jadi 0 di tab lain) → "Biaya berkas harus lebih dari 0…" lalu `refresh()`; `not_found` → "Berkas tidak ditemukan. Muat ulang daftar." |
| L | **Terapkan Template** | menu → (bila "Ganti template") dialog dampak + konfirmasi → `api.applyDocTemplate({mode, confirm?})` → `GeneratorResult` → "Tutup" memuat ulang. `400 confirm: required` (daftar usang) → tampilkan kotak lalu kirim ulang; `invalid_template` → muat ulang halaman |
| M | **Tambah / Ubah / Hapus berkas** | seperti D dan E untuk tugas; `409 admin_doc_limit_reached` → banner; hapus berkas tertaut: `ConfirmDialog` ditambah "Pos di anggaran tetap ada; hanya tautannya yang lepas" (benar: server mengosongkan `admin_doc_id`, angka pos tetap) |

| Area | Teks |
|---|---|
| Header | "Kelengkapan: {c} / {t} ({p})" · "Belum ada berkas" |
| Kolom | "No" · "Dokumen" · "Subjek" · "Jumlah" · "Biaya" · "Anggaran" · "Aksi" |
| Tombol | "Tambah Berkas" · "Terapkan Template" (menu: "Tambah yang kurang", "Ganti template") · "Kirim ke anggaran" · badge "✓ Di anggaran" · catatan "Angka di anggaran tidak ikut berubah" |
| Kosong | "Belum ada berkas. Tambah Berkas" · "Tidak ada berkas untuk subjek ini." |
| Dialog hapus | "Hapus berkas?" · "“{nama}” akan dihapus. Tindakan ini tidak dapat dibatalkan." |
| Konfirmasi template | "Saya mengerti status berkas ini akan kembali ke Belum" |
| Toast | "Berkas ditambahkan." · "Perubahan disimpan." · "Berkas dihapus." |

---

## 9. Pesan galat dan teks

Semua teks lewat `lib/planner/messages.ts` (01 FE bab 5.4 dan 11): kode lama tetap, berikut tambahan menu ini. **Pola pemakaian:** fungsi Menu 02 mengembalikan `null` bila bukan miliknya, dan pemanggil jatuh ke fungsi 01:

```ts
const text = timelineErrorMessage(res.status, res.error?.code, 'task') ?? plannerErrorMessage(res.status, res.error?.code);
const fieldText = timelineFieldMessage(field, reason) ?? fieldErrorMessage(field, reason);
```

Dua alasan memisahkannya dari tabel 01: teks `invalid_scope`/`invalid_template`/`not_found` bergantung pada aksi (Menu 01: sesi/wizard/proyek; Menu 02: tugas/berkas/generator), dan fungsi 01 belum ada sehingga Menu 02 tidak boleh bergantung pada bentuk internalnya.

### 9.1 Kode → teks

| Kode / kondisi | Teks Indonesia | Fungsi | Muncul pada |
|---|---|---|---|
| `invalid_input`, `invalid_date` | **per kolom** (9.2); tanpa kolom: "Periksa kembali data Anda." / "Periksa kembali tanggalnya." | 01 | modal tugas, berkas, generator, geser |
| `invalid_scope` | "Sesi tidak valid atau sudah dihapus. Muat ulang halaman." | `timelineErrorMessage` | tugas |
| `invalid_template` | "Template tidak tersedia untuk jenis acara ini. Muat ulang halaman." | `timelineErrorMessage` | generator, terapkan template |
| `task_limit_reached` | "Jumlah tugas sudah mencapai batas (500). Hapus tugas yang tidak dipakai lalu coba lagi." | `timelineErrorMessage` | tambah tugas, generator |
| `admin_doc_limit_reached` | "Jumlah berkas sudah mencapai batas (200). Hapus berkas yang tidak dipakai lalu coba lagi." | `timelineErrorMessage` | tambah berkas, terapkan template |
| `not_found`, `where = 'task'` / `'doc'` | "Tugas tidak ditemukan. Muat ulang daftar." / "Berkas tidak ditemukan. Muat ulang daftar." | `timelineErrorMessage` | ubah, hapus (dianggap sukses), status, sinkron |
| `not_found`, `where = 'project'` | "Acara tidak ditemukan." (halaman: `/404`) | 01 | semua |
| `rate_limited`, `where = 'export'` | "Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi." | `timelineErrorMessage` | ekspor |
| `invalid_json`, `project_read_only`, `conflict`, `401`, `5xx`, jaringan | sama dengan [01 FE bab 11.1](../01_DASHBOARD/FE.md#111-kode--teks-plannererrormessagestatus-code) | 01 | semua penulisan |

`where` ditentukan oleh **aksi** yang sedang berjalan, bukan oleh pesan server (pesan "task not found" / "document not found" / "project not found" berbahasa Inggris dan tidak boleh diurai).

### 9.2 Alasan per kolom

Alasan yang sudah ada di 01 FE bab 11.2 (wajib/`must not be empty`, panjang, tanggal valid, karakter kontrol, `must be between A and B`, `must not be before today`) tetap berlaku untuk kolom Menu 02 (mis. `phase_order: must be between 1 and 20` → "Nomor fase harus antara 1 dan 20."). Tambahan Menu 02 (`timelineFieldMessage`; **setiap alasan di `fixtures/planner-errors.json` sudah tercakup**, diuji):

| Pola alasan (regex) | Kolom | Teks (`{label}` = label kolom, 9.3) |
|---|---|---|
| `^must not be before start_date$` | `end_date` | "Tanggal selesai tidak boleh sebelum tanggal mulai." |
| `^must not be after end_date$` | `start_date` | "Tanggal mulai tidak boleh setelah tanggal selesai." |
| `^must be within the project schedule window$` | `start_date`, `end_date` | "{label} harus dalam rentang tiga tahun sebelum sampai setahun sesudah acara." |
| `^must keep every shifted task within the project schedule window$` | `days` | "Geseran ini membuat sebagian tugas keluar dari rentang jadwal (tiga tahun sebelum sampai setahun sesudah acara). Kurangi jumlah harinya." |
| `^must be a vendor of this project$` | `vendor_id` | "Vendor tidak ditemukan di acara ini. Muat ulang daftar vendor." |
| `^must be one of the event type subjects$` | `subject_role`, `subject` | "Subjek tidak valid." (kesalahan klien: kontrol yang salah) |
| `^must list every id exactly once$` | `ids` | "Urutan sudah usang karena daftar berubah. Muat ulang halaman." |
| `^must not be zero$` | `days` | "Jumlah hari tidak boleh 0." |
| `^must be at most 1826 in absolute value$` | `days` | "Paling banyak 1.826 hari (5 tahun)." |
| `^must be greater than 0$` | `nominal_cost` (sinkron anggaran) | "{label} harus lebih dari 0 untuk dikirim ke anggaran." |
| `^must be between 0 and (\d+)$` | `nominal_cost` saja | "Biaya paling banyak Rp 10.000.000.000.000." (kolom lain: tabel 01) |
| `^cannot be combined with vendor_id$` | `clear_vendor` | "Pilih vendor atau pilih "Tanpa vendor", tidak keduanya." (kesalahan klien) |
| `^must be one of ` | `status`, `mode`, … | "{label} tidak valid." (kesalahan klien) |
| `^must be a session scope_key$` | `scope` | "Sesi tidak valid." (kesalahan klien; UI tidak memakai parameter ini) |
| `confirm: required` | `confirm` | bukan teks galat: tampilkan kotak konfirmasi dan kirim ulang dengan `confirm: true` (bab 7.5) |
| `event_date: must not be before today` | `event_date` | pada generator: "Tanggal acara sudah lewat; ubah tanggal acara dulu." (di Menu 01: teks 01) |
| alasan lain | – | "{label} tidak valid." |

### 9.3 Label kolom

`TIMELINE_FIELD_LABELS` (kode di 9.4): `title` "Judul tugas", `document_name` "Nama berkas", `start_date` "Tanggal mulai", `end_date` "Tanggal selesai", `sub_event_scope` "Sesi", `category` "Kategori", `phase_order` "Nomor fase", `phase_name` "Nama fase", `pic` "PIC", `vendor_id` "Vendor", `notes` "Catatan", `status` "Status", `subject_role` "Subjek", `quantity_detail` "Jumlah", `nominal_cost` "Biaya", `days` "Jumlah hari", `ids` "Urutan", `confirm` "Konfirmasi", `preset` "Template", `mode` "Mode". Kolom tak dikenal: "Isian".

### 9.4 Kode `messages.ts` (tambahan)

Kamus status, teks peringatan (`warningText`), label kolom, dan dua fungsi. `lead_time_compressed` dan `lead_time_too_short` sudah ada di 01 FE bab 8.5; salinannya di sini hanya agar modul ini dapat diuji sendiri (satu sumber setelah 01 dibangun: hapus salah satunya). **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-timelinetestmjs)).

```ts
import { formatRupiah } from './format.ts';
import type { DocStatus, TaskStatus } from './types.ts';

// ---------- Tambahan Menu 02 untuk messages.ts (pola: pemanggil memakai `…Message(…) ?? kamus 01`) ----------

/** Aksi yang memanggil: menentukan teks untuk `not_found` dan `rate_limited`, yang tidak dapat dibedakan dari kode/pesan server. */
export type Where = 'project' | 'task' | 'doc' | 'export';

export const TASK_STATUS_LABELS: Record<TaskStatus, string> = { belum: 'Belum', proses: 'Sedang Berjalan', selesai: 'Selesai' };
/** Lencana pada baris tabel (lebih pendek dari nama kolom Kanban). */
export const TASK_STATUS_BADGES: Record<TaskStatus, string> = { belum: 'Belum', proses: 'Berjalan', selesai: 'Selesai' };
export const DOC_STATUS_LABELS: Record<DocStatus, string> = { belum: 'Belum', selesai: 'Selesai' };

/** Tambahan untuk WARNING_LABELS (01). `message` dari server berbahasa Inggris dan tidak pernah ditampilkan. {count} diganti `count`. */
export const TIMELINE_WARNING_LABELS: Record<string, string> = {
  lead_time_compressed: 'Jadwal dimampatkan karena waktu persiapan lebih pendek dari standar.',
  lead_time_too_short: 'Waktu persiapan sangat singkat; semua tugas dijepit ke rentang hari ini sampai Hari-H.',
  scope_fallback: '{count} tugas dipindah ke Umum karena sesinya sudah dihapus.',
  budget_link_lost: '{count} berkas kehilangan tautan ke anggaran; angka di anggaran tidak berubah.',
  estimate_differs: 'Estimasi di anggaran berbeda dari biaya berkas; angka di anggaran tidak diubah.',
  tasks_overdue_after_shift: '{count} tugas kini melewati tenggat setelah digeser.',
};
export const warningText = (code: string, count: number): string =>
  (TIMELINE_WARNING_LABELS[code] ?? 'Ada catatan pada hasil ini.').replace('{count}', String(count));

export const TIMELINE_FIELD_LABELS: Record<string, string> = {
  title: 'Judul tugas', document_name: 'Nama berkas', start_date: 'Tanggal mulai', end_date: 'Tanggal selesai',
  sub_event_scope: 'Sesi', category: 'Kategori', phase_order: 'Nomor fase', phase_name: 'Nama fase', pic: 'PIC',
  vendor_id: 'Vendor', notes: 'Catatan', status: 'Status', subject_role: 'Subjek', quantity_detail: 'Jumlah',
  nominal_cost: 'Biaya', days: 'Jumlah hari', ids: 'Urutan', confirm: 'Konfirmasi', preset: 'Template', mode: 'Mode',
};

type Rule = readonly [RegExp, (m: RegExpExecArray, label: string, field: string) => string | null]; // null = lanjut ke aturan berikutnya

/** Alasan `<kolom>: <alasan>` milik Menu 02 (yang sudah ditangani tabel 01 tidak diulang). Urutan penting: pola khusus lebih dulu. */
const TIMELINE_REASONS: readonly Rule[] = [
  [/^must not be before start_date$/, () => 'Tanggal selesai tidak boleh sebelum tanggal mulai.'],
  [/^must not be after end_date$/, () => 'Tanggal mulai tidak boleh setelah tanggal selesai.'],
  [/^must be within the project schedule window$/, (_m, l) => `${l} harus dalam rentang tiga tahun sebelum sampai setahun sesudah acara.`],
  [/^must keep every shifted task within the project schedule window$/, () => 'Geseran ini membuat sebagian tugas keluar dari rentang jadwal (tiga tahun sebelum sampai setahun sesudah acara). Kurangi jumlah harinya.'],
  [/^must be a vendor of this project$/, () => 'Vendor tidak ditemukan di acara ini. Muat ulang daftar vendor.'],
  [/^must be one of the event type subjects$/, () => 'Subjek tidak valid.'],
  [/^must list every id exactly once$/, () => 'Urutan sudah usang karena daftar berubah. Muat ulang halaman.'],
  [/^must not be zero$/, () => 'Jumlah hari tidak boleh 0.'],
  [/^must be at most 1826 in absolute value$/, () => 'Paling banyak 1.826 hari (5 tahun).'],
  [/^must be greater than 0$/, (_m, l) => `${l} harus lebih dari 0 untuk dikirim ke anggaran.`],
  [/^must be between 0 and (\d+)$/, (m, l, f) => (f === 'nominal_cost' ? `${l} paling banyak ${formatRupiah(Number(m[1]))}.` : null)],  // kolom lain: tabel 01
  [/^cannot be combined with vendor_id$/, () => 'Pilih vendor atau pilih "Tanpa vendor", tidak keduanya.'],
  [/^must be one of /, (_m, l) => `${l} tidak valid.`],
  [/^must be a session scope_key$/, () => 'Sesi tidak valid.'],
];

/** null = bukan alasan Menu 02: pakai fieldErrorMessage 01 (wajib, panjang, tanggal valid, karakter kontrol, angka). */
export function timelineFieldMessage(field: string, reason: string): string | null {
  const label = TIMELINE_FIELD_LABELS[field] ?? 'Isian';
  for (const [re, text] of TIMELINE_REASONS) {
    const m = re.exec(reason);
    const out = m ? text(m, label, field) : null;
    if (out !== null) return out;
  }
  return null;
}

/**
 * Kode galat Menu 02 yang teksnya bergantung konteks. null = pakai plannerErrorMessage 01 (invalid_json, conflict,
 * project_read_only, 401, 5xx, jaringan, rate_limited umum). `not_found` dibedakan oleh AKSI pemanggil (`where`), bukan oleh pesan server.
 */
export function timelineErrorMessage(status: number, code: string | undefined, where?: Where): string | null {
  if (where === 'export' && (status === 429 || code === 'rate_limited')) return 'Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi.';
  switch (code) {
    case 'task_limit_reached': return `Jumlah tugas sudah mencapai batas (500). Hapus tugas yang tidak dipakai lalu coba lagi.`;
    case 'admin_doc_limit_reached': return `Jumlah berkas sudah mencapai batas (200). Hapus berkas yang tidak dipakai lalu coba lagi.`;
    case 'invalid_scope': return 'Sesi tidak valid atau sudah dihapus. Muat ulang halaman.';
    case 'invalid_template': return 'Template tidak tersedia untuk jenis acara ini. Muat ulang halaman.';
    case 'not_found':
      if (where === 'task') return 'Tugas tidak ditemukan. Muat ulang daftar.';
      if (where === 'doc') return 'Berkas tidak ditemukan. Muat ulang daftar.';
      return null;
    default: return null;
  }
}
```

Teks `warnings`: `lead_time_compressed` dan `lead_time_too_short` (Generate), `scope_fallback` (Generate), `budget_link_lost` (Terapkan Template), `estimate_differs` (Kirim ke anggaran), `tasks_overdue_after_shift` (Geser). Kode tak dikenal: "Ada catatan pada hasil ini."; `message` Inggris dari server tidak pernah ditampilkan.

---

## 10. Gaya, responsif, aksesibilitas, keamanan

### 10.1 Gaya

Hanya token yang ada ([01 FE bab 12.1](../01_DASHBOARD/FE.md#121-gaya)): kartu `--color-surface`, batas `--color-border`; judul `--font-display`, isi `--font-body`; lencana mengikuti pola `.status` di `dashboard/index.astro`. Status Gantt dan Kanban: `selesai` terisi `--sage-600`, `proses` terisi `--sage-300`, `belum` hanya kontur `--color-border`; *Terlambat* memakai warna bahaya yang sudah dipakai kode (`#8e2f25`). Setiap status juga **teks** (Selesai / Berjalan / Belum), bukan hanya warna. Callout panduan: `--sage-50` dengan batas `--sage-200`. Jangan menambah token baru tanpa persetujuan.

### 10.2 Responsif

- `< 768px`: Tabel menjadi kartu tugas dengan checkbox besar; Kanban satu kolom per status dengan tab; Gantt menampilkan 4 minggu dengan gulir horizontal dan kolom kiri *sticky*; checklist dokumen menjadi kartu; dialog penuh layar.
- Target sentuh minimal 44×44 px (checkbox, ⋮, pegangan seret, chip).
- Tidak ada gulir horizontal pada lebar 360 px kecuali di dalam wadah Gantt; uji dengan judul 255 karakter dan nama berkas 255 karakter (`overflow-wrap: anywhere`).

### 10.3 Aksesibilitas

- Dialog `<dialog>` dengan `showModal()`; fokus kembali ke pemicu saat ditutup; formulir: `<label>`, `aria-invalid`, `aria-describedby`, ringkasan galat `role="alert"`.
- Seret-lepas selalu punya padanan papan ketik (Naik/Turun, "Pindahkan ke…").
- Chip sesi dan subjek adalah `role="tablist"`/tombol dengan `aria-pressed`; tab Jadwal / Dokumen memakai `role="tablist"` dan `aria-selected`.
- Hasil asinkron (menyimpan, pindah status, hasil generator, "Daftar belum diperbarui") di wilayah `aria-live="polite"`; baris yang sedang dikirim `aria-busy="true"`.
- Partial mengganti DOM: fokus dipulihkan lewat `data-focus-key` (bab 7.6); bila elemennya tak ada lagi (baris dihapus), fokus ke judul daftar, bukan ke `<body>`.
- Gantt: balok punya `aria-label` teks; garis Hari Ini `aria-hidden` dengan tanggalnya di legenda; Gantt dapat dilewati karena Tabel memuat data yang sama.
- Kontras dan indikator fokus terlihat; hormati `prefers-reduced-motion`.

### 10.4 Keamanan

- **Tidak ada HTML dari data:** judul, kategori, PIC, catatan, nama vendor, nama berkas, langkah panduan dirender sebagai teks (`textContent`/`createElement` di skrip klien; Astro meng-escape di server). Blok JSON `data-timeline-data` ditulis dengan `<` di-escape (`\u003c`, bab 7.6) dan dibaca dengan `JSON.parse(textContent)`; tidak ada `set:html` untuk data API.
- Tautan WhatsApp hanya dibentuk dari `vendor_wa` yang cocok `^62\d{6,15}$`, dengan `rel="noopener noreferrer"`.
- Path dibangun hanya dari `id` yang sudah divalidasi UUID; query lewat `URLSearchParams`.
- CSRF: BFF menolak non-GET tanpa `application/json`. Token tetap di cookie HttpOnly. Tidak ada `localStorage` untuk data proyek; `sessionStorage` hanya untuk pilihan tampilan.

---

## 11. Pengujian

Pola yang sudah ada: `node:test` di `frontend/tests/*.test.mjs`, dijalankan `npm test` (`node --experimental-strip-types --test tests/*.test.mjs`). Fixture di `frontend/tests/fixtures/` (bab 4.3).

### 11.1 Unit: `tests/planner-timeline.test.mjs`

Sembilan belas tes di bawah **sudah dijalankan**: kode bab 4.2, 5, 6, dan 9.4 diekstrak langsung dari dokumen ini ke `types.ts`, `timeline.ts`, `timelineApi.ts`, `messages.ts`, bersama `format.ts` dari 01 FE dan fixture asli. Hasil: `tsc` strict (`noUncheckedIndexedAccess`) bersih, 19/19 lulus. Mutasi yang dicoba pada kode (9 dari 9 menjatuhkan tes): `replace_all` memakai aturan `replace_template`; kemajuan tanpa memperhitungkan vendor; `withStatus` tidak menjadikan `overdue` `false`; `whatsappUrl` tanpa pola 62; `taskNumbers` tanpa urut; `moveByOne` selalu naik; `scopeOf` tanpa sesi usang; `docsImpact` tanpa membatasi ke `template_key`; `docsImpact` tanpa berkas selesai.

| Tes | Masukan → keluaran (fixture Proyek Contoh) |
|---|---|
| fixture | 50 tugas: `selesai` 15, `proses` 29, `belum` 6; `summarize` = `summary`; 44 `pdf` + 6 `adicara`; `sort_order` 1–50; 15 berkas (9 selesai); `shiftable_tasks` 35; sesi `all, lamaran, akad, resepsi`; Lamaran 2026-10-15 |
| `filterByScope`, `countByScope`, `scopeOf` | `SCOPE_ANY` → 50; hitungan `{* 50, all 29, lamaran 7, akad 4, resepsi 10}`; `nope` → 0; sesi usang → `all` (dan apa adanya tanpa `known`) |
| `taskNumbers` | 1…50; tahan lubang `sort_order` (kelipatan 3 dan satu dihapus → 1…49); angka di tampilan tersaring = angka global |
| `groupByPhase` (daftar dibalik) | urutan `[1,2,3,4]` berisi 7 / 8 / 20 / 15; nama fase 1 "Persiapan Awal & Lamaran"; isi urut `sort_order`; nama kosong tetap `''`; `[]` → `[]` |
| `phaseNameOf`, `defaultPhase`, `categoriesOf` | fase 3 → "Detail Acara & Perlengkapan"; fase 9 → `''`; bawaan `{4, "Finalisasi & Menuju Hari-H"}`; proyek kosong `{1, ''}`; kategori "Umum" pertama, sisanya urut abjad, tanpa duplikat; `[]` → `['Umum']` |
| `toggledStatus`, `withStatus`, `replaceTask`, `removeTask`, `summarize` | selesai ↔ belum, `proses` → selesai; `withStatus` selesai → `overdue false` dan masukan tidak berubah; ringkasan `{50,16,28,6,0}`; `[]` → semua 0 |
| `moveBefore`, `moveByOne` | `('c','a')` → `c,a,b,d`; `('a', null)` → `b,c,d,a`; `('a','c')` → `b,a,c,d`; `('b','b')`, id asing tak berubah; kelompok yang tersisip tugas fase lain tetap menukar anggota kelompok; ujung → `null`; hasil selalu permutasi lengkap |
| `generatorImpact` | `append` tanpa konfirmasi; sampel: `replace_template` {50 dihapus, 44 berisiko, konfirmasi}; semua "belum": `replace_template` tanpa konfirmasi tetapi `replace_all` **wajib**; tugas kustom tidak ikut `replace_template`; vendor dihitung sebagai kemajuan |
| `docsImpact` | `append` tanpa konfirmasi; sampel `replace_template` {15 dihapus, 9 selesai, 0 tertaut, konfirmasi}; semua belum → tanpa; tertaut → konfirmasi; berkas kustom diabaikan |
| `whatsappUrl` | `6281234567890` → `https://wa.me/6281234567890`; `undefined`, `''`, `0812…`, `+62…`, terlalu pendek/panjang, huruf, spasi → `null` |
| `mondayOf` | `2026-10-04` (Minggu) → `2026-09-28`; `2026-10-05` → dirinya; `2027-03-28` → `2027-03-22`; `2026-02-01` ("1Feb-2026 Minggu" di PDF) → `2026-01-26`; `2024-02-29` → `2024-02-26` |
| `buildGantt(tasks, '2026-10-04')` | `rangeStart 2026-09-28`, `days 182`, 26 minggu (minggu 26 mulai `2027-03-22`); `todayCol 6`; balok tugas 1 `col 6, span 2`; tugas 50 `col 181, span 1`; hari ini `2026-09-01` atau `2027-03-29` → `null`; `[]` → `null`; semua balok dalam rentang; tugas 3 (8 → 10 Okt): `duration_days 2` tetapi `span 3` |
| berkas | biaya 600.000 tanpa tautan → `canSyncBudget true`; biaya 0 → `false`; sudah tertaut → `false`; `filterDocs('bersama')` → 10; `'panitia'` → 0; `replaceDoc` |
| `LIMITS`, kamus | 255 / 60 / 100 / 100 / 2000, 500 tugas, 200 berkas; tiga label status; "Berjalan" vs "Sedang Berjalan" |
| katalog galat | **setiap** `message` berbentuk `<kolom>: <alasan>` di `fixtures/planner-errors.json` punya teks dari `timelineFieldMessage` atau pola 01 (nol alasan tanpa teks; tidak ada kata Inggris bocor); kode `invalid_scope`/`invalid_template` punya teks per kode |
| `timelineFieldMessage` | contoh teks tiap pola 9.2; `title: must not be empty` dan `phase_order: must be between 1 and 20` → `null` (milik 01); kolom/alasan asing → "Isian tidak valid." |
| `timelineErrorMessage` | kode limit; `not_found` per `where`; `rate_limited` hanya untuk `export`; `invalid_json` → `null` |
| `warningText` | `{count}` diganti; kode asing → "Ada catatan pada hasil ini." |
| `timelineApi` | 17 fungsi → metode + path persis (tabel 4.4); `setTaskStatus` body `{status}`; `syncBudget` body bawaan `{}`; id bukan UUID (`../etc`, huruf besar, `abc`) melempar; `exportHref` |

**Tes integrasi ke backend asli** (sekali jalan, tidak masuk `npm test` karena butuh server): ke-17 fungsi `timelineApi` dijalankan beruntun pada proyek baru. Memastikan antara lain: `phase_order` tanpa `phase_name` pada PATCH **tidak** menyalin nama; `moveByOne` → `reorderTasks` menghasilkan urutan yang diminta dan `sort_order` 1…n; `generatorImpact`/`docsImpact` memutuskan persis seperti server (status 400 ↔ `needsConfirm`) pada keadaan nyata; `syncBudget` dua kali → 201 lalu 200 dengan `budget_item_id` sama; `DELETE` kedua → 404; nama berkas ekspor sesuai pola.

### 11.2 Tes sumber (`tests/planner-wiring.test.mjs`, perluasan dari 01 FE bab 13.3)

- Halaman `timeline/index.astro` dan `live.astro` memuat `export const prerender = false`; `live.astro` memuat `export const partial = true`; keduanya memakai komponen yang sama (tidak ada markup daftar di halaman).
- `index.astro` memakai `PlannerLayout`.
- Tidak ada literal sesi atau peran tertulis di `components/planner/timeline` dan halamannya: `'lamaran'`, `'akad'`, `'resepsi'`, `'cpp'`, `'cpw'`, `'ortu_cpp'`, `'ortu_cpw'`, `'panitia'`, `'lembaga'` (hanya `types.ts` dan tes yang boleh). `'all'` dan `SCOPE_ANY` boleh.
- Tidak ada `innerHTML =`, `insertAdjacentHTML`, atau `set:html` yang memuat data API di komponen dan `timelineClient.ts`; tidak ada teks panduan atau label preset tertulis di komponen (kamus ada di `messages.ts`).
- Tidak ada `plannerFetch('projects/` atau `fetch('/dashboard/api/planner` di komponen: semua panggilan lewat `timelineApi`.
- `PlannerLayout.astro` menandai `timeline` sebagai `ready`; `ROUTES.md` memuat rute halaman dan partial.

### 11.3 Uji komponen dengan fixture (manual atau skrip)

Render tiap tampilan dengan: fixture penuh; `tasks: []` (keadaan kosong); satu tugas `overdue`; tugas dengan dan tanpa vendor (`vendor_wa` valid dan tidak valid); `sub_event_scope` yang tidak ada di sesi; judul 255 karakter; Hari-H di luar rentang hari ini; jenis seminar (tanpa kotak Adicara, panduan perizinan, chip Panitia/Lembaga/Umum); `other` (daftar berkas kosong, tanpa menu template); proyek `archived` (semua kontrol ubah nonaktif, ekspor aktif); berkas tertaut ke anggaran; fase tanpa nama.

### 11.4 QA manual (terhadap backend asli)

Bisa dijalankan sekarang (lingkungan di bab 12.1).

1. Proyek Contoh: 50 tugas, progres 30% (15/50), penghitung 15 / 29 / 6 / 0; sesi Lamaran bertanggal 15 Okt 2026 di Menu 01 (hasil [BE §4.6](BE.md#46-sambungan-ke-menu-01)).
2. Chip sesi: Lamaran 7 tugas; Umum 29; Akad 4; Resepsi 10; Semua 50; muat ulang mempertahankan `?scope=`; nilai `?scope=ngawur` diperlakukan Semua.
3. Centang tugas → batang dan penghitung berubah seketika dan tetap setelah partial; widget Timeline di dashboard ikut berubah setelah dimuat ulang; centang lagi → kembali. Dua klik cepat tidak mengirim dua permintaan.
4. Kanban: seret ke kolom lain, lalu matikan jaringan dan seret lagi → kartu kembali dan toast tampil.
5. Gantt: garis Hari Ini di kolom yang benar; tugas 3 berbalok 3 kolom; gulir horizontal; kolom kiri tetap; mobile 4 minggu.
6. Tambah tugas: kolom wajib; tanggal selesai < mulai → pesan di kolom; hapus sesinya dari tab lain lalu simpan → `invalid_scope` dan "Muat ulang halaman"; tambah ke-501 → pesan batas (butuh 500 tugas; boleh lewat API).
7. Ubah fase tugas dari 2 ke 4 tanpa mengetik nama → nama fase baru ikut (FE mengirim keduanya); lihat baris itu masuk kelompok FASE 4 dengan nama yang benar.
8. Urut ulang: Naik/Turun dan seret; nomor "No" berurut walau `sort_order` berlubang (hapus satu tugas dulu); reload mempertahankan urutan.
9. Vendor tertaut (setelah Menu 04): chip nama dan tombol WhatsApp membuka `wa.me/62…`.
10. Generator: `append` dua kali (kedua kali "Semua tugas template sudah ada"); `replace_template` dengan tugas berstatus selesai meminta konfirmasi dan menyebut 44 berisiko; `replace_template` ketika semua "belum" **tidak** meminta konfirmasi; `replace_all` **selalu** meminta konfirmasi bila ada tugas; proyek dengan Hari-H 10 hari lagi → peringatan dimampatkan; Hari-H hari ini/besok → peringatan sangat singkat; Hari-H lampau → pesan khusus; seminar `general` membuat 18 tugas.
11. Geser manual +7: jumlah sama dengan `shiftable_tasks`; hasil `shifted`; geser −40 → peringatan terlambat; geser 5000 → pesan "paling banyak 1.826 hari".
12. Ekspor: berkas `.xlsx` terunduh dengan nama `jadwal-<slug>-<tanggal>.xlsx`, sel diawali `=` tampil sebagai teks; permintaan ke-11 dalam semenit → pesan batas.
13. Tab Dokumen: 9 / 15; enam chip subjek; centang; Kirim ke anggaran pada Pendaftaran KUA → badge "✓ Di anggaran" dan pos Biaya KUA di Menu 03 (setelah ada); klik ganda cepat tetap satu pos; Ganti template meminta konfirmasi bila ada berkas selesai dan menyebut "tautan anggaran lepas" bila tertaut; berkas kustom tidak hilang.
14. Seminar: preset `general`, tanpa kotak Adicara, panduan perizinan, chip Panitia, Lembaga/Instansi, Umum; berkas baru tanpa subjek berjenis `umum`; jenis `other`: tanpa menu template berkas tetapi boleh menambah berkas.
15. Arsipkan: kontrol ubah nonaktif, ekspor aktif; tulis dari tab lain → `project_read_only` dan kontrol menjadi nonaktif.
16. Dua tab: hapus tugas di tab A, centang tugas itu di tab B → "Tugas tidak ditemukan. Muat ulang daftar." lalu daftar menyegarkan diri.
17. Sesi habis lalu klik aksi → diarahkan ke login (bukan layar kosong atau JSON).
18. Mobile 360 px: tidak ada gulir horizontal di luar Gantt; dialog muat; sentuh nyaman.
19. Papan ketik saja: seluruh alur (centang, urut, ubah, generator, berkas) dapat dijalankan; fokus terlihat dan kembali ke pemicu.
20. Pembaca layar ringkas: status, hasil generator, dan galat terbaca.

### 11.5 Verifikasi kontrak terhadap backend asli (cara mengulang)

Dilakukan 2026-10-21 terhadap build backend cabang `backend/planner` dan PostgreSQL 16.2 sementara (database `*_test`); ulangi setiap kali backend Menu 02 atau registry berubah, dan setelah Menu 03 menambah kolom pada pos anggaran yang disentuh `sync-budget`.

1. **Jalankan** backend pada database sementara (bab 12.1), daftarkan satu akun, buat Proyek Contoh.
2. **Panggil** setiap baris tabel 4.4 dengan body 4.5, ditambah kasus galatnya, dan simpan `status` + badan JSON: 120 panggilan, 119 berkas respons (respons login dibuang karena memuat token), ditambah POST pengisi untuk mencapai batas 500/200. Kasus yang dicakup: semua 17 endpoint; keempat filter `GET tasks` (`status`, `scope`, `phase`, `category`); POST dengan isian minimal dan penuh; PATCH tiap kolom, `clear_vendor`, `{}`; semua mode Generate (`append` berulang, `replace_*` dengan dan tanpa `confirm`, preset tidak valid, mode tidak valid, `preset` kosong); geser +, −, 0, 5000; reorder lengkap/kurang/ganda; berkas (subjek, biaya, status, sinkron tiga hasil, terapkan template tiga mode); proyek diarsipkan; jenis seminar, `other`, Hari-H 10 hari, Hari-H hari ini; 500 tugas dan 200 berkas (batas); tanpa token; proyek asing.
3. **Bandingkan** setiap objek JSON dengan tipenya: parser mengambil `interface` dari blok kode bab 4.2 dan 01 FE bab 4.2 (komentar dibuang, `?` = opsional, `T | null`, `T[]`, literal union, objek bersarang, `Record`), lalu untuk tiap objek melaporkan: kunci di JSON tetapi tidak di tipe, kunci wajib hilang, `null` pada tipe yang tidak memuat `null`, nilai di luar enum/literal, tipe primitif salah. Respons galat dibandingkan dengan `ApiError`.
4. **Uji diri pemeriksa**: ubah tipe dengan sengaja dan pastikan pemeriksa melapor. Lima mutasi dicoba dan semuanya dilaporkan: `vendor_id` dijadikan wajib, kunci `overdue` dibuang, literal `'proses'` dibuang dari `TaskStatus`, kunci wajib karangan ditambahkan ke `AdminDoc`, `shifted` diganti `string`.
5. **Cocokkan fixture**: bangun ulang 50 tugas dari tabel BE §5 lewat `Plan` dan bandingkan dengan respons (setelah menggeser tanggal ke baseline 4 Okt): harus identik.

Hasil 2026-10-21: 821 objek, **0 selisih** (rincian di bab 4.2). Yang **tidak** terbukti oleh panggilan nyata: bentuk `vendor_id`/`vendor_name`/`vendor_wa` pada tugas (Menu 04 belum ada; dibuktikan tes Go), hasil `sync-budget` dengan `linked: true` (butuh pos anggaran Menu 03), `409 conflict` (tabrakan kunci), dan galat `5xx`.

---

## 12. Urutan pengerjaan dan definisi selesai

### 12.1 Menjalankan lingkungan lokal

```bash
# Backend (butuh backend/.env: DB_*, JWT_SECRET; migrasi otomatis saat start). Gunakan database lokal/sementara, jangan produksi.
cd backend && make run                          # :8080  (lihat backend/README.md)

# Frontend
cd frontend && npm install
API_BASE_URL=http://localhost:8080 npm run dev  # :4321; daftar akun di /auth/daftar lalu masuk di /auth/masuk
```

Data uji: Proyek Contoh dari `/dashboard/planner` ("Lihat Proyek Contoh") sudah berisi 50 tugas dan 15 berkas karena modul Timeline terdaftar di backend. Contoh dibuat **hari ini** dengan Hari-H +175 hari, jadi tanggal tugasnya bergeser dari fixture (yang dibaku pada 4 Okt 2026) tetapi ukurannya sama persis. Untuk mencoba keadaan lain tanpa antarmuka (jenis seminar, Hari-H dekat, 500 tugas), panggil API langsung:

```bash
B=http://localhost:8080
TOKEN=$(curl -s $B/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"…","password":"…"}' | jq -r .tokens.access_token)
curl -s $B/v1/planner/projects -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{
  "event_type":"seminar","title":"Seminar Uji","event_date":"2026-12-01","metadata":{"organizer_name":"Komunitas"},
  "client_token":"uji-seminar-01","templates":{"timeline_preset":"general","admin_docs":true}}'
```

`metadata` wajib memuat kolom wajib jenis itu (`organizer_name` untuk seminar; daftar di registry). Hasil `GET /v1/planner/event-types` adalah sumber kebenaran untuk preset dan subjek.

### 12.2 Tahapan

| # | Tahap | Isi | Selesai bila |
|---|---|---|---|
| T0 | Prasyarat | Menu 01 FE: M0 (tipe, `format`, `messages`), M1 (BFF + `client`), `PlannerLayout`, `404.astro` | `npm test` 01 hijau; `GET /dashboard/api/planner/event-types` membalas |
| T1 | Fondasi menu ini | blok bab 4.2 ke `types.ts` (+ satu baris `EventTypeInfo extends …`), `timeline.ts` (bab 5), `timelineApi.ts` (bab 6.1), blok `messages.ts` (bab 9.4), `tests/fixtures/*` (bab 4.3), tes 11.1 | `npm test` dan `tsc` hijau; belum perlu backend. Kode di dokumen ini tinggal disalin: sudah lulus kedua pemeriksaan itu |
| T2 | Halaman, Tab Dokumen, Tabel | `index.astro` + `live.astro`, `TimelineHeader`, `ScopeChips`, `TaskTable`, `DocChecklist`, `GuideCallout`, `DocModal`, `timelineClient.ts` (alur A, I, J, K) | uji komponen 11.3 (kecuali Kanban/Gantt); QA 1–3 dan 13 |
| T3 | Kanban + Gantt | `KanbanBoard`, `GanttMatrix`, `ViewSwitcher` | QA 4–5 |
| T4 | Modal dan aksi | `TaskModal` (alur D, E), `GeneratorModal` (F), `ShiftDialog` (G), urut ulang (C), tanggal inline (B), ekspor (H), Terapkan Template (L), berkas (M) | QA 6–12 |
| T5 | Kualitas | aksesibilitas, responsif, `astro check`, `npm run build`, `format:check`, tes sumber 11.2 | QA 14–20; build bersih |

T1 dan sebagian T2–T3 dapat berjalan terhadap fixture; yang butuh backend sudah dapat dijalankan karena BE menu ini ada ([BE bab 7](BE.md#7-urutan-pengerjaan-dan-definisi-selesai)).

**Definisi selesai (seluruh Menu 02 FE):** semua tahap lulus; `npm test`, `npm run check`, `npm run build`, `npm run format:check` bersih; tes sumber 11.2 lulus; verifikasi kontrak 11.5 diulang bila backend berubah (nol selisih); `timeline.ready = true` di `PlannerLayout`; `ROUTES.md`, `docs/frontend/planner/` dan dokumen ini diperbarui bila perilaku berubah.

---

## 13. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01 (kode dan dokumen), kontrak, PDF, SQL yang dijalankan, dan **backend yang berjalan**. Dokumen mengikuti pilihan yang tertulis.

1. **Kontrak sudah dibuktikan, dengan tiga pengecualian** (bab 11.5): `vendor_*` pada tugas, `sync-budget` `linked: true`, dan `409 conflict` belum teramati pada backend yang berjalan (masing-masing menunggu Menu 04, Menu 03, dan tabrakan kunci). Tipenya mengikuti BE dan tes Go.
2. **Daftar vendor untuk dropdown.** Bentuk respons `GET …/vendors` ([04 BE §3.1](../04_VENDOR_DAN_BELANJA/BE.md#31-vendor)) belum ditetapkan di level yang dibutuhkan (`id`, `vendor_name`); kolom Vendor disembunyikan sampai ada. Tautan tugas → vendor pada Proyek Contoh dibuat overlay Menu 04.
3. **Label preset `general` tanpa angka.** `TEMPLATE_LABELS` 01 FE semula "Jadwal umum (15 tugas)", padahal `general` menghasilkan **18** tugas untuk khitanan, aqiqah, peresmian, launching, ulang tahun, wisuda, dan seminar (15 + 3 khusus jenis) dan 15 hanya untuk `other` ([BE §5.5](BE.md#55-template-jenis-non-pernikahan-usulan-adicara)). Angka dibuang dan 01 FE bab 8.4 disesuaikan ("Jadwal standar jenis acara"). Bila produk ingin angka, registry perlu mengirim jumlah per jenis.
4. **Ubah fase lewat PATCH tidak menyalin nama fase** (POST menyalin). Akibatnya satu tugas bisa membawa `phase_order 4` dengan `phase_name` fase lama. FE mengirim keduanya (bab 7.5) sehingga tidak terjadi; bila produk ingin server yang menyalin, itu perubahan BE kecil ([BE bab 8, butir 20](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
5. **Satu sumber render dengan partial** (bab 2.4) belum dijalankan di browser. Alternatif sah: `location.reload()` sesudah tiap aksi tulis (lebih kasar, logika render tetap SSR). Memutuskan sebelum membangun T2.
6. **Ekspor sebagai tautan** (teks 429 kurang ramah, tanpa `Retry-After`) atau `fetch` + Blob; pilih saat membangun.
7. **Gantt hanya baca.** Menyeret balok untuk mengubah jadwal tidak ada; mengubah tanggal lewat tabel atau modal.
8. **Nama fase per tugas.** `phase_name` disimpan di tiap tugas dan tidak ada entitas fase, sehingga mengganti nama satu fase berarti mengubah tiap tugas di dalamnya. Belum ada UI "ganti nama fase" (butuh endpoint massal bila diinginkan).
9. **Kategori tugas** tidak punya daftar di registry; dropdown diisi kategori yang sudah dipakai di proyek, jadi proyek tanpa tugas hanya punya "Umum" dan isian bebas.
10. **Progres tidak per sesi.** Penyaring sesi tidak mengubah header ([BE bab 8, butir 10](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
11. **Sinkron biaya sekali jalan**, tanpa "lepas" dan tanpa sinkron dua arah ([BE bab 8, butir 9](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
12. **"Berjalan: 29" pada Proyek Contoh** setia pada PDF tetapi janggal untuk tugas yang baru mulai November (BE bab 8, butir 7).
13. **Hari ini pada Gantt** dihitung saat render dan tidak bergeser melewati tengah malam tanpa muat ulang.
14. **Kinerja.** Server: 500 tugas = 370 kB JSON, kueri 0,2 ms ([BE §3.4](BE.md#34-validasi-dan-batas)). Browser belum diukur: ketiga tampilan dirender sekaligus (bab 2.4), jadi 500 tugas ≈ 1.500 baris/kartu/balok ditambah blok JSON data; rentang Gantt sampai ±4 tahun. Bila lambat: render tampilan non-aktif secara malas (minta partial per tampilan).
15. **Seret-lepas dibatasi satu fase**; memindahkan lintas fase hanya lewat modal.
16. **Tidak ada kontrol konkurensi di kolom yang sama.** Dua tab yang mengubah kolom *berbeda* aman (server menggabung di bawah kunci); mengubah kolom *sama* adalah "terakhir menang". Tidak ada ETag/`If-Match`; tidak ada polling di menu ini, jadi data tab lain usang sampai ada aksi (galat `not_found`/`invalid_scope`/`ids` memicu muat ulang area).
17. **Nama sesi sistem** dikirim server sebagai "Umum / Semua Sesi"; chip memakai teks tetap "Umum" supaya konsisten dengan dropdown modal. Bila produk ingin nama dari server, ubah satu baris di `ScopeChips`.
18. **Berkas ekspor** belum pernah dibuka di Excel/Numbers/LibreOffice; awalan `'` pada sel berawalan `=+-@` tampak sebagai karakter ([BE bab 8, butir 12](BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 14. Riwayat perubahan dokumen ini

**2026-10-21 (rincian untuk pengembang FE lain, dicocokkan dengan backend asli):** tipe bab 4.2 dibandingkan dengan 821 objek dari 120 panggilan (nol selisih) dan fixture diganti salinan respons asli (`fixtures/`); ditambah katalog endpoint, contoh respons, katalog galat (bab 4.4–4.6), aturan JSON yang diamati (4.1), kontrak props (1.4), keadaan URL (2.3), arsitektur satu-sumber-render (2.4), `timelineApi.ts` (6.1), alur interaksi dan atribut data (7.6), salinan teks (7.7, 8.3), `messages.ts` yang diuji (9.4), dan lingkungan lokal (12.1). **Koreksi terhadap versi sebelumnya:** (1) aturan konfirmasi `replace_all` salah (server meminta konfirmasi bila ada tugas satu pun, bukan hanya bila ada kemajuan); (2) label preset `general` "15 tugas" salah untuk tujuh jenis (18 tugas); (3) PATCH `phase_order` tidak menyalin nama fase; (4) alasan galat yang tidak terpetakan (`must not be after end_date`, `must keep every shifted task…`, `must be greater than 0`, `cannot be combined with vendor_id`, `must be one of …`, `must be a session scope_key`); (5) wireframe berkas menampilkan baris yang belum dikirim sebagai sudah di anggaran; (6) mekanisme render dan pembaruan sesudah aksi tidak ditetapkan; fixture hanya tiga baris contoh (kini 50 tugas dan 15 berkas penuh). Ditambahkan juga makna `count` pada peringatan (tugas yang dibuat) dan bahwa `lead_time_too_short` menggantikan `lead_time_compressed`. Rincian di [CHANGELOG spesifikasi](../CHANGELOG.md).

**2026-10-21 (awal):** FE Menu 02 ditulis mengikuti pola 01 FE sebelum backend ada, dengan 10 tes unit atas fixture buatan.
