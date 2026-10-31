package http_test

import (
	"context"
	"testing"
	"time"
)

// holdProjectLock keeps the project row locked from the test, the way another request would while it adds a row or
// reorders. It uses FOR NO KEY UPDATE: that conflicts with the FOR UPDATE of the module's own lock, but not with the
// FOR KEY SHARE that a foreign key check takes, so a request that skips the lock is not held up by accident.
func holdProjectLock(e *env, projectID string) (release func()) {
	e.t.Helper()
	tx, err := e.pool.Begin(context.Background())
	if err != nil {
		e.t.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(context.Background(), `SELECT id FROM planner_projects WHERE id = $1 FOR NO KEY UPDATE`, projectID).Scan(&id); err != nil {
		e.t.Fatal(err)
	}

	return func() {
		if err := tx.Commit(context.Background()); err != nil {
			e.t.Fatal(err)
		}
	}
}

// answersWithin runs call and reports whether it returned within d.
func answersWithin(d time.Duration, call func() resp) (resp, bool) {
	done := make(chan resp, 1)
	go func() { done <- call() }()
	select {
	case r := <-done:
		return r, true
	case <-time.After(d):
		return resp{}, false
	}
}

type lockCase struct {
	name string
	call func() resp
}

// 06 BE 2.9: the project lock belongs to what adds rows or changes an order (add an agenda or a song, apply a template,
// both reorders, and moving an agenda to another session). Status, delete and plain update are one statement and never
// wait for it, so the event-day screen is not held up by a template or a reorder. Each case runs alone against a lock
// the test holds, so one cannot change what another one needs.
func TestWritersTakeTheProjectLockAndTheLiveScreenDoesNot(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sess := sessionIDs(e, tok, id)
	a := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:00", "07:10", "A", nil))["id"].(string)
	b := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:11", "07:20", "B", nil))["id"].(string)
	c := addAgenda(e, tok, id, agendaBody(sess["akad"], "07:21", "07:30", "C", nil))["id"].(string)
	s1 := addSong(e, tok, id, songBody("Satu", "X", nil))["id"].(string)
	s2 := addSong(e, tok, id, songBody("Dua", "X", nil))["id"].(string)
	s3 := addSong(e, tok, id, songBody("Tiga", "X", nil))["id"].(string)

	waits := []lockCase{
		{"reorder agendas", func() resp {
			return e.do(tok, "POST", base(id)+"/rundowns/reorder", map[string]any{"session_id": sess["akad"], "ids": []string{c, b, a}})
		}},
		{"reorder songs", func() resp {
			return e.do(tok, "POST", base(id)+"/playlists/reorder", map[string]any{"ids": []string{s3, s2, s1}})
		}},
		{"add an agenda", func() resp {
			return e.do(tok, "POST", base(id)+"/rundowns", agendaBody(sess["resepsi"], "08:00", "", "D", nil))
		}},
		{"add a song", func() resp { return e.do(tok, "POST", base(id)+"/playlists", songBody("Empat", "X", nil)) }},
		{"move an agenda to another session", func() resp {
			return e.do(tok, "PUT", base(id)+"/rundowns/"+a, agendaBody(sess["lamaran"], "07:00", "07:10", "A", nil))
		}},
		{"apply the agenda template", func() resp { return applyAgendaTemplate(e, tok, id, nil) }},
		{"apply the song template", func() resp { return applyPlaylistTemplate(e, tok, id, nil) }},
	}
	free := []lockCase{
		{"read the live view", func() resp { return e.do(tok, "GET", base(id)+"/rundowns/summary", nil) }},
		{"read the list", func() resp { return e.do(tok, "GET", base(id)+"/rundowns", nil) }},
		{"set a status", func() resp { return setStatus(e, tok, id, b, "in_progress") }},
		{"update in place", func() resp {
			return e.do(tok, "PUT", base(id)+"/rundowns/"+b, agendaBody(sess["akad"], "07:11", "07:20", "B2", nil))
		}},
		{"update a song", func() resp { return e.do(tok, "PUT", base(id)+"/playlists/"+s1, songBody("Satu", "Y", nil)) }},
		{"delete a song", func() resp { return e.do(tok, "DELETE", base(id)+"/playlists/"+s2, nil) }},
		{"delete an agenda", func() resp { return e.do(tok, "DELETE", base(id)+"/rundowns/"+c, nil) }},
	}

	// the cases that must wait, one at a time (each one changes what the next needs, so the order is the one above)
	for _, tc := range waits {
		release := holdProjectLock(e, id)
		ch := make(chan resp, 1)
		go func() { ch <- tc.call() }()
		select {
		case r := <-ch:
			release()
			t.Errorf("%s did not wait for the project lock: %d %s", tc.name, r.status, r.body)

			continue
		case <-time.After(500 * time.Millisecond):
		}
		release()
		select {
		case r := <-ch:
			if r.status >= 300 {
				t.Errorf("%s after the lock was released: %d %s", tc.name, r.status, r.body)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("%s never finished", tc.name)
		}
	}

	// the cases that must not: with the lock held they still answer at once
	release := holdProjectLock(e, id)
	for _, tc := range free {
		if r, ok := answersWithin(5*time.Second, tc.call); !ok || r.status >= 300 {
			t.Errorf("%s: waited for the project lock or failed (answered %v, %d %s)", tc.name, ok, r.status, r.body)
		}
	}
	release()
}
