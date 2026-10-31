# Planner menu 06: rundown dan acara

Tanggal: 2026-10-31. Branch `backend/planner`. Spesifikasi: [PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/BE.md](../../APP_SPEC/PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/BE.md), [00_KONTRAK_BERSAMA](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/README.md), dan [TEMPLATE_EVENT](../../APP_SPEC/PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md) (bab 6–14). Kontrak API: [`backend/docs/openapi.yaml`](../../../backend/docs/openapi.yaml) (tag `Planner Rundown`). Dokumen menu lain yang disentuh modul ini: [menu 01](2026-10-21-planner-menu-01.md) (registry, widget, wizard, Proyek Contoh, hapus sesi), [menu 04](2026-10-23-planner-menu-04.md) (vendor yang menjadi PIC agenda; `NormalizeWA`), dan [migrasi 005–012](../migrations/2026-10-21-planner-migrations-005-012.md) (indeks unik yang ditambahkan ke 012).

Modul `plannerrundown` memiliki **susunan rundown menit demi menit per sesi** dengan status hari-H (belum, berjalan, selesai, dengan cap waktu), **playlist lagu** yang dapat ditautkan ke agenda dengan catatan cue, template tiap jenis acara (pernikahan: 33 agenda dan 30 lagu), dan **dua teks WhatsApp** (rundown dan briefing sound cue) yang disusun server. Seperti menu 02–05 ia tidak diimpor oleh `planner`; ia mendaftarkan diri lewat `SummaryProvider` (`rundown`) dan dua `TemplateApplier` (`rundown`, `playlist`), sehingga widget Rundown Hari-H di dashboard terisi dan wizard serta Proyek Contoh memuat 33 agenda dan 30 lagu. Ia **tidak punya overlay** (Proyek Contoh memuat template saja: semua `pending`, tanpa vendor dan tautan lagu) dan **tidak menulis tabel modul lain**.

## Yang ada

| Area | Isi |
|---|---|
| Agenda (10) | `GET/POST …/rundowns`, `GET …/rundowns/summary`, `GET …/rundowns/export-wa`, `POST …/rundowns/reorder`, `POST …/rundowns/apply-template`, `GET/PUT/DELETE …/rundowns/:rundown_id`, `PATCH …/rundowns/:rundown_id/status` |
| Playlist (7) | `GET/POST …/playlists`, `GET …/playlists/export-wa`, `POST …/playlists/reorder`, `POST …/playlists/apply-template`, `PUT/DELETE …/playlists/:playlist_id` |
| Registry | `GET /event-types` memuat `moment_tags` per jenis acara (TEMPLATE_EVENT bab 6) dan `templates_available.rundown` (semua jenis) serta `playlist` (pernikahan saja) |
| Template | pernikahan: Lamaran 11 + Akad 8 + Resepsi 14 agenda dan 30 lagu (PDF Hal. 14–16); delapan jenis lain: 3–9 agenda usulan (60 agenda; kunci `<jenis>.agenda_<NN>`) |
| Dashboard | provider `rundown` mengisi widget 6 (`RundownSummary`): jumlah sesi, agenda, selesai, berjalan, lagu, agenda berjalan paling awal, sesi bertanggal terdekat |
| Menu 01 | `ModeReplaceTemplate` dan `ApplyRequest.Confirm`; `EventTypeSpec.MomentTags` (`momentcatalog.go`) |

Total **17 endpoint**. Migrasi: `012_000_planner_rundown.sql` diubah **di tempat** untuk menambah `planner_rundowns_template_key_idx` dan `planner_playlists_template_key_idx` (unik parsial `(project_id, template_key)`); tanpa keduanya `append` ala menu 02–04 ditolak PostgreSQL. Database yang sudah menjalankan 012 lama harus `migrate fresh` atau menambahkan keduanya dengan tangan. Tidak ada perubahan frontend (BFF dan halaman Planner belum ada di repo).

## Struktur kode

