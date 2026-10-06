package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/service"
)

type InvitationManager interface {
	List(context.Context, string) ([]domain.Invitation, error)
	Create(context.Context, string, domain.InvitationWrite) (domain.Invitation, error)
	Get(context.Context, string, string) (domain.Invitation, error)
	Update(context.Context, string, string, domain.InvitationUpdate) (domain.Invitation, error)
	Delete(context.Context, string, string) error
	Publish(context.Context, string, string) (domain.Invitation, error)
	Unpublish(context.Context, string, string) (domain.Invitation, error)
}

type InvitationManagementHandler struct {
	service InvitationManager
	auth    *AuthHandler
}

type invitationWriteRequest struct {
	EventType     string                   `json:"event_type"`
	Slug          string                   `json:"slug"`
	Title         string                   `json:"title"`
	TemplateKey   string                   `json:"template_key"`
	AllowIndexing bool                     `json:"allow_indexing"`
	Hosts         []domain.InvitationHost  `json:"hosts"`
	Events        []domain.InvitationEvent `json:"events"`
	DesignData    map[string]string        `json:"design_data"`
}

type invitationUpdateRequest struct {
	EventType     *string                   `json:"event_type"`
	Slug          *string                   `json:"slug"`
	Title         *string                   `json:"title"`
	TemplateKey   *string                   `json:"template_key"`
	AllowIndexing *bool                     `json:"allow_indexing"`
	Hosts         *[]domain.InvitationHost  `json:"hosts"`
	Events        *[]domain.InvitationEvent `json:"events"`
	DesignData    *map[string]string        `json:"design_data"`
}

func NewInvitationManagementHandler(service InvitationManager, auth *AuthHandler) *InvitationManagementHandler {
	return &InvitationManagementHandler{service: service, auth: auth}
}

func (h *InvitationManagementHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, false)
	if !ok {
		return
	}
	invitations, err := h.service.List(r.Context(), identity.User.ID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": invitations})
}

func (h *InvitationManagementHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	var request invitationWriteRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	invitation, err := h.service.Create(r.Context(), identity.User.ID, request.toDomain())
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, invitation)
}

func (h *InvitationManagementHandler) Get(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, false)
	if !ok {
		return
	}
	invitation, err := h.service.Get(r.Context(), identity.User.ID, r.PathValue("id"))
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, invitation)
}

func (h *InvitationManagementHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	var request invitationUpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	invitation, err := h.service.Update(r.Context(), identity.User.ID, r.PathValue("id"), request.toDomain())
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, invitation)
}

func (h *InvitationManagementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	err := h.service.Delete(r.Context(), identity.User.ID, r.PathValue("id"))
	if h.handleError(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InvitationManagementHandler) Publish(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, h.service.Publish)
}

func (h *InvitationManagementHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, h.service.Unpublish)
}

func (h *InvitationManagementHandler) setStatus(
	w http.ResponseWriter,
	r *http.Request,
	operation func(context.Context, string, string) (domain.Invitation, error),
) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	invitation, err := operation(r.Context(), identity.User.ID, r.PathValue("id"))
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, invitation)
}

func (h *InvitationManagementHandler) handleError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, service.ErrInvalidInvitation), errors.Is(err, service.ErrInvalidID):
		writeError(w, http.StatusUnprocessableEntity, "invalid_invitation", "invitation data is invalid")
	case errors.Is(err, domain.ErrInvitationSlugConflict):
		writeError(w, http.StatusConflict, "slug_conflict", "invitation slug is already in use")
	case errors.Is(err, domain.ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "not_found", "invitation was not found")
	default:
		h.internalError(w, r, err)
	}
	return true
}

func (h *InvitationManagementHandler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "invitation operation failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func (r invitationWriteRequest) toDomain() domain.InvitationWrite {
	return domain.InvitationWrite{
		EventType: r.EventType, Slug: r.Slug, Title: r.Title, TemplateKey: r.TemplateKey,
		AllowIndexing: r.AllowIndexing, Hosts: r.Hosts, Events: r.Events, DesignData: r.DesignData,
	}
}

func (r invitationUpdateRequest) toDomain() domain.InvitationUpdate {
	return domain.InvitationUpdate{
		EventType: r.EventType, Slug: r.Slug, Title: r.Title, TemplateKey: r.TemplateKey,
		AllowIndexing: r.AllowIndexing, Hosts: r.Hosts, Events: r.Events, DesignData: r.DesignData,
	}
}
