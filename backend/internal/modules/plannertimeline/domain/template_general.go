// Code generated from TEMPLATE_EVENT chapters 2 and 7-14 by a one-off script; edit freely afterwards, TestTemplatesMatchSpec keeps the two in step.

package domain

import pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"

// generalTasks are the 15 tasks every non-wedding type starts from (R = 60).
var generalTasks = []TaskTemplate{
	{Key: "general.tetapkan_tujuan_tanggal_dan_panitia_inti", Title: "Tetapkan tujuan, tanggal, dan panitia inti", Scope: "all", Category: "Perencanaan", Phase: 1, Start: 60, End: 58},
	{Key: "general.tetapkan_anggaran_total", Title: "Tetapkan anggaran total", Scope: "all", Category: "Keuangan", Phase: 1, Start: 58, End: 56},
	{Key: "general.survei_dan_pilih_venue", Title: "Survei dan pilih venue", Scope: "all", Category: "Venue", Phase: 1, Start: 56, End: 50},
	{Key: "general.susun_daftar_tamu_peserta_awal", Title: "Susun daftar tamu/peserta awal", Scope: "all", Category: "Tamu", Phase: 1, Start: 52, End: 48},
	{Key: "general.susun_konsep_dan_rundown_awal", Title: "Susun konsep dan rundown awal", Scope: "all", Category: "Acara", Phase: 1, Start: 48, End: 44},
	{Key: "general.riset_dan_bandingkan_vendor", Title: "Riset dan bandingkan vendor", Scope: "all", Category: "Vendor", Phase: 2, Start: 46, End: 38},
	{Key: "general.urus_perizinan_dan_administrasi", Title: "Urus perizinan dan administrasi", Scope: "all", Category: "Administrasi", Phase: 2, Start: 44, End: 30},
	{Key: "general.kunci_vendor_utama_dan_bayar_dp", Title: "Kunci vendor utama dan bayar DP", Scope: "all", Category: "Vendor", Phase: 2, Start: 38, End: 34},
	{Key: "general.siapkan_undangan_dan_publikasi", Title: "Siapkan undangan dan publikasi", Scope: "all", Category: "Undangan", Phase: 3, Start: 34, End: 28},
	{Key: "general.sebarkan_undangan", Title: "Sebarkan undangan", Scope: "all", Category: "Undangan", Phase: 3, Start: 28, End: 26},
	{Key: "general.belanja_perlengkapan_dan_logistik", Title: "Belanja perlengkapan dan logistik", Scope: "all", Category: "Logistik", Phase: 3, Start: 26, End: 14},
	{Key: "general.follow_up_konfirmasi_kehadiran_rsvp", Title: "Follow-up konfirmasi kehadiran (RSVP)", Scope: "all", Category: "Tamu", Phase: 3, Start: 14, End: 10},
	{Key: "general.technical_meeting_vendor_dan_panitia", Title: "Technical meeting vendor dan panitia", Scope: "all", Category: "Panitia", Phase: 4, Start: 7, End: 6},
	{Key: "general.gladi_resik", Title: "Gladi resik", Scope: "all", Category: "Acara", Phase: 4, Start: 1, End: 0},
	{Key: "general.hari_h_acara", Title: "Hari-H acara", Scope: "all", Category: "Acara", Phase: 4, Start: 0, End: 0},
}

