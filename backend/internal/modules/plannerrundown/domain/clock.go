package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FieldError is a rule broken by one field of a body; the usecase answers it as 400 invalid_input "<field>: <reason>".
type FieldError struct{ Field, Reason string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Reason }

// clockPattern is what the server accepts after trim (06 BE 2.1): 07.46, 7.46, 07:46 and 7:46. Seconds, 24:00 and 07.6
// are refused; the column is a TIME and would take seconds, so the limit belongs here.
var clockPattern = regexp.MustCompile(`^([01]?[0-9]|2[0-3])[:.]([0-5][0-9])$`)

// ParseClock reads a time of day into minutes since midnight and its stored form "HH:MM".
func ParseClock(s string) (minutes int, hhmm string, ok bool) {
	m := clockPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, "", false
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])

	return h*60 + mi, fmt.Sprintf("%02d:%02d", h, mi), true
}

// DotClock is the way the UI and the WhatsApp text write a time: "07:46" → "07.46" (as the PDF does).
func DotClock(hhmm string) string { return strings.Replace(hhmm, ":", ".", 1) }

// Times is the clock part of an agenda after validation: both times in "HH:MM", End nil for an open agenda.
type Times struct {
	Start       string
	End         *string
	EndsNextDay bool
}

const timeReason = "must be a time like 07:46 or 07.46"

// ValidateTimes applies the rules of 06 BE 2.1. Only "end before start without ends_next_day" is also guarded by the
// database (a CHECK); the rest would be stored, so they are refused here:
//
//   - no end_time: an open agenda ("10.40 - selesai"); ends_next_day must be false.
//   - end ≥ start without ends_next_day: valid (the same time is a 0-minute marker).
//   - end < start without ends_next_day: refused.
//   - ends_next_day: end must be earlier than start (23.30 → 00.30 is 60 minutes); the same or later is refused.
func ValidateTimes(start string, end *string, endsNextDay bool) (Times, *FieldError) {
	startMin, startHHMM, ok := ParseClock(start)
	if !ok {
		return Times{}, &FieldError{"start_time", timeReason}
	}
	t := Times{Start: startHHMM, EndsNextDay: endsNextDay}

	if end == nil || strings.TrimSpace(*end) == "" {
		if endsNextDay {
			return Times{}, &FieldError{"ends_next_day", "requires end_time"}
		}

		return t, nil
	}
	endMin, endHHMM, ok := ParseClock(*end)
	if !ok {
		return Times{}, &FieldError{"end_time", timeReason}
	}
	switch {
	case endsNextDay && endMin >= startMin:
		return Times{}, &FieldError{"end_time", "must be earlier than start_time when ends_next_day is set"}
	case !endsNextDay && endMin < startMin:
		return Times{}, &FieldError{"end_time", "must not be before start_time unless ends_next_day is set"}
	}
	t.End = &endHHMM

	return t, nil
}

// Duration is the length of an agenda in minutes (the generated column of the table): the difference of the two times,
// plus a day when it ends the next day. Open agendas have none.
func (t Times) Duration() (int, bool) {
	if t.End == nil {
		return 0, false
	}
	s, _, _ := ParseClock(t.Start)
	e, _, _ := ParseClock(*t.End)
	d := e - s
	if t.EndsNextDay {
		d += 1440
	}

	return d, true
}
