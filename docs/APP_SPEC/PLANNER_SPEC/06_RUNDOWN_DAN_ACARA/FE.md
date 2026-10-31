# Menu 06: Rundown & Acara — Frontend Specification

Acuan pembangunan antarmuka dua tab: **Susunan Rundown** menit demi menit dan **Audio, Playlist & Arahan Teknis** (Astro SSR, TypeScript, CSS token). Ditulis untuk developer frontend yang belum pernah menyentuh modul Planner: kontrak data, aturan tampilan, algoritma, alur interaksi, kasus galat, dan pengujian ada di sini. Pola umum (SSR + BFF, struktur file, tanpa dependensi baru, token CSS) ada di [00_KONTRAK_BERSAMA bab 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens); kontrak backend di [BE.md](BE.md).

> **Status (2026-10-31):** frontend Planner belum ada satu baris pun, tetapi **backend menu ini sudah dibangun** (2026-10-31, 17 endpoint; [BE.md](BE.md) bab 7.3, [docs/backend/planner/2026-10-31-planner-menu-06.md](../../../backend/planner/2026-10-31-planner-menu-06.md)) dan **respons HTTP aslinya sama dengan fixture di bab ini** (dua fixture dikoreksi saat dicocokkan: `date` pada sesi Lamaran dan `sort_order` per sesi). Bab ini ditulis ulang mengikuti pola Menu 04 dan 05 setelah diaudit. Yang **sudah dijalankan**: kode bab 4.2, 5, 6, dan 9.3 di bawah (diekstrak dari dokumen ini ke berkas) lolos `tsc` strict dengan `noUncheckedIndexedAccess` (konfigurasi yang dipakai repo); 19 tes `node:test` ([bab 11](#11-pengujian)) lulus terhadap fixture di [fixtures/](fixtures/), termasuk pencocokan setiap method + path pembungkus API dengan tabel endpoint BE bab 3 (17 endpoint) dan pemetaan setiap pesan di katalog galat; **36 dari 36 mutasi kode** yang dicoba menjatuhkan tes (dua tes awalnya terlalu longgar dan diperketat); tipe TypeScript dibandingkan dengan DTO Go yang dikompilasi: **23 pasangan, 118 jalur JSON, nol selisih**. **Belum dijalankan:** semua halaman, komponen, skrip klien, polling, cetak, dan perilaku di peramban, serta tes TypeScript terhadap backend yang hidup. Fixture dibangkitkan dari data SQL audit, dicocokkan dengan DTO Go, dan sejak modul dibangun dengan respons HTTP aslinya (dicocokkan di sisi backend oleh tes Go, [BE bab 7.3](BE.md#73-hasil-implementasi)).
>
> Yang salah pada versi sebelumnya dan kini dikoreksi: wireframe menampilkan durasi 5/10/10/5/10/10 menit untuk baris Akad yang oleh server dihitung 4/9/9/4/9/9 (kolom `duration_minutes` = selisih jam); halaman memakai `AppLayout` + `nav_active` padahal semua halaman `[id]/**` memakai `PlannerLayout` ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)); tombol PDF dan endpoint PDF dijanjikan tanpa pustaka PDF (kini cetak browser); contoh pesan WhatsApp tidak dapat dihasilkan oleh aturan BE (highlights vs keterangan, "5m" vs "mnt"); kotak cari playlist tanpa parameter di backend; polling tracker tanpa partial (kontrak 1.3); tanpa tipe, fixture, katalog galat, kamus pesan, uji sumber, urutan kerja, dan celah.
>
> Angka pada wireframe berasal dari **Proyek Contoh** ([fixtures/](fixtures/)): Sinta & Arif, Hari-H Minggu 28 Maret 2027. Wireframe Tab 1 adalah **ilustrasi hari-H** (Proyek Contoh belum berada di hari-H: semua agenda `pending`): Lamaran selesai, Akad 1–4 selesai dan 5 berjalan ([fixtures/planner-rundown-live.json](fixtures/planner-rundown-live.json)). Wireframe adalah tata letak, bukan desain visual akhir.

## Daftar isi

