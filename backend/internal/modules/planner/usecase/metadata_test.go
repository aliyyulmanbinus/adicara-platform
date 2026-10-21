package usecase

import (
	"strings"
	"testing"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func spec(t *testing.T, key domain.EventType) domain.EventTypeSpec {
	t.Helper()
	s, ok := domain.DefaultRegistry().Get(key)
	if !ok {
		t.Fatalf("no spec %s", key)
	}

	return s
}

func TestMergeMetadataValidatesAgainstTheRegistry(t *testing.T) {
	cases := []struct {
		name  string
		event domain.EventType
		in    map[string]any
		err   string // "" = accepted; otherwise the exact message
	}{
		{"wedding minimum", domain.EventWedding, map[string]any{"bride_name": "Sinta", "groom_name": "Arif"}, ""},
		{"all wedding keys", domain.EventWedding, map[string]any{"bride_name": "S", "groom_name": "A", "bride_parents": "x", "groom_parents": "y", "hero_title": "Bismillah Nikah", "venue_name": "v", "city": "c"}, ""},
		{"missing required", domain.EventWedding, map[string]any{"bride_name": "Sinta"}, "groom_name: required"},
		{"empty required", domain.EventWedding, map[string]any{"bride_name": " ", "groom_name": "Arif"}, "bride_name: required"},
		{"unknown key", domain.EventWedding, map[string]any{"bride_name": "S", "groom_name": "A", "child_name": "x"}, "child_name: unknown key for this event type"},
		{"wrong type string", domain.EventWedding, map[string]any{"bride_name": 5, "groom_name": "A"}, "bride_name: must be a string"},
		{"hero title too long", domain.EventWedding, map[string]any{"bride_name": "S", "groom_name": "A", "hero_title": strings.Repeat("x", 61)}, "hero_title: must be at most 60 characters"},
		{"hero title 60 runes", domain.EventWedding, map[string]any{"bride_name": "S", "groom_name": "A", "hero_title": strings.Repeat("é", 60)}, ""},
		{"city too long", domain.EventWedding, map[string]any{"bride_name": "S", "groom_name": "A", "city": strings.Repeat("x", 81)}, "city: must be at most 80 characters"},
		{"control character", domain.EventWedding, map[string]any{"bride_name": "S\nX", "groom_name": "A"}, "bride_name: must not contain control characters"},
		{"aqiqah date", domain.EventAqiqah, map[string]any{"baby_name": "Rayyan", "birth_date": "2026-09-01"}, ""},
		{"aqiqah bad date", domain.EventAqiqah, map[string]any{"baby_name": "Rayyan", "birth_date": "01/09/2026"}, "birth_date: must be a date in YYYY-MM-DD format"},
		{"goat count in range", domain.EventAqiqah, map[string]any{"baby_name": "R", "goat_count": float64(2)}, ""},
		{"goat count above max", domain.EventAqiqah, map[string]any{"baby_name": "R", "goat_count": float64(51)}, "goat_count: must be between 0 and 50"},
		{"goat count fraction", domain.EventAqiqah, map[string]any{"baby_name": "R", "goat_count": 1.5}, "goat_count: must be an integer"},
		{"goat count as string", domain.EventAqiqah, map[string]any{"baby_name": "R", "goat_count": "2"}, "goat_count: must be an integer"},
		{"age zero", domain.EventBirthday, map[string]any{"celebrant_name": "Keysha", "age": float64(0)}, "age: must be between 1 and 120"},
		{"age 17", domain.EventBirthday, map[string]any{"celebrant_name": "Keysha", "age": float64(17)}, ""},
		{"negative participants", domain.EventSeminar, map[string]any{"organizer_name": "A", "expected_participants": float64(-1)}, "expected_participants: must be between 0 and 1000000"},
		{"other has no required key", domain.EventOther, map[string]any{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mergeMetadata(spec(t, tc.event), nil, tc.in)
			if tc.err == "" {
				if err != nil {
					t.Fatalf("must be accepted: %v", err)
				}

				return
			}
			wantMessage(t, err, "invalid_metadata", tc.err)
		})
	}
}

func TestMergeMetadataNormalizesValuesAndDoesNotModifyTheBase(t *testing.T) {
	wedding := spec(t, domain.EventWedding)
	base := map[string]any{"bride_name": "Sinta", "groom_name": "Arif", "city": "Bandung"}

	out, err := mergeMetadata(wedding, base, map[string]any{
		"city": "Jakarta", "venue_name": "  Sasana Kriya  ", "hero_title": "   ", "bride_parents": nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["city"] != "Jakarta" || out["venue_name"] != "Sasana Kriya" {
		t.Fatalf("out = %v", out)
	}
	if _, has := out["hero_title"]; has {
		t.Fatal("an optional text field sent blank is the same as absent")
	}
	if base["city"] != "Bandung" || len(base) != 3 {
		t.Fatalf("the base map was modified: %v", base)
	}

	aq, err := mergeMetadata(spec(t, domain.EventAqiqah), nil, map[string]any{"baby_name": "R", "goat_count": float64(2)})
	if err != nil {
		t.Fatal(err)
	}
	if aq["goat_count"] != int64(2) {
		t.Fatalf("integers are stored as integers, got %#v", aq["goat_count"])
	}

	// Required keys of a type are not required of another: wedding names do not satisfy a seminar.
	_, err = mergeMetadata(spec(t, domain.EventSeminar), map[string]any{"bride_name": "x"}, map[string]any{})
	wantMessage(t, err, "invalid_metadata", "organizer_name: required")
}
