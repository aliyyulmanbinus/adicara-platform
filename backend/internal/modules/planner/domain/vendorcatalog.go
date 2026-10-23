package domain

// The vendor and shopping data of the registry (menu 04). Wedding follows
// Wedding_Planner.pdf p. 9-10 and 12 (04 BE 5.1); the other eight types are
// Adicara proposals (TEMPLATE_EVENT 7-14 for the categories, 15.2 for the
// budget_item_keys, 15.4 for the rollup line). TestVendorCategoriesMatchTemplateEvent
// (plannervendor/domain) reads those documents and fails when this file drifts.

func vc(key, label string, budgetItemKeys ...string) VendorCategory {
	return VendorCategory{Key: key, Label: label, BudgetItemKeys: budgetItemKeys}
}

// vendorCategories: the order is the order of the checklist. A label for a key that the
// TEMPLATE_EVENT bullet writes without one ("venue", "dekorasi", ...) is the plain word.
var vendorCategories = map[EventType][]VendorCategory{
	EventWedding: {
		vc("venue", "Venue", "venue_dekor"), vc("dekorasi", "Decoration", "venue_dekor"), vc("wo", "Wedding Organizer"),
		vc("mua", "MUA", "rias_busana"), vc("catering", "Catering", "catering"), vc("fotografi", "Photography", "dokumentasi"),
		vc("attire", "Attire", "rias_busana"), vc("hand_bouquet", "Hand Bouquet"), vc("mc", "MC", "hiburan"),
		vc("entertainment", "Entertainment", "hiburan"), // the PDF spells it "Entertaiment"
	},
	EventKhitanan: {
		vc("katering", "Katering", "konsumsi"), vc("medis", "Dokter/Klinik", "medis"), vc("dekorasi", "Dekorasi/Tenda", "dekorasi_tenda"),
		vc("dokumentasi", "Dokumentasi", "dokumentasi"), vc("hiburan", "Hadroh/Hiburan", "hiburan"),
	},
	EventAqiqah: {
		vc("kambing", "Penyedia Kambing", "kambing"), vc("katering", "Katering/Box Nasi", "katering_box"),
		vc("dekorasi", "Dekorasi", "dekorasi_tempat"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
	},
	EventPeresmian: {
		vc("venue", "Venue", "venue_dekor"), vc("katering", "Katering", "katering"), vc("av", "Audio Visual", "av_panggung"),
		vc("dekorasi", "Dekorasi", "venue_dekor"), vc("dokumentasi", "Dokumentasi", "dokumentasi"), vc("mc", "MC", "prosesi_vip"),
		vc("keamanan", "Keamanan", "perizinan_keamanan"),
	},
	EventLaunching: {
		vc("venue", "Venue", "venue"), vc("av", "Audio Visual", "produksi_av"), vc("pr", "PR/Publikasi", "publikasi_pr"),
		vc("katering", "Katering", "katering"), vc("talent", "Talent/MC", "talent_mc"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
		vc("produksi_konten", "Produksi Konten", "produksi_av"),
	},
	EventBirthday: {
		vc("venue", "Venue", "venue"), vc("katering", "Katering", "katering_kue"), vc("kue", "Kue", "katering_kue"),
		vc("dekorasi", "Dekorasi", "dekorasi"), vc("hiburan", "Hiburan", "hiburan"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
	},
	EventWisuda: {
		vc("venue", "Venue", "venue_dekor"), vc("katering", "Katering", "katering"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
		vc("audio", "Audio", "audio_paduan_suara"), vc("percetakan", "Percetakan", "cetak"), vc("toga", "Penyedia Toga", "toga_atribut"),
	},
	EventSeminar: {
		vc("venue", "Venue", "venue"), vc("katering", "Katering", "katering_coffee_break"), vc("av", "Audio Visual", "av_streaming"),
		vc("streaming", "Live Streaming", "av_streaming"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
		vc("percetakan", "Percetakan", "kit_sertifikat"),
	},
	EventOther: {
		vc("venue", "Venue", "venue"), vc("katering", "Katering", "katering"), vc("dekorasi", "Dekorasi", "dekorasi"),
		vc("hiburan", "Hiburan", "hiburan"), vc("dokumentasi", "Dokumentasi", "dokumentasi"),
	},
}

var shoppingCategories = map[EventType][]string{
	EventWedding:   {"Perangkat Alat Solat", "Make Up & Skin Care", "Peralatan Mandi", "Pakaian Dalam", "Pakaian Luar", "Kebutuhan Lain"},
	EventKhitanan:  {"Busana dan perlengkapan", "Souvenir", "Konsumsi"},
	EventAqiqah:    {"Box dan kemasan", "Souvenir", "Perlengkapan acara"},
	EventPeresmian: {"Perlengkapan prosesi (pita, gunting, plakat)", "Cetak (spanduk, backdrop)", "Souvenir"},
	EventLaunching: {"Press kit", "Merchandise", "Cetak (backdrop, standee)"},
	EventBirthday:  {"Dekorasi dan balon", "Souvenir/goodie bag", "Aksesori photobooth"},
	EventWisuda:    {"Toga dan atribut", "Cetak", "Bunga/hadiah"},
	EventSeminar:   {"Seminar kit", "ID card dan lanyard", "Plakat/sertifikat", "Cetak (banner, backdrop)"},
	EventOther:     {"Perlengkapan", "Cetak", "Konsumsi"},
}

var shoppingRollups = map[EventType]ShoppingRollup{
	EventWedding:   {"seserahan", "Seserahan", "Hantaran"},
	EventKhitanan:  {"undangan_souvenir", "Undangan dan souvenir", "Perlengkapan"},
	EventAqiqah:    {"undangan_souvenir", "Undangan dan souvenir", "Perlengkapan"},
	EventPeresmian: {"prosesi_vip", "Prosesi peresmian dan VIP", "Prosesi"},
	EventLaunching: {"merchandise_kit", "Merchandise/kit", "Perlengkapan"},
	EventBirthday:  {"souvenir", "Souvenir", "Perlengkapan"},
	EventWisuda:    {"toga_atribut", "Toga dan atribut", "Perlengkapan"},
	EventSeminar:   {"kit_sertifikat", "Kit dan sertifikat", "Perlengkapan"},
	EventOther:     {"perlengkapan", "Perlengkapan", "Perlengkapan"}, // no 15.1 line fits: created when missing
}

// The accessors hand out copies, so a caller cannot change the shared tables.

func vendorCategoriesFor(t EventType) []VendorCategory {
	src := vendorCategories[t]
	out := make([]VendorCategory, len(src))
	for i, c := range src {
		c.BudgetItemKeys = append([]string{}, c.BudgetItemKeys...)
		out[i] = c
	}

	return out
}

func shoppingCategoriesFor(t EventType) []string {
	return append([]string{}, shoppingCategories[t]...)
}

func shoppingRollupFor(t EventType) ShoppingRollup { return shoppingRollups[t] }
