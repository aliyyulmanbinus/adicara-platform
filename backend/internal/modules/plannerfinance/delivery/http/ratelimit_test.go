package http

import (
	"testing"
	"time"
)

// The limiter follows its own clock only: the sweep clock is not started from
// the wall clock, so the test passes whatever day it runs on (the first copies
// of this limiter did not, and failed once the real date passed the fixed one).
func TestUserLimiterCountsPerKeyAndResetsAfterTheWindow(t *testing.T) {
	for _, start := range []time.Time{
		time.Date(2000, 1, 1, 10, 0, 0, 0, time.UTC), time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC), time.Date(2100, 1, 1, 10, 0, 0, 0, time.UTC),
	} {
		now := start
		l := newUserLimiter(3, time.Minute)
		l.now = func() time.Time { return now }

		for i := 0; i < 3; i++ {
			if !l.allow("alice") {
				t.Fatalf("%v: request %d of alice must pass", start.Year(), i+1)
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
			t.Fatalf("%v: expired counters must be dropped", start.Year())
		}
	}
}

func TestUserLimiterWithNoLimitLetsEverythingThrough(t *testing.T) {
	l := newUserLimiter(0, time.Minute)
	if l.limit > 0 {
		t.Fatal("a limit of 0 disables the middleware")
	}
}
