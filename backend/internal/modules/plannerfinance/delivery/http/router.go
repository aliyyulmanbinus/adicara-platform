package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/usecase"
)

const exportsPerMinute = 10 // GET /budget/export, per user

// RegisterRoutes mounts the 32 finance routes under /v1/planner/projects/:id,
// behind the same chain as every project route: bearer token, ProjectGuard (404
// for somebody else's project), WriteGuard (an archived project is read-only;
// GET and the export stay allowed).
//
// The static segments (/scenarios/compare, /scenarios/apply,
// /scenarios/apply-preview, /budget/items/reorder, /budget/matrix, …) sit beside
// the parameterised ones; Gin resolves the static ones first.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	exportLimit := newUserLimiter(exportsPerMinute, time.Minute)

	g := r.Group("/v1/planner/projects/:id")
	g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(planner), plannerhttp.WriteGuard())
	{
		g.GET("/scenarios", h.ListScenarios)
		g.POST("/scenarios", h.CreateScenario)
		g.POST("/scenarios/compare", h.CompareScenarios)
		g.POST("/scenarios/apply-preview", h.ApplyPreview)
		g.POST("/scenarios/apply", h.Apply)
		g.GET("/scenarios/:scenario_id", h.GetScenario)
		g.PUT("/scenarios/:scenario_id", h.UpdateScenario)
		g.DELETE("/scenarios/:scenario_id", h.DeleteScenario)
		g.POST("/scenarios/:scenario_id/clone", h.CloneScenario)
		g.POST("/scenarios/:scenario_id/load-preset", h.LoadPreset)
		g.POST("/scenarios/:scenario_id/items", h.CreateScenarioItem)
		g.PUT("/scenarios/:scenario_id/items/:item_id", h.UpdateScenarioItem)
		g.DELETE("/scenarios/:scenario_id/items/:item_id", h.DeleteScenarioItem)

		g.GET("/contributors", h.ListContributors)
		g.POST("/contributors", h.CreateContributor)
		g.PUT("/contributors/:contributor_id", h.UpdateContributor)
		g.DELETE("/contributors/:contributor_id", h.DeleteContributor)

		g.GET("/savings", h.Savings)
		g.POST("/savings", h.CreateEntry)
		g.PUT("/savings/:entry_id", h.UpdateEntry)
		g.DELETE("/savings/:entry_id", h.DeleteEntry)

		g.GET("/budget/cashflow-summary", h.Cashflow)
		g.GET("/budget/matrix", h.Matrix)
		g.GET("/budget/items", h.ListItems)
		g.POST("/budget/items", h.CreateItem)
		g.POST("/budget/items/reorder", h.ReorderItems)
		g.PUT("/budget/items/:item_id", h.UpdateItem)
		g.PATCH("/budget/items/:item_id/payment", h.RecordPayment)
		g.DELETE("/budget/items/:item_id/payment/:stage", h.ClearPayment)
		g.DELETE("/budget/items/:item_id", h.DeleteItem)
		g.POST("/budget/apply-template", h.ApplyTemplate)
		g.GET("/budget/export", exportLimit.middleware(), h.Export)
	}
}
