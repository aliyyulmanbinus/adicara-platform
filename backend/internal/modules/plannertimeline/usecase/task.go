package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

// TaskQuery is the raw query string of GET /tasks; an empty value means "not given".
type TaskQuery struct {
	Status, Category, Scope, Phase string
}

func (q TaskQuery) filter() (domain.TaskFilter, error) {
	var f domain.TaskFilter
	if q.Status != "" {
		if !domain.TaskStatus(q.Status).Valid() {
			return f, invalidInput("status", "must be one of belum, proses, selesai")
		}
		f.Status = q.Status
	}
	if q.Category != "" {
		c, err := cleanText("category", q.Category, maxCategoryRunes, false)
		if err != nil {
			return f, err
		}
		f.Category = c
	}
	if q.Scope != "" {
		if !scopePattern.MatchString(q.Scope) {
			return f, invalidInput("scope", "must be a session scope_key")
		}
		f.Scope = &q.Scope
	}
	if q.Phase != "" {
		n, err := strconv.Atoi(q.Phase)
		if err != nil || n < minPhaseOrder || n > maxPhaseOrder {
			return f, invalidInput("phase", fmt.Sprintf("must be between %d and %d", minPhaseOrder, maxPhaseOrder))
		}
		f.Phase = &n
	}

	return f, nil
}

// ListTasks returns the filtered tasks in sort_order; the summary always counts
// every task of the project.
func (s *Service) ListTasks(ctx context.Context, p pd.Project, q TaskQuery) (domain.TaskList, error) {
	f, err := q.filter()
	if err != nil {
		return domain.TaskList{}, err
	}
	at, _, err := s.clock(p)
	if err != nil {
		return domain.TaskList{}, err
	}

	return s.taskList(ctx, p.ID, f, at)
}

func (s *Service) taskList(ctx context.Context, projectID uuid.UUID, f domain.TaskFilter, at domain.At) (domain.TaskList, error) {
	tasks, err := s.repo.ListTasks(ctx, projectID, f, at)
	if err != nil {
		return domain.TaskList{}, err
	}
	sum, err := s.repo.SummarizeTasks(ctx, projectID, at)
	if err != nil {
		return domain.TaskList{}, err
	}

	return domain.TaskList{Tasks: tasks, Summary: sum}, nil
}

// checkScope rejects a scope that is not a session of the project. The foreign
// key is deferred, so without this the error would only surface at COMMIT.
func (s *Service) checkScope(ctx context.Context, projectID uuid.UUID, scope string) error {
	set, _, err := s.scopeSet(ctx, projectID)
	if err != nil {
		return err
	}
	if !set[scope] {
		return errUnknownScope()
	}

	return nil
}

func normalizeScope(raw string) (string, error) {
	scope := strings.TrimSpace(raw)
	if scope == "" {
		return pd.AllScope, nil
	}
	if !scopePattern.MatchString(scope) {
		return "", errUnknownScope()
	}

	return scope, nil
}

func (s *Service) checkVendor(ctx context.Context, projectID uuid.UUID, vendorID *uuid.UUID) error {
	if vendorID == nil {
		return nil
	}
	ok, err := s.repo.VendorInProject(ctx, projectID, *vendorID)
	if err != nil {
		return err
	}
	if !ok {
		return invalidInput("vendor_id", "must be a vendor of this project")
	}

	return nil
}

func checkPhaseOrder(n int) error {
	if n < minPhaseOrder || n > maxPhaseOrder {
		return invalidInput("phase_order", fmt.Sprintf("must be between %d and %d", minPhaseOrder, maxPhaseOrder))
	}

	return nil
}

