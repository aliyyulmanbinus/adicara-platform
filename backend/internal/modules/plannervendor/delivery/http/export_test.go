package http_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func openBook(t *testing.T, r resp) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(r.body))
	if err != nil {
		t.Fatalf("not a workbook: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func rawCell(t *testing.T, f *excelize.File, sheet, cell string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func TestContactSheetOfTheSampleProject(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	r := mustStatus(t, e.do(tok, "GET", base(id)+"/vendors/export-contacts?format=xlsx", nil), 200)
	if r.header.Get("Content-Type") != xlsxType || r.header.Get("Cache-Control") != "no-store" ||
		r.header.Get("Content-Disposition") != `attachment; filename="vendor-contoh-pernikahan-sinta-arif-2026-10-04.xlsx"` {
		t.Errorf("headers: %v", r.header)
	}
	f := openBook(t, r)
	if got := f.GetSheetList(); len(got) != 1 || got[0] != "Kontak Vendor" {
		t.Fatalf("sheets = %v", got)
	}
	rows, err := f.GetRows("Kontak Vendor")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3+7 { // title, blank, header, the 7 fixed vendors (the 3 in the pipeline are not on the sheet)
		t.Fatalf("%d rows: %v", len(rows), rows)
	}
	if rows[0][0] != "Kontak Vendor — Contoh: Pernikahan Sinta & Arif" || strings.Join(rows[2], "|") != "No|Sesi|Kategori|Vendor|PIC|Telepon|WhatsApp" {
		t.Errorf("head = %v / %v", rows[0], rows[2])
	}
	first := rows[3]
	if first[0] != "1" || first[1] != "Resepsi" || first[2] != "Venue" || first[3] != "Sasana Kriya Ballroom" || first[5] != "081200000001" || first[6] != "6281200000001" {
		t.Errorf("first vendor = %v", first)
	}
	if ok, link, err := f.GetCellHyperLink("Kontak Vendor", "G4"); err != nil || !ok || link != "https://wa.me/6281200000001" {
		t.Errorf("link = %v %q %v", ok, link, err)
	}

	// the sheet goes to the organiser: no price, no limit, no note anywhere
	for _, row := range rows {
		for _, c := range row {
			for _, secret := range []string{"000.000", "Harga", "Pagu", "Catatan", "negosiasi", "Fleur", "Kang Dedi"} {
				if strings.Contains(c, secret) {
					t.Errorf("the contact sheet leaks %q in %q", secret, c)
				}
			}
		}
	}
}

func TestContactSheetEscapesFormulas(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"title": `=Pesta "Ali" ✨ / ../x`})
	v := addVendor(e, tok, id, vendorBody("mua", `=HYPERLINK("http://evil","klik")`, map[string]any{"pic_name": "+62 attack", "agreed_price": 1_000_000}))
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+v["id"].(string)+"/status", map[string]any{"status": "fix"}), 200)

	r := mustStatus(t, e.do(tok, "GET", base(id)+"/vendors/export-contacts?format=xlsx", nil), 200)
	if cd := r.header.Get("Content-Disposition"); cd != `attachment; filename="vendor-pesta-ali-x-2026-10-04.xlsx"` {
		t.Errorf("Content-Disposition = %q (the title must reach the header only as a slug)", cd)
	}
	f := openBook(t, r)
	name, _ := f.GetCellValue("Kontak Vendor", "D4")
	pic, _ := f.GetCellValue("Kontak Vendor", "E4")
	title, _ := f.GetCellValue("Kontak Vendor", "A1")
	if name != `'=HYPERLINK("http://evil","klik")` || pic != "'+62 attack" || !strings.HasPrefix(title, "Kontak Vendor — ") {
		t.Errorf("name %q, pic %q, title %q", name, pic, title)
	}
	if formula, _ := f.GetCellFormula("Kontak Vendor", "D4"); formula != "" {
		t.Errorf("a vendor name became a formula: %q", formula)
	}
}

func TestShoppingSheetOfTheSampleProject(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	r := mustStatus(t, e.do(tok, "GET", base(id)+"/shopping-items/export-shopping-list?format=xlsx", nil), 200)
	if r.header.Get("Content-Type") != xlsxType || r.header.Get("Cache-Control") != "no-store" ||
		r.header.Get("Content-Disposition") != `attachment; filename="belanja-contoh-pernikahan-sinta-arif-2026-10-04.xlsx"` {
		t.Errorf("headers: %v", r.header)
	}
	f := openBook(t, r)
	if got := f.GetSheetList(); len(got) != 1 || got[0] != "Daftar Belanja" {
		t.Fatalf("sheets = %v", got)
	}
	rows, _ := f.GetRows("Daftar Belanja")
	if len(rows) != 3+37+1 {
		t.Fatalf("%d rows", len(rows))
	}
	if h := strings.Join(rows[2], "|"); h != "No|Sesi|Kategori|Barang|Jumlah|Satuan|Harga satuan estimasi|Total estimasi|Harga satuan riil|Total riil|Selisih|Alamat pembelian|Status|Catatan" {
		t.Errorf("header = %s", h)
	}
	if first := rows[3]; first[3] != "Mukena" || first[1] != "Umum" || first[12] != "Selesai" || first[2] != "Perangkat Alat Solat" {
		t.Errorf("first row = %v", first)
	}
	// amounts are numbers, not text: the raw value is the integer
	for cell, want := range map[string]string{"G4": "194000", "H4": "194000", "I4": "194000", "J4": "194000", "K4": "0", "E4": "1", "H5": "70500"} {
		if got := rawCell(t, f, "Daftar Belanja", cell); got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}
	// the TOTAL row: Σ estimate of every item, Σ actual of the finished ones, the count per status
	total := 3 + 37 + 1
	label, _ := f.GetCellValue("Daftar Belanja", "A41")
	status, _ := f.GetCellValue("Daftar Belanja", "M41")
	if label != "TOTAL" || rawCell(t, f, "Daftar Belanja", "H41") != "4578500" || rawCell(t, f, "Daftar Belanja", "J41") != "2063800" || status != "16 selesai · 21 proses · 0 belum" {
		t.Errorf("total row %d: %q %q %q", total, label, rawCell(t, f, "Daftar Belanja", "H41"), status)
	}
}

func TestExportRejectsOtherFormatsAndIsRateLimited(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	for _, path := range []string{"/vendors/export-contacts?format=pdf", "/vendors/export-contacts?format=XLSX", "/shopping-items/export-shopping-list?format=csv", "/shopping-items/export-shopping-list?format=pdf"} {
		wantError(t, e.do(tok, "GET", base(id)+path, nil), 400, "invalid_input", "format: must be xlsx")
	}

	// 10 requests a minute per user, both routes together; the limiter runs before the handler, so the 4 calls above
	// that were refused for their format were counted as well and 6 are left
	ok := 0
	for i := 0; i < 12; i++ {
		path := "/vendors/export-contacts?format=xlsx"
		if i%2 == 1 {
			path = "/shopping-items/export-shopping-list?format=xlsx"
		}
		switch r := e.do(tok, "GET", base(id)+path, nil); r.status {
		case 200:
			ok++
		case 429:
			if r.code() != "rate_limited" {
				t.Errorf("429 with code %q", r.code())
			}
		default:
			t.Fatalf("export %d: %d %s", i, r.status, r.body)
		}
	}
	if ok != 6 {
		t.Errorf("%d exports went through, want 6", ok)
	}
	// another user has his own count
	other := e.login("other")
	pid := createSample(e, other)
	mustStatus(t, e.do(other, "GET", base(pid)+"/vendors/export-contacts?format=xlsx", nil), 200)
}
