package domain

import pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"

// TemplateAgenda is one agenda of an event type's rundown template. ScopeKey names the session it belongs to (the
// registry seeds the session; a session the user has deleted is skipped with the warning scope_missing). End is empty
// for an open agenda. The key is the template_key that makes apply-template idempotent.
type TemplateAgenda struct {
	Key         string
	ScopeKey    string
	Start, End  string // "HH:MM"
	Name        string
	PIC         string
	Description string
	Highlights  string // lines starting with "- "
}

// TemplateSong is one song of the playlist template (the wedding only).
type TemplateSong struct {
	Key       string
	Title     string
	Artist    string
	MomentTag string
}

// AgendasFor is the rundown template of an event type: the 33 agendas of the wedding, the 3 to 9 of each other type.
func AgendasFor(t pd.EventType) []TemplateAgenda {
	if t == pd.EventWedding {
		out := make([]TemplateAgenda, 0, len(WeddingLamaran)+len(WeddingAkadResepsi))
		out = append(out, WeddingLamaran...)

		return append(out, WeddingAkadResepsi...)
	}

	return GeneralAgendas[string(t)]
}

// PlaylistFor is the playlist template of an event type; only the wedding has one.
func PlaylistFor(t pd.EventType) []TemplateSong {
	if t == pd.EventWedding {
		return WeddingPlaylist
	}

	return nil
}
