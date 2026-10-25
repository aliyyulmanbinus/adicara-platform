package http_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// arrivals are the guests of the sample table who walk in first in the illustration of the reception wireframe
// (05 BE 6): numbers 1, 2, 6, 3, 11, 4, 7, 9, 10, 14, 19, in that order.
var arrivals = []string{
	"Bpk. Prof. Bambang Soedjarwo", "Raditya Pratama & Pasangan", "Bpk. Dr. Ir. Gunawan (Sahabat Ayah)", "Bpk. Hendro & Keluarga",
	"Ibu Dra. Hj. Ratna Juwita", "Satria Wibawa (Kolega Kantor)", "Bayu Anggoro & Partner", "Fajar Nugraha", "Bpk. Rahmat Hidayat (Kolega)",
	"Sdri. Melisa Putri (Kolega Kantor)", "Sdri. Citra Lestari",
}

// morning checks the illustration in: 11 guests with the pax they were invited with, all given a souvenir.
func morning(t *testing.T, e *env, tok, id string) {
	t.Helper()
	for _, name := range arrivals {
		g := guestNamed(t, e, tok, id, name)
		mustStatus(t, checkin(e, tok, id, g["id"].(string), int(num(g["pax_count"])), true), 200)
	}
}

func kpi(e *env, tok, id string) map[string]any {
	return reception(e, tok, id, "limit=1")["kpi"].(map[string]any)
}

func TestScanFindsOnlyThisProjectsTickets(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	a, b := createProject(e, tok, nil), createProject(e, tok, nil)
	ga := addGuest(e, tok, a, guestBody("Tamu A", nil))
	gb := addGuest(e, tok, b, guestBody("Tamu B", nil))
	scan := func(project, token string) resp {
		return e.do(tok, "POST", guests(project)+"/scan-qr", map[string]any{"qr_code_token": token})
	}

	if g := mustStatus(t, scan(a, ga["qr_code_token"].(string)), 200).data(); g["id"] != ga["id"] {
		t.Errorf("scan = %v", g)
	}
	// Another project's ticket, an unknown one, a malformed one: all the same answer.
	for name, token := range map[string]string{
		"a ticket of another project": gb["qr_code_token"].(string), "an unknown ticket": strings.Repeat("a", 32), "an empty token": "",
		"blanks": "   ", "a long string": strings.Repeat("a", 65), "SQL": "' OR 1=1 --",
	} {
		if r := scan(a, token); r.status != 404 || r.code() != "not_found" || r.message() != "guest not found" {
			t.Errorf("%s: %d %s", name, r.status, r.body)
		}
	}
	wantError(t, e.do(tok, "POST", guests(a)+"/scan-qr", map[string]any{}), 404, "not_found", "guest not found")

	// A guest who has checked in is still found: the screen shows the time.
	mustStatus(t, checkin(e, tok, a, ga["id"].(string), 2, false), 200)
	if g := mustStatus(t, scan(a, " "+ga["qr_code_token"].(string)+" "), 200).data(); g["is_checked_in"] != true || g["checked_in_at"] == nil {
		t.Errorf("scan after check-in = %v", g)
	}
}

func TestCheckinRules(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	declined := guestNamed(t, e, tok, id, "Bpk. Joko Susilo (Ketua RT 04)")
	gid := declined["id"].(string)

	for _, pax := range []int{0, 21, -1} {
		wantError(t, checkin(e, tok, id, gid, pax, false), 400, "invalid_input", "pax_actual: must be between 1 and 20")
	}
	wantError(t, e.do(tok, "POST", guests(id)+"/"+gid+"/checkin", `{"pax_actual": "2"}`), 400, "invalid_json", "")
	wantError(t, checkin(e, tok, id, uuid.NewString(), 1, false), 404, "not_found", "guest not found")
	wantError(t, e.do(tok, "POST", guests(id)+"/nope/checkin", map[string]any{"pax_actual": 1}), 404, "not_found", "guest not found")

	// A guest who declined and came: checked in, with the head-count at the door, the answer untouched.
	got := mustStatus(t, checkin(e, tok, id, gid, 3, true), 200).data()
	if got["is_checked_in"] != true || got["checked_in_at"] == nil || num(got["pax_actual"]) != 3 || got["souvenir_received"] != true || got["rsvp_status"] != "declined" {
		t.Errorf("checked in = %v", got)
	}
	// The second tap loses; the client reads the guest again to show the time.
	wantError(t, checkin(e, tok, id, gid, 1, false), 409, "already_checked_in", "")
	if again := get(e, tok, guests(id)+"/"+gid); again["checked_in_at"] != got["checked_in_at"] || num(again["pax_actual"]) != 3 {
		t.Errorf("the second check-in changed the first: %v", again)
	}
}

