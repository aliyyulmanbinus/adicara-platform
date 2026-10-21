package domain

import "time"

// LoadLocation resolves an IANA zone name. It needs zoneinfo at runtime
// (cmd/api imports time/tzdata for that).
func LoadLocation(tz string) (*time.Location, error) {
	return time.LoadLocation(tz)
}

// CalendarDay returns the calendar date of now in loc as a DATE value
// (00:00 UTC). Truncating now to 24h would cut at the UTC boundary and be one
// day off between 00:00 and 06:59 WIB.
func CalendarDay(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DaysBetween is the number of calendar days from one DATE value (00:00 UTC) to another.
func DaysBetween(from, to time.Time) int {
	return int(to.Sub(from) / (24 * time.Hour))
}

// DaysUntil is the number of calendar days from today (in zone tz) to
// eventDate; negative once the date has passed.
func DaysUntil(now time.Time, tz string, eventDate time.Time) (int, error) {
	loc, err := LoadLocation(tz)
	if err != nil {
		return 0, err
	}
	ey, em, ed := eventDate.Date()

	return DaysBetween(CalendarDay(now, loc), time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC)), nil
}

// ComputePhase is the one definition of an event's phase. The Hub SQL
// (repository.ListCards) mirrors it; both are tested at midnight boundaries.
//
//	today <  event_date                → upcoming
//	event_date <= today <= end         → ongoing, day_number = today − event_date + 1
//	today >  end                       → finished
//
// where end = COALESCE(event_end_date, event_date).
func ComputePhase(now time.Time, loc *time.Location, eventDate time.Time, eventEnd *time.Time) PhaseInfo {
	today := CalendarDay(now, loc)
	end := eventDate
	if eventEnd != nil {
		end = *eventEnd
	}

	info := PhaseInfo{DaysRemaining: DaysBetween(today, eventDate), TotalDays: DaysBetween(eventDate, end) + 1}
	switch {
	case today.Before(eventDate):
		info.Phase = PhaseUpcoming
	case !today.After(end):
		info.Phase = PhaseOngoing
		info.DayNumber = DaysBetween(eventDate, today) + 1
	default:
		info.Phase = PhaseFinished
	}

	return info
}
