package domain

// The guest dataset of the sample project (05 BE 6). Wedding_Planner.pdf p. 13 lists no guest names: it only has two
// tables with six counters each, all at 0. The twenty guests are fictional, the phone numbers follow the pattern
// 08120000xxxx, and the table of 05 BE 6 is the source: TestSampleGuestsMatchSpec reads it.

// SampleGuest is one row of that table. Attending guests get pax_actual = pax_count; the answers of the guests whose
// invitation was not sent (15-20) stand for a spoken confirmation.
type SampleGuest struct {
	Name, Side, Category, Scope, Table string
	Pax                                int
	Phone                              string
	RSVP                               RSVPStatus
	Sent                               bool
}

// SampleGuests are in the order of the table; sort_order is the position (1-based).
var SampleGuests = []SampleGuest{
	{"Bpk. Prof. Bambang Soedjarwo", "pria", "vip", "resepsi", "VIP-01", 2, "081200000101", RSVPAttending, true},
	{"Raditya Pratama & Pasangan", "pria", "teman", "resepsi", "T-04", 2, "081200000102", RSVPAttending, true},
	{"Bpk. Hendro & Keluarga", "pria", "keluarga", "all", "FAM-01", 4, "081200000103", RSVPAttending, true},
	{"Satria Wibawa (Kolega Kantor)", "pria", "kolega", "resepsi", "T-10", 2, "081200000104", RSVPMaybe, true},
	{"Bpk. Joko Susilo (Ketua RT 04)", "pria", "tetangga", "resepsi", "T-08", 2, "081200000105", RSVPDeclined, true},
	{"Bpk. Dr. Ir. Gunawan (Sahabat Ayah)", "pria", "teman_ortu", "resepsi", "VIP-02", 2, "081200000106", RSVPAttending, true},
	{"Bayu Anggoro & Partner", "pria", "teman", "resepsi", "T-05", 2, "081200000107", RSVPUnconfirmed, true},
	{"Ibu Sri Wahyuni (Bibi CPP)", "pria", "keluarga", "all", "FAM-01", 3, "081200000108", RSVPAttending, true},
	{"Fajar Nugraha", "pria", "teman", "resepsi", "T-06", 1, "081200000109", RSVPUnconfirmed, true},
	{"Bpk. Rahmat Hidayat (Kolega)", "pria", "kolega", "resepsi", "T-11", 2, "081200000110", RSVPAttending, true},
	{"Ibu Dra. Hj. Ratna Juwita", "wanita", "vip", "resepsi", "VIP-03", 2, "081200000111", RSVPAttending, true},
	{"Anisa Rahmawati (Sahabat SMA)", "wanita", "teman", "all", "T-01", 2, "081200000112", RSVPAttending, true},
	{"Bpk. H. Iwan & Keluarga", "wanita", "keluarga", "resepsi", "FAM-02", 4, "081200000113", RSVPUnconfirmed, true},
	{"Sdri. Melisa Putri (Kolega Kantor)", "wanita", "kolega", "resepsi", "T-12", 1, "081200000114", RSVPAttending, true},
	{"Ibu Nurhasanah (Tetangga RW 02)", "wanita", "tetangga", "resepsi", "T-09", 2, "081200000115", RSVPMaybe, false},
	{"Ibu Hj. Fatimah (Sahabat Ibu)", "wanita", "teman_ortu", "resepsi", "VIP-04", 2, "081200000116", RSVPAttending, false},
	{"Sdri. Tiara Andini", "wanita", "teman", "resepsi", "T-02", 1, "081200000117", RSVPUnconfirmed, false},
	{"Bpk. Agus Santoso (Paman CPW)", "wanita", "keluarga", "all", "FAM-02", 3, "081200000118", RSVPAttending, false},
	{"Sdri. Citra Lestari", "wanita", "teman", "resepsi", "T-03", 2, "081200000119", RSVPAttending, false},
	{"Bpk. Dedi Sukamto (Kolega Ortu)", "wanita", "teman_ortu", "resepsi", "T-14", 2, "081200000120", RSVPDeclined, false},
}
