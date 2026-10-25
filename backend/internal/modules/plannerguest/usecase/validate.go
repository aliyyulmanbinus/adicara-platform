package usecase

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// Limits (05 BE 3.4 and 2.9). The counts are product limits, not database ones; the guest count is checked under the
// project lock so two requests cannot both pass it.
const (
	maxGuests = domain.MaxGuestsInProject

	maxNameRunes     = 255
	maxCategoryRunes = 60
	maxTableRunes    = 30
	maxPhoneRunes    = 40
	maxNoteRunes     = 2000
	maxSearchRunes   = 100

	minPax = 1
	maxPax = 20

	defaultListLimit, maxListLimit           = 100, 500
	defaultReceptionLimit, maxReceptionLimit = 50, 200
	recentCheckins                           = 10
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)

// Validation messages are "<field>: <reason>" so a form can mark the field.
func invalidInput(field, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_input", field+": "+reason)
}

func atMost(field string, n int) *apierr.Error {
	return invalidInput(field, fmt.Sprintf("must be at most %d characters", n))
}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// squash trims s and folds every run of white space (tabs and line breaks too) into one space.
func squash(s string) string { return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ") }

// cleanName is a required one-line text; double spaces are folded.
func cleanName(field, s string, max int) (string, error) {
	s = squash(s)
	switch {
	case s == "":
		return "", invalidInput(field, "required")
	case utf8.RuneCountInString(s) > max:
		return "", atMost(field, max)
	case hasControl(s):
		return "", invalidInput(field, "must not contain control characters")
	}

	return s, nil
}

// optionalLine is an optional one-line text; nil for an absent or blank value.
func optionalLine(field string, s *string, max int) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := squash(*s)
	switch {
	case v == "":
		return nil, nil
	case utf8.RuneCountInString(v) > max:
		return nil, atMost(field, max)
	case hasControl(v):
		return nil, invalidInput(field, "must not contain control characters")
	}

	return &v, nil
}

// optionalNote is free text where a line break is allowed (CRLF becomes LF); nil for an absent or blank note.
func optionalNote(field string, s *string, max int) (*string, error) {
	if s == nil {
		return nil, nil
	}
	n := strings.TrimSpace(strings.ReplaceAll(*s, "\r\n", "\n"))
	if n == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(n) > max {
		return nil, atMost(field, max)
	}
	if strings.IndexFunc(n, func(r rune) bool { return r != '\n' && unicode.IsControl(r) }) >= 0 {
		return nil, invalidInput(field, "must not contain control characters")
	}

	return &n, nil
}

// cleanSide resolves the side of a guest to a registry key. nil = not decided yet. A type without sides refuses any value.
func cleanSide(spec domain.GuestSpec, s *string) (*string, error) {
	if s == nil || squash(*s) == "" {
		return nil, nil
	}
	if len(spec.Sides) == 0 {
		return nil, invalidInput("guest_side", "must be empty for an event type without sides")
	}
	key, ok := spec.ResolveSide(*s)
	if !ok {
		return nil, invalidInput("guest_side", "must be a registry side key")
	}

	return &key, nil
}

// cleanCategory resolves a registry key or label; any other text is a free-text category.
func cleanCategory(spec domain.GuestSpec, s string) (string, error) {
	c := spec.ResolveCategory(s)
	switch {
	case utf8.RuneCountInString(c) > maxCategoryRunes:
		return "", atMost("category", maxCategoryRunes)
	case hasControl(c):
		return "", invalidInput("category", "must not contain control characters")
	}

	return c, nil
}

func cleanPax(field string, n, min int) error {
	if n < min || n > maxPax {
		return invalidInput(field, fmt.Sprintf("must be between %d and %d", min, maxPax))
	}

	return nil
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
		return nil, nil, atMost(field, maxPhoneRunes)
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

// scopeOf resolves a session scope: blank = all; anything that is not a session of the project is invalid_scope.
func scopeOf(raw string, scopes map[string]bool) (string, error) {
	scope := strings.TrimSpace(raw)
	if scope == "" {
		return pd.AllScope, nil
	}
	if !keyPattern.MatchString(scope) || !scopes[scope] {
		return "", errUnknownScope()
	}

	return scope, nil
}

// guestFields are the editable fields of a guest after validation.
type guestFields struct {
	name      string
	side      *string
	category  string
	scope     string
	table     *string
	pax       int
	phone, wa *string
	notes     *string
}

// checkGuest validates the body of POST and PUT in the order of the table of 05 BE 3.4. paxField is the name the
// head-count has in the body: pax_count, or pax_actual for a walk-in.
func checkGuest(spec domain.GuestSpec, scopes map[string]bool, req domain.GuestRequest, paxField string) (guestFields, error) {
	var f guestFields
	var err error
	if f.name, err = cleanName("name", req.Name, maxNameRunes); err != nil {
		return f, err
	}
	if f.side, err = cleanSide(spec, req.GuestSide); err != nil {
		return f, err
	}
	if f.category, err = cleanCategory(spec, req.Category); err != nil {
		return f, err
	}
	if f.scope, err = scopeOf(req.SubEventScope, scopes); err != nil {
		return f, err
	}
	if f.table, err = optionalLine("table_number", req.TableNumber, maxTableRunes); err != nil {
		return f, err
	}
	if err = cleanPax(paxField, req.PaxCount, minPax); err != nil {
		return f, err
	}
	f.pax = req.PaxCount
	if f.phone, f.wa, err = cleanPhone("phone_number", req.PhoneNumber); err != nil {
		return f, err
	}
	f.notes, err = optionalNote("notes", req.Notes, maxNoteRunes)

	return f, err
}

// ---------- queries ----------

// GuestFilter is the query string of GET /guests, as it arrived.
type GuestFilter struct {
	Side, Category, RSVP, Sent, Scope, Search string
	Limit, Offset                             string
}

// ReceptionFilter is the query string of GET /guests/reception-overview, as it arrived.
type ReceptionFilter struct {
	Status, Category, Search string
	Limit, Offset            string
}

func oneOf(field, v string, allowed ...string) error {
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}

	return invalidInput(field, "must be one of "+strings.Join(allowed, ", "))
}

