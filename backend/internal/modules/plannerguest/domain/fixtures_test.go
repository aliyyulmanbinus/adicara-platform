package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	beDoc      = "05_BUKU_TAMU/BE.md"
	fixtureDir = specRoot + "05_BUKU_TAMU/fixtures/"
)

var sampleNS = uuid.MustParse("6f1c1a52-0000-4000-8000-0000000005a0")

type sampleRow struct {
	No              int
	Name, Side      string
	Category, Scope string
	Table           string
	Pax             int
	Phone           string
	RSVP            RSVPStatus
	Sent            bool
}

var tableRow = regexp.MustCompile("^\\| (\\d+) \\| (.+?) \\| `(pria|wanita)` \\| `([a-z_]+)` \\| `([a-z_]+)` \\| ([A-Z0-9-]+) \\| (\\d+) \\| (\\d+) \\| `([a-z]+)` \\| (terkirim|belum) \\|$")

// sampleRows membaca tabel 20 tamu BE bab 6: dokumen adalah sumber tunggal data contoh.
func sampleRows(t testing.TB) []sampleRow {
	var out []sampleRow
	for _, line := range strings.Split(readSpec(t, beDoc), "\n") {
		m := tableRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		pax, _ := strconv.Atoi(m[7])
		out = append(out, sampleRow{No: n, Name: m[2], Side: m[3], Category: m[4], Scope: m[5], Table: m[6], Pax: pax, Phone: m[8], RSVP: RSVPStatus(m[9]), Sent: m[10] == "terkirim"})
	}
	return out
}

func sp(s string) *string { return &s }

func ts(s string) *time.Time {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &v
}

// Garis dasar fixture: sama dengan fixture Menu 02 (Proyek Contoh dibuat 2026-10-04, Hari-H 2027-03-28).
const baseline = "2026-10-04T03:00:00Z"

func sampleGuests(rows []sampleRow) []Guest {
	var gs []Guest
	for _, r := range rows {
		g := Guest{
			ID: uuid.NewSHA1(sampleNS, []byte(fmt.Sprintf("guest-%d", r.No))), Name: r.Name, GuestSide: sp(r.Side), Category: r.Category, SubEventScope: r.Scope,
			TableNumber: sp(r.Table), PaxCount: r.Pax, PhoneNumber: sp(r.Phone), FormattedWANumber: sp("62" + r.Phone[1:]),
			InvitationSentStatus: SentUnsent, RSVPStatus: r.RSVP,
			QRCodeToken: strings.ReplaceAll(uuid.NewSHA1(sampleNS, []byte(fmt.Sprintf("token-%d", r.No))).String(), "-", ""),
		}
		if r.Sent {
			g.InvitationSentStatus, g.InvitationSentAt = SentSent, ts(baseline)
		}
		if r.RSVP != RSVPUnconfirmed {
			g.RSVPSubmittedAt = ts(baseline)
		}
		if r.RSVP == RSVPAttending {
			g.PaxActual = r.Pax
		}
		gs = append(gs, g)
	}
	return gs
}

func aggregatesOf(gs []Guest) Aggregates {
	var a Aggregates
	cells := map[string]*Cell{}
	for _, g := range gs {
		a.Total++
		a.TotalPax += g.PaxCount
		if g.InvitationSentStatus != SentUnsent {
			a.Sent++
		}
		switch g.RSVPStatus {
		case RSVPAttending:
			a.Attending++
			if g.PaxActual > 0 {
				a.AttendingPax += g.PaxActual
			} else {
				a.AttendingPax += g.PaxCount
			}
		case RSVPDeclined:
			a.Declined++
		case RSVPMaybe:
			a.Maybe++
		default:
			a.Unconfirmed++
		}
		k := "-"
		if g.GuestSide != nil {
			k = *g.GuestSide
		}
		k += "/" + g.Category
		c := cells[k]
		if c == nil {
			c = &Cell{Side: g.GuestSide, Category: g.Category}
			cells[k] = c
		}
		c.Invitations++
		c.Pax += g.PaxCount
	}
	for _, c := range cells {
		a.Cells = append(a.Cells, *c)
	}
	return a
}

