package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

// The module has two keys on menu 01's plug-in points (timeline and
// admin_docs) while each interface has a single Key(), so every role is a
// small type that wraps the Service. Register them in internal/app/app.go.

// ---------- dashboard widgets ----------

type timelineProvider struct{ s *Service }

func (timelineProvider) Key() string { return "timeline" }

func (p timelineProvider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	at, _, err := p.s.clock(project)
	if err != nil {
		return nil, err
	}

	return p.s.repo.TimelineSummary(ctx, project.ID, at)
}

type adminDocsProvider struct{ s *Service }

func (adminDocsProvider) Key() string { return "admin_docs" }

func (p adminDocsProvider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	completed, total, pending, err := p.s.repo.DocSummaryWidget(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	return pd.AdminDocSummary{Label: p.s.spec(project).AdminLabel, Completed: completed, Total: total, Pending: pending}, nil
}

func (s *Service) TimelineProvider() pd.SummaryProvider  { return timelineProvider{s} }
func (s *Service) AdminDocsProvider() pd.SummaryProvider { return adminDocsProvider{s} }

// ---------- project wizard templates ----------

type timelineApplier struct{ s *Service }

func (timelineApplier) Key() string { return pd.TemplateTimeline }

// Apply runs inside the project-creation transaction; the wizard only appends.
func (a timelineApplier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	preset := ""
	if req.TimelinePreset != nil {
		preset = *req.TimelinePreset
	}
	r, err := a.s.generate(ctx, req.Project, preset, req.IncludeAdicaraTasks, domain.ModeAppend, false)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

type adminDocsApplier struct{ s *Service }

func (adminDocsApplier) Key() string { return pd.TemplateAdminDocs }

func (a adminDocsApplier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	r, err := a.s.applyDocs(ctx, req.Project, domain.ModeAppend, false)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

func (s *Service) TimelineApplier() pd.TemplateApplier  { return timelineApplier{s} }
func (s *Service) AdminDocsApplier() pd.TemplateApplier { return adminDocsApplier{s} }

// ---------- sample project (Wedding_Planner.pdf) ----------

type timelineOverlay struct{ s *Service }

func (timelineOverlay) Key() string { return "timeline" }

// ApplySample gives the 44 PDF tasks the statuses of PDF p. 3 (15 done, 29 in
// progress; the six Adicara tasks stay belum) and dates the Lamaran session on
// the start of the Lamaran task, so the sample agrees with its own timeline.
func (o timelineOverlay) ApplySample(ctx context.Context, p pd.Project) error {
	if err := o.s.repo.ApplySampleTaskStatus(ctx, p.ID); err != nil {
		return err
	}

	return o.s.repo.SetSessionDateFromTask(ctx, p.ID, "lamaran", "wedding.lamaran")
}

type adminDocsOverlay struct{ s *Service }

func (adminDocsOverlay) Key() string { return "admin_docs" }

func (o adminDocsOverlay) ApplySample(ctx context.Context, p pd.Project) error {
	return o.s.repo.ApplySampleDocStatus(ctx, p.ID, domain.SampleDocsDone())
}

func (s *Service) TimelineOverlay() pd.SampleOverlay  { return timelineOverlay{s} }
func (s *Service) AdminDocsOverlay() pd.SampleOverlay { return adminDocsOverlay{s} }
