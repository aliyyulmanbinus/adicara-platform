package http_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// importSample is the CSV of the reference fixture (05 BE 6): made by Excel with Indonesian settings (BOM, semicolons,
// CRLF). Of its six rows two are invalid and two are duplicates.
const importSample = "\xef\xbb\xbfNama Tamu;Pihak;Kategori;Sesi;No Meja;Pax;No WhatsApp;Catatan\r\n" +
	"Dewi Anggraini;Wanita;Teman;Resepsi;T-20;2;081200000201;\r\n" +
	"Hendra Wijaya;Pria;Kolega Kantor;Resepsi;;1;81200000202;\r\n" +
	"Maya Sari;Laki-laki;Teman;Resepsi;;2;;\r\n" +
	"Raditya Pratama & Pasangan;Pria;Teman;Resepsi;T-04;2;081200000102;\r\n" +
	"dewi  anggraini;Wanita;Teman;Resepsi;T-20;2;+62 812-0000-0201;\r\n" +
	"Sinta Maharani;Wanita;Keluarga;Akad Nikah;FAM-03;3;+62 812-0000-0203;Kursi roda\r\n"

func importBody(filename string, content []byte) map[string]any {
	return map[string]any{"filename": filename, "content_base64": base64.StdEncoding.EncodeToString(content)}
}

func importCSV(e *env, tok, id, csv, query string) resp {
	e.t.Helper()
	if query != "" {
		query = "?" + query
	}

	return e.do(tok, "POST", guests(id)+"/bulk-import"+query, importBody("tamu.csv", []byte(csv)))
}

func TestBulkImportDryRunWritesNothing(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	before := jsonOf(guestList(e, tok, id, "limit=500"))

	r := mustStatus(t, importCSV(e, tok, id, importSample, "dry_run=true"), 200).data()
	if r["dry_run"] != true || num(r["total_rows"]) != 6 || num(r["valid_rows"]) != 4 || num(r["invalid_rows"]) != 2 || num(r["created"]) != 0 || num(r["skipped"]) != 0 {
		t.Fatalf("result = %v", r)
	}
	errs := list(r["errors"])
	if len(errs) != 2 || num(errs[0]["row"]) != 3 || errs[0]["field"] != "phone_number" || errs[0]["reason"] != "must be an Indonesian phone number (08…, +62…)" ||
		num(errs[1]["row"]) != 4 || errs[1]["field"] != "guest_side" || errs[1]["reason"] != "must be a registry side key" {
		t.Errorf("errors = %v", errs)
	}
	dups := list(r["duplicates"])
	if len(dups) != 2 || num(dups[0]["row"]) != 5 || dups[0]["of_guest"] == nil || num(dups[1]["row"]) != 6 || num(dups[1]["of_row"]) != 2 {
		t.Errorf("duplicates = %v", dups)
	}
	if of := dups[0]["of_guest"].(string); of != guestNamed(t, e, tok, id, "Raditya Pratama & Pasangan")["id"] {
		t.Errorf("row 5 is a duplicate of %s", of)
	}
	preview := list(r["preview"])
	if len(preview) != 4 || preview[0]["name"] != "Dewi Anggraini" || preview[0]["guest_side"] != "wanita" || preview[0]["category"] != "teman" ||
		preview[0]["sub_event_scope"] != "resepsi" || preview[0]["table_number"] != "T-20" || num(preview[0]["pax_count"]) != 2 || preview[0]["formatted_wa_number"] != "6281200000201" {
		t.Errorf("preview = %v", preview)
	}
	if preview[3]["sub_event_scope"] != "akad" { // "Akad Nikah" is the name of the session, akad its key
		t.Errorf("session of row 7 = %v", preview[3]["sub_event_scope"])
	}
	if after := jsonOf(guestList(e, tok, id, "limit=500")); after != before {
		t.Error("a dry run changed the list")
	}
	// The lists are arrays even when empty.
	clean := mustStatus(t, importCSV(e, tok, id, "Nama Tamu\nSatu\n", "dry_run=true"), 200)
	if !strings.Contains(string(clean.body), `"errors":[]`) || !strings.Contains(string(clean.body), `"duplicates":[]`) {
		t.Errorf("empty lists are not arrays: %s", clean.body)
	}
}

