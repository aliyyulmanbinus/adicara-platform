package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

func book(t *testing.T, rows int, mutate func(f *excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]any{"Nama Tamu", "Pihak", "Kategori", "Sesi", "No Meja", "Pax", "No WhatsApp", "Catatan"})
	for i := 0; i < rows; i++ {
		r := i + 2
		f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", r), &[]any{fmt.Sprintf("Tamu nomor %d dari daftar", i), "Pria", "Teman", "Resepsi", fmt.Sprintf("T-%d", i%40), 2, fmt.Sprintf("0812%08d", i), "alergi udang"})
	}
	if mutate != nil {
		mutate(f)
	}
	b, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestAFullTemplateOf2000RowsIsReadWellInsideTheLimits(t *testing.T) {
	data := book(t, 2000, nil)
	t0 := time.Now()
	recs, err := Read(data)
	if err != nil || len(recs) != 2001 {
		t.Fatalf("err=%v records=%d", err, len(recs))
	}
	t.Logf("2000 rows: %d KiB zipped, read in %v", len(data)/1024, time.Since(t0).Round(time.Millisecond))
	if recs[1].Row != 2 || recs[2000].Row != 2001 {
		t.Errorf("row numbers: %d, %d", recs[1].Row, recs[2000].Row)
	}
}

func TestReadingStopsAfterTheRowLimitPlusOne(t *testing.T) {
	recs, err := Read(book(t, 3000, nil))
	if err != nil || len(recs) != domain.MaxImportRows+2 {
		t.Fatalf("err=%v records=%d, want %d", err, len(recs), domain.MaxImportRows+2)
	}
}

func TestNumericPhoneCellsLoseTheirLeadingZeroAndAreRefused(t *testing.T) {
	data := book(t, 0, func(f *excelize.File) {
		f.SetSheetRow("Sheet1", "A2", &[]any{"Budi", "Pria", "Teman", "Resepsi", "T-1", 2, 81200000101, ""})    // angka: Excel membuang 0
		f.SetSheetRow("Sheet1", "A3", &[]any{"Sari", "Pria", "Teman", "Resepsi", "T-2", 2, "081200000102", ""}) // teks
	})
	recs, err := Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if recs[1].Cells[6] != "81200000101" || recs[2].Cells[6] != "081200000102" {
		t.Fatalf("cells: %q %q", recs[1].Cells[6], recs[2].Cells[6])
	}
	cfg := domain.ImportConfig{Spec: domain.GuestSpec{Sides: []domain.Option{{Key: "pria", Label: "Pihak Pria (CPP)"}}}, Sessions: []domain.SessionRef{{ScopeKey: "all", Name: "Umum / Semua Sesi"}, {ScopeKey: "resepsi", Name: "Resepsi"}}}
	p, err := domain.BuildImportPlan(cfg, recs)
	if err != nil || p.Result.ValidRows != 1 || p.Result.InvalidRows != 1 || p.Result.Errors[0].Field != "phone_number" || p.Result.Errors[0].Row != 2 {
		t.Fatalf("%v %+v", err, p.Result)
	}
}

// bomb membuat .xlsx kecil yang mengklaim lembar XML sangat besar.
func bomb(t *testing.T, mib int) []byte {
	base := book(t, 1, nil)
	zr, _ := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		w, _ := zw.Create(f.Name)
		if f.Name == "xl/worksheets/sheet1.xml" {
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Nama Tamu</t></is></c></row>`)
			chunk := bytes.Repeat([]byte(`<row r="2"><c r="A2" t="inlineStr"><is><t>x</t></is></c></row>`), 1000)
			for n := 0; n < mib*1024*1024/len(chunk); n++ {
				w.Write(chunk)
			}
			io.WriteString(w, `</sheetData></worksheet>`)
			continue
		}
		rc, _ := f.Open()
		io.Copy(w, rc)
		rc.Close()
	}
	zw.Close()
	return out.Bytes()
}

func tempFiles(t *testing.T) int {
	m, _ := filepath.Glob(filepath.Join(os.TempDir(), "excelize*"))
	return len(m)
}

func TestADecompressionBombIsRefusedAndLeavesNoTempFiles(t *testing.T) {
	data := bomb(t, 300)
	before := tempFiles(t)
	t0 := time.Now()
	_, err := Read(data)
	if err == nil {
		t.Fatal("a 300 MiB sheet claimed by a ~1 MiB upload must be refused")
	}
	t.Logf("refused in %v: %v (%d KiB zipped)", time.Since(t0).Round(time.Millisecond), err, len(data)/1024)
	if after := tempFiles(t); after != before {
		t.Errorf("temp files left behind: %d -> %d", before, after)
	}
	// Di bawah batas ekstrak (32 MiB) yang sah tetap terbaca; di atas 64 MiB ditolak.
	if _, err := Read(bomb(t, 30)); err != nil {
		t.Errorf("30 MiB must still open (under the limit): %v", err)
	}
	if _, err := Read(bomb(t, 70)); err == nil {
		t.Errorf("70 MiB must be refused")
	}
}

func TestNotAnXLSX(t *testing.T) {
	if _, err := Read([]byte("hello")); err == nil {
		t.Error("expected an error")
	}
}