```text
internal/modules/plannerrundown/
  domain/       entity.go (DTO bab 4), clock.go (ParseClock, ValidateTimes), overview.go (BuildKPI, PlaylistKPIOf, SongFilter),
                export.go (dua teks WhatsApp), template.go, rundown_wedding.go, rundown_general.go, playlist_wedding.go
                (data template, dibangkitkan dari tabel spesifikasi), repository.go (antarmuka, sentinel)
  usecase/      service.go, validate.go, agenda.go, playlist.go, template.go, export.go, integration.go (provider, dua applier)
  repository/   postgres.go
  delivery/http router.go, handler.go, ratelimit.go
internal/modules/planner/domain/repository.go     + ModeReplaceTemplate, ApplyRequest.Confirm
internal/modules/planner/domain/eventtype.go      + MomentTags
internal/modules/planner/domain/momentcatalog.go  data registry sembilan jenis (TEMPLATE_EVENT bab 6)
internal/migrate/sql/012_000_planner_rundown.sql  + dua indeks unik template_key
internal/app/app.go                               + pemasangan modul (provider, dua applier, rute)
```

Mengikuti menu 01–05: usecase tidak mengimpor `pgx`; transaksi dibawa lewat `context` oleh `TxManager` yang **sama** dengan milik `planner`. DTO tanpa tag `binding`, validasi di usecase dengan pesan `<field>: <alasan>`, daftar tidak pernah `null`, tidak ada persentase di respons, `ErrNotFound`/`ErrInvalidScope`/`ErrConflict` dipakai ulang dari `planner/domain`. Data template dan registry dijaga tes yang **membaca dokumen spesifikasi** (`TestWeddingTemplateMatchesSpec`, `TestWeddingPlaylistMatchesSpec`, `TestGeneralTemplateMatchesTemplateEvent`, `TestMomentTagsMatchTemplateEvent`): mengubah satu tanpa yang lain menggagalkan tes.

## Perilaku yang perlu diketahui

