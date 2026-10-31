package http

import (
	"testing"
	"time"
)

func TestUserLimiterCountsPerKeyAndResetsAfterTheWindow(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	l := newUserLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if !l.allow("a") {
			t.Fatalf("call %d of A refused", i+1)
		}
	}
	if l.allow("a") {
		t.Error("the third call of A must be refused")
	}
	if !l.allow("b") {
		t.Error("B has its own count")
	}
	now = now.Add(61 * time.Second)
	if !l.allow("a") {
		t.Error("the window is over")
	}
	if len(l.hits) != 1 { // the sweep dropped the expired window of B
		t.Errorf("%d keys held", len(l.hits))
	}
}
