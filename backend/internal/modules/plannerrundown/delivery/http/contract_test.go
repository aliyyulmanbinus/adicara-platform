package http_test

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// walk calls visit for every value of a decoded JSON document with its path.
func walk(path string, v any, visit func(path string, v any)) {
	visit(path, v)
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			walk(path+"."+k, val, visit)
		}
	case []any:
		for i, val := range x {
			walk(fmt.Sprintf("%s[%d]", path, i), val, visit)
		}
	}
}

// Every answer of the module follows the contract of kontrak bab 6: no null, no *_percent, lists are [] and optional
// keys are left out. It is checked on a project with nothing in it and on the sample project.
func TestResponsesHaveNoNullsNoPercentsAndEmptyListsAreArrays(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	empty := createProject(e, tok, nil)
	sample := createSample(e, tok)
	sess := sessionIDs(e, tok, sample)
	items := rundownItems(e, tok, sample)
	mustStatus(t, setStatus(e, tok, sample, items[0]["id"].(string), "in_progress"), 200)

	var paths []string
	for _, id := range []string{empty, sample} {
		paths = append(paths,
			base(id)+"/rundowns", base(id)+"/rundowns/summary", base(id)+"/playlists",
			base(id)+"/rundowns?session_id="+sessionIDs(e, tok, id)["akad"], base(id)+"/playlists?status=ready",
			base(id)+"/rundowns/export-wa", base(id)+"/playlists/export-wa",
		)
	}
	paths = append(paths, base(sample)+"/rundowns/"+items[0]["id"].(string))
	for _, p := range paths {
		raw := mustStatus(t, e.do(tok, "GET", p, nil), 200).body
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		walk("", doc, func(path string, v any) {
			if v == nil {
				t.Errorf("GET %s: null at %s", p, path)
			}
			if strings.HasSuffix(path, "_percent") {
				t.Errorf("GET %s: percentage %s", p, path)
			}
		})
	}

	// the empty project: three sessions with zeroes, no items, lists not null
	ov := rundownOverview(e, tok, empty)
	if len(list(ov["sessions"])) != 3 || len(list(ov["items"])) != 0 || num(ov["kpi"].(map[string]any)["total_span_minutes"]) != 0 {
		t.Errorf("empty overview = %v", ov)
	}
	for _, s := range list(ov["sessions"]) {
		for _, k := range []string{"first_start", "last_end"} {
			if _, ok := s[k]; ok {
				t.Errorf("a session without agendas has %s: %v", k, s)
			}
		}
		if num(s["span_minutes"]) != 0 || num(s["agenda_count"]) != 0 {
			t.Errorf("session = %v", s)
		}
	}
	if raw := mustStatus(t, e.do(tok, "GET", base(empty)+"/playlists", nil), 200).body; !strings.Contains(string(raw), `"items":[]`) {
		t.Errorf("empty playlist: %s", raw)
	}
	live := get(e, tok, base(empty)+"/rundowns/summary")
	if _, ok := live["items"].([]any); !ok || len(live["items"].([]any)) != 0 {
		t.Errorf("empty live = %v", live)
	}
	// filtering by one session keeps the KPI and the sessions whole
	one := get(e, tok, base(sample)+"/rundowns?session_id="+sess["akad"])
	if len(list(one["items"])) != 8 || len(list(one["sessions"])) != 3 || num(one["kpi"].(map[string]any)["total_agendas"]) != 33 {
		t.Errorf("filtered overview: %d items, %d sessions, kpi %v", len(list(one["items"])), len(list(one["sessions"])), one["kpi"])
	}
	// template and playlist results always carry warnings as a list
	if raw := applyAgendaTemplate(e, tok, sample, nil).body; !strings.Contains(string(raw), `"warnings":[]`) {
		t.Errorf("apply result: %s", raw)
	}
}

func TestRundownQueryRejectsUnknownSessions(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	for _, q := range []string{"?session_id=nope", "?session_id=" + uuid.NewString(), "?session_id=" + strings.ToUpper(uuid.NewString())} {
		for _, p := range []string{"/rundowns", "/rundowns/export-wa"} {
			wantError(t, e.do(tok, "GET", base(id)+p+q, nil), 400, "invalid_scope", "session_id: must be a session of this project")
		}
	}
	var all string
	if err := e.pool.QueryRow(t.Context(), `SELECT id FROM planner_event_sessions WHERE project_id = $1 AND is_system`, id).Scan(&all); err != nil {
		t.Fatal(err)
	}
	wantError(t, e.do(tok, "GET", base(id)+"/rundowns?session_id="+all, nil), 400, "invalid_input", "session_id: must not be the system session")
}

