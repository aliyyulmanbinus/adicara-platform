package domain

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// The vectors are shared with the frontend (04 FE normalizeWA) and with menu 05, so a rule change shows up on both sides.
const waVectors = "../../../../../docs/APP_SPEC/PLANNER_SPEC/04_VENDOR_DAN_BELANJA/fixtures/planner-wa-vectors.json"

func TestNormalizeWAMatchesTheSharedVectors(t *testing.T) {
	raw, err := os.ReadFile(waVectors)
	if err != nil {
		t.Skipf("vectors not available: %v", err)
	}
	var vectors []struct {
		Input string  `json:"input"`
		WA    *string `json:"wa"` // null = invalid_phone
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) < 20 {
		t.Fatalf("only %d vectors", len(vectors))
	}

	for _, v := range vectors {
		got, err := NormalizeWA(v.Input)
		switch {
		case v.WA == nil && !errors.Is(err, ErrInvalidPhone):
			t.Errorf("NormalizeWA(%q) = %q, %v; want ErrInvalidPhone", v.Input, got, err)
		case v.WA != nil && (err != nil || got != *v.WA):
			t.Errorf("NormalizeWA(%q) = %q, %v; want %q", v.Input, got, err, *v.WA)
		}
	}
}

func TestNormalizeWAEdges(t *testing.T) {
	for in, want := range map[string]string{
		"":                   "",
		"   ":                "",
		"081298765432":       "6281298765432",
		"+62 0812 9876 5432": "6281298765432",
		"0062 812 9876 5432": "6281298765432",
	} {
		if got, err := NormalizeWA(in); err != nil || got != want {
			t.Errorf("NormalizeWA(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"8129876543", "+65 9123 4567", "0812abc", "08123456", "+6281298765432123456", "62+81298765432", "０８１２９８７６５４３２"} {
		if got, err := NormalizeWA(in); !errors.Is(err, ErrInvalidPhone) {
			t.Errorf("NormalizeWA(%q) = %q, %v; want ErrInvalidPhone", in, got, err)
		}
	}
}
