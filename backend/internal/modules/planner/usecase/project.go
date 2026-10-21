package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// errTokenReused aborts the creation transaction when the client_token was
// already used: nothing has been written yet, so the rollback is empty.
var errTokenReused = errors.New("client token already used")

func invalidTemplate(field string) *apierr.Error {
	return apierr.BadRequest("invalid_template", field+": not available for this event type")
}

// validateTemplates rejects templates the type does not own or whose module
// has no registered applier (those are not offered by GET /event-types).
func (s *Service) validateTemplates(spec domain.EventTypeSpec, t domain.InitTemplates) error {
	av := spec.Templates.Available(s.hasApplier)
	switch {
	case t.TimelinePreset != nil && !contains(av.TimelinePresets, *t.TimelinePreset):
		return invalidTemplate("timeline_preset")
	case t.IncludeAdicaraTasks && !av.IncludeAdicaraTasks:
		return invalidTemplate("include_adicara_tasks")
	case t.AdminDocs && !av.AdminDocs:
		return invalidTemplate("admin_docs")
	case t.BudgetLines && !av.BudgetLines:
		return invalidTemplate("budget_lines")
	case t.ShoppingList && !av.ShoppingList:
		return invalidTemplate("shopping_list")
	case t.Rundown && !av.Rundown:
		return invalidTemplate("rundown")
	case t.Playlist && !av.Playlist:
		return invalidTemplate("playlist")
	}

	return nil
}

// CreateProject validates, then creates the project, its sessions and the
// requested templates in one transaction. A repeated client_token answers with
// the original project and Created == false, without running templates again.
func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, req domain.InitProjectRequest) (domain.CreateProjectResult, error) {
	spec, ok := s.registry.Get(req.EventType)
	if !ok {
		return domain.CreateProjectResult{}, invalidInput("event_type", "unknown event type")
	}

	title, err := cleanText("title", req.Title, maxTitleRunes, true)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	tz, loc, err := resolveTimezone(req.Timezone)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}

	today := domain.CalendarDay(s.now(), loc)
	eventDate, err := parseDate("event_date", req.EventDate)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	if eventDate.Before(today) {
		return domain.CreateProjectResult{}, invalidDate("event_date", "must not be before today")
	}
	if eventDate.After(today.AddDate(0, 0, maxEventAheadDays)) {
		return domain.CreateProjectResult{}, invalidDate("event_date", "must be within 5 years from today")
	}
	prep := today
	if req.PrepStartDate != nil {
		if prep, err = parseDate("prep_start_date", *req.PrepStartDate); err != nil {
			return domain.CreateProjectResult{}, err
		}
	}
	end, err := parseDatePtr("event_end_date", req.EventEndDate)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	if err := checkSchedule(prep, eventDate, end); err != nil {
		return domain.CreateProjectResult{}, err
	}

	if err := validateMoney("target_budget", req.TargetBudget); err != nil {
		return domain.CreateProjectResult{}, err
	}
	metadata, err := mergeMetadata(spec, nil, req.Metadata)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	if err := s.validateTemplates(spec, req.Templates); err != nil {
		return domain.CreateProjectResult{}, err
	}
	if err := validateClientToken(req.ClientToken); err != nil {
		return domain.CreateProjectResult{}, err
	}

	// A retry must get its project back even when the user is at the limit.
	if req.ClientToken != nil {
		existing, err := s.repo.ProjectByClientToken(ctx, userID, *req.ClientToken)
		if err == nil {
			return domain.CreateProjectResult{Project: existing}, nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.CreateProjectResult{}, err
		}
	}
	n, err := s.repo.CountProjects(ctx, userID)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	if n >= s.maxProjects {
		return domain.CreateProjectResult{}, apierr.Conflict("project_limit_reached", fmt.Sprintf("at most %d projects per account", s.maxProjects))
	}

	p := domain.Project{
		ID: uuid.New(), UserID: userID, EventType: spec.Key, Title: title,
		EventDate: eventDate, EventEndDate: end, PrepStartDate: prep, Timezone: tz,
		TargetBudget: req.TargetBudget, Status: domain.StatusActive,
		SkippedVendorCategories: []string{}, Metadata: metadata,
	}

	applied, err := s.createInTx(ctx, &p, spec, req.ClientToken, req.Templates, false)
	if errors.Is(err, errTokenReused) {
		existing, gerr := s.repo.ProjectByClientToken(ctx, userID, *req.ClientToken)
		if gerr != nil {
			return domain.CreateProjectResult{}, gerr
		}

		return domain.CreateProjectResult{Project: existing}, nil
	}
	if err != nil {
		return domain.CreateProjectResult{}, mapRepoErr(err)
	}

	return domain.CreateProjectResult{Project: p, Created: true, Applied: applied}, nil
}

