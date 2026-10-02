package domain

import (
	"errors"
	"time"
)

var ErrGuestNotFound = errors.New("guest not found")

type Guest struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Group       string    `json:"group,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	PublicToken string    `json:"public_token"`
	RSVPStatus  string    `json:"rsvp_status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GuestWrite struct {
	Name  string
	Group string
	Phone string
	Notes string
}

type RSVPWrite struct {
	GuestToken    string
	Status        string
	AttendeeCount int
	Message       string
}

type RSVP struct {
	Status        string    `json:"status"`
	AttendeeCount int       `json:"attendee_count"`
	Message       string    `json:"message,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}
