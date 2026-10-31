package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors the repository returns; the usecase maps them to apierr.
var (
	ErrNotFound         = errors.New("not found")
	ErrSessionNameTaken = errors.New("session name already used in this project")
	ErrScopeKeyTaken    = errors.New("session scope_key already used in this project")
	ErrInvalidTimezone  = errors.New("unknown time zone")
	// ErrInvalidScope: a row pointed at a session scope that does not exist
	// (foreign key on *_sub_event_scope_fkey).
	ErrInvalidScope = errors.New("unknown session scope")
	// ErrConflict: a foreign key, unique or serialization failure surfaced at
	// commit (the foreign keys into planner_event_sessions are deferred). Safe
	// to retry.
	ErrConflict = errors.New("conflicting concurrent change")
)

// TxManager runs fn in one transaction. The ctx passed to fn carries the
// transaction; repositories reach it through db.Querier, so the usecase never
// imports pgx. A nested InTx joins the outer transaction.
type TxManager interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Repository interface {
	// InsertProject stores p (filling CreatedAt/UpdatedAt). With a clientToken
	// that is already used by the same user nothing is inserted and
	// inserted == false. For a sample project the same applies when the user
	// already has one.
	InsertProject(ctx context.Context, p *Project, clientToken *string) (inserted bool, err error)
	// ProjectForUser returns ErrNotFound for an unknown project and for one
	// owned by somebody else.
	ProjectForUser(ctx context.Context, id, userID uuid.UUID) (Project, error)
	// LockProject reloads the project with SELECT … FOR UPDATE. Needs a transaction.
	LockProject(ctx context.Context, id uuid.UUID) (Project, error)
	ProjectByClientToken(ctx context.Context, userID uuid.UUID, token string) (Project, error)
	SampleProject(ctx context.Context, userID uuid.UUID) (Project, error)
	// CountProjects counts the user's projects without the sample.
	CountProjects(ctx context.Context, userID uuid.UUID) (int, error)

	// ListCards and ListLite compute the phase in SQL against now, in each
	// project's time zone. status is "all" or a ProjectStatus.
	ListCards(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]ProjectCardResponse, error)
	ListLite(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]ProjectLiteResponse, error)

	// UpdateProject writes every editable column of p.
	UpdateProject(ctx context.Context, p *Project) error
	SetProjectStatus(ctx context.Context, id uuid.UUID, status ProjectStatus) error
	DeleteProject(ctx context.Context, id uuid.UUID) error

	// Coupling with planner_tasks and session dates for date changes.
	CountShiftableTasks(ctx context.Context, projectID uuid.UUID) (int, error)
	ShiftTemplateTasks(ctx context.Context, projectID uuid.UUID, days int) (int, error)
	CountOverdueTemplateTasks(ctx context.Context, projectID uuid.UUID, today time.Time) (int, error)
	ShiftSessionDates(ctx context.Context, projectID uuid.UUID, from, to time.Time) error
	CountSessionsOnOtherDates(ctx context.Context, projectID uuid.UUID, date time.Time) (int, error)

	ListSessions(ctx context.Context, projectID uuid.UUID) ([]Session, error)
	ListSessionsWithUsage(ctx context.Context, projectID uuid.UUID) ([]SessionWithUsage, error)
	SessionByID(ctx context.Context, projectID, sessionID uuid.UUID) (Session, error)
	// InsertSession returns ErrSessionNameTaken or ErrScopeKeyTaken on a unique violation.
	InsertSession(ctx context.Context, s *Session) error
	// UpdateSession replaces name, icon, date, times and location (never scope_key).
	UpdateSession(ctx context.Context, s *Session) error
	ReorderSessions(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) error
	SessionUsage(ctx context.Context, projectID uuid.UUID, s Session) (SessionUsage, error)
	// MoveSessionData re-points every row of the project from scope `from` to
	// `to` (algorithm: 01 BE §4.6) and deletes the `from` session. Agendas of
	// the session are removed by the cascade.
	MoveSessionData(ctx context.Context, projectID uuid.UUID, from, to string) error
}

type SummaryProvider interface {
	// Key is one of timeline, finance, vendors, guests, admin_docs, rundown.
	Key() string
	Summarize(ctx context.Context, p Project) (any, error)
}

// ApplyMode: append skips the template_keys that are already there; replace_template redoes the template rows of the
// project (what that means is the owning module's rule: menu 06 resets the agendas that have not started, in place).
type ApplyMode string

const (
	ModeAppend          ApplyMode = "append"
	ModeReplaceTemplate ApplyMode = "replace_template"
)

type ApplyRequest struct {
	Mode ApplyMode
	// Confirm is the user's yes to a replace_template that overwrites their changes. The wizard only appends and
	// never sets it.
	Confirm             bool
	Project             Project
	TimelinePreset      *string
	IncludeAdicaraTasks bool
}

type ApplyResult struct {
	Created  int      `json:"created"`
	Skipped  int      `json:"skipped"`
	Warnings []Notice `json:"warnings"`
}

// TemplateApplier is implemented by every module that owns a template. Apply
// runs inside the project-creation transaction (ctx carries it).
type TemplateApplier interface {
	// Key is one of timeline, admin_docs, budget, shopping, rundown, playlist.
	Key() string
	Apply(ctx context.Context, projectID uuid.UUID, req ApplyRequest) (ApplyResult, error)
}

// SampleOverlay lays the Wedding_Planner.pdf dataset (payments, savings,
// vendors, guests, statuses) over the template data of the sample project. It
// runs only from POST /projects/sample and tests, inside the same transaction.
type SampleOverlay interface {
	Key() string
	ApplySample(ctx context.Context, p Project) error
}

// Template and provider keys.
const (
	TemplateTimeline  = "timeline"
	TemplateAdminDocs = "admin_docs"
	TemplateBudget    = "budget"
	TemplateShopping  = "shopping"
	TemplateRundown   = "rundown"
	TemplatePlaylist  = "playlist"
)

// TemplateKeys is the order templates are applied in: budget lines before the
// shopping list (which rolls up into them), sessions already exist.
var TemplateKeys = []string{TemplateTimeline, TemplateAdminDocs, TemplateBudget, TemplateShopping, TemplateRundown, TemplatePlaylist}

var SummaryKeys = []string{"timeline", "finance", "vendors", "guests", "admin_docs", "rundown"}
