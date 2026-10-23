package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/usecase"
)

const exportsPerMinute = 10 // both exports together, per user

// RegisterRoutes mounts the 17 vendor and shopping routes under /v1/planner/projects/:id, behind the same chain as every
// project route: bearer token, ProjectGuard (404 for somebody else's project), WriteGuard (an archived project is
// read-only; GET and the exports stay allowed).
//
// The static segments (/vendors/export-contacts, /shopping-items/apply-template, …) sit beside the parameterised ones;
// Gin resolves the static ones first.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	exportLimit := newUserLimiter(exportsPerMinute, time.Minute)

	g := r.Group("/v1/planner/projects/:id")
	g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(planner), plannerhttp.WriteGuard())
	{
		g.GET("/vendors", h.ListVendors)
		g.POST("/vendors", h.CreateVendor)
		g.GET("/vendors/export-contacts", exportLimit.middleware(), h.ExportContacts)
		g.GET("/vendors/:vendor_id", h.GetVendor)
		g.PUT("/vendors/:vendor_id", h.UpdateVendor)
		g.PATCH("/vendors/:vendor_id/status", h.SetVendorStatus)
		g.POST("/vendors/:vendor_id/lock-deal", h.LockDeal)
		g.DELETE("/vendors/:vendor_id", h.DeleteVendor)

		g.GET("/shopping-items", h.ListShopping)
		g.POST("/shopping-items", h.CreateShopping)
		g.POST("/shopping-items/apply-template", h.ApplyTemplate)
		g.POST("/shopping-items/link-to-budget", h.LinkToBudget)
		g.POST("/shopping-items/unlink-from-budget", h.UnlinkFromBudget)
		g.GET("/shopping-items/export-shopping-list", exportLimit.middleware(), h.ExportShopping)
		g.PUT("/shopping-items/:item_id", h.UpdateShopping)
		g.PATCH("/shopping-items/:item_id/status", h.SetShoppingStatus)
		g.DELETE("/shopping-items/:item_id", h.DeleteShopping)
	}
}
