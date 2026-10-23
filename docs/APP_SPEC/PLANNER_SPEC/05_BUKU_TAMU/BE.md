# Menu 05: Buku Tamu & Undangan Digital — Backend Specification

Spesifikasi backend modul Go `plannerguest` (`backend/internal/modules/plannerguest/`) untuk **daftar tamu, RSVP, dan meja resepsionis hari-H** (PDF Hal. 13 sebagai struktur dasar). Aturan lintas modul (master sesi, uang, kode error, polling) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md).

> **Status ketergantungan.** Modul Undangan Digital **belum ada di backend** dan spesifikasinya (`DIGITAL_INVITATION_SPEC`) maupun layanan Gift (`VENDOR_GIFT_SPEC`) masih kosong. Modul ini dirancang agar **Mode Standalone** (Planner tanpa undangan digital) berfungsi penuh sekarang, dan integrasi undangan hanya bergantung pada dua antarmuka Go kecil (bab 5), bukan pada tabel atau FK modul lain.

---

## 1. Skema Database

Migrasi `011_000_planner_guests.sql` (sudah dijalankan di PostgreSQL 16):

```sql
-- +goose Up
-- Tabel tamu bersama untuk layanan Planner dan Undangan Digital (dual-key).
-- invitation_id sengaja TANPA foreign key: tabel undangan belum ada di backend.
-- Migrasi modul undangan menambahkan constraint-nya (lihat 05_BUKU_TAMU/BE.md bab 6).
CREATE TABLE planner_guests (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES planner_projects(id) ON DELETE CASCADE,
    invitation_id UUID,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    guest_side TEXT,                                    -- kunci sisi dari registry (wedding: pria/wanita); NULL = belum ditentukan
    category TEXT NOT NULL DEFAULT 'lainnya',           -- kunci kategori dari registry; kustom diperbolehkan
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    table_number TEXT,
    pax_count INT NOT NULL DEFAULT 2 CHECK (pax_count BETWEEN 1 AND 20),
    phone_number TEXT,
    formatted_wa_number TEXT,
    invitation_sent_status TEXT NOT NULL DEFAULT 'unsent' CHECK (invitation_sent_status IN ('unsent', 'sent', 'delivered')),
    invitation_sent_at TIMESTAMPTZ,
    rsvp_status TEXT NOT NULL DEFAULT 'unconfirmed' CHECK (rsvp_status IN ('unconfirmed', 'attending', 'declined', 'maybe')),
    pax_actual INT NOT NULL DEFAULT 0 CHECK (pax_actual >= 0),
    rsvp_submitted_at TIMESTAMPTZ,
    qr_code_token TEXT NOT NULL UNIQUE,                 -- 16 byte acak (hex), dibuat usecase
    is_checked_in BOOLEAN NOT NULL DEFAULT FALSE,
    checked_in_at TIMESTAMPTZ,
    souvenir_received BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (project_id IS NOT NULL OR invitation_id IS NOT NULL),
    -- Dengan project_id NULL (tamu milik undangan saja) FK komposit ini dilewati (MATCH SIMPLE).
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_guests_project_side_idx ON planner_guests (project_id, guest_side);
CREATE INDEX planner_guests_project_rsvp_idx ON planner_guests (project_id, rsvp_status);
CREATE INDEX planner_guests_project_category_idx ON planner_guests (project_id, category);
CREATE INDEX planner_guests_project_name_idx ON planner_guests (project_id, lower(name));
CREATE INDEX planner_guests_invitation_idx ON planner_guests (invitation_id) WHERE invitation_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_guests;
```

