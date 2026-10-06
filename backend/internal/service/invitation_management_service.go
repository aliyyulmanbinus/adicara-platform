package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
)

var (
	ErrInvalidInvitation = errors.New("invalid invitation")
	ErrInvalidID         = errors.New("invalid id")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,49}$`)
var instagramPattern = regexp.MustCompile(`^@?[A-Za-z0-9._]+$`)

var supportedTemplates = map[string]struct{}{
	"editorial-ivory":  {},
	"botanical-modern": {},
	"monochrome-luxe":  {},
	"batak-senja":      {},
}

var designFields = map[string]int{
	"opening_heading": 120, "opening_text": 1000, "greeting_text": 1000,
	"quote_text": 1000, "quote_source": 120, "bride_parents": 240,
	"groom_parents": 240, "bride_instagram": 80, "groom_instagram": 80,
	"couple_note": 240, "prayer_title": 120, "prayer_text": 1000,
	"prayer_source": 120, "gift_text": 1000, "gift_bank": 120,
	"gift_account_name": 120, "gift_account_number": 80,
	"closing_text": 1000, "rsvp_text": 500, "music_url": 1000,
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
		Hosts: invitation.Hosts, Events: invitation.Events, DesignData: invitation.DesignData,
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
	if input.DesignData == nil {
		input.DesignData = map[string]string{}
	}
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
	for key, value := range input.DesignData {
		input.DesignData[key] = strings.TrimSpace(value)
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
	if err := validateDesignData(input.DesignData); err != nil {
		return err
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
	if input.DesignData != nil {
		value := *input.DesignData
		for key, text := range value {
			value[key] = strings.TrimSpace(text)
		}
		input.DesignData = &value
	}
}

func validateUpdate(input domain.InvitationUpdate) error {
	if input.EventType == nil && input.Slug == nil && input.Title == nil && input.TemplateKey == nil &&
		input.AllowIndexing == nil && input.Hosts == nil && input.Events == nil && input.DesignData == nil {
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
		if len(*input.Hosts) < 1 || len(*input.Hosts) > 10 {
			return ErrInvalidInvitation
		}
		for _, host := range *input.Hosts {
			if len(host.Name) < 1 || len(host.Name) > 120 || len(host.Role) < 1 || len(host.Role) > 120 {
				return ErrInvalidInvitation
			}
		}
	}
	if input.Events != nil {
		probe := domain.InvitationWrite{EventType: "wedding", Slug: "valid-slug", Title: "Valid title", TemplateKey: "editorial-ivory", Hosts: []domain.InvitationHost{{Name: "Host", Role: "Host"}}, Events: *input.Events}
		if validateInvitation(probe) != nil {
			return ErrInvalidInvitation
		}
	}
	if input.DesignData != nil {
		if err := validateDesignData(*input.DesignData); err != nil {
			return err
		}
	}
	return nil
}

func validateDesignData(data map[string]string) error {
	for key, value := range data {
		limit, ok := designFields[key]
		if !ok || utf8.RuneCountInString(value) > limit {
			return ErrInvalidInvitation
		}
		if key == "music_url" && value != "" && (!validHTTPURL(value) || !strings.HasPrefix(strings.ToLower(value), "https://")) {
			return ErrInvalidInvitation
		}
		if (key == "bride_instagram" || key == "groom_instagram") && value != "" && !instagramPattern.MatchString(value) {
			return ErrInvalidInvitation
		}
	}
	return nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