// Two taps on the same guest at the same moment: one wins, the others see 409, and the data is that of the winner.
func TestConcurrentCheckinOneWins(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	gid := addGuest(e, tok, id, guestBody("Rebutan", nil))["id"].(string)

	statuses := concurrently(8, func(i int) int { return checkin(e, tok, id, gid, i+1, false).status })
	won, lost := 0, 0
	for _, s := range statuses {
		switch s {
		case 200:
			won++
		case 409:
			lost++
		default:
			t.Errorf("status %d", s)
		}
	}
	if won != 1 || lost != 7 {
		t.Errorf("%d won and %d lost, want 1 and 7", won, lost)
	}
	if n := count(e, `SELECT count(*) FROM planner_guests WHERE id=$1 AND is_checked_in AND checked_in_at IS NOT NULL AND pax_actual BETWEEN 1 AND 8`, gid); n != 1 {
		t.Error("the guest was not checked in by exactly one request")
	}
}

func TestCancelCheckin(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	g := guestNamed(t, e, tok, id, "Raditya Pratama & Pasangan")
	gid := g["id"].(string)

	mustStatus(t, checkin(e, tok, id, gid, 4, true), 200)
	cancelled := mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+gid+"/checkin", nil), 200).data()
	if cancelled["is_checked_in"] != false || cancelled["souvenir_received"] != false || cancelled["checked_in_at"] != nil {
		t.Errorf("cancelled = %v", cancelled)
	}
	if num(cancelled["pax_actual"]) != 4 { // the head-count at the door is not given back to the RSVP
		t.Errorf("pax_actual = %v", cancelled["pax_actual"])
	}
	// It can be done again: the guest can be checked in once more.
	if again := mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+gid+"/checkin", nil), 200).data(); jsonOf(again) != jsonOf(cancelled) {
		t.Errorf("a second cancel changed the guest:\n%v\n%v", again, cancelled)
	}
	mustStatus(t, checkin(e, tok, id, gid, 2, false), 200)

	// A guest who never checked in is returned as he is, souvenir mark included.
	other := guestNamed(t, e, tok, id, "Fajar Nugraha")["id"].(string)
	mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+other+"/souvenir", map[string]any{"souvenir_received": true}), 200)
	if r := mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+other+"/checkin", nil), 200).data(); r["souvenir_received"] != true || r["is_checked_in"] != false {
		t.Errorf("cancel on a guest who has not checked in = %v", r)
	}
	wantError(t, e.do(tok, "DELETE", guests(id)+"/"+uuid.NewString()+"/checkin", nil), 404, "not_found", "guest not found")
}

func TestSouvenirMark(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	gid := guestNamed(t, e, tok, id, "Anisa Rahmawati (Sahabat SMA)")["id"].(string)

	on := mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/souvenir", map[string]any{"souvenir_received": true}), 200).data()
	if on["souvenir_received"] != true || on["is_checked_in"] != false {
		t.Errorf("on = %v", on)
	}
	if off := mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/souvenir", map[string]any{"souvenir_received": false}), 200).data(); off["souvenir_received"] != false {
		t.Errorf("off = %v", off)
	}
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+gid+"/souvenir", map[string]any{"souvenir_received": "ya"}), 400, "invalid_json", "")
	wantError(t, e.do(tok, "PATCH", guests(id)+"/"+uuid.NewString()+"/souvenir", map[string]any{"souvenir_received": true}), 404, "not_found", "guest not found")
}