Keputusan desain:
- **`invitation_id` tanpa FK.** Tabel undangan belum ada di backend, sehingga FK ke sana membuat migrasi gagal. Kolom tetap ada untuk model *dual-key* (tamu dimiliki bersama layanan Planner dan Undangan); migrasi modul undangan nanti menambahkan constraint-nya (bab 5.3).
- **`project_id` nullable + `CHECK`.** Alur "undangan lebih dulu, planner menyusul" membutuhkan `UPDATE … WHERE project_id IS NULL`, jadi kolom tidak boleh `NOT NULL`: tamu milik undangan saja valid (`project_id IS NULL`, `invitation_id` terisi); semua query Planner selalu memfilter `project_id = $1`. FK komposit ke master sesi dilewati untuk baris tanpa proyek.
- **`guest_side` tanpa default `'pria'`.** Default itu membuat tamu seminar bernilai "pria". Nilai `NULL` berarti belum ditentukan; jenis acara tanpa pisah pihak (ulang tahun, wisuda, lainnya) tidak memakai kolom ini.
- **`personalized_url` dan `invitation_slug` dihapus** (denormalisasi yang basi bila slug berubah dan memakai `url_encode()` yang bukan fungsi PostgreSQL). Tautan personal dihitung saat dibaca (bab 2.1).
- **Token QR 128 bit** (`crypto/rand`, hex 32 karakter). Token tidak memuat potongan id proyek.
- `sub_event_scope` mengikuti master sesi (satu nilai per sesi; tidak ada varian seperti `resepsi_1`/`resepsi_2`). `pax_actual NOT NULL DEFAULT 0`; `rsvp_status` empat nilai (`maybe` ikut dihitung di dashboard).

---

## 2. Logika Bisnis

### 2.1 Tautan personal dan token QR
- Token: 16 byte acak → hex, dibuat usecase saat tamu dibuat; unik global (`UNIQUE`).
- **Tautan personal** = `{PUBLIC_BASE_URL}/i/{slug}?to={url.QueryEscape(nama)}` dihitung saat dibaca, dengan `slug` dari `InvitationLookup.PublicSlug(project.invitation_id)` (bab 5). Tanpa undangan terhubung (Standalone), `personalized_url` dan `whatsapp_url` bernilai `null` dan antarmuka menampilkan opsi "Undangan cetak / tautan eksternal".

### 2.2 Sinkronisasi RSVP
- RSVP publik berasal dari modul undangan dan masuk lewat antarmuka `GuestRSVPWriter` (bab 5), memperbarui `rsvp_status`, `pax_actual`, `rsvp_submitted_at` atas satu tamu (idempoten). Planner **tidak** mengekspos endpoint publik.
- Dasbor Planner menyegarkan lewat **polling** (`GET .../guests/summary`, 30 dtk; meja resepsionis 5 dtk). SSE/WebSocket tidak dipakai: arsitektur Adicara tidak memiliki keduanya dan Nginx belum dikonfigurasi untuk itu ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)).
- Pada Standalone, RSVP diubah manual: `PATCH .../guests/:guest_id/rsvp`.

### 2.3 Proyeksi porsi katering (diperbaiki)
Hanya tamu yang hadir yang masuk `confirmed_pax`; tamu yang menolak tidak pernah dihitung (menjumlahkan `pax_count` untuk status selain `attending` akan menghitung penolak sebagai hadir):

```sql
SELECT
  COALESCE(sum(CASE WHEN rsvp_status = 'attending' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0) AS confirmed_pax,
  COALESCE(sum(CASE WHEN rsvp_status IN ('maybe', 'unconfirmed') THEN pax_count END), 0)                   AS potential_pax,
  COALESCE(sum(CASE WHEN rsvp_status = 'declined' THEN pax_count END), 0)                                   AS declined_pax,
  count(*) FILTER (WHERE rsvp_status = 'attending')   AS attending_guests,
  count(*) FILTER (WHERE rsvp_status = 'declined')    AS declined_guests,
  count(*) FILTER (WHERE rsvp_status = 'maybe')       AS maybe_guests,
  count(*) FILTER (WHERE rsvp_status = 'unconfirmed') AS pending_guests
FROM planner_guests WHERE project_id = $1;
```

