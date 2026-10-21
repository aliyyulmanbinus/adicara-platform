package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

const (
	createsPerMinute    = 10 // POST /projects, per user
	dashboardsPerMinute = 60 // GET /projects/:id/dashboard, per user
)

// RegisterRoutes mounts /v1/planner. Everything needs a bearer token; every
// route with a project :id also goes through ProjectGuard and WriteGuard.
//
// Gin resolves the static segments (/projects/summary, /projects/sample) ahead
// of /projects/:id, so they do not need to be registered first.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	createLimit := newUserLimiter(createsPerMinute, time.Minute)
	dashboardLimit := newUserLimiter(dashboardsPerMinute, time.Minute)

	planner := r.Group("/v1/planner")
	planner.Use(authhttp.BearerAuth(authSvc))
	{
		planner.GET("/event-types", h.EventTypes)

		planner.GET("/projects", h.ListProjects)
		planner.POST("/projects", createLimit.middleware(), h.CreateProject)
		planner.GET("/projects/summary", h.Summary)
		planner.POST("/projects/sample", h.CreateSample)

		project := planner.Group("/projects/:id")
		project.Use(ProjectGuard(svc), WriteGuard())
		{
			project.GET("", h.GetProject)
			project.PATCH("", h.UpdateProject)
			project.DELETE("", h.DeleteProject)
			project.PATCH("/status", h.ChangeStatus)
			project.GET("/dashboard", dashboardLimit.middleware(), h.Dashboard)

			project.GET("/sessions", h.ListSessions)
			project.POST("/sessions", h.CreateSession)
			project.POST("/sessions/reorder", h.ReorderSessions)
			project.PUT("/sessions/:session_id", h.UpdateSession)
			project.DELETE("/sessions/:session_id", h.DeleteSession)
		}
	}
	// PUT /projects/:id/invitation is intentionally not registered until the
	// Undangan Digital module exists (PLANNER_SPEC/05_BUKU_TAMU/BE.md §5).
}