- **Rentang, bukan jumlah.** KPI waktu adalah rentang tiap sesi (`max(akhir) − min(mulai)`, dalam menit lewat `EXTRACT(EPOCH …)/60`), lalu dijumlah antar sesi: Proyek Contoh 100 + 120 + 225 = **445** menit (jumlah durasi 699). Aritmetika `time` tidak dipakai karena `'00:30'::time + interval '24 hours'` tetap 00:30; acara 22.00–23.00 dan 23.30–00.30 berentang 150 (`TestSessionSpanAcrossMidnight`). Agenda yang *mulai* sesudah tengah malam tidak dapat dimodelkan (BE bab 9 butir 3).
- **Aturan jam di usecase** (`ValidateTimes`): `07.46`, `7:46` dan sejenisnya disimpan `07:46`; detik, `24:00`, dan `07.6` ditolak; tanpa `end_time` agenda terbuka; `ends_next_day` mensyaratkan akhir lebih awal dari mulai. Basis data hanya menjaga satu dari aturan ini (`CHECK`).
- **Status satu pernyataan SQL.** `PATCH …/status` menulis status dan `started_at`/`completed_at` bersama: `in_progress` mengisi `started_at` sekali, `completed` mengisi keduanya, `pending` mengosongkan keduanya, status yang sama dikirim lagi tidak menggeser cap, membuka kembali agenda selesai mempertahankan jam mulainya. Semua perpindahan boleh dan beberapa agenda boleh berjalan bersamaan. `PUT` tidak menulis `status`. Cap waktu selalu UTC (repository mengubahnya; pgx mengembalikan zona proses).
- **Satu pernyataan, satu baris; kunci proyek hanya untuk menambah baris atau mengubah urutan.** Tambah agenda dan lagu, kedua `apply-template`, kedua `reorder`, dan `PUT` yang memindahkan agenda ke sesi lain mengambil `SELECT … FOR UPDATE` pada proyek; status, ubah, hapus, dan semua `GET` tidak menunggunya, sehingga layar hari-H tidak tertahan oleh template atau urut ulang. Dibuktikan dengan menahan kunci dari luar (`TestWritersTakeTheProjectLockAndTheLiveScreenDoesNot`), bukan dengan balapan yang kebetulan.
- **Urut ulang harus tepat semua id.** Daftar dengan id kembar, kurang, lebih, atau milik sesi lain ditolak `400 invalid_input: ids` dan tidak mengubah apa pun (SQL polos menerima id kembar tanpa galat). Urutan adalah `sort_order` per sesi (1…n).
- **Template.** `append` idempoten (`ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`); sesi yang sudah dihapus dilewati dengan satu notice `scope_missing` (`count` = agenda yang dilewati). `replace_template` (agenda saja) mengembalikan agenda template `pending` ke isi template **di tempat**: id, vendor, sesi, dan lagu tertaut tetap; yang berjalan atau selesai tidak disentuh (`kept`); agenda buatan pengguna dan agenda template yang dipindahkan ke sesi lain tidak disentuh. `reset` menghitung semua agenda template `pending` pada sesi template-nya dan `confirm: true` wajib bila `reset > 0` (`400 invalid_input: confirm: required`). Lagu hanya `append`.
- **Batas.** 500 agenda dan 500 lagu per proyek di bawah kunci proyek (tepat 500 lolos, serentak hanya satu yang lolos, template yang melewati batas gagal seluruhnya), `export-wa` 10/menit per pengguna untuk kedua ekspor bersama; `GET …/rundowns/summary` (polling 5 detik) tidak dibatasi.
- **Teks WhatsApp disusun server** (`domain/export.go`): rundown satu sesi atau semua sesi, dan briefing sound cue (lagu tertaut dikelompokkan per sesi dan agenda menurut jam, sisanya per *moment tag*). Teks Akad Proyek Contoh **sama** dengan `fixtures/planner-rundown-wa-akad.json`. Teks pengguna tidak di-escape (BE bab 9 butir 6). Tidak ada ekspor PDF; cetak lewat peramban adalah urusan frontend.
- **Tautan.** `vendor_id` dan `rundown_id` dijaga FK komposit `(…, project_id)` (`400 invalid_input` bila milik proyek lain); hapus vendor atau agenda hanya mengosongkan tautan; hapus sesi menghapus agendanya (`CASCADE`) dan Menu 01 menolaknya dengan `409 session_in_use` + `usage.agendas` kecuali `delete_agendas=true`.
- **Proyek arsip** baca-saja (`409 project_read_only`) di semua rute kecuali `GET` dan kedua ekspor; pengguna lain `404` di 17 rute; id yang bukan UUID kanonik `404`.

## Dependensi

Tidak ada yang baru.

## Penyimpangan dari spesifikasi dan celah yang ditemukan

