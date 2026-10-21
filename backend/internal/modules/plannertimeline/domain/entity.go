// Package domain holds the entities, requests and templates of planner menu 02
// (timeline and administrative documents). Money is int64 whole rupiah and
// calendar dates (DATE) are time.Time at 00:00 UTC; delivery/http renders them as
// "YYYY-MM-DD". Request structs carry no binding tags: validation lives in the
// usecase so a blank string reaches it and gets a field-specific error.
package domain

import (
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

type TaskStatus string

const (
	TaskBelum   TaskStatus = "belum"
	TaskProses  TaskStatus = "proses"
	TaskSelesai TaskStatus = "selesai"
)

func (s TaskStatus) Valid() bool { return s == TaskBelum || s == TaskProses || s == TaskSelesai }

type DocStatus string

const (
	DocBelum   DocStatus = "belum"
	DocSelesai DocStatus = "selesai"
)

func (s DocStatus) Valid() bool { return s == DocBelum || s == DocSelesai }

type Origin string

const (
	OriginPDF     Origin = "pdf"
	OriginAdicara Origin = "adicara"
	OriginCustom  Origin = "custom"
)

// Generation modes (POST /tasks/generate-defaults and /admin-docs/apply-template).
const (
	ModeAppend          = "append"
	ModeReplaceTemplate = "replace_template"
	ModeReplaceAll      = "replace_all"
)

// ---------- tasks ----------

// Task is a row of planner_tasks plus what the list joins in. Overdue is
// computed by the server in the project's time zone.
type Task struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	Title         string
	SubEventScope string
	Category      string
	PhaseOrder    int
	PhaseName     string
	StartDate     time.Time
	EndDate       time.Time
	DurationDays  int // generated column: max(end - start, 1)
	Status        TaskStatus
	PIC           string
	VendorID      *uuid.UUID
	VendorName    *string
	VendorWA      *string
	SortOrder     int
	Notes         string
	TemplateKey   *string
	Origin        Origin
	Overdue       bool
}

// TaskSummary counts the whole project, never the filtered list. Field names
// match pd.TimelineSummary (the dashboard widget) plus NotStarted.
type TaskSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	NotStarted int `json:"not_started"`
	Overdue    int `json:"overdue"`
}

type TaskList struct {
	Tasks   []Task
	Summary TaskSummary
}

// TaskFilter is the query of GET /tasks. Scope nil means every session; "all"
// means only the general (Umum) tasks.
type TaskFilter struct {
	Status   string
	Category string
	Scope    *string
	Phase    *int
}

type CreateTaskRequest struct {
	Title         string     `json:"title"`
	SubEventScope string     `json:"sub_event_scope"` // blank = "all"
	Category      string     `json:"category"`        // blank = "Umum"
	PhaseOrder    int        `json:"phase_order"`     // 0 = follow the task with the highest sort_order
	PhaseName     string     `json:"phase_name"`
	StartDate     string     `json:"start_date"`
	EndDate       string     `json:"end_date"`
	PIC           string     `json:"pic"`
	VendorID      *uuid.UUID `json:"vendor_id"`
	Notes         string     `json:"notes"`
}

// UpdateTaskRequest is a PATCH: only the fields sent change. ClearVendor
// exists because JSON null and "not sent" must be told apart.
type UpdateTaskRequest struct {
	Title         *string     `json:"title"`
	SubEventScope *string     `json:"sub_event_scope"`
	Category      *string     `json:"category"`
	PhaseOrder    *int        `json:"phase_order"`
	PhaseName     *string     `json:"phase_name"`
	StartDate     *string     `json:"start_date"`
	EndDate       *string     `json:"end_date"`
	PIC           *string     `json:"pic"`
	Notes         *string     `json:"notes"`
	Status        *TaskStatus `json:"status"`
	VendorID      *uuid.UUID  `json:"vendor_id"`
	ClearVendor   bool        `json:"clear_vendor"`
}

type UpdateTaskStatusRequest struct {
	Status TaskStatus `json:"status"`
}

// ReorderRequest serves tasks and documents: every id of the project, once.
type ReorderRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

type ShiftTasksRequest struct {
	Days int `json:"days"`
}

type ShiftTasksResponse struct {
	Shifted  int         `json:"shifted"`
	Warnings []pd.Notice `json:"warnings"`
}

type GenerateTimelineRequest struct {
	Preset              string `json:"preset"`
	IncludeAdicaraTasks bool   `json:"include_adicara_tasks"`
	Mode                string `json:"mode"` // blank = append
	Confirm             bool   `json:"confirm"`
}

// GenerateResult serves POST /tasks/generate-defaults and /admin-docs/apply-template.
type GenerateResult struct {
	Created  int         `json:"created"`
	Skipped  int         `json:"skipped"`
	Deleted  int         `json:"deleted"`
	Warnings []pd.Notice `json:"warnings"`
}

// ---------- documents ----------

type AdminDoc struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	DocumentName   string
	SubjectRole    string
	QuantityDetail string
	NominalCost    int64
	Status         DocStatus
	SortOrder      int
	Notes          string
	TemplateKey    *string
	Origin         Origin
	BudgetItemID   *uuid.UUID // the budget line that points at this document
}

type AdminDocList struct {
	Docs    []AdminDoc
	Summary DocSummary
}

// DocSummary counts every document of the project, not the filtered list.
type DocSummary struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
}

type DocFilter struct {
	Subject string
	Status  string
}

type CreateAdminDocRequest struct {
	DocumentName   string `json:"document_name"`
	SubjectRole    string `json:"subject_role"` // blank = "bersama" (wedding) or the first subject of the type
	QuantityDetail string `json:"quantity_detail"`
	NominalCost    int64  `json:"nominal_cost"`
	Notes          string `json:"notes"`
}

type UpdateAdminDocRequest struct {
	DocumentName   *string `json:"document_name"`
	SubjectRole    *string `json:"subject_role"`
	QuantityDetail *string `json:"quantity_detail"`
	NominalCost    *int64  `json:"nominal_cost"`
	Notes          *string `json:"notes"`
}

type UpdateAdminDocStatusRequest struct {
	Status DocStatus `json:"status"`
}

type ApplyDocsRequest struct {
	Mode    string `json:"mode"` // append | replace_template; blank = append
	Confirm bool   `json:"confirm"`
}

type SyncBudgetRequest struct {
	SubEventScope *string `json:"sub_event_scope"`
}

type SyncBudgetResponse struct {
	BudgetItemID uuid.UUID   `json:"budget_item_id"`
	Created      bool        `json:"created"`
	Linked       bool        `json:"linked"`
	Warnings     []pd.Notice `json:"warnings"`
}

// ---------- export ----------

// SessionRef is the part of a session the export and the scope check need.
type SessionRef struct {
	ScopeKey string
	Name     string
}

// ExportData is everything the spreadsheet is built from: all tasks of the
// project, in sort_order, and the session names.
type ExportData struct {
	Project  pd.Project
	Tasks    []Task
	Sessions []SessionRef
	Today    time.Time // calendar date in the project's zone
}

// At carries the clock of a read: the time zone of the project and "now"
// (a parameter, not now(), so midnight boundaries can be tested).
type At struct {
	Zone string
	Now  time.Time
}
