package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/usecase"
)

// maxBodyBytes bounds every JSON body; the largest legitimate one is a link-to-budget with 1.000 item ids (about 40 kB).
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

// ---------- vendors ----------

func (h *Handler) ListVendors(c *gin.Context) {
	out, err := h.svc.ListVendors(c.Request.Context(), plannerhttp.CurrentProject(c),
		usecase.VendorFilter{Scope: c.Query("scope"), Status: c.Query("status"), Category: c.Query("category")})
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) GetVendor(c *gin.Context) {
	id, ok := childID(c, "vendor_id", "vendor")
	if !ok {
		return
	}
	out, err := h.svc.GetVendor(c.Request.Context(), plannerhttp.CurrentProject(c), id)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateVendor(c *gin.Context) {
	var req domain.VendorRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateVendor(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateVendor(c *gin.Context) {
	id, ok := childID(c, "vendor_id", "vendor")
	if !ok {
		return
	}
	var req domain.VendorRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateVendor(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) SetVendorStatus(c *gin.Context) {
	id, ok := childID(c, "vendor_id", "vendor")
	if !ok {
		return
	}
	var req domain.VendorStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.SetVendorStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) LockDeal(c *gin.Context) {
	id, ok := childID(c, "vendor_id", "vendor")
	if !ok {
		return
	}
	var req domain.LockDealRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.LockDeal(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteVendor(c *gin.Context) {
	id, ok := childID(c, "vendor_id", "vendor")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteVendor(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

// ExportContacts streams the contact sheet of the fixed vendors as .xlsx. The file name comes from a slug of the title and
// the date, never from the raw title.
func (h *Handler) ExportContacts(c *gin.Context) {
	if !ginutil.XLSXFormat(c) {
		return
	}
	data, err := h.svc.ExportContacts(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	buf, err := writeBook(buildContactsWorkbook(data))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	ginutil.SendXLSX(c, "vendor", data.Project.Title, data.Today, buf)
}

// ---------- shopping ----------

func (h *Handler) ListShopping(c *gin.Context) {
	out, err := h.svc.ListShopping(c.Request.Context(), plannerhttp.CurrentProject(c),
		usecase.ShoppingFilter{Scope: c.Query("scope"), Category: c.Query("category"), Status: c.Query("status")})
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateShopping(c *gin.Context) {
	var req domain.ShoppingItemRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateShopping(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateShopping(c *gin.Context) {
	id, ok := childID(c, "item_id", "shopping item")
	if !ok {
		return
	}
	var req domain.UpdateShoppingItemRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UpdateShopping(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) SetShoppingStatus(c *gin.Context) {
	id, ok := childID(c, "item_id", "shopping item")
	if !ok {
		return
	}
	var req domain.ShoppingStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.SetShoppingStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteShopping(c *gin.Context) {
	id, ok := childID(c, "item_id", "shopping item")
	if !ok {
		return
	}
	noContent(c, h.svc.DeleteShopping(c.Request.Context(), plannerhttp.CurrentProject(c), id))
}

func (h *Handler) ApplyTemplate(c *gin.Context) {
	var req domain.ShoppingTemplateRequest
	if c.Request.ContentLength != 0 && !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.ApplyTemplate(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) LinkToBudget(c *gin.Context) {
	var req domain.LinkToBudgetRequest
	if c.Request.ContentLength != 0 && !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.LinkToBudget(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) UnlinkFromBudget(c *gin.Context) {
	var req domain.UnlinkFromBudgetRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.svc.UnlinkFromBudget(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

// ExportShopping streams the whole shopping list as .xlsx.
func (h *Handler) ExportShopping(c *gin.Context) {
	if !ginutil.XLSXFormat(c) {
		return
	}
	data, err := h.svc.ExportShopping(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	buf, err := writeBook(buildShoppingWorkbook(data))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	ginutil.SendXLSX(c, "belanja", data.Project.Title, data.Today, buf)
}