Semuanya dicatat di [BE.md bab 9](../../APP_SPEC/PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/BE.md#9-celah-dan-keputusan-yang-menunggu-pemilik-produk) (butir 17–26) dan [CHANGELOG spesifikasi](../../APP_SPEC/PLANNER_SPEC/CHANGELOG.md).

1. **DTO tanpa tag `binding`.** Versi audit memakai `binding:"required,max=255"`, yang akan menjawab `400 invalid_json` tanpa nama field; tabel validasi menjanjikan `<field>: <alasan>`, jadi usecase yang memvalidasi (seperti menu 04 dan 05).
2. **`sort_order` per sesi (1…n).** Fixture `planner-rundowns.json` memberi agenda Resepsi 9–22 (nomor tabel PDF); dikoreksi menjadi 1–14.
3. **Lamaran Proyek Contoh bertanggal** (15 Okt 2026, dari overlay menu 02); fixture `planner-rundowns.json` dan `planner-rundown-live.json` mendapat `date` pada sesi itu dan `next_session` pada widget adalah Lamaran, bukan Akad Nikah (data tiruan audit tidak bertanggal).
4. **`reset` menghitung semua agenda template `pending`**, bukan hanya yang berbeda dari template (sama dengan `replaceImpact` di FE); agenda template yang dipindahkan pengguna dihitung `skipped`.
5. **Baris lokasi teks WhatsApp:** `location_name` sesi menggantikan `venue_name` proyek (kota tetap ikut); keterangan agenda berbaris-baris dilipat menjadi satu baris.
6. **`PUT` yang memindahkan agenda ke sesi lain** menaruhnya paling akhir di sesi itu dan mengambil kunci proyek.
7. **Provider dashboard** menerima "hari ini" dari jam usecase (bukan `now()` basis data) supaya tes dapat menetapkannya.
8. **Satu mutan setara:** applier wizard tidak memakai `ApplyRequest.Mode` (selalu `append`, seperti applier menu 02–04); mengubahnya ke `replace_template` tak terlihat pada proyek baru, jadi tidak dapat dibedakan lewat HTTP dan tidak diuji.

## Pengujian

Dijalankan 2026-10-31 pada PostgreSQL 16.2 lokal, database sementara `adicara_rundown06_test` (dibuat khusus dan dihapus setelahnya; database `adicara` tidak disentuh), koneksi pgx mode simple-protocol seperti produksi, `go test -p 1`:

| Lapisan | Hasil |
|---|---|
| `go test -p 1 -count=1 ./...` | **520 tes tingkat atas lulus, 0 gagal, 0 di-skip** (448 sebelumnya + 72 baru: 44 HTTP, 16 domain, 9 usecase, 2 repository, 1 migrasi) |
| `-race` pada `plannerrundown/...` | lulus, tanpa `DATA RACE` |
| Cakupan modul (`-coverpkg`, gabungan semua paket tes) | 94,9% pernyataan; sisanya cabang galat yang hanya terjadi bila basis data gagal |
| `go vet ./...`, `gofmt -l` | bersih |
| `redocly lint backend/docs/openapi.yaml` | valid; 3 peringatan lama (lisensi, `localhost`, `/healthz` tanpa 4XX), tidak ada yang baru |
| Kontrak | `TestOpenAPIDocumentsEveryRundownRoute`: 17 rute terdaftar = 17 operasi bertag `Planner Rundown`; `TestOpenAPISchemasMatchTheGoDTOs`: 20 skema sama dengan tag JSON struct Go (kunci, dan untuk respons kewajiban hadir); `TestOpenAPIEventTypeHasMomentTags` |
| Mutasi | **40 mutasi kode produksi, 39 digagalkan tes, 1 setara** (applier wizard memakai `replace_template`; pada proyek baru hasilnya sama dengan `append`). Putaran pertama (33) meninggalkan tiga yang selamat dan satu yang tak terkompilasi; semuanya menambah atau memperbaiki tes: PIC unik (agenda vendor dengan teks PIC berbeda), urutan template yang dikembalikan `replace_template`, dan kunci proyek (balapan serentak tidak membuktikannya; diganti tes yang menahan kunci dari luar, lalu delapan mutan "tanpa kunci" per operasi dan satu "status menunggu kunci" semuanya tertangkap) |
| Universalitas | `TestEveryEventTypeGetsItsRundown`: sembilan jenis acara dari registry menjalani alur yang sama lewat router, PostgreSQL, dan wiring asli (template dalam sesi seed registry, kunci, `replace_template` pada rundown yang belum disentuh, kedua teks WhatsApp tanpa kata khas pernikahan, lagu dengan setiap moment tag registry, widget dashboard, status, urut ulang, hapus sesi dengan `delete_agendas`; satu pengguna per jenis karena ekspor dibatasi per pengguna); `TestEventTypesCarryMomentTagsAndTemplates` |
| Dokumen ↔ kode | tabel BE bab 5.2–5.4 = data template, TEMPLATE_EVENT bab 6–14 = registry dan template delapan jenis; tes **dijalankan dengan `-v`** dan tidak di-skip |
| **Respons HTTP asli = fixture FE** | `TestSampleProjectGoldenRundown` (daftar agenda dan playlist), `TestHariHLiveAndWhatsAppMatchTheFixtures` (status hari-H dan teks Akad), `TestErrorCatalogueMatchesTheServer` (17 galat: status, kode, dan pesan; gagal untuk entri yang tak dapat dipancing dan untuk pemancing yang tak ada di katalog) sama dengan lima fixture di `PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/fixtures/` setelah id dan waktu dinormalkan; dua fixture dikoreksi (butir 2 dan 3 di atas) |
| Tanpa `TEST_DATABASE_URL` | tes domain, usecase murni, repository murni (`mapErr`), pembatas laju, dan tes yang membaca dokumen lulus; tes DB di-skip |

Yang dibuktikan (pilihan): uji emas Proyek Contoh lewat seluruh tumpukan (33 agenda, rentang 100/120/225 = 445, 17 PIC unik, durasi Akad 20/4/9/9/4/9/9/49, dua agenda terbuka tanpa `end_time`, 30 lagu dengan 6 momen dan 30 `needs_review`, widget dashboard 3/33/30 dan "Ijab Kabul · Akad Nikah" saat berjalan); sesi lewat tengah malam berentang 150; status `pending → in_progress → completed → in_progress → pending` dengan cap yang tidak bergeser; delapan urut ulang serentak menghasilkan `sort_order` 1…8 yang berbeda; empat `apply-template` serentak menghasilkan tepat 33 agenda; 500 agenda dan 500 lagu tepat (serentak hanya satu lolos; template pada 480 gagal seluruhnya); `replace_template` mempertahankan id, vendor, dan satu lagu tertaut, tidak menyentuh agenda berjalan, selesai, atau buatan sendiri, dan mengembalikan urutan template; tujuh penulis menunggu kunci proyek dan tujuh pembaca atau penulis satu pernyataan tidak; `export-wa` 10/menit (kedua ekspor bersama) dan 60 polling `summary` berturut-turut tanpa batas; pengguna lain `404` di 17 rute; proyek arsip menjawab `409 project_read_only` di 11 rute tulis dan tetap `200` di 6 rute baca.

Menjalankan sendiri (membutuhkan database `…_test` yang **dihapus isinya**):

```bash
createdb adicara_planner_test
cd backend
TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/adicara_planner_test?sslmode=disable' go test -p 1 ./...
```

## Belum diuji atau belum ada

- **Frontend menu 06**: tidak disentuh. Tipe TypeScript di [FE.md](../../APP_SPEC/PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/FE.md) cocok dengan DTO Go dan fixture-nya sama dengan respons asli, tetapi belum ada satu baris pun kode halaman. **BFF Planner** belum ada di repo.
- **Kinerja**: daftar tanpa paginasi pada 500 agenda atau 500 lagu, dan polling `summary` dari beberapa tablet pada satu akun, tidak diukur; hanya kebenarannya yang diuji.
- **Teks WhatsApp belum dibuka di WhatsApp:** tanda `*` dan `_` dalam teks pengguna dapat memformat teks (tidak di-escape; BE bab 9 butir 6), dan panjang pesan untuk rundown besar (33 agenda dengan keterangan) tidak diperiksa terhadap batas WhatsApp.
- **Agenda yang mulai sesudah tengah malam** tidak dapat dimodelkan (rentang salah); hanya agenda yang *berakhir* sesudahnya yang didukung (BE bab 9 butir 3).
- **Database yang sudah menjalankan 012 lama** (tanpa dua indeks) tidak diperbaiki otomatis; `migrate fresh` atau dua `CREATE UNIQUE INDEX` dengan tangan.
- Image Docker, PostgreSQL 18 (versi Compose), Nginx di depan rute baru, beberapa replika API (pembatas laju per proses).
