package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

type stubInvitationRepository struct {
	invitation domain.Invitation
	err        error
	called     bool
}

func (s *stubInvitationRepository) FindPublishedBySlug(context.Context, string) (domain.Invitation, error) {
	s.called = true
	return s.invitation, s.err
}

func TestPublishedBySlugRejectsInvalidSlug(t *testing.T) {
	repository := &stubInvitationRepository{}
	service := NewInvitationService(repository)

	_, err := service.PublishedBySlug(context.Background(), "Invalid Slug")

	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}
	if repository.called {
		t.Fatal("repository must not be called for an invalid slug")
	}
}

func TestPublishedBySlugReturnsRepositoryResult(t *testing.T) {
	want := domain.Invitation{Slug: "nara-dan-arka", Title: "Pernikahan Nara & Arka"}
	repository := &stubInvitationRepository{invitation: want}
	service := NewInvitationService(repository)

	got, err := service.PublishedBySlug(context.Background(), want.Slug)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Slug != want.Slug || got.Title != want.Title {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
