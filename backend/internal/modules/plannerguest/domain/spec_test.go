package domain

import "testing"

func TestResolveSide(t *testing.T) {
	specs := specsFromTemplateEvent(t)
	w, sem, bd := specs["wedding"], specs["seminar"], specs["birthday"]
	cases := []struct {
		spec  GuestSpec
		in    string
		want  string
		known bool
	}{
		{w, "pria", "pria", true},
		{w, "  PRIA ", "pria", true},
		{w, "Pihak Pria", "pria", true},
		{w, "pihak   pria (cpp)", "pria", true},
		{w, "Pihak Wanita (CPW)", "wanita", true},
		{w, "", "", true},
		{w, "   ", "", true},
		{w, "laki-laki", "", false},
		{sem, "Peserta Eksternal", "eksternal", true},
		{sem, "panitia/internal", "internal", true},
		{sem, "pria", "", false},
		{bd, "", "", true},
		{bd, "pria", "", false}, // jenis tanpa sisi menolak isian apa pun
	}
	for _, c := range cases {
		got, ok := c.spec.ResolveSide(c.in)
		if got != c.want || ok != c.known {
			t.Errorf("ResolveSide(%q) = %q,%v; want %q,%v", c.in, got, ok, c.want, c.known)
		}
	}
}

func TestResolveCategory(t *testing.T) {
	specs := specsFromTemplateEvent(t)
	w, sem, oth := specs["wedding"], specs["seminar"], specs["other"]
	cases := []struct {
		spec GuestSpec
		in   string
		want string
	}{
		{w, "teman", "teman"},
		{w, "Teman Orang Tua", "teman_ortu"},
		{w, "TEMAN_ORTU", "teman_ortu"},
		{w, "vip", "vip"},
		{w, "", "lainnya"},
		{w, "Lainnya", "lainnya"},
		{w, "  Kolega   Kantor ", "Kolega Kantor"}, // kustom: spasi dilipat, huruf dipertahankan
		{w, "teman ortu", "teman ortu"},            // bukan label registry: kustom
		{sem, "Peserta", "peserta"},
		{oth, "umum", "umum"},
	}
	for _, c := range cases {
		if got := c.spec.ResolveCategory(c.in); got != c.want {
			t.Errorf("ResolveCategory(%q) = %q; want %q", c.in, got, c.want)
		}
	}
	if got := w.CategoryLabel("teman_ortu"); got != "Teman Orang Tua" {
		t.Errorf("label = %q", got)
	}
	if got := w.CategoryLabel("lainnya"); got != "Lainnya" {
		t.Errorf("label = %q", got)
	}
	if got := w.CategoryLabel("Kolega Kantor"); got != "Kolega Kantor" {
		t.Errorf("label = %q", got)
	}
}