// Pagi hari-H (ilustrasi): 11 tamu sudah masuk, jumlah pax riil = pax undangan, semuanya menerima souvenir.
var arrivals = []struct {
	No   int
	Time string // jam lokal WIB
}{{1, "11:14"}, {2, "11:22"}, {6, "11:31"}, {3, "11:38"}, {11, "11:45"}, {4, "11:52"}, {7, "11:58"}, {9, "12:03"}, {10, "12:07"}, {14, "12:11"}, {19, "12:16"}}

func dayOf(gs []Guest) []Guest {
	out := append([]Guest(nil), gs...)
	for _, a := range arrivals {
		g := &out[a.No-1]
		g.IsCheckedIn, g.SouvenirReceived = true, true
		g.PaxActual = g.PaxCount
		hh, mm, _ := strings.Cut(a.Time, ":")
		g.CheckedInAt = ts(fmt.Sprintf("2027-03-28T%02d:%s:00Z", mustAtoi(hh)-7, mm))
	}
	return out
}

func mustAtoi(s string) int { n, _ := strconv.Atoi(s); return n }

func receptionOf(spec GuestSpec, gs []Guest) ReceptionOverview {
	var r ReceptionOverview
	cat := map[string]int{}
	var expected []Guest
	for _, g := range gs {
		if g.IsCheckedIn {
			r.KPI.CheckedInGuests++
			r.KPI.ActualPax += g.PaxActual
			r.KPI.ExpectedPax += g.PaxActual
		}
		if g.RSVPStatus != RSVPDeclined || g.IsCheckedIn {
			r.KPI.ExpectedGuests++
			cat[g.Category]++
			expected = append(expected, g)
			if !g.IsCheckedIn {
				if g.PaxActual > 0 {
					r.KPI.ExpectedPax += g.PaxActual
				} else {
					r.KPI.ExpectedPax += g.PaxCount
				}
			}
		}
		if g.SouvenirReceived {
			r.KPI.SouvenirDistributed++
		}
	}
	r.KPI.PendingGuests = r.KPI.ExpectedGuests - r.KPI.CheckedInGuests
	stock := 40
	r.KPI.SouvenirStock = &stock
	r.Filters = ReceptionFilters{All: r.KPI.ExpectedGuests, CheckedIn: r.KPI.CheckedInGuests, Pending: r.KPI.PendingGuests, Categories: []CategoryCount{}}
	for _, o := range spec.Categories {
		r.Filters.Categories = append(r.Filters.Categories, CategoryCount{Key: o.Key, Label: o.Label, Invitations: cat[o.Key]})
	}
	var in []Guest
	for _, g := range expected {
		if g.IsCheckedIn {
			in = append(in, g)
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].CheckedInAt.After(*in[j].CheckedInAt) })
	r.RecentCheckins = append([]Guest{}, in[:10]...)
	sort.Slice(expected, func(i, j int) bool { return strings.ToLower(expected[i].Name) < strings.ToLower(expected[j].Name) })
	r.Guests, r.Total, r.Limit, r.Offset = expected, len(expected), 50, 0
	return r
}

func projectionOf(gs []Guest) CateringProjection {
	var p CateringProjection
	for _, g := range gs {
		switch g.RSVPStatus {
		case RSVPAttending:
			p.AttendingGuests++
			if g.PaxActual > 0 {
				p.ConfirmedPax += g.PaxActual
			} else {
				p.ConfirmedPax += g.PaxCount
			}
		case RSVPDeclined:
			p.DeclinedGuests++
			p.DeclinedPax += g.PaxCount
		case RSVPMaybe:
			p.MaybeGuests++
			p.PotentialPax += g.PaxCount
		default:
			p.PendingGuests++
			p.PotentialPax += g.PaxCount
		}
	}
	p.BufferPercent, p.BufferPax = DefaultBufferPercent, 0
	p.RecommendedPortions, _ = RecommendedPortions(p.ConfirmedPax, p.BufferPercent, p.BufferPax)
	return p
}

const dryRunCSV = "\xef\xbb\xbfNama Tamu;Pihak;Kategori;Sesi;No Meja;Pax;No WhatsApp;Catatan\r\n" +
	"Dewi Anggraini;Wanita;Teman;Resepsi;T-20;2;081200000201;\r\n" +
	"Hendra Wijaya;Pria;Kolega Kantor;Resepsi;;1;81200000202;\r\n" +
	"Maya Sari;Laki-laki;Teman;Resepsi;;2;;\r\n" +
	"Raditya Pratama & Pasangan;Pria;Teman;Resepsi;T-04;2;081200000102;\r\n" +
	"dewi  anggraini;Wanita;Teman;Resepsi;T-20;2;+62 812-0000-0201;\r\n" +
	"Sinta Maharani;Wanita;Keluarga;Akad Nikah;FAM-03;3;+62 812-0000-0203;Kursi roda\r\n"

