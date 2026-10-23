package domain

import (
	"strings"
	"testing"
)

func TestCategoryKeyFromLabel(t *testing.T) {
	for in, want := range map[string]string{
		"Photobooth":           "photobooth",
		"Sewa Mobil Pengantin": "sewa_mobil_pengantin",
		"  Venue  ":            "venue",
		"Dekorasi & Tenda":     "dekorasi_tenda",
		"Café Étoile":          "cafe_etoile",
		"🎉🎉":                   "kategori",
		"":                     "kategori",
		"ツ":                    "kategori",
		"A---B___C":            "a_b_c",
		"Hand Bouquet":         "hand_bouquet",
		"Satu dua tiga empat lima enam tujuh delapan sembilan": "satu_dua_tiga_empat_lima_enam_tujuh_dela",
	} {
		if got := CategoryKeyFromLabel(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
	// the cut never ends in "_" and the result always fits the column
	for _, in := range []string{strings.Repeat("ab ", 30), strings.Repeat("a", 80)} {
		if got := CategoryKeyFromLabel(in); len(got) > MaxCategoryKeyLen || strings.HasSuffix(got, "_") {
			t.Errorf("%q: %q", in, got)
		}
	}
}
