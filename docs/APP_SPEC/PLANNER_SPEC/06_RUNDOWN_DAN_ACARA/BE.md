# Menu 06: Rundown & Acara — Backend Specification

Spesifikasi backend modul Go `plannerrundown` (`backend/internal/modules/plannerrundown/`) untuk **susunan rundown menit demi menit** (PDF Hal. 14) dan **audio/playlist** (PDF Hal. 15–16). Tabel sesi (`planner_event_sessions`) milik modul `planner` sebagai master bersama ([01 BE](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi), [kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)); modul ini menambah agenda dan lagu di atasnya. Aturan lintas modul ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

> **Status (2026-10-31):** modul `plannerrundown` **sudah diimplementasikan** (`backend/internal/modules/plannerrundown/`, 17 endpoint, provider `rundown`, applier `rundown` dan `playlist`, `replace_template` di tempat; rangkuman dan penyimpangan: [docs/backend/planner/2026-10-31-planner-menu-06.md](../../../backend/planner/2026-10-31-planner-menu-06.md)). Migrasi `012_000_planner_rundown.sql` ditambah dua indeks unik `template_key` di tempat dan blok DDL bab 1 tetap **identik** dengannya; Menu 01 mendapat `ModeReplaceTemplate`, `ApplyRequest.Confirm`, dan `moment_tags`. Respons HTTP asli **sama dengan kelima fixture FE** setelah id dan waktu dinormalkan ([bab 7.3](#73-hasil-implementasi)); dua fixture dikoreksi saat dicocokkan ([bab 9 butir 17 dan 18](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Frontend dan BFF belum ada ([FE.md](FE.md)). Spesifikasi ini ditulis ulang mengikuti pola Menu 04 dan 05 setelah **diaudit dengan menjalankannya** terhadap PostgreSQL 16.2 dan membaca ulang PDF Hal. 14–16 dengan `pdfminer` ([bab 7.1](#71-yang-sudah-dijalankan-pada-audit-spesifikasi)); setelah itu dibangun pada hari yang sama. Yang ternyata benar pada audit: DDL, 33 agenda dan 30 lagu (cocok dengan PDF tanpa selisih), rentang 100 + 120 + 225 = **445 menit**, 17 PIC unik. Yang salah atau kurang pada versi sebelumnya (dikoreksi sebelum kode ditulis):
> - **DTO Go tidak dapat dikompilasi** (`Playlist` dipakai tetapi tidak didefinisikan), struct bersarang tanpa tag JSON menghasilkan kunci `TotalAgendas`/`ID`, daftar kosong menjadi `null`, dan `ProgressPercent` melanggar kontrak "backend tidak mengirim persentase"; query §2.2 juga tidak memilih empat kolom yang ada di DTO. DTO kini dikompilasi dan fixture FE dicocokkan dengannya ([bab 4](#4-dto-go)).
> - **Rumus rentang salah untuk acara yang melewati tengah malam** (acara 22.00–23.00 dan 23.30–00.30 berentang 150 menit; rumus lama menghasilkan 1.500 atau 60). Aturan `ends_next_day` hanya sebagian dijaga basis data ([bab 2.1](#21-waktu-durasi-dan-rentang)).
> - **Ekspor PDF dijanjikan** (dua endpoint) padahal pustaka PDF belum disetujui ([kontrak 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)); kini dihapus, cetak lewat stylesheet browser seperti Menu 05 ([bab 2.4](#24-pesan-whatsapp)).
> - **`replace_template` memutus tautan lagu↔agenda tanpa peringatan** (2 menjadi 0 pada percobaan) dan tidak punya `confirm`, tidak seperti Menu 02–04; kini reset-di-tempat yang mempertahankan id, vendor, dan tautan ([bab 2.6](#26-template)).
> - **Migrasi 012 tidak punya indeks unik `(project_id, template_key)`** yang dimiliki tugas, dokumen, dan belanja; pola `append` Menu 02–04 (`ON CONFLICT … DO NOTHING`) gagal tanpanya ([bab 1.1](#11-indeks-unik-template_key-ditambahkan-ke-migrasi-012-saat-modul-dibangun)). Kedua indeks kini ada di migrasi 012.
> - Tanpa batas jumlah, bentuk respons, tabel validasi, kontrak urut ulang, dan bagian konkurensi (ada di Menu 04 dan 05); pesan WhatsApp memakai `highlights` padahal contoh FE memakai keterangan; durasi di wireframe FE salah pada enam dari delapan baris Akad (07.26–07.35 tertulis 10m, server mengirim 9; 07.21–07.25 tertulis 5m, server mengirim 4).
>
> **Belum dibangun dan belum dijalankan:** frontend, BFF, cetak browser, teks WhatsApp yang dibuka di WhatsApp, dan perilaku di bawah beban (daftar 500 agenda, polling dari beberapa tablet). Blok SQL dan DTO di dokumen ini kini dijalankan oleh kode modul ([bab 7.2](#72-pengujian-pada-implementasi)); bab 7.1 tetap catatan audit sebelum kode ada.

---

## 1. Skema Database

Migrasi `012_000_planner_rundown.sql` (**identik** dengan blok di bawah; dibandingkan per karakter, termasuk dua indeks unik `template_key` yang ditambahkan saat modul dibangun; naik bersama 001–012 pada PostgreSQL 16.2; sesi ada di migrasi 005, vendor di 006):

```sql
-- +goose Up
CREATE TABLE planner_rundowns (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME,                                      -- NULL = "selesai" (terbuka), mis. "10.40 - selesai"
    ends_next_day BOOLEAN NOT NULL DEFAULT FALSE,
    duration_minutes INT GENERATED ALWAYS AS (
        CASE WHEN end_time IS NULL THEN NULL
             ELSE (EXTRACT(EPOCH FROM (end_time - start_time)) / 60)::int + CASE WHEN ends_next_day THEN 1440 ELSE 0 END
        END
    ) STORED,
    activity_name TEXT NOT NULL CHECK (char_length(activity_name) BETWEEN 1 AND 255),
    pic_name TEXT NOT NULL DEFAULT '',
    vendor_id UUID,
    description TEXT NOT NULL DEFAULT '',
    highlights TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    sort_order INT NOT NULL DEFAULT 0,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    CHECK (end_time IS NULL OR ends_next_day OR end_time >= start_time),
    FOREIGN KEY (session_id, project_id)
        REFERENCES planner_event_sessions (id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id)
);

CREATE INDEX planner_rundowns_session_sort_idx ON planner_rundowns (session_id, sort_order);
CREATE INDEX planner_rundowns_project_status_idx ON planner_rundowns (project_id, status);
CREATE INDEX planner_rundowns_vendor_idx ON planner_rundowns (vendor_id) WHERE vendor_id IS NOT NULL;
CREATE UNIQUE INDEX planner_rundowns_template_key_idx ON planner_rundowns (project_id, template_key) WHERE template_key IS NOT NULL;

CREATE TABLE planner_playlists (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    rundown_id UUID,
    song_title TEXT NOT NULL CHECK (char_length(song_title) BETWEEN 1 AND 255),
    artist TEXT NOT NULL CHECK (char_length(artist) BETWEEN 1 AND 255),
    moment_tag TEXT NOT NULL DEFAULT 'Lainnya',
    audio_format TEXT NOT NULL DEFAULT 'digital_playback' CHECK (audio_format IN ('digital_playback', 'live_band', 'acoustic', 'sound_fx')),
    spotify_url TEXT,
    youtube_url TEXT,
    audio_cue_notes TEXT,
    status TEXT NOT NULL DEFAULT 'needs_review' CHECK (status IN ('ready', 'needs_review', 'missing_file')),
    sort_order INT NOT NULL DEFAULT 0,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (rundown_id, project_id)
        REFERENCES planner_rundowns (id, project_id) ON DELETE SET NULL (rundown_id)
);

CREATE INDEX planner_playlists_project_sort_idx ON planner_playlists (project_id, sort_order);
CREATE INDEX planner_playlists_rundown_idx ON planner_playlists (rundown_id) WHERE rundown_id IS NOT NULL;
CREATE UNIQUE INDEX planner_playlists_template_key_idx ON planner_playlists (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_playlists;
DROP TABLE IF EXISTS planner_rundowns;
```

Keputusan desain:
- Jam bertipe **`TIME`**, bukan `VARCHAR(10)`; `end_time` boleh `NULL` karena PDF memuat agenda terbuka ("10.40 - selesai") dan agenda satu-jam ("09.15"). Dengan begitu dua baris PDF itu tersimpan apa adanya (dijalankan: 33 agenda masuk, 2 dengan `end_time NULL` dan `duration_minutes NULL`). `duration_minutes` adalah kolom generated: **selisih jam tanpa `+1`** (07.21–07.25 = **4** menit, 08.11–09.00 = 49, 07.46–07.50 = 4).
- **Hanya satu aturan `ends_next_day` yang dijaga basis data**: `end_time < start_time` tanpa `ends_next_day` ditolak oleh `CHECK` (dijalankan). Sisanya diterima basis data: `ends_next_day` dengan `end_time > start_time` (durasi 1.500 menit), dengan `end_time = start_time` (1.440), dan dengan `end_time NULL`. Aturannya karena itu di usecase ([bab 2.1](#21-waktu-durasi-dan-rentang)).
- `session_id` dan `vendor_id` memakai FK komposit `(…, project_id)`: agenda tidak dapat menunjuk sesi atau vendor proyek lain (dijalankan: `violates foreign key constraint "planner_rundowns_session_id_project_id_fkey"`). Hapus vendor mengosongkan `vendor_id` saja (`SET NULL (vendor_id)`; `project_id` tetap); hapus sesi menghapus agendanya (`CASCADE`; dijalankan: Resepsi dihapus, 33 → 19 agenda).
- `started_at`/`completed_at` mencatat kapan tiap agenda mulai/selesai (tracker live). **Basis data tidak menjaga kesesuaian `status` dengan kedua cap itu** (dijalankan: `completed` tanpa `completed_at` dan `pending` dengan `started_at` diterima); satu pernyataan SQL di [bab 2.3](#23-live-stage-manager) adalah satu-satunya jalan menulis `status`.
- Hapus agenda mengosongkan `planner_playlists.rundown_id` saja (`SET NULL (rundown_id)`; dijalankan). Tautan lagu↔agenda karena itu **hilang diam-diam** pada setiap penghapusan agenda, termasuk yang dilakukan penghapusan sesi; `replace_template` dirancang agar tidak menghapus agenda ([bab 2.6](#26-template)).
- `template_key` untuk mode *append* (dengan indeks unik parsial, bab 1.1); playlist default `needs_review` (PDF tidak memuat format atau kesiapan audio), bukan `ready`.
- Tabel sesi, ikon, dan seeder sesi per jenis acara ada di modul `planner` (01).

### 1.1 Indeks unik `template_key` (ditambahkan ke migrasi 012 saat modul dibangun)

```sql
CREATE UNIQUE INDEX planner_rundowns_template_key_idx ON planner_rundowns (project_id, template_key) WHERE template_key IS NOT NULL;
CREATE UNIQUE INDEX planner_playlists_template_key_idx ON planner_playlists (project_id, template_key) WHERE template_key IS NOT NULL;
```

`planner_tasks`, `planner_admin_docs`, dan `planner_shopping_items` sudah punya indeks serupa (`…_template_key_idx`, migrasi 007 dan 009) dan `append` mereka memakai `INSERT … ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`. Pada migrasi 012 **sebelum** kedua indeks ini ada, pernyataan itu gagal dengan `there is no unique or exclusion constraint matching the ON CONFLICT specification` dan sisipan duplikat `template_key` yang sama **diterima** (dua baris); dengan indeks, pernyataan yang sama menjawab `INSERT 0 0`. Keduanya kini ada di berkas migrasi (diubah di tempat pada tahap pengembangan; migrasi belum dirilis ke `main`; database yang sudah menjalankan 012 lama harus `migrate fresh`). Dijaga `TestRundownTemplateKeyIndexesMakeAppendIdempotent` (`internal/migrate`) dan, lewat HTTP, `TestAppendConcurrentlyCreatesOnce` (empat `apply-template` serentak menghasilkan tepat 33 agenda). Constraint lain yang bisa ditambahkan tetapi tidak diterapkan: kesesuaian `status` dengan `started_at`/`completed_at` dan `ends_next_day` (dijaga usecase).

### 1.2 Tabel milik modul lain yang disentuh

| Tabel | Pemilik | Akses modul ini |
|---|---|---|
| `planner_event_sessions` | Menu 01 | baca (`JOIN` nama, ikon, tanggal, lokasi, `scope_key`); tidak pernah ditulis. FK komposit dan `CASCADE` dari hapus sesi ada di basis data |
| `planner_vendors` | Menu 04 | baca (`LEFT JOIN` nama, kategori, PIC, telepon, `formatted_wa_number`); tidak pernah ditulis |
| `planner_projects` | Menu 01 | kunci baris `SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE` (bab 2.9), tidak ditulis |

Modul ini **tidak menulis ke tabel modul lain** dan tidak menambah pengecualian di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture); tambahan pada pembacaan lintas modul yang sudah ada ("Menu 02, 03, dan 06 `JOIN planner_vendors`") tidak diperlukan.

---

## 2. Logika Bisnis

### 2.1 Waktu, durasi, dan rentang

Jam di JSON `"HH:MM"`. Masukan `07.46`, `7.46`, `07:46`, dan `7:46` diterima server (`^([01]?[0-9]|2[0-3])[:.]([0-5][0-9])$` setelah `trim`) dan disimpan sebagai `07:46`; detik, `24:00`, dan `07.6` ditolak `400 invalid_input: start_time: must be a time like 07:46 or 07.46` (kolom basis data menerima detik, jadi pembatasan ini milik usecase). Penyajian memakai titik (`07.46`, seperti PDF).

**Aturan jam akhir** (usecase; hanya baris pertama yang juga dijaga `CHECK`):

| Masukan | Aturan | Galat (`invalid_input`) |
|---|---|---|
| `end_time` kosong | agenda terbuka; `ends_next_day` harus `false` | `ends_next_day: requires end_time` |
| `end_time ≥ start_time`, `ends_next_day = false` | sah (sama = agenda 0 menit/penanda) | – |
| `end_time < start_time`, `ends_next_day = false` | ditolak | `end_time: must not be before start_time unless ends_next_day is set` |
| `ends_next_day = true` | `end_time` harus **lebih awal** dari `start_time` (23.30 → 00.30 = 60 menit); sama atau lebih akhir ditolak | `end_time: must be earlier than start_time when ends_next_day is set` |

**Rentang (KPI), bukan jumlah durasi.** Agenda paralel (sesi makan, hiburan, salaman) dan yang tumpang tindih tidak boleh dijumlahkan: pada data PDF jumlah durasi 699 menit, rentang 445. Per sesi `span = max(akhir mutlak) − min(mulai)`; total = Σ rentang sesi. **Jam dihitung sebagai menit** (`EXTRACT(EPOCH …)/60`), bukan aritmetika `time`: `'00:30'::time + interval '24 hours'` tetap 00:30, sehingga rumus yang menambah 24 jam pada tipe `time` salah (dijalankan: 60 menit, seharusnya 150). Akhir mutlak = `COALESCE(end_time, start_time)` menit + 1.440 bila `ends_next_day`:

```sql
-- $1 = project_id. Dijalankan pada Proyek Contoh: lamaran 100, akad 120, resepsi 225 (total 445).
SELECT s.id AS session_id,
       (max(EXTRACT(EPOCH FROM COALESCE(r.end_time, r.start_time))/60 + CASE WHEN r.ends_next_day THEN 1440 ELSE 0 END)
        - min(EXTRACT(EPOCH FROM r.start_time)/60))::int AS span_minutes
FROM planner_event_sessions s JOIN planner_rundowns r ON r.session_id = s.id
WHERE s.project_id = $1 GROUP BY s.id;
```

Data PDF: Lamaran 09.00–10.40 = 100 menit (agenda 11 terbuka dihitung dari jam mulainya), Akad 07.00–09.00 = 120, Resepsi 09.15–13.00 = 225 (agenda 9 terbuka mulai 09.15) → **445 menit (7 jam 25 menit)**. Acara lewat tengah malam diuji: A 22.00–23.00 dan B 23.30–00.30 (`ends_next_day`) berentang **150**; durasi kolom generated untuk B = 60 (benar).

**Batas yang diketahui:** agenda yang *mulai* sesudah tengah malam tidak dapat dimodelkan, karena `start_time` tidak membawa hari; menambahkan agenda 00.15–00.45 pada sesi di atas membuat rentang 1.455 menit (seharusnya 165; dijalankan). Keputusan ada di [bab 9 butir 3](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk); sampai itu diputuskan, acara yang melewati tengah malam hanya boleh punya agenda yang *berakhir* sesudahnya.

**PIC unik** (kartu KPI) = jumlah pelaksana berbeda: untuk tiap agenda yang punya vendor atau `pic_name` tak kosong, `vendor_id` bila ada, selain itu `pic_name` setelah `lower(btrim())`:

```sql
-- $1 = project_id. Proyek Contoh: 17. Tiga agenda MC ditautkan ke satu vendor: 18 (teks "MC" masih dipakai agenda lain).
SELECT count(DISTINCT CASE WHEN vendor_id IS NOT NULL THEN 'v:' || vendor_id::text ELSE 'p:' || lower(btrim(pic_name)) END)
FROM planner_rundowns WHERE project_id = $1 AND (vendor_id IS NOT NULL OR btrim(pic_name) <> '');
```

### 2.2 PIC dan kontak vendor

`GET /rundowns` menggabungkan `planner_vendors` agar panitia dapat langsung membuka WhatsApp PIC dari baris agenda (dijalankan; kolom vendor ada di migrasi 006):

```sql
-- $1 = project_id, $2 = session_id atau NULL. Hanya meja yang disaring sesi; kpi dan sesi selalu seluruh proyek (bab 2.8).
SELECT r.id, r.session_id, to_char(r.start_time,'HH24:MI') AS start_time, to_char(r.end_time,'HH24:MI') AS end_time,
       r.ends_next_day, r.duration_minutes, r.activity_name, r.pic_name, r.description, r.highlights,
       r.status, r.started_at, r.completed_at, r.sort_order, r.template_key,
       s.name AS session_name, s.icon AS session_icon, s.scope_key,
       v.id AS vendor_id, v.vendor_name, v.category_label, v.pic_name AS vendor_pic,
       v.phone_number AS vendor_phone, v.formatted_wa_number AS vendor_wa,
       (SELECT count(*) FROM planner_playlists p WHERE p.rundown_id = r.id AND p.project_id = r.project_id) AS linked_songs
FROM planner_rundowns r
JOIN planner_event_sessions s ON s.id = r.session_id
LEFT JOIN planner_vendors v ON v.id = r.vendor_id AND v.project_id = r.project_id
WHERE r.project_id = $1 AND ($2::uuid IS NULL OR r.session_id = $2)
ORDER BY s.sort_order, r.sort_order, r.start_time;
```

Versi lama tidak memilih `ends_next_day`, `started_at`, `completed_at`, dan `linked_songs` padahal DTO `Rundown` memuatnya. Tautan WhatsApp dibentuk klien dari `formatted_wa_number` ([04 BE §2.3](../04_VENDOR_DAN_BELANJA/BE.md#23-normalisasi-nomor-whatsapp)); agenda tanpa vendor tidak punya tombol WA (teks PIC bebas bukan nomor).

### 2.3 Live Stage Manager

`PATCH …/rundowns/:rundown_id/status` `{"status": "pending" | "in_progress" | "completed"}` adalah **satu pernyataan** untuk ketiga status dan semua perpindahan (dijalankan untuk `pending → in_progress → completed → in_progress → pending`):

```sql
-- $1 = id, $2 = project_id, $3 = status.
UPDATE planner_rundowns SET
  status       = $3,
  started_at   = CASE $3 WHEN 'pending'   THEN NULL ELSE COALESCE(started_at, now()) END,
  completed_at = CASE $3 WHEN 'completed' THEN COALESCE(completed_at, now()) ELSE NULL END,
  updated_at   = now()
WHERE id = $1 AND project_id = $2
RETURNING id, status, started_at, completed_at;
```

Invarian yang dihasilkan: `pending` ⇒ kedua cap kosong; `in_progress` ⇒ `started_at` terisi, `completed_at` kosong (membuka kembali agenda selesai mempertahankan jam mulainya); `completed` ⇒ keduanya terisi; mengirim status yang sama lagi tidak mengubah cap. **Semua perpindahan diizinkan** (urutan Belum → LIVE → Selesai hanya aturan tampilan FE, supaya koreksi saat panik tetap mungkin). Beberapa agenda boleh `in_progress` bersamaan (sesi makan, hiburan, dan salaman PDF memang paralel). `PUT` agenda **tidak menulis `status`** (tidak ada di badan permintaan), sehingga cap dan status tidak dapat berselisih lewat jalur lain. Hanya pemilik proyek yang dapat mengubah status (`ProjectGuard`); proyek `archived` menjawab `409 project_read_only`. Pembaruan antarperangkat lewat polling 5 detik `GET …/rundowns/summary` (bukan SSE); jawabannya membawa `server_time` (UTC) supaya klien dapat menampilkan "berjalan N menit" tanpa mengandalkan jam perangkat.

### 2.4 Pesan WhatsApp

`GET …/rundowns/export-wa?session_id=` menyusun teks terstruktur yang **netral jenis acara** di server (satu ikon per jenis dari registry; tidak ada pemetaan ikon kedua di modul ini). Berbeda dari Menu 04 dan 05 yang membentuk pesan di klien, teks ini membutuhkan seluruh agenda dan dipakai apa adanya oleh tombol salin, sehingga satu sumbernya adalah server ([bab 9 butir 5](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Aturan:

- Kepala: `📋 *RUNDOWN ACARA {NAMA SESI HURUF BESAR}*` (tanpa `session_id`: `📋 *RUNDOWN ACARA*`), `{ikon jenis} *{judul proyek}*`, `🗓 *{hari}, {tanggal}*` dari `session_date` sesi (kosong → `event_date`; tanpa `session_id`, tanggal ada di kepala tiap sesi), `🏛 *{lokasi}*` dari `location_name` sesi, lalu `metadata.venue_name` dan `metadata.city` digabung `", "`. Baris yang datanya kosong dihilangkan.
- Tiap agenda: `⏰ *HH.MM - HH.MM* (N mnt) : Agenda`; agenda terbuka `⏰ *HH.MM - selesai* : Agenda` (tanpa durasi); `   👤 PIC: …` bila ada; `   📝 {keterangan}` bila ada; tiap butir *highlights* `   • …` (awalan "- " dibuang). Satu baris kosong antar agenda.
- Tanpa `session_id`: tiap sesi diawali `*{ikon} {NAMA SESI}*` (dan tanggalnya bila ada), dipisah garis `────────────────────────`. Sesi tanpa agenda dilewati.
- Penutup `(Dibuat otomatis melalui Adicara Event Planner)`. Teks pengguna tidak di-escape: tanda `*` atau `_` di dalam nama agenda dapat memformat teks di WhatsApp ([bab 9 butir 6](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

Contoh (sesi Akad Nikah Proyek Contoh; keluaran penuh ada di [fixtures/planner-rundown-wa-akad.json](fixtures/planner-rundown-wa-akad.json), dibangkitkan dari data DB audit dengan aturan di atas; teks yang dihasilkan modul sama dengan berkas itu, `TestAgendaTextOfTheAkadIsTheFixture` dan `TestHariHLiveAndWhatsAppMatchTheFixtures`):

```text
📋 *RUNDOWN ACARA AKAD NIKAH*
💍 *Contoh: Pernikahan Sinta & Arif*
🗓 *Minggu, 28 Maret 2027*
🏛 *Sasana Kriya Ballroom, Jakarta Selatan*
────────────────────────
⏰ *07.00 - 07.20* (20 mnt) : Persiapan
   👤 PIC: CMP & CMW
   📝 Penghulu datang dan melakukan pemeriksaan kelengkapan persyaratan administrasi, kelengkapan dan kesiapan calon pengantin, serta kehadiran wali dan saksi.

…

⏰ *07.46 - 07.50* (4 mnt) : Ijab Kabul
   👤 PIC: Penghulu
   📝 Ijab kabul yang dipimpin oleh penghulu dan dilakukan dan dilakukan oleh wali nikah.

…

⏰ *08.11 - 09.00* (49 mnt) : Selesai akad
   👤 PIC: MC
   📝 Persiapan melanjutkan ke acara resepsi.
────────────────────────
(Dibuat otomatis melalui Adicara Event Planner)
```

Versi lama memakai `highlights` untuk baris 📝, padahal agenda Akad dan Resepsi PDF tidak punya kolom highlights (hanya Lamaran), sehingga contoh pada FE tidak akan pernah dihasilkan. **Cetak** (cue card dan sound cue sheet) memakai `window.print()` dengan stylesheet cetak di FE; **tidak ada ekspor PDF** sampai pustaka PDF disetujui (versi lama mencantumkan `export-pdf` untuk agenda dan playlist).

### 2.5 Urut ulang

`POST …/rundowns/reorder` `{"session_id": "…", "ids": ["…"]}`: `ids` harus **tepat semua agenda sesi itu, masing-masing sekali**. Usecase (di bawah kunci proyek) memeriksa jumlahnya sama dengan jumlah agenda sesi dan tiap id milik sesi itu; selain itu `400 invalid_input: ids: must list every agenda of the session exactly once`. Pemeriksaan ini wajib: dijalankan, id kembar dalam daftar **lolos tanpa galat** (satu baris menerima salah satu urutan), dan daftar yang tidak lengkap meninggalkan `sort_order` bentrok.

```sql
-- $1 = ids, $2 = project_id, $3 = session_id. Dijalankan: sesi Akad dibalik (8 baris), sesi lain tidak tersentuh.
UPDATE planner_rundowns r SET sort_order = u.ord, updated_at = now()
FROM (SELECT unnest($1::uuid[]) AS id, generate_series(1, cardinality($1::uuid[])) AS ord) u
WHERE r.id = u.id AND r.project_id = $2 AND r.session_id = $3;
```

`POST …/playlists/reorder` `{"ids": [...]}` memakai pola yang sama untuk seluruh lagu proyek (tanpa lingkup sesi). Daftar menyajikan agenda per sesi, jadi urutan antar sesi mengikuti urutan sesi (`planner_event_sessions.sort_order`), bukan `sort_order` agenda.

### 2.6 Template

Dua applier terdaftar lewat `RegisterTemplateAppliers` ([kontrak 1.1.1](../00_KONTRAK_BERSAMA/README.md#111-transaksi-dan-template-lintas-modul)): `rundown` dan `playlist`. Wizard memanggil keduanya dengan `append` di dalam transaksi pembuatan proyek; pengguna memanggil `POST …/rundowns/apply-template` dan `POST …/playlists/apply-template` `{mode, confirm?}`. **`ApplyMode` di `planner/domain` punya `ModeAppend` dan `ModeReplaceTemplate`, dan `ApplyRequest` punya `Confirm`**: keduanya ditambahkan saat modul ini dibangun, bersama `moment_tags` (perubahan di Menu 01, bab 6); wizard memakai `append` saja.

**Pencocokan sesi** lewat `scope_key` (`lamaran`, `akad`, `resepsi`, `prosesi_doa`, …): sesi yang sudah dihapus pengguna dilewati dengan warning `scope_missing` (`count` = agenda yang dilewati). **Kunci template**: pernikahan `wedding.lamaran_NN`, `wedding.rundown_NN`, `wedding.lagu_NN` (bab 5); delapan jenis lain `<kunci jenis>.agenda_<NN>` dengan `NN` = urutan baris pada tabel agenda jenis itu di [TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (dua digit; mis. `seminar.agenda_03`); versi lama tidak mendefinisikannya.

- **`append`** (bawaan bila `mode` kosong): menyisipkan agenda yang `template_key`-nya belum ada. Idempoten: `INSERT … ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING RETURNING id` (memakai indeks bab 1.1; dijalankan: `INSERT 0 0` untuk kunci yang sudah ada). `created` = baris masuk, `skipped` = sisanya. Agenda template berstatus `pending`, tanpa vendor, `sort_order` melanjutkan `max + 1` pada sesinya menurut urutan template. Tidak memerlukan `confirm`.
- **`replace_template`** (hanya agenda): **reset di tempat**, bukan hapus-lalu-sisip, dalam satu transaksi di bawah kunci proyek: (1) agenda template yang masih `pending` dikembalikan ke isi template (`start_time`, `end_time`, `ends_next_day`, nama, `pic_name`, keterangan, highlights, `sort_order`) dengan `UPDATE … FROM (template) WHERE project_id = $1 AND template_key = t.template_key AND status = 'pending' AND session_id = <sesi dari scope_key>`; (2) agenda template yang tidak ada lagi dibuat seperti `append`; (3) agenda `in_progress`/`completed` **tidak disentuh** dan dihitung `kept`. **`id`, `vendor_id`, dan tautan lagu tetap** (dijalankan: agenda 12 yang diubah pengguna kembali ke "Iring-iring", `id` sama, vendor tetap, 1 lagu tetap tertaut; agenda berjalan dan selesai yang namanya diubah tidak berubah). Agenda buatan pengguna (tanpa `template_key`) tidak disentuh. `confirm: true` **wajib bila ada agenda yang akan di-reset** (`reset > 0`), karena perubahan pengguna pada agenda itu tertimpa; selain itu `400 invalid_input: confirm: required`. Hasil: `{created, skipped: 0, reset, kept, warnings}`.
- **Playlist hanya `append`**: lagu template tidak punya status berjalan, jadi tidak ada yang "dipertahankan" dan reset akan menimpa format, cue, dan status yang sudah diisi pengguna; `mode: replace_template` pada playlist `400 invalid_input: mode: must be append`. Jenis acara tanpa template lagu (delapan non-pernikahan) `400 invalid_template: playlist: not available for this event type`. Lagu template: `audio_format = digital_playback`, `status = needs_review`, tanpa tautan, `moment_tag` dari bab 5.4.
- **Proyek Contoh:** wizard `append` saja, tanpa overlay (bab 6): 33 agenda dan 30 lagu, semua `pending`, tanpa vendor dan tautan lagu.

Versi lama menulis `replace_template` sebagai "hanya menghapus agenda bersumber template yang masih `pending`". Dijalankan: itu menghapus 33 agenda dan menurunkan lagu yang tertaut dari 2 menjadi 0, tanpa warning.

### 2.7 Playlist dan sound cue

Tiap lagu dapat ditautkan ke agenda (`rundown_id`, FK komposit, `SET NULL` saat agenda dihapus), diberi *moment tag* dari registry jenis acara (kustom boleh; daftar di [TEMPLATE_EVENT bab 6](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#6-moment-tag-audio-satu-set-per-jenis-sumber-tunggal-untuk-06); `EventTypeSpec.MomentTags` dan `moment_tags` pada `GET /event-types`, bab 6), format (`digital_playback | live_band | acoustic | sound_fx`), tautan Spotify/YouTube (`http`/`https`, bab 3.4), catatan cue (titik mulai, fade, ducking, level volume), dan status kesiapan (`ready | needs_review | missing_file`). Respons memuat `agenda` ringkas (`id`, `activity_name`, `session_name`, jam) agar kolom Agenda tidak membutuhkan permintaan kedua.

`GET …/playlists/export-wa` menyusun briefing soundman/band di server, dengan pola dan aturan dasar yang sama dengan bab 2.4: judul `🎵 *SOUND CUE {judul proyek}*`; lagu yang tertaut ke agenda dikelompokkan per sesi dan diurutkan menurut jam agenda (`📜 *AKAD NIKAH*` → `⏰ 07.46 Ijab Kabul`), lagu tanpa tautan dikelompokkan per *moment tag* di bawah `*Belum ditautkan ke agenda*`; tiap lagu `• {judul} - {penyanyi} ({format})`, dan `   ↳ Cue: …` bila catatan cue ada. **Cue sheet cetak** memakai stylesheet cetak yang sama dengan cue card rundown (tanpa ekspor PDF).

### 2.8 Ringkasan, KPI, dan sesi

`GET …/rundowns` mengembalikan `RundownOverview`; `GET …/rundowns/summary` mengembalikan `RundownLive` (KPI, ringkasan sesi, dan `{id, status, started_at, completed_at}` tiap agenda, tanpa teks agenda). **`kpi` dan `sessions` selalu seluruh proyek**; hanya `items` yang mengikuti `?session_id=`. Persentase tidak dikirim ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)): klien menghitung `percent(completed, total_agendas)`.

```sql
-- $1 = project_id. Ringkasan sesi non-sistem, termasuk sesi tanpa agenda (nol). Dijalankan, termasuk sesi lewat tengah malam:
-- last_end memilih akhir mutlak terlambat (00:30 pada contoh 23.30 → 00.30, bukan 23:00 seperti max(time)).
SELECT s.id, s.scope_key, s.name, s.icon, to_char(s.session_date,'YYYY-MM-DD') AS date,
       count(r.id) AS agenda_count,
       count(r.id) FILTER (WHERE r.status = 'completed')   AS completed,
       count(r.id) FILTER (WHERE r.status = 'in_progress') AS in_progress,
       COALESCE((max(EXTRACT(EPOCH FROM COALESCE(r.end_time, r.start_time))/60 + CASE WHEN r.ends_next_day THEN 1440 ELSE 0 END)
                - min(EXTRACT(EPOCH FROM r.start_time)/60))::int, 0) AS span_minutes,
       to_char(min(r.start_time), 'HH24:MI') AS first_start,
       (array_agg(to_char(COALESCE(r.end_time, r.start_time), 'HH24:MI')
          ORDER BY EXTRACT(EPOCH FROM COALESCE(r.end_time, r.start_time))/60 + CASE WHEN r.ends_next_day THEN 1440 ELSE 0 END DESC))[1] AS last_end
FROM planner_event_sessions s LEFT JOIN planner_rundowns r ON r.session_id = s.id
WHERE s.project_id = $1 AND NOT s.is_system
GROUP BY s.id, s.scope_key, s.session_date, s.sort_order ORDER BY s.sort_order;
```

`kpi.total_agendas`, `completed`, `in_progress` = Σ sesi; `pending = total − completed − in_progress`; `total_span_minutes` = Σ `span_minutes`; `sessions` = jumlah sesi non-sistem proyek; `distinct_pics` dari kueri bab 2.1. Pada Proyek Contoh: 3 sesi, 33 agenda, rentang 445, 17 PIC unik; pada ilustrasi hari-H di FE (Lamaran selesai, Akad 1–4 selesai dan 5 berjalan): 15 selesai, 1 berjalan, 17 menunggu ([fixtures/planner-rundown-live.json](fixtures/planner-rundown-live.json)).

KPI audio (`GET …/playlists`; selalu seluruh proyek, `items` mengikuti filter `moment_tag`, `format`, `status`):

```sql
-- $1 = project_id. Proyek Contoh: 30 lagu, 30 digital_playback, 6 momen, 0 siap, 30 perlu ditinjau.
SELECT count(*) AS total_tracks,
       count(*) FILTER (WHERE audio_format = 'digital_playback') AS digital_playback,
       count(*) FILTER (WHERE audio_format = 'live_band')        AS live_band,
       count(*) FILTER (WHERE audio_format = 'acoustic')         AS acoustic,
       count(*) FILTER (WHERE audio_format = 'sound_fx')         AS sound_fx,
       count(DISTINCT moment_tag) AS mapped_moments,
       count(*) FILTER (WHERE status = 'ready')        AS ready,
       count(*) FILTER (WHERE status = 'needs_review') AS needs_review,
       count(*) FILTER (WHERE status = 'missing_file') AS missing_file
FROM planner_playlists WHERE project_id = $1;
```

Daftar lagu **tidak berpaginasi** (batas 500, bab 2.9) dan tidak punya parameter pencarian: pencarian judul/artis/cue dilakukan di klien ([FE bab 7.2](FE.md#72-pemetaan-data--tampilan)); versi lama menampilkan kotak cari di FE tanpa parameter di backend.

### 2.9 Konkurensi dan batas

- **Satu pernyataan, satu baris:** status (bab 2.3), ubah, dan hapus agenda/lagu adalah satu `UPDATE`/`DELETE` dengan `WHERE id = $1 AND project_id = $2`.
- **Kunci proyek** (`SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE`, pola 02–05) hanya untuk yang menambah baris atau mengubah urutan: tambah agenda, tambah lagu, `apply-template`, dan kedua `reorder`, supaya batas jumlah, `sort_order`, dan `append` tidak berbalapan. Kunci ini dan indeks bab 1.1 menjaga idempotensi `append` bersama-sama.
- **Batas jumlah (usulan):** 500 agenda dan 500 lagu per proyek → `409 agenda_limit_reached` / `409 playlist_limit_reached` (kode baru, [kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#kode-error-planner)). Template terbesar 33 agenda (pernikahan); 500 mencakup acara beberapa hari ([bab 9 butir 2](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Template yang melewati batas gagal seluruhnya (semua-atau-tidak).
- **Frekuensi:** `export-wa` (agenda dan playlist) 10/menit per pengguna, pola `ratelimit.go` (salinan ketujuh). Polling `rundowns/summary` (5 detik per tab) **tidak dibatasi**, seperti polling meja resepsionis Menu 05; beberapa tablet panitia pada satu akun tidak boleh saling memblokir.
- **Data pribadi:** nomor telepon vendor ada di respons agenda; hanya pemilik proyek yang membacanya; tidak ada endpoint publik.

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, `BearerAuth` + `ProjectGuard` + `WriteGuard`. **CRUD sesi tidak di sini**, melainkan `GET|POST|PUT|DELETE /sessions` pada [01 BE §3.3](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi). Rute statis (`summary`, `reorder`, `apply-template`, `export-wa`) berdampingan dengan `:rundown_id`/`:playlist_id` dan terselesaikan benar di Gin (diuji pada audit Menu 01); id anak yang bukan UUID kanonik atau milik proyek lain `404 not_found` lebih dulu.

### 3.1 Agenda (10 endpoint)
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/rundowns?session_id=` | `RundownOverview` (bab 2.8); `items` disaring bila `session_id`. |
| `GET` | `/rundowns/summary` | `RundownLive` (polling 5 detik). |
| `GET` | `/rundowns/:rundown_id` | `Rundown`. |
| `POST` | `/rundowns` | Tambah agenda (`201 Rundown`). |
| `PUT` | `/rundowns/:rundown_id` | Ganti penuh (tanpa `status`). |
| `PATCH` | `/rundowns/:rundown_id/status` | Bab 2.3. |
| `POST` | `/rundowns/reorder` | Bab 2.5 (`204`). |
| `DELETE` | `/rundowns/:rundown_id` | `204`. |
| `POST` | `/rundowns/apply-template` | Bab 2.6 (`TemplateResult`). |
| `GET` | `/rundowns/export-wa?session_id=` | `WAExport` (bab 2.4). |

### 3.2 Playlist (7 endpoint)
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/playlists?moment_tag=&format=&status=` | `PlaylistOverview`. |
| `POST` | `/playlists` | Tambah lagu (`201 Playlist`). |
| `PUT` / `DELETE` | `/playlists/:playlist_id` | Ganti penuh (`Playlist`) / `204`. |
| `POST` | `/playlists/reorder` | Bab 2.5 (`204`). |
| `POST` | `/playlists/apply-template` | Bab 2.6 (`TemplateResult`). |
| `GET` | `/playlists/export-wa` | `WAExport` (bab 2.7). |

Tidak ada `export-pdf` (dihapus, bab 2.4) dan tidak ada `GET /playlists/:id` (FE memuat ulang daftar).

### 3.3 Bentuk respons

| Endpoint | `data` |
|---|---|
| `GET /rundowns` | `RundownOverview` `{sessions[], kpi, items[]}` |
| `GET /rundowns/summary` | `RundownLive` `{kpi, sessions[], items[], server_time}` |
| `GET /rundowns/:id`, `PUT`, `PATCH …/status`, `POST /rundowns` (`201`) | `Rundown` |
| `POST /rundowns/reorder`, `POST /playlists/reorder`, `DELETE …` | `204` tanpa badan |
| `POST …/apply-template` | `TemplateResult` `{created, skipped, reset, kept, warnings[]}` |
| `GET …/export-wa` | `WAExport` `{text}` (bukan berkas) |
| `GET /playlists` | `PlaylistOverview` `{kpi, items[]}` |
| `POST /playlists` (`201`), `PUT /playlists/:id` | `Playlist` |

Kunci opsional **dihilangkan** bila kosong (`omitempty`, tipe FE `?:`): `end_time`, `duration_minutes` (agenda terbuka), `vendor`, `started_at`, `completed_at`, `template_key`, `date`, `first_start`, `last_end`, `rundown_id`, `agenda`, `spotify_url`, `youtube_url`, `audio_cue_notes`, dan `pic_name`/`phone_number`/`formatted_wa_number` pada `vendor`. Tidak ada `null` pada respons. Semua daftar (`sessions`, `items`, `warnings`) selalu `[]`, tidak pernah `null` (diuji, bab 7.2). Tidak ada `*_percent` pada respons.

### 3.4 Validasi dan batas

Semua galat berbentuk `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)); teks dipangkas (`trim`), panjang dihitung dalam rune. Angka batas adalah usulan ([bab 9 butir 2](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

| Field | Aturan | Galat |
|---|---|---|
| `session_id` | sesi non-sistem milik proyek; sesi sistem `all` ditolak | `400 invalid_scope: session_id: must be a session of this project` · `400 invalid_input: session_id: must not be the system session` |
| `start_time`, `end_time` | bab 2.1 | `400 invalid_input` (tabel bab 2.1) |
| `ends_next_day` | bab 2.1 | `400 invalid_input` (tabel bab 2.1) |
| `activity_name` | 1–255, tanpa karakter kontrol | `required`, `must be at most 255 characters` |
| `pic_name` | ≤ 150 | `must be at most 150 characters` |
| `vendor_id` | vendor proyek ini atau kosong | `400 invalid_input: vendor_id: must be a vendor of this project` |
| `description`, `highlights` | ≤ 2.000 | `must be at most 2000 characters` |
| `status` (PATCH) | tiga nilai | `must be one of pending, in_progress, completed` |
| `ids` (reorder) | tepat semua id dalam lingkup, tanpa kembar | `ids: must list every agenda of the session exactly once` (lagu: `…every song of the project exactly once`) |
| `mode` | `append` (bawaan) atau `replace_template`; lagu hanya `append` | `must be one of append, replace_template` · `must be append` |
| `confirm` | `true` wajib pada `replace_template` bila `reset > 0` | `400 invalid_input: confirm: required` |
| `song_title`, `artist` | 1–255 | `required`, `must be at most 255 characters` |
| `moment_tag` | ≤ 100; kosong = `Lainnya`; teks bebas (registry hanya saran) | `must be at most 100 characters` |
| `audio_format` | kosong = `digital_playback` | `must be one of digital_playback, live_band, acoustic, sound_fx` |
| `status` (lagu) | kosong = `needs_review` | `must be one of ready, needs_review, missing_file` |
| `spotify_url`, `youtube_url` | ≤ 2.000; `http`/`https` yang sah | `must be an http or https URL` |
| `audio_cue_notes` | ≤ 2.000 | `must be at most 2000 characters` |
| `rundown_id` | agenda proyek ini atau kosong | `400 invalid_input: rundown_id: must be an agenda of this project` |
| Query `session_id` | UUID sesi proyek ini | `session_id: must be a session of this project` |
| Query `moment_tag` (≤ 100), `format`, `status` | enum seperti di atas | `must be one of …` |
| Jumlah | ≤ 500 agenda dan ≤ 500 lagu per proyek, di bawah kunci proyek, semua-atau-tidak | `409 agenda_limit_reached` · `409 playlist_limit_reached` |
| Frekuensi | `export-wa` 10/menit per pengguna | `429 rate_limited` |
| Proyek `archived` | semua penulisan (termasuk status) | `409 project_read_only` |

Kode galat baru (ditambahkan ke tabel kontrak bab 6): `agenda_limit_reached`, `playlist_limit_reached`. Kode lain sudah ada: `invalid_scope`, `invalid_template`, `session_in_use` (Menu 01; mengembalikan `usage.agendas`), `project_read_only`, `rate_limited`. **Notice**: `scope_missing` sudah terdaftar di kontrak; tidak ada notice baru (reset-di-tempat tidak memutus tautan, bab 2.6). Urutan pemeriksaan: id anak tidak valid → `404`; bentuk dan isi badan menurut urutan field di tabel; batas jumlah `409` di dalam transaksi berkunci, sesudah validasi bentuk. `PATCH` memvalidasi hanya fieldnya; `PUT` seluruh badan.

---

## 4. DTO (Go)

Blok di bawah adalah DTO yang **ada di kode** (`backend/internal/modules/plannerrundown/domain/entity.go`; komentar di sana berbahasa Inggris), dikompilasi dan diuji terhadap fixture FE. Dua perbedaan dari versi audit: (1) **tanpa tag `binding`**, seperti Menu 04 dan 05: kegagalan `binding` hanya dapat dijawab `400 invalid_json` yang tidak menyebut field, padahal tabel [bab 3.4](#34-validasi-dan-batas) menjanjikan `<field>: <alasan>`, jadi usecase yang memvalidasi; (2) nama permintaan `RundownRequest` dan `PlaylistRequest` (satu tipe untuk POST dan PUT; versi audit memakai `Create…Request` dan alias `Update…Request`). `Notice` adalah `planner/domain.Notice`; sesi dan vendor dipakai lewat id.

```go
package domain

// Sesi dan vendor milik Menu 01 dan 04 dipakai lewat id; pd = planner/domain (Notice).

type RundownStatus string // "pending" | "in_progress" | "completed"
type AudioFormat string   // "digital_playback" | "live_band" | "acoustic" | "sound_fx"
type AudioStatus string   // "ready" | "needs_review" | "missing_file"

// ---------- permintaan ----------

// RundownRequest dipakai POST dan PUT (PUT mengganti penuh; status hanya lewat PATCH).
type RundownRequest struct {
	SessionID    uuid.UUID  `json:"session_id"` // sesi non-sistem milik proyek
	StartTime    string     `json:"start_time"` // "HH:MM" atau "HH.MM"
	EndTime      *string    `json:"end_time"`   // nil = terbuka ("selesai")
	EndsNextDay  bool       `json:"ends_next_day"`
	ActivityName string     `json:"activity_name"`
	PICName      string     `json:"pic_name"`
	VendorID     *uuid.UUID `json:"vendor_id"` // harus vendor proyek ini (400 invalid_input: vendor_id)
	Description  string     `json:"description"`
	Highlights   string     `json:"highlights"` // satu butir per baris, berawalan "- "
}

type RundownStatusRequest struct {
	Status RundownStatus `json:"status"`
}

type ReorderRundownsRequest struct {
	SessionID uuid.UUID   `json:"session_id"`
	IDs       []uuid.UUID `json:"ids"` // tepat semua agenda sesi itu, sekali masing-masing
}

type ReorderPlaylistsRequest struct {
	IDs []uuid.UUID `json:"ids"` // tepat semua lagu proyek, sekali masing-masing
}

type ApplyTemplateRequest struct {
	Mode    string `json:"mode"`    // append (bila kosong) | replace_template (hanya agenda; lagu hanya append)
	Confirm bool   `json:"confirm"` // wajib true bila replace_template akan mengubah agenda (bab 2.6)
}

// PlaylistRequest dipakai POST dan PUT.
type PlaylistRequest struct {
	RundownID     *uuid.UUID  `json:"rundown_id"` // harus agenda proyek ini (400 invalid_input: rundown_id)
	SongTitle     string      `json:"song_title"`
	Artist        string      `json:"artist"`
	MomentTag     string      `json:"moment_tag"`   // kosong = "Lainnya"
	AudioFormat   AudioFormat `json:"audio_format"` // kosong = digital_playback
	SpotifyURL    *string     `json:"spotify_url"`  // http/https
	YouTubeURL    *string     `json:"youtube_url"`  // http/https
	AudioCueNotes *string     `json:"audio_cue_notes"`
	Status        AudioStatus `json:"status"` // kosong = needs_review
}

// ---------- respons agenda ----------

type RundownVendor struct {
	ID                uuid.UUID `json:"id"`
	VendorName        string    `json:"vendor_name"`
	CategoryLabel     string    `json:"category_label"`
	PICName           string    `json:"pic_name,omitempty"`
	PhoneNumber       string    `json:"phone_number,omitempty"`
	FormattedWANumber string    `json:"formatted_wa_number,omitempty"` // 62…; tautan wa.me dibentuk klien
}

type Rundown struct {
	ID              uuid.UUID      `json:"id"`
	SessionID       uuid.UUID      `json:"session_id"`
	SessionName     string         `json:"session_name"`
	SessionIcon     string         `json:"session_icon"`
	ScopeKey        string         `json:"scope_key"`
	StartTime       string         `json:"start_time"`         // "HH:MM"
	EndTime         *string        `json:"end_time,omitempty"` // dihilangkan bila terbuka
	EndsNextDay     bool           `json:"ends_next_day"`
	DurationMinutes *int           `json:"duration_minutes,omitempty"` // dihilangkan bila terbuka
	ActivityName    string         `json:"activity_name"`
	PICName         string         `json:"pic_name"`
	Vendor          *RundownVendor `json:"vendor,omitempty"`
	Description     string         `json:"description"`
	Highlights      string         `json:"highlights"`
	Status          RundownStatus  `json:"status"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	LinkedSongs     int            `json:"linked_songs"` // lagu yang menunjuk agenda ini
	SortOrder       int            `json:"sort_order"`
	TemplateKey     string         `json:"template_key,omitempty"`
}

type RundownSessionSummary struct {
	ID          uuid.UUID `json:"id"`
	ScopeKey    string    `json:"scope_key"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Date        *string   `json:"date,omitempty"` // session_date "YYYY-MM-DD"
	AgendaCount int       `json:"agenda_count"`
	Completed   int       `json:"completed"`
	InProgress  int       `json:"in_progress"`
	SpanMinutes int       `json:"span_minutes"`          // rentang sesi (bab 2.1); 0 bila tanpa agenda
	FirstStart  *string   `json:"first_start,omitempty"` // "HH:MM"; dihilangkan bila tanpa agenda
	LastEnd     *string   `json:"last_end,omitempty"`    // "HH:MM" (akhir terlambat; jam mulai bila terbuka)
}

type RundownKPI struct {
	Sessions         int `json:"sessions"` // sesi non-sistem proyek
	TotalAgendas     int `json:"total_agendas"`
	Completed        int `json:"completed"`
	InProgress       int `json:"in_progress"`
	Pending          int `json:"pending"`
	TotalSpanMinutes int `json:"total_span_minutes"` // Σ rentang per sesi (bab 2.1)
	DistinctPICs     int `json:"distinct_pics"`      // pic_name tak kosong ∪ vendor tertaut, tanpa duplikat
}

// RundownOverview: kpi dan sessions selalu seluruh proyek; items mengikuti ?session_id=.
// Persentase tidak dikirim (kontrak bab 6): klien menghitung percent(completed, total_agendas).
type RundownOverview struct {
	Sessions []RundownSessionSummary `json:"sessions"`
	KPI      RundownKPI              `json:"kpi"`
	Items    []Rundown               `json:"items"`
}

type RundownLiveItem struct {
	ID          uuid.UUID     `json:"id"`
	Status      RundownStatus `json:"status"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// RundownLive: jawaban GET /rundowns/summary (polling 5 detik).
type RundownLive struct {
	KPI        RundownKPI              `json:"kpi"`
	Sessions   []RundownSessionSummary `json:"sessions"`
	Items      []RundownLiveItem       `json:"items"`
	ServerTime time.Time               `json:"server_time"` // UTC
}

// ---------- respons playlist ----------

type PlaylistAgenda struct {
	ID           uuid.UUID `json:"id"`
	ActivityName string    `json:"activity_name"`
	SessionName  string    `json:"session_name"`
	StartTime    string    `json:"start_time"`
	EndTime      *string   `json:"end_time,omitempty"`
}

type Playlist struct {
	ID            uuid.UUID       `json:"id"`
	RundownID     *uuid.UUID      `json:"rundown_id,omitempty"`
	Agenda        *PlaylistAgenda `json:"agenda,omitempty"`
	SongTitle     string          `json:"song_title"`
	Artist        string          `json:"artist"`
	MomentTag     string          `json:"moment_tag"`
	AudioFormat   AudioFormat     `json:"audio_format"`
	SpotifyURL    string          `json:"spotify_url,omitempty"`
	YouTubeURL    string          `json:"youtube_url,omitempty"`
	AudioCueNotes string          `json:"audio_cue_notes,omitempty"`
	Status        AudioStatus     `json:"status"`
	SortOrder     int             `json:"sort_order"`
	TemplateKey   string          `json:"template_key,omitempty"`
}

type FormatCounts struct {
	DigitalPlayback int `json:"digital_playback"`
	LiveBand        int `json:"live_band"`
	Acoustic        int `json:"acoustic"`
	SoundFX         int `json:"sound_fx"`
}

type PlaylistKPI struct {
	TotalTracks   int          `json:"total_tracks"`
	ByFormat      FormatCounts `json:"by_format"`
	MappedMoments int          `json:"mapped_moments"` // moment tag berbeda yang punya lagu
	Ready         int          `json:"ready"`
	NeedsReview   int          `json:"needs_review"`
	MissingFile   int          `json:"missing_file"`
}

// PlaylistOverview: kpi selalu seluruh proyek; items mengikuti filter moment_tag/format/status.
type PlaylistOverview struct {
	KPI   PlaylistKPI `json:"kpi"`
	Items []Playlist  `json:"items"`
}

// ---------- hasil operasi ----------

type TemplateResult struct {
	Created  int         `json:"created"`
	Skipped  int         `json:"skipped"`
	Reset    int         `json:"reset"` // agenda template pending dikembalikan ke isi template (replace_template)
	Kept     int         `json:"kept"`  // agenda template in_progress/completed yang tidak disentuh
	Warnings []pd.Notice `json:"warnings"`
}

type WAExport struct {
	Text string `json:"text"`
}
```

Aturan JSON: kunci `snake_case` dari tag (struct bersarang selalu bertag; versi lama menghasilkan `TotalAgendas` dan `ID`); daftar diinisialisasi `[]` sebelum diserialisasi (`nil` slice menjadi `null`, bab 3.3); `FormatCounts` selalu membawa keempat format. `RundownRequest` dan `PlaylistRequest` memakai pointer untuk nilai yang boleh kosong; `trim`, jumlah rune, enum, URL, jam, dan kepemilikan id dijaga usecase ([bab 3.4](#34-validasi-dan-batas)), jadi badan yang bentuknya benar tetapi isinya salah dijawab `400 invalid_input` dengan nama field, dan hanya JSON yang tidak dapat dibaca (atau tipe yang salah, atau lebih dari 64 KiB) yang dijawab `400 invalid_json`. Waktu (`started_at`, `completed_at`, `server_time`) selalu UTC (pgx mengembalikan `timestamptz` dalam zona proses; repository mengubahnya).

---

## 5. Master Data

### 5.1 Sesi dan agenda per jenis acara
Sesi bawaan sembilan jenis ada di [TEMPLATE_EVENT bab 3](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#3-ringkasan-registry-sembilan-jenis) dan **selalu** dibuat bersama proyek (bukan oleh seeder rundown). Agenda template non-pernikahan: [TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (9, 7, 7, 9, 7, 9, 9, dan 3 agenda; jumlah dan nama sesinya dibandingkan dengan README modul: sama). Kunci template mereka `<kunci jenis>.agenda_<NN>` (bab 2.6). Moment tag audio per jenis: [TEMPLATE_EVENT bab 6](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#6-moment-tag-audio-satu-set-per-jenis-sumber-tunggal-untuk-06). Hanya pernikahan punya lagu bawaan.

### 5.2 Rundown Lamaran: 11 Agenda (PDF Hal. 14, tabel kiri)

Sesi `lamaran`. Teks persis PDF. Kolom *Highlights* PDF dipisah dengan `;` dan disimpan sebagai baris berawalan "- " pada `highlights`. Agenda 10 dan 11 terbuka sesuai PDF (agenda 11: "10.40 - selesai", `end_time NULL`). PIC yang kosong pada PDF tetap kosong. `status` template `pending`.

| No | `template_key` | Waktu | Agenda (PDF) | PIC (PDF) | Keterangan (PDF) | Highlights (PDF) |
|:-:|---|---|---|---|---|---|
| 1 | `wedding.lamaran_01` | 09.00 - 09.10 | Pembukaan | MC | Penyambutan keluarga CMP dan mempersilahkan duduk | Keluarga besar pihak pria hadir; Persiapan menyambut keluarga besar pihak pria; Pembukaan oleh MC; Mempersilahkan duduk |
| 2 | `wedding.lamaran_02` | 09.10 - 09.25 | Sambutan Penyerahan CMP | Perwakilan Keluarga CMP | Sambutan Penyerahan Keluarga CMP | Kata Sambutan; Menyampaikan maksud & tujuan kedatangan; Meyakinkan keluarga CMW |
| 3 | `wedding.lamaran_03` | 09.25 - 09.40 | Sambut Penerimaan CMW | Perwakilan Keluarga CMW | Sambutan Penerimaan Keluarga CMW | Kata Sambutan; Mengucapkan terima kasih atas kedatangan; Memberikan tanggapan; Menyerahkan jawaban ke CMW |
| 4 | `wedding.lamaran_04` | 09.40 - 09.50 | Pemanggilan dan perkenalan CMP | CMP | Perkenalan CMP | MC mempersilahkan CMP berdiri untuk mengutarakan secara langsung; Meminta izin kpd orangtua; Meminta izin kpd orang tua CMW; Mengutarakan kpd CMW |
| 5 | `wedding.lamaran_05` | 09.50 - 10.00 | Pemanggilan dan perkenalan CMW | CMW | Perkenalan CMW | MC mempersilahkan CMW masuk untuk menjawab secara langsung (tepok pundak); Mengucapkan terima kasih; Meminta izin kpd orangtua; Mengutarakan kpd CMW |
| 6 | `wedding.lamaran_06` | 10.00 - 10.10 | Perkenalan 2 Keluarga | Perwakilan Keluarga CMP & CMW | Perkenalan 2 keluarga yang akan diwakilkan oleh salah satu keluarga calon mempelai | – |
| 7 | `wedding.lamaran_07` | 10.10 - 10.20 | Menceritakan kisah cinta calon mempelai | CMP & CMW | Pertanyaan dari MC | – |
| 8 | `wedding.lamaran_08` | 10.20 - 10.25 | Pemanggilan kedua ibu mempelai & penyematan cincin | Kedua Ibu CMP & CMW | Ibu calon mempelai untuk penyerahan cincin dilanjutkan dengan penyematan cincin | – |
| 9 | `wedding.lamaran_09` | 10.25 - 10.35 | Games with calon mempelai | – | Sesi Games (Shoe Game & Games untuk Tamu) | – |
| 10 | `wedding.lamaran_10` | 10.35 - 10.40 | DOA | – | – | – |
| 11 | `wedding.lamaran_11` | 10.40 - selesai | Ramah Tamah - penutup | MC | Foto bersama & Makan siang | – |

### 5.3 Rundown Akad dan Resepsi: 22 Agenda (PDF Hal. 14, tabel kanan)

Judul PDF "Rundown Resepsi dan Pernikahan". Agenda 1–8 masuk sesi `akad` (07.00–09.00), agenda 9–22 sesi `resepsi` (14 agenda, 09.15–13.00): total **8 + 14 = 22** seperti PDF, tanpa baris tambahan. Agenda 9 ("Resepsi", hanya `09.15`) terbuka; agenda 3 PDF bertuliskan "Pembacaan ayat suci Qori" tanpa kolom PIC, dipisah menjadi agenda "Pembacaan ayat suci" dengan PIC "Qori". Nama agenda 17–22 di PDF berhuruf kecil ("sesi makan", "entertain", "sesi salaman", "penutupan acara", "lempar bunga", "sesi foto") dan ditulis dengan huruf awal kapital; "Iring -iring" ditulis "Iring-iring". Teks keterangan sama dengan PDF termasuk ejaannya ("Kutbah", "dilakukan dan dilakukan").

| No | `template_key` | Sesi | Waktu | Agenda (PDF) | PIC (PDF) | Keterangan (PDF) |
|:-:|---|:-:|---|---|---|---|
| 1 | `wedding.rundown_01` | `akad` | 07.00 - 07.20 | Persiapan | CMP & CMW | Penghulu datang dan melakukan pemeriksaan kelengkapan persyaratan administrasi, kelengkapan dan kesiapan calon pengantin, serta kehadiran wali dan saksi. |
| 2 | `wedding.rundown_02` | `akad` | 07.21 - 07.25 | Pembukaan acara | MC | Pembawa acara melakukan pembukaan acara akad nikah. |
| 3 | `wedding.rundown_03` | `akad` | 07.26 - 07.35 | Pembacaan ayat suci | Qori | Pembacaan ayat-ayat kitab suci. |
| 4 | `wedding.rundown_04` | `akad` | 07.36 - 07.45 | Khutbah | Ustadz/Kyai | Kutbah pernikahan dan pengumuman hasil kelengkapan persyaratan. |
| 5 | `wedding.rundown_05` | `akad` | 07.46 - 07.50 | Ijab Kabul | Penghulu | Ijab kabul yang dipimpin oleh penghulu dan dilakukan dan dilakukan oleh wali nikah. |
| 6 | `wedding.rundown_06` | `akad` | 07.51 - 08.00 | Doa bersama | Kyai/penghulu | Doa bersama dan penyerahan buku nikah. |
| 7 | `wedding.rundown_07` | `akad` | 08.01 - 08.10 | Sesi foto | Fotografer | Sesi foto bersama dan penyerahan maskawin. |
| 8 | `wedding.rundown_08` | `akad` | 08.11 - 09.00 | Selesai akad | MC | Persiapan melanjutkan ke acara resepsi. |
| 9 | `wedding.rundown_09` | `resepsi` | 09.15 | Resepsi | – | Pasangan pengantin sudah siap di lokasi acara. |
| 10 | `wedding.rundown_10` | `resepsi` | 09.15 - 09.30 | Persiapan resepsi | CMP & CMW | Menunggu tamu undangan berdatangan. |
| 11 | `wedding.rundown_11` | `resepsi` | 09.31 - 09.35 | Pembukaan acara | MC | Acara dibuka oleh pembawa acara. |
| 12 | `wedding.rundown_12` | `resepsi` | 09.36 - 09.40 | Iring-iring | CMP & CMW | Pengantin dan iring-iringannya berjalan ke pelaminan. |
| 13 | `wedding.rundown_13` | `resepsi` | 09.41 - 10.15 | Sambutan | Pihak keluarga | Pidato penyambutan dari keluarga. |
| 14 | `wedding.rundown_14` | `resepsi` | 09.50 - 10.20 | Acara adat | Yang bertugas | Acara adat jika ada. |
| 15 | `wedding.rundown_15` | `resepsi` | 10.20 - 10.21 | Doa bersama | Kyai/Ustadz | Doa bersama. |
| 16 | `wedding.rundown_16` | `resepsi` | 10.21 - 10.25 | Potong kue | CMP & CMW | Potong kue pernikahan yang dilakukan kedua mempelai. |
| 17 | `wedding.rundown_17` | `resepsi` | 10.26 - 12.30 | Sesi makan | MC | Tamu undangan dipersilahkan mengambil makanan dan minum. |
| 18 | `wedding.rundown_18` | `resepsi` | 10.26 - 12.30 | Entertain | Pengisi acara | Acara hiburan berlangsung. |
| 19 | `wedding.rundown_19` | `resepsi` | 10.30 - 12.30 | Sesi salaman | CMP & CMW | Tamu undangan bisa mengantre ke pelaminan untuk memberikan selamat kepada calon pengantin. |
| 20 | `wedding.rundown_20` | `resepsi` | 12.31 - 12.33 | Penutupan acara | MC | Penutupan acara oleh pembawa acara. |
| 21 | `wedding.rundown_21` | `resepsi` | 12.35 - 12.40 | Lempar bunga | CMP & CMW | Pengantin keluar dari lokasi undangan dan melempar buket bunga. |
| 22 | `wedding.rundown_22` | `resepsi` | 12.41 - 13.00 | Sesi foto | Fotografer | Melakukan sesi foto terakhir dan pulang dari lokasi acara. |

Agenda 17, 18, dan 19 berjalan paralel (10.26–12.30 dan 10.30–12.30) dan agenda 13–14 tumpang tindih; ini apa adanya dari PDF dan sebab KPI memakai rentang, bukan penjumlahan. Total agenda template pernikahan = **33**.

### 5.4 Playlist Pernikahan: 30 Lagu (PDF Hal. 15–16)

Judul dan penyanyi sesuai PDF (dua koreksi ejaan pada artis: "Admesh Kamaleng" → Andmesh Kamaleng, "Calumm Scott" → Calum Scott). "Janji Suci" muncul dua kali di PDF (no. 2 dan 12). **PDF tidak memuat moment tag, format, atau status**; kolom tag di bawah adalah saran Adicara. Template: `audio_format = digital_playback`, `status = needs_review`, tanpa tautan, tanpa `rundown_id` (pengguna menautkan sendiri).

| No | `template_key` | Judul (PDF) | Penyanyi (PDF) | Moment tag (saran Adicara) |
|:-:|---|---|---|---|
| 1 | `wedding.lagu_01` | Akad | Payung Teduh | Akad Khidmat |
| 2 | `wedding.lagu_02` | Janji Suci | Yovie & Nuno | Akad Khidmat |
| 3 | `wedding.lagu_03` | Teman Hidup | Tulus | Akad Khidmat |
| 4 | `wedding.lagu_04` | Kesempurnaan Cinta | Rizky Febian | Kirab / Entrance |
| 5 | `wedding.lagu_05` | Marry You | Maroon 5 | Kirab / Entrance |
| 6 | `wedding.lagu_06` | Kamulah Satu-Satunya | Dewa 19 | Kirab / Entrance |
| 7 | `wedding.lagu_07` | Cinta Kan Membawamu | Dewa 19 | Prosesi Adat |
| 8 | `wedding.lagu_08` | Aku Milikmu | Ari Lasso | Prosesi Adat |
| 9 | `wedding.lagu_09` | Anugrah Terindah Yang Pernah Kumiliki | Sheila On 7 | Prosesi Adat |
| 10 | `wedding.lagu_10` | Kala Cinta Menggoda | Chrisye | Santap & Ramah Tamah |
| 11 | `wedding.lagu_11` | Cantik | Kahitna | Santap & Ramah Tamah |
| 12 | `wedding.lagu_12` | Janji Suci | Yovie & Nuno | Santap & Ramah Tamah |
| 13 | `wedding.lagu_13` | Jatuh Hati | Raisa | Santap & Ramah Tamah |
| 14 | `wedding.lagu_14` | Cinta Luar Biasa | Andmesh Kamaleng | Santap & Ramah Tamah |
| 15 | `wedding.lagu_15` | Sugar | Maroon 5 | Santap & Ramah Tamah |
| 16 | `wedding.lagu_16` | Just The Way You Are | Bruno Mars | Santap & Ramah Tamah |
| 17 | `wedding.lagu_17` | Kau Adalah | Isyana Sarasvati | Santap & Ramah Tamah |
| 18 | `wedding.lagu_18` | Anything You Want | Reality Club | Santap & Ramah Tamah |
| 19 | `wedding.lagu_19` | Lebih Indah | Adera | Santap & Ramah Tamah |
| 20 | `wedding.lagu_20` | You Are The Reason | Calum Scott | Santap & Ramah Tamah |
| 21 | `wedding.lagu_21` | Thinking Out Loud | Ed Sheeran | Kirab / Entrance |
| 22 | `wedding.lagu_22` | Perfect | Ed Sheeran | Kirab / Entrance |
| 23 | `wedding.lagu_23` | Mine | Petra Sihombing | Santap & Ramah Tamah |
| 24 | `wedding.lagu_24` | Asmalibrasi | Soegi Bornean | Santap & Ramah Tamah |
| 25 | `wedding.lagu_25` | Komang | Raim Laode | Santap & Ramah Tamah |
| 26 | `wedding.lagu_26` | Untukku | Chrisye | Penutup |
| 27 | `wedding.lagu_27` | Sempurna | Andra and The Backbone | Penutup |
| 28 | `wedding.lagu_28` | Shallow | Lady Gaga | Games & Lempar Bunga |
| 29 | `wedding.lagu_29` | A Piece Of You | Nathaniel Constantin | Penutup |
| 30 | `wedding.lagu_30` | Last Night On Earth | Green Day | Penutup |

---

## 6. Sambungan ke Menu 01

Kait yang sudah ada di kode `planner` sebelum modul ini (dibaca, bukan diubah) dan yang **ditambahkan** saat modul dibangun (2026-10-31):

| Kait | Keadaan |
|---|---|
| Provider `rundown` (`SummaryKeys`) dan tipe `RundownSummary` `{sessions, agendas, completed, in_progress, songs, current_agenda, next_session}` | ada (`planner/domain/summary.go`); SQL-nya di [01 BE §5.2](../01_DASHBOARD/BE.md#52-query-per-provider) dijalankan apa adanya pada audit dan kini ada di `repository.Dashboard` (satu perubahan: "hari ini" dikirim dari jam usecase, bukan `now()`, supaya tes dapat menetapkannya). Pada Proyek Contoh nyata dengan Lamaran selesai dan agenda ke-5 Akad berjalan: 3 sesi, 33 agenda, 11 selesai, 1 berjalan, 30 lagu, `current_agenda` "Ijab Kabul · Akad Nikah"; `next_session` adalah **Lamaran** (bertanggal 15 Okt 2026 oleh overlay Menu 02), bukan Akad Nikah seperti pada data tiruan audit yang Lamarannya tak bertanggal (`TestSampleProjectDashboardWidget`). Modul ini mengimplementasikan `Key()` dan `Summarize` |
| Applier `rundown` dan `playlist` (`TemplateKeys`, urutan terakhir) dan `TemplateSet.Rundown`/`Playlist` | ada; pernikahan memiliki keduanya, delapan jenis lain hanya `Rundown` (`generalTemplates`, `planner/domain/eventtype.go`). Setelah applier terdaftar, `GET /event-types` mengisi `templates_available.rundown`/`playlist` dan `sampleTemplates` (`usecase/sample.go`) memintanya untuk Proyek Contoh |
| `usage.agendas` per sesi, `delete_agendas` pada `DELETE /sessions/:id`, `409 session_in_use` | ada (`usecase/session.go`); agenda ikut terhapus oleh `CASCADE` dan tautan lagu menjadi kosong (dijalankan) |
| `ApplyMode` | **ditambah**: `ModeReplaceTemplate` dan `ApplyRequest.Confirm` (`planner/domain/repository.go`); wizard tetap hanya `ModeAppend` |
| Registry `EventTypeSpec.MomentTags` dan `moment_tags` pada `GET /event-types` | **ditambah** (`planner/domain/momentcatalog.go`; data: TEMPLATE_EVENT bab 6; `TestMomentTagsMatchTemplateEvent` membaca dokumen itu) |
| Kode galat `agenda_limit_reached`, `playlist_limit_reached` | ada di kontrak bab 6 |
| `openapi.yaml` tag `Planner Rundown` dan dokumen [docs/backend/planner/2026-10-31-planner-menu-06.md](../../../backend/planner/2026-10-31-planner-menu-06.md) | **ada**: 17 operasi dan 20 skema (lint valid); `TestOpenAPIDocumentsEveryRundownRoute` dan `TestOpenAPISchemasMatchTheGoDTOs` menjaganya |
| Overlay Proyek Contoh | **tidak ada**: kontrak bab 5 dan 01 BE §4.5 hanya memuat template (33 agenda, 30 lagu) untuk 06, tanpa status, vendor, atau tautan lagu; ilustrasi hari-H di FE bukan data tersimpan |

Pemasangan di `internal/app/app.go` (pola Menu 04 dan 05): `plannerSvc.RegisterSummary(…, rundownSvc.Provider())`; `plannerSvc.RegisterTemplateAppliers(…, rundownSvc.RundownApplier(), rundownSvc.PlaylistApplier())`; `rundownhttp.RegisterRoutes(ginEngine, idn, plannerSvc, rundownSvc)`. Tidak ada `RegisterSampleOverlays` untuk modul ini.

Tugas yang menjadi dasar integrasi dengan Menu 02 sudah ada di kode untuk **semua sembilan jenis**: pernikahan memakai `wedding.adicara_technical_meeting` (H-7, Menu 04 dan 06), `wedding.adicara_gladi_resik` (H-1, Menu 06), dan `wedding.adicara_hari_h` (Hari-H, Menu 06) di `plannertimeline/domain/template_wedding.go`; delapan jenis lain memakai `general.technical_meeting_vendor_dan_panitia`, `general.gladi_resik`, dan `general.hari_h_acara` di `template_general.go` (sesi `all`, offset sama). Tidak ada sinkronisasi dua arah: tugas tidak berubah saat agenda berubah.

---

## 7. Pengujian

### 7.1 Yang sudah dijalankan pada audit spesifikasi

Dijalankan 2026-10-31 pada PostgreSQL 16.2 lokal, database sementara `adicara_audit06_test` (dibuat khusus dengan migrasi 001–012 dari berkas nyata lewat `go run ./cmd/migrate up`, dihapus setelahnya; database `adicara` tidak disentuh). Proyek Contoh **tiruan** (pengguna, proyek, empat sesi, dan agenda/lagu dari tabel bab 5) karena modul 02–05 dan API tidak dijalankan; ini SQL dan kode acuan, bukan HTTP.

| Yang diperiksa | Hasil |
|---|---|
| DDL bab 1 vs berkas migrasi 012 | identik per karakter; migrasi naik (14 tabel + 1 view) |
| PDF Hal. 14–16 dibaca ulang dengan `pdfminer` (layout dua tabel berdampingan) dan dibandingkan **secara terprogram** dengan tabel bab 5.2–5.4 | 11 + 22 agenda dan 30 lagu cocok: jam, PIC, keterangan, highlights, judul, penyanyi. Selisih yang nyata hanya yang tercatat: kapitalisasi awal agenda 17–22 ("sesi makan" → "Sesi makan"), "Iring -iring" → "Iring-iring", agenda 3 dipecah menjadi agenda + PIC "Qori", dua ejaan penyanyi. Kata "dilakukan dan dilakukan" dan "Kutbah" tetap seperti PDF. Judul tabel PDF hal. 15 adalah "LIST LAGU PERNIKAHAN" (menu samping: "PLAYLIST PERNIKAHAN") |
| 33 agenda dan 30 lagu dimasukkan dari tabel bab 5 | semua diterima; 2 agenda terbuka (`end_time NULL`, `duration_minutes NULL`); durasi baris Akad 20, 4, 9, 9, 4, 9, 9, 49 |
| Rentang | 100 / 120 / 225 = **445**; jumlah durasi naif 699; PIC unik 17 (18 bila tiga agenda MC ditautkan ke satu vendor) |
| Acara lewat tengah malam | A 22.00–23.00 + B 23.30–00.30: durasi B 60; rumus lama 1.500 atau 60; rumus menit-numerik **150**; agenda 00.15–00.45 tanpa penanda hari: rentang 1.455 (seharusnya 165) |
| `CHECK` dan DDL | `end < start` tanpa `ends_next_day` ditolak; `ends_next_day` dengan `end > start` diterima (1.500 menit), `end = start` (1.440), `end NULL`; detik pada `TIME` diterima; `completed` tanpa `completed_at` dan `pending` dengan `started_at` diterima |
| FK | sesi proyek lain ditolak; vendor dihapus → `vendor_id NULL`, `project_id` utuh; agenda dihapus → `rundown_id NULL`; sesi Resepsi dihapus → 33 → 19 agenda, tautan 0 |
| `append` seperti 02–04 pada migrasi 012 | `ON CONFLICT (project_id, template_key) …` gagal (tanpa indeks); duplikat `template_key` diterima; dengan dua indeks bab 1.1 → `INSERT 0 0` |
| `replace_template` versi lama (hapus agenda template `pending`) | 33 agenda terhapus, lagu tertaut 2 → 0 tanpa warning |
| Reset di tempat | agenda `pending` yang diubah kembali ke template dengan `id`, vendor, dan 1 lagu tertaut utuh; agenda `in_progress` dan `completed` yang diubah namanya tidak tersentuh |
| Urut ulang | SQL berjalan (sesi Akad dibalik; sesi lain tidak tersentuh dengan `AND session_id = $3`); id kembar dalam daftar lolos tanpa galat |
| Transisi status | satu pernyataan: `pending → in_progress → completed → in_progress → pending` menghasilkan invarian bab 2.3 |
| Provider dashboard 01 BE §5.2 (tiga kueri) | berjalan apa adanya: 3 sesi, 33 agenda, selesai/berjalan, 30 lagu, agenda berjalan, sesi terdekat |
| Kueri agenda lengkap (bab 2.2), ringkasan sesi (termasuk sesi tanpa agenda dan sesi lewat tengah malam), KPI audio | berjalan; `last_end` 00:30 pada sesi lewat tengah malam; sesi kosong semua nol |
| DTO Go bab 4 versi lama | **tidak dikompilasi** (`undefined: Playlist`); dengan stub, kunci JSON `TotalAgendas`, `ID`, `PICName`, `sessions:null`, `items:null` |
| DTO Go bab 4 versi baru | `go build`, `go vet`, `gofmt` bersih; empat fixture di [fixtures/](fixtures/) (`planner-rundowns`, `planner-rundown-live`, `planner-playlists`, `planner-rundown-wa-akad`) di-decode dengan `DisallowUnknownFields` dan diserialisasi ulang sama (nol selisih); `planner-rundown-errors.json` (17 galat) dicek bentuknya; dua mutasi (hapus `omitempty` pada `end_time`, ubah satu tag) menjatuhkan tes |
| Tipe TypeScript ([FE bab 4.2](FE.md#42-tambahan-pada-srclibplannertypests)) vs DTO Go | 23 pasangan tipe, 118 jalur JSON, nol selisih (kunci dan opsionalitas `omitempty` ↔ `?:`); `tsc` strict dengan `noUncheckedIndexedAccess` bersih |
| Kode acuan FE (`rundown.ts`, [FE bab 5](FE.md#5-library-libplannerrundownts)) | 12 tes `node:test` lulus terhadap fixture; **18 dari 18 mutasi** kode menjatuhkan tes |
| Jumlah agenda template non-pernikahan dan kunci sesinya | 9, 7, 7, 9, 7, 9, 9, 3 agenda per jenis (dihitung dari tabel TEMPLATE_EVENT bab 7–14) sama dengan README modul; 25 kunci sesi pada sembilan jenis di registry kode (`planner/domain/eventtype.go`) sama dengan TEMPLATE_EVENT bab 3 |

### 7.2 Pengujian pada implementasi

✔ = ada dan lulus terhadap PostgreSQL 16.2 (`TEST_DATABASE_URL`, `go test -p 1`); nama tes ada di kolom kanan. Tes di paket `delivery/http` melewati router, `BearerAuth`, usecase Menu 01–06, dan PostgreSQL, dipasang seperti `internal/app/app.go`.

| Uji | Status | Tes |
|---|---|---|
| Template pernikahan memuat tepat 11 + 8 + 14 = 33 agenda dan 30 lagu; dua agenda terbuka tersimpan | ✔ | `TestSampleProjectGoldenRundown`, `TestWeddingTemplateMatchesSpec`, `TestWeddingPlaylistMatchesSpec` (membaca tabel bab 5.2–5.4) |
| Rentang Proyek Contoh 100 / 120 / 225 = 445; sesi lewat tengah malam 150; PIC unik 17 | ✔ | `TestSampleProjectGoldenRundown`, `TestSessionSpanAcrossMidnight`, `TestAgendaVendorLink` (PIC unik: vendor sekali, huruf besar-kecil dan spasi) |
| Aturan jam akhir bab 2.1 (tabel): empat penolakan, `ends_next_day` dengan `end NULL`, 1.500 menit ditolak usecase | ✔ | `TestValidateTimes`, `TestParseClock` (domain), `TestAgendaTimeRules` |
| Agenda dan lagu tidak dapat menunjuk sesi, vendor, atau agenda proyek lain; sesi sistem `all` ditolak | ✔ | `TestAgendaTextValidation`, `TestAgendaVendorLink`, `TestPlaylistValidation`, `TestRundownQueryRejectsUnknownSessions` |
| Hapus sesi dengan agenda: `409 session_in_use` tanpa `delete_agendas`; dengan itu agenda terhapus dan tautan lagu kosong | ✔ | `TestSessionUsageAndDeletion`, `TestDeleteRundownKeepsSongsUnlinked` |
| `apply-template`: `append` dua kali `created 0`; wizard Proyek Contoh = 33 + 30; `scope_missing`; jenis tanpa lagu `invalid_template` | ✔ | `TestAppendIsIdempotentAndFillsGaps`, `TestAppendConcurrentlyCreatesOnce`, `TestWizardCreatesTheTemplates`, `TestSessionRemovedBySomeoneIsSkippedWithAWarning`, `TestPlaylistTemplateOnlyAppends` |
| `replace_template`: `confirm` wajib bila `reset > 0`; id, vendor, tautan lagu tetap; agenda berjalan/selesai tidak disentuh; agenda kustom tidak disentuh | ✔ | `TestReplaceTemplateResetsInPlace` (juga urutan template kembali), `TestReplaceTemplateEdgeCases` (rundown kosong tanpa `confirm`, agenda yang dipindah pengguna) |
| Status: invarian cap, status sama dua kali tidak mengubah cap, dua agenda `in_progress` sekaligus, proyek `archived` `409` | ✔ | `TestStatusInvariants`, `TestArchivedProjectIsReadOnlyButStillReadable` |
| Urut ulang: id kembar/kurang/lebih/sesi lain ditolak; dua urut ulang berbalapan konsisten | ✔ | `TestReorderRundowns`, `TestReorderRundownsConcurrently` (delapan serentak), `TestReorderPlaylists`, `TestSameIDs` |
| Batas 500 agenda dan 500 lagu (tepat 500 lolos, 501 ditolak, juga serentak); template melewati batas gagal seluruhnya | ✔ | `TestAgendaLimit`, `TestPlaylistLimit`, `TestTemplateThatPassesTheLimitCreatesNothing` |
| Bentuk respons: tidak ada `null`, daftar kosong `[]`, kunci opsional dihilangkan, tanpa `*_percent`; fixture FE sama dengan respons asli | ✔ | `TestResponsesHaveNoNullsNoPercentsAndEmptyListsAreArrays`, `TestSampleProjectGoldenRundown` (fixture agenda dan playlist), `TestHariHLiveAndWhatsAppMatchTheFixtures` (fixture live dan teks Akad), `TestErrorCatalogueMatchesTheServer` (17 galat) |
| Teks WhatsApp: agenda terbuka, tanpa PIC/keterangan, sesi tanpa tanggal, `venue_name` + `city`, semua sesi vs satu sesi | ✔ | `TestAgendaTextRules`, `TestAgendaTextOfTheAkadIsTheFixture`, `TestPlaylistText` (domain), `TestWhatsAppTextsOverHTTP` |
| Provider `rundown` dan `usage.agendas`; widget dashboard Proyek Contoh (3 sesi, 33 agenda, 30 lagu) | ✔ | `TestSampleProjectDashboardWidget`, `TestSessionUsageAndDeletion` |
| Sembilan jenis acara menjalani alur penuh: template agenda, `replace_template` pada rundown yang belum disentuh (`reset` = jumlah agenda template, lalu `append` `skipped` sebanyak itu), dua teks WhatsApp (judul, ikon, dan sesi jenis itu; tanpa kata khas pernikahan), lagu dengan setiap moment tag registry, widget dashboard, status, urut ulang, hapus sesi; moment tag dari registry | ✔ | `TestEveryEventTypeGetsItsRundown`, `TestEventTypesCarryMomentTagsAndTemplates`, `TestGeneralTemplateMatchesTemplateEvent`, `TestMomentTagsMatchTemplateEvent`, `TestTemplateIsConsistentWithTheRegistry` |
| `export-wa` 10/menit; `GET /rundowns/summary` tidak dibatasi | ✔ | `TestExportRateLimitAndUnlimitedPolling`, `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow` |
| Akses: tanpa token `401`, pengguna lain `404` di 17 rute, id bukan UUID kanonik `404`, badan rusak `400 invalid_json` | ✔ | `TestEveryRouteNeedsATokenAndHidesTheProjectFromOthers`, `TestMalformedBodiesAreInvalidJSON` |
| Kontrak: 17 rute terdaftar = 17 operasi, 20 skema = tag JSON struct Go, `moment_tags` pada `PlannerEventType` | ✔ | `TestOpenAPIDocumentsEveryRundownRoute`, `TestOpenAPISchemasMatchTheGoDTOs`, `TestOpenAPIEventTypeHasMomentTags`, `TestOpenAPIMentionsTheCodesOfThisMenu` |
| Migrasi: dua indeks unik `template_key` membuat `append` idempoten | ✔ | `TestRundownTemplateKeyIndexesMakeAppendIdempotent` (`internal/migrate`) |
| `-race`, `go vet`, `gofmt`, cakupan, mutasi kode produksi | ✔ | hasilnya di [bab 7.3](#73-hasil-implementasi) |

### 7.3 Hasil implementasi

Dijalankan 2026-10-31 pada PostgreSQL 16.2 lokal, database sementara `adicara_rundown06_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 -v ./...` | **520 tes tingkat atas lulus, 0 gagal, 0 di-skip** (448 sebelumnya + 72 baru: 44 HTTP, 16 domain, 9 usecase, 2 repository, 1 migrasi) |
| `-race` pada `plannerrundown/...` dan `planner/...` | lulus, tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`, gabungan semua paket tes) | 94,9% pernyataan; sisanya cabang galat yang hanya terjadi bila basis data gagal |
| `go vet ./...`, `gofmt -l` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Kontrak | 17 rute terdaftar = 17 operasi bertag `Planner Rundown`; 20 skema sama dengan tag JSON struct Go; `moment_tags` pada `PlannerEventType` |
| Mutasi kode produksi | **40 mutasi, 39 digagalkan tes, 1 setara** (applier wizard memakai `replace_template`; pada proyek baru hasilnya sama dengan `append`, jadi tak terbedakan lewat HTTP). Putaran pertama (33) meninggalkan tiga yang selamat dan satu yang tak terkompilasi; semuanya menambah atau memperbaiki tes: PIC unik (agenda vendor dengan teks PIC berbeda), urutan template yang dikembalikan `replace_template`, dan kunci proyek (balapan serentak tidak membuktikannya; diganti tes yang menahan kunci dari luar, lalu delapan mutan "tanpa kunci" per operasi dan satu "status menunggu kunci" semuanya tertangkap). Dicakup: aturan jam, cap status, rentang lewat tengah malam, PIC unik, urut ulang, template (`confirm`, agenda berjalan, dipindah, urutan), batas jumlah, kunci proyek, KPI, teks WhatsApp, pembatas laju, UTC, pemetaan galat |
| Universalitas | sembilan jenis acara dari registry menjalani template, `replace_template`, kedua teks WhatsApp, lagu per moment tag, widget dashboard, status, urut ulang, dan hapus sesi lewat router dan PostgreSQL asli (`TestEveryEventTypeGetsItsRundown`, satu pengguna per jenis karena kedua ekspor dibatasi per pengguna). Dua mutasi pada tes ini (ikon jenis diganti ikon pernikahan; judul moment tag lagu tanpa agenda dihapus dari teks) digagalkan. Untuk pernikahan, `reset` 33 pada rundown yang belum disentuh (isinya sama dengan Proyek Contoh, yang tidak punya overlay untuk 06) juga dijaga tes ini |
| Dokumen ↔ kode | tabel bab 5.2–5.4 = data template; TEMPLATE_EVENT bab 6–14 = registry dan template delapan jenis; `-v` memastikan tidak ada yang di-skip |
| **Respons HTTP asli = fixture FE** | daftar agenda, playlist, status hari-H, teks WhatsApp Akad, dan katalog 17 galat sama dengan kelima fixture di [fixtures/](fixtures/) (id dan waktu dinormalkan); dua fixture dikoreksi (butir 17 dan 18 bab 9). Pembandingnya dapat gagal: ia menemukan kedua selisih itu dan cap waktu yang keluar dalam zona lokal |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, repository murni, pembatas laju, kontrak berkas, dan tes yang membaca dokumen lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh lewat seluruh tumpukan (33 agenda, rentang 100/120/225 = 445, 17 PIC, durasi Akad 20/4/9/9/4/9/9/49, dua agenda terbuka, 30 lagu dengan 6 momen); ilustrasi hari-H (Lamaran selesai, Akad 1–4 selesai dan 5 berjalan: 15 selesai, 1 berjalan, 17 menunggu) sama dengan fixture; sesi lewat tengah malam berentang 150; delapan urut ulang serentak menghasilkan `sort_order` 1…8 yang berbeda; empat `apply-template` serentak menghasilkan tepat 33 agenda; 500 agenda dan 500 lagu tepat (serentak hanya satu lolos); template pada 480 gagal seluruhnya; `replace_template` mempertahankan id, vendor, dan lagu tertaut dan mengembalikan urutan template; tujuh penulis menunggu kunci proyek dan tujuh operasi satu pernyataan atau baca tidak; `export-wa` 10/menit untuk kedua ekspor bersama dan 60 polling `summary` berturut-turut tanpa batas.

Belum diuji: frontend dan BFF; kinerja daftar 500 agenda/lagu dan polling dari beberapa tablet; teks WhatsApp di WhatsApp (pemformatan `*`/`_` dan batas panjang pesan); PostgreSQL 18 (versi Compose); beberapa replika API (pembatas laju per proses).

---

## 8. Urutan pengerjaan dan definisi selesai

Posisi di urutan pembangunan ([kontrak bab 9](../00_KONTRAK_BERSAMA/README.md#9-urutan-pembangunan-dan-uji-penerimaan)): terakhir, setelah `planner` (01), `plannertimeline` (02), `plannerfinance` (03), `plannervendor` (04), dan `plannerguest` (05); tabel dan registry yang dibutuhkan sudah ada. Dikerjakan 2026-10-31 dengan urutan di bawah; semuanya ✔.

1. ✔ **Migrasi:** dua indeks bab 1.1 ditambahkan ke `012_000_planner_rundown.sql` di tempat (tahap pengembangan: database lama harus `migrate fresh`); blok DDL bab 1 diperbarui dan dibandingkan per karakter dengan berkas; tes migrasi ditambah.
2. ✔ **Domain:** DTO bab 4, `ParseClock`/`ValidateTimes`, penyusun teks WhatsApp, data template (`rundown_wedding.go`, `rundown_general.go`, `playlist_wedding.go`; bab 5 dan TEMPLATE_EVENT 7–14, dibangkitkan dari tabelnya dan dijaga tes yang membaca tabel itu). Rentang dan PIC unik tetap SQL (bab 2.1), bukan Go.
3. ✔ **Menu 01:** `ModeReplaceTemplate` + `Confirm`, `EventTypeSpec.MomentTags` + `moment_tags`; kode galat sudah ada di kontrak.
4. ✔ **Repository** dan **usecase:** daftar + ringkasan, CRUD, status, urut ulang, template (`append`, reset di tempat), playlist, `LockProject`; `Summarize` untuk provider `rundown`.
5. ✔ **Delivery:** 17 rute, DTO ke JSON (daftar `[]`), pemetaan galat, pembatas laju `export-wa` 10/menit.
6. ✔ **Sambungan** di `internal/app/app.go` (bab 6): provider, dua applier, rute.
7. ✔ **Kontrak dan dokumen:** `backend/docs/openapi.yaml` (tag `Planner Rundown`); `docs/backend/planner/2026-10-31-planner-menu-06.md`; `docs/api.md`, `docs/database.md`, `docs/architecture.md`, `docs/backend/CHANGELOG.md`; entri CHANGELOG spesifikasi.
8. ✔ **Uji emas** Proyek Contoh dan seluruh tabel bab 7.2; fixture FE dicocokkan dengan respons HTTP asli (dua fixture dikoreksi, bab 9 butir 17 dan 18).

**Definisi selesai:** semua butir bab 7.2 lulus terhadap PostgreSQL ✔; Proyek Contoh memuat 33 agenda (445 menit) dan 30 lagu ✔; widget Rundown di dashboard terisi dan `usage.agendas` tetap benar ✔; `GET /event-types` memuat `moment_tags` dan `templates_available.rundown`/`playlist` ✔; OpenAPI lolos lint ✔; dokumen modul menyebut setiap penyimpangan dari dokumen ini ✔ ([bab 9 butir 17–26](#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

---

## 9. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–05 (kode dan dokumen), kontrak, PDF, dan SQL yang dijalankan. Dokumen mengikuti pilihan yang tertulis; "alternatif" adalah yang bisa dipilih bila produk tidak setuju.

1. ✔ **Indeks unik `template_key`: diterapkan** pada migrasi 012 saat modul dibangun (bab 1.1). Konsekuensinya database yang sudah menjalankan 012 lama harus `migrate fresh` atau menambahkan kedua indeks dengan tangan; migrasi belum dirilis ke `main`.
2. **Angka batas adalah usulan tanpa pengukuran:** 500 agenda, 500 lagu, teks 255/150/100/2.000, `export-wa` 10/menit. Kinerja daftar tanpa paginasi pada 500 baris belum diukur.
3. **Agenda yang mulai sesudah tengah malam tidak dapat dimodelkan** (rentang salah: 1.455 menit, seharusnya 165). Alternatif: kolom `day_offset SMALLINT NOT NULL DEFAULT 0` (hari mulai dan akhir relatif Hari-H sesi) menggantikan `ends_next_day`, dengan rumus rentang yang memakainya; migrasi 012 boleh diubah di tempat pada tahap pengembangan. Sementara itu hanya agenda yang *berakhir* sesudah tengah malam yang didukung.
4. **`replace_template` adalah reset di tempat**, berbeda dari 02–04 yang menghapus lalu menyisipkan (agenda berpindah `id`). Alasannya tautan lagu, vendor, dan `id` tetap. Alternatif: hapus lalu sisip dengan memulihkan tautan lagu/vendor berdasarkan `template_key`. `sort_order` agenda template direset ke urutan template.
5. **Teks WhatsApp disusun server**, berbeda dari Menu 04 dan 05 (klien). Alternatif: bentuk di klien dari `RundownOverview` (menghapus dua endpoint dan membuatnya teruji `node:test`); konsekuensinya aturan bab 2.4 pindah ke FE.
6. **Karakter format WhatsApp (`*`, `_`, `~`) dalam teks pengguna tidak di-escape.** Alternatif: sisipkan karakter lebar-nol sesudah penanda.
7. **Lagu hanya `append`** (tanpa `replace_template`, bab 2.6). Alternatif: reset hanya `song_title`/`artist`/`moment_tag` lagu template, sehingga format dan cue pengguna aman.
8. **Pelaksana unik** menghitung vendor sekali dan teks PIC apa adanya setelah `lower(btrim())`; "Kyai/penghulu" dan "Penghulu" dihitung dua, "Ustadz/Kyai" dan "Kyai/Ustadz" dua. Alternatif: hanya vendor dan nama PIC tunggal.
9. **Agenda 0 menit diizinkan** (`end_time = start_time`, tanpa `ends_next_day`) sebagai penanda; PDF tidak punya satu pun (terpendek 1 menit, agenda 15).
10. **Satu agenda satu sesi, dan bukan sesi sistem `all`**: agenda yang membentang dua sesi (mis. makan siang) dibuat dua baris. Alternatif: izinkan `all` sebagai "lintas sesi" dengan tampilan sendiri (kini sesi `all` tidak tampil di tab rundown, kontrak bab 4).
11. **Nomor urut tidak disimpan:** nomor 1…n per sesi dihitung klien dari urutan agenda. PDF Hal. 14 kanan bernomor 1–22 dalam satu tabel; Adicara memecahnya menjadi Akad 1–8 dan Resepsi 1–14 (agenda 9 PDF menjadi agenda 1 Resepsi), jadi nomor tidak sama dengan PDF, sedangkan isi barisnya sama.
12. **Tautan lagu hanya satu agenda** (`rundown_id` tunggal): lagu yang dipakai di dua momen diduplikasi sebagai dua baris. Alternatif: tabel penghubung.
13. **Tanpa `origin`** pada agenda/lagu (02 punya `pdf`/`adicara`/`custom`); `template_key` membedakan template dari kustom, tetapi tidak membedakan PDF dari usulan Adicara.
14. **Polling `summary` tidak dibatasi** dan belum diukur dengan beberapa tablet pada satu akun (sama dengan Menu 05 butir 23).
15. **Moment tag adalah teks bebas** yang tersimpan apa adanya; "penutup" dan "Penutup" dua tag (sama dengan kategori kustom Menu 05 butir 13).
16. **Hari-H "live" tidak dikaitkan dengan tanggal:** status dapat diubah kapan saja, termasuk jauh sebelum atau sesudah acara; tidak ada pengaman "hanya pada hari acara".

**Ditemukan saat modul dibangun (2026-10-31; butir 17–26).** Diketahui dari menjalankan kode terhadap PostgreSQL; bukan keputusan baru pemilik produk kecuali bila disebut "alternatif".

17. **`sort_order` adalah urutan dalam sesi (1…n), bukan nomor PDF.** Fixture `planner-rundowns.json` memberi agenda Resepsi `sort_order` 9–22 (nomor tabel PDF yang dipakai audit); kode memberi 1–14 (`max + 1` pada sesinya, dan `replace_template` mengembalikan urutan template per sesi). Fixture dikoreksi; FE tidak terpengaruh karena nomor tampilan dihitung klien (butir 11).
18. **Lamaran Proyek Contoh bertanggal** (15 Okt 2026, dari overlay Menu 02), sedangkan data tiruan audit tidak. Fixture `planner-rundowns.json` dan `planner-rundown-live.json` mendapat `date` pada sesi Lamaran; `next_session` pada widget dashboard Proyek Contoh adalah Lamaran, bukan Akad Nikah (bab 6).
19. **DTO tanpa tag `binding`** (bab 4): validasi bentuk dan isi di usecase dengan pesan `<field>: <alasan>`. Versi audit memakai `binding:"required,max=255"` yang akan menjawab `400 invalid_json` tanpa nama field, bertentangan dengan tabel bab 3.4.
20. **`reset` menghitung semua agenda template `pending` pada sesi template-nya**, bukan hanya yang isinya berbeda dari template. Jadi `replace_template` pada Proyek Contoh yang belum diubah pun `reset: 33` dan memerlukan `confirm` (sama dengan contoh di FE bab 5 dan `replaceImpact`). Alternatif: hitung hanya yang berbeda, sehingga `reset 0` tanpa `confirm` untuk rundown yang belum disentuh; biayanya pembanding sepuluh kolom per agenda dan dialog FE tidak lagi sama dengan `replaceImpact`.
21. **Agenda template yang dipindahkan pengguna ke sesi lain** tidak di-reset dan tidak dibuat ulang; dihitung `skipped`. Pada keadaan biasa `skipped` pada `replace_template` adalah 0 (bab 2.6 tidak menyebut kasus ini).
22. **`scope_missing` satu notice per permintaan**: `count` = jumlah agenda template yang sesinya sudah dihapus; agenda itu tidak dihitung `skipped` (arti `skipped` tetap "sudah ada"; pesan FE "{skipped} sudah ada" tetap benar).
23. **Baris lokasi teks WhatsApp:** `location_name` sesi **menggantikan** `venue_name` proyek (Lamaran di rumah keluarga tidak boleh disebut di ballroom), kota tetap ikut; tanpa `session_id`, tempat sesi yang punya `location_name` ditulis di bawah judul sesinya. Bab 2.4 menulisnya sebagai "dari `location_name` sesi, lalu `venue_name` dan `city`" (dapat dibaca sebagai gabungan); teks Akad Proyek Contoh sama dengan fixture karena sesinya tak punya `location_name`. Keterangan agenda yang berbaris-baris dilipat menjadi satu baris (indentasi), catatan cue berbaris-baris dijorokkan di bawah `↳ Cue:`.
24. **`PUT` yang memindahkan agenda ke sesi lain** menaruhnya paling akhir di sesi itu dan mengambil kunci proyek (mengubah urutan); bab 2.9 hanya menyebut tambah, template, dan urut ulang.
25. **Waktu selalu UTC** (`started_at`, `completed_at`, `server_time`): pgx mengembalikan `timestamptz` dalam zona proses (di Mac pengembang `+07:00`), jadi repository mengubahnya ke UTC; fixture dan kontrak menyebut `Z`.
26. **`apply-template` tanpa badan berarti `append`**; `mode: replace_template` pada playlist `400 mode: must be append`, nilai lain `must be one of append, replace_template`; `?session_id=` yang tidak sah pada daftar dan ekspor `400 invalid_scope` (sama dengan di badan), sesi sistem `400 invalid_input`.


Keputusan yang diambil pada spesifikasi ini (2026-10-31; ringkas di [CHANGELOG](../CHANGELOG.md)): `export-pdf` dihapus; `replace_template` reset-di-tempat dengan `confirm`; lagu `append` saja; kunci template non-pernikahan `<jenis>.agenda_<NN>`; rentang dalam menit numerik; `PUT` tanpa `status`; reorder berlingkup sesi; KPI dan sesi seluruh proyek; persentase dihitung klien; tidak ada overlay Proyek Contoh; dua kode galat baru.
