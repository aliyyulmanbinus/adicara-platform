package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
)

type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(c *gin.Context) {
	var body registerRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	user, err := h.svc.Register(c.Request.Context(), body.Email, body.Password, body.Username)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": newUserResponse(user)})
}

func (h *Handler) Login(c *gin.Context) {
	var body loginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	identifier := body.Email
	if identifier == "" {
		identifier = body.Username
	}

	user, tokens, err := h.svc.Login(c.Request.Context(), identifier, body.Password)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newUserResponse(user), "tokens": newTokenPairResponse(tokens)})
}

func (h *Handler) Refresh(c *gin.Context) {
	var body refreshRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	tokens, err := h.svc.Refresh(c.Request.Context(), body.RefreshToken)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": newTokenPairResponse(tokens)})
}

func (h *Handler) Logout(c *gin.Context) {
	var body logoutRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	if err := h.svc.Logout(c.Request.Context(), body.RefreshToken); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var body changePasswordRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), CurrentUserID(c), body.CurrentPassword, body.NewPassword); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

const userIDContextKey = "userID"

func BearerAuth(svc *usecase.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			ginutil.Error(c, apierr.Unauthorized("missing bearer token"))
			c.Abort()
			return
		}

		id, err := svc.Authenticate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			ginutil.Error(c, err)
			c.Abort()
			return
		}

		c.Set(userIDContextKey, id)
		c.Next()
	}
}

// CurrentUserID reads the id BearerAuth stored on the request context.
func CurrentUserID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(userIDContextKey)
	uid, _ := id.(uuid.UUID)
	return uid
}
