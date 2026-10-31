package usecase

import (
	"context"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// metaString reads a text of the project's metadata (venue_name, city); anything else is empty.
func metaString(p pd.Project, key string) string {
	s, _ := p.Metadata[key].(string)

	return s
}

// ExportRundownWA is GET /rundowns/export-wa (06 BE 2.4): the rundown of one session (?session_id=) or of every session
// that has agendas, as a WhatsApp message. sessionID is the raw query value, empty for all.
func (s *Service) ExportRundownWA(ctx context.Context, p pd.Project, sessionID string) (domain.WAExport, error) {
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.WAExport{}, err
	}

	var filter *domain.SessionRef
	if sessionID != "" {
		id, err := parseSessionQuery(sessionID)
		if err != nil {
			return domain.WAExport{}, err
		}
		sess, err := s.session(ctx, p.ID, id)
		if err != nil {
			return domain.WAExport{}, err
		}
		filter = &sess
	}
	agendas, err := s.repo.ListAgendas(ctx, p.ID, nil)
	if err != nil {
		return domain.WAExport{}, err
	}

	et, _ := s.registry.Get(p.EventType)
	out := domain.AgendaExport{
		ProjectTitle: p.Title, EventIcon: et.Icon, EventDate: p.EventDate,
		Venue: metaString(p, "venue_name"), City: metaString(p, "city"), Single: filter != nil,
	}
	for _, sess := range sessions {
		if sess.IsSystem || (filter != nil && sess.ID != filter.ID) {
			continue
		}
		es := domain.ExportSession{Session: sess, Agendas: []domain.Rundown{}}
		for _, a := range agendas {
			if a.SessionID == sess.ID {
				es.Agendas = append(es.Agendas, a)
			}
		}
		out.Sessions = append(out.Sessions, es)
	}

	return domain.WAExport{Text: out.Text()}, nil
}

// ExportPlaylistWA is GET /playlists/export-wa (06 BE 2.7): the briefing for the soundman or the band.
func (s *Service) ExportPlaylistWA(ctx context.Context, p pd.Project) (domain.WAExport, error) {
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.WAExport{}, err
	}
	songs, err := s.repo.ListPlaylists(ctx, p.ID)
	if err != nil {
		return domain.WAExport{}, err
	}

	out := domain.PlaylistExport{ProjectTitle: p.Title, Songs: songs}
	for _, sess := range sessions {
		if !sess.IsSystem {
			out.Sessions = append(out.Sessions, sess)
		}
	}

	return domain.WAExport{Text: out.Text()}, nil
}
