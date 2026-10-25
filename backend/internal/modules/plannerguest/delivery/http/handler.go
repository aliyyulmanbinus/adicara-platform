package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/delivery/http/xlsx"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/usecase"
)

const (
	// maxBodyBytes bounds every JSON body but the import: the largest legitimate one is a guest with notes (about 8 kB).
	maxBodyBytes = 64 << 10
	// maxImportBodyBytes bounds the body of bulk-import: a 5 MiB file is about 6.7 MiB of base64 (05 BE 2.6).
	maxImportBodyBytes = 7<<20 + 512<<10

	xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler { return &Handler{svc: svc} }

// bindJSON reads a body of at most limit bytes. Anything that is not the expected JSON is 400 invalid_json, a body over
// the limit too, except the upload of the import, which is 413 file_too_large.
func bindJSON(c *gin.Context, dst any, limit int64) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	if err := c.ShouldBindJSON(dst); err != nil {
		var tooBig *http.MaxBytesError
		if limit == maxImportBodyBytes && errors.As(err, &tooBig) {
			ginutil.Error(c, errFileTooLarge())
			return false
		}
		ginutil.Error(c, apierr.BadRequest("invalid_json", "invalid JSON body"))
		return false
	}

	return true
}

func errFileTooLarge() *apierr.Error {
	return apierr.New(http.StatusRequestEntityTooLarge, "file_too_large", "file too large")
}

