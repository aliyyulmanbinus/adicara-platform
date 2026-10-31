package http_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPlaylistCreateDefaultsAndShape(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	s := addSong(e, tok, id, songBody("  Akad  ", " Payung Teduh ", nil))
	want := map[string]any{
		"song_title": "Akad", "artist": "Payung Teduh", "moment_tag": "Lainnya", "audio_format": "digital_playback",
		"status": "needs_review", "sort_order": float64(1),
	}
	for k, w := range want {
		if s[k] != w {
			t.Errorf("%s = %#v, want %#v", k, s[k], w)
		}
	}
	for _, k := range []string{"rundown_id", "agenda", "spotify_url", "youtube_url", "audio_cue_notes", "template_key"} {
		if _, ok := s[k]; ok {
			t.Errorf("%s must be omitted: %v", k, s)
		}
	}

	full := addSong(e, tok, id, songBody("Janji Suci", "Yovie & Nuno", map[string]any{
		"moment_tag": "Akad Khidmat", "audio_format": "live_band", "status": "ready",
		"spotify_url": "https://open.spotify.com/track/x", "youtube_url": " https://youtu.be/x ", "audio_cue_notes": "Mulai 0:12\r\nFade out 0:40",
	}))
	if full["moment_tag"] != "Akad Khidmat" || full["audio_format"] != "live_band" || full["status"] != "ready" || num(full["sort_order"]) != 2 ||
		full["youtube_url"] != "https://youtu.be/x" || full["audio_cue_notes"] != "Mulai 0:12\nFade out 0:40" {
		t.Errorf("song = %v", full)
	}
	// a blank link is no link
	blank := addSong(e, tok, id, songBody("X", "Y", map[string]any{"spotify_url": "  ", "audio_cue_notes": ""}))
	if _, ok := blank["spotify_url"]; ok {
		t.Errorf("a blank link is kept: %v", blank)
	}
}

func TestPlaylistValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	post := func(over map[string]any) resp {
		return e.do(tok, "POST", base(id)+"/playlists", songBody("Akad", "Payung Teduh", over))
	}

	wantError(t, post(map[string]any{"song_title": " "}), 400, "invalid_input", "song_title: required")
	wantError(t, post(map[string]any{"song_title": strings.Repeat("a", 256)}), 400, "invalid_input", "song_title: must be at most 255 characters")
	wantError(t, post(map[string]any{"artist": ""}), 400, "invalid_input", "artist: required")
	wantError(t, post(map[string]any{"artist": strings.Repeat("a", 256)}), 400, "invalid_input", "artist: must be at most 255 characters")
	wantError(t, post(map[string]any{"moment_tag": strings.Repeat("a", 101)}), 400, "invalid_input", "moment_tag: must be at most 100 characters")
	wantError(t, post(map[string]any{"audio_format": "vinyl"}), 400, "invalid_input", "audio_format: must be one of digital_playback, live_band, acoustic, sound_fx")
	wantError(t, post(map[string]any{"status": "done"}), 400, "invalid_input", "status: must be one of ready, needs_review, missing_file")
	for _, bad := range []string{"ftp://x.com/a", "javascript:alert(1)", "not a url", "https://", strings.Repeat("a", 2001), "https://x.com/\x07"} {
		wantError(t, post(map[string]any{"spotify_url": bad}), 400, "invalid_input", "spotify_url: must be an http or https URL")
		wantError(t, post(map[string]any{"youtube_url": bad}), 400, "invalid_input", "youtube_url: must be an http or https URL")
	}
	wantError(t, post(map[string]any{"audio_cue_notes": strings.Repeat("a", 2001)}), 400, "invalid_input", "audio_cue_notes: must be at most 2000 characters")
	mustStatus(t, post(map[string]any{"moment_tag": strings.Repeat("é", 100), "audio_cue_notes": strings.Repeat("a", 2000)}), 201)
	wantError(t, e.do(tok, "POST", base(id)+"/playlists", `{"song_title": 3}`), 400, "invalid_json", "")

	// an agenda of another project, or none at all, is refused
	other := e.login("other")
	oid := createProject(e, other, nil)
	oa := addAgenda(e, other, oid, agendaBody(sessionIDs(e, other, oid)["akad"], "07:00", "", "x", nil))
	for _, bad := range []string{oa["id"].(string), uuid.NewString()} {
		wantError(t, post(map[string]any{"rundown_id": bad}), 400, "invalid_input", "rundown_id: must be an agenda of this project")
	}
}