// typeTasks are the three extra tasks per type, merged into generalTasks chronologically.
var typeTasks = map[pd.EventType][]TaskTemplate{
	pd.EventKhitanan: {
		{Key: "khitanan.booking_dokter_tenaga_medis", Title: "Booking dokter/tenaga medis", Scope: "all", Category: "Umum", Phase: 2, Start: 40, End: 38},
		{Key: "khitanan.pesan_konsumsi_syukuran", Title: "Pesan konsumsi syukuran", Scope: "all", Category: "Umum", Phase: 3, Start: 20, End: 18},
		{Key: "khitanan.siapkan_busana_khitan_dan_perlengkapan_doa", Title: "Siapkan busana khitan dan perlengkapan doa", Scope: "all", Category: "Umum", Phase: 3, Start: 18, End: 14},
	},
	pd.EventAqiqah: {
		{Key: "aqiqah.tentukan_hari_pelaksanaan_dan_ustadz", Title: "Tentukan hari pelaksanaan dan ustadz", Scope: "all", Category: "Umum", Phase: 3, Start: 18, End: 16},
		{Key: "aqiqah.pesan_kambing_dan_katering_aqiqah", Title: "Pesan kambing dan katering aqiqah", Scope: "all", Category: "Umum", Phase: 3, Start: 30, End: 26},
		{Key: "aqiqah.siapkan_dokumen_kelahiran_dan_nama", Title: "Siapkan dokumen kelahiran dan nama", Scope: "all", Category: "Umum", Phase: 2, Start: 40, End: 36},
	},
	pd.EventPeresmian: {
		{Key: "peresmian.koordinasi_izin_lingkungan_pengelola_gedung", Title: "Koordinasi izin lingkungan/pengelola gedung", Scope: "all", Category: "Umum", Phase: 2, Start: 44, End: 36},
		{Key: "peresmian.siapkan_prosesi_gunting_pita_dan_daftar_vip", Title: "Siapkan prosesi gunting pita dan daftar VIP", Scope: "all", Category: "Umum", Phase: 3, Start: 30, End: 24},
		{Key: "peresmian.undang_media_dan_mitra", Title: "Undang media dan mitra", Scope: "all", Category: "Umum", Phase: 3, Start: 28, End: 20},
	},
	pd.EventLaunching: {
		{Key: "launching.finalisasi_materi_produk_dan_keynote", Title: "Finalisasi materi produk dan keynote", Scope: "all", Category: "Umum", Phase: 2, Start: 40, End: 30},
		{Key: "launching.undang_media_pers_dan_kreator", Title: "Undang media/pers dan kreator", Scope: "all", Category: "Umum", Phase: 3, Start: 30, End: 20},
		{Key: "launching.uji_teknis_demo_dan_av", Title: "Uji teknis demo dan AV", Scope: "all", Category: "Umum", Phase: 4, Start: 6, End: 4},
	},
	pd.EventBirthday: {
		{Key: "birthday.tentukan_tema_dan_dresscode", Title: "Tentukan tema dan dresscode", Scope: "all", Category: "Umum", Phase: 1, Start: 52, End: 46},
		{Key: "birthday.pesan_kue_dan_dekorasi", Title: "Pesan kue dan dekorasi", Scope: "all", Category: "Umum", Phase: 3, Start: 30, End: 24},
		{Key: "birthday.siapkan_playlist_dan_games", Title: "Siapkan playlist dan games", Scope: "all", Category: "Umum", Phase: 3, Start: 14, End: 10},
	},
	pd.EventWisuda: {
		{Key: "wisuda.koordinasi_senat_fakultas", Title: "Koordinasi senat/fakultas", Scope: "all", Category: "Umum", Phase: 1, Start: 56, End: 48},
		{Key: "wisuda.alokasi_toga_dan_tiket_keluarga", Title: "Alokasi toga dan tiket keluarga", Scope: "all", Category: "Umum", Phase: 3, Start: 34, End: 26},
		{Key: "wisuda.gladi_prosesi", Title: "Gladi prosesi", Scope: "all", Category: "Umum", Phase: 4, Start: 3, End: 2},
	},
	pd.EventSeminar: {
		{Key: "seminar.undang_dan_konfirmasi_pembicara", Title: "Undang dan konfirmasi pembicara", Scope: "all", Category: "Umum", Phase: 1, Start: 58, End: 50},
		{Key: "seminar.buka_registrasi_peserta", Title: "Buka registrasi peserta", Scope: "all", Category: "Umum", Phase: 1, Start: 50, End: 48},
		{Key: "seminar.siapkan_materi_sertifikat_dan_kit", Title: "Siapkan materi, sertifikat, dan kit", Scope: "all", Category: "Umum", Phase: 3, Start: 24, End: 12},
	},
}

