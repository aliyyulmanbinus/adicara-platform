package http

import (
	"time"

	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/usecase"
)

const exportsPerMinute = 10 // both WhatsApp exports together, per user

// RegisterRoutes mounts the 17 rundown and playlist routes under /v1/planner/projects/:id, behind the same chain as every
// project route: bearer token, ProjectGuard (404 for somebody else's project), WriteGuard (an archived project is
// read-only; GET stays allowed, and the status of an agenda cannot be moved either).
//
// The static segments (/rundowns/summary, /rundowns/reorder, /rundowns/apply-template, /rundowns/export-wa) sit beside the
// parameterised ones; Gin resolves the static ones first. /rundowns/summary is polled every 5 seconds by the Hari-H
// screen and is deliberately not rate limited: several tablets of the committee may share one account.
func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, planner *plannerusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	exportLimit := newUserLimiter(exportsPerMinute, time.Minute)

	g := r.Group("/v1/planner/projects/:id")
	g.Use(authhttp.BearerAuth(authSvc), plannerhttp.ProjectGuard(planner), plannerhttp.WriteGuard())
	{
		g.GET("/rundowns", h.ListRundowns)
		g.GET("/rundowns/summary", h.Live)
		g.GET("/rundowns/export-wa", exportLimit.middleware(), h.ExportRundownWA)
		g.POST("/rundowns", h.CreateRundown)
		g.POST("/rundowns/reorder", h.ReorderRundowns)
		g.POST("/rundowns/apply-template", h.ApplyRundownTemplate)
		g.GET("/rundowns/:rundown_id", h.GetRundown)
		g.PUT("/rundowns/:rundown_id", h.UpdateRundown)
		g.PATCH("/rundowns/:rundown_id/status", h.SetStatus)
		g.DELETE("/rundowns/:rundown_id", h.DeleteRundown)

		g.GET("/playlists", h.ListPlaylists)
		g.GET("/playlists/export-wa", exportLimit.middleware(), h.ExportPlaylistWA)
		g.POST("/playlists", h.CreatePlaylist)
		g.POST("/playlists/reorder", h.ReorderPlaylists)
		g.POST("/playlists/apply-template", h.ApplyPlaylistTemplate)
		g.PUT("/playlists/:playlist_id", h.UpdatePlaylist)
		g.DELETE("/playlists/:playlist_id", h.DeletePlaylist)
	}
}
