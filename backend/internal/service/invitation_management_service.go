package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

var (
	ErrInvalidInvitation = errors.New("invalid invitation")
	ErrInvalidID         = errors.New("invalid id")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,49}$`)

var supportedTemplates = map[string]struct{}{
	"editorial-ivory":  {},
	"botanical-modern": {},
	"monochrome-luxe":  {},
}

type InvitationManagementRepository interface {
	ListByOwner(context.Context, string) ([]domain.Invitation, error)
	CreateForOwner(context.Context, string, domain.InvitationWrite) (domain.Invitation, error)
	FindByOwner(context.Context, string, string) (domain.Invitation, error)
	UpdateForOwner(context.Context, string, string, domain.InvitationUpdate) (domain.Invitation, error)
	DeleteForOwner(context.Context, string, string) error
	SetStatusForOwner(context.Context, string, string, string) (domain.Invitation, error)
}

type InvitationManagementService struct {
	repository InvitationManagementRepository
}

func NewInvitationManagementService(repository InvitationManagementRepository) *InvitationManagementService {
	return &InvitationManagementService{repository: repository}
}

func (s *InvitationManagementService) List(ctx context.Context, userID string) ([]domain.Invitation, error) {
	return s.repository.ListByOwner(ctx, userID)
}

func (s *InvitationManagementService) Create(ctx context.Context, userID string, input domain.InvitationWrite) (domain.Invitation, error) {
	input = normalizeInvitation(input)
	if err := validateInvitation(input); err != nil {
		return domain.Invitation{}, err
	}
	return s.repository.CreateForOwner(ctx, userID, input)
}

func (s *InvitationManagementService) Get(ctx context.Context, userID, invitationID string) (domain.Invitation, error) {
	if !uuidPattern.MatchString(invitationID) {
		return domain.Invitation{}, ErrInvalidID
	}
	return s.repository.FindByOwner(ctx, userID, invitationID)
}

func (s *InvitationManagementService) Update(ctx context.Context, userID, invitationID string, input domain.InvitationUpdate) (domain.Invitation, error) {
	if !uuidPattern.MatchString(invitationID) {
		return domain.Invitation{}, ErrInvalidID
	}
	normalizeUpdate(&input)
	if err := validateUpdate(input); err != nil {
		return domain.Invitation{}, err
	}
	return s.repository.UpdateForOwner(ctx, userID, invitationID, input)
}

func (s *InvitationManagementService) Delete(ctx context.Context, userID, invitationID string) error {
	if !uuidPattern.MatchString(invitationID) {
		return ErrInvalidID
	}
	return s.repository.DeleteForOwner(ctx, userID, invitationID)
}

func (s *InvitationManagementService) Publish(ctx context.Context, userID, invitationID string) (domain.Invitation, error) {
	invitation, err := s.Get(ctx, userID, invitationID)
	if err != nil {
		return domain.Invitation{}, err
	}
	input := domain.InvitationWrite{
		EventType: invitation.EventType, Slug: invitation.Slug, Title: invitation.Title,
		TemplateKey: invitation.TemplateKey, AllowIndexing: invitation.AllowIndexing,
		Hosts: invitation.Hosts, Events: invitation.Events,
	}
	if err := validateInvitation(input); err != nil {
		return domain.Invitation{}, err
	}
	return s.repository.SetStatusForOwner(ctx, userID, invitationID, "published")
}

func (s *InvitationManagementService) Unpublish(ctx context.Context, userID, invitationID string) (domain.Invitation, error) {
	if !uuidPattern.MatchString(invitationID) {
		return domain.Invitation{}, ErrInvalidID
	}
	return s.repository.SetStatusForOwner(ctx, userID, invitationID, "draft")
}

func normalizeInvitation(input domain.InvitationWrite) domain.InvitationWrite {
	input.EventType = strings.ToLower(strings.TrimSpace(input.EventType))
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.Title = strings.TrimSpace(input.Title)
	input.TemplateKey = strings.TrimSpace(input.TemplateKey)
	for index := range input.Hosts {
		input.Hosts[index].Name = strings.TrimSpace(input.Hosts[index].Name)
		input.Hosts[index].Role = strings.TrimSpace(input.Hosts[index].Role)
	}
	for index := range input.Events {
		input.Events[index].Name = strings.TrimSpace(input.Events[index].Name)
		input.Events[index].Timezone = strings.TrimSpace(input.Events[index].Timezone)
		input.Events[index].VenueName = strings.TrimSpace(input.Events[index].VenueName)
		input.Events[index].VenueAddress = strings.TrimSpace(input.Events[index].VenueAddress)
		input.Events[index].MapURL = strings.TrimSpace(input.Events[index].MapURL)
	}
	return input
}

func validateInvitation(input domain.InvitationWrite) error {
	if !eventTypePattern.MatchString(input.EventType) || len(input.Slug) < 3 || len(input.Slug) > 100 || !slugPattern.MatchString(input.Slug) {
		return ErrInvalidInvitation
	}
	if len(input.Title) < 3 || len(input.Title) > 160 {
		return ErrInvalidInvitation
	}
	if _, ok := supportedTemplates[input.TemplateKey]; !ok {
		return ErrInvalidInvitation
	}
	if len(input.Hosts) < 1 || len(input.Hosts) > 10 || len(input.Events) < 1 || len(input.Events) > 20 {
		return ErrInvalidInvitation
	}
	for _, host := range input.Hosts {
		if len(host.Name) < 1 || len(host.Name) > 120 || len(host.Role) < 1 || len(host.Role) > 120 {
			return ErrInvalidInvitation
		}
	}
	for _, event := range input.Events {
		if event.StartAt.IsZero() || len(event.Name) < 1 || len(event.Name) > 120 || len(event.Timezone) < 1 || len(event.Timezone) > 100 ||
			len(event.VenueName) < 1 || len(event.VenueName) > 160 || len(event.VenueAddress) < 1 || len(event.VenueAddress) > 500 {
			return ErrInvalidInvitation
		}
		if event.EndAt != nil && !event.EndAt.After(event.StartAt) {
			return ErrInvalidInvitation
		}
		if event.MapURL != "" && !validHTTPURL(event.MapURL) {
			return ErrInvalidInvitation
		}
	}
	return nil
}

func normalizeUpdate(input *domain.InvitationUpdate) {
	if input.EventType != nil {
		value := strings.ToLower(strings.TrimSpace(*input.EventType))
		input.EventType = &value
	}
	if input.Slug != nil {
		value := strings.ToLower(strings.TrimSpace(*input.Slug))
		input.Slug = &value
	}
	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		input.Title = &value
	}
	if input.TemplateKey != nil {
		value := strings.TrimSpace(*input.TemplateKey)
		input.TemplateKey = &value
	}
	if input.Hosts != nil {
		value := normalizeInvitation(domain.InvitationWrite{Hosts: *input.Hosts}).Hosts
		input.Hosts = &value
	}
	if input.Events != nil {
		value := normalizeInvitation(domain.InvitationWrite{Events: *input.Events}).Events
		input.Events = &value
	}
}

func validateUpdate(input domain.InvitationUpdate) error {
	if input.EventType == nil && input.Slug == nil && input.Title == nil && input.TemplateKey == nil &&
		input.AllowIndexing == nil && input.Hosts == nil && input.Events == nil {
		return ErrInvalidInvitation
	}
	if input.EventType != nil && !eventTypePattern.MatchString(*input.EventType) {
		return ErrInvalidInvitation
	}
	if input.Slug != nil && (len(*input.Slug) < 3 || len(*input.Slug) > 100 || !slugPattern.MatchString(*input.Slug)) {
		return ErrInvalidInvitation
	}
	if input.Title != nil && (len(*input.Title) < 3 || len(*input.Title) > 160) {
		return ErrInvalidInvitation
	}
	if input.TemplateKey != nil {
		if _, ok := supportedTemplates[*input.TemplateKey]; !ok {
			return ErrInvalidInvitation
		}
	}
	if input.Hosts != nil {
		probe := domain.InvitationWrite{EventType: "wedding", Slug: "valid-slug", Title: "Valid title", TemplateKey: "editorial-ivory", Hosts: *input.Hosts, Events: []domain.InvitationEvent{{Name: "Event", Timezone: "Asia/Jakarta", VenueName: "Venue", VenueAddress: "Address"}}}
		if validateInvitation(probe) != nil {
			return ErrInvalidInvitation
		}
	}
	if input.Events != nil {
		probe := domain.InvitationWrite{EventType: "wedding", Slug: "valid-slug", Title: "Valid title", TemplateKey: "editorial-ivory", Hosts: []domain.InvitationHost{{Name: "Host", Role: "Host"}}, Events: *input.Events}
		if validateInvitation(probe) != nil {
			return ErrInvalidInvitation
		}
	}
	return nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