func importOf(t testing.TB, spec GuestSpec, gs []Guest) ImportResult {
	var existing []ExistingGuest
	for _, g := range gs {
		existing = append(existing, ExistingGuest{ID: g.ID, Name: g.Name, WA: *g.FormattedWANumber})
	}
	recs, err := DecodeCSV([]byte(dryRunCSV))
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildImportPlan(ImportConfig{Spec: spec, Sessions: sessions, Existing: existing}, recs)
	if err != nil {
		t.Fatal(err)
	}
	p.Result.DryRun = true
	return p.Result
}

func fixtures(t testing.TB) map[string]any {
	spec := specsFromTemplateEvent(t)["wedding"]
	rows := sampleRows(t)
	if len(rows) != 20 {
		t.Fatalf("sample table has %d rows, want 20", len(rows))
	}
	gs := sampleGuests(rows)
	day := dayOf(gs)
	return map[string]any{
		"planner-guests.json":              GuestList{Overview: spec.AssembleOverview(aggregatesOf(gs)), Guests: gs, Total: len(gs), Limit: 100, Offset: 0},
		"planner-reception.json":           receptionOf(spec, day),
		"planner-catering-projection.json": projectionOf(gs),
		"planner-import-dryrun.json":       importOf(t, spec, gs),
		"planner-import-sample.csv":        []byte(dryRunCSV),
	}
}

func render(v any) []byte {
	if raw, ok := v.([]byte); ok {
		return raw
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// WRITE_FIXTURES=1 menulis ulang berkas; tanpa itu tes memastikan berkas di repo sama dengan keluaran kode.
func TestFixturesAreCurrent(t *testing.T) {
	for name, v := range fixtures(t) {
		want := render(v)
		if os.Getenv("WRITE_FIXTURES") == "1" {
			if err := os.WriteFile(fixtureDir+name, want, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(fixtureDir + name)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s is out of date (run with WRITE_FIXTURES=1)", name)
		}
	}
}

func TestSampleDayOfNumbersMatchTheWireframe(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	r := receptionOf(spec, dayOf(sampleGuests(sampleRows(t))))
	k := r.KPI
	if k.CheckedInGuests != 11 || k.ExpectedGuests != 18 || k.PendingGuests != 7 || k.ActualPax != 22 || k.ExpectedPax != 39 || k.SouvenirDistributed != 11 || *k.SouvenirStock-k.SouvenirDistributed != 29 {
		t.Errorf("%+v", k)
	}
	if r.Filters.All != 18 || r.Filters.CheckedIn != 11 || r.Filters.Pending != 7 || len(r.RecentCheckins) != 10 || r.RecentCheckins[0].Name != "Sdri. Citra Lestari" {
		t.Errorf("%+v / %s", r.Filters, r.RecentCheckins[0].Name)
	}
	// "belum hadir" menurut Anisa (#12) masih di luar, Bayu (#7) sudah masuk
	for _, g := range r.Guests {
		if strings.HasPrefix(g.Name, "Anisa") && g.IsCheckedIn {
			t.Error("Anisa must still be outside")
		}
	}
}

func TestSampleProjectionAndImportNumbers(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	gs := sampleGuests(sampleRows(t))
	p := projectionOf(gs)
	if p.ConfirmedPax != 27 || p.PotentialPax != 12 || p.DeclinedPax != 4 || p.RecommendedPortions != 34 || p.AttendingGuests != 12 || p.MaybeGuests != 2 || p.PendingGuests != 4 || p.DeclinedGuests != 2 {
		t.Errorf("%+v", p)
	}
	r := importOf(t, spec, gs)
	if r.TotalRows != 6 || r.ValidRows != 4 || r.InvalidRows != 2 || len(r.Duplicates) != 2 || len(r.Errors) != 2 || len(r.Preview) != 4 {
		t.Errorf("%+v", r)
	}
	if r.Errors[0].Row != 3 || r.Errors[0].Field != "phone_number" || r.Errors[1].Row != 4 || r.Errors[1].Field != "guest_side" {
		t.Errorf("%+v", r.Errors)
	}
}
