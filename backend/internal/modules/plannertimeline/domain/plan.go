package domain

import (
	"math"
	"time"
)

// Plan scales the offsets of a schedule template to one project (TEMPLATE_EVENT
// chapter 1). All DATE values are time.Time at 00:00 UTC; "today" and the floor
// are calendar dates in the project's zone, computed by the caller, so Plan
// never loads a location.
type Plan struct {
	EventDate time.Time
	Floor     time.Time // max(prep_start_date, today): no task starts before this
	K         float64   // offset multiplier
	Avail     int       // EventDate - Floor, in days
	TooShort  bool      // Avail < 3: every task is clamped to [Floor, EventDate]
	Squeezed  bool      // Avail < leadTimeDays: the schedule was compressed
}

// NewPlan returns ok == false when EventDate is before Floor: there is no
// schedule to make for the past.
//
// leadTimeDays is the standard horizon of the event type (registry); horizon is
// the R the template was written for (180 for the wedding, 60 for general).
func NewPlan(eventDate, prepStart, today time.Time, leadTimeDays, horizon int) (p Plan, ok bool) {
	floor := prepStart
	if today.After(floor) {
		floor = today
	}
	avail := int(eventDate.Sub(floor).Hours() / 24)
	if avail < 0 {
		return Plan{}, false
	}

	k := float64(leadTimeDays) / float64(horizon)
	if avail < leadTimeDays {
		k *= float64(avail) / float64(leadTimeDays)
	}

	return Plan{EventDate: eventDate, Floor: floor, K: k, Avail: avail, TooShort: avail < 3, Squeezed: avail < leadTimeDays}, true
}

// round rounds half up (floor(x + 0.5)), not to even.
func round(x float64) int { return int(math.Floor(x + 0.5)) }

// At turns the offsets of a template task (H-startOffset to H-endOffset) into dates.
func (p Plan) At(startOffset, endOffset int) (start, end time.Time) {
	if p.TooShort {
		return p.Floor, p.EventDate
	}

	day := func(n int) time.Time { return p.EventDate.AddDate(0, 0, -n) }
	start, end = day(round(float64(startOffset)*p.K)), day(round(float64(endOffset)*p.K))
	if start.Before(p.Floor) {
		start = p.Floor
	}
	if end.Before(start) {
		end = start
	}

	return start, end
}
