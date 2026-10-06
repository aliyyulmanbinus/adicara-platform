package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID
	Email    string
	Username string
	Name     *string
	RoleID   int16
	Status   string
}

const RoleCustomer int16 = 2
const StatusActive = "active"

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// AuthState is what BearerAuth needs to decide whether a signed access token
// may still be used: the account must exist and be active, and the token must
// not predate the last password change.
type AuthState struct {
	Status            string
	PasswordChangedAt *time.Time
}
