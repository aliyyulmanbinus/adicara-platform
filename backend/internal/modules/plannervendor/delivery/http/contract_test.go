package http_test

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// The contract of the module is backend/docs/openapi.yaml. These tests keep it in step with the code:
// every route is documented (and the other way round) and every schema has the keys of its Go struct.
// They need the database only for the routes (newEnv); the schema test reads files only.

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

func normalise(path string) string { return pathParam.ReplaceAllString(path, "{}") }

func TestOpenAPIDocumentsEveryVendorRoute(t *testing.T) {
	doc := loadOpenAPI(t)
	e := newEnv(t)

	documented := map[string]bool{}
	for path, item := range doc["paths"].(node) {
		for method, op := range item.(node) {
			if o, ok := op.(node); ok && method != "parameters" {
				if tags, _ := o["tags"].([]any); len(tags) == 1 && tags[0] == "Planner Vendor" {
					documented[strings.ToUpper(method)+" "+normalise(path)] = true
				}
			}
		}
	}

	registered := map[string]bool{}
	for _, r := range e.engine.Routes() {
		rest, found := strings.CutPrefix(r.Path, "/v1/planner/projects/:id/")
		if found && regexp.MustCompile(`^(vendors|shopping-items)(/|$)`).MatchString(rest) {
			registered[r.Method+" "+normalise(r.Path)] = true
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

// shape lists the JSON paths of a Go type with whether the key is always sent (no omitempty).
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

// TestOpenAPISchemasMatchTheGoDTOs compares the keys of every schema with the JSON tags of its struct, and for
// the responses which keys are always present (required) with which are not omitempty. Requests are compared by key only:
// what a client must send is decided by the usecase.
func TestOpenAPISchemasMatchTheGoDTOs(t *testing.T) {
	doc := loadOpenAPI(t)
	pairs := map[string]any{
		"PlannerLinePayment": domain.LinePayment{}, "PlannerVendor": domain.Vendor{}, "PlannerLimitGroup": domain.LimitGroup{},
		"PlannerVendorCategoryState": domain.CategoryState{}, "PlannerVendorKPI": domain.VendorKPI{}, "PlannerVendorOverview": domain.VendorOverview{},
		"PlannerVendorRequest": domain.VendorRequest{}, "PlannerVendorStatusRequest": domain.VendorStatusRequest{}, "PlannerVendorResult": domain.VendorResult{},
		"PlannerLockDealRequest": domain.LockDealRequest{}, "PlannerSharedItem": domain.SharedItem{}, "PlannerLockDealResult": domain.LockDealResult{},
		"PlannerShoppingItem": domain.ShoppingItem{}, "PlannerShoppingKPI": domain.ShoppingKPI{}, "PlannerRollupLine": domain.RollupLine{},
		"PlannerRollupTarget": domain.RollupTarget{}, "PlannerShoppingOverview": domain.ShoppingOverview{},
		"PlannerShoppingItemRequest": domain.ShoppingItemRequest{}, "PlannerUpdateShoppingItemRequest": domain.UpdateShoppingItemRequest{},
		"PlannerShoppingStatusRequest": domain.ShoppingStatusRequest{}, "PlannerShoppingTemplateRequest": domain.ShoppingTemplateRequest{},
		"PlannerShoppingTemplateResult": domain.ShoppingTemplateResult{}, "PlannerLinkToBudgetRequest": domain.LinkToBudgetRequest{},
		"PlannerLinkToBudgetResult": domain.LinkToBudgetResult{}, "PlannerUnlinkFromBudgetRequest": domain.UnlinkFromBudgetRequest{},
		"PlannerUnlinkFromBudgetResult": domain.UnlinkFromBudgetResult{},
	}
	if len(pairs) != 26 {
		t.Fatalf("%d schema pairs, the module has 26", len(pairs))
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
			if strings.Contains(p, "warnings[].") { // PlannerNotice belongs to menu 01
				continue
			}
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
