package domain

import "testing"

func TestPersonalURL(t *testing.T) {
	got := PersonalURL("https://adicara.id/", "sinta-arif", "Bpk. Prof. Bambang S. & Istri", "0123456789abcdef0123456789abcdef")
	want := "https://adicara.id/i/sinta-arif?g=0123456789abcdef0123456789abcdef&to=Bpk.+Prof.+Bambang+S.+%26+Istri"
	if got != want {
		t.Errorf("%s", got)
	}
	// A slug with special characters cannot leave the path.
	if got := PersonalURL("https://adicara.id", "a/b?c", "X", "t"); got != "https://adicara.id/i/a%2Fb%3Fc?g=t&to=X" {
		t.Errorf("%s", got)
	}
}
