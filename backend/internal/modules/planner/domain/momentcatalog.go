package domain

// The suggested moment tags of the playlist (menu 06), TEMPLATE_EVENT 6. Wedding follows the groups the suggestions of
// 06 BE 5.4 use; the other eight types are Adicara proposals. TestMomentTagsMatchTemplateEvent (plannerrundown/domain)
// reads that document and fails when this file drifts. A song may carry any other text: the registry only suggests.

var momentTags = map[EventType][]string{
	EventWedding:   {"Akad Khidmat", "Kirab / Entrance", "Prosesi Adat", "Santap & Ramah Tamah", "Games & Lempar Bunga", "Penutup"},
	EventKhitanan:  {"Sholawat & Doa", "Hiburan Ceria", "Jamuan Santai"},
	EventAqiqah:    {"Sholawat & Doa", "Syukuran"},
	EventPeresmian: {"Mars Perusahaan", "Prosesi Peresmian", "Networking Santai"},
	EventLaunching: {"Countdown & Reveal", "Applause & Fanfare", "Networking Ambience"},
	EventBirthday:  {"Welcome Music", "Happy Birthday Moment", "Party / Dance", "Chill / Closing"},
	EventWisuda:    {"Paduan Suara / Hymne", "Prosesi Masuk", "Pelantikan", "Lagu Perpisahan"},
	EventSeminar:   {"Opening Bumper", "Background Presentasi", "Jeda Istirahat", "Closing Jingle"},
	EventOther:     {"Pembukaan", "Acara Utama", "Penutup"},
}

// momentTagsFor hands out a copy, so a caller cannot change the shared table. Every type has tags, so the slice is
// never nil (GET /event-types sends [] and not null even for a type added without any).
func momentTagsFor(t EventType) []string { return append([]string{}, momentTags[t]...) }
