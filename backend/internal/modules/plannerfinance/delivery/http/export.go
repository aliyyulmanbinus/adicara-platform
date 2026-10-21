package http

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

// The spreadsheet of GET /budget/export (03 BE 3.11), laid out like pages 5-7
// of Wedding_Planner.pdf: Rekap (the session matrix), Anggaran (one block per
// session) and Tabungan (the monthly deposits). Amounts are numbers with the
// format "Rp" #,##0; colors are the sage tokens of the frontend.
const (
	sheetRekap    = "Rekap"
	sheetAnggaran = "Anggaran"
	sheetTabungan = "Tabungan"

	colorHeader = "526E4D" // sage-600
	colorBand   = "D1DFCE" // sage-200
	colorTotal  = "EEF3EC"

	rupiahFormat = `"Rp" #,##0`
)

var statusLabel = map[string]string{
	domain.StatusUnpaid: "Belum dibayar", domain.StatusDPPaid: "DP dibayar", domain.StatusPartial: "Sebagian", domain.StatusSettled: "Lunas",
}

// safeCell implements the contract rule against formula injection (kontrak
// bab 2 butir 5): text that starts with = + - @ gets a leading apostrophe.
// excelize stores user text as a string, never as a formula, so this is a
// second layer; the apostrophe is written as a character and therefore visible.
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
func (w *sheetWriter) number(col, row int, v int64) {
	w.keep(w.f.SetCellValue(w.name, w.cell(col, row), v))
}
func (w *sheetWriter) style(fromCol, fromRow, toCol, toRow, style int) {
	w.keep(w.f.SetCellStyle(w.name, w.cell(fromCol, fromRow), w.cell(toCol, toRow), style))
}

type styles struct{ head, money, moneyBold, title, total, band int }

func newStyles(f *excelize.File) (styles, error) {
	var st styles
	var err error
	money := rupiahFormat
	if st.head, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorHeader}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	}); err != nil {
		return st, err
	}
	if st.money, err = f.NewStyle(&excelize.Style{CustomNumFmt: &money}); err != nil {
		return st, err
	}
	if st.moneyBold, err = f.NewStyle(&excelize.Style{
		CustomNumFmt: &money, Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorTotal}},
	}); err != nil {
		return st, err
	}
	if st.title, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12, Color: colorHeader}}); err != nil {
		return st, err
	}
	if st.total, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorTotal}}}); err != nil {
		return st, err
	}
	st.band, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorBand}}})

	return st, err
}

// buildWorkbook draws all three sheets from the export data.
func buildWorkbook(data domain.ExportData) (*excelize.File, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", sheetRekap); err != nil {
		return nil, err
	}
	for _, n := range []string{sheetAnggaran, sheetTabungan} {
		if _, err := f.NewSheet(n); err != nil {
			return nil, err
		}
	}
	st, err := newStyles(f)
	if err != nil {
		return nil, err
	}

	names := map[string]string{}
	for _, s := range data.Sessions {
		names[s.ScopeKey] = s.Name
	}
	sessionName := func(scope string) string {
		if n := names[scope]; n != "" {
			return n
		}

		return scope
	}

	for _, draw := range []func() error{
		func() error { return drawRekap(f, st, data, sessionName) },
		func() error { return drawAnggaran(f, st, data, sessionName) },
		func() error { return drawTabungan(f, st, data) },
	} {
		if err := draw(); err != nil {
			_ = f.Close()
			return nil, err
		}
	}

	return f, nil
}

// drawRekap is the session matrix of PDF p. 5: the sessions as columns and the
// rows Tagihan Awal, DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan Sisa.
// Rows the PDF does not have appear only when they are not zero.
func drawRekap(f *excelize.File, st styles, data domain.ExportData, sessionName func(string) string) error {
	w := &sheetWriter{f: f, name: sheetRekap}
	m := data.Matrix

	w.text(1, 1, "REKAP ANGGARAN: "+data.Project.Title)
	w.style(1, 1, 1, 1, st.title)

	w.text(1, 2, "Keterangan")
	for i, r := range m.Rows {
		w.text(i+2, 2, sessionName(r.SubEventScope))
	}
	totalCol := len(m.Rows) + 2
	w.text(totalCol, 2, "Total")
	w.style(1, 2, totalCol, 2, st.head)

	type line struct {
		label string
		get   func(domain.MatrixRow) int64
		show  bool
	}
	t := m.Total
	lines := []line{
		{"Tagihan (Awal)", func(r domain.MatrixRow) int64 { return r.Estimate }, true},
		{"DP/Lunas", func(r domain.MatrixRow) int64 { return r.DP }, true},
		{"Termin 1", func(r domain.MatrixRow) int64 { return r.Term1 }, true},
		{"Termin 2", func(r domain.MatrixRow) int64 { return r.Term2 }, true},
		{"Dibayar dari belanja", func(r domain.MatrixRow) int64 { return r.RollupPaid }, t.RollupPaid != 0},
		{"Total Dibayar", func(r domain.MatrixRow) int64 { return r.Paid }, true},
		{"Tagihan (Sisa)", func(r domain.MatrixRow) int64 { return r.Remaining }, true},
		{"Lebih bayar", func(r domain.MatrixRow) int64 { return r.Overpaid }, t.Overpaid != 0},
	}
	row := 3
	for _, l := range lines {
		if !l.show {
			continue
		}
		w.text(1, row, l.label)
		for i, r := range m.Rows {
			w.number(i+2, row, l.get(r))
		}
		w.number(totalCol, row, l.get(t))
		w.style(2, row, totalCol, row, st.money)
		row++
	}

	row++
	if a := m.ActiveScenario; a != nil {
		w.text(1, row, "Skenario aktif")
		w.text(2, row, a.Name)
		w.text(1, row+1, "Target skenario")
		w.number(2, row+1, a.TotalTarget)
		w.text(1, row+2, "Sisa plafon")
		w.number(2, row+2, a.Unallocated)
		w.style(2, row+1, 2, row+2, st.money)
	}

	w.keep(f.SetColWidth(sheetRekap, "A", "A", 26))
	w.keep(f.SetColWidth(sheetRekap, "B", "H", 18))

	return w.err
}

