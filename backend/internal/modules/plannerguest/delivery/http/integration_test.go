package http_test

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
)

// Deleting a session moves its guests to the session the user names (menu 01, 01 BE 4.6); menu 05 owns the guests, menu 01
// the move. The guests keep everything else.
func TestDeletingASessionMovesItsGuests(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	var akad string
	sessions := mustStatus(t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"]
	for _, s := range list(sessions) {
		if s["scope_key"] == "akad" {
			akad = s["id"].(string)
		}
	}
	// The sample guests are in "resepsi" and "all"; put three of them in "akad" first.
	exec(e, `UPDATE planner_guests SET sub_event_scope='akad' WHERE project_id=$1 AND name IN ('Fajar Nugraha', 'Sdri. Tiara Andini', 'Sdri. Citra Lestari')`, id)
	before := guestOverview(e, tok, id)

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+akad+"?move_to=resepsi", nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1 AND sub_event_scope='akad'`, id); n != 0 {
		t.Errorf("%d guests still point at the deleted session", n)
	}
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1 AND sub_event_scope='resepsi'`, id); n != 16 {
		t.Errorf("%d guests in resepsi, want 16", n)
	}
	if after := guestOverview(e, tok, id); jsonOf(after) != jsonOf(before) {
		t.Errorf("the move changed the overview:\n%v\n%v", before, after)
	}
}

// A guest with no project belongs to the invitation module (05 BE 5): the planner never reads, counts, finds or writes it.
func TestInvitationOnlyGuestsNeverAppearInPlanner(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	inv := uuid.New()
	exec(e, `INSERT INTO planner_guests (id, project_id, invitation_id, name, qr_code_token, pax_count, rsvp_status) VALUES ($1, NULL, $2, 'Tamu undangan saja', 'invonly0123456789invonly01234567', 4, 'attending')`, uuid.New(), inv)
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, inv, id)

	if n := num(guestOverview(e, tok, id)["total"]); n != 20 {
		t.Errorf("total = %d, want 20", n)
	}
	for _, query := range []string{"search=undangan", "rsvp=attending&limit=500"} {
		for _, g := range list(guestList(e, tok, id, query)["guests"]) {
			if g["name"] == "Tamu undangan saja" {
				t.Errorf("?%s returned a guest of the invitation module", query)
			}
		}
	}
	wantError(t, e.do(tok, "POST", guests(id)+"/scan-qr", map[string]any{"qr_code_token": "invonly0123456789invonly01234567"}), 404, "not_found", "guest not found")
	if p := get(e, tok, guests(id)+"/catering-projection"); num(p["confirmed_pax"]) != 27 {
		t.Errorf("confirmed_pax = %v, want 27 (the guest of the invitation module has 4)", p["confirmed_pax"])
	}
	if r := reception(e, tok, id, ""); num(r["kpi"].(map[string]any)["expected_guests"]) != 18 {
		t.Errorf("kpi = %v", r["kpi"])
	}
	// The planner's own writers cannot reach it by id either.
	var other string
	if err := e.pool.QueryRow(t.Context(), `SELECT id FROM planner_guests WHERE project_id IS NULL`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	wantError(t, e.do(tok, "GET", guests(id)+"/"+other, nil), 404, "not_found", "guest not found")
	wantError(t, e.do(tok, "DELETE", guests(id)+"/"+other, nil), 404, "not_found", "guest not found")
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id IS NULL`); n != 1 {
		t.Error("the guest of the invitation module is gone")
	}
}

func TestDeleteProjectCascadesGuests(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 20 {
		t.Fatalf("%d guests", n)
	}
	title := get(e, tok, base(id)+"/dashboard")["project"].(map[string]any)["title"].(string)
	mustStatus(t, e.do(tok, "DELETE", base(id)+"?confirm="+url.QueryEscape(title), nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_guests`); n != 0 {
		t.Errorf("%d guests left after the project was deleted", n)
	}
}
