package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxCategoryKeyLen is the column limit (^[a-z0-9_]{1,40}$).
const MaxCategoryKeyLen = 40

// CategoryKeyFromLabel derives the category_key of a custom vendor category: lower-case ASCII (accents are folded),
// everything outside [a-z0-9] becomes "_", runs and the ends of "_" are dropped, cut to 40. A label with no Latin
// letter or digit (emoji, other scripts) becomes "kategori". The rule is the one of planner/domain.ScopeKeyFromName
// (without its cut to 36 and its reserved words); a label "Venue" gives the registry key `venue`, so a custom category
// named like a registry one merges with it. ginutil.Slug is not used: it writes hyphens and answers "proyek".
func CategoryKeyFromLabel(label string) string {
	key := underscoreKey(label)
	if len(key) > MaxCategoryKeyLen {
		key = strings.TrimRight(key[:MaxCategoryKeyLen], "_")
	}
	if key == "" {
		return "kategori"
	}

	return key
}

// underscoreKey folds accents and lower-cases label, turns every run outside [a-z0-9] into one "_" and drops the ends.
func underscoreKey(label string) string {
	var b strings.Builder
	pendingSep := false
	for _, r := range norm.NFD.String(label) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingSep && b.Len() > 0 {
				b.WriteByte('_')
			}
			pendingSep = false
			b.WriteRune(r)
		} else {
			pendingSep = true
		}
	}

	return b.String()
}
