package domain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func date(s string) *time.Time {
	d, _ := time.Parse("2006-01-02", s)

	return &d
}

// agendaOf builds the agenda of a template as the server returns it.
func agendaOf(t TemplateAgenda) Rundown {
	r := Rundown{ID: uuid.New(), StartTime: t.Start, ActivityName: t.Name, PICName: t.PIC, Description: t.Description, Highlights: t.Highlights, Status: StatusPending}
	if t.End != "" {
		e := t.End
		r.EndTime = &e
		times, _ := ValidateTimes(t.Start, &e, false)
		d, _ := times.Duration()
		r.DurationMinutes = &d
	}

	return r
}

func agendasOf(ts []TemplateAgenda, scope string) []Rundown {
	var out []Rundown
	for _, t := range ts {
		if t.ScopeKey == scope {
			out = append(out, agendaOf(t))
		}
	}

	return out
}

var akad = SessionRef{ID: uuid.New(), ScopeKey: "akad", Name: "Akad Nikah", Icon: "📜", Date: date("2027-03-28")}

func TestLongDate(t *testing.T) {
	for in, want := range map[string]string{
		"2027-03-28": "Minggu, 28 Maret 2027", "2026-10-04": "Minggu, 4 Oktober 2026", "2027-01-01": "Jumat, 1 Januari 2027",
		"2027-12-31": "Jumat, 31 Desember 2027", "2028-02-29": "Selasa, 29 Februari 2028", "2026-08-17": "Senin, 17 Agustus 2026",
	} {
		if got := LongDate(*date(in)); got != want {
			t.Errorf("LongDate(%s) = %q, want %q", in, got, want)
		}
	}
}

// The text of the akad of the sample project is the one of the frontend spec's fixture (built from the PDF data with the
// rules of 06 BE 2.4), so the builder and the spec cannot drift apart.
func TestAgendaTextOfTheAkadIsTheFixture(t *testing.T) {
	raw, err := os.ReadFile("../../../../../docs/APP_SPEC/PLANNER_SPEC/06_RUNDOWN_DAN_ACARA/fixtures/planner-rundown-wa-akad.json")
	if err != nil {
		t.Skipf("spec not available: %v", err)
	}
	var fixture struct{ Text string }
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}

	got := AgendaExport{
		ProjectTitle: "Contoh: Pernikahan Sinta & Arif", EventIcon: "💍", EventDate: *date("2027-03-28"),
		Venue: "Sasana Kriya Ballroom", City: "Jakarta Selatan", Single: true,
		Sessions: []ExportSession{{Session: akad, Agendas: agendasOf(WeddingAkadResepsi, "akad")}},
	}.Text()
	if got != fixture.Text {
		t.Errorf("text:\n%s\n--- fixture\n%s", got, fixture.Text)
	}
}