// CreateTask adds a custom task at the end of the list. It runs under the
// project lock so a double click cannot pass the limit of 500.
func (s *Service) CreateTask(ctx context.Context, p pd.Project, req domain.CreateTaskRequest) (domain.Task, error) {
	title, err := cleanText("title", req.Title, maxTitleRunes, true)
	if err != nil {
		return domain.Task{}, err
	}
	category, err := cleanText("category", req.Category, maxCategoryRunes, false)
	if err != nil {
		return domain.Task{}, err
	}
	if category == "" {
		category = defaultCategory
	}
	phaseName, err := cleanText("phase_name", req.PhaseName, maxPhaseRunes, false)
	if err != nil {
		return domain.Task{}, err
	}
	pic, err := cleanText("pic", req.PIC, maxPICRunes, false)
	if err != nil {
		return domain.Task{}, err
	}
	notes, err := cleanNote("notes", req.Notes, maxNoteRunes)
	if err != nil {
		return domain.Task{}, err
	}
	if req.PhaseOrder != 0 {
		if err := checkPhaseOrder(req.PhaseOrder); err != nil {
			return domain.Task{}, err
		}
	}
	start, err := parseDate("start_date", req.StartDate)
	if err != nil {
		return domain.Task{}, err
	}
	end, err := parseDate("end_date", req.EndDate)
	if err != nil {
		return domain.Task{}, err
	}
	if end.Before(start) {
		return domain.Task{}, invalidDate("end_date", "must not be before start_date")
	}
	if err := checkWindow(p, "start_date", start); err != nil {
		return domain.Task{}, err
	}
	if err := checkWindow(p, "end_date", end); err != nil {
		return domain.Task{}, err
	}
	scope, err := normalizeScope(req.SubEventScope)
	if err != nil {
		return domain.Task{}, err
	}
	at, _, err := s.clock(p)
	if err != nil {
		return domain.Task{}, err
	}

	var id uuid.UUID
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		if err := s.checkScope(ctx, p.ID, scope); err != nil {
			return err
		}
		if err := s.checkVendor(ctx, p.ID, req.VendorID); err != nil {
			return err
		}
		n, err := s.repo.CountTasks(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxTasks {
			return apierr.Conflict("task_limit_reached", fmt.Sprintf("at most %d tasks per project", maxTasks))
		}

		order, name := req.PhaseOrder, phaseName
		if order == 0 {
			// Follow the task at the end of the list; an empty project starts at phase 1.
			lastOrder, lastName, found, err := s.repo.LastTaskPhase(ctx, p.ID)
			if err != nil {
				return err
			}
			order = 1
			if found {
				order = lastOrder
				if name == "" {
					name = lastName
				}
			}
		} else if name == "" {
			if known, found, err := s.repo.PhaseName(ctx, p.ID, order); err != nil {
				return err
			} else if found {
				name = known
			}
		}

		sortOrder, err := s.repo.NextTaskSortOrder(ctx, p.ID)
		if err != nil {
			return err
		}
		t := domain.Task{
			ID: uuid.New(), ProjectID: p.ID, Title: title, SubEventScope: scope, Category: category,
			PhaseOrder: order, PhaseName: name, StartDate: start, EndDate: end, Status: domain.TaskBelum, PIC: pic,
			VendorID: req.VendorID, SortOrder: sortOrder, Notes: notes, Origin: domain.OriginCustom,
		}
		id = t.ID

		return s.repo.InsertTask(ctx, &t)
	})
	if err != nil {
		return domain.Task{}, mapErr(err, "task")
	}

	t, err := s.repo.TaskByID(ctx, p.ID, id, at)

	return t, mapErr(err, "task")
}

// UpdateTask applies a PATCH. It reads, merges and writes the task under the
// project lock so two tabs changing different fields do not overwrite each other.
// Only the fields that are sent are validated: changing the status of a task whose
// dates fell outside the window (after Hari-H moved) still works.
func (s *Service) UpdateTask(ctx context.Context, p pd.Project, id uuid.UUID, req domain.UpdateTaskRequest) (domain.Task, error) {
	if req.ClearVendor && req.VendorID != nil {
		return domain.Task{}, invalidInput("clear_vendor", "cannot be combined with vendor_id")
	}
	if req.Status != nil && !req.Status.Valid() {
		return domain.Task{}, invalidInput("status", "must be one of belum, proses, selesai")
	}
	at, _, err := s.clock(p)
	if err != nil {
		return domain.Task{}, err
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		t, err := s.repo.TaskByID(ctx, p.ID, id, at)
		if err != nil {
			return err
		}

		if req.Title != nil {
			if t.Title, err = cleanText("title", *req.Title, maxTitleRunes, true); err != nil {
				return err
			}
		}
		if req.Category != nil {
			if t.Category, err = cleanText("category", *req.Category, maxCategoryRunes, false); err != nil {
				return err
			}
			if t.Category == "" {
				t.Category = defaultCategory
			}
		}
		if req.PhaseOrder != nil {
			if err := checkPhaseOrder(*req.PhaseOrder); err != nil {
				return err
			}
			t.PhaseOrder = *req.PhaseOrder
		}
		if req.PhaseName != nil {
			if t.PhaseName, err = cleanText("phase_name", *req.PhaseName, maxPhaseRunes, false); err != nil {
				return err
			}
		}
		if req.PIC != nil {
			if t.PIC, err = cleanText("pic", *req.PIC, maxPICRunes, false); err != nil {
				return err
			}
		}
		if req.Notes != nil {
			if t.Notes, err = cleanNote("notes", *req.Notes, maxNoteRunes); err != nil {
				return err
			}
		}
		if req.Status != nil {
			t.Status = *req.Status
		}

		if req.StartDate != nil {
			if t.StartDate, err = parseDate("start_date", *req.StartDate); err != nil {
				return err
			}
			if err := checkWindow(p, "start_date", t.StartDate); err != nil {
				return err
			}
		}
		if req.EndDate != nil {
			if t.EndDate, err = parseDate("end_date", *req.EndDate); err != nil {
				return err
			}
			if err := checkWindow(p, "end_date", t.EndDate); err != nil {
				return err
			}
		}
		switch {
		case req.EndDate != nil && t.EndDate.Before(t.StartDate):
			return invalidDate("end_date", "must not be before start_date")
		case req.StartDate != nil && t.StartDate.After(t.EndDate):
			return invalidDate("start_date", "must not be after end_date")
		}

		if req.SubEventScope != nil {
			scope, err := normalizeScope(*req.SubEventScope)
			if err != nil {
				return err
			}
			if err := s.checkScope(ctx, p.ID, scope); err != nil {
				return err
			}
			t.SubEventScope = scope
		}
		switch {
		case req.ClearVendor:
			t.VendorID = nil
		case req.VendorID != nil:
			if err := s.checkVendor(ctx, p.ID, req.VendorID); err != nil {
				return err
			}
			t.VendorID = req.VendorID
		}

		return s.repo.UpdateTask(ctx, &t)
	})
	if err != nil {
		return domain.Task{}, mapErr(err, "task")
	}

	t, err := s.repo.TaskByID(ctx, p.ID, id, at)

	return t, mapErr(err, "task")
}

