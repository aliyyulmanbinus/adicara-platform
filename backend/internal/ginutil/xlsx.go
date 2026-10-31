package ginutil

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
)

// XLSXContentType is the media type of every spreadsheet the API sends.
const XLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// XLSXFormat reports whether the request may go on to build a workbook. ?format may be left out (the default is xlsx) or
// be xlsx; any other value (the PDF library is not approved) answers 400 invalid_input "format: must be xlsx", and false
// is returned. Every spreadsheet route of the API starts with it so they all read the parameter the same way.
func XLSXFormat(c *gin.Context) bool {
	if f := c.Query("format"); f != "" && f != "xlsx" {
		Error(c, apierr.BadRequest("invalid_input", "format: must be xlsx"))
		return false
	}

	return true
}

// SendXLSX answers with the workbook as an attachment named <prefix>-<slug of title>-<day>.xlsx; the name never comes
// from the raw title. A workbook holds the user's own plans, money and contacts, so it is never cached.
func SendXLSX(c *gin.Context, prefix, title string, day time.Time, body []byte) {
	name := fmt.Sprintf("%s-%s-%s.xlsx", prefix, Slug(title), day.Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, XLSXContentType, body)
}
