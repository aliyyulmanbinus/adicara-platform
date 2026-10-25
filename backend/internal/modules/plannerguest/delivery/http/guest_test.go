package http_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	guesthttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/delivery/http"
	guestusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/usecase"
)

func TestGuestLifecycle(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	g := addGuest(e, tok, id, map[string]any{"name": "  Budi   Santoso ", "pax_count": 3})
	gid := g["id"].(string)
	// Defaults: no side, the fallback category, the session "all", nobody has answered or visited.
	if g["name"] != "Budi Santoso" || g["category"] != "lainnya" || g["sub_event_scope"] != "all" || num(g["pax_count"]) != 3 ||
		g["rsvp_status"] != "unconfirmed" || num(g["pax_actual"]) != 0 || g["invitation_sent_status"] != "unsent" ||
		g["is_checked_in"] != false || g["souvenir_received"] != false || len(g["qr_code_token"].(string)) != 32 {
		t.Fatalf("created = %v", g)
	}
	for _, k := range []string{"guest_side", "table_number", "phone_number", "formatted_wa_number", "personalized_url", "invitation_sent_at", "rsvp_submitted_at", "checked_in_at", "notes"} {
		if _, has := g[k]; has {
			t.Errorf("%s is sent for an empty value: %v", k, g[k])
		}
	}
	if got := get(e, tok, guests(id)+"/"+gid); jsonOf(got) != jsonOf(g) {
		t.Errorf("GET differs from POST:\n%v\n%v", got, g)
	}

	// PUT replaces the editable fields: what is not sent is cleared.
	put := mustStatus(t, e.do(tok, "PUT", guests(id)+"/"+gid, map[string]any{
		"name": "Budi S.", "guest_side": "wanita", "category": "Teman Orang Tua", "sub_event_scope": "resepsi", "table_number": " T-01 ",
		"pax_count": 2, "phone_number": "0812 3456 7890", "notes": "Alergi kacang\nKursi roda",
	}), 200).data()
	if put["name"] != "Budi S." || put["guest_side"] != "wanita" || put["category"] != "teman_ortu" || put["sub_event_scope"] != "resepsi" ||
		put["table_number"] != "T-01" || put["phone_number"] != "0812 3456 7890" || put["formatted_wa_number"] != "6281234567890" || put["notes"] != "Alergi kacang\nKursi roda" {
		t.Errorf("updated = %v", put)
	}
	cleared := mustStatus(t, e.do(tok, "PUT", guests(id)+"/"+gid, map[string]any{"name": "Budi S.", "pax_count": 1}), 200).data()
	for _, k := range []string{"guest_side", "table_number", "phone_number", "formatted_wa_number", "notes"} {
		if _, has := cleared[k]; has {
			t.Errorf("PUT did not clear %s: %v", k, cleared[k])
		}
	}
	if cleared["category"] != "lainnya" || cleared["sub_event_scope"] != "all" {
		t.Errorf("PUT without category/scope = %v", cleared)
	}

	// Invitation status: sent stamps the time, unsent clears it.
	sent := mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/sent-status", map[string]any{"sent_status": "sent"}), 200).data()
	if sent["invitation_sent_status"] != "sent" || sent["invitation_sent_at"] == nil {
		t.Errorf("sent = %v", sent)
	}
	unsent := mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/sent-status", map[string]any{"sent_status": "unsent"}), 200).data()
	if unsent["invitation_sent_status"] != "unsent" || unsent["invitation_sent_at"] != nil {
		t.Errorf("unsent = %v", unsent)
	}

	mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+gid, nil), 204)
	wantError(t, e.do(tok, "GET", guests(id)+"/"+gid, nil), 404, "not_found", "guest not found")
	wantError(t, e.do(tok, "DELETE", guests(id)+"/"+gid, nil), 404, "not_found", "guest not found")
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 0 {
		t.Errorf("%d guests left", n)
	}
}

func TestGuestValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	cases := []struct {
		name         string
		over         map[string]any
		status       int
		code, reason string
	}{
		{"empty name", map[string]any{"name": "   "}, 400, "invalid_input", "name: required"},
		{"name too long", map[string]any{"name": strings.Repeat("a", 256)}, 400, "invalid_input", "name: must be at most 255 characters"},
		{"name with a control character", map[string]any{"name": "Bud\u0007i"}, 400, "invalid_input", "name: must not contain control characters"},
		{"unknown side", map[string]any{"guest_side": "laki-laki"}, 400, "invalid_input", "guest_side: must be a registry side key"},
		{"category too long", map[string]any{"category": strings.Repeat("k", 61)}, 400, "invalid_input", "category: must be at most 60 characters"},
		{"unknown session", map[string]any{"sub_event_scope": "sesi_tak_ada"}, 400, "invalid_scope", "sub_event_scope: unknown session"},
		{"session with a bad key", map[string]any{"sub_event_scope": "Resepsi!"}, 400, "invalid_scope", ""},
		{"table too long", map[string]any{"table_number": strings.Repeat("m", 31)}, 400, "invalid_input", "table_number: must be at most 30 characters"},
		{"pax zero", map[string]any{"pax_count": 0}, 400, "invalid_input", "pax_count: must be between 1 and 20"},
		{"pax 21", map[string]any{"pax_count": 21}, 400, "invalid_input", "pax_count: must be between 1 and 20"},
		{"phone not Indonesian", map[string]any{"phone_number": "+65 9123 4567"}, 400, "invalid_phone", "phone_number: must be an Indonesian phone number (08…, +62…)"},
		{"phone without a prefix", map[string]any{"phone_number": "81200000101"}, 400, "invalid_phone", ""},
		{"phone too long", map[string]any{"phone_number": strings.Repeat("1", 41)}, 400, "invalid_input", "phone_number: must be at most 40 characters"},
		{"notes too long", map[string]any{"notes": strings.Repeat("n", 2001)}, 400, "invalid_input", "notes: must be at most 2000 characters"},
		{"notes with a control character", map[string]any{"notes": "a\u0000b"}, 400, "invalid_input", "notes: must not contain control characters"},
	}
	for _, c := range cases {
		wantError(t, e.do(tok, "POST", guests(id), guestBody("Tamu uji", c.over)), c.status, c.code, c.reason)
	}
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 0 {
		t.Fatalf("a refused request wrote %d guests", n)
	}

	// The same rules on PUT, and a guest of the project is needed first.
	g := addGuest(e, tok, id, guestBody("Tamu uji", nil))
	wantError(t, e.do(tok, "PUT", guests(id)+"/"+g["id"].(string), guestBody("Tamu uji", map[string]any{"pax_count": 99})), 400, "invalid_input", "pax_count: must be between 1 and 20")

	// What is accepted at the border.
	for name, over := range map[string]map[string]any{
		"255-letter name": {"name": strings.Repeat("n", 255)}, "60-letter category": {"category": strings.Repeat("k", 60)},
		"pax 1": {"pax_count": 1}, "pax 20": {"pax_count": 20}, "30-letter table": {"table_number": strings.Repeat("m", 30)},
		"+62 phone": {"phone_number": "+62 812-0000-0001"}, "2000-letter notes": {"notes": strings.Repeat("n", 2000)},
		"blank phone": {"phone_number": "  "}, "null side": {"guest_side": nil},
	} {
		if r := e.do(tok, "POST", guests(id), guestBody("Batas", over)); r.status != 201 {
			t.Errorf("%s: %d %s", name, r.status, r.body)
		}
	}

	// A free-text category stays as typed (spaces folded); a registry label is stored as its key.
	if c := addGuest(e, tok, id, guestBody("Kustom", map[string]any{"category": "  Kolega   Kantor "}))["category"]; c != "Kolega Kantor" {
		t.Errorf("custom category = %v", c)
	}
	if c := addGuest(e, tok, id, guestBody("Label", map[string]any{"category": "teman orang tua"}))["category"]; c != "teman_ortu" {
		t.Errorf("category by label = %v", c)
	}
	if s := addGuest(e, tok, id, guestBody("Sisi", map[string]any{"guest_side": "Pihak Wanita"}))["guest_side"]; s != "wanita" {
		t.Errorf("side by label = %v", s)
	}
}

