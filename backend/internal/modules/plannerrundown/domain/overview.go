package domain

import "strings"

// BuildKPI assembles the headline numbers of the rundown from the session summaries (06 BE 2.8): the totals are the sums
// over the sessions and pending is what is left. The distinct PICs come from the database, they are not a sum.
func BuildKPI(sessions []RundownSessionSummary, distinctPICs int) RundownKPI {
	k := RundownKPI{Sessions: len(sessions), DistinctPICs: distinctPICs}
	for _, s := range sessions {
		k.TotalAgendas += s.AgendaCount
		k.Completed += s.Completed
		k.InProgress += s.InProgress
		k.TotalSpanMinutes += s.SpanMinutes
	}
	k.Pending = k.TotalAgendas - k.Completed - k.InProgress

	return k
}

// PlaylistKPIOf counts the whole playlist of the project. A moment tag counts as typed ("Penutup" and "penutup" are two).
func PlaylistKPIOf(items []Playlist) PlaylistKPI {
	k := PlaylistKPI{TotalTracks: len(items)}
	moments := map[string]bool{}
	for _, it := range items {
		moments[it.MomentTag] = true
		switch it.AudioFormat {
		case FormatDigitalPlayback:
			k.ByFormat.DigitalPlayback++
		case FormatLiveBand:
			k.ByFormat.LiveBand++
		case FormatAcoustic:
			k.ByFormat.Acoustic++
		case FormatSoundFX:
			k.ByFormat.SoundFX++
		}
		switch it.Status {
		case AudioReady:
			k.Ready++
		case AudioNeedsReview:
			k.NeedsReview++
		case AudioMissingFile:
			k.MissingFile++
		}
	}
	k.MappedMoments = len(moments)

	return k
}

// SongFilter is the query of GET /playlists; an empty field does not filter.
type SongFilter struct {
	MomentTag string
	Format    AudioFormat
	Status    AudioStatus
}

// Keep reports whether a song passes the filter. The moment tag is compared as typed, like the KPI counts it.
func (f SongFilter) Keep(s Playlist) bool {
	return (f.MomentTag == "" || s.MomentTag == f.MomentTag) &&
		(f.Format == "" || s.AudioFormat == f.Format) &&
		(f.Status == "" || s.Status == f.Status)
}

// AudioFormatLabel is the label of a format in the WhatsApp text.
func AudioFormatLabel(f AudioFormat) string {
	switch f {
	case FormatLiveBand:
		return "Live Band"
	case FormatAcoustic:
		return "Akustik"
	case FormatSoundFX:
		return "Sound FX"
	}

	return "Digital Playback"
}

// HighlightItems splits the highlights of an agenda into its items: one per line, the "- " bullet removed, blanks dropped.
func HighlightItems(h string) []string {
	var out []string
	for _, line := range strings.Split(h, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line != "" {
			out = append(out, line)
		}
	}

	return out
}
