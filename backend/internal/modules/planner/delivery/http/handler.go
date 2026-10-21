package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

// maxBodyBytes bounds every JSON body; the largest legitimate one (a project
// with its metadata) is well under a kilobyte.
const maxBodyBytes = 64 << 10

type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler {
	return &Handler{svc: svc}
}

func bindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}

	return true
}

// project renders p with its phase; a project whose zone cannot be loaded is a
// server problem (500), not a client one.
func (h *Handler) project(p domain.Project) (projectResponse, error) {
	phase, err := h.svc.Phase(p)
	if err != nil {
		return projectResponse{}, err
	}

	return newProjectResponse(p, phase), nil
}

// ---------- registry ----------

func (h *Handler) EventTypes(c *gin.Context) {
	// Static per release.
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"data": h.svc.EventTypes()})
}

// ---------- projects ----------

func (h *Handler) ListProjects(c *gin.Context) {
	uid := authhttp.CurrentUserID(c)
	status := c.Query("status")

	switch c.Query("view") {
	case "lite":
		out, err := h.svc.ListLite(c.Request.Context(), uid, status)
		if err != nil {
			ginutil.Error(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": out})
	case "":
		out, err := h.svc.ListCards(c.Request.Context(), uid, status)
		if err != nil {
			ginutil.Error(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": out})
	default:
		ginutil.Error(c, apierr.BadRequest("invalid_input", "view: must be lite or empty"))
	}
}

func (h *Handler) Summary(c *gin.Context) {
	out, err := h.svc.Summary(c.Request.Context(), authhttp.CurrentUserID(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (h *Handler) createdResponse(c *gin.Context, res domain.CreateProjectResult) {
	p, err := h.project(res.Project)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	body := gin.H{"project": p, "created": res.Created}
	status := http.StatusOK
	if res.Created {
		body["applied"] = res.Applied
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"data": body})
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req domain.InitProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	res, err := h.svc.CreateProject(c.Request.Context(), authhttp.CurrentUserID(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	h.createdResponse(c, res)
}

func (h *Handler) CreateSample(c *gin.Context) {
	res, err := h.svc.CreateSample(c.Request.Context(), authhttp.CurrentUserID(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	h.createdResponse(c, res)
}

func (h *Handler) GetProject(c *gin.Context) {
	detail, err := h.svc.ProjectDetail(c.Request.Context(), CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	p, err := h.project(detail.Project)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"project": p, "shiftable_tasks": detail.ShiftableTasks}})
}

func (h *Handler) UpdateProject(c *gin.Context) {
	var req domain.UpdateProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	res, err := h.svc.UpdateProject(c.Request.Context(), CurrentProject(c).ID, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	p, err := h.project(res.Project)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"project": p, "warnings": nonNilNotices(res.Warnings), "hints": nonNilNotices(res.Hints)}})
}

func nonNilNotices(n []domain.Notice) []domain.Notice {
	if n == nil {
		return []domain.Notice{}
	}

	return n
}

func (h *Handler) ChangeStatus(c *gin.Context) {
	var req domain.ChangeProjectStatusRequest
	if !bindJSON(c, &req) {
		return
	}

	updated, err := h.svc.ChangeStatus(c.Request.Context(), CurrentProject(c), req.Status)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	p, err := h.project(updated)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"project": p}})
}

func (h *Handler) DeleteProject(c *gin.Context) {
	if err := h.svc.DeleteProject(c.Request.Context(), CurrentProject(c), c.Query("confirm")); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) Dashboard(c *gin.Context) {
	d, err := h.svc.Dashboard(c.Request.Context(), CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": d})
}

// ---------- sessions ----------

func sessionID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := ParseID(c.Param("session_id"))
	if !ok {
		ginutil.Error(c, apierr.NotFound("session"))
	}

	return id, ok
}

func (h *Handler) ListSessions(c *gin.Context) {
	out, err := h.svc.ListSessions(c.Request.Context(), CurrentProject(c).ID)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newSessionResponses(out)})
}

func (h *Handler) CreateSession(c *gin.Context) {
	var req domain.CreateSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	s, err := h.svc.CreateSession(c.Request.Context(), CurrentProject(c).ID, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": s.Brief()})
}

func (h *Handler) UpdateSession(c *gin.Context) {
	id, ok := sessionID(c)
	if !ok {
		return
	}
	var req domain.UpdateSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	s, err := h.svc.UpdateSession(c.Request.Context(), CurrentProject(c).ID, id, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": s.Brief()})
}

func (h *Handler) ReorderSessions(c *gin.Context) {
	var req domain.ReorderSessionsRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.ReorderSessions(c.Request.Context(), CurrentProject(c).ID, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newSessionBriefs(out)})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	id, ok := sessionID(c)
	if !ok {
		return
	}

	deleteAgendas := false
	if v := c.Query("delete_agendas"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			ginutil.Error(c, apierr.BadRequest("invalid_input", "delete_agendas: must be true or false"))
			return
		}
		deleteAgendas = b
	}

	if err := h.svc.DeleteSession(c.Request.Context(), CurrentProject(c).ID, id, c.Query("move_to"), deleteAgendas); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
