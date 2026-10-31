package domain

import "testing"

func TestParseClock(t *testing.T) {
	ok := map[string][2]any{
		"07.46": {466, "07:46"}, "7.46": {466, "07:46"}, "07:46": {466, "07:46"}, "7:46": {466, "07:46"}, " 07:46 ": {466, "07:46"},
		"00:00": {0, "00:00"}, "0.00": {0, "00:00"}, "23:59": {1439, "23:59"}, "12.05": {725, "12:05"}, "19:30": {1170, "19:30"},
	}
	for in, want := range ok {
		m, s, good := ParseClock(in)
		if !good || m != want[0] || s != want[1] {
			t.Errorf("ParseClock(%q) = %d %q %v, want %v", in, m, s, good, want)
		}
	}
	for _, in := range []string{"", "24:00", "24.00", "7.6", "07.6", "07:46:00", "07:60", "-1:00", "ab:cd", "7", "0746", "07 46", "07:4", "1:2:3", "07.466"} {
		if _, _, good := ParseClock(in); good {
			t.Errorf("ParseClock(%q) was accepted", in)
		}
	}
}

func ptr(s string) *string { return &s }

// The table of 06 BE 2.1: only the first refusal is also a CHECK of the database.
func TestValidateTimes(t *testing.T) {
	cases := []struct {
		name       string
		start      string
		end        *string
		nextDay    bool
		field, why string // empty = valid
		wantEnd    string
		wantDur    int
		open       bool
	}{
		{name: "closed", start: "7.46", end: ptr("07:50"), wantEnd: "07:50", wantDur: 4},
		{name: "open", start: "10.40", end: nil, open: true},
		{name: "blank end is open", start: "10.40", end: ptr("  "), open: true},
		{name: "marker of zero minutes", start: "09:00", end: ptr("09.00"), wantEnd: "09:00", wantDur: 0},
		{name: "from midnight to 23:59", start: "00:00", end: ptr("23:59"), wantEnd: "23:59", wantDur: 1439},
		{name: "across midnight", start: "23:30", end: ptr("00:30"), nextDay: true, wantEnd: "00:30", wantDur: 60},
		{name: "across midnight, one minute", start: "23:59", end: ptr("00:00"), nextDay: true, wantEnd: "00:00", wantDur: 1},
		{name: "bad start", start: "7.6", field: "start_time", why: timeReason},
		{name: "bad end", start: "07:00", end: ptr("24:00"), field: "end_time", why: timeReason},
		{name: "end before start", start: "08:00", end: ptr("07:59"), field: "end_time", why: "must not be before start_time unless ends_next_day is set"},
		{name: "next day, end after start", start: "22:00", end: ptr("23:00"), nextDay: true, field: "end_time", why: "must be earlier than start_time when ends_next_day is set"},
		{name: "next day, end equal", start: "22:00", end: ptr("22:00"), nextDay: true, field: "end_time", why: "must be earlier than start_time when ends_next_day is set"},
		{name: "next day without end", start: "22:00", end: nil, nextDay: true, field: "ends_next_day", why: "requires end_time"},
		{name: "next day with blank end", start: "22:00", end: ptr(""), nextDay: true, field: "ends_next_day", why: "requires end_time"},
	}
	for _, c := range cases {
		got, err := ValidateTimes(c.start, c.end, c.nextDay)
		if c.field != "" {
			if err == nil || err.Field != c.field || err.Reason != c.why {
				t.Errorf("%s: error = %v, want %s: %s", c.name, err, c.field, c.why)
			}

			continue
		}
		if err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
			continue
		}
		if c.open != (got.End == nil) || (!c.open && *got.End != c.wantEnd) || got.EndsNextDay != c.nextDay {
			t.Errorf("%s: times = %+v", c.name, got)
		}
		d, has := got.Duration()
		if has == c.open || (has && d != c.wantDur) {
			t.Errorf("%s: duration = %d %v, want %d", c.name, d, has, c.wantDur)
		}
	}
}

func TestDotClock(t *testing.T) {
	if got := DotClock("07:46"); got != "07.46" {
		t.Errorf("DotClock = %q", got)
	}
}

func TestFieldErrorReadsLikeTheAPIMessage(t *testing.T) {
	_, err := ValidateTimes("7.6", nil, false)
	if err == nil || err.Error() != "start_time: must be a time like 07:46 or 07.46" {
		t.Errorf("error = %v", err)
	}
}
