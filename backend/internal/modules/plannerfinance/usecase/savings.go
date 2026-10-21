package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

// ---------- contributors ----------

// role validates role_label against the contributor roles of the event type; blank is "other".
func (s *Service) role(eventType pd.EventType, raw *string) (string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return defaultRole, nil
	}
	role := strings.TrimSpace(*raw)
	spec, _ := s.registry.Get(eventType)
	for _, r := range spec.ContributorRoles {
		if r.Key == role {
			return role, nil
		}
	}

	return "", invalidInput("role_label", "must be one of the event type roles")
}

func (s *Service) ListContributors(ctx context.Context, p pd.Project) ([]domain.Contributor, error) {
	return s.repo.ListContributors(ctx, p.ID)
}

// CreateContributor adds a contributor (at most 20), under the project lock.
func (s *Service) CreateContributor(ctx context.Context, p pd.Project, req domain.ContributorRequest) (domain.Contributor, error) {
	name, err := cleanText("name", req.Name, maxNameRunes, true)
	if err != nil {
		return domain.Contributor{}, err
	}
	role, err := s.role(p.EventType, req.RoleLabel)
	if err != nil {
		return domain.Contributor{}, err
	}

	c := domain.Contributor{ID: uuid.New(), Name: name, RoleLabel: role}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountContributors(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxContributors {
			return apierr.Conflict("contributor_limit_reached", fmt.Sprintf("at most %d contributors per project", maxContributors))
		}

		return s.repo.InsertContributor(ctx, p.ID, &c)
	})
	if err != nil {
		return domain.Contributor{}, mapErr(err, "contributor")
	}

	out, err := s.repo.ContributorByID(ctx, p.ID, c.ID)

	return out, mapErr(err, "contributor")
}

// UpdateContributor replaces name and role (a blank role becomes "other").
func (s *Service) UpdateContributor(ctx context.Context, p pd.Project, id uuid.UUID, req domain.ContributorRequest) (domain.Contributor, error) {
	name, err := cleanText("name", req.Name, maxNameRunes, true)
	if err != nil {
		return domain.Contributor{}, err
	}
	role, err := s.role(p.EventType, req.RoleLabel)
	if err != nil {
		return domain.Contributor{}, err
	}
	if err := s.repo.UpdateContributor(ctx, p.ID, &domain.Contributor{ID: id, Name: name, RoleLabel: role}); err != nil {
		return domain.Contributor{}, mapErr(err, "contributor")
	}

	out, err := s.repo.ContributorByID(ctx, p.ID, id)

	return out, mapErr(err, "contributor")
}

// DeleteContributor refuses a contributor that has deposits. The check is made
// first because the foreign key is deferred: the DELETE itself succeeds and the
// transaction would only fail at COMMIT. A deposit that slips in between the
// check and the commit ends as 409 conflict.
func (s *Service) DeleteContributor(ctx context.Context, p pd.Project, id uuid.UUID) error {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		c, err := s.repo.ContributorByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if c.EntryCount > 0 {
			return apierr.Conflict("contributor_in_use", "the contributor has deposits; delete or move them first")
		}

		return s.repo.DeleteContributor(ctx, p.ID, id)
	})

	return mapErr(err, "contributor")
}

// ---------- deposits ----------

func (s *Service) validateEntry(ctx context.Context, p pd.Project, req domain.SavingsEntryRequest) (domain.SavingsEntry, error) {
	date, err := parseDate("entry_date", req.EntryDate)
	if err != nil {
		return domain.SavingsEntry{}, err
	}
	if err := checkWindow(p, "entry_date", date); err != nil {
		return domain.SavingsEntry{}, err
	}
	if req.Amount <= 0 {
		return domain.SavingsEntry{}, invalidInput("amount", "must be greater than 0")
	}
	if err := validateMoney("amount", req.Amount); err != nil {
		return domain.SavingsEntry{}, err
	}
	notes, err := optionalNote("notes", req.Notes)
	if err != nil {
		return domain.SavingsEntry{}, err
	}
	if _, err := s.repo.ContributorByID(ctx, p.ID, req.ContributorID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.SavingsEntry{}, invalidInput("contributor_id", "must be a contributor of this project")
		}

		return domain.SavingsEntry{}, err
	}

	return domain.SavingsEntry{ContributorID: req.ContributorID, EntryDate: date.Format(pd.DateLayout), Amount: req.Amount, Notes: notes}, nil
}

// CreateEntry records a deposit (at most 2000 per project), under the project lock.
func (s *Service) CreateEntry(ctx context.Context, p pd.Project, req domain.SavingsEntryRequest) (domain.SavingsEntry, error) {
	var out domain.SavingsEntry
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		e, err := s.validateEntry(ctx, p, req)
		if err != nil {
			return err
		}
		e.ID = uuid.New()

		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountEntries(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxEntries {
			return apierr.Conflict("savings_entry_limit_reached", fmt.Sprintf("at most %d deposits per project", maxEntries))
		}
		if err := s.repo.InsertEntry(ctx, p.ID, &e); err != nil {
			return err
		}
		out, err = s.repo.EntryByID(ctx, p.ID, e.ID)

		return err
	})
	if err != nil {
		return domain.SavingsEntry{}, mapErr(err, "deposit")
	}

	return out, nil
}

