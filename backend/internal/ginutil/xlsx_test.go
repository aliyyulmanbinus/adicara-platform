package ginutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func run(t *testing.T, target string, h gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", h)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	return rec
}

func TestXLSXFormatAcceptsNothingOrXLSXAndRefusesTheRest(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  bool
	}{
		{"", true}, {"?format=", true}, {"?format=xlsx", true},
		{"?format=pdf", false}, {"?format=XLSX", false}, {"?format=csv", false}, {"?format=xlsx%20", false},
	} {
		var ok bool
		rec := run(t, "/x"+tc.query, func(c *gin.Context) {
			if ok = XLSXFormat(c); ok {
				c.Status(http.StatusNoContent)
			}
		})
		if ok != tc.want {
			t.Errorf("%q: XLSXFormat = %v, want %v", tc.query, ok, tc.want)
		}
		if !tc.want {
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"code":"invalid_input"`) || !strings.Contains(rec.Body.String(), "format: must be xlsx") {
				t.Errorf("%q: refused with %d %s", tc.query, rec.Code, rec.Body.String())
			}
		} else if rec.Code != http.StatusNoContent {
			t.Errorf("%q: answered %d, the handler should have gone on", tc.query, rec.Code)
		}
	}
}

func TestSendXLSXNamesTheFileFromASlugAndNeverCachesIt(t *testing.T) {
	day := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	rec := run(t, "/x", func(c *gin.Context) {
		SendXLSX(c, "jadwal", "Pesta \"Ulang\" Tahun 🎂 \\ X\r\nSet-Cookie: a=b", day, []byte("PK"))
	})
	if rec.Code != http.StatusOK || rec.Body.String() != "PK" {
		t.Fatalf("answered %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != XLSXContentType {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
	cd := rec.Header().Get("Content-Disposition")
	if want := `attachment; filename="jadwal-pesta-ulang-tahun-x-set-cookie-a-b-2026-10-04.xlsx"`; cd != want {
		t.Errorf("Content-Disposition = %q, want %q", cd, want)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Errorf("a title must never reach a header: Set-Cookie = %q", rec.Header().Get("Set-Cookie"))
	}
}
