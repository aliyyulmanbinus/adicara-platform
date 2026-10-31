package http_test

import (
	"testing"
)

// seedAgendas inserts n plain agendas straight into the table (the limit is what is under test, not 500 requests).
func seedAgendas(e *env, projectID, sessionID string, n int) {
	e.t.Helper()
	exec(e, `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name, sort_order)
		SELECT gen_random_uuid(), $1, $2, '08:00', 'Agenda ' || g, g FROM generate_series(1, $3) g`, projectID, sessionID, n)
}

func seedSongs(e *env, projectID string, n int) {
	e.t.Helper()
	exec(e, `INSERT INTO planner_playlists (id, project_id, song_title, artist, sort_order)
		SELECT gen_random_uuid(), $1, 'Lagu ' || g, 'Penyanyi', g FROM generate_series(1, $2) g`, projectID, n)
}

// 500 agendas per project, checked under the project lock (06 BE 2.9): the 500th passes, the 501st does not, and two
// requests at 499 let exactly one through.
func TestAgendaLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	seedAgendas(e, id, sess["akad"], 498)

	post := func(i int) int {
		return e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["resepsi"], "10:00", "", "Baru", nil)).status
	}
	if got := post(0); got != 201 { // 499
		t.Fatalf("the 499th agenda: %d", got)
	}
	// 499 now; two at once: exactly one fits
	got := concurrently(2, post)
	created := 0
	for _, s := range got {
		switch s {
		case 201:
			created++
		case 409:
		default:
			t.Fatalf("status %d", s)
		}
	}
	if created != 1 {
		t.Fatalf("%v: %d of two concurrent requests passed the limit, want 1", got, created)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["resepsi"], "10:00", "", "Baru", nil)), 409, "agenda_limit_reached", "at most 500 agendas per project")
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 500 {
		t.Errorf("%d agendas, want 500", n)
	}
}

func TestPlaylistLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	seedSongs(e, id, 498)

	post := func(i int) int { return e.do(tok, "POST", base(id)+"/playlists", songBody("Baru", "X", nil)).status }
	if got := post(0); got != 201 {
		t.Fatalf("the 499th song: %d", got)
	}
	created := 0
	for _, s := range concurrently(2, post) {
		if s == 201 {
			created++
		} else if s != 409 {
			t.Fatalf("status %d", s)
		}
	}
	if created != 1 {
		t.Fatalf("%d of two concurrent requests passed the limit, want 1", created)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/playlists", songBody("Baru", "X", nil)), 409, "playlist_limit_reached", "at most 500 songs per project")
	if n := count(e, `SELECT count(*) FROM planner_playlists`); n != 500 {
		t.Errorf("%d songs, want 500", n)
	}
}

// A template that would pass the limit fails as a whole: nothing of it is created.
func TestTemplateThatPassesTheLimitCreatesNothing(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	seedAgendas(e, id, sess["akad"], 480)
	seedSongs(e, id, 480)

	wantError(t, applyAgendaTemplate(e, tok, id, nil), 409, "agenda_limit_reached", "")
	wantError(t, applyPlaylistTemplate(e, tok, id, nil), 409, "playlist_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 480 {
		t.Errorf("%d agendas after the refused template, want 480", n)
	}
	if n := count(e, `SELECT count(*) FROM planner_playlists`); n != 480 {
		t.Errorf("%d songs after the refused template, want 480", n)
	}
}
