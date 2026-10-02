package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) FindPublishedBySlug(ctx context.Context, slug string) (domain.Invitation, error) {
	const invitationQuery = `
		SELECT id::text, status, event_type, slug, title, template_key, allow_indexing, created_at, updated_at
		FROM invitations
		WHERE slug = $1 AND status = 'published'`

	var invitation domain.Invitation
	err := r.pool.QueryRow(ctx, invitationQuery, slug).Scan(
		&invitation.ID,
		&invitation.Status,
		&invitation.EventType,
		&invitation.Slug,
		&invitation.Title,
		&invitation.TemplateKey,
		&invitation.AllowIndexing,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Invitation{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.Invitation{}, fmt.Errorf("query invitation: %w", err)
	}

	hosts, err := r.findHosts(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}
	events, err := r.findEvents(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}

	invitation.Hosts = hosts
	invitation.Events = events
	return invitation, nil
}

func (r *InvitationRepository) findHosts(ctx context.Context, invitationID string) ([]domain.InvitationHost, error) {
	const query = `
		SELECT name, role
		FROM invitation_hosts
		WHERE invitation_id = $1
		ORDER BY sort_order, id`

	rows, err := r.pool.Query(ctx, query, invitationID)
	if err != nil {
		return nil, fmt.Errorf("query invitation hosts: %w", err)
	}
	defer rows.Close()

	hosts := make([]domain.InvitationHost, 0)
	for rows.Next() {
		var host domain.InvitationHost
		if err := rows.Scan(&host.Name, &host.Role); err != nil {
			return nil, fmt.Errorf("scan invitation host: %w", err)
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitation hosts: %w", err)
	}
	return hosts, nil
}

func (r *InvitationRepository) findEvents(ctx context.Context, invitationID string) ([]domain.InvitationEvent, error) {
	const query = `
		SELECT name, start_at, end_at, timezone, venue_name, venue_address, COALESCE(map_url, '')
		FROM invitation_events
		WHERE invitation_id = $1
		ORDER BY sort_order, start_at`

	rows, err := r.pool.Query(ctx, query, invitationID)
	if err != nil {
		return nil, fmt.Errorf("query invitation events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.InvitationEvent, 0)
	for rows.Next() {
		var event domain.InvitationEvent
		if err := rows.Scan(
			&event.Name,
			&event.StartAt,
			&event.EndAt,
			&event.Timezone,
			&event.VenueName,
			&event.VenueAddress,
			&event.MapURL,
		); err != nil {
			return nil, fmt.Errorf("scan invitation event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitation events: %w", err)
	}
	return events, nil
}
