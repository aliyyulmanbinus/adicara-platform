package xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

const (
	sheetData  = "Tamu"
	sheetGuide = "Petunjuk"
)

// Template makes the empty import file: exactly the column headers BuildImportPlan recognises, dropdowns from the
// registry and the session master (the lists sit on the Petunjuk sheet because Excel limits an inline list formula to
// 255 characters), phone columns typed as text (Excel then keeps the leading 0) and no sample row (a sample row
// would be imported if it were not deleted).
func Template(in domain.TemplateInput) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", sheetData); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(sheetGuide); err != nil {
		return nil, err
	}

	headers := []string{"Nama " + in.GuestWord}
	if len(in.Sides) > 0 {
		headers = append(headers, "Pihak")
	}
	headers = append(headers, "Kategori", "Sesi", "No Meja", "Pax", "No WhatsApp", "Catatan")
	col := map[string]string{}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellStr(sheetData, cell, h); err != nil {
			return nil, err
		}
		col[h], _ = excelize.ColumnNumberToName(i + 1)
	}
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	last, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(sheetData, "A1", last+"1", bold)

	text, _ := f.NewStyle(&excelize.Style{NumFmt: 49}) // "@": text
	for _, h := range []string{"No WhatsApp", "No Meja"} {
		if err := f.SetColStyle(sheetData, col[h], text); err != nil {
			return nil, err
		}
	}
	for i := range headers {
		w := 18.0
		if i == 0 {
			w = 34
		}
		c, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetData, c, c, w)
	}

	// Petunjuk sheet: the value lists in columns A (side), B (category), C (session), then the notes on the columns.
	_ = f.SetSheetRow(sheetGuide, "A1", &[]any{"Pihak", "Kategori", "Sesi"})
	_ = f.SetCellStyle(sheetGuide, "A1", "C1", bold)
	for i, o := range in.Sides {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetCellStr(sheetGuide, cell, o.Label)
	}
	for i, o := range in.Cats {
		cell, _ := excelize.CoordinatesToCellName(2, i+2)
		_ = f.SetCellStr(sheetGuide, cell, o.Label)
	}
	for i, s := range in.Sessions {
		cell, _ := excelize.CoordinatesToCellName(3, i+2)
		_ = f.SetCellStr(sheetGuide, cell, s)
	}
	notes := []string{
		"Isi satu baris per " + lower(in.GuestWord) + " pada lembar " + sheetData + "; hanya kolom pertama yang wajib.",
		"Pax: 1-20 (kosong = 1). No WhatsApp: 08..., +62..., atau 62...; kolom ini bertipe teks supaya 0 di depan tidak hilang.",
		"Kategori boleh memakai nama sendiri (kategori baru). Pihak dan Sesi harus salah satu nilai di kolom ini; Sesi kosong = Umum.",
		"Hapus baris yang tidak dipakai; baris kosong dilewati. Maksimal 2.000 baris dan 5 MB.",
	}
	for i, n := range notes {
		cell, _ := excelize.CoordinatesToCellName(5, i+1)
		_ = f.SetCellStr(sheetGuide, cell, n)
	}

	// Dropdowns: data column -> list on the Petunjuk sheet. Category is not strict (a new category is allowed).
	dropdown := func(header, listCol string, n int, strict bool) error {
		if n == 0 {
			return nil
		}
		c := col[header]
		v := excelize.NewDataValidation(true)
		v.SetSqref(fmt.Sprintf("%s2:%s%d", c, c, domain.MaxImportRows+1))
		v.SetSqrefDropList(fmt.Sprintf("%s!$%s$2:$%s$%d", sheetGuide, listCol, listCol, n+1))
		v.ShowErrorMessage = strict
		return f.AddDataValidation(sheetData, v)
	}
	for _, d := range []struct {
		header, list string
		n            int
		strict       bool
	}{{"Pihak", "A", len(in.Sides), true}, {"Kategori", "B", len(in.Cats), false}, {"Sesi", "C", len(in.Sessions), true}} {
		if err := dropdown(d.header, d.list, d.n, d.strict); err != nil {
			return nil, err
		}
	}
	pax := excelize.NewDataValidation(true)
	paxCol := col["Pax"]
	pax.SetSqref(fmt.Sprintf("%s2:%s%d", paxCol, paxCol, domain.MaxImportRows+1))
	_ = pax.SetRange(1, 20, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween)
	pax.SetError(excelize.DataValidationErrorStyleStop, "Pax tidak valid", "Isi bilangan bulat 1 sampai 20.")
	if err := f.AddDataValidation(sheetData, pax); err != nil {
		return nil, err
	}

	f.SetActiveSheet(0)
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func lower(s string) string {
	if s == "Tamu" {
		return "tamu"
	}
	return "peserta"
}