func TestWalkIn(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	w := mustStatus(t, e.do(tok, "POST", guests(id)+"/walk-in", map[string]any{
		"name": " Tamu  Dadakan ", "guest_side": "wanita", "category": "vip", "table_number": "VIP-05", "pax_actual": 3, "souvenir_received": true, "notes": "Rekan Ayah",
	}), 201).data()
	if w["name"] != "Tamu Dadakan" || w["guest_side"] != "wanita" || w["category"] != "vip" || w["table_number"] != "VIP-05" || num(w["pax_count"]) != 3 ||
		num(w["pax_actual"]) != 3 || w["rsvp_status"] != "attending" || w["rsvp_submitted_at"] == nil || w["is_checked_in"] != true || w["checked_in_at"] == nil ||
		w["souvenir_received"] != true || w["sub_event_scope"] != "all" || w["notes"] != "Rekan Ayah" || len(w["qr_code_token"].(string)) != 32 {
		t.Errorf("walk-in = %v", w)
	}
	// He is a guest of the list like the others, last in order, and counted at the desk.
	l := list(guestList(e, tok, id, "limit=500")["guests"])
	if len(l) != 21 || l[20]["id"] != w["id"] {
		t.Errorf("the walk-in is not last of %d guests", len(l))
	}
	if k := kpi(e, tok, id); num(k["checked_in_guests"]) != 1 || num(k["expected_guests"]) != 19 || num(k["actual_pax"]) != 3 || num(k["souvenir_distributed"]) != 1 {
		t.Errorf("kpi = %v", k)
	}

	// Cancelling his check-in leaves an attending guest who has not come; deleting him removes him.
	c := mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+w["id"].(string)+"/checkin", nil), 200).data()
	if c["rsvp_status"] != "attending" || c["is_checked_in"] != false {
		t.Errorf("cancelled walk-in = %v", c)
	}

	for _, c := range []struct {
		name string
		body map[string]any
		code string
		msg  string
	}{
		{"no name", map[string]any{"pax_actual": 1}, "invalid_input", "name: required"},
		{"no pax", map[string]any{"name": "X"}, "invalid_input", "pax_actual: must be between 1 and 20"},
		{"pax 21", map[string]any{"name": "X", "pax_actual": 21}, "invalid_input", "pax_actual: must be between 1 and 20"},
		{"a side that does not exist", map[string]any{"name": "X", "pax_actual": 1, "guest_side": "kiri"}, "invalid_input", "guest_side: must be a registry side key"},
		{"a category that is too long", map[string]any{"name": "X", "pax_actual": 1, "category": strings.Repeat("c", 61)}, "invalid_input", "category: must be at most 60 characters"},
	} {
		wantError(t, e.do(tok, "POST", guests(id)+"/walk-in", c.body), 400, c.code, c.msg)
	}
}

// The illustration of the wireframe: 11 of the 18 expected guests are in, 22 of 39 pax, 11 souvenirs. Then a guest who
// declined arrives: he counts, and "not yet" does not drop below the guests who really are still outside.
func TestReceptionKPIWithADeclinedArrival(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	k := kpi(e, tok, id)
	if num(k["checked_in_guests"]) != 0 || num(k["expected_guests"]) != 18 || num(k["pending_guests"]) != 18 || num(k["actual_pax"]) != 0 || num(k["expected_pax"]) != 39 {
		t.Fatalf("before the event: %v", k)
	}
	morning(t, e, tok, id)
	k = kpi(e, tok, id)
	if num(k["checked_in_guests"]) != 11 || num(k["expected_guests"]) != 18 || num(k["pending_guests"]) != 7 || num(k["actual_pax"]) != 22 ||
		num(k["expected_pax"]) != 39 || num(k["souvenir_distributed"]) != 11 {
		t.Fatalf("morning: %v", k)
	}

	joko := guestNamed(t, e, tok, id, "Bpk. Joko Susilo (Ketua RT 04)")["id"].(string) // declined, 2 invited
	mustStatus(t, checkin(e, tok, id, joko, 2, false), 200)
	k = kpi(e, tok, id)
	if num(k["checked_in_guests"]) != 12 || num(k["expected_guests"]) != 19 || num(k["pending_guests"]) != 7 || num(k["actual_pax"]) != 24 || num(k["expected_pax"]) != 41 {
		t.Errorf("with a declined arrival: %v (still outside must stay 7)", k)
	}
	if g := get(e, tok, guests(id)+"/"+joko); g["rsvp_status"] != "declined" {
		t.Errorf("the check-in changed the answer: %v", g["rsvp_status"])
	}
	// Taking him off again restores the first figures.
	mustStatus(t, e.do(tok, "DELETE", guests(id)+"/"+joko+"/checkin", nil), 200)
	if k = kpi(e, tok, id); num(k["expected_guests"]) != 18 || num(k["pending_guests"]) != 7 {
		t.Errorf("after the cancel: %v", k)
	}
}

