package http_test

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestAgendaCreateNormalisesAndReturnsTheShape(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)

	a := addAgenda(e, tok, id, agendaBody(sess["akad"], "7.46", "07:50", "  Ijab Kabul  ", map[string]any{
		"pic_name": " Penghulu ", "description": "Ijab kabul\r\ndipimpin penghulu", "highlights": "- Satu\n- Dua",
	}))
	want := map[string]any{
		"session_id": sess["akad"], "session_name": "Akad Nikah", "session_icon": "📜", "scope_key": "akad",
		"start_time": "07:46", "end_time": "07:50", "ends_next_day": false, "duration_minutes": float64(4),
		"activity_name": "Ijab Kabul", "pic_name": "Penghulu", "description": "Ijab kabul\ndipimpin penghulu", "highlights": "- Satu\n- Dua",
		"status": "pending", "linked_songs": float64(0), "sort_order": float64(1),
	}
	for k, w := range want {
		if !reflect.DeepEqual(a[k], w) {
			t.Errorf("%s = %#v, want %#v", k, a[k], w)
		}
	}
	for _, k := range []string{"vendor", "template_key", "started_at", "completed_at"} {
		if _, ok := a[k]; ok {
			t.Errorf("%s must be omitted: %v", k, a)
		}
	}
	if _, err := uuid.Parse(a["id"].(string)); err != nil {
		t.Errorf("id = %v", a["id"])
	}

	// the order is per session: the next agenda of the akad is 2, the first of the resepsi is 1
	b := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:51", "08:00", "Doa bersama", nil))
	c := addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:15", "", "Resepsi", nil))
	if num(b["sort_order"]) != 2 || num(c["sort_order"]) != 1 {
		t.Errorf("sort_order = %v and %v", b["sort_order"], c["sort_order"])
	}
	// an open agenda has no end and no duration
	if _, ok := c["end_time"]; ok {
		t.Errorf("open agenda has an end_time: %v", c)
	}
	if _, ok := c["duration_minutes"]; ok {
		t.Errorf("open agenda has a duration: %v", c)
	}

	// GET returns the same agenda; a malformed or foreign id is 404
	got := get(e, tok, base(id)+"/rundowns/"+a["id"].(string))
	if !reflect.DeepEqual(got, a) {
		t.Errorf("GET = %v, POST = %v", got, a)
	}
	wantError(t, e.do(tok, "GET", base(id)+"/rundowns/not-a-uuid", nil), 404, "not_found", "agenda not found")
	wantError(t, e.do(tok, "GET", base(id)+"/rundowns/"+uuid.NewString(), nil), 404, "not_found", "agenda not found")
}

// PUT replaces everything but the status and the template key; moving an agenda to another session puts it last there.
func TestAgendaUpdateKeepsStatusAndTemplateKey(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	items := rundownItems(e, tok, id)
	a := agendaByKey(t, items, "wedding.rundown_12") // Iring-iring, resepsi
	aid := a["id"].(string)
	mustStatus(t, setStatus(e, tok, id, aid, "in_progress"), 200)

	body := agendaBody(sess["resepsi"], "09:37", "09:42", "Iring-iring (diubah)", map[string]any{"pic_name": ""})
	r := mustStatus(t, e.do(tok, "PUT", base(id)+"/rundowns/"+aid, body), 200).data()
	if r["activity_name"] != "Iring-iring (diubah)" || r["start_time"] != "09:37" || r["pic_name"] != "" || r["description"] != "" ||
		r["status"] != "in_progress" || r["template_key"] != "wedding.rundown_12" || r["started_at"] == nil || num(r["sort_order"]) != 4 {
		t.Fatalf("after PUT: %v", r)
	}

	// to another session: last there
	body = agendaBody(sess["akad"], "09:00", "09:05", "Pindah", nil)
	r = mustStatus(t, e.do(tok, "PUT", base(id)+"/rundowns/"+aid, body), 200).data()
	if r["scope_key"] != "akad" || num(r["sort_order"]) != 9 {
		t.Errorf("moved agenda = %v, want last (9) of the akad", r)
	}

	// an unknown agenda
	wantError(t, e.do(tok, "PUT", base(id)+"/rundowns/"+uuid.NewString(), body), 404, "not_found", "agenda not found")
}

