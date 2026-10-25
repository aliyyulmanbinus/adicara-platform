package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/usecase"
)

// Limits are the caps per user and minute. A reception desk scans one ticket every few seconds, so scan-qr is generous;
// the import is rare and heavy; the three downloads (list, attendance, template) share one counter. A limit <= 0 turns
// that cap off (tests that make many calls; production uses DefaultLimits).
type Limits struct{ Scans, Imports, Downloads int }

var DefaultLimits = Limits{Scans: 120, Imports: 5, Downloads: 10}

// RegisterRoutes mounts the 19 guest routes under /v1/planner/projects/:id, behind the same chain as every project route:
// bearer token, ProjectGuard (404 for somebody else's project), WriteGuard (an archived project is read-only; GET, the
// downloads and the template stay allowed).
//
// The static segments (/guests/summary, /guests/export, …) sit beside /guests/:guest_id; Gin resolves the static ones first.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service) {
	RegisterRoutesWithLimits(r, authSvc, planner, svc, DefaultLimits)
}

// RegisterRoutesWithLimits is RegisterRoutes with other caps.
func RegisterRoutesWithLimits(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service, limits Limits) {
	h := NewHandler(svc)
	scanLimit := newUserLimiter(limits.Scans, time.Minute)
	importLimit := newUserLimiter(limits.Imports, time.Minute)
	downloadLimit := newUserLimiter(limits.Downloads, time.Minute)

	g := r.Group("/v1/planner/projects/:id/guests")
	g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(planner), plannerhttp.WriteGuard())
	{
		// Guest list and invitations.
		g.GET("", h.ListGuests)
		g.POST("", h.CreateGuest)
		g.GET("/summary", h.Summary)
		g.GET("/catering-projection", h.CateringProjection)
		g.GET("/export", downloadLimit.middleware(), h.ExportGuests)
		g.GET("/import-template", downloadLimit.middleware(), h.ImportTemplate)
		g.POST("/bulk-import", importLimit.middleware(), h.BulkImport)

		// Reception desk.
		g.GET("/reception-overview", h.ReceptionOverview)
		g.GET("/export-attendance", downloadLimit.middleware(), h.ExportAttendance)
		g.POST("/scan-qr", scanLimit.middleware(), h.ScanQR)
		g.POST("/walk-in", h.WalkIn)

		g.GET("/:guest_id", h.GetGuest)
		g.PUT("/:guest_id", h.UpdateGuest)
		g.DELETE("/:guest_id", h.DeleteGuest)
		g.PATCH("/:guest_id/sent-status", h.SetSentStatus)
		g.PATCH("/:guest_id/rsvp", h.SetRSVP)
		g.PATCH("/:guest_id/souvenir", h.SetSouvenir)
		g.POST("/:guest_id/checkin", h.Checkin)
		g.DELETE("/:guest_id/checkin", h.CancelCheckin)
	}
}
