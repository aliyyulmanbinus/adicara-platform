package usecase

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

// Limits (03 BE 4.5). The counts are product limits, not database ones, and
// are checked under the project lock so a double click cannot pass them.
const (
	maxScenarios     = 3
	maxScenarioItems = 100
	maxLines         = 300
	maxContributors  = 20
	maxEntries       = 2000

	maxNameRunes     = 100
	maxItemNameRunes = 150
	maxCategoryRunes = 60
	maxUnitRunes     = 30
	maxNoteRunes     = 2000
	maxURLRunes      = 500

	windowBeforeDays = 1095 // a date may lie this long before Hari-H…
	windowAfterDays  = 365  // …and this long after the last day of the event

	defaultRole = "other"
)

var (
	scopePattern   = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
	itemKeyPattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
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
// LF), every other control character is not.
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

// optionalNote returns nil for an absent or blank note.
func optionalNote(field string, s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	n, err := cleanNote(field, *s, maxNoteRunes)
	if err != nil || n == "" {
		return nil, err
	}

	return &n, nil
}

func validateMoney(field string, v int64) error {
	if v < 0 || v > domain.MaxMoney {
		return invalidInput(field, fmt.Sprintf("must be between 0 and %d", domain.MaxMoney))
	}

	return nil
}

func validateTotal(field string, v int64) error {
	if v < 1 || v > domain.MaxMoney {
		return invalidInput(field, fmt.Sprintf("must be between 1 and %d", domain.MaxMoney))
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

// checkWindow keeps a payment or deposit date inside Hari-H − 1095 days … the
// last day + 365 days (the window of menu 02). A date in the future is allowed.
func checkWindow(p pd.Project, field string, d time.Time) error {
	lo, hi := p.EventDate.AddDate(0, 0, -windowBeforeDays), p.LastDay().AddDate(0, 0, windowAfterDays)
	if d.Before(lo) || d.After(hi) {
		return invalidDate(field, "must be within the project schedule window")
	}

	return nil
}

func normalizeScope(raw string) (string, error) {
	scope := strings.TrimSpace(raw)
	if scope == "" {
		return pd.AllScope, nil
	}
	if !scopePattern.MatchString(scope) {
		return "", errUnknownScope()
	}

	return scope, nil
}

// cleanKey validates an optional item_key; blank means "no key".
func cleanKey(field string, k *string) (*string, error) {
	if k == nil {
		return nil, nil
	}
	key := strings.TrimSpace(*k)
	if key == "" {
		return nil, nil
	}
	if !itemKeyPattern.MatchString(key) {
		return nil, invalidInput(field, "must match ^[a-z0-9_]{1,40}$")
	}

	return &key, nil
}

// cleanReceiptURL accepts only http and https links.
func cleanReceiptURL(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	u, err := url.Parse(v)
	if utf8.RuneCountInString(v) > maxURLRunes || hasControl(v) || err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, invalidInput("receipt_url", "must be an http or https URL")
	}

	return &v, nil
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

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}

	return false
}
