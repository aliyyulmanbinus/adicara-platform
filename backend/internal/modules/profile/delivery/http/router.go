package http

import (
	"github.com/gin-gonic/gin"

	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/profile/usecase"
)

func RegisterRoutes(r gin.IRouter, authSvc *authusecase.Service, svc *usecase.Service) {
	h := NewHandler(svc)
	me := r.Group("/v1/me")
	me.Use(authhttp.BearerAuth(authSvc))
	me.GET("", h.Me)
	me.PATCH("", h.PatchMe)
}
