// Package domain holds the planner menu 06 types: the agendas of the rundown (minute by minute, per session) and the
// songs of the playlist, with the rules that need no database (clock times, the span of a session, the WhatsApp
// text, the templates). The sessions belong to menu 01 and the vendors to menu 04; they are read, never written.
// Every response type carries snake_case tags, lists are initialised before they are serialised (an empty list is
// [], not null), optional keys are omitted, and no percentage is sent (the client computes it).
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ErrNotFound, ErrInvalidScope and ErrConflict are the sentinels of the planner modules (the shared TxManager already
// returns pd.ErrConflict for a failed COMMIT); they are reused so one mapping serves every planner module.
var (
	ErrNotFound     = pd.ErrNotFound
	ErrInvalidScope = pd.ErrInvalidScope
	ErrConflict     = pd.ErrConflict

	// ErrInvalidVendor: an agenda pointed at a vendor of another project (the composite foreign key).
	ErrInvalidVendor = errors.New("vendor does not belong to the project")
	// ErrInvalidAgenda: a song pointed at an agenda of another project (the composite foreign key).
	ErrInvalidAgenda = errors.New("agenda does not belong to the project")
)

type RundownStatus string

const (
	StatusPending    RundownStatus = "pending"
	StatusInProgress RundownStatus = "in_progress"
	StatusCompleted  RundownStatus = "completed"
)

func (s RundownStatus) Valid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusCompleted
}

type AudioFormat string

const (
	FormatDigitalPlayback AudioFormat = "digital_playback"
	FormatLiveBand        AudioFormat = "live_band"
	FormatAcoustic        AudioFormat = "acoustic"
	FormatSoundFX         AudioFormat = "sound_fx"
)

func (f AudioFormat) Valid() bool {
	return f == FormatDigitalPlayback || f == FormatLiveBand || f == FormatAcoustic || f == FormatSoundFX
}

type AudioStatus string

const (
	AudioReady       AudioStatus = "ready"
	AudioNeedsReview AudioStatus = "needs_review"
	AudioMissingFile AudioStatus = "missing_file"
)

// DefaultMomentTag is what a song gets without a moment tag.
const DefaultMomentTag = "Lainnya"

// The modes of apply-template (the same words as menus 02-04).
const (
	ModeAppend          = "append"
	ModeReplaceTemplate = "replace_template"
)

func (s AudioStatus) Valid() bool {
	return s == AudioReady || s == AudioNeedsReview || s == AudioMissingFile
}

// ---------- requests ----------
//
// No `binding` tags: the usecase validates and answers "<field>: <reason>" (a binding failure would be an opaque
// invalid_json), like menus 04 and 05. Text is trimmed and measured in runes there too.

// RundownRequest is the body of POST and PUT of an agenda (PUT replaces it whole; the status only moves through PATCH).
type RundownRequest struct {
	SessionID    uuid.UUID  `json:"session_id"` // a non-system session of the project
	StartTime    string     `json:"start_time"` // "HH:MM" or "HH.MM"
	EndTime      *string    `json:"end_time"`   // nil = open ("selesai")
	EndsNextDay  bool       `json:"ends_next_day"`
	ActivityName string     `json:"activity_name"`
	PICName      string     `json:"pic_name"`
	VendorID     *uuid.UUID `json:"vendor_id"` // a vendor of the project, or empty
	Description  string     `json:"description"`
	Highlights   string     `json:"highlights"` // one item per line, starting with "- "
}

type RundownStatusRequest struct {
	Status RundownStatus `json:"status"`
}

type ReorderRundownsRequest struct {
	SessionID uuid.UUID   `json:"session_id"`
	IDs       []uuid.UUID `json:"ids"` // exactly the agendas of that session, once each
}

type ReorderPlaylistsRequest struct {
	IDs []uuid.UUID `json:"ids"` // exactly the songs of the project, once each
}

type ApplyTemplateRequest struct {
	Mode    string `json:"mode"`    // append (when empty) | replace_template (agendas only; the playlist only appends)
	Confirm bool   `json:"confirm"` // must be true when replace_template resets agendas
}

// PlaylistRequest is the body of POST and PUT of a song.
type PlaylistRequest struct {
	RundownID     *uuid.UUID  `json:"rundown_id"` // an agenda of the project, or empty
	SongTitle     string      `json:"song_title"`
	Artist        string      `json:"artist"`
	MomentTag     string      `json:"moment_tag"`   // empty = "Lainnya"
	AudioFormat   AudioFormat `json:"audio_format"` // empty = digital_playback
	SpotifyURL    *string     `json:"spotify_url"`
	YouTubeURL    *string     `json:"youtube_url"`
	AudioCueNotes *string     `json:"audio_cue_notes"`
	Status        AudioStatus `json:"status"` // empty = needs_review
}

// ---------- agenda responses ----------

type RundownVendor struct {
	ID                uuid.UUID `json:"id"`
	VendorName        string    `json:"vendor_name"`
	CategoryLabel     string    `json:"category_label"`
	PICName           string    `json:"pic_name,omitempty"`
	PhoneNumber       string    `json:"phone_number,omitempty"`
	FormattedWANumber string    `json:"formatted_wa_number,omitempty"` // 62…; the wa.me link is built by the client
}

