package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
)

// maxBodyBytes bounds every JSON body; the largest legitimate one is a reorder
// list of 500 UUIDs (about 20 kB).
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

// bindOptionalJSON is bindJSON for a body that may be absent: an empty body
// leaves dst untouched.
func bindOptionalJSON(c *gin.Context, dst any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
	if err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return true
	}
	if err := json.Unmarshal(body, dst); err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}

	return true
}

// childID reads a canonical UUID from a path parameter; anything else is 404.
func childID(c *gin.Context, param, resource string) (uuid.UUID, bool) {
	id, ok := plannerhttp.ParseID(c.Param(param))
	if !ok {
		ginutil.Error(c, apierr.NotFound(resource))
	}

	return id, ok
}

// ---------- tasks ----------

func (h *Handler) ListTasks(c *gin.Context) {
	out, err := h.svc.ListTasks(c.Request.Context(), plannerhttp.CurrentProject(c), usecase.TaskQuery{
		Status: c.Query("status"), Category: c.Query("category"), Scope: c.Query("scope"), Phase: c.Query("phase"),
	})
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newTaskListResponse(out)})
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req domain.CreateTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	t, err := h.svc.CreateTask(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": newTaskResponse(t)})
}

func (h *Handler) UpdateTask(c *gin.Context) {
	id, ok := childID(c, "task_id", "task")
	if !ok {
		return
	}
	var req domain.UpdateTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	t, err := h.svc.UpdateTask(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newTaskResponse(t)})
}

func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	id, ok := childID(c, "task_id", "task")
	if !ok {
		return
	}
	var req domain.UpdateTaskStatusRequest
	if !bindJSON(c, &req) {
		return
	}

	t, err := h.svc.SetTaskStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req.Status)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newTaskResponse(t)})
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id, ok := childID(c, "task_id", "task")
	if !ok {
		return
	}
	if err := h.svc.DeleteTask(c.Request.Context(), plannerhttp.CurrentProject(c), id); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ReorderTasks(c *gin.Context) {
	var req domain.ReorderRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.ReorderTasks(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newTaskListResponse(out)})
}

func (h *Handler) ShiftTasks(c *gin.Context) {
	var req domain.ShiftTasksRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.ShiftTasks(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	out.Warnings = notices(out.Warnings)

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (h *Handler) GenerateDefaults(c *gin.Context) {
	var req domain.GenerateTimelineRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.GenerateDefaults(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	out.Warnings = notices(out.Warnings)

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// ExportSpreadsheet streams all tasks of the project as .xlsx. The file name
// comes from a slug of the title and the date, never from the raw title.
func (h *Handler) ExportSpreadsheet(c *gin.Context) {
	if !ginutil.XLSXFormat(c) {
		return
	}
	data, err := h.svc.Export(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	f, err := buildWorkbook(data)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	defer func() { _ = f.Close() }()
	buf, err := f.WriteToBuffer()
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	ginutil.SendXLSX(c, "jadwal", data.Project.Title, data.Today, buf.Bytes())
}

// ---------- documents ----------

func (h *Handler) ListDocs(c *gin.Context) {
	out, err := h.svc.ListDocs(c.Request.Context(), plannerhttp.CurrentProject(c), c.Query("subject"), c.Query("status"))
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newDocListResponse(out)})
}

func (h *Handler) CreateDoc(c *gin.Context) {
	var req domain.CreateAdminDocRequest
	if !bindJSON(c, &req) {
		return
	}

	d, err := h.svc.CreateDoc(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": newDocResponse(d)})
}

func (h *Handler) UpdateDoc(c *gin.Context) {
	id, ok := childID(c, "doc_id", "document")
	if !ok {
		return
	}
	var req domain.UpdateAdminDocRequest
	if !bindJSON(c, &req) {
		return
	}

	d, err := h.svc.UpdateDoc(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newDocResponse(d)})
}

func (h *Handler) UpdateDocStatus(c *gin.Context) {
	id, ok := childID(c, "doc_id", "document")
	if !ok {
		return
	}
	var req domain.UpdateAdminDocStatusRequest
	if !bindJSON(c, &req) {
		return
	}

	d, err := h.svc.SetDocStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req.Status)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newDocResponse(d)})
}

func (h *Handler) DeleteDoc(c *gin.Context) {
	id, ok := childID(c, "doc_id", "document")
	if !ok {
		return
	}
	if err := h.svc.DeleteDoc(c.Request.Context(), plannerhttp.CurrentProject(c), id); err != nil {
		ginutil.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ReorderDocs(c *gin.Context) {
	var req domain.ReorderRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.ReorderDocs(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": newDocListResponse(out)})
}

func (h *Handler) ApplyDocTemplate(c *gin.Context) {
	var req domain.ApplyDocsRequest
	if !bindJSON(c, &req) {
		return
	}

	out, err := h.svc.ApplyDocTemplate(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	out.Warnings = notices(out.Warnings)

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// SyncBudget answers 201 when a new budget line was created and 200 otherwise.
// The body is optional ({"sub_event_scope": "akad"}).
func (h *Handler) SyncBudget(c *gin.Context) {
	id, ok := childID(c, "doc_id", "document")
	if !ok {
		return
	}
	var req domain.SyncBudgetRequest
	if !bindOptionalJSON(c, &req) {
		return
	}

	out, err := h.svc.SyncBudget(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	out.Warnings = notices(out.Warnings)

	status := http.StatusOK
	if out.Created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"data": out})
}
