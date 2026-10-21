package http

import (
	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

// taskResponse is a task as the API shows it: dates as YYYY-MM-DD. The vendor
// keys and template_key are omitted when empty.
type taskResponse struct {
	ID            uuid.UUID         `json:"id"`
	Title         string            `json:"title"`
	SubEventScope string            `json:"sub_event_scope"`
	Category      string            `json:"category"`
	PhaseOrder    int               `json:"phase_order"`
	PhaseName     string            `json:"phase_name"`
	StartDate     string            `json:"start_date"`
	EndDate       string            `json:"end_date"`
	DurationDays  int               `json:"duration_days"`
	Status        domain.TaskStatus `json:"status"`
	PIC           string            `json:"pic"`
	VendorID      *uuid.UUID        `json:"vendor_id,omitempty"`
	VendorName    *string           `json:"vendor_name,omitempty"`
	VendorWA      *string           `json:"vendor_wa,omitempty"`
	SortOrder     int               `json:"sort_order"`
	Notes         string            `json:"notes"`
	TemplateKey   *string           `json:"template_key,omitempty"`
	Origin        domain.Origin     `json:"origin"`
	Overdue       bool              `json:"overdue"`
}

func newTaskResponse(t domain.Task) taskResponse {
	return taskResponse{
		ID: t.ID, Title: t.Title, SubEventScope: t.SubEventScope, Category: t.Category, PhaseOrder: t.PhaseOrder,
		PhaseName: t.PhaseName, StartDate: t.StartDate.Format(pd.DateLayout), EndDate: t.EndDate.Format(pd.DateLayout),
		DurationDays: t.DurationDays, Status: t.Status, PIC: t.PIC, VendorID: t.VendorID, VendorName: t.VendorName,
		VendorWA: t.VendorWA, SortOrder: t.SortOrder, Notes: t.Notes, TemplateKey: t.TemplateKey, Origin: t.Origin, Overdue: t.Overdue,
	}
}

type taskListResponse struct {
	Tasks   []taskResponse     `json:"tasks"` // always an array
	Summary domain.TaskSummary `json:"summary"`
}

func newTaskListResponse(l domain.TaskList) taskListResponse {
	out := taskListResponse{Tasks: make([]taskResponse, len(l.Tasks)), Summary: l.Summary}
	for i, t := range l.Tasks {
		out.Tasks[i] = newTaskResponse(t)
	}

	return out
}

type docResponse struct {
	ID             uuid.UUID        `json:"id"`
	DocumentName   string           `json:"document_name"`
	SubjectRole    string           `json:"subject_role"`
	QuantityDetail string           `json:"quantity_detail"`
	NominalCost    int64            `json:"nominal_cost"`
	Status         domain.DocStatus `json:"status"`
	SortOrder      int              `json:"sort_order"`
	Notes          string           `json:"notes"`
	TemplateKey    *string          `json:"template_key,omitempty"`
	Origin         domain.Origin    `json:"origin"`
	BudgetItemID   *uuid.UUID       `json:"budget_item_id,omitempty"`
}

func newDocResponse(d domain.AdminDoc) docResponse {
	return docResponse{
		ID: d.ID, DocumentName: d.DocumentName, SubjectRole: d.SubjectRole, QuantityDetail: d.QuantityDetail,
		NominalCost: d.NominalCost, Status: d.Status, SortOrder: d.SortOrder, Notes: d.Notes,
		TemplateKey: d.TemplateKey, Origin: d.Origin, BudgetItemID: d.BudgetItemID,
	}
}

type docListResponse struct {
	Docs    []docResponse     `json:"docs"` // always an array
	Summary domain.DocSummary `json:"summary"`
}

func newDocListResponse(l domain.AdminDocList) docListResponse {
	out := docListResponse{Docs: make([]docResponse, len(l.Docs)), Summary: l.Summary}
	for i, d := range l.Docs {
		out.Docs[i] = newDocResponse(d)
	}

	return out
}

// notices makes a nil warnings slice an empty array.
func notices(n []pd.Notice) []pd.Notice {
	if n == nil {
		return []pd.Notice{}
	}

	return n
}