// createInTx writes the project row, the default sessions and the requested
// templates atomically. With overlays it also lays the sample dataset over
// them. errTokenReused means another request won the insert.
func (s *Service) createInTx(ctx context.Context, p *domain.Project, spec domain.EventTypeSpec, token *string,
	t domain.InitTemplates, overlays bool) (map[string]domain.ApplyResult, error) {
	applied := map[string]domain.ApplyResult{}

	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		inserted, err := s.repo.InsertProject(ctx, p, token)
		if err != nil {
			return err
		}
		if !inserted {
			return errTokenReused
		}
		if err := s.seedSessions(ctx, *p, spec); err != nil {
			return err
		}
		if applied, err = s.applyTemplates(ctx, *p, t); err != nil {
			return err
		}
		if overlays {
			for _, o := range s.overlays {
				if err := o.ApplySample(ctx, *p); err != nil {
					return fmt.Errorf("sample overlay %s: %w", o.Key(), err)
				}
			}
		}

		return nil
	})

	return applied, err
}

// seedSessions creates the system session 'all' and the defaults of the type.
// This is unconditional: it does not depend on any template choice.
func (s *Service) seedSessions(ctx context.Context, p domain.Project, spec domain.EventTypeSpec) error {
	all := domain.Session{
		ID: uuid.New(), ProjectID: p.ID, ScopeKey: domain.AllScope, Name: "Umum / Semua Sesi", Icon: defaultSessionIcon,
		IsSystem: true, SortOrder: 0,
	}
	if err := s.repo.InsertSession(ctx, &all); err != nil {
		return err
	}

	for i, seed := range spec.Sessions {
		sess := domain.Session{
			ID: uuid.New(), ProjectID: p.ID, ScopeKey: seed.ScopeKey, Name: seed.Name, Icon: seed.Icon, SortOrder: i + 1,
		}
		if seed.SameDay {
			d := p.EventDate
			sess.SessionDate = &d
		}
		if err := s.repo.InsertSession(ctx, &sess); err != nil {
			return err
		}
	}

	return nil
}

// applyTemplates runs the requested appliers in dependency order.
func (s *Service) applyTemplates(ctx context.Context, p domain.Project, t domain.InitTemplates) (map[string]domain.ApplyResult, error) {
	want := map[string]bool{
		domain.TemplateTimeline:  t.TimelinePreset != nil || t.IncludeAdicaraTasks,
		domain.TemplateAdminDocs: t.AdminDocs,
		domain.TemplateBudget:    t.BudgetLines,
		domain.TemplateShopping:  t.ShoppingList,
		domain.TemplateRundown:   t.Rundown,
		domain.TemplatePlaylist:  t.Playlist,
	}

	applied := map[string]domain.ApplyResult{}
	for _, key := range domain.TemplateKeys {
		if !want[key] {
			continue
		}
		res, err := s.appliers[key].Apply(ctx, p.ID, domain.ApplyRequest{
			Mode: domain.ModeAppend, Project: p, TimelinePreset: t.TimelinePreset, IncludeAdicaraTasks: t.IncludeAdicaraTasks,
		})
		if err != nil {
			return nil, fmt.Errorf("apply template %s: %w", key, err)
		}
		if res.Warnings == nil {
			res.Warnings = []domain.Notice{}
		}
		applied[key] = res
	}

	return applied, nil
}

// ProjectDetail adds shiftable_tasks (template tasks not yet done) for the
// "shift the schedule?" dialog.
func (s *Service) ProjectDetail(ctx context.Context, p domain.Project) (domain.ProjectDetail, error) {
	n, err := s.repo.CountShiftableTasks(ctx, p.ID)
	if err != nil {
		return domain.ProjectDetail{}, err
	}

	return domain.ProjectDetail{Project: p, ShiftableTasks: n}, nil
}

