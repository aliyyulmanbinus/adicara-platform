package usecase

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

const (
	defaultTimezone   = "Asia/Jakarta"
	maxTitleRunes     = 255
	maxMoney          = 10_000_000_000_000 // 10^13: safe for a JavaScript Number and thousands of summed rows
	maxEventAheadDays = 1826               // 5 years
	maxEventSpanDays  = 30
	minClientToken    = 8
	maxClientToken    = 64

	maxSessions        = 20 // non-system sessions per project
	maxSessionName     = 100
	maxIconRunes       = 8
	maxLocationRunes   = 150
	defaultSessionIcon = "📋"

	maxSkippedCategories = 50
)

// Same shape as the CHECK on planner_projects.timezone. Existence is checked
// twice: time.LoadLocation here and the planner_check_timezone trigger in PostgreSQL.
var tzPattern = regexp.MustCompile(`^(UTC|(Africa|America|Antarctica|Arctic|Asia|Atlantic|Australia|Europe|Indian|Pacific)/[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)?)$`)

var (
	clockPattern    = regexp.MustCompile(`^([01]?\d|2[0-3])[:.]([0-5]\d)$`)
	categoryPattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
)

// Validation messages are "<field>: <reason>" so a form can mark the field.
func invalidInput(field, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_input", field+": "+reason)
}

func invalidDate(field, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_date", field+": "+reason)
}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// cleanText trims s and enforces 1..max runes (0..max when optional) with no
// control characters. A column CHECK would only fail late with a 500.
func cleanText(field, s string, max int, required bool) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "" && required:
		return "", invalidInput(field, "must not be empty")
	case utf8.RuneCountInString(s) > max:
		return "", invalidInput(field, fmt.Sprintf("must be at most %d characters", max))
	case hasControl(s):
		return "", invalidInput(field, "must not contain control characters")
	}

	return s, nil
}

func validateMoney(field string, v int64) error {
	if v < 0 || v > maxMoney {
		return invalidInput(field, fmt.Sprintf("must be between 0 and %d", int64(maxMoney)))
	}

	return nil
}

// resolveTimezone returns the zone name (default Asia/Jakarta) and its location.
func resolveTimezone(raw *string) (string, *time.Location, error) {
	tz := defaultTimezone
	if raw != nil {
		tz = *raw
	}
	if !tzPattern.MatchString(tz) {
		return "", nil, apierr.BadRequest("invalid_timezone", "timezone: must be a canonical IANA name such as Asia/Jakarta")
	}
	loc, err := domain.LoadLocation(tz)
	if err != nil {
		return "", nil, apierr.BadRequest("invalid_timezone", "timezone: unknown time zone")
	}

	return tz, loc, nil
}

func parseDate(field, v string) (time.Time, error) {
	t, err := time.Parse(domain.DateLayout, v)
	if err != nil {
		return time.Time{}, invalidDate(field, "must be a valid date in YYYY-MM-DD format")
	}

	return t, nil
}

func parseDatePtr(field string, v *string) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}
	t, err := parseDate(field, *v)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// checkSchedule validates the relations between a project's dates.
func checkSchedule(prep, event time.Time, end *time.Time) error {
	if prep.After(event) {
		return invalidDate("prep_start_date", "must not be after event_date")
	}
	if end != nil {
		if end.Before(event) {
			return invalidDate("event_end_date", "must not be before event_date")
		}
		if end.After(event.AddDate(0, 0, maxEventSpanDays)) {
			return invalidDate("event_end_date", fmt.Sprintf("must be within %d days of event_date", maxEventSpanDays))
		}
	}

	return nil
}

func validateClientToken(v *string) error {
	if v == nil {
		return nil
	}
	if n := len(*v); n < minClientToken || n > maxClientToken {
		return invalidInput("client_token", fmt.Sprintf("must be %d to %d characters", minClientToken, maxClientToken))
	}

	return nil
}

func validateSkippedCategories(in []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, c := range in {
		if !categoryPattern.MatchString(c) {
			return nil, invalidInput("skipped_vendor_categories", "each key must be 1 to 40 characters of a-z, 0-9 or _")
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	if len(out) > maxSkippedCategories {
		return nil, invalidInput("skipped_vendor_categories", fmt.Sprintf("at most %d categories", maxSkippedCategories))
	}

	return out, nil
}

// ---------- sessions ----------

type sessionFields struct {
	name, icon                string
	date                      *time.Time
	startTime, endTime, place *string
}

func validateSessionFields(req domain.CreateSessionRequest) (sessionFields, error) {
	var f sessionFields
	var err error

	if f.name, err = cleanText("name", req.Name, maxSessionName, true); err != nil {
		return f, err
	}

	f.icon = defaultSessionIcon
	if req.Icon != nil && strings.TrimSpace(*req.Icon) != "" {
		icon := strings.TrimSpace(*req.Icon)
		if n := utf8.RuneCountInString(icon); n > maxIconRunes || hasControl(icon) {
			return f, invalidInput("icon", fmt.Sprintf("must be 1 to %d characters without control characters", maxIconRunes))
		}
		f.icon = icon
	}

	if req.SessionDate != nil && strings.TrimSpace(*req.SessionDate) != "" {
		d, err := parseDate("session_date", strings.TrimSpace(*req.SessionDate))
		if err != nil {
			return f, err
		}
		f.date = &d
	}

	if f.startTime, err = parseClock("start_time", req.StartTime); err != nil {
		return f, err
	}
	if f.endTime, err = parseClock("end_time", req.EndTime); err != nil {
		return f, err
	}

	if req.LocationName != nil {
		place, err := cleanText("location_name", *req.LocationName, maxLocationRunes, false)
		if err != nil {
			return f, err
		}
		if place != "" {
			f.place = &place
		}
	}

	return f, nil
}

// parseClock accepts "HH:MM" and "HH.MM" and returns "HH:MM". Blank means unset.
func parseClock(field string, v *string) (*string, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	m := clockPattern.FindStringSubmatch(strings.TrimSpace(*v))
	if m == nil {
		return nil, invalidInput(field, `must be a time such as "07:30" or "07.30"`)
	}
	var h, min int
	_, _ = fmt.Sscanf(m[1], "%d", &h)
	_, _ = fmt.Sscanf(m[2], "%d", &min)
	out := fmt.Sprintf("%02d:%02d", h, min)

	return &out, nil
}