// PUT never touches what other endpoints own (05 BE 3.1): the answer, the invitation status, the visit, the souvenir, the
// ticket and the place in the list.
func TestPutKeepsRSVPCheckinAndToken(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	before := guestNamed(t, e, tok, id, "Raditya Pratama & Pasangan")
	gid := before["id"].(string)
	mustStatus(t, checkin(e, tok, id, gid, 2, true), 200)
	before = guestNamed(t, e, tok, id, "Raditya Pratama & Pasangan")

	mustStatus(t, e.do(tok, "PUT", guests(id)+"/"+gid, guestBody("Raditya P.", map[string]any{"pax_count": 5})), 200)
	after := guestNamed(t, e, tok, id, "Raditya P.")

	for _, k := range []string{"id", "rsvp_status", "pax_actual", "rsvp_submitted_at", "invitation_sent_status", "invitation_sent_at", "qr_code_token", "is_checked_in", "checked_in_at", "souvenir_received"} {
		if jsonOf(before[k]) != jsonOf(after[k]) {
			t.Errorf("PUT changed %s: %v -> %v", k, before[k], after[k])
		}
	}
	if num(after["pax_count"]) != 5 {
		t.Errorf("pax_count = %v", after["pax_count"])
	}
	// The order of the list is the order of creation: the edited guest stays second.
	if names := list(guestList(e, tok, id, "")["guests"]); names[1]["id"] != gid {
		t.Errorf("the edited guest moved: %v", names[1]["name"])
	}
}

func TestGuestLimitOf5000(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	insertGuests(e, id, 4999)

	g := addGuest(e, tok, id, guestBody("Tamu ke-5000", nil)) // the 5000th is the last one allowed
	wantError(t, e.do(tok, "POST", guests(id), guestBody("Tamu ke-5001", nil)), 409, "guest_limit_reached", "")
	wantError(t, e.do(tok, "POST", guests(id)+"/walk-in", map[string]any{"name": "Datang", "pax_actual": 1}), 409, "guest_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 5000 {
		t.Errorf("%d guests, want 5000", n)
	}
	// Another project has its own count, and a delete makes room again.
	other := createProject(e, tok, nil)
	addGuest(e, tok, other, guestBody("Satu", nil))
	mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+g["id"].(string), nil), 204)
	addGuest(e, tok, id, guestBody("Pengganti", nil))
}

// Two requests at the same time must not both take the last place: the count and the insert are under the project lock.
func TestConcurrentCreatesRespectTheLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	insertGuests(e, id, 4995)

	statuses := concurrently(12, func(i int) int {
		return e.do(tok, "POST", guests(id), guestBody(fmt.Sprintf("Serentak %d", i), nil)).status
	})
	created, refused := 0, 0
	for _, s := range statuses {
		switch s {
		case 201:
			created++
		case 409:
			refused++
		default:
			t.Errorf("status %d", s)
		}
	}
	if created != 5 || refused != 7 {
		t.Errorf("%d created and %d refused, want 5 and 7", created, refused)
	}
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, id); n != 5000 {
		t.Errorf("%d guests, want 5000", n)
	}
	if n := count(e, `SELECT count(DISTINCT sort_order) FROM planner_guests WHERE project_id=$1 AND name LIKE 'Serentak%'`, id); n != 5 {
		t.Errorf("%d distinct sort_order among the 5 new guests", n)
	}
}