// UpdateProject applies a PATCH. It runs under a row lock on the project so
// two tabs cannot interleave, and shifting tasks and session dates happens in
// the same transaction as the date change (atomic, and not repeatable by a
// double click, unlike a separate relative "shift" call).
func (s *Service) UpdateProject(ctx context.Context, projectID uuid.UUID, req domain.UpdateProjectRequest) (domain.UpdateProjectResult, error) {
	var res domain.UpdateProjectResult

	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		p, err := s.repo.LockProject(ctx, projectID)
		if err != nil {
			return err
		}
		spec, _ := s.registry.Get(p.EventType) // the CHECK on event_type guarantees it exists

		oldEvent := p.EventDate
		if req.Title != nil {
			if p.Title, err = cleanText("title", *req.Title, maxTitleRunes, true); err != nil {
				return err
			}
		}
		if req.Timezone != nil {
			if p.Timezone, _, err = resolveTimezone(req.Timezone); err != nil {
				return err
			}
		}
		zone, err := domain.LoadLocation(p.Timezone)
		if err != nil {
			return fmt.Errorf("project %s: %w", p.ID, err)
		}

		if req.EventDate != nil {
			if p.EventDate, err = parseDate("event_date", *req.EventDate); err != nil {
				return err
			}
		}
		delta := domain.DaysBetween(oldEvent, p.EventDate)

		if req.ClearEventEndDate && req.EventEndDate != nil {
			return invalidInput("clear_event_end_date", "cannot be combined with event_end_date")
		}
		switch {
		case req.ClearEventEndDate:
			p.EventEndDate = nil
		case req.EventEndDate != nil:
			if p.EventEndDate, err = parseDatePtr("event_end_date", req.EventEndDate); err != nil {
				return err
			}
		case p.EventEndDate != nil && delta != 0:
			// The duration stays: the end moves with Hari-H unless it was sent.
			shifted := p.EventEndDate.AddDate(0, 0, delta)
			p.EventEndDate = &shifted
		}
		if req.PrepStartDate != nil {
			if p.PrepStartDate, err = parseDate("prep_start_date", *req.PrepStartDate); err != nil {
				return err
			}
		}
		// A date in the past is allowed here (correcting data), but the
		// relations between the dates must hold.
		if err := checkSchedule(p.PrepStartDate, p.EventDate, p.EventEndDate); err != nil {
			return err
		}

		if req.TargetBudget != nil {
			if err := validateMoney("target_budget", *req.TargetBudget); err != nil {
				return err
			}
			p.TargetBudget = *req.TargetBudget
		}
		if req.ClearSavingsTarget && req.SavingsTarget != nil {
			return invalidInput("clear_savings_target", "cannot be combined with savings_target")
		}
		switch {
		case req.ClearSavingsTarget:
			p.SavingsTarget = nil
		case req.SavingsTarget != nil:
			if err := validateMoney("savings_target", *req.SavingsTarget); err != nil {
				return err
			}
			v := *req.SavingsTarget
			p.SavingsTarget = &v
		}
		if req.SkippedVendorCategories != nil {
			if p.SkippedVendorCategories, err = validateSkippedCategories(*req.SkippedVendorCategories); err != nil {
				return err
			}
		}
		if req.Metadata != nil {
			if p.Metadata, err = mergeMetadata(spec, p.Metadata, req.Metadata); err != nil {
				return err
			}
		}

		if err := s.repo.UpdateProject(ctx, &p); err != nil {
			return err
		}
		if delta != 0 {
			if res.Warnings, res.Hints, err = s.followDateChange(ctx, p, oldEvent, delta, req.ShiftTasks, zone); err != nil {
				return err
			}
		}
		res.Project = p

		return nil
	})
	if err != nil {
		return domain.UpdateProjectResult{}, mapRepoErr(err)
	}

	return res, nil
}