// The error catalogue of the frontend spec (fixtures/planner-rundown-errors.json) is what the server answers: every entry
// is provoked, and the test fails for an entry it cannot provoke and for a provocation that is not in the catalogue.
func TestErrorCatalogueMatchesTheServer(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	items := rundownItems(e, tok, id)
	akad := agendaByKey(t, items, "wedding.rundown_05")["id"].(string)
	var all string
	if err := e.pool.QueryRow(t.Context(), `SELECT id FROM planner_event_sessions WHERE project_id = $1 AND is_system`, id).Scan(&all); err != nil {
		t.Fatal(err)
	}
	other := e.login("other")
	oid := createProject(e, other, nil)
	foreignAgenda := addAgenda(e, other, oid, agendaBody(sessionIDs(e, other, oid)["akad"], "07:00", "", "x", nil))["id"].(string)
	foreignVendor := mustStatus(t, e.do(other, "POST", base(oid)+"/vendors", map[string]any{"sub_event_scope": "all", "category_key": "mc", "vendor_name": "Lain"}), 201).data()["vendor"].(map[string]any)["id"].(string)
	seminar := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{
		"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "Panitia"},
	})), 201).data()["project"].(map[string]any)["id"].(string)

	post := func(over map[string]any) resp {
		return e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["akad"], "07:00", "07:10", "Uji", over))
	}
	triggers := map[string]func() resp{
		"start_time: must be a time like 07:46 or 07.46": func() resp { return post(map[string]any{"start_time": "7.6"}) },
		"end_time: must not be before start_time unless ends_next_day is set": func() resp {
			return post(map[string]any{"start_time": "08:00", "end_time": "07:00"})
		},
		"end_time: must be earlier than start_time when ends_next_day is set": func() resp {
			return post(map[string]any{"ends_next_day": true})
		},
		"ends_next_day: requires end_time":              func() resp { return post(map[string]any{"end_time": nil, "ends_next_day": true}) },
		"session_id: must be a session of this project": func() resp { return post(map[string]any{"session_id": uuid.NewString()}) },
		"session_id: must not be the system session":    func() resp { return post(map[string]any{"session_id": all}) },
		"vendor_id: must be a vendor of this project":   func() resp { return post(map[string]any{"vendor_id": foreignVendor}) },
		"rundown_id: must be an agenda of this project": func() resp {
			return e.do(tok, "POST", base(id)+"/playlists", songBody("x", "y", map[string]any{"rundown_id": foreignAgenda}))
		},
		"ids: must list every agenda of the session exactly once": func() resp {
			return e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": sess["akad"], "ids": []string{akad}})
		},
		"mode: must be one of append, replace_template": func() resp { return applyAgendaTemplate(e, tok, id, map[string]any{"mode": "x"}) },
		"mode: must be append": func() resp {
			return applyPlaylistTemplate(e, tok, id, map[string]any{"mode": "replace_template"})
		},
		"confirm: required": func() resp { return applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template"}) },
		"spotify_url: must be an http or https URL": func() resp {
			return e.do(tok, "POST", base(id)+"/playlists", songBody("x", "y", map[string]any{"spotify_url": "ftp://x"}))
		},
		"playlist: not available for this event type": func() resp { return applyPlaylistTemplate(e, tok, seminar, nil) },
		"the session still has rundown agendas":       func() resp { return e.do(tok, "DELETE", base(id)+"/sessions/"+sess["akad"], nil) },
		"at most 500 agendas per project": func() resp {
			seedAgendas(e, id, sess["akad"], 500-33-0)

			return post(nil)
		},
		"at most 500 songs per project": func() resp {
			seedSongs(e, id, 470)

			return e.do(tok, "POST", base(id)+"/playlists", songBody("x", "y", nil))
		},
	}

	catalogue := readFixture(t, "planner-rundown-errors.json").([]any)
	seen := map[string]bool{}
	for _, raw := range catalogue {
		entry := raw.(map[string]any)
		msg := entry["message"].(string)
		seen[msg] = true
		trigger, ok := triggers[msg]
		if !ok {
			t.Errorf("the catalogue has %q and the test cannot provoke it", msg)
			continue
		}
		r := trigger()
		if got := fmt.Sprint(r.status, " ", r.code(), " ", r.message()); got != fmt.Sprint(int(entry["status"].(float64)), " ", entry["code"], " ", msg) {
			t.Errorf("%q: the server answers %s %s", msg, got, r.body)
		}
	}
	var extra []string
	for msg := range triggers {
		if !seen[msg] {
			extra = append(extra, msg)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("provoked but not in the catalogue: %v", extra)
	}
	if len(catalogue) != 17 {
		t.Errorf("the catalogue has %d entries, the test was written for 17", len(catalogue))
	}
}

// Every route needs a bearer token, B sees A's project as missing, and an archived project is read-only (06 BE 3).
type route struct {
	method, path string
	body         any
}

func rundownRoutes(sess map[string]string, a, song string) []route {
	body := agendaBody(sess["akad"], "07:00", "07:10", "x", nil)

	return []route{
		{"GET", "/rundowns", nil}, {"GET", "/rundowns/summary", nil}, {"GET", "/rundowns/export-wa", nil},
		{"POST", "/rundowns", body}, {"POST", "/rundowns/reorder", map[string]any{"session_id": sess["akad"], "ids": []string{a}}},
		{"POST", "/rundowns/apply-template", map[string]any{}},
		{"GET", "/rundowns/" + a, nil}, {"PUT", "/rundowns/" + a, body},
		{"PATCH", "/rundowns/" + a + "/status", map[string]any{"status": "in_progress"}}, {"DELETE", "/rundowns/" + a, nil},
		{"GET", "/playlists", nil}, {"GET", "/playlists/export-wa", nil}, {"POST", "/playlists", songBody("x", "y", nil)},
		{"POST", "/playlists/reorder", map[string]any{"ids": []string{song}}}, {"POST", "/playlists/apply-template", map[string]any{}},
		{"PUT", "/playlists/" + song, songBody("x", "y", nil)}, {"DELETE", "/playlists/" + song, nil},
	}
}

func TestEveryRouteNeedsATokenAndHidesTheProjectFromOthers(t *testing.T) {
	e := newEnv(t)
	owner, intruder := e.login("owner"), e.login("intruder")
	id := createProject(e, owner, nil)
	sess := sessionIDs(e, owner, id)
	a := addAgenda(e, owner, id, agendaBody(sess["akad"], "07:00", "", "x", nil))["id"].(string)
	song := addSong(e, owner, id, songBody("x", "y", nil))["id"].(string)
	routes := rundownRoutes(sess, a, song)
	if len(routes) != 17 {
		t.Fatalf("the module has 17 routes, the test lists %d", len(routes))
	}
	before := jsonOf([]any{rundownOverview(e, owner, id), playlistOverview(e, owner, id)})

	for _, rt := range routes {
		if r := e.do("", rt.method, base(id)+rt.path, rt.body); r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt.method, rt.path, r.status, r.body)
		}
		wantError(t, e.do(intruder, rt.method, base(id)+rt.path, rt.body), 404, "not_found", "project not found")
	}
	own := createProject(e, intruder, nil)
	for _, rt := range []route{
		{"GET", "/rundowns/" + a, nil}, {"PUT", "/rundowns/" + a, agendaBody(sessionIDs(e, intruder, own)["akad"], "07:00", "", "x", nil)},
		{"PATCH", "/rundowns/" + a + "/status", map[string]any{"status": "completed"}}, {"DELETE", "/rundowns/" + a, nil},
		{"PUT", "/playlists/" + song, songBody("x", "y", nil)}, {"DELETE", "/playlists/" + song, nil},
	} {
		wantError(t, e.do(intruder, rt.method, base(own)+rt.path, rt.body), 404, "not_found", "")
	}
	// ids that are not canonical UUIDs are 404 as well, never a query, whatever the method
	body := agendaBody(sess["akad"], "07:00", "", "x", nil)
	for _, bad := range []string{"nope", "{" + a + "}", strings.ToUpper(strings.ReplaceAll(a, "-", ""))} {
		for method, path := range map[string]string{"GET": "/rundowns/", "DELETE": "/rundowns/"} {
			wantError(t, e.do(owner, method, base(id)+path+bad, nil), 404, "not_found", "agenda not found")
		}
		wantError(t, e.do(owner, "PUT", base(id)+"/rundowns/"+bad, body), 404, "not_found", "agenda not found")
		wantError(t, e.do(owner, "PATCH", base(id)+"/rundowns/"+bad+"/status", map[string]any{"status": "pending"}), 404, "not_found", "agenda not found")
		wantError(t, e.do(owner, "DELETE", base(id)+"/playlists/"+bad, nil), 404, "not_found", "song not found")
		wantError(t, e.do(owner, "PUT", base(id)+"/playlists/"+bad, songBody("x", "y", nil)), 404, "not_found", "song not found")
	}
	if after := jsonOf([]any{rundownOverview(e, owner, id), playlistOverview(e, owner, id)}); after != before {
		t.Error("the intruder changed something")
	}
}

