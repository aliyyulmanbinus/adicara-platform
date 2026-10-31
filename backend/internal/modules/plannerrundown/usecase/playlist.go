package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// SongFilter is the query of GET /playlists; an empty field does not filter.
type SongFilter struct{ MomentTag, AudioFormat, Status string }

func (f SongFilter) parse() (domain.SongFilter, error) {
	out := domain.SongFilter{MomentTag: strings.TrimSpace(f.MomentTag), AudioFormat: domain.AudioFormat(f.AudioFormat), Status: domain.AudioStatus(f.Status)}
	if utf8.RuneCountInString(out.MomentTag) > maxMomentRunes {
		return out, invalidInput("moment_tag", "must be at most 100 characters")
	}
	if out.AudioFormat != "" && !out.AudioFormat.Valid() {
		return out, invalidInput("audio_format", "must be one of digital_playback, live_band, acoustic, sound_fx")
	}
	if out.Status != "" && !out.Status.Valid() {
		return out, invalidInput("status", "must be one of ready, needs_review, missing_file")
	}

	return out, nil
}

// ListPlaylists is GET /playlists: the KPI describes the whole playlist; only the items follow the filters.
func (s *Service) ListPlaylists(ctx context.Context, p pd.Project, f SongFilter) (domain.PlaylistOverview, error) {
	filter, err := f.parse()
	if err != nil {
		return domain.PlaylistOverview{}, err
	}
	all, err := s.repo.ListPlaylists(ctx, p.ID)
	if err != nil {
		return domain.PlaylistOverview{}, err
	}

	items := make([]domain.Playlist, 0, len(all))
	for _, song := range all {
		if filter.Keep(song) {
			items = append(items, song)
		}
	}

	return domain.PlaylistOverview{KPI: domain.PlaylistKPIOf(all), Items: items}, nil
}

// songWrite validates a POST or PUT body in the order of 06 BE 3.4 and returns the row to store (without the id). The
// agenda the song points at is checked by the composite foreign key when the row is written.
func songWrite(req domain.PlaylistRequest) (domain.SongWrite, error) {
	w := domain.SongWrite{RundownID: req.RundownID}
	var err error
	if w.SongTitle, err = cleanText("song_title", req.SongTitle, maxSongRunes, true); err != nil {
		return w, err
	}
	if w.Artist, err = cleanText("artist", req.Artist, maxSongRunes, true); err != nil {
		return w, err
	}
	if w.MomentTag, err = cleanText("moment_tag", req.MomentTag, maxMomentRunes, false); err != nil {
		return w, err
	}
	if w.MomentTag == "" {
		w.MomentTag = domain.DefaultMomentTag
	}
	if w.AudioFormat = req.AudioFormat; w.AudioFormat == "" {
		w.AudioFormat = domain.FormatDigitalPlayback
	} else if !w.AudioFormat.Valid() {
		return w, invalidInput("audio_format", "must be one of digital_playback, live_band, acoustic, sound_fx")
	}
	if w.Status = req.Status; w.Status == "" {
		w.Status = domain.AudioNeedsReview
	} else if !w.Status.Valid() {
		return w, invalidInput("status", "must be one of ready, needs_review, missing_file")
	}
	if w.SpotifyURL, err = cleanURL("spotify_url", req.SpotifyURL); err != nil {
		return w, err
	}
	if w.YouTubeURL, err = cleanURL("youtube_url", req.YouTubeURL); err != nil {
		return w, err
	}
	w.CueNotes, err = optionalNote("audio_cue_notes", req.AudioCueNotes, maxNoteRunes)

	return w, err
}

// CreatePlaylist is POST /playlists; the song goes last and the limit is checked under the project lock.
func (s *Service) CreatePlaylist(ctx context.Context, p pd.Project, req domain.PlaylistRequest) (domain.Playlist, error) {
	w, err := songWrite(req)
	if err != nil {
		return domain.Playlist{}, err
	}
	w.ID = uuid.New()

	var out domain.Playlist
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountPlaylists(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxSongs {
			return playlistLimitReached()
		}
		if err := s.repo.InsertPlaylist(ctx, p.ID, &w); err != nil {
			return err
		}
		out, err = s.repo.PlaylistByID(ctx, p.ID, w.ID)

		return err
	})
	if err != nil {
		return domain.Playlist{}, mapErr(err, "song")
	}

	return out, nil
}

// UpdatePlaylist is PUT /playlists/:id: the whole body replaces the song, except its order and template key.
func (s *Service) UpdatePlaylist(ctx context.Context, p pd.Project, id uuid.UUID, req domain.PlaylistRequest) (domain.Playlist, error) {
	w, err := songWrite(req)
	if err != nil {
		return domain.Playlist{}, err
	}
	w.ID = id

	var out domain.Playlist
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdatePlaylist(ctx, p.ID, &w); err != nil {
			return err
		}
		var err error
		out, err = s.repo.PlaylistByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.Playlist{}, mapErr(err, "song")
	}

	return out, nil
}

func (s *Service) DeletePlaylist(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeletePlaylist(ctx, p.ID, id), "song")
}

// ReorderPlaylists is POST /playlists/reorder: ids must be exactly the songs of the project, each once.
func (s *Service) ReorderPlaylists(ctx context.Context, p pd.Project, req domain.ReorderPlaylistsRequest) error {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		have, err := s.repo.PlaylistIDs(ctx, p.ID)
		if err != nil {
			return err
		}
		if !sameIDs(have, req.IDs) {
			return invalidInput("ids", "must list every song of the project exactly once")
		}

		return s.repo.ReorderPlaylists(ctx, p.ID, req.IDs)
	})

	return mapErr(err, "song")
}
