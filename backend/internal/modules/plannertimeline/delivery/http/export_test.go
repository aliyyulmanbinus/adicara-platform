package http

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

func date(s string) time.Time {
	t, err := time.Parse(pd.DateLayout, s)
	if err != nil {
		panic(err)
	}

	return t
}

// sampleExport builds the 50 tasks of the sample project (Hari-H 28 Mar 2027,
// created on 4 Oct 2026) from the real templates and the real scaling.
func sampleExport(t *testing.T) domain.ExportData {
	t.Helper()
	sched, ok := domain.BuildSchedule(domain.PresetWeddingPDF44, pd.EventWedding, true)
	if !ok {
		t.Fatal("no schedule")
	}
	plan, _ := domain.NewPlan(date("2027-03-28"), date("2026-10-04"), date("2026-10-04"), 180, sched.Horizon)

	var tasks []domain.Task
	for i, tt := range sched.Tasks {
		s, e := plan.At(tt.Start, tt.End)
		status := domain.TaskBelum
		switch {
		case i < 15:
			status = domain.TaskSelesai
		case i < 44:
			status = domain.TaskProses
		}
		tasks = append(tasks, domain.Task{
			ID: uuid.New(), Title: tt.Title, SubEventScope: tt.Scope, StartDate: s, EndDate: e,
			DurationDays: max(int(e.Sub(s).Hours()/24), 1), Status: status, SortOrder: i + 1,
		})
	}

	return domain.ExportData{
		Project: pd.Project{Title: "Contoh"}, Tasks: tasks, Today: date("2026-10-04"),
		Sessions: []domain.SessionRef{{ScopeKey: "all", Name: "Umum / Semua Sesi"}, {ScopeKey: "lamaran", Name: "Lamaran"}, {ScopeKey: "akad", Name: "Akad Nikah"}, {ScopeKey: "resepsi", Name: "Resepsi"}},
	}
}