func TestAgendaTimeRules(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)

	const timeMsg = "must be a time like 07:46 or 07.46"
	bad := []struct {
		name string
		body map[string]any
		msg  string
	}{
		{"seconds", agendaBody(sess["akad"], "07:46:00", "08:00", "x", nil), "start_time: " + timeMsg},
		{"24:00", agendaBody(sess["akad"], "24:00", "", "x", nil), "start_time: " + timeMsg},
		{"one-digit minute", agendaBody(sess["akad"], "07.6", "", "x", nil), "start_time: " + timeMsg},
		{"empty start", agendaBody(sess["akad"], "", "", "x", nil), "start_time: " + timeMsg},
		{"bad end", agendaBody(sess["akad"], "07:00", "25.00", "x", nil), "end_time: " + timeMsg},
		{"end before start", agendaBody(sess["akad"], "08:00", "07:59", "x", nil), "end_time: must not be before start_time unless ends_next_day is set"},
		{"next day, end after start", agendaBody(sess["akad"], "22:00", "23:00", "x", map[string]any{"ends_next_day": true}), "end_time: must be earlier than start_time when ends_next_day is set"},
		{"next day, end equal", agendaBody(sess["akad"], "22:00", "22:00", "x", map[string]any{"ends_next_day": true}), "end_time: must be earlier than start_time when ends_next_day is set"},
		{"next day without end", agendaBody(sess["akad"], "22:00", "", "x", map[string]any{"ends_next_day": true}), "ends_next_day: requires end_time"},
	}
	for _, c := range bad {
		wantError(t, e.do(tok, "POST", base(id)+"/rundowns", c.body), 400, "invalid_input", c.msg)
		t.Log("refused:", c.name)
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 0 {
		t.Fatalf("%d agendas were stored although every body was refused", n)
	}

	// accepted spellings and the marker of 0 minutes
	for _, in := range []string{"07.46", "7.46", "07:46", "7:46", " 07:46 "} {
		a := addAgenda(e, tok, id, agendaBody(sess["akad"], in, "", "x", nil))
		if a["start_time"] != "07:46" {
			t.Errorf("%q stored as %v", in, a["start_time"])
		}
	}
	z := addAgenda(e, tok, id, agendaBody(sess["akad"], "09:00", "09:00", "Penanda", nil))
	if num(z["duration_minutes"]) != 0 {
		t.Errorf("a marker has duration %v", z["duration_minutes"])
	}
}

// An event that goes past midnight: 22.00-23.00 and 23.30-00.30 span 150 minutes, and the second agenda lasts 60. The
// naive "time + 24 hours" arithmetic gives 1.500 or 60 (06 BE 2.1).
func TestSessionSpanAcrossMidnight(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)

	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "22:00", "23:00", "A", nil))
	b := addAgenda(e, tok, id, agendaBody(sess["resepsi"], "23:30", "00:30", "B", map[string]any{"ends_next_day": true}))
	if num(b["duration_minutes"]) != 60 || b["ends_next_day"] != true {
		t.Fatalf("B = %v", b)
	}
	ov := rundownOverview(e, tok, id)
	for _, s := range list(ov["sessions"]) {
		if s["scope_key"] != "resepsi" {
			continue
		}
		if num(s["span_minutes"]) != 150 || s["first_start"] != "22:00" || s["last_end"] != "00:30" {
			t.Errorf("session = %v, want span 150, 22:00 to 00:30", s)
		}
	}
	if num(ov["kpi"].(map[string]any)["total_span_minutes"]) != 150 {
		t.Errorf("kpi = %v", ov["kpi"])
	}
}

func TestAgendaTextValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	post := func(over map[string]any) resp {
		return e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["akad"], "07:00", "07:10", "Persiapan", over))
	}

	wantError(t, post(map[string]any{"activity_name": "  "}), 400, "invalid_input", "activity_name: required")
	wantError(t, post(map[string]any{"activity_name": strings.Repeat("a", 256)}), 400, "invalid_input", "activity_name: must be at most 255 characters")
	mustStatus(t, post(map[string]any{"activity_name": strings.Repeat("é", 255)}), 201) // runes, not bytes
	wantError(t, post(map[string]any{"activity_name": "a\x07b"}), 400, "invalid_input", "activity_name: must not contain control characters")
	wantError(t, post(map[string]any{"pic_name": strings.Repeat("a", 151)}), 400, "invalid_input", "pic_name: must be at most 150 characters")
	wantError(t, post(map[string]any{"description": strings.Repeat("a", 2001)}), 400, "invalid_input", "description: must be at most 2000 characters")
	wantError(t, post(map[string]any{"highlights": strings.Repeat("a", 2001)}), 400, "invalid_input", "highlights: must be at most 2000 characters")
	mustStatus(t, post(map[string]any{"description": strings.Repeat("a", 2000), "highlights": "- a\n- b"}), 201)

	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", `{"session_id": 5}`), 400, "invalid_json", "")
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", `not json`), 400, "invalid_json", "")
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody("", "07:00", "", "x", map[string]any{"session_id": nil})), 400, "invalid_input", "session_id: required")
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody(uuid.NewString(), "07:00", "", "x", nil)), 400, "invalid_scope", "session_id: must be a session of this project")
	// the system session `all` is not a place for agendas
	var all string
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM planner_event_sessions WHERE project_id = $1 AND is_system`, id).Scan(&all); err != nil {
		t.Fatal(err)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody(all, "07:00", "", "x", nil)), 400, "invalid_input", "session_id: must not be the system session")

	// a session of another project is the same invalid_scope, not a leak
	other := e.login("other")
	oid := createProject(e, other, nil)
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sessionIDs(e, other, oid)["akad"], "07:00", "", "x", nil)), 400, "invalid_scope", "session_id: must be a session of this project")
}

// The PIC can be a vendor of menu 04: the agenda then carries the vendor with the number the client opens in WhatsApp.
func TestAgendaVendorLink(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	v := mustStatus(t, e.do(tok, "POST", base(id)+"/vendors", map[string]any{
		"sub_event_scope": "all", "category_key": "mc", "vendor_name": "MC Budi", "pic_name": "Budi", "phone_number": "0812-3456-7890",
	}), 201).data()["vendor"].(map[string]any)
	vid := v["id"].(string)

	a := addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:31", "09:35", "Pembukaan acara", map[string]any{"vendor_id": vid, "pic_name": "Budi MC"}))
	vend := a["vendor"].(map[string]any)
	if vend["id"] != vid || vend["vendor_name"] != "MC Budi" || vend["category_label"] != "MC" || vend["pic_name"] != "Budi" ||
		vend["phone_number"] != "0812-3456-7890" || vend["formatted_wa_number"] != "6281234567890" {
		t.Fatalf("vendor = %v", vend)
	}

	// distinct PICs: the vendor counts once whatever the agenda's own text says; texts compare without case and padding
	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:36", "", "b", map[string]any{"vendor_id": vid}))
	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:37", "", "c", map[string]any{"pic_name": "MC"}))
	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:38", "", "d", map[string]any{"pic_name": " mc "}))
	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:39", "", "e", map[string]any{"pic_name": "Qori"}))
	addAgenda(e, tok, id, agendaBody(sess["resepsi"], "09:40", "", "f", nil)) // no PIC at all
	if n := num(rundownOverview(e, tok, id)["kpi"].(map[string]any)["distinct_pics"]); n != 3 {
		t.Errorf("distinct_pics = %d, want 3 (the vendor, mc, qori)", n)
	}

	// a vendor of another project, or none at all, is refused
	other := e.login("other")
	oid := createProject(e, other, nil)
	ov := mustStatus(t, e.do(other, "POST", base(oid)+"/vendors", map[string]any{"sub_event_scope": "all", "category_key": "mc", "vendor_name": "MC Lain"}), 201).data()["vendor"].(map[string]any)
	for _, bad := range []string{ov["id"].(string), uuid.NewString()} {
		wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["resepsi"], "10:00", "", "x", map[string]any{"vendor_id": bad})), 400, "invalid_input", "vendor_id: must be a vendor of this project")
	}

	// deleting the vendor clears the link and keeps the agenda
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/vendors/"+vid, nil), 204)
	got := get(e, tok, base(id)+"/rundowns/"+a["id"].(string))
	if _, ok := got["vendor"]; ok || got["pic_name"] != "Budi MC" {
		t.Errorf("agenda after the vendor was deleted = %v", got)
	}
}

func TestStatusInvariants(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	a := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:00", "07:20", "Persiapan", nil))
	b := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:21", "07:25", "Pembukaan", nil))
	aid, bid := a["id"].(string), b["id"].(string)
	read := func(agenda string) map[string]any { return get(e, tok, base(id)+"/rundowns/"+agenda) }

	// pending → in_progress: started_at set, completed_at empty
	r := mustStatus(t, setStatus(e, tok, id, aid, "in_progress"), 200).data()
	started := r["started_at"]
	if r["status"] != "in_progress" || started == nil || r["completed_at"] != nil {
		t.Fatalf("in_progress = %v", r)
	}
	// the same status again changes no time stamp
	r = mustStatus(t, setStatus(e, tok, id, aid, "in_progress"), 200).data()
	if r["started_at"] != started {
		t.Errorf("started_at moved from %v to %v", started, r["started_at"])
	}
	// completed: both stamps; the start is kept
	r = mustStatus(t, setStatus(e, tok, id, aid, "completed"), 200).data()
	done := r["completed_at"]
	if r["status"] != "completed" || r["started_at"] != started || done == nil {
		t.Fatalf("completed = %v", r)
	}
	r = mustStatus(t, setStatus(e, tok, id, aid, "completed"), 200).data()
	if r["completed_at"] != done {
		t.Errorf("completed_at moved from %v to %v", done, r["completed_at"])
	}
	// reopening keeps the start and drops the end
	r = mustStatus(t, setStatus(e, tok, id, aid, "in_progress"), 200).data()
	if r["started_at"] != started || r["completed_at"] != nil || r["status"] != "in_progress" {
		t.Fatalf("reopened = %v", r)
	}
	// two agendas may run at once (the meal, the entertainment and the handshakes of the PDF run in parallel)
	mustStatus(t, setStatus(e, tok, id, bid, "in_progress"), 200)
	if k := rundownOverview(e, tok, id)["kpi"].(map[string]any); num(k["in_progress"]) != 2 || num(k["pending"]) != 0 {
		t.Errorf("kpi = %v", k)
	}
	// PUT does not touch the status or the stamps
	mustStatus(t, e.do(tok, "PUT", base(id)+"/rundowns/"+aid, agendaBody(sess["akad"], "07:00", "07:20", "Persiapan 2", nil)), 200)
	if got := read(aid); got["status"] != "in_progress" || got["started_at"] != started {
		t.Errorf("PUT changed the status: %v", got)
	}
	// back to pending clears both
	r = mustStatus(t, setStatus(e, tok, id, aid, "pending"), 200).data()
	if r["status"] != "pending" || r["started_at"] != nil || r["completed_at"] != nil {
		t.Fatalf("pending = %v", r)
	}

	wantError(t, setStatus(e, tok, id, aid, "done"), 400, "invalid_input", "status: must be one of pending, in_progress, completed")
	wantError(t, setStatus(e, tok, id, aid, ""), 400, "invalid_input", "status: must be one of pending, in_progress, completed")
	wantError(t, setStatus(e, tok, id, uuid.NewString(), "pending"), 404, "not_found", "agenda not found")
	wantError(t, setStatus(e, tok, id, "x", "pending"), 404, "not_found", "agenda not found")
}

func TestReorderRundowns(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)

	ids := func(scope string) []string {
		var out []string
		for _, a := range rundownItems(e, tok, id) {
			if a["scope_key"] == scope {
				out = append(out, a["id"].(string))
			}
		}

		return out
	}
	reorder := func(session string, order []string) resp {
		return e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": session, "ids": order})
	}
	akad, resepsi := ids("akad"), ids("resepsi")
	rev := make([]string, len(akad))
	for i, v := range akad {
		rev[len(akad)-1-i] = v
	}

	r := mustStatus(t, reorder(sess["akad"], rev), 204)
	if len(r.body) != 0 {
		t.Errorf("204 has a body: %s", r.body)
	}
	if got := ids("akad"); !reflect.DeepEqual(got, rev) {
		t.Fatalf("akad order = %v, want %v", got, rev)
	}
	var orders []int64
	for _, a := range rundownItems(e, tok, id) {
		if a["scope_key"] == "akad" {
			orders = append(orders, num(a["sort_order"]))
		}
	}
	if !reflect.DeepEqual(orders, []int64{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("akad sort_order = %v, want 1..8", orders)
	}
	if got := ids("resepsi"); !reflect.DeepEqual(got, resepsi) {
		t.Errorf("another session's order changed: %v", got)
	}

	// every refusal leaves the order alone
	foreign := ids("resepsi")[0]
	dup := append([]string{rev[0]}, rev[:len(rev)-1]...)
	cases := map[string][]string{
		"a repeated id":            dup,
		"one id missing":           rev[1:],
		"an extra id":              append(append([]string{}, rev...), foreign),
		"an id of another session": append([]string{foreign}, rev[1:]...),
		"an unknown id":            append([]string{uuid.NewString()}, rev[1:]...),
		"an empty list":            {},
	}
	for name, order := range cases {
		wantError(t, reorder(sess["akad"], order), 400, "invalid_input", "ids: must list every agenda of the session exactly once")
		if got := ids("akad"); !reflect.DeepEqual(got, rev) {
			t.Fatalf("%s changed the order to %v", name, got)
		}
	}
	wantError(t, reorder(uuid.NewString(), rev), 400, "invalid_scope", "session_id: must be a session of this project")
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns/reorder", `{"session_id":"`+sess["akad"]+`"}`), 400, "invalid_input", "ids: must list every agenda of the session exactly once")
}

// Two reorders at the same moment end in one of the two orders, never in a mix with repeated sort_order values.
func TestReorderRundownsConcurrently(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	var akad []string
	for _, a := range rundownItems(e, tok, id) {
		if a["scope_key"] == "akad" {
			akad = append(akad, a["id"].(string))
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		order := append([]string{}, akad...)
		for j := 0; j < i; j++ { // a different rotation each
			order = append(order[1:], order[0])
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r := e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": sess["akad"], "ids": order}); r.status != 204 {
				t.Errorf("reorder = %d %s", r.status, r.body)
			}
		}()
	}
	wg.Wait()

	seen := map[int64]bool{}
	for _, a := range rundownItems(e, tok, id) {
		if a["scope_key"] == "akad" {
			seen[num(a["sort_order"])] = true
		}
	}
	if len(seen) != len(akad) {
		t.Errorf("sort_order values = %v, want %d distinct", seen, len(akad))
	}
}

func TestDeleteRundownKeepsSongsUnlinked(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	a := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:46", "07:50", "Ijab Kabul", nil))
	s := addSong(e, tok, id, songBody("Akad", "Payung Teduh", map[string]any{"rundown_id": a["id"]}))
	if s["rundown_id"] != a["id"] || s["agenda"].(map[string]any)["activity_name"] != "Ijab Kabul" {
		t.Fatalf("song = %v", s)
	}
	if got := get(e, tok, base(id)+"/rundowns/"+a["id"].(string)); num(got["linked_songs"]) != 1 {
		t.Errorf("linked_songs = %v", got["linked_songs"])
	}

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/rundowns/"+a["id"].(string), nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/rundowns/"+a["id"].(string), nil), 404, "not_found", "agenda not found")
	items := list(playlistOverview(e, tok, id)["items"])
	if len(items) != 1 {
		t.Fatalf("songs = %v", items)
	}
	if _, ok := items[0]["rundown_id"]; ok {
		t.Errorf("the song still points at the deleted agenda: %v", items[0])
	}
	if _, ok := items[0]["agenda"]; ok {
		t.Errorf("the song still carries an agenda: %v", items[0])
	}
}
