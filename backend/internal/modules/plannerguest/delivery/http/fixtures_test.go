package http_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The fixtures of the frontend spec (05_BUKU_TAMU/fixtures) are built from the table of 05 BE 6 by the reference functions.
// These tests put the REAL responses of the stack next to them: the structure, the keys, the types and the values must agree.
// Ids, tickets and times are replaced by a marker (a UUID and a clock differ from run to run); that they are present is kept.

const fixtureDir = "../../../../../../docs/APP_SPEC/PLANNER_SPEC/05_BUKU_TAMU/fixtures/"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(fixtureDir + name)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}

	return raw
}

func fixtureJSON(t *testing.T, name string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(readFixture(t, name), &v); err != nil {
		t.Fatal(err)
	}

	return v
}

// normalise replaces what changes from run to run: ids and tickets by "<id>", times by "<time>".
func normalised(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			switch {
			case (k == "id" || k == "of_guest" || k == "qr_code_token") && isString(e):
				out[k] = "<id>"
			case strings.HasSuffix(k, "_at") && isString(e):
				out[k] = "<time>"
			default:
				out[k] = normalised(e)
			}
		}

		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalised(e)
		}

		return out
	}

	return v
}

func isString(v any) bool { _, ok := v.(string); return ok }

// firstDiff names the first place two decoded documents differ ("" when they are equal).
func firstDiff(path string, got, want any) string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return path + ": not an object"
		}
		keys := map[string]bool{}
		for k := range g {
			keys[k] = true
		}
		for k := range w {
			keys[k] = true
		}
		var sorted []string
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			gv, inGot := g[k]
			wv, inWant := w[k]
			switch {
			case !inGot:
				return path + "." + k + ": missing in the response"
			case !inWant:
				return path + "." + k + ": not in the fixture"
			}
			if d := firstDiff(path+"."+k, gv, wv); d != "" {
				return d
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return fmt.Sprintf("%s: %d elements, the fixture has %d", path, len(g), len(w))
		}
		for i := range w {
			if d := firstDiff(fmt.Sprintf("%s[%d]", path, i), g[i], w[i]); d != "" {
				return d
			}
		}
	default:
		if !reflect.DeepEqual(got, want) {
			return fmt.Sprintf("%s: response %v, fixture %v", path, got, want)
		}
	}

	return ""
}

func sameJSON(t *testing.T, what string, got, want any) {
	t.Helper()
	if d := firstDiff("$", normalised(got), normalised(want)); d != "" {
		t.Errorf("%s differs from the fixture at %s", what, d)
	}
}

func rawData(t *testing.T, r resp) any {
	t.Helper()
	var env struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(r.body, &env); err != nil {
		t.Fatal(err)
	}

	return env.Data
}

func TestResponsesMatchTheFEFixtures(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	// planner-guests.json: the list of the sample project.
	sameJSON(t, "guest list", rawData(t, e.do(tok, "GET", guests(id)+"?limit=100", nil)), fixtureJSON(t, "planner-guests.json"))
	// planner-catering-projection.json
	sameJSON(t, "catering projection", rawData(t, e.do(tok, "GET", guests(id)+"/catering-projection", nil)), fixtureJSON(t, "planner-catering-projection.json"))
	// planner-import-dryrun.json: the CSV of the fixture folder against the sample project.
	csv := readFixture(t, "planner-import-sample.csv")
	sameJSON(t, "import dry run", rawData(t, e.do(tok, "POST", guests(id)+"/bulk-import?dry_run=true", importBody("planner-import-sample.csv", csv))), fixtureJSON(t, "planner-import-dryrun.json"))
	if string(csv) != importSample {
		t.Error("the CSV of the tests and the CSV of the fixtures differ")
	}

	// planner-reception.json: the illustration of the event day. The 11 guests it lists as in are checked in at the times it
	// gives, with the head-count and the souvenir it shows, and the stock it names (40) comes from a finished souvenir item.
	day := fixtureJSON(t, "planner-reception.json").(map[string]any)
	for _, g := range list(day["recent_checkins"]) {
		exec(e, `UPDATE planner_guests SET is_checked_in=TRUE, checked_in_at=$1, pax_actual=$2, souvenir_received=$3 WHERE project_id=$4 AND name=$5`,
			g["checked_in_at"], num(g["pax_actual"]), g["souvenir_received"], id, g["name"])
	}
	// The fixture lists 10 of the 11 in recent_checkins; the 11th is in the queue.
	for _, g := range list(day["guests"]) {
		if g["is_checked_in"] == true {
			exec(e, `UPDATE planner_guests SET is_checked_in=TRUE, checked_in_at=$1, pax_actual=$2, souvenir_received=$3 WHERE project_id=$4 AND name=$5`,
				g["checked_in_at"], num(g["pax_actual"]), g["souvenir_received"], id, g["name"])
		}
	}
	mustStatus(t, e.do(tok, "POST", base(id)+"/shopping-items", map[string]any{
		"sub_event_scope": "all", "category": "Souvenir", "item_name": "Tumbler", "quantity": 40, "estimated_price": 5000, "purpose": "souvenir", "status": "selesai",
	}), 201)
	sameJSON(t, "reception", rawData(t, e.do(tok, "GET", guests(id)+"/reception-overview", nil)), day)
}