// typeDocs are the permit documents per type (quantity "1 berkas", cost 0).
var typeDocs = map[pd.EventType][]DocTemplate{
	pd.EventKhitanan: {
		{Key: "khitanan.surat_izin_lingkungan_rt_rw_opsional", Name: "Surat izin lingkungan RT/RW (opsional)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "khitanan.konfirmasi_dokter_klinik", Name: "Konfirmasi dokter/klinik", Subject: "panitia", Quantity: "1 berkas"},
		{Key: "khitanan.pemesanan_tenda_kursi", Name: "Pemesanan tenda/kursi", Subject: "panitia", Quantity: "1 berkas"},
	},
	pd.EventAqiqah: {
		{Key: "aqiqah.akta_surat_keterangan_lahir", Name: "Akta/surat keterangan lahir", Subject: "umum", Quantity: "1 berkas"},
		{Key: "aqiqah.surat_izin_lingkungan_opsional", Name: "Surat izin lingkungan (opsional)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "aqiqah.konfirmasi_rumah_potong_penyedia_kambing", Name: "Konfirmasi rumah potong/penyedia kambing", Subject: "panitia", Quantity: "1 berkas"},
	},
	pd.EventPeresmian: {
		{Key: "peresmian.izin_keramaian_lingkungan_rt_rw_kelurahan_polsek", Name: "Izin keramaian/lingkungan (RT/RW, kelurahan, Polsek)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "peresmian.izin_pengelola_gedung_venue", Name: "Izin pengelola gedung/venue", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "peresmian.izin_usaha_nib_siup_bila_relevan", Name: "Izin usaha (NIB/SIUP, bila relevan)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "peresmian.surat_pemberitahuan_ke_dinas_terkait", Name: "Surat pemberitahuan ke Dinas terkait", Subject: "lembaga", Quantity: "1 berkas"},
	},
	pd.EventLaunching: {
		{Key: "launching.izin_venue_pengelola", Name: "Izin venue/pengelola", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "launching.perjanjian_media_partner", Name: "Perjanjian media partner", Subject: "panitia", Quantity: "1 berkas"},
		{Key: "launching.izin_keramaian_bila_massal", Name: "Izin keramaian (bila massal)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "launching.lisensi_musik_hak_cipta_lmkn_bila_memutar_musik", Name: "Lisensi musik/hak cipta (LMKN) bila memutar musik", Subject: "lembaga", Quantity: "1 berkas"},
	},
	pd.EventBirthday: {
		{Key: "birthday.izin_venue_pengelola_bila_di_luar_rumah", Name: "Izin venue/pengelola (bila di luar rumah)", Subject: "lembaga", Quantity: "1 berkas"},
	},
	pd.EventWisuda: {
		{Key: "wisuda.izin_penggunaan_gedung_auditorium", Name: "Izin penggunaan gedung/auditorium", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "wisuda.daftar_wisudawan_dari_akademik", Name: "Daftar wisudawan dari akademik", Subject: "panitia", Quantity: "1 berkas"},
		{Key: "wisuda.izin_dokumentasi", Name: "Izin dokumentasi", Subject: "umum", Quantity: "1 berkas"},
	},
	pd.EventSeminar: {
		{Key: "seminar.kontrak_venue", Name: "Kontrak venue", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "seminar.perjanjian_pembicara", Name: "Perjanjian pembicara", Subject: "panitia", Quantity: "1 berkas"},
		{Key: "seminar.izin_keramaian_bila_500_orang", Name: "Izin keramaian (bila > 500 orang)", Subject: "lembaga", Quantity: "1 berkas"},
		{Key: "seminar.lisensi_musik_lmkn_bila_relevan", Name: "Lisensi musik (LMKN) bila relevan", Subject: "lembaga", Quantity: "1 berkas"},
	},
}
