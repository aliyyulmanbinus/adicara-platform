package http_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------- rundown and playlist helpers ----------

// sessionIDs maps scope_key to the id of every session of a project.
func sessionIDs(e *env, tok, id string) map[string]string {
	e.t.Helper()
	out := map[string]string{}
	raw, _ := mustStatus(e.t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"].([]any)
	for _, s := range list(raw) {
		out[s["scope_key"].(string)] = s["id"].(string)
	}

	return out
}

// agendaBody is a valid POST /rundowns body; the overrides replace its keys. An empty end leaves the agenda open.
func agendaBody(sessionID, start, end, name string, over map[string]any) map[string]any {
	m := map[string]any{"session_id": sessionID, "start_time": start, "end_time": end, "activity_name": name}
	if end == "" {
		delete(m, "end_time")
	}
	for k, v := range over {
		m[k] = v
	}

	return m
}

func addAgenda(e *env, tok, id string, body map[string]any) map[string]any {
	e.t.Helper()

	return mustStatus(e.t, e.do(tok, "POST", base(id)+"/rundowns", body), 201).data()
}

func rundownOverview(e *env, tok, id string) map[string]any { return get(e, tok, base(id)+"/rundowns") }

func rundownItems(e *env, tok, id string) []map[string]any {
	return list(rundownOverview(e, tok, id)["items"])
}

// agendaByKey finds the agenda with a template key.
func agendaByKey(t *testing.T, items []map[string]any, key string) map[string]any {
	t.Helper()
	for _, it := range items {
		if it["template_key"] == key {
			return it
		}
	}
	t.Fatalf("no agenda with template_key %q", key)

	return nil
}

func setStatus(e *env, tok, id, agendaID, status string) resp {
	e.t.Helper()

	return e.do(tok, "PATCH", base(id)+"/rundowns/"+agendaID+"/status", map[string]any{"status": status})
}

func songBody(title, artist string, over map[string]any) map[string]any {
	m := map[string]any{"song_title": title, "artist": artist}
	for k, v := range over {
		m[k] = v
	}

	return m
}

func addSong(e *env, tok, id string, body map[string]any) map[string]any {
	e.t.Helper()

	return mustStatus(e.t, e.do(tok, "POST", base(id)+"/playlists", body), 201).data()
}

func playlistOverview(e *env, tok, id string) map[string]any {
	return get(e, tok, base(id)+"/playlists")
}

func applyAgendaTemplate(e *env, tok, id string, body any) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/rundowns/apply-template", body)
}

// ---------- fixtures of the spec ----------

// readFixture loads one of the JSON files the frontend spec was written against, so the two cannot drift.
func readFixture(t *testing.T, name string) any {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "..", "..", "docs", "APP_SPEC", "PLANNER_SPEC", "06_RUNDOWN_DAN_ACARA", "fixtures")
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skipf("spec fixture not available: %v", err) // a build of the backend alone has no docs folder
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}

	return v
}

// normalise makes two JSON documents comparable when they differ only in generated values: every UUID becomes
// "<id N>" with N the order of first appearance (both documents list the same things in the same order), and every
// time stamp becomes "<time>". Their presence still counts: a key that is there in one and missing in the other differs.
func normalise(v any, ids map[string]int) any {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys) // a fixed order, so the ids are numbered the same way in both documents
		out := make(map[string]any, len(x))
		for _, k := range keys {
			out[k] = normalise(x[k], ids)
		}

		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = normalise(val, ids)
		}

		return out
	case string:
		if _, err := uuid.Parse(x); err == nil && len(x) == 36 {
			n, ok := ids[x]
			if !ok {
				n = len(ids) + 1
				ids[x] = n
			}

			return "<id " + strconv.Itoa(n) + ">"
		}
		if len(x) >= 20 && strings.HasSuffix(x, "Z") && x[4] == '-' && x[10] == 'T' {
			return "<time>"
		}
	}

	return v
}
