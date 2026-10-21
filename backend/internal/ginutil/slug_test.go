package ginutil

import "testing"

func TestSlug(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Contoh: Pernikahan Sinta & Arif", "contoh-pernikahan-sinta-arif"},
		{"Café Ünïcode", "cafe-unicode"},
		{"  --Hello,   World!!  ", "hello-world"},
		{"🎂🎉", "proyek"},
		{"", "proyek"},
		{"a\r\nb\"c", "a-b-c"},
		{"2026-10-21", "2026-10-21"},
		{"123456789012345678901234567890123456789 x", "123456789012345678901234567890123456789"}, // cut at 40, trailing "-" dropped
	}
	for _, tc := range tests {
		if got := Slug(tc.in); got != tc.want {
			t.Errorf("Slug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	long := Slug("kata-kata panjang sekali yang melebihi empat puluh karakter untuk nama berkas")
	if len(long) > maxSlugLen || long[len(long)-1] == '-' {
		t.Errorf("long slug %q (%d)", long, len(long))
	}
}
