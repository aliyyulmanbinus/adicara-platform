package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
)

const exportsPerMinute = 10 // GET /tasks/export-spreadsheet, per user

// RegisterRoutes mounts the timeline and document routes under
// /v1/planner/projects/:id, behind the same chain as every project route:
// bearer token, ProjectGuard (404 for somebody else's project), WriteGuard
// (an archived project is read-only; GET and the export stay allowed).
//
// The static segments (/tasks/reorder, /tasks/shift, /tasks/generate-defaults,
// /tasks/export-spreadsheet, /admin-docs/reorder, /admin-docs/apply-template)
// sit beside /tasks/:task_id and /admin-docs/:doc_id; Gin resolves the static
// ones first.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	exportLimit := newUserLimiter(exportsPerMinute, time.Minute)

	g := r.Group("/v1/planner/projects/:id")
	g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(planner), plannerhttp.WriteGuard())
	{
		g.GET("/tasks", h.ListTasks)
		g.POST("/tasks", h.CreateTask)
		g.POST("/tasks/reorder", h.ReorderTasks)
		g.POST("/tasks/shift", h.ShiftTasks)
		g.POST("/tasks/generate-defaults", h.GenerateDefaults)
		g.GET("/tasks/export-spreadsheet", exportLimit.middleware(), h.ExportSpreadsheet)
		g.PATCH("/tasks/:task_id", h.UpdateTask)
		g.PATCH("/tasks/:task_id/status", h.UpdateTaskStatus)
		g.DELETE("/tasks/:task_id", h.DeleteTask)

		g.GET("/admin-docs", h.ListDocs)
		g.POST("/admin-docs", h.CreateDoc)
		g.POST("/admin-docs/reorder", h.ReorderDocs)
		g.POST("/admin-docs/apply-template", h.ApplyDocTemplate)
		g.PATCH("/admin-docs/:doc_id", h.UpdateDoc)
		g.PATCH("/admin-docs/:doc_id/status", h.UpdateDocStatus)
		g.DELETE("/admin-docs/:doc_id", h.DeleteDoc)
		g.POST("/admin-docs/:doc_id/sync-budget", h.SyncBudget)
	}
}