1. [Route, file, dan perubahan pada kode yang sudah ada](#1-route-file-dan-perubahan-pada-kode-yang-sudah-ada)
2. [Alur data dan penanganan galat](#2-alur-data-dan-penanganan-galat)
3. [Prasyarat yang harus sudah ada dari Menu 01 dan 04 FE](#3-prasyarat-yang-harus-sudah-ada-dari-menu-01-dan-04-fe)
4. [Kontrak data: tipe, fixture, katalog endpoint, katalog galat](#4-kontrak-data-tipe-fixture-katalog-endpoint-katalog-galat)
5. [Library `lib/planner/rundown.ts`](#5-library-libplannerrundownts)
6. [Pembungkus panggilan dan BFF](#6-pembungkus-panggilan-dan-bff)
7. [Tab 1: Susunan Rundown](#7-tab-1-susunan-rundown)
8. [Tab 2: Audio, Playlist & Arahan Teknis](#8-tab-2-audio-playlist--arahan-teknis)
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
| Rundown & Acara | `/dashboard/planner/[id]/rundown` | `src/pages/dashboard/planner/[id]/rundown/index.astro` | dua tab lewat `?tab=rundown` (default) dan `?tab=playlist`; nilai lain jatuh ke `rundown`. Tab dipilih server-side; bagian tab lain tidak dimuat. Sesi, tampilan, dan filter ada di URL (bab 2.3) |
| Partial | `/dashboard/planner/[id]/rundown/live` | `src/pages/dashboard/planner/[id]/rundown/live.astro` (`export const partial = true`) | HTML area yang diperbarui polling dan setelah aksi tulis (bab 2.4); menerima parameter URL yang sama dengan halaman |
| Komponen | – | `src/components/planner/rundown/*.astro` | lihat 1.2 |
| Library | – | `src/lib/planner/rundown.ts` (murni, bab 5), `rundownApi.ts` (pembungkus panggilan, bab 6), `rundownMessages.ts` (bab 9.3), `rundownClient.ts` (skrip klien halaman, bab 7.5 dan 8.5) | |
| Tes + fixture | – | `tests/planner-rundown.test.mjs`, `tests/planner-rundown-api.test.mjs`, `tests/planner-rundown-messages.test.mjs`, `tests/fixtures/planner-{rundowns,rundown-live,playlists,rundown-wa-akad,rundown-errors}.json` | bab 4.3 dan 11 |

Halaman memakai **`PlannerLayout`** (bilah sekunder berisi Project Switcher dan sub-navigasi, [01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) dengan `active="rundown"`, **bukan** `AppLayout` langsung dan tanpa `nav_active` (versi lama keliru).

```html
<nav class="rundown-tabs" role="tablist">
  <a href="?tab=rundown"  class="tab-item active" role="tab" aria-selected="true"><i class="ph ph-clock"></i> Susunan Rundown</a>
  <a href="?tab=playlist" class="tab-item"        role="tab" aria-selected="false"><i class="ph ph-music-notes"></i> Audio, Playlist & Arahan Teknis</a>
</nav>
```

### 1.2 Komponen

| Komponen | Props utama | Dipakai di | Klien |
|---|---|---|---|
| `RundownTabs` | `active`, `baseHref` | tab bar | navigasi penuh (`?tab=`) |
| `RundownKpi` | `kpi: RundownKPI`, `live: boolean` | Tab 1 | diganti oleh partial |
| `SessionTabs` | `sessions: RundownSessionSummary[]`, `selected`, `hrefs` | Tab 1 | tautan biasa; sesi tanpa agenda tetap tampil dengan `0` |
| `RundownTable` | `groups: SessionGroup[]`, `readOnly`, `project` | Tab 1 (mode Tabel) | badge status, menu aksi, tombol `💬 WA`, naik/turun |
| `RundownTimeline` | `groups`, `readOnly` | Tab 1 (mode Timeline) | drag handle (HTML5) dan tombol naik/turun, badge status |
| `LiveStatusBadge` | `status`, `startedAt?`, `serverTime`, `disabled` | tabel dan timeline | ketuk mengganti status (`nextStatus`), denyut saat LIVE |
| `RundownModal` | `sessions`, `defaultSessionId`, `vendors`, `item?` | Tab 1 | tambah/ubah agenda (`normalizeTimes`) |
| `TemplateMenu` | `impact: ReplaceImpact`, `readOnly` | Tab 1 dan Tab 2 | menu "Terapkan Template"; membuka `ConfirmReplaceDialog` |
| `ConfirmReplaceDialog` | `impact`, `onConfirm` | Tab 1 | memanggil `applyRundownTemplate({ mode: 'replace_template', confirm: true })` |
| `WaPreviewModal` | `title`, `text` | kedua tab | pratinjau dan salin teks dari server |
| `AudioKpi` | `kpi: PlaylistKPI` | Tab 2 | – |
| `SongFilters` | `momentTags`, `value` | Tab 2 | pencarian di klien; momen/format/status di URL |
| `PlaylistTable` | `items`, `readOnly` | Tab 2 | ubah status, hapus, tombol Spotify/YouTube, tautan agenda |
| `PlaylistModal` | `momentTags`, `agendas`, `item?` | Tab 2 | tambah/ubah lagu |

### 1.3 Perubahan pada file yang sudah ada (kecil, sengaja)

| File | Perubahan | Mengapa |
|---|---|---|
| `src/layouts/PlannerLayout.astro` | entri `rundown` pada `MENUS` menjadi `ready: true` | menu aktif sebagai tautan ([01 FE bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)) |
| `src/lib/planner/types.ts` | tambahan bab 4.2 | tipe respons dan badan permintaan |
| `src/lib/planner/messages.ts` | memanggil `rundownErrorMessage` lebih dulu, lalu `plannerErrorMessage` (bab 9) | kode dan alasan baru |
| `src/lib/planner/eventTypes.ts` | `EventTypeInfo.moment_tags: string[]` | dari `GET /event-types` (BE sudah menambahkan bidangnya, [BE bab 6](BE.md#6-sambungan-ke-menu-01)) |
| `tests/planner-wiring.test.mjs` | butir bab 11.2 | uji sumber |
| (tidak diubah) | BFF `[...path].ts`, `client.ts`, `AppLayout.astro`, `middleware.ts` | semua panggilan JSON ≤ 64 KiB dan path lolos `^[a-z0-9/_-]+$` (`export-wa`, `apply-template`, `summary`, `reorder`); tidak ada unduhan berkas |

`astro check` (bagian dari `npm run build`) harus lolos: TypeScript `strict` dan `noUncheckedIndexedAccess` aktif.

---

## 2. Alur data dan penanganan galat

### 2.1 Pemuatan per tab

Halaman memuat paralel di server (`backendFetch` dengan token): data proyek dari `PlannerLayout` (`ProjectLite`: `status`, `phase`), registry jenis acara (cache 1 jam, [01 FE bab 5.6](../01_DASHBOARD/FE.md#56-serverts)), dan satu panggilan sesuai tab:

| Tab | Panggilan SSR | Dimuat lazy saat modal dibuka |
|---|---|---|
| `rundown` | `GET rundowns` (`RundownOverview`; sesi, KPI, dan semua agenda; penyaringan sesi dilakukan di klien dari `items`) | `GET vendors` ([Menu 04](../04_VENDOR_DAN_BELANJA/BE.md#31-vendor)) untuk pilihan vendor di `RundownModal` |
| `playlist` | `GET playlists` (`PlaylistOverview`) | `GET rundowns` untuk pilihan agenda (dikelompokkan per sesi) di `PlaylistModal` |

Sesi (tab, dropdown, header) berasal dari `RundownOverview.sessions` (sudah tanpa sesi sistem `all`). Menambah, mengubah, atau menghapus sesi dilakukan di **Edit Acara → Kelola Sesi** (Menu 01); tab rundown hanya menyediakan tautan pintas ke sana. Bila proyek belum punya sesi non-sistem (jenis `other` yang sesinya dihapus): keadaan kosong "Belum ada sesi" dengan tautan ke Kelola Sesi, tanpa tombol tambah agenda.

### 2.2 Hasil backend → tindakan halaman

| Kondisi | Tindakan |
|---|---|
| `401` atau pengalihan login | `plannerFetch` mengalihkan ke `/auth/masuk` |
| `404 not_found` pada halaman (proyek) | `Astro.rewrite('/404')` |
| `404 not_found` pada aksi (agenda/lagu sudah dihapus perangkat lain) | toast "Agenda tidak ditemukan", muat ulang partial |
| proyek `archived` (`project.status`) | banner "Acara ini diarsipkan"; semua kontrol tulis dinonaktifkan sejak SSR; `409 project_read_only` pada aksi tetap dipetakan (kamus 01) |
| `400 invalid_input` dengan `<field>: <alasan>` | teks per kolom di bawah field (bab 9.2); `confirm: required` bukan galat: buka `ConfirmReplaceDialog` lalu ulangi dengan `confirm: true` (pola [03 FE](../03_KEUANGAN/FE.md): server menjawab `confirm: required` padahal hitungan lokal nol) |
| `400 invalid_scope` / `invalid_template` | toast; sesi tidak valid → muat ulang halaman; template lagu tidak ada → menu template lagu disembunyikan |
| `409 agenda_limit_reached` / `playlist_limit_reached` | toast; tombol tambah dinonaktifkan sampai muat ulang |
| `409 session_in_use` | hanya dari Kelola Sesi (Menu 01); bukan dari halaman ini |
| `429 rate_limited` (`export-wa`) | toast "Terlalu banyak permintaan"; tombol salin dinonaktifkan 10 detik |
| `5xx`, `503`, jaringan | toast; polling mundur (bab 2.4); perubahan status dikembalikan (rollback) ke status sebelumnya |
| `PATCH status` gagal | badge kembali ke status server dan toast; ketukan berikutnya memakai status terakhir dari server |

### 2.3 State di URL dan di peramban

| Parameter | Nilai | Arti |
|---|---|---|
| `tab` | `rundown` (bawaan) · `playlist` | tab |
| `session` | `scope_key` sesi | sesi terpilih pada mode *Satu sesi*; tidak ada atau tidak dikenal → sesi pertama yang punya agenda |
| `show` | `one` (bawaan) · `all` | *Satu sesi* atau *Semua* (tabel ditumpuk per sesi; untuk dicetak sebagai "Master Event Rundown Book") |
| `view` | `table` (bawaan) · `timeline` | tampilan |
| `momen`, `format`, `status` | filter Tab 2 | diteruskan ke `GET playlists` (`moment_tag`, `audio_format`, `status`) |
| `q` | teks | pencarian lagu di klien (judul, penyanyi, cue) |

Nilai tak dikenal diabaikan (bukan galat). Pilihan tampilan terakhir boleh diingat di `localStorage` (dibungkus `try/catch`, halaman benar tanpanya); URL selalu menang.

### 2.4 Satu sumber render dan pembaruan sesudah aksi

Daftar dan KPI dirender oleh **komponen yang sama** di halaman penuh dan partial `rundown/live` ([kontrak 1.3](../00_KONTRAK_BERSAMA/README.md#13-frontend-astro-ssr-typescript-css-tokens), pola [01 FE bab 9.5](../01_DASHBOARD/FE.md#95-pembaruan-berkala-polling-lewat-partial)); skrip klien hanya menukar isi DOM, bukan merender sendiri.

1. **Sesudah aksi tulis** (status, tambah, ubah, hapus, urut ulang, template): ambil partial dengan parameter URL halaman, ganti isi `#rundown-live`.
2. **Polling** `GET rundowns/summary` (`RundownLive`) tiap **5 detik** bila `project.phase === 'ongoing'` atau `kpi.in_progress > 0`, selain itu tiap **30 detik**; berhenti saat tab tersembunyi (`document.visibilityState`), mundur ×2 hingga 60 detik setelah galat dan kembali normal setelah sukses. Bila tanda tangan `items` (`id:status` terurut) berbeda dari yang ada di DOM, ambil partial dan tukar isinya; bila sama, hanya perbarui angka KPI dari JSON.
3. **Tidak menimpa isian yang sedang dibuka:** bila modal, dialog, atau seret-lepas sedang aktif, penukaran ditunda sampai ditutup.
4. **Status optimistik:** ketukan badge langsung mengubah tampilan dan memanggil `setStatus`; gagal → rollback (bab 2.2). `server_time` dari `RundownLive` dipakai untuk "berjalan N menit" pada badge LIVE; jam perangkat tidak dipakai.

---

## 3. Prasyarat yang harus sudah ada dari Menu 01 dan 04 FE

Bab ini tidak menulis ulang apa yang sudah ditetapkan di sana; pembangunan 06 baru boleh dimulai bila hal berikut sudah ada atau dibangun lebih dulu:

| Dari | Kebutuhan |
|---|---|
| 01 FE | BFF `[...path].ts` dan `client.ts` (`plannerFetch`, [bab 5.5 dan 6](../01_DASHBOARD/FE.md#6-bff-dashboardapiplanner)); `PlannerLayout` ([bab 10](../01_DASHBOARD/FE.md#10-project-switcher-dan-plannerlayout)); `types.ts` (`ProjectLite`, `SessionBrief`, `EventTypeInfo`), `format.ts` (`percent`, `formatPercent`, `formatDate`), `messages.ts` (`plannerErrorMessage`, `fieldErrorMessage`, `parseFieldError`; [bab 11](../01_DASHBOARD/FE.md#11-pesan-galat)); `loadRegistry`/`isUuid`/`toPageError` (`server.ts`); halaman `404.astro` |
| 04 FE | `Vendor`, `waLink`, `normalizeWA`, `PlannerFetch` ([`vendorApi.ts`](../04_VENDOR_DAN_BELANJA/FE.md#61-vendorapits)) dan `safeHref` |
| BE | modul `plannerrundown` ([BE bab 8](BE.md#8-urutan-pengerjaan-dan-definisi-selesai)) dan `moment_tags` pada `GET /event-types` |

BE sudah ada, jadi halaman dapat dikembangkan terhadap API asli; fixture tetap dipakai tes unit ([bab 12](#12-urutan-pengerjaan-dan-definisi-selesai)).

---

## 4. Kontrak data: tipe, fixture, katalog endpoint, katalog galat

### 4.1 Aturan JSON yang dikirim backend

- Kunci opsional **dihilangkan** bila kosong (tipe `?:`): agenda terbuka tidak punya `end_time` dan `duration_minutes`; `vendor`, `started_at`, `completed_at`, `template_key`, `rundown_id`, `agenda`, `spotify_url`, `youtube_url`, `audio_cue_notes`, `date`, `first_start`, `last_end`. **Tidak ada `null`** pada respons dan semua daftar selalu `[]` ([BE bab 3.3](BE.md#33-bentuk-respons)). Badan permintaan boleh mengirim `null` untuk `end_time`, `vendor_id`, `rundown_id`, tautan, dan cue.
- Jam `"HH:MM"` (titik hanya untuk tampilan); tanggal sesi `"YYYY-MM-DD"`; cap waktu RFC 3339 UTC.
- **Tidak ada persentase** pada respons: klien menghitung `percent(completed, total_agendas)` dan menampilkan "–" bila `total_agendas = 0`.
- `kpi` dan `sessions` selalu seluruh proyek; hanya `items` yang mengikuti filter.
- Durasi adalah `duration_minutes` dari server (selisih jam, tanpa `+1`): 07.21–07.25 = **4** menit. Klien tidak menghitung ulang durasi.

### 4.2 Tambahan pada `src/lib/planner/types.ts`

Nama sama dengan struct DTO Go ([BE bab 4](BE.md#4-dto-go)); dibandingkan secara terprogram dengan tag JSON Go (23 pasangan, 118 jalur, nol selisih; kunci dan opsionalitas `omitempty` ↔ `?:`).

```ts
// Tambahan pada src/lib/planner/types.ts (Menu 06). Nama sama dengan struct DTO Go (BE bab 4).
export type RundownStatus = 'pending' | 'in_progress' | 'completed';
export type AudioFormat = 'digital_playback' | 'live_band' | 'acoustic' | 'sound_fx';
export type AudioStatus = 'ready' | 'needs_review' | 'missing_file';

export interface Notice { code: string; message: string; count: number }

export interface RundownVendor {
  id: string; vendor_name: string; category_label: string;
  pic_name?: string; phone_number?: string; formatted_wa_number?: string;
}

export interface Rundown {
  id: string; session_id: string; session_name: string; session_icon: string; scope_key: string;
  start_time: string;                 // "HH:MM"
  end_time?: string;                  // tidak ada = agenda terbuka ("selesai")
  ends_next_day: boolean;
  duration_minutes?: number;          // tidak ada = agenda terbuka
  activity_name: string; pic_name: string;
  vendor?: RundownVendor;
  description: string; highlights: string;
  status: RundownStatus;
  started_at?: string; completed_at?: string;
  linked_songs: number; sort_order: number;
  template_key?: string;
}

export interface RundownSessionSummary {
  id: string; scope_key: string; name: string; icon: string;
  date?: string;                      // "YYYY-MM-DD"
  agenda_count: number; completed: number; in_progress: number; span_minutes: number;
  first_start?: string; last_end?: string;
}

export interface RundownKPI {
  sessions: number; total_agendas: number; completed: number; in_progress: number; pending: number;
  total_span_minutes: number; distinct_pics: number;
}

export interface RundownOverview { sessions: RundownSessionSummary[]; kpi: RundownKPI; items: Rundown[] }

export interface RundownLiveItem { id: string; status: RundownStatus; started_at?: string; completed_at?: string }
export interface RundownLive { kpi: RundownKPI; sessions: RundownSessionSummary[]; items: RundownLiveItem[]; server_time: string }

export interface PlaylistAgenda { id: string; activity_name: string; session_name: string; start_time: string; end_time?: string }

export interface Playlist {
  id: string; rundown_id?: string; agenda?: PlaylistAgenda;
  song_title: string; artist: string; moment_tag: string; audio_format: AudioFormat;
  spotify_url?: string; youtube_url?: string; audio_cue_notes?: string;
  status: AudioStatus; sort_order: number; template_key?: string;
}

export interface FormatCounts { digital_playback: number; live_band: number; acoustic: number; sound_fx: number }
export interface PlaylistKPI {
  total_tracks: number; by_format: FormatCounts; mapped_moments: number;
  ready: number; needs_review: number; missing_file: number;
}
export interface PlaylistOverview { kpi: PlaylistKPI; items: Playlist[] }

export interface TemplateResult { created: number; skipped: number; reset: number; kept: number; warnings: Notice[] }
export interface WAExport { text: string }

// ---------- badan permintaan ----------
export interface CreateRundownRequest {
  session_id: string; start_time: string; end_time?: string | null; ends_next_day: boolean;
  activity_name: string; pic_name: string; vendor_id?: string | null; description: string; highlights: string;
}
export type UpdateRundownRequest = CreateRundownRequest;   // PUT mengganti penuh; status hanya lewat PATCH
export interface RundownStatusRequest { status: RundownStatus }
export interface ReorderRundownsRequest { session_id: string; ids: string[] }
export interface ReorderPlaylistsRequest { ids: string[] }
export interface ApplyTemplateRequest { mode?: 'append' | 'replace_template'; confirm?: boolean }
export interface CreatePlaylistRequest {
  rundown_id?: string | null; song_title: string; artist: string; moment_tag?: string;
  audio_format?: AudioFormat; spotify_url?: string | null; youtube_url?: string | null;
  audio_cue_notes?: string | null; status?: AudioStatus;
}
export type UpdatePlaylistRequest = CreatePlaylistRequest;
```

### 4.3 Fixture

Lima berkas di [fixtures/](fixtures/), disalin ke `frontend/tests/fixtures/` saat FE dibangun. Dibangkitkan dari data SQL audit Proyek Contoh ([BE bab 7.1](BE.md#71-yang-sudah-dijalankan-pada-audit-spesifikasi)) dengan id deterministik (UUID v5) dan cap waktu 4 Okt 2026 / Hari-H 28 Maret 2027, lalu di-decode dengan `DisallowUnknownFields` ke DTO Go dan diserialisasi ulang sama (nol selisih). Sejak modul dibangun, respons HTTP asli **sama dengan** kelima berkas setelah id dan waktu dinormalkan (`TestSampleProjectGoldenRundown`, `TestHariHLiveAndWhatsAppMatchTheFixtures`, `TestErrorCatalogueMatchesTheServer`); dua berkas dikoreksi saat dicocokkan (`date` sesi Lamaran, `sort_order` per sesi).

| Berkas | Isi |
|---|---|
| `planner-rundowns.json` | `RundownOverview` Proyek Contoh: 3 sesi, 33 agenda `pending`, rentang 445, 17 PIC unik, 2 agenda terbuka |
| `planner-rundown-live.json` | `RundownLive` ilustrasi hari-H: 15 selesai, 1 berjalan, 17 menunggu |
| `planner-playlists.json` | `PlaylistOverview`: 30 lagu `digital_playback` + `needs_review`, 6 momen |
| `planner-rundown-wa-akad.json` | `WAExport` sesi Akad (aturan [BE bab 2.4](BE.md#24-pesan-whatsapp)); sama dengan teks yang dihasilkan modul |
| `planner-rundown-errors.json` | 17 galat `{status, code, message, source}`; `source` menyebut asalnya: 16 dari modul `plannerrundown` (dicocokkan dengan jawaban server oleh `TestErrorCatalogueMatchesTheServer`) dan 1 (`session_in_use`) dari kode Menu 01 |

### 4.4 Katalog endpoint

Semua di bawah `projects/:id/` lewat BFF; semua penulisan juga mungkin menjawab `401`, `404 not_found` (proyek), `409 project_read_only`, dan `5xx`/jaringan, yang tidak diulang di kolom Galat. Nama fungsi = `rundownApi` (bab 6). Deskripsi lengkap: [BE bab 3](BE.md#3-endpoint). `rid` = id agenda, `sid` = id lagu. Pencocokan method + path dengan tabel BE dijalankan oleh `tests/planner-rundown-api.test.mjs`.

| Aksi UI | Fungsi | Metode dan path | Body | Sukses | Galat khusus |
|---|---|---|---|---|---|
| Muat rundown | `listRundowns` | `GET rundowns[?session_id=]` | – | `200 RundownOverview` | `404` (sesi) |
| Polling | `liveState` | `GET rundowns/summary` | – | `200 RundownLive` | – |
| (satu agenda) | `getRundown` | `GET rundowns/:rid` | – | `200 Rundown` | `404` |
| Tambah agenda | `createRundown` | `POST rundowns` | `CreateRundownRequest` | `201 Rundown` | `409 agenda_limit_reached`, `400 invalid_input`/`invalid_scope` |
| Ubah agenda | `updateRundown` | `PUT rundowns/:rid` | `UpdateRundownRequest` | `200 Rundown` | seperti POST; `404` |
| Ketuk badge | `setStatus` | `PATCH rundowns/:rid/status` | `{ status }` | `200 Rundown` | `404` |
| Urut ulang | `reorderRundowns` | `POST rundowns/reorder` | `{ session_id, ids }` | `204` | `400 invalid_input: ids` |
| Hapus agenda | `deleteRundown` | `DELETE rundowns/:rid` | – | `204` | `404` (dianggap sukses) |
| Terapkan template | `applyRundownTemplate` | `POST rundowns/apply-template` | `{ mode?, confirm? }` | `200 TemplateResult` | `400 invalid_input: confirm` (dialog), `409 agenda_limit_reached` |
| Salin WA | `rundownWA` | `GET rundowns/export-wa[?session_id=]` | – | `200 WAExport` | `429` (10/menit) |
| Muat playlist | `listPlaylists` | `GET playlists[?moment_tag=&audio_format=&status=]` | – | `200 PlaylistOverview` | `400 invalid_input` (URL diketik tangan) |
| Tambah lagu | `createSong` | `POST playlists` | `CreatePlaylistRequest` | `201 Playlist` | `409 playlist_limit_reached`, `400 invalid_input` |
| Ubah lagu | `updateSong` | `PUT playlists/:sid` | `UpdatePlaylistRequest` | `200 Playlist` | seperti POST; `404` |
| Hapus lagu | `deleteSong` | `DELETE playlists/:sid` | – | `204` | `404` (dianggap sukses) |
| Urut ulang lagu | `reorderSongs` | `POST playlists/reorder` | `{ ids }` | `204` | `400 invalid_input: ids` |
| Template lagu | `applyPlaylistTemplate` | `POST playlists/apply-template` | `{ mode: 'append' }` | `200 TemplateResult` | `400 invalid_template`, `409 playlist_limit_reached` |
| Salin WA soundman | `playlistWA` | `GET playlists/export-wa` | – | `200 WAExport` | `429` |

### 4.5 Contoh permintaan dan respons

Contoh respons agenda diambil dari fixture. Angka hasil operasi (tambah, template, urut ulang) adalah yang dijawab backend sungguhan pada Proyek Contoh (`TestAgendaTimeRules`, `TestAppendIsIdempotentAndFillsGaps`, `TestReplaceTemplateResetsInPlace`, `TestReorderRundowns`, dan `TestEveryEventTypeGetsItsRundown` untuk `reset 33` pada rundown pernikahan yang belum disentuh).

Agenda terbuka dan agenda berdurasi pada respons `GET rundowns` (dari `planner-rundowns.json`; kunci dipotong):

```json
{ "id": "…", "scope_key": "lamaran", "start_time": "10:40", "ends_next_day": false,
  "activity_name": "Ramah Tamah - penutup", "pic_name": "MC", "description": "Foto bersama & Makan siang",
  "highlights": "", "status": "pending", "linked_songs": 0, "sort_order": 11, "template_key": "wedding.lamaran_11" }
{ "id": "…", "scope_key": "akad", "start_time": "07:21", "end_time": "07:25", "ends_next_day": false, "duration_minutes": 4,
  "activity_name": "Pembukaan acara", "pic_name": "MC", "status": "pending", "linked_songs": 0, "sort_order": 2 }
```

**Tambah agenda** melewati tengah malam (`POST rundowns`): `{"session_id":"…","start_time":"23:30","end_time":"00:30","ends_next_day":true,"activity_name":"After party","pic_name":"DJ","description":"","highlights":""}` → `201` dengan `duration_minutes: 60`.

**Ganti template** (`POST rundowns/apply-template` `{"mode":"replace_template"}` pada Proyek Contoh): tanpa `confirm` → `400 invalid_input: confirm: required`; dengan `confirm: true` → `{"created":0,"skipped":0,"reset":33,"kept":0,"warnings":[]}` (semua `pending`). Pada ilustrasi hari-H: `reset 17, kept 16`. **Tambah yang belum ada** (`append`) pada Proyek Contoh yang sudah memuat template: `{"created":0,"skipped":33,"reset":0,"kept":0,"warnings":[]}`.

**Urut ulang:** `{"session_id":"<akad>","ids":[8 id]}` → `204`; daftar tidak lengkap atau kembar → `400 invalid_input: ids: must list every agenda of the session exactly once`.

### 4.6 Katalog galat

Dibaca oleh `tests/planner-rundown-messages.test.mjs`: setiap baris `planner-rundown-errors.json` harus punya teks spesifik di bab 9 (kecuali `session_in_use`, milik kamus 01, dan `confirm: required`, yang membuka dialog). Status, kode, dan pesan tiap baris sama dengan jawaban server (`TestErrorCatalogueMatchesTheServer`).

| Status | Kode | Pesan (Inggris) | Tindakan |
|---|---|---|---|
| 400 | `invalid_input` | `start_time: must be a time like 07:46 or 07.46` | di bawah kolom jam |
| 400 | `invalid_input` | `end_time: must not be before start_time unless ends_next_day is set` | di bawah jam selesai |
| 400 | `invalid_input` | `end_time: must be earlier than start_time when ends_next_day is set` | di bawah jam selesai |
| 400 | `invalid_input` | `ends_next_day: requires end_time` | di bawah centang |
| 400 | `invalid_scope` | `session_id: must be a session of this project` | muat ulang |
| 400 | `invalid_input` | `session_id: must not be the system session` | di bawah sesi |
| 400 | `invalid_input` | `vendor_id: must be a vendor of this project` | muat ulang daftar vendor |
| 400 | `invalid_input` | `rundown_id: must be an agenda of this project` | muat ulang |
| 400 | `invalid_input` | `ids: must list every agenda of the session exactly once` | muat ulang |
| 400 | `invalid_input` | `mode: must be one of append, replace_template` / `mode: must be append` | bug klien |
| 400 | `invalid_input` | `confirm: required` | dialog konfirmasi |
| 400 | `invalid_input` | `spotify_url: must be an http or https URL` | di bawah tautan |
| 400 | `invalid_template` | `playlist: not available for this event type` | sembunyikan menu template lagu |
| 409 | `agenda_limit_reached` | `at most 500 agendas per project` | toast |
| 409 | `playlist_limit_reached` | `at most 500 songs per project` | toast |
| 409 | `session_in_use` | `the session still has rundown agendas` | milik Kelola Sesi (Menu 01) |

---

## 5. Library `lib/planner/rundown.ts`

Aturan 01 FE bab 5: modul **murni**, tanpa alias `@/`, tanpa `enum`, supaya `node --experimental-strip-types` dapat mengimpornya. Ia mengimpor `format.ts` (Menu 01: `percent`) dan `types.ts`. **Sudah dijalankan** ([bab 11.1](#111-unit-testsplanner-rundowntestmjs)).

```ts
// src/lib/planner/rundown.ts: murni (tanpa alias @/, tanpa enum) supaya node --experimental-strip-types dapat mengimpornya.
import { percent } from './format.ts';
import type {
  AudioFormat, AudioStatus, Playlist, Rundown, RundownKPI, RundownSessionSummary, RundownStatus,
} from './types.ts';

const CLOCK = /^(\d{1,2})[.:](\d{2})$/;

/** `7.46`, `07.46`, `07:46`, `7:46` -> "07:46"; selain itu null (detik, 24:00, 07.6 ditolak). */
export function parseClock(input: string): string | null {
  const m = CLOCK.exec(input.trim());
  if (!m) return null;
  const [, hh, mm] = m;
  if (hh === undefined || mm === undefined) return null;
  if (Number(hh) > 23 || Number(mm) > 59) return null;
  return `${hh.padStart(2, '0')}:${mm}`;
}

/** "07:46" -> "07.46" (format PDF dan pesan WhatsApp). */
export const formatClock = (hhmm: string): string => hhmm.replace(':', '.');

/** Durasi baris: "20 mnt"; agenda terbuka "–". */
export const durationLabel = (a: Pick<Rundown, 'duration_minutes'>): string =>
  a.duration_minutes === undefined ? '–' : `${a.duration_minutes} mnt`;

/** "07.00 – 07.20"; agenda terbuka "10.40 – selesai" (juga bila PDF hanya memuat jam mulai). */
export const agendaTimeLabel = (a: Pick<Rundown, 'start_time' | 'end_time'>): string =>
  `${formatClock(a.start_time)} – ${a.end_time === undefined ? 'selesai' : formatClock(a.end_time)}`;

/** Rentang sesi/total: 445 -> "7 jam 25 menit", 120 -> "2 jam", 59 -> "59 menit". */
export function spanLabel(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h === 0) return `${m} menit`;
  return m === 0 ? `${h} jam` : `${h} jam ${m} menit`;
}

export interface SessionGroup { session: RundownSessionSummary; rows: { no: number; item: Rundown }[] }

/** Tabel mandiri per sesi: urutan sesi dan agenda dari server, nomor mulai 1 pada tiap sesi. */
export function groupBySession(sessions: RundownSessionSummary[], items: Rundown[]): SessionGroup[] {
  return sessions.map((session) => ({
    session,
    rows: items.filter((i) => i.session_id === session.id).map((item, idx) => ({ no: idx + 1, item })),
  }));
}

export type TimesResult =
  | { ok: true; start_time: string; end_time: string | null }
  | { ok: false; field: 'start_time' | 'end_time' | 'ends_next_day'; reason: string };

const BAD_CLOCK = 'must be a time like 07:46 or 07.46';

/** Mencerminkan aturan server (BE bab 2.1); alasan memakai kosakata server supaya pemetaan galatnya satu. */
export function normalizeTimes(startRaw: string, endRaw: string, endsNextDay: boolean): TimesResult {
  const start = parseClock(startRaw);
  if (start === null) return { ok: false, field: 'start_time', reason: BAD_CLOCK };
  if (endRaw.trim() === '') {
    return endsNextDay ? { ok: false, field: 'ends_next_day', reason: 'requires end_time' } : { ok: true, start_time: start, end_time: null };
  }
  const end = parseClock(endRaw);
  if (end === null) return { ok: false, field: 'end_time', reason: BAD_CLOCK };
  if (endsNextDay && end >= start) return { ok: false, field: 'end_time', reason: 'must be earlier than start_time when ends_next_day is set' };
  if (!endsNextDay && end < start) return { ok: false, field: 'end_time', reason: 'must not be before start_time unless ends_next_day is set' };
  return { ok: true, start_time: start, end_time: end };
}

export interface ReplaceImpact { reset: number; kept: number }

/** "Ganti template" (BE bab 2.6): template pending dikembalikan ke isinya, yang berjalan/selesai tidak disentuh. */
export function replaceImpact(items: Pick<Rundown, 'template_key' | 'status'>[]): ReplaceImpact {
  let reset = 0;
  let kept = 0;
  for (const i of items) {
    if (i.template_key === undefined) continue;
    if (i.status === 'pending') reset++;
    else kept++;
  }
  return { reset, kept };
}

/** Ketukan badge: Belum -> LIVE -> Selesai. Selesai tidak berlanjut; membukanya kembali lewat menu aksi. */
export const nextStatus = (s: RundownStatus): RundownStatus | null =>
  s === 'pending' ? 'in_progress' : s === 'in_progress' ? 'completed' : null;

export const progressOf = (k: Pick<RundownKPI, 'completed' | 'total_agendas'>): number | null =>
  percent(k.completed, k.total_agendas);

export interface SongFilter { q: string; momentTag: string; format: AudioFormat | ''; status: AudioStatus | '' }

/** Daftar lagu tidak berpaginasi (BE bab 2.9): pencarian judul/artis/cue dilakukan di klien. */
export function filterSongs(items: Playlist[], f: SongFilter): Playlist[] {
  const q = f.q.trim().toLowerCase();
  return items.filter((s) =>
    (f.momentTag === '' || s.moment_tag === f.momentTag) &&
    (f.format === '' || s.audio_format === f.format) &&
    (f.status === '' || s.status === f.status) &&
    (q === '' || `${s.song_title} ${s.artist} ${s.audio_cue_notes ?? ''}`.toLowerCase().includes(q)));
}

/** Pilihan momen: tag registry jenis acara dulu, lalu tag kustom yang dipakai lagu (tanpa duplikat). */
export function momentTagOptions(registry: string[], used: string[]): string[] {
  const seen = new Set(registry);
  const extra: string[] = [];
  for (const t of used) if (!seen.has(t)) { seen.add(t); extra.push(t); }
  return [...registry, ...extra.sort((a, b) => a.localeCompare(b, 'id'))];
}
```

Catatan: `normalizeTimes` mencerminkan aturan server ([BE bab 2.1](BE.md#21-waktu-durasi-dan-rentang)) dan memakai kosakata alasan yang sama dengan server, sehingga pemetaan pesan satu (bab 9). Server tetap yang memutuskan: bila klien dan server berbeda, galat server yang ditampilkan. Durasi **tidak** dihitung di klien (`durationLabel` hanya memformat `duration_minutes`).

---

## 6. Pembungkus panggilan dan BFF

Tidak ada file BFF baru. Pemanggilan klien memakai `plannerFetch('projects/:id/…')` ([01 FE bab 5.5](../01_DASHBOARD/FE.md#55-clientts-khusus-browser)) dengan path yang lolos aturan BFF `^[a-z0-9/_-]+$`; semua jawaban berupa JSON atau `204` (tidak ada unduhan berkas, jadi `backendFetchRaw` tidak dipakai). Badan permintaan terbesar jauh di bawah 64 KiB.

### 6.1 `rundownApi.ts`

Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan sebagai argumen sehingga modul dapat diuji tanpa peramban; tipe `PlannerFetch` dari `vendorApi.ts` (Menu 04). Setiap id di path diperiksa UUID huruf kecil dan melempar `Error` bila tidak (kesalahan pemrogram, bukan permintaan ke server).

```ts
// src/lib/planner/rundownApi.ts
import type {
  ApplyTemplateRequest, AudioFormat, AudioStatus, CreatePlaylistRequest, CreateRundownRequest, Playlist, PlaylistOverview, Rundown,
  RundownLive, RundownOverview, RundownStatus, TemplateResult, UpdatePlaylistRequest, UpdateRundownRequest, WAExport,
} from './types.ts';
import type { PlannerFetch } from './vendorApi.ts';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
/** Path dibangun hanya dari UUID huruf kecil; selain itu kesalahan pemrogram (lempar), bukan permintaan ke server. */
const uuid = (v: string): string => {
  if (!UUID.test(v)) throw new Error(`bukan UUID: ${v}`);
  return v;
};

export interface SongQuery { momentTag?: string; audioFormat?: AudioFormat | ''; status?: AudioStatus | '' }

/** Satu fungsi per aksi UI (kolom "Fungsi" di bab 4.4). `plannerFetch` disuntikkan agar modul teruji tanpa peramban. */
export function rundownApi(call: PlannerFetch, projectId: string) {
  const p = `projects/${uuid(projectId)}`;
  const r = `${p}/rundowns`;
  const s = `${p}/playlists`;
  const bySession = (sessionId?: string) => (sessionId ? { query: { session_id: uuid(sessionId) } } : undefined);
  return {
    // ----- Tab 1: rundown -----
    /** `kpi` dan `sessions` selalu seluruh proyek; `items` disaring bila `sessionId`. */
    listRundowns: (sessionId?: string) => call<RundownOverview>(r, bySession(sessionId)),
    /** Polling 5 detik: KPI, ringkasan sesi, dan status tiap agenda. */
    liveState: () => call<RundownLive>(`${r}/summary`),
    getRundown: (id: string) => call<Rundown>(`${r}/${uuid(id)}`),
    createRundown: (body: CreateRundownRequest) => call<Rundown>(r, { method: 'POST', body }),
    /** PUT mengganti penuh dan tidak membawa status (status hanya lewat setStatus). */
    updateRundown: (id: string, body: UpdateRundownRequest) => call<Rundown>(`${r}/${uuid(id)}`, { method: 'PUT', body }),
    setStatus: (id: string, status: RundownStatus) => call<Rundown>(`${r}/${uuid(id)}/status`, { method: 'PATCH', body: { status } }),
    /** `ids` harus tepat semua agenda sesi itu, sekali masing-masing. */
    reorderRundowns: (sessionId: string, ids: string[]) =>
      call<null>(`${r}/reorder`, { method: 'POST', body: { session_id: uuid(sessionId), ids: ids.map(uuid) } }),
    deleteRundown: (id: string) => call<null>(`${r}/${uuid(id)}`, { method: 'DELETE' }),
    /** `replace_template` tanpa `confirm` dijawab `400 invalid_input: confirm: required` bila ada agenda yang akan di-reset. */
    applyRundownTemplate: (body: ApplyTemplateRequest) => call<TemplateResult>(`${r}/apply-template`, { method: 'POST', body }),
    rundownWA: (sessionId?: string) => call<WAExport>(`${r}/export-wa`, bySession(sessionId)),
    // ----- Tab 2: playlist -----
    listPlaylists: (f: SongQuery = {}) => {
      const query: Record<string, string> = {};
      if (f.momentTag) query.moment_tag = f.momentTag;
      if (f.audioFormat) query.audio_format = f.audioFormat;
      if (f.status) query.status = f.status;
      return call<PlaylistOverview>(s, Object.keys(query).length > 0 ? { query } : undefined);
    },
    createSong: (body: CreatePlaylistRequest) => call<Playlist>(s, { method: 'POST', body }),
    updateSong: (id: string, body: UpdatePlaylistRequest) => call<Playlist>(`${s}/${uuid(id)}`, { method: 'PUT', body }),
    deleteSong: (id: string) => call<null>(`${s}/${uuid(id)}`, { method: 'DELETE' }),
    reorderSongs: (ids: string[]) => call<null>(`${s}/reorder`, { method: 'POST', body: { ids: ids.map(uuid) } }),
    /** Lagu hanya `append`. */
    applyPlaylistTemplate: () => call<TemplateResult>(`${s}/apply-template`, { method: 'POST', body: { mode: 'append' } }),
    playlistWA: () => call<WAExport>(`${s}/export-wa`),
  };
}
```

---

## 7. Tab 1: Susunan Rundown

### 7.1 Wireframe (sesi Akad Nikah dari PDF; ilustrasi hari-H)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────┐ ┌────────────────────┐ ┌────────────────────────┐ ┌──────────────────┐   │
│ │ 📋 Total Agenda│ │ ⏳ Total Rentang   │ │ 🎯 Progres Live        │ │ 👥 PIC & Vendor  │   │
│ │ 33 · 3 sesi    │ │ 7 jam 25 menit     │ │ 15 selesai · 1 LIVE ·  │ │ 17 unik          │   │
│ │                │ │ (100+120+225 mnt)  │ │ 17 menunggu [████░] 45%│ │                  │   │
│ └────────────────┘ └────────────────────┘ └────────────────────────┘ └──────────────────┘   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ Sesi: [💍 Lamaran (11)] [★ 📜 Akad Nikah (8)] [🏛️ Resepsi (14)]   Tampilan: (•) Satu sesi ( ) Semua│
│ Mode: [📋 Tabel] [⏱️ Timeline]            📅 Minggu, 28 Maret 2027 · 07.00 – 09.00 · 8 agenda   │
│ [ + Tambah Agenda ]  [ 💬 Salin WA Sesi Ini ]  [ 💬 Salin WA Semua ]  [ 🖨️ Cetak Cue Card ]  [ Terapkan Template ▾ ] │
│ ✏️ Kelola sesi di Edit Acara (Menu 01)                                                         │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Waktu          Dur     Agenda               PIC / Vendor    Keterangan                Live  │
│  1 07.00 – 07.20  20 mnt  Persiapan            CMP & CMW       Penghulu datang dan …     🟢    │
│  2 07.21 – 07.25   4 mnt  Pembukaan acara      MC              Pembawa acara melakukan … 🟢    │
│  3 07.26 – 07.35   9 mnt  Pembacaan ayat suci  Qori            Pembacaan ayat-ayat …     🟢    │
│  4 07.36 – 07.45   9 mnt  Khutbah              Ustadz/Kyai     Kutbah pernikahan dan …   🟢    │
│  5 07.46 – 07.50   4 mnt  Ijab Kabul           Penghulu        Ijab kabul yang dipimpin… 🟡 LIVE│
│  6 07.51 – 08.00   9 mnt  Doa bersama          Kyai/penghulu   Doa bersama dan penyerah… ⚪    │
│  7 08.01 – 08.10   9 mnt  Sesi foto            Fotografer      Sesi foto bersama dan …   ⚪    │
│  8 08.11 – 09.00  49 mnt  Selesai akad         MC              Persiapan melanjutkan ke… ⚪    │
│  Rekap sesi: 8 agenda · rentang 07.00 – 09.00 (2 jam) · 4 selesai, 1 LIVE, 3 menunggu         │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
 Agenda terbuka tampil "10.40 – selesai" dan durasi "–" (mis. Ramah Tamah pada sesi Lamaran, dan
 "Resepsi" 09.15 pada sesi Resepsi). Durasi = selisih jam (07.21–07.25 = 4 mnt), sama dengan server.
```

Durasi dan teks kolom di atas adalah isi `planner-rundowns.json` (diperiksa terprogram pada audit); kolom Keterangan dipotong untuk wireframe.

### 7.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| Kartu **Total Agenda** | `kpi.total_agendas`, `kpi.sessions` | "33 · 3 sesi" |
| Kartu **Total Rentang** | `kpi.total_span_minutes`, `sessions[].span_minutes` | `spanLabel` ("7 jam 25 menit"); rincian "(100+120+225 mnt)" dari `span_minutes` sesi; bukan jumlah durasi ([BE bab 2.1](BE.md#21-waktu-durasi-dan-rentang)) |
| Kartu **Progres Live** | `kpi.completed/in_progress/pending`, `progressOf(kpi)` | batang + `formatPercent`; `total_agendas = 0` → "Belum ada agenda" tanpa batang |
| Kartu **PIC & Vendor** | `kpi.distinct_pics` | "17 unik"; 0 → "Belum ada PIC" |
| `SessionTabs` | `sessions[]` | nama, ikon, `agenda_count`; sesi tanpa agenda tetap tampil "(0)" |
| Header sesi | `sessions[]` | `date` (`formatDate`, "Minggu, 28 Maret 2027"; kosong → "Tanggal belum diisi"), `first_start`–`last_end`, `agenda_count` |
| Tabel | `groupBySession(sessions, items)` | satu tabel per sesi, nomor 1…n per sesi; urutan dari server |
| Kolom Waktu / Dur | `agendaTimeLabel`, `durationLabel` | agenda terbuka "10.40 – selesai" dan "–" |
| Kolom PIC / Vendor | `vendor` atau `pic_name` | vendor: nama vendor + tombol `💬 WA` (`waLink(formatted_wa_number)`; tanpa nomor → tanpa tombol); tanpa vendor: teks PIC; keduanya kosong: "–" |
| Kolom Keterangan | `description`, `highlights` | keterangan; highlights sebagai butir `•` di bawahnya (awalan "- " dibuang) |
| Badge `🎵 N lagu` | `linked_songs` | tampil bila > 0; membuka Tab Audio terfilter agenda itu |
| `LiveStatusBadge` | `status`, `started_at`, `server_time` | ⚪ Belum Mulai · 🟡 LIVE ON STAGE (denyut; "berjalan N menit") · 🟢 Selesai |
| Rekap sesi | `sessions[]` | "8 agenda · rentang 07.00 – 09.00 (2 jam) · 4 selesai, 1 LIVE, 3 menunggu" |

### 7.3 Kontrol

- **Tampilan:** *Tabel* dan *Timeline* (kartu vertikal dengan penanda waktu, garis alur, drag handle; jeda antar agenda terlihat). **Sesi:** *Satu sesi* atau *Semua* (ditumpuk, dengan kepala tiap sesi).
- **Live Stage Manager:** badge 1-ketukan `⚪ → 🟡 → 🟢` (`nextStatus`). Beberapa agenda boleh LIVE bersamaan (sesi makan, hiburan, salaman). Membuka kembali agenda selesai (`pending`) hanya lewat menu aksi, supaya tidak tertukar saat panik.
- **Urut ulang:** seret handle (HTML5 native) atau tombol ▲▼ (keyboard dan layar sentuh); hasilnya satu `reorderRundowns(session_id, ids lengkap sesi)`; gagal → daftar dikembalikan.
- **Aksi baris** (menu `•••`): ubah, tandai Belum, hapus (konfirmasi). **Tombol** di atas tabel: tambah agenda, salin WA sesi/semua, **cetak cue card** (`window.print()` dengan stylesheet cetak, bab 10; tidak ada ekspor PDF), terapkan template.
- Semua kontrol tulis dinonaktifkan pada proyek `archived`.

### 7.4 Modal dan dialog

- **`RundownModal`:** *Sesi* (dropdown dari `sessions`, tanpa `all`); *Jam mulai* dan *Jam selesai* (input teks yang menerima `07.46` atau `07:46`; jam selesai kosong = terbuka); centang *Berakhir hari berikutnya*; durasi ditampilkan dari respons server (tidak diisi manual); *Nama agenda*\*; *PIC* (teks bebas) **atau** *Vendor* (dropdown dari Menu 04; bila vendor dipilih, teks PIC tetap boleh diisi); keterangan; highlights (satu butir per baris, disimpan sebagai baris berawalan "- "). **Status tidak ada di modal** (hanya lewat badge). `normalizeTimes` memeriksa sebelum mengirim; galat server ditampilkan di bawah kolom (bab 9.2).
- **`TemplateMenu`** ("Terapkan Template ▾"; hanya bila `templates_available.rundown`): *Tambah yang belum ada* (`append`, aman diulang; menampilkan "N ditambahkan, M sudah ada") dan *Kembalikan agenda template yang belum berjalan* (`replace_template`). Bagi yang kedua `replaceImpact(items)` memberi angka untuk dialog: "N agenda template yang belum berjalan akan dikembalikan ke isi template (perubahan Anda pada agenda itu tertimpa; vendor dan lagu tertaut tetap); M yang sedang atau sudah berjalan tidak diubah". `reset = 0` → kirim tanpa `confirm`; bila server menjawab `confirm: required` padahal lokal tidak, tampilkan dialog lalu ulangi. Warning `scope_missing` ditampilkan sebagai "Agenda untuk sesi yang sudah dihapus dilewati ({count})".
- **`WaPreviewModal`:** pratinjau teks dari `rundownWA` (satu sesi atau semua), tombol Salin (`navigator.clipboard`, cadangan `<textarea>` + `execCommand`), tombol "Buka WhatsApp" (`https://wa.me/?text=<encodeURIComponent>`).

### 7.5 Alur interaksi (Tab 1)

1. **Ketuk badge** → rollback-aman: ubah tampilan, `setStatus(id, nextStatus(status))`; gagal → kembalikan dan toast; sukses → ambil partial.
2. **Tambah agenda** → modal → `normalizeTimes` → `createRundown` → tutup, partial, sorot baris baru. Batas 500 → toast.
3. **Ubah** → modal terisi → `updateRundown` (`PUT` penuh; semua field dikirim, tanpa status).
4. **Hapus** → konfirmasi "Hapus agenda 'X'? Lagu yang tertaut akan lepas tautannya." → `deleteRundown`; `404` dianggap sukses.
5. **Urut ulang** → `reorderRundowns(session, ids)` → partial.
6. **Template** → bab 7.4.
7. **Salin WA** → `rundownWA(session?)` → pratinjau → salin. `429` → tombol nonaktif 10 detik.
8. **Cetak** → `window.print()`; stylesheet cetak menampilkan mode *Semua* bila dipilih, satu sesi per halaman.

### 7.6 Salinan teks

| Konteks | Teks |
|---|---|
| Kosong, ada sesi | "Belum ada agenda. Tambahkan satu, atau terapkan template rundown." |
| Kosong, tanpa sesi | "Belum ada sesi acara. Tambahkan sesi di Edit Acara terlebih dahulu." |
| Arsip | "Acara ini diarsipkan. Buka arsip untuk mengubah rundown." |
| Badge LIVE | "LIVE ON STAGE · berjalan {N} menit" |
| Konfirmasi hapus | "Hapus agenda '{nama}'? Lagu yang tertaut akan lepas tautannya." |
| Hasil `append` | "{created} agenda ditambahkan, {skipped} sudah ada." |
| Hasil `replace_template` | "{reset} agenda dikembalikan ke template, {kept} yang sedang atau sudah berjalan tidak diubah." |

---

## 8. Tab 2: Audio, Playlist & Arahan Teknis

### 8.1 Wireframe (30 lagu PDF; kolom tag/format/kesiapan adalah tambahan Adicara)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌────────────────────────┐      │
│ │ 🎵 Koleksi     │ │ 🎸 Format        │ │ 🏷️ Momen         │ │ ⚡ Kesiapan             │      │
│ │ 30 lagu        │ │ 30 Digital       │ │ 6 momen terpeta- │ │ 0 siap · 30 perlu       │      │
│ │ (template PDF) │ │ (belum diatur)   │ │ kan              │ │ ditinjau                │      │
│ └────────────────┘ └──────────────────┘ └──────────────────┘ └────────────────────────┘      │
│ 🔍 [ judul / artis / cue … ]  Momen:[Semua ▼] Format:[Semua ▼] Status:[Semua ▼]               │
│ [ + Tambah Audio ] [ 💬 Salin WA Soundman & Band ] [ 🖨️ Cetak Cue Sheet ] [Terapkan Template ▾]│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ No Judul · Penyanyi                  Momen                 Format    Cue (arahan soundman)  Agenda        Status│
│  1 Akad · Payung Teduh               Akad Khidmat          Digital   –                      –             ⚠ Tinjau│
│  2 Janji Suci · Yovie & Nuno         Akad Khidmat          Digital   –                      –             ⚠ Tinjau│
│  5 Marry You · Maroon 5              Kirab / Entrance      Digital   –                      –             ⚠ Tinjau│
│ 15 Sugar · Maroon 5                  Santap & Ramah Tamah  Digital   –                      –             ⚠ Tinjau│
│ 28 Shallow · Lady Gaga               Games & Lempar Bunga  Digital   –                      –             ⚠ Tinjau│
│ … 25 lagu lainnya                                                                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Pada template semua lagu `digital_playback` + `needs_review` + tanpa tautan agenda, karena PDF hanya memuat judul dan penyanyi. Setelah pengguna menautkan lagu ke agenda dan mengisi cue, kolom Agenda menampilkan mis. "Iring-iring (09.36–09.40)" (dari `agenda`) dan KPI berubah. Hanya pernikahan punya lagu bawaan; pada jenis lain tab ini kosong dan menu template lagu tidak tampil.

### 8.2 Pemetaan data → tampilan

| Elemen | Sumber | Aturan |
|---|---|---|
| `AudioKpi` Koleksi | `kpi.total_tracks` | "30 lagu"; 0 → "Belum ada lagu" |
| Format | `kpi.by_format` | angka per format yang > 0 ("30 Digital"); semua 0 → "–" |
| Momen | `kpi.mapped_moments` | "6 momen terpetakan" |
| Kesiapan | `kpi.ready/needs_review/missing_file` | "0 siap · 30 perlu ditinjau" (+ "· N file belum ada" bila > 0) |
| Filter | URL `momen`, `format`, `status` → server; `q` → klien (`filterSongs`) | KPI selalu seluruh proyek; daftar tidak berpaginasi |
| Opsi momen | `momentTagOptions(type.moment_tags, items.map(moment_tag))` | tag registry dulu, lalu tag kustom yang terpakai |
| Kolom Agenda | `agenda` | "{nama agenda} ({jam mulai}–{jam selesai})" atau "–"; menautkan ke baris agenda di Tab 1 |
| Kolom Cue | `audio_cue_notes` | dipotong; selengkapnya di modal |
| Status | `status` | ✅ Siap · ⚠ Tinjau · ⛔ File belum ada |
| Preview | `spotify_url`, `youtube_url` | tombol hanya bila `safeHref` menerimanya; `rel="noopener noreferrer"`, `target="_blank"` |

### 8.3 Komponen dan modal

- **`PlaylistModal`:** judul\*, penyanyi\*, momen (dropdown + ketik baru), format (Live Band / Akustik / Digital Playback / Sound FX), tautan agenda (dropdown seluruh agenda proyek dikelompokkan per sesi; dimuat lazy), Spotify, YouTube, catatan cue (titik mulai, fade in/out, ducking saat MC berbicara, level volume, titik berhenti), status (Siap Diputar / Review / File Belum Ada). `PUT` penuh: semua field dikirim.
- **Template lagu** (`TemplateMenu`, hanya `templates_available.playlist`): hanya *Tambah yang belum ada*.
- **Salin WA Soundman & Band:** `playlistWA()` → `WaPreviewModal` ([BE bab 2.7](BE.md#27-playlist-dan-sound-cue)). **Cetak Cue Sheet:** `window.print()` dengan stylesheet cetak yang sama dengan cue card rundown; tidak ada PDF.

### 8.4 Pemeriksaan sebelum mengirim

- Judul dan penyanyi tidak kosong setelah `trim`; tautan Spotify/YouTube harus `http://` atau `https://` (klien memberi tahu lebih awal; server tetap yang memutuskan); panjang ≤ batas ([BE bab 3.4](BE.md#34-validasi-dan-batas)); momen kosong dikirim sebagai kosong (server memakai "Lainnya").
- Menautkan lagu ke agenda tidak mengubah agenda; badge `🎵 N lagu` di Tab 1 berubah pada partial berikutnya.

### 8.5 Alur interaksi (Tab 2)

1. **Tambah/ubah/hapus lagu** → modal atau konfirmasi → `createSong`/`updateSong`/`deleteSong` → partial.
2. **Ubah status cepat** (ikon status pada baris) → `updateSong` penuh dengan status baru (PUT mengganti penuh; klien mengirim semua field).
3. **Filter** → navigasi URL (momen/format/status) atau filter klien (`q`).
4. **Urut ulang** → `reorderSongs(ids lengkap proyek)`; hanya pada tampilan tanpa filter (dengan filter, tombol urut dinonaktifkan karena daftar tidak lengkap).
5. **Template lagu** → `applyPlaylistTemplate()` → "{created} lagu ditambahkan, {skipped} sudah ada."
6. **Salin WA** → pratinjau → salin.

### 8.6 Salinan teks

| Konteks | Teks |
|---|---|
| Kosong, tanpa template lagu (`templates_available.playlist` tidak ada; kini delapan jenis non-pernikahan) | "Belum ada lagu atau sound cue. Tambahkan lagu pembuka, bumper, atau efek suara acara Anda." |
| Kosong, template lagu tersedia (`templates_available.playlist`; kini pernikahan) | "Belum ada lagu. Terapkan template untuk memuat lagu dari daftar acuan." (tanpa angka: jumlahnya milik template, bukan FE) |
| Hasil template | "{created} lagu ditambahkan, {skipped} sudah ada." |
| Filter tanpa hasil | "Tidak ada lagu yang cocok dengan filter." |

---

## 9. Pesan galat dan teks

### 9.1 Kode → teks

Kamus 01 ([bab 11.1](../01_DASHBOARD/FE.md#111-kode--teks-plannererrormessagestatus-code)) berlaku untuk `not_found`, `project_read_only`, `conflict`, `rate_limited`, `session_in_use`, dan 5xx. Tambahan Menu 06 (`rundownErrorMessage`, bab 9.3; dipanggil lebih dulu, `null` berarti pakai kamus 01):

| Kode / kondisi | Teks Indonesia | Muncul pada |
|---|---|---|
| `agenda_limit_reached` | "Maksimal 500 agenda per acara." | tambah agenda, template |
| `playlist_limit_reached` | "Maksimal 500 lagu per acara." | tambah lagu, template |
| `invalid_template` | "Jenis acara ini belum punya template lagu." | template lagu |
| `invalid_scope` | "Sesi tidak valid. Muat ulang halaman." | tambah/ubah agenda |
| `invalid_input` | per kolom (bab 9.2); `confirm: required` membuka dialog | semua formulir |

### 9.2 Alasan per kolom

Format backend `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)); pola alasan dari Menu 01 (`required`, `must be at most N characters`) ditambah milik Menu 06:

| Pola alasan | Teks (`{label}` dari kamus kolom) |
|---|---|
| `must be a time like 07:46 or 07.46` | "{label} harus berupa jam, mis. 07.46." |
| `must not be before start_time unless ends_next_day is set` | "Jam selesai tidak boleh sebelum jam mulai. Centang "berakhir hari berikutnya" bila acara melewati tengah malam." |
| `must be earlier than start_time when ends_next_day is set` | "Bila berakhir hari berikutnya, jam selesai harus lebih awal dari jam mulai (mis. 23.30 → 00.30)." |
| `requires end_time` | "Isi jam selesai, atau hilangkan centang "berakhir hari berikutnya"." |
| `must not be the system session` | "Pilih sesi acara, bukan "Umum"." |
| `must be a session of this project` | "Sesi tidak ditemukan. Muat ulang halaman." |
| `must be a vendor of this project` | "Vendor tidak ditemukan. Muat ulang daftar vendor." |
| `must be an agenda of this project` | "Agenda tidak ditemukan. Muat ulang halaman." |
| `must list every (agenda|song) of the (session|project) exactly once` | "Urutan tidak lengkap. Muat ulang halaman lalu coba lagi." |
| `must be an http or https URL` | "{label} harus berupa tautan http atau https." |
| `required` / `must not be empty` | "{label} wajib diisi." |
| `must be at most N characters` | "{label} maksimal N karakter." |
| `must be one of …` / `must be append` | "{label} tidak valid." (kesalahan klien) |

### 9.3 Kode `rundownMessages.ts`

```ts
// src/lib/planner/rundownMessages.ts: tambahan untuk messages.ts (01 FE bab 11). Mengembalikan null bila kamus 01 berlaku.
const LABELS: Record<string, string> = {
  session_id: 'Sesi', start_time: 'Jam mulai', end_time: 'Jam selesai', ends_next_day: 'Berakhir hari berikutnya',
  activity_name: 'Nama agenda', pic_name: 'PIC', vendor_id: 'Vendor', description: 'Keterangan', highlights: 'Highlights',
  song_title: 'Judul lagu', artist: 'Penyanyi', moment_tag: 'Momen', audio_format: 'Format', status: 'Status',
  spotify_url: 'Tautan Spotify', youtube_url: 'Tautan YouTube', audio_cue_notes: 'Catatan cue', rundown_id: 'Agenda',
  ids: 'Urutan', mode: 'Mode', confirm: 'Konfirmasi',
};

const FIELD_RULES: [RegExp, (label: string, m: RegExpExecArray) => string][] = [
  [/^must be a time like 07:46 or 07\.46$/, (l) => `${l} harus berupa jam, mis. 07.46.`],
  [/^must not be before start_time unless ends_next_day is set$/, () => 'Jam selesai tidak boleh sebelum jam mulai. Centang "berakhir hari berikutnya" bila acara melewati tengah malam.'],
  [/^must be earlier than start_time when ends_next_day is set$/, () => 'Bila berakhir hari berikutnya, jam selesai harus lebih awal dari jam mulai (mis. 23.30 → 00.30).'],
  [/^requires end_time$/, () => 'Isi jam selesai, atau hilangkan centang "berakhir hari berikutnya".'],
  [/^must not be the system session$/, () => 'Pilih sesi acara, bukan "Umum".'],
  [/^must be a session of this project$/, () => 'Sesi tidak ditemukan. Muat ulang halaman.'],
  [/^must be a vendor of this project$/, () => 'Vendor tidak ditemukan. Muat ulang daftar vendor.'],
  [/^must be an agenda of this project$/, () => 'Agenda tidak ditemukan. Muat ulang halaman.'],
  [/^must list every (agenda|song) of the (session|project) exactly once$/, () => 'Urutan tidak lengkap. Muat ulang halaman lalu coba lagi.'],
  [/^must be an http or https URL$/, (l) => `${l} harus berupa tautan http atau https.`],
  [/^(required|must not be empty)$/, (l) => `${l} wajib diisi.`],
  [/^must be at most (\d+) characters$/, (l, m) => `${l} maksimal ${m[1]} karakter.`],
  [/^must (be one of .*|be append)$/, (l) => `${l} tidak valid.`],   // kesalahan klien: kontrol yang salah
];

/** `invalid_input` dengan pesan `<field>: <alasan>` -> teks Indonesia per kolom; null bila tidak dikenal. */
export function rundownFieldMessage(message: string): string | null {
  const m = /^([a-z_]+): (.+)$/.exec(message);
  if (!m) return null;
  const [, field, reason] = m;
  if (field === undefined || reason === undefined) return null;
  const label = LABELS[field] ?? field;
  for (const [re, text] of FIELD_RULES) {
    const hit = re.exec(reason);
    if (hit) return text(label, hit);
  }
  return null;
}

/** `confirm: required` bukan galat bagi pengguna: buka dialog konfirmasi lalu ulangi dengan `confirm: true`. */
export const isConfirmRequired = (code: string | undefined, message: string | undefined): boolean =>
  code === 'invalid_input' && message === 'confirm: required';

/** Kode baru dan kode yang bermakna lain di Menu 06; null = pakai plannerErrorMessage (01 FE bab 11.1). */
export function rundownErrorMessage(code: string | undefined, message?: string): string | null {
  switch (code) {
    case 'agenda_limit_reached': return 'Maksimal 500 agenda per acara.';
    case 'playlist_limit_reached': return 'Maksimal 500 lagu per acara.';
    case 'invalid_template': return 'Jenis acara ini belum punya template lagu.';
    case 'invalid_scope': return 'Sesi tidak valid. Muat ulang halaman.';
    case 'invalid_input': return message === undefined ? null : rundownFieldMessage(message);
    default: return null;
  }
}
```

---

## 10. Gaya, responsif, aksesibilitas, keamanan

### 10.1 Gaya

Hanya token yang ada di `tokens.css` (`--sage-*`, `--color-*`, `--space-*`, `--radius-*`, `--font-display`, `--font-body`). Badge status memakai kombinasi **ikon + teks** (⚪ Belum / 🟡 LIVE / 🟢 Selesai), bukan warna saja; denyut LIVE dimatikan di bawah `prefers-reduced-motion`. **Stylesheet cetak** (`@media print`): sembunyikan header, bilah sekunder, tombol, dan kolom aksi; huruf besar, kontras tinggi, A4 atau A5 (`@page`), satu sesi per halaman pada mode *Semua*, tabel tanpa potong baris (`break-inside: avoid`); cue sheet lagu memakai gaya yang sama. Ini menggantikan ekspor PDF.

### 10.2 Responsif

- < 768px: KPI grid 2×2; tabel agenda menjadi **Rundown Cue Cards** (badge waktu tebal di kiri atas, nama agenda besar, badge status 1-ketukan, tombol `💬 WA PIC` besar); mode Timeline tetap kartu satu kolom; Tab Audio menjadi kartu per lagu dengan tombol preview besar; `SessionTabs` dapat digulir horizontal; modal menjadi *bottom sheet*.
- Seret-lepas diganti tombol ▲▼ pada layar sentuh.

### 10.3 Aksesibilitas

- Badge status adalah `<button>` dengan `aria-label="Status {agenda}: {status}. Ketuk untuk {aksi berikutnya}"`; perubahan status diumumkan lewat wilayah `aria-live="polite"`. Seret-lepas selalu punya padanan keyboard (▲▼).
- Tab memakai `role="tablist"`/`tab`; sesi terpilih `aria-current`; modal menjebak fokus dan menutup dengan `Esc`; galat kolom terhubung lewat `aria-describedby`; target sentuh ≥ 44 px.
- Penukaran partial tidak merebut fokus: bila fokus ada di dalam area yang ditukar, fokus dipulihkan ke elemen dengan `data-id` yang sama.

### 10.4 Keamanan

- Teks pengguna (nama agenda, PIC, keterangan, cue, nama vendor) dirender dengan `textContent`/atribut Astro, **tidak pernah** `innerHTML` dengan data; tes sumber (bab 11.2) menjaganya.
- Tautan dari pengguna (`spotify_url`, `youtube_url`) dirender hanya bila `safeHref` (Menu 04) menerimanya (`http`/`https` sah), dengan `rel="noopener noreferrer"`. Tautan `wa.me` dibentuk dari `formatted_wa_number` (digit) dan `encodeURIComponent` untuk teks.
- Tidak ada unduhan berkas, jadi tidak ada risiko `Content-Disposition`/formula injection di modul ini. Teks WhatsApp dari server tidak di-escape ([BE bab 9 butir 6](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- Aksi tulis memakai BFF (JSON saja; `Content-Type: application/json` wajib, pertahanan CSRF). Pelaksana status adalah pemilik proyek; tidak ada tautan publik ke tracker.

---

## 11. Pengujian

### 11.1 Unit: `tests/planner-rundown*.test.mjs`

Tiga berkas, **19 tes, semuanya sudah dijalankan** (`node --experimental-strip-types --test`, Node 22.17) terhadap fixture:

| Berkas | Tes |
|---|---|
| `planner-rundown.test.mjs` (12) | `parseClock` (7 masukan sah, 10 tidak sah), `formatClock`/`durationLabel`/`agendaTimeLabel` (agenda terbuka 10.40 dan 09.15), `spanLabel` (445 → "7 jam 25 menit", 120, 100, 225, 59, 0), durasi Akad fixture = 20, 4, 9, 9, 4, 9, 9, 49, `groupBySession` (11/8/14, nomor mulai 1; sesi kosong tetap tampil), `normalizeTimes` (10 kasus, termasuk lewat tengah malam dan setiap penolakan), alasan `normalizeTimes` ada di katalog galat, `replaceImpact` (33/0 dan 17/16), `nextStatus`/`progressOf`, angka KPI fixture (33, 445, 17; 15/1/17), `filterSongs`/`momentTagOptions` |
| `planner-rundown-api.test.mjs` (4) | semua path lolos `^[a-z0-9/_-]+$`; method + path **sama dengan tabel endpoint BE bab 3** (17); query dan badan; setiap fungsi berparameter id melempar untuk id bukan UUID tanpa permintaan keluar |
| `planner-rundown-messages.test.mjs` (3) | setiap galat di katalog punya teks spesifik (kecuali milik 01 dan konfirmasi); teks khas (batas 150/255/2000, URL, wajib diisi, urutan lagu); `isConfirmRequired` |

**36 dari 36 mutasi** yang dicoba menjatuhkan tes (18 pada `rundown.ts`, 10 pada `rundownApi.ts`, 8 pada `rundownMessages.ts`: ambang jam dan menit, pengisi nol, cabang jam bulat pada `spanLabel`, label "selesai", penomoran, kedua arah aturan jam akhir dan aturan `ends_next_day` tanpa jam selesai, penghitung `reset`/`kept`, urutan status, argumen persen, huruf besar-kecil pencarian, penyaringan momen, pengurutan tag, path, metode, badan, validasi UUID, dan aturan pesan). Dua tes awalnya terlalu longgar (id tidak divalidasi pada `reorderRundowns` dan `deleteRundown` lolos; batas karakter hanya diuji untuk 255) dan diperketat. `tsc` strict dengan `noUncheckedIndexedAccess` bersih.

### 11.2 Tes sumber (`tests/planner-wiring.test.mjs`, perluasan dari 01 FE bab 13.3)

○ belum ada kodenya. Yang harus dijaga saat FE dibangun: halaman rundown memakai `PlannerLayout` dan tidak `AppLayout`; `rundown/live.astro` memuat `export const partial = true`; tidak ada `innerHTML =` dengan variabel di `src/components/planner/rundown/`; tidak ada string `export-pdf` atau `application/pdf` di modul; `MENUS` memuat `rundown` dengan `ready: true`; setiap fungsi `rundownApi` yang dipakai komponen ada di tabel bab 4.4.

### 11.3 Uji komponen dengan fixture (manual atau skrip)

○ Render `RundownTable` dari `planner-rundowns.json` (3 tabel 11/8/14, dua agenda terbuka "–"), `RundownKpi` dari fixture sampel dan `planner-rundown-live.json` (15/1/17, 45%), `AudioKpi` dari `planner-playlists.json`, dan keadaan kosong (tanpa sesi, tanpa agenda, tanpa lagu).

### 11.4 QA manual (terhadap backend asli)

○ Dijalankan saat BE ada: 1) Proyek Contoh: 33 agenda, 445 menit, 17 PIC, 30 lagu; 2) ketuk badge dua perangkat: perubahan terbaca ≤ 5 detik, dua agenda LIVE bersamaan; 3) agenda lewat tengah malam (23.30 → 00.30) durasi 60 dan rentang sesi benar; 4) agenda terbuka tampil "selesai"; 5) tautkan vendor: tombol WA membuka `wa.me/62…`; 6) tautkan lagu ke agenda: badge `🎵` dan kolom Agenda; 7) `replace_template`: dialog, agenda berjalan tidak berubah, vendor dan tautan lagu tetap; 8) hapus agenda bertaut lagu: lagu lepas tautan; 9) hapus sesi di Kelola Sesi: `session_in_use` dengan jumlah agenda; 10) salin WA satu sesi dan semua; 11) cetak mode Semua; 12) proyek arsip: semua kontrol nonaktif; 13) sembilan jenis acara: template agenda sesuai TEMPLATE_EVENT, Tab Audio kosong kecuali pernikahan, moment tag dari registry; 14) layar < 768 px: kartu cue.

### 11.5 Verifikasi kontrak (cara mengulang)

Skrip audit tidak disimpan di repo; caranya: (1) buat database sementara berakhiran `_test`, `go run ./cmd/migrate up`, masukkan agenda dan lagu dari [BE bab 5](BE.md#5-master-data) (blok SQL di BE bab 2 dijalankan apa adanya); (2) ekstrak blok Go bab 4 BE ke modul sementara, `go vet`, lalu decode fixture dengan `DisallowUnknownFields` dan bandingkan serialisasi ulang; (3) ekstrak kode bab 4.2, 5, 6, dan 9.3 ke berkas, `tsc --noEmit` strict, salin fixture ke `tests/fixtures/`, `npm test`; (4) bandingkan kunci interface TypeScript dengan tag JSON struct Go. Hasil 2026-10-31: DTO bersih, fixture nol selisih, 23 pasangan tipe / 118 jalur / 0 selisih, 19 tes lulus. Respons HTTP asli sudah dicocokkan dengan fixture di sisi backend ([BE bab 7.3](BE.md#73-hasil-implementasi)); yang tersisa untuk FE: jalankan `npm test` terhadap backend yang hidup (QA bab 11.4) dan catat hasilnya di sini.

---

## 12. Urutan pengerjaan dan definisi selesai

### 12.1 Menjalankan lingkungan lokal

Sama dengan [05 FE bab 12.1](../05_BUKU_TAMU/FE.md#121-menjalankan-lingkungan-lokal): Compose atau `go run ./cmd/api` + `npm run dev` (proxy `API_BASE_URL`), pengguna uji, Proyek Contoh lewat `POST /v1/planner/projects/sample`. Menu 06 membutuhkan backend modul `plannerrundown` ([BE bab 8](BE.md#8-urutan-pengerjaan-dan-definisi-selesai)); tanpanya halaman hanya dapat diuji terhadap fixture.

### 12.2 Tahapan

| # | Tahap | Isi | Uji |
|---|---|---|---|
| M0 | Prasyarat | bab 3 ada (01 dan 04 FE) | – |
| M1 | Kontrak | tambahkan tipe bab 4.2, `rundown.ts`, `rundownApi.ts`, `rundownMessages.ts`, fixture ke `tests/fixtures/` | `npm test` (19 tes), `tsc` |
| M2 | Tab 1 baca | halaman, `RundownKpi`, `SessionTabs`, `RundownTable`, `LiveStatusBadge` (tanpa aksi) terhadap fixture, lalu BE | bab 11.3 dan 11.4 butir 1, 4 |
| M3 | Tab 1 tulis | `RundownModal`, status, hapus, urut ulang, `TemplateMenu`, `ConfirmReplaceDialog`, `WaPreviewModal` | bab 11.4 butir 2, 3, 5–10 |
| M4 | Polling dan partial | `rundown/live.astro`, `rundownClient.ts` (interval, tanda tangan, mundur) | bab 11.4 butir 2 |
| M5 | Timeline dan cetak | `RundownTimeline`, stylesheet cetak | bab 11.4 butir 11, 14 |
| M6 | Tab 2 | `AudioKpi`, `SongFilters`, `PlaylistTable`, `PlaylistModal`, template lagu, WA soundman | bab 11.4 butir 6, 13 |
| M7 | Selesai | tes sumber bab 11.2, QA bab 11.4, `astro check`, sub-navigasi `ready` | semua |

**Definisi selesai:** `npm run build` (termasuk `astro check`) lolos; 19 tes unit dan tes sumber lulus; QA bab 11.4 terhadap backend asli; fixture dicocokkan dengan respons asli (di sisi backend sudah; di sisi FE: `npm test` terhadap backend yang hidup); tidak ada dependensi baru; dokumen ini memuat setiap penyimpangan dari rancangan.

---

## 13. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–05 (dokumen), kontrak, PDF, dan kode acuan yang dijalankan; keputusan backend ada di [BE bab 9](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk).

1. **Fixture dibangkitkan dari SQL audit** dan sudah dicocokkan dengan respons HTTP asli di sisi backend (dua dikoreksi); yang belum: tes TypeScript terhadap backend yang hidup.
2. **Interval polling** 5 detik saat acara berlangsung atau ada agenda LIVE, 30 detik selain itu: angka usulan, belum diukur; beberapa tablet pada satu akun tidak dibatasi.
3. **Rollback status optimistik** menampilkan status server sesudah galat; bila dua panitia mengetuk badge yang sama hampir bersamaan, yang terakhir menang (satu pernyataan SQL, tanpa versi/ETag).
4. **Cetak lewat peramban** (bukan PDF): tata letak A4/A5 belum diuji di peramban dan printer nyata; hasil tergantung pengaturan cetak pengguna.
5. **Pencarian lagu di klien** (daftar ≤ 500, tanpa paginasi); bila batas dinaikkan, pencarian perlu pindah ke server.
6. **Pilihan vendor** memuat seluruh vendor proyek (≤ 200) saat modal dibuka; tanpa pencarian.
7. **Tombol "Buka WhatsApp"** pada pratinjau mengirim teks lewat `wa.me/?text=`; teks panjang (rundown semua sesi) dapat terpotong oleh batas URL peramban atau aplikasi, sehingga "Salin" adalah jalur utama.
8. **Sesi tanpa tanggal** (mis. sesi buatan pengguna yang tanggalnya dikosongkan; sesi Lamaran Proyek Contoh bertanggal 15 Okt 2026) tampil "Tanggal belum diisi"; teks WhatsApp satu sesi memakai `event_date` ([BE bab 2.4](BE.md#24-pesan-whatsapp)), yang bisa berbeda hari dari sesi lain.
9. **Agenda lewat tengah malam**: tampilan menampilkan jam akhir apa adanya ("00.30") tanpa penanda hari, dan agenda yang mulai sesudah tengah malam belum didukung ([BE bab 9 butir 3](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
10. **Tidak ada ekspor PDF**: README dan dokumen lama menjanjikannya; hanya cetak browser sampai pustaka PDF disetujui.
11. **Angka dialog `replace_template` adalah perkiraan klien.** `replaceImpact` menghitung semua agenda ber-`template_key` yang `pending`; server tidak me-reset agenda template yang sudah dipindahkan pengguna ke sesi lain (dihitung `skipped`, [BE bab 9 butir 21](BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)), jadi dialog dapat menyebut satu atau beberapa agenda lebih banyak daripada `reset` yang dijawab server. Pesan hasil ("{reset} agenda dikembalikan") memakai angka server, jadi yang ditampilkan sesudahnya tetap benar. Alternatif: FE tidak menampilkan angka di dialog, hanya penjelasan.

---

## 14. Riwayat perubahan dokumen ini

- **2026-10-31:** ditulis ulang mengikuti pola Menu 04 dan 05 setelah audit: koreksi wireframe (durasi), layout (`PlannerLayout`), cetak browser menggantikan PDF, pesan WhatsApp dari server, partial untuk polling, kontrak data (tipe, fixture, katalog endpoint dan galat), kode `rundown.ts`/`rundownApi.ts`/`rundownMessages.ts` yang dijalankan, pengujian, urutan kerja, dan celah.
- **2026-10-31 (sesudah backend dibangun):** pernyataan yang menjadi basi dikoreksi: fixture kini sama dengan respons HTTP asli, contoh hasil operasi dan katalog galat dijawab server sungguhan, sesi Lamaran Proyek Contoh bertanggal; teks keadaan kosong playlist tidak lagi memuat angka 30 (bergantung pada `templates_available.playlist`, bukan jenis acara); celah 11 (angka dialog `replace_template`).
- Versi awal (2026-10-21): wireframe dan perilaku tanpa kontrak data.
