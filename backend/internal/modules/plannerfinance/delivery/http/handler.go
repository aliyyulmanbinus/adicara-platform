package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/usecase"
)

// maxBodyBytes bounds every JSON body; the largest legitimate one is a reorder
// list of 300 UUIDs (about 12 kB).
const maxBodyBytes = 64 << 10

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

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

// bindJSONWithout is bindJSON for a body that must not carry the given keys:
// the struct has no field for them, so decoding alone would drop them silently.
func bindJSONWithout(c *gin.Context, dst any, forbidden ...string) bool {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
	if err != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}
	var keys map[string]json.RawMessage
	if json.Unmarshal(body, &keys) != nil || json.Unmarshal(body, dst) != nil {
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}
	for _, k := range forbidden {
		if _, found := keys[k]; found {
			ginutil.Error(c, apierr.BadRequest("invalid_input", k+": cannot be changed"))
			return false
		}
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

// ---------- scenarios ----------

func (h *Handler) ListScenarios(c *gin.Context) {
	out, err := h.svc.ListScenarios(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) GetScenario(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	out, err := h.svc.GetScenario(c.Request.Context(), plannerhttp.CurrentProject(c), id)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateScenario(c *gin.Context) {
	var req domain.CreateScenarioRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateScenario(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateScenario(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	var req domain.UpdateScenarioRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateScenario(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteScenario(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteScenario(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) CloneScenario(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	var req domain.CloneScenarioRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CloneScenario(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) LoadPreset(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	var req domain.LoadPresetRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.LoadPreset(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateScenarioItem(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	var req domain.ScenarioItemRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateScenarioItem(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateScenarioItem(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	itemID, ok := childID(c, "item_id", "scenario item")
	if !ok {
		return
	}
	var req domain.UpdateScenarioItemRequest
	if !bindJSONWithout(c, &req, "item_key") {
		return
	}
	out, err := h.svc.UpdateScenarioItem(c.Request.Context(), plannerhttp.CurrentProject(c), id, itemID, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteScenarioItem(c *gin.Context) {
	id, ok := childID(c, "scenario_id", "scenario")
	if !ok {
		return
	}
	itemID, ok := childID(c, "item_id", "scenario item")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteScenarioItem(c.Request.Context(), plannerhttp.CurrentProject(c), id, itemID))
}

func (h *Handler) CompareScenarios(c *gin.Context) {
	var req domain.CompareRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CompareScenarios(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ApplyPreview(c *gin.Context) {
	var req domain.ApplyScenarioRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.ApplyPreview(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) Apply(c *gin.Context) {
	var req domain.ApplyScenarioRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.Apply(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

// ---------- contributors and savings ----------

func (h *Handler) ListContributors(c *gin.Context) {
	out, err := h.svc.ListContributors(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateContributor(c *gin.Context) {
	var req domain.ContributorRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateContributor(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateContributor(c *gin.Context) {
	id, ok := childID(c, "contributor_id", "contributor")
	if !ok {
		return
	}
	var req domain.ContributorRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateContributor(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteContributor(c *gin.Context) {
	id, ok := childID(c, "contributor_id", "contributor")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteContributor(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) Savings(c *gin.Context) {
	var contributor *uuid.UUID
	if raw := c.Query("contributor_id"); raw != "" {
		id, ok := plannerhttp.ParseID(raw)
		if !ok {
			ginutil.Error(c, apierr.BadRequest("invalid_input", "contributor_id: must be a valid id"))
			return
		}
		contributor = &id
	}
	out, err := h.svc.Savings(c.Request.Context(), plannerhttp.CurrentProject(c), contributor)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateEntry(c *gin.Context) {
	var req domain.SavingsEntryRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateEntry(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateEntry(c *gin.Context) {
	id, ok := childID(c, "entry_id", "deposit")
	if !ok {
		return
	}
	var req domain.SavingsEntryRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateEntry(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteEntry(c *gin.Context) {
	id, ok := childID(c, "entry_id", "deposit")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteEntry(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

// ---------- budget ----------

func (h *Handler) Cashflow(c *gin.Context) {
	out, err := h.svc.Cashflow(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) Matrix(c *gin.Context) {
	out, err := h.svc.Matrix(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ListItems(c *gin.Context) {
	out, err := h.svc.ListItems(c.Request.Context(), plannerhttp.CurrentProject(c), c.Query("scope"))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateItem(c *gin.Context) {
	var req domain.BudgetItemRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateItem(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	id, ok := childID(c, "item_id", "budget item")
	if !ok {
		return
	}
	var req domain.BudgetItemRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateItem(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) RecordPayment(c *gin.Context) {
	id, ok := childID(c, "item_id", "budget item")
	if !ok {
		return
	}
	var req domain.PaymentRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.RecordPayment(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ClearPayment(c *gin.Context) {
	id, ok := childID(c, "item_id", "budget item")
	if !ok {
		return
	}
	out, err := h.svc.ClearPayment(c.Request.Context(), plannerhttp.CurrentProject(c), id, c.Param("stage"))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	id, ok := childID(c, "item_id", "budget item")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteItem(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) ReorderItems(c *gin.Context) {
	var req struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.ReorderItems(c.Request.Context(), plannerhttp.CurrentProject(c), req.IDs)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ApplyTemplate(c *gin.Context) {
	var req domain.ApplyTemplateRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.ApplyTemplate(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

// Export streams the whole budget, the session matrix and the savings as .xlsx.
// The file name comes from a slug of the title and the date, never from the raw title.
func (h *Handler) Export(c *gin.Context) {
	if f := c.Query("format"); f != "xlsx" {
		ginutil.Error(c, apierr.BadRequest("invalid_input", "format: must be xlsx"))
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

	name := fmt.Sprintf("keuangan-%s-%s.xlsx", ginutil.Slug(data.Project.Title), data.Today.Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Data(http.StatusOK, xlsxContentType, buf.Bytes())
}
