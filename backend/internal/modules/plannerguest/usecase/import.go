package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// allSessionName is the name of the system session "all" (planner usecase, project creation).
const allSessionName = "Umum / Semua Sesi"

// importSessions are the sessions a file may name, including the system session all.
func importSessions(sessions []domain.SessionRef) []domain.SessionRef {
	out := make([]domain.SessionRef, 0, len(sessions)+1)
	hasAll := false
	for _, sess := range sessions {
		hasAll = hasAll || sess.ScopeKey == pd.AllScope
		out = append(out, sess)
	}
	if !hasAll {
		out = append(out, domain.SessionRef{ScopeKey: pd.AllScope, Name: allSessionName})
	}

	return out
}

// Import reads the records of a file (the handler decoded the upload) and either reports what would happen (dryRun,
// nothing is written) or writes the valid rows in one transaction under the project lock (05 BE 2.6). Rows with errors
// refuse the whole file unless opts.SkipInvalid; the total after the import may not pass the guest limit.
func (s *Service) Import(ctx context.Context, p pd.Project, recs []domain.Record, opts domain.ImportOptions, dryRun bool) (domain.ImportResult, error) {
	spec, _ := s.spec(p)

	if dryRun {
		sessions, err := s.repo.Sessions(ctx, p.ID)
		if err != nil {
			return domain.ImportResult{}, err
		}
		existing, err := s.repo.ExistingForImport(ctx, p.ID)
		if err != nil {
			return domain.ImportResult{}, err
		}
		plan, err := domain.BuildImportPlan(domain.ImportConfig{Spec: spec, Sessions: importSessions(sessions), Existing: existing, Options: opts}, recs)
		if err != nil {
			return domain.ImportResult{}, fileError(err)
		}
		plan.Result.DryRun = true

		return plan.Result, nil
	}

	var result domain.ImportResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return mapErr(err)
		}
		sessions, err := s.repo.Sessions(ctx, p.ID)
		if err != nil {
			return err
		}
		existing, err := s.repo.ExistingForImport(ctx, p.ID) // read again inside the lock
		if err != nil {
			return err
		}
		plan, err := domain.BuildImportPlan(domain.ImportConfig{Spec: spec, Sessions: importSessions(sessions), Existing: existing, Options: opts}, recs)
		if err != nil {
			return fileError(err)
		}
		if plan.Blocked {
			return invalidInput("rows", fmt.Sprintf("%d rows are invalid", plan.Result.InvalidRows))
		}
		if len(existing)+len(plan.Insert) > maxGuests {
			return errLimit()
		}

		guests := make([]domain.Guest, 0, len(plan.Insert))
		for _, pg := range plan.Insert {
			f := guestFields{name: pg.Name, side: pg.Side, category: pg.Category, scope: pg.Scope, table: pg.Table, pax: pg.Pax, phone: pg.Phone, wa: pg.WA, notes: pg.Notes}
			g, err := newGuest(f)
			if err != nil {
				return err
			}
			guests = append(guests, g)
		}
		if len(guests) > 0 {
			if err := s.repo.InsertGuests(ctx, p.ID, p.InvitationID, guests); err != nil {
				return mapErr(err)
			}
		}
		plan.Result.Created = len(guests)
		result = plan.Result

		return nil
	})
	if err != nil {
		return domain.ImportResult{}, mapErr(err)
	}

	return result, nil
}

// fileError turns a domain.FileError into 400 invalid_file ("file: <reason>"); any other error passes through.
func fileError(err error) error {
	var fe *domain.FileError
	if errors.As(err, &fe) {
		return apierr.BadRequest("invalid_file", "file: "+fe.Reason)
	}

	return err
}

// ImportTemplate gathers what the template of this project is made from: the words and lists of the registry, and the
// session names of the project (05 BE 2.6).
func (s *Service) ImportTemplate(ctx context.Context, p pd.Project) (domain.TemplateExport, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.TemplateExport{}, err
	}
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.TemplateExport{}, err
	}
	spec, et := s.spec(p)
	names := []string{allSessionName}
	for _, sess := range sessions {
		if sess.ScopeKey != pd.AllScope {
			names = append(names, sess.Name)
		}
	}

	return domain.TemplateExport{
		Project: p, Today: today,
		Input: domain.TemplateInput{GuestWord: et.Labels.Guest, Sides: spec.Sides, Cats: spec.Categories, Sessions: names},
	}, nil
}

// ExportGuests gathers the whole list in order (05 BE 2.6).
func (s *Service) ExportGuests(ctx context.Context, p pd.Project) (domain.GuestExport, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.GuestExport{}, err
	}
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.GuestExport{}, err
	}
	guests, err := s.repo.AllGuests(ctx, p.ID)
	if err != nil {
		return domain.GuestExport{}, err
	}
	spec, _ := s.spec(p)

	return domain.GuestExport{Project: p, Today: today, Spec: spec, Sessions: sessions, Guests: guests}, nil
}

// ExportAttendance gathers the final attendance sheet: the expected guests (not declined, or declined but came).
func (s *Service) ExportAttendance(ctx context.Context, p pd.Project) (domain.AttendanceExport, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.AttendanceExport{}, err
	}
	loc, err := pd.LoadLocation(p.Timezone)
	if err != nil {
		return domain.AttendanceExport{}, err
	}
	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return domain.AttendanceExport{}, err
	}
	all, err := s.repo.AllGuests(ctx, p.ID)
	if err != nil {
		return domain.AttendanceExport{}, err
	}
	counts, err := s.repo.ReceptionCounts(ctx, p.ID)
	if err != nil {
		return domain.AttendanceExport{}, err
	}
	spec, _ := s.spec(p)
	guests := make([]domain.Guest, 0, len(all))
	for _, g := range all {
		if g.RSVPStatus != domain.RSVPDeclined || g.IsCheckedIn {
			guests = append(guests, g)
		}
	}

	return domain.AttendanceExport{Project: p, Today: today, Location: loc, Spec: spec, Sessions: sessions, Guests: guests, KPI: counts.KPI}, nil
}