func cleanSearch(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxSearchRunes {
		return "", atMost("search", maxSearchRunes)
	}

	return s, nil
}

// cleanQueryKey checks a key in the query string; noun names it in the message ("side key", "session scope_key").
func cleanQueryKey(field, noun, v string) error {
	if v != "" && !keyPattern.MatchString(v) {
		return invalidInput(field, "must be a "+noun)
	}

	return nil
}

// page reads limit and offset: limit is def when absent and must be 1..max, offset is 0 when absent and must be >= 0.
func page(limitRaw, offsetRaw string, def, max int) (limit, offset int, err error) {
	limit = def
	if limitRaw != "" {
		n, perr := strconv.Atoi(limitRaw)
		if perr != nil || n < 1 || n > max {
			return 0, 0, invalidInput("limit", fmt.Sprintf("must be between 1 and %d", max))
		}
		limit = n
	}
	if offsetRaw != "" {
		n, perr := strconv.Atoi(offsetRaw)
		if perr != nil || n < 0 {
			return 0, 0, invalidInput("offset", "must be at least 0")
		}
		offset = n
	}

	return limit, offset, nil
}

func (f GuestFilter) parse() (domain.GuestQuery, error) {
	q := domain.GuestQuery{Side: f.Side, Category: strings.TrimSpace(f.Category), RSVP: f.RSVP, Sent: f.Sent, Scope: f.Scope}
	if q.Side != domain.SideNone {
		if err := cleanQueryKey("side", "side key", q.Side); err != nil {
			return q, err
		}
	}
	if utf8.RuneCountInString(q.Category) > maxCategoryRunes {
		return q, atMost("category", maxCategoryRunes)
	}
	if q.RSVP != "" {
		if err := oneOf("rsvp", q.RSVP, "unconfirmed", "attending", "declined", "maybe"); err != nil {
			return q, err
		}
	}
	if q.Sent != "" {
		if err := oneOf("sent", q.Sent, "sent", "unsent"); err != nil {
			return q, err
		}
	}
	if err := cleanQueryKey("scope", "session scope_key", q.Scope); err != nil {
		return q, err
	}
	var err error
	if q.Search, err = cleanSearch(f.Search); err != nil {
		return q, err
	}
	q.Limit, q.Offset, err = page(f.Limit, f.Offset, defaultListLimit, maxListLimit)

	return q, err
}

func (f ReceptionFilter) parse() (domain.ReceptionQuery, error) {
	q := domain.ReceptionQuery{Status: f.Status, Category: strings.TrimSpace(f.Category)}
	if q.Status == "" {
		q.Status = domain.StatusAll
	}
	if err := oneOf("status", q.Status, domain.StatusAll, domain.StatusIn, domain.StatusPending); err != nil {
		return q, err
	}
	if utf8.RuneCountInString(q.Category) > maxCategoryRunes {
		return q, atMost("category", maxCategoryRunes)
	}
	var err error
	if q.Search, err = cleanSearch(f.Search); err != nil {
		return q, err
	}
	q.Limit, q.Offset, err = page(f.Limit, f.Offset, defaultReceptionLimit, maxReceptionLimit)

	return q, err
}

// bufferOf reads the two parameters of the catering projection (whole numbers, defaults 25 and 0).
func bufferOf(percentRaw, paxRaw string) (percent, pax int, err error) {
	percent = domain.DefaultBufferPercent
	if percentRaw != "" {
		n, perr := strconv.Atoi(percentRaw)
		if perr != nil || n < 0 || n > domain.MaxBufferPercent {
			return 0, 0, invalidInput("buffer_percent", fmt.Sprintf("must be between 0 and %d", domain.MaxBufferPercent))
		}
		percent = n
	}
	if paxRaw != "" {
		n, perr := strconv.Atoi(paxRaw)
		if perr != nil || n < 0 || n > domain.MaxBufferPax {
			return 0, 0, invalidInput("buffer_pax", fmt.Sprintf("must be between 0 and %d", domain.MaxBufferPax))
		}
		pax = n
	}

	return percent, pax, nil
}
