# Menu 05: Buku Tamu & Undangan Digital — Backend Specification

Spesifikasi backend modul Go `plannerguest` (`backend/internal/modules/plannerguest/`) untuk **daftar tamu, RSVP, dan meja resepsionis hari-H** (PDF Hal. 13 sebagai struktur dasar). Aturan lintas modul (master sesi, uang, kode error, polling, BFF) ada di [00_KONTRAK_BERSAMA](../00_KONTRAK_BERSAMA/README.md); titik sambung yang sudah disediakan Menu 01 (`SummaryProvider`, `SampleOverlay`, registry) dijelaskan di [bab 7](#7-sambungan-ke-menu-01).

> **Status (2026-10-25):** modul `plannerguest` **sudah diimplementasikan** (`backend/internal/modules/plannerguest/`, 19 endpoint, provider dan overlay `guests`, registry `GuestSides`/`GuestCategories`; rangkuman dan penyimpangan: [docs/backend/planner/2026-10-25-planner-menu-05.md](../../../backend/planner/2026-10-25-planner-menu-05.md)). Tabelnya sudah ada sejak migrasi `011_000_planner_guests.sql` di `backend/internal/migrate/sql/` (**identik** dengan blok SQL bab 1; tidak ada migrasi baru) dan Menu 01–04 sudah memakainya: pemindahan tamu saat sesi dihapus ([01 BE §4.6](../01_DASHBOARD/BE.md#46-sesi-acara-master-sesi)), `usage.guests` pada sesi, `guest_pax_count` pada kartu Hub, tipe `GuestSummary` di `planner/domain/summary.go`, dan hapus proyek yang ikut menghapus tamu. Widget `guests` di dashboard terisi sejak provider terdaftar. Frontend menu ini belum ada ([FE.md](FE.md)); respons HTTP asli **sama dengan fixture FE** ([bab 8.4](#84-hasil-implementasi-2026-10-25)). Blok kode di dokumen ini adalah acuan yang diaudit; berkas di repo adalah sumbernya sekarang (komentar diterjemahkan ke bahasa kode yang ada; `Option`, `GuestSide`, `RSVPSummary`, `RSVPStatus`, dan `NormalizeWA` dimiliki `planner/domain` dan dipakai lewat alias atau awalan `pd.`; `TemplateInput` ada di paket `domain`; `GuestSpec.SideLabel` menerima `*string` (nil = "Belum ditentukan") dan `ReceptionFilters` ada di `overview.go`. Perubahan perilaku tercatat di [bab 10 butir 29–38](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
>
> Dokumen ini diaudit dengan **menjalankan** SQL dan algoritmanya sebelum kode ditulis: PostgreSQL 16.2 sementara dengan 12 migrasi nyata, **backend Menu 01–04 yang sebenarnya** (Proyek Contoh dan sembilan jenis acara dibuat lewat HTTP), PDF Hal. 13 dibaca ulang dengan `pdfminer`, dan modul acuan Go (`excelize` v2.11.0, Gin 1.12) berisi DTO bab 4, logika bab 2, dan pembaca impor ([bab 8.1](#81-yang-sudah-dijalankan-pada-audit-spesifikasi)). Blok kode di dokumen ini disalin dari berkas yang **dikompilasi dan diuji** (43 tes Go, 24 mutasi kode semuanya digagalkan) dan DTO-nya dibandingkan kunci demi kunci dengan tipe TypeScript FE (21 pasangan tipe, 125 jalur JSON, nol selisih). Versi sebelumnya (327 baris) benar pada DDL dan angka contoh, tetapi memuat janji yang tidak dapat dipenuhi atau gagal bila dijalankan: ekspor PDF yang pustakanya belum disetujui, impor `multipart` yang ditolak BFF, tautan personal tanpa identitas tamu sehingga RSVP publik tidak dapat dikaitkan, FK undangan yang membuat hapus undangan gagal, pembulatan porsi dengan `float64`, penghitung "belum hadir" yang salah ketika tamu yang menolak tetap datang, `AttendancePercent` (kontrak melarang persentase di respons), dan templat impor yang tidak punya endpoint. Semuanya tercatat di [CHANGELOG spesifikasi](../CHANGELOG.md) dan [bab 10](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk).
>
> Pola dokumen mengikuti [01 BE](../01_DASHBOARD/BE.md), [02 BE](../02_TIMELINE_DAN_TUGAS/BE.md), [03 BE](../03_KEUANGAN/BE.md), dan [04 BE](../04_VENDOR_DAN_BELANJA/BE.md): DTO tanpa tag `binding` (validasi di usecase dengan pesan `<field>: <alasan>`), bentuk respons per endpoint ([bab 3.3](#33-bentuk-respons)), validasi dan batas ([bab 3.4](#34-validasi-dan-batas)), bab sambungan ke Menu 01 ([bab 7](#7-sambungan-ke-menu-01)), pengujian dipetakan ke nama tes ([bab 8](#8-pengujian)), urutan pengerjaan ([bab 9](#9-urutan-pengerjaan-dan-definisi-selesai)), dan celah yang menunggu keputusan ([bab 10](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Bab 5 (kontrak modul Undangan Digital) dan bab 6 (data contoh) tetap di nomor semula karena dirujuk Menu 01 dan 04.

> **Status ketergantungan.** Modul Undangan Digital **belum ada di backend** dan spesifikasinya (`DIGITAL_INVITATION_SPEC`) maupun layanan Gift (`VENDOR_GIFT_SPEC`) masih kosong. Modul ini dirancang agar **Mode Standalone** (Planner tanpa undangan digital) berfungsi penuh sekarang, dan integrasi undangan hanya bergantung pada dua antarmuka Go kecil ([bab 5](#5-kontrak-antarmuka-dengan-modul-undangan-digital)), bukan pada tabel atau FK modul lain.

> **Acuan angka.** PDF Hal. 13 (dibaca ulang dengan `pdfminer`) hanya berisi **dua tabel berdampingan**, *LIST TAMU MEMPELAI LAKI-LAKI* dan *LIST TAMU MEMPELAI PEREMPUAN*, masing-masing dengan **enam penghitung** (Teman, Kolega, Keluarga, Tetangga, Teman Orang Tua, VIP) bernilai 0; tidak ada nama tamu. Dua puluh tamu pada [bab 6](#6-data-contoh-proyek-contoh) fiktif dan hanya ada di Proyek Contoh.

---

## 1. Skema Database

Migrasi `011_000_planner_guests.sql` (sudah ada dan sudah dijalankan di PostgreSQL 16; blok di bawah **identik** dengan berkas itu):

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

Keputusan desain dan hal yang terbukti saat dijalankan:
- **`invitation_id` tanpa FK.** Tabel undangan belum ada di backend, sehingga FK ke sana membuat migrasi gagal. Kolom tetap ada untuk model *dual-key* (tamu dimiliki bersama layanan Planner dan Undangan). Migrasi modul undangan menambahkan constraint-nya dengan aturan di [bab 5.3](#53-migrasi-modul-undangan): `ON DELETE SET NULL` saja **tidak cukup** (dijalankan: menghapus undangan yang masih punya tamu-saja-undangan gagal pada `CHECK`).
- **`project_id` nullable + `CHECK`.** Alur "undangan lebih dulu, planner menyusul" membutuhkan `UPDATE … WHERE project_id IS NULL`: tamu milik undangan saja valid (`project_id IS NULL`, `invitation_id` terisi). Semua query Planner selalu memfilter `project_id = $1`, jadi baris itu tidak pernah muncul di Planner. FK komposit ke master sesi dilewati untuk baris tanpa proyek (`MATCH SIMPLE`, dijalankan) dan **diperiksa kembali saat `COMMIT`** bila baris ditautkan ke proyek (FK `DEFERRABLE`; dijalankan: `sub_event_scope` yang tidak ada di proyek gagal pada `COMMIT`, bukan pada `UPDATE`).
- **`guest_side` tanpa default `'pria'`.** Default itu membuat tamu seminar bernilai "pria". `NULL` berarti belum ditentukan; jenis acara tanpa pisah pihak (ulang tahun, wisuda, lainnya) tidak memakai kolom ini.
- **`category`** menyimpan kunci registry, kunci cadangan `lainnya` (bawaan bila tidak diisi; **bukan** kategori registry jenis acara mana pun, [TEMPLATE_EVENT 4.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#42-label-sisi-dan-kategori-tamu-milik-menu-05)), atau teks kategori kustom (≤ 60) apa adanya ([bab 2.8](#28-ringkasan-penghitung-kategori-dan-pihak)).
- **`pax_actual` memegang dua hal dalam satu kolom:** jumlah yang menyatakan hadir lewat RSVP, lalu jumlah yang benar-benar datang sesudah check-in (menimpa). Membatalkan check-in tidak mengembalikan angka RSVP lama ([bab 10 butir 3](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)). `0` berarti "ikut `pax_count`". DDL tidak membatasi atasnya (dijalankan: `pax_actual = 500` diterima) dan `pax_count` dibatasi 1–20, sehingga walk-in dengan `pax_actual = 21` melanggar `planner_guests_pax_count_check` (500 bila tidak dijaga); **usecase** membatasi semua jumlah pax pada 1–20 ([bab 3.4](#34-validasi-dan-batas)).
- **`invitation_sent_status`** tiga nilai; `delivered` hanya akan ditulis modul undangan (WhatsApp `wa.me` tidak memberi tanda terkirim). Planner menulis `unsent` dan `sent`; keduanya selain `unsent` dihitung "terkirim" (sama dengan kueri provider Menu 01).
- **`personalized_url` dan `invitation_slug` tidak disimpan** (denormalisasi yang basi bila slug berubah). Tautan personal dihitung saat dibaca ([bab 2.1](#21-tautan-personal-dan-token-qr)).
- **Token QR 128 bit** (`crypto/rand`, hex 32 karakter), `UNIQUE`; tidak memuat potongan id proyek.
- `sub_event_scope` mengikuti master sesi (satu nilai per sesi). `rsvp_status` empat nilai (`maybe` ikut dihitung di dashboard).
- **Constraint yang disarankan tetapi belum ada di migrasi** (diizinkan diubah di tempat pada tahap pengembangan): `CHECK (pax_actual BETWEEN 0 AND 20)` dan `CHECK (is_checked_in = (checked_in_at IS NOT NULL))`. Usecase sudah menjamin keduanya (satu `UPDATE` menulis kedua kolom); constraint hanya pertahanan berlapis ([bab 10 butir 26](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

### 1.1 Tabel dan view milik modul lain yang disentuh

| Tabel | Siapa | Cara | Keterangan |
|---|---|---|---|
| `planner_projects` | Menu 05 | `SELECT … FOR UPDATE` (kunci proyek) | batas 5.000 tamu dan urutan `sort_order` saat tambah, walk-in, dan impor; bukan penulisan |
| `planner_shopping_items` (Menu 04) | Menu 05 | baca-saja | stok souvenir ([bab 2.5](#25-meja-resepsionis-hari-h)) |
| `planner_event_sessions` | Menu 05 | FK komposit `DEFERRABLE` | nama sesi dibaca untuk impor dan templat |
| `planner_guests` | Menu 01 | `UPDATE sub_event_scope` saat sesi dihapus; `count`/`sum` untuk `usage` dan `guest_pax_count` Hub | sudah ada di kode; terbukti: tamu sesi yang dihapus berpindah ke `move_to` |
| `planner_guests` | Menu 01 | `ON DELETE CASCADE` dari proyek | terbukti: hapus proyek menghapus tamunya |

Menu 05 tidak menulis tabel modul lain. Pengisian porsi katering ke pos anggaran dilakukan **klien** lewat endpoint Menu 03 ([bab 2.3](#23-proyeksi-porsi-katering-diperbaiki)), bukan lewat SQL antar modul.

---

## 2. Logika Bisnis

### 2.1 Tautan personal dan token QR

- Token: 16 byte acak → hex 32 karakter, dibuat usecase saat tamu dibuat; unik global (`UNIQUE`). Bentrok (hampir mustahil) tidak dicoba ulang di dalam transaksi, sebab pernyataan yang gagal membatalkan transaksi PostgreSQL: indeks unik menjawab `409 conflict` dan mengulang permintaan memberi token baru ([bab 10 butir 29](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Tautan personal** = `domain.PersonalURL(PUBLIC_BASE_URL, slug, nama, token)` = `{PUBLIC_BASE_URL}/i/{slug}?g={token}&to={nama}` ([bab 5.1](#51-antarmuka-go-dimiliki-plannerdomain)), dihitung saat dibaca. `slug` dari `InvitationLookup.PublicSlug(userID, project.invitation_id)`, **satu kali per permintaan** (bukan per tamu). Tanpa undangan terhubung (Standalone, dan setiap build sampai modul undangan ada), `personalized_url` hilang dari respons.
- **Mengapa `g`:** versi sebelumnya hanya membawa `to={nama}`, padahal `GuestRSVPWriter.ApplyRSVP` membutuhkan id tamu yang tidak dimiliki halaman publik. Mencocokkan nama rapuh (nama kembar) dan dapat dipalsukan. Token tiket 128 bit tidak dapat ditebak, jadi tautan itu sendiri adalah kredensial tamu (juga tiket QR); konsekuensinya tautan yang diteruskan memberi penerimanya hak RSVP dan check-in sebagai tamu itu, sama seperti tiket. Alternatif: kolom token RSVP terpisah ([bab 10 butir 7](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- Halaman `/i/[slug]` yang ada di repo membaca `to` (lalu `guest`) untuk menyapa, dan hanya memeriksa `guest` untuk `noindex` ([bab 10 butir 27](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

### 2.2 Sinkronisasi RSVP dan status kirim

- RSVP publik berasal dari modul undangan dan masuk lewat `GuestRSVPWriter.ApplyRSVP(invitationID, qrToken, status, pax)` ([bab 5.1](#51-antarmuka-go-dimiliki-plannerdomain)). Planner **tidak** mengekspos endpoint publik.
- **RSVP manual** (Standalone): `PATCH …/guests/:guest_id/rsvp` `{rsvp_status, pax_actual}`. `pax_actual` hanya berarti untuk `attending` (0 = ikut `pax_count`); status lain menyimpan 0. `rsvp_submitted_at` diisi `now()` pada setiap jawaban baru selain `unconfirmed` dan dikosongkan bila kembali ke `unconfirmed` (dijalankan); **jawaban yang sama dikirim lagi** (permintaan yang diulang, formulir publik yang dikirim dua kali) membiarkan cap lama ([bab 10 butir 30](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)). **Tamu yang sudah check-in tidak dapat diubah RSVP-nya** (`409 already_checked_in`; batalkan check-in dulu), karena `pax_actual` sudah menjadi jumlah di pintu. `ApplyRSVP` pada tamu yang sudah check-in tidak mengubah apa pun dan berhasil.
- **Status kirim:** `PATCH …/sent-status` `{sent_status: "unsent"|"sent"}` mengisi `invitation_sent_at = now()` atau mengosongkannya. Membuka WhatsApp tidak mengubah status; pengguna menekan "Tandai terkirim".
- Dasbor Planner menyegarkan lewat **polling**: `GET …/guests/summary` 30 dtk, meja resepsionis 5 dtk ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)). SSE/WebSocket tidak dipakai: arsitektur Adicara tidak memiliki keduanya.

### 2.3 Proyeksi porsi katering (diperbaiki)

Hanya tamu yang hadir yang masuk `confirmed_pax`; tamu yang menolak tidak pernah dihitung (menjumlahkan `pax_count` untuk status selain `attending` menghitung penolak sebagai hadir). Dijalankan terhadap 20 tamu contoh: `27 / 12 / 4`.

```sql
-- $1 = project_id
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

`recommended_portions = ceil(confirmed_pax × (100 + buffer_percent) / 100) + buffer_pax` dengan **aritmetika bulat**. Parameter query `buffer_percent` (bilangan bulat 0–100, bawaan 25) dan `buffer_pax` (0–1.000, bawaan 0); tidak disimpan. Versi sebelumnya memakai `buffer_rate` pecahan (`1,25`) dan `ceil` `float64`, yang menambah satu porsi pada kasus tertentu (`200 × 1,1` = `220,00000000000003` → 221 padahal 220); pada ruang 0–3.000 tamu × 0–100% pembulatan `float64` berbeda dari jawaban eksak pada **663 dari 303.101** pasangan, dan versi bilangan bulat tidak pernah ([tes](#82-pengujian-pada-implementasi)).

```go
// backend/internal/modules/plannerguest/domain/projection.go
package domain

import "errors"

// Batas parameter proyeksi (usulan; BE 3.4). Persentase bulat, bukan pecahan desimal: 200 x 1,1 menjadi 221
// dengan float64 (220,00000000000003 dibulatkan ke atas) padahal jawabannya 220.
const (
	DefaultBufferPercent = 25
	MaxBufferPercent     = 100
	MaxBufferPax         = 1000
)

var ErrBuffer = errors.New("buffer out of range")

// RecommendedPortions = ceil(confirmedPax x (100 + bufferPercent) / 100) + bufferPax, dengan aritmetika bulat.
func RecommendedPortions(confirmedPax, bufferPercent, bufferPax int) (int, error) {
	if confirmedPax < 0 || bufferPercent < 0 || bufferPercent > MaxBufferPercent || bufferPax < 0 || bufferPax > MaxBufferPax {
		return 0, ErrBuffer
	}
	return (confirmedPax*(100+bufferPercent)+99)/100 + bufferPax, nil
}
```

`potential_pax` ditampilkan terpisah (tidak masuk rekomendasi). Hasilnya **tidak** otomatis mengubah anggaran dan tidak ada SQL antar modul: modal pos Menu 03 ([03 FE bab 9.3](../03_KEUANGAN/FE.md#93-komponen-dan-modal)) memanggil `GET …/guests/catering-projection`, mengisi `quantity` dengan `recommended_portions` dan satuan "porsi", lalu mengirim `PUT …/budget/items/:item_id` (mode `unit`) atas tindakan pengguna ([03 BE §3.4](../03_KEUANGAN/BE.md#34-pos-berharga-satuan)). Terbukti pada backend Menu 03 asli: `PUT` pos katering Proyek Contoh dengan `pricing_mode: unit`, `quantity: 34` dijawab `200`; **`PUT` mengganti penuh dan tanpa `vendor_id` melepas vendor pos itu** (jadi modal mengirim ulang `vendor_id`, [03 BE bab 9 butir 11](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)); `quantity: 0` ditolak `400 quantity: must be between 0.01 and 1000000`, sehingga tombol dinonaktifkan bila `recommended_portions = 0`. Tombol muncul pada pos berharga satuan mana pun ([03 BE bab 9 butir 8](../03_KEUANGAN/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

### 2.4 Pesan WhatsApp undangan

**Dibentuk klien**, seperti pesan vendor Menu 04 ([04 BE §2.3](../04_VENDOR_DAN_BELANJA/BE.md#23-normalisasi-nomor-whatsapp)): backend menyimpan `phone_number` (apa adanya) dan `formatted_wa_number` (`62…`, hasil `NormalizeWA`, hilang bila kosong) serta memberi `personalized_url` bila ada; klien menyusun `https://wa.me/{62…}?text={encodeURIComponent}` ([FE bab 5](FE.md#5-library-libplannerguestts), `waMessage`/`guestWaLink`). Versi sebelumnya menyusunnya di server (`whatsapp_url` per tamu): menambah perkiraan 600 byte × 100 tamu pada setiap pembacaan daftar, memerlukan nama bulan dan hari Indonesia di Go, dan membuat templat berada di dua tempat.

```text
Yth. {nama tamu},

Tanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i pada {judul acara}:
📅 {tanggal sesi tamu atau Hari-H}
📍 {nama sesi} ({mulai}–{selesai} jika ada)        ← hanya untuk sesi nyata (bukan Umum)
🏛️ {lokasi sesi, atau venue_name proyek}, {city}

Detail acara dan konfirmasi kehadiran (RSVP): {personalized_url}     ← bila undangan terhubung
Mohon konfirmasi kehadiran dengan membalas pesan ini.                ← pengganti baris di atas pada Standalone

Terima kasih.
```

Baris yang datanya kosong dihilangkan. **Standalone tetap punya tombol WhatsApp** (selama nomor tamu sah): pesannya tanpa tautan, dengan baris "Mohon konfirmasi kehadiran dengan membalas pesan ini", dan RSVP dicatat manual. Versi sebelumnya menjadikan `whatsapp_url` `null` pada Standalone dan menyebut "tombol WA memakai tautan eksternal pengguna", padahal tidak ada kolom tempat tautan itu disimpan; hasilnya mode utama yang bisa dibangun sekarang tidak punya WhatsApp sama sekali. Teks tidak menyebut jenis acara tertentu. Tanggal dan jam memakai sesi tamu bila ada (`session_date`, `start_time`, `end_time`, `location_name` pada `SessionBrief`), selain itu Hari-H; judul, `venue_name`, dan `city` berasal dari proyek ([`metadata`](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#5-kolom-metadata-proyek-planner_projectsmetadata), semua jenis acara).

### 2.5 Meja resepsionis hari-H

- **Scan QR:** `POST …/guests/scan-qr` `{qr_code_token}` → `SELECT … WHERE project_id = $1 AND qr_code_token = $2` (dijalankan: token proyek lain tidak dikenali). Token tidak dikenal, salah bentuk, atau milik proyek lain → `404 not_found` (tidak membedakan). Tamu yang sudah check-in tetap `200` (antarmuka menampilkan jam masuknya). Batas 120 permintaan/menit/pengguna (hanya rute ini).
- **Check-in** (satu ketukan), atomik dan aman terhadap ketukan ganda (dijalankan dengan dua sesi `psql`: sesi pertama menahan baris 1,5 detik, sesi kedua menunggu lalu mendapat **0 baris**):

```sql
-- $1 = guest_id, $2 = project_id, $3 = pax_actual (1-20), $4 = souvenir_received
UPDATE planner_guests
SET is_checked_in = TRUE, checked_in_at = now(), pax_actual = $3, souvenir_received = $4, updated_at = now()
WHERE id = $1 AND project_id = $2 AND is_checked_in = FALSE
RETURNING id, name, table_number, pax_actual, checked_in_at;
```

  Nol baris: tamu tidak ada → `404`; ada → `409 already_checked_in`. Respons galat hanya `{code, message}`, jadi **klien mengambil ulang** `GET …/guests/:guest_id` untuk menampilkan "Sudah masuk pukul 11.14". Check-in **tidak mengubah RSVP**: tamu `declined`, `maybe`, atau `unconfirmed` yang datang tetap tercatat begitu dan dihitung hadir di meja resepsionis (butir berikut).
- **Batalkan:** `DELETE …/guests/:guest_id/checkin` → `UPDATE … SET is_checked_in = FALSE, checked_in_at = NULL, souvenir_received = FALSE` (dijalankan); `pax_actual` dibiarkan. Idempoten: tamu yang belum check-in dijawab `200` apa adanya. Tamu walk-in yang dibatalkan tetap ada sebagai tamu `attending` yang belum datang (hapus dengan `DELETE`).
- **Walk-in:** `POST …/guests/walk-in` membuat tamu baru `rsvp_status = 'attending'`, `pax_count = pax_actual` (1–20), `rsvp_submitted_at = now()`, sesi `all`, sudah check-in, token dibuat; memeriksa batas 5.000 di bawah kunci proyek. `201 Guest`.
- **Souvenir:** `PATCH …/guests/:guest_id/souvenir` `{souvenir_received}`. **Satuan: per undangan** (satu tanda per baris tamu, berapa pun `pax`); stok = `floor(Σ quantity)` barang `purpose = 'souvenir'` berstatus `selesai` ([04 BE §2.5](../04_VENDOR_DAN_BELANJA/BE.md#25-stok-souvenir); dijalankan pada backend asli: 300 + 50,5 selesai dan 20 proses → `souvenir_stock: 350`). `souvenir_stock` hilang bila tak ada barang bertanda souvenir. Klien menghitung `stok − diserahkan` (bisa negatif: peringatan). Alternatif satuan per orang ([bab 10 butir 2](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Definisi angka meja resepsionis** (versi sebelumnya: "belum hadir" = total tanpa penolak − sudah hadir, yang **salah bila tamu penolak tetap datang**: dengan 12 hadir dan satu penolak di antaranya menjawab 6 padahal 7 masih di luar; dijalankan):

```sql
-- $1 = project_id
SELECT count(*) FILTER (WHERE is_checked_in)                                           AS checked_in_guests,
       count(*) FILTER (WHERE rsvp_status <> 'declined' OR is_checked_in)              AS expected_guests,
       COALESCE(sum(pax_actual) FILTER (WHERE is_checked_in), 0)                       AS actual_pax,
       COALESCE(sum(CASE WHEN is_checked_in THEN pax_actual
                         WHEN rsvp_status <> 'declined' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0) AS expected_pax,
       count(*) FILTER (WHERE souvenir_received)                                       AS souvenir_distributed
FROM planner_guests WHERE project_id = $1;

SELECT category, count(*) FROM planner_guests
WHERE project_id = $1 AND (rsvp_status <> 'declined' OR is_checked_in) GROUP BY category;       -- chip kategori

SELECT count(*) AS items, COALESCE(floor(sum(quantity) FILTER (WHERE status = 'selesai')), 0)::bigint AS stock
FROM planner_shopping_items WHERE project_id = $1 AND purpose = 'souvenir';                      -- items = 0 → tanpa souvenir_stock
```

  `expected_guests` = tamu yang belum menolak, atau menolak tetapi sudah hadir; `pending_guests = expected_guests − checked_in_guests` (selalu ≥ 0). `ReceptionKPI` tidak memuat persentase (kontrak bab 6): klien membagi dengan `percent`.
- **Antrean:** `GET …/guests/reception-overview?status=&category=&search=&limit=&offset=` mengembalikan **satu paket** untuk satu siklus polling 5 detik: KPI, `filters` (jumlah Semua/Sudah hadir/Belum hadir dan per kategori; tanpa pencarian; kategori registry juga yang 0, lalu `lainnya` dan teks kustom yang dipakai tamu yang diharapkan, `custom: true`), `recent_checkins` (10 terbaru), dan satu halaman `guests` (bawaan 50, maks 200; versi sebelumnya memuat seluruh antrean tanpa batas pada tiap polling). `status`: `all` (bawaan) = tamu yang diharapkan, `in` = sudah hadir, `pending` = diharapkan dan belum hadir. **Tamu yang menolak dan belum datang disembunyikan dari `all` dan `pending`, tetapi muncul bila `search` diisi** (tamu menolak yang tiba di pintu harus bisa dicari; dijalankan: pencarian "dedi" menemukannya hanya dengan aturan ini). Urutan: `in` menurut `checked_in_at` terbaru; selain itu nama (tanpa membedakan huruf), lalu `id`.
- **Batasan:** satu check-in per tamu. Tamu yang hadir pada dua sesi berbeda dicatat sebagai dua baris tamu (check-in per sesi ditunda: [bab 10 butir 16](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **QR dalam Standalone:** Planner tidak membuat gambar QR (membutuhkan pustaka; [bab 10 butir 8](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)). Tiket ber-QR berasal dari modul undangan; sampai ada, jalur utama Standalone adalah pencarian nama/meja, dan ekspor daftar tamu memuat kolom *Kode Tiket* (token) untuk dicetak dengan alat luar.

### 2.6 Impor dan ekspor

#### Impor

`POST …/guests/bulk-import?dry_run=&skip_invalid=&skip_duplicates=`. **Berkas dikirim sebagai JSON**, bukan `multipart/form-data`:

```json
{ "filename": "tamu.xlsx", "content_base64": "UEsDBBQ…" }
```

Alasannya dari kontrak yang sudah ada: BFF hanya meneruskan `application/json` dan memotong badan di 64 KiB ([01 FE bab 6](../01_DASHBOARD/FE.md#6-bff-dashboardapiplanner)), dan `security.checkOrigin` Astro memeriksa POST berbentuk form di balik proxy yang menghentikan TLS ([01 FE bab 2](../01_DASHBOARD/FE.md#3-prasyarat-yang-sudah-ada-di-repo)); versi sebelumnya (`multipart`) tidak dapat melewati keduanya. Berkas 5 MB menjadi sekitar 6,7 MiB base64, di bawah `client_max_body_size 8m` Nginx. Perubahan yang dibutuhkan: BFF mengizinkan **batas badan 7,5 MiB hanya untuk jalur `…/guests/bulk-import`** (jalur lain tetap 64 KiB); backend membaca badan lewat `http.MaxBytesReader(7,5 MiB)` (lebih → `413 file_too_large`) dan `DecodeUpload` memeriksa hasil decode (> 5 MiB → `413 file_too_large`).

```go
// backend/internal/modules/plannerguest/domain/upload.go
package domain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
)

// ErrTooLarge menjadi 413 file_too_large (berkas lebih dari MaxImportBytes setelah di-decode).
var ErrTooLarge = errors.New("file too large")

type FileKind int

const (
	KindCSV FileKind = iota
	KindXLSX
)

// DecodeUpload membuka ImportRequest: ekstensi nama berkas harus .csv atau .xlsx dan isinya harus cocok dengan
// ekstensi itu (.xlsx adalah arsip zip, "PK\x03\x04"). Pembacaan badan HTTP dibatasi lebih dulu oleh
// http.MaxBytesReader (7,5 MiB); fungsi ini membatasi hasil decode pada 5 MiB.
func DecodeUpload(r ImportRequest) ([]byte, FileKind, error) {
	name := strings.ToLower(strings.TrimSpace(r.Filename))
	var kind FileKind
	switch {
	case strings.HasSuffix(name, ".csv"):
		kind = KindCSV
	case strings.HasSuffix(name, ".xlsx"):
		kind = KindXLSX
	default:
		return nil, 0, &FileError{"filename must end with .csv or .xlsx"}
	}
	if base64.StdEncoding.DecodedLen(len(r.ContentBase64)) > MaxImportBytes+3 {
		return nil, 0, ErrTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(r.ContentBase64)
	if err != nil {
		return nil, 0, &FileError{"content_base64 is not valid base64"}
	}
	if len(data) > MaxImportBytes {
		return nil, 0, ErrTooLarge
	}
	if len(data) == 0 {
		return nil, 0, &FileError{"empty file"}
	}
	isZip := bytes.HasPrefix(data, []byte("PK\x03\x04"))
	if kind == KindXLSX && !isZip {
		return nil, 0, &FileError{"the content is not an .xlsx file"}
	}
	if kind == KindCSV && isZip {
		return nil, 0, &FileError{"the content is not a CSV file"}
	}
	return data, kind, nil
}
```

Pembacaan berkas (`.csv` atau `.xlsx`) lalu `BuildImportPlan` (kode di bawah; **sudah dijalankan**, [bab 8.3](#83-hasil-kode-acuan-2026-10-25)). Aturan yang terbukti dibutuhkan:

| Masalah nyata | Penanganan |
|---|---|
| CSV "UTF-8" dari Excel berbahasa Indonesia memakai **BOM** dan pemisah **titik koma**: `encoding/csv` bawaan membaca seluruh baris sebagai **satu kolom** berawalan `﻿` (dijalankan) | BOM dibuang; pemisah `,` `;` atau tab dideteksi dari baris pertama di luar tanda kutip (seri → koma) |
| CSV Windows (bukan UTF-8) | bila bukan UTF-8 yang sah, didekode sebagai Windows-1252 (`golang.org/x/text/encoding/charmap`; `x/text` sudah ada di `go.mod`); selain itu `invalid_file` |
| Excel menyimpan nomor yang diketik `081200000101` sebagai **angka** dan membuang `0` di depan: sel terbaca `81200000101` (dijalankan) | **tidak ditebak** (`NormalizeWA` menolak nomor tanpa awalan, sama dengan Menu 04): baris itu galat `phone_number`; templat membuat kolom telepon bertipe teks supaya tidak terjadi |
| `.xlsx` adalah arsip zip: unggahan ≈ 1 MiB yang mengklaim 300 MiB XML diekstrak ke direktori sementara tanpa batas wajar (bawaan `excelize`: 16 GB; dijalankan) | `Options{UnzipSizeLimit: 64 MiB, UnzipXMLSizeLimit: 16 MiB}` (ditolak seketika; templat 2.000 baris × 8 kolom hanya ±82 KiB terkompresi), `defer f.Close()` menghapus berkas sementara, dan pembacaan berhenti setelah 2.001 baris berisi |
| Judul kolom bervariasi ("Nama Peserta" pada templat seminar, huruf besar, spasi ganda, `*`) | judul dicocokkan tanpa membedakan huruf setelah spasi dilipat dan `*` dibuang; alias di kode; hanya kolom nama yang wajib; kolom tak dikenal diabaikan; judul ganda: yang pertama |
| Baris kosong di tengah berkas | dilewati, nomor baris berkas tetap benar (baris judul = 1) |

```go
// backend/internal/modules/plannerguest/domain/importer.go
package domain

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
)

// Batas impor (usulan; BE 3.4).
const (
	MaxImportRows      = 2000
	MaxImportBytes     = 5 << 20
	MaxGuestsInProject = 5000
)

// FileError menjadi 400 invalid_file; Reason muncul sebagai "file: <Reason>".
type FileError struct{ Reason string }

func (e *FileError) Error() string { return e.Reason }

// Record adalah satu baris berkas: nomor baris (baris judul = 1) dan isi selnya sebagai teks.
type Record struct {
	Row   int
	Cells []string
}

// DecodeCSV membaca CSV buatan Excel apa adanya: UTF-8 dengan/tanpa BOM atau Windows-1252, pemisah koma,
// titik koma (Excel berbahasa Indonesia), atau tab. Baris kosong dilewati; nomor baris adalah nomor baris
// tempat rekaman dimulai.
func DecodeCSV(data []byte) ([]Record, error) {
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, &FileError{"not a text file"}
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		dec, err := charmap.Windows1252.NewDecoder().Bytes(data)
		if err != nil {
			return nil, &FileError{"not valid UTF-8 or Windows-1252"}
		}
		data = dec
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	var out []Record
	for {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &FileError{"malformed CSV: " + err.Error()}
		}
		line, _ := r.FieldPos(0)
		out = append(out, Record{Row: line, Cells: cells})
	}
	return out, nil
}

// detectDelimiter menghitung pemisah di luar tanda kutip pada baris pertama yang tidak kosong; seri -> koma.
func detectDelimiter(data []byte) rune {
	count := map[rune]int{}
	inQuote := false
	for _, b := range data {
		if b == '"' {
			inQuote = !inQuote
		}
		if !inQuote {
			switch b {
			case ',', ';', '\t':
				count[rune(b)]++
			case '\n':
				if count[',']+count[';']+count['\t'] > 0 {
					goto done
				}
			}
		}
	}
done:
	best := ','
	for _, d := range []rune{';', '\t'} {
		if count[d] > count[best] {
			best = d
		}
	}
	return best
}

// ---------- Kolom ----------

type Column int

const (
	ColName Column = iota
	ColSide
	ColCategory
	ColSession
	ColTable
	ColPax
	ColPhone
	ColNotes
)

// Judul templat: Nama Tamu | Pihak | Kategori | Sesi | No Meja | Pax | No WhatsApp | Catatan.
// Jenis seminar memakai "Nama Peserta"; keduanya dikenali di semua jenis.
var columnAliases = map[Column][]string{
	ColName:     {"nama tamu", "nama peserta", "nama"},
	ColSide:     {"pihak", "sisi"},
	ColCategory: {"kategori"},
	ColSession:  {"sesi"},
	ColTable:    {"no meja", "nomor meja", "meja"},
	ColPax:      {"pax", "jumlah pax", "jumlah"},
	ColPhone:    {"no whatsapp", "no wa", "whatsapp", "no hp", "no telepon", "telepon"},
	ColNotes:    {"catatan", "keterangan"},
}

func normHeader(s string) string {
	return strings.ToLower(squash(strings.TrimSpace(strings.TrimRight(strings.TrimSpace(s), "*"))))
}

// MapColumns mencari kolom menurut judul (tanpa membedakan huruf, spasi dilipat, tanda * dibuang). Hanya kolom
// nama yang wajib; kolom lain yang tidak ada dianggap kosong; kolom tak dikenal diabaikan; judul ganda: yang pertama.
func MapColumns(header []string) (map[Column]int, error) {
	m := map[Column]int{}
	for i, h := range header {
		n := normHeader(h)
		for col, aliases := range columnAliases {
			for _, a := range aliases {
				if n == a {
					if _, dup := m[col]; !dup {
						m[col] = i
					}
				}
			}
		}
	}
	if _, ok := m[ColName]; !ok {
		return nil, &FileError{"missing column: Nama Tamu"}
	}
	return m, nil
}

// ---------- Baris ----------

type SessionRef struct{ ScopeKey, Name string }

type ExistingGuest struct {
	ID   uuid.UUID
	Name string
	WA   string // formatted_wa_number, "" bila kosong
}

type ImportOptions struct{ SkipInvalid, SkipDuplicates bool }

type ImportConfig struct {
	Spec     GuestSpec
	Sessions []SessionRef // termasuk sesi sistem all ("Umum / Semua Sesi")
	Existing []ExistingGuest
	Options  ImportOptions
}

// PlannedGuest adalah baris valid yang siap ditulis; Token dibuat usecase saat menulis.
type PlannedGuest struct {
	Row       int
	Name      string
	Side      *string
	Category  string
	Scope     string
	Table     *string
	Pax       int
	Phone     *string
	WA        *string
	Notes     *string
	Duplicate bool
}

type ImportPlan struct {
	Result  ImportResult
	Insert  []PlannedGuest
	Blocked bool // ada baris galat dan SkipInvalid=false: tidak ada yang boleh ditulis
}

const (
	reasonPhone = "must be an Indonesian phone number (08…, +62…)"
)

func atMost(n int) string { return "must be at most " + strconv.Itoa(n) + " characters" }

func hasControl(s string, allowNewline bool) bool {
	for _, r := range s {
		if unicode.IsControl(r) && !(allowNewline && (r == '\n' || r == '\t' || r == '\r')) {
			return true
		}
	}
	return false
}

func cell(rec Record, cols map[Column]int, c Column) string {
	if i, ok := cols[c]; ok && i < len(rec.Cells) {
		return rec.Cells[i]
	}
	return ""
}

func blank(rec Record) bool {
	for _, c := range rec.Cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BuildImportPlan memeriksa seluruh berkas (recs[0] = baris judul) dan menyusun hasil pratinjau serta baris yang
// akan ditulis. Tidak menulis apa pun: dry_run dan impor sungguhan memakai fungsi yang sama.
func BuildImportPlan(cfg ImportConfig, recs []Record) (ImportPlan, error) {
	plan := ImportPlan{Result: ImportResult{Errors: []ImportRowError{}, Duplicates: []ImportDuplicate{}, Preview: []ImportPreviewRow{}}}
	if len(recs) == 0 {
		return plan, &FileError{"empty file"}
	}
	cols, err := MapColumns(recs[0].Cells)
	if err != nil {
		return plan, err
	}
	var rows []Record
	for _, r := range recs[1:] {
		if !blank(r) {
			rows = append(rows, r)
		}
	}
	if len(rows) > MaxImportRows {
		return plan, &FileError{"too many rows (max " + strconv.Itoa(MaxImportRows) + ")"}
	}
	plan.Result.TotalRows = len(rows)

	scopeBy := map[string]string{}
	for _, s := range cfg.Sessions {
		scopeBy[strings.ToLower(squash(s.Name))] = s.ScopeKey
		scopeBy[strings.ToLower(s.ScopeKey)] = s.ScopeKey
	}
	scopeBy["umum"], scopeBy["semua"] = "all", "all"

	existing := map[string]uuid.UUID{}
	for _, g := range cfg.Existing {
		k := dedupeKey(g.Name, g.WA)
		if _, ok := existing[k]; !ok {
			existing[k] = g.ID
		}
	}
	firstRow := map[string]int{}

	for _, rec := range rows {
		var errs []ImportRowError
		bad := func(field, reason string) {
			errs = append(errs, ImportRowError{Row: rec.Row, Field: field, Reason: reason})
		}
		g := PlannedGuest{Row: rec.Row, Category: FallbackCategory, Scope: "all", Pax: 1}

		g.Name = squash(cell(rec, cols, ColName))
		switch {
		case g.Name == "":
			bad("name", "required")
		case utf8.RuneCountInString(g.Name) > 255:
			bad("name", atMost(255))
		case hasControl(g.Name, false):
			bad("name", "must not contain control characters")
		}

		if v := cell(rec, cols, ColSide); squash(v) != "" {
			if len(cfg.Spec.Sides) == 0 {
				bad("guest_side", "must be empty for an event type without sides")
			} else if k, ok := cfg.Spec.ResolveSide(v); ok {
				g.Side = ptr(k)
			} else {
				bad("guest_side", "must be a registry side key")
			}
		}

		g.Category = cfg.Spec.ResolveCategory(cell(rec, cols, ColCategory))
		if utf8.RuneCountInString(g.Category) > 60 {
			bad("category", atMost(60))
		} else if hasControl(g.Category, false) {
			bad("category", "must not contain control characters")
		}

		if v := strings.ToLower(squash(cell(rec, cols, ColSession))); v != "" {
			if k, ok := scopeBy[v]; ok {
				g.Scope = k
			} else {
				bad("sub_event_scope", "must be a session of this project")
			}
		}

		if v := squash(cell(rec, cols, ColTable)); v != "" {
			if utf8.RuneCountInString(v) > 30 {
				bad("table_number", atMost(30))
			} else if hasControl(v, false) {
				bad("table_number", "must not contain control characters")
			}
			g.Table = ptr(v)
		}

		if v := strings.TrimSpace(cell(rec, cols, ColPax)); v != "" {
			n, err := strconv.Atoi(v)
			switch {
			case err != nil:
				bad("pax_count", "must be an integer")
			case n < 1 || n > 20:
				bad("pax_count", "must be between 1 and 20")
			default:
				g.Pax = n
			}
		}

		if v := strings.TrimSpace(cell(rec, cols, ColPhone)); v != "" {
			if utf8.RuneCountInString(v) > 40 {
				bad("phone_number", atMost(40))
			} else if wa, err := NormalizeWA(v); err != nil {
				bad("phone_number", reasonPhone)
			} else {
				g.Phone = ptr(v)
				g.WA = ptr(wa)
			}
		}

		if v := strings.TrimSpace(cell(rec, cols, ColNotes)); v != "" {
			if utf8.RuneCountInString(v) > 2000 {
				bad("notes", atMost(2000))
			} else if hasControl(v, true) {
				bad("notes", "must not contain control characters")
			}
			g.Notes = ptr(v)
		}

		if len(errs) > 0 {
			plan.Result.InvalidRows++
			plan.Result.Errors = append(plan.Result.Errors, errs...)
			continue
		}

		wa := ""
		if g.WA != nil {
			wa = *g.WA
		}
		k := dedupeKey(g.Name, wa)
		if id, ok := existing[k]; ok {
			g.Duplicate = true
			plan.Result.Duplicates = append(plan.Result.Duplicates, ImportDuplicate{Row: rec.Row, Name: g.Name, OfGuest: &id})
		} else if first, ok := firstRow[k]; ok {
			g.Duplicate = true
			f := first
			plan.Result.Duplicates = append(plan.Result.Duplicates, ImportDuplicate{Row: rec.Row, Name: g.Name, OfRow: &f})
		} else {
			firstRow[k] = rec.Row
		}

		plan.Result.ValidRows++
		if len(plan.Result.Preview) < 5 {
			plan.Result.Preview = append(plan.Result.Preview, ImportPreviewRow{Row: g.Row, Name: g.Name, GuestSide: g.Side, Category: g.Category, SubEventScope: g.Scope, TableNumber: g.Table, PaxCount: g.Pax, FormattedWANumber: g.WA})
		}
		if g.Duplicate && cfg.Options.SkipDuplicates {
			plan.Result.Skipped++
			continue
		}
		plan.Insert = append(plan.Insert, g)
	}

	if plan.Result.InvalidRows > 0 {
		if cfg.Options.SkipInvalid {
			plan.Result.Skipped += plan.Result.InvalidRows
		} else {
			plan.Blocked = true
			plan.Insert = nil
		}
	}
	return plan, nil
}

// dedupeKey: nama tanpa membedakan huruf dan spasi ganda, ditambah nomor WhatsApp ternormalisasi (kosong bila tak ada).
// Dua tamu bernama sama tanpa nomor dianggap duplikat; yang bernomor berbeda tidak.
func dedupeKey(name, wa string) string { return strings.ToLower(squash(name)) + "|" + wa }
```

```go
// backend/internal/modules/plannerguest/delivery/http/xlsx/xlsx.go
package xlsx

import (
	"bytes"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// Batas ekstraksi: berkas .xlsx adalah arsip zip, jadi 5 MB terkompresi dapat mengklaim gigabita XML.
// Bawaan excelize 16 GB (diuji: 1 MiB terkompresi yang mengklaim 300 MiB diekstrak ke direktori sementara).
// Templat 2.000 baris x 8 kolom hanya sekitar 1 MB tanpa kompresi.
const (
	unzipSizeLimit    = 64 << 20
	unzipXMLSizeLimit = 16 << 20
)

// Read mengambil lembar pertama sebagai rekaman teks. Baris kosong dilewati (nomor barisnya tetap benar) dan
// pembacaan berhenti setelah MaxImportRows+1 baris berisi (judul + data), supaya lembar sangat panjang tidak
// dibaca seluruhnya; BuildImportPlan lalu menolak berkas dengan terlalu banyak baris.
func Read(data []byte) ([]domain.Record, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: unzipSizeLimit, UnzipXMLSizeLimit: unzipXMLSizeLimit})
	if err != nil {
		return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
	}
	defer f.Close() // menghapus berkas sementara hasil ekstraksi

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, &domain.FileError{Reason: "the workbook has no sheet"}
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
	}
	defer rows.Close()

	var out []domain.Record
	n := 0
	for rows.Next() {
		n++
		cells, err := rows.Columns()
		if err != nil {
			return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
		}
		blank := true
		for _, c := range cells {
			if len(bytes.TrimSpace([]byte(c))) > 0 {
				blank = false
				break
			}
		}
		if blank {
			continue
		}
		out = append(out, domain.Record{Row: n, Cells: cells})
		if len(out) > domain.MaxImportRows+1 {
			break
		}
	}
	return out, nil
}
```

- **Kolom templat:** `Nama Tamu | Pihak | Kategori | Sesi | No Meja | Pax | No WhatsApp | Catatan` (seminar: `Nama Peserta`; jenis tanpa sisi: tanpa `Pihak`). `Pihak` dan `Kategori` dipetakan ke kunci registry lewat kunci atau label (tanpa membedakan huruf; "Pihak Pria" cocok dengan label "Pihak Pria (CPP)"); `Pihak` yang tak dikenal menolak baris itu; `Kategori` yang tak dikenal menjadi kategori kustom; `Sesi` dipetakan lewat nama sesi atau `scope_key` (kosong, "Umum", dan "Semua" = `all`), tak dikenal menolak baris. `Pax` kosong = **1** (DDL bawaan 2 tidak dipakai).
- **Pratinjau (`dry_run=true`)** tidak menulis apa pun: hasilnya `ImportResult` ([bab 4](#4-dto-go)) berisi hitungan, galat per kolom per baris (alasan baku [bab 3.4](#34-validasi-dan-batas)), duplikat, dan lima baris valid pertama. **Duplikat** = nama sama (tanpa membedakan huruf dan spasi ganda) **dan** nomor WhatsApp ternormalisasi sama (tanpa nomor: nama saja); di dalam berkas (baris kedua dan seterusnya) atau terhadap tamu proyek. Duplikat hanya dilaporkan dan tetap diimpor kecuali `skip_duplicates=true`.
- **Impor sungguhan** (tanpa `dry_run`): satu transaksi di bawah kunci proyek; tamu yang ada dibaca ulang di dalam kunci. Ada baris galat dan tanpa `skip_invalid=true` → `400 invalid_input` (`rows: 3 rows are invalid`), tidak ada yang ditulis; dengan `skip_invalid` hanya baris valid ditulis (`skipped` menghitung yang dilewati). Total tamu setelah impor > 5.000 → `409 guest_limit_reached`, semuanya dibatalkan. Penulisan: `INSERT` multi-baris per 500 baris (`VALUES`, dijalankan), token dibuat per baris, `sort_order` melanjutkan urutan berkas. Berkas tak terbaca → `400 invalid_file` (`file: <alasan>`); lebih dari 2.000 baris berisi → `400 invalid_file` (`file: too many rows (max 2000)`); > 5 MiB → `413 file_too_large`. Batas **5 impor/menit/pengguna** ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi); `dry_run` ikut terhitung).
- **Templat:** `GET …/guests/import-template?format=xlsx` (endpoint baru; versi sebelumnya menjanjikan "unduh templat" tanpa endpoint). Lembar `Tamu` berisi judul tebal dan **tanpa baris contoh** (baris contoh akan ikut terimpor bila tidak dihapus); lembar `Petunjuk` berisi daftar pihak, kategori, dan sesi proyek serta penjelasan kolom. Dropdown mengarah ke lembar `Petunjuk` (rumus daftar langsung dibatasi 255 karakter oleh Excel; proyek dengan 20 sesi melampauinya), kategori tidak ketat (kategori baru diperbolehkan), `Pax` hanya bilangan bulat 1–20, kolom `No WhatsApp` dan `No Meja` bertipe teks. Terbukti: templat dibaca balik oleh pembaca impor (judul dikenali, nol baris data), validasi dropdown dan `Pax` ada, gaya kolom telepon `@`, templat seminar "Nama Peserta" dikenali (belum dibuka di Excel/Numbers).

```go
// backend/internal/modules/plannerguest/delivery/http/xlsx/template.go
package xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// TemplateInput adalah semua yang dibutuhkan templat satu proyek; semuanya dari registry dan master sesi.
type TemplateInput struct {
	GuestWord string          // Labels.Guest: "Tamu" atau "Peserta"
	Sides     []domain.Option // kosong = kolom Pihak tidak dibuat
	Cats      []domain.Option
	Sessions  []string // nama sesi; "Umum / Semua Sesi" selalu ada
}

const (
	sheetData  = "Tamu"
	sheetGuide = "Petunjuk"
)

// Template membuat berkas impor kosong: judul kolom persis yang dikenali BuildImportPlan, dropdown dari registry dan master
// sesi (daftar pada lembar Petunjuk karena rumus daftar langsung dibatasi 255 karakter), kolom telepon bertipe teks (Excel
// tidak membuang 0 di depan), tanpa baris contoh (baris contoh akan ikut terimpor bila tidak dihapus).
func Template(in TemplateInput) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", sheetData); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(sheetGuide); err != nil {
		return nil, err
	}

	headers := []string{"Nama " + in.GuestWord}
	if len(in.Sides) > 0 {
		headers = append(headers, "Pihak")
	}
	headers = append(headers, "Kategori", "Sesi", "No Meja", "Pax", "No WhatsApp", "Catatan")
	col := map[string]string{}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellStr(sheetData, cell, h); err != nil {
			return nil, err
		}
		col[h], _ = excelize.ColumnNumberToName(i + 1)
	}
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	last, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(sheetData, "A1", last+"1", bold)

	text, _ := f.NewStyle(&excelize.Style{NumFmt: 49}) // "@": teks
	for _, h := range []string{"No WhatsApp", "No Meja"} {
		if err := f.SetColStyle(sheetData, col[h], text); err != nil {
			return nil, err
		}
	}
	for i := range headers {
		w := 18.0
		if i == 0 {
			w = 34
		}
		c, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetData, c, c, w)
	}

	// Lembar Petunjuk: daftar nilai di kolom A (pihak), B (kategori), C (sesi), lalu penjelasan kolom.
	_ = f.SetSheetRow(sheetGuide, "A1", &[]any{"Pihak", "Kategori", "Sesi"})
	_ = f.SetCellStyle(sheetGuide, "A1", "C1", bold)
	for i, o := range in.Sides {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetCellStr(sheetGuide, cell, o.Label)
	}
	for i, o := range in.Cats {
		cell, _ := excelize.CoordinatesToCellName(2, i+2)
		_ = f.SetCellStr(sheetGuide, cell, o.Label)
	}
	for i, s := range in.Sessions {
		cell, _ := excelize.CoordinatesToCellName(3, i+2)
		_ = f.SetCellStr(sheetGuide, cell, s)
	}
	notes := []string{
		"Isi satu baris per " + lower(in.GuestWord) + " pada lembar " + sheetData + "; hanya kolom pertama yang wajib.",
		"Pax: 1-20 (kosong = 1). No WhatsApp: 08..., +62..., atau 62...; kolom ini bertipe teks supaya 0 di depan tidak hilang.",
		"Kategori boleh memakai nama sendiri (kategori baru). Pihak dan Sesi harus salah satu nilai di kolom ini; Sesi kosong = Umum.",
		"Hapus baris yang tidak dipakai; baris kosong dilewati. Maksimal 2.000 baris dan 5 MB.",
	}
	for i, n := range notes {
		cell, _ := excelize.CoordinatesToCellName(5, i+1)
		_ = f.SetCellStr(sheetGuide, cell, n)
	}

	// Dropdown: kolom data -> daftar pada lembar Petunjuk. Kategori tidak ketat (kategori baru diperbolehkan).
	dropdown := func(header, listCol string, n int, strict bool) error {
		if n == 0 {
			return nil
		}
		c := col[header]
		v := excelize.NewDataValidation(true)
		v.SetSqref(fmt.Sprintf("%s2:%s%d", c, c, domain.MaxImportRows+1))
		v.SetSqrefDropList(fmt.Sprintf("%s!$%s$2:$%s$%d", sheetGuide, listCol, listCol, n+1))
		v.ShowErrorMessage = strict
		return f.AddDataValidation(sheetData, v)
	}
	for _, d := range []struct {
		header, list string
		n            int
		strict       bool
	}{{"Pihak", "A", len(in.Sides), true}, {"Kategori", "B", len(in.Cats), false}, {"Sesi", "C", len(in.Sessions), true}} {
		if err := dropdown(d.header, d.list, d.n, d.strict); err != nil {
			return nil, err
		}
	}
	pax := excelize.NewDataValidation(true)
	paxCol := col["Pax"]
	pax.SetSqref(fmt.Sprintf("%s2:%s%d", paxCol, paxCol, domain.MaxImportRows+1))
	_ = pax.SetRange(1, 20, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween)
	pax.SetError(excelize.DataValidationErrorStyleStop, "Pax tidak valid", "Isi bilangan bulat 1 sampai 20.")
	if err := f.AddDataValidation(sheetData, pax); err != nil {
		return nil, err
	}

	f.SetActiveSheet(0)
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func lower(s string) string {
	if s == "Tamu" {
		return "tamu"
	}
	return "peserta"
}
```

#### Ekspor

`GET …/guests/export?format=xlsx` (daftar tamu) dan `GET …/guests/export-attendance?format=xlsx` (rekap kehadiran final). **Hanya `.xlsx`**: versi sebelumnya menjanjikan PDF untuk keduanya, tetapi satu pustaka PDF belum disetujui ([kontrak bab 1.4](../00_KONTRAK_BERSAMA/README.md#14-kontrak-api-dan-dokumentasi)); Menu 02–04 juga hanya `.xlsx`. Aturan bersama (`excelize`, `SetCellStr`, awalan `'` pada sel berawalan `=`/`+`/`-`/`@`, nama berkas dari slug ASCII, `Cache-Control: no-store`): [kontrak bab 2](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan). Satu pembatas **10/menit/pengguna** untuk ketiga unduhan (daftar, rekap, templat), dihitung sebelum handler.

| Berkas | Lembar dan kolom |
|---|---|
| `tamu-<slug>-<tanggal>.xlsx` | `Daftar Tamu`: judul proyek, tanggal ekspor; kolom No · Nama · Pihak · Kategori · Sesi · No Meja · Pax · No WhatsApp · Status Undangan · RSVP · Pax Hadir (RSVP) · Kode Tiket · Catatan; baris TOTAL (jumlah tamu dan pax). Pax angka; telepon dan kode tiket teks. Kode Tiket memungkinkan pembuatan QR dengan alat luar ([bab 2.5](#25-meja-resepsionis-hari-h)); berkas ini hanya untuk pemilik proyek |
| `kehadiran-<slug>-<tanggal>.xlsx` | `Kehadiran`: No · Nama · Pihak · Kategori · Sesi · No Meja · Pax Undangan · Pax Hadir · Jam Masuk (`HH.MM` zona proyek) · Souvenir · RSVP; hanya tamu yang diharapkan atau hadir; baris TOTAL (tamu hadir, pax hadir, souvenir diserahkan) |
| `templat-tamu-<slug>-<tanggal>.xlsx` | lihat Templat di atas |

Tata letak belum dibuka di Excel/Numbers (kode pembentuk templat dan pembaca dijalankan dengan `excelize`).

### 2.7 Pencarian, filter, dan urutan

`GET …/guests?side=&category=&rsvp=&sent=&scope=&search=&limit=&offset=` mengembalikan `GuestList`: halaman tamu dan `overview` seluruh proyek ([bab 2.8](#28-ringkasan-penghitung-kategori-dan-pihak)). Semua kondisi memakai parameter (tidak ada teks yang disambung); dijalankan apa adanya:

```sql
-- $1 project_id · $2 side ('none' = belum ditentukan) · $3 category · $4 rsvp · $5 scope · $6 sent ('sent'|'unsent') · $7 search · $8 limit · $9 offset
SELECT id, name, guest_side, category, sub_event_scope, table_number, pax_count, phone_number, formatted_wa_number,
       invitation_sent_status, invitation_sent_at, rsvp_status, pax_actual, rsvp_submitted_at, qr_code_token,
       is_checked_in, checked_in_at, souvenir_received, notes
FROM planner_guests
WHERE project_id = $1
  AND ($2::text IS NULL OR ($2 = 'none' AND guest_side IS NULL) OR guest_side = $2)
  AND ($3::text IS NULL OR category = $3)
  AND ($4::text IS NULL OR rsvp_status = $4)
  AND ($5::text IS NULL OR sub_event_scope = $5)
  AND ($6::text IS NULL OR (CASE WHEN $6 = 'sent' THEN invitation_sent_status <> 'unsent' ELSE invitation_sent_status = 'unsent' END))
  AND ($7::text IS NULL OR strpos(lower(name), lower($7)) > 0 OR strpos(lower(COALESCE(table_number, '')), lower($7)) > 0
       OR strpos(COALESCE(phone_number, ''), $7) > 0 OR strpos(COALESCE(formatted_wa_number, ''), $7) > 0)
ORDER BY sort_order, created_at, id
LIMIT $8 OFFSET $9;
-- total yang cocok (untuk halaman): pernyataan yang sama tanpa ORDER BY/LIMIT, SELECT count(*)
```

- **Pencarian memakai `strpos`, bukan `LIKE`/`ILIKE`:** `search=%` mencocokkan 0 baris, sedangkan `ILIKE '%'||'%'||'%'` mencocokkan semua 20 (dijalankan), dan tidak perlu meloloskan `%`, `_`, `\`. Mencari nama, nomor meja, nomor apa adanya, atau bentuk `62…` (dijalankan). Pemindaian satu proyek (≤ 5.000 baris); indeks `lower(name)` tidak membantu pencarian substring ([bab 10 butir 22](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- Urutan tetap `sort_order, created_at, id` (urutan dibuat; impor melanjutkan urutan berkas); tidak ada urut ulang manual. `limit` 1–500 (bawaan 100), `offset` ≥ 0.
- `side=none` memilih tamu tanpa pihak; nilai `none` bukan kunci registry sah.

### 2.8 Ringkasan, penghitung kategori, dan pihak

`GET …/guests/summary` mengembalikan hanya `GuestOverview` (ringan, untuk polling 30 detik); `GET …/guests` menyertakan yang sama. **Selalu seluruh proyek**, tidak terpengaruh filter, pencarian, atau halaman. Dua kueri:

```sql
-- $1 = project_id
SELECT count(*) AS total, COALESCE(sum(pax_count), 0) AS total_pax,
       count(*) FILTER (WHERE invitation_sent_status <> 'unsent') AS sent,
       count(*) FILTER (WHERE rsvp_status = 'attending')   AS attending,
       count(*) FILTER (WHERE rsvp_status = 'declined')    AS declined,
       count(*) FILTER (WHERE rsvp_status = 'maybe')       AS maybe,
       count(*) FILTER (WHERE rsvp_status = 'unconfirmed') AS unconfirmed,
       COALESCE(sum(CASE WHEN rsvp_status = 'attending' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0) AS attending_pax
FROM planner_guests WHERE project_id = $1;

SELECT guest_side, category, count(*) AS invitations, sum(pax_count)::bigint AS pax
FROM planner_guests WHERE project_id = $1 GROUP BY guest_side, category;
```

Kueri pertama sama dengan kueri provider `guests` Menu 01 ([01 BE §5.2](../01_DASHBOARD/BE.md#52-query-per-provider)); satu `AssembleOverview` melayani provider, `summary`, dan daftar, sehingga angka dashboard dan halaman tidak dapat berbeda. Kode berikut **sudah dijalankan** terhadap keluaran kedua kueri di atas pada 20 tamu contoh (nol selisih dengan angka SQL) dan terhadap sembilan jenis acara:

```go
// backend/internal/modules/plannerguest/domain/spec.go
package domain

import (
	"strings"
	"unicode"
)

// FallbackCategory dipakai bila kategori tidak diisi. Ia bukan kategori registry jenis acara mana pun
// (TEMPLATE_EVENT 4.2); penghitung hanya menampilkannya bila ada tamu yang memakainya.
const (
	FallbackCategory      = "lainnya"
	FallbackCategoryLabel = "Lainnya"
	SideNone              = "none" // nilai filter ?side= untuk tamu yang pihaknya belum ditentukan
)

// GuestSpec adalah bagian registry jenis acara yang dibaca Menu 05 (EventTypeSpec.GuestSides/GuestCategories).
type GuestSpec struct {
	Sides      []Option
	Categories []Option
}

// squash memangkas dan melipat spasi (termasuk tab dan baris baru) menjadi satu spasi.
func squash(s string) string { return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ") }

// labelCore membuang keterangan di ujung label dalam kurung: "Pihak Pria (CPP)" -> "Pihak Pria".
func labelCore(label string) string {
	if i := strings.LastIndex(label, " ("); i > 0 && strings.HasSuffix(label, ")") {
		return label[:i]
	}
	return label
}

func (s GuestSpec) match(opts []Option, input string) (string, bool) {
	in := squash(input)
	if in == "" {
		return "", false
	}
	for _, o := range opts {
		if strings.EqualFold(in, o.Key) || strings.EqualFold(in, o.Label) || strings.EqualFold(in, labelCore(o.Label)) {
			return o.Key, true
		}
	}
	return "", false
}

// ResolveSide memetakan isian (kunci atau label, tanpa membedakan huruf) ke kunci sisi registry.
// Kosong -> ("", true): pihak belum ditentukan. Tidak dikenal -> ("", false). Jenis tanpa sisi menolak isian apa pun.
func (s GuestSpec) ResolveSide(input string) (string, bool) {
	if squash(input) == "" {
		return "", true
	}
	return s.match(s.Sides, input)
}

// ResolveCategory memetakan isian ke kunci registry; teks lain menjadi kategori kustom (spasi dilipat, huruf
// dipertahankan); kosong menjadi FallbackCategory. Panjang (maks 60) diperiksa pemanggil.
func (s GuestSpec) ResolveCategory(input string) string {
	in := squash(input)
	if in == "" {
		return FallbackCategory
	}
	if k, ok := s.match(s.Categories, in); ok {
		return k
	}
	if strings.EqualFold(in, FallbackCategory) || strings.EqualFold(in, FallbackCategoryLabel) {
		return FallbackCategory
	}
	return in
}

// CategoryLabel: label registry, "Lainnya" untuk kunci cadangan, selain itu teks kustom apa adanya.
func (s GuestSpec) CategoryLabel(key string) string {
	for _, o := range s.Categories {
		if o.Key == key {
			return o.Label
		}
	}
	if key == FallbackCategory {
		return FallbackCategoryLabel
	}
	return key
}

func (s GuestSpec) sideLabel(key string) string {
	for _, o := range s.Sides {
		if o.Key == key {
			return o.Label
		}
	}
	return key
}

func (s GuestSpec) registryCategory(key string) bool {
	for _, o := range s.Categories {
		if o.Key == key {
			return true
		}
	}
	return false
}
```

```go
// backend/internal/modules/plannerguest/domain/overview.go
package domain

import (
	"sort"
	"strings"
)

// Cell adalah satu baris hasil `GROUP BY guest_side, category` (bab 2.8); Side nil = belum ditentukan.
type Cell struct {
	Side        *string
	Category    string
	Invitations int
	Pax         int
}

// Aggregates adalah seluruh masukan AssembleOverview: satu baris ringkasan dan sel pihak x kategori.
type Aggregates struct {
	Total, TotalPax                                       int
	Sent                                                  int // sent + delivered
	Attending, Declined, Maybe, Unconfirmed, AttendingPax int
	Cells                                                 []Cell
}

// categoryOrder memberi urutan tetap: kategori registry menurut urutan registry, lalu "lainnya", lalu kategori
// kustom menurut abjad (tanpa membedakan huruf).
func (s GuestSpec) categoryOrder(used map[string]bool) []string {
	var keys []string
	seen := map[string]bool{}
	for _, o := range s.Categories {
		keys = append(keys, o.Key)
		seen[o.Key] = true
	}
	if used[FallbackCategory] && !seen[FallbackCategory] {
		keys = append(keys, FallbackCategory)
		seen[FallbackCategory] = true
	}
	var custom []string
	for k := range used {
		if !seen[k] {
			custom = append(custom, k)
		}
	}
	sort.Slice(custom, func(i, j int) bool {
		a, b := strings.ToLower(custom[i]), strings.ToLower(custom[j])
		if a != b {
			return a < b
		}
		return custom[i] < custom[j]
	})
	return append(keys, custom...)
}

// AssembleOverview menyusun GuestOverview dari hasil SQL: menambah entri registry yang bernilai nol, label,
// urutan, dan pihak "belum ditentukan". Jenis tanpa sisi tidak pernah menghasilkan `sides` atau `side_categories`.
func (s GuestSpec) AssembleOverview(a Aggregates) GuestOverview {
	o := GuestOverview{
		Total: a.Total, TotalPax: a.TotalPax,
		Sides: []GuestSide{}, Categories: []CategoryCount{}, SideCategories: []SideCategoryCount{},
		Invitation: InvitationCounts{Sent: a.Sent, Unsent: a.Total - a.Sent},
		RSVP:       RSVPSummary{Attending: a.Attending, Declined: a.Declined, Maybe: a.Maybe, Unconfirmed: a.Unconfirmed, AttendingPax: a.AttendingPax},
	}

	used := map[string]bool{}
	byCat := map[string]*CategoryCount{}
	for _, c := range a.Cells {
		used[c.Category] = true
		cc := byCat[c.Category]
		if cc == nil {
			cc = &CategoryCount{Key: c.Category}
			byCat[c.Category] = cc
		}
		cc.Invitations += c.Invitations
		cc.Pax += c.Pax
	}
	order := s.categoryOrder(used)
	rank := map[string]int{}
	for i, k := range order {
		rank[k] = i
		cc := byCat[k]
		if cc == nil {
			cc = &CategoryCount{Key: k}
		}
		cc.Label = s.CategoryLabel(k)
		cc.Custom = !s.registryCategory(k)
		o.Categories = append(o.Categories, *cc)
	}

	if len(s.Sides) == 0 {
		return o
	}

	// Pihak: semua sisi registry (juga yang 0), lalu sisi tak dikenal bila ada datanya, lalu "belum ditentukan".
	type tot struct{ inv, pax int }
	known := map[string]*tot{}
	var none tot
	for _, c := range a.Cells {
		if c.Side == nil {
			none.inv += c.Invitations
			none.pax += c.Pax
			continue
		}
		t := known[*c.Side]
		if t == nil {
			t = &tot{}
			known[*c.Side] = t
		}
		t.inv += c.Invitations
		t.pax += c.Pax
	}
	sideRank := map[string]int{}
	for i, sd := range s.Sides {
		k := sd.Key
		sideRank[k] = i
		t := known[k]
		if t == nil {
			t = &tot{}
		}
		o.Sides = append(o.Sides, GuestSide{Key: &k, Label: sd.Label, Invitations: t.inv, Pax: t.pax})
		delete(known, k)
	}
	var extra []string
	for k := range known {
		extra = append(extra, k)
	}
	sort.Strings(extra)
	for i, k := range extra {
		k := k
		sideRank[k] = len(s.Sides) + i
		o.Sides = append(o.Sides, GuestSide{Key: &k, Label: k, Invitations: known[k].inv, Pax: known[k].pax})
	}
	if none.inv > 0 {
		o.Sides = append(o.Sides, GuestSide{Label: "Belum ditentukan", Invitations: none.inv, Pax: none.pax})
	}

	// Sel pihak x kategori: hanya yang terisi, urut pihak lalu kategori (urutan yang sama dengan di atas).
	sideIdx := func(p *string) int {
		if p == nil {
			return 1 << 30
		}
		return sideRank[*p]
	}
	for _, c := range a.Cells {
		if c.Invitations > 0 {
			o.SideCategories = append(o.SideCategories, SideCategoryCount{Side: c.Side, Category: c.Category, Invitations: c.Invitations, Pax: c.Pax})
		}
	}
	sort.Slice(o.SideCategories, func(i, j int) bool {
		x, y := o.SideCategories[i], o.SideCategories[j]
		if sideIdx(x.Side) != sideIdx(y.Side) {
			return sideIdx(x.Side) < sideIdx(y.Side)
		}
		return rank[x.Category] < rank[y.Category]
	})
	return o
}
```

- **Sisi dan kategori dari registry** ([TEMPLATE_EVENT 4 dan 4.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori)): `sides` kosong untuk jenis tanpa sisi (ulang tahun, wisuda, lainnya), sehingga seminar tidak pernah menampilkan "Pria/Wanita" (diuji). `categories` memuat semua kategori registry (juga yang 0), lalu `lainnya` dan kategori kustom hanya bila terpakai (`custom: true`), kustom menurut abjad. Jumlah `sides[].invitations`, `categories[].invitations`, dan empat status RSVP masing-masing sama dengan `total` (diuji).
- **`side_categories` mengembalikan fidelitas PDF Hal. 13:** PDF punya enam penghitung **di setiap tabel pihak**; versi sebelumnya hanya satu set gabungan sehingga memilih "Pihak Pria" tetap menampilkan hitungan gabungan. Sel hanya yang terisi; FE menjumlahkan sel pihak terpilih ([FE bab 5](FE.md#5-library-libplannerguestts), `categoryPills`).
- Tipe `GuestSide` dan `RSVPSummary` adalah milik Menu 01 (`planner/domain/summary.go`, sudah ada; `GuestSide.Key` boleh `null` = belum ditentukan, satu-satunya `null` di menu ini).

### 2.9 Konkurensi dan batas

- **Satu pernyataan, satu baris:** check-in, batal, RSVP, status kirim, souvenir, ubah, dan hapus adalah satu `UPDATE`/`DELETE` dengan `WHERE id = $1 AND project_id = $2`; kunci baris cukup (check-in ganda terbukti).
- **Kunci proyek** (`SELECT id FROM planner_projects WHERE id = $1 FOR UPDATE`, dijalankan) hanya untuk yang menambah baris: tambah, walk-in, dan impor, supaya batas jumlah dan `sort_order` tidak berbalapan. Tanpa kunci, dua tambah serentak pada 4.999 tamu sama-sama dapat lolos (tidak dijalankan untuk modul ini; pola dan buktinya sama dengan batas 300 pos di [04 BE bab 9 butir 28](../04_VENDOR_DAN_BELANJA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Batas jumlah (usulan):** 5.000 tamu per proyek → `409 guest_limit_reached` (kode baru, [kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#kode-error-planner)); 2.000 baris dan 5 MiB per impor. Angka 5.000 mencakup pernikahan besar dan seminar ([bab 10 butir 1](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Frekuensi:** `scan-qr` 120/menit, impor 5/menit, unduhan 10/menit, semuanya per pengguna (pola `ratelimit.go`, salinan keenam; ekstraksi dicatat di [02 BE bab 8 butir 13](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
- **Data pribadi** ([kontrak bab 2 butir 7](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)): nama dan telepon hanya terbaca oleh pemilik proyek; tidak ada endpoint publik. Tamu `project_id IS NULL` tidak pernah dibaca atau ditulis Planner.

---

## 3. Endpoint

`/v1/planner/projects/:id/...`, `BearerAuth` + `ProjectGuard` + `WriteGuard` (rantai middleware dan pemasangan di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)). Respons `{"data": ...}`, galat `{"error": {code, message}}`. Id anak di URL (`:guest_id`) harus UUID kanonik 36 karakter, selain itu `404 not_found`. Rute statis (`/guests/summary`, `/guests/export`, …) berdampingan dengan `/guests/:guest_id`; Gin menyelesaikan segmen statis lebih dulu (**dijalankan pada Gin v1.12.0 untuk ke-19 rute**: nol salah arah). Penulisan di proyek `archived` ditolak `409 project_read_only`; `GET` (termasuk ekspor dan templat) tetap boleh. Total **19 endpoint** (12 daftar + 7 meja resepsionis).

### 3.1 Daftar Tamu & Undangan

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/guests?side=&category=&rsvp=&sent=&scope=&search=&limit=&offset=` | `GuestList`: halaman tamu (bawaan 100, maks 500) **dan** `overview` seluruh proyek (bab 2.7–2.8). |
| `GET` | `/guests/summary` | Hanya `GuestOverview` (ringan; polling 30 dtk). |
| `GET` | `/guests/:guest_id` | `Guest`. Dipakai FE untuk mengambil ulang tamu sesudah `409 already_checked_in`. |
| `POST` | `/guests` | Tambah. `201 Guest`. `409 guest_limit_reached`. |
| `PUT` | `/guests/:guest_id` | Mengganti nama, pihak, kategori, sesi, meja, pax, telepon, catatan ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)): `guest_side`, `table_number`, `phone_number`, `notes` yang tidak dikirim dikosongkan. RSVP, status kirim, check-in, souvenir, token, dan urutan tidak pernah berubah. `Guest`. |
| `PATCH` | `/guests/:guest_id/sent-status` | `{"sent_status": "unsent"\|"sent"}`. `Guest`. |
| `PATCH` | `/guests/:guest_id/rsvp` | `{"rsvp_status", "pax_actual"}` (RSVP manual). `409 already_checked_in` bila sudah check-in. `Guest`. |
| `DELETE` | `/guests/:guest_id` | `204`. |
| `POST` | `/guests/bulk-import?dry_run=&skip_invalid=&skip_duplicates=` | JSON `{filename, content_base64}` (bab 2.6). `ImportResult`. |
| `GET` | `/guests/import-template?format=xlsx` | Berkas templat impor (bab 2.6). |
| `GET` | `/guests/export?format=xlsx` | Daftar tamu. |
| `GET` | `/guests/catering-projection?buffer_percent=&buffer_pax=` | `CateringProjection` (bab 2.3). |

### 3.2 Meja Resepsionis

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/guests/reception-overview?status=&category=&search=&limit=&offset=` | `ReceptionOverview`: KPI, jumlah filter, 10 check-in terbaru, dan satu halaman antrean (bawaan 50, maks 200); polling 5 dtk (bab 2.5). |
| `POST` | `/guests/scan-qr` | `ScanQRRequest` → `Guest` (juga bila sudah check-in); `404` untuk tiket tak dikenal. |
| `POST` | `/guests/:guest_id/checkin` | `CheckinRequest` → `Guest`; `409 already_checked_in`. |
| `DELETE` | `/guests/:guest_id/checkin` | Batalkan check-in (idempoten). `Guest`. |
| `PATCH` | `/guests/:guest_id/souvenir` | `{"souvenir_received": true}`. `Guest`. |
| `POST` | `/guests/walk-in` | `WalkInRequest` → `201 Guest`. `409 guest_limit_reached`. |
| `GET` | `/guests/export-attendance?format=xlsx` | Rekap kehadiran final dengan jam masuk. |

Tidak ada `seed-default`/template tamu: contoh tamu hanya ada di Proyek Contoh (bab 6), dan wizard tidak membuat tamu. Tidak ada `reorder`: daftar urut `sort_order` (bab 2.7).

### 3.3 Bentuk respons

| Endpoint | `data` |
|---|---|
| `GET /guests` | `GuestList` `{overview, guests[], total, limit, offset}` |
| `GET /guests/summary` | `GuestOverview` |
| `GET /guests/:id`, `POST /guests` (`201`), `PUT`, `PATCH …/sent-status`, `…/rsvp`, `…/souvenir`, `POST …/checkin`, `DELETE …/checkin`, `POST …/scan-qr`, `POST …/walk-in` (`201`) | `Guest` |
| `DELETE /guests/:id` | `204` |
| `POST …/bulk-import` | `ImportResult` |
| `GET …/catering-projection` | `CateringProjection` |
| `GET …/reception-overview` | `ReceptionOverview` `{kpi, filters, recent_checkins[], guests[], total, limit, offset}` |
| `GET …/export*`, `…/import-template` | berkas (bukan JSON): `Content-Disposition: attachment; filename="tamu-<slug>-<tanggal>.xlsx"` / `kehadiran-…` / `templat-tamu-…` |

Kunci opsional **dihilangkan** bila kosong (`omitempty`, tipe FE `?:`): `guest_side`, `table_number`, `phone_number`, `formatted_wa_number`, `personalized_url`, `invitation_sent_at`, `rsvp_submitted_at`, `checked_in_at`, `notes`, `souvenir_stock`, `custom`, `of_row`, `of_guest` (dan `guest_side`, `table_number`, `formatted_wa_number` pada baris pratinjau). Satu-satunya `null` adalah `GuestSide.key` (milik Menu 01) dan `side_categories[].side` (pihak belum ditentukan). Semua daftar (`guests`, `sides`, `categories`, `side_categories`, `recent_checkins`, `errors`, `duplicates`, `preview`) selalu array, bukan `null`. **Tidak ada persentase di respons mana pun** (versi sebelumnya punya `AttendancePercent`); klien menghitung dengan `percent` ([FE bab 5](FE.md#5-library-libplannerguestts)), selaras kontrak bab 6 (pembagi nol → `null` → "–"). Waktu (`checked_in_at`, dst.) RFC 3339 UTC; klien memformat jam di zona waktu proyek.

### 3.4 Validasi dan batas

Semua galat berbentuk `<field>: <alasan>` ([kontrak bab 6](../00_KONTRAK_BERSAMA/README.md#6-konvensi-data)); pesan memakai kosakata Menu 01–04 ditambah alasan di bawah. Angka batas adalah usulan ([bab 10 butir 1](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)) dan dijaga usecase. Teks dipangkas (`trim`) dan panjang dihitung dalam rune; nama, kategori, meja, dan telepon tanpa karakter kontrol (`notes` boleh baris baru); nama dan meja melipat spasi ganda.

| Field | Aturan | Galat |
|---|---|---|
| `name` | 1–255 | `400 invalid_input` (`required`, `must be at most 255 characters`) |
| `guest_side` | kosong = belum ditentukan; selain itu kunci (atau label) sisi registry jenis acara proyek, disimpan sebagai kunci; jenis tanpa sisi menolak isian apa pun | `must be a registry side key` · `must be empty for an event type without sides` |
| `category` | kosong = `lainnya`; kunci/label registry → kunci; teks lain = kategori kustom (≤ 60) | `must be at most 60 characters` |
| `sub_event_scope` | ada di master sesi (termasuk `all`); kosong = `all`; diperiksa sebelum menulis (FK-nya ditunda) | `400 invalid_scope` |
| `table_number` | ≤ 30 | `must be at most 30 characters` |
| `pax_count` | bilangan bulat 1–20 | `must be an integer` · `must be between 1 and 20` |
| `phone_number` | ≤ 40; harus lolos `NormalizeWA`; kosong diizinkan | `400 invalid_phone` (`phone_number: must be an Indonesian phone number (08…, +62…)`) |
| `notes` | ≤ 2.000 | `must be at most 2000 characters` |
| `sent_status` | `unsent` atau `sent` | `must be one of unsent, sent` |
| `rsvp_status` | empat nilai | `must be one of unconfirmed, attending, declined, maybe` |
| `pax_actual` (RSVP) | 0–20; hanya berarti untuk `attending` | `must be between 0 and 20` |
| `pax_actual` (check-in, walk-in) | 1–20; walk-in juga menjadi `pax_count` | `must be between 1 and 20` |
| `souvenir_received` | boolean | `400 invalid_json` bila bukan boolean |
| `qr_code_token` | token proyek ini; bentuk salah atau tak dikenal sama saja | `404 not_found` |
| `filename` / `content_base64` | berakhiran `.csv`/`.xlsx`; base64 sah; ≤ 5 MiB; isi cocok dengan ekstensi; ≥ 1 byte | `400 invalid_file` (`file: filename must end with .csv or .xlsx` dst.) · `413 file_too_large` |
| Baris impor | aturan kolom di atas per baris; ≤ 2.000 baris berisi; kolom nama wajib | galat per baris di `ImportResult.errors`; `400 invalid_file` (`file: missing column: Nama Tamu`, `file: too many rows (max 2000)`, `file: empty file`) |
| Query `side` | `none` atau kunci (`^[a-z0-9_]{1,40}$`) | `400 invalid_input: side: must be a side key` |
| Query `scope` | kunci sesi (`^[a-z0-9_]{1,40}$`) | `scope: must be a session scope_key` |
| Query `category` | ≤ 60 | `must be at most 60 characters` |
| Query `rsvp`, `sent`, `status` | enum di atas; `status` (meja) = `all`/`in`/`pending` | `must be one of …` |
| Query `search` | ≤ 100 rune setelah `trim`; kosong = tanpa pencarian | `must be at most 100 characters` |
| Query `limit` / `offset` | daftar 1–500 (bawaan 100); meja 1–200 (bawaan 50); `offset` ≥ 0 | `must be between 1 and 500` · `must be at least 0` |
| Query `buffer_percent`, `buffer_pax` | bilangan bulat 0–100 (bawaan 25) dan 0–1.000 (bawaan 0) | `must be between 0 and 100` · `must be between 0 and 1000` |
| Query `dry_run`, `skip_invalid`, `skip_duplicates` | `true` atau `false`; kosong = `false` | `must be true or false` |
| Query `format` (ekspor, templat) | kosong atau `xlsx` | `400 invalid_input: format: must be xlsx` |
| Jumlah | ≤ 5.000 tamu per proyek, di bawah kunci proyek, semua-atau-tidak | `409 guest_limit_reached` |
| Frekuensi | `scan-qr` 120/menit; impor 5/menit; ekspor + rekap + templat 10/menit; per pengguna | `429 rate_limited` |
| Check-in ganda; ubah RSVP sesudah check-in | satu `UPDATE … WHERE is_checked_in = FALSE`; RSVP ditolak bila `is_checked_in` | `409 already_checked_in` |

Kode galat baru (ditambahkan ke tabel kontrak bab 6): `guest_limit_reached`. Kode lain sudah ada: `invalid_phone`, `invalid_file`, `file_too_large`, `already_checked_in`, `invalid_scope`, `project_read_only`. Tidak ada warning/notice baru di menu ini.

Urutan pemeriksaan: id anak yang bukan UUID kanonik atau milik proyek lain → `404` lebih dulu; kemudian bentuk dan isi body (berurutan menurut field di tabel); batas jumlah `409` diperiksa di dalam transaksi berkunci, sesudah validasi bentuk. `PATCH` memvalidasi hanya field-nya; `PUT` memvalidasi seluruh body.

---

## 4. DTO (Go)

Blok di bawah **sudah dikompilasi** (`go vet`, `gofmt`, dengan tipe Menu 01 yang sama dengan `planner/domain`) dan dibandingkan kunci demi kunci dengan tipe TypeScript [FE bab 4.2](FE.md#42-tambahan-pada-srclibplannertypests): **21 pasangan tipe, 125 jalur JSON, nol selisih** pada kunci, opsionalitas (`omitempty` ↔ `?:`), `null`, dan jenis nilai untuk tipe respons (tipe permintaan: kunci dan jenis saja). Pembanding diuji dengan enam mutasi tipe TS (semuanya dilaporkan). Versi sebelumnya memakai tag `binding`, struct anonim tanpa tag JSON, `AttendancePercent float64`, `whatsapp_url`, dan `reception_notes` yang tidak punya kolom; semuanya diganti.

```go
// backend/internal/modules/plannerguest/domain/entity.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ---------- Tipe milik Menu 01 yang dipakai ulang (planner/domain, sudah ada di kode) ----------

type Option struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type GuestSide struct {
	Key         *string `json:"key"` // nil = belum ditentukan
	Label       string  `json:"label"`
	Invitations int     `json:"invitations"`
	Pax         int     `json:"pax"`
}

type RSVPSummary struct {
	Attending    int `json:"attending"`
	Declined     int `json:"declined"`
	Maybe        int `json:"maybe"`
	Unconfirmed  int `json:"unconfirmed"`
	AttendingPax int `json:"attending_pax"`
}

// ---------- Registry (tambahan pada planner/domain.EventTypeSpec) ----------
//
//	GuestSides      []Option `json:"guest_sides"`      // kosong = tanpa pemisahan pihak
//	GuestCategories []Option `json:"guest_categories"` // kategori bawaan; teks lain = kategori kustom

type RSVPStatus string

const (
	RSVPUnconfirmed RSVPStatus = "unconfirmed"
	RSVPAttending   RSVPStatus = "attending"
	RSVPDeclined    RSVPStatus = "declined"
	RSVPMaybe       RSVPStatus = "maybe"
)

type SentStatus string

const (
	SentUnsent    SentStatus = "unsent"
	SentSent      SentStatus = "sent"
	SentDelivered SentStatus = "delivered" // hanya ditulis modul undangan (bab 5); Planner menulis unsent/sent
)

// ---------- Permintaan (tanpa tag binding: validasi di usecase, pesan "<field>: <alasan>") ----------

// GuestRequest adalah isi POST dan PUT. PUT mengganti seluruh field ini; status RSVP, undangan, check-in,
// souvenir, token, dan urutan tidak pernah berubah lewat PUT.
type GuestRequest struct {
	Name          string  `json:"name"`
	GuestSide     *string `json:"guest_side"`      // kunci sisi registry; kosong = belum ditentukan
	Category      string  `json:"category"`        // kunci/label registry atau teks kustom; kosong = "lainnya"
	SubEventScope string  `json:"sub_event_scope"` // kosong = "all"
	TableNumber   *string `json:"table_number"`
	PaxCount      int     `json:"pax_count"` // 1-20
	PhoneNumber   *string `json:"phone_number"`
	Notes         *string `json:"notes"`
}

type SentStatusRequest struct {
	SentStatus SentStatus `json:"sent_status"` // unsent | sent
}

// RSVPRequest: PaxActual hanya berarti untuk attending (0 = ikut pax_count); status lain menyimpan 0.
type RSVPRequest struct {
	RSVPStatus RSVPStatus `json:"rsvp_status"`
	PaxActual  int        `json:"pax_actual"`
}

type ScanQRRequest struct {
	QRCodeToken string `json:"qr_code_token"`
}

type CheckinRequest struct {
	PaxActual        int  `json:"pax_actual"` // 1-20
	SouvenirReceived bool `json:"souvenir_received"`
}

type SouvenirRequest struct {
	SouvenirReceived bool `json:"souvenir_received"`
}

type WalkInRequest struct {
	Name             string  `json:"name"`
	GuestSide        *string `json:"guest_side"`
	Category         string  `json:"category"`
	TableNumber      *string `json:"table_number"`
	PaxActual        int     `json:"pax_actual"` // 1-20; juga menjadi pax_count
	SouvenirReceived bool    `json:"souvenir_received"`
	Notes            *string `json:"notes"`
}

// ---------- Respons ----------

// Guest: kunci opsional dihilangkan bila kosong (tidak ada null di menu ini, kecuali GuestSide.Key milik Menu 01).
type Guest struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	GuestSide            *string    `json:"guest_side,omitempty"`
	Category             string     `json:"category"`
	SubEventScope        string     `json:"sub_event_scope"`
	TableNumber          *string    `json:"table_number,omitempty"`
	PaxCount             int        `json:"pax_count"`
	PhoneNumber          *string    `json:"phone_number,omitempty"`
	FormattedWANumber    *string    `json:"formatted_wa_number,omitempty"` // 62...; tautan wa.me dibentuk klien
	PersonalizedURL      *string    `json:"personalized_url,omitempty"`    // hanya bila undangan terhubung
	InvitationSentStatus SentStatus `json:"invitation_sent_status"`
	InvitationSentAt     *time.Time `json:"invitation_sent_at,omitempty"`
	RSVPStatus           RSVPStatus `json:"rsvp_status"`
	PaxActual            int        `json:"pax_actual"`
	RSVPSubmittedAt      *time.Time `json:"rsvp_submitted_at,omitempty"`
	QRCodeToken          string     `json:"qr_code_token"`
	IsCheckedIn          bool       `json:"is_checked_in"`
	CheckedInAt          *time.Time `json:"checked_in_at,omitempty"`
	SouvenirReceived     bool       `json:"souvenir_received"`
	Notes                *string    `json:"notes,omitempty"`
}

type CategoryCount struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Custom      bool   `json:"custom,omitempty"` // bukan kategori registry jenis acara ("lainnya" dan teks kustom)
	Invitations int    `json:"invitations"`
	Pax         int    `json:"pax"`
}

// SideCategoryCount: satu sel tabel pihak x kategori (hanya sel yang terisi); Side nil = pihak belum ditentukan.
type SideCategoryCount struct {
	Side        *string `json:"side"`
	Category    string  `json:"category"`
	Invitations int     `json:"invitations"`
	Pax         int     `json:"pax"`
}

type InvitationCounts struct {
	Sent   int `json:"sent"` // sent + delivered
	Unsent int `json:"unsent"`
}

// GuestOverview selalu dihitung atas seluruh proyek, tidak terpengaruh filter, pencarian, atau halaman.
type GuestOverview struct {
	Total          int                 `json:"total"`
	TotalPax       int                 `json:"total_pax"`
	Sides          []GuestSide         `json:"sides"`           // kosong untuk jenis tanpa sisi
	Categories     []CategoryCount     `json:"categories"`      // kategori registry (juga yang 0), lalu "lainnya"/kustom yang terpakai
	SideCategories []SideCategoryCount `json:"side_categories"` // kosong untuk jenis tanpa sisi
	Invitation     InvitationCounts    `json:"invitation"`
	RSVP           RSVPSummary         `json:"rsvp"` // empat status; jumlah = Total
}

type GuestList struct {
	Overview GuestOverview `json:"overview"`
	Guests   []Guest       `json:"guests"`
	Total    int           `json:"total"` // yang cocok dengan filter (untuk halaman), bukan Overview.Total
	Limit    int           `json:"limit"`
	Offset   int           `json:"offset"`
}

type ReceptionKPI struct {
	CheckedInGuests     int  `json:"checked_in_guests"`
	ExpectedGuests      int  `json:"expected_guests"` // belum menolak, atau menolak tetapi hadir
	PendingGuests       int  `json:"pending_guests"`  // ExpectedGuests - CheckedInGuests
	ActualPax           int  `json:"actual_pax"`
	ExpectedPax         int  `json:"expected_pax"`
	SouvenirDistributed int  `json:"souvenir_distributed"`
	SouvenirStock       *int `json:"souvenir_stock,omitempty"` // hilang bila tak ada barang bertanda souvenir (Menu 04)
}

type ReceptionFilters struct {
	All        int             `json:"all"` // = ExpectedGuests
	CheckedIn  int             `json:"checked_in"`
	Pending    int             `json:"pending"`
	Categories []CategoryCount `json:"categories"` // hanya tamu yang diharapkan; Pax tidak diisi
}

type ReceptionOverview struct {
	KPI            ReceptionKPI     `json:"kpi"`
	Filters        ReceptionFilters `json:"filters"`
	RecentCheckins []Guest          `json:"recent_checkins"` // maks 10, terbaru dulu
	Guests         []Guest          `json:"guests"`
	Total          int              `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
}

type CateringProjection struct {
	ConfirmedPax        int `json:"confirmed_pax"` // hanya attending
	PotentialPax        int `json:"potential_pax"` // maybe + unconfirmed (tidak masuk rekomendasi)
	DeclinedPax         int `json:"declined_pax"`
	RecommendedPortions int `json:"recommended_portions"`
	AttendingGuests     int `json:"attending_guests"`
	DeclinedGuests      int `json:"declined_guests"`
	MaybeGuests         int `json:"maybe_guests"`
	PendingGuests       int `json:"pending_guests"` // unconfirmed
	BufferPercent       int `json:"buffer_percent"`
	BufferPax           int `json:"buffer_pax"`
}

// ---------- Impor ----------

// ImportRequest adalah isi POST bulk-import. Berkas dikirim sebagai JSON (bukan multipart): BFF hanya meneruskan
// application/json (kontrak bab 1.3) dan Astro memeriksa origin untuk multipart di balik proxy TLS.
// 5 MiB berkas menjadi sekitar 6,7 MiB base64, di bawah client_max_body_size 8m.
type ImportRequest struct {
	Filename      string `json:"filename"`       // berakhiran .csv atau .xlsx
	ContentBase64 string `json:"content_base64"` // isi berkas, base64 standar
}

type ImportRowError struct {
	Row    int    `json:"row"` // nomor baris di berkas (baris judul = 1)
	Field  string `json:"field"`
	Reason string `json:"reason"` // alasan baku BE 3.4
}

type ImportDuplicate struct {
	Row     int        `json:"row"`
	Name    string     `json:"name"`
	OfRow   *int       `json:"of_row,omitempty"`   // sama dengan baris lain di berkas ini
	OfGuest *uuid.UUID `json:"of_guest,omitempty"` // sama dengan tamu yang sudah ada
}

type ImportPreviewRow struct {
	Row               int     `json:"row"`
	Name              string  `json:"name"`
	GuestSide         *string `json:"guest_side,omitempty"`
	Category          string  `json:"category"`
	SubEventScope     string  `json:"sub_event_scope"`
	TableNumber       *string `json:"table_number,omitempty"`
	PaxCount          int     `json:"pax_count"`
	FormattedWANumber *string `json:"formatted_wa_number,omitempty"`
}

type ImportResult struct {
	DryRun      bool               `json:"dry_run"`
	TotalRows   int                `json:"total_rows"`
	ValidRows   int                `json:"valid_rows"`
	InvalidRows int                `json:"invalid_rows"`
	Created     int                `json:"created"`
	Skipped     int                `json:"skipped"` // baris galat dan duplikat yang dilewati
	Errors      []ImportRowError   `json:"errors"`
	Duplicates  []ImportDuplicate  `json:"duplicates"`
	Preview     []ImportPreviewRow `json:"preview"` // maks 5 baris valid pertama
}
```

---

## 5. Kontrak Antarmuka dengan Modul Undangan Digital

Spesifikasi `DIGITAL_INVITATION_SPEC` masih kosong. Apa yang **dibutuhkan Planner** dinyatakan di sini sebagai kontrak minimum supaya modul undangan kelak dapat ditulis tanpa mengubah Planner. Seluruh blok bab 5.1 sudah dikompilasi dan diuji (`PersonalURL`).

### 5.1 Antarmuka Go (dimiliki `planner/domain`)

```go
// backend/internal/modules/planner/domain/invitation.go
package domain

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// ErrNotFound adalah galat bersama planner/domain (sudah ada); dipakai kedua antarmuka di bawah.
var ErrNotFound = errors.New("not found")

// InvitationLookup diimplementasikan modul Undangan Digital dan dipakai Planner untuk tautan personal.
// Selama modul itu belum ada, Planner dipasang tanpa implementasinya dan personalized_url tidak pernah ada.
type InvitationLookup interface {
	// PublicSlug mengembalikan slug publik undangan milik userID; ErrNotFound bila undangan tidak ada atau bukan miliknya.
	PublicSlug(ctx context.Context, userID, invitationID uuid.UUID) (string, error)
}

// GuestRSVPWriter diimplementasikan Planner (plannerguest) dan dipanggil modul undangan saat tamu mengisi RSVP publik.
// Tamu ditemukan lewat token tiket pada tautan personal (parameter g) dan harus milik invitationID itu; modul undangan
// tidak pernah membaca atau menulis planner_guests.
type GuestRSVPWriter interface {
	// ApplyRSVP idempoten: status dan pax yang sama berhasil tanpa perubahan. Tamu yang sudah check-in tidak diubah
	// dan hasilnya sukses (tamu sudah hadir; formulir publik tidak perlu menampilkan galat di depan pintu).
	// pax diabaikan kecuali status attending (0 = ikut kuota). ErrNotFound bila tak ada tamu dengan pasangan token
	// dan undangan itu; status atau pax di luar batas menghasilkan apierr 400 invalid_input seperti usecase lain.
	ApplyRSVP(ctx context.Context, invitationID uuid.UUID, qrToken string, status RSVPStatus, pax int) error
}

// PersonalURL membentuk tautan personal: {baseURL}/i/{slug}?g={token}&to={nama}. Parameter `to` dibaca halaman undangan
// untuk menyapa tamu; `g` mengidentifikasi tamu untuk RSVP publik (token 128 bit, tidak dapat ditebak).
func PersonalURL(baseURL, slug, guestName, qrToken string) string {
	q := url.Values{"to": {guestName}, "g": {qrToken}}
	return strings.TrimRight(baseURL, "/") + "/i/" + url.PathEscape(slug) + "?" + q.Encode()
}
```

Di kode, `RSVPStatus` dan konstantanya ikut berada di `planner/domain/invitation.go` (antarmuka itu menamainya, dan `planner/domain` tidak boleh mengimpor modul tamu); `plannerguest/domain` memakainya lewat alias. `ErrNotFound` sudah ada di `repository.go` sehingga tidak dideklarasikan lagi.

Dua perubahan terhadap versi sebelumnya, keduanya ditemukan saat mencocokkan bab 2.1 dengan bab 5: (1) `ApplyRSVP` menerima **token tiket**, bukan `guestID`, karena halaman publik hanya memegang tautan personal dan tidak dapat mengetahui id tamu; (2) tautan personal membawa token itu (`g`). Modul undangan tidak boleh membaca atau menulis `planner_guests` langsung; ia memanggil `GuestRSVPWriter`. Kepemilikan: baris tamu selalu dimiliki Planner (bila ada proyek) atau modul undangan (bila `project_id IS NULL`), tidak pernah keduanya sekaligus menulis. `PUBLIC_BASE_URL` ada di `.env.example` tetapi **belum dibaca kode** ([backend/README.md](../../../../backend/README.md)); implementasi menambahkannya ke `internal/config` bersamaan dengan modul undangan (tanpa undangan nilai itu tidak dipakai).

### 5.2 Alur sinkronisasi

| Kasus | Alur |
|---|---|
| **A. Bundling dari awal** | Wizard menerima `invitation_id`; `planner_projects.invitation_id` diisi. Tamu yang dibuat lewat Planner mengisi `invitation_id` dari proyek; tamu yang dibuat di modul undangan menulis ke baris yang sama. Satu sumber data. |
| **B. Undangan dulu, Planner menyusul** | Tamu sudah ada dengan `invitation_id = X` dan `project_id IS NULL`. Saat proyek ditautkan (`PUT /v1/planner/projects/:id/invitation {invitation_id: X}`, hanya terdaftar bila modul undangan ada): `UPDATE planner_guests SET project_id = $1, sub_event_scope = 'all', updated_at = now() WHERE invitation_id = $2 AND project_id IS NULL`. Kolom `project_id` nullable membuat ini valid. **`sub_event_scope` direset ke `all`**: sesi milik modul undangan belum tentu ada di proyek, dan FK komposit yang ditunda gagal saat `COMMIT` (dijalankan dengan sesi `sesi_tak_ada`). |
| **C. Planner dulu, undangan menyusul** | Tamu sudah ada dengan `project_id = Y`. Saat undangan dibuat dan ditautkan: set `planner_projects.invitation_id`, lalu `UPDATE planner_guests SET invitation_id = $1, updated_at = now() WHERE project_id = $2 AND invitation_id IS NULL` (dijalankan: 20 baris). Tautan personal otomatis tersedia (dihitung saat dibaca). |
| **Standalone** | `invitation_id` kosong; RSVP manual; tombol WhatsApp memakai pesan tanpa tautan ([bab 2.4](#24-pesan-whatsapp-undangan)). |

### 5.3 Migrasi modul undangan

Migrasi modul undangan menambahkan `ALTER TABLE planner_guests ADD CONSTRAINT planner_guests_invitation_fk FOREIGN KEY (invitation_id) REFERENCES <tabel_undangan>(id) ON DELETE SET NULL;` dan hal yang sama untuk `planner_projects.invitation_id`. Sampai itu, kedua kolom hanyalah UUID bebas, dan endpoint `PUT …/invitation` tidak terdaftar.

**`ON DELETE SET NULL` saja tidak cukup untuk `planner_guests`** (versi sebelumnya berhenti di sini). Dijalankan: menghapus undangan yang masih punya tamu-saja-undangan (`project_id IS NULL`) membuat `invitation_id` menjadi `NULL` pada baris yang tidak punya `project_id`, dan `CHECK (project_id IS NOT NULL OR invitation_id IS NOT NULL)` menolaknya (`violates check constraint "planner_guests_check"`). Aturan yang terbukti berhasil: **modul undangan menghapus tamu-saja-undangan lebih dulu**, baru undangannya:

```sql
DELETE FROM planner_guests WHERE invitation_id = $1 AND project_id IS NULL;   -- lalu: DELETE FROM <tabel_undangan> WHERE id = $1
```

Tamu milik proyek hanya kehilangan `invitation_id` (tetap ada; dijalankan). `ON DELETE CASCADE` ditolak: ia menghapus tamu yang dimiliki proyek Planner bila undangan dihapus. Penghapusan akun: tamu-saja-undangan terhapus lewat FK undangan → pengguna milik modul undangan ([bab 10 butir 24](#10-celah-dan-keputusan-yang-menunggu-pemilik-produk)).

### 5.4 Yang tidak ada di kontrak

Amplop/angpau digital lewat payment gateway **tidak ada** di modul ini: Adicara tidak menampung dana (prinsip Tab Tabungan) dan layanan Gift belum punya spesifikasi. Jika kelak ada, hasilnya ditampilkan baca-saja di dashboard, bukan ditulis ke buku kas tabungan. Gambar QR untuk tiket tamu juga tidak dibuat Planner ([bab 2.5](#25-meja-resepsionis-hari-h)).

---

## 6. Data Contoh (Proyek Contoh)

PDF Hal. 13 **tidak berisi nama tamu**: ia hanya memuat dua tabel ("LIST TAMU MEMPELAI LAKI-LAKI/PEREMPUAN") dengan enam kategori relasi berhitung 0 (Teman, Kolega, Keluarga, Tetangga, Teman Orang Tua, VIP). Dua puluh tamu di bawah ini **fiktif** (bukan dari PDF), hanya dimuat di Proyek Contoh dengan nomor telepon berpola `08120000xxxx`. Tidak ada tamu yang sudah check-in (Proyek Contoh berada di H-175). Tabel ini dibaca oleh tes (`TestSampleGuestsMatchSpec`, [bab 8.2](#82-pengujian-pada-implementasi)) dan pembangkit fixture; jangan mengubah bentuk kolomnya tanpa mengubah keduanya.

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

Ringkasan (dihitung dari SQL pada tabel ini): 20 undangan, **43 pax** (Pria 10 undangan / 22 pax; Wanita 10 / 21); terkirim 14; RSVP: 12 hadir (27 pax), 2 ragu, 2 tidak, 4 belum; kategori Teman 6 · Kolega 3 · Keluarga 4 · Tetangga 2 · Teman Orang Tua 3 · VIP 2, dan per pihak (tabel PDF): Pria 3/2/2/1/1/1, Wanita 3/1/2/1/2/1. Enam kategori PDF tercakup semuanya. Proyeksi katering: dikonfirmasi 27 pax, belum pasti 12, menolak 4, rekomendasi 34 porsi (cadangan 25%).

Kolom turunan yang diisi overlay: `formatted_wa_number` (`62…`), `pax_actual = pax_count` untuk `attending` (selain itu 0), `invitation_sent_at` untuk yang terkirim dan `rsvp_submitted_at` untuk yang sudah merespons (`now()` saat Proyek Contoh dibuat; fixture memakai garis dasar 2026-10-04), `qr_code_token` acak, `sort_order = No`. RSVP pada tamu 15–20 yang undangannya belum terkirim berarti konfirmasi lisan.

**Ilustrasi hari-H** (bukan data Proyek Contoh, yang belum berada di hari-H; dipakai wireframe Tab 2 dan [fixtures/planner-reception.json](fixtures/planner-reception.json)): tamu nomor 1, 2, 6, 3, 11, 4, 7, 9, 10, 14, 19 masuk berurutan antara 11.14 dan 12.16 dengan pax riil = pax undangan (22 pax) dan semuanya menerima souvenir (stok contoh 40). Hasil: 11 dari 18 tamu yang diharapkan (7 belum), 22 dari 39 pax, 11 souvenir diserahkan (sisa 29); penghitung tidak memuat 2 tamu yang menolak. Fixture lainnya ada di [fixtures/](fixtures/): `planner-guests.json` (`GuestList` Proyek Contoh), `planner-catering-projection.json`, `planner-import-dryrun.json` dengan `planner-import-sample.csv` (CSV Excel dengan BOM dan titik koma; 6 baris: 4 valid, 2 galat, 2 duplikat). Dibangun dari tabel ini dan fungsi acuan bab 2, dibangkitkan dan dijaga oleh `TestFixturesAreCurrent`, dan **sama dengan respons HTTP asli** modul ini (`TestResponsesMatchTheFEFixtures`: daftar, proyeksi, pratinjau impor, dan paket meja resepsionis setelah id, tiket, dan waktu dinormalkan).

---

## 7. Sambungan ke Menu 01

`plannerguest.Service` memenuhi dua peran dari `planner/domain` (`repository.go`, kode yang sudah ada) dan menambah registry:

| Antarmuka | Kunci | Isi |
|---|---|---|
| `SummaryProvider` | `guests` | `domain.GuestSummary` (`planner/domain/summary.go`, **sudah ada**): `invitations = Total`, `total_pax`, `sides = Sides` (label dari registry; `key` nil = belum ditentukan; kosong untuk jenis tanpa sisi), `sent = Invitation.Sent`, `rsvp = RSVP`. Dibangun dari `AssembleOverview` ([bab 2.8](#28-ringkasan-penghitung-kategori-dan-pihak)); kueri ringkasan sama dengan [01 BE §5.2](../01_DASHBOARD/BE.md#52-query-per-provider) |
| `TemplateApplier` | – | **tidak ada**: wizard tidak membuat tamu ([kontrak bab 5](../00_KONTRAK_BERSAMA/README.md#5-template-vs-data-contoh)); `TemplateSet` tidak punya bendera tamu |
| `SampleOverlay` | `guests` | 20 tamu bab 6 pada sesi `resepsi`/`all`; didaftarkan **terakhir** (tidak bergantung pada overlay lain) |

Pemasangan di `internal/app/app.go` (menambah ke baris Menu 02–04; registry dan `TxManager` adalah instance yang sama):

```go
guestSvc := guestusecase.New(guestrepo.NewPostgresRepository(pool), plannerTx, registry, guestusecase.Options{ /* Invitations: nil sampai modul undangan ada */ })
plannerSvc.RegisterSummary(timelineSvc.TimelineProvider(), timelineSvc.AdminDocsProvider(), financeSvc.Provider(), vendorSvc.Provider(), guestSvc.Provider())
plannerSvc.RegisterSampleOverlays(timelineSvc.TimelineOverlay(), timelineSvc.AdminDocsOverlay(), financeSvc.Overlay(), vendorSvc.VendorsOverlay(), vendorSvc.ShoppingOverlay(), guestSvc.GuestsOverlay())
guesthttp.RegisterRoutes(ginEngine, idn, plannerSvc, guestSvc)
```

Kunci terdaftar dua kali atau tak dikenal membuat start gagal (panic), sengaja.

**Registry.** Menu ini menambah ke `EventTypeSpec` (nilai: [TEMPLATE_EVENT bab 4 dan 4.2](../00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md#4-peran-sisi-tamu-dan-kategori), sembilan jenis):

```go
// pada EventTypeSpec (planner/domain/eventtype.go):
GuestSides      []Option `json:"guest_sides"`      // kosong = tanpa pemisahan pihak
GuestCategories []Option `json:"guest_categories"` // kategori bawaan jenis; teks lain = kategori kustom
```

`GET /event-types` mengirim `guest_sides` dan `guest_categories` sebagai `[{key, label}]` (FE bab 4.2, `EventTypeGuestInfo`); `Labels.Guest`/`GuestMenu` sudah ada. Tes keselarasan membaca TEMPLATE_EVENT: kunci bab 4 = kunci 4.2 untuk kesembilan jenis (sudah dijalankan pada kode acuan, `TestKeysOfSection4MatchTheLabelsOfSection42`), dan kunci cadangan `lainnya` bukan kategori registry mana pun.

**Pengecualian SQL.** Tidak ada penulisan lintas modul. Pembacaan: stok souvenir dari `planner_shopping_items` (sudah tercatat di [kontrak bab 1.1](../00_KONTRAK_BERSAMA/README.md#11-backend-go-gin-module-based-clean-architecture)). Pembacaan lintas modul yang sudah ada dan kini punya pemilik: kartu Hub (`guest_pax_count` = Σ `pax_count` seluruh tamu, termasuk yang menolak: ini pax diundang, bukan pax hadir), `usage.guests` pada sesi, dan pemindahan tamu saat sesi dihapus.

---

## 8. Pengujian

### 8.1 Yang sudah dijalankan pada audit spesifikasi

Dijalankan 2026-10-25 pada PostgreSQL 16.2 lokal, database sementara `planner_spec05_scratch_test` (dibuat khusus dengan migrasi 001–012 dari berkas nyata lewat `go run ./cmd/migrate up`, dihapus setelahnya; database `adicara` tidak disentuh), koneksi `psql`, dan **backend Menu 01–04 yang sebenarnya** pada port sementara (pengguna uji, `POST /projects/sample`, sembilan proyek per jenis acara). Skrip dan modul Go/TS ada di scratchpad sesi, bukan di repo.

| Butir | Hasil |
|---|---|
| Blok SQL bab 1 identik dengan migrasi 011; naik di atas 001–012 | ✔ perbandingan teks; 14 tabel + 1 view `planner_*` |
| PDF Hal. 13 dibaca ulang (`pdfminer`): dua tabel, enam penghitung per tabel bernilai 0, tanpa nama | ✔ sama dengan bab 6 (versi lama hanya punya satu set penghitung gabungan) |
| 20 tamu bab 6 dimasukkan ke Proyek Contoh asli: 20 / 43 pax; pihak 10/22 dan 10/21; terkirim 14; RSVP 12/2/2/4; hadir 27 pax; kategori 6/3/4/2/3/2; per pihak 3/2/2/1/1/1 dan 3/1/2/1/2/1; harapan hari-H 39 pax / 18 tamu | ✔ SQL dan `AssembleOverview` (nol selisih) |
| Constraint: tamu undangan-saja diizinkan; scope yang tidak ada di proyek gagal baru saat `COMMIT`; walk-in 21 pax melanggar `planner_guests_pax_count_check`; `pax_actual = 500` diterima DDL | ✔ (bab 1) |
| Check-in atomik: dua sesi `psql`, sesi kedua mendapat 0 baris; token proyek lain 0 baris; batal check-in mereset flag | ✔ |
| Penghitung "belum hadir" aturan lama dengan satu penolak yang hadir: 6 (benar 7); definisi baru `expected = bukan penolak ∨ hadir` | ✔ bab 2.5 |
| Pencarian: `strpos` dengan `%` → 0 baris, `ILIKE` → 20; nomor apa adanya dan `62…`; tamu penolak hanya dapat dicari dengan `search` | ✔ |
| Semua blok SQL bab 2 (daftar + filter, ringkasan, proyeksi, scan, check-in, batal, walk-in, KPI, antrean, RSVP, kunci proyek, tamu yang ada, `INSERT` multi-baris, stok souvenir) dijalankan **apa adanya** (hanya `$n` diganti nilai) dalam transaksi yang di-rollback | ✔ tanpa galat |
| Proyeksi: `27 / 12 / 4`; pembulatan `float64` berbeda dari jawaban eksak pada 663 dari 303.101 pasangan (tamu 0–3.000 × persen 0–100); versi bilangan bulat tidak pernah | ✔ `TestRecommendedPortionsNeverDiffersFromExactCeil` |
| FK undangan `ON DELETE SET NULL` menghapus undangan berisi tamu-saja-undangan → `planner_guests_check` menolak; urutan "hapus tamu-saja-undangan lalu undangan" berhasil; tamu proyek hanya kehilangan `invitation_id`; kasus B dan C | ✔ bab 5.3 |
| Backend Menu 04 asli: barang souvenir 300 + 50,5 selesai dan 20 proses → `souvenir_stock: 350` | ✔ |
| Backend Menu 03 asli: `PUT /budget/items/:id` `pricing_mode: unit`, `quantity: 34` → `200`; tanpa `vendor_id` melepas vendor; `quantity: 0` → `400` | ✔ bab 2.3 |
| Backend Menu 01 asli: hapus sesi `akad` dengan `move_to=resepsi` memindahkan 3 tamu (6 di `resepsi`); `guest_pax_count` Hub = 12; hapus proyek menghapus tamunya; widget `guests` `null` sebelum provider | ✔ |
| Gin v1.12.0: ke-19 rute (statis berdampingan dengan `:guest_id`, `GET`/`POST`/`PUT`/`PATCH`/`DELETE`) | ✔ nol salah arah |
| CSV Excel: BOM + titik koma → satu kolom dengan `encoding/csv` bawaan; Windows-1252; sel telepon numerik `81200000101` | ✔ bab 2.6 |
| `excelize` v2.11.0: batas ekstrak bawaan 16 GB; unggahan 1 MiB yang mengklaim 300 MiB diekstrak ke direktori sementara; dengan `UnzipSizeLimit` 64 MiB ditolak seketika; templat 2.000 baris terbaca 29 ms | ✔ |
| Templat impor dibuat dengan `excelize` dan dibaca balik pembaca impor (judul dikenali, nol baris, dropdown, gaya teks) | ✔ (belum dibuka di Excel/Numbers) |
| Blok DTO bab 4: `gofmt`, `go vet`; kunci sama dengan tipe TS FE (21 pasangan, 125 jalur, nol selisih); pembanding diuji dengan enam mutasi | ✔ |
| Seluruh tautan relatif di folder PLANNER_SPEC (termasuk ke bab-bab dokumen ini) | ✔ nol rusak |

**Tidak dijalankan pada audit** (butir pertama sampai ketiga kemudian dijalankan pada implementasi, [bab 8.4](#84-hasil-implementasi-2026-10-25)): kode modul `plannerguest`, jalur HTTP dan Gin untuk modul ini, balapan pada impor dan batas 5.000, ekspor/templat `.xlsx` yang dibuka di Excel/Numbers, latensi pada 5.000 tamu, PostgreSQL 18 (versi Compose), Docker, Nginx, dan kamera/`BarcodeDetector`.

### 8.2 Pengujian pada implementasi

Nama tes; semuanya ✔ (ada dan lulus pada implementasi, 2026-10-25). Tes DB butuh `TEST_DATABASE_URL` ke database `…_test`, dijalankan `go test -p 1`; tanpa itu tes DB di-skip. Berkas ada di `backend/internal/modules/plannerguest/{domain,usecase,delivery/http,delivery/http/xlsx}` dan `planner/domain` (`TestPersonalURL`). Tes yang berasal dari kode acuan ([bab 8.3](#83-hasil-kode-acuan-2026-10-25)) dipindahkan apa adanya; sisanya ditulis untuk implementasi.

| Butir | Tes |
|---|---|
| Data contoh sama dengan tabel spesifikasi; registry sama dengan TEMPLATE_EVENT dan README | ✔ `TestSampleGuestsMatchSpec` (membaca bab 6), `TestRegistryHasNineTypesWithTheDocumentedShape`, `TestKeysOfSection4MatchTheLabelsOfSection42`, `TestEventTypesCarryTheGuestRegistry` (kode registry = TEMPLATE_EVENT 4.2), `TestGuestRegistryIsNotShared`, `TestReadmeNineTypeTableMatchesTemplateEvent`, `TestReadmeNineTypeTableMatchesRegistry` |
| Sisi dan kategori: kunci/label/tanpa membedakan huruf, kustom, cadangan, jenis tanpa sisi | ✔ `TestResolveSide`, `TestResolveCategory`, `TestCleanSideRefusesAnySideForATypeWithoutSides` |
| Proyeksi: tamu `declined` tidak masuk; bilangan bulat = ceil eksak; batas parameter | ✔ `TestRecommendedPortions`, `TestRecommendedPortionsNeverDiffersFromExactCeil`, `TestBufferParameters` |
| Ringkasan: Proyek Contoh vs SQL; proyek kosong tanpa `null`; seminar tidak memunculkan Pria/Wanita; jenis tanpa sisi; urutan kategori; sisi tak dikenal; sel pihak terurut; satu tamu tanpa pihak; chip filter meja | ✔ `TestSampleProjectOverview`, `TestOverviewOfAnEmptyProject`, `TestSeminarShowsItsOwnSidesAndNeverPriaWanita`, `TestEventTypesWithoutSidesNeverEmitSides`, `TestCategoryOrderIsRegistryThenFallbackThenCustomAlphabetical`, `TestAnUnknownSideKeyStaysVisibleSoTheSumsHold`, `TestSideCategoryCellsAreSortedWhateverTheInputOrder`, `TestASingleGuestWithoutASideStillShowsTheUndecidedEntry`, `TestReceptionFiltersListTheRegistryThenTheUsedFreeText`, `TestSideLabel` |
| Impor: BOM + titik koma, pemisah lain, tanda kutip, Windows-1252; judul; validasi tiap kolom; baris kosong dan nomor baris; bawaan; sesi; jenis tanpa sisi; kategori kustom; duplikat; `skip_*`; batas 2.000; hasil tanpa `null` | ✔ `TestExcelCSVWithBOMAndSemicolonIsReadAsFourColumnsNotOne`, `TestDelimitersAndQuoting`, `TestWindows1252IsDecoded`, `TestHeaderMapping`, `TestRowValidation`, `TestBlankRowsAreSkippedAndRowNumbersAreFileLines`, `TestDefaultsOfABareRow`, `TestSessionsResolveByNameOrKey`, `TestAnEventTypeWithoutSidesRejectsAFilledPihakButNotAnEmptyOne`, `TestCustomCategoryAndRegistryLabel`, `TestDuplicates`, `TestSkipInvalidWritesTheValidRows`, `TestRowLimitAndPreview`, `TestResultJSONHasNoNullLists` |
| Berkas: `DecodeUpload` (ekstensi, base64, isi cocok, 5 MiB); `.xlsx` 2.000 baris dalam batas, berhenti setelah 2.001, telepon numerik, bom dekompresi ditolak tanpa berkas sementara; templat dibaca balik dan memuat dropdown/gaya teks/kata Peserta | ✔ `TestDecodeUpload`, `TestDecodeUploadLimit`, `TestAFullTemplateOf2000RowsIsReadWellInsideTheLimits`, `TestReadingStopsAfterTheRowLimitPlusOne`, `TestNumericPhoneCellsLoseTheirLeadingZeroAndAreRefused`, `TestADecompressionBombIsRefusedAndLeavesNoTempFiles`, `TestNotAnXLSX`, `TestTheTemplateIsReadBackByTheImporter`, `TestTemplateDropdownsPhoneColumnAndSeminarWording` |
| Tautan personal membawa token dan meloloskan karakter; hanya ada bila modul undangan, URL dasar, dan proyek bertaut; pencarian slug gagal tidak disembunyikan | ✔ `TestPersonalURL` (`planner/domain`), `TestPersonalLinksNeedAnInvitation`, `TestLinksDistinguishAMissingInvitationFromABrokenLookup` |
| Angka fixture sama dengan wireframe; fixture di repo sama dengan keluaran kode **dan dengan respons HTTP asli** | ✔ `TestSampleDayOfNumbersMatchTheWireframe`, `TestSampleProjectionAndImportNumbers`, `TestFixturesAreCurrent`, `TestResponsesMatchTheFEFixtures`, `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays` (setiap pesan contoh di katalog galat FE 4.6 dihasilkan backend) |
| CRUD tamu: tambah/ubah/hapus, validasi tiap baris bab 3.4 (urutan dan pesan), `PUT` tidak menyentuh RSVP/check-in/token/urutan, kategori kustom, batas 5.000 (termasuk balapan) | ✔ `TestGuestLifecycle`, `TestGuestValidation`, `TestCheckGuestNormalisesAndValidatesInTheOrderOfTheSpec`, `TestTextLimitsCountRunes`, `TestPutKeepsRSVPCheckinAndToken`, `TestGuestLimitOf5000`, `TestConcurrentCreatesRespectTheLimit` |
| RSVP/status kirim: `rsvp_submitted_at`, `pax_actual` hanya `attending`, ditolak sesudah check-in, `ApplyRSVP` idempoten dan mengabaikan tamu hadir, token undangan lain `ErrNotFound` | ✔ `TestRSVPStampsAndClears`, `TestCheckRSVPStoresPaxOnlyForAttending`, `TestRSVPLockedAfterCheckin`, `TestApplyRSVPByTokenIsIdempotent`, `TestApplyRSVPNeverCrossesInvitations` |
| Meja resepsionis: scan (token proyek lain 404), check-in ganda serentak (satu sukses, tujuh `409`), batal (flag, `pax_actual` tetap), souvenir, walk-in (batas 20, batas jumlah), KPI dengan penolak yang hadir (pending 7), antrean (`status`, pencarian melebarkan ke penolak, urutan, halaman), chip kategori kustom, stok souvenir hilang/ada/0, paket polling satu halaman | ✔ `TestScanFindsOnlyThisProjectsTickets`, `TestCheckinRules`, `TestConcurrentCheckinOneWins`, `TestCancelCheckin`, `TestSouvenirMark`, `TestWalkIn`, `TestReceptionKPIWithADeclinedArrival`, `TestReceptionQueueRules`, `TestReceptionFiltersShowCustomCategories`, `TestSouvenirStockOmittedWithoutItems`, `TestReceptionPollingPacketStaysSmall` |
| Daftar: filter `side=none`/kategori/RSVP/kirim/sesi, halaman, `strpos` (`%` tidak mencocokkan semua), total vs halaman, overview tidak terpengaruh filter, parameter query | ✔ `TestListFilters`, `TestSearchHasNoWildcards`, `TestOverviewIgnoresFilters`, `TestPageParameters`, `TestGuestFilterParsing` |
| Impor lewat HTTP: JSON base64, `dry_run` tidak menulis, atomik, `skip_invalid`/`skip_duplicates`, `.xlsx`, batas total 5.000 (juga serentak), 413, `invalid_file`, 5/menit; templat dan ekspor dibaca ulang (`excelize`), sel `=` aman, nama berkas dari slug, jam masuk di zona proyek, 10/menit bersama, `scan-qr` 120/menit | ✔ `TestBulkImportDryRunWritesNothing`, `TestBulkImportIsAtomic`, `TestBulkImportFlags`, `TestBulkImportRejectsBadFiles`, `TestBulkImportRespectsTheGuestLimit`, `TestConcurrentImportsRespectTheLimit`, `TestImportRateLimit`, `TestBulkImportReadsAWorkbook`, `TestBulkImportAcceptsTheLargestFile`, `TestTemplateEndpointReadBack`, `TestGuestExports`, `TestAttendanceEntryTimeFollowsTheProjectZone`, `TestExportsRejectOtherFormats`, `TestDownloadsShareARateLimit`, `TestScanRateLimit`, `TestUserLimiterCountsPerKeyAndResetsAfterTheWindow`, `TestSafeCellPrefixesFormulas` |
| **Uji emas Proyek Contoh** (kontrak bab 9) | ✔ `TestSampleProjectGuestGolden`: 20 / 43, 10/22 dan 10/21, terkirim 14, RSVP 12/2/2/4 dan 27 pax, enam kategori dan per pihak, proyeksi 27/12/4 → 34, widget dashboard `guests` sama dengan `GuestSummary`, Hub `guest_pax_count` 43, `usage.guests` 16 / 4 / 0 |
| Sembilan jenis acara menjalani alur yang sama (tambah tamu per kategori dengan sisi registry, sisi jenis lain ditolak, ringkasan menjumlah ke total, impor dengan label, check-in, widget, templat dengan kata jenisnya, kedua ekspor, wizard tidak membuat tamu) | ✔ `TestEveryEventTypeRunsTheWholeGuestFlow` |
| Integrasi: sesi dihapus memindahkan tamu; tamu `project_id IS NULL` tidak muncul di query Planner mana pun; hapus proyek menghapus tamu; modul terpasang di host | ✔ `TestDeletingASessionMovesItsGuests`, `TestInvitationOnlyGuestsNeverAppearInPlanner`, `TestDeleteProjectCascadesGuests`, `TestGuestWiring` |
| Isolasi: pengguna B `404` di 19 rute proyek A; tanpa token `401`; arsip menolak tulis (`GET`, ekspor, templat tetap boleh); body rusak dan terlalu besar | ✔ `TestUserBGetsNotFoundOnEveryGuestEndpoint`, `TestEveryGuestRouteNeedsABearerToken`, `TestArchivedProjectIsReadOnlyButStillReadable`, `TestGuestEndpointsRejectMalformedBodies` |
| Kontrak OpenAPI: 19 operasi dan 24 skema = rute dan tag JSON struct; `PlannerEventType` memuat `guest_sides`/`guest_categories` | ✔ `TestOpenAPIDocumentsEveryGuestRoute`, `TestOpenAPISchemasMatchTheGoDTOs`, `TestOpenAPIEventTypeHasTheGuestRegistry` |

### 8.3 Hasil kode acuan (2026-10-25)

Modul acuan (scratchpad sesi; paket `domain` dan `xlsx`) berisi seluruh blok Go di dokumen ini. **43 fungsi tes Go lulus** (termasuk satu pembangkit bentuk JSON untuk pembanding tipe), `go vet` dan `gofmt` bersih, cakupan `domain` 93,5% dan `xlsx` 90,4%. **24 mutasi kode produksi** (pembulatan, persen buffer, hitungan belum terkirim, label tanpa kurung, pembuangan BOM, deteksi pemisah, kunci duplikat, batas pax, baris kosong, batas baris, Windows-1252, urutan sel, pihak belum ditentukan, judul ganda, normalisasi telepon, batas ekstrak, batas baca, dropdown, gaya teks telepon, kolom Pihak, kata Peserta, batas 5 MiB, isi zip pada CSV, tautan tanpa token): **semuanya digagalkan tes** (empat mutan pertama yang selamat memperkuat empat tes: pembulatan diuji pada semua persen 0–100, urutan sel dengan masukan terbalik, satu tamu tanpa pihak, dan judul ganda). Fixture di repo dibangkitkan oleh `TestFixturesAreCurrent` dan angkanya sama dengan wireframe dan SQL. Tes-tes ini dipindahkan ke modul apa adanya (jalur dokumen dibuat relatif; `TestDumpShapes` tinggal di scratchpad).

### 8.4 Hasil implementasi (2026-10-25)

Dijalankan pada PostgreSQL 16.2 lokal, database sementara `planner_guest_impl_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`. Rangkuman, penyimpangan, dan yang belum diuji: [docs/backend/planner/2026-10-25-planner-menu-05.md](../../../backend/planner/2026-10-25-planner-menu-05.md).

| Butir | Hasil |
|---|---|
| Seluruh suite backend | ✔ 448 tes tingkat atas lulus, 0 gagal, 0 di-skip (334 sebelumnya + 114 baru) |
| `-race` pada `plannerguest/...` dan `planner/...` | ✔ tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`) | ✔ 91,9% pernyataan |
| `go vet`, `gofmt`, `redocly lint backend/docs/openapi.yaml` | ✔ bersih; OpenAPI valid (3 peringatan lama) |
| 19 rute terdaftar = 19 operasi OpenAPI; 24 skema = tag JSON struct Go; `PlannerEventType` memuat registry tamu | ✔ `TestOpenAPI*` |
| Uji emas Proyek Contoh lewat router, PostgreSQL, dan wiring asli | ✔ `TestSampleProjectGuestGolden`: semua angka bab 6 |
| **Respons HTTP asli sama dengan keempat fixture FE** (daftar, proyeksi, pratinjau impor, paket meja resepsionis) | ✔ `TestResponsesMatchTheFEFixtures`, nol selisih setelah id, tiket, dan waktu dinormalkan; diuji dapat gagal |
| **Setiap baris katalog galat FE bab 4.6 dihasilkan backend dengan status, kode, dan pesan yang tertulis** | ✔ `TestErrorCatalogOfTheFEDocIsWhatTheBackendSays`; diuji dapat gagal |
| Balapan: delapan check-in serentak pada satu tamu; 12 tambah serentak pada 4.995 tamu; dua impor serentak pada 4.990 tamu | ✔ satu menang / tepat lima berhasil dengan `sort_order` unik / satu 200 dan satu 409 |
| Batas 5.000 tepat (ke-5.000 lolos, ke-5.001 `409`) untuk tambah, walk-in, dan impor | ✔ |
| Sembilan jenis acara menjalani alur yang sama | ✔ `TestEveryEventTypeRunsTheWholeGuestFlow` |
| Mutasi kode produksi (SQL, usecase, delivery) | ✔ 33 mutasi kode produksi (kunci proyek dan batas, SQL antrean dan KPI, cap RSVP, stok souvenir, urutan daftar, pembatas laju, batas badan impor, awalan rumus, zona jam masuk, dan lainnya) semuanya digagalkan tes; dua awalnya selamat (batas badan impor dan zona jam masuk, mesin uji kebetulan berzona WIB) dan menambah dua tes, empat tidak terkompilasi dan ditulis ulang |
| Isolasi pengguna B di 19 rute; tanpa token 401; arsip baca-saja; badan rusak/terlalu besar | ✔ |

**Belum diuji:** frontend dan BFF (belum ada), modul Undangan Digital (pencarian slug tiruan), ekspor/templat `.xlsx` di Excel/Numbers/LibreOffice, kinerja pada 5.000 tamu, kolasi `postgres:18-alpine`, PostgreSQL 18, Docker, Nginx, beberapa replika, dan kamera/`BarcodeDetector`.

---

## 9. Urutan pengerjaan dan definisi selesai

Posisi di urutan pembangunan ([kontrak bab 9](../00_KONTRAK_BERSAMA/README.md#9-urutan-pembangunan-dan-uji-penerimaan)): setelah `planner` (Menu 01), `plannertimeline` (02), `plannerfinance` (03), dan `plannervendor` (04); tabel dan registry yang dibutuhkan sudah ada.

1. **Domain**: DTO bab 4, `GuestSpec` (`ResolveSide`, `ResolveCategory`, `AssembleOverview`), `RecommendedPortions`, parser impor (`DecodeCSV`, `BuildImportPlan`, `DecodeUpload`), `PersonalURL`, antarmuka bab 5; registry `GuestSides`/`GuestCategories` untuk sembilan jenis beserta `guest_sides`/`guest_categories` di `GET /event-types`; tes data dan registry.
2. **Repository** dan **usecase**: daftar + ringkasan, CRUD, RSVP/status kirim, meja resepsionis (scan, check-in, batal, walk-in, souvenir, KPI, antrean), proyeksi, impor/ekspor/templat (`xlsx`), `ApplyRSVP`, dengan `LockProject`.
3. **Delivery**: 19 rute, DTO ke JSON, pemetaan galat (`guest_limit_reached` baru), tiga pembatas laju (scan 120, impor 5, unduhan 10), batas badan 7,5 MiB untuk `bulk-import`; **perubahan BFF** satu jalur (batas badan khusus `…/guests/bulk-import`, [01 FE bab 6](../01_DASHBOARD/FE.md#6-bff-dashboardapiplanner)).
4. **Sambungan** di `internal/app/app.go` ([bab 7](#7-sambungan-ke-menu-01)): provider `guests`, overlay `guests`, rute.
5. **Kontrak dan dokumen**: operasi dan skema ke `backend/docs/openapi.yaml` (tag `Planner Guest`); dokumen `docs/backend/planner/<tanggal>-planner-menu-05.md`; `docs/api.md`, `docs/database.md` (tidak ada tabel baru), `docs/backend/CHANGELOG.md`; kode galat baru di kontrak bab 6; entri CHANGELOG spesifikasi.
6. **Uji emas** Proyek Contoh dan seluruh tabel [bab 8.2](#82-pengujian-pada-implementasi); `-race`, `go vet`, `gofmt`, cakupan.

**Status: enam langkah selesai 2026-10-25** ([bab 8.4](#84-hasil-implementasi-2026-10-25)); langkah 3 tanpa perubahan BFF karena BFF Planner belum ada di repo (batas badan 7,5 MiB untuk `…/guests/bulk-import` ikut saat BFF dibuat).

**Definisi selesai (terpenuhi):** semua butir bab 8.2 lulus terhadap PostgreSQL; widget Buku Tamu di dashboard terisi (`GuestSummary`) dan `usage.guests` tetap benar; Proyek Contoh memuat 20 tamu dengan angka bab 6; `GET /event-types` memuat sisi dan kategori kesembilan jenis; OpenAPI lolos lint; dokumen modul menyebut setiap penyimpangan dari spesifikasi ini.

---

## 10. Celah dan keputusan yang menunggu pemilik produk

Temuan saat mencocokkan dokumen ini dengan Menu 01–04 (kode dan dokumen), kontrak, PDF, dan SQL yang dijalankan. Dokumen mengikuti pilihan yang tertulis; kolom "alternatif" adalah yang bisa dipilih bila produk tidak setuju.

1. **Angka batas adalah usulan tanpa pengukuran:** 5.000 tamu per proyek, 2.000 baris dan 5 MB per impor, pax 1–20, teks 255/60/30/40/2.000, pencarian 100. Daftar berpaginasi (bawaan 100); antrean meja 50.
2. **Satuan souvenir per undangan** (satu tanda per baris tamu, bukan per orang); stok `floor(Σ quantity)` dibandingkan dengan jumlah tamu yang menerima. Alternatif: kolom jumlah souvenir per tamu (default = `pax_actual`).
3. **`pax_actual` satu kolom untuk dua hal** (RSVP lalu pintu); membatalkan check-in tidak mengembalikan angka RSVP. Alternatif: kolom `pax_checked_in` terpisah (migrasi 011 boleh diubah di tempat pada tahap pengembangan).
4. **Tamu yang menolak tetapi datang:** RSVP tetap `declined`, dihitung hadir di meja resepsionis (`expected = bukan penolak ∨ hadir`). Alternatif: check-in mengubah RSVP menjadi `attending` (kehilangan informasi bahwa ia sempat menolak, dan tidak bisa dipulihkan saat check-in dibatalkan).
5. **RSVP tidak dapat diubah sesudah check-in** (`409 already_checked_in`; batalkan dulu). Alternatif: izinkan dan abaikan `pax_actual`.
6. **Standalone tanpa "tautan eksternal".** Versi sebelumnya menyebutnya tanpa kolom penyimpan; kini WhatsApp mengirim pesan tanpa tautan. Alternatif: bidang `invitation_url` pada `metadata` proyek (Menu 01) yang disisipkan ke pesan.
7. **Tautan personal membawa token tiket (`g`)**, jadi tautan = kredensial RSVP dan tiket. Alternatif: token RSVP terpisah (kolom baru) sehingga tautan yang bocor tidak dapat dipindai di pintu.
8. **Planner tidak membuat gambar QR.** Penghasil QR membutuhkan pustaka (Go atau JS) dan "tanpa dependensi baru" berlaku di kedua sisi; kolom *Kode Tiket* pada ekspor memungkinkan alat luar. Pemindai (FE) hanya membaca QR; `BarcodeDetector` tidak tersedia di semua peramban (mis. Safari iOS) sehingga ada input manual, dan kamera membutuhkan HTTPS. Keputusan pemilik: menyetujui satu pustaka QR.
9. **Ekspor hanya `.xlsx`** (PDF menunggu persetujuan pustaka; spesifikasi lama menjanjikan PDF untuk daftar dan rekap). Tata letak belum dibuka di Excel/Numbers; awalan `'` tampak di sel ([02 BE bab 8 butir 12](../02_TIMELINE_DAN_TUGAS/BE.md#8-celah-dan-keputusan-yang-menunggu-pemilik-produk)).
10. **Impor lewat JSON base64**, bukan `multipart` (BFF dan `checkOrigin`, bab 2.6); BFF perlu batas badan khusus satu jalur. Alternatif: pengecualian `multipart` di BFF dengan pemeriksaan origin sendiri.
11. **Sel telepon numerik dari Excel ditolak, tidak ditebak** (sama dengan `NormalizeWA`); templat memformat kolom sebagai teks. Alternatif: menambahkan `0` pada angka 9–13 digit tanpa awalan (menebak).
12. **Duplikat hanya dilaporkan** dan tetap diimpor kecuali `skip_duplicates`; kuncinya nama + nomor (tamu tanpa nomor yang bernama sama dianggap duplikat, mungkin dua orang berbeda).
13. **Kategori kustom disimpan sebagai teks apa adanya:** "Kolega Kantor" dan "kolega kantor" menjadi dua kategori; `lainnya` adalah kunci cadangan, bukan kategori registry. Alternatif: menyamakan huruf besar-kecil saat menyimpan.
14. **`Pax` impor kosong = 1** dan `pax_count` wajib pada tambah manual; bawaan DDL 2 tidak dipakai.
15. **Kata "undangan" dan "pendaftaran":** registry hanya punya `Labels.Guest` dan `Labels.GuestMenu`; FE menurunkan "pendaftaran" dari `Labels.Guest = "Peserta"`. Alternatif: bidang ketiga di `Labels`.
16. **Satu check-in per tamu**; tamu yang hadir di dua sesi dicatat sebagai dua baris. Alternatif: tabel check-in per sesi.
17. **`delivered` hanya akan ditulis modul undangan**; Planner tidak punya cara mengetahuinya dari `wa.me`.
18. **Perubahan kontrak undangan:** `ApplyRSVP` berbasis token; kasus B mereset sesi ke `all`; FK undangan `SET NULL` dengan penghapusan tamu-saja-undangan lebih dulu ([bab 5](#5-kontrak-antarmuka-dengan-modul-undangan-digital)). Modul undangan belum punya spesifikasi, jadi ini kontrak minimum yang harus dipenuhinya.
19. **Tidak ada urut ulang manual** tamu; daftar urut `sort_order` (urutan dibuat), antrean meja urut nama.
20. **Banner Standalone tanpa tautan aksi** sampai modul undangan ada (versi sebelumnya punya tombol "Aktifkan Undangan Digital Adicara" yang tidak dapat berfungsi).
21. **`GET /guests/:guest_id`** dipakai klien hanya untuk mengambil ulang tamu sesudah `409 already_checked_in`.
22. **Pencarian substring memindai proyek** (≤ 5.000 baris); indeks `lower(name)` tidak membantu `strpos`. Kinerja pada 5.000 tamu belum diukur.
23. **Kinerja polling meja:** satu panggilan per 5 detik membawa satu halaman antrean (50) + KPI; belum diukur dengan beberapa tablet pada satu akun (batas `scan-qr` per pengguna, bukan per perangkat).
24. **Privasi akun:** tamu `project_id IS NULL` tidak terhapus lewat proyek; penghapusannya mengikuti FK undangan → pengguna, tanggung jawab modul undangan ([kontrak bab 2 butir 7](../00_KONTRAK_BERSAMA/README.md#2-isolasi-data-dan-keamanan)).
25. **Perubahan terhadap spesifikasi sebelumnya yang menyentuh dokumen lain** (semuanya di [CHANGELOG](../CHANGELOG.md)): kontrak (status, kode galat `guest_limit_reached`, batas badan BFF impor, matriks #8 dan #11, bab 10), TEMPLATE_EVENT 4.2 (label sisi dan kategori, baru), 01 FE bab 6 (pengecualian batas badan BFF), README folder (status).
26. **Constraint DDL yang disarankan** belum diterapkan: `pax_actual BETWEEN 0 AND 20` dan `is_checked_in = (checked_in_at IS NOT NULL)` (bab 1).
27. **Halaman `/i/[slug]` memeriksa `guest`, bukan `to`, untuk `noindex`**: tautan personal Planner memakai `to` dan `g`, jadi halaman personal dapat terindeks. Perbaikan milik modul undangan; kanonik halaman tetap tanpa query.
28. **Hub `guest_pax_count`** menjumlah pax semua tamu (diundang, termasuk penolak), bukan pax yang hadir; kartu menyebutnya "Tamu 43 pax".

Keputusan yang diambil saat membangun modul (2026-10-25; rinci di [dokumen modul](../../../backend/planner/2026-10-25-planner-menu-05.md)):

29. **Bentrok token tidak diulang** (rancangan awal menyebut maksimal 3 kali): di dalam transaksi pernyataan yang gagal membatalkan transaksi, jadi indeks unik menjawab `409 conflict` dan mengulang permintaan memberi token baru. Peluang bentrok pada 128 bit tidak terukur.
30. **Cap `rsvp_submitted_at` dibiarkan bila jawaban yang sama dikirim lagi** (`ApplyRSVP` harus idempoten tanpa perubahan; satu pernyataan SQL melayani `PATCH …/rsvp` dan `ApplyRSVP`). Alternatif: cap diperbarui setiap kali.
31. **`filters.categories` pada paket meja** memuat kategori registry (juga yang 0) dan, bila dipakai tamu yang diharapkan, `lainnya` dan teks kustom (`custom: true`); fixture acuan hanya mendaftar kategori registry (FE sudah menyaring "kustom hanya bila ada").
32. **Pembatas laju dihitung sebelum handler**, jadi permintaan yang gagal validasi ikut terhitung (impor 5/menit, unduhan 10/menit bersama, `scan-qr` 120/menit). Kode menyediakan `Limits` / `RegisterRoutesWithLimits` supaya tes yang memanggil puluhan kali dapat mematikannya; produksi memakai `DefaultLimits`.
33. **Badan lebih dari 64 KiB pada rute selain impor** dijawab `400 invalid_json` (sama dengan menu 02–04); `413 file_too_large` hanya untuk impor (berkas > 5 MiB setelah di-decode, atau badan > 7,5 MiB).
34. **Pesan tiga kode galat** mengikuti contoh katalog FE bab 4.6: `the guest is already checked in`, `at most 5000 guests per project`, `file too large`.
35. **Walk-in memakai jam aplikasi** (`checked_in_at`, `rsvp_submitted_at`), check-in biasa `now()` basis data; keduanya UTC.
36. **`dry_run` tidak mengunci proyek dan tidak memeriksa batas 5.000**; hanya impor sungguhan yang melakukannya (di dalam kunci). Pratinjau bisa berkata "siap" untuk berkas yang kemudian ditolak `409 guest_limit_reached`.
37. **Urutan nama antrean mengikuti kolasi basis data** (`ORDER BY lower(name), id`): diuji pada `en_US.UTF-8` (hasilnya sama dengan urutan byte pada data contoh); `postgres:18-alpine` (musl, urutan byte) belum dicoba dan dapat mengurutkan tanda baca dan spasi secara berbeda dari macOS atau glibc.
38. **`PUBLIC_BASE_URL` belum dibaca `internal/config`** dan `Options.Invitations` kosong di `app.go`: keduanya ditambahkan bersamaan dengan modul undangan (bab 5.1). Tanpa itu tidak ada tamu yang memiliki `personalized_url`.
