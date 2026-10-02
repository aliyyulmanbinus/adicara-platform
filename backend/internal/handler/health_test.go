package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubReadinessChecker struct {
	err error
}

func (s stubReadinessChecker) Ping(context.Context) error {
	return s.err
}

func TestLiveness(t *testing.T) {
	handler := NewHealthHandler(stubReadinessChecker{})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	handler.Liveness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestReadinessReportsDatabaseFailure(t *testing.T) {
	handler := NewHealthHandler(stubReadinessChecker{err: errors.New("unavailable")})
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	handler.Readiness(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
