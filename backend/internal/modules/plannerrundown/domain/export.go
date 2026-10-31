package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The two WhatsApp texts (06 BE 2.4 and 2.7) are built here, on the server, because they need every agenda or song and
// the copy button uses them as they are. They are plain WhatsApp markup: *bold* lines, one blank line between agendas.
// User text is not escaped (06 BE 9, item 6).

const (
	waRule   = "────────────────────────"
	waFooter = "(Dibuat otomatis melalui Adicara Event Planner)"
)

var weekdayID = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

var monthID = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// LongDate writes a calendar date the way the PDF does: "Minggu, 28 Maret 2027".
func LongDate(d time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", weekdayID[d.Weekday()], d.Day(), monthID[d.Month()-1], d.Year())
}

// ExportSession is a session with its agendas in display order.
type ExportSession struct {
	Session SessionRef
	Agendas []Rundown
}

// AgendaExport is what the text of the rundown is made of. Single is true when one session was asked for (Sessions then
// holds that one); otherwise every session with agendas gets its own section.
type AgendaExport struct {
	ProjectTitle string
	EventIcon    string
	EventDate    time.Time
	Venue, City  string // metadata.venue_name and metadata.city
	Single       bool
	Sessions     []ExportSession
}

// oneLine folds the line breaks of a free text into spaces, so the indentation of the text around it holds.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// location is the line under the date: the place of the session when it has one, otherwise the venue of the project,
// then the city. An empty result means no line.
func location(session *string, venue, city string) string {
	place := strings.TrimSpace(venue)
	if session != nil && strings.TrimSpace(*session) != "" {
		place = strings.TrimSpace(*session)
	}
	var parts []string
	for _, p := range []string{place, strings.TrimSpace(city)} {
		if p != "" {
			parts = append(parts, p)
		}
	}

	return strings.Join(parts, ", ")
}

// Text is the rundown as a WhatsApp message.
func (e AgendaExport) Text() string {
	var b strings.Builder
	e.writeHeader(&b)
	for _, es := range e.Sessions {
		if len(es.Agendas) == 0 {
			continue
		}
		b.WriteString(waRule + "\n")
		if !e.Single {
			writeSessionHeading(&b, es.Session)
		}
		for i, a := range es.Agendas {
			if i > 0 {
				b.WriteString("\n")
			}
			writeAgenda(&b, a)
		}
	}
	b.WriteString(waRule + "\n" + waFooter)

	return b.String()
}

// writeHeader is the part above the first rule: the title, and for one session its date and place.
func (e AgendaExport) writeHeader(b *strings.Builder) {
	if !e.Single || len(e.Sessions) != 1 {
		b.WriteString("📋 *RUNDOWN ACARA*\n" + e.EventIcon + " *" + e.ProjectTitle + "*\n")
		if loc := location(nil, e.Venue, e.City); loc != "" {
			b.WriteString("🏛 *" + loc + "*\n")
		}

		return
	}
	s := e.Sessions[0].Session
	date := e.EventDate
	if s.Date != nil {
		date = *s.Date
	}
	b.WriteString("📋 *RUNDOWN ACARA " + strings.ToUpper(s.Name) + "*\n" + e.EventIcon + " *" + e.ProjectTitle + "*\n")
	b.WriteString("🗓 *" + LongDate(date) + "*\n")
	if loc := location(s.Location, e.Venue, e.City); loc != "" {
		b.WriteString("🏛 *" + loc + "*\n")
	}
}

// writeSessionHeading opens a section of the all-sessions text: the name, then the date and the place when it has them.
func writeSessionHeading(b *strings.Builder, s SessionRef) {
	b.WriteString("*" + s.Icon + " " + strings.ToUpper(s.Name) + "*\n")
	if s.Date != nil {
		b.WriteString("🗓 *" + LongDate(*s.Date) + "*\n")
	}
	if s.Location != nil && strings.TrimSpace(*s.Location) != "" {
		b.WriteString("🏛 *" + strings.TrimSpace(*s.Location) + "*\n")
	}
}