func TestBulkImportIsAtomic(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	before := jsonOf(guestList(e, tok, id, "limit=500"))

	wantError(t, importCSV(e, tok, id, importSample, ""), 400, "invalid_input", "rows: 2 rows are invalid")
	if after := jsonOf(guestList(e, tok, id, "limit=500")); after != before {
		t.Error("a refused import wrote rows")
	}
}

func TestBulkImportFlags(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	other := createSample2(e, tok) // a copy of the twenty guests, made before the first import changes the sample

	// skip_invalid writes the four valid rows, duplicates included (they are only reported).
	r := mustStatus(t, importCSV(e, tok, id, importSample, "skip_invalid=true"), 200).data()
	if r["dry_run"] != false || num(r["created"]) != 4 || num(r["skipped"]) != 2 || num(r["invalid_rows"]) != 2 {
		t.Fatalf("skip_invalid = %v", r)
	}
	all := list(guestList(e, tok, id, "limit=500")["guests"])
	if len(all) != 24 {
		t.Fatalf("%d guests, want 24", len(all))
	}
	// The order of the file is the order of the list, after the existing guests.
	var added []string
	for _, g := range all[20:] {
		added = append(added, g["name"].(string))
	}
	if got := strings.Join(added, "|"); got != "Dewi Anggraini|Raditya Pratama & Pasangan|dewi anggraini|Sinta Maharani" {
		t.Errorf("added = %s", got)
	}
	sinta := all[23]
	if sinta["sub_event_scope"] != "akad" || num(sinta["pax_count"]) != 3 || sinta["notes"] != "Kursi roda" || sinta["formatted_wa_number"] != "6281200000203" ||
		sinta["phone_number"] != "+62 812-0000-0203" || sinta["rsvp_status"] != "unconfirmed" || len(sinta["qr_code_token"].(string)) != 32 {
		t.Errorf("Sinta = %v", sinta)
	}

	// skip_duplicates leaves the duplicates out too: of the six rows only the two that are new and valid come in.
	r = mustStatus(t, importCSV(e, tok, other, importSample, "skip_invalid=true&skip_duplicates=true"), 200).data()
	if num(r["created"]) != 2 || num(r["skipped"]) != 4 || num(r["valid_rows"]) != 4 {
		t.Errorf("both flags = %v", r)
	}
	// The same file twice: the second time every valid row is a duplicate of a guest.
	r = mustStatus(t, importCSV(e, tok, other, importSample, "skip_invalid=true&skip_duplicates=true"), 200).data()
	if num(r["created"]) != 0 || len(list(r["duplicates"])) != 4 {
		t.Errorf("repeated import = %v", r)
	}

	wantError(t, importCSV(e, tok, id, importSample, "dry_run=yes"), 400, "invalid_input", "dry_run: must be true or false")
	wantError(t, importCSV(e, tok, id, importSample, "skip_invalid=1"), 400, "invalid_input", "skip_invalid: must be true or false")
	wantError(t, importCSV(e, tok, id, importSample, "skip_duplicates=TRUE"), 400, "invalid_input", "skip_duplicates: must be true or false")
}

