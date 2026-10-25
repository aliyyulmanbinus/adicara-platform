package domain

import (
	"math"
	"testing"
)

func TestRecommendedPortions(t *testing.T) {
	cases := []struct{ confirmed, pct, pax, want int }{
		{27, 25, 0, 34},   // sample project: 27 x 1.25 = 33.75
		{200, 10, 0, 220}, // float64: ceil(200 x 1.1) = 221
		{10, 10, 5, 16},
		{0, 25, 0, 0},
		{0, 25, 10, 10},
		{1, 0, 0, 1},
		{4, 100, 0, 8},
		{3, 33, 0, 4}, // 3.99
		{100000, 100, 1000, 201000},
	}
	for _, c := range cases {
		got, err := RecommendedPortions(c.confirmed, c.pct, c.pax)
		if err != nil || got != c.want {
			t.Errorf("RecommendedPortions(%d,%d,%d) = %d,%v; want %d", c.confirmed, c.pct, c.pax, got, err, c.want)
		}
	}
	for _, bad := range [][3]int{{-1, 25, 0}, {10, -1, 0}, {10, 101, 0}, {10, 25, -1}, {10, 25, 1001}} {
		if _, err := RecommendedPortions(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("RecommendedPortions(%v) must fail", bad)
		}
	}
}

// Jawab dengan bilangan bulat selalu sama dengan ceil rasional; float64 tidak.
func TestRecommendedPortionsNeverDiffersFromExactCeil(t *testing.T) {
	floatWrong := 0
	for c := 0; c <= 3000; c++ {
		for p := 0; p <= 100; p++ {
			got, _ := RecommendedPortions(c, p, 0)
			exactNum := c * (100 + p)
			want := exactNum / 100
			if exactNum%100 != 0 {
				want++
			}
			if got != want {
				t.Fatalf("(%d,%d): %d != %d", c, p, got, want)
			}
			if int(math.Ceil(float64(c)*(1+float64(p)/100))) != want {
				floatWrong++
			}
		}
	}
	if floatWrong == 0 {
		t.Fatal("expected float64 to be wrong somewhere; the reason for integer percent has vanished")
	}
	t.Logf("float64 gives a different answer in %d of %d (guests, percent) pairs", floatWrong, 3001*101)
}
