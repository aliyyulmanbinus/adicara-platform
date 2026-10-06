package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

const maxImageBytes = 5 << 20

var mediaIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type InvitationMediaRepository interface {
	PutMedia(context.Context, string, string, string, int, string, []byte) (domain.InvitationMedia, error)
	PublicMedia(context.Context, string) (string, []byte, error)
}

type InvitationMediaHandler struct {
	repository InvitationMediaRepository
	auth       *AuthHandler
}

func NewInvitationMediaHandler(repository InvitationMediaRepository, auth *AuthHandler) *InvitationMediaHandler {
	return &InvitationMediaHandler{repository: repository, auth: auth}
}

func (h *InvitationMediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.auth.requireSession(w, r, true)
	if !ok {
		return
	}
	if !mediaIDPattern.MatchString(r.PathValue("id")) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_id", "invitation id is invalid")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "image upload is invalid or too large")
		return
	}
	kind := r.FormValue("kind")
	position, err := strconv.Atoi(r.FormValue("position"))
	if err != nil || position < 0 || position > 4 || (kind != "gallery" && (kind != "bride" && kind != "groom" || position != 0)) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_media", "image kind or position is invalid")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "image is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil || len(content) == 0 || len(content) > maxImageBytes {
		writeError(w, http.StatusUnprocessableEntity, "invalid_media", "image must be 5 MB or smaller")
		return
	}
	mime := http.DetectContentType(content)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_media", "use a JPEG, PNG, or WebP image")
		return
	}
	item, err := h.repository.PutMedia(r.Context(), identity.User.ID, r.PathValue("id"), kind, position, mime, content)
	if errors.Is(err, domain.ErrInvitationNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "invitation was not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "media upload failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "image could not be saved")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *InvitationMediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !mediaIDPattern.MatchString(r.PathValue("mediaId")) {
		http.NotFound(w, r)
		return
	}
	mime, content, err := h.repository.PublicMedia(r.Context(), r.PathValue("mediaId"))
	if errors.Is(err, domain.ErrInvitationNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "media read failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "image could not be loaded")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(content)
}
