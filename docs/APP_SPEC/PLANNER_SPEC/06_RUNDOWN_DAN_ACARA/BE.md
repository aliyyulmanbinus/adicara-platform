# Menu 06: Rundown & Acara — Backend Specification

Spesifikasi backend modul Go `plannerrundown` (`backend/internal/modules/plannerrundown/`) untuk **susunan rundown menit demi menit** (PDF Hal. 14) dan **audio/playlist** (PDF Hal. 15–16). Tabel sesi (`planner_event_sessions`) kini milik modul `planner` sebagai master bersama ([01 BE](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi), [kontrak bab 4](../00_KONTRAK_BERSAMA/README.md#4-master-sesi-bersama-planner_event_sessions)); modul ini menambah agenda di atasnya. Aturan lintas modul ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

---

## 1. Skema Database

Migrasi `012_000_planner_rundown.sql` (sudah dijalankan di PostgreSQL 16; sesi ada di migrasi 005, vendor di 006):

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

-- +goose Down
DROP TABLE IF EXISTS planner_playlists;
DROP TABLE IF EXISTS planner_rundowns;
```

Keputusan desain:
- Jam bertipe **`TIME`**, bukan `VARCHAR(10)`; `end_time` boleh `NULL` karena PDF memuat agenda terbuka ("10.40 - selesai") dan agenda satu-jam ("09.15"). Dengan begitu dua baris PDF tersebut dapat disimpan apa adanya. `ends_next_day` menangani acara lewat tengah malam; `duration_minutes` adalah kolom generated (`NULL` untuk agenda terbuka).
- `session_id` dan `vendor_id` memakai FK komposit `(…, project_id)` sehingga agenda tidak bisa menunjuk sesi/vendor proyek lain.
- `started_at`/`completed_at` mencatat kapan tiap agenda mulai/selesai pada hari-H (tracker live).
- `template_key` untuk mode *append*; playlist default `needs_review` (PDF tidak memuat format atau kesiapan audio), bukan `ready`.
- Tabel sesi, ikon, dan seeder sesi per jenis acara ada di modul `planner` (01).

---

## 2. Logika Bisnis

### 2.1 Waktu dan durasi
Jam di JSON `"HH:MM"`. Input pengguna `07.46` (format PDF) dinormalisasi ke `07:46` di FE dan juga diterima server. `duration_minutes` dihitung basis data. **Total durasi** pada KPI adalah *rentang*, bukan jumlah durasi (agenda paralel seperti sesi makan, hiburan, dan salaman tidak boleh dijumlahkan): per sesi `span = max(COALESCE(end_time, start_time)) − min(start_time)` (+1440 bila `ends_next_day`), total = Σ span sesi. Data PDF: Lamaran 09:00–10:40 = 100 menit, Akad 07:00–09:00 = 120, Resepsi 09:15–13:00 = 225 → **445 menit (7 jam 25 menit)**.

### 2.2 PIC dan kontak vendor
`GET /rundowns` menggabungkan `planner_vendors` agar panitia dapat langsung membuka WhatsApp PIC dari baris agenda:

```sql
SELECT r.id, r.session_id, r.start_time, r.end_time, r.duration_minutes, r.activity_name, r.pic_name,
       r.description, r.highlights, r.status, r.sort_order,
       s.name AS session_name, s.icon AS session_icon, s.scope_key,
       v.id AS vendor_id, v.vendor_name, v.category_label, v.pic_name AS vendor_pic,
       v.phone_number AS vendor_phone, v.formatted_wa_number AS vendor_wa
FROM planner_rundowns r
JOIN planner_event_sessions s ON s.id = r.session_id
LEFT JOIN planner_vendors v ON v.id = r.vendor_id AND v.project_id = r.project_id
WHERE r.project_id = $1 AND ($2::uuid IS NULL OR r.session_id = $2)
ORDER BY s.sort_order, r.sort_order, r.start_time;
```

### 2.3 Live Stage Manager
`PATCH .../rundowns/:id/status` `{"status": "in_progress" | "completed" | "pending"}`: `in_progress` mengisi `started_at` (jika kosong), `completed` mengisi `completed_at` (dan `started_at` bila kosong), `pending` mengosongkan keduanya. Beberapa agenda boleh `in_progress` bersamaan (sesi makan, hiburan, dan salaman PDF memang paralel). Hanya pemilik proyek yang dapat mengubah status; pembaruan terbaca oleh perangkat lain lewat polling 5 detik (`GET .../rundowns/summary`), bukan SSE.

### 2.4 Pesan WhatsApp rundown
`GET .../rundowns/export-wa?session_id=` menyusun teks terstruktur yang **netral jenis acara**: judul dari `project.title`, ikon jenis dari registry (satu ikon per jenis; tidak ada pemetaan ikon kedua di modul ini), tanggal dari `session_date` sesi (bila kosong dari `event_date`), venue dari `metadata.venue_name` atau `location_name` sesi, lalu baris `⏰ HH:MM - HH:MM (N mnt) : Agenda` dengan `👤 PIC` dan `📝 highlights` bila ada. Agenda terbuka ditulis `HH:MM - selesai`.

### 2.5 Urut ulang agenda
Daftar id lengkap dalam satu sesi atau proyek; usecase memverifikasi semuanya milik proyek, lalu:

```sql
UPDATE planner_rundowns r SET sort_order = u.ord, updated_at = now()
FROM (SELECT unnest($1::uuid[]) AS id, generate_series(1, cardinality($1::uuid[])) AS ord) u
WHERE r.id = u.id AND r.project_id = $2;
```

### 2.6 Template
`POST .../rundowns/apply-template` `{"mode": "append" | "replace_template"}` (dengan mode eksplisit agar tidak menggandakan atau menimpa status live): memasang agenda template jenis acara ke sesi yang ada, dicocokkan lewat `scope_key`. Sesi yang tidak ada (dihapus pengguna) dilewati dengan warning `scope_missing`. `append` melewati `template_key` yang ada; `replace_template` hanya menghapus agenda bersumber template yang masih `pending` (agenda `in_progress`/`completed` dipertahankan dan dilaporkan). Playlist mengikuti pola yang sama (`POST .../playlists/apply-template`).

### 2.7 Playlist dan sound cue
Tiap lagu dapat ditautkan ke agenda (`rundown_id`, FK komposit, `SET NULL` saat agenda dihapus), diberi *moment tag* dari registry jenis acara (kustom boleh), format (`digital_playback | live_band | acoustic | sound_fx`), tautan Spotify/YouTube (http/https), catatan cue (fade, ducking, titik mulai), dan status kesiapan (`ready | needs_review | missing_file`). Ekspor WA untuk soundman/band dan PDF *cue sheet*.

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, `BearerAuth` + `ProjectGuard` + `WriteGuard`. **CRUD sesi tidak di sini**, melainkan `GET|POST|PUT|DELETE /sessions` pada [01 BE §3.3](../01_DASHBOARD/BE.md#33-sesi-acara-master-sesi).

### 3.1 Agenda
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/rundowns?session_id=` | Agenda + `overview` (KPI §4) + ringkasan per sesi. |
| `GET` | `/rundowns/summary` | Hanya KPI + status live (polling). |
| `GET` | `/rundowns/:rundown_id` | Detail. |
| `POST` | `/rundowns` | Tambah agenda. |
| `PUT` | `/rundowns/:rundown_id` | Ubah. |
| `PATCH` | `/rundowns/:rundown_id/status` | §2.3. |
| `POST` | `/rundowns/reorder` | §2.5. |
| `DELETE` | `/rundowns/:rundown_id` | `204`. |
| `POST` | `/rundowns/apply-template` | §2.6. |
| `GET` | `/rundowns/export-wa?session_id=` | Teks WA. |
| `GET` | `/rundowns/export-pdf?session_id=` | Cue card PDF (dependensi PDF: [kontrak 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)). |

### 3.2 Playlist
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/playlists?moment_tag=&format=&status=` | Daftar + KPI audio. |
| `POST` / `PUT` / `DELETE` | `/playlists[/:playlist_id]` | CRUD. |
| `POST` | `/playlists/reorder` | Urut ulang. |
| `POST` | `/playlists/apply-template` | §2.6. |
| `GET` | `/playlists/export-wa` | Briefing soundman/band. |
| `GET` | `/playlists/export-pdf` | Sound cue sheet. |

---

## 4. DTO (Go)

```go
package domain

type RundownStatus string // "pending" | "in_progress" | "completed"

type CreateRundownRequest struct {
	SessionID    uuid.UUID  `json:"session_id" binding:"required"`    // sesi non-sistem milik proyek
	StartTime    string     `json:"start_time" binding:"required"`    // "HH:MM" atau "HH.MM"
	EndTime      *string    `json:"end_time"`                         // nil = terbuka ("selesai")
	EndsNextDay  bool       `json:"ends_next_day"`
	ActivityName string     `json:"activity_name" binding:"required,max=255"`
	PICName      string     `json:"pic_name"`
	VendorID     *uuid.UUID `json:"vendor_id"`
	Description  string     `json:"description"`
	Highlights   string     `json:"highlights"`
}
// UpdateRundownRequest: field yang sama + Status (opsional).

type Rundown struct {
	ID              uuid.UUID     `json:"id"`
	SessionID       uuid.UUID     `json:"session_id"`
	SessionName     string        `json:"session_name"`
	SessionIcon     string        `json:"session_icon"`
	ScopeKey        string        `json:"scope_key"`
	StartTime       string        `json:"start_time"`
	EndTime         *string       `json:"end_time,omitempty"`
	EndsNextDay     bool          `json:"ends_next_day"`
	DurationMinutes *int          `json:"duration_minutes,omitempty"` // nil untuk agenda terbuka
	ActivityName    string        `json:"activity_name"`
	PICName         string        `json:"pic_name"`
	Vendor          *struct{ ID uuid.UUID; Name, CategoryLabel string; PICName, PhoneNumber, WhatsAppNumber *string } `json:"vendor,omitempty"`
	Description     string        `json:"description"`
	Highlights      string        `json:"highlights"`
	Status          RundownStatus `json:"status"`
	StartedAt       *time.Time    `json:"started_at,omitempty"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	LinkedSongs     int           `json:"linked_songs"` // lagu yang menunjuk agenda ini
	SortOrder       int           `json:"sort_order"`
}

type RundownOverview struct {
	Sessions []struct{ ID uuid.UUID; ScopeKey, Name, Icon string; AgendaCount, SpanMinutes int; FirstStart, LastEnd *string; Date *string } `json:"sessions"`
	KPI struct {
		TotalAgendas, Completed, InProgress, Pending int
		TotalSpanMinutes                             int // Σ rentang per sesi (§2.1)
		ProgressPercent                              float64
		DistinctPICs                                 int // pic_name tak kosong ∪ vendor, tanpa duplikat
	} `json:"kpi"`
	Items []Rundown `json:"items"`
}

type CreatePlaylistRequest struct {
	RundownID     *uuid.UUID `json:"rundown_id"`
	SongTitle     string     `json:"song_title" binding:"required,max=255"`
	Artist        string     `json:"artist" binding:"required,max=255"`
	MomentTag     string     `json:"moment_tag" binding:"required,max=100"`
	AudioFormat   string     `json:"audio_format" binding:"required,oneof=digital_playback live_band acoustic sound_fx"`
	SpotifyURL    *string    `json:"spotify_url"`
	YouTubeURL    *string    `json:"youtube_url"`
	AudioCueNotes *string    `json:"audio_cue_notes"`
	Status        string     `json:"status" binding:"omitempty,oneof=ready needs_review missing_file"`
}
// UpdatePlaylistRequest: field yang sama.

type PlaylistOverview struct {
	KPI struct {
		TotalTracks                                    int
		ByFormat                                       map[string]int // live_band, digital_playback, acoustic, sound_fx
		MappedMoments                                  int            // moment tag berbeda yang punya lagu
		Ready, NeedsReview, MissingFile                int
	} `json:"kpi"`
	Items []Playlist `json:"items"`
}
```

---

## 5. Master Data

### 5.1 Sesi dan agenda per jenis acara
Sesi bawaan sembilan jenis ada di [TEMPLATE_EVENT bab 3](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#3-ringkasan-registry-sembilan-jenis) dan **selalu** dibuat bersama proyek (bukan oleh seeder rundown). Agenda template non-pernikahan: [TEMPLATE_EVENT bab 7–14](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (semua sembilan jenis punya agenda template). Moment tag audio per jenis: [TEMPLATE_EVENT bab 6](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#6-moment-tag-audio-satu-set-per-jenis-sumber-tunggal-untuk-06).

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

Judul PDF "Rundown Resepsi dan Pernikahan". Agenda 1–8 masuk sesi `akad` (07.00–09.00), agenda 9–22 sesi `resepsi` (14 agenda, 09.15–13.00): total **8 + 14 = 22** seperti PDF, tanpa baris tambahan. Agenda 9 ("Resepsi", hanya `09.15`) terbuka; agenda 3 PDF bertuliskan "Pembacaan ayat suci Qori" tanpa kolom PIC, dipisah menjadi agenda "Pembacaan ayat suci" dengan PIC "Qori".

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

## 6. Pengujian yang Dibutuhkan
- Template pernikahan memuat tepat 11 + 8 + 14 = 33 agenda; dua agenda terbuka tersimpan (`end_time IS NULL`, durasi `NULL`).
- Rentang KPI Proyek Contoh: 100 + 120 + 225 = 445 menit.
- Agenda paralel tidak dijumlahkan; `ends_next_day` benar; `end_time < start_time` tanpa `ends_next_day` ditolak.
- Agenda tidak dapat menunjuk sesi atau vendor proyek lain; hapus sesi dengan agenda memerlukan `delete_agendas`.
- `apply-template`: `append` dua kali tidak menggandakan; `replace_template` mempertahankan agenda yang sudah berjalan.
- Tidak ada agenda atau PIC fiktif pada template.
