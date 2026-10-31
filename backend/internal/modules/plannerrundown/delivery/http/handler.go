package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/usecase"
)

// maxBodyBytes bounds every JSON body; the largest legitimate one is a reorder of 500 ids (about 20 kB) or an agenda with
// two texts of 2.000 characters.
const maxBodyBytes = 64 << 10

type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler { return &Handler{svc: svc} }

func bindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}

	return true
}

// bindOptional is bindJSON for a request whose body may be absent (apply-template without a body means append).
func bindOptional(c *gin.Context, dst any) bool {
	return c.Request.ContentLength == 0 || bindJSON(c, dst)
}

// childID reads a canonical UUID from a path parameter; anything else is 404.
func childID(c *gin.Context, param, resource string) (uuid.UUID, bool) {
	id, ok := plannerhttp.ParseID(c.Param(param))
	if !ok {
		ginutil.Error(c, apierr.NotFound(resource))
	}

	return id, ok
}

func respond(c *gin.Context, status int, data any, err error) {
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	c.JSON(status, gin.H{"data": data})
}

func noContent(c *gin.Context, err error) {
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- agendas ----------

func (h *Handler) ListRundowns(c *gin.Context) {
	out, err := h.svc.ListRundowns(c.Request.Context(), plannerhttp.CurrentProject(c), c.Query("session_id"))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) Live(c *gin.Context) {
	out, err := h.svc.Live(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) GetRundown(c *gin.Context) {
	id, ok := childID(c, "rundown_id", "agenda")
	if !ok {
		return
	}
	out, err := h.svc.GetRundown(c.Request.Context(), plannerhttp.CurrentProject(c), id)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateRundown(c *gin.Context) {
	var req domain.RundownRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateRundown(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateRundown(c *gin.Context) {
	id, ok := childID(c, "rundown_id", "agenda")
	if !ok {
		return
	}
	var req domain.RundownRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateRundown(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) SetStatus(c *gin.Context) {
	id, ok := childID(c, "rundown_id", "agenda")
	if !ok {
		return
	}
	var req domain.RundownStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.SetStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteRundown(c *gin.Context) {
	id, ok := childID(c, "rundown_id", "agenda")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteRundown(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) ReorderRundowns(c *gin.Context) {
	var req domain.ReorderRundownsRequest
	if !bindJSON(c, &req) {
		return
	}
	noContent(c, h.svc.ReorderRundowns(c.Request.Context(), plannerhttp.CurrentProject(c), req))
}

func (h *Handler) ApplyRundownTemplate(c *gin.Context) {
	var req domain.ApplyTemplateRequest
	if !bindOptional(c, &req) {
		return
	}
	out, err := h.svc.ApplyRundownTemplate(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ExportRundownWA(c *gin.Context) {
	out, err := h.svc.ExportRundownWA(c.Request.Context(), plannerhttp.CurrentProject(c), c.Query("session_id"))
	respond(c, http.StatusOK, out, err)
}

// ---------- playlist ----------

func (h *Handler) ListPlaylists(c *gin.Context) {
	out, err := h.svc.ListPlaylists(c.Request.Context(), plannerhttp.CurrentProject(c), usecase.SongFilter{
		MomentTag: c.Query("moment_tag"), Format: c.Query("format"), Status: c.Query("status"),
	})
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreatePlaylist(c *gin.Context) {
	var req domain.PlaylistRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreatePlaylist(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdatePlaylist(c *gin.Context) {
	id, ok := childID(c, "playlist_id", "song")
	if !ok {
		return
	}
	var req domain.PlaylistRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdatePlaylist(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeletePlaylist(c *gin.Context) {
	id, ok := childID(c, "playlist_id", "song")
	if !ok {
		return
	}
	noContent(c, h.svc.DeletePlaylist(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) ReorderPlaylists(c *gin.Context) {
	var req domain.ReorderPlaylistsRequest
	if !bindJSON(c, &req) {
		return
	}
	noContent(c, h.svc.ReorderPlaylists(c.Request.Context(), plannerhttp.CurrentProject(c), req))
}

func (h *Handler) ApplyPlaylistTemplate(c *gin.Context) {
	var req domain.ApplyTemplateRequest
	if !bindOptional(c, &req) {
		return
	}
	out, err := h.svc.ApplyPlaylistTemplate(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ExportPlaylistWA(c *gin.Context) {
	out, err := h.svc.ExportPlaylistWA(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}
