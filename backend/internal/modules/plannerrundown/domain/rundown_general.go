package domain

// The default agendas of the eight non-wedding event types (TEMPLATE_EVENT 7-14, Adicara proposals): 9, 7, 7, 9, 7, 9, 9
// and 3 rows, 60 in all. The key is "<type>.agenda_<NN>", NN the row in the table of the type (two digits from 01): the
// order, not the title, because titles repeat. TestGeneralTemplateMatchesTemplateEvent reads that document.

// GeneralAgendas maps an event type key to its agenda template.
var GeneralAgendas = map[string][]TemplateAgenda{
	"khitanan": {
		{Key: "khitanan.agenda_01", ScopeKey: "prosesi_doa", Start: "08:00", End: "08:30", Name: "Persiapan dan penyambutan keluarga", PIC: "Panitia"},
		{Key: "khitanan.agenda_02", ScopeKey: "prosesi_doa", Start: "08:30", End: "08:40", Name: "Pembukaan dan sambutan keluarga", PIC: "MC"},
		{Key: "khitanan.agenda_03", ScopeKey: "prosesi_doa", Start: "08:40", End: "09:00", Name: "Pembacaan Al-Qur'an dan shalawat", PIC: "Qari/Ustadz"},
		{Key: "khitanan.agenda_04", ScopeKey: "prosesi_doa", Start: "09:00", End: "09:15", Name: "Doa dan tausiyah singkat", PIC: "Ustadz"},
		{Key: "khitanan.agenda_05", ScopeKey: "hiburan", Start: "09:15", End: "10:00", Name: "Hadroh / hiburan", PIC: "Pengisi acara"},
		{Key: "khitanan.agenda_06", ScopeKey: "hiburan", Start: "10:00", End: "10:15", Name: "Games anak", PIC: "MC"},
		{Key: "khitanan.agenda_07", ScopeKey: "jamuan", Start: "10:15", End: "11:30", Name: "Santap bersama", PIC: "Katering"},
		{Key: "khitanan.agenda_08", ScopeKey: "jamuan", Start: "11:30", End: "11:45", Name: "Foto bersama keluarga", PIC: "Fotografer"},
		{Key: "khitanan.agenda_09", ScopeKey: "jamuan", Start: "11:45", End: "12:00", Name: "Penutupan", PIC: "MC"},
	},
	"aqiqah": {
		{Key: "aqiqah.agenda_01", ScopeKey: "prosesi_aqiqah", Start: "09:00", End: "09:15", Name: "Pembukaan dan penyambutan", PIC: "MC"},
		{Key: "aqiqah.agenda_02", ScopeKey: "prosesi_aqiqah", Start: "09:15", End: "09:30", Name: "Pembacaan Al-Qur'an dan shalawat", PIC: "Qari/Ustadz"},
		{Key: "aqiqah.agenda_03", ScopeKey: "prosesi_aqiqah", Start: "09:30", End: "09:45", Name: "Pencukuran rambut dan pemberian nama", PIC: "Ustadz dan orang tua"},
		{Key: "aqiqah.agenda_04", ScopeKey: "prosesi_aqiqah", Start: "09:45", End: "10:00", Name: "Doa aqiqah dan tausiyah", PIC: "Ustadz"},
		{Key: "aqiqah.agenda_05", ScopeKey: "syukuran", Start: "10:00", End: "11:30", Name: "Santap bersama dan ramah tamah", PIC: "Katering"},
		{Key: "aqiqah.agenda_06", ScopeKey: "syukuran", Start: "11:30", End: "11:45", Name: "Pembagian box/nasi aqiqah", PIC: "Panitia"},
		{Key: "aqiqah.agenda_07", ScopeKey: "syukuran", Start: "11:45", End: "12:00", Name: "Foto bersama dan penutupan", PIC: "Fotografer / MC"},
	},
	"peresmian": {
		{Key: "peresmian.agenda_01", ScopeKey: "pembukaan", Start: "08:00", End: "08:10", Name: "Registrasi tamu", PIC: "Panitia"},
		{Key: "peresmian.agenda_02", ScopeKey: "pembukaan", Start: "08:10", End: "08:15", Name: "Pembukaan MC", PIC: "MC"},
		{Key: "peresmian.agenda_03", ScopeKey: "pembukaan", Start: "08:15", End: "08:30", Name: "Sambutan pimpinan", PIC: "Pimpinan"},
		{Key: "peresmian.agenda_04", ScopeKey: "prosesi_peresmian", Start: "08:30", End: "08:40", Name: "Gunting pita / peresmian", PIC: "VIP dan pimpinan"},
		{Key: "peresmian.agenda_05", ScopeKey: "prosesi_peresmian", Start: "08:40", End: "09:00", Name: "Tur fasilitas / showcase", PIC: "Pemandu"},
		{Key: "peresmian.agenda_06", ScopeKey: "ramah_tamah", Start: "09:00", End: "10:00", Name: "Ramah tamah dan jamuan", PIC: "Katering"},
		{Key: "peresmian.agenda_07", ScopeKey: "ramah_tamah", Start: "10:00", End: "10:10", Name: "Doa dan penutup", PIC: "MC"},
	},
	"launching": {
		{Key: "launching.agenda_01", ScopeKey: "opening", Start: "13:00", End: "13:30", Name: "Registrasi dan welcome drink", PIC: "Panitia"},
		{Key: "launching.agenda_02", ScopeKey: "opening", Start: "13:30", End: "13:35", Name: "Pembukaan MC", PIC: "MC"},
		{Key: "launching.agenda_03", ScopeKey: "opening", Start: "13:35", End: "13:50", Name: "Sambutan founder/CEO", PIC: "Founder"},
		{Key: "launching.agenda_04", ScopeKey: "peluncuran", Start: "13:50", End: "14:10", Name: "Presentasi brand dan visi", PIC: "Presenter"},
		{Key: "launching.agenda_05", ScopeKey: "peluncuran", Start: "14:10", End: "14:20", Name: "Video teaser", PIC: "Tim AV"},
		{Key: "launching.agenda_06", ScopeKey: "peluncuran", Start: "14:20", End: "14:30", Name: "Product reveal", PIC: "Founder"},
		{Key: "launching.agenda_07", ScopeKey: "peluncuran", Start: "14:30", End: "15:00", Name: "Demo produk dan tanya jawab", PIC: "Tim produk"},
		{Key: "launching.agenda_08", ScopeKey: "networking", Start: "15:00", End: "15:30", Name: "Sesi pers dan wawancara", PIC: "PR"},
		{Key: "launching.agenda_09", ScopeKey: "networking", Start: "15:30", End: "17:00", Name: "Networking dan jamuan", PIC: "Katering"},
	},
	"birthday": {
		{Key: "birthday.agenda_01", ScopeKey: "welcome", Start: "17:00", End: "17:30", Name: "Registrasi dan sambutan tamu", PIC: "Panitia"},
		{Key: "birthday.agenda_02", ScopeKey: "welcome", Start: "17:30", End: "17:35", Name: "Pembukaan MC", PIC: "MC"},
		{Key: "birthday.agenda_03", ScopeKey: "acara_utama", Start: "17:35", End: "17:50", Name: "Sambutan orang tua/panitia", PIC: "Keluarga"},
		{Key: "birthday.agenda_04", ScopeKey: "acara_utama", Start: "17:50", End: "18:00", Name: "Tiup lilin dan potong kue", PIC: "Yang berulang tahun"},
		{Key: "birthday.agenda_05", ScopeKey: "acara_utama", Start: "18:00", End: "18:30", Name: "Games dan hiburan interaktif", PIC: "MC"},
		{Key: "birthday.agenda_06", ScopeKey: "acara_utama", Start: "18:30", End: "19:30", Name: "Santap dan ramah tamah", PIC: "Tamu"},
		{Key: "birthday.agenda_07", ScopeKey: "after_party", Start: "19:30", End: "21:00", Name: "Musik, karaoke, atau DJ set", PIC: "DJ / pengisi acara"},
	},
	"wisuda": {
		{Key: "wisuda.agenda_01", ScopeKey: "prosesi_senat", Start: "08:00", End: "08:30", Name: "Registrasi wisudawan dan tamu", PIC: "Panitia"},
		{Key: "wisuda.agenda_02", ScopeKey: "prosesi_senat", Start: "08:30", End: "08:45", Name: "Prosesi masuk senat", PIC: "Protokol"},
		{Key: "wisuda.agenda_03", ScopeKey: "prosesi_senat", Start: "08:45", End: "08:50", Name: "Pembukaan dan Indonesia Raya", PIC: "MC / paduan suara"},
		{Key: "wisuda.agenda_04", ScopeKey: "prosesi_senat", Start: "08:50", End: "09:10", Name: "Laporan rektor/dekan", PIC: "Pimpinan"},
		{Key: "wisuda.agenda_05", ScopeKey: "pelantikan", Start: "09:10", End: "09:20", Name: "Pengukuhan lulusan", PIC: "Rektor"},
		{Key: "wisuda.agenda_06", ScopeKey: "pelantikan", Start: "09:20", End: "10:30", Name: "Pemanggilan dan penyematan", PIC: "Protokol"},
		{Key: "wisuda.agenda_07", ScopeKey: "pelantikan", Start: "10:30", End: "10:45", Name: "Pidato wisudawan terbaik", PIC: "Wisudawan"},
		{Key: "wisuda.agenda_08", ScopeKey: "pelantikan", Start: "10:45", End: "11:00", Name: "Doa, penutup, Gaudeamus Igitur", PIC: "MC / paduan suara"},
		{Key: "wisuda.agenda_09", ScopeKey: "foto_studio", Start: "11:00", End: "14:00", Name: "Foto studio dan keluarga", PIC: "Fotografer"},
	},
	"seminar": {
		{Key: "seminar.agenda_01", ScopeKey: "registrasi", Start: "07:30", End: "08:00", Name: "Registrasi dan welcome kit", PIC: "Panitia registrasi"},
		{Key: "seminar.agenda_02", ScopeKey: "registrasi", Start: "08:00", End: "08:05", Name: "Indonesia Raya", PIC: "MC dan hadirin"},
		{Key: "seminar.agenda_03", ScopeKey: "registrasi", Start: "08:05", End: "08:15", Name: "Sambutan pembuka", PIC: "Ketua panitia"},
		{Key: "seminar.agenda_04", ScopeKey: "sesi_pagi", Start: "08:15", End: "09:00", Name: "Keynote speech", PIC: "Keynote speaker"},
		{Key: "seminar.agenda_05", ScopeKey: "sesi_pagi", Start: "09:00", End: "10:00", Name: "Panel diskusi", PIC: "Moderator dan panelis"},
		{Key: "seminar.agenda_06", ScopeKey: "sesi_pagi", Start: "10:00", End: "10:15", Name: "Tanya jawab", PIC: "Moderator"},
		{Key: "seminar.agenda_07", ScopeKey: "coffee_break", Start: "10:15", End: "10:45", Name: "Coffee break dan networking", PIC: "Katering"},
		{Key: "seminar.agenda_08", ScopeKey: "sesi_sore", Start: "10:45", End: "12:00", Name: "Workshop / breakout session", PIC: "Fasilitator"},
		{Key: "seminar.agenda_09", ScopeKey: "sesi_sore", Start: "12:00", End: "12:10", Name: "Penutupan dan foto bersama", PIC: "MC"},
	},
	"other": {
		{Key: "other.agenda_01", ScopeKey: "sesi_1", Start: "09:00", End: "09:15", Name: "Pembukaan", PIC: "MC"},
		{Key: "other.agenda_02", ScopeKey: "sesi_1", Start: "09:15", End: "11:00", Name: "Acara utama", PIC: "Penyelenggara"},
		{Key: "other.agenda_03", ScopeKey: "sesi_1", Start: "11:00", End: "11:15", Name: "Penutupan", PIC: "MC"},
	},
}