func TestReceptionQueueRules(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	morning(t, e, tok, id)

	names := func(query string) []string {
		var out []string
		for _, g := range list(reception(e, tok, id, query)["guests"]) {
			out = append(out, g["name"].(string))
		}

		return out
	}

	// All = the 18 expected guests, by name; the two who declined and did not come are left out.
	all := names("limit=200")
	if len(all) != 18 {
		t.Fatalf("all = %d guests: %v", len(all), all)
	}
	for _, n := range all {
		if strings.Contains(n, "Joko") || strings.Contains(n, "Dedi") {
			t.Errorf("a guest who declined is in the queue: %s", n)
		}
	}
	for i := 1; i < len(all); i++ {
		if strings.ToLower(all[i-1]) > strings.ToLower(all[i]) {
			t.Errorf("not sorted by name: %q before %q", all[i-1], all[i])
		}
	}
	// In = the 11 who came, newest first; pending = the 7 who did not, by name.
	in := names("status=in")
	if len(in) != 11 || in[0] != "Sdri. Citra Lestari" || in[10] != "Bpk. Prof. Bambang Soedjarwo" {
		t.Errorf("in = %v", in)
	}
	pending := names("status=pending")
	if len(pending) != 7 {
		t.Errorf("pending = %v", pending)
	}
	if pending[0] != "Anisa Rahmawati (Sahabat SMA)" { // by name: she is first of the 7 who are still outside
		t.Errorf("pending starts with %q", pending[0])
	}

	// A search widens the queue to the guests who declined: one of them may stand at the door.
	if got := names("search=dedi"); len(got) != 1 || !strings.Contains(got[0], "Dedi") {
		t.Errorf("search=dedi = %v", got)
	}
	if got := names("status=pending&search=joko"); len(got) != 1 {
		t.Errorf("a declined guest cannot be found in pending: %v", got)
	}
	if got := names("status=in&search=joko"); len(got) != 0 {
		t.Errorf("he has not come: %v", got)
	}
	if got := names("search=VIP-0"); len(got) != 4 {
		t.Errorf("search by table = %v", got)
	}
	if got := names("search=%25"); len(got) != 0 {
		t.Errorf("search=%% matched %v", got)
	}

	// Category and page.
	if got := names("category=vip"); len(got) != 2 {
		t.Errorf("category=vip = %v", got)
	}
	if got := names("category=tetangga"); len(got) != 1 { // two neighbours: one declined, one maybe
		t.Errorf("category=tetangga = %v", got)
	}
	r := reception(e, tok, id, "limit=5&offset=10")
	if len(list(r["guests"])) != 5 || num(r["total"]) != 18 || num(r["limit"]) != 5 || num(r["offset"]) != 10 {
		t.Errorf("page = %d guests, total %v", len(list(r["guests"])), r["total"])
	}
	if d := reception(e, tok, id, ""); num(d["limit"]) != 50 || len(list(d["guests"])) != 18 {
		t.Errorf("the default page is 50: limit %v, %d guests", d["limit"], len(list(d["guests"])))
	}

	// The packet: filters count the expected guests (not the search), the newest check-ins are 10 and in order.
	full := reception(e, tok, id, "search=zzz")
	f := full["filters"].(map[string]any)
	if num(f["all"]) != 18 || num(f["checked_in"]) != 11 || num(f["pending"]) != 7 {
		t.Errorf("filters = %v", f)
	}
	counts := map[string]int64{}
	for _, c := range list(f["categories"]) {
		counts[c["key"].(string)] = num(c["invitations"])
		if _, has := c["pax"]; has && num(c["pax"]) != 0 {
			t.Errorf("filter category %v has pax", c)
		}
	}
	if len(counts) != 6 || counts["tetangga"] != 1 || counts["teman_ortu"] != 2 || counts["teman"] != 6 || counts["kolega"] != 3 || counts["keluarga"] != 4 || counts["vip"] != 2 {
		t.Errorf("filter categories = %v", counts)
	}
	recent := list(full["recent_checkins"])
	if len(recent) != 10 || recent[0]["name"] != "Sdri. Citra Lestari" || recent[9]["name"] != "Raditya Pratama & Pasangan" {
		t.Errorf("recent = %d, first %v", len(recent), recent[0]["name"])
	}

	for query, msg := range map[string]string{
		"status=done": "status: must be one of all, in, pending", "limit=0": "limit: must be between 1 and 200", "limit=201": "limit: must be between 1 and 200",
		"offset=-3": "offset: must be at least 0", "category=" + strings.Repeat("k", 61): "category: must be at most 60 characters",
	} {
		wantError(t, e.do(tok, "GET", guests(id)+"/reception-overview?"+query, nil), 400, "invalid_input", msg)
	}
}

