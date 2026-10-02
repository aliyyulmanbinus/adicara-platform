package domain

import (
	"errors"
	"time"
)

var (
	ErrEmailConflict      = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid session")
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserCredential struct {
	User
	PasswordHash string
}

type SessionIdentity struct {
	User          User
	TokenHash     []byte
	CSRFTokenHash []byte
	ExpiresAt     time.Time
}
