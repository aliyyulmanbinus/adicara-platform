package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ErrNotFound, ErrInvalidScope and ErrConflict are the sentinels of the
// planner module (the shared TxManager already returns pd.ErrConflict for a
// failed COMMIT); they are reused so one mapping serves both.
var (
	ErrNotFound     = pd.ErrNotFound
	ErrInvalidScope = pd.ErrInvalidScope
	ErrConflict     = pd.ErrConflict
	// ErrInvalidVendor: a task pointed at a vendor of another project (the
	// composite foreign key planner_tasks_vendor_id_project_id_fkey).
	ErrInvalidVendor = errors.New("vendor does not belong to the project")
)

// BudgetItemRef is the part of a planner_budget_items row that the sync needs.
type BudgetItemRef struct {
	ID       uuid.UUID
	Estimate int64
}

// Repository is the PostgreSQL side of the module. Every method that takes a
// projectID filters by it; ids that come from a URL are always looked up with
// both (WHERE id = $x AND project_id = $y).
type Repository interface {
	// LockProject takes SELECT … FOR UPDATE on the project row. Needs a transaction.
	LockProject(ctx context.Context, projectID uuid.UUID) error
	// Sessions lists the sessions of the project (scope_key and name), in order.
	Sessions(ctx context.Context, projectID uuid.UUID) ([]SessionRef, error)
	VendorInProject(ctx context.Context, projectID, vendorID uuid.UUID) (bool, error)

	// ---------- tasks ----------
	ListTasks(ctx context.Context, projectID uuid.UUID, f TaskFilter, at At) ([]Task, error)
	SummarizeTasks(ctx context.Context, projectID uuid.UUID, at At) (TaskSummary, error)
	TaskByID(ctx context.Context, projectID, id uuid.UUID, at At) (Task, error)
	CountTasks(ctx context.Context, projectID uuid.UUID) (int, error)
	NextTaskSortOrder(ctx context.Context, projectID uuid.UUID) (int, error)
	// LastTaskPhase is the phase of the task with the highest sort_order.
	LastTaskPhase(ctx context.Context, projectID uuid.UUID) (order int, name string, found bool, err error)
	// PhaseName is the name another task already gives to phase `order`.
	PhaseName(ctx context.Context, projectID uuid.UUID, order int) (name string, found bool, err error)
	InsertTask(ctx context.Context, t *Task) error
	// UpdateTask writes every editable column; ErrNotFound when no row matches.
	UpdateTask(ctx context.Context, t *Task) error
	SetTaskStatus(ctx context.Context, projectID, id uuid.UUID, status TaskStatus) error
	DeleteTask(ctx context.Context, projectID, id uuid.UUID) error
	// ReorderTasks writes sort_order = 1..n in the order of ids and returns how
	// many rows of this project matched.
	ReorderTasks(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (changed int, err error)
	// ShiftBounds describes the tasks a manual shift moves: template tasks that are not done.
	ShiftBounds(ctx context.Context, projectID uuid.UUID) (n int, minStart, maxEnd time.Time, err error)
	ShiftTasks(ctx context.Context, projectID uuid.UUID, days int) (int, error)
	CountOverdueShifted(ctx context.Context, projectID uuid.UUID, today time.Time) (int, error)

	// Template tasks.
	TemplateTaskKeys(ctx context.Context, projectID uuid.UUID) (map[string]bool, error)
	// InsertTemplateTask inserts unless the template_key already exists in the project.
	InsertTemplateTask(ctx context.Context, t *Task) (inserted bool, err error)
	// CountProgressTasks counts template tasks that would lose progress when replaced:
	// status other than belum, or linked to a vendor.
	CountProgressTasks(ctx context.Context, projectID uuid.UUID) (int, error)
	DeleteTemplateTasks(ctx context.Context, projectID uuid.UUID) (int, error)
	DeleteAllTasks(ctx context.Context, projectID uuid.UUID) (int, error)

	// ---------- documents ----------
	ListDocs(ctx context.Context, projectID uuid.UUID, f DocFilter) ([]AdminDoc, error)
	SummarizeDocs(ctx context.Context, projectID uuid.UUID) (DocSummary, error)
	DocByID(ctx context.Context, projectID, id uuid.UUID) (AdminDoc, error)
	// LockDoc is DocByID with FOR UPDATE. Needs a transaction.
	LockDoc(ctx context.Context, projectID, id uuid.UUID) (AdminDoc, error)
	CountDocs(ctx context.Context, projectID uuid.UUID) (int, error)
	NextDocSortOrder(ctx context.Context, projectID uuid.UUID) (int, error)
	InsertDoc(ctx context.Context, d *AdminDoc) error
	UpdateDoc(ctx context.Context, d *AdminDoc) error
	SetDocStatus(ctx context.Context, projectID, id uuid.UUID, status DocStatus) error
	DeleteDoc(ctx context.Context, projectID, id uuid.UUID) error
	ReorderDocs(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (changed int, err error)

	DocTemplateKeys(ctx context.Context, projectID uuid.UUID) (map[string]bool, error)
	InsertTemplateDoc(ctx context.Context, d *AdminDoc) (inserted bool, err error)
	// DocsProgress counts template documents that are done (finished) and
	// template documents a budget line points at (linked).
	DocsProgress(ctx context.Context, projectID uuid.UUID) (finished, linked int, err error)
	DeleteTemplateDocs(ctx context.Context, projectID uuid.UUID) (int, error)

	// ---------- sync with the budget (tables of menu 03) ----------
	LinkedBudgetItem(ctx context.Context, projectID, docID uuid.UUID) (BudgetItemRef, bool, error)
	// UnlinkedBudgetItem finds the line (scope, item_key) that no document points at, FOR UPDATE.
	UnlinkedBudgetItem(ctx context.Context, projectID uuid.UUID, scope, itemKey string) (BudgetItemRef, bool, error)
	// LinkBudgetItem points the line at the document and fills a zero estimate with cost.
	LinkBudgetItem(ctx context.Context, projectID, itemID, docID uuid.UUID, cost int64) error
	InsertBudgetItem(ctx context.Context, projectID uuid.UUID, scope, name string, docID uuid.UUID, cost int64) (uuid.UUID, error)

	// ---------- dashboard widgets and the sample dataset ----------
	TimelineSummary(ctx context.Context, projectID uuid.UUID, at At) (pd.TimelineSummary, error)
	DocSummaryWidget(ctx context.Context, projectID uuid.UUID) (completed, total int, pending []string, err error)
	// ApplySampleTaskStatus sets the PDF statuses on the pdf-origin tasks (by sort_order).
	ApplySampleTaskStatus(ctx context.Context, projectID uuid.UUID) error
	ApplySampleDocStatus(ctx context.Context, projectID uuid.UUID, doneKeys []string) error
	// SetSessionDateFromTask dates the session `scope` on the start date of the task with templateKey.
	SetSessionDateFromTask(ctx context.Context, projectID uuid.UUID, scope, templateKey string) error
}