func writeAgenda(b *strings.Builder, a Rundown) {
	b.WriteString(agendaLine(a) + "\n")
	if a.PICName != "" {
		b.WriteString("   👤 PIC: " + a.PICName + "\n")
	}
	if d := oneLine(a.Description); d != "" {
		b.WriteString("   📝 " + d + "\n")
	}
	for _, h := range HighlightItems(a.Highlights) {
		b.WriteString("   • " + h + "\n")
	}
}

// agendaLine is "⏰ *07.00 - 07.20* (20 mnt) : Persiapan"; an open agenda has no end and no duration.
func agendaLine(a Rundown) string {
	if a.EndTime == nil {
		return "⏰ *" + DotClock(a.StartTime) + " - selesai* : " + a.ActivityName
	}
	dur := ""
	if a.DurationMinutes != nil {
		dur = fmt.Sprintf(" (%d mnt)", *a.DurationMinutes)
	}

	return "⏰ *" + DotClock(a.StartTime) + " - " + DotClock(*a.EndTime) + "*" + dur + " : " + a.ActivityName
}

// PlaylistExport is what the sound cue briefing is made of: the sessions in display order (for the order and the icon of
// a section) and every song of the project in playlist order.
type PlaylistExport struct {
	ProjectTitle string
	Sessions     []SessionRef
	Songs        []Playlist
}

// Text is the briefing for the sound crew. Songs tied to an agenda are grouped by session and agenda, in clock order;
// the songs without an agenda come last, grouped by moment tag.
func (e PlaylistExport) Text() string {
	var b strings.Builder
	b.WriteString("🎵 *SOUND CUE " + e.ProjectTitle + "*\n")

	linked := map[string][]Playlist{} // session name → songs
	var loose []Playlist
	for _, s := range e.Songs {
		if s.Agenda == nil {
			loose = append(loose, s)
		} else {
			linked[s.Agenda.SessionName] = append(linked[s.Agenda.SessionName], s)
		}
	}
	for _, sess := range e.Sessions {
		if songs := linked[sess.Name]; len(songs) > 0 {
			b.WriteString(waRule + "\n*" + sess.Icon + " " + strings.ToUpper(sess.Name) + "*\n")
			writeLinkedSongs(&b, songs)
		}
	}
	if len(loose) > 0 {
		b.WriteString(waRule + "\n*Belum ditautkan ke agenda*\n")
		writeLooseSongs(&b, loose)
	}
	b.WriteString(waRule + "\n" + waFooter)

	return b.String()
}

// writeLinkedSongs writes the songs of one session in the clock order of their agendas; inside an agenda they keep the
// playlist order. The sort is stable, so two agendas that start together stay in the order their first songs have.
func writeLinkedSongs(b *strings.Builder, songs []Playlist) {
	sort.SliceStable(songs, func(i, j int) bool { return songs[i].Agenda.StartTime < songs[j].Agenda.StartTime })
	var last string
	for _, s := range songs {
		if id := s.Agenda.ID.String(); id != last {
			b.WriteString("⏰ " + DotClock(s.Agenda.StartTime) + " " + s.Agenda.ActivityName + "\n")
			last = id
		}
		writeSong(b, s)
	}
}

// writeLooseSongs groups the songs without an agenda by moment tag, in the order the tags first appear.
func writeLooseSongs(b *strings.Builder, songs []Playlist) {
	var tags []string
	byTag := map[string][]Playlist{}
	for _, s := range songs {
		if _, seen := byTag[s.MomentTag]; !seen {
			tags = append(tags, s.MomentTag)
		}
		byTag[s.MomentTag] = append(byTag[s.MomentTag], s)
	}
	for _, tag := range tags {
		b.WriteString("*" + tag + "*\n")
		for _, s := range byTag[tag] {
			writeSong(b, s)
		}
	}
}

func writeSong(b *strings.Builder, s Playlist) {
	b.WriteString("• " + s.SongTitle + " - " + s.Artist + " (" + AudioFormatLabel(s.AudioFormat) + ")\n")
	if s.AudioCueNotes == "" {
		return
	}
	lines := strings.Split(strings.TrimSpace(s.AudioCueNotes), "\n")
	b.WriteString("   ↳ Cue: " + strings.TrimSpace(lines[0]) + "\n")
	for _, l := range lines[1:] {
		if l = strings.TrimSpace(l); l != "" {
			b.WriteString("      " + l + "\n")
		}
	}
}
