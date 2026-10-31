package http_test

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

func applyPlaylistTemplate(e *env, tok, id string, body any) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/playlists/apply-template", body)
}

// The wizard creates the template once, inside the project transaction (06 BE 2.6); a type without a playlist is refused
// for it and no project remains.
func TestWizardCreatesTheTemplates(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{"rundown": true, "playlist": true}})), 201)
	applied := r.data()["applied"].(map[string]any)
	id := r.data()["project"].(map[string]any)["id"].(string)
	for key, want := range map[string]int64{"rundown": 33, "playlist": 30} {
		a := applied[key].(map[string]any)
		if num(a["created"]) != want || num(a["skipped"]) != 0 || len(list(a["warnings"])) != 0 {
			t.Errorf("applied.%s = %v", key, a)
		}
	}
	if _, ok := applied["rundown"].(map[string]any)["reset"]; ok {
		t.Errorf("the wizard result carries reset: %v", applied["rundown"])
	}

	items := rundownItems(e, tok, id)
	if len(items) != 33 {
		t.Fatalf("%d agendas", len(items))
	}
	for _, a := range items {
		if a["status"] != "pending" || a["vendor"] != nil || num(a["linked_songs"]) != 0 {
			t.Fatalf("a template agenda is not plain: %v", a)
		}
	}
	for i, s := range list(playlistOverview(e, tok, id)["items"]) {
		want := domain.WeddingPlaylist[i]
		if s["song_title"] != want.Title || s["artist"] != want.Artist || s["moment_tag"] != want.MomentTag || s["template_key"] != want.Key ||
			s["audio_format"] != "digital_playback" || s["status"] != "needs_review" || s["rundown_id"] != nil || num(s["sort_order"]) != int64(i+1) {
			t.Fatalf("song %d = %v", i, s)
		}
	}

	// a seminar has agendas but no songs: asking for the playlist is refused, nothing is left behind
	seminar := func(templates map[string]any) resp {
		return e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{
			"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "Panitia"}, "templates": templates,
		}))
	}
	wantError(t, seminar(map[string]any{"rundown": true, "playlist": true}), 400, "invalid_template", "")
	if n := count(e, `SELECT count(*) FROM planner_projects WHERE title = 'Seminar'`); n != 0 {
		t.Errorf("%d projects remain after the refused wizard", n)
	}
	s := mustStatus(t, seminar(map[string]any{"rundown": true}), 201).data()
	if n := num(s["applied"].(map[string]any)["rundown"].(map[string]any)["created"]); n != 9 {
		t.Errorf("seminar rundown created %d, want 9", n)
	}
}

