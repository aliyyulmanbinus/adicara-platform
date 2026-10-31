package usecase

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
)

// Limits (06 BE 3.4 and 2.9). The counts are product limits, not database ones, and are checked under the project lock
// so a double click cannot pass them.
const (
	maxAgendas = 500
	maxSongs   = 500

	maxActivityRunes = 255
	maxPICRunes      = 150
	maxNoteRunes     = 2000 // description, highlights and audio cue notes
	maxSongRunes     = 255  // song title and artist
	maxMomentRunes   = 100
	maxURLRunes      = 2000

	reasonVendorOfProject = "must be a vendor of this project"
	reasonAgendaOfProject = "must be an agenda of this project"
)

// Validation messages are "<field>: <reason>" so a form can mark the field.
func invalidInput(field, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_input", field+": "+reason)
}

func agendaLimitReached() *apierr.Error {
	return apierr.Conflict("agenda_limit_reached", "at most 500 agendas per project")
}

func playlistLimitReached() *apierr.Error {
	return apierr.Conflict("playlist_limit_reached", "at most 500 songs per project")
}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// cleanText trims s and enforces 1..max runes (0..max when optional) with no control characters.
func cleanText(field, s string, max int, required bool) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "" && required:
		return "", invalidInput(field, "required")
	case utf8.RuneCountInString(s) > max:
		return "", invalidInput(field, fmt.Sprintf("must be at most %d characters", max))
	case hasControl(s):
		return "", invalidInput(field, "must not contain control characters")
	}

	return s, nil
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

// cleanURL accepts only http and https links; a blank one is none.
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
