package http

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// The spreadsheets of the guest list and the attendance sheet (05 BE 2.6). Colors are the sage tokens of the frontend
// (the same as the exports of menus 03 and 04).
const (
	sheetGuests     = "Daftar Tamu"
	sheetAttendance = "Kehadiran"

	colorHeader = "526E4D" // sage-600
	colorTotal  = "EEF3EC"
)

var (
	sentLabel = map[domain.SentStatus]string{domain.SentUnsent: "Belum dikirim", domain.SentSent: "Terkirim", domain.SentDelivered: "Diterima"}
	rsvpLabel = map[domain.RSVPStatus]string{
		domain.RSVPUnconfirmed: "Belum konfirmasi", domain.RSVPAttending: "Hadir", domain.RSVPDeclined: "Tidak hadir", domain.RSVPMaybe: "Ragu-ragu",
	}
)

// safeCell implements the contract rule against formula injection (kontrak bab 2 butir 5): text that starts with
// = + - @ gets a leading apostrophe. excelize stores user text as a string, never as a formula, so this is a second
// layer; the apostrophe is written as a character and therefore visible.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@", rune(s[0])) {
		return "'" + s
	}

	return s
}

type sheetWriter struct {
	f    *excelize.File
	name string
	err  error
}

func (w *sheetWriter) cell(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)

	return c
}

func (w *sheetWriter) keep(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *sheetWriter) text(col, row int, s string) {
	w.keep(w.f.SetCellStr(w.name, w.cell(col, row), safeCell(s)))
}

func (w *sheetWriter) number(col, row int, v any) {
	w.keep(w.f.SetCellValue(w.name, w.cell(col, row), v))
}

func (w *sheetWriter) style(fromCol, fromRow, toCol, toRow, style int) {
	w.keep(w.f.SetCellStyle(w.name, w.cell(fromCol, fromRow), w.cell(toCol, toRow), style))
}

type styles struct{ head, title, total int }

func newStyles(f *excelize.File) (styles, error) {
	var st styles
	var err error
	if st.head, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorHeader}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	}); err != nil {
		return st, err
	}
	if st.title, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12, Color: colorHeader}}); err != nil {
		return st, err
	}
	st.total, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorTotal}}})

	return st, err
}

func newBook(sheet string) (*excelize.File, styles, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, styles{}, err
	}
	st, err := newStyles(f)

	return f, st, err
}

func header(w *sheetWriter, st styles, row int, titles []string) {
	for i, t := range titles {
		w.text(i+1, row, t)
	}
	w.style(1, row, len(titles), row, st.head)
}

func widths(w *sheetWriter, sheet string, ws []float64) {
	for col, width := range ws {
		name, _ := excelize.ColumnNumberToName(col + 1)
		w.keep(w.f.SetColWidth(sheet, name, name, width))
	}
}

func sessionNames(sessions []domain.SessionRef) map[string]string {
	names := map[string]string{"all": "Umum"}
	for _, s := range sessions {
		if s.ScopeKey != "all" {
			names[s.ScopeKey] = s.Name
		}
	}

	return names
}

func sessionName(names map[string]string, scope string) string {
	if n, ok := names[scope]; ok {
		return n
	}

	return scope
}

// attendingPax is the head-count a guest answered with: pax_actual, or pax_count when that is still 0.
func attendingPax(g domain.Guest) int {
	if g.PaxActual > 0 {
		return g.PaxActual
	}

	return g.PaxCount
}

