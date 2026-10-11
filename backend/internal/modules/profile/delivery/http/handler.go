package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/profile/usecase"
)

type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Me(c *gin.Context) {
	me, err := h.svc.Get(c.Request.Context(), authhttp.CurrentUserID(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user": newUserResponse(me.User),
		},
	})
}

func (h *Handler) PatchMe(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	out, err := h.svc.UpdateProfile(c.Request.Context(), authhttp.CurrentUserID(c), body.Name)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newUserResponse(out)})
}
