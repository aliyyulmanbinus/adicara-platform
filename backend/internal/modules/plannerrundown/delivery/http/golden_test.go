package http_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

// firstDiff returns the path and the two values where a and b first differ, or "" when they are equal.
func firstDiff(a, b any, path string) (string, any, any) {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			return path, a, b
		}
		keys := map[string]bool{}
		for k := range x {
			keys[k] = true
		}
		for k := range y {
			keys[k] = true
		}
		for k := range keys {
			xv, xok := x[k]
			yv, yok := y[k]
			if xok != yok {
				return path + "." + k, xv, yv
			}
			if p, l, r := firstDiff(xv, yv, path+"."+k); p != "" {
				return p, l, r
			}
		}

		return "", nil, nil
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return path + " (length)", a, b
		}
		for i := range x {
			if p, l, r := firstDiff(x[i], y[i], path+"["+strconv.Itoa(i)+"]"); p != "" {
				return p, l, r
			}
		}

		return "", nil, nil
	}
	if !reflect.DeepEqual(a, b) {
		return path, a, b
	}

	return "", nil, nil
}

// compareFixture fails unless got equals the fixture, ignoring generated ids and time stamps (see normalise).
func compareFixture(t *testing.T, name, key string, got any) {
	t.Helper()
	want := readFixture(t, name).(map[string]any)[key]
	if path, g, w := firstDiff(normalise(got, map[string]int{}), normalise(want, map[string]int{}), key); path != "" {
		gj, _ := json.Marshal(g)
		wj, _ := json.Marshal(w)
		t.Fatalf("%s differs from the response at %s\n  response: %.300s\n  fixture:  %.300s", name, path, gj, wj)
	}
}

// The sample project (Proyek Contoh) holds the template as the wizard made it: 33 agendas and 30 songs, all pending
// (06 BE 2.6 and 6). The figures are those of the spec: spans 100 / 120 / 225 = 445 minutes, 17 distinct PICs.
func TestSampleProjectGoldenRundown(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	ov := rundownOverview(e, tok, id)
	k := ov["kpi"].(map[string]any)
	want := map[string]int64{"sessions": 3, "total_agendas": 33, "completed": 0, "in_progress": 0, "pending": 33, "total_span_minutes": 445, "distinct_pics": 17}
	for key, w := range want {
		if num(k[key]) != w {
			t.Errorf("kpi.%s = %v, want %d", key, k[key], w)
		}
	}
	spans := map[string][3]any{"lamaran": {100, "09:00", "10:40"}, "akad": {120, "07:00", "09:00"}, "resepsi": {225, "09:15", "13:00"}}
	got := map[string]bool{}
	for _, s := range list(ov["sessions"]) {
		w := spans[s["scope_key"].(string)]
		if int(num(s["span_minutes"])) != w[0] || s["first_start"] != w[1] || s["last_end"] != w[2] {
			t.Errorf("session %v = %v, want %v", s["scope_key"], s, w)
		}
		got[s["scope_key"].(string)] = true
	}
	if len(got) != 3 {
		t.Errorf("sessions = %v", got)
	}

	// the two open agendas (lamaran 11 "10.40 - selesai", resepsi 9 "09.15") keep no end and no duration
	items := list(ov["items"])
	for _, key := range []string{"wedding.lamaran_11", "wedding.rundown_09"} {
		a := agendaByKey(t, items, key)
		if _, ok := a["end_time"]; ok {
			t.Errorf("%s has an end_time: %v", key, a)
		}
		if _, ok := a["duration_minutes"]; ok {
			t.Errorf("%s has a duration: %v", key, a)
		}
	}
	// the durations of the akad are the differences of the clock, not the PDF's "+1" (06 BE 1)
	var akad []int64
	for _, a := range items {
		if a["scope_key"] == "akad" {
			akad = append(akad, num(a["duration_minutes"]))
		}
	}
	if !reflect.DeepEqual(akad, []int64{20, 4, 9, 9, 4, 9, 9, 49}) {
		t.Errorf("akad durations = %v", akad)
	}

	songs := playlistOverview(e, tok, id)
	pk := songs["kpi"].(map[string]any)
	if num(pk["total_tracks"]) != 30 || num(pk["mapped_moments"]) != 6 || num(pk["ready"]) != 0 || num(pk["needs_review"]) != 30 || num(pk["missing_file"]) != 0 ||
		num(pk["by_format"].(map[string]any)["digital_playback"]) != 30 {
		t.Errorf("playlist kpi = %v", pk)
	}

	compareFixture(t, "planner-rundowns.json", "overview", ov)
	compareFixture(t, "planner-playlists.json", "overview", songs)
}

