package http

import (
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

type userResponse struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Name     *string   `json:"name"`
}

func newUserResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Username: u.Username, Name: u.Name}
}
