package domain

import (
	"errors"
	"time"
)

var ErrInvitationNotFound = errors.New("invitation not found")
var ErrInvitationSlugConflict = errors.New("invitation slug already exists")

type Invitation struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	EventType     string            `json:"event_type"`
	Slug          string            `json:"slug"`
	Title         string            `json:"title"`
	TemplateKey   string            `json:"template_key"`
	AllowIndexing bool              `json:"allow_indexing"`
	Hosts         []InvitationHost  `json:"hosts"`
	Events        []InvitationEvent `json:"events"`
	DesignData    map[string]string `json:"design_data"`
	Media         []InvitationMedia `json:"media"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type InvitationWrite struct {
	EventType     string
	Slug          string
	Title         string
	TemplateKey   string
	AllowIndexing bool
	Hosts         []InvitationHost
	Events        []InvitationEvent
	DesignData    map[string]string
}

type InvitationUpdate struct {
	EventType     *string
	Slug          *string
	Title         *string
	TemplateKey   *string
	AllowIndexing *bool
	Hosts         *[]InvitationHost
	Events        *[]InvitationEvent
	DesignData    *map[string]string
}

type InvitationMedia struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type InvitationHost struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type InvitationEvent struct {
	Name         string     `json:"name"`
	StartAt      time.Time  `json:"start_at"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	Timezone     string     `json:"timezone"`
	VenueName    string     `json:"venue_name"`
	VenueAddress string     `json:"venue_address"`
	MapURL       string     `json:"map_url,omitempty"`
}
