package domain

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}

	return t
}

func ymd(t time.Time) string { return t.Format("2006-01-02") }

// duration is the PDF convention (generated column): end - start, at least 1.
func duration(s, e time.Time) int { return max(int(e.Sub(s).Hours()/24), 1) }

func TestSampleProjectScalesBy175Over180(t *testing.T) {
	p, ok := NewPlan(day("2027-03-28"), day("2026-10-04"), day("2026-10-04"), 180, 180)
	if !ok || p.Avail != 175 || !p.Squeezed || p.TooShort {
		t.Fatalf("plan %+v", p)
	}

	for _, c := range []struct {
		name         string
		start, end   int
		wantS, wantE string
		wantDur      int
	}{
		{"task 1 lands on today", 180, 179, "2026-10-04", "2026-10-05", 1},
		{"task 3", 176, 174, "2026-10-08", "2026-10-10", 2},
		{"task 9", 165, 164, "2026-10-19", "2026-10-20", 1},
		{"task 16", 150, 149, "2026-11-02", "2026-11-03", 1},
		{"task 7 Lamaran is 15 Oct, not 10 Oct", 169, 168, "2026-10-15", "2026-10-16", 1},
		{"Adicara: finalisasi tamu", 45, 44, "2027-02-12", "2027-02-13", 1},
		{"Adicara: Hari-H", 0, 0, "2027-03-28", "2027-03-28", 1},
	} {
		s, e := p.At(c.start, c.end)
		if ymd(s) != c.wantS || ymd(e) != c.wantE || duration(s, e) != c.wantDur {
			t.Errorf("%s: H-%d/H-%d = %s..%s (%d days), want %s..%s (%d days)",
				c.name, c.start, c.end, ymd(s), ymd(e), duration(s, e), c.wantS, c.wantE, c.wantDur)
		}
	}
}

func TestEnoughTimeUsesTheStandardHorizon(t *testing.T) {
	p, ok := NewPlan(day("2027-12-31"), day("2026-10-04"), day("2026-10-04"), 180, 180)
	if !ok || p.K != 1 || p.Squeezed {
		t.Fatalf("more than 180 days: %+v", p)
	}
	if s, _ := p.At(180, 179); ymd(s) != "2027-07-04" {
		t.Errorf("start %s", ymd(s))
	}

	// 175 days is still less than 180: compressed, not standard.
	if p, _ := NewPlan(day("2027-03-28"), day("2026-01-01"), day("2026-10-04"), 180, 180); !p.Squeezed || p.Avail != 175 {
		t.Errorf("%+v", p)
	}
}

func TestAqiqahWithTenDaysCollapsesGladiResikToHariH(t *testing.T) {
	p, _ := NewPlan(day("2026-10-14"), day("2026-10-04"), day("2026-10-04"), 21, 60)
	if p.Avail != 10 || !p.Squeezed || p.K < 0.1666 || p.K > 0.1667 {
		t.Fatalf("%+v", p)
	}
	if s, e := p.At(1, 0); ymd(s) != "2026-10-14" || ymd(e) != "2026-10-14" || duration(s, e) != 1 {
		t.Errorf("gladi resik %s..%s", ymd(s), ymd(e))
	}
	if s, _ := p.At(60, 58); s.Before(day("2026-10-04")) {
		t.Errorf("a task in the past: %s", ymd(s))
	}
}

func TestTooShortClampsEverythingAndPastEventIsRejected(t *testing.T) {
	p, ok := NewPlan(day("2026-10-06"), day("2026-10-04"), day("2026-10-04"), 180, 180)
	if !ok || !p.TooShort {
		t.Fatalf("%+v", p)
	}
	if s, e := p.At(100, 90); ymd(s) != "2026-10-04" || ymd(e) != "2026-10-06" {
		t.Errorf("%s..%s", ymd(s), ymd(e))
	}
	if _, ok := NewPlan(day("2026-10-03"), day("2026-10-01"), day("2026-10-04"), 180, 180); ok {
		t.Error("an event in the past must be rejected")
	}
	if p, ok := NewPlan(day("2026-10-04"), day("2026-10-04"), day("2026-10-04"), 180, 180); !ok || p.Avail != 0 || !p.TooShort {
		t.Errorf("Hari-H is today: %+v", p)
	}
}

func TestPrepStartLaterThanTodayWins(t *testing.T) {
	p, _ := NewPlan(day("2027-03-28"), day("2026-12-01"), day("2026-10-04"), 180, 180)
	if ymd(p.Floor) != "2026-12-01" {
		t.Errorf("floor %s", ymd(p.Floor))
	}
	if s, _ := p.At(180, 179); s.Before(p.Floor) {
		t.Errorf("start %s before the floor", ymd(s))
	}
}

// The PDF durations survive the scaling when there is enough time, and the
// three two-day tasks (3, 4, 12) shrink to one day when it is compressed.
func TestPDFDurationsSurviveScalingOnlyWithoutCompression(t *testing.T) {
	for _, avail := range []int{175, 180} {
		p, _ := NewPlan(day("2027-03-28"), day("2027-03-28").AddDate(0, 0, -avail), day("2027-03-28").AddDate(0, 0, -avail), 180, 180)
		for i, tt := range weddingTasks {
			s, e := p.At(tt.Start, tt.End)
			if got, want := duration(s, e), max(tt.Start-tt.End, 1); got != want {
				t.Errorf("avail %d task %d: duration %d, want %d", avail, i+1, got, want)
			}
		}
	}

	p, _ := NewPlan(day("2027-03-28"), day("2026-11-28"), day("2026-11-28"), 180, 180) // 120 days
	shrunk := 0
	for _, tt := range weddingTasks {
		s, e := p.At(tt.Start, tt.End)
		if duration(s, e) < max(tt.Start-tt.End, 1) {
			shrunk++
		}
	}
	if shrunk != 3 {
		t.Errorf("with 120 days %d tasks shrink, want the 3 two-day ones", shrunk)
	}
}