func TestRSVPStampsAndClears(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	gid := addGuest(e, tok, id, guestBody("Penjawab", map[string]any{"pax_count": 4}))["id"].(string)
	rsvp := func(status string, pax int) map[string]any {
		return mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": status, "pax_actual": pax}), 200).data()
	}

	// attending without a number follows the invitation (0 = pax_count); the answer is stamped.
	a := rsvp("attending", 0)
	if a["rsvp_status"] != "attending" || num(a["pax_actual"]) != 0 || a["rsvp_submitted_at"] == nil {
		t.Errorf("attending = %v", a)
	}
	stamp := a["rsvp_submitted_at"]
	if b := rsvp("attending", 3); num(b["pax_actual"]) != 3 || b["rsvp_submitted_at"] == stamp {
		t.Errorf("attending 3 = %v (the stamp must be renewed when the answer changes)", b)
	}
	stamp = rsvp("attending", 3)["rsvp_submitted_at"]
	if again := rsvp("attending", 3); again["rsvp_submitted_at"] != stamp {
		t.Errorf("the same answer moved the stamp: %v -> %v", stamp, again["rsvp_submitted_at"])
	}
	// Any other status stores 0, whatever number came with it, and the sample counts it as such.
	if d := rsvp("declined", 9); d["rsvp_status"] != "declined" || num(d["pax_actual"]) != 0 || d["rsvp_submitted_at"] == nil {
		t.Errorf("declined = %v", d)
	}
	if m := rsvp("maybe", 2); num(m["pax_actual"]) != 0 || m["rsvp_submitted_at"] == nil {
		t.Errorf("maybe = %v", m)
	}
	// Back to unconfirmed: the stamp is cleared.
	if u := rsvp("unconfirmed", 0); u["rsvp_status"] != "unconfirmed" || u["rsvp_submitted_at"] != nil {
		t.Errorf("unconfirmed = %v", u)
	}

	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": "hadir"}), 400, "invalid_input", "rsvp_status: must be one of unconfirmed, attending, declined, maybe")
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": "attending", "pax_actual": 21}), 400, "invalid_input", "pax_actual: must be between 0 and 20")
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": "attending", "pax_actual": -1}), 400, "invalid_input", "pax_actual: must be between 0 and 20")
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/sent-status", map[string]any{"sent_status": "delivered"}), 400, "invalid_input", "sent_status: must be one of unsent, sent")
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/sent-status", map[string]any{}), 400, "invalid_input", "sent_status: must be one of unsent, sent")
}

// A guest who has checked in is at the door: pax_actual is the head-count there, so the answer can no longer be changed.
func TestRSVPLockedAfterCheckin(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	gid := addGuest(e, tok, id, guestBody("Di pintu", nil))["id"].(string)
	mustStatus(t, checkin(e, tok, id, gid, 2, false), 200)

	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": "declined"}), 409, "already_checked_in", "")
	if g := get(e, tok, guests(id)+"/"+gid); g["rsvp_status"] != "unconfirmed" || num(g["pax_actual"]) != 2 {
		t.Errorf("the refused change altered the guest: %v", g)
	}
	mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+gid+"/checkin", nil), 200)
	mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/rsvp", map[string]any{"rsvp_status": "declined"}), 200)
	// An unknown guest is a 404, not a 409.
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+uuid.NewString()+"/rsvp", map[string]any{"rsvp_status": "declined"}), 404, "not_found", "guest not found")
}

