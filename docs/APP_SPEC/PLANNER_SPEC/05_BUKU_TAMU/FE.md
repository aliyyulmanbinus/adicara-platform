# Menu 05: Buku Tamu & Undangan Digital — Frontend Specification

Acuan pembangunan antarmuka dua tab: **Daftar Tamu & Undangan** dan **Meja Resepsionis & Check-in Hari-H** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: kontrak data, aturan tampilan, algoritma, alur interaksi, kasus galat, dan pengujian ada di sini. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); fondasi yang dibangun Menu 01 (tipe, `format.ts`, `messages.ts`, `client.ts`, BFF, `PlannerLayout`) dan `vendor.ts` Menu 04 (`normalizeWA`, `waLink`) **dipakai apa adanya** dan dirujuk ke [01 FE](../01_DASHBOARD/FE.md) dan [04 FE](../04_VENDOR_DAN_BELANJA/FE.md), bukan diulang. Pola dokumen ini mengikuti [04 FE](../04_VENDOR_DAN_BELANJA/FE.md), [03 FE](../03_KEUANGAN/FE.md), dan [02 FE](../02_TIMELINE_DAN_TUGAS/FE.md).

> **Status (2026-10-25):** frontend Planner belum ada satu baris pun, tetapi **backend menu ini sudah diimplementasikan** ([BE.md](BE.md), [dokumen modul](../../../backend/planner/2026-10-25-planner-menu-05.md)) dan kontrak di bab 4 **dicocokkan dengan HTTP yang berjalan**: respons asli `GET guests`, `catering-projection`, `bulk-import?dry_run=true` pada Proyek Contoh, dan paket `reception-overview` untuk ilustrasi hari-H **sama dengan keempat fixture JSON** (struktur, tipe, dan nilai; id, tiket, dan waktu dinormalkan; `TestResponsesMatchTheFEFixtures`), dan **setiap baris katalog galat bab 4.6** dihasilkan backend dengan status, kode, dan pesan yang tertulis (`TestErrorCatalogOfTheFEDocIsWhatTheBackendSays`). Tipe TypeScript bab 4.2 dicocokkan dengan **DTO Go di [BE bab 4](BE.md#4-dto-go)** (skema JSON diambil dengan `reflect` dan dibandingkan kunci demi kunci dengan tipe TS): **21 pasangan tipe, 125 jalur JSON, nol selisih** pada kunci, opsionalitas, `null`, dan jenis nilai (pembanding diuji dengan enam mutasi tipe dan melaporkan semuanya); DTO di kode sama dengan blok itu (alias untuk tipe Menu 01). Menu ini bergantung pada Menu 01 FE (M0–M1 dan `PlannerLayout`), `vendor.ts` Menu 04, dan satu perubahan BFF ([bab 1.3](#13-perubahan-pada-file-yang-sudah-ada); BFF belum ada di repo, jadi perubahannya ikut saat BFF dibuat).
>
> Kode di bab 1.4, 4.2, 5, 6, dan 9.4 **sudah dijalankan**: `tsc` strict (dengan `noUncheckedIndexedAccess`, konfigurasi Astro yang dipakai repo) bersih bersama kode 01–04 yang diekstrak dari dokumennya; 28 tes `node:test` ([bab 11](#11-pengujian)) lulus terhadap fixture di atas, termasuk 26 vektor `normalizeWA` yang sama dengan backend; **24 dari 24 mutasi kode** yang dicoba menjatuhkan tes; semua 16 pembungkus `guestApi` diuji metode, path, query, dan body-nya. Yang bergantung pada Astro atau browser (halaman, partial, dialog, kamera, unduhan) belum diuji dan ditandai *(belum diuji di browser)*.

> Angka pada wireframe berasal dari **Proyek Contoh** (20 tamu fiktif, [BE bab 6](BE.md#6-data-contoh-proyek-contoh)) yang dibuat 4 Okt 2026: Sinta & Arif, H-175, Hari-H Minggu 28 Maret 2027, mode Standalone. Wireframe Tab 2 adalah **ilustrasi hari-H** (Proyek Contoh belum berada di hari-H; 11 tamu sudah masuk, stok souvenir contoh 40). Wireframe adalah tata letak, bukan desain visual akhir: warna, jarak, dan tipografi memakai token yang sudah ada (bab 10.1). Emoji pada wireframe untuk **sesi** adalah emoji dari API; ikon **kontrol** memakai Phosphor dan tidak digambar. Versi sebelumnya menulis tombol *Kirim WA bergilir* dan *Aktifkan Undangan Digital Adicara* yang tidak punya spesifikasi atau modul; keduanya dibuang.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat](#2-alur-data-dan-penanganan-galat)
3. [Prasyarat yang harus sudah ada dari Menu 01 dan 04 FE](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-dan-04-fe)
4. [Kontrak data: tipe, fixture, contoh respons, katalog galat](#4-kontrak-data-tipe-fixture-contoh-respons-katalog-galat)
5. [Library `lib/planner/guest.ts`](#5-library-libplannerguestts)
6. [Pembungkus panggilan, BFF, dan unduhan](#6-pembungkus-panggilan-bff-dan-unduhan)
7. [Tab 1: Daftar Tamu & Undangan](#7-tab-1-daftar-tamu--undangan)
8. [Tab 2: Meja Resepsionis & Check-in Hari-H](#8-tab-2-meja-resepsionis--check-in-hari-h)
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
| Buku Tamu | `/dashboard/planner/[id]/tamu` | `src/pages/dashboard/planner/[id]/tamu/index.astro` | dua tab lewat `?tab=tamu` (default) dan `?tab=checkin`; nilai lain jatuh ke `tamu`. Tab dipilih server-side; bagian tab lain tidak dimuat. Filter, pencarian, dan halaman ada di URL (bab 2.3) |
| Partial | `/dashboard/planner/[id]/tamu/live` | `src/pages/dashboard/planner/[id]/tamu/live.astro` (`export const partial = true`) | HTML area yang diperbarui setelah aksi tulis dan oleh polling (bab 2.4); menerima parameter yang sama dengan halaman; `?part=summary` hanya kartu dan penghitung (Tab 1) |
| Komponen | – | `src/components/planner/guest/*.astro` | lihat 1.2 |
| Library | – | `src/lib/planner/guest.ts` (murni, bab 5), `guestApi.ts` (pembungkus panggilan, bab 6), `guestProps.ts` (kontrak props, bab 1.4), `guestClient.ts` (skrip klien halaman, bab 7.5 dan 8.5) | |
| Tes + fixture | – | `tests/planner-guest.test.mjs`, `tests/fixtures/planner-{guests,reception,catering-projection,import-dryrun,wa-vectors}.json` (`planner-wa-vectors.json` dari [Menu 04](../04_VENDOR_DAN_BELANJA/fixtures/planner-wa-vectors.json)) | bab 4.3 dan 11 |

Tidak ada file BFF baru: semua panggilan lewat `/dashboard/api/planner/[...path].ts` dari Menu 01 (bab 6), dengan **satu perubahan** (bab 1.3). Halaman memakai `PlannerLayout` (bilah sekunder berisi Project Switcher dan sub-navigasi), **bukan** `AppLayout` langsung dan tanpa `nav_active` (versi awal dokumen ini menulis `AppLayout.astro` dan `nav_active = 'planner'`, yang tidak ada di desain 01 FE bab 10). Judul menu dan kata benda "Tamu" berasal dari `labels` registry ([TEMPLATE_EVENT 3.1](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#31-sebutan-antarmuka-labels)); seminar menjadi "Peserta & RSVP", dan kata teknis "undangan" menjadi "pendaftaran" (`invitationNoun`, bab 13 butir 3).

```html
<nav class="guest-tabs" role="tablist">
  <a href="?tab=tamu"    class="tab-item active" role="tab" aria-selected="true"><i class="ph ph-users-three"></i> Daftar {Tamu} & {Undangan}</a>
  <a href="?tab=checkin" class="tab-item"        role="tab" aria-selected="false"><i class="ph ph-qr-code"></i> Meja Resepsionis & Check-in</a>
</nav>
```

### 1.2 Komponen

Props lengkap (TypeScript) di bab 1.4. Kolom "Klien" menyebut perilaku yang dipasang skrip halaman; alurnya di bab 7.5 dan 8.5.

| Komponen | Props utama | Dipakai di | Klien |
|---|---|---|---|
| `GuestTabs` | `active`, `project`, `baseHref` | tab bar | navigasi penuh (`?tab=`) |
| `GuestKpiCards` | `overview`, `project` | Tab 1 | diganti oleh polling ringkasan (30 dtk) |
| `SideSwitcher` | `label`, `pills`, `selected`, `hrefs` | Tab 1 | tautan biasa; tidak dirender bila `pills` kosong (jenis tanpa sisi) |
| `CategoryCounters` | `label`, `pills`, `selected`, `hrefs` | Tab 1 | tautan biasa; hitungan mengikuti pihak terpilih |
| `GuestFilters` | `query`, `sessions`, `baseHref` | Tab 1 | `<form method="get">` untuk sesi, RSVP, status kirim, dan pencarian |
| `GuestToolbar` | `project`, `readOnly`, `importHref`, `exportHref`, `templateHref` | Tab 1 | Tambah Tamu, Impor, Ekspor |
| `StandaloneBanner` | `project` | Tab 1 | – (teks saja; tanpa tombol, bab 13 butir 20) |
| `GuestTable` | `guests`, `offset`, `sessions`, `sides`, `categories`, `project`, `readOnly` | Tab 1 | WhatsApp (`guestWaLink`), salin tautan, tandai terkirim, ubah RSVP, ubah, hapus; pada layar sempit menjadi kartu |
| `GuestPager` | `total`, `limit`, `offset`, `query`, `baseHref` | Tab 1 | tautan halaman |
| `GuestModal` | `sessions`, `sides`, `categories`, `defaultScope`, `project` | Tab 1 | tambah / ubah tamu (`validateGuestForm`, `guestBody`) |
| `GuestImportModal` | `project`, `sessions`, `templateHref` | Tab 1 | unduh templat, pilih berkas, pratinjau `dry_run`, konfirmasi |
| `ReceptionKpiCards` | `kpi`, `project` | Tab 2 | diganti oleh polling (5 dtk) |
| `ReceptionToolbar` | `query`, `filters`, `baseHref`, `readOnly`, `attendanceHref` | Tab 2 | Scan QR, Walk-in, pencarian, chip status, dropdown kategori, ekspor rekap |
| `ReceptionTable` | `guests`, `sessions`, `sides`, `project`, `readOnly` | Tab 2 | ketuk baris → `QuickCheckinSheet`; Batal; tombol souvenir |
| `RecentCheckins` | `guests`, `timezone` | Tab 2 | – |
| `QrScanModal` | `project` | Tab 2 | kamera + `BarcodeDetector`, atau input manual |
| `QuickCheckinSheet` | `project`, `sides`, `sessions` | Tab 2 | lembar check-in satu ketukan; menangani `409 already_checked_in` |
| `WalkInModal` | `sides`, `categories`, `project` | Tab 2 | `walkIn` |
| `ConfirmDialog` | `title`, `message`, `confirmLabel`, `danger` | semua | satu `<dialog>` dipakai bersama (dari 02–04 FE; bila belum ada, buat di sini) |

### 1.3 Perubahan pada file yang sudah ada

| File | Perubahan | Mengapa |
|---|---|---|
| `src/layouts/PlannerLayout.astro` | `MENUS`: `tamu.ready = true` | satu bendera; menu lain tidak berubah ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) |
| `src/lib/planner/types.ts` | tambah blok bab 4.2; ubah `export interface EventTypeInfo {` menjadi `… extends EventTypeGuestInfo {` (bila Menu 02–04 FE sudah ada: `extends EventTypeAdminInfo, EventTypeFinanceInfo, EventTypeVendorInfo, EventTypeGuestInfo`) | kontrak menu ini |
| `src/lib/planner/messages.ts` | tambah blok bab 9.4 (teks, alasan per kolom, galat berkonteks) | teks galat |
| BFF `src/pages/dashboard/api/planner/[...path].ts` | **batas badan 7,5 MiB hanya untuk jalur `…/guests/bulk-import`** (jalur lain tetap 64 KiB; [01 FE bab 6](../01_DASHBOARD/FE.md#6-bff-dashboardapiplanner) aturan 2 dan batas badan) | impor mengirim berkas sebagai JSON base64 (≈ 6,7 MiB untuk berkas 5 MB; [BE bab 2.6](BE.md#26-impor-dan-ekspor)). Aturan `application/json` untuk non-GET **tidak berubah**: tidak ada `multipart`, jadi tidak menyentuh `checkOrigin` |
| `src/lib/api/backend.ts` dan BFF | `backendFetchRaw` dan aliran berkas untuk `…/guests/export`, `…/guests/export-attendance`, dan `…/guests/import-template` | sama dengan perubahan 02–04 FE bab 1.3 (kerjakan sekali) |
| Modal pos Menu 03 (`BudgetItemModal`) | tombol "Gunakan proyeksi RSVP" memakai `guestApi(...).cateringProjection()`, `canUseProjection`, `projectionText`; `rsvpEnabled = PlannerLayout.MENUS.tamu.ready` | [03 FE bab 9.3](../03_KEUANGAN/FE.md#93-komponen-dan-modal); modal mengirim `PUT` pos dengan **semua** isian termasuk `vendor_id` (bab 7.5 alur L) |
| Banner souvenir Menu 04 | membaca `kpi.souvenir_distributed` dan `kpi.souvenir_stock` dari `reception-overview?limit=1` | [04 FE bab 2.1](../04_VENDOR_DAN_BELANJA/FE.md#21-pemuatan-per-tab) |
| `ROUTES.md` | baris halaman `/dashboard/planner/[id]/tamu` dan partial `…/tamu/live` | file itu minta diperbarui bersama setiap halaman baru |
| `docs/frontend/planner/` | dokumen bertanggal dan entri `CHANGELOG.md` | aturan repo: kode dan dokumen satu paket |
| (tidak diubah) | `AppLayout.astro`, `middleware.ts`, `tokens.css`, `global.css` | |

`astro check` (bagian dari `npm run build`) harus lolos; TypeScript `strict` dan `noUncheckedIndexedAccess` aktif (akses `array[i]` bertipe `T | undefined`).

### 1.4 Kontrak props

Disalin ke `interface Props` tiap komponen (atau diimpor dari `guestProps.ts`); **sudah dikompilasi** `tsc` strict bersama bab 4.2 dan 5. Props adalah objek biasa yang dapat diserialisasi SSR (tanpa fungsi atau `Map`): tautan filter dibangun di frontmatter halaman (`guestHref`/`receptionHref`) dan dikirim sebagai `hrefs`.

```ts
// src/lib/planner/guestProps.ts
import type { Pill, GuestQuery, ReceptionQuery } from './guest.ts';
import type {
  Guest, GuestOption, GuestOverview, GuestTab, Metadata, ReceptionFilters, ReceptionKPI, SessionBrief,
} from './types.ts';

/** Konteks proyek yang dibaca halaman dari GET projects/:id dan registry, lalu diteruskan ke komponen (bukan dibaca ulang). */
export interface GuestProject {
  id: string; title: string; eventDate: string; timezone: string; metadata: Metadata;
  archived: boolean; eventTypeKey: string;
  hasInvitation: boolean;                       // project.invitation_id !== null
  guestWord: string;                            // labels.guest: "Tamu" / "Peserta"
  invitationNoun: 'undangan' | 'pendaftaran';   // invitationNoun(labels)
}

// ----- Tab 1 -----
export interface GuestTabsProps { active: GuestTab; project: GuestProject; baseHref: string }
export interface GuestKpiCardsProps { overview: GuestOverview; project: GuestProject }
/** `href(key)` dibangun server dengan `guestHref(base, query, { side: key })`; komponen tidak merakit URL sendiri. */
export interface PillRowProps { label: string; pills: Pill[]; selected: string; hrefs: Record<string, string> }
export type SideSwitcherProps = PillRowProps;
export type CategoryCountersProps = PillRowProps;
export interface GuestFiltersProps { query: GuestQuery; sessions: SessionBrief[]; baseHref: string }
export interface GuestToolbarProps { project: GuestProject; readOnly: boolean; importHref: string; exportHref: string; templateHref: string }
export interface StandaloneBannerProps { project: GuestProject }
export interface GuestTableProps {
  guests: Guest[]; offset: number; sessions: SessionBrief[]; sides: GuestOption[]; categories: GuestOption[];
  project: GuestProject; readOnly: boolean;
}
export interface GuestPagerProps { total: number; limit: number; offset: number; query: GuestQuery; baseHref: string }
export interface GuestModalProps {
  sessions: SessionBrief[]; sides: GuestOption[]; categories: GuestOption[]; defaultScope: string; project: GuestProject;
}
export interface GuestImportModalProps { project: GuestProject; sessions: SessionBrief[]; templateHref: string }

// ----- Tab 2 -----
export interface ReceptionKpiCardsProps { kpi: ReceptionKPI; project: GuestProject }
export interface ReceptionToolbarProps { query: ReceptionQuery; filters: ReceptionFilters; baseHref: string; readOnly: boolean; attendanceHref: string }
export interface ReceptionTableProps {
  guests: Guest[]; sessions: SessionBrief[]; sides: GuestOption[]; project: GuestProject; readOnly: boolean;
}
export interface RecentCheckinsProps { guests: Guest[]; timezone: string }
export interface QrScanModalProps { project: GuestProject }
export interface QuickCheckinSheetProps { project: GuestProject; sides: GuestOption[]; sessions: SessionBrief[] }
export interface WalkInModalProps { sides: GuestOption[]; categories: GuestOption[]; project: GuestProject }
```

`GuestProject` dibentuk satu kali di frontmatter halaman dari `GET projects/:id` dan registry: `archived = project.status === 'archived'`, `hasInvitation = project.invitation_id !== null`, `guestWord = labels.guest`, `invitationNoun = invitationNoun(labels)`. `readOnly` pada komponen = `project.archived`.

---

## 2. Alur data dan penanganan galat

```text
SSR (baca)    : halaman → backendFetch('/v1/planner/…', { accessToken }) → HTML
Interaksi     : <script> klien → guestApi(plannerFetch, id) → BFF /dashboard/api/planner/… → Go API
Pembaruan     : sesudah aksi tulis / polling → fetch('/dashboard/planner/:id/tamu/live?tab=…') → HTML partial → ganti [data-guest-live]
Unduhan       : <a href="/dashboard/api/planner/projects/:id/guests/export?format=xlsx"> → BFF (aliran berkas)
Registry      : lib/planner/eventTypes.ts → GET /v1/planner/event-types (cache memori proses 1 jam)
```

`backendFetch` mengembalikan seluruh payload di `result.data`, jadi isi respons ada di `result.data.data`.

### 2.1 Pemuatan per tab

| Tab | Panggilan (paralel dengan `Promise.all`) | Catatan |
|---|---|---|
| `tamu` | `GET projects/:id/guests?<guestApiQuery>` + `GET projects/:id/sessions` + `GET projects/:id` + registry | satu panggilan memberi halaman tamu **dan** `overview` (kartu, pil pihak, penghitung kategori; selalu seluruh proyek); `sessions` memberi nama sesi, filter, dan konteks pesan WA; `GET projects/:id` memberi `status`, `invitation_id`, `title`, `event_date`, `timezone`, `metadata`; registry memberi `guest_sides`, `guest_categories`, dan `labels`. Pemfilteran dan pemotongan halaman dilakukan **server** (daftar bisa 5.000 baris), jadi mengganti filter adalah navigasi penuh |
| `checkin` | `GET projects/:id/guests/reception-overview?<receptionApiQuery>` + `GET projects/:id/sessions` + `GET projects/:id` + registry | satu panggilan memberi KPI, jumlah filter, 10 check-in terbaru, dan satu halaman antrean (50); registry memberi sisi dan kategori untuk modal walk-in |

Validasi `Astro.params.id` seperti 01 FE bab 2.1 (UUID huruf kecil, selain itu `Astro.rewrite('/404')` tanpa memanggil backend).

### 2.2 Hasil backend → tindakan halaman

| Hasil | Tindakan |
|---|---|
| `2xx` | render |
| `401` | `Astro.redirect('/auth/masuk')` |
| `404 not_found` pada proyek | `Astro.rewrite('/404')` (tidak membedakan "bukan milik Anda" dan "tidak ada") |
| `5xx` atau `BackendUnavailableError` | kerangka halaman + banner "Buku tamu sedang tidak tersedia" + tombol "Muat ulang"; **bukan 404** |
| `GET guests`/`reception-overview` gagal, proyek berhasil | tab tetap tampil; isi diganti banner galat lokal dengan tombol "Muat ulang"; tab lain tidak ikut gagal |
| `400 invalid_input` dari URL (`limit`, `side`, …) | tidak terjadi lewat halaman: `parseGuestQuery`/`parseReceptionQuery` membuang nilai tidak sah sebelum memanggil backend |
| Registry gagal | pakai cache lama; bila tidak ada, pil pihak dan kategori tetap tampil dari `overview` (yang membawa label), hanya dropdown modal diganti isian teks bebas |

Aturan umum: `export const prerender = false`; teks galat ke pengguna selalu dari `lib/planner/messages.ts`, tidak pernah `error.message` mentah (pesan server berbahasa Inggris). Proyek `archived`: semua kontrol ubah dinonaktifkan dengan keterangan "Acara diarsipkan" (`GET`, ekspor, dan templat tetap berjalan; **check-in juga tidak**: meja resepsionis menulis); tetap tangani `409 project_read_only` bila status berubah di tab lain. Proyek `completed` tetap bisa diedit.

Kerangka frontmatter `tamu/index.astro` *(sketsa, belum dijalankan)*:

```ts
export const prerender = false;
const id = Astro.params.id ?? '';
if (!isUuid(id)) return Astro.rewrite('/404');
const sp = Astro.url.searchParams;
const tab = sp.get('tab') === 'checkin' ? 'checkin' : 'tamu';
const accessToken = Astro.locals.accessToken;                     // diisi middleware (token sudah segar)
const base = `/v1/planner/projects/${id}`;
const [detail, registry, sessions] = await Promise.all([
  backendFetch<Envelope<ProjectDetail>>(base, { accessToken }),
  loadRegistry(Astro),
  backendFetch<Envelope<SessionWithUsage[]>>(`${base}/sessions`, { accessToken }),
]);
// … toPageError(detail) menerapkan tabel 2.2; selanjutnya per tab, mis. tamu:
const q = parseGuestQuery(sp);
const list = await backendFetch<Envelope<GuestList>>(`${base}/guests?${new URLSearchParams(guestApiQuery(q))}`, { accessToken });
const pills = { sides: sidePills(list.data.data.overview), categories: categoryPills(list.data.data.overview, q.side) };
```

### 2.3 State di URL dan di peramban

| Penyimpan | Kunci | Nilai | Ditulis oleh | Dibaca oleh |
|---|---|---|---|---|
| URL | `tab` | `tamu` (default) · `checkin` | tautan tab | SSR: bagian yang dimuat |
| URL (Tab 1) | `side` | kunci sisi registry atau `none`; tak ada = Semua | tautan pil | `parseGuestQuery` → `GET guests?side=` |
| URL (Tab 1) | `category` | kunci registry atau teks kategori kustom (≤ 60) | tautan pil | idem |
| URL (Tab 1) | `rsvp`, `sent`, `scope` | `attending\|maybe\|declined\|unconfirmed` · `sent\|unsent` · `scope_key` sesi (`all` = Umum) | `<form method="get">` | idem |
| URL (Tab 1) | `q` | pencarian (≤ 100) | `<form method="get">` (Enter) | idem (`search=`) |
| URL (Tab 1) | `page` | 1, 2, … (100 tamu per halaman) | tautan halaman | idem (`offset = (page-1)×100`) |
| URL (Tab 2) | `status`, `category`, `q`, `page` | `in\|pending` (tak ada = semua) · kunci kategori · pencarian · halaman (50) | tautan chip, form, tautan halaman | `parseReceptionQuery` → `GET guests/reception-overview` |

Nilai yang tidak sah dibuang secara diam-diam (`parseGuestQuery`/`parseReceptionQuery`): URL buatan tangan tidak merusak halaman. Mengubah filter mengembalikan ke halaman 1 (`guestHref`/`receptionHref`). Tidak ada `localStorage` atau `sessionStorage`; tidak ada data tamu di penyimpan peramban.

### 2.4 Satu sumber render dan pembaruan sesudah aksi

Mengikuti pola partial [04 FE bab 2.4](../04_VENDOR_DAN_BELANJA/FE.md#24-satu-sumber-render-dan-pembaruan-sesudah-aksi), [03 FE bab 2.4](../03_KEUANGAN/FE.md#24-satu-sumber-render-dan-pembaruan-sesudah-aksi), dan [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial): **tidak ada logika render ganda di klien**.

1. SSR merender area `[data-guest-live]` dari komponen yang sama dengan `live.astro`. Filter dan halaman adalah navigasi penuh (server memotong halaman); tidak ada penyaringan di klien.
2. **Aksi tulis berhasil** → klien memanggil `GET /dashboard/planner/{id}/tamu/live?tab=…&…` (`Accept: text/html`, parameter URL halaman yang sedang tampil), mengganti isi `[data-guest-live]`, dan memulihkan fokus lewat `data-focus-key`. Tidak ada pembaruan optimistik: penghitung, pil, dan KPI dihitung server.
3. **Polling (pengganti SSE)**: Tab 1 memanggil `live?part=summary` tiap **30 detik** (hanya kartu KPI dan pil/penghitung; **tabel tidak diganti** supaya baris tidak melompat, diganti hanya saat pengguna menekan "Muat ulang" atau setelah aksi tulis). Tab 2 memanggil `live?tab=checkin&…` tiap **5 detik** dan mengganti KPI, chip, `RecentCheckins`, dan antrean. Aturan polling ([kontrak bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens)): berhenti saat `document.visibilityState !== 'visible'`, mundur ke dua kali interval setelah galat, jitter ±10%, **ditunda selagi dialog/lembar terbuka atau kolom pencarian sedang diketik**, dan tidak mengganti elemen yang sedang fokus (pulihkan lewat `data-focus-key`).
4. Partial gagal (`5xx`, jaringan): jangan kosongkan area; tampilkan banner kecil "Data belum diperbarui" + tombol "Muat ulang" (`location.reload()`), `aria-live="polite"`. `401` atau pengalihan ke login: `location.assign('/auth/masuk')`.
5. Galat yang menandakan data usang (`not_found`, `invalid_scope`, `409 conflict`) selalu ditutup dengan memuat ulang area lewat partial.

*(Belum diuji di browser, sama seperti 01–04. Alternatif yang sah: `location.reload()` sesudah tiap aksi tulis, dan polling yang memanggil `guests/summary` / `reception-overview` lewat BFF JSON lalu memperbarui angka saja.)*

---

## 3. Prasyarat yang harus sudah ada dari Menu 01 dan 04 FE

**Belum ada satu pun di repo** (frontend Planner belum dibangun). Menu ini mengimpor, tidak membangun ulang:

| Kebutuhan | Berasal dari |
|---|---|
| `types.ts` (`Notice`, `SessionBrief`, `SessionWithUsage`, `EventTypeInfo`, `Project`, `ProjectDetail`, `Metadata`, `Envelope`, `ApiError`) | 01 FE bab 4.2 |
| `format.ts` (`formatDate`, `percent`, `formatPercent`, `barWidth`, `charCount`, `parseFieldError`) | 01 FE bab 5.1 |
| `vendor.ts` (`normalizeWA`, `waLink`) dan tipe `PlannerFetch` dari `vendorApi.ts` | 04 FE bab 5 dan 6.1. Dua fungsi itu kecil dan sudah teruji; bila Menu 04 FE belum dibangun, bangun keduanya lebih dulu |
| `messages.ts` (`plannerErrorMessage`, `fieldErrorMessage`, dan bila ada `vendorErrorMessage`/`financeErrorMessage`) | 01 FE bab 5.4 dan 11, 03–04 FE bab 9–10 |
| `client.ts` (`plannerFetch`: tangani sesi habis, `204`, backend mati; metode `PUT`/`PATCH`) dan `server.ts` (`loadRegistry`, `isUuid`) | 01 FE bab 5.5–5.6 |
| BFF `[...path].ts` dan `backend.ts` (`backendFetchRaw`) | 01 FE bab 6 dan 1.3 |
| `PlannerLayout.astro` (Project Switcher, sub-navigasi `MENUS`) dan `404.astro` | 01 FE bab 10 dan 1.1 |
| Pola `<dialog>`, `textContent` untuk string API | 01 FE bab 12 |
| Backend menu ini | [BE.md](BE.md) (sudah diimplementasikan: modul `plannerguest`, 19 endpoint; kontrak dan fixture sama dengan respons asli) |

Karena itu urutan kerja ([bab 12](#12-urutan-pengerjaan-dan-definisi-selesai)) dimulai dari M0, M1, dan `PlannerLayout` di Menu 01 FE.

---

## 4. Kontrak data: tipe, fixture, contoh respons, katalog galat

### 4.1 Aturan JSON yang dikirim backend

Selain aturan umum [01 FE bab 4.1](../01_DASHBOARD/FE.md#41-aturan-json-yang-dikirim-backend) (amplop `{ "data": … }`, tanggal `"YYYY-MM-DD"` tanpa zona, uang bilangan bulat). Butir di bawah berasal dari BE.md, SQL yang dijalankan, dan DTO Go, dan **sudah diamati pada HTTP asli** untuk respons baca (sama dengan fixture) dan semua galat bab 4.6.

**Bentuk respons**

- **Kunci dihilangkan bila kosong** (`omitempty`): `guest_side`, `table_number`, `phone_number`, `formatted_wa_number`, `personalized_url`, `invitation_sent_at`, `rsvp_submitted_at`, `checked_in_at`, `notes`, `souvenir_stock`, `custom`, `of_row`, `of_guest`. Tipe memakai `?:`; jangan memeriksa `=== null`. Satu-satunya `null`: `sides[].key` (pihak belum ditentukan, milik Menu 01) dan `side_categories[].side`.
- **Daftar selalu array**, tidak pernah `null`, termasuk saat kosong (`guests`, `sides`, `categories`, `side_categories`, `recent_checkins`, `errors`, `duplicates`, `preview`).
- **Tidak ada persentase di respons.** Hitung dengan `percent` (rasio, `null` bila penyebut ≤ 0 → tampilkan "–"): bagian terkirim (`kpiView().sentShare`), kehadiran (`receptionView().attendanceShare`), pax (`paxShare`). Versi sebelumnya punya `AttendancePercent` di respons; kontrak bab 6 melarangnya.
- **`overview` selalu seluruh proyek**, tidak terpengaruh filter atau halaman; `total` pada `GuestList` adalah yang **cocok dengan filter** (untuk halaman), bukan `overview.total`. Empat status RSVP berjumlah `overview.total`.
- **Pil kategori per pihak** dihitung klien dari `side_categories` (`categoryPills`); jangan menjumlah ulang dari halaman tamu.
- **Meja resepsionis**: `expected_guests` mencakup tamu yang menolak tetapi sudah hadir; `pending_guests = expected − checked_in` (≥ 0). `souvenir_stock` hilang bila tak ada barang bertanda souvenir (tampilkan hanya jumlah diserahkan). Antrean `all`/`pending` menyembunyikan tamu yang menolak dan belum hadir **kecuali** pencarian diisi.
- **Waktu** (`checked_in_at`, `invitation_sent_at`, `rsvp_submitted_at`) RFC 3339 UTC: format jam di **zona waktu proyek** dengan `clock` (bukan zona peramban).
- **`pax_actual`**: 0 berarti "ikut `pax_count`"; sesudah check-in adalah jumlah yang datang.
- **`message` galat berbahasa Inggris** dan hanya untuk log. Tampilkan teks dari `messages.ts` berdasarkan `code` (+ kolom/alasan untuk `invalid_*`).
- `sort_order` tidak dikirim; kolom "No" memakai `guestNo(offset, index)`.

**Permintaan**

- **Nomor ponsel dikirim apa adanya** (`phone_number`); server menormalisasi. Klien memakai `normalizeWA` hanya untuk memberi tahu pengguna lebih awal (validasi formulir, tombol WA); hasilnya tidak dikirim.
- **Kategori**: kirim apa yang dipilih/diketik pengguna (kunci registry atau teks); server memetakan label ke kunci dan menyimpan teks lain sebagai kategori kustom. Pihak: kirim kunci.
- **Impor mengirim JSON**, bukan `multipart`: `{ filename, content_base64 }` (`toBase64`); flag `dry_run`/`skip_invalid`/`skip_duplicates` di query. Batas 5 MB diperiksa klien lebih dulu (`fileProblem`).
- **`PUT` mengganti penuh**: formulir ubah mengirim **semua** isian (`guestBody(formFromGuest(g))`); RSVP, status kirim, check-in, souvenir, dan token tidak berubah lewat `PUT`.
- **Kunci yang tidak dikenal diabaikan tanpa galat**: andalkan tipe TypeScript.
- **Teks dipangkas** (`trim`) oleh server; panjang dihitung dalam *code point*: pakai `charCount`/`Array.from`, bukan `.length`.
- **Aman diulang:** semua `GET`, `PUT`, `PATCH` (hasil sama), `DELETE …/checkin`, `scan-qr`. **Tidak** aman diulang: `POST` pembuat (tamu, walk-in), impor sungguhan tanpa `skip_duplicates`. `DELETE` tamu kedua kali menjawab `404`: perlakukan sebagai berhasil dan segarkan. `POST …/checkin` kedua menjawab `409 already_checked_in`: tampilkan jam masuk, bukan galat.

### 4.2 Tambahan pada `src/lib/planner/types.ts`

Hanya sintaks yang dapat dihapus (tanpa `enum`), seperti 01 FE. `Notice`, `SessionBrief`, `Project`, `Metadata`, `ApiError` berasal dari blok 01. Perubahan pada blok 01 ada di [bab 1.3](#13-perubahan-pada-file-yang-sudah-ada) (`EventTypeInfo`).

```ts
// ---------- Menu 05: buku tamu dan meja resepsionis (ditambahkan ke akhir types.ts) ----------
// Tambahan registry (GET /event-types): sisi dan kategori tamu jenis acara. Ubah satu baris di blok 01:
// `export interface EventTypeInfo {` → `export interface EventTypeInfo extends EventTypeGuestInfo {`
// (bila Menu 02–04 FE sudah dibangun: `extends EventTypeAdminInfo, EventTypeFinanceInfo, EventTypeVendorInfo, EventTypeGuestInfo`).
export interface GuestOption { key: string; label: string }
export interface EventTypeGuestInfo { guest_sides: GuestOption[]; guest_categories: GuestOption[] }

export type RSVPStatus = 'unconfirmed' | 'attending' | 'declined' | 'maybe';
export type SentStatus = 'unsent' | 'sent' | 'delivered';
export type AttendanceFilter = 'all' | 'in' | 'pending';
export type GuestTab = 'tamu' | 'checkin';

export interface Guest {
  id: string; name: string;
  guest_side?: string;               // kunci sisi registry; hilang = belum ditentukan
  category: string;                  // kunci registry, 'lainnya', atau teks kustom
  sub_event_scope: string;
  table_number?: string;
  pax_count: number;                 // kuota undangan, 1–20
  phone_number?: string;             // apa adanya
  formatted_wa_number?: string;      // 62…; tautan wa.me dibentuk klien
  personalized_url?: string;         // hanya bila undangan digital terhubung
  invitation_sent_status: SentStatus;
  invitation_sent_at?: string;       // RFC 3339 UTC
  rsvp_status: RSVPStatus;
  pax_actual: number;                // 0 = ikut pax_count (RSVP); sesudah check-in: jumlah yang datang
  rsvp_submitted_at?: string;
  qr_code_token: string;
  is_checked_in: boolean;
  checked_in_at?: string;
  souvenir_received: boolean;
  notes?: string;
}

export interface GuestSideCount { key: string | null; label: string; invitations: number; pax: number } // key null = belum ditentukan
export interface CategoryCount { key: string; label: string; custom?: boolean; invitations: number; pax: number }
export interface SideCategoryCount { side: string | null; category: string; invitations: number; pax: number }
export interface RSVPCounts { attending: number; declined: number; maybe: number; unconfirmed: number; attending_pax: number }
export interface GuestOverview {
  total: number; total_pax: number;
  sides: GuestSideCount[];               // kosong untuk jenis acara tanpa sisi
  categories: CategoryCount[];           // kategori registry (juga yang 0), lalu 'lainnya' dan kustom yang terpakai
  side_categories: SideCategoryCount[];  // sel pihak × kategori yang terisi; kosong untuk jenis tanpa sisi
  invitation: { sent: number; unsent: number };
  rsvp: RSVPCounts;                      // empat status; jumlah = total
}
export interface GuestList { overview: GuestOverview; guests: Guest[]; total: number; limit: number; offset: number }

export interface ReceptionKPI {
  checked_in_guests: number; expected_guests: number; pending_guests: number;
  actual_pax: number; expected_pax: number;
  souvenir_distributed: number; souvenir_stock?: number; // stok hilang bila tak ada barang bertanda souvenir (Menu 04)
}
export interface ReceptionFilters { all: number; checked_in: number; pending: number; categories: CategoryCount[] }
export interface ReceptionOverview {
  kpi: ReceptionKPI; filters: ReceptionFilters;
  recent_checkins: Guest[]; guests: Guest[]; total: number; limit: number; offset: number;
}

export interface CateringProjection {
  confirmed_pax: number; potential_pax: number; declined_pax: number; recommended_portions: number;
  attending_guests: number; declined_guests: number; maybe_guests: number; pending_guests: number;
  buffer_percent: number; buffer_pax: number;
}

export interface ImportRowError { row: number; field: string; reason: string }
export interface ImportDuplicate { row: number; name: string; of_row?: number; of_guest?: string }
export interface ImportPreviewRow {
  row: number; name: string; guest_side?: string; category: string; sub_event_scope: string;
  table_number?: string; pax_count: number; formatted_wa_number?: string;
}
export interface ImportResult {
  dry_run: boolean; total_rows: number; valid_rows: number; invalid_rows: number; created: number; skipped: number;
  errors: ImportRowError[]; duplicates: ImportDuplicate[]; preview: ImportPreviewRow[];
}

// ----- badan permintaan (BE bab 4) -----
export interface GuestBody {
  name: string; guest_side?: string; category?: string; sub_event_scope?: string;
  table_number?: string; pax_count: number; phone_number?: string; notes?: string;
}
export interface RSVPBody { rsvp_status: RSVPStatus; pax_actual: number }
export interface CheckinBody { pax_actual: number; souvenir_received: boolean }
export interface WalkInBody {
  name: string; guest_side?: string; category?: string; table_number?: string;
  pax_actual: number; souvenir_received: boolean; notes?: string;
}
export interface ImportBody { filename: string; content_base64: string }
```

Dicocokkan dengan blok DTO Go ([BE bab 4](BE.md#4-dto-go)): skema JSON diambil dengan `reflect` (jalur bersarang, `omitempty` = opsional, pointer tanpa `omitempty` = nullable, jenis nilai) dan dibandingkan dengan tipe TS di atas (nama Go ↔ TS: `GuestSide` ↔ `GuestSideCount`, `RSVPSummary` ↔ `RSVPCounts`, `GuestRequest` ↔ `GuestBody`, `RSVPRequest` ↔ `RSVPBody`, `CheckinRequest` ↔ `CheckinBody`, `WalkInRequest` ↔ `WalkInBody`, `ImportRequest` ↔ `ImportBody`, `Option` ↔ `GuestOption`). Hasil: **0 selisih** pada 21 pasangan dan 125 jalur (kunci hanya di satu sisi, opsionalitas, `null`, jenis nilai); untuk tipe permintaan hanya kunci dan jenis nilai yang dibandingkan (opsionalitas permintaan dijaga usecase). Pembanding diuji dengan enam mutasi tipe TS (kunci dihapus, opsional dijadikan wajib, jenis diganti, kunci berlebih, `| null` dihapus, jenis elemen larik diganti) dan melaporkan semuanya. Cara mengulang: [bab 11.5](#115-verifikasi-kontrak-cara-mengulang).

### 4.3 Fixture

Empat berkas JSON dan satu CSV di [`fixtures/`](fixtures/) (folder ini) disalin ke `frontend/tests/fixtures/`, ditambah [`planner-wa-vectors.json`](../04_VENDOR_DAN_BELANJA/fixtures/planner-wa-vectors.json) dari Menu 04 (26 vektor `normalizeWA`). Dibangkitkan oleh `TestFixturesAreCurrent` (kode acuan [BE bab 8.3](BE.md#83-hasil-kode-acuan-2026-10-25)) dari tabel 20 tamu [BE bab 6](BE.md#6-data-contoh-proyek-contoh), dengan id dan token tetap (`uuid` v5 dan 32 heksa dari id; tes memeriksa fixture di repo sama dengan keluaran kode), dan **sama dengan respons HTTP asli** modul ([BE bab 8.4](BE.md#84-hasil-implementasi-2026-10-25)): yang berbeda hanya id, tiket, dan waktu, sebab backend membuatnya sendiri. Garis dasar tanggal sama dengan fixture Menu 02: Proyek Contoh dibuat 2026-10-04, Hari-H 2027-03-28.

| Berkas | Isi | Bentuk |
|---|---|---|
| `planner-guests.json` | 20 tamu (halaman 1 dari 1, `limit` 100); `overview`: 20 / 43 pax, pihak 10/22 dan 10/21, terkirim 14, RSVP 12/2/2/4 dengan 27 pax, enam kategori `6/3/4/2/3/2` dan `side_categories` per pihak; tidak ada tamu yang check-in | `GuestList` |
| `planner-reception.json` | **ilustrasi hari-H**: 11 tamu masuk 11.14–12.16 (UTC 04.14–05.16, WIB), `kpi` 11 / 18 / 7 / 22 pax / 39 pax / 11 souvenir / stok 40; `filters` 18 / 11 / 7 dan enam kategori (hanya tamu yang diharapkan: tetangga 1, teman orang tua 2); 10 `recent_checkins`; `guests` 18 (urut nama; dua tamu yang menolak tidak ikut) | `ReceptionOverview` |
| `planner-catering-projection.json` | `confirmed_pax` 27, `potential_pax` 12, `declined_pax` 4, `recommended_portions` 34, 12 hadir · 2 ragu · 4 belum · 2 menolak, `buffer_percent` 25 | `CateringProjection` |
| `planner-import-dryrun.json` | hasil `dry_run` atas `planner-import-sample.csv`: 6 baris, 4 valid, 2 galat (baris 3 `phone_number` angka tanpa 0 di depan; baris 4 `guest_side: laki-laki`), 2 duplikat (baris 5 sama dengan tamu nomor 2; baris 6 sama dengan baris 2), `created` 0 | `ImportResult` |
| `planner-import-sample.csv` | CSV seperti dari Excel Indonesia: BOM UTF-8, pemisah `;`, CRLF | teks |

Contoh baris (dari `planner-guests.json`, tamu 1):

```json
{ "id": "…", "name": "Bpk. Prof. Bambang Soedjarwo", "guest_side": "pria", "category": "vip", "sub_event_scope": "resepsi",
  "table_number": "VIP-01", "pax_count": 2, "phone_number": "081200000101", "formatted_wa_number": "6281200000101",
  "invitation_sent_status": "sent", "invitation_sent_at": "2026-10-04T03:00:00Z", "rsvp_status": "attending", "pax_actual": 2,
  "rsvp_submitted_at": "2026-10-04T03:00:00Z", "qr_code_token": "…32 heksa…", "is_checked_in": false, "souvenir_received": false }
```

**Variasi yang wajib diuji** (turunkan dari fixture di dalam tes, jangan membuat berkas baru): proyek kosong (`guests: []`, semua kategori 0, `sides` registry nol); jenis tanpa sisi (`sides: []` → pil pihak tersembunyi, kolom Pihak hilang); seminar (kata "Peserta", sisi Panitia/Peserta Eksternal); tamu tanpa pihak (`key: null` → pil "Belum ditentukan"); kategori kustom dan `lainnya` (terpakai / 0); tamu tanpa nomor (tombol WA nonaktif); `personalized_url` ada (tombol Salin tautan; pesan dengan tautan); tamu `declined` yang sudah hadir ("Menolak, tetap hadir"); `souvenir_stock` hilang / kurang dari diserahkan; `recommended_portions` 0; hasil impor tanpa galat / semua galat / hanya duplikat; nama 255 karakter dan catatan 2.000 karakter; sesi usang (tampil sebagai Umum); proyek `archived`; halaman kedua (`offset` 100).

### 4.4 Katalog endpoint

Semua di bawah `projects/:id/` (`:id` = UUID proyek) lewat BFF; semua penulisan juga mungkin menjawab `401`, `404 not_found` (proyek), `409 project_read_only`, dan `5xx`/jaringan, yang tidak diulang di kolom Galat. Nama fungsi = `guestApi` (bab 6). Deskripsi lengkap: [BE bab 3](BE.md#3-endpoint). `gid` = id tamu.

| Aksi UI | Fungsi | Metode dan path | Body | Sukses | Galat khusus |
|---|---|---|---|---|---|
| Muat daftar | `listGuests` | `GET guests[?side=&category=&rsvp=&sent=&scope=&search=&limit=&offset=]` | – | `200 GuestList` | `400 invalid_input` (hanya bila URL diketik tangan; `parseGuestQuery` mencegahnya) |
| Polling Tab 1 | `summary` | `GET guests/summary` | – | `200 GuestOverview` | – |
| (satu tamu) | `getGuest` | `GET guests/:gid` | – | `200 Guest` | `404` (tamu) |
| Tambah tamu | `createGuest` | `POST guests` | `GuestBody` | `201 Guest` | `409 guest_limit_reached`, `400 invalid_input`/`invalid_phone`/`invalid_scope` |
| Ubah tamu | `updateGuest` | `PUT guests/:gid` | `GuestBody` | `200 Guest` | seperti POST; `404` |
| Tandai terkirim | `setSentStatus` | `PATCH guests/:gid/sent-status` | `{ sent_status }` | `200 Guest` | `404` |
| Ubah RSVP | `setRSVP` | `PATCH guests/:gid/rsvp` | `RSVPBody` | `200 Guest` | `409 already_checked_in` (batalkan check-in dulu), `400 invalid_input: pax_actual`; `404` |
| Hapus tamu | `deleteGuest` | `DELETE guests/:gid` | – | `204` | `404` (dianggap sukses) |
| Impor | `importGuests` | `POST guests/bulk-import[?dry_run=&skip_invalid=&skip_duplicates=]` | `ImportBody` | `200 ImportResult` | `400 invalid_file`, `413 file_too_large`, `400 invalid_input: rows`, `409 guest_limit_reached`, `429 rate_limited` (5/menit) |
| Unduh templat | `importTemplateHref` | `GET guests/import-template?format=xlsx` | – | `200` berkas `.xlsx` | `429` (10/menit, bersama ekspor) |
| Ekspor daftar | `exportGuestsHref` | `GET guests/export?format=xlsx` | – | `200` berkas `.xlsx` | `429` |
| Proyeksi katering | `cateringProjection` | `GET guests/catering-projection[?buffer_percent=&buffer_pax=]` | – | `200 CateringProjection` | `400 invalid_input: buffer_percent`/`buffer_pax` |
| Muat meja | `receptionOverview` | `GET guests/reception-overview[?status=&category=&search=&limit=&offset=]` | – | `200 ReceptionOverview` | – |
| Scan QR | `scanQR` | `POST guests/scan-qr` | `{ qr_code_token }` | `200 Guest` | `404 not_found` (tiket tak dikenal), `429` (120/menit) |
| Check-in | `checkin` | `POST guests/:gid/checkin` | `CheckinBody` | `200 Guest` | `409 already_checked_in` (ambil ulang `getGuest`), `400 invalid_input: pax_actual` |
| Batal check-in | `cancelCheckin` | `DELETE guests/:gid/checkin` | – | `200 Guest` | `404` |
| Souvenir | `setSouvenir` | `PATCH guests/:gid/souvenir` | `{ souvenir_received }` | `200 Guest` | `404` |
| Walk-in | `walkIn` | `POST guests/walk-in` | `WalkInBody` | `201 Guest` | `409 guest_limit_reached`, `400 invalid_input` |
| Ekspor rekap | `exportAttendanceHref` | `GET guests/export-attendance?format=xlsx` | – | `200` berkas `.xlsx` | `429` |

### 4.5 Contoh permintaan dan respons

Bentuk dari BE bab 3 dan 2 serta fixture. Respons baca sudah dibandingkan dengan backend asli (fixture sama, bab 4.3); respons tulis adalah `Guest` yang sama. Ulangi bila backend berubah ([bab 11.5](#115-verifikasi-kontrak-cara-mengulang)).

**Check-in satu ketukan** (`POST guests/:gid/checkin`): `{ "pax_actual": 2, "souvenir_received": true }` → `200 { "data": Guest }` dengan `is_checked_in: true`, `checked_in_at: "2027-03-28T04:14:00Z"` (tampil "11.14" di Asia/Jakarta). Ketukan kedua → `409 { "error": { "code": "already_checked_in", "message": "…" } }`; lembar memanggil `getGuest(gid)` dan menampilkan `alreadyInText(guest, timezone)` = "Sudah masuk pukul 11.14" dengan tombol *Batalkan check-in* dan *Tutup*.

**Scan** (`POST guests/scan-qr`): isi QR `https://adicara.id/i/sinta-arif?g=0123…cdef&to=Bpk.+Prof.` atau token mentah → `extractTicketToken` → `{ "qr_code_token": "0123…cdef" }` → `200 Guest` (juga bila sudah check-in); isi yang bukan tiket (`extractTicketToken` = `null`) langsung "Tiket tidak dikenali" tanpa permintaan; token tak dikenal → `404`.

**Impor** (`POST guests/bulk-import?dry_run=true`): badan `{ "filename": "tamu.csv", "content_base64": "77u/TmFtYSBUYW11…" }` (BOM + `Nama Tamu;Pihak;…`) → `200 ImportResult` seperti [`planner-import-dryrun.json`](fixtures/planner-import-dryrun.json): `errors: [{row:3, field:"phone_number", reason:"must be an Indonesian phone number (08…, +62…)"}, {row:4, field:"guest_side", reason:"must be a registry side key"}]`, `duplicates: [{row:5, name:"Raditya Pratama & Pasangan", of_guest:"…"}, {row:6, name:"dewi anggraini", of_row:2}]`. Konfirmasi: kirim ulang tanpa `dry_run`, dengan `skip_invalid=true` bila pengguna mencentang "lewati baris galat" (tanpa itu: `400 invalid_input` `rows: 2 rows are invalid`).

**Proyeksi** (`GET guests/catering-projection`): `200` seperti [`planner-catering-projection.json`](fixtures/planner-catering-projection.json). Modal pos Menu 03 mengisi `quantity = 34`, satuan "porsi", lalu `PUT` pos dengan semua isian.

**Unduhan** (`GET guests/export?format=xlsx`): `200`, `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="tamu-<slug>-<tanggal>.xlsx"`, `Cache-Control: no-store`. Permintaan ke-11 dalam semenit → `429 {"error":{"code":"rate_limited","message":"too many requests, try again later"}}`.

### 4.6 Katalog galat

Bentuk: `{ "error": { "code": "…", "message": "…" } }` (+ `details` pada beberapa kode Menu 01). Untuk `invalid_input`/`invalid_phone`/`invalid_file`, `message` = `<kolom>: <alasan>` dan dipetakan ke teks per kolom (bab 9.2); kode lain dipetakan per kode (bab 9.1). Daftar alasan baku: [BE §3.4](BE.md#34-validasi-dan-batas); tes 11.1 memastikan **setiap** alasan di sana punya teks Indonesia atau diserahkan eksplisit ke fungsi Menu 04/03/01.

| Status | Kode | Contoh `message` | Penyebab dan penanganan UI |
|:-:|---|---|---|
| 400 | `invalid_input` | `name: required` · `pax_count: must be between 1 and 20` · `guest_side: must be a registry side key` · `rows: 2 rows are invalid` | pesan di bawah kolom (bab 9.2); `rows:` ditampilkan di panel impor |
| 400 | `invalid_phone` | `phone_number: must be an Indonesian phone number (08…, +62…)` | di bawah kolom nomor ponsel; klien sudah memberi tahu lebih awal lewat `normalizeWA` |
| 400 | `invalid_scope` | `sub_event_scope: unknown session` | sesi dihapus di tab lain → muat ulang area |
| 400 | `invalid_file` | `file: missing column: Nama Tamu` · `file: too many rows (max 2000)` | panel impor (bab 9.2: `guestFieldMessage('file', …)`) |
| 400 | `invalid_json` | `invalid JSON body` | kesalahan klien (tipe salah, body kosong tanpa `{}`) |
| 401 | `unauthorized` | `missing bearer token` | `plannerFetch` mengalihkan ke login |
| 404 | `not_found` | `project not found` · `guest not found` | dibedakan oleh **aksi** (`where`): `scan` → "Tiket tidak dikenali"; setelah `DELETE` kedua dianggap sukses |
| 409 | `project_read_only` | `the project is archived; restore it to make changes` | nonaktifkan semua kontrol ubah, teks 01 |
| 409 | `guest_limit_reached` | `at most 5000 guests per project` | banner di modal / panel impor; tidak ada baris yang tertulis |
| 409 | `already_checked_in` | `the guest is already checked in` | **bukan galat tampil**: ambil ulang tamu dan tampilkan jam masuk (check-in), atau "Batalkan check-in dulu" (ubah RSVP) |
| 409 | `conflict` | `the project was changed concurrently, try again` | jarang; teks 01, aman dicoba ulang |
| 413 | `file_too_large` | `file too large` | "Ukuran berkas melebihi 5 MB." |
| 429 | `rate_limited` | `too many requests, try again later` | scan, impor, dan unduhan (bab 9.1) |
| 5xx / jaringan | `internal`, `service_unavailable` | – | teks umum 01; jangan mengosongkan layar |

---

## 5. Library `lib/planner/guest.ts`

Aturan 01 FE bab 5: modul **murni**, tanpa alias `@/`, tanpa `enum`, supaya tes `node --experimental-strip-types` dapat mengimpornya. Ia mengimpor `format.ts` (Menu 01: `formatDate`, `percent`) dan `vendor.ts` (Menu 04: `normalizeWA`, `waLink`). **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-guesttestmjs)). Isi: istilah RSVP, pil pihak/kategori (termasuk hitungan per pihak seperti dua tabel PDF), kartu metrik, pesan dan tautan WhatsApp, jam check-in di zona proyek, angka meja resepsionis, keputusan tombol proyeksi dan impor, validasi formulir dengan alasan baku server, state URL, dan token tiket dari isi QR.

```ts
// src/lib/planner/guest.ts
import { formatDate, percent } from './format.ts';
import { normalizeWA, waLink } from './vendor.ts';
import type {
  AttendanceFilter, CateringProjection, CategoryCount, Guest, GuestBody, GuestOverview, GuestTab, ImportResult,
  Project, ReceptionFilters, ReceptionKPI, RSVPStatus, SessionBrief,
} from './types.ts';

/** Batas yang ditegakkan backend (BE bab 3.4); klien memakainya untuk `maxlength`, `max`, dan tombol, bukan sebagai pengganti server. */
export const LIMITS = {
  name: 255, table: 30, phone: 40, notes: 2000, category: 60, search: 100,
  paxMin: 1, paxMax: 20,
  guests: 5000, importRows: 2000, importBytes: 5 * 1024 * 1024,
  page: 100, pageMax: 500, receptionPage: 50, receptionPageMax: 200,
  recent: 10, bufferPercentMax: 100, bufferPaxMax: 1000, defaultBufferPercent: 25,
} as const;

// ---------- istilah ----------

export const RSVP_ORDER: readonly RSVPStatus[] = ['attending', 'maybe', 'declined', 'unconfirmed'];
export const RSVP_LABELS: Record<RSVPStatus, string> = { attending: 'Hadir', maybe: 'Ragu', declined: 'Tidak hadir', unconfirmed: 'Belum merespons' };
export type Tone = 'good' | 'warn' | 'bad' | 'muted';
export const RSVP_TONES: Record<RSVPStatus, Tone> = { attending: 'good', maybe: 'warn', declined: 'bad', unconfirmed: 'muted' };

/** Kata benda teknis "undangan": seminar memakai "pendaftaran" (registry hanya punya `labels.guest`, 05 FE bab 13 butir 3). */
export const invitationNoun = (labels: { guest: string }): 'undangan' | 'pendaftaran' => (labels.guest === 'Peserta' ? 'pendaftaran' : 'undangan');

// ---------- pil pihak dan kategori ----------

export interface Pill { key: string; label: string; count: number }
export const FILTER_ALL = '';
export const SIDE_NONE = 'none';

/** [Semua] + satu pil per pihak (dan "Belum ditentukan" bila ada). Kosong bila jenis acara tanpa sisi: pemisah disembunyikan. */
export function sidePills(o: GuestOverview): Pill[] {
  if (o.sides.length === 0) return [];
  return [
    { key: FILTER_ALL, label: 'Semua', count: o.total },
    ...o.sides.map((s) => ({ key: s.key ?? SIDE_NONE, label: s.label, count: s.invitations })),
  ];
}

/**
 * Penghitung kategori. Tanpa pihak terpilih: seluruh proyek. Dengan pihak terpilih: hitungan tabel pihak itu saja (PDF Hal. 13 punya
 * enam penghitung per tabel). Kategori registry selalu tampil (juga 0); kategori kustom dan 'lainnya' hanya bila terpakai.
 */
export function categoryPills(o: GuestOverview, side: string = FILTER_ALL): Pill[] {
  const counts = new Map<string, number>();
  if (side === FILTER_ALL || o.sides.length === 0) {
    for (const c of o.categories) counts.set(c.key, c.invitations);
  } else {
    const want = side === SIDE_NONE ? null : side;
    for (const c of o.side_categories) if (c.side === want) counts.set(c.category, (counts.get(c.category) ?? 0) + c.invitations);
  }
  return o.categories
    .map((c) => ({ key: c.key, label: c.label, count: counts.get(c.key) ?? 0, custom: c.custom === true }))
    .filter((c) => !c.custom || c.count > 0)
    .map(({ key, label, count }) => ({ key, label, count }));
}

/** Nomor urut baris pada halaman ke-N (kolom "No"): bukan `sort_order`. */
export const guestNo = (offset: number, index: number): number => offset + index + 1;

// ---------- kartu metrik ----------

export interface KpiView {
  invitations: number; pax: number; sent: number; unsent: number; sentShare: number | null;
  rsvp: { key: RSVPStatus; label: string; tone: Tone; count: number }[]; rsvpTotal: number; attendingPax: number;
}
export function kpiView(o: GuestOverview): KpiView {
  return {
    invitations: o.total, pax: o.total_pax, sent: o.invitation.sent, unsent: o.invitation.unsent, sentShare: percent(o.invitation.sent, o.total),
    rsvp: RSVP_ORDER.map((key) => ({ key, label: RSVP_LABELS[key], tone: RSVP_TONES[key], count: o.rsvp[key] })),
    rsvpTotal: o.rsvp.attending + o.rsvp.declined + o.rsvp.maybe + o.rsvp.unconfirmed,
    attendingPax: o.rsvp.attending_pax,
  };
}

// ---------- WhatsApp ----------

export interface MessageContext { title: string; date: string; sessionName?: string; startTime?: string; endTime?: string; venue?: string }

const dot = (hhmm: string): string => hhmm.replace(':', '.');

/**
 * Konteks pesan: tanggal dan jam dari sesi tamu bila punya, selain itu Hari-H; baris sesi hanya untuk sesi nyata (bukan Umum);
 * tempat dari `location_name` sesi, lalu `metadata.venue_name` (+ kota).
 */
export function messageContext(project: Pick<Project, 'title' | 'event_date' | 'metadata'>, session?: Pick<SessionBrief, 'scope_key' | 'name' | 'session_date' | 'start_time' | 'end_time' | 'location_name'>): MessageContext {
  const real = session !== undefined && session.scope_key !== 'all';
  const meta = project.metadata;
  const city = typeof meta.city === 'string' && meta.city.trim() !== '' ? meta.city.trim() : '';
  const base = (real ? session.location_name : undefined) || (typeof meta.venue_name === 'string' ? meta.venue_name.trim() : '');
  const venue = [base, city].filter((s) => s !== '' && s !== undefined).join(', ');
  return {
    title: project.title,
    date: (real ? session.session_date : undefined) ?? project.event_date,
    ...(real ? { sessionName: session.name } : {}),
    ...(real && session.start_time ? { startTime: session.start_time } : {}),
    ...(real && session.end_time ? { endTime: session.end_time } : {}),
    ...(venue !== '' ? { venue } : {}),
  };
}

/** Templat BE bab 2.4 (netral jenis acara). Baris yang datanya kosong dihilangkan. Tanpa tautan personal (Standalone) tamu diminta membalas pesan. */
export function waMessage(guestName: string, ctx: MessageContext, personalizedUrl?: string): string {
  const times = ctx.startTime ? ` (${dot(ctx.startTime)}${ctx.endTime ? `–${dot(ctx.endTime)}` : ''})` : '';
  const detail = [`📅 ${formatDate(ctx.date)}`, ctx.sessionName ? `📍 ${ctx.sessionName}${times}` : '', ctx.venue ? `🏛️ ${ctx.venue}` : ''].filter((l) => l !== '');
  const rsvp = personalizedUrl ? `Detail acara dan konfirmasi kehadiran (RSVP): ${personalizedUrl}` : 'Mohon konfirmasi kehadiran dengan membalas pesan ini.';
  return [
    `Yth. ${guestName},`, '',
    `Tanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i pada ${ctx.title}:`,
    ...detail, '', rsvp, '', 'Terima kasih.',
  ].join('\n');
}

/** null = tombol WA nonaktif ("Nomor belum diisi"). Status kirim tidak berubah otomatis: pengguna menekan "Tandai terkirim". */
export function guestWaLink(g: Pick<Guest, 'name' | 'formatted_wa_number' | 'personalized_url'>, ctx: MessageContext): string | null {
  return g.formatted_wa_number ? waLink(g.formatted_wa_number, waMessage(g.name, ctx, g.personalized_url)) : null;
}

// ---------- meja resepsionis ----------

/** "11.14": jam di zona waktu proyek (bukan zona peramban). */
export function clock(iso: string, timeZone: string): string {
  return new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone }).format(new Date(iso));
}
export const alreadyInText = (g: Pick<Guest, 'checked_in_at'>, timeZone: string): string =>
  g.checked_in_at ? `Sudah masuk pukul ${clock(g.checked_in_at, timeZone)}` : 'Sudah masuk';

/** Tamu yang menolak tetapi datang: tetap `declined` di RSVP, dihitung hadir di meja resepsionis (BE bab 2.5). */
export const isDeclinedArrival = (g: Pick<Guest, 'rsvp_status' | 'is_checked_in'>): boolean => g.rsvp_status === 'declined' && g.is_checked_in;
/** RSVP tidak dapat diubah sesudah check-in (BE bab 2.2): batalkan check-in lebih dulu. */
export const canEditRSVP = (g: Pick<Guest, 'is_checked_in'>): boolean => !g.is_checked_in;

export interface ReceptionView { attendanceShare: number | null; paxShare: number | null; souvenirRemaining: number | null; souvenirShort: boolean }
export function receptionView(k: ReceptionKPI): ReceptionView {
  const remaining = k.souvenir_stock === undefined ? null : k.souvenir_stock - k.souvenir_distributed;
  return {
    attendanceShare: percent(k.checked_in_guests, k.expected_guests),
    paxShare: percent(k.actual_pax, k.expected_pax),
    souvenirRemaining: remaining,
    souvenirShort: remaining !== null && remaining < 0,
  };
}

export function statusChips(f: ReceptionFilters): { key: AttendanceFilter; label: string; count: number }[] {
  return [
    { key: 'all', label: 'Semua', count: f.all },
    { key: 'in', label: 'Sudah hadir', count: f.checked_in },
    { key: 'pending', label: 'Belum hadir', count: f.pending },
  ];
}
export const categoryChips = (f: ReceptionFilters): CategoryCount[] => f.categories.filter((c) => !c.custom || c.invitations > 0);

/** Pax awal pada lembar check-in: jumlah RSVP bila ada, selain itu kuota undangan; selalu 1–20. */
export const defaultCheckinPax = (g: Pick<Guest, 'pax_actual' | 'pax_count'>): number =>
  Math.min(LIMITS.paxMax, Math.max(LIMITS.paxMin, g.pax_actual > 0 ? g.pax_actual : g.pax_count));

// ---------- proyeksi katering (dipakai modal pos Menu 03) ----------

/** Tombol "Gunakan proyeksi RSVP" aktif hanya bila ada porsi: `quantity` pos harus ≥ 0,01 (Menu 03). */
export const canUseProjection = (p: Pick<CateringProjection, 'recommended_portions'>): boolean => p.recommended_portions > 0;
export function projectionText(p: CateringProjection): string {
  if (p.confirmed_pax === 0) return p.recommended_portions > 0 ? `Belum ada tamu yang hadir; hanya cadangan ${p.buffer_pax} porsi.` : 'Belum ada tamu yang menyatakan hadir.';
  const buffer = p.buffer_percent > 0 ? ` + cadangan ${p.buffer_percent}%` : '';
  const extra = p.buffer_pax > 0 ? ` + ${p.buffer_pax} porsi` : '';
  const maybe = p.potential_pax > 0 ? ` ${p.potential_pax} pax lain belum pasti (ragu atau belum merespons) dan tidak dihitung.` : '';
  return `${p.confirmed_pax} pax hadir${buffer}${extra} = ${p.recommended_portions} porsi.${maybe}`;
}

// ---------- impor ----------

export interface ImportOptions { skipInvalid: boolean; skipDuplicates: boolean }
export function importDecision(r: ImportResult, o: ImportOptions): { willCreate: number; canSubmit: boolean } {
  const dup = o.skipDuplicates ? r.duplicates.length : 0;
  const willCreate = Math.max(0, r.valid_rows - dup);
  return { willCreate, canSubmit: (r.invalid_rows === 0 || o.skipInvalid) && willCreate > 0 };
}
/** Pemeriksaan ringan sebelum membaca berkas; validasi sebenarnya di server. null = lolos. */
export function fileProblem(f: { name: string; size: number }): string | null {
  if (!/\.(csv|xlsx)$/i.test(f.name)) return 'Berkas harus berakhiran .csv atau .xlsx.';
  if (f.size === 0) return 'Berkas kosong.';
  if (f.size > LIMITS.importBytes) return 'Ukuran berkas melebihi 5 MB.';
  return null;
}
/** ArrayBuffer → base64 tanpa menumpuk argumen (btoa pada string 5 MB sekaligus tidak aman). */
export function toBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(bin);
}

// ---------- formulir tamu ----------

export interface GuestForm {
  name: string; guest_side: string; category: string; sub_event_scope: string;
  table_number: string; pax_count: string; phone_number: string; notes: string;
}
export const emptyGuestForm = (): GuestForm => ({ name: '', guest_side: '', category: '', sub_event_scope: 'all', table_number: '', pax_count: '1', phone_number: '', notes: '' });

const chars = (s: string): number => Array.from(s).length;

/** Cermin validasi server (BE bab 3.4) dengan alasan yang sama, supaya satu kamus pesan melayani klien dan server. {} = lolos. */
export function validateGuestForm(f: GuestForm): Record<string, string> {
  const e: Record<string, string> = {};
  const name = f.name.trim();
  if (name === '') e.name = 'required';
  else if (chars(name) > LIMITS.name) e.name = `must be at most ${LIMITS.name} characters`;
  if (chars(f.category.trim()) > LIMITS.category) e.category = `must be at most ${LIMITS.category} characters`;
  if (chars(f.table_number.trim()) > LIMITS.table) e.table_number = `must be at most ${LIMITS.table} characters`;
  const pax = f.pax_count.trim();
  if (!/^\d+$/.test(pax)) e.pax_count = 'must be an integer';
  else if (Number(pax) < LIMITS.paxMin || Number(pax) > LIMITS.paxMax) e.pax_count = `must be between ${LIMITS.paxMin} and ${LIMITS.paxMax}`;
  const phone = f.phone_number.trim();
  if (chars(phone) > LIMITS.phone) e.phone_number = `must be at most ${LIMITS.phone} characters`;
  else if (phone !== '' && normalizeWA(phone) === null) e.phone_number = 'must be an Indonesian phone number (08…, +62…)';
  if (chars(f.notes.trim()) > LIMITS.notes) e.notes = `must be at most ${LIMITS.notes} characters`;
  return e;
}

/** Kolom kosong tidak dikirim (server memakai bawaannya); PUT mengganti penuh, jadi formulir edit mengirim semua kolom yang terisi. */
export function guestBody(f: GuestForm): GuestBody {
  const opt = (s: string): string | undefined => (s.trim() === '' ? undefined : s.trim());
  const b: GuestBody = { name: f.name.trim(), pax_count: Number(f.pax_count) };
  const side = opt(f.guest_side), cat = opt(f.category), table = opt(f.table_number), phone = opt(f.phone_number), notes = f.notes.trim() === '' ? undefined : f.notes.trim();
  if (side) b.guest_side = side;
  if (cat) b.category = cat;
  if (f.sub_event_scope !== '') b.sub_event_scope = f.sub_event_scope;
  if (table) b.table_number = table;
  if (phone) b.phone_number = phone;
  if (notes) b.notes = notes;
  return b;
}
export const formFromGuest = (g: Guest): GuestForm => ({
  name: g.name, guest_side: g.guest_side ?? '', category: g.category, sub_event_scope: g.sub_event_scope,
  table_number: g.table_number ?? '', pax_count: String(g.pax_count), phone_number: g.phone_number ?? '', notes: g.notes ?? '',
});

// ---------- state di URL ----------

export interface GuestQuery {
  tab: GuestTab; side: string; category: string; rsvp: '' | RSVPStatus; sent: '' | 'sent' | 'unsent'; scope: string; search: string; page: number;
}
export const defaultGuestQuery = (): GuestQuery => ({ tab: 'tamu', side: '', category: '', rsvp: '', sent: '', scope: '', search: '', page: 1 });

const KEY = /^[a-z0-9_]{1,40}$/;
/** Nilai yang tidak sah jatuh ke bawaan (tidak pernah dikirim ke server): URL buatan tangan tidak merusak halaman. */
export function parseGuestQuery(sp: URLSearchParams): GuestQuery {
  const q = defaultGuestQuery();
  if (sp.get('tab') === 'checkin') q.tab = 'checkin';
  const side = sp.get('side') ?? '';
  if (KEY.test(side)) q.side = side;
  const cat = (sp.get('category') ?? '').trim();
  if (cat !== '' && chars(cat) <= LIMITS.category) q.category = cat;
  const rsvp = sp.get('rsvp');
  if (rsvp !== null && (RSVP_ORDER as readonly string[]).includes(rsvp)) q.rsvp = rsvp as RSVPStatus;
  const sent = sp.get('sent');
  if (sent === 'sent' || sent === 'unsent') q.sent = sent;
  const scope = sp.get('scope') ?? '';
  if (KEY.test(scope)) q.scope = scope;
  const search = (sp.get('q') ?? '').trim();
  if (search !== '' && chars(search) <= LIMITS.search) q.search = search;
  const page = Number(sp.get('page'));
  if (Number.isInteger(page) && page >= 1 && page <= 100000) q.page = page;
  return q;
}

/** Query untuk GET …/guests (BE bab 3.1): hanya yang terisi. */
export function guestApiQuery(q: GuestQuery, limit: number = LIMITS.page): Record<string, string> {
  const out: Record<string, string> = { limit: String(limit), offset: String((q.page - 1) * limit) };
  if (q.side) out.side = q.side;
  if (q.category) out.category = q.category;
  if (q.rsvp) out.rsvp = q.rsvp;
  if (q.sent) out.sent = q.sent;
  if (q.scope) out.scope = q.scope;
  if (q.search) out.search = q.search;
  return out;
}

/** Tautan halaman: mengubah filter mengembalikan ke halaman 1; hanya nilai bukan-bawaan yang masuk URL. */
export function guestHref(base: string, q: GuestQuery, patch: Partial<GuestQuery> = {}): string {
  const filterChanged = (['side', 'category', 'rsvp', 'sent', 'scope', 'search'] as const).some((k) => k in patch && patch[k] !== q[k]);
  const n: GuestQuery = { ...q, ...patch, page: patch.page ?? (filterChanged ? 1 : q.page) };
  const sp = new URLSearchParams();
  if (n.tab !== 'tamu') sp.set('tab', n.tab);
  if (n.side) sp.set('side', n.side);
  if (n.category) sp.set('category', n.category);
  if (n.rsvp) sp.set('rsvp', n.rsvp);
  if (n.sent) sp.set('sent', n.sent);
  if (n.scope) sp.set('scope', n.scope);
  if (n.search) sp.set('q', n.search);
  if (n.page > 1) sp.set('page', String(n.page));
  const s = sp.toString();
  return s === '' ? base : `${base}?${s}`;
}
export const pageCount = (total: number, limit: number): number => Math.max(1, Math.ceil(total / limit));

export interface ReceptionQuery { status: AttendanceFilter; category: string; search: string; page: number }
export const defaultReceptionQuery = (): ReceptionQuery => ({ status: 'all', category: '', search: '', page: 1 });

/** Parameter URL tab meja resepsionis: `status` (in|pending), `category`, `q`, `page`. Nilai tidak sah jatuh ke bawaan. */
export function parseReceptionQuery(sp: URLSearchParams): ReceptionQuery {
  const q = defaultReceptionQuery();
  const status = sp.get('status');
  if (status === 'in' || status === 'pending') q.status = status;
  const cat = (sp.get('category') ?? '').trim();
  if (cat !== '' && chars(cat) <= LIMITS.category) q.category = cat;
  const search = (sp.get('q') ?? '').trim();
  if (search !== '' && chars(search) <= LIMITS.search) q.search = search;
  const page = Number(sp.get('page'));
  if (Number.isInteger(page) && page >= 1 && page <= 100000) q.page = page;
  return q;
}

/** Tautan halaman tab meja resepsionis (selalu `tab=checkin`); mengubah filter mengembalikan ke halaman 1. */
export function receptionHref(base: string, q: ReceptionQuery, patch: Partial<ReceptionQuery> = {}): string {
  const filterChanged = (['status', 'category', 'search'] as const).some((k) => k in patch && patch[k] !== q[k]);
  const n: ReceptionQuery = { ...q, ...patch, page: patch.page ?? (filterChanged ? 1 : q.page) };
  const sp = new URLSearchParams({ tab: 'checkin' });
  if (n.status !== 'all') sp.set('status', n.status);
  if (n.category) sp.set('category', n.category);
  if (n.search) sp.set('q', n.search);
  if (n.page > 1) sp.set('page', String(n.page));
  return `${base}?${sp.toString()}`;
}

/**
 * Token tiket dari isi QR: token mentah (32 heksa) atau tautan personal (`…?g=<token>`). Generator QR sering membesarkan huruf
 * (mode alfanumerik), jadi hasilnya dinormalkan ke huruf kecil. null = bukan tiket Adicara ("Tiket tidak dikenali" tanpa permintaan).
 */
export function extractTicketToken(text: string): string | null {
  const m = /(?<![0-9a-f])[0-9a-f]{32}(?![0-9a-f])/i.exec(text.trim());
  return m ? m[0].toLowerCase() : null;
}
export function receptionApiQuery(q: ReceptionQuery, limit: number = LIMITS.receptionPage): Record<string, string> {
  const out: Record<string, string> = { limit: String(limit), offset: String((q.page - 1) * limit) };
  if (q.status !== 'all') out.status = q.status;
  if (q.category) out.category = q.category;
  if (q.search) out.search = q.search;
  return out;
}

// ---------- pembaruan lokal sesudah aksi ----------

export const replaceGuest = (list: Guest[], next: Guest): Guest[] => list.map((g) => (g.id === next.id ? next : g));
export const removeGuest = (list: Guest[], id: string): Guest[] => list.filter((g) => g.id !== id);
```

---

## 6. Pembungkus panggilan, BFF, dan unduhan

Tidak ada file BFF baru. Pemanggilan klien memakai `plannerFetch('projects/:id/…')` (01 FE bab 5.5) dengan path yang lolos aturan BFF `^[a-z0-9/_-]+$` (`bulk-import`, `import-template`, `reception-overview`, `scan-qr`, `walk-in`, `sent-status`, `catering-projection`, `export-attendance`, `checkin`, `souvenir`, `summary` semuanya lolos; query `?dry_run=` dan `?format=xlsx` bukan bagian path; diuji). Semua non-GET mengirim `Content-Type: application/json` dengan body (`'{}'` bila kosong). Menu ini memakai metode `PUT`, `PATCH`, dan `DELETE`.

### 6.1 `guestApi.ts`

Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan sebagai argumen sehingga modul bisa diuji tanpa peramban; tipe `PlannerFetch` diambil dari `vendorApi.ts` (Menu 04). Setiap id di path diperiksa UUID huruf kecil dan melempar `Error` bila tidak (kesalahan pemrogram, bukan permintaan ke server). **Sudah dijalankan**: tes unit memeriksa metode, path, query, dan body ke-16 fungsi (18 panggilan), bahwa path lolos aturan BFF, dan bahwa id yang bukan UUID (`bukan-uuid`, `../../auth`, `x`, UUID huruf besar) melempar.

```ts
// src/lib/planner/guestApi.ts
import type {
  CateringProjection, CheckinBody, Guest, GuestBody, GuestList, GuestOverview, ImportBody, ImportResult, ReceptionOverview, RSVPBody, WalkInBody,
} from './types.ts';
import type { PlannerFetch } from './vendorApi.ts';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
/** Path dibangun hanya dari UUID huruf kecil; selain itu kesalahan pemrogram (lempar), bukan permintaan ke server. */
const uuid = (v: string): string => {
  if (!UUID.test(v)) throw new Error(`bukan UUID: ${v}`);
  return v;
};

/** Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). Semua non-GET mengirim body (plannerFetch mengisi '{}' bila undefined). */
export function guestApi(call: PlannerFetch, projectId: string) {
  const p = `projects/${uuid(projectId)}/guests`;
  const g = (id: string) => `${p}/${uuid(id)}`;
  return {
    // ----- Tab 1: daftar tamu -----
    /** `query` dari `guestApiQuery`. `overview` selalu seluruh proyek; `guests` yang disaring dan dipotong halaman. */
    listGuests: (query: Record<string, string>) => call<GuestList>(p, { query }),
    /** Polling 30 dtk: hanya penghitung. */
    summary: () => call<GuestOverview>(`${p}/summary`),
    getGuest: (id: string) => call<Guest>(g(id)),
    createGuest: (body: GuestBody) => call<Guest>(p, { method: 'POST', body }),
    /** PUT mengganti nama, pihak, kategori, sesi, meja, pax, telepon, dan catatan; RSVP, kirim, check-in, dan token tidak berubah. */
    updateGuest: (id: string, body: GuestBody) => call<Guest>(g(id), { method: 'PUT', body }),
    setSentStatus: (id: string, sent_status: 'unsent' | 'sent') => call<Guest>(`${g(id)}/sent-status`, { method: 'PATCH', body: { sent_status } }),
    /** Ditolak `409 already_checked_in` bila tamu sudah check-in (batalkan check-in lebih dulu). */
    setRSVP: (id: string, body: RSVPBody) => call<Guest>(`${g(id)}/rsvp`, { method: 'PATCH', body }),
    deleteGuest: (id: string) => call<null>(g(id), { method: 'DELETE' }),
    /** Berkas dikirim sebagai JSON (`ImportBody`), bukan multipart. `dry_run` tidak menulis apa pun. */
    importGuests: (body: ImportBody, o: { dryRun?: boolean; skipInvalid?: boolean; skipDuplicates?: boolean } = {}) => {
      const query: Record<string, string> = {};
      if (o.dryRun) query.dry_run = 'true';
      if (o.skipInvalid) query.skip_invalid = 'true';
      if (o.skipDuplicates) query.skip_duplicates = 'true';
      return call<ImportResult>(`${p}/bulk-import`, { method: 'POST', body, query });
    },
    /** Dipanggil modal pos Menu 03 ("Gunakan proyeksi RSVP"). Bawaan server: 25% dan 0 porsi. */
    cateringProjection: (o: { bufferPercent?: number; bufferPax?: number } = {}) => {
      const query: Record<string, string> = {};
      if (o.bufferPercent !== undefined) query.buffer_percent = String(o.bufferPercent);
      if (o.bufferPax !== undefined) query.buffer_pax = String(o.bufferPax);
      return call<CateringProjection>(`${p}/catering-projection`, { query });
    },

    // ----- Tab 2: meja resepsionis -----
    /** Polling 5 dtk; `query` dari `receptionApiQuery`. */
    receptionOverview: (query: Record<string, string>) => call<ReceptionOverview>(`${p}/reception-overview`, { query }),
    /** Token tidak dikenal atau milik proyek lain: `404 not_found` ("Tiket tidak dikenali"). Tamu yang sudah check-in tetap `200`. */
    scanQR: (qr_code_token: string) => call<Guest>(`${p}/scan-qr`, { method: 'POST', body: { qr_code_token } }),
    /** `409 already_checked_in`: ambil ulang tamu (`getGuest`) untuk menampilkan jam masuk sebelumnya. */
    checkin: (id: string, body: CheckinBody) => call<Guest>(`${g(id)}/checkin`, { method: 'POST', body }),
    /** Idempoten; juga menghapus tanda souvenir. */
    cancelCheckin: (id: string) => call<Guest>(`${g(id)}/checkin`, { method: 'DELETE' }),
    setSouvenir: (id: string, souvenir_received: boolean) => call<Guest>(`${g(id)}/souvenir`, { method: 'PATCH', body: { souvenir_received } }),
    walkIn: (body: WalkInBody) => call<Guest>(`${p}/walk-in`, { method: 'POST', body }),
  };
}

/** Tujuan tautan unduhan .xlsx (GET biasa lewat BFF). Hanya `xlsx`: PDF belum disetujui. */
const dl = (projectId: string, what: string): string => `/dashboard/api/planner/projects/${uuid(projectId)}/guests/${what}?format=xlsx`;
export const importTemplateHref = (projectId: string): string => dl(projectId, 'import-template');
export const exportGuestsHref = (projectId: string): string => dl(projectId, 'export');
export const exportAttendanceHref = (projectId: string): string => dl(projectId, 'export-attendance');
```

Pemakaian di skrip klien:

```ts
const api = guestApi(plannerFetch, projectId);
const res = await api.checkin(guestId, { pax_actual: 2, souvenir_received: true });
if (!res.ok) {
  if (res.error?.code === 'already_checked_in') return showAlreadyIn((await api.getGuest(guestId)).data);   // jam masuk sebelumnya
  toast(guestErrorMessage(res.status, res.error?.code, 'guest') ?? plannerErrorMessage(res.status, res.error?.code)); return;
}
playBeep(); await refreshArea();                                                                           // partial
```

### 6.2 Unduhan `.xlsx` dan impor

Tautan biasa `<a href={exportGuestsHref(project.id)} download>`, `exportAttendanceHref`, dan `importTemplateHref`; `GET` tidak butuh `Content-Type`, cookie sesi ikut. BFF meneruskan berkas sebagai aliran beserta `Content-Disposition` dari backend dan menambahkan `Cache-Control: no-store` (berkas berisi data pribadi). Bila sesi habis, middleware membalas pengalihan ke halaman login. `429` membuka halaman galat BFF; alternatif `fetch` + Blob agar bisa menampilkan teks batas dicatat di bab 13 butir 8. **Hanya `xlsx`**: versi awal menjanjikan PDF untuk daftar dan rekap, yang pustakanya belum disetujui.

Impor: berkas dibaca dengan `File.arrayBuffer()`, diubah `toBase64`, dikirim lewat `importGuests` (JSON; BFF memberi batas badan 7,5 MiB hanya untuk jalur ini, bab 1.3). Pratinjau selalu `dry_run: true`; tombol "Impor" mengirim ulang berkas yang sama tanpa `dry_run`.

---

## 7. Tab 1: Daftar Tamu & Undangan

### 7.1 Wireframe (Proyek Contoh; mode Standalone)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Buku Tamu & RSVP                                                  [ 💍 Contoh: Pernikahan Sinta & Arif ▼ ] │
│ [★ Daftar Tamu & Undangan]   [Meja Resepsionis & Check-in]                                             │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌────────────────┐ ┌─────────────────────┐ ┌─────────────────┐ ┌───────────────────────────┐        │
│ │ 👥 Undangan    │ │ ⚖️ Per Pihak        │ │ 📤 Terkirim     │ │ ✉️ RSVP                   │        │
│ │ 20 · 43 pax    │ │ Pria 10 · 22 pax    │ │ 14 dari 20      │ │ 🟢 Hadir 12 (27 pax)      │        │
│ │                │ │ Wanita 10 · 21 pax  │ │ [███████░░] 70% │ │ 🟡 Ragu 2   🔴 Tidak 2    │        │
│ │                │ │                     │ │ 6 belum         │ │ ⚪ Belum merespons 4       │        │
│ └────────────────┘ └─────────────────────┘ └─────────────────┘ └───────────────────────────┘        │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ℹ️ Mode Standalone: kirim undangan cetak atau pesan WhatsApp biasa dan ubah status RSVP secara manual.  │
│    Undangan Digital Adicara (tautan personal dan RSVP otomatis) belum tersedia.                       │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Pihak:     [Semua 20] [Pihak Pria (CPP) 10] [Pihak Wanita (CPW) 10]                                  │
│ Kategori:  [Teman 6] [Kolega 3] [Keluarga 4] [Tetangga 2] [Teman Orang Tua 3] [VIP 2]                 │
│ Sesi: [Semua ▼]  RSVP: [Semua ▼]  Kirim: [Semua ▼]  🔍 [ nama, no meja, no WA … ]  [Cari]             │
│ [ + Tambah Tamu ]   [ 📥 Impor Excel/CSV ]   [ 📤 Ekspor ]                                            │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Nama                        Pihak Kategori  Sesi    Meja   Pax Undangan          RSVP      Aksi   │
│  1 Bpk. Prof. Bambang S.       Pria  VIP       Resepsi VIP-01  2  ✓ Terkirim        🟢 Hadir  💬 ✏️ ⋮│
│  2 Raditya Pratama & Pasangan  Pria  Teman     Resepsi T-04    2  ✓ Terkirim        🟢 Hadir  💬 ✏️ ⋮│
│  3 Bpk. Hendro & Keluarga      Pria  Keluarga  Umum    FAM-01  4  ✓ Terkirim        🟢 Hadir  💬 ✏️ ⋮│
│  4 Satria Wibawa (Kolega)      Pria  Kolega    Resepsi T-10    2  ✓ Terkirim        🟡 Ragu   💬 ✏️ ⋮│
│  5 Bpk. Joko Susilo (RT 04)    Pria  Tetangga  Resepsi T-08    2  ✓ Terkirim        🔴 Tidak  💬 ✏️ ⋮│
│ … halaman 1 dari 1 (100 per halaman)                                                                  │
│ TOTAL 20 tamu                                              43 pax   14 terkirim     12 hadir (27 pax) │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Memilih **Pihak Pria (CPP)** menggeser penghitung kategori ke tabel pihak itu (seperti dua tabel PDF Hal. 13), dan tabel hanya menampilkan 10 tamunya:

```text
│ Pihak:     [Semua 20] [★ Pihak Pria (CPP) 10] [Pihak Wanita (CPW) 10]                                │
│ Kategori:  [Teman 3] [Kolega 2] [Keluarga 2] [Tetangga 1] [Teman Orang Tua 1] [VIP 1]                 │
```

Kolom Undangan menampilkan tombol **Salin tautan** dan **Kirim WA** bila `personalized_url` ada (undangan terhubung); pada Standalone hanya **Kirim WA** (pesan tanpa tautan). Menu ⋮ per baris: *Tandai terkirim* / *Tandai belum terkirim*, *Salin tautan* (bila ada), *Hapus*. Baris "TOTAL" memakai angka `overview` (seluruh proyek), bukan halaman; pada halaman terfilter ia berlabel "TOTAL PROYEK".

### 7.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu 1 `GuestKpiCards` | `overview.total`, `total_pax` | "{total} · {pax} pax"; judul "Undangan"/"Pendaftaran" menurut `invitationNoun`; `total = 0` → "Belum ada {tamu}" |
| Kartu 2 | `overview.sides` | satu baris per pihak registry "{label} {n} · {pax} pax" (+ "Belum ditentukan" bila ada); **kartu tidak dirender** untuk jenis tanpa sisi |
| Kartu 3 | `kpiView().sent`, `unsent`, `sentShare` | "{sent} dari {total}" + bar `barWidth` + `formatPercent(p, 0)`; `sentShare = null` → "–" |
| Kartu 4 | `kpiView().rsvp` | empat status selalu tampil dengan warna dan teks; `attendingPax` di samping "Hadir"; jumlah empat status = total undangan |
| `SideSwitcher` | `sidePills(overview)`, `query.side` | tidak dirender bila daftar kosong; pil aktif = `query.side` (`''` = Semua, `none` = Belum ditentukan); tautan `guestHref(base, query, { side })` |
| `CategoryCounters` | `categoryPills(overview, query.side)` | enam kategori PDF untuk pernikahan; hitungan mengikuti pihak terpilih; kategori kustom/`lainnya` hanya bila terpakai; pil aktif = `query.category` |
| `GuestFilters` | `sessions`, `query` | dropdown sesi (master sesi; "Umum" untuk `all`; sesi usang tidak ada), RSVP (4 status), Kirim (terkirim/belum), pencarian |
| `GuestTable` | `guests`, `guestNo(offset, i)` | kolom Pihak dan Kategori memakai label registry (`categoryPills` label; kategori kustom apa adanya; sesi usang → Umum); catatan tamu sebagai ikon dengan tooltip |
| Kolom Undangan | `invitation_sent_status`, `personalized_url`, `formatted_wa_number` | lencana "Terkirim"/"Belum"; tombol WA nonaktif "Nomor belum diisi" tanpa nomor; `guestWaLink(g, ctx)` dengan `ctx = messageContext(project, session tamu)` |
| Kolom RSVP | `rsvp_status`, `RSVP_LABELS`, `RSVP_TONES`, `is_checked_in` | lencana + dropdown ubah (Hadir/Ragu/Tidak hadir/Belum merespons); `canEditRSVP` salah (sudah check-in) → dropdown nonaktif dengan "Sudah hadir 11.14 (batalkan check-in di Meja Resepsionis)"; `isDeclinedArrival` → "Menolak, tetap hadir" |
| `GuestPager` | `total`, `limit`, `offset`, `pageCount` | "halaman {page} dari {n} ({limit} per halaman)"; tautan sebelumnya/berikutnya; disembunyikan bila satu halaman |
| `StandaloneBanner` | `project.hasInvitation` | tampil bila `false`; tanpa tombol |
| Keadaan kosong | `overview.total = 0` | "Belum ada tamu. Tambahkan tamu pertama atau impor dari Excel." (dengan dua tombol) |
| Hasil filter kosong | `guests = []`, `overview.total > 0` | "Tidak ada tamu yang cocok." + tautan "Hapus filter" |

### 7.3 Kontrol

- **`+ Tambah Tamu`** membuka `GuestModal`; sesi bawaan Umum. **`📥 Impor Excel/CSV`** membuka `GuestImportModal`. **`📤 Ekspor`** = `exportGuestsHref` (daftar lengkap seluruh proyek, bukan halaman terfilter). Tombol tulis nonaktif bila `project.archived` (keterangan "Acara diarsipkan").
- **Pil dan filter** adalah navigasi penuh (server memfilter dan memotong halaman); `GuestFilters` adalah `<form method="get">` yang menyimpan `side`/`category`/`tab` sebagai kolom tersembunyi.
- **Menu ⋮ per baris**: *Ubah*, *Tandai terkirim/belum*, *Salin tautan* (bila ada), *Hapus*. **Dropdown RSVP** langsung memanggil `setRSVP`.
- **Muat ulang** (tombol di atas tabel): memuat ulang area lewat partial. Polling 30 detik hanya menyegarkan kartu dan penghitung; bila angka berubah sedangkan tabel belum dimuat ulang, tampil "Ada perubahan data. Muat ulang" di atas tabel (agar baris tidak melompat).

### 7.4 Modal dan dialog

1. **`GuestModal`** (tambah dan ubah; `PUT` mengganti penuh, jadi formulir mengisi awal **semua** kolom dari baris lewat `formFromGuest`): nama\* (≤ 255), pihak (radio dari `guest_sides` registry + "Belum ditentukan"; **tidak dirender** untuk jenis tanpa sisi), kategori (dropdown: kategori registry, "Lainnya", dan "Kategori baru…" dengan isian teks ≤ 60), sesi (master sesi, termasuk Umum), nomor meja (≤ 30), pax (1–20, bawaan 1; stepper), nomor ponsel (≤ 40; pratinjau "Dikirim sebagai +62…" dari `normalizeWA`; `null` → galat kolom tanpa permintaan), catatan (≤ 2.000; alergi, kursi roda). Galat server (`invalid_phone`, alasan per kolom) tampil di bawah kolom.
2. **`GuestImportModal`** (tiga langkah dalam satu dialog): (1) *Unduh templat* (`importTemplateHref`) dan pilih berkas; `fileProblem` memeriksa ekstensi/ukuran sebelum membaca. (2) *Pratinjau* (`dry_run`): ringkasan "{valid} dari {total} baris siap diimpor", daftar galat per baris ("Baris 3 · Nomor ponsel: …" lewat `guestFieldMessage`), duplikat ("Baris 5 sama dengan tamu yang sudah ada", "Baris 6 sama dengan baris 2"), dan lima baris pertama; kotak centang **Lewati baris galat** (`skipInvalid`) dan **Lewati duplikat** (`skipDuplicates`); tombol *Impor {willCreate} tamu* aktif menurut `importDecision`. (3) *Hasil*: "{created} tamu ditambahkan, {skipped} dilewati", lalu muat ulang area. `409 guest_limit_reached`, `400 invalid_file`, `413` tampil sebagai banner di langkah 1–2. Petunjuk tetap di dialog: "Excel: simpan nomor sebagai teks (templat sudah begitu). Nomor yang hilang angka 0-nya akan ditolak."
3. **`ConfirmDialog`**: hapus tamu ("{nama} dihapus dari daftar."), tandai belum terkirim tidak perlu konfirmasi.

### 7.5 Alur interaksi (Tab 1)

| Alur | Langkah | Panggilan |
|---|---|---|
| A. Filter | klik pil / kirim form → navigasi penuh dengan URL baru (halaman 1) | `listGuests` (SSR) |
| B. Tambah | modal → `validateGuestForm` → `createGuest` → partial; galat kolom di formulir | `createGuest` |
| C. Ubah | modal terisi (`formFromGuest`) → `updateGuest` (semua kolom) → partial | `updateGuest` |
| D. Hapus | konfirmasi → `deleteGuest`; `404` dianggap sukses → partial | `deleteGuest` |
| E. Ubah RSVP | dropdown → `setRSVP({ rsvp_status, pax_actual: 0 })` (0 = ikut kuota `pax_count`; jumlah yang benar-benar datang dicatat saat check-in) → partial; `409 already_checked_in` → toast "Batalkan check-in dulu" | `setRSVP` |
| F. Tandai terkirim | menu → `setSentStatus(id, 'sent'\|'unsent')` → partial. Membuka WhatsApp **tidak** menandai terkirim | `setSentStatus` |
| G. WhatsApp | `guestWaLink(guest, messageContext(project, sesiTamu))` di tab baru (`rel="noopener noreferrer"`); tanpa nomor: tombol nonaktif | – |
| H. Salin tautan | `navigator.clipboard.writeText(guest.personalized_url)` → toast "Tautan disalin"; peramban tanpa clipboard: tampilkan tautan di kotak yang bisa dipilih | – |
| I. Impor | modal: pilih berkas → `fileProblem` → `file.arrayBuffer()` → `toBase64` → `importGuests(body, { dryRun: true })` → pratinjau → konfirmasi → `importGuests(body, { skipInvalid, skipDuplicates })` → partial | `importGuests` |
| J. Ekspor / templat | tautan | `exportGuestsHref`, `importTemplateHref` |
| K. Polling | tiap 30 dtk: `live?part=summary` → ganti `[data-guest-summary]` (kartu + pil + penghitung); tabel tidak diganti | partial |
| L. Proyeksi untuk Menu 03 | modal pos (03): `cateringProjection()` → `projectionText(p)`; tombol "Gunakan proyeksi RSVP" aktif bila `canUseProjection(p)`; klik mengisi `quantity = p.recommended_portions`, satuan "porsi", mode *Jumlah × harga satuan*; **simpan memakai `PUT` pos dengan semua isian termasuk `vendor_id`** (03 FE; `PUT` tanpa `vendor_id` melepas vendor) | `cateringProjection` (+ `updateLine` 03) |

#### 7.5.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Dipakai untuk |
|---|---|---|
| Baris / kartu tamu | `data-guest-id`, `data-sent`, `data-rsvp`, `data-checked-in`, `data-focus-key` | menu aksi, dropdown RSVP, fokus setelah partial |
| Tombol WA | `data-wa-href` (hasil `guestWaLink`, dihitung server) | alur G tanpa merakit URL di klien |
| Tombol salin | `data-copy` (`personalized_url`) | alur H |
| Blok data | `<script type="application/json" data-guest-data>` | daftar tamu halaman ini untuk modal ubah; `<` di-escape `<` dan dibaca dengan `JSON.parse(textContent)` |
| Area | `data-guest-live`, `data-guest-summary` | alur K dan penggantian partial |

### 7.6 Salinan teks

| Tempat | Teks |
|---|---|
| Keadaan kosong | "Belum ada tamu. Tambahkan tamu pertama atau impor dari Excel." (seminar: "Belum ada peserta. …") |
| Tombol | "+ Tambah Tamu" · "Impor Excel/CSV" · "Ekspor" · "Muat ulang" (seminar: "Tamu" → "Peserta") |
| RSVP | "Hadir" · "Ragu" · "Tidak hadir" · "Belum merespons" |
| Kirim | "Terkirim" · "Belum" · "Tandai terkirim" · "Tandai belum terkirim" |
| WhatsApp | "Kirim WA" · "Nomor belum diisi" |
| Standalone | `STANDALONE_BANNER_TEXT` (bab 9.4) |
| Data berubah | "Ada perubahan data. Muat ulang" |
| Hasil filter kosong | "Tidak ada tamu yang cocok." |

---

## 8. Tab 2: Meja Resepsionis & Check-in Hari-H

Antarmuka cepat dan ramah sentuh untuk tablet/laptop panitia. Segar lewat polling 5 detik (bukan SSE/WebSocket).

### 8.1 Wireframe (ilustrasi hari-H)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌───────────────────┐ ┌───────────────────┐ ┌──────────────────────────┐ ┌───────────────────────┐  │
│ │ 👥 Tamu Check-in  │ │ 🚶 Pax di Lokasi  │ │ 🎁 Souvenir              │ │ ⏳ Belum Hadir        │  │
│ │ 11 dari 18        │ │ 22 dari 39 pax    │ │ 11 diserahkan            │ │ 7 tamu                │  │
│ │ [██████░░░] 61%   │ │ [██████░░░] 56%   │ │ sisa 29 dari stok 40     │ │                       │  │
│ └───────────────────┘ └───────────────────┘ └──────────────────────────┘ └───────────────────────┘  │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│  [ 📷 SCAN QR TIKET ]                         [ + TAMU WALK-IN ]                    [ 📤 Rekap ]     │
│  🔎 [ ketik nama, meja (VIP-01), atau no WA … ]             Segar otomatis tiap 5 detik · [Muat ulang]│
│  [ Semua 18 ] [ Sudah hadir 11 ] [ Belum hadir 7 ]   Kategori: [ Semua ▼ ]                           │
│  Terbaru: 12.16 Citra Lestari · 12.11 Melisa Putri · 12.07 Rahmat Hidayat · 12.03 Fajar Nugraha …    │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Jam    Nama                           Pihak  Kategori   Meja    Pax  Souvenir  Status     Aksi        │
│ --.--  Anisa Rahmawati (Sahabat SMA)  Wanita Teman      T-01     2   –         Belum      [ ✓ Check-in ]│
│ 11.58  Bayu Anggoro & Partner         Pria   Teman      T-05     2   ✓         Hadir      [Batal]     │
│ --.--  Bpk. Agus Santoso (Paman CPW)  Wanita Keluarga   FAM-02   3   –         Belum      [ ✓ Check-in ]│
│ 11.31  Bpk. Dr. Ir. Gunawan (Sah.Ayah)Pria   Teman Orang Tua VIP-02 2  ✓       Hadir      [Batal]     │
│ --.--  Bpk. H. Iwan & Keluarga        Wanita Keluarga   FAM-02   4   –         Belum      [ ✓ Check-in ]│
│ 11.38  Bpk. Hendro & Keluarga         Pria   Keluarga   FAM-01   4   ✓         Hadir      [Batal]     │
│ 11.14  Bpk. Prof. Bambang S.          Pria   VIP        VIP-01   2   ✓         Hadir      [Batal]     │
│ … 18 tamu · halaman 1 dari 1 (50 per halaman)                                                         │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Antrean diurutkan menurut nama (chip *Sudah hadir*: yang terbaru dulu). Dua tamu yang menolak (Joko Susilo, Dedi Sukamto) tidak muncul di antrean maupun hitungan; mengetik namanya di kotak pencarian menampilkannya, dan check-in-nya tetap bisa dilakukan (tampil "Menolak, tetap hadir" dan dihitung hadir).

### 8.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu 1 `ReceptionKpiCards` | `kpi.checked_in_guests`, `expected_guests`, `receptionView().attendanceShare` | "{n} dari {m}" + bar + `formatPercent(p, 0)`; `expected = 0` → "–" |
| Kartu 2 | `kpi.actual_pax`, `expected_pax`, `paxShare` | "{a} dari {b} pax" |
| Kartu 3 | `kpi.souvenir_distributed`, `souvenir_stock`, `receptionView().souvenirRemaining/souvenirShort` | "{n} diserahkan"; bila `souvenir_stock` ada: "sisa {stok−n} dari stok {stok}"; `souvenirShort` → merah "Stok kurang {abs}"; tanpa stok: hanya jumlah diserahkan (satuan: per undangan) |
| Kartu 4 | `kpi.pending_guests` | "{n} tamu"; keterangan "(tidak termasuk yang menolak)" |
| Chip status | `statusChips(filters)`, `query.status` | Semua · Sudah hadir · Belum hadir dengan hitungan; tautan `receptionHref(base, query, { status })` |
| Dropdown kategori | `categoryChips(filters)` | "{label} ({n})" hanya tamu yang diharapkan; kategori kustom hanya bila ada |
| `RecentCheckins` | `recent_checkins` | sampai 10 nama terbaru dengan `clock(checked_in_at, timezone)` |
| `ReceptionTable` | `guests` | Jam (`clock`, "--.--" bila belum), Nama, Pihak (kolom hilang untuk jenis tanpa sisi), Kategori, Meja, Pax (`pax_actual` bila hadir, selain itu kuota), Souvenir (✓/tombol *Beri*), Status (Hadir / Belum / "Menolak, tetap hadir"), Aksi (*✓ Check-in* atau *Batal*) |
| Keadaan kosong | `total = 0` | tanpa pencarian: "Belum ada tamu. Tambahkan di tab Daftar Tamu atau gunakan Tamu Walk-in."; dengan pencarian: "Tidak ditemukan. Coba nama lain atau tambahkan sebagai Tamu Walk-in." |
| Tombol tulis | `project.archived` | nonaktif dengan keterangan "Acara diarsipkan" (check-in tidak dapat ditulis) |

### 8.3 Komponen dan modal

1. **`QrScanModal`** — `navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } })` + `new BarcodeDetector({ formats: ['qr_code'] })` bila tersedia (tanpa dependensi baru); pembacaan per bingkai lewat `requestAnimationFrame` dibatasi ±10 kali/detik. Isi QR diproses `extractTicketToken`: `null` → "Tiket tidak dikenali." tanpa permintaan; selain itu `scanQR(token)` → `QuickCheckinSheet`. Bila `BarcodeDetector` atau kamera tidak tersedia (mis. Safari iOS, izin ditolak, bukan HTTPS; dukungan peramban berubah dan belum diuji di perangkat): tampilkan `SCAN_UNSUPPORTED_TEXT` dan kotak ketik nama/nomor meja/kode tiket (kode 32 heksa juga dikenali `extractTicketToken`). Bunyi *beep* saat berhasil lewat `AudioContext` (osilator, tanpa berkas bunyi); tutup kamera (`track.stop()`) saat dialog ditutup.
2. **`QuickCheckinSheet`** — dibuka dari ketukan baris, hasil scan, atau hasil pencarian: nama (huruf besar), lencana kategori/pihak, kotak kuning nomor meja ("MEJA VIP-01"), stepper pax riil (bawaan `defaultCheckinPax`, 1–20), kotak centang souvenir (bawaan aktif bila stok ada/tidak diketahui), catatan tamu (hanya baca; alergi, kursi roda), tombol besar **KONFIRMASI HADIR & ARAHKAN**. Tamu yang sudah check-in (hasil scan) langsung menampilkan `alreadyInText` dengan *Batalkan check-in* dan *Tutup*. `409 already_checked_in` pada konfirmasi: `getGuest` lalu tampilan yang sama (tidak ada check-in ganda).
3. **`WalkInModal`** — nama\*, pihak/kategori (opsional; pihak tidak dirender untuk jenis tanpa sisi), pax datang (1–20), meja, souvenir; tamu langsung berstatus hadir (`rsvp_status = attending` di server).
4. **Ekspor rekap** `.xlsx` (`exportAttendanceHref`); batas 10/menit.
5. **Sentuh**: tombol Scan dan Walk-in tinggi 56 px, target sentuh minimum 48 px, baris longgar, umpan balik audio.

### 8.4 Pemeriksaan sebelum mengirim

Pax riil 1–20 (stepper tidak melewati batas); walk-in: nama wajib (`validateGuestForm` untuk nama/meja/pax); tidak ada pemeriksaan stok (stok kurang hanya peringatan di kartu). Tombol aksi dinonaktifkan selama permintaan berjalan (`aria-busy`).

### 8.5 Alur interaksi (Tab 2)

| Alur | Langkah | Panggilan |
|---|---|---|
| M. Scan | modal → `extractTicketToken` → `scanQR` → `QuickCheckinSheet` (atau "Tiket tidak dikenali") | `scanQR` |
| N. Check-in | lembar → `checkin({ pax_actual, souvenir_received })` → beep + toast "{nama} masuk · {meja}" → partial | `checkin` |
| O. Sudah masuk | `409 already_checked_in` → `getGuest` → "Sudah masuk pukul 11.14" + *Batalkan check-in* / *Tutup* | `getGuest` |
| P. Batal | konfirmasi ("Batalkan check-in {nama}? Tanda souvenir ikut dihapus.") → `cancelCheckin` → partial | `cancelCheckin` |
| Q. Souvenir | tombol *Beri* / ✓ di baris → `setSouvenir` → partial | `setSouvenir` |
| R. Walk-in | modal → `walkIn` → toast → partial; `409 guest_limit_reached` → banner | `walkIn` |
| S. Cari / filter | form `GET` (`q`), chip, dropdown kategori → navigasi penuh dengan URL baru | `receptionOverview` (SSR) |
| T. Polling | tiap 5 dtk: `live?tab=checkin&…` → ganti area; ditunda selagi lembar/dialog terbuka atau mengetik | partial |
| U. Rekap | tautan | `exportAttendanceHref` |

#### 8.5.1 Atribut data yang dibaca skrip klien

| Elemen | Atribut | Dipakai untuk |
|---|---|---|
| Baris / kartu antrean | `data-guest-id`, `data-checked-in`, `data-souvenir`, `data-pax`, `data-focus-key` | ketuk → lembar, tombol Batal/Beri, fokus setelah partial |
| Blok data | `<script type="application/json" data-guest-data>` | tamu halaman ini untuk lembar check-in tanpa permintaan tambahan |
| Area | `data-guest-live` | polling dan penggantian partial |

### 8.6 Salinan teks

| Tempat | Teks |
|---|---|
| Tombol | "Scan QR Tiket" · "+ Tamu Walk-in" · "✓ Check-in" · "Batal" · "Rekap" |
| Lembar | "KONFIRMASI HADIR & ARAHKAN" · "MEJA {nomor}" · `alreadyInText` |
| Menolak, tetap hadir | "Menolak, tetap hadir" |
| Pemindai | `SCAN_UNSUPPORTED_TEXT` · `SCAN_UNKNOWN_TEXT` |
| Stok kurang | "Stok kurang {n}" |
| Polling | "Segar otomatis tiap 5 detik" |

---

## 9. Pesan galat dan teks

Semua teks lewat `lib/planner/messages.ts` (01 FE bab 5.4 dan 11): kode lama tetap, berikut tambahan menu ini. **Pola pemakaian** sama dengan 02–04: fungsi Menu 05 mengembalikan `null` bila bukan miliknya, dan pemanggil jatuh ke fungsi Menu 04/03 lalu 01:

```ts
const text = guestErrorMessage(res.status, res.error?.code, 'guest') ?? vendorErrorMessage(res.status, res.error?.code) ?? plannerErrorMessage(res.status, res.error?.code);
const fieldText = guestFieldMessage(field, reason) ?? vendorFieldMessage(field, reason) ?? fieldErrorMessage(field, reason);
```

### 9.1 Kode → teks

| Kode / kondisi | Teks Indonesia | Fungsi |
|---|---|---|
| `guest_limit_reached` | "Jumlah tamu sudah mencapai batas (5.000). Hapus tamu yang tidak dipakai lalu coba lagi." | `guestErrorMessage` |
| `already_checked_in` | `ALREADY_CHECKED_IN_TEXT` — pada check-in **bukan galat tampil**: ambil ulang tamu dan tampilkan `alreadyInText`; pada ubah RSVP: "Tamu ini sudah check-in." (batalkan check-in dulu) | idem |
| `invalid_phone` | "Nomor ponsel tidak valid. Gunakan 08… atau +62…." | idem |
| `invalid_file` | "Berkas tidak dapat dibaca. Gunakan templat .xlsx atau simpan sebagai CSV." (alasan rinci: `guestFieldMessage('file', …)`) | idem |
| `file_too_large` | "Ukuran berkas melebihi 5 MB." | idem |
| `not_found`, `where = 'guest'` | "Tamu tidak ditemukan. Muat ulang daftar." | idem |
| `not_found`, `where = 'scan'` | `SCAN_UNKNOWN_TEXT` "Tiket tidak dikenali." | idem |
| `not_found`, `where = 'project'` | "Acara tidak ditemukan." (halaman: `/404`) | 01 |
| `rate_limited`, `where = 'export'` / `'import'` / `'scan'` | "Terlalu banyak ekspor." / "Terlalu banyak impor." / "Terlalu banyak pemindaian." + "Tunggu sebentar lalu coba lagi." | `guestErrorMessage` |
| `invalid_scope` | "Sesi tidak valid atau sudah dihapus. Muat ulang halaman." | 03 (`financeErrorMessage`) |
| `invalid_input`, `invalid_date` | **per kolom** (9.2); tanpa kolom: "Periksa kembali data Anda." | 01 |
| `invalid_json`, `project_read_only`, `conflict`, `401`, `5xx`, jaringan | sama dengan [01 FE bab 11.1](../01_DASHBOARD/FE.md#111-kode--teks-plannererrormessagestatus-code) | 01 |

`where` ditentukan oleh **aksi** yang sedang berjalan, bukan oleh pesan server.

### 9.2 Alasan per kolom

Alasan yang sudah ada di 01 FE bab 11.2 (wajib/`required`, `must not be empty`, panjang, karakter kontrol, `must be an integer`, `must be at least N`, `must be between A and B`, `must be one of …`) tetap berlaku. Tambahan Menu 05 (`guestFieldMessage`; **setiap alasan baku BE §3.4 sudah tercakup atau diserahkan eksplisit**, diuji):

| Pola alasan (regex) | Kolom | Teks (`{label}` = label kolom, 9.3) |
|---|---|---|
| `^must be an Indonesian phone number \(08…, \+62…\)$` | `phone_number` | "Nomor ponsel tidak valid. Gunakan 08… atau +62…." |
| `^must be a registry side key$` | `guest_side` | "Pihak tidak dikenal. Pilih dari daftar." |
| `^must be empty for an event type without sides$` | `guest_side` | "Jenis acara ini tidak memiliki pembagian pihak; kosongkan kolom Pihak." |
| `^must be a session of this project$` | `sub_event_scope` (baris impor) | "Sesi tidak ditemukan di acara ini." |
| `^(\d+) rows are invalid$` | `rows` | "{n} baris pada berkas tidak valid. Periksa pratinjau, perbaiki berkas, atau centang "lewati baris galat"." |
| `^must be one of unsent, sent$` | `sent_status` | "Status kirim harus "belum" atau "terkirim"." |
| `^missing column: Nama Tamu$` | `file` | "Kolom "Nama Tamu" tidak ditemukan pada baris judul. Gunakan templat dari tombol Unduh templat." |
| `^too many rows \(max (\d+)\)$` | `file` | "Berkas memuat lebih dari 2.000 baris. Bagi menjadi beberapa berkas." |
| `^empty file$` | `file` | "Berkas kosong." |
| `not a readable .xlsx file` · `the content is not an .xlsx file` · `the content is not a CSV file` · `not a text file` · `malformed CSV: …` · `not valid UTF-8 or Windows-1252` · `content_base64 is not valid base64` · `the workbook has no sheet` | `file`, `content_base64` | "Berkas tidak dapat dibaca. Gunakan templat .xlsx atau simpan sebagai CSV." |
| `^filename must end with \.csv or \.xlsx$` | `filename` | "Berkas harus berakhiran .csv atau .xlsx." |
| `required`, `must be at most N characters`, `must be an integer`, `must be between A and B` (`name`, `category`, `table_number`, `pax_count`, `pax_actual`, `notes`, `buffer_percent`, `buffer_pax`), `must be one of …` (`rsvp_status`) | – | diserahkan ke Menu 04/03/01 ("{label} …") |

Alasan galat **per baris impor** (`ImportResult.errors[]`, memakai kamus yang sama) ditampilkan "Baris {row} · {label}: {teks}".

### 9.3 Label kolom

`GUEST_FIELD_LABELS` (kode di 9.4): `name` "Nama", `guest_side` "Pihak", `category` "Kategori", `sub_event_scope` "Sesi", `table_number` "Nomor meja", `pax_count` "Jumlah pax", `phone_number` "Nomor ponsel", `notes` "Catatan", `pax_actual` "Pax yang hadir", `rsvp_status` "Status RSVP", `sent_status` "Status kirim", `qr_code_token` "Kode tiket", `souvenir_received` "Souvenir", `buffer_percent` "Cadangan (%)", `buffer_pax` "Cadangan (porsi)", `filename` "Nama berkas", `content_base64` "Isi berkas", `rows` "Baris", `file` "Berkas". Kolom tak dikenal: "Isian".

### 9.4 Kode `messages.ts` (tambahan)

Teks tetap, label kolom, dan dua fungsi galat. Istilah RSVP, label status, dan teks tampilan lain ada di `guest.ts` (bab 5) karena murni dan diuji bersamanya. **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-guesttestmjs)).

```ts
// src/lib/planner/messages.ts (tambahan; berkas acuan guestMessages.ts)
// ---------- Tambahan Menu 05 untuk messages.ts (pola: pemanggil memakai `…Message(…) ?? kamus Menu 04/03/01`) ----------

/** Aksi yang memanggil: menentukan teks `not_found` dan `rate_limited`, yang tidak dapat dibedakan dari kode/pesan server. */
export type GuestWhere = 'project' | 'guest' | 'scan' | 'export' | 'import';

export const ALREADY_CHECKED_IN_TEXT = 'Tamu ini sudah check-in.';
export const SCAN_UNSUPPORTED_TEXT = 'Pemindai tidak didukung di perangkat ini. Ketik nama atau nomor meja untuk mencari tamu.';
export const SCAN_UNKNOWN_TEXT = 'Tiket tidak dikenali.';
export const STANDALONE_BANNER_TEXT = 'Mode Standalone: kirim undangan cetak atau pesan WhatsApp biasa dan ubah status RSVP secara manual. Undangan Digital Adicara (tautan personal dan RSVP otomatis) belum tersedia.';

export const GUEST_FIELD_LABELS: Record<string, string> = {
  name: 'Nama', guest_side: 'Pihak', category: 'Kategori', sub_event_scope: 'Sesi', table_number: 'Nomor meja', pax_count: 'Jumlah pax',
  phone_number: 'Nomor ponsel', notes: 'Catatan', pax_actual: 'Pax yang hadir', rsvp_status: 'Status RSVP', sent_status: 'Status kirim',
  qr_code_token: 'Kode tiket', souvenir_received: 'Souvenir', buffer_percent: 'Cadangan (%)', buffer_pax: 'Cadangan (porsi)',
  filename: 'Nama berkas', content_base64: 'Isi berkas', rows: 'Baris', file: 'Berkas',
};

type Rule = readonly [RegExp, (m: RegExpExecArray, label: string, field: string) => string | null]; // null = lanjut ke aturan berikutnya

/** Alasan `<kolom>: <alasan>` milik Menu 05, juga alasan per baris impor. Urutan penting: pola khusus lebih dulu. */
const GUEST_REASONS: readonly Rule[] = [
  [/^must be an Indonesian phone number \(08…, \+62…\)$/, () => 'Nomor ponsel tidak valid. Gunakan 08… atau +62….'],
  [/^must be a registry side key$/, () => 'Pihak tidak dikenal. Pilih dari daftar.'],
  [/^must be empty for an event type without sides$/, () => 'Jenis acara ini tidak memiliki pembagian pihak; kosongkan kolom Pihak.'],
  [/^must be a session of this project$/, () => 'Sesi tidak ditemukan di acara ini.'],
  [/^(\d+) rows are invalid$/, (m) => `${m[1]} baris pada berkas tidak valid. Periksa pratinjau, perbaiki berkas, atau centang "lewati baris galat".`],
  [/^must be one of unsent, sent$/, () => 'Status kirim harus "belum" atau "terkirim".'],
  [/^missing column: Nama Tamu$/, () => 'Kolom "Nama Tamu" tidak ditemukan pada baris judul. Gunakan templat dari tombol Unduh templat.'],
  [/^too many rows \(max (\d+)\)$/, (m) => `Berkas memuat lebih dari ${Number(m[1]).toLocaleString('id-ID')} baris. Bagi menjadi beberapa berkas.`],
  [/^empty file$/, () => 'Berkas kosong.'],
  [/^(not a readable \.xlsx file|the content is not an \.xlsx file|the content is not a CSV file|not a text file|malformed CSV: .*|not valid UTF-8 or Windows-1252|content_base64 is not valid base64|the workbook has no sheet)$/,
    () => 'Berkas tidak dapat dibaca. Gunakan templat .xlsx atau simpan sebagai CSV.'],
  [/^filename must end with \.csv or \.xlsx$/, () => 'Berkas harus berakhiran .csv atau .xlsx.'],
];

/** null = bukan alasan Menu 05: pakai vendorFieldMessage / financeFieldMessage lalu fieldErrorMessage 01. */
export function guestFieldMessage(field: string, reason: string): string | null {
  const label = GUEST_FIELD_LABELS[field] ?? 'Isian';
  for (const [re, text] of GUEST_REASONS) {
    const m = re.exec(reason);
    const out = m ? text(m, label, field) : null;
    if (out !== null) return out;
  }
  return null;
}

/**
 * Kode galat Menu 05 yang teksnya bergantung konteks. null = pakai vendorErrorMessage / financeErrorMessage / plannerErrorMessage.
 * `already_checked_in` bukan galat tampil: lembar check-in mengambil ulang tamu dan menampilkan jam masuk (`alreadyInText`).
 * `invalid_file` membawa pesan `file: <alasan>` yang dipetakan `guestFieldMessage`.
 */
export function guestErrorMessage(status: number, code: string | undefined, where?: GuestWhere): string | null {
  if (status === 429 || code === 'rate_limited') {
    switch (where) {
      case 'export': return 'Terlalu banyak ekspor. Tunggu sebentar lalu coba lagi.';
      case 'import': return 'Terlalu banyak impor. Tunggu sebentar lalu coba lagi.';
      case 'scan': return 'Terlalu banyak pemindaian. Tunggu sebentar lalu coba lagi.';
      default: return null;
    }
  }
  switch (code) {
    case 'guest_limit_reached': return 'Jumlah tamu sudah mencapai batas (5.000). Hapus tamu yang tidak dipakai lalu coba lagi.';
    case 'already_checked_in': return ALREADY_CHECKED_IN_TEXT;
    case 'invalid_phone': return 'Nomor ponsel tidak valid. Gunakan 08… atau +62….';
    case 'invalid_file': return 'Berkas tidak dapat dibaca. Gunakan templat .xlsx atau simpan sebagai CSV.';
    case 'file_too_large': return 'Ukuran berkas melebihi 5 MB.';
    case 'not_found':
      switch (where) {
        case 'guest': return 'Tamu tidak ditemukan. Muat ulang daftar.';
        case 'scan': return SCAN_UNKNOWN_TEXT;
        default: return null;
      }
    default: return null;
  }
}
```

---

## 10. Gaya, responsif, aksesibilitas, keamanan

### 10.1 Gaya

Hanya token yang ada ([01 FE bab 12.1](../01_DASHBOARD/FE.md#121-gaya)): kartu `--color-surface`, batas `--color-border`; judul `--font-display`, isi `--font-body`; lencana mengikuti pola `.status` di `dashboard/index.astro`. Status RSVP: *Hadir* terisi `--sage-600`, *Ragu* kontur `--sage-400` dengan teks, *Tidak hadir* kontur `#8e2f25` (warna bahaya yang sudah dipakai kode) dengan teks, *Belum merespons* hanya kontur `--color-border`; selalu disertai teks, bukan hanya warna. Kotak nomor meja pada lembar check-in memakai `--sage-100` berbingkai `--sage-600` dengan huruf besar tebal (versi awal menyebut "kotak kuning", yang tidak ada di token; **jangan menambah token baru tanpa persetujuan**). Peringatan stok kurang dan "Menolak, tetap hadir" memakai `#8e2f25`. Banner Standalone `--sage-50` dengan batas `--sage-200`.

### 10.2 Responsif

- `< 768px`: KPI grid 2×2; tabel tamu menjadi *Guest Cards* (nama + lencana kategori/pihak, sesi/meja/pax, status RSVP, tombol Kirim WA / Salin tautan / Tandai terkirim sesuai mode); tabel meja menjadi kartu besar yang dapat diketuk; dialog penuh layar; pil dapat digulir horizontal dalam wadahnya.
- **Meja resepsionis dioptimalkan tablet landscape 1024 px**: tombol Scan dan Walk-in tinggi 56 px, target sentuh minimal 48 px (lainnya 44), baris longgar.
- Tidak ada gulir horizontal pada lebar 360 px kecuali di dalam wadah pil; uji dengan nama tamu 255 karakter, catatan 2.000 karakter, dan nomor meja 30 karakter (`overflow-wrap: anywhere`).

### 10.3 Aksesibilitas

- Dialog `<dialog>` dengan `showModal()`; fokus kembali ke pemicu saat ditutup; formulir: `<label>`, `aria-invalid`, `aria-describedby`, ringkasan galat `role="alert"`.
- Tab bar `role="tablist"` dengan `aria-selected`; pil `aria-current="true"` pada yang aktif; tabel memakai `<table>` dengan `<th scope>`.
- Status RSVP, kehadiran, dan stok kurang punya teks, bukan hanya warna atau ikon. Hasil asinkron (check-in berhasil, hasil impor, "Data belum diperbarui", beep) di wilayah `aria-live="polite"`; baris yang sedang dikirim `aria-busy="true"`.
- Pemindai: selain kamera selalu ada jalur papan ketik (ketik nama/meja/kode); bunyi beep disertai teks.
- Partial mengganti DOM (terutama polling 5 dtk): fokus dipulihkan lewat `data-focus-key`; polling ditunda selagi pengguna mengetik atau lembar terbuka; bila elemennya hilang, ke judul area.
- Kontras dan indikator fokus terlihat; hormati `prefers-reduced-motion`.

### 10.4 Keamanan

- **Tidak ada HTML dari data:** nama tamu, catatan, nomor meja, kategori kustom, dan nama sesi dirender sebagai teks (`textContent`/`createElement`; Astro meng-escape di server). Blok JSON `data-guest-data` ditulis dengan `<` di-escape (`<`); tidak ada `set:html` untuk data API.
- **Token tiket** (`qr_code_token`) tidak dirender, tidak ada di URL halaman, dan **dibuang dari blok `data-guest-data`**; ia hanya ada di ekspor daftar tamu (untuk pemilik). Tautan personal hanya dirender sebagai teks/nilai salin dan tombol WA; bila dijadikan `href` harus lolos `safeHref`.
- **Tautan WhatsApp** dibentuk dari `formatted_wa_number` (hanya digit, hasil normalisasi server) dan `encodeURIComponent` pada teks; nomor mentah pengguna tidak pernah masuk ke URL. Dibuka dengan `rel="noopener noreferrer"` dan `target="_blank"`.
- Path dibangun hanya dari `id` yang sudah divalidasi UUID; query lewat `URLSearchParams`.
- CSRF: BFF menolak non-GET tanpa `application/json` (impor tetap JSON). Token tetap di cookie HttpOnly. Tidak ada `localStorage` atau `sessionStorage`.
- **Kamera:** `getUserMedia` butuh HTTPS (atau `localhost`) **dan** header `Permissions-Policy` yang mengizinkannya. `deploy/nginx/adicara.conf` saat ini mengirim `Permissions-Policy: camera=(), microphone=(), geolocation=()` untuk seluruh situs, yang **memblokir kamera di produksi**; harus diubah menjadi `camera=(self)` (mikrofon dan geolokasi tetap kosong) sebelum pemindai dapat dipakai ([bab 13 butir 5](#13-celah-dan-keputusan-yang-menunggu-pemilik-produk)). CSP yang ada (`default-src 'self'`, `connect-src 'self'`) tidak menghalangi pemindai atau `AudioContext`.
- Data tamu bersifat pribadi: respons ekspor `Cache-Control: no-store`; tidak ada nama atau nomor telepon di URL halaman selain pencarian `q` yang diketik pengguna sendiri; ekspor daftar memuat kode tiket sehingga hanya untuk pemilik proyek.

---

## 11. Pengujian

Pola yang sudah ada: `node:test` di `frontend/tests/*.test.mjs`, dijalankan `npm test` (`node --experimental-strip-types --test tests/*.test.mjs`). Fixture di `frontend/tests/fixtures/` (bab 4.3).

### 11.1 Unit: `tests/planner-guest.test.mjs`

28 tes di bawah **sudah dijalankan**: kode bab 1.4, 4.2, 5, 6, 9.4 dikompilasi bersama `types.ts`, `format.ts`, `finance.ts`, dan `vendor.ts` yang diekstrak dari 01–04 FE, dan fixture bab 4.3. Hasil: `tsc` strict (`noUncheckedIndexedAccess`) bersih, 28/28 lulus. Mutasi yang dicoba pada kode (**24 dari 24 menjatuhkan tes**; satu mutan pertama yang selamat memperkuat satu tes, yaitu pil pihak "belum ditentukan"): kategori kustom 0 tetap tampil; pil kategori tidak mengikuti pihak; pihak belum ditentukan; jenis tanpa sisi; jam sesi Umum; sesi Umum disebut; tautan WA selalu memakai tautan personal; zona waktu jam masuk; stok tepat habis dianggap kurang; impor tanpa baris dapat dikirim; filter tidak kembali ke halaman 1; batas 5 MB bergeser; pax tanpa batas atas; pax awal check-in mengabaikan RSVP; potongan base64 diperbesar; proyeksi nol dianggap bisa dipakai; pesan "tiket tidak dikenali" dihapus; `skip_duplicates` tidak dikirim; batal check-in memakai `POST`; persen kehadiran memakai penyebut salah; token tiket tidak dijadikan huruf kecil; token 33 heksa diterima; status meja bebas; filter meja tidak kembali ke halaman 1.

| Tes | Masukan → keluaran (fixture Proyek Contoh) |
|---|---|
| `kpiView` | 20 undangan · 43 pax; terkirim 14 (70%), belum 6; RSVP berurutan hadir/ragu/tidak/belum `12/2/2/4` berjumlah total; 27 pax hadir |
| `sidePills` | Semua 20 · Pihak Pria (CPP) 10 · Pihak Wanita (CPW) 10; jenis tanpa sisi → `[]`; pihak `key: null` → pil `none` |
| `categoryPills` | Semua `6/3/4/2/3/2` (urutan registry, "Teman Orang Tua"); pihak pria `3/2/2/1/1/1`, wanita `3/1/2/1/2/1` (jumlah 10); kategori kustom dan `lainnya` hanya bila terpakai; sel pihak `none` |
| `messageContext`, `waMessage`, `guestWaLink` | pesan lengkap dengan tautan, tanpa tautan (Standalone), tanpa sesi/tempat; sesi Umum tidak disebut; tempat dari sesi lebih dulu, + kota; jam tanpa selesai; `https://wa.me/6281200000101?text=…` ter-decode sama dengan pesan; tanpa nomor → `null` |
| `clock`, `alreadyInText` | `04:14Z` → "11.14" (Asia/Jakarta), "04.14" (UTC), lewat tengah malam; "Sudah masuk pukul 11.14" |
| fixture hari-H | 11 dari 18 (61%), 22 dari 39 pax, 11 souvenir sisa 29, 7 belum hadir; chip `18/11/7`; 10 check-in terbaru; enam chip kategori; 18 tamu antrean |
| `receptionView` | stok hilang → `null`; stok kurang → negatif dan `souvenirShort`; tepat habis tidak kurang; pembagi nol → `null` |
| `isDeclinedArrival`, `canEditRSVP`, `defaultCheckinPax` | tamu menolak yang hadir; RSVP terkunci sesudah check-in; pax awal 4 / 3 / terpotong 20 |
| `canUseProjection`, `projectionText` | 34 porsi aktif; 0 porsi nonaktif; empat teks (dengan/tanpa cadangan, tanpa tamu hadir, hanya cadangan) |
| `importDecision`, `fileProblem`, `toBase64` | empat kombinasi `skipInvalid`/`skipDuplicates` (`willCreate` 4/4/2); tanpa baris tersisa tidak aktif; ekstensi, kosong, 5 MB tepat lolos, 5 MB + 1 ditolak; `toBase64` 5 MB sama dengan `Buffer`; ukuran base64 di bawah batas BFF |
| `validateGuestForm`, `guestBody`, `formFromGuest` | nama kosong/256/😀×255 (rune); kategori 61; meja 31; pax 0/21/2,5/kosong; telepon salah; catatan 2.001; **26 vektor `normalizeWA` Menu 04 sepakat dengan validasi telepon**; kolom kosong tidak dikirim; `formFromGuest` → `guestBody` bolak-balik |
| `parseGuestQuery`, `guestApiQuery`, `guestHref`, `pageCount` | nilai sah; nilai tidak sah jatuh ke bawaan (termasuk `page=2.5`, `q` 101 karakter, `scope=../`); filter mengembalikan ke halaman 1, halaman tetap bila filter sama; `offset = (page-1)×100` |
| `parseReceptionQuery`, `receptionHref`, `receptionApiQuery` | `status` hanya `in`/`pending`; filter kembali ke halaman 1; `offset` kelipatan 50 |
| `extractTicketToken` | token mentah, huruf besar, tautan personal (`?g=`), tautan huruf besar → token huruf kecil; kosong, 31/33 heksa, huruf non-heksa, URL tanpa token → `null` |
| `replaceGuest`, `removeGuest`, `invitationNoun` | penggantian satu baris; "pendaftaran" hanya untuk `Peserta` |
| `guestApi` | 18 panggilan ke-16 fungsi → metode + path + body + query persis (tabel 4.4); path lolos `^[a-z0-9/_-]+$`; id bukan UUID (`bukan-uuid`, `../../auth`, `x`, UUID huruf besar) melempar; `export`/`import-template`/`export-attendance` hanya `xlsx` |
| pesan | semua kode baru bertext; `not_found` per `where` (`scan` → "Tiket tidak dikenali"); `rate_limited` per `where` dan `null` untuk lainnya; 17 alasan Menu 05 punya teks, 6 alasan Menu 04/03/01 diserahkan (`null`); alasan galat per baris fixture impor semuanya bertext; label kolom |

### 11.2 Tes sumber (`tests/planner-wiring.test.mjs`, perluasan dari 01 FE bab 13.3)

- Halaman `tamu/index.astro` dan `live.astro` memuat `export const prerender = false`; `live.astro` memuat `export const partial = true`; keduanya memakai komponen yang sama (tidak ada markup tab di halaman).
- `index.astro` memakai `PlannerLayout`, bukan `AppLayout`.
- Tidak ada literal sisi, kategori, atau sesi tertulis di `components/planner/guest` dan halamannya: `'pria'`, `'wanita'`, `'teman'`, `'kolega'`, `'keluarga'`, `'vip'`, `'resepsi'`, `'akad'`, `'lamaran'` (hanya `types.ts` dan tes yang boleh). `'all'`, `'none'`, dan `FILTER_ALL` boleh.
- Tidak ada kata "Pria"/"Wanita"/"Mempelai" tertulis di komponen (label dari registry).
- Tidak ada `innerHTML =`, `insertAdjacentHTML`, atau `set:html` yang memuat data API di komponen dan `guestClient.ts`.
- Tidak ada `plannerFetch('projects/` atau `fetch('/dashboard/api/planner` di komponen: semua panggilan lewat `guestApi`.
- Tidak ada pembagian persen tertulis di komponen (`/ ... * 100`): semua lewat `percent`.
- Tidak ada `qr_code_token` di komponen kecuali `QrScanModal` dan `QuickCheckinSheet` (yang membaca token hasil pemindaian, bukan tamu); tidak ada di `data-guest-data`.
- Tidak ada `multipart`/`FormData` pada `guestClient.ts` (impor memakai JSON).
- Tidak ada `href=` yang memuat string pengguna selain lewat `safeHref`/`waLink`/`guestWaLink`.
- `PlannerLayout.astro` menandai `tamu` sebagai `ready`; `ROUTES.md` memuat rute halaman dan partial; BFF memuat batas badan khusus `bulk-import`.

### 11.3 Uji komponen dengan fixture (manual atau skrip)

Render tiap tab dengan: fixture penuh; proyek kosong; variasi bab 4.3 (jenis tanpa sisi, seminar, pihak belum ditentukan, kategori kustom, tanpa nomor, `personalized_url` ada, menolak tetapi hadir, stok hilang/kurang, proyeksi 0, hasil impor tanpa galat/semua galat/hanya duplikat, nama 255 karakter, sesi usang, halaman kedua); proyek `archived` (semua kontrol ubah dan check-in nonaktif, ekspor aktif); lebar 360 px dan tablet landscape 1024 px.

### 11.4 QA manual (terhadap backend asli)

**Belum bisa dijalankan** karena halamannya belum ada (backend sudah). Daftar yang harus lulus begitu halaman dibangun, dipetakan ke alur:

1. Proyek Contoh: kartu 20 · 43 pax, pihak 10/22 dan 10/21, terkirim 14 (70%), RSVP 12/2/2/4 dengan 27 pax; enam pil kategori `6/3/4/2/3/2`; memilih Pihak Pria menggeser penghitung ke `3/2/2/1/1/1`; banner Standalone tampil; tombol WA tampil untuk semua tamu bernomor (pesan tanpa tautan).
2. Tambah tamu: nomor `0812…`, `+62…`, `8123…` (galat kolom tanpa permintaan); kategori baru "Kolega Kantor" (kustom, ikut penghitung); pax 0 dan 21 ditolak; batas 5.000.
3. Ubah dan hapus; `PUT` tidak mengubah RSVP/check-in; dua tab: hapus di tab A, ubah di tab B → "Tamu tidak ditemukan. Muat ulang daftar."
4. RSVP manual lewat dropdown; ubah ke Hadir → penghitung dan kartu berubah setelah partial; tamu yang sudah check-in tidak dapat diubah.
5. Tandai terkirim / belum; membuka WhatsApp tidak mengubah status.
6. Jenis seminar: kata "Peserta" dan "pendaftaran"; pil "Panitia/Internal" dan "Peserta Eksternal", **tidak ada** "Pria/Wanita"; jenis ulang tahun: tanpa pemisah pihak dan tanpa kolom Pihak; modal tidak menawarkan pihak.
7. Impor: unduh templat (Excel Indonesia: simpan CSV UTF-8 → BOM + titik koma terbaca); pratinjau `planner-import-sample.csv`: 4 valid, 2 galat dengan teks Indonesia, 2 duplikat; centang lewati; hasil; berkas 5 MB + 1 → pesan ukuran; berkas `.xls` ditolak; 2.001 baris ditolak; angka telepon tanpa 0 ditolak; impor kedua dalam semenit sebanyak enam kali → pesan batas.
8. Ekspor daftar dan rekap (`.xlsx`); sel berawalan `=` tampil sebagai teks; kolom Kode Tiket ada.
9. Meja resepsionis: kartu hari-H; cari "bambang" → check-in satu ketukan → beep, jam "11.14" dalam zona proyek; ketukan ganda dari dua tablet → "Sudah masuk pukul …"; batal check-in → kembali ke Belum hadir dan souvenir terhapus.
10. Tamu yang menolak datang: tidak ada di antrean; mencari namanya menampilkannya; check-in berhasil dan tampil "Menolak, tetap hadir"; "Belum hadir" turun satu dan "Check-in" naik satu (tidak ada angka yang melewati 100%).
11. Scan QR: kamera (HTTPS dan `camera=(self)`), token mentah, tautan personal, tiket proyek lain → "Tiket tidak dikenali"; peramban tanpa `BarcodeDetector` → input manual.
12. Walk-in: tamu baru langsung hadir; pax 21 ditolak; terhitung di KPI.
13. Souvenir: stok dari Menu 04 (barang bertanda souvenir selesai) dikurangi diserahkan; melebihi stok → "Stok kurang".
14. Proyeksi dari modal pos Menu 03: 34 porsi (27 hadir + 25%); `PUT` pos mempertahankan vendor; tanpa tamu hadir tombol nonaktif; tamu `declined` tidak ikut.
15. Pindah sesi (hapus sesi dengan `move_to`): tamunya berpindah dan kolom Sesi mengikuti.
16. Arsipkan: kontrol ubah dan check-in nonaktif, ekspor aktif; tulis dari tab lain → `project_read_only`.
17. Polling: ubah RSVP dari tab lain → kartu Tab 1 berubah ≤ 30 dtk tanpa tabel melompat; Tab 2 ≤ 5 dtk; polling berhenti pada tab tersembunyi dan tidak menimpa kolom pencarian yang sedang diketik.
18. Sesi habis lalu klik aksi → diarahkan ke login. Mobile 360 px, papan ketik saja, pembaca layar ringkas.

### 11.5 Verifikasi kontrak (cara mengulang)

Dilakukan 2026-10-25 terhadap blok DTO Go di [BE bab 4](BE.md#4-dto-go) dan tipe bab 4.2 (skrip ada di scratchpad sesi, bukan di repo). Backend menu ini sudah ada, jadi langkah 6 menjalankan panggilan nyata seperti [02 FE bab 11.5](../02_TIMELINE_DAN_TUGAS/FE.md#115-verifikasi-kontrak-terhadap-backend-asli-cara-mengulang), dan sudah menjadi tes di backend.

1. **Kompilasi** blok Go dalam modul sementara (tipe Menu 01 identik dengan yang asli), `gofmt`, `go vet`.
2. **Ambil skema** dengan program `reflect` kecil: untuk tiap struct DTO, kunci JSON, `omitempty` (opsional), pointer tanpa `omitempty` (nullable), dan jenis nilai (`TestDumpShapes`, `DUMP_SHAPES=berkas.json`).
3. **Baca tipe TS** (`types.ts` yang dirakit dari 01–04 FE ditambah blok bab 4.2) dan bandingkan: kunci di satu sisi saja, opsionalitas, `null`, jenis nilai, jenis elemen larik. Tipe permintaan: kunci dan jenis saja.
4. **Uji diri pemeriksa**: ubah tipe TS dengan sengaja (kunci dihapus, opsional dijadikan wajib, jenis diganti, kunci berlebih, `| null` dihapus, jenis elemen larik diganti) dan pastikan dilaporkan. Enam mutasi dicoba; semuanya dilaporkan.
5. **Cocokkan fixture**: ke-4 fixture dibangkitkan dari tabel bab 6 oleh `TestFixturesAreCurrent`; vektor `NormalizeWA` diuji di Go dan TS dengan berkas yang sama (Menu 04).
6. **Bandingkan dengan backend asli** (dilakukan 2026-10-25, ada di repo): `TestResponsesMatchTheFEFixtures` membuat Proyek Contoh lewat router dan PostgreSQL asli, memanggil `GET guests`, `GET guests/catering-projection`, `POST guests/bulk-import?dry_run=true` (CSV fixture), dan `GET guests/reception-overview` (setelah 11 tamu ilustrasi dicatat hadir pada waktu fixture dan stok souvenir 40), lalu membandingkan dengan keempat fixture JSON setelah `id`/`qr_code_token`/`*_at` dinormalkan; kunci yang berlebih, kurang, atau bernilai lain dilaporkan dengan jalurnya. `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays` membaca tabel bab 4.6 dari dokumen ini dan membuat backend menghasilkan tiap pesan contoh (status, kode, dan pesan harus sama; baris tanpa skenario gagal). Keduanya diuji dapat gagal dengan mengubah satu nilai fixture dan satu pesan katalog. Menjalankan: `TEST_DATABASE_URL='postgres://…/adicara_planner_test?sslmode=disable' go test -p 1 -run 'MatchTheFEFixtures|ErrorCatalog' ./internal/modules/plannerguest/delivery/http/` dari `backend/`.

Hasil 2026-10-25: 21 pasangan tipe, 125 jalur, **0 selisih**. Respons HTTP asli sama dengan fixture dan katalog galat (lihat status di atas). Yang **tidak** terbukti: `5xx`, `409 conflict`, tampilan `.xlsx` di Excel/Numbers, kamera/`BarcodeDetector` pada perangkat, dan partial Astro di browser.

---

## 12. Urutan pengerjaan dan definisi selesai

### 12.1 Menjalankan lingkungan lokal

Halaman menu ini belum ada, jadi yang dapat dijalankan sekarang adalah pustaka acuan dan tesnya (backend-nya sudah dapat dipanggil):

```bash
cd frontend && npm install
npm test            # setelah tes dan fixture bab 4.3 / 11.1 disalin: node --experimental-strip-types --test tests/*.test.mjs
npx tsc --noEmit    # types.ts + guest.ts + guestApi.ts + guestMessages.ts + guestProps.ts di bawah konfigurasi strict repo
```

Setelah backend dan halaman dibangun: jalankan seperti [02 FE bab 12.1](../02_TIMELINE_DAN_TUGAS/FE.md#121-menjalankan-lingkungan-lokal) (`make run` untuk backend, `API_BASE_URL=http://localhost:8080 npm run dev` untuk frontend). Proyek Contoh dari `/dashboard/planner` ("Lihat Proyek Contoh") akan berisi 20 tamu setelah overlay `guests` terdaftar ([BE bab 7](BE.md#7-sambungan-ke-menu-01)). Pemindai kamera hanya berjalan di `localhost` atau HTTPS.

### 12.2 Tahapan

| # | Tahap | Isi | Selesai bila |
|---|---|---|---|
| T0 | Prasyarat | Menu 01 FE: M0 (tipe, `format`, `messages`), M1 (BFF + `client`), `PlannerLayout`, `404.astro`; `vendor.ts` Menu 04 (`normalizeWA`, `waLink`); **perubahan BFF** batas badan `bulk-import` (bab 1.3); **Nginx `camera=(self)`** (bab 10.4) | `npm test` 01 dan 04 hijau; `GET /dashboard/api/planner/event-types` membalas; header `Permissions-Policy` mengizinkan kamera |
| T1 | Fondasi menu ini | blok bab 4.2 ke `types.ts` (+ satu perubahan satu baris, bab 1.3), `guest.ts` (bab 5), `guestApi.ts` (bab 6.1), blok `messages.ts` (bab 9.4), `guestProps.ts` (1.4), `tests/fixtures/*` (4.3), tes 11.1 | `npm test` dan `tsc` hijau; belum perlu backend. Kode di dokumen ini tinggal disalin: sudah lulus kedua pemeriksaan itu |
| T2 | Tab 1 (baca) | `index.astro` + `live.astro`, `GuestTabs`, `GuestKpiCards`, `SideSwitcher`, `CategoryCounters`, `GuestFilters`, `GuestTable` + `GuestPager`, `StandaloneBanner`, tombol WA (alur A, G, H, K) | uji komponen 11.3 (Tab 1); QA 1, 6, 17, 18 |
| T3 | Tab 1 (tulis) | `GuestModal`, dropdown RSVP, tandai terkirim, hapus, `GuestImportModal`, `guestClient.ts` (alur B–F, I, J) | QA 2–5, 7, 8, 15 |
| T4 | Tab 2 (baca) | `ReceptionKpiCards`, `ReceptionToolbar`, `ReceptionTable`, `RecentCheckins`, polling 5 dtk (alur S, T, U) | uji komponen 11.3 (Tab 2) |
| T5 | Tab 2 (tulis) | `QuickCheckinSheet`, `QrScanModal`, `WalkInModal`, batal, souvenir (alur M–R) | QA 9–13, 16 |
| T6 | Sambungan | tombol proyeksi di modal pos Menu 03 (alur L), banner souvenir Menu 04 | QA 14 |
| T7 | Kualitas | aksesibilitas, responsif (360 px dan tablet 1024 px), `astro check`, `npm run build`, `format:check`, tes sumber 11.2 | build bersih; QA 18 |

T2–T6 membutuhkan backend menu ini ([BE bab 9](BE.md#9-urutan-pengerjaan-dan-definisi-selesai)); sampai ada, kerjakan dengan fixture. Karena Menu 05 membaca `GuestSummary` yang sudah ada di dashboard, widget Buku Tamu (01 FE bab 9.4) tidak ikut dikerjakan di sini.

**Definisi selesai (seluruh Menu 05 FE):** semua tahap lulus; `npm test`, `npm run check`, `npm run build`, `npm run format:check` bersih; tes sumber 11.2 lulus; verifikasi kontrak 11.5 diulang terhadap backend asli (nol selisih); `tamu.ready = true` di `PlannerLayout`; tombol proyeksi di modal pos Menu 03 berfungsi; `ROUTES.md`, `docs/frontend/planner/` dan dokumen ini diperbarui bila perilaku berubah.

---

## 13. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–04 (kode dan dokumen), kontrak, PDF, dan Nginx. Dokumen mengikuti pilihan yang tertulis; keputusan sisi backend ada di [BE bab 10](BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk).

1. **Kontrak sudah dibuktikan terhadap HTTP asli untuk respons baca dan galat** (fixture dan katalog bab 4.6 sama dengan backend); yang belum: respons tulis selain bentuk `Guest`, `5xx`, dan `409 conflict`. Ulangi bab 11.5 saat membangun.
2. **Filter dan halaman dilakukan server**, jadi mengganti pil atau filter adalah navigasi penuh (tidak ada penyaringan di klien seperti Menu 04). Alasannya daftar sampai 5.000 baris. Alternatif: menyaring di klien untuk proyek kecil (≤ 100).
3. **"Undangan" menjadi "pendaftaran" untuk seminar diturunkan dari `labels.guest === 'Peserta'`** (`invitationNoun`), karena registry hanya punya `Labels.Guest` dan `Labels.GuestMenu`. Alternatif: bidang ketiga di `Labels` ([BE bab 10 butir 15](BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
4. **Pesan WhatsApp berbahasa Indonesia tetap** dan netral jenis acara; tanpa tautan pada Standalone. Tidak ada penyesuaian per jenis acara atau per tamu (mis. sapaan "Bapak/Ibu").
5. **Kamera diblokir di produksi oleh Nginx** (`Permissions-Policy: camera=()` di `deploy/nginx/adicara.conf` baris 51) dan pemindai tidak berfungsi sampai diubah menjadi `camera=(self)`. `BarcodeDetector` tidak tersedia di semua peramban (mis. Safari iOS; belum diuji di perangkat); jalur ketik selalu ada. Planner tidak membuat gambar QR ([BE bab 10 butir 8](BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
6. **Satu sumber render dengan partial dan polling** (bab 2.4) belum dijalankan di browser; alternatif `location.reload()` dan polling JSON yang hanya memperbarui angka.
7. **Satu akun, banyak tablet:** pembatas `scan-qr` dan sesi per pengguna, bukan per perangkat; beberapa meja resepsionis pada satu akun berbagi 120 pemindaian/menit.
8. **Ekspor sebagai tautan** (teks 429 kurang ramah, tanpa `Retry-After`) atau `fetch` + Blob; pilih saat membangun. Hanya `.xlsx`; PDF menunggu pustaka ([BE bab 10 butir 9](BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
9. **Impor memuat berkas ke memori peramban sebagai `ArrayBuffer` lalu string base64** (≈ 5 MB + 6,7 MB sementara). Aman di laptop dan tablet modern; belum diukur di perangkat lama.
10. **Check-in menunggu respons server** (tanpa pembaruan optimistik dan tanpa antrean luring): sinyal buruk di lokasi acara menahan antrean. Mode luring (simpan check-in lokal lalu sinkron) bukan bagian spesifikasi.
11. **Satuan souvenir per undangan** ([BE bab 10 butir 2](BE.md#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)); kartu menyebutnya "diserahkan" tanpa menyebut "pcs".
12. **Chip kategori meja resepsionis adalah dropdown**, bukan chip VIP/Keluarga seperti wireframe lama (kategori VIP tidak ada pada semua jenis acara: peresmian memakai `pimpinan_vip`, aqiqah tidak punya).
13. **Proyek `completed` tetap bisa check-in dan mengedit**; hanya `archived` baca-saja ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)).
14. **Kinerja browser belum diukur.** Server membatasi 100 baris (Tab 1) dan 50 baris (Tab 2) per halaman; polling 5 detik mengganti satu halaman antrean beserta KPI.
15. **Awalan `'` pada sel Excel** tampak sebagai karakter ([02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)); ekspor dan templat belum dibuka di aplikasi spreadsheet.
16. **Banner Standalone tanpa tombol aksi** sampai modul Undangan Digital ada; versi awal punya tombol "Aktifkan Undangan Digital Adicara".
17. **Kategori kustom dibedakan huruf besar-kecil** ("Kolega Kantor" ≠ "kolega kantor"); dropdown modal menawarkan kategori yang sudah dipakai untuk mengurangi ejaan ganda.
18. **Hari ini dan tengah malam:** jam check-in memakai zona proyek (`clock`), bukan zona peramban; acara lintas tengah malam menampilkan jam lokal yang benar tetapi tanpa tanggal (rekap `.xlsx` memuat jam saja).

---

## 14. Riwayat perubahan dokumen ini

**2026-10-25 (backend menu 05 diimplementasikan dan dicocokkan):** status, bab 4.3–4.5, 11.5 (langkah 6), dan 13 butir 1 disesuaikan. Respons HTTP asli `GET guests`, `catering-projection`, `bulk-import?dry_run=true`, dan `reception-overview` sama dengan keempat fixture JSON (`TestResponsesMatchTheFEFixtures`, nol selisih setelah id, tiket, dan waktu dinormalkan), dan setiap baris katalog galat bab 4.6 dihasilkan backend dengan status, kode, dan pesan yang tertulis (`TestErrorCatalogOfTheFEDocIsWhatTheBackendSays`). Tiga contoh pesan galat di bab 4.6 (`already_checked_in`, `guest_limit_reached`, `file_too_large`) tidak berubah: backend yang disamakan dengannya. Isi bab lain tidak berubah.

**2026-10-25 (disamakan dengan pola 01–04 FE, dicocokkan dengan DTO Go yang dikompilasi dan angka Proyek Contoh dari SQL):** dokumen diperluas dari ±125 baris menjadi acuan lengkap: daftar isi, route dan file, kontrak props, alur data dan galat, prasyarat, tipe TypeScript yang dicocokkan dengan DTO Go (21 pasangan, 125 jalur, nol selisih), fixture, katalog endpoint dan galat, `guest.ts`, `guestApi.ts`, `guestMessages.ts` dengan kode yang sudah dijalankan, alur interaksi, pesan galat, aksesibilitas, pengujian, urutan pengerjaan. **Koreksi terhadap versi sebelumnya:** (1) layout `AppLayout`/`nav_active` diganti `PlannerLayout`; (2) pesan WhatsApp dibentuk klien (`wa.me` dari `formatted_wa_number`), bukan `whatsapp_url` dari server, dan Standalone tetap punya tombol WA (tanpa tautan); (3) `AttendancePercent` dibuang, persentase dihitung klien; (4) penghitung kategori per pihak seperti dua tabel PDF (sel `side_categories`), bukan satu set gabungan; (5) "belum hadir" dihitung dari `expected_guests` sehingga tamu yang menolak tetapi hadir tidak merusak angka; (6) antrean meja berpaginasi (50) dan satu panggilan polling membawa KPI + antrean; chip VIP/Keluarga menjadi dropdown kategori; (7) impor memakai JSON base64 (BFF), templat memiliki endpoint, dan ekspor hanya `.xlsx` (PDF dibuang); (8) tombol *Kirim WA bergilir* dan *Aktifkan Undangan Digital Adicara* dibuang; (9) proyeksi katering memakai persen bilangan bulat, dan tombol di modal pos Menu 03 dinonaktifkan pada 0 porsi; (10) pemindai membaca token mentah dan tautan personal (`extractTicketToken`), dan Nginx perlu `camera=(self)`; (11) "kotak kuning" nomor meja diganti token yang ada; (12) `reception_notes` pada check-in dibuang (tidak ada kolom).
