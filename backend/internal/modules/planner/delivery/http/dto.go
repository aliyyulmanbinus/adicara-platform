package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// projectResponse is a project as the API shows it: dates as YYYY-MM-DD and
// the phase computed by the server in the project's own time zone.
type projectResponse struct {
	ID            uuid.UUID        `json:"id"`
	EventType     domain.EventType `json:"event_type"`
	Title         string           `json:"title"`
	EventDate     string           `json:"event_date"`
	EventEndDate  *string          `json:"event_end_date"`
	PrepStartDate string           `json:"prep_start_date"`
	Timezone      string           `json:"timezone"`
	domain.PhaseInfo
	TargetBudget            int64                `json:"target_budget"`
	SavingsTarget           *int64               `json:"savings_target"`
	Status                  domain.ProjectStatus `json:"status"`
	IsSample                bool                 `json:"is_sample"`
	SkippedVendorCategories []string             `json:"skipped_vendor_categories"`
	InvitationID            *uuid.UUID           `json:"invitation_id"`
	Metadata                map[string]any       `json:"metadata"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
}

func newProjectResponse(p domain.Project, phase domain.PhaseInfo) projectResponse {
	r := projectResponse{
		ID: p.ID, EventType: p.EventType, Title: p.Title, EventDate: p.EventDate.Format(domain.DateLayout),
		PrepStartDate: p.PrepStartDate.Format(domain.DateLayout), Timezone: p.Timezone, PhaseInfo: phase,
		TargetBudget: p.TargetBudget, SavingsTarget: p.SavingsTarget, Status: p.Status, IsSample: p.IsSample,
		SkippedVendorCategories: p.SkippedVendorCategories, InvitationID: p.InvitationID, Metadata: p.Metadata,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if p.EventEndDate != nil {
		end := p.EventEndDate.Format(domain.DateLayout)
		r.EventEndDate = &end
	}
	if r.SkippedVendorCategories == nil {
		r.SkippedVendorCategories = []string{}
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}

	return r
}

// sessionResponse is a row of GET /sessions: the brief plus how many rows
// point at the session.
type sessionResponse struct {
	domain.SessionBrief
	Usage domain.SessionUsage `json:"usage"`
}

func newSessionResponses(in []domain.SessionWithUsage) []sessionResponse {
	out := make([]sessionResponse, len(in))
	for i, s := range in {
		out[i] = sessionResponse{SessionBrief: s.Brief(), Usage: s.Usage}
	}

	return out
}

func newSessionBriefs(in []domain.Session) []domain.SessionBrief {
	out := make([]domain.SessionBrief, len(in))
	for i, s := range in {
		out[i] = s.Brief()
	}

	return out
}
