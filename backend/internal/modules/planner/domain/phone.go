package domain

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid phone number")

// NormalizeWA turns the Indonesian mobile number a user typed into the wa.me form (62…, digits only).
// It lives here, not in a module, because the vendor module (menu 04) and the guest module (menu 05)
// both use it and the other planner modules may only import this package.
//
// An empty number (after trimming) is "" without an error: the number is optional. The rules, in order:
//  1. spaces, '-', '.', '(' and ')' are dropped; the only other character allowed is a '+' at the very start;
//  2. a "+" or "00" prefix is dropped (international); otherwise a leading "0" becomes "62" (national);
//  3. the result must start with "62" (Indonesian numbers only); a "0" right after "62" is dropped ("+62 0812…" is a common typo);
//  4. the final length is 10–15 digits.
//
// A number with no prefix ("8123456789") is refused: nothing is guessed.
func NormalizeWA(input string) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(input) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return "", ErrInvalidPhone
		}
	}
	s := b.String()
	if s == "" {
		return "", nil
	}

	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	}
	if !strings.HasPrefix(s, "62") {
		return "", ErrInvalidPhone
	}
	s = "62" + strings.TrimPrefix(s[2:], "0")
	if len(s) < 10 || len(s) > 15 {
		return "", ErrInvalidPhone
	}

	return s, nil
}