// buildGuestsWorkbook is the whole list in order with a TOTAL row. Pax are numbers; phone numbers and ticket codes are
// text, so a spreadsheet does not turn 0812… into a number. The ticket code lets the owner print QR codes with a tool of
// their own (the planner draws none, 05 BE 10.8); the file is for the owner of the project only.
func buildGuestsWorkbook(data domain.GuestExport) (*excelize.File, error) {
	f, st, err := newBook(sheetGuests)
	if err != nil {
		return nil, err
	}
	w := &sheetWriter{f: f, name: sheetGuests}
	names := sessionNames(data.Sessions)

	w.text(1, 1, "Daftar Tamu — "+data.Project.Title)
	w.style(1, 1, 1, 1, st.title)
	w.text(1, 2, "Diekspor "+data.Today.Format("2006-01-02"))
	header(w, st, 4, []string{"No", "Nama", "Pihak", "Kategori", "Sesi", "No Meja", "Pax", "No WhatsApp", "Status Undangan", "RSVP",
		"Pax Hadir (RSVP)", "Kode Tiket", "Catatan"})
	row, pax := 5, 0
	for i, g := range data.Guests {
		w.number(1, row, i+1)
		w.text(2, row, g.Name)
		if len(data.Spec.Sides) > 0 {
			w.text(3, row, data.Spec.SideLabel(g.GuestSide))
		}
		w.text(4, row, data.Spec.CategoryLabel(g.Category))
		w.text(5, row, sessionName(names, g.SubEventScope))
		if g.TableNumber != nil {
			w.text(6, row, *g.TableNumber)
		}
		w.number(7, row, g.PaxCount)
		if g.PhoneNumber != nil {
			w.text(8, row, *g.PhoneNumber)
		}
		w.text(9, row, sentLabel[g.InvitationSentStatus])
		w.text(10, row, rsvpLabel[g.RSVPStatus])
		if g.RSVPStatus == domain.RSVPAttending {
			w.number(11, row, attendingPax(g))
		}
		w.text(12, row, g.QRCodeToken)
		if g.Notes != nil {
			w.text(13, row, *g.Notes)
		}
		pax += g.PaxCount
		row++
	}

	w.text(1, row, "TOTAL")
	w.text(2, row, fmt.Sprintf("%d tamu", len(data.Guests)))
	w.number(7, row, pax)
	w.style(1, row, 13, row, st.total)
	widths(w, sheetGuests, []float64{6, 34, 20, 18, 18, 12, 7, 18, 16, 16, 12, 34, 30})

	return f, w.err
}

// buildAttendanceWorkbook is the final attendance sheet: the guests who were expected or came, with the entry time in the
// zone of the project (HH.MM) and a TOTAL row.
func buildAttendanceWorkbook(data domain.AttendanceExport) (*excelize.File, error) {
	f, st, err := newBook(sheetAttendance)
	if err != nil {
		return nil, err
	}
	w := &sheetWriter{f: f, name: sheetAttendance}
	names := sessionNames(data.Sessions)

	w.text(1, 1, "Rekap Kehadiran — "+data.Project.Title)
	w.style(1, 1, 1, 1, st.title)
	w.text(1, 2, "Diekspor "+data.Today.Format("2006-01-02"))
	header(w, st, 4, []string{"No", "Nama", "Pihak", "Kategori", "Sesi", "No Meja", "Pax Undangan", "Pax Hadir", "Jam Masuk", "Souvenir", "RSVP"})
	row := 5
	for i, g := range data.Guests {
		w.number(1, row, i+1)
		w.text(2, row, g.Name)
		if len(data.Spec.Sides) > 0 {
			w.text(3, row, data.Spec.SideLabel(g.GuestSide))
		}
		w.text(4, row, data.Spec.CategoryLabel(g.Category))
		w.text(5, row, sessionName(names, g.SubEventScope))
		if g.TableNumber != nil {
			w.text(6, row, *g.TableNumber)
		}
		w.number(7, row, g.PaxCount)
		if g.IsCheckedIn {
			w.number(8, row, g.PaxActual)
			if g.CheckedInAt != nil {
				w.text(9, row, entryTime(*g.CheckedInAt, data.Location))
			}
		}
		if g.SouvenirReceived {
			w.text(10, row, "Ya")
		}
		w.text(11, row, rsvpLabel[g.RSVPStatus])
		row++
	}

	w.text(1, row, "TOTAL")
	w.text(2, row, fmt.Sprintf("%d hadir dari %d tamu", data.KPI.CheckedInGuests, data.KPI.ExpectedGuests))
	w.number(8, row, data.KPI.ActualPax)
	w.text(10, row, fmt.Sprintf("%d diserahkan", data.KPI.SouvenirDistributed))
	w.style(1, row, 11, row, st.total)
	widths(w, sheetAttendance, []float64{6, 34, 20, 18, 18, 12, 13, 11, 11, 16, 16})

	return f, w.err
}

// entryTime is HH.MM in the zone of the project.
func entryTime(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}

	return t.In(loc).Format("15.04")
}

// writeBook serialises the workbook and closes it. A failure while drawing is returned as is.
func writeBook(f *excelize.File, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
