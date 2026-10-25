package http_test

import (
	"fmt"
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Menu 05 is not a wedding feature. Every event type of the registry goes through the whole flow on the same endpoints with
// nothing but the registry to tell them apart: its sides (or none), its categories, the words of the template, the import,
// the reception desk and the dashboard widget.
func TestEveryEventTypeRunsTheWholeGuestFlow(t *testing.T) {
	e := newEnv(t)
	defer func() { e.t = t }()

	for _, spec := range pd.DefaultRegistry().All() {
		t.Run(string(spec.Key), func(t *testing.T) {
			e.t = t
			tok := e.login("user_" + string(spec.Key))
			id := createProject(e, tok, map[string]any{
				"event_type": string(spec.Key), "title": "Uji " + spec.Label, "event_date": inDays(120), "metadata": metadataFor(spec),
			})
			scope := spec.Sessions[0].ScopeKey

			// GET /event-types carries the registry of this type, as [] and not null when it has no sides.
			var got map[string]any
			for _, et := range list(mustStatus(t, e.do(tok, "GET", "/v1/planner/event-types", nil), 200).json()["data"]) {
				if et["key"] == string(spec.Key) {
					got = et
				}
			}
			if got == nil || len(list(got["guest_sides"])) != len(spec.GuestSides) || len(list(got["guest_categories"])) != len(spec.GuestCategories) {
				t.Fatalf("event type = %v", got)
			}
			if _, isList := got["guest_sides"].([]any); !isList {
				t.Fatalf("guest_sides = %v, want a list", got["guest_sides"])
			}

			// One guest per category (and per side, round and round); a side the type does not have is refused.
			for i, c := range spec.GuestCategories {
				body := guestBody(fmt.Sprintf("Tamu %s %d", c.Label, i), map[string]any{"category": c.Key, "sub_event_scope": scope, "pax_count": 2})
				if len(spec.GuestSides) > 0 {
					body["guest_side"] = spec.GuestSides[i%len(spec.GuestSides)].Key
				} else {
					delete(body, "guest_side")
				}
				g := addGuest(e, tok, id, body)
				if g["category"] != c.Key || g["sub_event_scope"] != scope {
					t.Fatalf("guest = %v", g)
				}
			}
			if len(spec.GuestSides) == 0 {
				wantError(t, e.do(tok, "POST", guests(id), guestBody("Pihak", map[string]any{"guest_side": "pria", "sub_event_scope": scope})), 400, "invalid_input", "guest_side: must be empty for an event type without sides")
			} else {
				wantError(t, e.do(tok, "POST", guests(id), guestBody("Pihak", map[string]any{"guest_side": "pria_x", "sub_event_scope": scope})), 400, "invalid_input", "guest_side: must be a registry side key")
				// Another type's key is not this type's: the wedding's "pria" is nobody's side but the wedding's.
				if spec.Key != pd.EventWedding {
					wantError(t, e.do(tok, "POST", guests(id), guestBody("Pihak", map[string]any{"guest_side": "pria", "sub_event_scope": scope})), 400, "invalid_input", "guest_side: must be a registry side key")
				}
			}

			// The overview: the registry sides and categories, and every breakdown adds up to the total.
			n := int64(len(spec.GuestCategories))
			ov := guestOverview(e, tok, id)
			if num(ov["total"]) != n || num(ov["total_pax"]) != 2*n {
				t.Fatalf("overview = %v", ov)
			}
			sides := list(ov["sides"])
			if len(sides) != len(spec.GuestSides) || len(list(ov["side_categories"])) > len(spec.GuestCategories) || (len(spec.GuestSides) == 0 && len(list(ov["side_categories"])) != 0) {
				t.Errorf("sides = %v, side_categories = %v", sides, ov["side_categories"])
			}
			var bySide, byCat int64
			for i, s := range sides {
				bySide += num(s["invitations"])
				if s["key"] != spec.GuestSides[i].Key || s["label"] != spec.GuestSides[i].Label {
					t.Errorf("side %d = %v", i, s)
				}
				if spec.Key != pd.EventWedding && (s["key"] == "pria" || s["key"] == "wanita") {
					t.Errorf("%s shows a side of the wedding: %v", spec.Key, s)
				}
			}
			for i, c := range list(ov["categories"]) {
				byCat += num(c["invitations"])
				if c["key"] != spec.GuestCategories[i].Key || c["label"] != spec.GuestCategories[i].Label || num(c["invitations"]) != 1 {
					t.Errorf("category %d = %v", i, c)
				}
			}
			r := ov["rsvp"].(map[string]any)
			byRSVP := num(r["attending"]) + num(r["declined"]) + num(r["maybe"]) + num(r["unconfirmed"])
			if (len(spec.GuestSides) > 0 && bySide != n) || byCat != n || byRSVP != n {
				t.Errorf("sums: sides %d, categories %d, rsvp %d, want %d", bySide, byCat, byRSVP, n)
			}

			// The import: header words by the label of the type, the first side and category by label, a free-text category.
			first := spec.GuestCategories[0]
			row := fmt.Sprintf("%s;%s;;%d;0812000%05d", "Impor Satu", first.Label, 2, 1)
			header := "Nama " + spec.Labels.Guest + ";Kategori;No Meja;Pax;No WhatsApp"
			if len(spec.GuestSides) > 0 {
				header = "Nama " + spec.Labels.Guest + ";Pihak;Kategori;No Meja;Pax;No WhatsApp"
				row = fmt.Sprintf("%s;%s;%s;;%d;0812000%05d", "Impor Satu", spec.GuestSides[0].Label, first.Label, 2, 1)
			}
			imp := mustStatus(t, importCSV(e, tok, id, header+"\n"+row+"\nImpor Dua;;;;\n", ""), 200).data()
			if num(imp["created"]) != 2 || num(imp["invalid_rows"]) != 0 {
				t.Fatalf("import = %v", imp)
			}
			one := guestNamed(t, e, tok, id, "Impor Satu")
			if one["category"] != first.Key || (len(spec.GuestSides) > 0 && one["guest_side"] != spec.GuestSides[0].Key) || (len(spec.GuestSides) == 0 && one["guest_side"] != nil) {
				t.Errorf("imported guest = %v", one)
			}

			// The reception desk: check in one, and the figures follow.
			mustStatus(t, checkin(e, tok, id, one["id"].(string), 2, true), 200)
			k := kpi(e, tok, id)
			if num(k["checked_in_guests"]) != 1 || num(k["expected_guests"]) != n+2 || num(k["pending_guests"]) != n+1 || num(k["actual_pax"]) != 2 {
				t.Errorf("kpi = %v", k)
			}
			filters := reception(e, tok, id, "")["filters"].(map[string]any)
			if len(list(filters["categories"])) < len(spec.GuestCategories) {
				t.Errorf("filter categories = %v", filters["categories"])
			}

			// The dashboard widget is the same overview. "Impor Dua" has no side, so a type with sides shows the "not decided"
			// entry after its own.
			wantSides := len(spec.GuestSides)
			if wantSides > 0 {
				wantSides++
			}
			w := get(e, tok, base(id)+"/dashboard")["guests"].(map[string]any)
			if num(w["invitations"]) != n+2 || len(list(w["sides"])) != wantSides {
				t.Errorf("dashboard.guests = %v", w)
			}

			// The files: the template speaks the word of the type, the exports are workbooks.
			tmpl := mustStatus(t, e.do(tok, "GET", guests(id)+"/import-template?format=xlsx", nil), 200)
			if h := rowsOf(t, openSheet(t, tmpl), "Tamu"); len(h) != 1 || h[0][0] != "Nama "+spec.Labels.Guest || strings.Contains(strings.Join(h[0], "|"), "Pihak") != (len(spec.GuestSides) > 0) {
				t.Errorf("template header = %v", h)
			}
			for _, path := range []string{"/export?format=xlsx", "/export-attendance?format=xlsx"} {
				if r := mustStatus(t, e.do(tok, "GET", guests(id)+path, nil), 200); len(r.body) < 1000 || string(r.body[:2]) != "PK" {
					t.Errorf("%s: %d bytes, not a workbook", path, len(r.body))
				}
			}

			// No guest comes with a template: the wizard creates none.
			fresh := createProject(e, tok, map[string]any{"event_type": string(spec.Key), "title": "Kosong", "event_date": inDays(120), "metadata": metadataFor(spec)})
			if num(guestOverview(e, tok, fresh)["total"]) != 0 {
				t.Error("a new project has guests")
			}
		})
	}
}
