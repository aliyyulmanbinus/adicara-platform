package domain

import (
	"strings"
	"unicode"
)

// FallbackCategory is used when no category is given. It is not a registry category of any event type
// (TEMPLATE_EVENT 4.2); the counters show it only when a guest uses it.
const (
	FallbackCategory      = "lainnya"
	FallbackCategoryLabel = "Lainnya"
	UndecidedSideLabel    = "Belum ditentukan"
	SideNone              = "none" // value of the ?side= filter for guests whose side is not decided yet
)

// GuestSpec is the part of the event type registry that menu 05 reads (EventTypeSpec.GuestSides/GuestCategories).
type GuestSpec struct {
	Sides      []Option
	Categories []Option
}

// squash trims and folds whitespace (tabs and line breaks included) into single spaces.
func squash(s string) string { return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ") }

// labelCore drops a trailing note in parentheses: "Pihak Pria (CPP)" -> "Pihak Pria".
func labelCore(label string) string {
	if i := strings.LastIndex(label, " ("); i > 0 && strings.HasSuffix(label, ")") {
		return label[:i]
	}
	return label
}

func (s GuestSpec) match(opts []Option, input string) (string, bool) {
	in := squash(input)
	if in == "" {
		return "", false
	}
	for _, o := range opts {
		if strings.EqualFold(in, o.Key) || strings.EqualFold(in, o.Label) || strings.EqualFold(in, labelCore(o.Label)) {
			return o.Key, true
		}
	}
	return "", false
}

// ResolveSide maps an input (key or label, case-insensitive) to a registry side key.
// Empty -> ("", true): side not decided. Unknown -> ("", false). A type without sides refuses any input.
func (s GuestSpec) ResolveSide(input string) (string, bool) {
	if squash(input) == "" {
		return "", true
	}
	return s.match(s.Sides, input)
}

// ResolveCategory maps an input to a registry key; other text becomes a free-text category (whitespace folded,
// case kept); empty becomes FallbackCategory. The caller checks the length (60 at most).
func (s GuestSpec) ResolveCategory(input string) string {
	in := squash(input)
	if in == "" {
		return FallbackCategory
	}
	if k, ok := s.match(s.Categories, in); ok {
		return k
	}
	if strings.EqualFold(in, FallbackCategory) || strings.EqualFold(in, FallbackCategoryLabel) {
		return FallbackCategory
	}
	return in
}

// CategoryLabel: the registry label, "Lainnya" for the fallback key, otherwise the free text as it is.
func (s GuestSpec) CategoryLabel(key string) string {
	for _, o := range s.Categories {
		if o.Key == key {
			return o.Label
		}
	}
	if key == FallbackCategory {
		return FallbackCategoryLabel
	}
	return key
}

// SideLabel is the label of a side key; nil is "Belum ditentukan" (the same word as the overview). A key the registry
// does not know is shown as it is.
func (s GuestSpec) SideLabel(key *string) string {
	if key == nil {
		return UndecidedSideLabel
	}
	for _, o := range s.Sides {
		if o.Key == *key {
			return o.Label
		}
	}

	return *key
}

func (s GuestSpec) registryCategory(key string) bool {
	for _, o := range s.Categories {
		if o.Key == key {
			return true
		}
	}
	return false
}
