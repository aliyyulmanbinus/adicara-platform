package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/service"
)

type GuestManager interface {
	List(context.Context, string, string) ([]domain.Guest, error)
	Create(context.Context, string, string, domain.GuestWrite) (domain.Guest, error)
	Update(context.Context, string, string, string, domain.GuestWrite) (domain.Guest, error)
	Delete(context.Context, string, string, string) error
	RSVP(context.Context, string, domain.RSVPWrite) (domain.RSVP, error)
}

type GuestHandler struct {
	service GuestManager
	auth    *AuthHandler
}

type guestRequest struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	Phone string `json:"phone"`
	Notes string `json:"notes"`
}
type rsvpRequest struct {
	GuestToken    string `json:"guest_token"`
	Status        string `json:"status"`
	AttendeeCount int    `json:"attendee_count"`
	Message       string `json:"message"`
}

func NewGuestHandler(service GuestManager, auth *AuthHandler) *GuestHandler {
	return &GuestHandler{service: service, auth: auth}
}

func (h *GuestHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, false)
	if !ok {
		return
	}
	guests, err := h.service.List(r.Context(), identity.User.ID, r.PathValue("id"))
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"guests": guests})
}

func (h *GuestHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	var request guestRequest
	if decodeJSON(w, r, &request) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	guest, err := h.service.Create(r.Context(), identity.User.ID, r.PathValue("id"), request.domain())
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, guest)
}

func (h *GuestHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	var request guestRequest
	if decodeJSON(w, r, &request) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	guest, err := h.service.Update(r.Context(), identity.User.ID, r.PathValue("id"), r.PathValue("guestId"), request.domain())
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, guest)
}

func (h *GuestHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	err := h.service.Delete(r.Context(), identity.User.ID, r.PathValue("id"), r.PathValue("guestId"))
	if h.handleError(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *GuestHandler) RSVP(w http.ResponseWriter, r *http.Request) {
	var request rsvpRequest
	if decodeJSON(w, r, &request) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	result, err := h.service.RSVP(r.Context(), r.PathValue("slug"), domain.RSVPWrite{GuestToken: request.GuestToken, Status: request.Status, AttendeeCount: request.AttendeeCount, Message: request.Message})
	if h.handleError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *GuestHandler) handleError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, service.ErrInvalidGuest), errors.Is(err, service.ErrInvalidID):
		writeError(w, http.StatusUnprocessableEntity, "invalid_guest", "guest or rsvp data is invalid")
	case errors.Is(err, domain.ErrGuestNotFound), errors.Is(err, domain.ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "not_found", "guest or invitation was not found")
	default:
		slog.ErrorContext(r.Context(), "guest operation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
	return true
}

func (r guestRequest) domain() domain.GuestWrite {
	return domain.GuestWrite{Name: r.Name, Group: r.Group, Phone: r.Phone, Notes: r.Notes}
}
