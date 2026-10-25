package http_test

import (
	"regexp"
	"strings"
	"testing"
)

// The golden test of the contract (chapter 9): the sample project holds the 20 guests of 05 BE 6, and every figure the spec
// quotes comes out of the real stack.
func TestSampleProjectGuestGolden(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	ov := guestOverview(e, tok, id)
	if num(ov["total"]) != 20 || num(ov["total_pax"]) != 43 {
		t.Fatalf("total = %v / %v pax, want 20 / 43", ov["total"], ov["total_pax"])
	}
	if inv := ov["invitation"].(map[string]any); num(inv["sent"]) != 14 || num(inv["unsent"]) != 6 {
		t.Errorf("invitation = %v, want 14 / 6", inv)
	}
	if r := ov["rsvp"].(map[string]any); num(r["attending"]) != 12 || num(r["declined"]) != 2 || num(r["maybe"]) != 2 || num(r["unconfirmed"]) != 4 || num(r["attending_pax"]) != 27 {
		t.Errorf("rsvp = %v, want 12/2/2/4 and 27 pax", r)
	}

	sides := list(ov["sides"])
	if len(sides) != 2 || sides[0]["key"] != "pria" || sides[0]["label"] != "Pihak Pria (CPP)" || num(sides[0]["invitations"]) != 10 || num(sides[0]["pax"]) != 22 ||
		sides[1]["key"] != "wanita" || num(sides[1]["invitations"]) != 10 || num(sides[1]["pax"]) != 21 {
		t.Errorf("sides = %v", sides)
	}

	// The six categories of the PDF, p. 13, in registry order.
	wantCats := []struct {
		key   string
		count int64
	}{{"teman", 6}, {"kolega", 3}, {"keluarga", 4}, {"tetangga", 2}, {"teman_ortu", 3}, {"vip", 2}}
	cats := list(ov["categories"])
	if len(cats) != len(wantCats) {
		t.Fatalf("categories = %v", cats)
	}
	for i, w := range wantCats {
		if cats[i]["key"] != w.key || num(cats[i]["invitations"]) != w.count || cats[i]["custom"] != nil {
			t.Errorf("category %d = %v, want %s %d", i, cats[i], w.key, w.count)
		}
	}

	// ... and per side, like the two tables of the PDF: Pria 3/2/2/1/1/1, Wanita 3/1/2/1/2/1.
	perSide := map[string][]int64{"pria": {3, 2, 2, 1, 1, 1}, "wanita": {3, 1, 2, 1, 2, 1}}
	order := []string{"teman", "kolega", "keluarga", "tetangga", "teman_ortu", "vip"}
	for side, want := range perSide {
		for i, key := range order {
			var got int64
			for _, c := range list(ov["side_categories"]) {
				if c["side"] == side && c["category"] == key {
					got = num(c["invitations"])
				}
			}
			if got != want[i] {
				t.Errorf("%s / %s = %d, want %d", side, key, got, want[i])
			}
		}
	}

	// The list: 20 guests in the order of the table, a token each, WhatsApp in the 62 form, no personal link (standalone).
	l := guestList(e, tok, id, "")
	gs := list(l["guests"])
	if len(gs) != 20 || num(l["total"]) != 20 || num(l["limit"]) != 100 || num(l["offset"]) != 0 {
		t.Fatalf("list = %d guests, total %v limit %v", len(gs), l["total"], l["limit"])
	}
	tokenForm := regexp.MustCompile(`^[0-9a-f]{32}$`)
	seen := map[string]bool{}
	for i, g := range gs {
		tk := g["qr_code_token"].(string)
		if !tokenForm.MatchString(tk) || seen[tk] {
			t.Errorf("guest %d token %q", i+1, tk)
		}
		seen[tk] = true
		if _, has := g["personalized_url"]; has {
			t.Errorf("guest %d has a personal link without an invitation", i+1)
		}
		if g["is_checked_in"] != false || g["souvenir_received"] != false {
			t.Errorf("guest %d is checked in before the event", i+1)
		}
	}
	first := gs[0]
	if first["name"] != "Bpk. Prof. Bambang Soedjarwo" || first["formatted_wa_number"] != "6281200000101" || first["phone_number"] != "081200000101" ||
		first["table_number"] != "VIP-01" || first["rsvp_status"] != "attending" || num(first["pax_actual"]) != 2 || first["invitation_sent_status"] != "sent" {
		t.Errorf("first guest = %v", first)
	}
	last := gs[19]
	if last["name"] != "Bpk. Dedi Sukamto (Kolega Ortu)" || last["rsvp_status"] != "declined" || num(last["pax_actual"]) != 0 || last["invitation_sent_status"] != "unsent" {
		t.Errorf("last guest = %v", last)
	}
	if _, has := last["invitation_sent_at"]; has {
		t.Errorf("an unsent invitation has a sent time: %v", last)
	}

	// The projection: 27 confirmed, 12 undecided, 4 declined; 27 + 25% = 34 portions.
	p := get(e, tok, guests(id)+"/catering-projection")
	if num(p["confirmed_pax"]) != 27 || num(p["potential_pax"]) != 12 || num(p["declined_pax"]) != 4 || num(p["recommended_portions"]) != 34 ||
		num(p["buffer_percent"]) != 25 || num(p["buffer_pax"]) != 0 || num(p["attending_guests"]) != 12 || num(p["declined_guests"]) != 2 || num(p["maybe_guests"]) != 2 || num(p["pending_guests"]) != 4 {
		t.Errorf("projection = %v", p)
	}

	// The dashboard widget is the same overview, reduced (menu 01): 20 invitations, 43 pax, 14 sent.
	w := get(e, tok, base(id)+"/dashboard")["guests"].(map[string]any)
	if num(w["invitations"]) != 20 || num(w["total_pax"]) != 43 || num(w["sent"]) != 14 || w["rsvp"].(map[string]any)["attending_pax"] != float64(27) {
		t.Errorf("dashboard.guests = %v", w)
	}
	ws := list(w["sides"])
	if len(ws) != 2 || ws[0]["label"] != "Pihak Pria (CPP)" || num(ws[0]["pax"]) != 22 {
		t.Errorf("dashboard.guests.sides = %v", ws)
	}

	// The Hub card (menu 01) and the usage of the sessions: pax of all guests, and the guests of each session.
	cards := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects", nil), 200)
	if !strings.Contains(string(cards.body), `"guest_pax_count":43`) {
		t.Errorf("the Hub card does not say 43 pax: %s", cards.body)
	}
	usage := map[string]int64{}
	sessions := mustStatus(t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"]
	for _, s := range list(sessions) {
		usage[s["scope_key"].(string)] = num(s["usage"].(map[string]any)["guests"])
	}
	if usage["resepsi"] != 16 || usage["all"] != 4 || usage["akad"] != 0 {
		t.Errorf("session usage = %v, want resepsi 16, all 4, akad 0", usage)
	}
}
