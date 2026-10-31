package http_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// financeRoutes lists the 32 routes of the module with a body that is valid for
// a project that has the ids in ids (so a 404 can only come from the guard).
type route struct {
	method, path string
	body         any
}

func financeRoutes(ids map[string]string) []route {
	sc, item, line, contrib, entry := ids["scenario"], ids["scenario_item"], ids["line"], ids["contributor"], ids["entry"]

	return []route{
		{"GET", "/scenarios", nil},
		{"POST", "/scenarios", map[string]any{"name": "x", "total_target": 1}},
		{"POST", "/scenarios/compare", map[string]any{"a": sc, "b": uuid.NewString()}},
		{"POST", "/scenarios/apply-preview", map[string]any{"scenario_id": sc, "mode": "smart_adjust"}},
		{"POST", "/scenarios/apply", map[string]any{"scenario_id": sc, "mode": "smart_adjust"}},
		{"GET", "/scenarios/" + sc, nil},
		{"PUT", "/scenarios/" + sc, map[string]any{"name": "x", "total_target": 1}},
		{"DELETE", "/scenarios/" + sc, nil},
		{"POST", "/scenarios/" + sc + "/clone", map[string]any{"new_name": "x"}},
		{"POST", "/scenarios/" + sc + "/load-preset", map[string]any{"preset_key": "wedding_10jt", "confirm": true}},
		{"POST", "/scenarios/" + sc + "/items", map[string]any{"category": "x", "item_name": "x"}},
		{"PUT", "/scenarios/" + sc + "/items/" + item, map[string]any{"category": "x", "item_name": "x"}},
		{"DELETE", "/scenarios/" + sc + "/items/" + item, nil},
		{"GET", "/contributors", nil},
		{"POST", "/contributors", map[string]any{"name": "x"}},
		{"PUT", "/contributors/" + contrib, map[string]any{"name": "x"}},
		{"DELETE", "/contributors/" + contrib, nil},
		{"GET", "/savings", nil},
		{"POST", "/savings", map[string]any{"contributor_id": contrib, "entry_date": today, "amount": 1}},
		{"PUT", "/savings/" + entry, map[string]any{"contributor_id": contrib, "entry_date": today, "amount": 1}},
		{"DELETE", "/savings/" + entry, nil},
		{"GET", "/budget/cashflow-summary", nil},
		{"GET", "/budget/matrix", nil},
		{"GET", "/budget/items", nil},
		{"POST", "/budget/items", map[string]any{"category": "x", "item_name": "x"}},
		{"POST", "/budget/items/reorder", map[string]any{"ids": []string{}}},
		{"PUT", "/budget/items/" + line, map[string]any{"category": "x", "item_name": "x"}},
		{"PATCH", "/budget/items/" + line + "/payment", map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today}},
		{"DELETE", "/budget/items/" + line + "/payment/dp", nil},
		{"DELETE", "/budget/items/" + line, nil},
		{"POST", "/budget/apply-template", map[string]any{}},
		{"GET", "/budget/export?format=xlsx", nil},
	}
}