// SetTaskStatus is the quick toggle (checkbox, Kanban). There are no
// transition rules: a finished task may go back to belum.
func (s *Service) SetTaskStatus(ctx context.Context, p pd.Project, id uuid.UUID, status domain.TaskStatus) (domain.Task, error) {
	if !status.Valid() {
		return domain.Task{}, invalidInput("status", "must be one of belum, proses, selesai")
	}
	at, _, err := s.clock(p)
	if err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.SetTaskStatus(ctx, p.ID, id, status); err != nil {
		return domain.Task{}, mapErr(err, "task")
	}

	t, err := s.repo.TaskByID(ctx, p.ID, id, at)

	return t, mapErr(err, "task")
}

func (s *Service) DeleteTask(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteTask(ctx, p.ID, id), "task")
}

// ReorderTasks sets sort_order = 1..n. The list must name every task of the
// project exactly once; anything else (missing, repeated, or the id of another
// project's task) rolls back and answers 400 invalid_input: ids.
func (s *Service) ReorderTasks(ctx context.Context, p pd.Project, req domain.ReorderRequest) (domain.TaskList, error) {
	at, _, err := s.clock(p)
	if err != nil {
		return domain.TaskList{}, err
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountTasks(ctx, p.ID)
		if err != nil {
			return err
		}
		if err := checkIDs(req.IDs, n); err != nil {
			return err
		}
		changed, err := s.repo.ReorderTasks(ctx, p.ID, req.IDs)
		if err != nil {
			return err
		}
		if changed != n {
			return invalidInput("ids", "must list every id exactly once")
		}

		return nil
	})
	if err != nil {
		return domain.TaskList{}, mapErr(err, "task")
	}

	return s.taskList(ctx, p.ID, domain.TaskFilter{}, at)
}

// ShiftTasks moves the template tasks that are not done by days (negative =
// earlier). It is a manual action and not idempotent: every call moves again.
// Moving Hari-H goes through PATCH /projects/:id with shift_tasks instead.
func (s *Service) ShiftTasks(ctx context.Context, p pd.Project, req domain.ShiftTasksRequest) (domain.ShiftTasksResponse, error) {
	switch {
	case req.Days == 0:
		return domain.ShiftTasksResponse{}, invalidInput("days", "must not be zero")
	case req.Days > maxShiftDays || req.Days < -maxShiftDays:
		return domain.ShiftTasksResponse{}, invalidInput("days", fmt.Sprintf("must be at most %d in absolute value", maxShiftDays))
	}
	today, err := s.today(p)
	if err != nil {
		return domain.ShiftTasksResponse{}, err
	}

	res := domain.ShiftTasksResponse{Warnings: []pd.Notice{}}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, lo, hi, err := s.repo.ShiftBounds(ctx, p.ID)
		if err != nil || n == 0 {
			return err
		}
		winLo, winHi := scheduleWindow(p)
		if lo.AddDate(0, 0, req.Days).Before(winLo) || hi.AddDate(0, 0, req.Days).After(winHi) {
			return invalidDate("days", "must keep every shifted task within the project schedule window")
		}

		if res.Shifted, err = s.repo.ShiftTasks(ctx, p.ID, req.Days); err != nil {
			return err
		}
		overdue, err := s.repo.CountOverdueShifted(ctx, p.ID, today)
		if err != nil {
			return err
		}
		if overdue > 0 {
			res.Warnings = append(res.Warnings, notice("tasks_overdue_after_shift", "some shifted tasks are now past their due date", overdue))
		}

		return nil
	})
	if err != nil {
		return domain.ShiftTasksResponse{}, mapErr(err, "task")
	}

	return res, nil
}

// Export returns everything the spreadsheet is built from: all tasks of the
// project (not a filtered list) and the session names.
func (s *Service) Export(ctx context.Context, p pd.Project) (domain.ExportData, error) {
	at, loc, err := s.clock(p)
	if err != nil {
		return domain.ExportData{}, err
	}
	tasks, err := s.repo.ListTasks(ctx, p.ID, domain.TaskFilter{}, at)
	if err != nil {
		return domain.ExportData{}, err
	}
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.ExportData{}, err
	}

	return domain.ExportData{Project: p, Tasks: tasks, Sessions: sessions, Today: pd.CalendarDay(at.Now, loc)}, nil
}
