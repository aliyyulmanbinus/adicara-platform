package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/service"
)

type PublishedInvitationFinder interface {
	PublishedBySlug(ctx context.Context, slug string) (domain.Invitation, error)
}

type InvitationHandler struct {
	service PublishedInvitationFinder
}

func NewInvitationHandler(service PublishedInvitationFinder) *InvitationHandler {
	return &InvitationHandler{service: service}
}

func (h *InvitationHandler) GetPublished(w http.ResponseWriter, r *http.Request) {
	invitation, err := h.service.PublishedBySlug(r.Context(), r.PathValue("slug"))
	switch {
	case errors.Is(err, service.ErrInvalidSlug):
		writeError(w, http.StatusBadRequest, "invalid_slug", "slug format is invalid")
		return
	case errors.Is(err, domain.ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "not_found", "published invitation was not found")
		return
	case err != nil:
		slog.ErrorContext(r.Context(), "failed to get published invitation", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}

	writeJSON(w, http.StatusOK, invitation)
}
