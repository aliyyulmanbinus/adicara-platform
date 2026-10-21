package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

func invalidTemplate(field string) *apierr.Error {
	return apierr.BadRequest("invalid_template", field+": not available for this event type")
}

func confirmRequired() *apierr.Error { return invalidInput("confirm", "required") }

// parseMode defaults to append. replaceAll is false for documents, which have
// no "replace everything" mode.
func parseMode(raw string, replaceAll bool) (string, error) {
	switch mode := strings.TrimSpace(raw); {
	case mode == "":
		return domain.ModeAppend, nil
	case mode == domain.ModeAppend, mode == domain.ModeReplaceTemplate, mode == domain.ModeReplaceAll && replaceAll:
		return mode, nil
	}
	if replaceAll {
		return "", invalidInput("mode", "must be one of append, replace_template, replace_all")
	}

	return "", invalidInput("mode", "must be one of append, replace_template")
}

// GenerateDefaults is POST /tasks/generate-defaults: it lays a schedule
// template over the project. The preset must be one the event type owns.
func (s *Service) GenerateDefaults(ctx context.Context, p pd.Project, req domain.GenerateTimelineRequest) (domain.GenerateResult, error) {
	mode, err := parseMode(req.Mode, true)
	if err != nil {
		return domain.GenerateResult{}, err
	}
	preset := strings.TrimSpace(req.Preset)
	if preset == "" {
		return domain.GenerateResult{}, invalidInput("preset", "required")
	}
	spec, _ := s.registry.Get(p.EventType)
	if !containsString(spec.Templates.TimelinePresets, preset) {
		return domain.GenerateResult{}, invalidTemplate("preset")
	}

	return s.generate(ctx, p, preset, req.IncludeAdicaraTasks, mode, req.Confirm)
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}

	return false
}

// generate resolves the template, scales its offsets to the project and
// writes the tasks, all in one transaction under the project lock. When called
// from the project wizard the transaction is the creation one (it is joined).
//
// The result of a call is all or nothing: a limit or confirm error rolls back
// the deletes as well.
func (s *Service) generate(ctx context.Context, p pd.Project, preset string, includeAdicara bool, mode string, confirm bool) (domain.GenerateResult, error) {
	sched, ok := domain.BuildSchedule(preset, p.EventType, includeAdicara)
	if !ok {
		return domain.GenerateResult{}, invalidTemplate("preset")
	}
	spec, _ := s.registry.Get(p.EventType)
	_, loc, err := s.clock(p)
	if err != nil {
		return domain.GenerateResult{}, err
	}
	plan, ok := domain.NewPlan(p.EventDate, p.PrepStartDate, pd.CalendarDay(s.now(), loc), spec.LeadTimeDays, sched.Horizon)
	if !ok {
		return domain.GenerateResult{}, invalidDate("event_date", "must not be before today")
	}

	res := domain.GenerateResult{Warnings: []pd.Notice{}}
	fallback := 0
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		scopes, _, err := s.scopeSet(ctx, p.ID)
		if err != nil {
			return err
		}
		existing, err := s.repo.CountTasks(ctx, p.ID)
		if err != nil {
			return err
		}

		var present map[string]bool
		switch mode {
		case domain.ModeAppend:
			if present, err = s.repo.TemplateTaskKeys(ctx, p.ID); err != nil {
				return err
			}
		case domain.ModeReplaceTemplate:
			progress, err := s.repo.CountProgressTasks(ctx, p.ID)
			if err != nil {
				return err
			}
			if progress > 0 && !confirm {
				return confirmRequired()
			}
			if res.Deleted, err = s.repo.DeleteTemplateTasks(ctx, p.ID); err != nil {
				return err
			}
		case domain.ModeReplaceAll:
			if existing > 0 && !confirm {
				return confirmRequired()
			}
			if res.Deleted, err = s.repo.DeleteAllTasks(ctx, p.ID); err != nil {
				return err
			}
		}

		var todo []domain.TaskTemplate
		for _, t := range sched.Tasks {
			if present[t.Key] {
				res.Skipped++
			} else {
				todo = append(todo, t)
			}
		}
		if existing-res.Deleted+len(todo) > maxTasks {
			return apierr.Conflict("task_limit_reached", fmt.Sprintf("at most %d tasks per project", maxTasks))
		}

		next, err := s.repo.NextTaskSortOrder(ctx, p.ID)
		if err != nil {
			return err
		}
		for i, tt := range todo {
			scope, fell := tt.Scope, false
			if !scopes[scope] {
				scope, fell = pd.AllScope, true
			}
			start, end := plan.At(tt.Start, tt.End)
			key := tt.Key
			t := domain.Task{
				ID: uuid.New(), ProjectID: p.ID, Title: tt.Title, SubEventScope: scope, Category: tt.Category,
				PhaseOrder: tt.Phase, PhaseName: sched.PhaseName(tt.Phase), StartDate: start, EndDate: end,
				Status: domain.TaskBelum, SortOrder: next + i, TemplateKey: &key, Origin: tt.Origin,
			}
			inserted, err := s.repo.InsertTemplateTask(ctx, &t)
			if err != nil {
				return err
			}
			switch {
			case !inserted:
				res.Skipped++ // another request created it between the read and the insert
			case fell:
				res.Created++
				fallback++
			default:
				res.Created++
			}
		}

		return nil
	})
	if err != nil {
		return domain.GenerateResult{}, mapErr(err, "task")
	}

	if res.Created > 0 {
		switch {
		case plan.TooShort:
			res.Warnings = append(res.Warnings, notice("lead_time_too_short", "less than 3 days until the event: every task is placed between today and Hari-H", res.Created))
		case plan.Squeezed:
			res.Warnings = append(res.Warnings, notice("lead_time_compressed", "less time than the standard lead time: the schedule was compressed", res.Created))
		}
		if fallback > 0 {
			res.Warnings = append(res.Warnings, notice("scope_fallback", "the session of some tasks no longer exists: they were placed under Umum", fallback))
		}
	}

	return res, nil
}