// The public RSVP of the invitation module: found by invitation and ticket token (05 BE 5.1).
func TestApplyRSVPByTokenIsIdempotent(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	inv := uuid.New()
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, inv, id) // as if the invitation module had linked it
	g := addGuest(e, tok, id, guestBody("Tamu undangan", map[string]any{"pax_count": 3}))
	token := g["qr_code_token"].(string)
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE project_id=$1 AND invitation_id=$2`, id, inv); n != 1 {
		t.Fatalf("a guest made through the planner must carry the invitation of the project (%d)", n)
	}
	w := e.guest.RSVPWriter()
	ctx := context.Background()

	if err := w.ApplyRSVP(ctx, inv, token, pd.RSVPAttending, 2); err != nil {
		t.Fatal(err)
	}
	got := get(e, tok, guests(id)+"/"+g["id"].(string))
	if got["rsvp_status"] != "attending" || num(got["pax_actual"]) != 2 || got["rsvp_submitted_at"] == nil {
		t.Fatalf("after ApplyRSVP = %v", got)
	}
	// The same answer again changes nothing, the stamp included.
	if err := w.ApplyRSVP(ctx, inv, token, pd.RSVPAttending, 2); err != nil {
		t.Fatal(err)
	}
	if again := get(e, tok, guests(id)+"/"+g["id"].(string)); jsonOf(again) != jsonOf(got) {
		t.Errorf("a repeated answer changed the guest:\n%v\n%v", got, again)
	}
	// A new answer replaces it; declined stores no head-count.
	if err := w.ApplyRSVP(ctx, inv, token, pd.RSVPDeclined, 7); err != nil {
		t.Fatal(err)
	}
	if d := get(e, tok, guests(id)+"/"+g["id"].(string)); d["rsvp_status"] != "declined" || num(d["pax_actual"]) != 0 {
		t.Errorf("declined = %v", d)
	}

	// Out of range is a 400 like everywhere, and writes nothing.
	var ae *apierr.Error
	if err := w.ApplyRSVP(ctx, inv, token, "hadir", 1); !errors.As(err, &ae) || ae.Status != 400 {
		t.Errorf("a bad status: %v", err)
	}
	if err := w.ApplyRSVP(ctx, inv, token, pd.RSVPAttending, 21); !errors.As(err, &ae) || ae.Status != 400 {
		t.Errorf("a bad pax: %v", err)
	}

	// A guest who has checked in is left alone and the call succeeds.
	mustStatus(t, checkin(e, tok, id, g["id"].(string), 4, false), 200)
	if err := w.ApplyRSVP(ctx, inv, token, pd.RSVPAttending, 1); err != nil {
		t.Fatalf("the form of a guest at the door must not fail: %v", err)
	}
	if c := get(e, tok, guests(id)+"/"+g["id"].(string)); c["rsvp_status"] != "declined" || num(c["pax_actual"]) != 4 {
		t.Errorf("ApplyRSVP changed a guest who checked in: %v", c)
	}
}

func TestApplyRSVPNeverCrossesInvitations(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	a, b := createProject(e, tok, nil), createProject(e, tok, nil)
	invA, invB := uuid.New(), uuid.New()
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, invA, a)
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, invB, b)
	ga := addGuest(e, tok, a, guestBody("Tamu A", nil))
	gb := addGuest(e, tok, b, guestBody("Tamu B", nil))
	standalone := addGuest(e, tok, createProject(e, tok, nil), guestBody("Tamu mandiri", nil))
	w := e.guest.RSVPWriter()
	ctx := context.Background()

	for name, call := range map[string]error{
		"another invitation's token":  w.ApplyRSVP(ctx, invA, gb["qr_code_token"].(string), pd.RSVPAttending, 1),
		"the right token, wrong inv.": w.ApplyRSVP(ctx, invB, ga["qr_code_token"].(string), pd.RSVPAttending, 1),
		"an unknown token":            w.ApplyRSVP(ctx, invA, strings.Repeat("0", 32), pd.RSVPAttending, 1),
		"an empty token":              w.ApplyRSVP(ctx, invA, "", pd.RSVPAttending, 1),
		"a standalone guest":          w.ApplyRSVP(ctx, uuid.New(), standalone["qr_code_token"].(string), pd.RSVPAttending, 1),
	} {
		if !errors.Is(call, pd.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, call)
		}
	}
	for _, c := range []struct{ project, guest string }{{a, ga["id"].(string)}, {b, gb["id"].(string)}} {
		if g := get(e, tok, guests(c.project)+"/"+c.guest); g["rsvp_status"] != "unconfirmed" {
			t.Errorf("a refused ApplyRSVP changed %v", g["name"])
		}
	}
}

func TestListFilters(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	addGuest(e, tok, id, guestBody("Tanpa pihak", map[string]any{"guest_side": nil, "category": "Rekan Bisnis"}))

	total := func(query string) int64 { return num(guestList(e, tok, id, query)["total"]) }
	for query, want := range map[string]int64{
		"": 21, "side=pria": 10, "side=wanita": 10, "side=none": 1, "category=vip": 2, "category=Rekan Bisnis": 1, "rsvp=declined": 2,
		"rsvp=attending": 12, "rsvp=maybe": 2, "rsvp=unconfirmed": 5, "sent=sent": 14, "sent=unsent": 7, "scope=all": 4, "scope=resepsi": 17,
		"side=pria&category=teman": 3, "side=wanita&rsvp=attending&sent=sent": 3, "scope=resepsi&category=keluarga": 1,
		"side=pria&side=wanita": 10, // the first value wins, like every query string of the API
	} {
		if got := total(strings.ReplaceAll(query, " ", "%20")); got != want {
			t.Errorf("?%s: %d guests, want %d", query, got, want)
		}
	}

	// Pages: the total is that of the filter; the overview ignores the page and the filter.
	page := guestList(e, tok, id, "limit=5&offset=15")
	if len(list(page["guests"])) != 5 || num(page["total"]) != 21 || num(page["limit"]) != 5 || num(page["offset"]) != 15 {
		t.Errorf("page = %v guests, total %v", len(list(page["guests"])), page["total"])
	}
	if tail := guestList(e, tok, id, "limit=5&offset=20"); len(list(tail["guests"])) != 1 {
		t.Errorf("the last page holds %d guests", len(list(tail["guests"])))
	}
	if beyond := guestList(e, tok, id, "offset=500"); len(list(beyond["guests"])) != 0 || num(beyond["total"]) != 21 {
		t.Errorf("beyond the end = %v", beyond)
	}
	// The order is the order of creation: the 21st guest is last.
	if last := list(guestList(e, tok, id, "limit=500")["guests"]); last[20]["name"] != "Tanpa pihak" || last[0]["name"] != "Bpk. Prof. Bambang Soedjarwo" {
		t.Errorf("order = %v ... %v", last[0]["name"], last[20]["name"])
	}

	for query, msg := range map[string]string{
		"limit=0": "limit: must be between 1 and 500", "limit=501": "limit: must be between 1 and 500", "limit=x": "limit: must be between 1 and 500",
		"offset=-1": "offset: must be at least 0", "offset=x": "offset: must be at least 0", "rsvp=hadir": "rsvp: must be one of unconfirmed, attending, declined, maybe",
		"sent=delivered": "sent: must be one of sent, unsent", "side=Pria!": "side: must be a side key", "scope=Resepsi": "scope: must be a session scope_key",
		"category=" + strings.Repeat("k", 61): "category: must be at most 60 characters", "search=" + strings.Repeat("s", 101): "search: must be at most 100 characters",
	} {
		wantError(t, e.do(tok, "GET", guests(id)+"?"+query, nil), 400, "invalid_input", msg)
	}
}

// Search is a substring match, not a pattern: `%` and `_` mean themselves (05 BE 2.7).
func TestSearchHasNoWildcards(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	addGuest(e, tok, id, guestBody("Diskon 50% Hemat_Banget", map[string]any{"table_number": "Z-9", "phone_number": ""}))

	names := func(query string) []string {
		var out []string
		for _, g := range list(guestList(e, tok, id, "search="+query)["guests"]) {
			out = append(out, g["name"].(string))
		}

		return out
	}
	for query, want := range map[string]int{
		"%25": 1, "_": 1, "50%25": 1, "%25%25": 0, "__": 0, "Diskon": 1, "bambang": 1, "BAMBANG": 1, "%20bambang%20": 1, // trimmed
		"vip-0": 4, "T-1": 4, // tables: T-10, T-11, T-12, T-14
		"081200000105": 1, "6281200000105": 1, "0812000001": 20, // as typed, in the 62 form, and a prefix of every number
		"zzz": 0, "": 21,
	} {
		if got := names(query); len(got) != want {
			t.Errorf("search=%q: %d guests %v, want %d", query, len(got), got, want)
		}
	}
	// The filter of the search joins the others, and the overview does not move.
	l := guestList(e, tok, id, "search=vip&rsvp=attending")
	if num(l["total"]) != 4 || num(l["overview"].(map[string]any)["total"]) != 21 {
		t.Errorf("search+filter = total %v, overview %v", l["total"], l["overview"].(map[string]any)["total"])
	}
}

// The overview is the whole project whatever the page shows (05 BE 2.8).
func TestOverviewIgnoresFilters(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	want := jsonOf(guestOverview(e, tok, id))

	for _, query := range []string{"rsvp=declined", "side=none", "search=zzz", "limit=1&offset=3", "category=vip&scope=all"} {
		if got := jsonOf(guestList(e, tok, id, query)["overview"]); got != want {
			t.Errorf("?%s moved the overview:\n%s\n%s", query, got, want)
		}
	}
}

// A personal link needs an invitation module, a public base URL and a project linked to an invitation; without one of them
// the guest simply has none (05 BE 2.1).
func TestPersonalLinksNeedAnInvitation(t *testing.T) {
	inv := uuid.New()
	e := newEnvWith(t, guestusecase.Options{Invitations: fakeInvitations{inv: "sinta-arif"}, PublicBaseURL: "https://adicara.id/"}, guesthttp.Limits{})
	tok := e.login("owner")
	linked, standalone, gone := createProject(e, tok, nil), createProject(e, tok, nil), createProject(e, tok, nil)
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, inv, linked)
	exec(e, `UPDATE planner_projects SET invitation_id=$1 WHERE id=$2`, uuid.New(), gone) // an invitation that no longer exists

	g := addGuest(e, tok, linked, guestBody("Bpk. A & Istri", nil))
	want := "https://adicara.id/i/sinta-arif?g=" + g["qr_code_token"].(string) + "&to=Bpk.+A+%26+Istri"
	if g["personalized_url"] != want {
		t.Errorf("personalized_url = %v, want %v", g["personalized_url"], want)
	}
	// Every endpoint that returns a guest carries it, and so does the list.
	if got := get(e, tok, guests(linked)+"/"+g["id"].(string)); got["personalized_url"] != want {
		t.Errorf("GET = %v", got["personalized_url"])
	}
	if l := list(guestList(e, tok, linked, "")["guests"]); len(l) != 1 || l[0]["personalized_url"] != want {
		t.Errorf("list = %v", l)
	}
	if s := mustStatus(t, e.do(tok, "POST", guests(linked)+"/scan-qr", map[string]any{"qr_code_token": g["qr_code_token"]}), 200).data(); s["personalized_url"] != want {
		t.Errorf("scan = %v", s["personalized_url"])
	}

	for name, pid := range map[string]string{"a standalone project": standalone, "a deleted invitation": gone} {
		other := addGuest(e, tok, pid, guestBody("Tanpa tautan", nil))
		if _, has := other["personalized_url"]; has {
			t.Errorf("%s: the guest has a link: %v", name, other["personalized_url"])
		}
		if l := list(guestList(e, tok, pid, "")["guests"]); len(l) != 1 || l[0]["personalized_url"] != nil {
			t.Errorf("%s: list = %v", name, l)
		}
	}
}