// The dashboard widget of menu 01 for the sample project: 3 sessions, 33 agendas, 30 songs, nothing live yet.
func TestSampleProjectDashboardWidget(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	widget := func() map[string]any {
		return get(e, tok, base(id)+"/dashboard")["rundown"].(map[string]any)
	}
	w := widget()
	if num(w["sessions"]) != 3 || num(w["agendas"]) != 33 || num(w["completed"]) != 0 || num(w["in_progress"]) != 0 || num(w["songs"]) != 30 || w["current_agenda"] != nil {
		t.Fatalf("widget = %v", w)
	}
	// the Lamaran of the sample project has a date (the start of its task, 15 Oct 2026), so it is the next session
	next := w["next_session"].(map[string]any)
	if next["name"] != "Lamaran" || next["date"] != "2026-10-15" {
		t.Errorf("next_session = %v", next)
	}

	// Lamaran done and the fifth agenda of the akad running: the widget names it
	items := rundownItems(e, tok, id)
	for _, a := range items {
		if a["scope_key"] == "lamaran" {
			mustStatus(t, setStatus(e, tok, id, a["id"].(string), "completed"), 200)
		}
	}
	mustStatus(t, setStatus(e, tok, id, agendaByKey(t, items, "wedding.rundown_05")["id"].(string), "in_progress"), 200)
	w = widget()
	cur := w["current_agenda"].(map[string]any)
	if num(w["completed"]) != 11 || num(w["in_progress"]) != 1 || cur["activity"] != "Ijab Kabul" || cur["session"] != "Akad Nikah" {
		t.Errorf("widget = %v", w)
	}
}

// The illustration of Hari-H in the frontend spec (Lamaran done, the first four agendas of the akad done and the fifth
// running) is what the live endpoint answers for those statuses; the WhatsApp text of the akad is the one of the fixture.
func TestHariHLiveAndWhatsAppMatchTheFixtures(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)

	text := get(e, tok, base(id)+"/rundowns/export-wa?session_id="+sess["akad"])["text"]
	wantText := readFixture(t, "planner-rundown-wa-akad.json").(map[string]any)["text"]
	if text != wantText {
		t.Fatalf("whatsapp text of the akad:\n%s\n--- fixture\n%s", text, wantText)
	}

	items := rundownItems(e, tok, id)
	akad := 0
	for _, a := range items {
		switch {
		case a["scope_key"] == "lamaran":
			mustStatus(t, setStatus(e, tok, id, a["id"].(string), "completed"), 200)
		case a["scope_key"] == "akad":
			akad++
			switch {
			case akad <= 4:
				mustStatus(t, setStatus(e, tok, id, a["id"].(string), "completed"), 200)
			case akad == 5:
				mustStatus(t, setStatus(e, tok, id, a["id"].(string), "in_progress"), 200)
			}
		}
	}
	live := get(e, tok, base(id)+"/rundowns/summary")
	if live["server_time"] != "2026-10-04T05:00:00Z" {
		t.Errorf("server_time = %v (the clock of the test is fixed)", live["server_time"])
	}
	compareFixture(t, "planner-rundown-live.json", "live", live)
}