// createSample2 makes a second project with the same twenty guests (a user has one sample project, so it is a copy).
func createSample2(e *env, tok string) string {
	e.t.Helper()
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_guests (id, project_id, name, guest_side, category, sub_event_scope, table_number, pax_count, phone_number, formatted_wa_number, qr_code_token, sort_order)
		SELECT gen_random_uuid(), $1, name, guest_side, category, sub_event_scope, table_number, pax_count, phone_number, formatted_wa_number, md5(random()::text || id::text), sort_order
		FROM planner_guests WHERE project_id = (SELECT id FROM planner_projects WHERE is_sample LIMIT 1)`, id)

	return id
}

func TestBulkImportRejectsBadFiles(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	post := func(body any) resp { return e.do(tok, "POST", guests(id)+"/bulk-import", body) }

	for name, c := range map[string]struct {
		body any
		msg  string
	}{
		"a text file":             {importBody("tamu.txt", []byte("Nama Tamu\nSatu\n")), "file: filename must end with .csv or .xlsx"},
		"no extension":            {importBody("tamu", []byte("x")), "file: filename must end with .csv or .xlsx"},
		"an empty name":           {importBody("", []byte("x")), "file: filename must end with .csv or .xlsx"},
		"broken base64":           {map[string]any{"filename": "a.csv", "content_base64": "!!!"}, "file: content_base64 is not valid base64"},
		"no content":              {importBody("a.csv", nil), "file: empty file"},
		"a zip called .csv":       {importBody("a.csv", []byte("PK\x03\x04junk")), "file: the content is not a CSV file"},
		"text called .xlsx":       {importBody("a.xlsx", []byte("Nama Tamu\nSatu\n")), "file: the content is not an .xlsx file"},
		"a broken .xlsx":          {importBody("a.xlsx", []byte("PK\x03\x04junk")), "file: not a readable .xlsx file"},
		"a binary .csv":           {importBody("a.csv", []byte("Nama\x00Tamu")), "file: not a text file"},
		"a header without a name": {importBody("a.csv", []byte("Pihak;Pax\npria;2\n")), "file: missing column: Nama Tamu"},
		"only a header":           {importBody("a.csv", []byte("Nama Tamu\n")), ""},
	} {
		r := post(c.body)
		if c.msg == "" { // a header and nothing else is a valid file with no rows
			if r.status != 200 || num(r.data()["total_rows"]) != 0 {
				t.Errorf("%s: %d %s", name, r.status, r.body)
			}

			continue
		}
		if r.status != 400 || r.code() != "invalid_file" || r.message() != c.msg {
			t.Errorf("%s: %d %s %q, want 400 invalid_file %q", name, r.status, r.code(), r.message(), c.msg)
		}
	}

	var rows strings.Builder
	rows.WriteString("Nama Tamu\n")
	for i := 0; i < 2001; i++ {
		fmt.Fprintf(&rows, "Tamu %d\n", i)
	}
	wantError(t, post(importBody("a.csv", []byte(rows.String()))), 400, "invalid_file", "file: too many rows (max 2000)")

	// 5 MiB is the limit of the file after decoding; 7.5 MiB the limit of the body.
	wantError(t, post(importBody("a.csv", bytes.Repeat([]byte("a"), 5<<20+1))), 413, "file_too_large", "")
	wantError(t, post(`{"filename": "a.csv", "content_base64": "`+strings.Repeat("A", 8<<20)+`"}`), 413, "file_too_large", "")
	wantError(t, post("{not json"), 400, "invalid_json", "invalid JSON body")
	wantError(t, post(`{"filename": 7}`), 400, "invalid_json", "invalid JSON body")
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 0 {
		t.Errorf("%d guests written by refused imports", n)
	}
	// A big file that is allowed still goes through: 2000 rows.
	rows.Reset()
	rows.WriteString("Nama Tamu;Pax\n")
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&rows, "Tamu %d;2\n", i)
	}
	if r := mustStatus(t, post(importBody("a.csv", []byte(rows.String()))), 200).data(); num(r["created"]) != 2000 {
		t.Errorf("2000 rows = %v", r)
	}
	if n := count(e, `SELECT count(DISTINCT sort_order) FROM planner_guests WHERE project_id=$1`, id); n != 2000 {
		t.Errorf("%d distinct sort_order", n)
	}
}

func TestBulkImportRespectsTheGuestLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	insertGuests(e, id, 4999)

	// A dry run reports; the real import refuses and writes nothing (two rows would make 5001).
	mustStatus(t, importCSV(e, tok, id, "Nama Tamu\nSatu\nDua\n", "dry_run=true"), 200)
	wantError(t, importCSV(e, tok, id, "Nama Tamu\nSatu\nDua\n", ""), 409, "guest_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 4999 {
		t.Errorf("%d guests, want 4999", n)
	}
	mustStatus(t, importCSV(e, tok, id, "Nama Tamu\nSatu\n", ""), 200)
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 5000 {
		t.Errorf("%d guests, want 5000", n)
	}
}

// Two imports at once under the lock: the limit holds, and the sort order has no duplicates.
func TestConcurrentImportsRespectTheLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	insertGuests(e, id, 4990)

	csv := "Nama Tamu\n" + strings.Repeat("Baru\n", 6) // six each: only one of two fits
	statuses := concurrently(2, func(int) int { return importCSV(e, tok, id, csv, "").status })
	if !(statuses[0] == 200 && statuses[1] == 409 || statuses[0] == 409 && statuses[1] == 200) {
		t.Errorf("statuses = %v, want one 200 and one 409", statuses)
	}
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 4996 {
		t.Errorf("%d guests, want 4996", n)
	}
}

func TestImportRateLimit(t *testing.T) {
	e := newLimitedEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	for i := 0; i < 5; i++ { // a dry run counts too
		mustStatus(t, importCSV(e, tok, id, "Nama Tamu\nSatu\n", "dry_run=true"), 200)
	}
	wantError(t, importCSV(e, tok, id, "Nama Tamu\nSatu\n", "dry_run=true"), 429, "rate_limited", "")
	// The limit is the user's, not the route's of one project... but another user is not affected.
	other := e.login("other")
	mustStatus(t, importCSV(e, other, createProject(e, other, nil), "Nama Tamu\nSatu\n", "dry_run=true"), 200)
}

func TestBulkImportReadsAWorkbook(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	f := excelize.NewFile()
	rows := [][]any{
		{"Nama Peserta", "Pihak", "Kategori", "Sesi", "No Meja", "Pax", "No WhatsApp", "Catatan"},
		{"Ani Wijaya", "Pihak Pria", "VIP", "", "A1", 2, "081200000301", ""},
		{},
		{"Budi Hartono", "wanita", "Kolega Kantor", "Resepsi", "", "", "+62 812-0000-0302", "Vegetarian"},
		{"Citra Dewi", "", "", "", "", "", 81200000303, ""}, // a number typed in Excel loses its leading 0: refused, not guessed
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow("Sheet1", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	post := func(query string) resp {
		return e.do(tok, "POST", guests(id)+"/bulk-import?"+query, importBody("peserta.xlsx", buf.Bytes()))
	}

	r := mustStatus(t, post("dry_run=true"), 200).data()
	errs := list(r["errors"])
	if num(r["total_rows"]) != 3 || num(r["valid_rows"]) != 2 || num(r["invalid_rows"]) != 1 || len(errs) != 1 || num(errs[0]["row"]) != 5 || errs[0]["field"] != "phone_number" {
		t.Fatalf("dry run = %v", r)
	}
	mustStatus(t, post("skip_invalid=true"), 200)
	gs := list(guestList(e, tok, id, "")["guests"])
	if len(gs) != 2 || gs[0]["name"] != "Ani Wijaya" || gs[0]["guest_side"] != "pria" || gs[0]["category"] != "vip" || gs[0]["sub_event_scope"] != "all" || num(gs[0]["pax_count"]) != 2 ||
		gs[1]["name"] != "Budi Hartono" || gs[1]["guest_side"] != "wanita" || gs[1]["category"] != "Kolega Kantor" || gs[1]["sub_event_scope"] != "resepsi" || num(gs[1]["pax_count"]) != 1 {
		t.Errorf("guests = %v", gs)
	}
}

func openSheet(t *testing.T, r resp) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(r.body))
	if err != nil {
		t.Fatalf("not a workbook: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func rowsOf(t *testing.T, f *excelize.File, sheet string) [][]string {
	t.Helper()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func TestTemplateEndpointReadBack(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	wedding := createProject(e, tok, nil)

	r := mustStatus(t, e.do(tok, "GET", guests(wedding)+"/import-template?format=xlsx", nil), 200)
	if r.header.Get("Content-Type") != xlsxType || r.header.Get("Cache-Control") != "no-store" ||
		r.header.Get("Content-Disposition") != `attachment; filename="templat-tamu-pernikahan-sinta-arif-2026-10-04.xlsx"` {
		t.Errorf("headers = %v", r.header)
	}
	f := openSheet(t, r)
	data := rowsOf(t, f, "Tamu")
	if len(data) != 1 || strings.Join(data[0], "|") != "Nama Tamu|Pihak|Kategori|Sesi|No Meja|Pax|No WhatsApp|Catatan" {
		t.Errorf("the data sheet must hold the header only: %v", data)
	}
	guide := rowsOf(t, f, "Petunjuk")
	var sessions, sides []string
	for _, row := range guide[1:] {
		if len(row) > 0 && row[0] != "" {
			sides = append(sides, row[0])
		}
		if len(row) > 2 && row[2] != "" {
			sessions = append(sessions, row[2])
		}
	}
	if strings.Join(sides, "|") != "Pihak Pria (CPP)|Pihak Wanita (CPW)" || strings.Join(sessions, "|") != "Umum / Semua Sesi|Lamaran|Akad Nikah|Resepsi" {
		t.Errorf("guide: sides %v, sessions %v", sides, sessions)
	}

	// A seminar speaks of "Peserta" and has its own two sides; a birthday has no sides, so no Pihak column.
	seminar := createProject(e, tok, map[string]any{"event_type": "seminar", "title": "Seminar Nasional", "metadata": map[string]any{"organizer_name": "BEM"}})
	h := rowsOf(t, openSheet(t, mustStatus(t, e.do(tok, "GET", guests(seminar)+"/import-template?format=xlsx", nil), 200)), "Tamu")
	if len(h) != 1 || strings.Join(h[0], "|") != "Nama Peserta|Pihak|Kategori|Sesi|No Meja|Pax|No WhatsApp|Catatan" {
		t.Errorf("seminar header = %v", h)
	}
	birthday := createProject(e, tok, map[string]any{"event_type": "birthday", "title": "Ulang Tahun Ibu", "metadata": map[string]any{"celebrant_name": "Ibu"}})
	h = rowsOf(t, openSheet(t, mustStatus(t, e.do(tok, "GET", guests(birthday)+"/import-template?format=xlsx", nil), 200)), "Tamu")
	if len(h) != 1 || strings.Join(h[0], "|") != "Nama Tamu|Kategori|Sesi|No Meja|Pax|No WhatsApp|Catatan" {
		t.Errorf("birthday header = %v", h)
	}
	// The file it produces is a file the importer reads: nothing to import, no error.
	tmpl := mustStatus(t, e.do(tok, "GET", guests(wedding)+"/import-template?format=xlsx", nil), 200)
	if res := mustStatus(t, e.do(tok, "POST", guests(wedding)+"/bulk-import?dry_run=true", importBody("templat.xlsx", tmpl.body)), 200).data(); num(res["total_rows"]) != 0 {
		t.Errorf("template read back = %v", res)
	}
}

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func TestGuestExports(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	addGuest(e, tok, id, guestBody("=HYPERLINK(\"http://x\")", map[string]any{"notes": "@cmd", "table_number": "-1"}))
	morning(t, e, tok, id)
	// The entry time of the first guest is fixed: 04.14 UTC is 11.14 in Jakarta.
	exec(e, `UPDATE planner_guests SET checked_in_at='2027-03-28T04:14:00Z' WHERE project_id=$1 AND name='Bpk. Prof. Bambang Soedjarwo'`, id)

	title := get(e, tok, base(id)+"/dashboard")["project"].(map[string]any)["title"].(string)
	slug := "contoh-pernikahan-sinta-arif" // the file name is a slug of the title, never the title itself
	r := mustStatus(t, e.do(tok, "GET", guests(id)+"/export?format=xlsx", nil), 200)
	if r.header.Get("Content-Type") != xlsxType || r.header.Get("Cache-Control") != "no-store" ||
		r.header.Get("Content-Disposition") != `attachment; filename="tamu-`+slug+`-2026-10-04.xlsx"` {
		t.Errorf("headers = %v (title %q)", r.header, title)
	}
	rows := rowsOf(t, openSheet(t, r), "Daftar Tamu")
	// title, date, blank, header, 21 guests, TOTAL
	if len(rows) != 4+21+1 || rows[0][0] != "Daftar Tamu — "+title || rows[3][1] != "Nama" || rows[3][11] != "Kode Tiket" {
		t.Fatalf("%d rows; header %v", len(rows), rows[3])
	}
	first := rows[4]
	if first[1] != "Bpk. Prof. Bambang Soedjarwo" || first[2] != "Pihak Pria (CPP)" || first[3] != "VIP" || first[4] != "Resepsi" || first[5] != "VIP-01" || first[6] != "2" ||
		first[7] != "081200000101" || first[8] != "Terkirim" || first[9] != "Hadir" || first[10] != "2" || len(first[11]) != 32 {
		t.Errorf("first row = %v", first)
	}
	hostile := rows[24]
	if hostile[1] != `'=HYPERLINK("http://x")` || hostile[5] != "'-1" || hostile[12] != "'@cmd" {
		t.Errorf("a name that starts with = must not reach the sheet as a formula: %v", hostile)
	}
	if total := rows[25]; total[0] != "TOTAL" || total[1] != "21 tamu" || total[6] != "45" {
		t.Errorf("total = %v", total)
	}

	// Attendance: the 18 expected guests (the two who declined are not in it) plus the one added; the entry time in Jakarta.
	a := mustStatus(t, e.do(tok, "GET", guests(id)+"/export-attendance?format=xlsx", nil), 200)
	if a.header.Get("Content-Disposition") != `attachment; filename="kehadiran-`+slug+`-2026-10-04.xlsx"` {
		t.Errorf("headers = %v", a.header)
	}
	att := rowsOf(t, openSheet(t, a), "Kehadiran")
	if len(att) != 4+19+1 {
		t.Fatalf("%d rows, want 24", len(att))
	}
	var bambang []string
	for _, row := range att[4:] {
		if strings.Contains(row[1], "Joko") || strings.Contains(row[1], "Dedi") {
			t.Errorf("a guest who declined and did not come is in the attendance sheet: %v", row)
		}
		if row[1] == "Bpk. Prof. Bambang Soedjarwo" {
			bambang = row
		}
	}
	if len(bambang) < 11 || bambang[6] != "2" || bambang[7] != "2" || bambang[8] != "11.14" || bambang[9] != "Ya" || bambang[10] != "Hadir" {
		t.Errorf("Bambang's row = %v", bambang)
	}
	if total := att[len(att)-1]; total[0] != "TOTAL" || total[1] != "11 hadir dari 19 tamu" || total[7] != "22" || total[9] != "11 diserahkan" {
		t.Errorf("total = %v", total)
	}
}

