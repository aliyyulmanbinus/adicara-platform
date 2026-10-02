package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

type stubGuestRepository struct{ rsvp domain.RSVPWrite }

func (s *stubGuestRepository) ListByOwner(context.Context, string, string) ([]domain.Guest, error) {
	return nil, nil
}
func (s *stubGuestRepository) CreateForOwner(context.Context, string, string, string, domain.GuestWrite) (domain.Guest, error) {
	return domain.Guest{}, nil
}
func (s *stubGuestRepository) UpdateForOwner(context.Context, string, string, string, domain.GuestWrite) (domain.Guest, error) {
	return domain.Guest{}, nil
}
func (s *stubGuestRepository) DeleteForOwner(context.Context, string, string, string) error {
	return nil
}
func (s *stubGuestRepository) SubmitRSVP(_ context.Context, _ string, input domain.RSVPWrite) (domain.RSVP, error) {
	s.rsvp = input
	return domain.RSVP{Status: input.Status, AttendeeCount: input.AttendeeCount}, nil
}

func TestRSVPForAbsentGuestForcesZeroAttendees(t *testing.T) {
	repository := &stubGuestRepository{}
	service := NewGuestService(repository)
	_, err := service.RSVP(context.Background(), "nara-dan-arka", domain.RSVPWrite{GuestToken: "1234567890123456789012345678901234567890123", Status: "tidak_hadir", AttendeeCount: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository.rsvp.AttendeeCount != 0 {
		t.Fatalf("expected zero attendees, got %d", repository.rsvp.AttendeeCount)
	}
}

func TestRSVPRejectsInvalidToken(t *testing.T) {
	service := NewGuestService(&stubGuestRepository{})
	_, err := service.RSVP(context.Background(), "nara-dan-arka", domain.RSVPWrite{GuestToken: "short", Status: "hadir", AttendeeCount: 1})
	if !errors.Is(err, ErrInvalidGuest) {
		t.Fatalf("expected ErrInvalidGuest, got %v", err)
	}
}
