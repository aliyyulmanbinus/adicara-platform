package usecase

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// Limits (04 BE 3.4 and 2.9). The counts are product limits, not database ones, and are checked under the project
// lock so a double click cannot pass them.
const (
	maxVendors       = 200
	maxShoppingItems = 1000
	maxBudgetLines   = 300 // the limit of menu 03; a line made here counts too

	maxCategoryLabelRunes = 60
	maxVendorNameRunes    = 255
	maxPICRunes           = 255
	maxPhoneRunes         = 40
	maxURLRunes           = 500
	maxNoteRunes          = 2000
	maxItemCategoryRunes  = 60
	maxItemNameRunes      = 255
	maxUnitRunes          = 30
	maxLocationRunes      = 500
	maxLineNameRunes      = 150 // planner_budget_items.item_name
	maxLineCategoryRunes  = 60

	reasonVendorOfProject     = "must be a vendor of this project"
	reasonBudgetLineOfProject = "must be a budget line of this project"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)

// Validation messages are "<field>: <reason>" so a form can mark the field.
func invalidInput(field, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_input", field+": "+reason)
}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// cleanText trims s and enforces 1..max runes (0..max when optional) with no control characters.
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

// optionalText returns nil for an absent or blank value.
func optionalText(field string, s *string, max int) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v, err := cleanText(field, *s, max, false)
	if err != nil || v == "" {
		return nil, err
	}

	return &v, nil
}

// cleanNote is cleanText for free text: a line break is allowed (CRLF becomes LF), every other control character is not.
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
func optionalNote(field string, s *string, max int) (*string, error) {
	if s == nil {
		return nil, nil
	}
	n, err := cleanNote(field, *s, max)
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

func validateDeal(field string, v int64) error {
	if v < 1 || v > domain.MaxMoney {
		return invalidInput(field, fmt.Sprintf("must be between 1 and %d", domain.MaxMoney))
	}

	return nil
}

// cleanURL accepts only http and https links.
func cleanURL(field string, s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	u, err := url.Parse(v)
	if utf8.RuneCountInString(v) > maxURLRunes || hasControl(v) || err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, invalidInput(field, "must be an http or https URL")
	}

	return &v, nil
}

// cleanPhone keeps the number as typed (trimmed) and derives the wa.me form from it.
func cleanPhone(field string, s *string) (phone, wa *string, err error) {
	if s == nil {
		return nil, nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil, nil
	}
	if utf8.RuneCountInString(v) > maxPhoneRunes {
		return nil, nil, invalidInput(field, fmt.Sprintf("must be at most %d characters", maxPhoneRunes))
	}
	digits, perr := pd.NormalizeWA(v)
	if perr != nil {
		return nil, nil, apierr.BadRequest("invalid_phone", field+": must be an Indonesian phone number (08…, +62…)")
	}
	if digits == "" {
		return &v, nil, nil
	}

	return &v, &digits, nil
}

func normalizeScope(raw string) (string, error) {
	scope := strings.TrimSpace(raw)
	if scope == "" {
		return pd.AllScope, nil
	}
	if !keyPattern.MatchString(scope) {
		return "", errUnknownScope()
	}

	return scope, nil
}

// truncRunes cuts s to n runes (names are cut, never refused, when a vendor name becomes a budget line name).
func truncRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	return string([]rune(s)[:n])
}
