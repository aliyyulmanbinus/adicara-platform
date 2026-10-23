package http

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// The spreadsheets of the two exports (04 BE 2.7). Amounts are numbers with the format "Rp" #,##0; colors are the sage
// tokens of the frontend (the same as the budget export of menu 03).
const (
	sheetContacts = "Kontak Vendor"
	sheetShopping = "Daftar Belanja"

	colorHeader = "526E4D" // sage-600
	colorTotal  = "EEF3EC"

	rupiahFormat   = `"Rp" #,##0`
	quantityFormat = `#,##0.##`
)

var itemStatusLabel = map[string]string{
	domain.ShoppingNotStarted: "Belum", domain.ShoppingInProgress: "Proses", domain.ShoppingDone: "Selesai",
}

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

type styles struct{ head, money, qty, title, total, totalMoney, link int }

func newStyles(f *excelize.File) (styles, error) {
	var st styles
	var err error
	money, qty := rupiahFormat, quantityFormat
	total := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorTotal}}
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
	if st.qty, err = f.NewStyle(&excelize.Style{CustomNumFmt: &qty}); err != nil {
		return st, err
	}
	if st.title, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12, Color: colorHeader}}); err != nil {
		return st, err
	}
	if st.total, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: total}); err != nil {
		return st, err
	}
	if st.totalMoney, err = f.NewStyle(&excelize.Style{CustomNumFmt: &money, Font: &excelize.Font{Bold: true}, Fill: total}); err != nil {
		return st, err
	}
	st.link, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "0563C1", Underline: "single"}})

	return st, err
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

// buildContactsWorkbook is the sheet for the organiser or the committee: the fixed vendors, in order, with the contact
// only. No price, no limit, no note.
func buildContactsWorkbook(data domain.VendorExport) (*excelize.File, error) {
	f, st, err := newBook(sheetContacts)
	if err != nil {
		return nil, err
	}
	w := &sheetWriter{f: f, name: sheetContacts}
	names := sessionNames(data.Sessions)

	w.text(1, 1, "Kontak Vendor — "+data.Project.Title)
	w.style(1, 1, 1, 1, st.title)
	header(w, st, 3, []string{"No", "Sesi", "Kategori", "Vendor", "PIC", "Telepon", "WhatsApp"})
	row := 4
	for i, v := range data.Vendors {
		w.number(1, row, i+1)
		w.text(2, row, sessionName(names, v.SubEventScope))
		w.text(3, row, v.CategoryLabel)
		w.text(4, row, v.VendorName)
		if v.PICName != nil {
			w.text(5, row, *v.PICName)
		}
		if v.PhoneNumber != nil {
			w.text(6, row, *v.PhoneNumber)
		}
		if v.FormattedWANumber != nil {
			cell := w.cell(7, row)
			w.text(7, row, *v.FormattedWANumber)
			w.keep(f.SetCellHyperLink(sheetContacts, cell, "https://wa.me/"+*v.FormattedWANumber, "External"))
			w.style(7, row, 7, row, st.link)
		}
		row++
	}
	for col, width := range []float64{6, 16, 22, 34, 22, 18, 18} {
		name, _ := excelize.ColumnNumberToName(col + 1)
		w.keep(f.SetColWidth(sheetContacts, name, name, width))
	}

	return f, w.err
}

// buildShoppingWorkbook is the whole shopping list in order, with a TOTAL row: the estimate of every item, the actual
// total of the finished ones and the count per status.
func buildShoppingWorkbook(data domain.ShoppingExport) (*excelize.File, error) {
	f, st, err := newBook(sheetShopping)
	if err != nil {
		return nil, err
	}
	w := &sheetWriter{f: f, name: sheetShopping}
	names := sessionNames(data.Sessions)

	w.text(1, 1, "Daftar Belanja — "+data.Project.Title)
	w.style(1, 1, 1, 1, st.title)
	header(w, st, 3, []string{"No", "Sesi", "Kategori", "Barang", "Jumlah", "Satuan", "Harga satuan estimasi", "Total estimasi",
		"Harga satuan riil", "Total riil", "Selisih", "Alamat pembelian", "Status", "Catatan"})
	row := 4
	for i, it := range data.Items {
		w.number(1, row, i+1)
		w.text(2, row, sessionName(names, it.SubEventScope))
		w.text(3, row, it.Category)
		w.text(4, row, it.ItemName)
		w.number(5, row, it.Quantity)
		if it.Unit != nil {
			w.text(6, row, *it.Unit)
		}
		w.number(7, row, it.EstimatedPrice)
		w.number(8, row, it.EstimatedTotal)
		w.number(9, row, it.ActualPrice)
		w.number(10, row, it.ActualTotal)
		if it.PriceDelta != nil {
			w.number(11, row, *it.PriceDelta)
		}
		if it.PurchaseLocation != nil {
			w.text(12, row, *it.PurchaseLocation)
		}
		w.text(13, row, itemStatusLabel[it.Status])
		if it.Notes != nil {
			w.text(14, row, *it.Notes)
		}
		row++
	}
	if len(data.Items) > 0 {
		w.style(5, 4, 5, row-1, st.qty)
		w.style(7, 4, 11, row-1, st.money)
	}

	w.text(1, row, "TOTAL")
	w.number(8, row, data.KPI.EstimatedTotal)
	w.number(10, row, data.KPI.ActualTotal)
	w.text(13, row, fmt.Sprintf("%d selesai · %d proses · %d belum", data.KPI.Done, data.KPI.InProgress, data.KPI.NotStarted))
	w.style(1, row, 14, row, st.total)
	w.style(8, row, 8, row, st.totalMoney)
	w.style(10, row, 10, row, st.totalMoney)
	for col, width := range []float64{6, 16, 24, 34, 9, 9, 16, 16, 16, 16, 14, 34, 28, 30} {
		name, _ := excelize.ColumnNumberToName(col + 1)
		w.keep(f.SetColWidth(sheetShopping, name, name, width))
	}

	return f, w.err
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
