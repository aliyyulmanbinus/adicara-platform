package server

import (
	"net/http"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/handler"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/middleware"
)

func New(
	address string,
	health *handler.HealthHandler,
	invitation *handler.InvitationHandler,
	auth *handler.AuthHandler,
	management *handler.InvitationManagementHandler,
	guests *handler.GuestHandler,
) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Liveness)
	mux.HandleFunc("GET /readyz", health.Readiness)
	mux.HandleFunc("GET /api/v1/public/invitations/{slug}", invitation.GetPublished)
	mux.HandleFunc("POST /api/v1/auth/register", auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", auth.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", auth.Logout)
	mux.HandleFunc("GET /api/v1/me", auth.Me)
	mux.HandleFunc("GET /api/v1/invitations", management.List)
	mux.HandleFunc("POST /api/v1/invitations", management.Create)
	mux.HandleFunc("GET /api/v1/invitations/{id}", management.Get)
	mux.HandleFunc("PATCH /api/v1/invitations/{id}", management.Update)
	mux.HandleFunc("DELETE /api/v1/invitations/{id}", management.Delete)
	mux.HandleFunc("POST /api/v1/invitations/{id}/publish", management.Publish)
	mux.HandleFunc("POST /api/v1/invitations/{id}/unpublish", management.Unpublish)
	mux.HandleFunc("GET /api/v1/invitations/{id}/guests", guests.List)
	mux.HandleFunc("POST /api/v1/invitations/{id}/guests", guests.Create)
	mux.HandleFunc("PATCH /api/v1/invitations/{id}/guests/{guestId}", guests.Update)
	mux.HandleFunc("DELETE /api/v1/invitations/{id}/guests/{guestId}", guests.Delete)
	mux.HandleFunc("POST /api/v1/public/invitations/{slug}/rsvp", guests.RSVP)

	return &http.Server{
		Addr:              address,
		Handler:           middleware.Recover(middleware.RequestLogger(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
