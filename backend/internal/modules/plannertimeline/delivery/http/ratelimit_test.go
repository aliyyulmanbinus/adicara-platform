package http

import (
	"testing"
	"time"
)

func TestUserLimiterCountsPerKeyAndResetsAfterTheWindow(t *testing.T) {
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	l := newUserLimiter(3, time.Minute)
	l.now = func() time.Time { return now }
	l.lastSweep = now // newUserLimiter starts the sweep clock at the wall-clock time; the test clock must start it too

	for i := 0; i < 3; i++ {
		if !l.allow("alice") {
			t.Fatalf("request %d of alice must pass", i+1)
		}
	}
	if l.allow("alice") {
		t.Fatal("the 4th request inside the window must be refused")
	}
	if !l.allow("bob") {
		t.Fatal("another user has their own counter")
	}

	now = now.Add(59 * time.Second)
	if l.allow("alice") {
		t.Fatal("still inside the window")
	}
	now = now.Add(2 * time.Second)
	if !l.allow("alice") {
		t.Fatal("a new window must start after it elapsed")
	}

	// Stale entries are swept so the map cannot grow without bound.
	now = now.Add(5 * time.Minute)
	l.allow("carol")
	if _, kept := l.hits["bob"]; kept {
		t.Fatal("expired counters must be dropped")
	}
}