func TestExportsRejectOtherFormats(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	for _, path := range []string{"/export", "/export-attendance", "/import-template"} {
		for _, q := range []string{"?format=pdf", "?format=XLSX", "?format=csv"} {
			wantError(t, e.do(tok, "GET", guests(id)+path+q, nil), 400, "invalid_input", "format: must be xlsx")
		}
	}
}

// The list, the attendance sheet and the template share one counter of 10 a minute.
func TestDownloadsShareARateLimit(t *testing.T) {
	e := newLimitedEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	paths := []string{"/export", "/export-attendance", "/import-template"}
	for i := 0; i < 10; i++ {
		mustStatus(t, e.do(tok, "GET", guests(id)+paths[i%3]+"?format=xlsx", nil), 200)
	}
	for _, p := range paths {
		wantError(t, e.do(tok, "GET", guests(id)+p+"?format=xlsx", nil), 429, "rate_limited", "")
	}
	// Reading the list is not a download.
	mustStatus(t, e.do(tok, "GET", guests(id), nil), 200)
	// Another user has his own count.
	other := e.login("other")
	mustStatus(t, e.do(other, "GET", guests(createProject(e, other, nil))+"/export?format=xlsx", nil), 200)
}

func TestScanRateLimit(t *testing.T) {
	e := newLimitedEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	scan := func() resp {
		return e.do(tok, "POST", guests(id)+"/scan-qr", map[string]any{"qr_code_token": strings.Repeat("a", 32)})
	}
	for i := 0; i < 120; i++ {
		if r := scan(); r.status != 404 {
			t.Fatalf("scan %d: %d", i+1, r.status)
		}
	}
	wantError(t, scan(), 429, "rate_limited", "")
	// The check-in of the guest who is standing there is not a scan: it must still work.
	g := addGuest(e, tok, id, guestBody("Di pintu", nil))["id"].(string)
	mustStatus(t, checkin(e, tok, id, g, 1, false), 200)
}