// UpdateEntry corrects a deposit, also moving it to another contributor.
func (s *Service) UpdateEntry(ctx context.Context, p pd.Project, id uuid.UUID, req domain.SavingsEntryRequest) (domain.SavingsEntry, error) {
	var out domain.SavingsEntry
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		// An id of another project is a 404 before the body is looked at.
		if _, err := s.repo.EntryByID(ctx, p.ID, id); err != nil {
			return err
		}
		e, err := s.validateEntry(ctx, p, req)
		if err != nil {
			return err
		}
		e.ID = id
		if err := s.repo.UpdateEntry(ctx, p.ID, &e); err != nil {
			return err
		}
		out, err = s.repo.EntryByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.SavingsEntry{}, mapErr(err, "deposit")
	}

	return out, nil
}

func (s *Service) DeleteEntry(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteEntry(ctx, p.ID, id), "deposit")
}

// ---------- the savings screen ----------

// Savings is GET /savings: the KPI, contributors, milestones, the monthly
// recommendation, the monthly matrix and the deposits. contributor filters only
// the deposits; every other part covers the whole project.
func (s *Service) Savings(ctx context.Context, p pd.Project, contributor *uuid.UUID) (domain.SavingsOverview, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	return s.savings(ctx, p, today, contributor)
}

func (s *Service) savings(ctx context.Context, p pd.Project, today time.Time, contributor *uuid.UUID) (domain.SavingsOverview, error) {
	collected, err := s.repo.SavingsCollected(ctx, p.ID)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	contributors, err := s.repo.ListContributors(ctx, p.ID)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	milestones, err := s.repo.Milestones(ctx, p.ID)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	unscheduled, err := s.repo.UnscheduledEstimate(ctx, p.ID)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	cells, err := s.repo.MonthCells(ctx, p.ID)
	if err != nil {
		return domain.SavingsOverview{}, err
	}
	entries, err := s.repo.ListEntries(ctx, p.ID, contributor)
	if err != nil {
		return domain.SavingsOverview{}, err
	}

	target := p.TargetBudget
	if p.SavingsTarget != nil {
		target = *p.SavingsTarget
	}
	kpi := domain.SavingsKPI{Target: target, Collected: collected, Shortfall: max(target-collected, 0), TargetFollowsBudget: p.SavingsTarget == nil}

	return domain.SavingsOverview{
		KPI: kpi, Contributors: contributors, Milestones: milestones, UnscheduledEstimate: unscheduled,
		Recommendation: recommend(kpi.Shortfall, len(contributors), milestones, p.EventDate, today),
		Matrix:         savingsMatrix(cells), Entries: entries,
	}, nil
}

// recommend is the monthly recommendation of 03 BE 3.6. The target date is the
// last dated session (basis "session"), otherwise Hari-H. months_left is the
// number of 30-day months to it, rounded up and at least 1, so a date that is
// near or past still asks for the whole shortfall at once. Amounts round up to
// Rp 1.000. Today is a parameter so the figures can be fixed in tests.
func recommend(shortfall int64, contributors int, milestones []domain.Milestone, eventDate, today time.Time) domain.Recommendation {
	targetDate, basis := eventDate, "event_date"
	if n := len(milestones); n > 0 {
		last, err := time.Parse(pd.DateLayout, milestones[n-1].SessionDate)
		if err == nil {
			targetDate, basis = last, "session"
		}
	}

	days := int64(targetDate.Sub(today).Hours() / 24)
	months := max(domain.CeilDiv(days, 30), 1)
	monthly := domain.CeilDiv(shortfall, months*1000) * 1000

	rec := domain.Recommendation{MonthsLeft: int(months), RecommendedMonthly: monthly, TargetDate: targetDate.Format(pd.DateLayout), Basis: basis}
	if contributors > 0 {
		per := domain.CeilDiv(monthly, int64(contributors)*1000) * 1000
		rec.PerContributor = &per
	}

	return rec
}

// savingsMatrix folds the (month, contributor) cells into one row per month.
// The repository returns every contributor for every month, so a row always
// lists them all.
func savingsMatrix(cells []domain.MonthCell) domain.SavingsMatrix {
	m := domain.SavingsMatrix{Months: []string{}, Rows: []domain.SavingsMonth{}}
	for _, c := range cells {
		if n := len(m.Rows); n == 0 || m.Rows[n-1].Month != c.Month {
			m.Months = append(m.Months, c.Month)
			m.Rows = append(m.Rows, domain.SavingsMonth{Month: c.Month, ByContributor: map[string]int64{}})
		}
		row := &m.Rows[len(m.Rows)-1]
		row.ByContributor[c.ContributorID.String()] = c.Amount
		row.Total += c.Amount
	}

	return m
}