func TestArchivedProjectIsReadOnlyButStillReadable(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	a := rundownItems(e, tok, id)[0]["id"].(string)
	song := list(playlistOverview(e, tok, id)["items"])[0]["id"].(string)
	before := jsonOf([]any{rundownOverview(e, tok, id), playlistOverview(e, tok, id)})
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range rundownRoutes(sess, a, song) {
		r := e.do(tok, rt.method, base(id)+rt.path, rt.body)
		if rt.method == "GET" {
			if r.status != 200 {
				t.Errorf("GET %s on an archived project: %d %s", rt.path, r.status, r.body)
			}

			continue
		}
		wantError(t, r, 409, "project_read_only", "")
	}
	if after := jsonOf([]any{rundownOverview(e, tok, id), playlistOverview(e, tok, id)}); after != before {
		t.Error("an archived project changed")
	}
}

func TestMalformedBodiesAreInvalidJSON(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	a := rundownItems(e, tok, id)[0]["id"].(string)
	song := list(playlistOverview(e, tok, id)["items"])[0]["id"].(string)
	for _, rt := range []route{
		{"POST", "/rundowns", "{not json"}, {"POST", "/rundowns", `[]`}, {"PUT", "/rundowns/" + a, `{"activity_name": 7}`},
		{"PATCH", "/rundowns/" + a + "/status", `{"status": 1}`}, {"POST", "/rundowns/reorder", `{"ids": "x"}`},
		{"POST", "/rundowns/reorder", `{"session_id": "x"}`}, {"POST", "/rundowns/apply-template", `{"confirm": "yes"}`},
		{"POST", "/playlists", `{"song_title": ["x"]}`}, {"POST", "/playlists/reorder", `{"ids": [1]}`},
		{"POST", "/playlists/apply-template", `{"mode": 1}`}, {"PUT", "/playlists/" + song, `{"song_title": 1}`},
	} {
		wantError(t, e.do(tok, rt.method, base(id)+rt.path, rt.body), 400, "invalid_json", "")
	}
	// a body over 64 KiB is refused the same way
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", `{"activity_name": "`+strings.Repeat("a", 70<<10)+`"}`), 400, "invalid_json", "")
}

