# Menu 04: Vendor & Belanja Kebutuhan — Frontend Specification

Acuan pembangunan antarmuka dua tab: **Direktori & Kontak Vendor** dan **Belanja Pengadaan & Logistik** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: kontrak data, aturan tampilan, algoritma, alur interaksi, kasus galat, dan pengujian ada di sini. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); fondasi yang dibangun Menu 01 (tipe, `format.ts`, `messages.ts`, `client.ts`, BFF, `PlannerLayout`) dan `finance.ts` Menu 03 (`paymentBadge`, `SCOPE_ANY`) **dipakai apa adanya** dan dirujuk ke [01 FE](../01_DASHBOARD/FE.md) dan [03 FE](../03_KEUANGAN/FE.md), bukan diulang. Pola dokumen ini mengikuti [03 FE](../03_KEUANGAN/FE.md) dan [02 FE](../02_TIMELINE_DAN_TUGAS/FE.md).

> **Status (2026-10-23):** frontend Planner belum ada satu baris pun, tetapi **backend menu ini sudah ada** ([BE.md](BE.md), [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md)) dan kontrak di bab 4 dapat dicocokkan dengannya. Tipe TypeScript bab 4.2 dicocokkan dengan **blok DTO Go di [BE bab 4](BE.md#4-dto-go) yang dikompilasi** (skema JSON diambil dengan `reflect` dan dibandingkan kunci demi kunci dengan tipe TS memakai compiler API TypeScript): **26 pasangan tipe, 378 jalur JSON, nol selisih** pada kunci, opsionalitas, `null`, dan jenis nilai (pembanding diuji dengan enam mutasi tipe dan melaporkan semuanya); struct di kode backend sama dengan blok itu (26 struct, nama, tipe, dan tag JSON identik) dan sama dengan skema `backend/docs/openapi.yaml` (dijaga tes). Fixture ([bab 4.3](#43-fixture)) dibangun dari **Proyek Contoh yang dibuat backend Menu 01–03 asli** (vendor dan barang ditambahkan dari tabel [BE bab 5](BE.md#5-master-data)); setelah modul ini ada, respons `GET /vendors` dan `GET /shopping-items` pada Proyek Contoh dibandingkan dengan `planner-vendors.json` dan `planner-shopping.json`: **struktur, tipe, dan nilai sama persis** setelah id dinormalkan (jadi fixture kini sama dengan respons HTTP asli untuk kedua pembacaan itu). Menu ini bergantung pada Menu 01 FE (M0–M1 dan `PlannerLayout`) dan pada `finance.ts` Menu 03 FE ([bab 3](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-dan-03-fe)).
>
> Kode di bab 1.4, 4.2, 5, 6, dan 9.4 **sudah dijalankan**: `tsc` strict (dengan `noUncheckedIndexedAccess`, konfigurasi Astro yang dipakai repo) bersih bersama kode 01–03 yang diekstrak dari dokumennya; 19 tes `node:test` ([bab 11](#11-pengujian)) lulus terhadap fixture di atas, termasuk vektor `NormalizeWA` yang sama dengan backend dan 300 pasangan `round()` yang dihitung PostgreSQL; **26 dari 27 mutasi kode** yang dicoba menjatuhkan tes (satu mutan setara: pemeriksaan awalan `http` di `safeHref` tumpang-tindih dengan pemeriksaan protokol `URL`); semua 16 pembungkus `vendorApi` diuji metode, path, dan body-nya. Yang bergantung pada Astro atau browser (halaman, partial, dialog, unduhan) belum diuji dan ditandai *(belum diuji di browser)*.

> Angka pada wireframe berasal dari **Proyek Contoh** (dataset PDF Hal. 9–10 dan 12; nama vendor fiktif) yang dibuat 22 Okt 2026: Sinta & Arif, H-175, skenario aktif 75 Jt. Pagu Catering Rp 30.000.000 adalah hasil skenario 75 Jt ([BE §2.1](BE.md#21-pagu-vendor-dari-skenario-aktif)); versi awal wireframe menulis 25.000.000 dan status bayar Venue "DP 0,9 jt", yang keduanya tidak mungkin: pagu skenario untuk `catering` resepsi adalah 30 Jt, dan Venue berbagi pos dengan Dekorasi sehingga tidak bertaut sendiri. Wireframe adalah tata letak, bukan desain visual akhir: warna, jarak, dan tipografi memakai token yang sudah ada (bab 10.1). Emoji pada wireframe untuk **sesi** adalah emoji dari API; ikon **kontrol** memakai Phosphor dan tidak digambar.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat](#2-alur-data-dan-penanganan-galat)
3. [Prasyarat yang harus sudah ada dari Menu 01 dan 03 FE](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-dan-03-fe)
4. [Kontrak data: tipe, fixture, contoh respons, katalog galat](#4-kontrak-data-tipe-fixture-contoh-respons-katalog-galat)
5. [Library `lib/planner/vendor.ts`](#5-library-libplannervendorts)
6. [Pembungkus panggilan, BFF, dan unduhan](#6-pembungkus-panggilan-bff-dan-unduhan)
7. [Tab 1: Direktori & Kontak Vendor](#7-tab-1-direktori--kontak-vendor)
8. [Tab 2: Belanja Pengadaan & Logistik](#8-tab-2-belanja-pengadaan--logistik)
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
| Vendor & Belanja | `/dashboard/planner/[id]/vendor` | `src/pages/dashboard/planner/[id]/vendor/index.astro` | dua tab lewat `?tab=vendor` (default) dan `?tab=belanja`; nilai lain jatuh ke `vendor`. Tab dipilih server-side; bagian tab lain tidak dimuat. Parameter lain di bab 2.3 |
| Partial | `/dashboard/planner/[id]/vendor/live` | `src/pages/dashboard/planner/[id]/vendor/live.astro` (`export const partial = true`) | HTML area yang diperbarui setelah aksi tulis (bab 2.4); menerima `?tab=` dan parameter yang sama dengan halaman. Satu sumber render: memakai komponen yang sama dengan `index.astro` |
| Komponen | – | `src/components/planner/vendor/*.astro` | lihat 1.2 |
| Library | – | `src/lib/planner/vendor.ts` (murni, bab 5), `vendorApi.ts` (pembungkus panggilan, bab 6), `vendorProps.ts` (kontrak props, bab 1.4), `vendorClient.ts` (skrip klien halaman, bab 7.5 dan 8.5) | |
| Tes + fixture | – | `tests/planner-vendor.test.mjs`, `tests/fixtures/planner-{vendors,shopping,wa-vectors,line-totals}.json` (+ `planner-sessions.json` dari Menu 02) | bab 4.3 dan 11 |

Tidak ada file BFF baru: semua panggilan lewat `/dashboard/api/planner/[...path].ts` dari Menu 01 (bab 6). Halaman memakai `PlannerLayout` (bilah sekunder berisi Project Switcher dan sub-navigasi), **bukan** `AppLayout` langsung (versi awal dokumen ini menulis `AppLayout` dan `nav_active = 'planner'`, yang tidak ada di desain 01 FE bab 10).

### 1.2 Komponen

Props lengkap (TypeScript) di bab 1.4. Kolom "Klien" menyebut perilaku yang dipasang skrip halaman; alurnya di bab 7.5 dan 8.5.

| Komponen | Props utama | Dipakai di | Klien |
|---|---|---|---|
| `VendorTabs` | `active` | tab bar | navigasi penuh (`?tab=`) |
| `VendorKpiCards` | `kpi`, `missing` | Tab 1 | – |
| `CategoryChecklist` | `categories`, `activeCategory`, `readOnly` | Tab 1 | klik chip → saring kategori; "Tandai tidak diperlukan" / "Diperlukan lagi" → `setSkippedCategories` |
| `VendorFilters` | `counts`, `sessions`, `scope`, `status` | Tab 1 | chip sesi dan status (atribut `hidden`, `?scope=`/`?status=`) |
| `VendorTable` | `rows`, `sessions`, `project`, `readOnly` | Tab 1 | WhatsApp (`waLink`), ubah, ubah status, kunci FIX, hapus; braket kelompok pagu; pada layar sempit menjadi kartu |
| `LimitGroupSummary` | `group` | Tab 1 | baris ringkasan di bawah anggota kelompok: pagu, penawaran, selisih, status bayar pos bersama |
| `VendorModal` | `categories`, `customCategories`, `sessions`, `defaultScope` | Tab 1 | tambah / ubah vendor (`categoryPayload`, `normalizeWA`) |
| `LockDealModal` | `vendor`, `group` | Tab 1 | `lockDeal`; menampilkan hasil dan peringatan |
| `ShoppingKpiCards` | `kpi` | Tab 2 | – |
| `RollupBanner` | `target`, `rollups`, `readOnly` | Tab 2 | `linkToBudget` (dengan konfirmasi), `unlinkFromBudget` |
| `ShoppingFilters` | `items`, `sessions`, `scope`, `category`, `status` | Tab 2 | chip sesi, kategori, status |
| `ShoppingTable` | `items`, `numbers`, `sessions`, `readOnly` | Tab 2 | toggle status inline, ubah, hapus; tautan alamat (`safeHref`); pada layar sempit menjadi kartu |
| `ShoppingItemModal` | `sessions`, `categories`, `vendors`, `rollupTargetId`, `defaultScope` | Tab 2 | tambah / ubah barang (`parseQuantity`, `lineTotal`) |
| `ShoppingTemplateMenu` | `type`, `items`, `project` | Tab 2 | tambah yang kurang / ganti template (`shoppingTemplateImpact`) |
| `ConfirmDialog` | `title`, `message`, `confirmLabel`, `danger` | semua | satu `<dialog>` dipakai bersama (dari 02/03 FE; bila belum ada, buat di sini) |

### 1.3 Perubahan pada file yang sudah ada

| File | Perubahan | Mengapa |
|---|---|---|
| `src/layouts/PlannerLayout.astro` | `MENUS`: `vendor.ready = true` | satu bendera; menu lain tidak berubah ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) |
| `src/lib/planner/types.ts` | tambah blok bab 4.2; ubah `export interface EventTypeInfo {` menjadi `… extends EventTypeVendorInfo {` (bila Menu 02 dan 03 FE sudah ada: `extends EventTypeAdminInfo, EventTypeFinanceInfo, EventTypeVendorInfo`); pada `UpdateProjectBody` tambah `skipped_vendor_categories?: string[]` | kontrak menu ini. Backend Menu 01 **sudah** menerima kolom itu (`UpdateProjectRequest`), tetapi blok tipe 01 FE bab 4.2 belum memuatnya |
| `src/lib/planner/messages.ts` | tambah blok bab 9.4 (label status, teks peringatan, alasan per kolom, galat berkonteks) | teks galat |
| `src/lib/api/backend.ts` dan BFF | `backendFetchRaw` dan aliran berkas untuk `…/vendors/export-contacts` dan `…/shopping-items/export-shopping-list` | satu-satunya respons non-JSON; sama dengan perubahan 02 FE bab 1.3 dan 03 FE bab 1.3 (kerjakan sekali) |
| `ROUTES.md` | baris halaman `/dashboard/planner/[id]/vendor` dan partial `…/vendor/live` | file itu minta diperbarui bersama setiap halaman baru |
| Dropdown vendor di 02 FE (modal tugas) dan 03 FE (`BudgetItemModal`) | memakai `vendorApi.listVendors()` dan tipe `Vendor` dari bab 4.2 (`id`, `vendor_name`, `category_label`); menu tanpa Menu 04 menyembunyikan kolom | bentuk respons `GET …/vendors` kini ditetapkan ([02 FE bab 13 butir 2](../02_TIMELINE_DAN_TUGAS/FE.md#13-celah-dan-keputusan-yang-menunggu-pemilik-produk)) |
| `docs/frontend/planner/` | dokumen bertanggal dan entri `CHANGELOG.md` | aturan repo: kode dan dokumen satu paket |
| (tidak diubah) | `AppLayout.astro`, `middleware.ts`, `tokens.css`, `global.css` | |

`astro check` (bagian dari `npm run build`) harus lolos; TypeScript `strict` dan `noUncheckedIndexedAccess` aktif (akses `array[i]` bertipe `T | undefined`).

### 1.4 Kontrak props

Disalin ke `interface Props` tiap komponen (atau diimpor dari `vendorProps.ts`); **sudah dikompilasi** `tsc` strict bersama bab 4.2 dan 5. `VendorRow` dan `VendorCounts` dari bab 5; objek biasa (bukan `Map`) untuk `numbers` karena props SSR tidak membawa `Map`: `Object.fromEntries(itemNumbers(items))`.

```ts
import type { VendorCounts, VendorRow } from './vendor.ts';
import type {
  CategoryState, EventTypeInfo, LimitGroup, RollupLine, RollupTarget, SessionBrief, ShoppingItem, ShoppingKPI, Vendor, VendorCategory, VendorKPI,
} from './types.ts';

export type VendorTab = 'vendor' | 'belanja';

/** Konteks proyek yang dibaca halaman dari GET projects/:id dan diteruskan ke komponen (bukan dibaca ulang). */
export interface VendorProject {
  id: string; title: string; timezone: string; archived: boolean; eventTypeKey: string; skippedCategories: string[];
}

export interface VendorTabsProps { active: VendorTab }
export interface VendorKpiCardsProps { kpi: VendorKPI; missing: CategoryState[] }
export interface CategoryChecklistProps { categories: CategoryState[]; activeCategory: string | null; readOnly: boolean }
export interface VendorFiltersProps { counts: VendorCounts; sessions: SessionBrief[]; scope: string; status: string }
export interface VendorTableProps { rows: VendorRow[]; sessions: SessionBrief[]; project: VendorProject; readOnly: boolean }
export interface LimitGroupSummaryProps { group: LimitGroup }
export interface VendorModalProps {
  categories: VendorCategory[]; customCategories: CategoryState[]; sessions: SessionBrief[]; defaultScope: string;
}
export interface LockDealModalProps { vendor: Vendor; group: LimitGroup | null }
export interface ShoppingKpiCardsProps { kpi: ShoppingKPI }
export interface RollupBannerProps { target: RollupTarget; rollups: RollupLine[]; readOnly: boolean }
export interface ShoppingFiltersProps { items: ShoppingItem[]; sessions: SessionBrief[]; scope: string; category: string; status: string }
export interface ShoppingTableProps { items: ShoppingItem[]; numbers: Record<string, number>; sessions: SessionBrief[]; readOnly: boolean }
export interface ShoppingItemModalProps {
  sessions: SessionBrief[]; categories: string[]; vendors: Pick<Vendor, 'id' | 'vendor_name' | 'category_label'>[];
  rollupTargetId: string | null; defaultScope: string;
}
export interface ShoppingTemplateMenuProps { type: EventTypeInfo; items: ShoppingItem[]; project: VendorProject }
```

`VendorProject` dibentuk satu kali di frontmatter halaman dari `GET projects/:id`: `archived = detail.project.status === 'archived'`, `skippedCategories = project.skipped_vendor_categories`, `title`, `timezone`, `eventTypeKey = project.event_type`. `readOnly` pada komponen = `project.archived`.

---

## 2. Alur data dan penanganan galat

```text
SSR (baca)    : halaman → backendFetch('/v1/planner/…', { accessToken }) → HTML
Interaksi     : <script> klien → vendorApi(plannerFetch, id) → BFF /dashboard/api/planner/… → Go API
Pembaruan     : sesudah aksi tulis → fetch('/dashboard/planner/:id/vendor/live?tab=…') → HTML partial → ganti [data-vendor-live]
Unduhan       : <a href="/dashboard/api/planner/projects/:id/vendors/export-contacts?format=xlsx"> → BFF (aliran berkas)
Registry      : lib/planner/eventTypes.ts → GET /v1/planner/event-types (cache memori proses 1 jam)
```

`backendFetch` mengembalikan seluruh payload di `result.data`, jadi isi respons ada di `result.data.data`.

### 2.1 Pemuatan per tab

| Tab | Panggilan (paralel dengan `Promise.all`) | Catatan |
|---|---|---|
| `vendor` | `GET projects/:id/vendors` + `GET projects/:id/sessions` + `GET projects/:id` + registry | satu panggilan memberi KPI, kategori, kelompok pagu, dan vendor; `sessions` memberi chip dan nama sesi (sesi sistem `all` ditampilkan "Umum"); `GET projects/:id` memberi `status` (arsip), `skipped_vendor_categories`, `title`, `event_type`; registry memberi `vendor_categories` untuk modal. Penyaringan dilakukan di klien dari daftar penuh (bab 2.4) |
| `belanja` | `GET projects/:id/shopping-items` + `GET projects/:id/sessions` + `GET projects/:id` + registry | `shopping_categories` dari registry untuk dropdown (digabung dengan kategori yang sudah dipakai: `categoryOptions`). Daftar vendor untuk dropdown vendor rekanan dimuat **saat modal barang dibuka** (`vendorApi.listVendors()`); galat atau `404` → kolom Vendor disembunyikan. Serah terima souvenir dari Menu 05 dimuat bila Menu 05 ada (`GET projects/:id/guests/reception-overview`, [05 BE](../05_BUKU_TAMU/BE.md)); tidak tersedia → banner stok tidak ditampilkan |

Validasi `Astro.params.id` seperti 01 FE bab 2.1 (UUID huruf kecil, selain itu `Astro.rewrite('/404')` tanpa memanggil backend).

### 2.2 Hasil backend → tindakan halaman

| Hasil | Tindakan |
|---|---|
| `2xx` | render |
| `401` | `Astro.redirect('/auth/masuk')` |
| `404 not_found` pada proyek | `Astro.rewrite('/404')` (tidak membedakan "bukan milik Anda" dan "tidak ada") |
| `5xx` atau `BackendUnavailableError` | kerangka halaman + banner "Vendor & belanja sedang tidak tersedia" + tombol "Muat ulang"; **bukan 404** |
| `GET vendors`/`shopping-items` gagal, proyek berhasil | tab tetap tampil; isi diganti banner galat lokal dengan tombol "Muat ulang"; tab lain tidak ikut gagal |
| Registry gagal | pakai cache lama; bila tidak ada, dropdown kategori diganti isian teks bebas (kategori kustom) |

Aturan umum: `export const prerender = false`; teks galat ke pengguna selalu dari `lib/planner/messages.ts`, tidak pernah `error.message` mentah (pesan server berbahasa Inggris). Proyek `archived`: semua kontrol ubah dinonaktifkan dengan keterangan "Acara diarsipkan" (`GET` dan ekspor tetap berjalan); tetap tangani `409 project_read_only` bila status berubah di tab lain. Proyek `completed` tetap bisa diedit.

Kerangka frontmatter `vendor/index.astro` *(sketsa, belum dijalankan)*:

```ts
export const prerender = false;
const id = Astro.params.id ?? '';
if (!isUuid(id)) return Astro.rewrite('/404');
const q = Astro.url.searchParams;
const tab = q.get('tab') === 'belanja' ? 'belanja' : 'vendor';
const accessToken = Astro.locals.accessToken;                     // diisi middleware (token sudah segar)
const base = `/v1/planner/projects/${id}`;
const [detail, registry, sessions] = await Promise.all([
  backendFetch<Envelope<ProjectDetail>>(base, { accessToken }),
  loadRegistry(Astro),
  backendFetch<Envelope<SessionWithUsage[]>>(`${base}/sessions`, { accessToken }),
]);
// … toPageError(detail) menerapkan tabel 2.2; selanjutnya per tab, mis. vendor:
const overview = await backendFetch<Envelope<VendorOverview>>(`${base}/vendors`, { accessToken });
const known = new Set(sessions.data.data.map((s) => s.scope_key));
const rows = tableRows(overview.data.data.vendors, overview.data.data.limit_groups);
const counts = countVendors(overview.data.data.vendors, known);
const missing = missingCategories(overview.data.data.categories);
```

### 2.3 State di URL dan di peramban

| Penyimpan | Kunci | Nilai | Ditulis oleh | Dibaca oleh |
|---|---|---|---|---|
| URL | `tab` | `vendor` (default) · `belanja` | tautan tab (navigasi penuh) | SSR: bagian yang dimuat |
| URL | `scope` | `scope_key` sesi atau `all` (Umum); tak ada = Semua | klien, `history.replaceState` | SSR menandai chip aktif dan memberi `hidden` pada baris lain; nilai yang tidak dikenal diperlakukan sebagai Semua |
| URL | `status` | status vendor (Tab 1) atau status belanja (Tab 2); tak ada = Semua | idem | idem |
| URL | `category` | `category_key` (Tab 1) atau teks kategori (Tab 2); tak ada = Semua | idem (klik chip kategori) | idem |

Tidak ada `localStorage` atau `sessionStorage`; tidak ada data proyek di penyimpan peramban.

### 2.4 Satu sumber render dan pembaruan sesudah aksi

Mengikuti pola partial [03 FE bab 2.4](../03_KEUANGAN/FE.md#24-satu-sumber-render-dan-pembaruan-sesudah-aksi), [02 FE bab 2.4](../02_TIMELINE_DAN_TUGAS/FE.md#24-satu-sumber-render-dan-pembaruan-sesudah-aksi), dan [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial): **tidak ada logika render ganda di klien**.

1. SSR merender area `[data-vendor-live]` dari komponen yang sama dengan `live.astro`. Memilih chip (sesi, status, kategori) hanyalah mengubah atribut `hidden` di klien, tanpa memanggil server; baris dan kartu membawa atribut data ([tabel 7.5.1](#751-atribut-data-yang-dibaca-skrip-klien) dan [8.5.1](#851-atribut-data-yang-dibaca-skrip-klien)).
2. Aksi tulis berhasil → klien memanggil `GET /dashboard/planner/{id}/vendor/live?tab=…&…` (`Accept: text/html`), mengganti isi `[data-vendor-live]`, lalu menerapkan ulang chip aktif dari URL. Fokus dipertahankan lewat `data-focus-key`.
3. Tidak ada pembaruan optimistik di menu ini: pagu, selisih, status bayar, dan angka gulungan dihitung server (kelompok pagu bergantung pada vendor lain), jadi setiap aksi menunggu respons lalu menyegarkan area. Tombol aksi dinonaktifkan selama permintaan berjalan (`aria-busy`), sehingga klik ganda tidak mengirim dua permintaan.
4. Partial gagal (`5xx`, jaringan): jangan kosongkan area; tampilkan banner kecil "Data belum diperbarui" + tombol "Muat ulang" (`location.reload()`), `aria-live="polite"`. `401` atau pengalihan ke login: `location.assign('/auth/masuk')`.
5. Tidak ada polling di menu ini. Galat yang menandakan data usang (`not_found`, `invalid_scope`, `409 conflict`) selalu ditutup dengan memuat ulang area lewat partial.

*(Belum diuji di browser, sama seperti 01–03. Alternatif yang sah: `location.reload()` sesudah tiap aksi tulis.)*

---

## 3. Prasyarat yang harus sudah ada dari Menu 01 dan 03 FE

**Belum ada satu pun di repo** (frontend Planner belum dibangun). Menu ini mengimpor, tidak membangun ulang:

| Kebutuhan | Berasal dari |
|---|---|
| `types.ts` (`Notice`, `SessionBrief`, `SessionWithUsage`, `EventTypeInfo`, `Project`, `ProjectDetail`, `UpdateProjectBody`, `UpdateProjectResult`, `Envelope`, `ApiError`, `PaymentStatus`) | 01 FE bab 4.2 dan 03 FE bab 4.2 (`PaymentStatus`) |
| `format.ts` (`formatRupiah`, `formatRupiahShort`, `percent`, `formatPercent`, `barWidth`, `charCount`, `parseFieldError`) | 01 FE bab 5.1 |
| `finance.ts` (`paymentBadge`, `SCOPE_ANY`) dan `PAYMENT_BADGE_LABELS` | 03 FE bab 5 dan 10.4; dipakai `paymentText`/`vendorScope` (bab 5). Bila Menu 03 FE belum dibangun, bangun dua simbol itu lebih dulu (keduanya kecil dan sudah teruji di dokumen 03) |
| `financeApi.ts` (`listLines`, `updateLine`) | 03 FE bab 6.1; dipakai hanya oleh aksi "Samakan estimasi pos" (bab 7.5, alur F) |
| `messages.ts` (`plannerErrorMessage`, `fieldErrorMessage`, `financeErrorMessage`, `financeFieldMessage`) | 01 FE bab 5.4 dan 11, 03 FE bab 10.4 |
| `client.ts` (`plannerFetch`: tangani sesi habis, `204`, backend mati; metode `PUT`/`PATCH`) dan `server.ts` (`loadRegistry`, `isUuid`) | 01 FE bab 5.5–5.6 |
| BFF `[...path].ts` dan perubahan `backend.ts` (`PUT`, `details`, `backendFetchRaw`) | 01 FE bab 6 dan 1.3 |
| `PlannerLayout.astro` (Project Switcher, sub-navigasi `MENUS`) dan `404.astro` | 01 FE bab 10 dan 1.1 |
| Pola `<dialog>`, `textContent` untuk string API | 01 FE bab 12 |

Karena itu urutan kerja ([bab 12](#12-urutan-pengerjaan-dan-definisi-selesai)) dimulai dari M0, M1, dan `PlannerLayout` di Menu 01 FE.

---

## 4. Kontrak data: tipe, fixture, contoh respons, katalog galat

### 4.1 Aturan JSON yang dikirim backend

Selain aturan umum [01 FE bab 4.1](../01_DASHBOARD/FE.md#41-aturan-json-yang-dikirim-backend) (amplop `{ "data": … }`, tanggal `"YYYY-MM-DD"` tanpa zona, uang bilangan bulat). Berbeda dengan 02 FE, butir di bawah **berasal dari BE.md dan SQL yang dijalankan** dan sekarang juga diamati pada backend yang berjalan (tes HTTP di [catatan implementasi](../../../backend/planner/2026-10-23-planner-menu-04.md)); tidak ada satu pun yang berubah, kecuali pos cadangan gulungan (bab 8.2, baris `RollupBanner`).

**Bentuk respons**

- **Kunci dihilangkan bila kosong** (`omitempty`): `pic_name`, `phone_number`, `formatted_wa_number`, `portfolio_url`, `limit_group`, `price_delta`, `notes`, `payment`, `delta` dan `line` pada kelompok, `budget_item_id` (pada `SharedItem`, `LockDealResult`, `RollupTarget`, `ShoppingItem`), `shared_item`, `unit`, `vendor_id`, `vendor_name`, `purpose`, `template_key`, `purchase_location`, `souvenir_stock`. Tipe memakai `?:`; jangan memeriksa `=== null`. **Tidak ada `null` yang dikirim** di menu ini.
- **Daftar selalu array**, tidak pernah `null`, termasuk saat kosong (`vendors`, `categories`, `limit_groups`, `vendor_ids`, `items`, `rollups`, `warnings`).
- **Tidak ada persentase di respons.** Hitung dengan `percent` (rasio, `null` bila penyebut ≤ 0 → tampilkan "–"): progres kategori (`categoryProgress`), selesai/total (`completionRatio`). Jangan menjumlah hasil `formatPercent` per bagian sendiri.
- **Pagu efektif adalah hasil server.** `budget_limit` pada vendor = pagu manual atau pagu skenario (`limit_source`); pada `group` dan `none` ia `0` dan **bukan** "pagu nol". `price_delta` hilang bila pagu atau penawaran belum ada: tampilkan "–", bukan "hemat penuh". Jangan menghitung ulang selisih di klien dari `budget_limit` untuk vendor `group`.
- **`delta` kelompok** positif dengan `priced < members` adalah **sisa pagu**, bukan hemat: pilih kata dengan `groupVerdict`.
- **`payment`/`line` adalah angka view** (`estimate`, `paid`, `remaining`, `payment_status`); klien tidak menghitung status. `remaining < 0` = lebih bayar. Vendor pada pos bersama tidak punya `payment`; statusnya ada di `limit_groups[].line`.
- **Vendor `batal`** tampil di tabel (redup) tetapi tidak ada di kelompok pagu dan tidak dihitung di KPI.
- **Total baris belanja** (`estimated_total`, `actual_total`) dihitung server dan sama dengan `lineTotal(quantity, harga)`; `quantity` adalah angka dua desimal (satu-satunya `float` di menu ini).
- **`message` galat dan `warnings[].message` berbahasa Inggris** dan hanya untuk log. Tampilkan teks dari `messages.ts` berdasarkan `code` (+ kolom/alasan untuk `invalid_*`).
- `sort_order` bukan nomor tampilan: kolom "No" belanja memakai `itemNumbers`.

**Permintaan**

- **Bilangan bulat murni.** Uang dikirim sebagai integer (hanya digit); jangan mengirim string atau desimal (`invalid_json` tanpa nama kolom). Ubah isian teks dengan `parseRupiah` (03 FE bab 5); `null` berarti isian tidak valid, tampilkan galat kolom dan jangan mengirim. Jumlah barang memakai `parseQuantity` (dua desimal; "1.000" ditolak).
- **Nomor ponsel dikirim apa adanya** (`phone_number`); server menormalisasi. Klien memakai `normalizeWA` hanya untuk memberi tahu pengguna lebih awal (tombol WA, validasi formulir); hasilnya tidak dikirim.
- **Kategori**: kategori registry dikirim sebagai `category_key`; kategori kustom (baru atau sudah dipakai) hanya sebagai `category_label` dan kuncinya diturunkan server, juga pada `PUT`. Gunakan `categoryPayload`.
- **Kunci yang tidak dikenal diabaikan tanpa galat**: andalkan tipe TypeScript.
- **Teks dipangkas** (`trim`) oleh server; panjang dihitung dalam *code point*: pakai `charCount` (01 FE), bukan `.length`.
- **`PUT` mengganti penuh** (vendor dan barang): kirim **semua** isian formulir. Khusus barang: `PUT` **tidak mengubah** tautan gulungan (`budget_item_id`), `template_key`, dan urutan. `budget_limit` yang tidak dikirim berarti 0 (ikut skenario).
- **`status: fix` mensyaratkan `agreed_price > 0`** (`fixNeedsPrice`); gunakan `lockDeal` untuk menyinkronkan anggaran. Meninggalkan `fix` pada vendor yang bertaut ke pos melepas tautan: minta konfirmasi (`releasesBudgetLine`).
- **Aman diulang:** semua `GET`, `PUT` (hasil sama), `lockDeal` dengan harga sama, `applyShoppingTemplate` mode `append`, `linkToBudget` pada pos yang sudah rollup. **Tidak** aman diulang: semua `POST` pembuat (vendor, barang), `replace_template`, `unlinkFromBudget` (kedua kali melepas 0). `DELETE` kedua kali menjawab `404`: perlakukan sebagai berhasil dan segarkan.

### 4.2 Tambahan pada `src/lib/planner/types.ts`

Hanya sintaks yang dapat dihapus (tanpa `enum`), seperti 01 FE. `Notice`, `SessionBrief`, `PaymentStatus`, `ApiError`, `UpdateProjectResult` berasal dari blok 01 dan 03. Perubahan pada blok 01 ada di [bab 1.3](#13-perubahan-pada-file-yang-sudah-ada) (`EventTypeInfo` dan `UpdateProjectBody`).

```ts
// ---------- Menu 04: vendor dan belanja (ditambahkan ke akhir types.ts) ----------
// Tambahan registry (GET /event-types): kategori vendor dan kategori belanja jenis acara. Ubah satu baris di blok 01:
// `export interface EventTypeInfo {` → `export interface EventTypeInfo extends EventTypeVendorInfo {`
// (bila Menu 02 dan 03 FE sudah dibangun: `extends EventTypeAdminInfo, EventTypeFinanceInfo, EventTypeVendorInfo`).
export interface VendorCategory { key: string; label: string }
export interface EventTypeVendorInfo { vendor_categories: VendorCategory[]; shopping_categories: string[] }

export type VendorStatus = 'rencana' | 'negosiasi' | 'fix' | 'batal';
export type LimitSource = 'manual' | 'scenario' | 'group' | 'none';
export type CategoryStateKey = 'fix' | 'pipeline' | 'empty';
export type ShoppingStatus = 'belum' | 'proses' | 'selesai';
export type ShoppingPurpose = 'souvenir';

// ----- Tab 1: vendor -----
/** Angka efektif satu pos anggaran (view planner_budget_lines): pos yang bertaut ke vendor, atau pos bersama sebuah kelompok pagu. */
export interface LinePayment {
  budget_item_id: string; item_name: string;
  estimate: number; paid: number; remaining: number; // remaining < 0 = lebih bayar
  payment_status: PaymentStatus;
}
export interface Vendor {
  id: string; sub_event_scope: string; category_key: string; category_label: string; vendor_name: string;
  pic_name?: string; phone_number?: string; formatted_wa_number?: string; portfolio_url?: string;
  budget_limit: number;                // pagu efektif; 0 bila limit_source 'group' atau 'none'
  limit_source: LimitSource;
  limit_group?: string;                // item_key kelompok bila limit_source 'group'
  agreed_price: number;                // 0 = belum ada penawaran
  price_delta?: number;                // budget_limit − agreed_price (positif = hemat); tidak ada bila salah satunya 0 atau limit_source group/none
  status: VendorStatus; notes?: string; sort_order: number;
  payment?: LinePayment;               // pos yang menunjuk vendor ini (lock-deal); tidak ada bila belum bertaut
}
export interface LimitGroup {
  item_key: string; sub_event_scope: string;
  limit: number; total_agreed: number; // total_agreed = Σ agreed_price anggota yang bukan batal (penawaran dan deal)
  delta?: number;                      // limit − total_agreed; tidak ada bila belum ada anggota yang punya penawaran
  members: number; priced: number; vendor_ids: string[];
  line?: LinePayment;                  // pos anggaran bersama kelompok, bila ada
}
export interface CategoryState { key: string; label: string; state: CategoryStateKey; vendors: number; custom: boolean; skipped: boolean }
export interface VendorKPI {
  categories_total: number; categories_fixed: number; skipped: number;
  fixed_vendors: number; pipeline: number; fixed_total: number; pipeline_offers: number;
  groups_total: number; groups_over: number;
}
export interface VendorOverview {
  kpi: VendorKPI; categories: CategoryState[]; limit_groups: LimitGroup[]; // selalu seluruh proyek
  vendors: Vendor[];                                                        // disaring ?scope=&status=&category=
}
export interface VendorBody {          // POST dan PUT (PUT mengganti penuh: kirim semua isian formulir)
  sub_event_scope: string; category_key?: string; category_label?: string; vendor_name: string;
  pic_name?: string; phone_number?: string; portfolio_url?: string;
  budget_limit: number; agreed_price: number; status: VendorStatus; notes?: string;
}
export interface VendorStatusBody { status: VendorStatus }
export interface VendorResult { vendor: Vendor; warnings: Notice[] }
export interface LockDealBody { agreed_price: number; sync_to_budget_items: boolean }
export interface SharedItem { item_key: string; item_name: string; budget_item_id?: string; plan: number; total_agreed: number }
export interface LockDealResult { vendor: Vendor; budget_item_id?: string; created: boolean; shared_item?: SharedItem; warnings: Notice[] }

// ----- Tab 2: belanja -----
export interface ShoppingItem {
  id: string; sub_event_scope: string; category: string; item_name: string;
  quantity: number;                    // dua desimal
  unit?: string;
  estimated_price: number; actual_price: number;  // PER SATUAN; actual_price 0 = belum dibeli
  estimated_total: number; actual_total: number;  // hasil server: round(quantity × harga)
  price_delta?: number;                // estimated_total − actual_total (positif = hemat); tidak ada bila actual_price 0
  purchase_location?: string;          // teks bebas; URL http/https dirender sebagai tautan (safeHref)
  vendor_id?: string; vendor_name?: string; budget_item_id?: string; purpose?: ShoppingPurpose;
  status: ShoppingStatus; sort_order: number; template_key?: string; notes?: string;
}
export interface ShoppingKPI {
  estimated_total: number; actual_total: number;   // actual_total hanya barang selesai
  done: number; in_progress: number; not_started: number; total: number;
  souvenir_stock?: number;                         // tidak ada bila tak ada barang souvenir
}
export interface RollupLine { budget_item_id: string; item_name: string; sub_event_scope: string; estimate: number; paid: number; linked_items: number }
export interface RollupTarget {
  budget_item_id?: string;             // tidak ada: pos belum ada dan akan dibuat saat "Hitung pos dari daftar ini"
  item_key: string; item_name: string; sub_event_scope: string;
  rollup: boolean; has_payments: boolean; estimate: number; paid: number;
}
export interface ShoppingOverview {
  kpi: ShoppingKPI; rollups: RollupLine[]; default_target: RollupTarget; // selalu seluruh proyek
  items: ShoppingItem[];                                                  // disaring ?scope=&category=&status=
}
export interface ShoppingItemBody {    // POST
  sub_event_scope: string; category: string; item_name: string; quantity?: number; unit?: string;
  estimated_price: number; actual_price: number; purchase_location?: string; vendor_id?: string;
  budget_item_id?: string;             // hanya POST: pos rollup tujuan (kosong = tidak digulung)
  purpose?: ShoppingPurpose; status: ShoppingStatus; notes?: string;
}
export type ShoppingItemUpdate = Omit<ShoppingItemBody, 'budget_item_id'>; // PUT mengganti penuh, tetapi tautan rollup tidak berubah
export interface ShoppingStatusBody { status: ShoppingStatus }
export interface ShoppingTemplateBody { mode: 'append' | 'replace_template'; confirm?: boolean }
export interface ShoppingTemplateResult { created: number; skipped: number; deleted: number; warnings: Notice[] }
export interface LinkToBudgetBody { budget_item_id?: string; item_ids?: string[]; confirm?: boolean }
export interface LinkToBudgetResult { line: RollupLine; created: boolean; linked: number }
export interface UnlinkBody { budget_item_id: string }
export interface UnlinkResult { unlinked: number }
```

**Hasil pencocokan dengan DTO Go** (blok [BE bab 4](BE.md#4-dto-go) dikompilasi, lalu skema JSON-nya diambil dengan `reflect` dan dibandingkan dengan interface di atas oleh pembanding berbasis compiler API TypeScript; **bukan** terhadap respons HTTP):

| Kelompok | Pasangan tipe (Go ↔ TS) | Jalur JSON |
|---|:-:|:-:|
| Respons Tab 1 (`Vendor`, `LinePayment`, `LimitGroup`, `CategoryState`, `VendorKPI`, `VendorOverview`, `VendorResult`, `SharedItem`, `LockDealResult`) | 9 | 207 |
| Respons Tab 2 (`ShoppingItem`, `ShoppingKPI`, `RollupLine`, `RollupTarget`, `ShoppingOverview`, `ShoppingTemplateResult`, `LinkToBudgetResult`, `UnlinkResult`) | 8 | 115 |
| Permintaan (9 tipe: vendor, status vendor, lock-deal, barang, ubah barang, status barang, template, link, unlink) | 9 | 56 |
| **Jumlah** | **26** | **378** |

Hasil: **0 selisih** (kunci hanya di satu sisi, opsionalitas `omitempty` ↔ `?:`, `null` ↔ pointer tanpa `omitempty`, jenis nilai); untuk tipe permintaan hanya kunci dan jenis nilai yang dibandingkan (opsionalitas permintaan dijaga usecase). Pembanding diuji dengan enam mutasi tipe TS (opsional dijadikan wajib pada dua kunci, kunci diganti nama, jenis diganti, `| null` ditambahkan, kunci berlebih) dan melaporkan semuanya. Cara mengulang: [bab 11.5](#115-verifikasi-kontrak-cara-mengulang).

### 4.3 Fixture

Empat berkas di [`fixtures/`](fixtures/) (folder ini) disalin ke `frontend/tests/fixtures/`, ditambah `planner-sessions.json` dari [Menu 02](../02_TIMELINE_DAN_TUGAS/fixtures/planner-sessions.json) (sesi sistem `all` + Lamaran + Akad + Resepsi). **Bukan respons endpoint** (endpoint belum ada). `planner-vendors.json` dan `planner-shopping.json` dibangun dari Proyek Contoh yang dibuat backend Menu 01–03 asli di PostgreSQL 16.2 sementara: tabel [BE bab 5.2–5.3](BE.md#5-master-data) dimasukkan seperti overlay (10 vendor; `vendor_id` pada pos `catering` dan `dokumentasi`; 37 barang berstatus PDF), pagu dibaca dari skenario aktif 75 Jt, lalu dilewatkan ke fungsi acuan Go ([BE bab 2.1](BE.md#21-pagu-vendor-dari-skenario-aktif)) dan DTO bab 4. Id diganti id sintetis tetap supaya tes deterministik (vendor `…-000400000001…10`, barang `…-000400010001…37`, pos anggaran mengikuti fixture 03: `…-000300000008` Catering resepsi, `…10` Sewa Gedung & Dekorasi, `…13` Seserahan), dan kunci kosong dibuang oleh `omitempty` seperti respons asli. `planner-sessions.json` (Menu 02) menulis `usage.vendors` dan `usage.shopping_items` 0; pada Proyek Contoh lengkap sesi Resepsi memuat `usage.vendors` 10 dan sesi Umum `usage.shopping_items` 37.

| Berkas | Isi | Bentuk |
|---|---|---|
| `planner-vendors.json` | 10 vendor (7 fix, 2 negosiasi, 1 rencana); KPI `7 / 10 / 7 / 3 / 58.000.000 / 4.300.000 / 3 / 0`; Catering `scenario` 30.000.000, deal 25.000.000, selisih +5.000.000, `payment` terbayar 15.000.000 `partial`; Photography `scenario` 3.000.000 pas, `dp_paid`; tiga kelompok pagu (`venue_dekor`, `rias_busana`, `hiburan`) berselisih 0 dengan `line` pos bersama (`dp_paid` 900.000, `dp_paid` 600.000, `settled` 3.400.000) | `VendorOverview` |
| `planner-shopping.json` | 37 barang (sesi `all`, `template_key = wedding.seserahan_…`, 16 selesai berharga riil = estimasi, 21 proses); KPI `4.578.500 / 2.063.800 / 16 / 21 / 0 / 37`; `rollups: []`; `default_target` Seserahan (`has_payments: true`, estimasi dan terbayar 3.500.000 menurut PDF Hal. 5–6) | `ShoppingOverview` |
| `planner-wa-vectors.json` | 26 pasangan `{input, wa}` (`wa: null` = tidak valid, `""` = kosong); **sama persis** dengan tes backend `NormalizeWA` | `{input: string, wa: string \| null}[]` |
| `planner-line-totals.json` | 300 pasangan `{q100, price, total}` dari `round(q100/100 × price)` PostgreSQL (`total: null` bila > 10¹³; termasuk tepat 10¹³ dan 10¹³+1); **sama persis** dengan tes backend `LineTotal` | `{q100: number, price: number, total: number \| null}[]` |

Contoh baris (dari `planner-vendors.json`, Catering):

```json
{ "id": "00000000-0000-4000-8000-000400000005", "sub_event_scope": "resepsi", "category_key": "catering", "category_label": "Catering",
  "vendor_name": "Katering Bu Minah (500 pax)", "phone_number": "081200000005", "formatted_wa_number": "6281200000005",
  "budget_limit": 30000000, "limit_source": "scenario", "agreed_price": 25000000, "price_delta": 5000000, "status": "fix", "sort_order": 5,
  "payment": { "budget_item_id": "00000000-0000-4000-8000-000300000008", "item_name": "Catering", "estimate": 25000000, "paid": 15000000,
               "remaining": 10000000, "payment_status": "partial" } }
```

**Variasi yang wajib diuji** (turunkan dari fixture di dalam tes, jangan membuat berkas baru): proyek kosong (`vendors: []`, semua kategori `empty`); vendor `batal` (bukan anggota kelompok, redup); kelompok dengan anggota tanpa penawaran (`priced < members`, "sisa pagu"); kelompok melebihi pagu (`delta < 0`); vendor pagu manual dalam kelompok (keluar dari kelompok); `limit_source: none`; vendor tanpa nomor (tombol WA nonaktif); pos bertaut dengan estimasi berbeda dari deal ("Pos ≠ deal"); lebih bayar; semua kategori dilewati (`categories_total: 0`, progres "–"); kategori kustom; sesi usang (tampil sebagai Umum); nama vendor 255 karakter; belanja: barang berjumlah 300 satuan (total ratusan ribu), `purpose: souvenir` (stok), gulungan aktif (`default_target.rollup: true`, `rollups` terisi), pos belum ada (`default_target` tanpa `budget_item_id`), `price_delta` negatif (lebih mahal), jenis seminar (kategori belanja 4, tanpa template), proyek `archived`.

### 4.4 Katalog endpoint

Semua di bawah `projects/:id/` (`:id` = UUID proyek) lewat BFF; semua penulisan juga mungkin menjawab `401`, `404 not_found` (proyek), `409 project_read_only`, dan `5xx`/jaringan, yang tidak diulang di kolom Galat. Nama fungsi = `vendorApi` (bab 6). Deskripsi lengkap: [BE bab 3](BE.md#3-endpoint). `vid` = id vendor, `iid` = id barang.

| Aksi UI | Fungsi | Metode dan path | Body | Sukses | Galat khusus |
|---|---|---|---|---|---|
| Muat vendor | `listVendors` | `GET vendors[?scope=&status=&category=]` | – | `200 VendorOverview` | `400 invalid_input: status`/`scope`/`category` (hanya bila URL diketik tangan; halaman menyaring di klien) |
| (satu vendor) | `getVendor` | `GET vendors/:vid` | – | `200 Vendor` | `404` (vendor) |
| Tambah vendor | `createVendor` | `POST vendors` | `VendorBody` | `201 VendorResult` | `409 vendor_limit_reached`, `400 invalid_input`/`invalid_phone`/`invalid_scope` |
| Ubah vendor | `updateVendor` | `PUT vendors/:vid` | `VendorBody` | `200 VendorResult` | seperti POST; `404` |
| Ubah status | `setVendorStatus` | `PATCH vendors/:vid/status` | `{ status }` | `200 VendorResult` | `400 invalid_input: agreed_price` (fix tanpa deal); `404` |
| Kunci FIX | `lockDeal` | `POST vendors/:vid/lock-deal` | `LockDealBody` | `200 LockDealResult` | `400 invalid_input: agreed_price`; `404` |
| Hapus vendor | `deleteVendor` | `DELETE vendors/:vid` | – | `204` | `404` (dianggap sukses) |
| Lewati kategori | `setSkippedCategories` | `PATCH projects/:id` (Menu 01) | `{ skipped_vendor_categories }` | `200 UpdateProjectResult` | `400 invalid_input: skipped_vendor_categories` |
| Ekspor kontak | `exportContactsHref` | `GET vendors/export-contacts?format=xlsx` | – | `200` berkas `.xlsx` | `429 rate_limited` (10/menit) |
| Muat belanja | `listShopping` | `GET shopping-items[?scope=&category=&status=]` | – | `200 ShoppingOverview` | `400 invalid_input: status`/`scope` |
| Tambah barang | `createShoppingItem` | `POST shopping-items` | `ShoppingItemBody` | `201 ShoppingItem` | `409 shopping_item_limit_reached`, `invalid_input` (termasuk `estimated_price`/`actual_price`: total terlalu besar), `invalid_scope` |
| Ubah barang | `updateShoppingItem` | `PUT shopping-items/:iid` | `ShoppingItemUpdate` | `200 ShoppingItem` | seperti POST; `404` |
| Ubah status | `setShoppingStatus` | `PATCH shopping-items/:iid/status` | `{ status }` | `200 ShoppingItem` | `404` |
| Hapus barang | `deleteShoppingItem` | `DELETE shopping-items/:iid` | – | `204` | `404` (dianggap sukses) |
| Terapkan template | `applyShoppingTemplate` | `POST shopping-items/apply-template` | `ShoppingTemplateBody` | `200 ShoppingTemplateResult` | `400 invalid_input: confirm`/`mode`, `invalid_template`, `409 shopping_item_limit_reached` |
| Hitung pos dari daftar | `linkToBudget` | `POST shopping-items/link-to-budget` | `LinkToBudgetBody` | `200 LinkToBudgetResult` | `409 rollup_conflict` (minta konfirmasi), `400 invalid_input: budget_item_id`/`item_ids` (pos atau barang bukan milik proyek); id yang bukan UUID → `400 invalid_json` ([BE bab 9 butir 24](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)) |
| Lepas gulungan | `unlinkFromBudget` | `POST shopping-items/unlink-from-budget` | `{ budget_item_id }` | `200 UnlinkResult` | `400 invalid_input: budget_item_id` (pos bukan milik proyek, juga badan kosong); id bukan UUID → `400 invalid_json` |
| Ekspor belanja | `exportShoppingHref` | `GET shopping-items/export-shopping-list?format=xlsx` | – | `200` berkas `.xlsx` | `429 rate_limited` (10/menit) |

### 4.5 Contoh permintaan dan respons

Bentuk dari BE bab 3 dan 2 serta fixture. Kunci dan tipenya sama dengan respons backend yang berjalan: `GET /vendors` dan `GET /shopping-items` dibandingkan langsung dengan fixture (nol selisih); cuplikan respons tulis di bawah (`LockDealResult`, `VendorResult`, `LinkToBudgetResult`, hasil template) dicocokkan dengan **respons asli** backend pada Proyek Contoh (2026-10-23): kunci, tipe, nilai, dan kode galat sama; yang berbeda hanya id (dinormalkan) dan kunci `vendor` yang dipotong.

**Kunci FIX, kategori berbagi pos** (`POST vendors/:vid/lock-deal` untuk Venue, `sync_to_budget_items: true`): pos `venue_dekor` tidak ditautkan dan estimasinya tidak berubah.

```json
// permintaan
{ "agreed_price": 12000000, "sync_to_budget_items": true }
// respons (LockDealResult; vendor dipotong)
{ "data": { "vendor": { "id": "00000000-0000-4000-8000-000400000001", "category_key": "venue", "limit_source": "group", "limit_group": "venue_dekor",
    "budget_limit": 0, "agreed_price": 12000000, "status": "fix", "…": "…" },
  "created": false,
  "shared_item": { "item_key": "venue_dekor", "item_name": "Sewa Gedung & Dekorasi", "budget_item_id": "00000000-0000-4000-8000-000300000010",
    "plan": 20000000, "total_agreed": 20000000 },
  "warnings": [ { "code": "shared_budget_item", "message": "the budget line is shared by several vendor categories and was not changed", "count": 1 } ] } }
```

**Kunci FIX, pos baru** (Harmony Wedding Organizer, `wo` tanpa pos): `created: true`, `vendor.payment` memuat pos yang baru dibuat (`estimate` = deal, `paid` 0, `payment_status: "unpaid"`), `warnings: []`.

**Meninggalkan FIX** (`PATCH vendors/:vid/status` `{ "status": "negosiasi" }` pada katering yang bertaut): `{ "data": { "vendor": { …"status": "negosiasi"… }, "warnings": [ { "code": "budget_link_released", "message": "…", "count": 1 } ] } }`; `vendor.payment` hilang.

**Hitung pos dari daftar** (`POST shopping-items/link-to-budget`). Tanpa `confirm` pada pos berpembayaran: `409 { "error": { "code": "rollup_conflict", … } }` → tampilkan `ROLLUP_CONFIRM_TEXT`, lalu kirim ulang dengan `confirm: true`:

```json
// permintaan (kedua)
{ "confirm": true }
// respons (LinkToBudgetResult; angka dari BE §2.4 yang dijalankan pada Proyek Contoh)
{ "data": { "line": { "budget_item_id": "00000000-0000-4000-8000-000300000013", "item_name": "Seserahan", "sub_event_scope": "resepsi",
    "estimate": 4578500, "paid": 2063800, "linked_items": 37 }, "created": false, "linked": 37 } }
```

**Ganti template** (`POST shopping-items/apply-template` `{ "mode": "replace_template" }` pada Proyek Contoh): tanpa `confirm` → `400 invalid_input: confirm: required` (37 barang bekerja); dengan `confirm: true` → `{ "data": { "created": 37, "skipped": 0, "deleted": 37, "warnings": [] } }`.

**Unduhan** (`GET vendors/export-contacts?format=xlsx`): `200`, `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="vendor-<slug>-<tanggal>.xlsx"`. Header teramati: `Cache-Control: no-store`; contoh nama berkas `vendor-contoh-pernikahan-sinta-arif-2026-10-04.xlsx`. Permintaan ke-11 dalam semenit → `429 {"error":{"code":"rate_limited","message":"too many requests, try again later"}}` (teramati). Isi berkas `.xlsx` belum dibuka di Excel/Numbers.

### 4.6 Katalog galat

Bentuk: `{ "error": { "code": "…", "message": "…" } }` (+ `details` pada beberapa kode Menu 01). Untuk `invalid_input`/`invalid_date`/`invalid_phone`, `message` = `<kolom>: <alasan>` dan dipetakan ke teks per kolom (bab 9.2); kode lain dipetakan per kode (bab 9.1). Daftar alasan baku: [BE §3.4](BE.md#34-validasi-dan-batas); tes 11.1 memastikan **setiap** alasan di sana punya teks Indonesia atau diserahkan eksplisit ke fungsi Menu 03/01.

| Status | Kode | Contoh `message` | Penyebab dan penanganan UI |
|:-:|---|---|---|
| 400 | `invalid_input` | `vendor_name: must not be empty` · `agreed_price: must be greater than 0 when status is fix` · `estimated_price: the total (quantity × price) must be at most 10000000000000` · `confirm: required` | pesan di bawah kolom (bab 9.2); `confirm: required` bukan galat tampil, melainkan perintah menampilkan kotak konfirmasi lalu mengirim ulang dengan `confirm` (bab 8.5, alur R) |
| 400 | `invalid_phone` | `phone_number: must be an Indonesian phone number (08…, +62…)` | di bawah kolom nomor ponsel; klien sudah memberi tahu lebih awal lewat `normalizeWA` |
| 400 | `invalid_scope` | `sub_event_scope: unknown session` | sesi dihapus di tab lain → muat ulang area |
| 400 | `invalid_template` | `shopping_list: not available for this event type` | registry usang atau jenis tanpa template → muat ulang halaman |
| 400 | `invalid_json` | `invalid JSON body` | kesalahan klien (tipe salah, uang bukan integer, body kosong tanpa `{}`) |
| 401 | `unauthorized` | `missing bearer token` | `plannerFetch` mengalihkan ke login |
| 404 | `not_found` | `project not found` · `vendor not found` · `shopping item not found` | dibedakan oleh **aksi** (`where`), bukan oleh pesan; setelah `DELETE` kedua dianggap sukses |
| 409 | `project_read_only` | `the project is archived; restore it to make changes` | nonaktifkan semua kontrol ubah, teks 01 |
| 409 | `vendor_limit_reached` / `shopping_item_limit_reached` | `at most 200 vendors per project` | banner di modal; tidak ada baris yang tertulis |
| 409 | `rollup_conflict` | `the line has stored payments that are ignored while it computes from the shopping list; send confirm: true to continue` | bukan galat tampil: kotak konfirmasi (`ROLLUP_CONFIRM_TEXT`), kirim ulang dengan `confirm: true` |
| 409 | `conflict` | `the project was changed concurrently, try again` | jarang; teks 01, aman dicoba ulang |
| 429 | `rate_limited` | `too many requests, try again later` | hanya ekspor di menu ini |
| 5xx / jaringan | `internal`, `service_unavailable` | – | teks umum 01; jangan mengosongkan layar |

---

## 5. Library `lib/planner/vendor.ts`

Aturan 01 FE bab 5: modul **murni**, tanpa alias `@/`, tanpa `enum`, supaya tes `node --experimental-strip-types` dapat mengimpornya. Ia mengimpor `format.ts` (Menu 01) dan `finance.ts` (Menu 03; `paymentBadge`, `SCOPE_ANY`). **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-vendortestmjs)).

```ts
import { formatRupiahShort, percent } from './format.ts';
import { paymentBadge, SCOPE_ANY } from './finance.ts';
import type { CategoryState, LimitGroup, LinePayment, RollupTarget, ShoppingItem, ShoppingStatus, Vendor, VendorKPI, VendorStatus } from './types.ts';

/** Batas yang ditegakkan backend (BE §4.5); klien memakainya untuk `maxlength` dan tombol, bukan sebagai pengganti server. */
export const LIMITS = {
  vendorName: 255, picName: 255, phone: 40, portfolioUrl: 500, notes: 2000, categoryLabel: 60,
  itemName: 255, category: 60, unit: 30, purchaseLocation: 500,
  money: 10_000_000_000_000, quantityMin: 0.01, quantityMax: 1_000_000, quantityMaxX100: 100_000_000,
  vendors: 200, shoppingItems: 1000, skippedCategories: 50,
} as const;

// ---------- nomor WhatsApp ----------

/**
 * Cermin `planner/domain.NormalizeWA` (BE §2.3); vektor bersama `fixtures/planner-wa-vectors.json` diuji di kedua sisi.
 * '' = nomor kosong (diizinkan); null = tidak valid (`invalid_phone`). Tidak menebak nomor tanpa awalan ("8123…" → null).
 */
export function normalizeWA(input: string): string | null {
  let s = '';
  for (const [i, ch] of Array.from(input.trim()).entries()) {
    if (ch >= '0' && ch <= '9') s += ch;
    else if (ch === '+' && i === 0) s += ch;
    else if (!' -.()'.includes(ch)) return null;
  }
  if (s === '') return '';
  if (s.startsWith('+')) s = s.slice(1);
  else if (s.startsWith('00')) s = s.slice(2);
  else if (s.startsWith('0')) s = `62${s.slice(1)}`;
  if (!s.startsWith('62')) return null;
  s = `62${s.slice(2).replace(/^0/, '')}`;
  return s.length >= 10 && s.length <= 15 ? s : null;
}

/** Templat pesan 04 BE §2.3. Sesi Umum ('all') tidak disebut. `to` = vendor.formatted_wa_number (sudah ternormalisasi). */
export function waMessage(v: Pick<Vendor, 'pic_name' | 'vendor_name'>, projectTitle: string, sessionName?: string): string {
  const to = v.pic_name ?? v.vendor_name;
  return `Halo ${to}, saya ingin menanyakan perihal kerja sama untuk acara ${projectTitle}${sessionName ? ` pada sesi ${sessionName}` : ''}.`;
}
export const waLink = (wa: string, text?: string): string => `https://wa.me/${wa}${text ? `?text=${encodeURIComponent(text)}` : ''}`;

// ---------- tautan ----------

/** Alamat yang boleh dijadikan tautan: hanya http/https yang sah (javascript:, data:, //host tidak pernah). null = tampilkan sebagai teks. */
export function safeHref(input: string): string | null {
  const s = input.trim();
  if (!/^https?:\/\//i.test(s)) return null;
  try {
    const u = new URL(s);
    return u.protocol === 'http:' || u.protocol === 'https:' ? u.href : null;
  } catch {
    return null;
  }
}
/** "https://www.shopee.co.id/produk/123" → "shopee.co.id/produk/…" (teks tautan; `href` tetap penuh). */
export function linkLabel(href: string, max = 28): string {
  const u = new URL(href);
  const host = u.hostname.replace(/^www\./, '');
  const path = u.pathname === '/' ? '' : u.pathname;
  const full = host + path;
  return full.length <= max ? full : `${full.slice(0, max - 1)}…`;
}

// ---------- jumlah dan harga ----------

/** Isian jumlah → angka dengan paling banyak dua desimal ("2", "2,5", "2.5", "0,25"). Menolak pemisah ribuan ("1.000"): null. */
export function parseQuantity(input: string): number | null {
  const m = /^(\d{1,7})(?:[.,](\d{1,2}))?$/.exec(input.trim());
  if (!m) return null;
  const q100 = Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0'));
  return q100 >= 1 && q100 <= LIMITS.quantityMaxX100 ? q100 / 100 : null;
}
/**
 * round(q100/100 × harga) tanpa float (BigInt), setengah ke atas seperti kolom generated PostgreSQL; null bila hasil > 10¹³
 * (server menolak dengan `400 invalid_input`). Vektor `fixtures/planner-line-totals.json` dihasilkan PostgreSQL 16.2.
 */
export function lineTotalX100(q100: number, price: number): number | null {
  if (!Number.isSafeInteger(q100) || !Number.isSafeInteger(price) || q100 < 0 || price < 0) return null;
  const t = (BigInt(q100) * BigInt(price) + 50n) / 100n;
  return t <= BigInt(LIMITS.money) ? Number(t) : null;
}
export function lineTotal(quantity: number, price: number): number | null {
  const q100 = Math.round(quantity * 100);
  return Math.abs(quantity * 100 - q100) > 1e-6 ? null : lineTotalX100(q100, price);
}

// ---------- Tab 1: vendor ----------

/** Sesi vendor untuk tampilan: sesi yang tidak ada di `known` (data usang) dianggap 'all' (Umum). */
export const vendorScope = (v: Pick<Vendor, 'sub_event_scope'>, known?: ReadonlySet<string>): string =>
  known === undefined || v.sub_event_scope === 'all' || known.has(v.sub_event_scope) ? v.sub_event_scope : 'all';

export interface VendorFilter { scope?: string; status?: VendorStatus | typeof SCOPE_ANY; category?: string }
/** Chip sesi, status, dan kategori. Tidak memanggil server (atribut `hidden`); SCOPE_ANY / kosong = semua. */
export function filterVendors(vendors: readonly Vendor[], f: VendorFilter, known?: ReadonlySet<string>): Vendor[] {
  return vendors.filter(
    (v) =>
      (f.scope === undefined || f.scope === SCOPE_ANY || vendorScope(v, known) === f.scope) &&
      (f.status === undefined || f.status === SCOPE_ANY || v.status === f.status) &&
      (f.category === undefined || v.category_key === f.category),
  );
}
export interface VendorCounts { all: number; byStatus: Record<VendorStatus, number>; byScope: Record<string, number> }
export function countVendors(vendors: readonly Vendor[], known?: ReadonlySet<string>): VendorCounts {
  const c: VendorCounts = { all: vendors.length, byStatus: { rencana: 0, negosiasi: 0, fix: 0, batal: 0 }, byScope: {} };
  for (const v of vendors) {
    c.byStatus[v.status] += 1;
    const k = vendorScope(v, known);
    c.byScope[k] = (c.byScope[k] ?? 0) + 1;
  }
  return c;
}

export type VendorRow = { kind: 'single'; vendor: Vendor } | { kind: 'group'; group: LimitGroup; vendors: Vendor[] };
/**
 * Baris tabel: vendor yang berbagi satu pos pagu (`limit_groups[].vendor_ids`) dikelompokkan berdampingan (braket) di posisi
 * anggota pertama; urutan lain mengikuti `sort_order` dari server. Vendor batal tidak pernah anggota kelompok.
 */
export function tableRows(vendors: readonly Vendor[], groups: readonly LimitGroup[]): VendorRow[] {
  const groupOf = new Map<string, LimitGroup>();
  for (const g of groups) for (const id of g.vendor_ids) groupOf.set(id, g);
  const placed = new Set<string>();
  const rows: VendorRow[] = [];
  for (const v of vendors) {
    if (placed.has(v.id)) continue;
    const g = groupOf.get(v.id);
    if (g === undefined) {
      rows.push({ kind: 'single', vendor: v });
      continue;
    }
    const members = vendors.filter((x) => g.vendor_ids.includes(x.id));
    for (const m of members) placed.add(m.id);
    rows.push({ kind: 'group', group: g, vendors: members });
  }
  return rows;
}

export type DeltaTone = 'none' | 'saving' | 'even' | 'over';
/** Selisih satu vendor (pagu − penawaran): positif hemat (hijau), negatif melebihi pagu (merah); tidak ada → '–'. */
export const deltaTone = (delta: number | undefined): DeltaTone => (delta === undefined ? 'none' : delta > 0 ? 'saving' : delta < 0 ? 'over' : 'even');
export type GroupVerdict = 'none' | 'remaining' | 'saving' | 'even' | 'over';
/**
 * Selisih kelompok: 'remaining' = masih ada anggota tanpa penawaran, jadi selisih positif adalah sisa pagu (bukan hemat);
 * 'over' = penawaran sudah melebihi pagu, apa pun anggota lain; 'none' = belum ada penawaran sama sekali.
 */
export function groupVerdict(g: Pick<LimitGroup, 'delta' | 'members' | 'priced'>): GroupVerdict {
  if (g.delta === undefined) return 'none';
  if (g.delta < 0) return 'over';
  if (g.priced < g.members) return 'remaining';
  return g.delta === 0 ? 'even' : 'saving';
}

/** Teks kolom "Status bayar" dari angka pos tertaut atau pos bersama; '–' bila belum ada pos. Memakai `paymentBadge` Menu 03. */
export function paymentText(p: Pick<LinePayment, 'estimate' | 'paid' | 'remaining' | 'payment_status'> | undefined): string {
  if (p === undefined) return '–';
  switch (paymentBadge(p)) {
    case 'unpaid': return 'Belum DP';
    case 'dp_paid': return `DP ${formatRupiahShort(p.paid)}`;
    case 'partial': return `${formatRupiahShort(p.paid)} dari ${formatRupiahShort(p.estimate)}`;
    case 'settled': return 'Lunas';
    case 'overpaid': return `Lebih bayar ${formatRupiahShort(-p.remaining)}`;
    case 'unbudgeted': return `Dibayar ${formatRupiahShort(p.paid)} (belum dianggarkan)`;
  }
}
/** Pos tertaut yang estimasinya berbeda dari deal (mis. skenario diterapkan ulang): lencana "Pos ≠ deal". */
export const estimateDiffers = (v: Pick<Vendor, 'status' | 'agreed_price' | 'payment'>): boolean =>
  v.status === 'fix' && v.payment !== undefined && v.agreed_price > 0 && v.payment.estimate !== v.agreed_price;

/** Hanya vendor `fix` berdeal yang boleh berstatus fix (BE §2.2): kolom `agreed_price` wajib > 0. */
export const fixNeedsPrice = (status: VendorStatus, agreedPrice: number): boolean => status === 'fix' && agreedPrice <= 0;
/** Meninggalkan `fix` melepas tautan pos (peringatan `budget_link_released`): minta konfirmasi bila vendor punya pos tertaut. */
export const releasesBudgetLine = (v: Pick<Vendor, 'status' | 'payment'>, next: VendorStatus): boolean =>
  v.status === 'fix' && next !== 'fix' && v.payment !== undefined;

// ---------- Tab 1: kategori ----------

/**
 * Isian kategori untuk `VendorBody`: kategori registry dikirim sebagai `category_key`; kategori kustom (sudah dipakai atau baru)
 * hanya sebagai `category_label` dan kuncinya diturunkan server (`CategoryKeyFromLabel`), juga pada PUT. Label kosong → null.
 */
export function categoryPayload(registry: readonly { key: string }[], choice: { key?: string; label: string }): { category_key: string } | { category_label: string } | null {
  if (choice.key !== undefined && registry.some((c) => c.key === choice.key)) return { category_key: choice.key };
  const label = choice.label.trim();
  return label === '' ? null : { category_label: label };
}

/** Kategori yang belum terkunci dan tidak dilewati ("Belum: Hand Bouquet, MC, Entertainment"). */
export const missingCategories = (cats: readonly CategoryState[]): CategoryState[] => cats.filter((c) => !c.skipped && c.state !== 'fix');
/** Rasio kategori terkunci; null bila tidak ada kategori yang dihitung (semua dilewati): tampilkan "–". */
export const categoryProgress = (k: Pick<VendorKPI, 'categories_fixed' | 'categories_total'>): number | null => percent(k.categories_fixed, k.categories_total);
/**
 * Isi `skipped_vendor_categories` sesudah "Tandai tidak diperlukan" / "Diperlukan lagi" (PATCH proyek Menu 01 mengganti seluruh
 * daftar). null bila melewati batas 50. Hanya kategori kosong yang boleh dilewati (antarmuka); server tidak memeriksanya.
 */
export function toggleSkipped(current: readonly string[], key: string, skip: boolean): string[] | null {
  const rest = current.filter((k) => k !== key);
  const next = skip ? [...rest, key] : rest;
  return next.length > LIMITS.skippedCategories ? null : next;
}

// ---------- Tab 2: belanja ----------

export const SHOPPING_CYCLE: readonly ShoppingStatus[] = ['belum', 'proses', 'selesai'];
/** Toggle inline: belum → proses → selesai → belum. */
export const nextShoppingStatus = (s: ShoppingStatus): ShoppingStatus => SHOPPING_CYCLE[(SHOPPING_CYCLE.indexOf(s) + 1) % SHOPPING_CYCLE.length] ?? 'belum';

export interface ShoppingFilter { scope?: string; category?: string; status?: ShoppingStatus | typeof SCOPE_ANY }
export function filterItems(items: readonly ShoppingItem[], f: ShoppingFilter, known?: ReadonlySet<string>): ShoppingItem[] {
  const scopeOf = (i: ShoppingItem) => (known === undefined || i.sub_event_scope === 'all' || known.has(i.sub_event_scope) ? i.sub_event_scope : 'all');
  return items.filter(
    (i) =>
      (f.scope === undefined || f.scope === SCOPE_ANY || scopeOf(i) === f.scope) &&
      (f.category === undefined || i.category === f.category) &&
      (f.status === undefined || f.status === SCOPE_ANY || i.status === f.status),
  );
}
export interface ItemSums { estimated: number; actual: number; done: number; inProgress: number; notStarted: number; total: number }
/** Jumlah baris yang sedang tampil (footer tabel); aturan sama dengan `ShoppingKPI`: realisasi hanya barang selesai. */
export function sumItems(items: readonly ShoppingItem[]): ItemSums {
  const s: ItemSums = { estimated: 0, actual: 0, done: 0, inProgress: 0, notStarted: 0, total: items.length };
  for (const i of items) {
    s.estimated += i.estimated_total;
    if (i.status === 'selesai') { s.done += 1; s.actual += i.actual_total; }
    else if (i.status === 'proses') s.inProgress += 1;
    else s.notStarted += 1;
  }
  return s;
}
/** Jumlah per kategori (chip), urutan kemunculan pertama menurut `sort_order`. */
export function countByCategory(items: readonly ShoppingItem[]): [string, number][] {
  const m = new Map<string, number>();
  for (const i of [...items].sort((a, b) => a.sort_order - b.sort_order)) m.set(i.category, (m.get(i.category) ?? 0) + 1);
  return [...m.entries()];
}
/** Opsi kategori modal: registry jenis acara dulu, lalu kategori kustom yang sudah dipakai. */
export function categoryOptions(registry: readonly string[], items: readonly ShoppingItem[]): string[] {
  const out = [...registry];
  for (const [c] of countByCategory(items)) if (!out.includes(c)) out.push(c);
  return out;
}
/** Kolom "No": urutan 1-based pada daftar LENGKAP menurut sort_order (sort_order bisa berlubang). */
export function itemNumbers(all: readonly ShoppingItem[]): Map<string, number> {
  return new Map([...all].sort((a, b) => a.sort_order - b.sort_order).map((i, k) => [i.id, k + 1]));
}
export const completionRatio = (k: { done: number; total: number }): number | null => percent(k.done, k.total);

export interface TemplateImpact { removed: number; atRisk: number; needsConfirm: boolean }
/**
 * Dampak "Ganti template" (`replace_template`): semua barang ber-`template_key` dihapus; yang membawa pekerjaan pengguna
 * (status bukan belum, harga riil, tautan pos/vendor, tanda souvenir) mewajibkan `confirm: true` (BE §2.6). Server tetap penentu:
 * bila ia menjawab `confirm: required` padahal hitungan lokal tidak meminta, tampilkan kotak konfirmasi lalu kirim ulang.
 */
export function shoppingTemplateImpact(items: readonly ShoppingItem[]): TemplateImpact {
  const tpl = items.filter((i) => i.template_key !== undefined);
  const atRisk = tpl.filter(
    (i) => i.status !== 'belum' || i.actual_price > 0 || i.budget_item_id !== undefined || i.vendor_id !== undefined || i.purpose !== undefined,
  ).length;
  return { removed: tpl.length, atRisk, needsConfirm: atRisk > 0 };
}

export type RollupState = 'off' | 'off_with_payments' | 'on';
/** Keadaan banner rollup: 'off_with_payments' = mengaktifkan akan meminta konfirmasi (pembayaran tersimpan diabaikan; `rollup_conflict`). */
export const rollupState = (t: Pick<RollupTarget, 'rollup' | 'has_payments'>): RollupState => (t.rollup ? 'on' : t.has_payments ? 'off_with_payments' : 'off');
/** Stok souvenir dikurangi yang sudah diserahkan (Menu 05); null bila tidak ada barang souvenir selesai yang dicatat. */
export const souvenirLeft = (stock: number | undefined, distributed: number): number | null => (stock === undefined ? null : stock - distributed);
```

| Fungsi | Dipakai oleh | Catatan |
|---|---|---|
| `normalizeWA`, `waMessage`, `waLink` | `VendorModal` (pratinjau dan galat dini), `VendorTable` (tombol WA) | cermin `NormalizeWA` backend; vektor bersama 26 kasus; `''` kosong, `null` tidak valid; tidak menebak nomor tanpa awalan; `waMessage` tanpa klausa sesi untuk Umum |
| `safeHref`, `linkLabel` | `ShoppingTable` (kolom Alamat), `VendorTable` (portofolio) | hanya `http`/`https` yang sah; `javascript:`, `data:`, `//host`, teks biasa → `null` (tampil sebagai teks); label = host tanpa `www.` + jalur dipendekkan |
| `parseQuantity`, `lineTotalX100`, `lineTotal` | `ShoppingItemModal` (pratinjau dan galat dini) | tanpa float: `BigInt`, setengah ke atas seperti PostgreSQL; > 10¹³ → `null` (server menolak dengan `400`); "1.000" ditolak (bukan 1000) |
| `filterVendors`, `countVendors`, `vendorScope`, `filterItems` | chip sesi/status/kategori | `SCOPE_ANY` = semua; `'all'` = hanya Umum; sesi usang tampil sebagai Umum |
| `tableRows` | `VendorTable` | anggota kelompok pagu berdampingan di posisi anggota pertama; sisanya urutan server |
| `deltaTone`, `groupVerdict` | kolom Selisih, `LimitGroupSummary` | kelompok: `remaining` (sisa pagu) selama `priced < members`; `over` begitu `delta < 0` |
| `paymentText`, `estimateDiffers` | kolom Status bayar, lencana "Pos ≠ deal" | memakai `paymentBadge` Menu 03 (lebih bayar, belum dianggarkan) |
| `fixNeedsPrice`, `releasesBudgetLine` | `VendorModal`, menu status | cermin aturan server (deal wajib; konfirmasi melepas pos) |
| `missingCategories`, `categoryProgress`, `categoryPayload`, `toggleSkipped` | `VendorKpiCards`, `CategoryChecklist`, `VendorModal` | progres `null` bila semua kategori dilewati; kategori kustom dikirim sebagai label; batas 50 |
| `nextShoppingStatus`, `sumItems`, `countByCategory`, `categoryOptions`, `itemNumbers`, `completionRatio` | `ShoppingTable`, `ShoppingFilters`, `ShoppingKpiCards` | realisasi hanya barang selesai (`actual_total`); nomor tampilan dari `sort_order` |
| `shoppingTemplateImpact`, `rollupState`, `souvenirLeft` | `ShoppingTemplateMenu`, `RollupBanner` | meniru syarat `confirm` server; bila server menjawab `confirm: required` padahal lokal tidak meminta, tampilkan kotak konfirmasi lalu kirim ulang |
| `LIMITS` | `maxlength`, tombol | cermin batas server (BE §3.4); bukan pengganti server |

---

## 6. Pembungkus panggilan, BFF, dan unduhan

Tidak ada file BFF baru. Pemanggilan klien memakai `plannerFetch('projects/:id/…')` (01 FE bab 5.5) dengan path yang lolos aturan BFF `^[a-z0-9/_-]+$` (`export-contacts`, `lock-deal`, `link-to-budget`, `unlink-from-budget`, `apply-template`, `shopping-items`, `status` semuanya lolos; query `?scope=` dan `?format=xlsx` bukan bagian path). Semua non-GET mengirim `Content-Type: application/json` dengan body (`'{}'` bila kosong). Menu ini memakai metode `PUT` dan `PATCH`.

### 6.1 `vendorApi.ts`

Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan sebagai argumen sehingga modul bisa diuji tanpa peramban. Setiap id di path diperiksa UUID huruf kecil dan melempar `Error` bila tidak (kesalahan pemrogram, bukan permintaan ke server). **Sudah dijalankan**: tes unit memeriksa metode, path, dan body ke-16 fungsi (18 panggilan), bahwa setiap fungsi dipakai, dan bahwa id yang bukan UUID (`../../auth`, `x`) melempar.

```ts
import type {
  ApiError, LinkToBudgetBody, LinkToBudgetResult, LockDealBody, LockDealResult, ShoppingItem, ShoppingItemBody, ShoppingItemUpdate,
  ShoppingOverview, ShoppingStatus, ShoppingTemplateBody, ShoppingTemplateResult, UnlinkResult, UpdateProjectResult, Vendor, VendorBody,
  VendorOverview, VendorResult, VendorStatus,
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
const only = (q: Record<string, string | undefined>): { query: Record<string, string> } | undefined => {
  const entries = Object.entries(q).filter((e): e is [string, string] => e[1] !== undefined && e[1] !== '');
  return entries.length === 0 ? undefined : { query: Object.fromEntries(entries) };
};

/** Satu fungsi per aksi di UI (kolom "Fungsi" di bab 4.4). Semua non-GET mengirim body (plannerFetch mengisi '{}' bila undefined). */
export function vendorApi(call: PlannerFetch, projectId: string) {
  const p = `projects/${uuid(projectId)}`;
  const vd = (id: string) => `${p}/vendors/${uuid(id)}`;
  const sh = (id: string) => `${p}/shopping-items/${uuid(id)}`;
  return {
    // ----- Tab 1: vendor -----
    /** Filter server menyaring `vendors`; KPI, kategori, dan kelompok pagu selalu seluruh proyek. Halaman memilih menyaring di klien (hidden). */
    listVendors: (f?: { scope?: string; status?: VendorStatus; category?: string }) => call<VendorOverview>(`${p}/vendors`, only({ ...f })),
    getVendor: (id: string) => call<Vendor>(vd(id)),
    createVendor: (body: VendorBody) => call<VendorResult>(`${p}/vendors`, { method: 'POST', body }),
    updateVendor: (id: string, body: VendorBody) => call<VendorResult>(vd(id), { method: 'PUT', body }),
    /** `fix` hanya bila `agreed_price > 0`; meninggalkan `fix` melepas tautan pos (peringatan budget_link_released). */
    setVendorStatus: (id: string, status: VendorStatus) => call<VendorResult>(`${vd(id)}/status`, { method: 'PATCH', body: { status } }),
    lockDeal: (id: string, body: LockDealBody) => call<LockDealResult>(`${vd(id)}/lock-deal`, { method: 'POST', body }),
    deleteVendor: (id: string) => call<null>(vd(id), { method: 'DELETE' }),
    /** Tidak ada endpoint kategori di Menu 04: memakai PATCH proyek Menu 01, yang MENGGANTI seluruh daftar (`toggleSkipped`). */
    setSkippedCategories: (keys: string[]) => call<UpdateProjectResult>(p, { method: 'PATCH', body: { skipped_vendor_categories: keys } }),

    // ----- Tab 2: belanja -----
    listShopping: (f?: { scope?: string; category?: string; status?: ShoppingStatus }) => call<ShoppingOverview>(`${p}/shopping-items`, only({ ...f })),
    createShoppingItem: (body: ShoppingItemBody) => call<ShoppingItem>(`${p}/shopping-items`, { method: 'POST', body }),
    /** PUT mengganti penuh, tetapi tidak mengubah tautan rollup (`budget_item_id`), `template_key`, dan urutan. */
    updateShoppingItem: (id: string, body: ShoppingItemUpdate) => call<ShoppingItem>(sh(id), { method: 'PUT', body }),
    setShoppingStatus: (id: string, status: ShoppingStatus) => call<ShoppingItem>(`${sh(id)}/status`, { method: 'PATCH', body: { status } }),
    deleteShoppingItem: (id: string) => call<null>(sh(id), { method: 'DELETE' }),
    applyShoppingTemplate: (body: ShoppingTemplateBody) => call<ShoppingTemplateResult>(`${p}/shopping-items/apply-template`, { method: 'POST', body }),
    /** Tanpa `budget_item_id`: pos bawaan jenis acara (`default_target`); tanpa `item_ids`: semua barang yang belum bertaut ke pos rollup lain. */
    linkToBudget: (body: LinkToBudgetBody) => call<LinkToBudgetResult>(`${p}/shopping-items/link-to-budget`, { method: 'POST', body }),
    unlinkFromBudget: (budgetItemId: string) => call<UnlinkResult>(`${p}/shopping-items/unlink-from-budget`, { method: 'POST', body: { budget_item_id: uuid(budgetItemId) } }),
  };
}

/** Tujuan tautan unduhan .xlsx (GET biasa lewat BFF; bukan plannerFetch karena responsnya berkas). Hanya `xlsx`: PDF belum disetujui. */
export const exportContactsHref = (projectId: string): string => `/dashboard/api/planner/projects/${uuid(projectId)}/vendors/export-contacts?format=xlsx`;
export const exportShoppingHref = (projectId: string): string => `/dashboard/api/planner/projects/${uuid(projectId)}/shopping-items/export-shopping-list?format=xlsx`;
```

Pemakaian di skrip klien:

```ts
const api = vendorApi(plannerFetch, projectId);
const res = await api.lockDeal(vendorId, { agreed_price, sync_to_budget_items: true });
if (!res.ok) { toast(vendorErrorMessage(res.status, res.error?.code, 'vendor') ?? plannerErrorMessage(res.status, res.error?.code)); return; }
showLockDealResult(res.data);   // LockDealResult | null: peringatan lewat vendorWarningText, pos bersama lewat sharedItemText
await refreshArea();            // partial
```

### 6.2 Unduhan `.xlsx`

Tautan biasa `<a href={exportContactsHref(project.id)} download>` dan `<a href={exportShoppingHref(project.id)} download>`; `GET` tidak butuh `Content-Type`, cookie sesi ikut. BFF meneruskan berkas sebagai aliran beserta `Content-Disposition` dari backend dan menambahkan `Cache-Control: no-store` (berkas berisi data pribadi). Bila sesi habis, middleware membalas pengalihan ke halaman login. `429` membuka halaman galat BFF; alternatif `fetch` + Blob agar bisa menampilkan teks batas dicatat di bab 13 butir 8. **Hanya `xlsx`**: versi awal menjanjikan PDF, yang belum disetujui pustakanya.

---

## 7. Tab 1: Direktori & Kontak Vendor

### 7.1 Wireframe (Proyek Contoh)

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Vendor & Belanja                                         [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ]       │
│ [★ Direktori & Kontak Vendor]   [Belanja Pengadaan & Logistik]                                           │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────┐ ┌────────────────────┐ ┌────────────────────┐ ┌────────────────────┐         │
│ │ Kategori Terkunci    │ │ Deal Vendor FIX    │ │ Kandidat Berjalan  │ │ Kelompok Pagu      │         │
│ │ 7 dari 10     70%    │ │ Rp 58.000.000      │ │ 3 vendor           │ │ 3 kelompok         │         │
│ │ [███████░░░]         │ │ 7 vendor FIX       │ │ penawaran Rp 4,3 Jt│ │ 0 melebihi pagu    │         │
│ │ Belum: Hand Bouquet, │ │                    │ │                    │ │                    │         │
│ │ MC, Entertainment    │ │                    │ │                    │ │                    │         │
│ └──────────────────────┘ └────────────────────┘ └────────────────────┘ └────────────────────┘         │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Kategori: [✅ Venue] [✅ Decoration] [✅ Wedding Organizer] [✅ MUA] [✅ Catering] [✅ Photography]        │
│           [✅ Attire] [⏳ Hand Bouquet] [⏳ MC] [⏳ Entertainment]                                        │
│ Sesi: [Semua (10)] [Lamaran (0)] [Akad Nikah (0)] [Resepsi (10)]                                         │
│ Status: [Semua (10)] [FIX (7)] [Negosiasi (2)] [Rencana (1)] [Batal (0)]                                 │
│ [ + Tambah Vendor ]                                                      [ 📥 Unduh Kontak Panitia & WO ] │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│   Kategori      Vendor                       PIC / WA          Pagu        Deal/Tawaran Selisih  Bayar   │
│ ┌ Venue         Sasana Kriya Ballroom        081200000001 💬   kelompok    12.000.000          –      FIX│
│ └ Decoration    Pelaminan Mega Indah         081200000002 💬   kelompok     8.000.000          –      FIX│
│   ╰ Pagu kelompok 20.000.000 · penawaran 20.000.000 · Pas        Pos "Sewa Gedung & Dekorasi": DP Rp 900 Rb │
│   Wedding Org.  Harmony Wedding Organizer    081200000003 💬   –           5.000.000           –      FIX│
│ ┌ MUA           Sanggar Rias Ayu             081200000004 💬   kelompok     3.500.000          –      FIX│
│ └ Attire        Butik Kebaya Anggun          081200000007 💬   kelompok     1.500.000          –      FIX│
│   ╰ Pagu kelompok 5.000.000 · penawaran 5.000.000 · Pas          Pos "Rias & Baju Pengantin": DP Rp 600 Rb  │
│   Catering      Katering Bu Minah (500 pax)  081200000005 💬   30.000.000  25.000.000  Hemat 5 jt FIX│
│                                              Rp 15 Jt dari Rp 25 Jt                                      │
│   Photography   Studio Cahaya                081200000006 💬   3.000.000   3.000.000   Pas   FIX│
│                                              DP Rp 300 Rb                                                │
│   Hand Bouquet  Fleur Hand Bouquet           081200000008 💬   –           900.000             –  Rencana [🔒 FIX]│
│ ┌ MC            Kang Dedi Master MC          081200000009 💬   kelompok     2.000.000          –  Negosiasi [🔒 FIX]│
│ └ Entertainment Akustik Nada Kirana          081200000010 💬   kelompok     1.400.000          –  Negosiasi [🔒 FIX]│
│   ╰ Pagu kelompok 3.400.000 · penawaran 3.400.000 · Pas          Pos "Hiburan & Pengisi acara": Lunas       │
│ Tiap baris: [💬 WA] [✏️ Ubah] [⋮ Status · Kunci FIX · Hapus]                                              │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Kolom Status bayar berada di sebelah kanan (dipotong pada gambar): Catering dan Photography menampilkan teks `paymentText` di bawah baris; vendor pada pos bersama tidak punya status bayar sendiri, jadi barisnya "–" dan status pos bersama ada di baris ringkasan kelompok (`╰`). Chip sesi menampilkan semua sesi non-sistem (redup bila 0) dan **Umum** hanya bila ada vendor di sana. Pada wireframe lama: pagu Catering 25.000.000 (seharusnya 30.000.000 dengan selisih hemat 5.000.000), status bayar Venue "DP 0,9 jt" (Venue tidak bertaut sendiri), dan chip "Riset (3)" yang menjumlah Rencana dan Negosiasi; semuanya dikoreksi.

### 7.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu 1 `VendorKpiCards` | `kpi.categories_fixed/total`, `categoryProgress`, `missingCategories` | "7 dari 10" + bar `barWidth` + `formatPercent(p, 0)`; `categories_total = 0` (semua dilewati) → "–" dan "Semua kategori dilewati"; "Belum: …" menyebut paling banyak tiga nama lalu "+n" |
| Kartu 2 | `kpi.fixed_total`, `fixed_vendors` | `formatRupiah(fixed_total)` dan "{n} vendor FIX" |
| Kartu 3 | `kpi.pipeline`, `pipeline_offers` | "{n} vendor"; baris "penawaran {formatRupiahShort}" disembunyikan bila 0 |
| Kartu 4 | `kpi.groups_total`, `groups_over` | "{n} kelompok" dan "{m} melebihi pagu" (merah bila m > 0); `groups_total = 0` → "–" dan keterangan "Terapkan skenario di Keuangan untuk melihat pagu" |
| `CategoryChecklist` | `categories` | satu chip per kategori: ✅ `fix` (Terkunci) · ⏳ `pipeline` (Berjalan) · kosong; angka `vendors`; klik menyaring tabel; kategori kosong menawarkan "Tambah vendor" dan "Tandai tidak diperlukan"; kategori `skipped` redup dengan "Diperlukan lagi"; kategori `custom` bertanda |
| `VendorFilters` | `countVendors`, `sessions` | chip Semua + semua sesi non-sistem (redup bila 0) + Umum bila > 0; chip status Semua, FIX, Negosiasi, Rencana, Batal |
| `VendorTable` | `tableRows` | satu baris per vendor atau satu blok per kelompok pagu (braket ┌ └ + baris ringkasan `LimitGroupSummary`); vendor `batal` redup di akhir blok urutan server |
| Kolom Pagu | `limit_source`, `budget_limit` | `manual` → angka + "manual"; `scenario` → angka + "skenario"; `group` → "kelompok" (angka ada di baris ringkasan); `none` → "–" |
| Kolom Deal/Tawaran | `agreed_price`, `status` | `0` → "–"; vendor `fix` → "Deal", lainnya "Tawaran" (teks kecil) |
| Kolom Selisih | `price_delta`, `deltaTone` | `saving` → "Hemat Rp X" hijau; `over` → "Melebihi Rp X" merah; `even` → "Pas"; `none` → "–" (**bukan** "hemat penuh") |
| `LimitGroupSummary` | `LimitGroup`, `groupVerdict` | "Pagu kelompok {limit} · penawaran {total_agreed} · {Hemat/Pas/Melebihi/Sisa pagu} {|delta|}"; `none` → "Belum ada penawaran"; di kanan `paymentText(group.line)` dengan nama pos; tanpa `line` tidak ditampilkan |
| Kolom Status bayar | `paymentText(vendor.payment)` | "–" bila belum bertaut; `estimateDiffers` → lencana "Pos ≠ deal" dengan estimasi pos di tooltip |
| Kolom Status | `VENDOR_STATUS_LABELS` | lencana; vendor bukan `fix` dan bukan `batal` menampilkan tombol **🔒 FIX** |
| Tombol WA | `formatted_wa_number`, `waLink`, `waMessage` | tanpa nomor → tombol nonaktif "Nomor belum diisi"; selain itu buka `wa.me` di tab baru dengan templat pesan (nama sesi bukan Umum) |
| Keadaan kosong | `vendors = []` | "Belum ada vendor. Mulai dari checklist kategori di atas atau tambah vendor." |

### 7.3 Kontrol

- **`+ Tambah Vendor`** membuka `VendorModal`; sesi bawaan = chip sesi aktif bila sesi nyata, selain itu Umum. **`📥 Unduh Kontak Panitia & WO`** = `exportContactsHref` (hanya vendor FIX; tanpa harga, bab 6.2). Tombol tulis nonaktif bila `project.archived` (keterangan "Acara diarsipkan").
- **Chip** (kategori, sesi, status) menyaring baris lewat atribut `hidden` dan `history.replaceState` (`?category=`, `?scope=`, `?status=`); tidak memanggil server. Blok kelompok yang sebagian anggotanya tersaring tetap menampilkan ringkasan kelompok penuh.
- **Menu ⋮ per baris**: *Ubah status* (Rencana, Negosiasi, Batal; **FIX membuka `LockDealModal`**, bukan `setVendorStatus`), *Kunci FIX* (bila bukan fix), *Hapus*.

### 7.4 Modal dan dialog

1. **`VendorModal`** (tambah dan ubah; `PUT` mengganti penuh, jadi formulir mengisi awal **semua** kolom dari baris): sesi (master sesi, termasuk Umum), kategori (dropdown: kategori registry, kategori kustom yang dipakai, dan "Kategori baru…" dengan isian teks ≤ 60; `categoryPayload`), nama vendor (≤ 255), tautan portofolio (URL lengkap `http(s)://…`, ≤ 500; versi awal menyebut "URL/IG" tetapi hanya URL lengkap yang diterima), PIC (≤ 255), nomor ponsel (≤ 40; pratinjau "Dikirim sebagai +62…" dari `normalizeWA`, `null` → galat kolom `invalid_phone` tanpa permintaan), pagu (opsional; kosong = ikut skenario; keterangan menampilkan pagu skenario yang sedang berlaku bila ada), penawaran/deal (`parseRupiah`; 0 = belum ada), status (Rencana, Negosiasi, FIX, Batal; `fixNeedsPrice` → galat kolom "wajib diisi untuk FIX"), catatan (≤ 2.000, baris baru boleh). Mengubah vendor `fix` bertaut pos menjadi status lain → `ConfirmDialog` ("Tautan ke pos anggaran akan dilepas; pos tetap ada") bila `releasesBudgetLine`.
2. **`LockDealModal`**: harga deal final (`parseRupiah`, terisi dari penawaran), kotak centang **Sinkronkan ke pos anggaran** (bawaan aktif) dengan keterangan menurut keadaan: kategori berbagi pos (`group`) → "Pos '{nama}' dipakai bersama {anggota}; estimasinya tidak akan diubah." Hasil (`LockDealResult`) ditampilkan di panel yang sama sebelum ditutup: **ditautkan** ("Vendor ditautkan ke pos '{vendor.payment.item_name}'; estimasi menjadi {deal}."), **pos baru** ("Pos baru '{nama}' dibuat dengan estimasi {deal}."), **pos bersama** (`sharedItemText` + tombol **Samakan estimasi pos dengan total deal** dan tautan "Buka Realisasi"), **pos satuan** (`vendorWarningText('unit_pricing')`), atau **tanpa sinkron** ("Vendor dikunci tanpa mengubah anggaran.").
3. **`ConfirmDialog`**: hapus vendor ("Vendor dihapus; pos anggaran, tugas, belanja, dan agenda yang menunjuknya tetap ada tanpa vendor."), melepas FIX, "Tandai tidak diperlukan" tidak perlu konfirmasi.

### 7.5 Alur interaksi (Tab 1)

| Alur | Langkah | Panggilan |
|---|---|---|
| A. Chip | klik → `hidden` pada baris lain + `replaceState` | – |
| B. Tambah | modal → `createVendor` → partial; galat `invalid_phone`/kolom di formulir | `createVendor` |
| C. Ubah | modal terisi → `updateVendor` (semua kolom) → partial; peringatan `estimate_differs`/`budget_link_released` ditampilkan sebagai toast | `updateVendor` |
| D. Ubah status | menu → `setVendorStatus` → partial (FIX → alur E) | `setVendorStatus` |
| E. Kunci FIX | modal → `lockDeal` → panel hasil → tutup → partial | `lockDeal` |
| F. Samakan estimasi pos | panel hasil pos bersama → `financeApi.listLines` lalu `financeApi.updateLine(pos, {…semua kolom pos, initial_estimate = total deal})` (PUT mengganti penuh dan `vendor_id` yang tak dikirim melepas vendor: kirim kembali seluruh isian pos) → partial. Pos berharga satuan: tombol nonaktif ("Ubah harga satuan di Keuangan") | `listLines`, `updateLine` (03) |
| G. Hapus | konfirmasi → `deleteVendor`; `404` dianggap sukses → partial | `deleteVendor` |
| H. Kategori dilewati | "Tandai tidak diperlukan" / "Diperlukan lagi" → `setSkippedCategories(toggleSkipped(project.skippedCategories, key, skip))` (mengganti seluruh daftar; `null` → toast batas) → partial | `setSkippedCategories` |
| I. WhatsApp | `waLink(vendor.formatted_wa_number, waMessage(vendor, project.title, namaSesi))` di tab baru (`rel="noopener noreferrer"`) | – |
| J. Ekspor | tautan | `exportContactsHref` |

#### 7.5.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Dipakai untuk |
|---|---|---|
| Baris / kartu vendor | `data-vendor-id`, `data-scope`, `data-status`, `data-category`, `data-group`, `data-fix-release`, `data-focus-key` | chip, konfirmasi melepas FIX, fokus setelah partial |
| Baris ringkasan kelompok | `data-group-key`, `data-line-id` | alur F |
| Chip kategori | `data-category-key`, `data-state`, `data-skipped` | alur A, H |
| Blok data | `<script type="application/json" data-vendor-data>` | daftar vendor untuk modal ubah dan pratinjau; `<` di-escape `<` dan dibaca dengan `JSON.parse(textContent)` |

### 7.6 Salinan teks

| Tempat | Teks |
|---|---|
| Keadaan kosong | "Belum ada vendor. Mulai dari checklist kategori di atas atau tambah vendor." |
| Tombol tambah / ekspor | "+ Tambah Vendor" / "Unduh Kontak Panitia & WO" |
| Kategori kosong | "Tambah vendor" · "Tandai tidak diperlukan" · (dilewati) "Diperlukan lagi" |
| Pagu | "manual" · "skenario" · "kelompok" |
| Selisih | "Hemat Rp X" · "Melebihi Rp X" · "Pas" · "Sisa pagu Rp X" · "Belum ada penawaran" |
| Melepas FIX | "Tautan vendor ke pos anggaran akan dilepas; pos tetap ada sebagai pos biasa." |
| Ekspor kontak | "Berkas ini hanya memuat kontak vendor FIX, tanpa harga." |

---

## 8. Tab 2: Belanja Pengadaan & Logistik

### 8.1 Wireframe (37 barang PDF; Proyek Contoh)

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ [Direktori & Kontak Vendor]   [★ Belanja Pengadaan & Logistik]                                           │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────┐ ┌───────────────────┐ ┌────────────────────┐ ┌────────────────────┐              │
│ │ Total Estimasi   │ │ Realisasi         │ │ Selesai            │ │ On Proses          │              │
│ │ Rp 4.578.500     │ │ Rp 2.063.800      │ │ 16 dari 37         │ │ 21 dari 37         │              │
│ │ 37 barang        │ │ (barang selesai)  │ │ [████░░░░░] 43,2%  │ │ [█████░░░░] 56,8%  │              │
│ └──────────────────┘ └───────────────────┘ └────────────────────┘ └────────────────────┘              │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ 🧺 Pos 'Seserahan' belum dihitung dari belanja dan sudah punya pembayaran tercatat.                      │
│                                                         [ Hitung pos dari daftar ini ]                    │
│ Sesi: [Semua (37)] [Umum (37)] [Lamaran (0)] [Akad Nikah (0)] [Resepsi (0)]                              │
│ Status: [Semua (37)] [Selesai (16)] [On Proses (21)] [Belum (0)]                                         │
│ Kategori: [Semua] [Perangkat Alat Solat 4] [Make Up & Skin Care 14] [Peralatan Mandi 8]                  │
│           [Pakaian Dalam 4] [Pakaian Luar 5] [Kebutuhan Lain 2]                                          │
│ [ + Tambah Barang ]  [ Terapkan Template ▾ ]                                  [ 📥 Unduh (.xlsx) ]        │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Kategori               Barang                    Jml Satuan  Hrg satuan  Est. total  Riil total  Alamat Status   │
│  1 Perangkat Alat Solat   Mukena                      1  –         194.000     194.000     194.000   –    ✅ Selesai │
│  2 Perangkat Alat Solat   Al-Quran                    1  –          70.500      70.500           0   –    🟡 On Proses│
│  5 Make Up & Skin Care    Cushion                     1  –         175.700     175.700     175.700   –    ✅ Selesai │
│ 37 Kebutuhan Lain         Box Seserahan (10 Kotak)    1  –          37.900      37.900           0   –    🟡 On Proses│
├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ TOTAL 37 barang                                                               4.578.500   2.063.800                │
│  • 16 selesai                                                                 2.063.800   2.063.800                │
│  • 21 on proses                                                               2.514.700           0                │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Kolom *Alamat* pada Proyek Contoh kosong ("–"): PDF asli berisi tautan pribadi pembuatnya dan tidak disalin; contoh barang dengan alamat: `https://www.shopee.co.id/…` tampil sebagai tautan `shopee.co.id/…`, "Toko Mawar, Jl. Melati 3" sebagai teks. Semua barang template berada di sesi Umum, jadi chip Lamaran, Akad, Resepsi berangka 0. Karena `has_payments` benar pada Proyek Contoh, tombol "Hitung pos" membuka konfirmasi dulu.

### 8.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu 1 | `kpi.estimated_total`, `total` | `formatRupiah` dan "{n} barang" |
| Kartu 2 | `kpi.actual_total` | `formatRupiah`; keterangan tetap "(barang selesai)" |
| Kartu 3 | `kpi.done`, `total`, `completionRatio` | "16 dari 37" + bar + `formatPercent(p, 1)`; `total = 0` → "–" |
| Kartu 4 | `kpi.in_progress`, `total`; `kpi.not_started` | "21 dari 37" + bar `percent(in_progress, total)`; baris "Belum {n}" bila > 0 |
| Banner stok | `kpi.souvenir_stock`, Menu 05 | tampil hanya bila `souvenir_stock` ada: "Stok souvenir {stok}"; bila serah terima Menu 05 tersedia: "· diserahkan {n} · sisa {souvenirLeft}" |
| `RollupBanner` | `default_target`, `rollups`, `rollupState`, `rollupBannerText` | `off`: teks + tombol **Hitung pos dari daftar ini**; `off_with_payments`: idem, tombol membuka konfirmasi `ROLLUP_CONFIRM_TEXT` lebih dulu; `on`: teks "dihitung dari daftar ini: {estimasi}, terbayar {terbayar}" + tombol **Lepas**; `budget_item_id` kosong: "belum ada; akan dibuat". `rollups` lebih dari satu → daftar tambahan dengan **Lepas** masing-masing. `default_target` adalah pos yang akan dipakai server: untuk peresmian, wisuda, dan seminar yang vendor MC/toga/percetakannya sudah terkunci, ia berpindah ke pos cadangan "Perlengkapan" ([BE bab 2.4](BE.md#24-daftar-belanja-digulung-ke-anggaran-rollup)); antarmuka cukup menampilkan `item_name` yang diterima, tanpa cabang khusus |
| Chip | `countItems` lokal (`countByCategory`, `filterItems`) | sesi (Semua + non-sistem + Umum bila > 0), status, kategori; angka dari daftar yang dimuat |
| `ShoppingTable` | `items`, `itemNumbers`, `safeHref`, `linkLabel` | kolom No (nomor tampilan), Kategori, Barang, Jml, Satuan, Harga satuan, Est. total, Riil total, Alamat, Status, Aksi; `quantity` ditampilkan dua desimal bila pecahan; `price_delta` kecil di bawah Riil total ("hemat Rp X"/"lebih mahal Rp X"); badge sumber *Digulung ke '{pos}'* bila `budget_item_id` ada; badge *Souvenir* bila `purpose`; alamat: `safeHref` ada → `<a target="_blank" rel="noopener noreferrer">` dengan `linkLabel`, selain itu teks (`textContent`) |
| Footer | `sumItems(visible)` | total baris yang tampil: estimasi, riil (barang selesai), jumlah selesai/proses/belum |
| Keadaan kosong | `items = []` | "Belum ada barang. Terapkan template atau tambah barang." (jenis tanpa template: "Belum ada template barang untuk jenis acara ini.") |

### 8.3 Komponen dan modal

1. **`ShoppingItemModal`** (tambah dan ubah; `PUT` mengganti penuh kecuali tautan gulungan): sesi (master sesi, bawaan Umum atau chip aktif), kategori (`categoryOptions` + "Kategori baru…" ≤ 60), nama barang (≤ 255), jumlah (`parseQuantity`, bawaan 1) dan satuan (≤ 30), harga estimasi dan riil **per satuan** (`parseRupiah`), pratinjau total `lineTotal(qty, harga)` untuk keduanya (hasil `null` → galat "Total (jumlah × harga) paling banyak Rp 10.000.000.000.000" tanpa permintaan), alamat pembelian (≤ 500; teks bebas, URL lengkap dirender sebagai tautan), vendor rekanan (opsional; dimuat saat modal dibuka), status, kotak centang **Barang ini souvenir tamu** (`purpose`), kotak centang **Hitung ke pos '{nama pos}'** (hanya pada tambah, hanya bila ada pos rollup aktif: `rollupTargetId`; bawaan aktif; dikirim sebagai `budget_item_id`), catatan. Mengubah status ke Selesai dengan harga riil kosong: keterangan "Harga riil diisi sama dengan estimasi."
2. **`ShoppingTemplateMenu`** ("Terapkan Template ▾"): tampil hanya bila `type.templates_available.shopping_list`; selain itu teks "Belum ada template barang untuk jenis acara ini (kategori tetap tersedia di dropdown)." *Tambah yang kurang* (`append`) dan *Ganti template* (`replace_template`): hitung `shoppingTemplateImpact`; `needsConfirm` → dialog yang menyebut jumlah barang dihapus dan yang membawa pekerjaan (status, harga riil, tautan, souvenir); hasil `ShoppingTemplateResult` dengan `warnings` (`vendorWarningText`). Server `confirm: required` yang tak diperkirakan lokal → dialog yang sama.
3. **`RollupBanner`**: bab 8.2; konfirmasi `ROLLUP_CONFIRM_TEXT` menyebut angka pembayaran tersimpan (`default_target.paid`).
4. **Ekspor**: tautan `exportShoppingHref` (bab 6.2).

### 8.4 Pemeriksaan sebelum mengirim

Nominal `null` dari `parseRupiah` → galat kolom tanpa permintaan; jumlah `null` dari `parseQuantity` → "Jumlah harus antara 0,01 dan 1.000.000."; total `null` dari `lineTotal` → pesan batas; nama barang kosong → "Nama barang wajib diisi."; sisanya dibiarkan ke server.

### 8.5 Alur interaksi (Tab 2)

| Alur | Langkah | Panggilan |
|---|---|---|
| K. Chip | klik → `hidden` + `replaceState`; footer dihitung ulang dengan `sumItems` | – |
| L. Tambah | modal → `createShoppingItem` (dengan `budget_item_id` bila dicentang) → partial | `createShoppingItem` |
| M. Ubah | modal terisi (semua kolom) → `updateShoppingItem` → partial | `updateShoppingItem` |
| N. Toggle status | klik lencana → `setShoppingStatus(nextShoppingStatus(status))` → partial (harga riil terisi server saat Selesai) | `setShoppingStatus` |
| O. Hapus | konfirmasi → `deleteShoppingItem`; `404` dianggap sukses → partial | `deleteShoppingItem` |
| P. Hitung pos | tombol banner → bila `has_payments`: konfirmasi `ROLLUP_CONFIRM_TEXT` → `linkToBudget({ confirm: true })`, selain itu `linkToBudget({})`; `409 rollup_conflict` yang tak terduga → konfirmasi lalu kirim ulang; hasil: banner "dihitung dari daftar ini" → partial | `linkToBudget` |
| Q. Lepas gulungan | konfirmasi ("Pos kembali memakai angka tersimpannya") → `unlinkFromBudget(budget_item_id)` → partial | `unlinkFromBudget` |
| R. Template | menu → `shoppingTemplateImpact` → konfirmasi bila perlu → `applyShoppingTemplate` | `applyShoppingTemplate` |
| S. Ekspor | tautan | `exportShoppingHref` |

#### 8.5.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Dipakai untuk |
|---|---|---|
| Baris / kartu barang | `data-item-id`, `data-scope`, `data-category`, `data-status`, `data-linked`, `data-est`, `data-actual`, `data-focus-key` | chip, footer, toggle, fokus setelah partial |
| Banner rollup | `data-rollup-state`, `data-target-id`, `data-has-payments`, `data-paid` | alur P, Q |
| Blok data | `<script type="application/json" data-vendor-data>` | daftar barang untuk modal ubah dan dampak template |

### 8.6 Salinan teks

| Tempat | Teks |
|---|---|
| Banner gulungan | `rollupBannerText` (bab 9.4) · "Hitung pos dari daftar ini" · "Lepas" |
| Konfirmasi gulungan | `ROLLUP_CONFIRM_TEXT` |
| Lepas gulungan | "Pos kembali memakai angka tersimpannya. Semua barang dilepas dari pos ini." |
| Ganti template | "{n} barang template akan dihapus; {m} di antaranya sudah berstatus, berharga riil, atau tertaut. Lanjutkan?" |
| Keadaan kosong | "Belum ada barang. Terapkan template atau tambah barang." |

---

## 9. Pesan galat dan teks

Semua teks lewat `lib/planner/messages.ts` (01 FE bab 5.4 dan 11): kode lama tetap, berikut tambahan menu ini. **Pola pemakaian** sama dengan 02 dan 03: fungsi Menu 04 mengembalikan `null` bila bukan miliknya, dan pemanggil jatuh ke fungsi Menu 03 lalu 01:

```ts
const text = vendorErrorMessage(res.status, res.error?.code, 'vendor') ?? financeErrorMessage(res.status, res.error?.code) ?? plannerErrorMessage(res.status, res.error?.code);
const fieldText = vendorFieldMessage(field, reason) ?? financeFieldMessage(field, reason) ?? fieldErrorMessage(field, reason);
```

### 9.1 Kode → teks

| Kode / kondisi | Teks Indonesia | Fungsi |
|---|---|---|
| `vendor_limit_reached` | "Jumlah vendor sudah mencapai batas (200). Hapus vendor yang tidak dipakai lalu coba lagi." | `vendorErrorMessage` |
| `shopping_item_limit_reached` | "Jumlah barang belanja sudah mencapai batas (1.000). Hapus barang yang tidak dipakai lalu coba lagi." | idem |
| `invalid_phone` | "Nomor ponsel tidak valid. Gunakan 08… atau +62…." | idem |
| `invalid_template` | "Template belanja tidak tersedia untuk jenis acara ini." | idem |
| `rollup_conflict` | `ROLLUP_CONFIRM_TEXT` — bukan galat tampil: kotak konfirmasi, lalu kirim ulang dengan `confirm: true` | idem |
| `not_found`, `where` = `vendor` / `shopping` | "Vendor tidak ditemukan. Muat ulang daftar." / "Barang tidak ditemukan. Muat ulang daftar." | idem |
| `not_found`, `where = 'project'` | "Acara tidak ditemukan." (halaman: `/404`) | 01 |
| `rate_limited`, `where = 'export'` | "Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi." | idem |
| `invalid_scope` | "Sesi tidak valid atau sudah dihapus. Muat ulang halaman." | 03 (`financeErrorMessage`) |
| `invalid_input`, `invalid_date` | **per kolom** (9.2); tanpa kolom: "Periksa kembali data Anda." | 01 |
| `invalid_json`, `project_read_only`, `conflict`, `401`, `5xx`, jaringan | sama dengan [01 FE bab 11.1](../01_DASHBOARD/FE.md#111-kode--teks-plannererrormessagestatus-code) | 01 |

`where` ditentukan oleh **aksi** yang sedang berjalan, bukan oleh pesan server.

### 9.2 Alasan per kolom

Alasan yang sudah ada di 01 FE bab 11.2 dan 03 FE bab 10.2 (wajib/`must not be empty`, panjang, karakter kontrol, `must be an integer`, `must be at least N`, `must be between 0 and N`, `must be between A and B` untuk `quantity`, `must be a vendor of this project`, `must be one of …`, `must be greater than 0`) tetap berlaku. Tambahan Menu 04 (`vendorFieldMessage`; **setiap alasan baku BE §3.4 sudah tercakup atau diserahkan eksplisit**, diuji):

| Pola alasan (regex) | Kolom | Teks (`{label}` = label kolom, 9.3) |
|---|---|---|
| `^must be an http or https URL$` | `portfolio_url` | "{label} harus diawali http:// atau https://." (kolom lain, mis. `receipt_url`: teks Menu 03) |
| `^must be an Indonesian phone number \(08…, \+62…\)$` | `phone_number` | "Nomor ponsel tidak valid. Gunakan 08… atau +62…." |
| `^must be a registry category key$` | `category_key` | "Kategori tidak dikenal. Pilih dari daftar atau ketik nama kategori baru." |
| `^required when category_key is empty$` | `category_label` | "Nama kategori wajib diisi untuk kategori baru." |
| `^must be greater than 0 when status is fix$` | `agreed_price` | "{label} wajib diisi (lebih dari 0) untuk vendor berstatus FIX." |
| `^the total \(quantity × price\) must be at most (\d+)$` | `estimated_price`, `actual_price` | "Total (jumlah × harga) paling banyak Rp 10.000.000.000.000." |
| `^must be a shopping rollup line of this project$` | `budget_item_id` | "Pos tujuan tidak valid. Muat ulang halaman." |
| `^must be a budget line of this project$` | `budget_item_id` | "Pos tidak ditemukan di acara ini. Muat ulang halaman." |
| `^must not be a unit-priced line$` | `budget_item_id` | "Pos berharga satuan tidak dapat dihitung dari belanja." |
| `^must not be linked to a vendor or document$` | `budget_item_id` | "Pos yang tertaut ke vendor atau dokumen tidak dapat dihitung dari belanja." |
| `^must be items of this project$` | `item_ids` | "Ada barang yang tidak ditemukan di acara ini. Muat ulang daftar." |
| `^at most (\d+) categories$` | `skipped_vendor_categories` | "Paling banyak 50 kategori yang dapat dilewati." (alasan Menu 01) |
| `confirm: required` | `confirm` | bukan teks galat: tampilkan kotak konfirmasi dan kirim ulang dengan `confirm` (bab 8.5, alur R) |
| alasan lain | – | diserahkan ke Menu 03 lalu 01 ("{label} tidak valid.") |

### 9.3 Label kolom

`VENDOR_FIELD_LABELS` (kode di 9.4): `sub_event_scope` "Sesi", `category_key`/`category_label`/`category` "Kategori", `vendor_name` "Nama vendor", `pic_name` "PIC", `phone_number` "Nomor ponsel", `portfolio_url` "Tautan portofolio", `budget_limit` "Pagu", `agreed_price` "Penawaran / deal", `status` "Status", `notes` "Catatan", `item_name` "Nama barang", `quantity` "Jumlah", `unit` "Satuan", `estimated_price` "Harga estimasi", `actual_price` "Harga riil", `purchase_location` "Alamat pembelian", `vendor_id` "Vendor", `budget_item_id` "Pos anggaran", `purpose` "Tujuan", `item_ids` "Barang", `mode` "Mode", `confirm` "Konfirmasi", `skipped_vendor_categories` "Kategori yang dilewati". Kolom tak dikenal: "Isian".

### 9.4 Kode `messages.ts` (tambahan)

Label status, teks peringatan, teks banner gulungan, label kolom, dan dua fungsi galat. **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-vendortestmjs)).

```ts
import { formatRupiah, formatRupiahShort } from './format.ts';
import type { CategoryStateKey, LimitSource, Notice, RollupTarget, ShoppingStatus, VendorStatus } from './types.ts';
import type { GroupVerdict } from './vendor.ts';

// ---------- Tambahan Menu 04 untuk messages.ts (pola: pemanggil memakai `…Message(…) ?? kamus Menu 03/01`) ----------

/** Aksi yang memanggil: menentukan teks `not_found` dan `rate_limited`, yang tidak dapat dibedakan dari kode/pesan server. */
export type VendorWhere = 'project' | 'vendor' | 'shopping' | 'export';

export const VENDOR_STATUS_LABELS: Record<VendorStatus, string> = { rencana: 'Rencana', negosiasi: 'Negosiasi', fix: 'FIX', batal: 'Batal' };
export const CATEGORY_STATE_LABELS: Record<CategoryStateKey, string> = { fix: 'Terkunci', pipeline: 'Berjalan', empty: 'Kosong' };
export const SHOPPING_STATUS_LABELS: Record<ShoppingStatus, string> = { belum: 'Belum', proses: 'On Proses', selesai: 'Selesai' };
export const LIMIT_SOURCE_LABELS: Record<LimitSource, string> = {
  manual: 'Pagu manual', scenario: 'Dari skenario aktif', group: 'Kelompok pagu', none: 'Tanpa pagu',
};
export const GROUP_VERDICT_LABELS: Record<GroupVerdict, string> = {
  none: 'Belum ada penawaran', remaining: 'Sisa pagu', saving: 'Hemat', even: 'Pas', over: 'Melebihi pagu',
};

/** Tambahan untuk WARNING_LABELS (01 dan 03). `message` dari server berbahasa Inggris dan tidak pernah ditampilkan. {count} diganti `count`. */
export const VENDOR_WARNING_LABELS: Record<string, string> = {
  shared_budget_item: 'Pos anggaran ini dipakai bersama beberapa vendor, jadi estimasinya tidak diubah.',
  unit_pricing: 'Estimasi pos dihitung dari jumlah × harga satuan, jadi tidak diubah ke harga deal.',
  estimate_differs: 'Estimasi pos anggaran berbeda dari harga deal vendor ini.',
  budget_link_released: 'Tautan vendor ke pos anggaran dilepas; pos tetap ada sebagai pos biasa.',
  budget_link_lost: '{count} barang yang digulung ke pos anggaran ikut terhapus; angka pos dihitung ulang.',
};
export const vendorWarningText = (n: Pick<Notice, 'code' | 'count'>): string =>
  (VENDOR_WARNING_LABELS[n.code] ?? 'Ada catatan pada hasil ini.').replace('{count}', String(n.count));

/** Pos bersama (`shared_item` pada hasil lock-deal): "Pos 'Sewa Gedung & Dekorasi' (Rp 20 Jt) dipakai bersama; total deal kelompok Rp 12 Jt." */
export const sharedItemText = (s: { item_name: string; plan: number; total_agreed: number }): string =>
  `Pos '${s.item_name}' (${formatRupiahShort(s.plan)}) dipakai bersama beberapa vendor; total penawaran dan deal kelompok ${formatRupiahShort(s.total_agreed)}.`;

/** Banner rollup di atas tabel belanja (Tab 2). Angka dari `default_target`; tombol dan konfirmasi di bab 8.3. */
export function rollupBannerText(t: Pick<RollupTarget, 'item_name' | 'rollup' | 'has_payments' | 'estimate' | 'paid' | 'budget_item_id'>): string {
  if (t.rollup) return `Pos '${t.item_name}' dihitung dari daftar ini: ${formatRupiah(t.estimate)}, terbayar ${formatRupiah(t.paid)}.`;
  if (t.budget_item_id === undefined) return `Pos '${t.item_name}' belum ada; akan dibuat saat dihitung dari daftar ini.`;
  return `Pos '${t.item_name}' belum dihitung dari belanja${t.has_payments ? ' dan sudah punya pembayaran tercatat' : ''}.`;
}
export const ROLLUP_CONFIRM_TEXT =
  'Pos ini sudah punya pembayaran tercatat. Pembayaran itu tetap tersimpan, tetapi diabaikan selama pos dihitung dari daftar belanja. Lanjutkan?';

export const VENDOR_FIELD_LABELS: Record<string, string> = {
  sub_event_scope: 'Sesi', category_key: 'Kategori', category_label: 'Kategori', vendor_name: 'Nama vendor', pic_name: 'PIC',
  phone_number: 'Nomor ponsel', portfolio_url: 'Tautan portofolio', budget_limit: 'Pagu', agreed_price: 'Penawaran / deal', status: 'Status',
  notes: 'Catatan', item_name: 'Nama barang', category: 'Kategori', quantity: 'Jumlah', unit: 'Satuan', estimated_price: 'Harga estimasi',
  actual_price: 'Harga riil', purchase_location: 'Alamat pembelian', vendor_id: 'Vendor', budget_item_id: 'Pos anggaran', purpose: 'Tujuan',
  item_ids: 'Barang', mode: 'Mode', confirm: 'Konfirmasi', skipped_vendor_categories: 'Kategori yang dilewati',
};

type Rule = readonly [RegExp, (m: RegExpExecArray, label: string, field: string) => string | null]; // null = lanjut ke aturan berikutnya

/** Alasan `<kolom>: <alasan>` milik Menu 04. Urutan penting: pola khusus lebih dulu. */
const VENDOR_REASONS: readonly Rule[] = [
  [/^must be an http or https URL$/, (_m, l, f) => (f === 'portfolio_url' ? `${l} harus diawali http:// atau https://.` : null)], // receipt_url: teks Menu 03
  [/^must be an Indonesian phone number \(08…, \+62…\)$/, () => 'Nomor ponsel tidak valid. Gunakan 08… atau +62….'],
  [/^must be a registry category key$/, () => 'Kategori tidak dikenal. Pilih dari daftar atau ketik nama kategori baru.'],
  [/^required when category_key is empty$/, () => 'Nama kategori wajib diisi untuk kategori baru.'],
  [/^must be greater than 0 when status is fix$/, (_m, l) => `${l} wajib diisi (lebih dari 0) untuk vendor berstatus FIX.`],
  [/^the total \(quantity × price\) must be at most (\d+)$/, (m) => `Total (jumlah × harga) paling banyak ${formatRupiah(Number(m[1]))}.`],
  [/^must be a shopping rollup line of this project$/, () => 'Pos tujuan tidak valid. Muat ulang halaman.'],
  [/^must be items of this project$/, () => 'Ada barang yang tidak ditemukan di acara ini. Muat ulang daftar.'],
  [/^must be a budget line of this project$/, () => 'Pos tidak ditemukan di acara ini. Muat ulang halaman.'],
  [/^must not be a unit-priced line$/, () => 'Pos berharga satuan tidak dapat dihitung dari belanja.'],
  [/^must not be linked to a vendor or document$/, () => 'Pos yang tertaut ke vendor atau dokumen tidak dapat dihitung dari belanja.'],
  [/^at most (\d+) categories$/, (m) => `Paling banyak ${m[1]} kategori yang dapat dilewati.`], // alasan Menu 01 untuk skipped_vendor_categories
];

/** null = bukan alasan Menu 04: pakai financeFieldMessage (Menu 03) lalu fieldErrorMessage 01. */
export function vendorFieldMessage(field: string, reason: string): string | null {
  const label = VENDOR_FIELD_LABELS[field] ?? 'Isian';
  for (const [re, text] of VENDOR_REASONS) {
    const m = re.exec(reason);
    const out = m ? text(m, label, field) : null;
    if (out !== null) return out;
  }
  return null;
}

/**
 * Kode galat Menu 04 yang teksnya bergantung konteks. null = pakai financeErrorMessage / plannerErrorMessage (invalid_json, conflict,
 * project_read_only, invalid_scope, 401, 5xx, jaringan). `not_found` dibedakan oleh AKSI pemanggil (`where`), bukan oleh pesan server.
 * `rollup_conflict` bukan galat tampil: ia meminta kotak konfirmasi (ROLLUP_CONFIRM_TEXT) lalu permintaan diulang dengan `confirm`.
 */
export function vendorErrorMessage(status: number, code: string | undefined, where?: VendorWhere): string | null {
  if (where === 'export' && (status === 429 || code === 'rate_limited')) return 'Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi.';
  switch (code) {
    case 'vendor_limit_reached': return 'Jumlah vendor sudah mencapai batas (200). Hapus vendor yang tidak dipakai lalu coba lagi.';
    case 'shopping_item_limit_reached': return 'Jumlah barang belanja sudah mencapai batas (1.000). Hapus barang yang tidak dipakai lalu coba lagi.';
    case 'invalid_phone': return 'Nomor ponsel tidak valid. Gunakan 08… atau +62….';
    case 'invalid_template': return 'Template belanja tidak tersedia untuk jenis acara ini.';
    case 'rollup_conflict': return ROLLUP_CONFIRM_TEXT;
    case 'not_found':
      switch (where) {
        case 'vendor': return 'Vendor tidak ditemukan. Muat ulang daftar.';
        case 'shopping': return 'Barang tidak ditemukan. Muat ulang daftar.';
        default: return null;
      }
    default: return null;
  }
}
```

---

## 10. Gaya, responsif, aksesibilitas, keamanan

### 10.1 Gaya

Hanya token yang ada ([01 FE bab 12.1](../01_DASHBOARD/FE.md#121-gaya)): kartu `--color-surface`, batas `--color-border`; judul `--font-display`, isi `--font-body`; lencana mengikuti pola `.status` di `dashboard/index.astro`. Status kategori: Terkunci terisi `--sage-600`, Berjalan `--sage-300`, Kosong hanya kontur `--color-border`. Status vendor: FIX terisi, Negosiasi dan Rencana kontur dengan teks, Batal redup. Selisih: hijau untuk hemat, merah (`#8e2f25`, warna bahaya yang sudah dipakai kode) untuk melebihi pagu, dengan teks "Hemat"/"Melebihi" (bukan hanya warna). Braket kelompok pagu digambar dengan garis kiri (`border-inline-start`) dan baris ringkasan `--sage-50` dengan batas `--sage-200`. Banner gulungan `--sage-50`. Jangan menambah token baru tanpa persetujuan.

### 10.2 Responsif

- `< 768px`: KPI grid 2×2; tabel vendor menjadi *Vendor Contact Cards* (lencana sesi dan kategori, pagu vs deal, tombol WA besar, braket kelompok menjadi kartu gabungan); tabel belanja menjadi *Shopping Checklist Cards* (kotak centang selesai, harga, tombol buka tautan); dialog penuh layar; chip dapat digulir horizontal dalam wadahnya.
- Target sentuh minimal 44×44 px (chip, ⋮, tombol WA, lencana status).
- Tidak ada gulir horizontal pada lebar 360 px kecuali di dalam wadah chip; uji dengan nama vendor 255 karakter, nama barang 255 karakter, dan nominal 14 digit (`overflow-wrap: anywhere`, angka `white-space: nowrap` di wadah yang bisa digulir).

### 10.3 Aksesibilitas

- Dialog `<dialog>` dengan `showModal()`; fokus kembali ke pemicu saat ditutup; formulir: `<label>`, `aria-invalid`, `aria-describedby`, ringkasan galat `role="alert"`.
- Tab bar `role="tablist"` dengan `aria-selected`; chip `aria-pressed`; tabel memakai `<table>` dengan `<th scope>`; braket kelompok juga punya padanan teks ("Kelompok Venue + Decoration") untuk pembaca layar, bukan hanya garis.
- Status kategori dan selisih punya teks, bukan hanya warna atau ikon.
- Hasil asinkron (menyimpan, hasil kunci FIX, hasil template, "Data belum diperbarui") di wilayah `aria-live="polite"`; baris yang sedang dikirim `aria-busy="true"`.
- Partial mengganti DOM: fokus dipulihkan lewat `data-focus-key`; bila elemennya hilang, ke judul area.
- Kontras dan indikator fokus terlihat; hormati `prefers-reduced-motion`.

### 10.4 Keamanan

- **Tidak ada HTML dari data:** nama vendor, PIC, catatan, nama barang, alamat pembelian, dan nama sesi dirender sebagai teks (`textContent`/`createElement`; Astro meng-escape di server). Blok JSON `data-vendor-data` ditulis dengan `<` di-escape (`<`); tidak ada `set:html` untuk data API.
- **Tautan dari pengguna** (`portfolio_url`, alamat pembelian) dirender sebagai tautan hanya lewat `safeHref` (`http`/`https` yang sah), dengan `rel="noopener noreferrer"` dan `target="_blank"`; `javascript:`, `data:`, `//host`, dan teks biasa tampil sebagai teks.
- **Tautan WhatsApp** dibentuk dari nomor yang sudah dinormalisasi server (`formatted_wa_number`, hanya digit) dan `encodeURIComponent` pada teks; nomor mentah pengguna tidak pernah masuk ke URL.
- Path dibangun hanya dari `id` yang sudah divalidasi UUID; query lewat `URLSearchParams`.
- CSRF: BFF menolak non-GET tanpa `application/json`. Token tetap di cookie HttpOnly. Tidak ada `localStorage` atau `sessionStorage`.
- Data vendor dan belanja bersifat pribadi: respons ekspor `Cache-Control: no-store`; tidak ada nominal atau nomor telepon di URL halaman; berkas kontak sengaja tanpa harga.

---

## 11. Pengujian

Pola yang sudah ada: `node:test` di `frontend/tests/*.test.mjs`, dijalankan `npm test` (`node --experimental-strip-types --test tests/*.test.mjs`). Fixture di `frontend/tests/fixtures/` (bab 4.3).

### 11.1 Unit: `tests/planner-vendor.test.mjs`

Sembilan belas tes di bawah **sudah dijalankan**: kode bab 4.2, 5, 6, 9.4 (plus 1.4) dikompilasi bersama `types.ts`, `format.ts`, dan `finance.ts` yang diekstrak dari 01–03 FE, dan fixture bab 4.3. Hasil: `tsc` strict (`noUncheckedIndexedAccess`) bersih, 19/19 lulus. Mutasi yang dicoba pada kode (**26 dari 27 menjatuhkan tes**): panjang minimal nomor diturunkan; "0" setelah `62` tidak dibuang; awalan `00` tidak dikenal; batas 10¹³ hilang; pembulatan setengah ke atas hilang; jumlah minimal 0 diterima; ambang `over` bergeser; `remaining` dimatikan; nada selisih tanpa `even`; pengelompokan ganda; `estimateDiffers` tanpa syarat `fix`; `releasesBudgetLine` tanpa syarat bertaut; batas 50 kategori; `toggleSkipped` menggandakan; realisasi memakai estimasi; dampak template hanya menghitung status; barang buatan pengguna ikut diganti; keadaan banner tanpa pembayaran; sesi usang tidak jatuh ke Umum; teks `partial`; batas vendor dalam pesan; teks `portfolio_url` tertukar; `rollup_conflict` tanpa teks; `PATCH` status jadi `PUT`; id gulungan tanpa validasi UUID; ekspor `format=pdf`. Satu mutan **selamat** karena setara: membuang pemeriksaan awalan `http` di `safeHref` (pemeriksaan protokol `URL` tetap menolak `javascript:` dan sisanya melempar).

| Tes | Masukan → keluaran (fixture Proyek Contoh) |
|---|---|
| `normalizeWA` | 26 vektor bersama `fixtures/planner-wa-vectors.json` (`081298765432`, `+62 0812…`, `0062 812…`, `021 5551234` → angka; `8129876543`, `+65…`, `0812abc`, `08123456`, `++62…` → `null`; kosong → `''`) |
| `lineTotalX100`, `lineTotal` | 300 pasangan `fixtures/planner-line-totals.json` sama dengan PostgreSQL; `lineTotal(2.5, 45000)` = 112.500; `lineTotal(0.005, 100)` = `null` (lebih dari dua desimal) |
| `parseQuantity` | `2`, `2,5`, `0.25`, `1000000` → angka; `''`, `0`, `0,00`, `1.000`, `1000001`, `-1`, `2,555`, `abc`, `1e3` → `null` |
| `safeHref`, `linkLabel` | `https://…` ok; `HTTP://Example.com` → `http://example.com/`; `javascript:`, `data:`, `//evil`, `ftp:`, teks alamat, `https://`, spasi → `null`; `shopee.co.id`; label dipendekkan ke 28 karakter |
| `waMessage`, `waLink` | teks dengan/tanpa PIC dan sesi; `encodeURIComponent` |
| fixture vendor | KPI `7 / 10 / 7 / 3 / 58.000.000 / 4.300.000 / 3 / 0`; progres 70%; belum terkunci: Hand Bouquet, MC, Entertainment; progres `null` bila total 0 |
| pagu | Catering `scenario` 30.000.000, deal 25.000.000, selisih +5.000.000 (`saving`); Photography `even`; tiga kelompok `even`; `groupVerdict`: `remaining`, `over`, `saving`, `none` |
| `tableRows` | tujuh blok: `venue_dekor:venue+dekorasi`, `wo`, `rias_busana:mua+attire`, `catering`, `fotografi`, `hand_bouquet`, `hiburan:mc+entertainment`; 10 vendor; vendor batal tidak anggota kelompok |
| `paymentText` | Catering "Rp 15 Jt dari Rp 25 Jt"; Photography "DP Rp 300 Rb"; Venue "–"; pos bersama "DP Rp 900 Rb" dan "Lunas"; belum, lebih bayar, belum dianggarkan |
| `estimateDiffers`, `fixNeedsPrice`, `releasesBudgetLine` | beda estimasi hanya untuk vendor `fix`; fix tanpa deal; melepas hanya bila bertaut |
| `vendorScope`, `countVendors`, `filterVendors` | sesi usang → Umum; `{10, fix 7, negosiasi 2, rencana 1, batal 0, resepsi 10}`; saringan status/kategori/sesi |
| `categoryPayload` | kunci registry → `category_key`; kategori kustom (juga yang sudah punya kunci) → `category_label` dipangkas; label kosong → `null` |
| `toggleSkipped` | ganti seluruh daftar, tanpa ganda, batas 50 → `null` |
| fixture belanja | KPI `4.578.500 / 2.063.800 / 16 / 21 / 0 / 37`; `sumItems` sama; enam kategori `[4, 14, 8, 4, 5, 2]`; subtotal Perangkat Alat Solat 597.300; realisasi hanya barang selesai dari `actual_total` |
| belanja | siklus status; `itemNumbers` 1…37 walau `sort_order` berlubang; `categoryOptions` registry ∪ dipakai |
| `shoppingTemplateImpact` | Proyek Contoh `{37, 37, konfirmasi}`; template segar tanpa konfirmasi; barang kustom tidak ikut; souvenir memicu konfirmasi |
| gulungan | `rollupState` `off_with_payments` / `off` / `on`; teks banner tiga keadaan; `souvenirLeft` |
| pesan | semua kode baru bertext; `not_found` per `where`; `rate_limited` hanya ekspor; 13 alasan Menu 04 punya teks, 4 alasan Menu 03/01 diserahkan (`null`); `portfolio_url` vs `receipt_url`; peringatan dengan `{count}`; `sharedItemText` |
| `vendorApi` | 18 panggilan ke-16 fungsi → metode + path + body + query persis (tabel 4.4); id bukan UUID (`bukan-uuid`, `../../auth`, `x`) melempar; `exportContactsHref`/`exportShoppingHref` hanya `xlsx` |

### 11.2 Tes sumber (`tests/planner-wiring.test.mjs`, perluasan dari 01 FE bab 13.3)

- Halaman `vendor/index.astro` dan `live.astro` memuat `export const prerender = false`; `live.astro` memuat `export const partial = true`; keduanya memakai komponen yang sama (tidak ada markup tab di halaman).
- `index.astro` memakai `PlannerLayout`, bukan `AppLayout`.
- Tidak ada literal sesi atau kategori tertulis di `components/planner/vendor` dan halamannya: `'lamaran'`, `'akad'`, `'resepsi'`, `'venue'`, `'catering'`, `'seserahan'` (hanya `types.ts` dan tes yang boleh). `'all'` dan `SCOPE_ANY` boleh.
- Tidak ada `innerHTML =`, `insertAdjacentHTML`, atau `set:html` yang memuat data API di komponen dan `vendorClient.ts`.
- Tidak ada `plannerFetch('projects/` atau `fetch('/dashboard/api/planner` di komponen: semua panggilan lewat `vendorApi`.
- Tidak ada `parseFloat`, `Number(` atas isian uang atau jumlah, atau `toFixed` di komponen dan `vendorClient.ts` (uang hanya lewat `parseRupiah`/`format`, jumlah lewat `parseQuantity`/`lineTotal`).
- Tidak ada pembagian persen tertulis di komponen (`/ ... * 100`): semua lewat `percent`.
- Tidak ada `href=` yang memuat string pengguna selain lewat `safeHref`/`waLink`.
- `PlannerLayout.astro` menandai `vendor` sebagai `ready`; `ROUTES.md` memuat rute halaman dan partial.

### 11.3 Uji komponen dengan fixture (manual atau skrip)

Render tiap tab dengan: fixture penuh; proyek kosong; variasi bab 4.3 (batal, anggota tanpa penawaran, kelompok melebihi pagu, pagu manual, `none`, tanpa nomor, "Pos ≠ deal", semua kategori dilewati, kategori kustom, sesi usang, nama 255 karakter, jumlah 300 satuan, souvenir, gulungan aktif, pos belum ada, jenis seminar); proyek `archived` (semua kontrol ubah nonaktif, ekspor aktif); lebar 360 px.

### 11.4 QA manual (terhadap backend asli)

**Belum bisa dijalankan** karena halamannya belum ada (backend menu ini sudah ada sejak 2026-10-23, jadi daftar ini dapat dijalankan penuh begitu frontend dibangun). Daftar yang harus lulus, dipetakan ke alur:

1. Proyek Contoh: KPI 7 dari 10, 58.000.000, 3 vendor / Rp 4,3 Jt, 3 kelompok 0 melebihi; Catering pagu 30.000.000 hemat 5.000.000 dengan status bayar "Rp 15 Jt dari Rp 25 Jt"; Photography pas "DP Rp 300 Rb"; ringkasan kelompok menampilkan status pos bersama.
2. Tambah vendor: nomor `0812…`, `+62…`, `8123…` (galat kolom tanpa permintaan); kategori kustom "Photobooth" (kunci diturunkan server, juga setelah diubah); kategori bernama "Venue" bergabung dengan kategori registry; `fix` tanpa deal ditolak; batas 200.
3. Kunci FIX: katering (tertaut), Venue (panel pos bersama + **Samakan estimasi pos** mengubah pos Menu 03), WO (pos baru), pos berharga satuan (estimasi tidak berubah + peringatan); dua vendor sekategori; klik ganda tidak membuat dua pos.
4. Meninggalkan FIX: konfirmasi; pos tetap ada; kunci ulang menautkan kembali.
5. Terapkan skenario lain di Menu 03, muat ulang Vendor: pagu ikut skenario baru; vendor tertaut menampilkan "Pos ≠ deal".
6. Kelompok: hapus penawaran satu anggota → "Sisa pagu"; naikkan melebihi pagu → merah; vendor batal keluar dari kelompok.
7. Kategori dilewati: progres bisa mencapai 100%; "Diperlukan lagi".
8. Jenis seminar: vendor Venue di sesi nyata mendapat pagu skenario persentase; kunci FIX menautkan pos template (bukan pos kedua).
9. Belanja: 37 barang, KPI; toggle status (Selesai mengisi harga riil); jumlah 300 × 45.000; nilai raksasa ditolak dengan pesan, bukan galat 500; alamat URL menjadi tautan, `javascript:alert(1)` menjadi teks; kategori kustom.
10. Gulungan pada Proyek Contoh: konfirmasi muncul; sesudahnya pos Seserahan 4.578.500 / 2.063.800 dan Keuangan Tab 3 menandai pos terkunci; tambah barang dengan kotak centang → ikut, tanpa → tidak; **Lepas** mengembalikan 3.500.000.
11. Template: `append` dua kali (kedua kali `created 0`); `replace_template` meminta konfirmasi; jenis lain menampilkan pesan tanpa template.
12. Souvenir: stok vs serah terima (setelah Menu 05).
13. Ekspor: dua berkas `.xlsx`; kontak hanya FIX dan tanpa harga; sel berawalan `=` tampil sebagai teks; permintaan ke-11 dalam semenit → pesan batas.
14. Arsipkan: kontrol ubah nonaktif, ekspor aktif; tulis dari tab lain → `project_read_only`.
15. Dua tab: hapus vendor di tab A, ubah di tab B → "Vendor tidak ditemukan. Muat ulang daftar."
16. Sesi habis lalu klik aksi → diarahkan ke login. Mobile 360 px, papan ketik saja, pembaca layar ringkas.

### 11.5 Verifikasi kontrak (cara mengulang)

Dilakukan 2026-10-22 terhadap blok DTO Go di [BE bab 4](BE.md#4-dto-go) dan tipe bab 4.2 (skrip ada di scratchpad sesi, bukan di repo). **Setelah backend menu ini ada**, ganti langkah 1–2 dengan panggilan nyata seperti [02 FE bab 11.5](../02_TIMELINE_DAN_TUGAS/FE.md#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang).

1. **Kompilasi** blok Go dalam modul sementara (stub `Notice` identik dengan yang asli), `gofmt`, `go vet`.
2. **Ambil skema** dengan program `reflect` kecil: untuk tiap struct DTO, jalur JSON bersarang (`limit_groups[].line.paid`), `omitempty` (opsional), pointer tanpa `omitempty` (nullable), dan jenis nilai.
3. **Baca tipe TS** dengan compiler API TypeScript (`ts.createProgram` atas `types.ts` yang dirakit dari 01–04 FE; `checker.getPropertiesOfType`) menjadi jalur yang sama dan bandingkan: kunci di satu sisi saja, opsionalitas, `null`, jenis nilai. Tipe permintaan: kunci dan jenis saja.
4. **Uji diri pemeriksa**: ubah tipe TS dengan sengaja (opsional → wajib pada dua kunci, kunci diganti nama, jenis diganti, `| null` ditambahkan, kunci berlebih) dan pastikan dilaporkan. Enam mutasi dicoba; semuanya dilaporkan.
5. **Cocokkan fixture**: `planner-vendors.json` harus sama dengan keluaran `BuildOverview` atas Proyek Contoh; `planner-shopping.json` dengan kueri BE §2.5 dan 2.4; vektor `NormalizeWA` dan `LineTotal` diuji di Go dan TS dengan berkas yang sama.

Hasil 2026-10-22: 26 pasangan tipe, 378 jalur, **0 selisih**. Hasil 2026-10-23 (backend ada): respons asli `GET /vendors` dan `GET /shopping-items` pada Proyek Contoh sama dengan fixture (struktur, tipe, nilai; nol selisih setelah id dinormalkan); struct Go di kode identik dengan blok BE bab 4; `payment` pada pos yang dibuat `lock-deal` terbukti ada (tes `TestLockDealMakesANewLine`). Cuplikan respons tulis di bab 4.5 dan katalog galat bab 4.6 kemudian dijalankan terhadap backend asli (2026-10-23): `lock-deal` kategori berbagi pos, `lock-deal` pos baru, meninggalkan `fix`, `link-to-budget` tanpa dan dengan `confirm`, `replace_template`, header dan `429` ekspor, serta 41 kasus galat; dua teks contoh di bab 4.6 (`invalid_template`, `rollup_conflict`) dan satu baris bab 4.4 berbeda dari server dan sudah dikoreksi. Pustaka `vendor.ts` juga dijalankan terhadap **respons asli kesembilan jenis acara** (satu vendor per kategori, setengahnya dikunci, satu kategori dilewati, tiga barang termasuk satu souvenir dan satu kategori kustom): `tableRows` menampilkan tiap vendor tepat sekali dan satu blok per kelompok pagu, `countVendors` dan `sumItems` sama dengan KPI server, `categories_total` tidak menghitung kategori yang dilewati, `normalizeWA(phone_number)` sama dengan `formatted_wa_number` untuk setiap vendor, stok souvenir 100, dan `default_target` terisi: 548 pemeriksaan, semua lulus (skrip ada di scratchpad sesi, bukan di repo). Yang **tidak** terbukti: galat `5xx`, `409 conflict` (tabrakan transaksi), dan tampilan `.xlsx` di Excel/Numbers.

---

## 12. Urutan pengerjaan dan definisi selesai

### 12.1 Menjalankan lingkungan lokal

Backend menu ini sudah ada, halaman Vendor belum, jadi yang dapat dijalankan sekarang adalah pustaka acuan dan tesnya (backend: `make run`, Proyek Contoh dari `POST /v1/planner/projects/sample` sudah berisi 10 vendor dan 37 barang):

```bash
cd frontend && npm install
npm test            # setelah tes dan fixture bab 4.3 / 11.1 disalin: node --experimental-strip-types --test tests/*.test.mjs
npx tsc --noEmit    # types.ts + vendor.ts + vendorApi.ts + messages.ts + vendorProps.ts di bawah konfigurasi strict repo
```

Setelah halaman dibangun: jalankan seperti [02 FE bab 12.1](../02_TIMELINE_DAN_TUGAS/FE.md#121-menjalankan-lingkungan-lokal) (`make run` untuk backend, `API_BASE_URL=http://localhost:8080 npm run dev` untuk frontend). Proyek Contoh dari `/dashboard/planner` ("Lihat Proyek Contoh") akan berisi 10 vendor dan 37 barang setelah overlay `vendors` dan `shopping` terdaftar ([BE bab 6](BE.md#6-sambungan-ke-menu-01)).

### 12.2 Tahapan

| # | Tahap | Isi | Selesai bila |
|---|---|---|---|
| T0 | Prasyarat | Menu 01 FE: M0 (tipe, `format`, `messages`), M1 (BFF + `client`), `PlannerLayout`, `404.astro`; `finance.ts` Menu 03 (`paymentBadge`, `SCOPE_ANY`) | `npm test` 01 dan 03 hijau; `GET /dashboard/api/planner/event-types` membalas |
| T1 | Fondasi menu ini | blok bab 4.2 ke `types.ts` (+ dua perubahan satu baris, bab 1.3), `vendor.ts` (bab 5), `vendorApi.ts` (bab 6.1), blok `messages.ts` (bab 9.4), `vendorProps.ts` (1.4), `tests/fixtures/*` (4.3), tes 11.1 | `npm test` dan `tsc` hijau; belum perlu backend. Kode di dokumen ini tinggal disalin: sudah lulus kedua pemeriksaan itu |
| T2 | Tab 1 (baca) | `index.astro` + `live.astro`, `VendorTabs`, `VendorKpiCards`, `CategoryChecklist`, `VendorFilters`, `VendorTable` + `LimitGroupSummary`, tombol WA (alur A, I, J) | uji komponen 11.3 (Tab 1); QA 1, 14–16 |
| T3 | Tab 1 (tulis) | `VendorModal`, menu status, `LockDealModal`, hapus, kategori dilewati, `vendorClient.ts` (alur B–H) | QA 2–8 |
| T4 | Tab 2 (baca) | `ShoppingKpiCards`, `ShoppingFilters`, `ShoppingTable`, `RollupBanner` tampilan (alur K, S) | uji komponen 11.3 (Tab 2) |
| T5 | Tab 2 (tulis) | `ShoppingItemModal`, toggle status, hapus, `ShoppingTemplateMenu`, gulungan (alur L–R) | QA 9–13 |
| T6 | Kualitas | aksesibilitas, responsif, `astro check`, `npm run build`, `format:check`, tes sumber 11.2 | build bersih; QA 16 |

T2–T5 membutuhkan backend menu ini ([BE bab 8](BE.md#8-urutan-pengerjaan-dan-definisi-selesai)); sampai ada, kerjakan dengan fixture. Karena Menu 04 membaca `VendorSummary` yang sudah ada di dashboard, widget Vendor (01 FE bab 9.4) tidak ikut dikerjakan di sini.

**Definisi selesai (seluruh Menu 04 FE):** semua tahap lulus; `npm test`, `npm run check`, `npm run build`, `npm run format:check` bersih; tes sumber 11.2 lulus; verifikasi kontrak 11.5 diulang terhadap backend asli (nol selisih); `vendor.ready = true` di `PlannerLayout`; dropdown vendor di modal tugas (02) dan pos (03) memakai `listVendors`; `ROUTES.md`, `docs/frontend/planner/` dan dokumen ini diperbarui bila perilaku berubah.

---

## 13. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–03 (kode dan dokumen), kontrak, PDF, dan SQL yang dijalankan. Dokumen mengikuti pilihan yang tertulis; keputusan sisi backend ada di [BE bab 9](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk).

1. **Kontrak dibuktikan terhadap HTTP asli hanya untuk dua pembacaan** (`GET /vendors`, `GET /shopping-items`; fixture sama dengan responsnya). Tipe cocok dengan blok DTO Go yang dikompilasi (bab 4.2) dan dengan struct di kode, tetapi cuplikan respons tulis bab 4.5 belum disalin dari respons asli; ulangi bab 11.5 saat membangun halaman.
2. **Status bayar kelompok pagu ditampilkan di baris ringkasan, bukan di tiap vendor.** Vendor pada pos bersama tidak bertaut satu per satu ([BE bab 2.2](BE.md#22-kunci-kerja-sama-lock-deal-tanpa-pos-ganda)), jadi kolom Status bayarnya "–". Alternatif: menampilkan status pos bersama di setiap anggota.
3. **"Samakan estimasi pos dengan total deal" memakai `PUT` pos Menu 03**, yang mengganti penuh dan melepas `vendor_id` yang tak dikirim: FE membaca pos lewat `listLines` dan mengirim ulang seluruh isiannya. Alternatif: `PATCH` estimasi atau endpoint khusus di Menu 03 ([03 BE bab 9 butir 11](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
4. **Format uang singkat** `formatRupiahShort` menghasilkan "Rp 900 Rb" dan "Rp 15 Jt" (bukan "0,9 jt" seperti wireframe lama).
5. **Kategori belanja tanpa kunci**: dropdown = registry ∪ yang sudah dipakai; dua ejaan berbeda menjadi dua kategori ([BE bab 9 butir 15](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
6. **Templat pesan WhatsApp** berbahasa Indonesia tetap ("kerja sama untuk acara …"); tidak ada penyesuaian per jenis acara.
7. **Chip sesi** menampilkan semua sesi non-sistem walau berangka 0 (redup), supaya vendor/barang yang dipindah ke sesi baru mudah ditemukan; "Umum" hanya bila ada isinya. Alternatif: menyembunyikan sesi kosong.
8. **Ekspor sebagai tautan** (teks 429 kurang ramah, tanpa `Retry-After`) atau `fetch` + Blob; pilih saat membangun. Hanya `.xlsx`; PDF menunggu pustaka ([BE bab 9 butir 9](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
9. **Satu sumber render dengan partial** (bab 2.4) belum dijalankan di browser; alternatif `location.reload()`.
10. **Tidak ada polling dan tidak ada kontrol konkurensi pada kolom yang sama** (terakhir menang); galat `not_found`/`invalid_scope`/`conflict` memicu muat ulang area. `PATCH` proyek untuk kategori dilewati mengganti seluruh daftar, jadi dua tab yang mengubah kategori berbeda dapat saling menimpa.
11. **Stok souvenir dibandingkan serah terima hanya bila Menu 05 ada**; sampai itu banner hanya menampilkan stok.
12. **Kinerja browser belum diukur.** Server: 200 vendor dan 1.000 barang adalah batas tampilan, bukan batas kinerja basis data. Tabel dirender sekaligus tanpa paginasi atau virtualisasi.
13. **Hari ini pada tanggal bawaan** tidak dipakai di menu ini (tidak ada tanggal); tidak ada masalah tengah malam.
14. **Nomor luar negeri ditolak** (`normalizeWA` hanya menerima nomor Indonesia; [BE bab 9 butir 8](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
15. **Awalan `'` pada sel Excel** tampak sebagai karakter ([02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)); ekspor belum dibuka di aplikasi spreadsheet.
16. **Label status**: "Rencana" menggantikan "Riset" (PDF: *List Rencana Vendor*); chip pipeline memisahkan Rencana dan Negosiasi.

---

## 14. Riwayat perubahan dokumen ini

**2026-10-22 (disamakan dengan pola 01–03 FE, dicocokkan dengan DTO Go yang dikompilasi dan data Proyek Contoh dari backend asli):** dokumen diperluas dari ±120 baris menjadi acuan lengkap: daftar isi, route dan file, kontrak props, alur data dan galat, prasyarat, tipe TypeScript yang dicocokkan dengan DTO Go (26 pasangan, 378 jalur, nol selisih), fixture, katalog endpoint dan galat, `vendor.ts`, `vendorApi.ts`, `messages.ts` dengan kode yang sudah dijalankan, alur interaksi, pesan galat, aksesibilitas, pengujian, urutan pengerjaan. **Koreksi terhadap versi sebelumnya:** (1) layout `AppLayout`/`nav_active` diganti `PlannerLayout`; (2) wireframe: pagu Catering 25.000.000 diganti 30.000.000 dengan selisih hemat 5.000.000 (skenario aktif 75 Jt); status bayar Venue "DP 0,9 jt" dihapus (Venue berbagi pos dengan Dekorasi dan tidak bertaut sendiri; status pos bersama pindah ke baris ringkasan kelompok); chip "Riset (3)" yang menjumlah Rencana dan Negosiasi dipisah; kolom Alamat contoh "🔗 …" menjadi "–" (tautan PDF tidak disalin); (3) pagu tidak lagi diisi otomatis ke kolom formulir tetapi dibaca dari skenario; (4) kategori kustom dikirim sebagai label, bukan kunci; (5) "URL/IG" pada portofolio diganti URL lengkap (hanya `http`/`https` diterima); (6) `export` PDF dibuang (pustaka belum disetujui) dan kontak diekspor tanpa harga; (7) kolom Alamat memakai `safeHref` (bukan "diawali http://" ad hoc) dan jumlah × harga dihitung tanpa float; (8) tombol "Hitung pos dari daftar" kini membaca `default_target` (pos bawaan jenis acara, pembayaran tersimpan) dan barang baru ikut lewat kotak centang; (9) komponen yang tersirat tetapi tidak terdaftar (`CategoryChecklist` aksi lewati, `LimitGroupSummary`, `ShoppingTemplateMenu`, `ConfirmDialog`) didaftarkan; (10) `PATCH …/status` vendor ditambahkan agar pipeline tidak membutuhkan `PUT` penuh.

**2026-10-23 (dicocokkan dengan backend yang sudah diimplementasikan):** tipe bab 4.2 dibandingkan ulang dengan struct Go **di kode** (26 pasangan, 378 jalur, nol selisih); kelima blok kode TypeScript bab 1.4, 4.2, 5, 6, dan 9.4 sama persis dengan yang dikompilasi (`tsc` strict bersih, 19 tes lulus). Contoh bab 4.5 dan katalog bab 4.6 dijalankan terhadap respons asli: teks contoh `invalid_template` dan `rollup_conflict` serta baris `unlinkFromBudget` dikoreksi, `listVendors`/`linkToBudget` mendapat galat yang sebenarnya, dan header ekspor serta `429` kini teramati. Pustaka `vendor.ts` dijalankan terhadap respons asli kesembilan jenis acara (548 pemeriksaan, bab 11.5), jadi klaim universal tidak lagi hanya bersandar pada fixture pernikahan. Pernyataan "backend belum ada" di bab 11.4 diganti: yang belum ada adalah halamannya.