`recommended_portions = ceil(confirmed_pax × (1 + buffer_rate)) + buffer_pax`; `buffer_rate` default 0,25 dan `buffer_pax` default 0 adalah parameter query (`GET .../guests/catering-projection?buffer_rate=&buffer_pax=`), tidak disimpan. `potential_pax` ditampilkan terpisah (tidak masuk rekomendasi). Hasilnya **tidak** otomatis mengubah anggaran: tombol "Gunakan proyeksi RSVP" di pos katering (Menu 03) mengisi `quantity` atas tindakan pengguna ([03 BE §3.4](../03_KEUANGAN/BE.md#34-pos-berharga-satuan)).

### 2.4 Pesan WhatsApp undangan
Backend menyusun URL `https://wa.me/{nomor}?text=…` dari templat **netral jenis acara**:

```text
Yth. {nama tamu},

Tanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i pada {judul acara}:
📅 {tanggal sesi atau Hari-H}
📍 {nama sesi} ({jam mulai}–{jam selesai} jika ada)
🏛️ {venue}

Detail acara dan konfirmasi kehadiran (RSVP): {tautan personal}

Terima kasih.
```

Baris yang datanya kosong dihilangkan. Judul/tanggal/venue berasal dari proyek dan sesi tamu (`session_date` bila ada); teks tidak menyebut jenis acara tertentu dan setiap sesi memakai tanggalnya sendiri. Nomor harus lolos `NormalizeWA` (sama dengan [04 BE §2.3](../04_VENDOR_DAN_BELANJA/BE.md#23-normalisasi-nomor-whatsapp); fungsinya berada di `planner/domain/phone.go` supaya dapat diimpor modul ini tanpa mengimpor `plannervendor`).

### 2.5 Meja resepsionis hari-H
- **Scan QR**: `POST .../guests/scan-qr` `{qr_code_token}` → `SELECT … WHERE project_id = $1 AND qr_code_token = $2` (token milik proyek lain tidak dikenali). Batas 120 permintaan/menit/pengguna.
- **Check-in** (1 ketukan), atomik dan idempoten terhadap ketukan ganda:

```sql
UPDATE planner_guests
SET is_checked_in = TRUE, checked_in_at = now(), pax_actual = $3, souvenir_received = $4, updated_at = now()
WHERE id = $1 AND project_id = $2 AND is_checked_in = FALSE
RETURNING id, name, table_number, pax_actual, checked_in_at;
```
  Nol baris + tamu ada → `409 already_checked_in` beserta jam check-in sebelumnya (antarmuka menampilkan "Sudah masuk pukul 11.14"). `DELETE .../guests/:guest_id/checkin` membatalkan (tombol *Batal*).
- **Walk-in**: tamu baru `rsvp_status = 'attending'`, `pax_count = pax_actual`, sudah check-in, token dibuat.
- **Souvenir**: `souvenir_received` sebagai serah terima; stok souvenir = `floor(Σ quantity)` barang `purpose='souvenir'` berstatus `selesai` (jumlah barang boleh berdesimal, stok bilangan bulat; hilang bila tak ada barang bertanda souvenir; [04 BE §2.5](../04_VENDOR_DAN_BELANJA/BE.md#25-stok-souvenir)); antarmuka menampilkan `stok − diserahkan`.
- **Batasan:** satu check-in per tamu. Tamu yang hadir pada dua sesi berbeda dicatat sebagai dua baris tamu, atau check-in per sesi ditunda ke fase berikutnya.

### 2.6 Impor dan ekspor
- **Impor** `POST .../guests/bulk-import` (`multipart/form-data`, field `file` `.csv`/`.xlsx`, maks 5 MB dan 2.000 baris; `413 file_too_large`/`400 invalid_file`). Kolom templat: `Nama Tamu | Pihak | Kategori | Sesi | No Meja | Pax | No WhatsApp | Catatan`; `Pihak`, `Kategori`, `Sesi` dipetakan ke kunci registry/master sesi lewat label (tidak peka huruf besar); `Pihak` atau `Sesi` yang tak dikenal menolak baris itu, `Kategori` tak dikenal menjadi kategori kustom.
- `?dry_run=true` mengembalikan pratinjau: baris valid, baris galat (nomor baris + kode), dan duplikat (nama sama tanpa memandang huruf + telepon sama, di dalam file atau terhadap data proyek). Tanpa `dry_run` seluruh impor satu transaksi (semua atau tidak sama sekali) kecuali `skip_invalid=true`.
- **Ekspor** daftar tamu dan rekap kehadiran `.xlsx`/PDF; sel di-escape ([kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)); 10/menit/pengguna.

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, `BearerAuth` + `ProjectGuard` + `WriteGuard` (Gin ≥ 1.8 mengizinkan segmen statis seperti `summary` berdampingan dengan `:guest_id`).

### 3.1 Daftar Tamu & Undangan
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/guests?side=&category=&rsvp=&scope=&search=&limit=&offset=` | Halaman tamu (default 100, maks 500) **dan** `overview` (KPI + penghitung kategori/sisi dihitung atas seluruh data, bukan halaman). |
| `GET` | `/guests/summary` | Hanya `overview` (ringan; untuk polling). |
| `GET` | `/guests/:guest_id` | Detail. |
| `POST` | `/guests` | Tambah. |
| `PUT` | `/guests/:guest_id` | Ubah data. |
| `PATCH` | `/guests/:guest_id/sent-status` | `{"sent_status": "sent"}`. |
| `PATCH` | `/guests/:guest_id/rsvp` | `{"rsvp_status", "pax_actual"}` (RSVP manual, Standalone). |
| `DELETE` | `/guests/:guest_id` | `204`. |
| `POST` | `/guests/bulk-import?dry_run=&skip_invalid=` | §2.6. |
| `GET` | `/guests/export?format=xlsx\|pdf` | Daftar tamu. |
| `GET` | `/guests/catering-projection?buffer_rate=&buffer_pax=` | §2.3. |

### 3.2 Meja Resepsionis
| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/guests/reception-overview?status=&search=` | 4 KPI + antrean + `recent_checkins` (polling 5 dtk). |
| `POST` | `/guests/scan-qr` | `ScanQRRequest`. |
| `POST` | `/guests/:guest_id/checkin` | `CheckinRequest`; `409 already_checked_in`. |
| `DELETE` | `/guests/:guest_id/checkin` | Batalkan check-in. |
| `PATCH` | `/guests/:guest_id/souvenir` | `{"souvenir_received": true}`. |
| `POST` | `/guests/walk-in` | `WalkInRequest`. |
| `GET` | `/guests/export-attendance?format=xlsx\|pdf` | Rekap kehadiran final dengan jam masuk. |

Tidak ada `seed-default` untuk tamu: contoh tamu hanya ada di Proyek Contoh (bab 6).

---

## 4. DTO (Go)

```go
package domain

type RSVPStatus string // "unconfirmed" | "attending" | "declined" | "maybe"

type CreateGuestRequest struct {
	Name          string  `json:"name" binding:"required,max=255"`
	GuestSide     *string `json:"guest_side"`                    // kunci sisi dari registry; nil = belum ditentukan
	Category      string  `json:"category"`                      // kunci registry atau kustom; default "lainnya"
	SubEventScope string  `json:"sub_event_scope"`               // default "all"; harus ada di master sesi
	TableNumber   *string `json:"table_number"`
	PaxCount      int     `json:"pax_count" binding:"required,gte=1,lte=20"`
	PhoneNumber   *string `json:"phone_number"`                  // dinormalisasi; 400 invalid_phone
	Notes         *string `json:"notes"`
}
// UpdateGuestRequest: field yang sama.

type CheckinRequest struct {
	PaxActual        int     `json:"pax_actual" binding:"required,gte=1"`
	SouvenirReceived bool    `json:"souvenir_received"`
	Notes            *string `json:"reception_notes"`
}

type WalkInRequest struct {
	Name             string  `json:"name" binding:"required,max=255"`
	GuestSide        *string `json:"guest_side"`
	Category         string  `json:"category"`
	TableNumber      *string `json:"table_number"`
	PaxActual        int     `json:"pax_actual" binding:"required,gte=1"`
	SouvenirReceived bool    `json:"souvenir_received"`
	Notes            *string `json:"notes"`
}

type Guest struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	GuestSide            *string    `json:"guest_side"`
	Category             string     `json:"category"`
	SubEventScope        string     `json:"sub_event_scope"`
	TableNumber          *string    `json:"table_number,omitempty"`
	PaxCount             int        `json:"pax_count"`
	PhoneNumber          *string    `json:"phone_number,omitempty"`
	PersonalizedURL      *string    `json:"personalized_url"` // null di Standalone
	WhatsAppURL          *string    `json:"whatsapp_url"`
	InvitationSentStatus string     `json:"invitation_sent_status"`
	RSVPStatus           RSVPStatus `json:"rsvp_status"`
	PaxActual            int        `json:"pax_actual"`
	QRCodeToken          string     `json:"qr_code_token"`
	IsCheckedIn          bool       `json:"is_checked_in"`
	CheckedInAt          *time.Time `json:"checked_in_at,omitempty"`
	SouvenirReceived     bool       `json:"souvenir_received"`
	Notes                *string    `json:"notes,omitempty"`
}

// Sisi dan kategori generik (tanpa field "groom"/"bride"); label dari registry.
type GuestOverview struct {
	Total, TotalPax      int
	Sides      []struct{ Key, Label string; Invitations, Pax int } `json:"sides"`      // kosong untuk jenis tanpa sisi
	Categories []struct{ Key, Label string; Invitations, Pax int } `json:"categories"`
	Invitation struct{ Sent, Unsent int } `json:"invitation"`
	RSVP       struct{ Attending, Declined, Maybe, Unconfirmed, AttendingPax int } `json:"rsvp"` // jumlah = Total
}

type ReceptionOverview struct {
	CheckedInGuests, TotalGuests   int     // TotalGuests tanpa yang menolak
	AttendancePercent              float64
	ActualPax, ExpectedPax         int
	SouvenirDistributed            int
	SouvenirStock                  *int    // nil bila tidak ada barang bertanda souvenir
	PendingGuests                  int
	RecentCheckins                 []Guest
}

type CateringProjection struct {
	ConfirmedPax, PotentialPax, DeclinedPax, RecommendedPortions int
	AttendingGuests, DeclinedGuests, MaybeGuests, PendingGuests  int
	BufferRate float64; BufferPax int
}
```

---

## 5. Kontrak Antarmuka dengan Modul Undangan Digital

Spesifikasi `DIGITAL_INVITATION_SPEC` masih kosong. Apa yang **dibutuhkan Planner** dinyatakan di sini sebagai kontrak minimum supaya modul undangan kelak dapat ditulis tanpa mengubah Planner.

### 5.1 Antarmuka Go (dimiliki `planner/domain`)
```go
// Diimplementasikan modul undangan; dipakai Planner.
type InvitationLookup interface {
	// PublicSlug memastikan undangan milik userID dan mengembalikan slug publiknya.
	PublicSlug(ctx context.Context, userID, invitationID uuid.UUID) (string, error) // ErrNotFound bila bukan miliknya
}

// Diimplementasikan Planner (plannerguest); dipanggil modul undangan saat tamu mengisi RSVP publik.
type GuestRSVPWriter interface {
	ApplyRSVP(ctx context.Context, invitationID, guestID uuid.UUID, status RSVPStatus, pax int) error
}
```
Modul undangan tidak boleh membaca atau menulis `planner_guests` langsung; ia memanggil `GuestRSVPWriter`. Kepemilikan: baris tamu selalu dimiliki Planner (bila ada proyek) atau modul undangan (bila `project_id IS NULL`), tidak pernah keduanya sekaligus menulis.

### 5.2 Alur sinkronisasi
| Kasus | Alur |
|---|---|
| **A. Bundling dari awal** | Wizard menerima `invitation_id`; `planner_projects.invitation_id` diisi. Tamu yang dibuat lewat Planner mengisi `invitation_id` dari proyek; tamu yang dibuat di modul undangan juga menulis ke baris yang sama. Satu sumber data. |
| **B. Undangan dulu, Planner menyusul** | Tamu sudah ada dengan `invitation_id = X` dan `project_id IS NULL`. Saat proyek ditautkan (`PUT /v1/planner/projects/:id/invitation {invitation_id: X}`, hanya terdaftar bila modul undangan ada): `UPDATE planner_guests SET project_id = $1 WHERE invitation_id = $2 AND project_id IS NULL`. Kolom `project_id` nullable membuat ini valid. |
| **C. Planner dulu, undangan menyusul** | Tamu sudah ada dengan `project_id = Y`. Saat undangan dibuat dan ditautkan: set `planner_projects.invitation_id` lalu `UPDATE planner_guests SET invitation_id = $1 WHERE project_id = $2`. Tautan personal otomatis tersedia (dihitung saat dibaca). |
| **Standalone** | `invitation_id` kosong; RSVP manual; tombol WA memakai tautan eksternal pengguna. |

### 5.3 Migrasi modul undangan
Migrasi modul undangan menambahkan: `ALTER TABLE planner_guests ADD CONSTRAINT planner_guests_invitation_fk FOREIGN KEY (invitation_id) REFERENCES <tabel_undangan>(id) ON DELETE SET NULL;` dan hal yang sama untuk `planner_projects.invitation_id`. Sampai itu, kedua kolom hanyalah UUID bebas, dan endpoint `PUT .../invitation` tidak terdaftar.

### 5.4 Yang tidak ada di kontrak
Amplop/angpau digital lewat payment gateway **tidak ada** di modul ini: Adicara tidak menampung dana (prinsip Tab Tabungan, 03) dan layanan Gift belum punya spesifikasi. Jika kelak ada, hasilnya ditampilkan baca-saja di dashboard, bukan ditulis ke buku kas tabungan.

---

## 6. Data Contoh (Proyek Contoh)

PDF Hal. 13 **tidak berisi nama tamu**: ia hanya memuat dua tabel ("List Tamu Mempelai Laki-Laki/Perempuan") dengan enam kategori relasi berhitung 0 (Teman, Kolega, Keluarga, Tetangga, Teman Orang Tua, VIP). Dua puluh tamu di bawah ini **fiktif** (bukan dari PDF), hanya dimuat di Proyek Contoh dengan nomor telepon berpola `08120000xxxx`. Tidak ada tamu yang sudah check-in (Proyek Contoh berada di H-175).

| No | Nama tamu | Pihak | Kategori | Sesi | Meja | Pax | Telepon (fiktif) | RSVP | Undangan |
|:-:|---|:-:|---|:-:|:-:|:-:|---|:-:|:-:|
| 1 | Bpk. Prof. Bambang Soedjarwo | `pria` | `vip` | `resepsi` | VIP-01 | 2 | 081200000101 | `attending` | terkirim |
| 2 | Raditya Pratama & Pasangan | `pria` | `teman` | `resepsi` | T-04 | 2 | 081200000102 | `attending` | terkirim |
| 3 | Bpk. Hendro & Keluarga | `pria` | `keluarga` | `all` | FAM-01 | 4 | 081200000103 | `attending` | terkirim |
| 4 | Satria Wibawa (Kolega Kantor) | `pria` | `kolega` | `resepsi` | T-10 | 2 | 081200000104 | `maybe` | terkirim |
| 5 | Bpk. Joko Susilo (Ketua RT 04) | `pria` | `tetangga` | `resepsi` | T-08 | 2 | 081200000105 | `declined` | terkirim |
| 6 | Bpk. Dr. Ir. Gunawan (Sahabat Ayah) | `pria` | `teman_ortu` | `resepsi` | VIP-02 | 2 | 081200000106 | `attending` | terkirim |
| 7 | Bayu Anggoro & Partner | `pria` | `teman` | `resepsi` | T-05 | 2 | 081200000107 | `unconfirmed` | terkirim |
| 8 | Ibu Sri Wahyuni (Bibi CPP) | `pria` | `keluarga` | `all` | FAM-01 | 3 | 081200000108 | `attending` | terkirim |
| 9 | Fajar Nugraha | `pria` | `teman` | `resepsi` | T-06 | 1 | 081200000109 | `unconfirmed` | terkirim |
| 10 | Bpk. Rahmat Hidayat (Kolega) | `pria` | `kolega` | `resepsi` | T-11 | 2 | 081200000110 | `attending` | terkirim |
| 11 | Ibu Dra. Hj. Ratna Juwita | `wanita` | `vip` | `resepsi` | VIP-03 | 2 | 081200000111 | `attending` | terkirim |
| 12 | Anisa Rahmawati (Sahabat SMA) | `wanita` | `teman` | `all` | T-01 | 2 | 081200000112 | `attending` | terkirim |
| 13 | Bpk. H. Iwan & Keluarga | `wanita` | `keluarga` | `resepsi` | FAM-02 | 4 | 081200000113 | `unconfirmed` | terkirim |
| 14 | Sdri. Melisa Putri (Kolega Kantor) | `wanita` | `kolega` | `resepsi` | T-12 | 1 | 081200000114 | `attending` | terkirim |
| 15 | Ibu Nurhasanah (Tetangga RW 02) | `wanita` | `tetangga` | `resepsi` | T-09 | 2 | 081200000115 | `maybe` | belum |
| 16 | Ibu Hj. Fatimah (Sahabat Ibu) | `wanita` | `teman_ortu` | `resepsi` | VIP-04 | 2 | 081200000116 | `attending` | belum |
| 17 | Sdri. Tiara Andini | `wanita` | `teman` | `resepsi` | T-02 | 1 | 081200000117 | `unconfirmed` | belum |
| 18 | Bpk. Agus Santoso (Paman CPW) | `wanita` | `keluarga` | `all` | FAM-02 | 3 | 081200000118 | `attending` | belum |
| 19 | Sdri. Citra Lestari | `wanita` | `teman` | `resepsi` | T-03 | 2 | 081200000119 | `attending` | belum |
| 20 | Bpk. Dedi Sukamto (Kolega Ortu) | `wanita` | `teman_ortu` | `resepsi` | T-14 | 2 | 081200000120 | `declined` | belum |

Ringkasan: 20 undangan, **43 pax** (Pria 10 undangan / 22 pax; Wanita 10 / 21); terkirim 14; RSVP: 12 hadir, 2 ragu, 2 tidak, 4 belum. Enam kategori PDF tercakup semuanya.

---

## 7. Pengujian yang Dibutuhkan
- Proyeksi katering: tamu `declined` tidak masuk `confirmed_pax`; `maybe`/`unconfirmed` hanya di `potential_pax`.
- Check-in ganda serentak: satu berhasil, satu `409 already_checked_in`; token proyek lain tidak dikenali.
- Impor: `dry_run` tidak menulis; duplikat terdeteksi; satu baris galat menggagalkan seluruh impor tanpa `skip_invalid`; file > 5 MB ditolak.
- Tamu `project_id IS NULL` tidak pernah muncul pada query Planner.
- Standalone: `personalized_url = null`; RSVP manual memperbarui penghitung.
- Sisi/kategori: seminar tidak memunculkan "Pria/Wanita".