// The export is limited to 10 a minute per user, both exports together; the polling endpoint is not limited at all.
func TestExportRateLimitAndUnlimitedPolling(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	for i := 0; i < 60; i++ {
		if r := e.do(tok, "GET", base(id)+"/rundowns/summary", nil); r.status != 200 {
			t.Fatalf("poll %d: %d %s", i, r.status, r.body)
		}
	}
	for i := 0; i < 10; i++ {
		path := "/rundowns/export-wa"
		if i%2 == 1 {
			path = "/playlists/export-wa"
		}
		if r := e.do(tok, "GET", base(id)+path, nil); r.status != 200 {
			t.Fatalf("export %d: %d %s", i, r.status, r.body)
		}
	}
	wantError(t, e.do(tok, "GET", base(id)+"/rundowns/export-wa", nil), 429, "rate_limited", "")
	wantError(t, e.do(tok, "GET", base(id)+"/playlists/export-wa", nil), 429, "rate_limited", "")
	if r := e.do(tok, "GET", base(id)+"/rundowns/summary", nil); r.status != 200 {
		t.Errorf("the polling endpoint was limited: %d", r.status)
	}
	// another user has a limit of their own
	other := e.login("other")
	oid := createProject(e, other, nil)
	mustStatus(t, e.do(other, "GET", base(oid)+"/rundowns/export-wa", nil), 200)
}
