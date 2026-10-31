package domain

import "testing"

func TestBuildKPI(t *testing.T) {
	sessions := []RundownSessionSummary{
		{AgendaCount: 11, Completed: 11, SpanMinutes: 100},
		{AgendaCount: 8, Completed: 4, InProgress: 1, SpanMinutes: 120},
		{AgendaCount: 14, SpanMinutes: 225},
		{}, // a session without agendas counts as a session and nothing else
	}
	got := BuildKPI(sessions, 17)
	want := RundownKPI{Sessions: 4, TotalAgendas: 33, Completed: 15, InProgress: 1, Pending: 17, TotalSpanMinutes: 445, DistinctPICs: 17}
	if got != want {
		t.Errorf("kpi = %+v, want %+v", got, want)
	}
	if empty := BuildKPI(nil, 0); empty != (RundownKPI{}) {
		t.Errorf("empty kpi = %+v", empty)
	}
}

func TestPlaylistKPIAndFilter(t *testing.T) {
	songs := []Playlist{
		{MomentTag: "Akad Khidmat", AudioFormat: FormatDigitalPlayback, Status: AudioNeedsReview},
		{MomentTag: "Akad Khidmat", AudioFormat: FormatLiveBand, Status: AudioReady},
		{MomentTag: "Penutup", AudioFormat: FormatAcoustic, Status: AudioMissingFile},
		{MomentTag: "penutup", AudioFormat: FormatSoundFX, Status: AudioReady}, // a tag counts as typed
	}
	got := PlaylistKPIOf(songs)
	want := PlaylistKPI{
		TotalTracks: 4, ByFormat: FormatCounts{DigitalPlayback: 1, LiveBand: 1, Acoustic: 1, SoundFX: 1},
		MappedMoments: 3, Ready: 2, NeedsReview: 1, MissingFile: 1,
	}
	if got != want {
		t.Errorf("kpi = %+v, want %+v", got, want)
	}

	keep := func(f SongFilter) (n int) {
		for _, s := range songs {
			if f.Keep(s) {
				n++
			}
		}

		return n
	}
	for name, c := range map[string]struct {
		f    SongFilter
		want int
	}{
		"none": {SongFilter{}, 4}, "tag": {SongFilter{MomentTag: "Akad Khidmat"}, 2}, "tag is case sensitive": {SongFilter{MomentTag: "PENUTUP"}, 0},
		"format": {SongFilter{Format: FormatLiveBand}, 1}, "status": {SongFilter{Status: AudioReady}, 2},
		"all three": {SongFilter{MomentTag: "Akad Khidmat", Format: FormatLiveBand, Status: AudioReady}, 1},
		"no match":  {SongFilter{MomentTag: "Akad Khidmat", Status: AudioMissingFile}, 0},
	} {
		if n := keep(c.f); n != c.want {
			t.Errorf("%s: %d songs, want %d", name, n, c.want)
		}
	}
}

func TestHighlightItems(t *testing.T) {
	got := HighlightItems("- Kata Sambutan\n-Tanpa spasi\n\n  - Berindent  \nTanpa tanda\n-")
	want := []string{"Kata Sambutan", "Tanpa spasi", "Berindent", "Tanpa tanda"}
	if len(got) != len(want) {
		t.Fatalf("items = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d = %q, want %q", i, got[i], want[i])
		}
	}
	if HighlightItems("") != nil {
		t.Error("no highlights must give no items")
	}
}
