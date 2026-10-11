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
	RoleID   int16     `json:"role_id"`
	Status   string    `json:"status"`
}

func newUserResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Username: u.Username, Name: u.Name, RoleID: u.RoleID, Status: u.Status}
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func newTokenPairResponse(t domain.TokenPair) tokenPairResponse {
	return tokenPairResponse{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, ExpiresIn: t.ExpiresIn}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
