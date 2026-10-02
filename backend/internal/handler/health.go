package handler

import (
	"context"
	"net/http"
	"time"
)

type ReadinessChecker interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	checker ReadinessChecker
}

func NewHealthHandler(checker ReadinessChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.checker.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "database is not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