func TestAgendaTextRules(t *testing.T) {
	open := agendaOf(TemplateAgenda{Start: "09:15", Name: "Resepsi", Description: "Pengantin siap\ndi lokasi   acara"})
	plain := agendaOf(TemplateAgenda{Start: "10:00", End: "10:30", Name: "Tanpa PIC dan keterangan"})
	full := agendaOf(TemplateAgenda{Start: "09:00", End: "09:10", Name: "Pembukaan", PIC: "MC", Description: "Sambutan", Highlights: "- Satu\n- Dua"})
	venue := func(s SessionRef, single bool, ss ...ExportSession) string {
		return AgendaExport{ProjectTitle: "Judul", EventIcon: "🎤", EventDate: *date("2027-03-28"), Venue: "Aula", City: "Bandung", Single: single, Sessions: ss}.Text()
	}

	// one session: header with the date of the session, the venue and the city
	one := venue(akad, true, ExportSession{Session: akad, Agendas: []Rundown{full, open, plain}})
	for _, want := range []string{
		"📋 *RUNDOWN ACARA AKAD NIKAH*\n🎤 *Judul*\n🗓 *Minggu, 28 Maret 2027*\n🏛 *Aula, Bandung*\n" + waRule + "\n",
		"⏰ *09.00 - 09.10* (10 mnt) : Pembukaan\n   👤 PIC: MC\n   📝 Sambutan\n   • Satu\n   • Dua\n\n",
		"⏰ *09.15 - selesai* : Resepsi\n   📝 Pengantin siap di lokasi acara\n\n", // open: no end and no duration; the break is folded
		"⏰ *10.00 - 10.30* (30 mnt) : Tanpa PIC dan keterangan\n" + waRule + "\n(Dibuat otomatis melalui Adicara Event Planner)",
	} {
		if !strings.Contains(one, want) {
			t.Errorf("text lacks %q:\n%s", want, one)
		}
	}
	if strings.Contains(one, "👤 PIC: \n") || strings.Contains(one, "📝 \n") {
		t.Errorf("an empty PIC or description is written:\n%s", one)
	}

	// a session without a date uses the date of the event; one with a place uses it instead of the venue, the city stays
	undated := SessionRef{ID: uuid.New(), Name: "Lamaran", Icon: "💍"}
	if got := venue(undated, true, ExportSession{Session: undated, Agendas: []Rundown{plain}}); !strings.Contains(got, "🗓 *Minggu, 28 Maret 2027*\n🏛 *Aula, Bandung*\n") {
		t.Errorf("undated session:\n%s", got)
	}
	place := "Rumah Keluarga"
	placed := SessionRef{ID: uuid.New(), Name: "Lamaran", Icon: "💍", Location: &place}
	if got := venue(placed, true, ExportSession{Session: placed, Agendas: []Rundown{plain}}); !strings.Contains(got, "🏛 *Rumah Keluarga, Bandung*\n") || strings.Contains(got, "Aula") {
		t.Errorf("session with a place:\n%s", got)
	}
	// no venue and no city: no line
	bare := AgendaExport{ProjectTitle: "J", EventIcon: "x", EventDate: *date("2027-03-28"), Single: true, Sessions: []ExportSession{{Session: undated, Agendas: []Rundown{plain}}}}.Text()
	if strings.Contains(bare, "🏛") {
		t.Errorf("a place line without a place:\n%s", bare)
	}

	// every session: no date in the header, each section has its own heading; a session without agendas is left out
	empty := SessionRef{ID: uuid.New(), Name: "Kosong", Icon: "∅"}
	all := venue(akad, false, ExportSession{Session: undated, Agendas: []Rundown{plain}}, ExportSession{Session: empty}, ExportSession{Session: akad, Agendas: []Rundown{full}})
	for _, want := range []string{
		"📋 *RUNDOWN ACARA*\n🎤 *Judul*\n🏛 *Aula, Bandung*\n" + waRule + "\n*💍 LAMARAN*\n⏰ *10.00 - 10.30*",
		waRule + "\n*📜 AKAD NIKAH*\n🗓 *Minggu, 28 Maret 2027*\n⏰ *09.00 - 09.10*",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("all-sessions text lacks %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "KOSONG") || strings.Contains(strings.SplitN(all, waRule, 2)[0], "🗓") {
		t.Errorf("all-sessions text:\n%s", all)
	}
	// a session that has its own place says so under its heading
	hall := "Gedung Serbaguna"
	inHall := SessionRef{ID: uuid.New(), Name: "Resepsi", Icon: "🏛️", Location: &hall}
	if got := venue(inHall, false, ExportSession{Session: inHall, Agendas: []Rundown{plain}}); !strings.Contains(got, "*🏛️ RESEPSI*\n🏛 *Gedung Serbaguna*\n⏰") {
		t.Errorf("all-sessions text with a place:\n%s", got)
	}
	if !strings.HasSuffix(all, waRule+"\n"+waFooter) || strings.Count(all, waRule) != 3 {
		t.Errorf("all-sessions text has %d rules:\n%s", strings.Count(all, waRule), all)
	}

	// one session asked for but it has no agenda: the header and the footer only
	none := venue(empty, true, ExportSession{Session: empty})
	if !strings.HasPrefix(none, "📋 *RUNDOWN ACARA KOSONG*") || strings.Contains(none, "⏰") {
		t.Errorf("empty session:\n%s", none)
	}
}

func TestPlaylistText(t *testing.T) {
	resepsi := SessionRef{ID: uuid.New(), Name: "Resepsi", Icon: "🏛️"}
	ijab := &PlaylistAgenda{ID: uuid.New(), ActivityName: "Ijab Kabul", SessionName: "Akad Nikah", StartTime: "07:46"}
	doa := &PlaylistAgenda{ID: uuid.New(), ActivityName: "Doa", SessionName: "Akad Nikah", StartTime: "07:51"}
	iring := &PlaylistAgenda{ID: uuid.New(), ActivityName: "Iring-iring", SessionName: "Resepsi", StartTime: "09:36"}
	songs := []Playlist{
		{SongTitle: "Doa Lagu", Artist: "A", AudioFormat: FormatAcoustic, Agenda: doa},
		{SongTitle: "Akad", Artist: "Payung Teduh", AudioFormat: FormatDigitalPlayback, Agenda: ijab, AudioCueNotes: "Mulai 0:12\nFade out 0:40\n\n"},
		{SongTitle: "Marry You", Artist: "Maroon 5", AudioFormat: FormatLiveBand, MomentTag: "Kirab"},
		{SongTitle: "Iring", Artist: "B", AudioFormat: FormatSoundFX, Agenda: iring},
		{SongTitle: "Perfect", Artist: "Ed Sheeran", AudioFormat: FormatDigitalPlayback, MomentTag: "Penutup"},
		{SongTitle: "Kamulah", Artist: "Dewa 19", AudioFormat: FormatDigitalPlayback, MomentTag: "Kirab"},
		{SongTitle: "Ijab Kedua", Artist: "C", AudioFormat: FormatDigitalPlayback, Agenda: ijab},
	}
	got := PlaylistExport{ProjectTitle: "Judul", Sessions: []SessionRef{akad, resepsi}, Songs: songs}.Text()

	want := strings.Join([]string{
		"🎵 *SOUND CUE Judul*",
		waRule,
		"*📜 AKAD NIKAH*",
		"⏰ 07.46 Ijab Kabul",
		"• Akad - Payung Teduh (Digital Playback)",
		"   ↳ Cue: Mulai 0:12",
		"      Fade out 0:40",
		"• Ijab Kedua - C (Digital Playback)",
		"⏰ 07.51 Doa",
		"• Doa Lagu - A (Akustik)",
		waRule,
		"*🏛️ RESEPSI*",
		"⏰ 09.36 Iring-iring",
		"• Iring - B (Sound FX)",
		waRule,
		"*Belum ditautkan ke agenda*",
		"*Kirab*",
		"• Marry You - Maroon 5 (Live Band)",
		"• Kamulah - Dewa 19 (Digital Playback)",
		"*Penutup*",
		"• Perfect - Ed Sheeran (Digital Playback)",
		waRule,
		waFooter,
	}, "\n")
	if got != want {
		t.Errorf("text:\n%s\n--- want\n%s", got, want)
	}

	// no songs at all: the title and the footer
	if empty := (PlaylistExport{ProjectTitle: "J"}).Text(); empty != "🎵 *SOUND CUE J*\n"+waRule+"\n"+waFooter {
		t.Errorf("empty playlist:\n%s", empty)
	}
}
