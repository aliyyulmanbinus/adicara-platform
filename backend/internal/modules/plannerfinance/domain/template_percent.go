// Code generated from PLANNER_SPEC (TEMPLATE_EVENT section 15.1) by a one-off script; edit freely afterwards,
// TestTemplatesMatchSpec keeps the two in step.

package domain

import pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"

// percentTemplates are the budget lines of the eight non-wedding event types as a share of the budget; every type adds up to 100.
var percentTemplates = map[pd.EventType][]PercentLine{
	pd.EventKhitanan: {
		{Key: "konsumsi", Name: "Konsumsi/katering", Category: "Catering", Percent: 40},
		{Key: "medis", Name: "Medis (dokter, perlengkapan)", Category: "Medis", Percent: 15},
		{Key: "dekorasi_tenda", Name: "Dekorasi dan tenda", Category: "Dekorasi", Percent: 12},
		{Key: "busana_mua", Name: "Busana dan MUA", Category: "Busana & MUA", Percent: 8},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 8},
		{Key: "undangan_souvenir", Name: "Undangan dan souvenir", Category: "Perlengkapan", Percent: 7},
		{Key: "hiburan", Name: "Pengajian/hiburan", Category: "Hiburan", Percent: 5},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 5},
	},
	pd.EventAqiqah: {
		{Key: "kambing", Name: "Kambing dan pengolahan", Category: "Catering", Percent: 45},
		{Key: "katering_box", Name: "Katering/box nasi", Category: "Catering", Percent: 25},
		{Key: "dekorasi_tempat", Name: "Dekorasi dan tempat", Category: "Venue & Dekorasi", Percent: 8},
		{Key: "undangan_souvenir", Name: "Undangan dan souvenir", Category: "Perlengkapan", Percent: 8},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 5},
		{Key: "ustadz_doa", Name: "Ustadz/doa", Category: "Prosesi", Percent: 4},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 5},
	},
	pd.EventPeresmian: {
		{Key: "venue_dekor", Name: "Venue dan dekorasi", Category: "Venue & Dekorasi", Percent: 20},
		{Key: "katering", Name: "Katering", Category: "Catering", Percent: 20},
		{Key: "av_panggung", Name: "Audio visual dan panggung", Category: "Audio Visual", Percent: 15},
		{Key: "publikasi_media", Name: "Publikasi dan media", Category: "Publikasi", Percent: 15},
		{Key: "prosesi_vip", Name: "Prosesi peresmian dan VIP", Category: "Prosesi", Percent: 10},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 8},
		{Key: "perizinan_keamanan", Name: "Perizinan dan keamanan", Category: "Legalitas", Percent: 5},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 7},
	},
	pd.EventLaunching: {
		{Key: "produksi_av", Name: "Produksi konten dan AV", Category: "Audio Visual", Percent: 25},
		{Key: "venue", Name: "Venue", Category: "Venue & Dekorasi", Percent: 20},
		{Key: "publikasi_pr", Name: "Publikasi/PR", Category: "Publikasi", Percent: 18},
		{Key: "katering", Name: "Katering", Category: "Catering", Percent: 12},
		{Key: "talent_mc", Name: "Talent dan MC", Category: "Hiburan", Percent: 8},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 7},
		{Key: "merchandise_kit", Name: "Merchandise/kit", Category: "Perlengkapan", Percent: 5},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 5},
	},
	pd.EventBirthday: {
		{Key: "katering_kue", Name: "Katering dan kue", Category: "Catering", Percent: 30},
		{Key: "venue", Name: "Venue", Category: "Venue & Dekorasi", Percent: 20},
		{Key: "dekorasi", Name: "Dekorasi", Category: "Venue & Dekorasi", Percent: 15},
		{Key: "hiburan", Name: "Hiburan (DJ/MC)", Category: "Hiburan", Percent: 15},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 8},
		{Key: "souvenir", Name: "Souvenir", Category: "Perlengkapan", Percent: 7},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 5},
	},
	pd.EventWisuda: {
		{Key: "venue_dekor", Name: "Venue dan dekorasi", Category: "Venue & Dekorasi", Percent: 25},
		{Key: "katering", Name: "Katering", Category: "Catering", Percent: 25},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 15},
		{Key: "toga_atribut", Name: "Toga dan atribut", Category: "Perlengkapan", Percent: 10},
		{Key: "audio_paduan_suara", Name: "Audio dan paduan suara", Category: "Audio Visual", Percent: 10},
		{Key: "cetak", Name: "Cetak (buku, sertifikat)", Category: "Perlengkapan", Percent: 8},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 7},
	},
	pd.EventSeminar: {
		{Key: "venue", Name: "Venue", Category: "Venue & Dekorasi", Percent: 25},
		{Key: "katering_coffee_break", Name: "Katering dan coffee break", Category: "Catering", Percent: 20},
		{Key: "honor_narasumber", Name: "Honor narasumber", Category: "Honor", Percent: 15},
		{Key: "av_streaming", Name: "AV dan streaming", Category: "Audio Visual", Percent: 12},
		{Key: "kit_sertifikat", Name: "Kit dan sertifikat", Category: "Perlengkapan", Percent: 8},
		{Key: "publikasi", Name: "Publikasi", Category: "Publikasi", Percent: 8},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 5},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 7},
	},
	pd.EventOther: {
		{Key: "venue", Name: "Venue", Category: "Venue & Dekorasi", Percent: 30},
		{Key: "katering", Name: "Katering", Category: "Catering", Percent: 30},
		{Key: "dekorasi", Name: "Dekorasi", Category: "Venue & Dekorasi", Percent: 10},
		{Key: "hiburan", Name: "Hiburan", Category: "Hiburan", Percent: 10},
		{Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Percent: 8},
		{Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Percent: 12},
	},
}
