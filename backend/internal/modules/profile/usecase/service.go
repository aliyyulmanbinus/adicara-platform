package usecase

import (
	"context"

	"github.com/google/uuid"

	authdomain "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
)

type Service struct {
	auth *authusecase.Service
}

func New(auth *authusecase.Service) *Service {
	return &Service{auth: auth}
}

type Me struct {
	User authdomain.User
}

func (s *Service) Get(ctx context.Context, uid uuid.UUID) (Me, error) {
	user, err := s.auth.Get(ctx, uid)
	if err != nil {
		return Me{}, err
	}
	return Me{User: user}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, uid uuid.UUID, name string) (authdomain.User, error) {
	return s.auth.UpdateProfile(ctx, uid, name)
}
