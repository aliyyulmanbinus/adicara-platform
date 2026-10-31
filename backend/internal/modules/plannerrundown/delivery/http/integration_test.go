package http_test

import (
	"reflect"
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// GET /event-types now carries the moment tags of the registry and offers the rundown to every type and the playlist to
// the wedding only (01 BE 3.1; the registry fields appear when their applier is registered).
func TestEventTypesCarryMomentTagsAndTemplates(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	r := mustStatus(t, e.do(tok, "GET", "/v1/planner/event-types", nil), 200)
	raw, _ := r.json()["data"].([]any)
	types := list(raw)
	if len(types) != 9 {
		t.Fatalf("%d event types", len(types))
	}
	reg := pd.DefaultRegistry()
	for _, et := range types {
		key := et["key"].(string)
		spec, _ := reg.Get(pd.EventType(key))
		var tags []string
		for _, tag := range et["moment_tags"].([]any) {
			tags = append(tags, tag.(string))
		}
		if !reflect.DeepEqual(tags, spec.MomentTags) || len(tags) == 0 {
			t.Errorf("%s: moment_tags = %v, want %v", key, tags, spec.MomentTags)
		}
		avail := et["templates_available"].(map[string]any)
		if avail["rundown"] != true || (avail["playlist"] == true) != (key == "wedding") {
			t.Errorf("%s: templates_available = %v", key, avail)
		}
	}
}

// Deleting a session that still has agendas is refused with the usage; with delete_agendas the agendas go and the songs
// that pointed at them stay, unlinked (01 BE 4.6, 06 BE 6).
func TestSessionUsageAndDeletion(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)

	raw, _ := mustStatus(t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"].([]any)
	usage := map[string]int64{}
	for _, s := range list(raw) {
		if u, ok := s["usage"].(map[string]any); ok {
			usage[s["scope_key"].(string)] = num(u["agendas"])
		}
	}
	if usage["lamaran"] != 11 || usage["akad"] != 8 || usage["resepsi"] != 14 {
		t.Fatalf("usage.agendas = %v", usage)
	}

	ijab := agendaByKey(t, rundownItems(e, tok, id), "wedding.rundown_05")["id"].(string)
	song := list(playlistOverview(e, tok, id)["items"])[0]
	mustStatus(t, e.do(tok, "PUT", base(id)+"/playlists/"+song["id"].(string), songBody(song["song_title"].(string), song["artist"].(string), map[string]any{"rundown_id": ijab})), 200)

	r := e.do(tok, "DELETE", base(id)+"/sessions/"+sess["akad"], nil)
	wantError(t, r, 409, "session_in_use", "")
	if n := num(r.json()["error"].(map[string]any)["details"].(map[string]any)["usage"].(map[string]any)["agendas"]); n != 8 {
		t.Errorf("details.usage.agendas = %d", n)
	}
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 33 {
		t.Fatalf("a refused delete removed agendas: %d left", n)
	}

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+sess["akad"]+"?delete_agendas=true", nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_rundowns`); n != 25 {
		t.Errorf("%d agendas after deleting the akad, want 25", n)
	}
	songs := list(playlistOverview(e, tok, id)["items"])
	if len(songs) != 30 {
		t.Fatalf("%d songs, the delete must not remove songs", len(songs))
	}
	if _, linked := songs[0]["rundown_id"]; linked {
		t.Errorf("the song still points at a deleted agenda: %v", songs[0])
	}
	if len(list(rundownOverview(e, tok, id)["sessions"])) != 2 {
		t.Errorf("sessions = %v", rundownOverview(e, tok, id)["sessions"])
	}
}

func TestWhatsAppTextsOverHTTP(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sess := sessionIDs(e, tok, id)
	text := func(path string) string { return get(e, tok, base(id)+path)["text"].(string) }

	// every session: three sections, 33 agendas, the open ones without an end
	all := text("/rundowns/export-wa")
	for _, want := range []string{"📋 *RUNDOWN ACARA*\n💍 *Contoh: Pernikahan Sinta & Arif*\n", "*💍 LAMARAN*\n🗓 *Kamis, 15 Oktober 2026*\n", "*📜 AKAD NIKAH*\n🗓 *Minggu, 28 Maret 2027*\n",
		"*🏛️ RESEPSI*", "⏰ *10.40 - selesai* : Ramah Tamah - penutup", "⏰ *09.15 - selesai* : Resepsi"} {
		if !strings.Contains(all, want) {
			t.Errorf("text lacks %q", want)
		}
	}
	if n := strings.Count(all, "⏰"); n != 33 {
		t.Errorf("%d agendas in the text, want 33", n)
	}
	if one := text("/rundowns/export-wa?session_id=" + sess["resepsi"]); strings.Count(one, "⏰") != 14 || !strings.HasPrefix(one, "📋 *RUNDOWN ACARA RESEPSI*") {
		t.Errorf("resepsi text:\n%s", one)
	}

	// the briefing: nothing is linked yet, so every song is under its moment tag
	brief := text("/playlists/export-wa")
	if !strings.HasPrefix(brief, "🎵 *SOUND CUE Contoh: Pernikahan Sinta & Arif*\n") || !strings.Contains(brief, "*Belum ditautkan ke agenda*\n*Akad Khidmat*\n• Akad - Payung Teduh (Digital Playback)\n") ||
		strings.Count(brief, "• ") != 30 {
		t.Errorf("briefing:\n%s", brief)
	}
	// link one song to the ijab kabul: its section appears, and the song leaves the unlinked ones
	first := list(playlistOverview(e, tok, id)["items"])[0]
	ijab := agendaByKey(t, rundownItems(e, tok, id), "wedding.rundown_05")["id"].(string)
	mustStatus(t, e.do(tok, "PUT", base(id)+"/playlists/"+first["id"].(string), songBody("Akad", "Payung Teduh", map[string]any{
		"rundown_id": ijab, "moment_tag": "Akad Khidmat", "audio_cue_notes": "Mulai 0:12",
	})), 200)
	brief = text("/playlists/export-wa")
	if !strings.Contains(brief, "*📜 AKAD NIKAH*\n⏰ 07.46 Ijab Kabul\n• Akad - Payung Teduh (Digital Playback)\n   ↳ Cue: Mulai 0:12\n") || strings.Count(brief, "• ") != 30 ||
		strings.Contains(brief, "*Akad Khidmat*\n• Akad - Payung Teduh") {
		t.Errorf("briefing after the link:\n%s", brief)
	}
}