// followDateChange moves what hangs on Hari-H when it changes: sessions dated
// on the old Hari-H follow it, template tasks follow only when asked.
func (s *Service) followDateChange(ctx context.Context, p domain.Project, oldEvent time.Time, delta int, shift bool, loc *time.Location) (warnings, hints []domain.Notice, err error) {
	warnings, hints = []domain.Notice{}, []domain.Notice{}

	if err := s.repo.ShiftSessionDates(ctx, p.ID, oldEvent, p.EventDate); err != nil {
		return nil, nil, err
	}
	other, err := s.repo.CountSessionsOnOtherDates(ctx, p.ID, p.EventDate)
	if err != nil {
		return nil, nil, err
	}
	if other > 0 {
		hints = append(hints, domain.Notice{
			Code: "sessions_dates_unchanged", Count: other,
			Message: "sessions on dates other than Hari-H keep their dates",
		})
	}

	shiftable, err := s.repo.CountShiftableTasks(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	if shiftable == 0 {
		return warnings, hints, nil
	}
	if !shift {
		days := delta
		hints = append(hints, domain.Notice{
			Code: "tasks_not_shifted", Count: shiftable, ShiftDays: &days,
			Message: "template tasks keep their dates; send shift_tasks to move them with Hari-H",
		})

		return warnings, hints, nil
	}

	if _, err := s.repo.ShiftTemplateTasks(ctx, p.ID, delta); err != nil {
		return nil, nil, err
	}
	overdue, err := s.repo.CountOverdueTemplateTasks(ctx, p.ID, domain.CalendarDay(s.now(), loc))
	if err != nil {
		return nil, nil, err
	}
	if overdue > 0 {
		warnings = append(warnings, domain.Notice{
			Code: "tasks_overdue_after_shift", Count: overdue,
			Message: "some shifted tasks are now past their due date",
		})
	}

	return warnings, hints, nil
}

// ChangeStatus moves a project between active, completed and archived without
// further conditions.
func (s *Service) ChangeStatus(ctx context.Context, p domain.Project, status domain.ProjectStatus) (domain.Project, error) {
	if !status.Valid() {
		return domain.Project{}, invalidInput("status", "must be one of active, completed, archived")
	}
	if err := s.repo.SetProjectStatus(ctx, p.ID, status); err != nil {
		return domain.Project{}, mapRepoErr(err)
	}

	return s.ProjectForUser(ctx, p.ID, p.UserID)
}

// DeleteProject removes the project and everything under it (CASCADE). confirm
// must equal the stored title exactly, to prevent a stray click.
func (s *Service) DeleteProject(ctx context.Context, p domain.Project, confirm string) error {
	if confirm != p.Title {
		return invalidInput("confirm", "must match the project title exactly")
	}

	return mapRepoErr(s.repo.DeleteProject(ctx, p.ID))
}

// ---------- Hub ----------

func validListStatus(status string) (string, error) {
	switch status {
	case "", "all":
		return "all", nil
	case string(domain.StatusActive), string(domain.StatusCompleted), string(domain.StatusArchived):
		return status, nil
	}

	return "", invalidInput("status", "must be one of active, completed, archived, all")
}

func (s *Service) ListCards(ctx context.Context, userID uuid.UUID, status string) ([]domain.ProjectCardResponse, error) {
	status, err := validListStatus(status)
	if err != nil {
		return nil, err
	}

	return s.repo.ListCards(ctx, userID, status, s.now())
}

// ListLite is the project switcher query: no aggregates.
func (s *Service) ListLite(ctx context.Context, userID uuid.UUID, status string) ([]domain.ProjectLiteResponse, error) {
	status, err := validListStatus(status)
	if err != nil {
		return nil, err
	}

	return s.repo.ListLite(ctx, userID, status, s.now())
}

// Summary builds the portfolio metrics. The sample project appears in
// Projects but is never counted in a number.
func (s *Service) Summary(ctx context.Context, userID uuid.UUID) (domain.PortfolioSummaryResponse, error) {
	cards, err := s.repo.ListCards(ctx, userID, "all", s.now())
	if err != nil {
		return domain.PortfolioSummaryResponse{}, err
	}

	out := domain.PortfolioSummaryResponse{Projects: cards}
	for i := range cards {
		c := cards[i]
		if c.IsSample {
			continue
		}
		switch c.Status {
		case domain.StatusActive:
			out.ActiveCount++
			out.CombinedTargetBudget += c.TargetBudget
			// Cards come sorted by date for active projects, so the first
			// match is the nearest; an ongoing event sorts before a later one.
			if out.NearestProject == nil && c.Phase != domain.PhaseFinished {
				nearest := c
				out.NearestProject = &nearest
			}
		case domain.StatusCompleted:
			out.CompletedCount++
		case domain.StatusArchived:
			out.ArchivedCount++
		}
	}

	return out, nil
}
