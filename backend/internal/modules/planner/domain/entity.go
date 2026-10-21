// Package domain holds the planner (menu 01) entities, the event type registry
// and the interfaces the other planner modules plug into. Money is int64 whole
// rupiah; calendar dates (DATE) are time.Time at 00:00 UTC and appear in JSON
// as "YYYY-MM-DD"; times of day appear as "HH:MM".
package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

type ProjectStatus string

const (
	StatusActive    ProjectStatus = "active"
	StatusCompleted ProjectStatus = "completed"
	StatusArchived  ProjectStatus = "archived"
)

func (s ProjectStatus) Valid() bool {
	return s == StatusActive || s == StatusCompleted || s == StatusArchived
}

type EventPhase string

const (
	PhaseUpcoming EventPhase = "upcoming"
	PhaseOngoing  EventPhase = "ongoing"
	PhaseFinished EventPhase = "finished"
)

// DateLayout is the wire and column format of every calendar date.
const DateLayout = "2006-01-02"

// AllScope is the scope_key of the system session every project has.
const AllScope = "all"

// PhaseInfo is computed in the project's own time zone (see ComputePhase); the
// frontend never recomputes it from the device clock.
type PhaseInfo struct {
	DaysRemaining int        `json:"days_remaining"` // event_date − today; 0 or negative from Hari-H on
	Phase         EventPhase `json:"phase"`
	DayNumber     int        `json:"day_number"` // N-th day while ongoing, otherwise 0
	TotalDays     int        `json:"total_days"` // 1 for a single-day event
}

type Project struct {
	ID                      uuid.UUID
	UserID                  uuid.UUID
	EventType               EventType
	Title                   string
	EventDate               time.Time  // 00:00 UTC
	EventEndDate            *time.Time // nil = single day
	PrepStartDate           time.Time
	Timezone                string
	TargetBudget            int64
	SavingsTarget           *int64
	Status                  ProjectStatus
	IsSample                bool
	SkippedVendorCategories []string
	InvitationID            *uuid.UUID // nil = standalone mode
	Metadata                map[string]any
	CreatedAt, UpdatedAt    time.Time
}

// LastDay is the final day of the event (EventDate for single-day events).
func (p Project) LastDay() time.Time {
	if p.EventEndDate != nil {
		return *p.EventEndDate
	}

	return p.EventDate
}

// Session is a row of planner_event_sessions, the one master list of
// sub-events every planner module scopes its data by.
type Session struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	ScopeKey     string
	Name         string
	Icon         string
	IsSystem     bool
	SortOrder    int
	SessionDate  *time.Time
	StartTime    *string // "HH:MM"
	EndTime      *string
	LocationName *string
}

func (s Session) Brief() SessionBrief {
	b := SessionBrief{
		ID: s.ID, ScopeKey: s.ScopeKey, Name: s.Name, Icon: s.Icon, IsSystem: s.IsSystem, SortOrder: s.SortOrder,
		StartTime: s.StartTime, EndTime: s.EndTime, LocationName: s.LocationName,
	}
	if s.SessionDate != nil {
		d := s.SessionDate.Format(DateLayout)
		b.SessionDate = &d
	}

	return b
}

// SessionBrief is what the dashboard and the session switcher need.
type SessionBrief struct {
	ID           uuid.UUID `json:"id"`
	ScopeKey     string    `json:"scope_key"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon"`
	IsSystem     bool      `json:"is_system"`
	SortOrder    int       `json:"sort_order"`
	SessionDate  *string   `json:"session_date,omitempty"`
	StartTime    *string   `json:"start_time,omitempty"`
	EndTime      *string   `json:"end_time,omitempty"`
	LocationName *string   `json:"location_name,omitempty"`
}

// SessionUsage counts the rows that point at a session; the delete dialog
// shows it and session_in_use carries it.
type SessionUsage struct {
	Tasks         int `json:"tasks"`
	Vendors       int `json:"vendors"`
	BudgetItems   int `json:"budget_items"`
	ScenarioItems int `json:"scenario_items"`
	ShoppingItems int `json:"shopping_items"`
	Guests        int `json:"guests"`
	Agendas       int `json:"agendas"`
}

// SessionWithUsage is the GET /sessions row.
type SessionWithUsage struct {
	Session
	Usage SessionUsage
}

// ---------- Requests ----------

// InitProjectRequest creates a project. Validation lives in the usecase (it
// trims, then validates), so the struct carries no binding tags: an empty or
// blank string must reach the usecase to get a field-specific error.
type InitProjectRequest struct {
	EventType     EventType      `json:"event_type"`
	Title         string         `json:"title"`
	EventDate     string         `json:"event_date"` // YYYY-MM-DD
	EventEndDate  *string        `json:"event_end_date"`
	PrepStartDate *string        `json:"prep_start_date"` // default: today in the project time zone
	Timezone      *string        `json:"timezone"`        // canonical IANA name, default Asia/Jakarta
	TargetBudget  int64          `json:"target_budget"`
	Metadata      map[string]any `json:"metadata"`
	Templates     InitTemplates  `json:"templates"`
	ClientToken   *string        `json:"client_token"` // 8-64 chars; makes a retried request a no-op
}

// InitTemplates lists what to create besides the default sessions. A template
// the event type does not own, or whose module has no registered applier, is
// rejected with invalid_template.
type InitTemplates struct {
	TimelinePreset      *string `json:"timeline_preset"` // "wedding_pdf_44" | "wedding_essential_22" | "general"
	IncludeAdicaraTasks bool    `json:"include_adicara_tasks"`
	AdminDocs           bool    `json:"admin_docs"`
	BudgetLines         bool    `json:"budget_lines"`
	ShoppingList        bool    `json:"shopping_list"`
	Rundown             bool    `json:"rundown"`
	Playlist            bool    `json:"playlist"`
}

