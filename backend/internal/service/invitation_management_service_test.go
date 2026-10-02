package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

type stubInvitationManagementRepository struct {
	ownerID string
	input   domain.InvitationWrite
}

func (s *stubInvitationManagementRepository) ListByOwner(context.Context, string) ([]domain.Invitation, error) {
	return []domain.Invitation{}, nil
}

func (s *stubInvitationManagementRepository) CreateForOwner(_ context.Context, ownerID string, input domain.InvitationWrite) (domain.Invitation, error) {
	s.ownerID = ownerID
	s.input = input
	return domain.Invitation{ID: "00000000-0000-4000-8000-000000000002", Slug: input.Slug}, nil
}

func (s *stubInvitationManagementRepository) FindByOwner(context.Context, string, string) (domain.Invitation, error) {
	return domain.Invitation{}, domain.ErrInvitationNotFound
}

func (s *stubInvitationManagementRepository) UpdateForOwner(context.Context, string, string, domain.InvitationUpdate) (domain.Invitation, error) {
	return domain.Invitation{}, nil
}

func (s *stubInvitationManagementRepository) DeleteForOwner(context.Context, string, string) error {
	return nil
}

func (s *stubInvitationManagementRepository) SetStatusForOwner(context.Context, string, string, string) (domain.Invitation, error) {
	return domain.Invitation{}, nil
}

func validInvitationWrite() domain.InvitationWrite {
	start := time.Date(2027, 12, 12, 8, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	return domain.InvitationWrite{
		EventType: "wedding", Slug: "nara-dan-arka", Title: "Pernikahan Nara & Arka",
		TemplateKey: "editorial-ivory",
		Hosts:       []domain.InvitationHost{{Name: "Nara", Role: "Mempelai wanita"}},
		Events: []domain.InvitationEvent{{
			Name: "Akad", StartAt: start, Timezone: "Asia/Jakarta",
			VenueName: "Pendopo", VenueAddress: "Jakarta",
		}},
	}
}

func TestCreateInvitationScopesWriteToOwnerAndNormalizesSlug(t *testing.T) {
	repository := &stubInvitationManagementRepository{}
	manager := NewInvitationManagementService(repository)
	input := validInvitationWrite()
	input.Slug = "  Nara-Dan-Arka  "

	_, err := manager.Create(context.Background(), "owner-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository.ownerID != "owner-1" || repository.input.Slug != "nara-dan-arka" {
		t.Fatalf("unexpected scoped create: owner=%q slug=%q", repository.ownerID, repository.input.Slug)
	}
}

func TestCreateInvitationRejectsUnsupportedTemplate(t *testing.T) {
	manager := NewInvitationManagementService(&stubInvitationManagementRepository{})
	input := validInvitationWrite()
	input.TemplateKey = "unknown-theme"
	_, err := manager.Create(context.Background(), "owner-1", input)
	if !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("expected ErrInvalidInvitation, got %v", err)
	}
}

func TestGetInvitationRejectsInvalidIDBeforeRepository(t *testing.T) {
	manager := NewInvitationManagementService(&stubInvitationManagementRepository{})
	_, err := manager.Get(context.Background(), "owner-1", "sequential-12")
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}
