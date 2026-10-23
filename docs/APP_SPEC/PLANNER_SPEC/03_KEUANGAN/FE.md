# Menu 03: Keuangan & Anggaran — Frontend Specification

Acuan pembangunan antarmuka tiga tab: **Simulasi & Alokasi Budget** → **Kas & Tabungan Acara** → **Realisasi Anggaran & Pembayaran** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: kontrak data, aturan tampilan, algoritma, alur interaksi, kasus galat, dan pengujian ada di sini. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); fondasi yang dibangun Menu 01 (tipe, `format.ts`, `messages.ts`, `client.ts`, BFF, `PlannerLayout`) **dipakai apa adanya** dan dirujuk ke [01 FE](../01_DASHBOARD/FE.md), bukan diulang. Pola dokumen ini mengikuti [02 FE](../02_TIMELINE_DAN_TUGAS/FE.md).

> **Status (2026-10-21):** frontend Planner belum ada satu baris pun. **Backend menu ini sudah ada** ([BE.md](BE.md); 32 endpoint; [dokumen implementasi](../../../backend/planner/2026-10-21-planner-menu-03.md); kontrak `backend/docs/openapi.yaml`, tag `Planner Finance`). Kontrak di bab 4 dicocokkan dengan **struct Go yang sebenarnya** (`backend/internal/modules/plannerfinance/domain/entity.go`, isinya sama dengan [BE bab 5](BE.md#5-dto-go)): skema JSON diambil dengan `reflect` dan dibandingkan kunci demi kunci dengan tipe TypeScript bab 4.2: 29 pasangan tipe, 329 jalur JSON, **nol selisih** pada kunci, opsionalitas, `null`, dan jenis nilai (pembanding diuji dengan enam mutasi tipe dan melaporkan semuanya); skema `openapi.yaml` yang sama dijaga tes backend. Yang **belum** dilakukan seperti 02 FE: mencocokkan dengan **respons HTTP asli**. Fixture ([bab 4.3](#43-fixture)) masih dibangun dari view `planner_budget_lines` asli dan SQL spesifikasi terhadap data emas PDF di PostgreSQL 16.2 sementara, bukan disalin dari endpoint (angkanya sama dengan respons uji emas backend, yang tidak mengandung id acak); ganti dengan salinan respons asli lalu ulangi [bab 12.5](#125-verifikasi-kontrak-cara-mengulang). Menu ini bergantung pada Menu 01 FE (M0–M1 dan `PlannerLayout`, [bab 3](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-fe)).
>
> Kode di bab 4.2, 5, 6, 10.4, dan 1.4 **sudah dijalankan**: `tsc` strict (dengan `noUncheckedIndexedAccess`, konfigurasi Astro yang dipakai repo) bersih; 15 tes `node:test` ([bab 12](#12-pengujian)) lulus terhadap fixture di atas; 19 dari 20 mutasi kode yang dicoba menjatuhkan tes (satu mutan setara: teks cadangan "Peran tidak valid." sama bermaknanya); ke-32 pembungkus `financeApi` diuji metode, path, dan body-nya. Yang bergantung pada Astro atau browser (halaman, partial, dialog, grafik SVG, unduhan) belum diuji dan ditandai *(belum diuji di browser)*.

> Angka pada wireframe berasal dari **Proyek Contoh** (dataset PDF Hal. 4–7) yang dibuat 4 Okt 2026: Sinta & Arif, 28 Maret 2027, H-175. Tanggal sesi Lamaran (15 Okt 2026) adalah hasil penskalaan jadwal Menu 02 ([02 BE §4.1](../02_TIMELINE_DAN_TUGAS/BE.md#41-generator-jadwal)), jadi bergeser bila contoh dibuat pada hari lain; angka rupiah tidak bergeser. Rekomendasi menabung "6 bulan" dihitung dari jarak ke sesi terakhir dan tetap 6 selama sisa waktu 151–180 hari. Wireframe adalah tata letak, bukan desain visual akhir: warna, jarak, dan tipografi memakai token yang sudah ada (bab 11.1). Emoji pada wireframe untuk **sesi** adalah emoji dari API; ikon **kontrol** memakai Phosphor dan tidak digambar.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat](#2-alur-data-dan-penanganan-galat)
3. [Prasyarat yang harus sudah ada dari Menu 01 FE](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-fe)
4. [Kontrak data: tipe, fixture, contoh respons, katalog galat](#4-kontrak-data-tipe-fixture-contoh-respons-katalog-galat)
5. [Library `lib/planner/finance.ts`](#5-library-libplannerfinancets)
6. [Pembungkus panggilan, BFF, dan unduhan](#6-pembungkus-panggilan-bff-dan-unduhan)
7. [Tab 1: Simulasi & Alokasi Budget](#7-tab-1-simulasi--alokasi-budget)
8. [Tab 2: Kas & Tabungan Acara](#8-tab-2-kas--tabungan-acara)
9. [Tab 3: Realisasi Anggaran & Pembayaran](#9-tab-3-realisasi-anggaran--pembayaran)
10. [Pesan galat dan teks](#10-pesan-galat-dan-teks)
11. [Gaya, responsif, aksesibilitas, keamanan](#11-gaya-responsif-aksesibilitas-keamanan)
12. [Pengujian](#12-pengujian)
13. [Urutan pengerjaan dan definisi selesai](#13-urutan-pengerjaan-dan-definisi-selesai)
14. [Celah dan keputusan yang menunggu pemilik produk](#14-celah-dan-keputusan-yang-menunggu-pemilik-produk)
15. [Riwayat perubahan dokumen ini](#15-riwayat-perubahan-dokumen-ini)

---

## 1. Route, file, dan perubahan pada kode yang sudah ada

### 1.1 File baru

| Halaman / berkas | Route | File | Fungsi |
|---|---|---|---|
| Keuangan & Anggaran | `/dashboard/planner/[id]/keuangan` | `src/pages/dashboard/planner/[id]/keuangan/index.astro` | tiga tab lewat `?tab=simulasi` (default), `?tab=tabungan`, `?tab=realisasi` (alias `?tab=anggaran`); nilai lain jatuh ke `simulasi`. Tab dipilih server-side; bagian tab lain tidak dimuat. Parameter lain di bab 2.3 |
| Partial | `/dashboard/planner/[id]/keuangan/live` | `src/pages/dashboard/planner/[id]/keuangan/live.astro` (`export const partial = true`) | HTML area yang diperbarui setelah aksi tulis (bab 2.4); menerima `?tab=` dan parameter yang sama dengan halaman. Satu sumber render: memakai komponen yang sama dengan `index.astro` |
| Komponen | – | `src/components/planner/finance/*.astro` | lihat 1.2 |
| Library | – | `src/lib/planner/finance.ts` (murni, bab 5), `financeApi.ts` (pembungkus panggilan, bab 6), `financeProps.ts` (kontrak props, bab 1.4), `financeClient.ts` (skrip klien halaman, bab 7.5, 8.5, 9.5) | |
| Tes + fixture | – | `tests/planner-finance.test.mjs`, `tests/fixtures/planner-{budget-lines,matrix,cashflow,savings,scenarios}.json` (+ `planner-sessions.json` dari Menu 02) | bab 4.3 dan 12 |

Tidak ada file BFF baru: semua panggilan lewat `/dashboard/api/planner/[...path].ts` dari Menu 01 (bab 6). Halaman memakai `PlannerLayout` (bilah sekunder berisi Project Switcher dan sub-navigasi), **bukan** `AppLayout` langsung (versi awal dokumen ini menulis `AppLayout` dan `nav_active`, yang tidak ada di desain 01 FE bab 10).

### 1.2 Komponen

Props lengkap (TypeScript) di bab 1.4. Kolom "Klien" menyebut perilaku yang dipasang skrip halaman; alurnya di bab 7.5, 8.5, dan 9.5.

| Komponen | Props utama | Dipakai di | Klien |
|---|---|---|---|
| `FinanceTabs` | `active` | tab bar | navigasi penuh (`?tab=`) |
| `ScenarioSwitcher` | `scenarios`, `selectedId`, `canAdd`, `readOnly` | Tab 1 | pilih skenario (`?scenario=`), `[+]`, menu ganti nama / duplikasi / hapus |
| `SubEventSelector` | `sessions`, `scenario`, `readOnly` | Tab 1 | kotak centang sesi → `updateScenario`; centang terakhir tidak bisa dilepas |
| `BudgetPresetBar` | `presets`, `scenario`, `readOnly` | Tab 1 | tombol paket → `loadPreset`; isian target + slider → `updateScenario` |
| `BudgetDonutChart` | `groups`, `shares`, `total` | Tab 1 | tooltip nominal dan persen (SVG murni) |
| `ScenarioItemsTable` | `groups`, `shares`, `unallocated`, `readOnly` | Tab 1 | ubah nama/nominal inline, hapus, "kembalikan ke nilai preset", tambah pos |
| `ScenarioItemModal` | `sessions`, `categories`, `defaultScope` | Tab 1 | tambah pos skenario |
| `ScenarioComparisonModal` | `scenarios` | Tab 1 | pilih dua skenario → `compareScenarios` |
| `ApplyScenarioModal` | `scenario`, `project` | Tab 1 | `previewApply` → konfirmasi → `applyScenario` |
| `SavingsKpiCards` | `overview`, `shares`, `readOnly` | Tab 2 | edit target (`setSavingsTarget`) |
| `SavingsAdvice` | `overview` | Tab 2 | rekomendasi bulanan dan milestone |
| `ContributorCards` | `contributors`, `shares`, `roles`, `readOnly` | Tab 2 | ubah / hapus kontributor |
| `ContributorModal` | `roles` | Tab 2 | tambah / ubah kontributor |
| `SavingsMatrix` | `overview`, `readOnly` | Tab 2 | klik sel → `cellAction` |
| `SavingsLedger` | `overview`, `filterId`, `readOnly` | Tab 2 | chip penyetor (`?contributor=`), ubah / hapus setoran |
| `SavingsEntryModal` | `contributors`, `roles`, `today` | Tab 2 | tambah / ubah setoran; `[+ Daftarkan baru]` |
| `SavingsTrendChart` | `overview` | Tab 2 | garis kumulatif, garis target, penanda sesi (SVG murni) |
| `CashflowBar` | `cashflow` | Tab 3 | – |
| `BudgetKpiCards` | `matrix`, `columns` | Tab 3 | – |
| `SessionMatrix` | `columns`, `total` | Tab 3 | gulir horizontal, kolom kiri *sticky* |
| `BudgetTable` | `lines`, `numbers`, `sessions`, `readOnly` | Tab 3 | chip sesi, bayar / ubah / hapus, Naik/Turun, seret |
| `PaymentDrawer` | `lines`, `today` | Tab 3 | pratinjau `paymentImpact`, bayar, hapus tahap |
| `BudgetItemModal` | `sessions`, `categories`, `vendorsEnabled`, `rsvpEnabled` | Tab 3 | tambah / ubah pos; harga satuan; "Gunakan proyeksi RSVP" |
| `TemplateMenu` | `type`, `project`, `lineCount` | Tab 3 | tambah pos yang belum ada / ganti pos template (`templateImpact`) |
| `ConfirmDialog` | `title`, `message`, `confirmLabel`, `danger` | semua | satu `<dialog>` dipakai bersama (dari 02 FE; bila belum ada, buat di sini) |

### 1.3 Perubahan pada file yang sudah ada

| File | Perubahan | Mengapa |
|---|---|---|
| `src/layouts/PlannerLayout.astro` | `MENUS`: `keuangan.ready = true` | satu bendera; menu lain tidak berubah ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) |
| `src/lib/planner/types.ts` | tambah blok bab 4.2; ubah `export interface EventTypeInfo {` menjadi `… extends EventTypeFinanceInfo {` (bila Menu 02 FE sudah ada: `extends EventTypeAdminInfo, EventTypeFinanceInfo`); pada `UpdateProjectBody` tambah `savings_target?: number; clear_savings_target?: boolean` | kontrak menu ini. Backend Menu 01 **sudah** menerima kedua kolom itu (`UpdateProjectRequest`), tetapi blok tipe 01 FE bab 4.2 belum memuatnya |
| `src/lib/planner/messages.ts` | tambah blok bab 10.4 (label status, teks peringatan, alasan per kolom, galat berkonteks) | teks galat |
| `src/lib/api/backend.ts` dan BFF | `backendFetchRaw` dan aliran berkas untuk `…/budget/export` | satu-satunya respons non-JSON; sama dengan perubahan 02 FE bab 1.3 (kerjakan sekali) |
| `ROUTES.md` | baris halaman `/dashboard/planner/[id]/keuangan` dan partial `…/keuangan/live` | file itu minta diperbarui bersama setiap halaman baru |
| `docs/frontend/planner/` | dokumen bertanggal dan entri `CHANGELOG.md` | aturan repo: kode dan dokumen satu paket |
| (tidak diubah) | `AppLayout.astro`, `middleware.ts`, `tokens.css`, `global.css` | |

`astro check` (bagian dari `npm run build`) harus lolos; TypeScript `strict` dan `noUncheckedIndexedAccess` aktif (akses `array[i]` bertipe `T | undefined`).

### 1.4 Kontrak props

Disalin ke `interface Props` tiap komponen (atau diimpor dari `financeProps.ts`); **sudah dikompilasi** `tsc` strict bersama bab 4.2 dan 5. `ScenarioGroup` dan `MatrixColumn` dari bab 5; objek biasa (bukan `Map`) untuk `shares` dan `numbers` karena props SSR tidak membawa `Map`: `Object.fromEntries(scenarioShares(sc))`, `Object.fromEntries(lineNumbers(lines))`.

```ts
import type { MatrixColumn, ScenarioGroup } from './finance.ts';
import type {
  BudgetLine, BudgetMatrix, CashflowSummary, Contributor, ContributorRole, EventTypeInfo, PresetInfo, SavingsOverview,
  Scenario, SessionBrief,
} from './types.ts';

export type FinanceTab = 'simulasi' | 'tabungan' | 'realisasi';
export type SavingsView = 'matriks' | 'daftar';

/** Konteks proyek yang dibaca halaman dari GET projects/:id dan diteruskan ke komponen (bukan dibaca ulang). */
export interface FinanceProject {
  id: string; timezone: string; eventDate: string; archived: boolean;
  targetBudget: number; savingsTarget: number | null; eventTypeKey: string;
}

export interface ScenarioSwitcherProps { scenarios: Scenario[]; selectedId: string; canAdd: boolean; readOnly: boolean }
export interface SubEventSelectorProps { sessions: SessionBrief[]; scenario: Scenario; readOnly: boolean }
export interface BudgetPresetBarProps { presets: PresetInfo[]; scenario: Scenario; readOnly: boolean }
export interface BudgetDonutChartProps { groups: ScenarioGroup[]; shares: Record<string, number | null>; total: number }
export interface ScenarioItemsTableProps { groups: ScenarioGroup[]; shares: Record<string, number | null>; unallocated: number; readOnly: boolean }
export interface ScenarioItemModalProps { sessions: SessionBrief[]; categories: string[]; defaultScope: string }
export interface ScenarioComparisonModalProps { scenarios: Scenario[] }
export interface ApplyScenarioModalProps { scenario: Scenario; project: FinanceProject }
export interface SavingsKpiCardsProps { overview: SavingsOverview; shares: Record<string, number | null>; readOnly: boolean }
export interface SavingsAdviceProps { overview: SavingsOverview }
export interface ContributorCardsProps { contributors: Contributor[]; shares: Record<string, number | null>; roles: ContributorRole[]; readOnly: boolean }
export interface ContributorModalProps { roles: ContributorRole[] }
export interface SavingsMatrixProps { overview: SavingsOverview; readOnly: boolean }
export interface SavingsLedgerProps { overview: SavingsOverview; filterId: string | null; readOnly: boolean }
export interface SavingsEntryModalProps { contributors: Contributor[]; roles: ContributorRole[]; today: string }
export interface SavingsTrendChartProps { overview: SavingsOverview }
export interface CashflowBarProps { cashflow: CashflowSummary }
export interface BudgetKpiCardsProps { matrix: BudgetMatrix; columns: MatrixColumn[] }
export interface SessionMatrixProps { columns: MatrixColumn[]; total: BudgetMatrix['total'] }
export interface BudgetTableProps { lines: BudgetLine[]; numbers: Record<string, number>; sessions: SessionBrief[]; readOnly: boolean }
export interface PaymentDrawerProps { lines: BudgetLine[]; today: string }
export interface BudgetItemModalProps { sessions: SessionBrief[]; categories: string[]; vendorsEnabled: boolean; rsvpEnabled: boolean }
export interface TemplateMenuProps { type: EventTypeInfo; project: FinanceProject; lineCount: number }
```

`FinanceProject` dibentuk satu kali di frontmatter halaman dari `GET projects/:id`: `archived = detail.project.status === 'archived'`, `targetBudget = project.target_budget`, `savingsTarget = project.savings_target`, `timezone`, `eventDate`, `eventTypeKey = project.event_type`. `readOnly` pada komponen = `project.archived`.

---

## 2. Alur data dan penanganan galat

```text
SSR (baca)    : halaman → backendFetch('/v1/planner/…', { accessToken }) → HTML
Interaksi     : <script> klien → financeApi(plannerFetch, id) → BFF /dashboard/api/planner/… → Go API
Pembaruan     : sesudah aksi tulis → fetch('/dashboard/planner/:id/keuangan/live?tab=…') → HTML partial → ganti [data-finance-live]
Unduhan       : <a href="/dashboard/api/planner/projects/:id/budget/export?format=xlsx"> → BFF (aliran berkas)
Registry      : lib/planner/eventTypes.ts → GET /v1/planner/event-types (cache memori proses 1 jam)
```

`backendFetch` mengembalikan seluruh payload di `result.data`, jadi isi respons ada di `result.data.data`.

### 2.1 Pemuatan per tab

| Tab | Panggilan (paralel dengan `Promise.all`) | Catatan |
|---|---|---|
| `simulasi` | `GET projects/:id/scenarios` + `GET projects/:id/sessions` + `GET projects/:id` + registry | skenario terpilih = `pickScenario(list, ?scenario=)`; `sessions` memberi chip, kotak centang, dan label kelompok (termasuk sesi sistem `all`: chip memakai teks tetap "Umum"); `GET projects/:id` memberi `status` (arsip), `target_budget`, `event_type`. Kategori untuk modal pos = kategori yang sudah dipakai di pos skenario (registry tidak punya daftar kategori anggaran, bab 14 butir 5) |
| `tabungan` | `GET projects/:id/savings` (+ `?contributor_id=` bila `?contributor=`) + `GET projects/:id` + registry | peran dari registry (`contributor_roles`); sesi tidak perlu dimuat: milestone sudah membawa nama dan tanggal sesi |
| `realisasi` | `GET projects/:id/budget/matrix` + `GET projects/:id/budget/cashflow-summary` + `GET projects/:id/budget/items` + `GET projects/:id/sessions` + `GET projects/:id` + registry | daftar vendor untuk dropdown dimuat **saat modal pos dibuka** (`GET projects/:id/vendors` → `VendorOverview.vendors[]`: `id`, `vendor_name`, `category_label`; [04 BE §3.1](../04_VENDOR_DAN_BELANJA/BE.md#31-vendor), [04 FE bab 4.2](../04_VENDOR_DAN_BELANJA/FE.md#42-tambahan-pada-srclibplannertypests)); `404`/galat atau Menu 04 belum ada → kolom Vendor disembunyikan. Proyeksi RSVP dimuat saat tombolnya ditekan (`GET projects/:id/guests/catering-projection`, [05 BE §2.3](../05_BUKU_TAMU/BE.md)); tidak tersedia → tombol disembunyikan (`rsvpEnabled = false`) |

Validasi `Astro.params.id` seperti 01 FE bab 2.1 (UUID huruf kecil, selain itu `Astro.rewrite('/404')` tanpa memanggil backend).

### 2.2 Hasil backend → tindakan halaman

| Hasil | Tindakan |
|---|---|
| `2xx` | render |
| `401` | `Astro.redirect('/auth/masuk')` |
| `404 not_found` pada proyek | `Astro.rewrite('/404')` (tidak membedakan "bukan milik Anda" dan "tidak ada") |
| `5xx` atau `BackendUnavailableError` | kerangka halaman + banner "Keuangan sedang tidak tersedia" + tombol "Muat ulang"; **bukan 404** |
| satu panggilan tab gagal, proyek berhasil | tab tetap tampil; bagian yang gagal diganti banner galat lokal dengan tombol "Muat ulang" (mis. matriks gagal, tabel pos tetap tampil); tab lain tidak ikut gagal |
| Registry gagal | pakai cache lama; bila tidak ada, dropdown peran diganti isian teks tidak aktif (tombol simpan kontributor nonaktif) |

Aturan umum: `export const prerender = false`; teks galat ke pengguna selalu dari `lib/planner/messages.ts`, tidak pernah `error.message` mentah (pesan server berbahasa Inggris). Proyek `archived`: semua kontrol ubah dinonaktifkan dengan keterangan "Acara diarsipkan" (`GET` dan ekspor tetap berjalan); tetap tangani `409 project_read_only` bila status berubah di tab lain. Proyek `completed` tetap bisa diedit (pelunasan susulan, [kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)).

Kerangka frontmatter `keuangan/index.astro` *(sketsa, belum dijalankan)*:

```ts
export const prerender = false;
const id = Astro.params.id ?? '';
if (!isUuid(id)) return Astro.rewrite('/404');
const q = Astro.url.searchParams;
const tab = ({ tabungan: 'tabungan', realisasi: 'realisasi', anggaran: 'realisasi' } as const)[q.get('tab') ?? ''] ?? 'simulasi';
const accessToken = Astro.locals.accessToken;                     // diisi middleware (token sudah segar)
const [detail, registry] = await Promise.all([
  backendFetch<Envelope<ProjectDetail>>(`/v1/planner/projects/${id}`, { accessToken }),
  loadRegistry(Astro),
]);
// … toPageError(detail) menerapkan tabel 2.2; selanjutnya per tab, mis. realisasi:
const base = `/v1/planner/projects/${id}`;
const [matrix, cash, lines, sessions] = await Promise.all([
  backendFetch<Envelope<BudgetMatrix>>(`${base}/budget/matrix`, { accessToken }),
  backendFetch<Envelope<CashflowSummary>>(`${base}/budget/cashflow-summary`, { accessToken }),
  backendFetch<Envelope<BudgetLineList>>(`${base}/budget/items`, { accessToken }),
  backendFetch<Envelope<SessionWithUsage[]>>(`${base}/sessions`, { accessToken }),
]);
const known = new Set(sessions.data.data.map((s) => s.scope_key));
const counts = countLines(lines.data.data.items, known);            // angka pada chip
const numbers = Object.fromEntries(lineNumbers(lines.data.data.items));
const columns = matrixColumns(matrix.data.data, sessions.data.data);
```

### 2.3 State di URL dan di peramban

| Penyimpan | Kunci | Nilai | Ditulis oleh | Dibaca oleh |
|---|---|---|---|---|
| URL | `tab` | `simulasi` (default) · `tabungan` · `realisasi` (alias `anggaran`) | tautan tab (navigasi penuh) | SSR: bagian yang dimuat |
| URL | `scenario` | id skenario | `ScenarioSwitcher` (navigasi penuh) | SSR (`pickScenario`); id yang tidak dikenal → skenario aktif |
| URL | `scope` | `scope_key` sesi atau `all` (Umum); tak ada = Semua | klien, `history.replaceState` | SSR menandai chip aktif dan memberi `hidden` pada baris lain; nilai yang tidak dikenal diperlakukan sebagai Semua |
| URL | `contributor` | id kontributor; tak ada = semua | chip penyetor (navigasi penuh) | SSR (`getSavings(id)`: hanya `entries` yang tersaring) |
| `sessionStorage` | `planner.finance.savingsView` | `matriks` (default) · `daftar` | pilihan tampilan setoran | skrip klien saat dimuat; nilai lain diabaikan. Dibungkus try/catch seperti `AuthForm` |

Tidak ada `localStorage`; tidak ada data proyek di penyimpan peramban.

### 2.4 Satu sumber render dan pembaruan sesudah aksi

Mengikuti pola partial [02 FE bab 2.4](../02_TIMELINE_DAN_TUGAS/FE.md#24-satu-sumber-render-dan-pembaruan-sesudah-aksi) dan [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial): **tidak ada logika render ganda di klien**.

1. SSR merender area `[data-finance-live]` dari komponen yang sama dengan `live.astro`. Memilih chip sesi (Tab 3), tampilan setoran (Tab 2), dan menyaring hanyalah mengubah atribut `hidden` di klien, tanpa memanggil server; baris dan sel membawa atribut data ([tabel 9.5.1](#951-atribut-data-yang-dibaca-skrip-klien)).
2. Aksi tulis berhasil → klien memanggil `GET /dashboard/planner/{id}/keuangan/live?tab=…&…` (`Accept: text/html`), mengganti isi `[data-finance-live]`, lalu menerapkan ulang chip dan tampilan aktif dari URL/`sessionStorage`. Fokus dipertahankan lewat `data-focus-key`.
3. Tidak ada pembaruan optimistik di menu ini kecuali **centang sesi** dan **lebar pratinjau drawer**: nilai uang selalu datang dari server (tabel bergantung pada `remaining`, `payment_status`, dan matriks yang dihitung server), jadi setiap aksi menunggu respons lalu menyegarkan area. Tombol aksi dinonaktifkan selama permintaan berjalan (`aria-busy`), sehingga klik ganda tidak mengirim dua permintaan.
4. Partial gagal (`5xx`, jaringan): jangan kosongkan area; tampilkan banner kecil "Data belum diperbarui" + tombol "Muat ulang" (`location.reload()`), `aria-live="polite"`. `401` atau pengalihan ke login: `location.assign('/auth/masuk')`.
5. Tidak ada polling di menu ini. Galat yang menandakan data usang (`not_found`, `invalid_scope`, `ids: must list every id exactly once`, `409 conflict`) selalu ditutup dengan memuat ulang area lewat partial.

*(Belum diuji di browser, sama seperti 01 dan 02. Alternatif yang sah: `location.reload()` sesudah tiap aksi tulis.)*

---

## 3. Prasyarat yang harus sudah ada dari Menu 01 FE

**Belum ada satu pun di repo** (frontend Planner belum dibangun). Menu ini mengimpor, tidak membangun ulang:

| Kebutuhan | Berasal dari |
|---|---|
| `types.ts` (`Notice`, `SessionBrief`, `SessionWithUsage`, `EventTypeInfo`, `Project`, `ProjectDetail`, `UpdateProjectBody`, `UpdateProjectResult`, `Envelope`, `ApiError`) | 01 FE bab 4.2 |
| `format.ts` (`formatDate`, `formatDateShort`, `formatRupiah`, `formatRupiahShort`, `percent`, `formatPercent`, `barWidth`, `todayIn`, `addDays`, `daysBetween`, `charCount`, `parseFieldError`) | 01 FE bab 5.1 |
| `messages.ts` (`plannerErrorMessage`, `fieldErrorMessage`, `WARNING_LABELS`) | 01 FE bab 5.4 dan 11 |
| `client.ts` (`plannerFetch`: tangani sesi habis, `204`, backend mati; **metode `PUT`** untuk skenario, kontributor, setoran, pos) dan `server.ts` (`loadRegistry`, `isUuid`) | 01 FE bab 5.5–5.6 |
| BFF `[...path].ts` dan perubahan `backend.ts` (`PUT`, `details`) | 01 FE bab 6 dan 1.3 |
| `PlannerLayout.astro` (Project Switcher, sub-navigasi `MENUS`) dan `404.astro` | 01 FE bab 10 dan 1.1 |
| Pola `<dialog>`, `sessionStorage` dibungkus try/catch, `textContent` untuk string API | 01 FE bab 12 |
| `moveBefore`, `moveByOne` (hanya mengenal id) untuk urut ulang pos | 02 FE bab 5 (`timeline.ts`); bila Menu 02 FE belum dibangun, salin keduanya ke `lib/planner/order.ts` dan ubah impor di 02 |

`format.ts` 01 FE **memformat** rupiah tetapi tidak **mengurai** isian pengguna; `parseRupiah` ada di bab 5 (versi awal dokumen ini menyebutnya sebagai bagian `format.ts` yang sudah ada, padahal tidak ada).

Karena itu urutan kerja ([bab 13](#13-urutan-pengerjaan-dan-definisi-selesai)) dimulai dari M0, M1, dan `PlannerLayout` di Menu 01 FE.

---

## 4. Kontrak data: tipe, fixture, contoh respons, katalog galat

### 4.1 Aturan JSON yang dikirim backend

Selain aturan umum [01 FE bab 4.1](../01_DASHBOARD/FE.md#41-aturan-json-yang-dikirim-backend) (amplop `{ "data": … }`, tanggal `"YYYY-MM-DD"` tanpa zona, uang bilangan bulat). Berbeda dengan 02 FE, butir di bawah **berasal dari BE.md dan SQL yang dijalankan, belum diamati pada backend yang berjalan**.

**Bentuk respons**

- **Kunci dihilangkan bila kosong** (`omitempty`): `item_key`, `preset_key`, `preset_amount`, `vendor_id`, `vendor_name`, `admin_doc_id`, `rollup_source`, `quantity`, `unit_label`, `unit_price`, `*_paid_at`, `*_receipt_url`, `notes`, `per_contributor`, `deleted`, `new_target_budget`, `total_target` pada `PresetInfo` percent. Tipe memakai `?:`; jangan memeriksa `=== null`. Satu-satunya `null` yang dikirim: `BudgetMatrix.active_scenario`.
- **Daftar selalu array**, tidak pernah `null`, termasuk saat kosong.
- **Tidak ada persentase di respons.** Hitung dengan `percent` (rasio, `null` bila penyebut ≤ 0 → tampilkan "–" atau keadaan kosong) dan `sharePercents` (bagian yang harus berjumlah 100). Jangan menjumlah hasil `formatPercent` per bagian sendiri (bab 5).
- **`remaining` dan `available_cash` boleh negatif** (`remaining < 0` = lebih bayar; kas bebas negatif = kas kurang). `unallocated` pada skenario boleh negatif (melebihi plafon).
- **`estimate`, `paid`, `remaining`, `payment_status` adalah hasil server** (view). Klien tidak menghitung ulang status dari angka; `paymentBadge` hanya memilih label. Kolom tabel `paid_total` dan `remaining_balance` **tidak dikirim** dan tidak boleh diminta.
- **`included` dan `allocated` pada skenario adalah hasil server**: pos dengan `included: false` (sesinya tidak dicentang) tidak digambar dan tidak dijumlah.
- **Matriks bulanan**: `by_contributor` memuat setiap kontributor (0 bila kosong) dan bulan berurutan tanpa celah; kolom tabel = urutan `contributors`.
- **Pos `rollup`** (`rollup_source: "shopping"`) membawa angka dari belanja; `dp_amount` dan sejenisnya bukan pembayaran efektif. Pembayaran terkunci (`409 rollup_locked`) dan inputnya dinonaktifkan.
- **Pada Proyek Contoh `*_paid_at` tidak ada** (PDF tidak memuat tanggal pembayaran): tampilkan "–", bukan tanggal kosong.
- **`message` galat dan `warnings[].message` berbahasa Inggris** dan hanya untuk log. Tampilkan teks dari `messages.ts` berdasarkan `code` (+ kolom/alasan untuk `invalid_*`).
- `sort_order` bukan nomor tampilan: kolom "No" memakai `lineNumbers`.

**Permintaan**

- **Bilangan bulat murni.** Uang dikirim sebagai integer (hanya digit); jangan mengirim string atau desimal (`invalid_json` tanpa nama kolom). Ubah isian teks dengan `parseRupiah` (bab 5); `null` berarti isian tidak valid, tampilkan galat kolom dan jangan mengirim.
- **`quantity` boleh desimal** (dua digit); itulah satu-satunya `float` di menu ini.
- **Kunci yang tidak dikenal diabaikan tanpa galat**: andalkan tipe TypeScript.
- **Teks dipangkas** (`trim`) oleh server; panjang dihitung dalam *code point*: pakai `charCount` (01 FE), bukan `.length`.
- **`PUT` mengganti penuh** (skenario, kontributor, setoran, pos, item skenario): kirim **semua** isian formulir. Khusus pos: **`vendor_id` yang tidak dikirim melepas vendor** yang tertaut (termasuk lewat `lock-deal`), jadi formulir mengisi awal `vendor_id` dari baris dan mengirimnya kembali. `receipt_url` yang tidak dikirim pada pembayaran mengosongkan bukti: isi awal dengan nilai lama.
- **Pembayaran `full`**: `amount` diabaikan server (memakai sisa saat menulis); jangan menampilkan nominal yang bisa diedit.
- **Aman diulang:** semua `GET`, `PUT` (hasil sama), `apply` skenario (kedua kali semua pos `same`, laporan kosong), `apply-template` mode `append`, `reorder` dengan daftar yang sama. **Tidak** aman diulang: semua `POST` pembuat (skenario, klon, pos, kontributor, setoran), `replace_template`, `clean_overwrite`. `DELETE` kedua kali menjawab `404`: perlakukan sebagai berhasil dan segarkan.

### 4.2 Tambahan pada `src/lib/planner/types.ts`

Hanya sintaks yang dapat dihapus (tanpa `enum`), seperti 01 FE. `Notice`, `SessionBrief`, `ApiError`, `UpdateProjectResult` berasal dari blok 01. Perubahan pada blok 01 ada di [bab 1.3](#13-perubahan-pada-file-yang-sudah-ada) (`EventTypeInfo` dan `UpdateProjectBody`).

```ts
// ---------- Menu 03: keuangan (ditambahkan ke akhir types.ts) ----------
// Tambahan registry (GET /event-types): peran kontributor jenis acara. Ubah satu baris di blok 01:
// `export interface EventTypeInfo {` → `export interface EventTypeInfo extends EventTypeFinanceInfo {`
// (bila Menu 02 FE sudah dibangun: `extends EventTypeAdminInfo, EventTypeFinanceInfo`).
export interface ContributorRole { key: string; label: string }
export interface EventTypeFinanceInfo { contributor_roles: ContributorRole[] }

export type PaymentStatus = 'unpaid' | 'dp_paid' | 'partial' | 'settled';
export type PaymentStage = 'dp' | 'term_1' | 'term_2';
export type PricingMode = 'lump' | 'unit';
export type BudgetSource = 'template' | 'scenario' | 'vendor' | 'admin_doc' | 'manual';

// ----- Tab 1: skenario -----
export interface ScenarioItem {
  id: string; sub_event_scope: string; item_key?: string; category: string; item_name: string;
  cost_amount: number;
  included: boolean;      // hasil server: sesinya ikut disimulasikan (active_sub_events kosong, 'all', atau terdaftar)
  preset_amount?: number; // nilai pos ini di preset asal skenario (ada bila item_key cocok); dasar "kembalikan ke nilai preset"
}
export interface Scenario {
  id: string; name: string; total_target: number; is_active: boolean;
  active_sub_events: string[];  // kosong = semua sesi non-sistem
  preset_key?: string;
  allocated: number;            // hasil server: Σ cost_amount item yang included
  items: ScenarioItem[];
}
export interface PresetInfo { key: string; kind: 'fixed' | 'percent'; total_target?: number } // total_target hanya pada 'fixed'
export interface ScenarioList { scenarios: Scenario[]; presets: PresetInfo[] }
export interface ScenarioResult { scenario: Scenario; warnings: Notice[] }

export interface ScenarioBody { name: string; total_target?: number; preset_key?: string; active_sub_events?: string[] }
export interface ScenarioUpdate { name: string; total_target: number; active_sub_events: string[] } // PUT: mengganti penuh
export interface ScenarioItemBody { sub_event_scope: string; item_key?: string; category: string; item_name: string; cost_amount: number }
export interface ScenarioItemUpdate { sub_event_scope: string; category: string; item_name: string; cost_amount: number } // item_key tidak berubah
export interface LoadPresetBody { preset_key: string; total_target?: number; confirm?: boolean }

export interface CompareRow { sub_event_scope: string; item_key?: string; item_name: string; a_amount: number; b_amount: number; delta: number }
export interface ScenarioComparison {
  a: { id: string; name: string; total_target: number }; b: { id: string; name: string; total_target: number };
  rows: CompareRow[]; totals: { a: number; b: number; delta: number }; // delta = b − a
}

export type ApplyMode = 'smart_adjust' | 'clean_overwrite';
export interface ApplyBody { scenario_id: string; mode: ApplyMode; sync_target_budget: boolean; sync_target_savings: boolean; confirm?: string }
export interface ApplyReport {
  mode: ApplyMode;
  updated: { sub_event_scope: string; item_key?: string; item_name: string; from: number; to: number }[];
  created: { sub_event_scope: string; item_key?: string; item_name: string; amount: number }[];
  skipped: { sub_event_scope: string; item_key?: string; item_name: string; reason: 'rollup' | 'unit_pricing' }[];
  deleted?: { lines: number; paid_total: number };   // hanya clean_overwrite
  untouched_count: number;
  estimate_after: number;                            // Σ estimasi semua pos sesudah penerapan
  new_target_budget?: number;                        // ada bila sync_target_budget
}

// ----- Tab 2: kas dan tabungan -----
export interface Contributor { id: string; name: string; role_label: string; total: number; entry_count: number }
export interface ContributorBody { name: string; role_label?: string }
export interface SavingsEntry {
  id: string; contributor_id: string; contributor_name: string; role_label: string;
  entry_date: string; amount: number; notes?: string;
}
export interface SavingsEntryBody { contributor_id: string; entry_date: string; amount: number; notes?: string }
export interface Milestone { sub_event_scope: string; name: string; session_date: string; target_amount: number; secured_amount: number; is_secured: boolean }
export interface SavingsMonth { month: string; by_contributor: Record<string, number>; total: number } // month "2026-02"; semua kontributor selalu ada (0 bila kosong)
export interface SavingsOverview {
  kpi: { target: number; collected: number; shortfall: number; target_follows_budget: boolean };
  contributors: Contributor[];
  milestones: Milestone[];
  unscheduled_estimate: number;  // Σ estimasi pos di sesi tanpa tanggal (termasuk Umum): tidak membentuk milestone
  recommendation: { months_left: number; recommended_monthly: number; per_contributor?: number; target_date: string; basis: 'session' | 'event_date' };
  matrix: { months: string[]; rows: SavingsMonth[] };
  entries: SavingsEntry[];       // disaring ?contributor_id= bila ada; bagian lain selalu seluruh proyek
}

// ----- Tab 3: realisasi -----
export interface BudgetLine {
  id: string; sub_event_scope: string; category: string; item_name: string; item_key?: string;
  source: BudgetSource;
  vendor_id?: string; vendor_name?: string; admin_doc_id?: string; rollup_source?: 'shopping';
  pricing_mode: PricingMode; quantity?: number; unit_label?: string; unit_price?: number;
  estimate: number;                                  // efektif (dari belanja bila rollup)
  dp_amount: number; term_1_amount: number; term_2_amount: number;
  dp_paid_at?: string; term_1_paid_at?: string; term_2_paid_at?: string;
  dp_receipt_url?: string; term_1_receipt_url?: string; term_2_receipt_url?: string;
  paid: number; remaining: number;                   // remaining < 0 = lebih bayar
  payment_status: PaymentStatus;
  sort_order: number; notes?: string;
}
export interface BudgetLineList { items: BudgetLine[] }
export interface BudgetItemBody {                    // POST dan PUT (PUT mengganti penuh: kirim semua kolom formulir)
  sub_event_scope: string; category: string; item_name: string; vendor_id?: string;
  pricing_mode?: PricingMode; initial_estimate?: number; quantity?: number; unit_label?: string; unit_price?: number; notes?: string;
}
export interface PaymentBody { payment_stage: PaymentStage | 'full'; amount?: number; paid_at: string; receipt_url?: string }
export interface ApplyTemplateBody { mode: 'append' | 'replace_template'; sub_event_scope?: string; confirm?: boolean }
export interface ApplyTemplateResult { created: number; skipped: number; deleted: number; warnings: Notice[] }

export interface CashflowSummary {
  total_savings_collected: number; total_paid: number; available_cash: number; // available_cash boleh negatif
  remaining_debt: number; overpaid: number; cash_deficit: number;
}
export interface MatrixRow {
  sub_event_scope: string; estimate: number; dp: number; term_1: number; term_2: number; rollup_paid: number;
  paid: number; remaining: number; overpaid: number; line_count: number;
}
export interface BudgetMatrix {
  kpi: {
    total_estimate: number; total_paid: number; total_remaining: number; total_overpaid: number; line_count: number;
    line_status: { settled: number; partial: number; unpaid: number }; // hanya pos berestimasi > 0
  };
  rows: MatrixRow[];                                 // satu per sesi yang punya pos, urut sesi
  total: MatrixRow;                                  // jumlah semua baris (sub_event_scope "")
  active_scenario: { id: string; name: string; total_target: number; unallocated: number } | null; // unallocated = total_target − Σ estimasi (boleh negatif)
}
```

**Hasil pencocokan dengan DTO Go** (blok [BE bab 5](BE.md#5-dto-go) dikompilasi, lalu skema JSON-nya diambil dengan `reflect` dan dibandingkan dengan interface di atas oleh parser; **bukan** terhadap respons HTTP):

| Kelompok | Pasangan tipe (Go ↔ TS) | Jalur JSON |
|---|:-:|:-:|
| Respons Tab 1 (`Scenario`, `ScenarioItem`, `PresetInfo`, `ScenarioList`, `ScenarioResult`, `ScenarioComparison`, `ApplyReport`) | 7 | 107 |
| Respons Tab 2 (`Contributor`, `SavingsEntry`, `Milestone`, `SavingsMonth`, `SavingsOverview`) | 5 | 60 |
| Respons Tab 3 (`BudgetLine`, `BudgetLineList`, `ApplyTemplateResult`, `CashflowSummary`, `MatrixRow`, `BudgetMatrix`) | 6 | 115 |
| Permintaan (11 tipe: skenario, item skenario, preset, apply, kontributor, setoran, pos, pembayaran, template) | 11 | 47 |
| **Jumlah** | **29** | **329** |

Hasil: **0 selisih** (kunci hanya di satu sisi, opsionalitas `omitempty` ↔ `?:`, `null` ↔ pointer tanpa `omitempty`, jenis nilai); untuk tipe permintaan hanya kunci dan jenis nilai yang dibandingkan (opsionalitas permintaan dijaga usecase). Pemeriksa diuji dengan enam mutasi tipe TS (opsional dijadikan wajib, kunci dibuang, `null` dibuang, jenis diganti, kunci diganti nama, kunci berlebih) dan melaporkan semuanya. Cara mengulang: [bab 12.5](#125-verifikasi-kontrak-cara-mengulang).

### 4.3 Fixture

Lima berkas di [`fixtures/`](fixtures/) (folder ini) disalin ke `frontend/tests/fixtures/`, ditambah `planner-sessions.json` dari [Menu 02](../02_TIMELINE_DAN_TUGAS/fixtures/planner-sessions.json) (sesi sistem `all` + Lamaran 15 Okt 2026 + Akad + Resepsi). **Bukan respons endpoint** (endpoint belum ada): dibangun dari view `planner_budget_lines` dan SQL spesifikasi ([BE bab 3](BE.md#3-logika-bisnis)) terhadap Proyek Contoh di PostgreSQL 16.2 sementara, dengan jam tetap 4 Okt 2026 (baseline sama dengan fixture Menu 02), `id` diganti id sintetis tetap supaya tes deterministik, dan `null` dibuang menurut aturan `omitempty`.

| Berkas | Isi | Bentuk |
|---|---|---|
| `planner-budget-lines.json` | 14 pos PDF: `sort_order` 1–14, `source: "template"`, tanpa `*_paid_at`; 7 `settled`, 6 `dp_paid`, 1 `partial`; Σ `estimate` 74.150.000, `paid` 33.550.000, `remaining` 40.600.000 | `BudgetLineList` |
| `planner-matrix.json` | KPI (`line_status {7, 7, 0}`), baris Lamaran / Akad / Resepsi, total `74.150.000 / 20.050.000 / 12.000.000 / 1.500.000 / 33.550.000 / 40.600.000`, `active_scenario` 75 Jt dengan `unallocated 850.000` | `BudgetMatrix` |
| `planner-cashflow.json` | terkumpul 47.500.000, terbayar 33.550.000, kas bebas 13.950.000, sisa 40.600.000, defisit 26.650.000 | `CashflowSummary` |
| `planner-savings.json` | KPI (target 75.000.000 ikut pagu, kekurangan 27.500.000), 2 kontributor (30.000.000 / 17.500.000), 3 milestone (Resepsi 39.750.000 dari 66.400.000), rekomendasi 4.584.000 / 2.292.000 selama 6 bulan, matriks 11 bulan, 22 setoran | `SavingsOverview` |
| `planner-scenarios.json` | 3 skenario (50 / 75 aktif / 100 Jt; 10 pos tiap skenario, `active_sub_events` `[akad, resepsi]`, `preset_amount` terisi) dan 6 preset pernikahan | `ScenarioList` |

Contoh baris (dari `planner-budget-lines.json`):

```json
{ "id": "00000000-0000-4000-8000-000300000008", "sub_event_scope": "resepsi", "category": "Catering", "item_name": "Catering",
  "item_key": "catering", "source": "template", "pricing_mode": "lump", "estimate": 25000000,
  "dp_amount": 5000000, "term_1_amount": 10000000, "term_2_amount": 0, "paid": 15000000, "remaining": 10000000,
  "payment_status": "partial", "sort_order": 8 }
```

**Variasi yang wajib diuji** (turunkan dari fixture di dalam tes, jangan membuat berkas baru): proyek kosong (tanpa skenario, kontributor, pos); pos lebih bayar (`remaining < 0`); pos berestimasi 0 yang sudah dibayar; pos `rollup` (`rollup_source`, kolom bayar terkunci); pos berharga satuan (`quantity`, `unit_price`); pos dengan `vendor_name` dan `admin_doc_id`; skenario yang melebihi plafon (`unallocated < 0`); skenario dengan pos `included: false`; tanpa skenario aktif (`active_scenario: null`); satu sel matriks bulanan berisi dua setoran; kontributor tanpa setoran; sesi tanpa tanggal (`unscheduled_estimate > 0`); `sub_event_scope` yang tidak ada di `sessions` (data usang: tampilkan sebagai "Umum"); nama pos 150 karakter; jenis seminar (peran `internal`/`sponsor`/`other`, preset `seminar_pct`); proyek `archived`.

### 4.4 Katalog endpoint

Semua di bawah `projects/:id/` (`:id` = UUID proyek) lewat BFF; semua penulisan juga mungkin menjawab `401`, `404 not_found` (proyek), `409 project_read_only`, dan `5xx`/jaringan, yang tidak diulang di kolom Galat. Nama fungsi = `financeApi` (bab 6). Deskripsi lengkap: [BE bab 4](BE.md#4-endpoint). `sid` = id skenario, `iid` = id item atau pos, `cid` = id kontributor, `eid` = id setoran.

| Aksi UI | Fungsi | Metode dan path | Body | Sukses | Galat khusus |
|---|---|---|---|---|---|
| Muat skenario | `listScenarios` | `GET scenarios` | – | `200 ScenarioList` | – |
| (satu skenario) | `getScenario` | `GET scenarios/:sid` | – | `200 Scenario` | `404` (skenario) |
| Buat skenario | `createScenario` | `POST scenarios` | `ScenarioBody` | `201 ScenarioResult` | `409 max_scenarios_reached`, `400 invalid_input`/`invalid_template`/`invalid_scope` |
| Ubah nama/target/sesi | `updateScenario` | `PUT scenarios/:sid` | `ScenarioUpdate` | `200 Scenario` | `invalid_input`, `invalid_scope`; `404` |
| Hapus skenario | `deleteScenario` | `DELETE scenarios/:sid` | – | `204` | `404` (dianggap sukses) |
| Duplikasi | `cloneScenario` | `POST scenarios/:sid/clone` | `{ new_name }` | `201 ScenarioResult` | `409 max_scenarios_reached` |
| Paket cepat | `loadPreset` | `POST scenarios/:sid/load-preset` | `LoadPresetBody` | `200 ScenarioResult` | `400 invalid_template`, `invalid_input: confirm`, `…: total_target` |
| Tambah pos skenario | `addScenarioItem` | `POST scenarios/:sid/items` | `ScenarioItemBody` | `201 ScenarioItem` | `409 scenario_item_limit_reached`, `invalid_input`, `invalid_scope` |
| Ubah pos skenario | `updateScenarioItem` | `PUT scenarios/:sid/items/:iid` | `ScenarioItemUpdate` | `200 ScenarioItem` | seperti POST; `404` (pos skenario) |
| Hapus pos skenario | `deleteScenarioItem` | `DELETE scenarios/:sid/items/:iid` | – | `204` | `404` |
| Bandingkan | `compareScenarios` | `POST scenarios/compare` | `{ a, b }` | `200 ScenarioComparison` | `invalid_input: b`, `404` |
| Pratinjau terapkan | `previewApply` | `POST scenarios/apply-preview` | `ApplyBody` | `200 ApplyReport` | `invalid_input` (`mode`), `404` |
| Terapkan | `applyScenario` | `POST scenarios/apply` | `ApplyBody` | `200 ApplyReport` | `invalid_input: confirm` (`required`/`must be RESET`), `404` |
| Muat kontributor | `listContributors` | `GET contributors` | – | `200 Contributor[]` | – |
| Tambah kontributor | `createContributor` | `POST contributors` | `ContributorBody` | `201 Contributor` | `409 contributor_limit_reached`, `invalid_input: role_label` |
| Ubah kontributor | `updateContributor` | `PUT contributors/:cid` | `ContributorBody` | `200 Contributor` | seperti POST; `404` |
| Hapus kontributor | `deleteContributor` | `DELETE contributors/:cid` | – | `204` | `409 contributor_in_use`, `409 conflict`; `404` |
| Muat tabungan | `getSavings` | `GET savings[?contributor_id=]` | – | `200 SavingsOverview` | – |
| Catat setoran | `createSaving` | `POST savings` | `SavingsEntryBody` | `201 SavingsEntry` | `409 savings_entry_limit_reached`, `invalid_input: contributor_id`, `invalid_date` |
| Koreksi setoran | `updateSaving` | `PUT savings/:eid` | `SavingsEntryBody` | `200 SavingsEntry` | seperti POST; `404` |
| Hapus setoran | `deleteSaving` | `DELETE savings/:eid` | – | `204` | `404` |
| Ubah target tabungan | `setSavingsTarget` | `PATCH projects/:id` (Menu 01) | `{ savings_target }` atau `{ clear_savings_target: true }` | `200 UpdateProjectResult` | `invalid_input: savings_target` |
| Muat kas | `getCashflow` | `GET budget/cashflow-summary` | – | `200 CashflowSummary` | – |
| Muat matriks | `getMatrix` | `GET budget/matrix` | – | `200 BudgetMatrix` | – |
| Muat pos | `listLines` | `GET budget/items[?scope=]` | – | `200 BudgetLineList` | – |
| Tambah pos | `createLine` | `POST budget/items` | `BudgetItemBody` | `201 BudgetLine` | `409 budget_item_limit_reached`, `invalid_input`, `invalid_scope` |
| Ubah pos | `updateLine` | `PUT budget/items/:iid` | `BudgetItemBody` | `200 BudgetLine` | seperti POST; `409 vendor_already_linked`; `404` (pos) |
| Hapus pos | `deleteLine` | `DELETE budget/items/:iid` | – | `204` | `404` (dianggap sukses) |
| Catat pembayaran | `recordPayment` | `PATCH budget/items/:iid/payment` | `PaymentBody` | `200 BudgetLine` | `400 invalid_stage`, `409 rollup_locked`, `invalid_input`, `invalid_date` |
| Hapus pembayaran | `clearPayment` | `DELETE budget/items/:iid/payment/:stage` | – | `200 BudgetLine` | `409 rollup_locked`; `404` |
| Urut ulang | `reorderLines` | `POST budget/items/reorder` | `{ ids }` (**semua** id) | `200 BudgetLineList` | `invalid_input: ids` |
| Terapkan template | `applyTemplate` | `POST budget/apply-template` | `ApplyTemplateBody` | `200 ApplyTemplateResult` | `invalid_input: confirm`/`mode`, `409 budget_item_limit_reached` |
| Ekspor | `exportHref` | `GET budget/export?format=xlsx` | – | `200` berkas `.xlsx` | `429 rate_limited` (10/menit) |

### 4.5 Contoh permintaan dan respons

Bentuk dari BE bab 4 dan 3 serta fixture; **bukan** cuplikan respons HTTP yang berjalan.

**Catat pembayaran** (`PATCH budget/items/:iid/payment`). `full` pada pos Fotografer (estimasi 1.000.000, DP 500.000): `amount` tidak dikirim, server mengisi sisa ke tahap kosong pertama.

```json
// permintaan
{ "payment_stage": "full", "paid_at": "2026-10-04" }
// respons (BudgetLine)
{ "data": { "id": "…", "sub_event_scope": "lamaran", "item_name": "Fotografer", "estimate": 1000000,
  "dp_amount": 500000, "term_1_amount": 500000, "term_2_amount": 0, "term_1_paid_at": "2026-10-04",
  "paid": 1000000, "remaining": 0, "payment_status": "settled", "…": "…" } }
```

**Pratinjau terapkan** (`POST scenarios/apply-preview`) untuk proyek berisi 14 pos template dan skenario 75 Jt ([BE §3.2](BE.md#32-menerapkan-skenario-smart-adjust): hasil SQL yang dijalankan; sebagian pos dipotong):

```json
{ "data": { "mode": "smart_adjust",
  "updated": [ { "sub_event_scope": "akad", "item_key": "biaya_kua", "item_name": "Biaya KUA", "from": 0, "to": 600000 },
               { "sub_event_scope": "resepsi", "item_key": "catering", "item_name": "Catering", "from": 0, "to": 30000000 } ],
  "created": [ { "sub_event_scope": "all", "item_key": "dana_darurat", "item_name": "Dana darurat", "amount": 1000000 } ],
  "skipped": [], "untouched_count": 5, "estimate_after": 75000000, "new_target_budget": 75000000 } }
```

**Hapus pembayaran** (`DELETE budget/items/:iid/payment/term_1`) mengembalikan `BudgetLine` utuh dengan tahap itu `0` dan tanpa `term_1_paid_at`. **Ubah target tabungan**: `PATCH projects/:id` `{"savings_target": 80000000}`; kembali ikut pagu: `{"clear_savings_target": true}`.

**Matriks (potongan fixture):** `kpi {"total_estimate": 74150000, "total_paid": 33550000, "total_remaining": 40600000, "total_overpaid": 0, "line_count": 14, "line_status": {"settled": 7, "partial": 7, "unpaid": 0}}`, `active_scenario {"name": "Opsi Ideal 75 Jt", "total_target": 75000000, "unallocated": 850000}`.

**Rekomendasi (potongan fixture):** `{"months_left": 6, "recommended_monthly": 4584000, "per_contributor": 2292000, "target_date": "2027-03-28", "basis": "session"}`.

**Unduhan** (`GET budget/export?format=xlsx`): `200`, `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="keuangan-<slug>-<tanggal>.xlsx"`. Respons ke-11 dalam semenit → `429 {"error":{"code":"rate_limited",…}}`. *(Belum teramati; format dari BE §3.11.)*

### 4.6 Katalog galat

Bentuk: `{ "error": { "code": "…", "message": "…" } }` (+ `details` pada beberapa kode Menu 01). Untuk `invalid_input`/`invalid_date`, `message` = `<kolom>: <alasan>` dan dipetakan ke teks per kolom (bab 10.2); kode lain dipetakan per kode (bab 10.1). Daftar alasan baku: [BE §4.5](BE.md#45-validasi-dan-batas); tes 12.1 memastikan **setiap** alasan di sana punya teks Indonesia atau diserahkan eksplisit ke fungsi 01.

| Status | Kode | Contoh `message` | Penyebab dan penanganan UI |
|:-:|---|---|---|
| 400 | `invalid_input` | `name: must not be empty` · `amount: must be greater than 0` · `confirm: required` · `confirm: must be RESET` · `ids: must list every id exactly once` | pesan di bawah kolom (bab 10.2); `confirm: required` bukan galat tampil, melainkan perintah menampilkan kotak konfirmasi lalu mengirim ulang dengan `confirm` (bab 7.5, 9.5); `ids` → muat ulang area |
| 400 | `invalid_date` | `paid_at: must be within the project schedule window` · `entry_date: must be a valid date in YYYY-MM-DD format` | di bawah kolom |
| 400 | `invalid_scope` | `sub_event_scope: unknown session` | sesi dihapus di tab lain → muat ulang area |
| 400 | `invalid_template` | `preset_key: not available for this event type` | registry usang → muat ulang halaman |
| 400 | `invalid_stage` | `payment_stage: all stages are filled or nothing is left to pay` | semua tahap terisi atau sisa ≤ 0 → teks per kode; tombol "Bayar sisa" seharusnya sudah nonaktif lewat `paymentImpact` |
| 400 | `invalid_json` | `invalid JSON body` | kesalahan klien (tipe salah, uang bukan integer, body kosong tanpa `{}`) |
| 401 | `unauthorized` | `missing bearer token` | `plannerFetch` mengalihkan ke login |
| 404 | `not_found` | `project not found` · `scenario not found` · … | dibedakan oleh **aksi** (`where`), bukan oleh pesan; setelah `DELETE` kedua dianggap sukses |
| 409 | `project_read_only` | `the project is archived; restore it to make changes` | nonaktifkan semua kontrol ubah, teks 01 |
| 409 | `max_scenarios_reached` | `at most 3 scenarios per project` | tombol `[+]` dan Duplikasi nonaktif; teks per kode |
| 409 | `scenario_item_limit_reached` / `budget_item_limit_reached` / `contributor_limit_reached` / `savings_entry_limit_reached` | `at most 300 budget items per project` | banner di modal; tidak ada baris yang tertulis |
| 409 | `contributor_in_use` | `the contributor still has savings entries` | kontributor punya setoran → teks per kode |
| 409 | `vendor_already_linked` | `the vendor is already linked to another budget item` | modal pos: pesan di kolom Vendor |
| 409 | `rollup_locked` | `the line is computed from the shopping list` | tombol bayar sudah nonaktif; teks per kode bila tetap sampai |
| 409 | `conflict` | `the project was changed concurrently, try again` | jarang; teks 01, aman dicoba ulang (juga hasil hapus kontributor yang berbalapan) |
| 429 | `rate_limited` | `too many requests, try again later` | hanya ekspor di menu ini |
| 5xx / jaringan | `internal`, `service_unavailable` | – | teks umum 01; jangan mengosongkan layar |

---

## 5. Library `lib/planner/finance.ts`

Aturan 01 FE bab 5: modul **murni**, tanpa alias `@/`, tanpa `enum`, supaya tes `node --experimental-strip-types` dapat mengimpornya. **Sudah dijalankan** ([bab 12.1](#121-unit-testsplanner-financetestmjs)).

```ts
import { percent } from './format.ts';
import type {
  BudgetLine, BudgetMatrix, MatrixRow, PaymentStage, PaymentStatus, SavingsEntry, SavingsOverview, Scenario,
  ScenarioItem, SessionBrief,
} from './types.ts';

/** Batas yang ditegakkan backend (BE §3.4); klien memakainya untuk `maxlength` dan tombol, bukan sebagai pengganti server. */
export const LIMITS = {
  name: 100, itemName: 150, category: 60, notes: 2000, unitLabel: 30, receiptUrl: 500,
  money: 10_000_000_000_000, quantityMax: 1_000_000,
  scenarios: 3, scenarioItems: 100, budgetItems: 300, contributors: 20, savingsEntries: 2000,
  percentPresetMinTotal: 100_000,
} as const;

// ---------- uang dan persen ----------

/**
 * Isian rupiah → bilangan bulat, tanpa float. Menerima "75000000", "75.000.000", "Rp 75.000.000"; menolak koma/desimal
 * ("1,5", "1.5"), tanda minus, huruf, kosong, dan nilai > 10¹³. null = tidak valid (tampilkan galat kolom, jangan menebak).
 */
export function parseRupiah(input: string): number | null {
  const s = input.replace(/\s+/g, '').replace(/^Rp\.?/i, '');
  if (!/^(?:\d{1,3}(?:\.\d{3})+|\d+)$/.test(s)) return null;
  const n = Number(s.replace(/\./g, ''));
  return Number.isSafeInteger(n) && n <= LIMITS.money ? n : null;
}

/**
 * Persentase tiap bagian terhadap `denominator`, dalam `digits` desimal, tanpa float (BigInt). Bila jumlah bagian = penyebut,
 * hasilnya dijamin berjumlah tepat 100 (sisa terbesar; seri: nilai lebih besar, lalu urutan). Bila tidak (plafon belum
 * terbagi habis), tiap bagian dibulatkan sendiri. Penyebut <= 0 → null di setiap tempat (keadaan kosong, bukan 0%).
 */
export function sharePercents(values: readonly number[], denominator: number, digits = 1): (number | null)[] {
  if (denominator <= 0) return values.map(() => null);
  const scale = 10 ** digits;
  const unit = BigInt(100 * scale), den = BigInt(denominator);
  const parts = values.map((v) => BigInt(Math.max(0, v)) * unit);
  const floors = parts.map((p) => p / den);
  const rems = parts.map((p) => p % den);
  if (values.reduce((a, b) => a + Math.max(0, b), 0) === denominator) {
    let left = Number(unit - floors.reduce((a, b) => a + b, 0n));
    const order = values.map((_, i) => i).sort((i, j) => {
      const r = rems[j]! - rems[i]!;
      if (r !== 0n) return r > 0n ? 1 : -1;
      return (values[j]! - values[i]!) || i - j;
    });
    for (const i of order) { if (left-- <= 0) break; floors[i] = floors[i]! + 1n; }
  } else {
    parts.forEach((_, i) => { if (rems[i]! * 2n >= den) floors[i] = floors[i]! + 1n; });
  }
  return floors.map((f) => Number(f) / scale);
}

// ---------- Tab 3: pos dan pembayaran ----------

export const SCOPE_ANY = '*';
export const PAYMENT_STAGES: readonly PaymentStage[] = ['dp', 'term_1', 'term_2'];

/** Sesi pos untuk tampilan: sesi yang tidak ada di `known` (data usang) dianggap 'all' (Umum). Tanpa `known`, apa adanya. */
export const lineScope = (l: { sub_event_scope: string }, known?: ReadonlySet<string>): string =>
  known === undefined || l.sub_event_scope === 'all' || known.has(l.sub_event_scope) ? l.sub_event_scope : 'all';

/** Chip sesi: SCOPE_ANY = semua; 'all' = hanya pos Umum (aturan sama dengan Menu 02). Tidak memanggil server. */
export const filterLines = (lines: BudgetLine[], scope: string, known?: ReadonlySet<string>): BudgetLine[] =>
  scope === SCOPE_ANY ? lines : lines.filter((l) => lineScope(l, known) === scope);

export function countLines(lines: BudgetLine[], known?: ReadonlySet<string>): Record<string, number> {
  const out: Record<string, number> = { [SCOPE_ANY]: lines.length };
  for (const l of lines) { const k = lineScope(l, known); out[k] = (out[k] ?? 0) + 1; }
  return out;
}

export type Badge = PaymentStatus | 'overpaid' | 'unbudgeted';
/** Lencana status satu pos. `remaining < 0` = lebih bayar; dibayar padahal estimasi 0 = belum dianggarkan. Teks: PAYMENT_BADGE_LABELS. */
export function paymentBadge(l: Pick<BudgetLine, 'estimate' | 'paid' | 'remaining' | 'payment_status'>): Badge {
  if (l.estimate === 0 && l.paid > 0) return 'unbudgeted';
  if (l.remaining < 0) return 'overpaid';
  return l.payment_status;
}

/** Pos yang angkanya dihitung dari daftar belanja: kolom bayar terkunci (409 rollup_locked) dan estimasi tidak bisa diubah. */
export const isRollup = (l: Pick<BudgetLine, 'rollup_source'>): boolean => l.rollup_source !== undefined;

/** Tahap kosong pertama menurut urutan DP, Termin 1, Termin 2 (sasaran "Pelunasan Penuh"); null bila ketiganya terisi. */
export function nextEmptyStage(l: Pick<BudgetLine, 'dp_amount' | 'term_1_amount' | 'term_2_amount'>): PaymentStage | null {
  if (l.dp_amount === 0) return 'dp';
  if (l.term_1_amount === 0) return 'term_1';
  if (l.term_2_amount === 0) return 'term_2';
  return null;
}
const stageAmount = (l: BudgetLine, s: PaymentStage): number => (s === 'dp' ? l.dp_amount : s === 'term_1' ? l.term_1_amount : l.term_2_amount);

export interface PaymentImpact {
  stage: PaymentStage | null;       // tahap yang akan ditulis (full: tahap kosong pertama)
  amount: number;                   // nominal yang akan ditulis (full: sisa)
  replaces: number;                 // > 0 = "Mengganti Rp X" (koreksi tahap yang sudah terisi)
  remainingBefore: number; remainingAfter: number; // sesudah < 0 = lebih bayar
  error: 'locked' | 'no_stage' | 'nothing_left' | null; // padanan 409 rollup_locked / 400 invalid_stage
}
/** Pratinjau drawer pembayaran dengan aturan server (BE §3.3). */
export function paymentImpact(l: BudgetLine, stage: PaymentStage | 'full', amount: number): PaymentImpact {
  const base = { remainingBefore: l.remaining };
  if (isRollup(l)) return { ...base, stage: null, amount: 0, replaces: 0, remainingAfter: l.remaining, error: 'locked' };
  if (stage === 'full') {
    const target = nextEmptyStage(l);
    if (target === null) return { ...base, stage: null, amount: 0, replaces: 0, remainingAfter: l.remaining, error: 'no_stage' };
    if (l.remaining <= 0) return { ...base, stage: target, amount: 0, replaces: 0, remainingAfter: l.remaining, error: 'nothing_left' };
    return { ...base, stage: target, amount: l.remaining, replaces: 0, remainingAfter: 0, error: null };
  }
  const replaces = stageAmount(l, stage);
  return { ...base, stage, amount, replaces, remainingAfter: l.remaining + replaces - amount, error: null };
}

/** Ganti satu pos dengan BudgetLine dari respons server (posisi tetap). */
export const replaceLine = (lines: BudgetLine[], next: BudgetLine): BudgetLine[] => lines.map((l) => (l.id === next.id ? next : l));
export const removeLine = (lines: BudgetLine[], id: string): BudgetLine[] => lines.filter((l) => l.id !== id);

/** Kolom "No": urutan 1-based pada daftar LENGKAP menurut sort_order (sort_order bisa berlubang). */
export function lineNumbers(all: BudgetLine[]): Map<string, number> {
  const sorted = [...all].sort((a, b) => a.sort_order - b.sort_order);
  return new Map(sorted.map((l, i) => [l.id, i + 1]));
}

// ---------- Tab 3: matriks sesi ----------

export interface MatrixColumn { scope: string; label: string; icon: string; row: MatrixRow; settlement: number | null }
/**
 * Kolom matriks = sesi yang punya pos, urutan dari server, label dari master sesi (sesi 'all' = "Umum"; sesi yang tidak
 * dikenal memakai kuncinya). Kolom Total dipisah (`m.total`). `settlement` = terbayar/estimasi, null bila estimasi 0.
 */
export function matrixColumns(m: BudgetMatrix, sessions: readonly SessionBrief[]): MatrixColumn[] {
  const byKey = new Map(sessions.map((s) => [s.scope_key, s]));
  return m.rows.map((row) => {
    const s = byKey.get(row.sub_event_scope);
    return {
      scope: row.sub_event_scope,
      label: row.sub_event_scope === 'all' ? 'Umum' : (s?.name ?? row.sub_event_scope),
      icon: s?.icon ?? '',
      row, settlement: percent(row.paid, row.estimate),
    };
  });
}

export interface TemplateImpact {
  removed: number;      // pos bersumber 'template' tanpa pembayaran: akan dihapus oleh replace_template
  skippedPaid: number;  // pos bersumber 'template' yang sudah dibayar: dilewati
  atRisk: number;       // di antara yang dihapus: punya estimasi > 0, vendor, dokumen, atau gulungan belanja
  needsConfirm: boolean; // aturan server (BE §3.9): confirm: true bila atRisk > 0
}
/**
 * Dampak "Ganti pos template" pada daftar lokal, mengikuti aturan server. Server tetap penentu: bila ia menjawab
 * `400 confirm: required` padahal hitungan lokal tidak meminta (mis. tautan belanja tanpa rollup yang tidak terlihat klien),
 * tampilkan kotak konfirmasi lalu kirim ulang dengan confirm: true.
 */
export function templateImpact(lines: readonly BudgetLine[]): TemplateImpact {
  const tpl = lines.filter((l) => l.source === 'template');
  const del = tpl.filter((l) => l.paid === 0);
  const atRisk = del.filter((l) => l.estimate > 0 || l.vendor_id !== undefined || l.admin_doc_id !== undefined || l.rollup_source !== undefined).length;
  return { removed: del.length, skippedPaid: tpl.length - del.length, atRisk, needsConfirm: atRisk > 0 };
}

// ---------- Tab 1: skenario ----------

export interface ScenarioGroup { scope: string; label: string; icon: string; items: ScenarioItem[]; subtotal: number }
/**
 * Pos yang ikut disimulasikan, dikelompokkan per sesi menurut urutan master sesi, Umum ('all') dan sesi tak dikenal di akhir.
 * Subtotal dalam rupiah.
 */
export function groupScenarioItems(sc: Scenario, sessions: readonly SessionBrief[]): ScenarioGroup[] {
  const order = new Map(sessions.filter((s) => !s.is_system).map((s, i) => [s.scope_key, i]));
  const groups = new Map<string, ScenarioItem[]>();
  for (const it of sc.items) {
    if (!it.included) continue;
    const g = groups.get(it.sub_event_scope);
    if (g) g.push(it); else groups.set(it.sub_event_scope, [it]);
  }
  const byKey = new Map(sessions.map((s) => [s.scope_key, s]));
  return [...groups.entries()]
    .sort(([a], [b]) => (order.get(a) ?? 1e9) - (order.get(b) ?? 1e9) || (a < b ? -1 : a > b ? 1 : 0))
    .map(([scope, items]) => ({
      scope, label: scope === 'all' ? 'Umum' : (byKey.get(scope)?.name ?? scope), icon: byKey.get(scope)?.icon ?? '',
      items, subtotal: items.reduce((a, i) => a + i.cost_amount, 0),
    }));
}

/** Persen tiap pos terhadap total_target (bukan subtotal sesi); urutan = pos yang included menurut `sc.items`. */
export function scenarioShares(sc: Scenario): Map<string, number | null> {
  const inc = sc.items.filter((i) => i.included);
  const pct = sharePercents(inc.map((i) => i.cost_amount), sc.total_target);
  return new Map(inc.map((i, k) => [i.id, pct[k] ?? null]));
}
/** Plafon yang belum terbagi (boleh negatif = melebihi plafon). */
export const unallocated = (sc: Scenario): number => sc.total_target - sc.allocated;
export const canAddScenario = (count: number): boolean => count < LIMITS.scenarios;
/** "Kembalikan ke nilai preset" tersedia bila pos punya nilai preset dan sudah berbeda. */
export const canResetToPreset = (i: ScenarioItem): boolean => i.preset_amount !== undefined && i.preset_amount !== i.cost_amount;

/** Skenario yang dipilih: ?scenario= bila ada di daftar, selain itu yang aktif, selain itu yang pertama; null bila belum ada skenario. */
export function pickScenario(list: readonly Scenario[], wanted?: string | null): Scenario | null {
  return list.find((s) => s.id === wanted) ?? list.find((s) => s.is_active) ?? list[0] ?? null;
}

/** Kotak centang SubEventSelector: `active_sub_events` kosong = semua sesi non-sistem tercentang. */
export function checkedSessions(sc: Pick<Scenario, 'active_sub_events'>, sessions: readonly SessionBrief[]): Set<string> {
  const real = sessions.filter((s) => !s.is_system).map((s) => s.scope_key);
  return new Set(sc.active_sub_events.length === 0 ? real : real.filter((k) => sc.active_sub_events.includes(k)));
}

/**
 * Kebalikannya, untuk PUT skenario: semua tercentang → [] ("semua sesi", termasuk sesi yang dibuat kemudian); sebagian → daftar
 * eksplisit menurut urutan sesi. Tidak ada yang tercentang → null: [] berarti SEMUA, jadi "tidak ada" tidak dapat dikirim dan
 * antarmuka tidak boleh membiarkan centang terakhir dilepas.
 */
export function toActiveSubEvents(checked: ReadonlySet<string>, sessions: readonly SessionBrief[]): string[] | null {
  const real = sessions.filter((s) => !s.is_system).map((s) => s.scope_key);
  const picked = real.filter((k) => checked.has(k));
  if (picked.length === 0) return null;
  return picked.length === real.length ? [] : picked;
}

// ---------- Tab 2: tabungan ----------

/** "2026-02" → "Februari 2026" (zona UTC supaya tanggal kalender tidak bergeser). */
const idMonth = new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric', timeZone: 'UTC' });
export const monthLabel = (m: string): string => idMonth.format(new Date(`${m}-01T00:00:00Z`));

/** Porsi tiap kontributor dari total terkumpul (bukan dari target), satu desimal, berjumlah 100. */
export function contributorShares(o: Pick<SavingsOverview, 'contributors'>, digits = 1): Map<string, number | null> {
  const totals = o.contributors.map((c) => c.total);
  const pct = sharePercents(totals, totals.reduce((a, b) => a + b, 0), digits);
  return new Map(o.contributors.map((c, i) => [c.id, pct[i] ?? null]));
}

export type CellAction =
  | { kind: 'create' }                      // bulan itu kosong untuk kontributor ini: buka SavingsEntryModal baru
  | { kind: 'edit'; entry: SavingsEntry }   // tepat satu setoran: sunting setoran itu
  | { kind: 'list'; entries: SavingsEntry[] }; // lebih dari satu: buka Daftar tersaring bulan + kontributor
/** Perilaku sel Matriks Bulanan (tanggal setoran bebas, jadi satu sel bisa memuat beberapa setoran). */
export function cellAction(entries: readonly SavingsEntry[], month: string, contributorId: string): CellAction {
  const hit = entries.filter((e) => e.contributor_id === contributorId && e.entry_date.startsWith(month));
  if (hit.length === 0) return { kind: 'create' };
  const [only] = hit;
  return hit.length === 1 && only ? { kind: 'edit', entry: only } : { kind: 'list', entries: hit };
}

/** Milestone: rasio terjamin; null bila target 0. */
export const milestoneRatio = (m: { target_amount: number; secured_amount: number }): number | null => percent(m.secured_amount, m.target_amount);
```

| Fungsi | Dipakai oleh | Catatan |
|---|---|---|
| `parseRupiah` | semua isian uang | Menolak koma/desimal, minus, huruf, kosong, dan > 10¹³; **tidak** menebak `1.5` sebagai 1,5 (menolak) tetapi `1.500` adalah 1500 (pemisah ribuan id-ID) |
| `sharePercents` | donut, kolom % tabel skenario, kartu proporsi sesi, porsi kontributor | Sisa terbesar tanpa float (`BigInt`), berjumlah tepat 100 bila bagian menjumlah penyebut; jumlah yang tidak sama dengan penyebut (plafon belum terbagi habis) dibulatkan per bagian; penyebut ≤ 0 → `null`. Versi awal wireframe menjumlah 100,1% (Tab 1) dan 99,9% (Tab 3); dikoreksi |
| `filterLines`, `countLines`, `lineScope` | chip sesi Tab 3 | `SCOPE_ANY` = semua; `'all'` = hanya pos Umum; sesi usang tampil sebagai Umum |
| `paymentBadge` | `BudgetTable`, `PaymentDrawer` | `unbudgeted` bila estimasi 0 tetapi sudah dibayar; `overpaid` bila `remaining < 0` |
| `nextEmptyStage`, `paymentImpact` | `PaymentDrawer` | meniru aturan server (BE §3.3) untuk pratinjau dan nonaktifkan tombol; server tetap penentu |
| `templateImpact` | `TemplateMenu` | meniru syarat `confirm` `replace_template`; bila server menjawab `confirm: required` padahal lokal tidak, tampilkan kotak konfirmasi lalu kirim ulang |
| `matrixColumns` | `SessionMatrix`, `BudgetKpiCards` | label dari master sesi, `all` = "Umum", sesi yang sudah dihapus memakai kuncinya |
| `groupScenarioItems`, `scenarioShares`, `unallocated`, `canResetToPreset`, `canAddScenario` | Tab 1 | Umum di akhir; hanya pos `included` |
| `pickScenario`, `checkedSessions`, `toActiveSubEvents` | Tab 1 | semua sesi tercentang ↔ `active_sub_events: []`; **tidak ada yang tercentang → `null`**: `[]` berarti semua, jadi centang terakhir tidak boleh dilepas |
| `monthLabel`, `contributorShares`, `milestoneRatio`, `cellAction` | Tab 2 | sel matriks: kosong → buat, satu setoran → sunting, beberapa → buka Daftar tersaring bulan + penyetor |
| `LIMITS` | `maxlength`, tombol | cermin batas server (BE §4.5); bukan pengganti server |

---

## 6. Pembungkus panggilan, BFF, dan unduhan

Tidak ada file BFF baru. Pemanggilan klien memakai `plannerFetch('projects/:id/…')` (01 FE bab 5.5) dengan path yang lolos aturan BFF `^[a-z0-9/_-]+$` (`cashflow-summary`, `load-preset`, `apply-preview`, `apply-template`, `term_1` semuanya lolos; query `?scope=` dan `?format=xlsx` bukan bagian path). Semua non-GET mengirim `Content-Type: application/json` dengan body (`'{}'` bila kosong). Menu ini memakai metode **`PUT`**: pastikan `PlannerFetch` dan BFF 01 mengizinkannya (01 FE bab 1.3 menambahkan `PUT` ke `backend.ts`).

### 6.1 `financeApi.ts`

Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan sebagai argumen sehingga modul bisa diuji tanpa peramban. Setiap id di path diperiksa UUID huruf kecil dan melempar `Error` bila tidak (kesalahan pemrogram, bukan permintaan ke server). **Sudah dijalankan**: tes unit memeriksa metode, path, dan body ke-32 fungsi, dan bahwa setiap fungsi dipakai tepat sekali.

```ts
import type {
  ApiError, ApplyBody, ApplyReport, ApplyTemplateBody, ApplyTemplateResult, BudgetItemBody, BudgetLine, BudgetLineList,
  BudgetMatrix, CashflowSummary, Contributor, ContributorBody, LoadPresetBody, PaymentBody, PaymentStage, SavingsEntry,
  SavingsEntryBody, SavingsOverview, ScenarioBody, ScenarioComparison, ScenarioItem, ScenarioItemBody, ScenarioItemUpdate,
  ScenarioList, ScenarioResult, ScenarioUpdate, Scenario, UpdateProjectResult,
} from './types.ts';

/** Bentuk hasil `plannerFetch` (01 FE bab 5.5): tidak pernah melempar; sesi habis sudah dialihkan di dalamnya. */
export interface PlannerResult<T> { ok: boolean; status: number; data: T | null; error: ApiError | null }
export type PlannerFetch = <T>(
  path: string,
  init?: { method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'; body?: unknown; query?: Record<string, string> },
) => Promise<PlannerResult<T>>;

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
/** Path dibangun hanya dari UUID huruf kecil; selain itu kesalahan pemrogram (lempar), bukan permintaan ke server. */
const uuid = (v: string): string => {
  if (!UUID.test(v)) throw new Error(`bukan UUID: ${v}`);
  return v;
};

/** Satu fungsi per aksi di UI (kolom "Fungsi" di bab 4.4). Semua non-GET mengirim body (plannerFetch mengisi '{}' bila undefined). */
export function financeApi(call: PlannerFetch, projectId: string) {
  const p = `projects/${uuid(projectId)}`;
  const sc = (id: string) => `${p}/scenarios/${uuid(id)}`;
  const co = (id: string) => `${p}/contributors/${uuid(id)}`;
  const sv = (id: string) => `${p}/savings/${uuid(id)}`;
  const bi = (id: string) => `${p}/budget/items/${uuid(id)}`;
  return {
    // ----- Tab 1: skenario -----
    listScenarios: () => call<ScenarioList>(`${p}/scenarios`),
    getScenario: (id: string) => call<Scenario>(sc(id)),
    createScenario: (body: ScenarioBody) => call<ScenarioResult>(`${p}/scenarios`, { method: 'POST', body }),
    updateScenario: (id: string, body: ScenarioUpdate) => call<Scenario>(sc(id), { method: 'PUT', body }),
    deleteScenario: (id: string) => call<null>(sc(id), { method: 'DELETE' }),
    cloneScenario: (id: string, newName: string) => call<ScenarioResult>(`${sc(id)}/clone`, { method: 'POST', body: { new_name: newName } }),
    loadPreset: (id: string, body: LoadPresetBody) => call<ScenarioResult>(`${sc(id)}/load-preset`, { method: 'POST', body }),
    addScenarioItem: (id: string, body: ScenarioItemBody) => call<ScenarioItem>(`${sc(id)}/items`, { method: 'POST', body }),
    updateScenarioItem: (id: string, itemId: string, body: ScenarioItemUpdate) => call<ScenarioItem>(`${sc(id)}/items/${uuid(itemId)}`, { method: 'PUT', body }),
    deleteScenarioItem: (id: string, itemId: string) => call<null>(`${sc(id)}/items/${uuid(itemId)}`, { method: 'DELETE' }),
    compareScenarios: (a: string, b: string) => call<ScenarioComparison>(`${p}/scenarios/compare`, { method: 'POST', body: { a: uuid(a), b: uuid(b) } }),
    previewApply: (body: ApplyBody) => call<ApplyReport>(`${p}/scenarios/apply-preview`, { method: 'POST', body }),
    applyScenario: (body: ApplyBody) => call<ApplyReport>(`${p}/scenarios/apply`, { method: 'POST', body }),

    // ----- Tab 2: kontributor, setoran, target -----
    listContributors: () => call<Contributor[]>(`${p}/contributors`),
    createContributor: (body: ContributorBody) => call<Contributor>(`${p}/contributors`, { method: 'POST', body }),
    updateContributor: (id: string, body: ContributorBody) => call<Contributor>(co(id), { method: 'PUT', body }),
    deleteContributor: (id: string) => call<null>(co(id), { method: 'DELETE' }),
    /** `contributorId` hanya menyaring `entries`; KPI, kartu, milestone, matriks selalu seluruh proyek. */
    getSavings: (contributorId?: string) =>
      call<SavingsOverview>(`${p}/savings`, contributorId === undefined ? undefined : { query: { contributor_id: uuid(contributorId) } }),
    createSaving: (body: SavingsEntryBody) => call<SavingsEntry>(`${p}/savings`, { method: 'POST', body }),
    updateSaving: (id: string, body: SavingsEntryBody) => call<SavingsEntry>(sv(id), { method: 'PUT', body }),
    deleteSaving: (id: string) => call<null>(sv(id), { method: 'DELETE' }),
    /** Tidak ada endpoint target di Menu 03: memakai PATCH proyek Menu 01. `null` = kembali mengikuti pagu (clear_savings_target). */
    setSavingsTarget: (value: number | null) =>
      call<UpdateProjectResult>(p, { method: 'PATCH', body: value === null ? { clear_savings_target: true } : { savings_target: value } }),

    // ----- Tab 3: realisasi -----
    getCashflow: () => call<CashflowSummary>(`${p}/budget/cashflow-summary`),
    getMatrix: () => call<BudgetMatrix>(`${p}/budget/matrix`),
    /** `scope` = scope_key sesi; 'all' = hanya pos Umum; tanpa argumen = semua sesi. */
    listLines: (scope?: string) => call<BudgetLineList>(`${p}/budget/items`, scope === undefined ? undefined : { query: { scope } }),
    createLine: (body: BudgetItemBody) => call<BudgetLine>(`${p}/budget/items`, { method: 'POST', body }),
    updateLine: (id: string, body: BudgetItemBody) => call<BudgetLine>(bi(id), { method: 'PUT', body }),
    deleteLine: (id: string) => call<null>(bi(id), { method: 'DELETE' }),
    recordPayment: (id: string, body: PaymentBody) => call<BudgetLine>(`${bi(id)}/payment`, { method: 'PATCH', body }),
    clearPayment: (id: string, stage: PaymentStage) => call<BudgetLine>(`${bi(id)}/payment/${stage}`, { method: 'DELETE' }),
    /** `ids` = SEMUA id pos proyek menurut urutan baru. */
    reorderLines: (ids: string[]) => call<BudgetLineList>(`${p}/budget/items/reorder`, { method: 'POST', body: { ids } }),
    applyTemplate: (body: ApplyTemplateBody) => call<ApplyTemplateResult>(`${p}/budget/apply-template`, { method: 'POST', body }),
  };
}

/** Tujuan tautan unduhan .xlsx (GET biasa lewat BFF; bukan plannerFetch karena responsnya berkas). */
export const exportHref = (projectId: string): string => `/dashboard/api/planner/projects/${uuid(projectId)}/budget/export?format=xlsx`;
```

Pemakaian di skrip klien:

```ts
const api = financeApi(plannerFetch, projectId);
const res = await api.recordPayment(lineId, { payment_stage: 'dp', amount, paid_at, receipt_url });
if (!res.ok) { toast(financeErrorMessage(res.status, res.error?.code, 'line') ?? plannerErrorMessage(res.status, res.error?.code)); return; }
await refreshArea();            // partial; res.data bertipe BudgetLine | null
```

### 6.2 Unduhan `.xlsx`

Tautan biasa `<a href={exportHref(project.id)} download>`; `GET` tidak butuh `Content-Type`, cookie sesi ikut. BFF meneruskan berkas sebagai aliran beserta `Content-Disposition` dari backend dan menambahkan `Cache-Control: no-store` (berkas berisi data pribadi). Bila sesi habis, middleware membalas pengalihan ke halaman login: tautan dibuka di tab yang sama sehingga pengguna sampai ke login dan, setelah masuk, kembali ke halaman. `429` membuka halaman galat BFF; alternatif `fetch` + Blob agar bisa menampilkan teks batas dicatat di bab 14 butir 8.

---

## 7. Tab 1: Simulasi & Alokasi Budget

### 7.1 Wireframe (preset 75 Jt aktif; Lamaran tidak disimulasikan)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ Keuangan & Anggaran                                      [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ]│
│ [★ 1 Simulasi]   [2 Kas & Tabungan]   [3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Skenario (maks. 3):  [ Opsi Hemat 50 Jt ]  [ ★ Opsi Ideal 75 Jt — AKTIF ]  [ Opsi 100 Jt ] [+]│
│ Rangkaian yang disimulasikan:  [ ] 💍 Lamaran   [✓] 📜 Akad Nikah   [✓] 🏛️ Resepsi   [+ Sesi] │
│ Paket cepat: [10 Jt] [20 Jt] [30 Jt] [50 Jt] [★ 75 Jt] [100 Jt]   Target: [ Rp 75.000.000 ] ━━●━│
├────────────────────────────────────┬─────────────────────────────────────────────────────────┤
│ 🍩 PROPORSI                        │ RINCIAN POS                                             │
│      Catering 40,0%                │ 📜 AKAD NIKAH                              Rp 600.000   │
│      Venue & Dekor 26,7%           │   Biaya KUA                       600.000   0,8%  ✏️ 🗑️ │
│      Mahar 6,7% · Rias 6,7% …      │ 🏛️ RESEPSI                              Rp 73.400.000   │
│                                    │   Mahar & cincin pernikahan     5.000.000   6,7%  ✏️ 🗑️ │
│ Terdistribusi Rp 75.000.000        │   Catering                     30.000.000  40,0%  ✏️ 🗑️ │
│ dari Rp 75.000.000 (sisa Rp 0)     │   Rias & baju pengantin         5.000.000   6,7%  ✏️ 🗑️ │
│                                    │   Sewa tempat & dekorasi       20.000.000  26,7%  ✏️ 🗑️ │
│                                    │   Undangan & souvenir           3.500.000   4,7%  ✏️ 🗑️ │
│                                    │   Dokumentasi                   3.000.000   4,0%  ✏️ 🗑️ │
│                                    │   Seserahan                     3.500.000   4,6%  ✏️ 🗑️ │
│                                    │   Hiburan & pengisi acara       3.400.000   4,5%  ✏️ 🗑️ │
│                                    │ 🛡️ UMUM                                    Rp 1.000.000 │
│                                    │   Dana darurat                  1.000.000   1,3%  ✏️ 🗑️ │
│                                    │ [+ Tambah pos di sesi ini]                              │
├────────────────────────────────────┴─────────────────────────────────────────────────────────┤
│ [⚖️ Bandingkan Skenario]                        [🚀 Jadikan Skenario Aktif & Terapkan ke Tab 3] │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Persentase dihitung terhadap `total_target` skenario (bukan terhadap subtotal sesi) dan berjumlah tepat 100,0 bila plafon terbagi habis (`sharePercents`): Seserahan 4,6% (bukan 4,7%) karena seri sisa terbesar dibagikan menurut nilai lalu urutan.

### 7.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Chip skenario | `scenarios[]` | ★ pada `is_active`; urut `total_target`; terpilih = `pickScenario` |
| Kotak centang sesi | `checkedSessions(sc, sessions)` | sesi non-sistem; `[+ Sesi]` membuat sesi lewat `POST projects/:id/sessions` ([01 BE §3.3](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi)); centang terakhir tidak bisa dilepas (`toActiveSubEvents` = `null`) |
| Tombol paket | `presets[]` | `fixed`: label `formatRupiahShort(total_target)`, ★ bila `scenario.preset_key` sama; `percent`: satu tombol "Persentase standar" yang meminta total (bawaan `target_budget` proyek, minimal Rp 100.000) |
| Isian target + slider | `scenario.total_target` | `parseRupiah` saat `blur`; slider 1 jt–200 jt langkah 1 jt hanya kenyamanan (di atas itu lewat isian); mengubah target **tidak** menskala pos |
| Donut | `groupScenarioItems` + `scenarioShares` | satu irisan per pos `included`; warna dari token; tooltip nominal dan persen; teks `Terdistribusi {allocated} dari {total_target} (sisa {unallocated})`, `unallocated < 0` → "melebihi plafon Rp X" |
| Tabel pos | `groupScenarioItems` | accordion per sesi (Umum terakhir) dengan subtotal; nama inline, nominal inline, `✏️` membuka `ScenarioItemModal`, `🗑️`, "↺ nilai preset" bila `canResetToPreset` |
| Keadaan kosong | `scenarios = []` | "Belum ada skenario" + tombol paket (pernikahan: enam paket; jenis lain: persentase) dan "Buat skenario kosong" |
| Sesi dicentang tetapi tanpa pos | – | "Belum ada pos untuk sesi ini" + `[+ Tambah pos di sesi ini]` (template pos per sesi tersedia di Tab 3, bab 14 butir 3) |

### 7.3 Kontrol

1. **`ScenarioSwitcher`**: maksimal 3; `[+]` nonaktif dengan tooltip "Maksimal 3 skenario. Hapus salah satu untuk membuat yang baru" saat `!canAddScenario` (juga bila server menjawab `max_scenarios_reached`). Menu per skenario: ganti nama (`updateScenario` dengan semua kolom dari skenario saat ini), duplikasi (nonaktif bila penuh; nama bawaan "{nama} (salinan)" dipotong 100 karakter), hapus (konfirmasi; skenario aktif menambah kalimat "Pagu vendor akan memakai pagu manual sampai Anda menerapkan skenario lain").
2. **`SubEventSelector`**: kotak centang; uncheck menyembunyikan pos sesi itu dari skenario dan perhitungannya.
3. **`BudgetPresetBar`**: memilih paket pada skenario yang sudah punya pos meminta konfirmasi ("Ganti semua pos skenario dengan paket {X}? Pos yang sudah Anda ubah hilang.") lalu `loadPreset({ preset_key, total_target?, confirm: true })`; skenario kosong tanpa konfirmasi.
4. **`ScenarioItemsTable`**: ubah, hapus, tambah; nominal dengan `inputmode="numeric"`, berformat ribuan saat `blur`.
5. **`ScenarioComparisonModal`**: dua `<select>` skenario; hasil `ScenarioComparison`: selisih `b − a` per baris, hijau bila turun (hemat) dan merah bila naik, total di footer.
6. **`ApplyScenarioModal`**: lihat 7.4.

### 7.4 Modal pratinjau terapkan

Memanggil `previewApply` setiap kali mode atau kotak centang berubah dan menampilkan hasilnya **sebelum** konfirmasi:

```text
Terapkan "Opsi Ideal 75 Jt" ke Realisasi?
  ✎ 9 pos diperbarui (estimasi saja)     Catering Resepsi 0 → 30.000.000, …
  ＋ 1 pos baru                          Dana darurat 1.000.000
  — 5 pos tidak tersentuh                 pos Lamaran, pos kustom
  ⚠ 0 pos dilewati                        (dihitung dari belanja / harga satuan)
Pembayaran, tanggal, bukti, dan vendor tidak diubah.
[x] Samakan pagu acara menjadi Rp 75.000.000     [x] Target tabungan ikut pagu
(•) Smart Adjust (disarankan)   ( ) Reset bersih — ketik RESET untuk menghapus semua pos & pembayaran
                                                               [ Batal ]  [ Terapkan ]
```

Mode **Reset bersih**: kotak "ketik RESET" (huruf besar persis), tombol aktif hanya bila cocok; pratinjau memakai `deleted` ("16 pos dan Rp 500.000 pembayaran akan dihapus") dan daftar `created` menggantikan `updated`. Sesudah `applyScenario` sukses: tutup modal, pindah ke `?tab=realisasi`. Laporan kosong (`updated` dan `created` kosong) → "Tidak ada yang berubah" dan tombol tetap boleh menekan (mengaktifkan skenario).

### 7.5 Alur interaksi (Tab 1)

| Alur | Langkah | Panggilan |
|---|---|---|
| A. Ganti skenario | klik chip → navigasi `?tab=simulasi&scenario=<id>` | – |
| B. Buat / duplikasi / ganti nama / hapus | modal nama (+ paket) → panggil → partial; hapus: `404` dianggap sukses | `createScenario`, `cloneScenario`, `updateScenario`, `deleteScenario` |
| C. Paket cepat | konfirmasi bila ada pos → panggil → partial | `loadPreset` |
| D. Sesi dan target | centang → `toActiveSubEvents` → `null` ditolak di klien; target `blur` → `parseRupiah` (null → galat kolom) → panggil → partial | `updateScenario` |
| E. Pos skenario | inline `blur`/Enter → kirim **semua** kolom pos (`sub_event_scope`, `category`, `item_name`, `cost_amount`); "↺" kirim `cost_amount = preset_amount` | `updateScenarioItem`, `addScenarioItem`, `deleteScenarioItem` |
| F. Bandingkan | pilih dua → panggil → tampilkan | `compareScenarios` |
| G. Terapkan | modal → `previewApply` (ulang bila opsi berubah) → `applyScenario` → `?tab=realisasi` | `previewApply`, `applyScenario` |

Galat yang ditutup dengan memuat ulang area: `not_found` pada skenario/pos skenario, `invalid_scope`, `conflict`.

### 7.6 Salinan teks

Judul "Simulasi & Alokasi Budget"; kosong: "Belum ada skenario. Pilih paket atau buat skenario kosong untuk mulai merencanakan."; `[+]` penuh: tooltip di atas; konfirmasi paket di 7.3; ringkasan donut di 7.2; modal terapkan di 7.4; galat dari bab 10. Label paket: `10 Jt` dst. dari `formatRupiahShort`; paket persentase: "Persentase standar {jenis}".

---

## 8. Tab 2: Kas & Tabungan Acara

> **Batasan:** Adicara bukan bank/escrow/dompet digital, tidak punya rekening penampung/VA, dan tidak menampung dana. Tab ini hanya buku catatan bersama; spanduk batasan ini tampil di atas tab. Angpau/amplop digital dari tamu **tidak** dicatat di sini.

### 8.1 Wireframe (Proyek Contoh)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [1 Simulasi]   [★ 2 Kas & Tabungan]   [3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌──────────────────┐                │
│ │ 🎯 Target      │ │ 💰 Terkumpul   │ │ ⏳ Kekurangan  │ │ 📈 Capaian       │                │
│ │ Rp 75.000.000  │ │ Rp 47.500.000  │ │ Rp 27.500.000  │ │ 63,3%  [━━━●───] │                │
│ │ ikut pagu ✏️   │ │ Arif 63% · Sinta 37% │ Rp 4,58 Jt/bln │ │                  │                │
│ └────────────────┘ └────────────────┘ └────────────────┘ └──────────────────┘                │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 💡 Disarankan menabung Rp 4.584.000/bulan (Rp 2.292.000/orang) selama 6 bulan menuju Hari-H.   │
│    ✅ Lamaran (15 Okt 2026) Rp 7.150.000 aman · ✅ Akad (28 Mar 2027) Rp 600.000 aman          │
│    🟡 Resepsi (28 Mar 2027) Rp 66.400.000: terkumpul Rp 39.750.000 (59,9%)                    │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 👥 KONTRIBUTOR                                                    [ + Tambah Kontributor ]    │
│ ┌──────────────────────────────────────┐ ┌─────────────────────────────────────────────────┐ │
│ │ Arif (Mempelai Pria)                 │ │ Sinta (Mempelai Wanita)                         │ │
│ │ Rp 30.000.000 · 63,2% · 11 setoran   │ │ Rp 17.500.000 · 36,8% · 11 setoran              │ │
│ └──────────────────────────────────────┘ └─────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Tampilan: (•) Matriks Bulanan  ( ) Daftar Setoran                [ + Catat Uang Masuk 💵 ]    │
│ ┌───────────┬───────────────┬───────────────┬───────────────┐                                │
│ │ Bulan     │ Arif (Pria)   │ Sinta (Wanita)│ Jumlah        │   Grafik tren saldo kumulatif   │
│ │ Februari  │     2.500.000 │       500.000 │     3.000.000 │   menuju garis target 75 jt     │
│ │ Maret     │     5.000.000 │     2.000.000 │     7.000.000 │   dengan penanda sesi:          │
│ │ April     │     1.500.000 │     1.000.000 │     2.500.000 │   ▲ Lamaran  ▲ Akad/Resepsi     │
│ │ …         │               │               │               │                                 │
│ │ Desember  │     6.000.000 │     4.000.000 │    10.000.000 │                                 │
│ ├───────────┼───────────────┼───────────────┼───────────────┤                                 │
│ │ TOTAL     │    30.000.000 │    17.500.000 │    47.500.000 │                                 │
│ └───────────┴───────────────┴───────────────┴───────────────┘                                 │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Contoh ini sengaja memuat setoran November–Desember 2026 yang, pada contoh yang dibuat Oktober 2026, belum terjadi: PDF mencatatnya begitu dan server menghitungnya sebagai terkumpul ([BE bab 9 butir 7](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

### 8.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Target | `kpi.target`, `kpi.target_follows_budget` | "ikut pagu" bila `true`; pensil membuka isian rupiah: nilai → `setSavingsTarget(n)`, kosong → `setSavingsTarget(null)` ("ikut pagu"); tidak ada `PATCH /savings/target` |
| Terkumpul / Kekurangan | `kpi.collected`, `kpi.shortfall` | porsi di bawah terkumpul: `contributorShares(overview, 0)` ("Arif 63% · Sinta 37%") |
| Capaian | `percent(collected, target)` | `formatPercent(…, 1)`; `target = 0` → "–" |
| "Rp 4,58 Jt/bln" | `recommendation.recommended_monthly` | `formatRupiahShort` |
| Saran | `recommendation`, `milestones` | `per_contributor` hilang → hanya total; milestone: ✅ bila `is_secured`, 🟡 bila sebagian (`milestoneRatio`); tanggal `formatDateShort(session_date)`; `unscheduled_estimate > 0` → ajakan "Isi tanggal sesi untuk target bertahap" |
| Kartu kontributor | `contributors`, `contributorShares(overview, 1)` | peran dari `roles` (label registry) |
| Matriks | `matrix.months`, `matrix.rows`, `contributors` | bulan `monthLabel`; kolom = urutan `contributors`; baris TOTAL dijumlah klien dari `rows[].total`; sel `0` ditampilkan "–" |
| Daftar | `entries` | kolom Tanggal, Uang Masuk, Penyetor, Aksi; footer TOTAL = Σ `entries` yang tampil; chip penyetor |
| Grafik tren | `matrix.rows` (kumulatif per akhir bulan), `kpi.target`, `milestones[].session_date` | garis saldo, garis target, penanda vertikal pada tanggal sesi; tanpa setoran → "Belum ada setoran" |
| Proyek nyata baru | kosong | tanpa kontributor dan setoran; ajakan "Tambah kontributor pertama"; tidak ada tombol "reset ke contoh" |

### 8.3 Komponen dan modal

1. **`SavingsKpiCards`**: empat kartu (bab 8.2).
2. **`ContributorCards`**: nama, peran, total, porsi, jumlah setoran; tambah/ubah/hapus (hapus ditolak bila punya setoran: `contributor_in_use`, tombol Hapus dinonaktifkan bila `entry_count > 0` dengan tooltip).
3. **`SavingsMatrix`** (tampilan bawaan) dan **`SavingsLedger`**: dua tampilan, pilihan di `sessionStorage`. Sel matriks mengikuti `cellAction`: kosong → modal setoran dengan tanggal `YYYY-MM-01` bulan itu dan penyetor kolom itu; tepat satu setoran → modal sunting setoran itu; beberapa → pindah ke Daftar tersaring penyetor dan bulan itu.
4. **`SavingsEntryModal`**: tanggal (bawaan hari ini di zona proyek, `todayIn`), nominal (`parseRupiah`, integer rupiah, > 0), penyetor (dropdown kontributor + `[+ Daftarkan baru]` yang membuat kontributor lebih dulu), catatan pribadi/lokasi simpanan (≤ 2.000 karakter). Tanggal setoran bebas (tidak harus tanggal 1) dan boleh mendatang.
5. **`ContributorModal`**: nama (≤ 100), peran (dropdown `contributor_roles`, bawaan `other`).
6. **`SavingsTrendChart`**: SVG murni, warna dari token.

### 8.4 Pemeriksaan sebelum mengirim

Nominal `null` dari `parseRupiah` → "Nominal tidak valid." tanpa permintaan; tanggal kosong → "Tanggal wajib diisi."; penyetor kosong → "Pilih penyetor."; sisanya dibiarkan ke server dan dipetakan (bab 10.2).

### 8.5 Alur interaksi (Tab 2)

| Alur | Langkah | Panggilan |
|---|---|---|
| H. Target | pensil → isian → simpan → partial | `setSavingsTarget` |
| I. Kontributor | modal → panggil → partial; hapus: konfirmasi; `409 contributor_in_use` → teks | `createContributor`, `updateContributor`, `deleteContributor` |
| J. Setoran | modal / sel matriks / baris Daftar → panggil → partial; hapus: konfirmasi | `createSaving`, `updateSaving`, `deleteSaving` |
| K. Saring penyetor | chip → navigasi `?tab=tabungan&contributor=<id>` (hanya `entries` yang tersaring; KPI dan matriks tetap seluruh proyek) | `getSavings(id)` |
| L. Ganti tampilan | radio → `hidden` + `sessionStorage` | – |

---

## 9. Tab 3: Realisasi Anggaran & Pembayaran

### 9.1 Wireframe (dataset PDF)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [1 Simulasi]   [2 Kas & Tabungan]   [★ 3 Realisasi]                                           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ 🌊 Kas terkumpul Rp 47.500.000 │ Terbayar Rp 33.550.000 │ Kas bebas Rp 13.950.000             │
│ ⚠ Sisa kewajiban Rp 40.600.000 melebihi kas bebas: butuh tambahan kas Rp 26.650.000           │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐ ┌──────────────────┐ ┌───────────────────┐ ┌───────────────────┐         │
│ │ Total Estimasi  │ │ Total Terbayar   │ │ Sisa Tagihan      │ │ Rasio Pelunasan   │         │
│ │ Rp 74.150.000   │ │ Rp 33.550.000    │ │ Rp 40.600.000     │ │ 45,2%  (ring)     │         │
│ │ Lam 9,6% Akad   │ │ DP 20,05 Jt      │ │ 🟢 7 lunas        │ │ Skenario aktif:   │         │
│ │ 0,8% Res 89,6%  │ │ T1 12 Jt T2 1,5  │ │ 🟡 7 berjalan     │ │ 75 Jt (sisa       │         │
│ │                 │ │                  │ │ 🔴 0 belum        │ │ plafon Rp 850 rb) │         │
│ └─────────────────┘ └──────────────────┘ └───────────────────┘ └───────────────────┘         │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Sesi: [Semua (14)] [💍 Lamaran (5)] [📜 Akad (1)] [🏛️ Resepsi (8)]                             │
│ [+ Catat Pembayaran] [+ Tambah Pos] [📥 Ekspor .xlsx] [Terapkan Template ▾]                   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ MATRIKS SESI           │ Lamaran      │ Akad       │ Resepsi       │ Total                    │
│ Tagihan Awal           │    7.150.000 │    600.000 │    66.400.000 │    74.150.000            │
│ DP / Lunas             │    3.250.000 │    600.000 │    16.200.000 │    20.050.000            │
│ Termin 1               │    1.000.000 │          0 │    11.000.000 │    12.000.000            │
│ Termin 2               │      500.000 │          0 │     1.000.000 │     1.500.000            │
│ Total Dibayar          │    4.750.000 │    600.000 │    28.200.000 │    33.550.000            │
│ Tagihan Sisa           │    2.400.000 │          0 │    38.200.000 │    40.600.000            │
│ Status                 │ 🟡 66,4%     │ ✅ LUNAS   │ 🟡 42,5%      │ 🟣 45,2%                 │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Sesi    Pos                    Estimasi      DP      Termin 1  Termin 2   Dibayar     Sisa  Status│
│  1 Lamaran Catering              2.000.000   500.000 1.000.000   500.000 2.000.000        0  LUNAS │
│  2 Lamaran Gaun/Kebaya             750.000   750.000         0         0   750.000        0  LUNAS │
│  3 Lamaran Fotografer            1.000.000   500.000         0         0   500.000  500.000  DP    │
│  4 Lamaran Cincin                2.500.000   600.000         0         0   600.000 1.900.000 DP    │
│  5 Lamaran Dekorasi                900.000   900.000         0         0   900.000        0  LUNAS │
│  6 Akad    Biaya KUA               600.000   600.000         0         0   600.000        0  LUNAS │
│  7 Resepsi Mahar                 3.000.000 3.000.000         0         0 3.000.000        0  LUNAS │
│  8 Resepsi Catering             25.000.000 5.000.000 10.000.000        0 15.000.000 10.000.000 SEBAGIAN│
│  9 Resepsi Rias & Baju Pengantin 5.000.000   600.000         0         0   600.000 4.400.000  DP    │
│ 10 Resepsi Sewa Gedung & Dekor. 20.000.000   900.000         0         0   900.000 19.100.000 DP    │
│ 11 Resepsi Undangan & souvenir   3.500.000 1.500.000         0         0 1.500.000 2.000.000  DP    │
│ 12 Resepsi Dokumentasi           3.000.000   300.000         0         0   300.000 2.700.000  DP    │
│ 13 Resepsi Seserahan 🧺 belanja  3.500.000 3.500.000         0         0 3.500.000        0  LUNAS │
│ 14 Resepsi Hiburan & Pengisi     3.400.000 1.400.000 1.000.000 1.000.000 3.400.000        0  LUNAS │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 🧺 = "Dihitung dari Belanja" bila pos digulung dari daftar belanja (Menu 04); kolom bayarnya terkunci.
```

Baris 13 bertanda 🧺 hanya setelah Menu 04 menggulung belanja ke pos itu; pada dataset PDF murni, `rollup_source` kosong dan pembayarannya bisa diubah.

### 9.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| `CashflowBar` | `getCashflow` | `Kas terkumpul = total_savings_collected`, `Terbayar = total_paid`, `Kas bebas = available_cash` (boleh negatif, merah); peringatan bila `cash_deficit > 0`: "Sisa kewajiban {remaining_debt} melebihi kas bebas: butuh tambahan kas {cash_deficit}"; selain itu pesan aman |
| Kartu 1 | `kpi.total_estimate`, `columns[].row.estimate` | mikro-teks proporsi sesi = `sharePercents(estimasi per kolom, total_estimate)` (9,6 / 0,8 / **89,6**) |
| Kartu 2 | `kpi.total_paid`, `total.dp/term_1/term_2` | "DP 20,05 Jt · T1 12 Jt · T2 1,5 Jt" dengan `formatRupiahShort` |
| Kartu 3 | `kpi.total_remaining`, `line_status` | pill: 🟢 lunas = `settled`, 🟡 berjalan = `partial`, 🔴 belum = `unpaid` (hanya pos berestimasi > 0); `total_overpaid > 0` → baris "Lebih bayar {rupiah}" |
| Kartu 4 | `percent(total_paid, total_estimate)`, `active_scenario` | ring + `formatPercent(…, 1)`; `active_scenario` → "Skenario aktif: {nama} (sisa plafon {unallocated})", negatif → "melebihi plafon {rupiah}"; `null` → "Belum ada skenario aktif" |
| Chip sesi | `countLines(lines, known)` | "Semua (14)", per sesi dari daftar yang sedang dimuat |
| `SessionMatrix` | `matrixColumns`, `matrix.total` | kolom = sesi yang punya pos (Umum terakhir) + Total; baris Status = `formatPercent(settlement, 1)`, 100% → "LUNAS"; `null` (estimasi 0) → "–" |
| `BudgetTable` | `lines`, `lineNumbers`, `paymentBadge` | kolom No (nomor tampilan, bukan `sort_order`), Sesi, Pos, Estimasi, DP, T1, T2, Dibayar, Sisa, Status, Aksi; `remaining < 0` merah "Lebih bayar"; penanda sumber *Dari vendor* (`vendor_id`), *Dari dokumen* (`admin_doc_id`), *Dihitung dari belanja* (`rollup_source`); pos berharga satuan menampilkan "{quantity} {unit_label} × {unit_price}" |
| Keadaan kosong | `items = []` | "Belum ada pos anggaran. Terapkan template atau tambah pos." |

### 9.3 Komponen dan modal

1. **`CashflowBar`**, **`BudgetKpiCards`**, **`SessionMatrix`**: bab 9.2.
2. **`BudgetTable`**: tabel 14+ kolom; pada layar sempit menjadi kartu (bab 11.2). Pos `rollup` menampilkan angka dari server dan tombol bayar nonaktif (`rollup_locked`).
3. **`PaymentDrawer`**: pilih pos (`<select>` dengan pencarian), tahap (`STAGE_LABELS`), nominal dengan tombol "Bayar sisa (Rp X)" (mengisi tahap kosong pertama dan mengganti tahap menjadi `full`), tanggal (bawaan hari ini), tautan bukti (http/https), **pratinjau sisa sebelum/sesudah** dari `paymentImpact` (`replaces > 0` → "Mengganti Rp X"; `remainingAfter < 0` → "Lebih bayar Rp X"). Galat `locked`/`no_stage`/`nothing_left` menonaktifkan tombol kirim dengan keterangan. Tombol "Hapus pembayaran tahap ini" muncul bila tahap terisi (konfirmasi → `clearPayment`). Pada tahap `full`, nominal tidak bisa diedit (server memakai sisa).
4. **`BudgetItemModal`**: sesi (dropdown master sesi), kategori (dropdown kategori yang sudah dipakai + isian bebas, ≤ 60), nama pos (≤ 150), mode harga (*Total* / *Jumlah × harga satuan* dengan satuan; tombol "Gunakan proyeksi RSVP" bila `rsvpEnabled`, mengisi `quantity` dari `recommended_portions` dan satuan "porsi" atas aksi pengguna), vendor terkait (opsional; **mengisi awal dari `vendor_id` baris dan mengirimnya kembali**, bab 4.1), catatan. Estimasi `Total` memakai `parseRupiah`; mode satuan menampilkan hasil `round(jumlah × harga)` hanya sebagai petunjuk (server yang menghitung). Pos `rollup`: estimasi dan mode harga dinonaktifkan dengan keterangan "Dihitung dari daftar belanja". Catatan tetap di bawah sesi: "Bila sesi tujuan sudah punya pos dengan kunci yang sama, pos ini menjadi pos biasa."
5. **`TemplateMenu`** ("Terapkan Template ▾"): *Tambah pos yang belum ada* (`append`, boleh per sesi: opsi "hanya sesi {X}") dan *Ganti pos template yang belum dibayar* (`replace_template`): hitung `templateImpact`; `needsConfirm` → dialog yang menyebut jumlah pos dihapus, yang berisiko, dan yang dilewati karena sudah dibayar; hasil `ApplyTemplateResult` ditampilkan dengan `warnings` (`financeWarningText`). Server `confirm: required` yang tidak diperkirakan lokal → dialog yang sama.
6. **Ekspor**: tautan `exportHref` (bab 6.2).

### 9.4 Pemeriksaan sebelum mengirim

Nominal/estimasi `null` dari `parseRupiah` → galat kolom tanpa permintaan; `amount` ≤ 0 pada tahap bukan `full` → "Nominal harus lebih dari 0."; tanggal bayar kosong → "Tanggal bayar wajib diisi."; tautan bukti tidak berawalan `http://`/`https://` → pesan 10.2; sisanya dibiarkan ke server.

### 9.5 Alur interaksi (Tab 3)

| Alur | Langkah | Panggilan |
|---|---|---|
| M. Chip sesi | klik → `hidden` pada baris lain + `?scope=` (`history.replaceState`) | – |
| N. Bayar | tombol baris / "Catat Pembayaran" → drawer → `recordPayment` → partial | `recordPayment` |
| O. Hapus pembayaran | drawer → konfirmasi → `clearPayment` → partial | `clearPayment` |
| P. Tambah / ubah pos | modal → `createLine` / `updateLine` (semua kolom) → partial | `createLine`, `updateLine` |
| Q. Hapus pos | konfirmasi (menyebut "pembayaran Rp X ikut hilang" bila `paid > 0`) → `deleteLine`; `404` dianggap sukses | `deleteLine` |
| R. Urut ulang | Naik/Turun atau seret → daftar id lengkap (`moveBefore`/`moveByOne`) → `reorderLines` | `reorderLines` |
| S. Template | menu → `templateImpact` → konfirmasi bila perlu → `applyTemplate` | `applyTemplate` |
| T. Ekspor | tautan | `exportHref` |

#### 9.5.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Dipakai untuk |
|---|---|---|
| Baris / kartu pos | `data-line-id`, `data-scope`, `data-rollup`, `data-paid`, `data-remaining`, `data-focus-key` | chip sesi, tombol bayar nonaktif, pratinjau, fokus setelah partial |
| Sel matriks bulanan | `data-month`, `data-contributor` | `cellAction` |
| Kartu skenario / pos skenario | `data-scenario-id`, `data-item-id` | aksi inline |
| Blok data | `<script type="application/json" data-finance-data>` | daftar pos / setoran untuk pratinjau drawer dan `cellAction`; `<` di-escape `<` dan dibaca dengan `JSON.parse(textContent)` |

---

## 10. Pesan galat dan teks

Semua teks lewat `lib/planner/messages.ts` (01 FE bab 5.4 dan 11): kode lama tetap, berikut tambahan menu ini. **Pola pemakaian** sama dengan 02: fungsi Menu 03 mengembalikan `null` bila bukan miliknya, dan pemanggil jatuh ke fungsi 01:

```ts
const text = financeErrorMessage(res.status, res.error?.code, 'line') ?? plannerErrorMessage(res.status, res.error?.code);
const fieldText = financeFieldMessage(field, reason) ?? fieldErrorMessage(field, reason);
```

### 10.1 Kode → teks

| Kode / kondisi | Teks Indonesia | Fungsi |
|---|---|---|
| `max_scenarios_reached` | "Maksimal 3 skenario. Hapus salah satu untuk membuat yang baru." | `financeErrorMessage` |
| `scenario_item_limit_reached` | "Jumlah pos dalam skenario sudah mencapai batas (100)." | idem |
| `budget_item_limit_reached` | "Jumlah pos anggaran sudah mencapai batas (300). Hapus pos yang tidak dipakai lalu coba lagi." | idem |
| `contributor_limit_reached` | "Jumlah kontributor sudah mencapai batas (20)." | idem |
| `savings_entry_limit_reached` | "Jumlah setoran sudah mencapai batas (2.000)." | idem |
| `contributor_in_use` | "Kontributor ini masih punya setoran. Hapus atau pindahkan setorannya dulu." | idem |
| `vendor_already_linked` | "Vendor ini sudah tertaut ke pos anggaran lain." | idem |
| `rollup_locked` | "Pos ini dihitung dari daftar belanja, jadi pembayarannya dicatat lewat belanja." | idem |
| `invalid_stage` | "Semua tahap pembayaran pos ini sudah terisi, atau tidak ada sisa untuk dilunasi." | idem |
| `invalid_scope` | "Sesi tidak valid atau sudah dihapus. Muat ulang halaman." | idem |
| `invalid_template` | "Paket atau template tidak tersedia untuk jenis acara ini. Muat ulang halaman." | idem |
| `not_found`, `where` = `scenario` / `scenario_item` / `contributor` / `saving` / `line` | "Skenario / Pos skenario / Kontributor / Setoran / Pos tidak ditemukan. Muat ulang daftar." | idem |
| `not_found`, `where = 'project'` | "Acara tidak ditemukan." (halaman: `/404`) | 01 |
| `rate_limited`, `where = 'export'` | "Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi." | idem |
| `invalid_input`, `invalid_date` | **per kolom** (10.2); tanpa kolom: "Periksa kembali data Anda." | 01 |
| `invalid_json`, `project_read_only`, `conflict`, `401`, `5xx`, jaringan | sama dengan [01 FE bab 11.1](../01_DASHBOARD/FE.md#111-kode--teks-plannererrormessagestatus-code) | 01 |

`where` ditentukan oleh **aksi** yang sedang berjalan, bukan oleh pesan server.

### 10.2 Alasan per kolom

Alasan yang sudah ada di 01 FE bab 11.2 (wajib/`must not be empty`, panjang, tanggal valid, karakter kontrol, `must be an integer`, `must be at least N`) tetap berlaku. Tambahan Menu 03 (`financeFieldMessage`; **setiap alasan baku BE §4.5 sudah tercakup atau diserahkan eksplisit**, diuji):

| Pola alasan (regex) | Kolom | Teks (`{label}` = label kolom, 10.3) |
|---|---|---|
| `^must be one of the event type roles$` | `role_label` | "Peran tidak valid untuk jenis acara ini." |
| `^must be a contributor of this project$` | `contributor_id` | "Penyetor tidak ditemukan di acara ini. Muat ulang daftar." |
| `^must be a vendor of this project$` | `vendor_id` | "Vendor tidak ditemukan di acara ini. Muat ulang daftar vendor." |
| `^must match the preset total$` | `total_target` | "Target harus sama dengan total paket yang dipilih." |
| `^must be at least (\d+)$` | `total_target` saja | "{label} minimal Rp 100.000." (kolom lain: tabel 01) |
| `^must be between 0 and (\d+)$` | kolom uang | "{label} paling banyak Rp 10.000.000.000.000." |
| `^must be between ([\d.]+) and (\d+)$` | `quantity` saja | "Jumlah harus antara 0,01 dan 1.000.000." |
| `^must be greater than 0$` | `amount` dll. | "{label} harus lebih dari 0." |
| `^required when pricing_mode is unit$` | `quantity`, `unit_price` | "{label} wajib diisi untuk harga per satuan." |
| `^the estimate \(quantity × unit_price\) must be at most (\d+)$` | `unit_price` | "Estimasi (jumlah × harga satuan) paling banyak Rp 10.000.000.000.000." |
| `^must be an http or https URL$` | `receipt_url` | "Tautan bukti harus diawali http:// atau https://." |
| `^must be within the project schedule window$` | `paid_at`, `entry_date` | "{label} harus dalam rentang tiga tahun sebelum sampai setahun sesudah acara." |
| `^must be RESET$` | `confirm` | "Ketik RESET (huruf besar) untuk melanjutkan." |
| `^must list every id exactly once$` | `ids` | "Urutan sudah usang karena daftar berubah. Muat ulang halaman." |
| `^must differ from ` | `b` | "Pilih dua skenario yang berbeda." |
| `^must be one of ` | `mode`, `payment_stage`, `pricing_mode`, … | "{label} tidak valid." (kesalahan klien: kontrol yang salah) |
| `confirm: required` | `confirm` | bukan teks galat: tampilkan kotak konfirmasi dan kirim ulang dengan `confirm` (bab 7.5, 9.5) |
| alasan lain | – | "{label} tidak valid." |

### 10.3 Label kolom

`FINANCE_FIELD_LABELS` (kode di 10.4): `name` "Nama", `new_name` "Nama baru", `total_target` "Target", `cost_amount` "Nominal", `item_name` "Nama pos", `category` "Kategori", `sub_event_scope` "Sesi", `active_sub_events` "Sesi yang disimulasikan", `preset_key` "Paket", `role_label` "Peran", `contributor_id` "Penyetor", `entry_date` "Tanggal", `amount` "Nominal", `notes` "Catatan", `initial_estimate` "Estimasi", `quantity` "Jumlah", `unit_label` "Satuan", `unit_price` "Harga satuan", `pricing_mode` "Mode harga", `vendor_id` "Vendor", `payment_stage` "Tahap", `paid_at` "Tanggal bayar", `receipt_url` "Tautan bukti", `confirm` "Konfirmasi", `mode` "Mode", `ids` "Urutan", `scenario_id` "Skenario", `savings_target` "Target tabungan", `a` "Skenario A", `b` "Skenario B". Kolom tak dikenal: "Isian".

### 10.4 Kode `messages.ts` (tambahan)

Label status dan tahap, teks peringatan, label kolom, dan dua fungsi. **Sudah dijalankan** ([bab 12.1](#121-unit-testsplanner-financetestmjs)).

```ts
import { formatRupiah } from './format.ts';
import type { PaymentStage } from './types.ts';
import type { Badge } from './finance.ts';

// ---------- Tambahan Menu 03 untuk messages.ts (pola: pemanggil memakai `…Message(…) ?? kamus 01`) ----------

/** Aksi yang memanggil: menentukan teks `not_found` dan `rate_limited`, yang tidak dapat dibedakan dari kode/pesan server. */
export type Where = 'project' | 'scenario' | 'scenario_item' | 'contributor' | 'saving' | 'line' | 'export';

export const PAYMENT_BADGE_LABELS: Record<Badge, string> = {
  unpaid: 'Belum', dp_paid: 'DP', partial: 'Sebagian', settled: 'Lunas', overpaid: 'Lebih bayar', unbudgeted: 'Belum dianggarkan',
};
export const STAGE_LABELS: Record<PaymentStage | 'full', string> = {
  dp: 'DP / Lunas', term_1: 'Termin 1', term_2: 'Termin 2', full: 'Pelunasan Penuh',
};
/** Kategori pos untuk lencana sumber pada BudgetTable. */
export const SOURCE_LABELS = { vendor: 'Dari vendor', admin_doc: 'Dari dokumen', shopping: 'Dihitung dari belanja' } as const;

/** Tambahan untuk WARNING_LABELS (01). `message` dari server berbahasa Inggris dan tidak pernah ditampilkan. {count} diganti `count`. */
export const FINANCE_WARNING_LABELS: Record<string, string> = {
  scope_fallback: '{count} pos dipindah ke Umum karena sesinya sudah dihapus.',
  shared_budget_item: 'Pos ini mewakili beberapa vendor, jadi estimasinya tidak diubah.',
};
export const financeWarningText = (code: string, count: number): string =>
  (FINANCE_WARNING_LABELS[code] ?? 'Ada catatan pada hasil ini.').replace('{count}', String(count));

export const FINANCE_FIELD_LABELS: Record<string, string> = {
  name: 'Nama', new_name: 'Nama baru', total_target: 'Target', cost_amount: 'Nominal', item_name: 'Nama pos', category: 'Kategori',
  sub_event_scope: 'Sesi', active_sub_events: 'Sesi yang disimulasikan', preset_key: 'Paket', role_label: 'Peran',
  contributor_id: 'Penyetor', entry_date: 'Tanggal', amount: 'Nominal', notes: 'Catatan', initial_estimate: 'Estimasi',
  quantity: 'Jumlah', unit_label: 'Satuan', unit_price: 'Harga satuan', pricing_mode: 'Mode harga', vendor_id: 'Vendor',
  payment_stage: 'Tahap', paid_at: 'Tanggal bayar', receipt_url: 'Tautan bukti', confirm: 'Konfirmasi', mode: 'Mode',
  ids: 'Urutan', scenario_id: 'Skenario', savings_target: 'Target tabungan', a: 'Skenario A', b: 'Skenario B',
};

type Rule = readonly [RegExp, (m: RegExpExecArray, label: string, field: string) => string | null]; // null = lanjut ke aturan berikutnya

/** Alasan `<kolom>: <alasan>` milik Menu 03. Urutan penting: pola khusus lebih dulu. */
const FINANCE_REASONS: readonly Rule[] = [
  [/^must be one of the event type roles$/, () => 'Peran tidak valid untuk jenis acara ini.'],
  [/^must be a contributor of this project$/, () => 'Penyetor tidak ditemukan di acara ini. Muat ulang daftar.'],
  [/^must be a vendor of this project$/, () => 'Vendor tidak ditemukan di acara ini. Muat ulang daftar vendor.'],
  [/^must match the preset total$/, () => 'Target harus sama dengan total paket yang dipilih.'],
  [/^must be at least (\d+)$/, (m, l, f) => (f === 'total_target' ? `${l} minimal ${formatRupiah(Number(m[1]))}.` : null)], // kolom lain: tabel 01
  [/^must be between 0 and (\d+)$/, (m, l) => `${l} paling banyak ${formatRupiah(Number(m[1]))}.`],
  [/^must be between ([\d.]+) and (\d+)$/, (m, l, f) => (f === 'quantity' ? `${l} harus antara ${m[1]!.replace('.', ',')} dan ${Number(m[2]).toLocaleString('id-ID')}.` : null)],
  [/^must be greater than 0$/, (_m, l) => `${l} harus lebih dari 0.`],
  [/^required when pricing_mode is unit$/, (_m, l) => `${l} wajib diisi untuk harga per satuan.`],
  [/^the estimate \(quantity × unit_price\) must be at most (\d+)$/, (m) => `Estimasi (jumlah × harga satuan) paling banyak ${formatRupiah(Number(m[1]))}.`],
  [/^must be an http or https URL$/, () => 'Tautan bukti harus diawali http:// atau https://.'],
  [/^must be within the project schedule window$/, (_m, l) => `${l} harus dalam rentang tiga tahun sebelum sampai setahun sesudah acara.`],
  [/^must be RESET$/, () => 'Ketik RESET (huruf besar) untuk melanjutkan.'],
  [/^must list every id exactly once$/, () => 'Urutan sudah usang karena daftar berubah. Muat ulang halaman.'],
  [/^cannot be combined with /, () => 'Isian saling bertentangan.'],
  [/^must be one of /, (_m, l) => `${l} tidak valid.`],
  [/^must differ from /, () => 'Pilih dua skenario yang berbeda.'],
];

/** null = bukan alasan Menu 03: pakai fieldErrorMessage 01 (wajib, panjang, tanggal valid, karakter kontrol, angka). */
export function financeFieldMessage(field: string, reason: string): string | null {
  const label = FINANCE_FIELD_LABELS[field] ?? 'Isian';
  for (const [re, text] of FINANCE_REASONS) {
    const m = re.exec(reason);
    const out = m ? text(m, label, field) : null;
    if (out !== null) return out;
  }
  return null;
}

/**
 * Kode galat Menu 03 yang teksnya bergantung konteks. null = pakai plannerErrorMessage 01 (invalid_json, conflict,
 * project_read_only, 401, 5xx, jaringan). `not_found` dibedakan oleh AKSI pemanggil (`where`), bukan oleh pesan server.
 */
export function financeErrorMessage(status: number, code: string | undefined, where?: Where): string | null {
  if (where === 'export' && (status === 429 || code === 'rate_limited')) return 'Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi.';
  switch (code) {
    case 'max_scenarios_reached': return 'Maksimal 3 skenario. Hapus salah satu untuk membuat yang baru.';
    case 'scenario_item_limit_reached': return 'Jumlah pos dalam skenario sudah mencapai batas (100).';
    case 'budget_item_limit_reached': return 'Jumlah pos anggaran sudah mencapai batas (300). Hapus pos yang tidak dipakai lalu coba lagi.';
    case 'contributor_limit_reached': return 'Jumlah kontributor sudah mencapai batas (20).';
    case 'savings_entry_limit_reached': return 'Jumlah setoran sudah mencapai batas (2.000).';
    case 'contributor_in_use': return 'Kontributor ini masih punya setoran. Hapus atau pindahkan setorannya dulu.';
    case 'vendor_already_linked': return 'Vendor ini sudah tertaut ke pos anggaran lain.';
    case 'rollup_locked': return 'Pos ini dihitung dari daftar belanja, jadi pembayarannya dicatat lewat belanja.';
    case 'invalid_stage': return 'Semua tahap pembayaran pos ini sudah terisi, atau tidak ada sisa untuk dilunasi.';
    case 'invalid_scope': return 'Sesi tidak valid atau sudah dihapus. Muat ulang halaman.';
    case 'invalid_template': return 'Paket atau template tidak tersedia untuk jenis acara ini. Muat ulang halaman.';
    case 'not_found':
      switch (where) {
        case 'scenario': return 'Skenario tidak ditemukan. Muat ulang daftar.';
        case 'scenario_item': return 'Pos skenario tidak ditemukan. Muat ulang daftar.';
        case 'contributor': return 'Kontributor tidak ditemukan. Muat ulang daftar.';
        case 'saving': return 'Setoran tidak ditemukan. Muat ulang daftar.';
        case 'line': return 'Pos tidak ditemukan. Muat ulang daftar.';
        default: return null;
      }
    default: return null;
  }
}
```

Teks `warnings`: `scope_fallback` (Buat skenario, Template), `shared_budget_item` (kunci vendor, dari Menu 04) dan `budget_link_lost` (Template, teks dari 02 FE bab 10.4). Kode tak dikenal: "Ada catatan pada hasil ini."; `message` Inggris dari server tidak pernah ditampilkan.

---

## 11. Gaya, responsif, aksesibilitas, keamanan

### 11.1 Gaya

Hanya token yang ada ([01 FE bab 12.1](../01_DASHBOARD/FE.md#121-gaya)): kartu `--color-surface`, batas `--color-border`; judul `--font-display`, isi `--font-body`; lencana mengikuti pola `.status` di `dashboard/index.astro`. Status pos: Lunas terisi `--sage-600`, DP dan Sebagian `--sage-300`, Belum hanya kontur `--color-border`; *Lebih bayar* dan defisit kas memakai warna bahaya yang sudah dipakai kode (`#8e2f25`). Setiap status juga **teks** (Lunas / DP / Sebagian / Belum / Lebih bayar), bukan hanya warna. Selisih perbandingan skenario: hijau untuk turun, merah untuk naik, dengan tanda `+`/`−` dan teks. Spanduk batasan Tab 2: `--sage-50` dengan batas `--sage-200`. Jangan menambah token baru tanpa persetujuan.

### 11.2 Responsif

- `< 768px`: KPI grid 2×2; matriks sesi (Tab 3) *scroll* horizontal dengan kolom kiri *sticky*; tabel pos menjadi *Financial Expense Cards* berprogress bar (`paid/estimate`); drawer pembayaran dan semua dialog penuh layar; matriks bulanan Tab 2 menjadi daftar per bulan (kartu bulan berisi baris per kontributor); donut di atas daftar pos Tab 1; kolom Tab 1 menjadi satu.
- Target sentuh minimal 44×44 px (chip, ⋮, pegangan seret, sel matriks).
- Tidak ada gulir horizontal pada lebar 360 px kecuali di dalam wadah matriks; uji dengan nama pos 150 karakter dan nominal 14 digit (`overflow-wrap: anywhere`, angka `white-space: nowrap` di wadah yang bisa digulir).

### 11.3 Aksesibilitas

- Dialog `<dialog>` dengan `showModal()`; fokus kembali ke pemicu saat ditutup; formulir: `<label>`, `aria-invalid`, `aria-describedby`, ringkasan galat `role="alert"`.
- Tab bar `role="tablist"` dengan `aria-selected`; chip sesi dan chip penyetor `aria-pressed`.
- Seret-lepas urut ulang selalu punya padanan papan ketik (Naik/Turun).
- Hasil asinkron (menyimpan, hasil terapkan, "Data belum diperbarui") di wilayah `aria-live="polite"`; baris yang sedang dikirim `aria-busy="true"`.
- Donut dan grafik tren punya padanan teks/tabel (daftar pos dengan persen; matriks bulanan); balok tidak membawa informasi hanya lewat warna.
- Pratinjau drawer (`remainingAfter`) di wilayah `aria-live`.
- Partial mengganti DOM: fokus dipulihkan lewat `data-focus-key`; bila elemennya hilang, ke judul area.
- Kontras dan indikator fokus terlihat; hormati `prefers-reduced-motion`.

### 11.4 Keamanan

- **Tidak ada HTML dari data:** nama skenario, pos, kontributor, vendor, catatan, dan nama sesi dirender sebagai teks (`textContent`/`createElement`; Astro meng-escape di server). Blok JSON `data-finance-data` ditulis dengan `<` di-escape (`<`); tidak ada `set:html` untuk data API.
- Tautan bukti pembayaran dirender hanya bila berawalan `http://` atau `https://`, dengan `rel="noopener noreferrer"` dan `target="_blank"`; selain itu ditampilkan sebagai teks.
- Path dibangun hanya dari `id` yang sudah divalidasi UUID; query lewat `URLSearchParams`.
- CSRF: BFF menolak non-GET tanpa `application/json`. Token tetap di cookie HttpOnly. Tidak ada `localStorage`; `sessionStorage` hanya untuk pilihan tampilan setoran.
- Data keuangan bersifat pribadi: respons ekspor `Cache-Control: no-store`; tidak ada nominal di URL.

---

## 12. Pengujian

Pola yang sudah ada: `node:test` di `frontend/tests/*.test.mjs`, dijalankan `npm test` (`node --experimental-strip-types --test tests/*.test.mjs`). Fixture di `frontend/tests/fixtures/` (bab 4.3).

### 12.1 Unit: `tests/planner-finance.test.mjs`

Lima belas tes di bawah **sudah dijalankan**: kode bab 4.2, 5, 6, dan 10.4 (plus 1.4) dikompilasi bersama `types.ts` dan `format.ts` dari 01 FE dan fixture bab 4.3. Hasil: `tsc` strict (`noUncheckedIndexedAccess`) bersih, 15/15 lulus. Mutasi yang dicoba pada kode (**19 dari 20 menjatuhkan tes**): sisa terbesar dimatikan; seri sisa tanpa nilai; `parseRupiah` menerima koma; urutan tahap tertukar; `full` memakai `amount` klien; lebih bayar tidak dikenali; `'all'` dianggap semua; `included` diabaikan; sel tanpa saringan kontributor; rasio sesi tanpa `null`; Umum tidak di akhir; `clearPayment` memakai `PATCH`; id tidak divalidasi; pengecualian `total_target` dibuang; dampak template tanpa estimasi; "tidak ada tercentang" dikirim `[]`; sesi sistem ikut tercentang; skenario aktif diabaikan; `templateImpact` menghitung pos yang sudah dibayar. Satu mutan **selamat** karena setara: membuang terjemahan `must be one of the event type roles` jatuh ke aturan `must be one of` yang menghasilkan "Peran tidak valid." (tidak ada kata Inggris bocor).

| Tes | Masukan → keluaran (fixture Proyek Contoh) |
|---|---|
| fixture | 14 pos: Σ 74.150.000 / 33.550.000 / 40.600.000, status 7 / 6 / 1 / 0, tanpa `*_paid_at`; matriks, kas, tabungan (22 setoran, rekomendasi), 3 skenario (75 Jt aktif, `allocated` = target), 6 preset |
| matriks | tiap baris = jumlah pos per sesi; `paid = dp + term_1 + term_2 + rollup_paid`; `estimate − paid = remaining − overpaid`; `line_count` |
| `parseRupiah` | `75000000`, `75.000.000`, `Rp 75.000.000`, `rp75.000.000`, `Rp. 600.000`, `1 500 000`, `0`, `007`, 10¹³ → angka; `''`, `Rp`, `abc`, `1,5`, `1.5`, `1.50`, `1.5000`, `1,500,000`, `-5`, `+5`, `1e3`, 10¹³+1, 20 digit, `12.34.567`, `.500`, `1..500`, angka lebar penuh → `null` |
| `sharePercents` | skenario 75 Jt `[0,8 · 6,7 · 40 · 6,7 · 26,7 · 4,7 · 4 · 4,6 · 4,5 · 1,3]` berjumlah 100,0; sesi `[9,6 · 0,8 · 89,6]`; kontributor `[63,2 · 36,8]` dan `[63 · 37]`; `[1,1,1]` → `[33,4 · 33,3 · 33,3]`; `[1,1,4]/6` 0 desimal → `[17,16,67]`; penyebut 0 → `null`; bagian tidak habis → dibulatkan sendiri; 10¹³ tanpa kehilangan presisi; 300 kasus acak: jumlah 100,0 dan tiap bagian < 0,1 dari nilai pasti |
| skenario | `scenarioShares`, `unallocated` (0 dan −1.500.000), `groupScenarioItems` (Akad 1 / Resepsi 8 / Umum 1, subtotal 600.000 / 73.400.000 / 1.000.000; pos tidak `included` tidak digambar), `canResetToPreset`, `canAddScenario` |
| `pickScenario`, `checkedSessions`, `toActiveSubEvents` | id dikenal / tidak / null / tanpa aktif / kosong; 75 Jt → `akad, resepsi`; `[]` → semua; semua → `[]`; sebagian → daftar menurut urutan sesi; **tak ada → `null`** |
| `paymentBadge` | `settled` / `dp_paid` / `partial`; `overpaid`; `unbudgeted`; `unpaid`; semua label ada |
| `nextEmptyStage`, `paymentImpact` | Fotografer: `full` → T1 500.000, sesudah 0; `term_2` 200.000 → sisa 300.000; `dp` 300.000 mengganti 500.000 → sisa 700.000; `term_1` 900.000 → −400.000; semua terisi → `no_stage`; DP = estimasi → `nothing_left`; rollup → `locked` |
| `filterLines`, `countLines`, `lineScope`, `lineNumbers`, `replaceLine`, `removeLine` | `{* 14, lamaran 5, akad 1, resepsi 8}`; `'all'` → 0; sesi usang → Umum; nomor 1…14 walau `sort_order` berlubang; masukan tidak berubah |
| `matrixColumns` | kolom Lamaran / Akad Nikah / Resepsi; rasio 66,4% / 100% / 42,5%, total 45,2%; `all` → "Umum" dengan rasio `null`; sesi hilang → kuncinya |
| tabungan | `monthLabel`, porsi 63,2 / 36,8, milestone 100% / 100% / 59,9%, `cellAction` (edit / create / list; kontributor lain tidak ikut), `by_contributor` lengkap |
| `templateImpact` | Proyek Contoh `{0, 14, 0, tanpa konfirmasi}`; template segar tanpa konfirmasi; estimasi / vendor / dokumen / rollup → konfirmasi; pos kustom tidak ikut; `[]` |
| `financeApi` | ke-32 fungsi → metode + path + body + query persis (tabel 4.4); id bukan UUID (`../etc`, huruf besar, `abc`, kosong, `…/../x`) melempar; `exportHref` |
| katalog galat | setiap alasan baku BE §4.5 punya teks (tanpa kata Inggris) atau `null` eksplisit untuk 01; `Target minimal Rp 100.000.`, `Nominal paling banyak Rp 10.000.000.000.000.`, `Jumlah harus antara 0,01 dan 1.000.000.` |
| `financeErrorMessage`, `financeWarningText` | semua kode baru bertext; `not_found` per `where`; `rate_limited` hanya untuk ekspor; kode 01 → `null`; `{count}` diganti |

### 12.2 Tes sumber (`tests/planner-wiring.test.mjs`, perluasan dari 01 FE bab 13.3)

- Halaman `keuangan/index.astro` dan `live.astro` memuat `export const prerender = false`; `live.astro` memuat `export const partial = true`; keduanya memakai komponen yang sama (tidak ada markup tab di halaman).
- `index.astro` memakai `PlannerLayout`, bukan `AppLayout`.
- Tidak ada literal sesi atau peran tertulis di `components/planner/finance` dan halamannya: `'lamaran'`, `'akad'`, `'resepsi'`, `'groom'`, `'bride'` (hanya `types.ts` dan tes yang boleh). `'all'` dan `SCOPE_ANY` boleh.
- Tidak ada `innerHTML =`, `insertAdjacentHTML`, atau `set:html` yang memuat data API di komponen dan `financeClient.ts`.
- Tidak ada `plannerFetch('projects/` atau `fetch('/dashboard/api/planner` di komponen: semua panggilan lewat `financeApi`.
- Tidak ada `parseFloat`, `Number(` atas isian uang, atau `toFixed` di komponen dan `financeClient.ts` (uang hanya lewat `parseRupiah`/`format`).
- Tidak ada pembagian persen tertulis di komponen (`/ ... * 100`): semua lewat `percent`/`sharePercents`.
- `PlannerLayout.astro` menandai `keuangan` sebagai `ready`; `ROUTES.md` memuat rute halaman dan partial.

### 12.3 Uji komponen dengan fixture (manual atau skrip)

Render tiap tab dengan: fixture penuh; proyek kosong; variasi bab 4.3 (lebih bayar, estimasi 0 terbayar, rollup, satuan, melebihi plafon, tanpa skenario aktif, sel dengan dua setoran, sesi tanpa tanggal, sesi usang, nama 150 karakter); jenis seminar (peran `internal`/`sponsor`/`other`, preset persentase); proyek `archived` (semua kontrol ubah nonaktif, ekspor aktif); lebar 360 px.

### 12.4 QA manual (terhadap backend asli)

**Belum bisa dijalankan** sampai backend menu ini ada. Daftar yang harus lulus, dipetakan ke alur:

1. Proyek Contoh: 14 pos, KPI 74.150.000 / 33.550.000 / 40.600.000, rasio 45,2%, `7 lunas · 7 berjalan · 0 belum`, Cashflow Bar defisit 26.650.000, skenario aktif 75 Jt dengan sisa plafon Rp 850.000.
2. Tab 1: tiga skenario; `[+]` nonaktif pada skenario ke-3; paket 100 Jt pada skenario berisi pos meminta konfirmasi; centang terakhir tidak bisa dilepas; ubah nominal pos → donut dan persen berubah (berjumlah 100,0); plafon melebihi → pesan "melebihi plafon".
3. Terapkan 75 Jt ke proyek baru berisi 14 pos template: pratinjau "9 diperbarui, 1 baru, 5 tak tersentuh"; terapkan lagi → "Tidak ada yang berubah"; berpindah ke 100 Jt berulang kali tanpa galat (`23505` pada aktivasi tidak boleh muncul).
4. Reset bersih: tombol nonaktif sampai mengetik `RESET`; pratinjau menyebut pos dan rupiah pembayaran yang hilang.
5. Tab 2: kartu, rekomendasi 4.584.000 / 2.292.000, milestone 100% / 100% / 59,9%; matriks 11 bulan; bulan tanpa setoran tetap muncul; klik sel kosong / satu setoran / dua setoran; hapus kontributor bersetoran ditolak; ubah target lalu kosongkan (ikut pagu).
6. Tab 3: bayar `DP`, `Termin 1`, `Pelunasan Penuh` (nominal terkunci, sisa terisi), bayar melebihi sisa (lebih bayar tampil), ganti tahap yang terisi ("Mengganti Rp X"), hapus pembayaran; pos `rollup` tidak bisa dibayar; klik ganda "Bayar sisa" tidak membuat dua pembayaran.
7. Pos berharga satuan: jumlah × harga; nilai raksasa (9.999.999.999,99 × 10¹³) ditolak dengan pesan, bukan galat 500; "Gunakan proyeksi RSVP" (setelah Menu 05).
8. Pindah pos ke sesi yang sudah punya kunci yang sama (mis. Catering Lamaran → Resepsi): berhasil, kunci hilang, tidak ada galat.
9. Template: `append` dua kali (kedua kali `created 0`); `replace_template` pada proyek yang estimasinya diisi skenario meminta konfirmasi dan melewati pos terbayar.
10. Ekspor `.xlsx`: nama berkas, tiga lembar, sel berawalan `=` tampil sebagai teks; permintaan ke-11 dalam semenit → pesan batas.
11. Arsipkan: kontrol ubah nonaktif, ekspor aktif; tulis dari tab lain → `project_read_only` dan kontrol menjadi nonaktif.
12. Dua tab: hapus pos di tab A, ubah pos itu di tab B → "Pos tidak ditemukan. Muat ulang daftar."
13. Sesi habis lalu klik aksi → diarahkan ke login. Mobile 360 px, papan ketik saja, pembaca layar ringkas.

### 12.5 Verifikasi kontrak (cara mengulang)

Dilakukan 2026-10-21 terhadap blok DTO Go di [BE bab 5](BE.md#5-dto-go) dan tipe bab 4.2 (skrip ada di scratchpad sesi, bukan di repo). **Setelah backend menu ini ada**, ganti langkah 1–2 dengan panggilan nyata seperti [02 FE bab 11.5](../02_TIMELINE_DAN_TUGAS/FE.md#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang).

1. **Kompilasi** blok Go dalam modul sementara bernama sama dengan modul backend (stub `Notice` identik dengan yang asli), `go vet`.
2. **Ambil skema** dengan program `reflect` kecil: untuk tiap struct DTO, jalur JSON bersarang (`items[].preset_amount`), `omitempty` (opsional), pointer tanpa `omitempty` (nullable), dan jenis nilai.
3. **Parse** interface dari blok 4.2 (subset TS: tipe primitif, literal union, `T[]`, `Record`, `| null`, objek bersarang) menjadi jalur yang sama dan bandingkan: kunci di satu sisi saja, opsionalitas, `null`, jenis nilai. Tipe permintaan: kunci dan jenis saja.
4. **Uji diri pemeriksa**: ubah tipe TS dengan sengaja (opsional → wajib, kunci dibuang, `| null` dibuang, jenis diganti, kunci diganti nama, kunci berlebih) dan pastikan dilaporkan. Enam mutasi dicoba; semuanya dilaporkan.
5. **Cocokkan fixture** dengan SQL: jumlah pos, matriks, kas, tabungan, dan skenario di `fixtures/` harus sama dengan hasil kueri BE bab 3 pada Proyek Contoh dengan jam 4 Okt 2026.

Hasil 2026-10-21: 29 pasangan tipe, 329 jalur, **0 selisih**. Yang **tidak** terbukti: bentuk respons HTTP sungguhan (nama kunci dan kode galat diambil dari dokumen BE, bukan dari server), dan galat `5xx`. Dua butir yang menunggu Menu 04 teramati 2026-10-23: pos tertaut vendor membawa `vendor_id` dan `vendor_name`, dan pos `rollup` dari belanja nyata (Proyek Contoh) membawa `rollup_source: "shopping"`, estimasi 4.578.500 dan terbayar 2.063.800, dengan pembayarannya ditolak `409 rollup_locked`.

---

## 13. Urutan pengerjaan dan definisi selesai

### 13.1 Menjalankan lingkungan lokal

Backend menu ini sudah ada ([BE.md](BE.md)), tetapi halaman Keuangan belum ada, jadi yang dapat dijalankan sekarang hanyalah pustaka acuan dan tesnya:

```bash
cd frontend && npm install
npm test            # setelah tes dan fixture bab 4.3 / 12.1 disalin: node --experimental-strip-types --test tests/*.test.mjs
npx tsc --noEmit    # types.ts + finance.ts + financeApi.ts + messages.ts + financeProps.ts di bawah konfigurasi strict repo
```

Setelah halaman dibangun: jalankan seperti [02 FE bab 12.1](../02_TIMELINE_DAN_TUGAS/FE.md#121-menjalankan-lingkungan-lokal) (`make run` untuk backend, `API_BASE_URL=http://localhost:8080 npm run dev` untuk frontend). Proyek Contoh dari `/dashboard/planner` ("Lihat Proyek Contoh") sudah berisi 14 pos dengan pembayarannya, 22 setoran, dan tiga skenario karena overlay `finance` terdaftar ([BE §3.12](BE.md#312-sambungan-ke-menu-01)).

### 13.2 Tahapan

| # | Tahap | Isi | Selesai bila |
|---|---|---|---|
| T0 | Prasyarat | Menu 01 FE: M0 (tipe, `format`, `messages`), M1 (BFF + `client`), `PlannerLayout`, `404.astro` | `npm test` 01 hijau; `GET /dashboard/api/planner/event-types` membalas |
| T1 | Fondasi menu ini | blok bab 4.2 ke `types.ts` (+ dua perubahan satu baris, bab 1.3), `finance.ts` (bab 5), `financeApi.ts` (bab 6.1), blok `messages.ts` (bab 10.4), `financeProps.ts` (1.4), `tests/fixtures/*` (4.3), tes 12.1 | `npm test` dan `tsc` hijau; belum perlu backend. Kode di dokumen ini tinggal disalin: sudah lulus kedua pemeriksaan itu |
| T2 | Tab 3 (Realisasi) | `index.astro` + `live.astro`, `FinanceTabs`, `CashflowBar`, `BudgetKpiCards`, `SessionMatrix`, `BudgetTable`, `PaymentDrawer`, `financeClient.ts` (alur M, N, O) | uji komponen 12.3 (Tab 3); QA 1, 6, 11–13 |
| T3 | Tab 3 lanjutan | `BudgetItemModal`, `TemplateMenu`, urut ulang, hapus, ekspor (alur P–T) | QA 7–10 |
| T4 | Tab 2 (Tabungan) | `SavingsKpiCards`, `SavingsAdvice`, `ContributorCards`, `SavingsMatrix`, `SavingsLedger`, modal, `SavingsTrendChart` (alur H–L) | QA 5 |
| T5 | Tab 1 (Simulasi) | `ScenarioSwitcher`, `SubEventSelector`, `BudgetPresetBar`, `BudgetDonutChart`, `ScenarioItemsTable`, modal, `ApplyScenarioModal` (alur A–G) | QA 2–4 |
| T6 | Kualitas | aksesibilitas, responsif, `astro check`, `npm run build`, `format:check`, tes sumber 12.2 | build bersih; QA 13 |

Urutan Tab 3 → Tab 2 → Tab 1 mengikuti keterbacaan nilai (Tab 3 berdiri sendiri pada data pos; Tab 1 butuh pos dan sesi) dan agar backend bisa dibangun bertahap ([BE bab 8](BE.md#8-urutan-pengerjaan-dan-definisi-selesai)). Karena Menu 03 membaca `FinanceSummary` yang sudah ada di dashboard, widget Keuangan (01 FE bab 9.4) tidak ikut dikerjakan di sini.

**Definisi selesai (seluruh Menu 03 FE):** semua tahap lulus; `npm test`, `npm run check`, `npm run build`, `npm run format:check` bersih; tes sumber 12.2 lulus; verifikasi kontrak 12.5 diulang terhadap backend asli (nol selisih); `keuangan.ready = true` di `PlannerLayout`; `ROUTES.md`, `docs/frontend/planner/` dan dokumen ini diperbarui bila perilaku berubah.

---

## 14. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01 dan 02 (kode dan dokumen), kontrak, PDF, dan SQL yang dijalankan. Dokumen mengikuti pilihan yang tertulis.

1. **Kontrak belum dibuktikan terhadap HTTP asli.** Tipe cocok dengan struct Go backend yang sudah jadi (bab 4.2: 29 pasangan, 329 jalur, nol selisih), tetapi fixture masih dibangun dari basis data, bukan disalin dari respons. Backend kini ada: ulangi bab 12.5 dengan respons asli.
2. **`PUT` pos mengganti penuh** dan `vendor_id` yang tidak dikirim melepas vendor ([BE bab 9 butir 11](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). FE mengisi awal dan mengirim kembali; alternatif `PATCH` + `clear_vendor` seperti Menu 02.
3. **Template pos per sesi di Tab 1 tidak ada.** Versi awal menjanjikan "mencentang sesi menawarkan pos template (estimasi 0)" pada skenario, tetapi tidak ada endpoint yang menambahkan item skenario dari template, dan FE tidak punya daftar template (bukan bagian registry). Pos template per sesi tersedia di Tab 3 (`apply-template` dengan `sub_event_scope`). Bila produk ingin di Tab 1, tambahkan `POST /scenarios/:id/items/from-template`. Ajakan "Alokasikan sisa plafon ke Resepsi" juga dibuang (bisa dilakukan lewat ubah nominal).
4. **Kategori pos tanpa registry.** Dropdown kategori diisi kategori yang sudah dipakai proyek; proyek kosong hanya isian bebas ([BE bab 9 butir 9](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
5. **Proyeksi RSVP pada pos satuan mana pun** (tidak ada penanda "pos katering" di registry); bergantung pada Menu 05 (tombol disembunyikan bila endpoint tidak ada).
6. **Persentase berjumlah 100,0** dengan seri sisa terbesar (bab 5) berarti satu irisan bisa tampil 0,1 lebih rendah dari pembulatan biasa (Seserahan 4,6% bukan 4,7%). Bila produk lebih suka pembulatan biasa dan jumlah ≠ 100, ganti `sharePercents` dengan pembulatan per bagian; wireframe lama menjumlah 100,1% dan 99,9%.
7. **Slider target** (1 jt–200 jt) hanya kenyamanan; nilai lain lewat isian.
8. **Ekspor sebagai tautan** (teks 429 kurang ramah, tanpa `Retry-After`) atau `fetch` + Blob; pilih saat membangun.
9. **Satu sumber render dengan partial** (bab 2.4) belum dijalankan di browser; alternatif `location.reload()`.
10. **Tidak ada polling dan tidak ada kontrol konkurensi pada kolom yang sama** (terakhir menang); galat `not_found`/`invalid_scope`/`ids`/`conflict` memicu muat ulang area.
11. **Setoran mendatang dihitung terkumpul** (Proyek Contoh memuat Nov–Des 2026); milestone dan defisit mengikutinya ([BE bab 9 butir 7](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
12. **Kinerja browser belum diukur.** Server: 300 pos dan 2.000 setoran adalah batas tampilan, bukan batas kinerja basis data. Tabel pos dan Daftar setoran dirender sekaligus tanpa paginasi.
13. **Hari ini pada tanggal bawaan** dihitung saat render (`todayIn`) dan tidak bergeser melewati tengah malam tanpa muat ulang.
14. **Awalan `'` pada sel Excel** tampak sebagai karakter ([02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)); ekspor belum dibuka di aplikasi spreadsheet.
15. **Milestone bertarget 0 berstatus `is_secured: true`** (0 dari 0). Proyek baru yang sesinya bertanggal Hari-H memuat dua milestone tanpa dana di `GET /savings`. Pilihan: sembunyikan milestone bertarget 0, atau tampilkan "belum ada anggaran" ([BE bab 9 butir 25](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
16. **Bentuk yang ditetapkan backend** (BE bab 9 butir 22, 23, 27): `GET /contributors` mengirim array (`data: Contributor[]`); `apply`, `apply-preview`, dan `compare` menolak id kosong `400 required`; `PUT /savings/:id` untuk id proyek lain `404` sebelum isian diperiksa; `GET /budget/export` wajib `format=xlsx`; kategori pos template jenis non-pernikahan adalah usulan ([BE bab 9 butir 18](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 15. Riwayat perubahan dokumen ini

**2026-10-21 (disamakan dengan pola 01 dan 02 FE, dicocokkan dengan DTO backend dan SQL yang dijalankan):** dokumen diperluas dari ±200 baris menjadi acuan lengkap: daftar isi, route dan file, kontrak props, alur data dan galat, prasyarat, tipe TypeScript yang dicocokkan dengan DTO Go, fixture, katalog endpoint dan galat, `finance.ts`, `financeApi.ts`, `messages.ts` dengan kode yang sudah dijalankan, alur interaksi, pesan galat, aksesibilitas, pengujian, urutan pengerjaan. **Koreksi terhadap versi sebelumnya:** (1) layout `AppLayout`/`nav_active` diganti `PlannerLayout`; (2) `format.ts` tidak mengurai rupiah, `parseRupiah` ditambahkan; (3) wireframe dan aturan "persen berjumlah 100" saling bertentangan (Tab 1 menjumlah 100,1%, Tab 3 99,9%): `sharePercents` ditambahkan dan angka wireframe dikoreksi; (4) `PATCH /savings/target` dihapus, target lewat `PATCH` proyek Menu 01; (5) tidak ada cara membatalkan pembayaran: `DELETE …/payment/:stage` ditambahkan; (6) KPI "Skenario aktif / sisa plafon" tidak punya sumber di DTO: `BudgetMatrix.active_scenario` ditambahkan; (7) "Paket cepat" dan "kembalikan ke nilai preset" tidak punya endpoint/data: `load-preset` dan `preset_amount`; (8) ajakan template per sesi di Tab 1 dibuang (butir 3); (9) komponen yang tersirat tetapi tidak terdaftar (`SavingsEntryModal`, `ContributorModal`, `ScenarioItemModal`, `TemplateMenu`) didaftarkan; (10) sel Matriks Bulanan "bisa diedit" tidak terdefinisi bila satu bulan memuat beberapa setoran: `cellAction`; (11) `LineStatus` memakai `in_progress` padahal widget Menu 01 sudah `partial`; (12) persentase di respons backend bertentangan dengan kontrak bab 6, dihapus dari DTO.

**2026-10-21 (backend menu 03 ada):** status dan bab 14 diperbarui: kontrak bab 4 dicocokkan ulang dengan struct Go yang sebenarnya (29 pasangan, 329 jalur, nol selisih), temuan implementasi backend yang menyentuh FE ditambahkan sebagai bab 14 butir 15–16 (milestone bertarget 0, bentuk respons yang ditetapkan). Fixture belum diganti dengan respons HTTP asli.

**2026-10-21 (awal):** FE Menu 03 ditulis mengikuti pola 01 FE sebelum backend ada: tiga wireframe, komponen, responsif, dan daftar pengujian.