// childID reads a canonical UUID from a path parameter; anything else is 404.
func childID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := plannerhttp.ParseID(c.Param("guest_id"))
	if !ok {
		ginutil.Error(c, apierr.NotFound("guest"))
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

// ---------- list and CRUD ----------

func (h *Handler) ListGuests(c *gin.Context) {
	out, err := h.svc.ListGuests(c.Request.Context(), plannerhttp.CurrentProject(c), usecase.GuestFilter{
		Side: c.Query("side"), Category: c.Query("category"), RSVP: c.Query("rsvp"), Sent: c.Query("sent"), Scope: c.Query("scope"),
		Search: c.Query("search"), Limit: c.Query("limit"), Offset: c.Query("offset"),
	})
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) Summary(c *gin.Context) {
	out, err := h.svc.Summary(c.Request.Context(), plannerhttp.CurrentProject(c))
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) GetGuest(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	out, err := h.svc.GetGuest(c.Request.Context(), plannerhttp.CurrentProject(c), id)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CreateGuest(c *gin.Context) {
	var req domain.GuestRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.CreateGuest(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

func (h *Handler) UpdateGuest(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	var req domain.GuestRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.UpdateGuest(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) DeleteGuest(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteGuest(c.Request.Context(), plannerhttp.CurrentProject(c), id); err != nil {
		ginutil.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SetSentStatus(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	var req domain.SentStatusRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.SetSentStatus(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) SetRSVP(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	var req domain.RSVPRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.SetRSVP(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CateringProjection(c *gin.Context) {
	out, err := h.svc.CateringProjection(c.Request.Context(), plannerhttp.CurrentProject(c), c.Query("buffer_percent"), c.Query("buffer_pax"))
	respond(c, http.StatusOK, out, err)
}

// ---------- reception ----------

func (h *Handler) ReceptionOverview(c *gin.Context) {
	out, err := h.svc.ReceptionOverview(c.Request.Context(), plannerhttp.CurrentProject(c), usecase.ReceptionFilter{
		Status: c.Query("status"), Category: c.Query("category"), Search: c.Query("search"), Limit: c.Query("limit"), Offset: c.Query("offset"),
	})
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) ScanQR(c *gin.Context) {
	var req domain.ScanQRRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.ScanQR(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) Checkin(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	var req domain.CheckinRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.Checkin(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) CancelCheckin(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	out, err := h.svc.CancelCheckin(c.Request.Context(), plannerhttp.CurrentProject(c), id)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) SetSouvenir(c *gin.Context) {
	id, ok := childID(c)
	if !ok {
		return
	}
	var req domain.SouvenirRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.SetSouvenir(c.Request.Context(), plannerhttp.CurrentProject(c), id, req)
	respond(c, http.StatusOK, out, err)
}

func (h *Handler) WalkIn(c *gin.Context) {
	var req domain.WalkInRequest
	if !bindJSON(c, &req, maxBodyBytes) {
		return
	}
	out, err := h.svc.WalkIn(c.Request.Context(), plannerhttp.CurrentProject(c), req)
	respond(c, http.StatusCreated, out, err)
}

// ---------- import ----------

// boolQuery reads true or false; absent is false.
func boolQuery(c *gin.Context, name string) (bool, bool) {
	switch c.Query(name) {
	case "", "false":
		return false, true
	case "true":
		return true, true
	}
	ginutil.Error(c, apierr.BadRequest("invalid_input", name+": must be true or false"))

	return false, false
}

// BulkImport takes the file as JSON {filename, content_base64} (05 BE 2.6), decodes it, and hands the records to the
// usecase. The body limit (7.5 MiB) is raised for this route only; the decoded file may be 5 MiB.
func (h *Handler) BulkImport(c *gin.Context) {
	dry, ok := boolQuery(c, "dry_run")
	if !ok {
		return
	}
	skipInvalid, ok := boolQuery(c, "skip_invalid")
	if !ok {
		return
	}
	skipDuplicates, ok := boolQuery(c, "skip_duplicates")
	if !ok {
		return
	}
	var req domain.ImportRequest
	if !bindJSON(c, &req, maxImportBodyBytes) {
		return
	}

	data, kind, err := domain.DecodeUpload(req)
	if err != nil {
		ginutil.Error(c, uploadError(err))
		return
	}
	var recs []domain.Record
	if kind == domain.KindXLSX {
		recs, err = xlsx.Read(data)
	} else {
		recs, err = domain.DecodeCSV(data)
	}
	if err != nil {
		ginutil.Error(c, uploadError(err))
		return
	}

	out, err := h.svc.Import(c.Request.Context(), plannerhttp.CurrentProject(c), recs, domain.ImportOptions{SkipInvalid: skipInvalid, SkipDuplicates: skipDuplicates}, dry)
	respond(c, http.StatusOK, out, err)
}

// uploadError maps what decoding a file can return: too large is 413, a file problem is 400 invalid_file.
func uploadError(err error) error {
	if errors.Is(err, domain.ErrTooLarge) {
		return errFileTooLarge()
	}
	var fe *domain.FileError
	if errors.As(err, &fe) {
		return apierr.BadRequest("invalid_file", "file: "+fe.Reason)
	}

	return err
}

// ---------- files ----------

// xlsxRequested answers 400 unless ?format=xlsx (the only format; the PDF library is not approved, 05 BE 2.6).
func xlsxRequested(c *gin.Context) bool {
	if c.Query("format") != "xlsx" {
		ginutil.Error(c, apierr.BadRequest("invalid_input", "format: must be xlsx"))
		return false
	}

	return true
}

// Personal data (names, phone numbers, ticket codes): never cached. The file name comes from a slug of the title and the
// date, never from the raw title.
func sendWorkbook(c *gin.Context, prefix, title string, day time.Time, body []byte) {
	name := fmt.Sprintf("%s-%s-%s.xlsx", prefix, ginutil.Slug(title), day.Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, xlsxContentType, body)
}

func (h *Handler) ExportGuests(c *gin.Context) {
	if !xlsxRequested(c) {
		return
	}
	data, err := h.svc.ExportGuests(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	buf, err := writeBook(buildGuestsWorkbook(data))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	sendWorkbook(c, "tamu", data.Project.Title, data.Today, buf)
}

func (h *Handler) ExportAttendance(c *gin.Context) {
	if !xlsxRequested(c) {
		return
	}
	data, err := h.svc.ExportAttendance(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	buf, err := writeBook(buildAttendanceWorkbook(data))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	sendWorkbook(c, "kehadiran", data.Project.Title, data.Today, buf)
}

func (h *Handler) ImportTemplate(c *gin.Context) {
	if !xlsxRequested(c) {
		return
	}
	data, err := h.svc.ImportTemplate(c.Request.Context(), plannerhttp.CurrentProject(c))
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	buf, err := xlsx.Template(data.Input)
	if err != nil {
		ginutil.Error(c, err)
		return
	}
	sendWorkbook(c, "templat-tamu", data.Project.Title, data.Today, buf)
}
