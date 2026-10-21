package ginutil

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// maxSlugLen keeps download file names short.
const maxSlugLen = 40

// Slug turns free text into an ASCII file-name part: accents are folded,
// everything outside [a-z0-9] becomes "-", runs and the ends of "-" are
// dropped, and the result is cut to 40 characters. Text with no Latin letter or
// digit (emoji, other scripts) becomes "proyek". Used for Content-Disposition
// names so a title with quotes, emoji or a line break can never reach a header
// (PLANNER_SPEC kontrak bab 2 butir 6).
func Slug(s string) string {
	var b strings.Builder
	pendingSep := false
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingSep && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingSep = false
			b.WriteRune(r)
		} else {
			pendingSep = true
		}
	}

	out := b.String()
	if len(out) > maxSlugLen {
		out = strings.TrimRight(out[:maxSlugLen], "-")
	}
	if out == "" {
		return "proyek"
	}

	return out
}