// drawAnggaran is PDF p. 5-6: one block per session with the columns No,
// Kegiatan, Tagihan (Awal), DP/Lunas, Termin 1, Termin 2, Total Dibayar, Tagihan
// (Sisa) and Status, and a Total row. A line that rolls up the shopping list
// shows its figures in Total Dibayar only and says so in Status.
func drawAnggaran(f *excelize.File, st styles, data domain.ExportData, sessionName func(string) string) error {
	w := &sheetWriter{f: f, name: sheetAnggaran}
	heads := []string{"No", "Kegiatan", "Tagihan (Awal)", "DP/Lunas", "Termin 1", "Termin 2", "Total Dibayar", "Tagihan (Sisa)", "Status"}

	bySession := map[string][]domain.BudgetLine{}
	for _, l := range data.Lines {
		bySession[l.SubEventScope] = append(bySession[l.SubEventScope], l)
	}

	row := 1
	for _, s := range data.Sessions {
		lines := bySession[s.ScopeKey]
		if len(lines) == 0 {
			continue
		}
		w.text(1, row, "ANGGARAN "+strings.ToUpper(sessionName(s.ScopeKey)))
		w.style(1, row, len(heads), row, st.band)
		row++
		for i, h := range heads {
			w.text(i+1, row, h)
		}
		w.style(1, row, len(heads), row, st.head)
		row++

		var sum domain.MatrixRow
		for i, l := range lines {
			w.number(1, row, int64(i+1))
			w.text(2, row, l.ItemName)
			for j, v := range []int64{l.Estimate, l.DPAmount, l.Term1Amount, l.Term2Amount, l.Paid, l.Remaining} {
				w.number(j+3, row, v)
			}
			w.style(3, row, 8, row, st.money)
			w.text(9, row, statusText(l))
			sum.Estimate += l.Estimate
			sum.DP += l.DPAmount
			sum.Term1 += l.Term1Amount
			sum.Term2 += l.Term2Amount
			sum.Paid += l.Paid
			sum.Remaining += l.Remaining
			row++
		}
		w.text(2, row, "Total")
		for j, v := range []int64{sum.Estimate, sum.DP, sum.Term1, sum.Term2, sum.Paid, sum.Remaining} {
			w.number(j+3, row, v)
		}
		w.style(1, row, 2, row, st.total)
		w.style(3, row, 8, row, st.moneyBold)
		w.style(9, row, 9, row, st.total)
		row += 2
	}

	w.keep(f.SetColWidth(sheetAnggaran, "A", "A", 5))
	w.keep(f.SetColWidth(sheetAnggaran, "B", "B", 40))
	w.keep(f.SetColWidth(sheetAnggaran, "C", "H", 17))
	w.keep(f.SetColWidth(sheetAnggaran, "I", "I", 24))

	return w.err
}

func statusText(l domain.BudgetLine) string {
	text := statusLabel[l.PaymentStatus]
	switch {
	case l.Estimate > 0 && l.Remaining < 0:
		text = "Lebih bayar"
	case l.RollupSource != nil:
		text += " (dihitung dari belanja)"
	}

	return text
}

// drawTabungan is PDF p. 7: the months as rows, one column per contributor and
// the sum, then the target, what is collected and the shortfall.
func drawTabungan(f *excelize.File, st styles, data domain.ExportData) error {
	w := &sheetWriter{f: f, name: sheetTabungan}
	sv := data.Savings

	w.text(1, 1, "Bulan")
	for i, c := range sv.Contributors {
		w.text(i+2, 1, c.Name)
	}
	sumCol := len(sv.Contributors) + 2
	w.text(sumCol, 1, "Jumlah")
	w.style(1, 1, sumCol, 1, st.head)

	row := 2
	for _, m := range sv.Matrix.Rows {
		w.text(1, row, m.Month)
		for i, c := range sv.Contributors {
			w.number(i+2, row, m.ByContributor[c.ID.String()])
		}
		w.number(sumCol, row, m.Total)
		w.style(2, row, sumCol, row, st.money)
		row++
	}

	w.text(1, row, "Total")
	for i, c := range sv.Contributors {
		w.number(i+2, row, c.Total)
	}
	w.number(sumCol, row, sv.KPI.Collected)
	w.style(1, row, 1, row, st.total)
	w.style(2, row, sumCol, row, st.moneyBold)
	row += 2

	for _, kv := range []struct {
		label string
		v     int64
	}{{"Target tabungan", sv.KPI.Target}, {"Terkumpul", sv.KPI.Collected}, {"Kekurangan", sv.KPI.Shortfall}} {
		w.text(1, row, kv.label)
		w.number(2, row, kv.v)
		w.style(2, row, 2, row, st.money)
		row++
	}
	if r := sv.Recommendation; r.RecommendedMonthly > 0 {
		w.text(1, row, fmt.Sprintf("Rekomendasi per bulan (%d bulan)", r.MonthsLeft))
		w.number(2, row, r.RecommendedMonthly)
		w.style(2, row, 2, row, st.money)
	}

	w.keep(f.SetColWidth(sheetTabungan, "A", "A", 32))
	w.keep(f.SetColWidth(sheetTabungan, "B", "E", 18))

	return w.err
}
