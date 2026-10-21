package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

const (
	sampleTitle        = "Contoh: Pernikahan Sinta & Arif"
	sampleDaysAhead    = 175 // "Towards the Day: 175 Days" on page 1 of Wedding_Planner.pdf
	sampleTargetBudget = 75_000_000
)

func sampleMetadata() map[string]any {
	return map[string]any{
		"hero_title":    "Bismillah Nikah",
		"bride_name":    "Sinta Maharani",
		"groom_name":    "Arif Pratama",
		"bride_parents": "Bpk. Bambang & Ibu Sri",
		"groom_parents": "Bpk. Hendra & Ibu Rina",
		"venue_name":    "Sasana Kriya Ballroom",
		"city":          "Jakarta Selatan",
	}
}

// CreateSample creates the sample wedding "Contoh: Pernikahan Sinta & Arif", at
// most one per user and idempotent: an existing one is returned with
// Created == false, also when two requests race (the partial unique index
// decides).
//
// It applies every wedding template that has a registered applier and then the
// registered dataset overlays. Modules that are not built yet contribute
// nothing, so until all of them exist the sample is only as full as the
// modules that do.
func (s *Service) CreateSample(ctx context.Context, userID uuid.UUID) (domain.CreateProjectResult, error) {
	if existing, err := s.repo.SampleProject(ctx, userID); err == nil {
		return domain.CreateProjectResult{Project: existing}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.CreateProjectResult{}, err
	}

	spec, _ := s.registry.Get(domain.EventWedding)
	tz, loc, err := resolveTimezone(nil)
	if err != nil {
		return domain.CreateProjectResult{}, err
	}
	today := domain.CalendarDay(s.now(), loc)

	metadata, err := mergeMetadata(spec, nil, sampleMetadata())
	if err != nil {
		return domain.CreateProjectResult{}, err
	}

	p := domain.Project{
		ID: uuid.New(), UserID: userID, EventType: domain.EventWedding, Title: sampleTitle,
		EventDate: today.AddDate(0, 0, sampleDaysAhead), PrepStartDate: today, Timezone: tz,
		TargetBudget: sampleTargetBudget, Status: domain.StatusActive, IsSample: true,
		SkippedVendorCategories: []string{}, Metadata: metadata,
	}

	applied, err := s.createInTx(ctx, &p, spec, nil, s.sampleTemplates(spec), true)
	if errors.Is(err, errTokenReused) {
		// Lost the race against another request of the same user.
		existing, gerr := s.repo.SampleProject(ctx, userID)
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

// sampleTemplates asks for every wedding template that can be applied now:
// the full PDF timeline plus the Adicara tasks, documents, budget lines,
// shopping list, rundown and playlist.
func (s *Service) sampleTemplates(spec domain.EventTypeSpec) domain.InitTemplates {
	av := spec.Templates.Available(s.hasApplier)
	t := domain.InitTemplates{
		IncludeAdicaraTasks: av.IncludeAdicaraTasks, AdminDocs: av.AdminDocs, BudgetLines: av.BudgetLines,
		ShoppingList: av.ShoppingList, Rundown: av.Rundown, Playlist: av.Playlist,
	}
	if contains(av.TimelinePresets, "wedding_pdf_44") {
		preset := "wedding_pdf_44"
		t.TimelinePreset = &preset
	}

	return t
}