type Rundown struct {
	ID              uuid.UUID      `json:"id"`
	SessionID       uuid.UUID      `json:"session_id"`
	SessionName     string         `json:"session_name"`
	SessionIcon     string         `json:"session_icon"`
	ScopeKey        string         `json:"scope_key"`
	StartTime       string         `json:"start_time"`         // "HH:MM"
	EndTime         *string        `json:"end_time,omitempty"` // omitted when open
	EndsNextDay     bool           `json:"ends_next_day"`
	DurationMinutes *int           `json:"duration_minutes,omitempty"` // omitted when open
	ActivityName    string         `json:"activity_name"`
	PICName         string         `json:"pic_name"`
	Vendor          *RundownVendor `json:"vendor,omitempty"`
	Description     string         `json:"description"`
	Highlights      string         `json:"highlights"`
	Status          RundownStatus  `json:"status"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	LinkedSongs     int            `json:"linked_songs"` // songs that point at this agenda
	SortOrder       int            `json:"sort_order"`
	TemplateKey     string         `json:"template_key,omitempty"`
}

type RundownSessionSummary struct {
	ID          uuid.UUID `json:"id"`
	ScopeKey    string    `json:"scope_key"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Date        *string   `json:"date,omitempty"` // session_date "YYYY-MM-DD"
	AgendaCount int       `json:"agenda_count"`
	Completed   int       `json:"completed"`
	InProgress  int       `json:"in_progress"`
	SpanMinutes int       `json:"span_minutes"`          // the span of the session (BE 2.1); 0 without agendas
	FirstStart  *string   `json:"first_start,omitempty"` // "HH:MM"; omitted without agendas
	LastEnd     *string   `json:"last_end,omitempty"`    // "HH:MM" (the latest end; the start of an open agenda)
}

type RundownKPI struct {
	Sessions         int `json:"sessions"` // non-system sessions of the project
	TotalAgendas     int `json:"total_agendas"`
	Completed        int `json:"completed"`
	InProgress       int `json:"in_progress"`
	Pending          int `json:"pending"`
	TotalSpanMinutes int `json:"total_span_minutes"` // Σ span per session (BE 2.1)
	DistinctPICs     int `json:"distinct_pics"`      // non-empty pic_name ∪ linked vendors, without duplicates
}

// RundownOverview: kpi and sessions always describe the whole project; items follow ?session_id=.
type RundownOverview struct {
	Sessions []RundownSessionSummary `json:"sessions"`
	KPI      RundownKPI              `json:"kpi"`
	Items    []Rundown               `json:"items"`
}

type RundownLiveItem struct {
	ID          uuid.UUID     `json:"id"`
	Status      RundownStatus `json:"status"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// RundownLive is GET /rundowns/summary (polled every 5 seconds by the Hari-H screen).
type RundownLive struct {
	KPI        RundownKPI              `json:"kpi"`
	Sessions   []RundownSessionSummary `json:"sessions"`
	Items      []RundownLiveItem       `json:"items"`
	ServerTime time.Time               `json:"server_time"` // UTC
}

// ---------- playlist responses ----------

type PlaylistAgenda struct {
	ID           uuid.UUID `json:"id"`
	ActivityName string    `json:"activity_name"`
	SessionName  string    `json:"session_name"`
	StartTime    string    `json:"start_time"`
	EndTime      *string   `json:"end_time,omitempty"`
}

type Playlist struct {
	ID            uuid.UUID       `json:"id"`
	RundownID     *uuid.UUID      `json:"rundown_id,omitempty"`
	Agenda        *PlaylistAgenda `json:"agenda,omitempty"`
	SongTitle     string          `json:"song_title"`
	Artist        string          `json:"artist"`
	MomentTag     string          `json:"moment_tag"`
	AudioFormat   AudioFormat     `json:"audio_format"`
	SpotifyURL    string          `json:"spotify_url,omitempty"`
	YouTubeURL    string          `json:"youtube_url,omitempty"`
	AudioCueNotes string          `json:"audio_cue_notes,omitempty"`
	Status        AudioStatus     `json:"status"`
	SortOrder     int             `json:"sort_order"`
	TemplateKey   string          `json:"template_key,omitempty"`
}

type FormatCounts struct {
	DigitalPlayback int `json:"digital_playback"`
	LiveBand        int `json:"live_band"`
	Acoustic        int `json:"acoustic"`
	SoundFX         int `json:"sound_fx"`
}

type PlaylistKPI struct {
	TotalTracks   int          `json:"total_tracks"`
	ByFormat      FormatCounts `json:"by_format"`
	MappedMoments int          `json:"mapped_moments"` // distinct moment tags that have a song
	Ready         int          `json:"ready"`
	NeedsReview   int          `json:"needs_review"`
	MissingFile   int          `json:"missing_file"`
}

// PlaylistOverview: kpi always describes the whole project; items follow the moment_tag, format and status filters.
type PlaylistOverview struct {
	KPI   PlaylistKPI `json:"kpi"`
	Items []Playlist  `json:"items"`
}

// ---------- results ----------

type TemplateResult struct {
	Created  int         `json:"created"`
	Skipped  int         `json:"skipped"`
	Reset    int         `json:"reset"` // pending template agendas put back to the template (replace_template)
	Kept     int         `json:"kept"`  // template agendas in progress or completed, left alone
	Warnings []pd.Notice `json:"warnings"`
}

type WAExport struct {
	Text string `json:"text"`
}