func TestSafeCellPrefixesFormulaStarters(t *testing.T) {
	for in, want := range map[string]string{
		"=cmd|' /C calc'!A0": "'=cmd|' /C calc'!A0", "+SUM(1+1)": "'+SUM(1+1)", "-2+3": "'-2+3", "@SUM(A1)": "'@SUM(A1)",
		"": "", "Survei venue": "Survei venue", "a=b": "a=b", "é=1": "é=1", "'=already": "'=already",
	} {
		if got := safeCell(in); got != want {
			t.Errorf("safeCell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMondayOf(t *testing.T) {
	for in, want := range map[string]string{
		"2026-10-04": "2026-09-28", // Sunday
		"2026-10-05": "2026-10-05", // Monday
		"2027-03-28": "2027-03-22",
		"2024-02-29": "2024-02-26", // leap day
		"2026-02-01": "2026-01-26", // "1 Feb 2026" of the PDF
	} {
		if got := mondayOf(date(in)).Format(pd.DateLayout); got != want {
			t.Errorf("mondayOf(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestWorkbookOfTheSampleProject(t *testing.T) {
	data := sampleExport(t)
	// Hostile titles: they must come out as text, never as formulas.
	data.Tasks[0].Title = "=cmd|' /C calc'!A0"
	data.Tasks[1].Title = "+SUM(1+1)"
	data.Tasks[2].Title = "-2+3"
	data.Tasks[3].Title = "@SUM(A1)"

	f, err := buildWorkbook(data)
	if err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	g, err := excelize.OpenReader(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}

	if sheets := g.GetSheetList(); len(sheets) != 1 || sheets[0] != "Timeline" {
		t.Fatalf("sheets = %v", sheets)
	}
	rows, _ := g.GetRows("Timeline")
	if len(rows) != 51 {
		t.Fatalf("%d rows, want header + 50 tasks", len(rows))
	}
	if got := strings.Join(rows[0][:7], "|"); got != "No|Tugas|Rangkaian Acara|Mulai|Selesai|Durasi (hari)|Status" {
		t.Fatalf("header = %s", got)
	}

	for i, want := range []string{"'=cmd|' /C calc'!A0", "'+SUM(1+1)", "'-2+3", "'@SUM(A1)"} {
		cell := fmt.Sprintf("B%d", i+2)
		v, _ := g.GetCellValue("Timeline", cell)
		formula, _ := g.GetCellFormula("Timeline", cell)
		typ, _ := g.GetCellType("Timeline", cell)
		if v != want || formula != "" || typ != excelize.CellTypeSharedString {
			t.Errorf("%s = %q formula %q type %v", cell, v, formula, typ)
		}
	}
	if v, _ := g.GetCellValue("Timeline", "B6"); strings.HasPrefix(v, "'") {
		t.Errorf("an ordinary title must not get an apostrophe: %q", v)
	}

	// No formula element anywhere in the file.
	zr, err := zip.NewReader(strings.NewReader(buf.String()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, zf := range zr.File {
		if strings.HasPrefix(zf.Name, "xl/worksheets/") {
			rc, _ := zf.Open()
			body, _ := io.ReadAll(rc)
			rc.Close()
			if strings.Contains(string(body), "<f>") || strings.Contains(string(body), "<f ") {
				t.Errorf("%s contains a formula", zf.Name)
			}
		}
	}

	// Row 2: session name, a real date (serial of 4 Oct 2026), duration, status label.
	if v, _ := g.GetCellValue("Timeline", "C2"); v != "Lamaran" {
		t.Errorf("C2 = %q", v)
	}
	if raw, _ := g.GetCellValue("Timeline", "D2", excelize.Options{RawCellValue: true}); raw != "46299" {
		t.Errorf("D2 raw = %q, want the serial number of 4 Oct 2026", raw)
	}
	if v, _ := g.GetCellValue("Timeline", "G2"); v != "Selesai" {
		t.Errorf("G2 = %q", v)
	}
	if v, _ := g.GetCellValue("Timeline", "F4"); v != "2" {
		t.Errorf("F4 = %q, task 3 lasts two days", v)
	}

	// Week headers: 28 Sep 2026 .. 22 Mar 2027 = 26 columns (H .. AG).
	first, _ := g.GetCellValue("Timeline", "H1")
	last, _ := g.GetCellValue("Timeline", "AG1")
	beyond, _ := g.GetCellValue("Timeline", "AH1")
	if first != "28/09" || last != "22/03" || beyond != "" {
		t.Errorf("week headers: first %q last %q beyond %q", first, last, beyond)
	}

	// The Gantt against a day-by-day computation (a different algorithm from the builder's).
	fill := map[domain.TaskStatus]string{domain.TaskBelum: colorTodo, domain.TaskProses: colorDoing, domain.TaskSelesai: colorDone}
	firstMonday := date("2026-09-28")
	filled := 0
	for i, tk := range data.Tasks {
		weeks := map[int]bool{}
		for d := tk.StartDate; !d.After(tk.EndDate); d = d.AddDate(0, 0, 1) {
			weeks[int(d.Sub(firstMonday).Hours()/24)/7] = true
		}
		for w := 0; w < 26; w++ {
			cell, _ := excelize.CoordinatesToCellName(firstWeek+w, i+2)
			id, _ := g.GetCellStyle("Timeline", cell)
			st, _ := g.GetStyle(id)
			got := ""
			if st != nil && len(st.Fill.Color) > 0 {
				got = st.Fill.Color[0]
			}
			want := ""
			if weeks[w] {
				want = fill[tk.Status]
				filled++
			}
			if got != want {
				t.Fatalf("task %d week %d: fill %q, want %q", i+1, w, got, want)
			}
		}
	}
	if filled != 61 {
		t.Errorf("%d filled cells, want 61", filled)
	}
}

func TestWorkbookWithoutTasksHasOnlyTheHeader(t *testing.T) {
	f, err := buildWorkbook(domain.ExportData{Project: pd.Project{Title: "Kosong"}, Today: date("2026-10-04")})
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := f.GetRows("Timeline")
	if len(rows) != 1 || len(rows[0]) != 7 {
		t.Fatalf("rows = %v", rows)
	}
}

func TestWorkbookKeepsASessionNameThatIsNoLongerKnown(t *testing.T) {
	data := domain.ExportData{
		Project: pd.Project{Title: "x"}, Today: date("2026-10-04"),
		Tasks: []domain.Task{{Title: "t", SubEventScope: "gone", StartDate: date("2026-10-04"), EndDate: date("2026-10-04"), DurationDays: 1, Status: domain.TaskBelum}},
	}
	f, err := buildWorkbook(data)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.GetCellValue("Timeline", "C2"); v != "gone" {
		t.Errorf("C2 = %q", v)
	}
}
