package domain

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxScopeKeyLen is the column limit (^[a-z0-9_]{1,40}$). Names are cut to 36
// so that a suffix up to _999 still fits.
const (
	MaxScopeKeyLen  = 40
	baseScopeKeyLen = 36
	maxScopeSuffix  = 999
)

// ScopeKeyFromName derives the base scope_key of a user-made session: accents
// are folded, everything outside [a-z0-9] becomes "_", runs and ends of "_"
// are dropped. A name with no Latin letter or digit (emoji, other scripts)
// becomes "sesi", and the reserved "all" becomes "all_sesi".
func ScopeKeyFromName(name string) string {
	var b strings.Builder
	pendingSep := false
	for _, r := range norm.NFD.String(name) {
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

	key := b.String()
	if len(key) > baseScopeKeyLen {
		key = strings.TrimRight(key[:baseScopeKeyLen], "_")
	}
	switch key {
	case "":
		return "sesi"
	case AllScope:
		return "all_sesi"
	}

	return key
}

// UniqueScopeKey returns base, or base_2, base_3, … (up to _999) — the first
// one not in taken. The boolean is false when all are taken.
func UniqueScopeKey(base string, taken map[string]bool) (string, bool) {
	if !taken[base] {
		return base, true
	}
	for n := 2; n <= maxScopeSuffix; n++ {
		if k := base + "_" + strconv.Itoa(n); !taken[k] {
			return k, true
		}
	}

	return "", false
}