// The filter chips show a free-text category once a guest uses it, and the fallback.
func TestReceptionFiltersShowCustomCategories(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	addGuest(e, tok, id, guestBody("Kustom", map[string]any{"category": "Rekan Bisnis"}))
	addGuest(e, tok, id, guestBody("Tanpa kategori", map[string]any{"category": ""}))
	declined := addGuest(e, tok, id, guestBody("Menolak", map[string]any{"category": "Rahasia"}))["id"].(string)
	mustStatus(t, e.do(tok, "PATCH", guests(id)+"/"+declined+"/rsvp", map[string]any{"rsvp_status": "declined"}), 200)

	var keys []string
	custom := map[string]bool{}
	for _, c := range list(reception(e, tok, id, "")["filters"].(map[string]any)["categories"]) {
		keys = append(keys, c["key"].(string))
		custom[c["key"].(string)], _ = c["custom"].(bool)
	}
	if got := strings.Join(keys, ","); got != "teman,kolega,keluarga,tetangga,teman_ortu,vip,lainnya,Rekan Bisnis" {
		t.Errorf("filter categories = %s (a category held only by a guest who declined must not appear)", got)
	}
	if custom["teman"] || !custom["lainnya"] || !custom["Rekan Bisnis"] {
		t.Errorf("custom flags = %v", custom)
	}
}

