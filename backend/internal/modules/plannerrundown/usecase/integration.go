package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// The module sits on three of menu 01's plug-in points. Each key of the host belongs to one role, so every role is a
// small type that wraps the Service (as in menus 02-04). Register them in internal/app/app.go; there is no sample
// overlay: the sample project holds the template (33 agendas and 30 songs), all pending, with no vendor or song link.
//
//	provider  "rundown"   the Hari-H widget of the dashboard
//	applier   "rundown"   the wizard template (the agendas of the type's sessions)
//	applier   "playlist"  the wizard template (the 30 songs of the wedding)

// ---------- dashboard widget ----------

type provider struct{ s *Service }

func (provider) Key() string { return "rundown" }

// Summarize is the rundown widget (01 BE 5.2): the counts, the agenda in progress and the next session.
func (p provider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	today, err := p.s.today(project)
	if err != nil {
		return nil, err
	}

	return p.s.repo.Dashboard(ctx, project.ID, today)
}

func (s *Service) Provider() pd.SummaryProvider { return provider{s} }

// ---------- project wizard templates ----------

type rundownApplier struct{ s *Service }

func (rundownApplier) Key() string { return pd.TemplateRundown }

// Apply runs inside the project-creation transaction; the wizard only appends.
func (a rundownApplier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	r, err := a.s.applyAgendaTemplate(ctx, req.Project, domain.ModeAppend, false)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

func (s *Service) RundownApplier() pd.TemplateApplier { return rundownApplier{s} }

type playlistApplier struct{ s *Service }

func (playlistApplier) Key() string { return pd.TemplatePlaylist }

func (a playlistApplier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	r, err := a.s.applyPlaylistTemplate(ctx, req.Project)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

func (s *Service) PlaylistApplier() pd.TemplateApplier { return playlistApplier{s} }
