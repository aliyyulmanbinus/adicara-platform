package usecase

import (
	"strings"
	"testing"
)

func TestCleanText(t *testing.T) {
	if got, err := cleanText("f", "  Ijab Kabul  ", 255, true); err != nil || got != "Ijab Kabul" {
		t.Errorf("trim: %q, %v", got, err)
	}
	if got, err := cleanText("f", "", 10, false); err != nil || got != "" {
		t.Errorf("optional empty: %q, %v", got, err)
	}
	wantAPIError(t, errOnly(cleanText("f", " ", 10, true)), 400, "invalid_input", "f: required")
	wantAPIError(t, errOnly(cleanText("f", strings.Repeat("é", 11), 10, true)), 400, "invalid_input", "f: must be at most 10 characters")
	if _, err := cleanText("f", strings.Repeat("é", 10), 10, true); err != nil { // 20 bytes, 10 runes
		t.Errorf("runes are counted, not bytes: %v", err)
	}
	for _, bad := range []string{"a\x07b", "a\nb", "a\tb", "a\u0085b"} {
		wantAPIError(t, errOnly(cleanText("f", bad, 10, true)), 400, "invalid_input", "f: must not contain control characters")
	}
}

func errOnly(_ string, err error) error { return err }

func TestCleanNote(t *testing.T) {
	if got, err := cleanNote("f", "  a\r\nb\n\nc  ", 20); err != nil || got != "a\nb\n\nc" {
		t.Errorf("note: %q, %v", got, err)
	}
	wantAPIError(t, errOnly(cleanNote("f", strings.Repeat("a", 21), 20)), 400, "invalid_input", "f: must be at most 20 characters")
	wantAPIError(t, errOnly(cleanNote("f", "a\x07b", 20)), 400, "invalid_input", "f: must not contain control characters")
	wantAPIError(t, errOnly(cleanNote("f", "a\tb", 20)), 400, "invalid_input", "f: must not contain control characters")

	if got, err := optionalNote("f", nil, 20); got != nil || err != nil {
		t.Errorf("nil note: %v, %v", got, err)
	}
	blank := "  \n "
	if got, err := optionalNote("f", &blank, 20); got != nil || err != nil {
		t.Errorf("blank note: %v, %v", got, err)
	}
	long := strings.Repeat("a", 21)
	if _, err := optionalNote("f", &long, 20); err == nil {
		t.Error("a long note was accepted")
	}
	ok := " x "
	if got, err := optionalNote("f", &ok, 20); err != nil || got == nil || *got != "x" {
		t.Errorf("note: %v, %v", got, err)
	}
}

func TestCleanURL(t *testing.T) {
	ok := []string{"https://open.spotify.com/track/1", "http://x.com", " https://youtu.be/x?t=1 "}
	for _, in := range ok {
		in := in
		got, err := cleanURL("u", &in)
		if err != nil || got == nil || *got != strings.TrimSpace(in) {
			t.Errorf("cleanURL(%q) = %v, %v", in, got, err)
		}
	}
	for _, in := range []string{"ftp://x.com", "javascript:alert(1)", "//x.com", "x.com", "https://", "https:///a", "https://x.com/\x07", strings.Repeat("a", 2001)} {
		in := in
		_, err := cleanURL("u", &in)
		wantAPIError(t, err, 400, "invalid_input", "u: must be an http or https URL")
	}
	blank := "  "
	if got, err := cleanURL("u", &blank); got != nil || err != nil {
		t.Errorf("blank: %v, %v", got, err)
	}
	if got, err := cleanURL("u", nil); got != nil || err != nil {
		t.Errorf("nil: %v, %v", got, err)
	}
}