// The stock comes from the shopping list of menu 04: the finished items marked souvenir, rounded down. No item marked, no key.
func TestSouvenirStockOmittedWithoutItems(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	if _, has := kpi(e, tok, id)["souvenir_stock"]; has {
		t.Fatalf("the sample list has no souvenir item but the stock is sent: %v", kpi(e, tok, id))
	}
	item := func(name string, qty float64, status string) {
		mustStatus(t, e.do(tok, "POST", base(id)+"/shopping-items", map[string]any{
			"sub_event_scope": "all", "category": "Souvenir", "item_name": name, "quantity": qty, "estimated_price": 5000, "purpose": "souvenir", "status": status,
		}), 201)
	}
	item("Tumbler", 300, "selesai")
	item("Pouch", 50.5, "selesai")
	item("Stiker", 20, "proses") // not finished: not in stock
	if s := kpi(e, tok, id)["souvenir_stock"]; num(s) != 350 {
		t.Errorf("souvenir_stock = %v, want 350 (floor of 350.5, the unfinished item left out)", s)
	}
	// Items marked souvenir but none finished: stock 0 is still stock (the key stays).
	other := createProject(e, tok, nil)
	mustStatus(t, e.do(tok, "POST", base(other)+"/shopping-items", map[string]any{
		"sub_event_scope": "all", "category": "Souvenir", "item_name": "Belum jadi", "quantity": 10, "estimated_price": 1, "purpose": "souvenir",
	}), 201)
	if k := kpi(e, tok, other); k["souvenir_stock"] == nil || num(k["souvenir_stock"]) != 0 {
		t.Errorf("kpi = %v", k)
	}
	// ... and a client subtracts: more handed out than in stock is negative on the screen, never an error here.
	guestID := addGuest(e, tok, other, guestBody("Penerima", nil))["id"].(string)
	mustStatus(t, e.do(tok, "PATCH", guests(other)+"/"+guestID+"/souvenir", map[string]any{"souvenir_received": true}), 200)
	if k := kpi(e, tok, other); num(k["souvenir_distributed"]) != 1 || num(k["souvenir_stock"]) != 0 {
		t.Errorf("kpi = %v", k)
	}
}

// Menu 04 sums the stock in Go (hundredths), menu 05 in SQL (numeric): two implementations of one rule. Whatever the quantities
// are, the shopping screen and the reception desk must show the same stock.
func TestSouvenirStockAgreesWithMenu04(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	stock04 := func() any {
		return mustStatus(t, e.do(tok, "GET", base(id)+"/shopping-items", nil), 200).data()["kpi"].(map[string]any)["souvenir_stock"]
	}

	// Fractions that add up just below and just on a whole number, finished and unfinished, so a rounding rule that differs
	// by one hundredth shows as a different stock.
	for i, c := range []struct {
		qty    float64
		status string
	}{{0.99, "selesai"}, {0.01, "selesai"}, {0.99, "selesai"}, {12.34, "proses"}, {0.5, "selesai"}, {0.49, "selesai"}, {1000000, "belum"}, {0.01, "selesai"}} {
		mustStatus(t, e.do(tok, "POST", base(id)+"/shopping-items", map[string]any{
			"sub_event_scope": "all", "category": "Souvenir", "item_name": fmt.Sprintf("Souvenir %d", i), "quantity": c.qty,
			"estimated_price": 1000, "purpose": "souvenir", "status": c.status,
		}), 201)
		if a, b := stock04(), kpi(e, tok, id)["souvenir_stock"]; a != b {
			t.Fatalf("after item %d: menu 04 says %v, menu 05 says %v", i, a, b)
		}
	}
	if s := kpi(e, tok, id)["souvenir_stock"]; num(s) != 2 { // 0.99+0.01+0.99+0.5+0.49+0.01 = 2.99: rounded down, never up
		t.Errorf("souvenir_stock = %v, want 2", s)
	}
}

// Check-ins of a big event: the queue answers with one page, not the whole list (05 BE 2.5).
func TestReceptionPollingPacketStaysSmall(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	insertGuests(e, id, 600)

	r := reception(e, tok, id, "")
	if len(list(r["guests"])) != 50 || num(r["total"]) != 600 || num(r["kpi"].(map[string]any)["expected_guests"]) != 600 {
		t.Errorf("packet = %d guests, total %v", len(list(r["guests"])), r["total"])
	}
	exec(e, `UPDATE planner_guests SET is_checked_in=TRUE, checked_in_at=now() - make_interval(secs => sort_order), pax_actual=1 WHERE project_id=$1 AND sort_order <= 1030`, id)
	r = reception(e, tok, id, "")
	if len(list(r["recent_checkins"])) != 10 {
		t.Errorf("recent = %d, want 10", len(list(r["recent_checkins"])))
	}
	if fmt.Sprint(r["kpi"].(map[string]any)["checked_in_guests"]) != "30" {
		t.Errorf("kpi = %v", r["kpi"])
	}
}
