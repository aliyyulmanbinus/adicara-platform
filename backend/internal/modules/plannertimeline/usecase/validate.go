package usecase

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Limits (02 BE 3.4). The counts are product limits, not database ones: 500
// tasks and 200 documents are about 370 kB and 114 kB of JSON, which the
// unpaginated screens can still draw.
const (
	maxTasks = 500
	maxDocs  = 200

	maxTitleRunes    = 255
	maxCategoryRunes = 60
	maxPhaseRunes    = 100
	maxPICRunes      = 100
	maxQuantityRunes = 100
	maxNoteRunes     = 2000
	minPhaseOrder    = 1
	maxPhaseOrder    = 20
	maxMoney         = 10_000_000_000_000 // 10^13, as in menu 01

	windowBeforeDays = 1095 // a task may start this long before Hari-H…
	windowAfterDays  = 365  // …and end this long after the last day of the event
	maxShiftDays     = 1826

	defaultCategory = "Umum"
)

var scopePattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)

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
// control characters.
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

// cleanNote is cleanText for free text: a line break is allowed (CRLF becomes
// LF), every other control character is not. Menu 01's cleanText cannot be
// reused because unicode.IsControl('\n') is true.
func cleanNote(field, s string, max int) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if utf8.RuneCountInString(s) > max {
		return "", invalidInput(field, fmt.Sprintf("must be at most %d characters", max))
	}
	if strings.IndexFunc(s, func(r rune) bool { return r != '\n' && unicode.IsControl(r) }) >= 0 {
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

func parseDate(field, v string) (time.Time, error) {
	t, err := time.Parse(pd.DateLayout, strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, invalidDate(field, "must be a valid date in YYYY-MM-DD format")
	}

	return t, nil
}

// scheduleWindow is the range of dates a task may use: from 1095 days before
// Hari-H to 365 days after the last day of the event. Without it a typo of the
// year would stretch the Gantt over centuries.
func scheduleWindow(p pd.Project) (lo, hi time.Time) {
	return p.EventDate.AddDate(0, 0, -windowBeforeDays), p.LastDay().AddDate(0, 0, windowAfterDays)
}

func checkWindow(p pd.Project, field string, d time.Time) error {
	lo, hi := scheduleWindow(p)
	if d.Before(lo) || d.After(hi) {
		return invalidDate(field, "must be within the project schedule window")
	}

	return nil
}

// checkIDs validates the shape of a reorder list: every id of the project
// exactly once. The repository then confirms that each id belongs to the project.
func checkIDs(ids []uuid.UUID, total int) error {
	if len(ids) != total {
		return invalidInput("ids", "must list every id exactly once")
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return invalidInput("ids", "must list every id exactly once")
		}
		seen[id] = true
	}

	return nil
}
