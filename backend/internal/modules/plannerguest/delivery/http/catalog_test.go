package http_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The error catalog of the frontend spec (05_BUKU_TAMU/FE.md, 4.6) names a status, a code and example messages for each
// failure. This test reads that table and makes the real backend produce every one of them: a message in the document that
// the backend does not say, or a row with no scenario here, fails.
func TestErrorCatalogOfTheFEDocIsWhatTheBackendSays(t *testing.T) {
	raw, err := os.ReadFile("../../../../../../docs/APP_SPEC/PLANNER_SPEC/05_BUKU_TAMU/FE.md")
	if err != nil {
		t.Skipf("FE.md not available: %v", err)
	}
	doc := string(raw)
	i := strings.Index(doc, "### 4.6 Katalog galat")
	j := strings.Index(doc[i:], "\n---")
	if i < 0 || j < 0 {
		t.Fatal("catalog not found")
	}
	row := regexp.MustCompile("^\\| (\\d{3}) \\| `([a-z_]+)` \\| (.+?) \\|")
	quoted := regexp.MustCompile("`([^`]+)`")

	e := newLimitedEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	checkedIn := guestNamed(t, e, tok, id, "Raditya Pratama & Pasangan")["id"].(string)
	mustStatus(t, checkin(e, tok, id, checkedIn, 2, false), 200)
	full := createProject(e, tok, nil)
	insertGuests(e, full, 5000)
	archived := createProject(e, tok, nil)
	mustStatus(t, e.do(tok, "PATCH", base(archived)+"/status", map[string]any{"status": "archived"}), 200)
	post := func(project, path string, body any) resp { return e.do(tok, "POST", guests(project)+path, body) }

	scenarios := map[string]func() resp{
		"name: required":                          func() resp { return post(id, "", guestBody(" ", nil)) },
		"pax_count: must be between 1 and 20":     func() resp { return post(id, "", guestBody("X", map[string]any{"pax_count": 0})) },
		"guest_side: must be a registry side key": func() resp { return post(id, "", guestBody("X", map[string]any{"guest_side": "kiri"})) },
		"rows: 2 rows are invalid":                func() resp { return importCSV(e, tok, id, importSample, "") },
		"phone_number: must be an Indonesian phone number (08…, +62…)": func() resp {
			return post(id, "", guestBody("X", map[string]any{"phone_number": "+65 9123 4567"}))
		},
		"sub_event_scope: unknown session": func() resp { return post(id, "", guestBody("X", map[string]any{"sub_event_scope": "tak_ada"})) },
		"file: missing column: Nama Tamu": func() resp {
			return e.do(tok, "POST", guests(id)+"/bulk-import", importBody("a.csv", []byte("Pax\n2\n")))
		},
		"file: too many rows (max 2000)": func() resp {
			return e.do(tok, "POST", guests(id)+"/bulk-import", importBody("a.csv", []byte("Nama Tamu\n"+strings.Repeat("x\n", 2001))))
		},
		"invalid JSON body":    func() resp { return post(id, "", "{") },
		"missing bearer token": func() resp { return e.do("", "GET", guests(id), nil) },
		"project not found":    func() resp { return e.do(tok, "GET", guests(uuid.NewString()), nil) },
		"guest not found":      func() resp { return e.do(tok, "GET", guests(id)+"/"+uuid.NewString(), nil) },
		"the project is archived; restore it to make changes": func() resp { return post(archived, "", guestBody("X", nil)) },
		"at most 5000 guests per project":                     func() resp { return post(full, "", guestBody("X", nil)) },
		"the guest is already checked in": func() resp {
			return post(id, "/"+checkedIn+"/checkin", map[string]any{"pax_actual": 1})
		},
		"file too large": func() resp {
			return e.do(tok, "POST", guests(id)+"/bulk-import", importBody("a.csv", []byte(strings.Repeat("a", 5<<20+1))))
		},
		"too many requests, try again later": func() resp {
			var r resp
			for i := 0; i < 130; i++ {
				r = post(id, "/scan-qr", map[string]any{"qr_code_token": "x"})
			}
			return r
		},
	}
	// Failures that a test cannot cause on demand: a concurrent change (409 conflict), the server (5xx).
	unreachable := map[string]bool{"the project was changed concurrently, try again": true}

	seen := map[string]bool{}
	rows := 0
	for _, line := range strings.Split(doc[i:i+j], "\n") {
		m := row.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rows++
		for _, q := range quoted.FindAllStringSubmatch(m[3], -1) {
			msg := q[1]
			if unreachable[msg] {
				continue
			}
			run, ok := scenarios[msg]
			if !ok {
				t.Errorf("%s %s: the message %q has no scenario in this test", m[1], m[2], msg)
				continue
			}
			seen[msg] = true
			r := run()
			if got := strconv.Itoa(r.status); got != m[1] || r.code() != m[2] || r.message() != msg {
				t.Errorf("%q: the backend says %d %s %q, the catalog says %s %s", msg, r.status, r.code(), r.message(), m[1], m[2])
			}
		}
	}
	if rows < 10 {
		t.Fatalf("only %d rows read from the catalog", rows)
	}
	for msg := range scenarios {
		if !seen[msg] {
			t.Errorf("scenario %q is not in the catalog", msg)
		}
	}
}