func TestPlaylistUpdateFilterAndKPI(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	a := addAgenda(e, tok, id, agendaBody(sessionIDs(e, tok, id)["akad"], "07:46", "07:50", "Ijab Kabul", nil))
	s1 := addSong(e, tok, id, songBody("Akad", "Payung Teduh", map[string]any{"moment_tag": "Akad Khidmat", "rundown_id": a["id"]}))
	addSong(e, tok, id, songBody("Marry You", "Maroon 5", map[string]any{"moment_tag": "Kirab / Entrance", "audio_format": "live_band", "status": "ready"}))
	addSong(e, tok, id, songBody("Perfect", "Ed Sheeran", map[string]any{"moment_tag": "Kirab / Entrance", "audio_format": "acoustic", "status": "missing_file"}))
	addSong(e, tok, id, songBody("Bumper", "FX", map[string]any{"audio_format": "sound_fx"}))

	ov := playlistOverview(e, tok, id)
	k := ov["kpi"].(map[string]any)
	by := k["by_format"].(map[string]any)
	if num(k["total_tracks"]) != 4 || num(k["mapped_moments"]) != 3 || num(k["ready"]) != 1 || num(k["needs_review"]) != 2 || num(k["missing_file"]) != 1 ||
		num(by["digital_playback"]) != 1 || num(by["live_band"]) != 1 || num(by["acoustic"]) != 1 || num(by["sound_fx"]) != 1 {
		t.Errorf("kpi = %v", k)
	}

	// the filters narrow the items and leave the KPI whole
	for query, want := range map[string]int{
		"?moment_tag=Kirab / Entrance": 2, "?moment_tag=kirab / entrance": 0, "?audio_format=live_band": 1, "?status=needs_review": 2,
		"?moment_tag=Kirab / Entrance&status=ready": 1, "?audio_format=sound_fx&status=ready": 0,
	} {
		got := get(e, tok, base(id)+"/playlists"+strings.ReplaceAll(query, " ", "%20"))
		if n := len(list(got["items"])); n != want {
			t.Errorf("%s: %d items, want %d", query, n, want)
		}
		if num(got["kpi"].(map[string]any)["total_tracks"]) != 4 {
			t.Errorf("%s: the KPI follows the filter: %v", query, got["kpi"])
		}
	}
	wantError(t, e.do(tok, "GET", base(id)+"/playlists?audio_format=vinyl", nil), 400, "invalid_input", "audio_format: must be one of digital_playback, live_band, acoustic, sound_fx")
	wantError(t, e.do(tok, "GET", base(id)+"/playlists?status=x", nil), 400, "invalid_input", "status: must be one of ready, needs_review, missing_file")
	wantError(t, e.do(tok, "GET", base(id)+"/playlists?moment_tag="+strings.Repeat("a", 101), nil), 400, "invalid_input", "moment_tag: must be at most 100 characters")

	// PUT replaces the whole song: a field left out is cleared; the order stays
	sid := s1["id"].(string)
	r := mustStatus(t, e.do(tok, "PUT", base(id)+"/playlists/"+sid, songBody("Akad (live)", "Payung Teduh", map[string]any{"audio_format": "live_band"})), 200).data()
	if r["song_title"] != "Akad (live)" || r["moment_tag"] != "Lainnya" || r["audio_format"] != "live_band" || num(r["sort_order"]) != 1 {
		t.Errorf("after PUT: %v", r)
	}
	if _, ok := r["rundown_id"]; ok {
		t.Errorf("PUT without rundown_id kept the link: %v", r)
	}
	wantError(t, e.do(tok, "PUT", base(id)+"/playlists/"+uuid.NewString(), songBody("x", "y", nil)), 404, "not_found", "song not found")
	wantError(t, e.do(tok, "PUT", base(id)+"/playlists/nope", songBody("x", "y", nil)), 404, "not_found", "song not found")

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/playlists/"+sid, nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/playlists/"+sid, nil), 404, "not_found", "song not found")
	if n := num(playlistOverview(e, tok, id)["kpi"].(map[string]any)["total_tracks"]); n != 3 {
		t.Errorf("%d songs after the delete", n)
	}
}

func TestReorderPlaylists(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	order := func() []string {
		var out []string
		for _, s := range list(playlistOverview(e, tok, id)["items"]) {
			out = append(out, s["id"].(string))
		}

		return out
	}
	cur := order()
	rev := make([]string, len(cur))
	for i, v := range cur {
		rev[len(cur)-1-i] = v
	}
	reorder := func(ids []string) resp {
		return e.do(tok, "POST", base(id)+"/playlists/reorder", map[string]any{"ids": ids})
	}

	mustStatus(t, reorder(rev), 204)
	if got := order(); !reflect.DeepEqual(got, rev) {
		t.Fatalf("order = %v", got)
	}
	const msg = "ids: must list every song of the project exactly once"
	for name, ids := range map[string][]string{
		"repeated": append([]string{rev[0]}, rev[:len(rev)-1]...), "missing": rev[1:], "extra": append(append([]string{}, rev...), uuid.NewString()),
		"unknown": append([]string{uuid.NewString()}, rev[1:]...), "empty": {},
	} {
		wantError(t, reorder(ids), 400, "invalid_input", msg)
		if got := order(); !reflect.DeepEqual(got, rev) {
			t.Fatalf("%s changed the order", name)
		}
	}
}