// UpdateProjectRequest is a PATCH: only the fields sent change. Pointers plus
// Clear* flags because JSON null and "not sent" must be told apart.
type UpdateProjectRequest struct {
	Title                   *string        `json:"title"`
	EventDate               *string        `json:"event_date"`
	EventEndDate            *string        `json:"event_end_date"`
	ClearEventEndDate       bool           `json:"clear_event_end_date"`
	PrepStartDate           *string        `json:"prep_start_date"`
	Timezone                *string        `json:"timezone"`
	TargetBudget            *int64         `json:"target_budget"`
	SavingsTarget           *int64         `json:"savings_target"`
	ClearSavingsTarget      bool           `json:"clear_savings_target"`
	SkippedVendorCategories *[]string      `json:"skipped_vendor_categories"`
	Metadata                map[string]any `json:"metadata"` // JSON Merge Patch: null deletes a key
	ShiftTasks              bool           `json:"shift_tasks"`
}

type ChangeProjectStatusRequest struct {
	Status ProjectStatus `json:"status"`
}

// CreateSessionRequest is also the PUT body, which replaces the session
// entirely: optional fields that are not sent are cleared.
type CreateSessionRequest struct {
	Name         string  `json:"name"`
	Icon         *string `json:"icon"`
	SessionDate  *string `json:"session_date"`
	StartTime    *string `json:"start_time"` // "HH:MM" or "HH.MM"
	EndTime      *string `json:"end_time"`
	LocationName *string `json:"location_name"`
}

type UpdateSessionRequest = CreateSessionRequest

type ReorderSessionsRequest struct {
	SessionIDs []uuid.UUID `json:"session_ids"` // every non-system session exactly once
}

// ---------- Hub ----------

type ProjectCardResponse struct {
	ID           uuid.UUID `json:"id"`
	EventType    EventType `json:"event_type"`
	Title        string    `json:"title"`
	EventDate    string    `json:"event_date"`
	EventEndDate *string   `json:"event_end_date,omitempty"`
	PhaseInfo
	Status         ProjectStatus  `json:"status"`
	IsSample       bool           `json:"is_sample"`
	TargetBudget   int64          `json:"target_budget"`
	TotalEstimate  int64          `json:"total_estimate"`
	TotalPaid      int64          `json:"total_paid"`
	TotalTasks     int            `json:"total_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	GuestPaxCount  int            `json:"guest_pax_count"`
	Metadata       map[string]any `json:"metadata"`
}

// ProjectLiteResponse feeds the project switcher: no aggregates.
type ProjectLiteResponse struct {
	ID            uuid.UUID     `json:"id"`
	EventType     EventType     `json:"event_type"`
	Title         string        `json:"title"`
	EventDate     string        `json:"event_date"`
	EventEndDate  *string       `json:"event_end_date,omitempty"`
	Status        ProjectStatus `json:"status"`
	IsSample      bool          `json:"is_sample"`
	DaysRemaining int           `json:"days_remaining"`
	Phase         EventPhase    `json:"phase"`
}

type PortfolioSummaryResponse struct {
	ActiveCount          int                   `json:"active_count"` // the sample project is never counted
	CompletedCount       int                   `json:"completed_count"`
	ArchivedCount        int                   `json:"archived_count"`
	CombinedTargetBudget int64                 `json:"combined_target_budget"` // active, non-sample
	NearestProject       *ProjectCardResponse  `json:"nearest_project,omitempty"`
	Projects             []ProjectCardResponse `json:"projects"` // all, sample included
}

// ---------- Create / update outcomes ----------

// Notice is a warning or hint attached to a successful response.
type Notice struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Count     int    `json:"count"`
	ShiftDays *int   `json:"shift_days,omitempty"`
}

type CreateProjectResult struct {
	Project Project
	Created bool
	Applied map[string]ApplyResult // only when Created
}

type UpdateProjectResult struct {
	Project  Project
	Warnings []Notice
	Hints    []Notice
}

type ProjectDetail struct {
	Project        Project
	ShiftableTasks int
}

// ---------- Dashboard ----------

type DashboardProject struct {
	ID           uuid.UUID `json:"id"`
	EventType    EventType `json:"event_type"`
	Title        string    `json:"title"`
	EventDate    string    `json:"event_date"`
	EventEndDate *string   `json:"event_end_date"`
	PhaseInfo
	Status           ProjectStatus  `json:"status"`
	IsSample         bool           `json:"is_sample"`
	InvitationLinked bool           `json:"invitation_linked"` // false = standalone mode
	TargetBudget     int64          `json:"target_budget"`
	SavingsTarget    *int64         `json:"savings_target"`
	Timezone         string         `json:"timezone"`
	Metadata         map[string]any `json:"metadata"`
}

// WidgetError replaces a widget whose provider failed.
type WidgetError struct {
	Error string `json:"error"` // internal | timeout
}

// WorkspaceDashboard: a widget is nil when no provider is registered for it,
// a WidgetError when its provider failed, otherwise the provider's summary.
type WorkspaceDashboard struct {
	ServerTime time.Time        `json:"server_time"`
	Project    DashboardProject `json:"project"`
	Sessions   []SessionBrief   `json:"sessions"` // without 'all' and without usage
	Timeline   any              `json:"timeline"`
	Finance    any              `json:"finance"`
	Vendors    any              `json:"vendors"`
	Guests     any              `json:"guests"`
	AdminDocs  any              `json:"admin_docs"`
	Rundown    any              `json:"rundown"`
}
