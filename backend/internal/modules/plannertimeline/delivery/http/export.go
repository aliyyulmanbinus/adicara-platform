package http

import (
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

// The spreadsheet of GET /tasks/export-spreadsheet (02 BE 4.4): one sheet named
// Timeline with the columns No, Tugas, Rangkaian Acara, Mulai, Selesai, Durasi,
// Status, then one column per week (Monday to Sunday) filled where a task is
// active. Colors are the sage tokens of the frontend (tokens.css).
const (
	sheetName = "Timeline"
	firstWeek = 8 // column H

	colorHeader = "526E4D" // sage-600
	colorDone   = "526E4D"
	colorDoing  = "97B394" // sage-400
	colorTodo   = "D1DFCE" // sage-200
)

var statusLabel = map[domain.TaskStatus]string{
	domain.TaskBelum: "Belum", domain.TaskProses: "Proses", domain.TaskSelesai: "Selesai",
}

var statusColor = map[domain.TaskStatus]string{
	domain.TaskBelum: colorTodo, domain.TaskProses: colorDoing, domain.TaskSelesai: colorDone,
}

// safeCell implements the contract rule against formula injection (kontrak
// bab 2 butir 5): text that starts with = + - @ gets a leading apostrophe. excelize
// stores user text as a string, never as a formula, so this is a second layer;
// the apostrophe is written as a character and therefore visible.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@", rune(s[0])) {
		return "'" + s
	}

	return s
}

// mondayOf is the Monday of the ISO week of d (00:00 UTC).
func mondayOf(d time.Time) time.Time { return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7)) }

func weekIndex(first, d time.Time) int { return int(mondayOf(d).Sub(first).Hours()/24) / 7 }

// buildWorkbook draws all tasks, in sort_order. A task fills every week that
// its inclusive range [start, end] touches.
func buildWorkbook(data domain.ExportData) (*excelize.File, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, err
	}

	head, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorHeader}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	dayFmt := "dd mmm yyyy"
	dayStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &dayFmt})
	if err != nil {
		return nil, err
	}
	weekFmt := "dd/mm"
	weekHead, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &weekFmt,
		Font:         &excelize.Font{Bold: true, Color: "FFFFFF", Size: 8},
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorHeader}},
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center", TextRotation: 90},
	})
	if err != nil {
		return nil, err
	}
	fills := map[domain.TaskStatus]int{}
	for status, color := range statusColor {
		if fills[status], err = f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{color}}}); err != nil {
			return nil, err
		}
	}

	for i, h := range []string{"No", "Tugas", "Rangkaian Acara", "Mulai", "Selesai", "Durasi (hari)", "Status"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellStr(sheetName, cell, h); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(sheetName, cell, cell, head); err != nil {
			return nil, err
		}
	}

	sessionName := make(map[string]string, len(data.Sessions))
	for _, s := range data.Sessions {
		sessionName[s.ScopeKey] = s.Name
	}

	weeks := 0
	var first time.Time
	if len(data.Tasks) > 0 {
		lo, hi := data.Tasks[0].StartDate, data.Tasks[0].EndDate
		for _, t := range data.Tasks {
			if t.StartDate.Before(lo) {
				lo = t.StartDate
			}
			if t.EndDate.After(hi) {
				hi = t.EndDate
			}
		}
		first = mondayOf(lo)
		weeks = weekIndex(first, hi) + 1
	}
	for w := 0; w < weeks; w++ {
		cell, _ := excelize.CoordinatesToCellName(firstWeek+w, 1)
		if err := f.SetCellValue(sheetName, cell, first.AddDate(0, 0, 7*w)); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(sheetName, cell, cell, weekHead); err != nil {
			return nil, err
		}
		col, _ := excelize.ColumnNumberToName(firstWeek + w)
		if err := f.SetColWidth(sheetName, col, col, 3); err != nil {
			return nil, err
		}
	}

	for i, t := range data.Tasks {
		row := i + 2
		at := func(col int) string { c, _ := excelize.CoordinatesToCellName(col, row); return c }
		session := sessionName[t.SubEventScope]
		if session == "" {
			session = t.SubEventScope
		}

		var werr error
		set := func(err error) {
			if werr == nil {
				werr = err
			}
		}
		set(f.SetCellValue(sheetName, at(1), i+1))
		set(f.SetCellStr(sheetName, at(2), safeCell(t.Title)))
		set(f.SetCellStr(sheetName, at(3), safeCell(session)))
		set(f.SetCellValue(sheetName, at(4), t.StartDate))
		set(f.SetCellValue(sheetName, at(5), t.EndDate))
		set(f.SetCellStyle(sheetName, at(4), at(5), dayStyle))
		set(f.SetCellValue(sheetName, at(6), t.DurationDays))
		set(f.SetCellStr(sheetName, at(7), statusLabel[t.Status]))
		for w := weekIndex(first, t.StartDate); w <= weekIndex(first, t.EndDate); w++ {
			cell := at(firstWeek + w)
			set(f.SetCellStyle(sheetName, cell, cell, fills[t.Status]))
		}
		if werr != nil {
			return nil, werr
		}
	}

	widths := []struct {
		from, to string
		w        float64
	}{{"A", "A", 5}, {"B", "B", 46}, {"C", "C", 22}, {"D", "E", 13}, {"F", "G", 12}}
	for _, c := range widths {
		if err := f.SetColWidth(sheetName, c.from, c.to, c.w); err != nil {
			return nil, err
		}
	}
	if err := f.SetRowHeight(sheetName, 1, 36); err != nil {
		return nil, err
	}
	// Freeze the header row and the No and Tugas columns.
	if err := f.SetPanes(sheetName, &excelize.Panes{Freeze: true, XSplit: 2, YSplit: 1, TopLeftCell: "C2", ActivePane: "bottomRight"}); err != nil {
		return nil, err
	}

	return f, nil
}
