package http_test

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// The contract of the module is backend/docs/openapi.yaml. These tests keep it in step with the code: every route is
// documented (and the other way round) and every schema has the keys of its Go struct. The route test needs the
// database only for the router (newEnv); the schema tests read files only.

const openapiPath = "../../../../../docs/openapi.yaml"

type node = map[string]any

func loadOpenAPI(t *testing.T) node {
	t.Helper()
	raw, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Skipf("openapi.yaml not available: %v", err)
	}
	var doc node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}

	return doc
}

var pathParam = regexp.MustCompile(`:[a-z_]+|\{[A-Za-z]+\}`)

func normalisePath(path string) string { return pathParam.ReplaceAllString(path, "{}") }

func TestOpenAPIDocumentsEveryRundownRoute(t *testing.T) {
	doc := loadOpenAPI(t)
	e := newEnv(t)

	documented := map[string]bool{}
	for path, item := range doc["paths"].(node) {
		for method, op := range item.(node) {
			if o, ok := op.(node); ok && method != "parameters" {
				if tags, _ := o["tags"].([]any); len(tags) == 1 && tags[0] == "Planner Rundown" {
					documented[strings.ToUpper(method)+" "+normalisePath(path)] = true
				}
			}
		}
	}

	registered := map[string]bool{}
	for _, r := range e.engine.Routes() {
		if rest, found := strings.CutPrefix(r.Path, "/v1/planner/projects/:id/"); found && regexp.MustCompile(`^(rundowns|playlists)(/|$)`).MatchString(rest) {
			registered[r.Method+" "+normalisePath(r.Path)] = true
		}
	}

	if len(registered) != 17 || len(documented) != 17 {
		t.Errorf("%d routes registered, %d documented; the module has 17", len(registered), len(documented))
	}
	for r := range registered {
		if !documented[r] {
			t.Errorf("%s is not in openapi.yaml", r)
		}
	}
	for d := range documented {
		if !registered[d] {
			t.Errorf("openapi.yaml documents %s, which is not registered", d)
		}
	}
}

// goShape lists the JSON paths of a Go type with whether the key is always sent (no omitempty).
func goShape(t reflect.Type, prefix string, out map[string]bool) {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		if t.Kind() == reflect.Slice {
			prefix += "[]"
		}
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t.String() == "uuid.UUID" || t.String() == "time.Time" {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		path := prefix + "." + parts[0]
		out[path] = len(parts) < 2 || parts[1] != "omitempty"
		goShape(f.Type, path, out)
	}
}

func oaShape(doc node, s node, prefix string, out map[string]bool) {
	if r, ok := s["$ref"].(string); ok {
		oaShape(doc, doc["components"].(node)["schemas"].(node)[strings.TrimPrefix(r, "#/components/schemas/")].(node), prefix, out)
		return
	}
	if items, ok := s["items"].(node); ok {
		oaShape(doc, items, prefix+"[]", out)
		return
	}
	required := map[string]bool{}
	if l, ok := s["required"].([]any); ok {
		for _, x := range l {
			required[x.(string)] = true
		}
	}
	props, _ := s["properties"].(node)
	for name, p := range props {
		path := prefix + "." + name
		out[path] = required[name]
		oaShape(doc, p.(node), path, out)
	}
}

// TestOpenAPISchemasMatchTheGoDTOs compares the keys of every schema with the JSON tags of its struct, and for the
// responses which keys are always present (required) with which are not omitempty. Requests are compared by key only:
// what a client must send is decided by the usecase.
func TestOpenAPISchemasMatchTheGoDTOs(t *testing.T) {
	doc := loadOpenAPI(t)
	pairs := map[string]any{
		"PlannerRundownVendor": domain.RundownVendor{}, "PlannerRundown": domain.Rundown{}, "PlannerRundownSession": domain.RundownSessionSummary{},
		"PlannerRundownKPI": domain.RundownKPI{}, "PlannerRundownOverview": domain.RundownOverview{}, "PlannerRundownLiveItem": domain.RundownLiveItem{},
		"PlannerRundownLive": domain.RundownLive{}, "PlannerRundownRequest": domain.RundownRequest{}, "PlannerRundownStatusRequest": domain.RundownStatusRequest{},
		"PlannerRundownReorderRequest": domain.ReorderRundownsRequest{}, "PlannerPlaylistReorderRequest": domain.ReorderPlaylistsRequest{},
		"PlannerRundownTemplateRequest": domain.ApplyTemplateRequest{}, "PlannerRundownTemplateResult": domain.TemplateResult{},
		"PlannerWAExport": domain.WAExport{}, "PlannerPlaylistAgenda": domain.PlaylistAgenda{}, "PlannerPlaylist": domain.Playlist{},
		"PlannerPlaylistFormatCounts": domain.FormatCounts{}, "PlannerPlaylistKPI": domain.PlaylistKPI{}, "PlannerPlaylistOverview": domain.PlaylistOverview{},
		"PlannerPlaylistRequest": domain.PlaylistRequest{},
	}
	if len(pairs) != 20 {
		t.Fatalf("%d schema pairs, the module has 20", len(pairs))
	}

	for name, v := range pairs {
		fromGo, fromDoc := map[string]bool{}, map[string]bool{}
		goShape(reflect.TypeOf(v), "", fromGo)
		oaShape(doc, node{"$ref": "#/components/schemas/" + name}, "", fromDoc)

		seen := map[string]bool{}
		var paths []string
		for _, set := range []map[string]bool{fromGo, fromDoc} {
			for p := range set {
				if !seen[p] {
					seen[p] = true
					paths = append(paths, p)
				}
			}
		}
		sort.Strings(paths)

		for _, p := range paths {
			g, inGo := fromGo[p]
			d, inDoc := fromDoc[p]
			switch {
			case !inGo:
				t.Errorf("%s%s is in openapi.yaml only", name, p)
			case !inDoc:
				t.Errorf("%s%s is in the Go struct only", name, p)
			case !strings.HasSuffix(name, "Request") && g != d:
				t.Errorf("%s%s: always sent in Go = %v, required in openapi.yaml = %v", name, p, g, d)
			}
		}
	}
}

// The event type schema carries the moment tags of this menu.
func TestOpenAPIEventTypeHasMomentTags(t *testing.T) {
	doc := loadOpenAPI(t)
	props := map[string]bool{}
	oaShape(doc, node{"$ref": "#/components/schemas/PlannerEventType"}, "", props)
	for _, k := range []string{".moment_tags"} {
		if _, has := props[k]; !has {
			t.Errorf("PlannerEventType%s is not in openapi.yaml", k)
		}
	}
	goPaths := map[string]bool{}
	goShape(reflect.TypeOf(pd.EventTypeSpec{}), "", goPaths)
	if _, has := goPaths[".moment_tags"]; !has {
		t.Error("EventTypeSpec.moment_tags is missing")
	}
}

// The error codes of this menu are documented where they can happen.
func TestOpenAPIMentionsTheCodesOfThisMenu(t *testing.T) {
	raw, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Skipf("openapi.yaml not available: %v", err)
	}
	for _, code := range []string{"agenda_limit_reached", "playlist_limit_reached", "scope_missing", "invalid_template", "project_read_only", "rate_limited"} {
		if !strings.Contains(string(raw), code) {
			t.Errorf("%s is not in openapi.yaml", code)
		}
	}
}