func TestAppendIsIdempotentAndFillsGaps(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	for _, body := range []any{nil, map[string]any{}, map[string]any{"mode": "append"}} {
		got := mustStatus(t, applyAgendaTemplate(e, tok, id, body), 200).data()
		if num(got["created"]) != 0 || num(got["skipped"]) != 33 || num(got["reset"]) != 0 || num(got["kept"]) != 0 || len(list(got["warnings"])) != 0 {
			t.Fatalf("append %v = %v", body, got)
		}
	}
	got := mustStatus(t, applyPlaylistTemplate(e, tok, id, nil), 200).data()
	if num(got["created"]) != 0 || num(got["skipped"]) != 30 {
		t.Fatalf("playlist append = %v", got)
	}

	// a deleted template agenda comes back alone, last in its session (the highest sort_order of the akad was 8)
	items := rundownItems(e, tok, id)
	gone := agendaByKey(t, items, "wedding.rundown_03")
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/rundowns/"+gone["id"].(string), nil), 204)
	got = mustStatus(t, applyAgendaTemplate(e, tok, id, nil), 200).data()
	if num(got["created"]) != 1 || num(got["skipped"]) != 32 {
		t.Fatalf("append after a delete = %v", got)
	}
	back := agendaByKey(t, rundownItems(e, tok, id), "wedding.rundown_03")
	if back["id"] == gone["id"] || num(back["sort_order"]) != 9 || back["activity_name"] != "Pembacaan ayat suci" {
		t.Errorf("recreated agenda = %v", back)
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 33 {
		t.Errorf("%d agendas", n)
	}
}

// Two clicks at the same moment must not double the rundown: the project lock and the unique index hold the line.
func TestAppendConcurrentlyCreatesOnce(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	var wg sync.WaitGroup
	created := make([]int64, 4)
	for i := range created {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := applyAgendaTemplate(e, tok, id, nil)
			if r.status != 200 {
				t.Errorf("apply = %d %s", r.status, r.body)
				return
			}
			created[i] = num(r.data()["created"])
		}()
	}
	wg.Wait()
	var total int64
	for _, c := range created {
		total += c
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 33 || total != 33 {
		t.Errorf("%d agendas in the table and %d reported as created, want 33", n, total)
	}
}

func TestSessionRemovedBySomeoneIsSkippedWithAWarning(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+sess["lamaran"]+"?delete_agendas=true", nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 22 {
		t.Fatalf("%d agendas after deleting the lamaran", n)
	}
	got := mustStatus(t, applyAgendaTemplate(e, tok, id, nil), 200).data()
	w := list(got["warnings"])
	if num(got["created"]) != 0 || num(got["skipped"]) != 22 || len(w) != 1 || w[0]["code"] != "scope_missing" || num(w[0]["count"]) != 11 {
		t.Errorf("append = %v", got)
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 22 {
		t.Errorf("%d agendas: the template was applied to a session that is gone", n)
	}
}

// replace_template resets the template agendas that have not started, in place (06 BE 2.6).
func TestReplaceTemplateResetsInPlace(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	items := rundownItems(e, tok, id)
	ids := func(n int) string {
		return agendaByKey(t, items, fmt.Sprintf("wedding.rundown_%02d", n))["id"].(string)
	}

	// the user changes agenda 12 (pending, with a vendor and a song), 5 (running) and 6 (done), and adds one of their own
	vendor := mustStatus(t, e.do(tok, "POST", base(id)+"/vendors", map[string]any{
		"sub_event_scope": "all", "category_key": "mc", "vendor_name": "MC Budi", "phone_number": "081234567890",
	}), 201).data()["vendor"].(map[string]any)["id"].(string)
	edit := func(n int, scope, name string, over map[string]any) {
		mustStatus(t, e.do(tok, "PUT", base(id)+"/rundowns/"+ids(n), agendaBody(sess[scope], "11:11", "11:22", name, over)), 200)
	}
	edit(12, "resepsi", "Diubah pengguna", map[string]any{"vendor_id": vendor, "pic_name": "Orang lain", "description": "kata saya"})
	edit(5, "akad", "Berjalan diubah", nil)
	edit(6, "akad", "Selesai diubah", nil)
	mustStatus(t, setStatus(e, tok, id, ids(5), "in_progress"), 200)
	mustStatus(t, setStatus(e, tok, id, ids(6), "completed"), 200)
	song := addSong(e, tok, id, songBody("Iring", "X", map[string]any{"rundown_id": ids(12)}))
	own := addAgenda(e, tok, id, agendaBody(sess["akad"], "10:00", "10:05", "Buatan sendiri", nil))

	// the user also puts the resepsi in the reverse order; a reset gives the template order back
	var templateOrder []string
	for _, a := range rundownItems(e, tok, id) {
		if a["scope_key"] == "resepsi" {
			templateOrder = append(templateOrder, a["id"].(string))
		}
	}
	reversed := make([]string, len(templateOrder))
	for i, v := range templateOrder {
		reversed[len(templateOrder)-1-i] = v
	}
	mustStatus(t, e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": sess["resepsi"], "ids": reversed}), 204)

	// no confirm: refused, and nothing changed
	wantError(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	wantError(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": false}), 400, "invalid_input", "confirm: required")
	if got := get(e, tok, base(id)+"/rundowns/"+ids(12)); got["activity_name"] != "Diubah pengguna" {
		t.Fatalf("a refused replace changed the agenda: %v", got)
	}

	got := mustStatus(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	if num(got["created"]) != 0 || num(got["skipped"]) != 0 || num(got["reset"]) != 31 || num(got["kept"]) != 2 || len(list(got["warnings"])) != 0 {
		t.Fatalf("replace = %v", got)
	}

	back := get(e, tok, base(id)+"/rundowns/"+ids(12))
	if back["activity_name"] != "Iring-iring" || back["start_time"] != "09:36" || back["end_time"] != "09:40" || back["pic_name"] != "CMP & CMW" ||
		back["description"] != "Pengantin dan iring-iringannya berjalan ke pelaminan." || num(back["sort_order"]) != 4 {
		t.Errorf("agenda 12 is not the template again: %v", back)
	}
	vend, _ := back["vendor"].(map[string]any)
	if vend["id"] != vendor || num(back["linked_songs"]) != 1 || back["status"] != "pending" {
		t.Errorf("agenda 12 lost its vendor or song: %v", back)
	}
	if s := list(playlistOverview(e, tok, id)["items"]); s[len(s)-1]["id"] != song["id"] || s[len(s)-1]["rundown_id"] != ids(12) {
		t.Errorf("the song lost its agenda: %v", s[len(s)-1])
	}
	for n, name := range map[int]string{5: "Berjalan diubah", 6: "Selesai diubah"} {
		if a := get(e, tok, base(id)+"/rundowns/"+ids(n)); a["activity_name"] != name || a["start_time"] != "11:11" {
			t.Errorf("agenda %d was touched: %v", n, a)
		}
	}
	if a := get(e, tok, base(id)+"/rundowns/"+own["id"].(string)); a["activity_name"] != "Buatan sendiri" {
		t.Errorf("the user's own agenda was touched: %v", a)
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 34 {
		t.Errorf("%d agendas, want 33 + 1 of the user", n)
	}

	// the order of the template is back in the resepsi, whatever the user had made of it
	var order []string
	var sortOrders []int64
	for _, a := range rundownItems(e, tok, id) {
		if a["scope_key"] == "resepsi" {
			order = append(order, a["id"].(string))
			sortOrders = append(sortOrders, num(a["sort_order"]))
		}
	}
	if !reflect.DeepEqual(order, templateOrder) || !reflect.DeepEqual(sortOrders, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}) {
		t.Errorf("resepsi after the reset: order %v, sort_order %v", order, sortOrders)
	}
}

func TestReplaceTemplateEdgeCases(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	// nothing to reset: no confirm needed, and it creates what is missing
	empty := createProject(e, tok, nil)
	got := mustStatus(t, applyAgendaTemplate(e, tok, empty, map[string]any{"mode": "replace_template"}), 200).data()
	if num(got["created"]) != 33 || num(got["reset"]) != 0 || num(got["kept"]) != 0 {
		t.Errorf("replace on an empty rundown = %v", got)
	}

	// an agenda the user moved to another session is theirs: counted as skipped, not reset
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	a := agendaByKey(t, rundownItems(e, tok, id), "wedding.rundown_12")
	mustStatus(t, e.do(tok, "PUT", base(id)+"/rundowns/"+a["id"].(string), agendaBody(sess["akad"], "09:36", "09:40", "Pindah sesi", nil)), 200)
	got = mustStatus(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	if num(got["reset"]) != 32 || num(got["skipped"]) != 1 || num(got["created"]) != 0 {
		t.Errorf("replace with a moved agenda = %v", got)
	}
	if moved := get(e, tok, base(id)+"/rundowns/"+a["id"].(string)); moved["activity_name"] != "Pindah sesi" || moved["scope_key"] != "akad" {
		t.Errorf("the moved agenda was reset: %v", moved)
	}

	wantError(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_all"}), 400, "invalid_input", "mode: must be one of append, replace_template")
	wantError(t, applyAgendaTemplate(e, tok, id, `{"mode": 3}`), 400, "invalid_json", "")
}

func TestPlaylistTemplateOnlyAppends(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	wantError(t, applyPlaylistTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 400, "invalid_input", "mode: must be append")
	wantError(t, applyPlaylistTemplate(e, tok, id, map[string]any{"mode": "x"}), 400, "invalid_input", "mode: must be one of append, replace_template")

	seminar := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{
		"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "Panitia"},
	})), 201).data()["project"].(map[string]any)["id"].(string)
	wantError(t, applyPlaylistTemplate(e, tok, seminar, nil), 400, "invalid_template", "playlist: not available for this event type")
}

// Every event type has its own rundown template, in the sessions its registry seeds; the wedding is the only one with songs.
func TestEveryEventTypeGetsItsRundown(t *testing.T) {
	e := newEnv(t)
	for _, spec := range pd.DefaultRegistry().All() {
		t.Run(string(spec.Key), func(t *testing.T) {
			tok := e.login("u_" + string(spec.Key)) // the two exports are limited per user, and every type uses both
			tmpl := domain.AgendasFor(spec.Key)
			templates := map[string]any{"rundown": true}
			if len(domain.PlaylistFor(spec.Key)) > 0 {
				templates["playlist"] = true
			}
			body := project(map[string]any{"event_type": string(spec.Key), "title": "T " + string(spec.Key), "metadata": metadataFor(spec), "templates": templates})
			r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", body), 201)
			id := r.data()["project"].(map[string]any)["id"].(string)
			if n := num(r.data()["applied"].(map[string]any)["rundown"].(map[string]any)["created"]); n != int64(len(tmpl)) {
				t.Fatalf("created %d agendas, want %d", n, len(tmpl))
			}

			ov := rundownOverview(e, tok, id)
			if k := ov["kpi"].(map[string]any); num(k["total_agendas"]) != int64(len(tmpl)) || num(k["sessions"]) != int64(len(spec.Sessions)) {
				t.Fatalf("kpi = %v", k)
			}
			if len(list(ov["sessions"])) != len(spec.Sessions) {
				t.Fatalf("sessions = %v", ov["sessions"])
			}
			items := list(ov["items"])
			for i, tm := range tmpl {
				want := tm.Key
				found := false
				for _, it := range items {
					if it["template_key"] == want {
						found = it["scope_key"] == tm.ScopeKey && it["activity_name"] == tm.Name && it["start_time"] == tm.Start
					}
				}
				if !found {
					t.Fatalf("template agenda %d (%s) is not in the rundown as defined", i, want)
				}
			}

			walkTheWholeMenu(t, e, tok, id, spec, len(tmpl))

			// the whole flow on the first session: status, reorder, delete session
			first := spec.Sessions[0].ScopeKey
			var ofFirst []string
			for _, it := range items {
				if it["scope_key"] == first {
					ofFirst = append(ofFirst, it["id"].(string))
				}
			}
			mustStatus(t, setStatus(e, tok, id, ofFirst[0], "in_progress"), 200)
			rev := make([]string, len(ofFirst))
			for i, v := range ofFirst {
				rev[len(ofFirst)-1-i] = v
			}
			mustStatus(t, e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": sessionIDs(e, tok, id)[first], "ids": rev}), 204)
			wantError(t, e.do(tok, "DELETE", base(id)+"/sessions/"+sessionIDs(e, tok, id)[first], nil), 409, "session_in_use", "")
			mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+sessionIDs(e, tok, id)[first]+"?delete_agendas=true", nil), 204)
			if n := count(e, `SELECT count(*) FROM planner_rundowns WHERE project_id = $1`, id); n != len(tmpl)-len(ofFirst) {
				t.Errorf("%d agendas after deleting %q, want %d", n, first, len(tmpl)-len(ofFirst))
			}
			if _, err := uuid.Parse(id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// walkTheWholeMenu is what a type has besides its template (06 README 4): replace_template on a rundown nobody touched
// resets all of it, the WhatsApp texts carry the type's own title, icon and sessions and nothing of the wedding, every
// moment tag of the registry is a tag a song can have, and the widget of the dashboard counts the template.
func walkTheWholeMenu(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec, agendas int) {
	t.Helper()
	wantError(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	got := mustStatus(t, applyAgendaTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	if num(got["reset"]) != int64(agendas) || num(got["kept"]) != 0 || num(got["created"]) != 0 {
		t.Errorf("replace_template on an untouched rundown = %v, want reset %d", got, agendas)
	}
	got = mustStatus(t, applyAgendaTemplate(e, tok, id, nil), 200).data()
	if num(got["created"]) != 0 || num(got["skipped"]) != int64(agendas) {
		t.Errorf("append after replace_template = %v", got)
	}

	text := get(e, tok, base(id)+"/rundowns/export-wa")["text"].(string)
	if !strings.Contains(text, "T "+string(spec.Key)) || !strings.Contains(text, spec.Icon) || !strings.Contains(text, "RUNDOWN ACARA") {
		t.Errorf("the rundown text lacks the title, the icon or the heading:\n%s", text)
	}
	for _, s := range spec.Sessions {
		if !strings.Contains(strings.ToUpper(text), strings.ToUpper(s.Name)) {
			t.Errorf("session %q is missing from the rundown text", s.Name)
		}
	}
	if spec.Key != pd.EventWedding {
		for _, word := range []string{"akad", "resepsi", "lamaran", "pengantin", "cmp"} {
			if strings.Contains(strings.ToLower(text), word) {
				t.Errorf("the rundown text of %s mentions %q", spec.Key, word)
			}
		}
	}

	hasSongs := len(domain.PlaylistFor(spec.Key)) > 0
	if hasSongs {
		mustStatus(t, applyPlaylistTemplate(e, tok, id, nil), 200)
	}
	first := rundownItems(e, tok, id)[0]["id"].(string)
	for i, tag := range spec.MomentTags {
		over := map[string]any{"moment_tag": tag}
		if i == 0 {
			over["rundown_id"] = first
		}
		addSong(e, tok, id, songBody("Lagu "+tag, "Artis", over))
	}
	want := int64(len(spec.MomentTags))
	if hasSongs {
		want += int64(len(domain.PlaylistFor(spec.Key)))
	}
	if kpi := playlistOverview(e, tok, id)["kpi"].(map[string]any); num(kpi["total_tracks"]) != want {
		t.Errorf("total_tracks = %v, want %d", kpi["total_tracks"], want)
	}
	songs := get(e, tok, base(id)+"/playlists/export-wa")["text"].(string)
	if !strings.Contains(songs, "SOUND CUE T "+string(spec.Key)) || !strings.Contains(songs, "Lagu "+spec.MomentTags[0]) {
		t.Errorf("the sound cue text lacks the title or the linked song:\n%s", songs)
	}
	if len(spec.MomentTags) > 1 && !strings.Contains(songs, "*"+spec.MomentTags[1]+"*") {
		t.Errorf("the songs without an agenda are not grouped by their moment tag:\n%s", songs)
	}

	widget := get(e, tok, base(id)+"/dashboard")["rundown"].(map[string]any)
	if num(widget["agendas"]) != int64(agendas) || num(widget["sessions"]) != int64(len(spec.Sessions)) {
		t.Errorf("dashboard widget = %v", widget)
	}
}
