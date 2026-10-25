package domain

// The guest sides and categories of the registry (menu 05). Keys are TEMPLATE_EVENT 4, labels 4.2; wedding follows
// Wedding_Planner.pdf p. 13 ("Teman Orang Tua"), the other eight types are Adicara proposals. TestGuestRegistryMatchesTemplateEvent
// (plannerguest/domain) reads that document and fails when this file drifts. The fallback category "lainnya" is deliberately
// not here: it is what a guest gets without a category (05 BE 2.8).

var guestSides = map[EventType][]Option{
	EventWedding:   {{"pria", "Pihak Pria (CPP)"}, {"wanita", "Pihak Wanita (CPW)"}},
	EventKhitanan:  {{"keluarga_ayah", "Keluarga Ayah"}, {"keluarga_ibu", "Keluarga Ibu"}},
	EventAqiqah:    {{"keluarga_ayah", "Keluarga Ayah"}, {"keluarga_ibu", "Keluarga Ibu"}},
	EventPeresmian: {{"internal", "Tuan Rumah/Internal"}, {"eksternal", "Undangan Eksternal"}},
	EventLaunching: {{"internal", "Internal"}, {"eksternal", "Eksternal"}},
	EventSeminar:   {{"internal", "Panitia/Internal"}, {"eksternal", "Peserta Eksternal"}},
	// birthday, wisuda and other have no sides: the guest list is not split.
}

var guestCategories = map[EventType][]Option{
	EventWedding: {{"teman", "Teman"}, {"kolega", "Kolega"}, {"keluarga", "Keluarga"}, {"tetangga", "Tetangga"}, {"teman_ortu", "Teman Orang Tua"}, {"vip", "VIP"}},
	EventKhitanan: {
		{"keluarga", "Keluarga"}, {"tetangga", "Tetangga"}, {"teman", "Teman"}, {"tokoh_agama", "Tokoh Agama"}, {"vip", "VIP"},
	},
	EventAqiqah: {{"keluarga", "Keluarga"}, {"tetangga", "Tetangga"}, {"teman", "Teman"}, {"tokoh_agama", "Tokoh Agama"}},
	EventPeresmian: {
		{"pimpinan_vip", "Pimpinan/VIP"}, {"mitra", "Mitra"}, {"media", "Media"}, {"karyawan", "Karyawan"},
		{"warga_sekitar", "Warga Sekitar"}, {"umum", "Umum"},
	},
	EventLaunching: {{"media", "Media"}, {"kreator", "Kreator"}, {"mitra", "Mitra"}, {"klien", "Klien"}, {"vip", "VIP"}, {"umum", "Umum"}},
	EventBirthday: {
		{"keluarga", "Keluarga"}, {"teman_sekolah", "Teman Sekolah"}, {"teman_kuliah", "Teman Kuliah"}, {"komunitas", "Komunitas"}, {"vip", "VIP"},
	},
	EventWisuda:  {{"wisudawan", "Wisudawan"}, {"orang_tua", "Orang Tua"}, {"dosen_pejabat", "Dosen/Pejabat"}, {"tamu_undangan", "Tamu Undangan"}, {"vip", "VIP"}},
	EventSeminar: {{"pembicara", "Pembicara"}, {"peserta", "Peserta"}, {"sponsor", "Sponsor"}, {"media", "Media"}, {"vip", "VIP"}},
	EventOther:   {{"umum", "Umum"}, {"vip", "VIP"}},
}

// The accessors hand out copies, so a caller cannot change the shared tables. A type without sides gets an empty,
// non-nil slice (GET /event-types sends [] and not null).

func guestSidesFor(t EventType) []Option { return append([]Option{}, guestSides[t]...) }

func guestCategoriesFor(t EventType) []Option { return append([]Option{}, guestCategories[t]...) }