func TestEveryFinanceRouteNeedsABearerToken(t *testing.T) {
	e := newEnv(t)
	id := uuid.NewString()
	ids := map[string]string{"scenario": uuid.NewString(), "scenario_item": uuid.NewString(), "line": uuid.NewString(), "contributor": uuid.NewString(), "entry": uuid.NewString()}

	routes := financeRoutes(ids)
	if len(routes) != 32 {
		t.Fatalf("the module has 32 routes, the test lists %d", len(routes))
	}
	for _, rt := range routes {
		if r := e.do("", rt.method, base(id)+rt.path, rt.body); r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
}

// B sees A's project as not existing, on every one of the 32 routes, and A's data is untouched.
func TestUserBGetsNotFoundOnEveryFinanceEndpoint(t *testing.T) {
	e := newEnv(t)
	a, b := e.login("owner"), e.login("intruder")
	id := createSample(e, a)

	sc := list(get(e, a, base(id)+"/scenarios")["scenarios"])[0]
	lines0 := list(get(e, a, base(id)+"/budget/items")["items"])
	sv := get(e, a, base(id)+"/savings")
	ids := map[string]string{
		"scenario": sc["id"].(string), "scenario_item": list(sc["items"])[0]["id"].(string), "line": lines0[0]["id"].(string),
		"contributor": list(sv["contributors"])[0]["id"].(string), "entry": list(sv["entries"])[0]["id"].(string),
	}
	before := jsonOf([]any{sc, lines0, sv})

	for _, rt := range financeRoutes(ids) {
		wantError(t, e.do(b, rt.method, base(id)+rt.path, rt.body), 404, "not_found", "project not found")
	}

	// B's own project does not know A's ids either.
	own := createProject(e, b, nil)
	for _, rt := range []route{
		{"GET", "/scenarios/" + ids["scenario"], nil},
		{"DELETE", "/scenarios/" + ids["scenario"], nil},
		{"POST", "/scenarios/" + ids["scenario"] + "/items", map[string]any{"category": "x", "item_name": "x"}},
		{"PUT", "/scenarios/" + ids["scenario"] + "/items/" + ids["scenario_item"], map[string]any{"category": "x", "item_name": "x"}},
		{"DELETE", "/scenarios/" + ids["scenario"] + "/items/" + ids["scenario_item"], nil},
		{"PUT", "/budget/items/" + ids["line"], map[string]any{"category": "x", "item_name": "x"}},
		{"PATCH", "/budget/items/" + ids["line"] + "/payment", map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today}},
		{"DELETE", "/budget/items/" + ids["line"] + "/payment/dp", nil},
		{"DELETE", "/budget/items/" + ids["line"], nil},
		{"PUT", "/contributors/" + ids["contributor"], map[string]any{"name": "x"}},
		{"DELETE", "/contributors/" + ids["contributor"], nil},
		{"PUT", "/savings/" + ids["entry"], map[string]any{"contributor_id": ids["contributor"], "entry_date": today, "amount": 1}},
		{"DELETE", "/savings/" + ids["entry"], nil},
	} {
		if r := e.do(b, rt.method, base(own)+rt.path, rt.body); r.status != 404 {
			t.Errorf("%s %s with another project's id: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
	wantError(t, e.do(b, "POST", base(own)+"/scenarios/apply", map[string]any{"scenario_id": ids["scenario"], "mode": "smart_adjust"}), 404, "not_found", "scenario not found")
	wantError(t, e.do(b, "POST", base(own)+"/scenarios/compare", map[string]any{"a": ids["scenario"], "b": uuid.NewString()}), 404, "not_found", "scenario not found")
	wantError(t, e.do(b, "POST", base(own)+"/savings", map[string]any{"contributor_id": ids["contributor"], "entry_date": today, "amount": 1}), 400, "invalid_input", "contributor_id: must be a contributor of this project")
	wantError(t, e.do(b, "POST", base(own)+"/budget/items/reorder", map[string]any{"ids": []string{ids["line"]}}), 400, "invalid_input", "ids: must list every id exactly once")

	// Ids that are not canonical UUIDs are 404 as well, never a query.
	for _, bad := range []string{"nope", "{" + ids["line"] + "}", strings.ToUpper(strings.ReplaceAll(ids["line"], "-", ""))} {
		wantError(t, e.do(a, "PUT", base(id)+"/budget/items/"+bad, map[string]any{"category": "x", "item_name": "x"}), 404, "not_found", "budget item not found")
		wantError(t, e.do(a, "GET", base(id)+"/scenarios/"+bad, nil), 404, "not_found", "scenario not found")
	}

	if after := jsonOf([]any{
		list(get(e, a, base(id)+"/scenarios")["scenarios"])[0], list(get(e, a, base(id)+"/budget/items")["items"]), get(e, a, base(id)+"/savings"),
	}); after != before {
		t.Error("the intruder changed something")
	}
}

func TestArchivedProjectIsReadOnlyButStillReadable(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sc := list(get(e, tok, base(id)+"/scenarios")["scenarios"])[0]
	sv := get(e, tok, base(id)+"/savings")
	line := list(get(e, tok, base(id)+"/budget/items")["items"])[0]
	ids := map[string]string{
		"scenario": sc["id"].(string), "scenario_item": list(sc["items"])[0]["id"].(string), "line": line["id"].(string),
		"contributor": list(sv["contributors"])[0]["id"].(string), "entry": list(sv["entries"])[0]["id"].(string),
	}
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range financeRoutes(ids) {
		r := e.do(tok, rt.method, base(id)+rt.path, rt.body)
		if rt.method == "GET" {
			if r.status != 200 {
				t.Errorf("GET %s on an archived project: %d %s", rt.path, r.status, r.body)
			}
			continue
		}
		wantError(t, r, 409, "project_read_only", "")
	}
	if len(list(get(e, tok, base(id)+"/budget/items")["items"])) != 14 {
		t.Error("an archived project changed")
	}
}

func TestFinanceDocumentsAreRejectedForMalformedBodies(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	for _, rt := range []route{
		{"POST", "/scenarios", "{not json"}, {"POST", "/scenarios", `{"name": 5}`}, {"POST", "/contributors", `[]`},
		{"POST", "/savings", `{"contributor_id": "nope"}`}, {"POST", "/budget/items", `{"initial_estimate": "lots"}`},
		{"PATCH", "/budget/items/" + uuid.NewString() + "/payment", `{"amount": 1.5}`}, {"POST", "/scenarios/apply", `{"scenario_id": 7}`},
	} {
		wantError(t, e.do(tok, rt.method, base(id)+rt.path, rt.body), 400, "invalid_json", "invalid JSON body")
	}
	big := `{"name": "` + strings.Repeat("a", 70<<10) + `"}`
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", big), 400, "invalid_json", "invalid JSON body")
}

// ---------- export ----------

func TestWorkbookOfTheSampleProject(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	r := mustStatus(t, e.do(tok, "GET", base(id)+"/budget/export?format=xlsx", nil), 200)
	if ct := r.header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := r.header.Get("Content-Disposition"); cd != `attachment; filename="keuangan-contoh-pernikahan-sinta-arif-2026-10-04.xlsx"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	f, err := excelize.OpenReader(bytes.NewReader(r.body))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 3 || got[0] != "Rekap" || got[1] != "Anggaran" || got[2] != "Tabungan" {
		t.Fatalf("sheets = %v", got)
	}

	rows := func(sheet string) [][]string {
		out, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}

		return out
	}
	cell := func(sheet, ref string) string {
		v, err := f.GetCellValue(sheet, ref, excelize.Options{RawCellValue: true})
		if err != nil {
			t.Fatal(err)
		}

		return v
	}

	// Rekap: the session matrix of PDF p. 5, the numbers stored as numbers.
	rekap := rows("Rekap")
	if rekap[1][0] != "Keterangan" || rekap[1][1] != "Lamaran" || rekap[1][2] != "Akad Nikah" || rekap[1][3] != "Resepsi" || rekap[1][4] != "Total" {
		t.Errorf("Rekap header = %v", rekap[1])
	}
	for ref, want := range map[string]string{"B3": "7150000", "E3": "74150000", "E4": "20050000", "E5": "12000000", "E6": "1500000", "E7": "33550000", "E8": "40600000", "D8": "38200000"} {
		if got := cell("Rekap", ref); got != want {
			t.Errorf("Rekap!%s = %q, want %q", ref, got, want)
		}
	}
	if len(rekap) < 12 || rekap[9][0] != "Skenario aktif" || rekap[9][1] != "Opsi Ideal 75 Jt" || cell("Rekap", "B12") != "850000" {
		t.Errorf("Rekap scenario block = %v", rekap[9:])
	}

	// Anggaran: one block per session, then the lines of PDF p. 5-6 and a Total row per block.
	ang := rows("Anggaran")
	if ang[0][0] != "ANGGARAN LAMARAN" || ang[1][1] != "Kegiatan" || ang[2][1] != "Catering" || cell("Anggaran", "C3") != "2000000" || cell("Anggaran", "G3") != "2000000" {
		t.Errorf("Anggaran block 1 = %v", ang[:3])
	}
	if ang[7][1] != "Total" || cell("Anggaran", "C8") != "7150000" || cell("Anggaran", "H8") != "2400000" || ang[2][8] != "Lunas" || ang[4][8] != "DP dibayar" {
		t.Errorf("Anggaran Lamaran total = %v", ang[7])
	}
	foundResepsi := false
	for _, r := range ang {
		if len(r) > 0 && r[0] == "ANGGARAN RESEPSI" {
			foundResepsi = true
		}
	}
	if !foundResepsi {
		t.Error("no ANGGARAN RESEPSI block")
	}

	// Tabungan: PDF p. 7.
	tab := rows("Tabungan")
	if tab[0][0] != "Bulan" || tab[0][1] != "Arif" || tab[0][2] != "Sinta" || tab[0][3] != "Jumlah" || tab[1][0] != "2026-02" || cell("Tabungan", "D2") != "3000000" {
		t.Errorf("Tabungan header and first row = %v %v", tab[0], tab[1])
	}
	if tab[12][0] != "Total" || cell("Tabungan", "B13") != "30000000" || cell("Tabungan", "C13") != "17500000" || cell("Tabungan", "D13") != "47500000" ||
		tab[14][0] != "Target tabungan" || cell("Tabungan", "B17") != "27500000" {
		t.Errorf("Tabungan totals = %v", tab[12:])
	}
}

func TestWorkbookNeutralisesFormulasInUserText(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	for _, name := range []string{"=HYPERLINK(\"http://evil\",\"x\")", "+1+1", "-2", "@SUM(A1)"} {
		mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": name, "initial_estimate": 1}), 201)
	}
	mustStatus(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": "=cmd"}), 201)

	r := mustStatus(t, e.do(tok, "GET", base(id)+"/budget/export?format=xlsx", nil), 200)
	f, err := excelize.OpenReader(bytes.NewReader(r.body))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, _ := f.GetRows("Anggaran")
	names := map[string]bool{}
	for _, row := range got {
		if len(row) > 1 {
			names[row[1]] = true
		}
	}
	for _, want := range []string{"'=HYPERLINK(\"http://evil\",\"x\")", "'+1+1", "'-2", "'@SUM(A1)"} {
		if !names[want] {
			t.Errorf("%q is not written as text with a leading apostrophe; cells: %v", want, names)
		}
	}
	for _, sheet := range f.GetSheetList() {
		cells, _ := f.GetRows(sheet)
		for ri, row := range cells {
			for ci, v := range row {
				if ref, _ := excelize.CoordinatesToCellName(ci+1, ri+1); strings.HasPrefix(v, "=") {
					formula, _ := f.GetCellFormula(sheet, ref)
					t.Errorf("%s!%s starts with = (%q, formula %q)", sheet, ref, v, formula)
				}
			}
		}
	}
}

func TestExportParametersAndRateLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	wantError(t, e.do(tok, "GET", base(id)+"/budget/export?format=pdf", nil), 400, "invalid_input", "format: must be xlsx")
	wantError(t, e.do(tok, "GET", base(id)+"/budget/export?format=csv", nil), 400, "invalid_input", "format: must be xlsx")

	// 10 exports a minute per user; another user is not affected.
	ok, limited := 0, 0
	for i := 0; i < 14; i++ {
		switch r := e.do(tok, "GET", base(id)+"/budget/export?format=xlsx", nil); r.status {
		case 200:
			ok++
		case 429:
			limited++
			if r.code() != "rate_limited" {
				t.Errorf("429 with code %q", r.code())
			}
		default:
			t.Fatalf("export: %d %s", r.status, r.body)
		}
	}
	// The two refused calls above (400) came first and were counted as well.
	if ok != 8 || limited != 6 {
		t.Errorf("%d exports answered, %d limited; want 8 and 6 after 2 earlier calls", ok, limited)
	}
	other := e.login("second")
	otherProject := createProject(e, other, nil)
	mustStatus(t, e.do(other, "GET", base(otherProject)+"/budget/export?format=xlsx", nil), 200)
}
