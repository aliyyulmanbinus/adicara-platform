package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

var ErrInvalidSlug = errors.New("invalid invitation slug")

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type InvitationRepository interface {
	FindPublishedBySlug(ctx context.Context, slug string) (domain.Invitation, error)
}

type InvitationService struct {
	repository InvitationRepository
}

func NewInvitationService(repository InvitationRepository) *InvitationService {
	return &InvitationService{repository: repository}
}

func (s *InvitationService) PublishedBySlug(ctx context.Context, slug string) (domain.Invitation, error) {
	if len(slug) < 3 || len(slug) > 100 || !slugPattern.MatchString(slug) {
		return domain.Invitation{}, ErrInvalidSlug
	}
	return s.repository.FindPublishedBySlug(ctx, slug)
}
