package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

var ErrInvalidGuest = errors.New("invalid guest")

type GuestRepository interface {
	ListByOwner(context.Context, string, string) ([]domain.Guest, error)
	CreateForOwner(context.Context, string, string, string, domain.GuestWrite) (domain.Guest, error)
	UpdateForOwner(context.Context, string, string, string, domain.GuestWrite) (domain.Guest, error)
	DeleteForOwner(context.Context, string, string, string) error
	SubmitRSVP(context.Context, string, domain.RSVPWrite) (domain.RSVP, error)
}

type GuestService struct{ repository GuestRepository }

func NewGuestService(repository GuestRepository) *GuestService {
	return &GuestService{repository: repository}
}

func (s *GuestService) List(ctx context.Context, ownerID, invitationID string) ([]domain.Guest, error) {
	if !uuidPattern.MatchString(invitationID) {
		return nil, ErrInvalidID
	}
	return s.repository.ListByOwner(ctx, ownerID, invitationID)
}

func (s *GuestService) Create(ctx context.Context, ownerID, invitationID string, input domain.GuestWrite) (domain.Guest, error) {
	if !uuidPattern.MatchString(invitationID) {
		return domain.Guest{}, ErrInvalidID
	}
	input = normalizeGuest(input)
	if !validGuest(input) {
		return domain.Guest{}, ErrInvalidGuest
	}
	token, err := randomToken()
	if err != nil {
		return domain.Guest{}, err
	}
	return s.repository.CreateForOwner(ctx, ownerID, invitationID, token, input)
}

func (s *GuestService) Update(ctx context.Context, ownerID, invitationID, guestID string, input domain.GuestWrite) (domain.Guest, error) {
	if !uuidPattern.MatchString(invitationID) || !uuidPattern.MatchString(guestID) {
		return domain.Guest{}, ErrInvalidID
	}
	input = normalizeGuest(input)
	if !validGuest(input) {
		return domain.Guest{}, ErrInvalidGuest
	}
	return s.repository.UpdateForOwner(ctx, ownerID, invitationID, guestID, input)
}

func (s *GuestService) Delete(ctx context.Context, ownerID, invitationID, guestID string) error {
	if !uuidPattern.MatchString(invitationID) || !uuidPattern.MatchString(guestID) {
		return ErrInvalidID
	}
	return s.repository.DeleteForOwner(ctx, ownerID, invitationID, guestID)
}

func (s *GuestService) RSVP(ctx context.Context, slug string, input domain.RSVPWrite) (domain.RSVP, error) {
	input.GuestToken = strings.TrimSpace(input.GuestToken)
	input.Status = strings.TrimSpace(input.Status)
	input.Message = strings.TrimSpace(input.Message)
	if len(slug) < 3 || !slugPattern.MatchString(slug) || len(input.GuestToken) != 43 ||
		(input.Status != "hadir" && input.Status != "tidak_hadir") || input.AttendeeCount < 0 || input.AttendeeCount > 20 || len(input.Message) > 500 {
		return domain.RSVP{}, ErrInvalidGuest
	}
	if input.Status == "hadir" && input.AttendeeCount < 1 {
		return domain.RSVP{}, ErrInvalidGuest
	}
	if input.Status == "tidak_hadir" {
		input.AttendeeCount = 0
	}
	return s.repository.SubmitRSVP(ctx, slug, input)
}

func normalizeGuest(input domain.GuestWrite) domain.GuestWrite {
	input.Name = strings.TrimSpace(input.Name)
	input.Group = strings.TrimSpace(input.Group)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Notes = strings.TrimSpace(input.Notes)
	return input
}

func validGuest(input domain.GuestWrite) bool {
	return len(input.Name) >= 1 && len(input.Name) <= 120 && len(input.Group) <= 120 && len(input.Phone) <= 40 && len(input.Notes) <= 500
}