// The largest file the import accepts (5 MiB once decoded, about 7 MiB as JSON) passes the body limit and the file limit;
// one byte more is refused by the file limit (TestBulkImportRejectsBadFiles). Blank lines are skipped, so the file is valid.
func TestBulkImportAcceptsTheLargestFile(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	content := append([]byte("Nama Tamu;Pax\nSatu;2\n"), bytes.Repeat([]byte("\n"), 5<<20-len("Nama Tamu;Pax\nSatu;2\n"))...)
	if len(content) != 5<<20 {
		t.Fatalf("file is %d bytes", len(content))
	}
	r := e.do(tok, "POST", guests(id)+"/bulk-import", importBody("besar.csv", content))
	if got := mustStatus(t, r, 200).data(); num(got["created"]) != 1 || num(got["total_rows"]) != 1 {
		t.Errorf("result = %v", got)
	}
	// A 3 MiB file of real rows (2000 rows with notes) is well over the 64 KiB of every other route.
	var rows strings.Builder
	rows.WriteString("Nama Tamu;Pax;Catatan\n")
	note := strings.Repeat("c", 1500)
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&rows, "Tamu %d;1;%s\n", i, note)
	}
	if got := mustStatus(t, e.do(tok, "POST", guests(id)+"/bulk-import?dry_run=true", importBody("catatan.csv", []byte(rows.String()))), 200).data(); num(got["valid_rows"]) != 2000 {
		t.Errorf("result = %v", got)
	}
}

// The entry time is written in the zone of the project, not in the zone of the server or of the database session: the same
// instant is 11.14 in Jakarta and 12.14 in Makassar.
func TestAttendanceEntryTimeFollowsTheProjectZone(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	for zone, want := range map[string]string{"Asia/Jakarta": "11.14", "Asia/Makassar": "12.14", "Asia/Jayapura": "13.14", "UTC": "04.14"} {
		id := createProject(e, tok, map[string]any{"timezone": zone})
		g := addGuest(e, tok, id, guestBody("Tamu "+zone, nil))["id"].(string)
		mustStatus(t, checkin(e, tok, id, g, 2, false), 200)
		exec(e, `UPDATE planner_guests SET checked_in_at='2027-03-28T04:14:00Z' WHERE id=$1`, g)

		rows := rowsOf(t, openSheet(t, mustStatus(t, e.do(tok, "GET", guests(id)+"/export-attendance?format=xlsx", nil), 200)), "Kehadiran")
		if len(rows) < 6 || len(rows[4]) < 9 || rows[4][8] != want {
			t.Errorf("%s: the sheet says %v, want %s", zone, rows, want)
		}
	}
}
