package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/service"
)

const sessionCookieName = "adicara_session"
const csrfCookieName = "adicara_csrf"

type AuthService interface {
	Register(context.Context, service.RegisterInput) (service.AuthResult, error)
	Login(context.Context, service.LoginInput) (service.AuthResult, error)
	Authenticate(context.Context, string) (domain.SessionIdentity, error)
	ValidateCSRF(domain.SessionIdentity, string) error
	Logout(context.Context, domain.SessionIdentity) error
}

type AuthHandler struct {
	service      AuthService
	cookieSecure bool
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User      domain.User `json:"user"`
	CSRFToken string      `json:"csrf_token"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func NewAuthHandler(service AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{service: service, cookieSecure: cookieSecure}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	result, err := h.service.Register(r.Context(), service.RegisterInput{
		Email: request.Email, DisplayName: request.DisplayName, Password: request.Password,
	})
	switch {
	case errors.Is(err, service.ErrInvalidAuthInput):
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", "email, name, or password does not meet requirements")
	case errors.Is(err, domain.ErrEmailConflict):
		writeError(w, http.StatusConflict, "email_conflict", "email is already registered")
	case err != nil:
		slog.ErrorContext(r.Context(), "failed to register user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	default:
		h.setSessionCookies(w, result.Session, result.CSRFToken, result.ExpiresAt)
		writeJSON(w, http.StatusCreated, authResponse{User: result.User, CSRFToken: result.CSRFToken, ExpiresAt: result.ExpiresAt})
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	result, err := h.service.Login(r.Context(), service.LoginInput{Email: request.Email, Password: request.Password})
	if errors.Is(err, domain.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to log in", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	h.setSessionCookies(w, result.Session, result.CSRFToken, result.ExpiresAt)
	writeJSON(w, http.StatusOK, authResponse{User: result.User, CSRFToken: result.CSRFToken, ExpiresAt: result.ExpiresAt})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.requireSession(w, r, false)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": identity.User, "expires_at": identity.ExpiresAt})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.requireSession(w, r, true)
	if !ok {
		return
	}
	if err := h.service.Logout(r.Context(), identity); err != nil {
		slog.ErrorContext(r.Context(), "failed to log out", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) requireSession(w http.ResponseWriter, r *http.Request, requireCSRF bool) (domain.SessionIdentity, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication is required")
		return domain.SessionIdentity{}, false
	}
	identity, err := h.service.Authenticate(r.Context(), cookie.Value)
	if errors.Is(err, domain.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication is required")
		return domain.SessionIdentity{}, false
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to authenticate session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return domain.SessionIdentity{}, false
	}
	if requireCSRF && h.service.ValidateCSRF(identity, r.Header.Get("X-CSRF-Token")) != nil {
		writeError(w, http.StatusForbidden, "invalid_csrf", "csrf token is invalid")
		return domain.SessionIdentity{}, false
	}
	return identity, true
}

func (h *AuthHandler) setSessionCookies(w http.ResponseWriter, token, csrfToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookieName, Value: csrfToken, Path: "/", Expires: expiresAt,
		HttpOnly: false, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: false, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}
