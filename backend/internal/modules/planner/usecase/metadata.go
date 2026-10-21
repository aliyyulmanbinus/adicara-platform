package usecase

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func invalidMetadata(key, reason string) *apierr.Error {
	return apierr.BadRequest("invalid_metadata", key+": "+reason)
}

// mergeMetadata applies patch to base as a JSON Merge Patch against the
// registry fields of the event type: an existing key is replaced, null
// deletes it, a key that is not mentioned stays. Keys outside the registry are
// rejected, values are checked for type and length, and every required field
// must still be present afterwards. base is not modified.
func mergeMetadata(spec domain.EventTypeSpec, base, patch map[string]any) (map[string]any, error) {
	fields := make(map[string]domain.FieldSpec, len(spec.MetadataFields))
	for _, f := range spec.MetadataFields {
		fields[f.Key] = f
	}

	out := make(map[string]any, len(base)+len(patch))
	for k, v := range base {
		out[k] = v
	}

	for key, raw := range patch {
		field, ok := fields[key]
		if !ok {
			return nil, invalidMetadata(key, "unknown key for this event type")
		}
		if raw == nil {
			delete(out, key)
			continue
		}
		v, keep, err := normalizeField(field, raw)
		if err != nil {
			return nil, err
		}
		if keep {
			out[key] = v
		} else {
			delete(out, key) // an optional text field sent empty is the same as absent
		}
	}

	for _, f := range spec.MetadataFields {
		if f.Required {
			if _, ok := out[f.Key]; !ok {
				return nil, invalidMetadata(f.Key, "required")
			}
		}
	}

	return out, nil
}

// normalizeField validates one non-null value. keep is false for an optional
// string that is empty after trimming.
func normalizeField(f domain.FieldSpec, raw any) (value any, keep bool, err error) {
	switch f.Type {
	case domain.FieldString, domain.FieldDate:
		s, ok := raw.(string)
		if !ok {
			return nil, false, invalidMetadata(f.Key, "must be a string")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			if f.Required {
				return nil, false, invalidMetadata(f.Key, "required")
			}

			return nil, false, nil
		}
		if hasControl(s) {
			return nil, false, invalidMetadata(f.Key, "must not contain control characters")
		}
		if f.Type == domain.FieldDate {
			if _, perr := time.Parse(domain.DateLayout, s); perr != nil {
				return nil, false, invalidMetadata(f.Key, "must be a date in YYYY-MM-DD format")
			}

			return s, true, nil
		}
		if f.MaxLen > 0 && utf8.RuneCountInString(s) > f.MaxLen {
			return nil, false, invalidMetadata(f.Key, fmt.Sprintf("must be at most %d characters", f.MaxLen))
		}

		return s, true, nil

	case domain.FieldInt:
		n, ok := raw.(float64) // encoding/json decodes every number as float64
		if !ok || n != math.Trunc(n) || math.IsInf(n, 0) {
			return nil, false, invalidMetadata(f.Key, "must be an integer")
		}
		if (f.Min != nil && n < float64(*f.Min)) || (f.Max != nil && n > float64(*f.Max)) {
			return nil, false, invalidMetadata(f.Key, intRange(f))
		}

		return int64(n), true, nil
	}

	return nil, false, invalidMetadata(f.Key, "unsupported field type")
}

func intRange(f domain.FieldSpec) string {
	switch {
	case f.Min != nil && f.Max != nil:
		return fmt.Sprintf("must be between %d and %d", *f.Min, *f.Max)
	case f.Min != nil:
		return fmt.Sprintf("must be at least %d", *f.Min)
	default:
		return fmt.Sprintf("must be at most %d", *f.Max)
	}
}
