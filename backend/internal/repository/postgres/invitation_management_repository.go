package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (r *InvitationRepository) ListByOwner(ctx context.Context, userID string) ([]domain.Invitation, error) {
	const query = `
		SELECT id::text, status, event_type, slug, title, template_key, allow_indexing, design_data, created_at, updated_at
		FROM invitations
		WHERE user_id = $1
		ORDER BY updated_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	invitations := make([]domain.Invitation, 0)
	for rows.Next() {
		invitation, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitations: %w", err)
	}
	rows.Close()
	for index := range invitations {
		invitations[index].Hosts, err = r.findHosts(ctx, invitations[index].ID)
		if err != nil {
			return nil, err
		}
		invitations[index].Events, err = r.findEvents(ctx, invitations[index].ID)
		if err != nil {
			return nil, err
		}
		invitations[index].Media, err = r.findMedia(ctx, invitations[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return invitations, nil
}

func (r *InvitationRepository) CreateForOwner(ctx context.Context, userID string, input domain.InvitationWrite) (domain.Invitation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Invitation{}, fmt.Errorf("begin invitation create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		INSERT INTO invitations (user_id, status, event_type, slug, title, template_key, allow_indexing, design_data)
		VALUES ($1, 'draft', $2, $3, $4, $5, $6, $7)
		RETURNING id::text, status, event_type, slug, title, template_key, allow_indexing, design_data, created_at, updated_at`
	invitation, err := scanInvitation(tx.QueryRow(ctx, query,
		userID, input.EventType, input.Slug, input.Title, input.TemplateKey, input.AllowIndexing, input.DesignData,
	))
	if isUniqueViolation(err) {
		return domain.Invitation{}, domain.ErrInvitationSlugConflict
	}
	if err != nil {
		return domain.Invitation{}, err
	}
	if err := replaceInvitationChildren(ctx, tx, invitation.ID, input.Hosts, input.Events); err != nil {
		return domain.Invitation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Invitation{}, fmt.Errorf("commit invitation create: %w", err)
	}
	invitation.Hosts = input.Hosts
	invitation.Events = input.Events
	return invitation, nil
}

func (r *InvitationRepository) FindByOwner(ctx context.Context, userID, invitationID string) (domain.Invitation, error) {
	const query = `
		SELECT id::text, status, event_type, slug, title, template_key, allow_indexing, design_data, created_at, updated_at
		FROM invitations
		WHERE id = $1 AND user_id = $2`
	invitation, err := scanInvitation(r.pool.QueryRow(ctx, query, invitationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Invitation{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Hosts, err = r.findHosts(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Events, err = r.findEvents(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Media, err = r.findMedia(ctx, invitation.ID)
	return invitation, err
}

func (r *InvitationRepository) UpdateForOwner(ctx context.Context, userID, invitationID string, input domain.InvitationUpdate) (domain.Invitation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Invitation{}, fmt.Errorf("begin invitation update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		UPDATE invitations
		SET event_type = COALESCE($3, event_type),
		    slug = COALESCE($4, slug),
		    title = COALESCE($5, title),
		    template_key = COALESCE($6, template_key),
		    allow_indexing = COALESCE($7, allow_indexing),
		    design_data = COALESCE($8, design_data),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, status, event_type, slug, title, template_key, allow_indexing, design_data, created_at, updated_at`
	invitation, err := scanInvitation(tx.QueryRow(ctx, query,
		invitationID, userID, input.EventType, input.Slug, input.Title, input.TemplateKey, input.AllowIndexing, input.DesignData,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Invitation{}, domain.ErrInvitationNotFound
	}
	if isUniqueViolation(err) {
		return domain.Invitation{}, domain.ErrInvitationSlugConflict
	}
	if err != nil {
		return domain.Invitation{}, err
	}

	if input.Hosts != nil {
		if err := replaceHosts(ctx, tx, invitation.ID, *input.Hosts); err != nil {
			return domain.Invitation{}, err
		}
	}
	if input.Events != nil {
		if err := replaceEvents(ctx, tx, invitation.ID, *input.Events); err != nil {
			return domain.Invitation{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Invitation{}, fmt.Errorf("commit invitation update: %w", err)
	}
	return r.FindByOwner(ctx, userID, invitationID)
}

func (r *InvitationRepository) DeleteForOwner(ctx context.Context, userID, invitationID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM invitations WHERE id = $1 AND user_id = $2`, invitationID, userID)
	if err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvitationNotFound
	}
	return nil
}

func (r *InvitationRepository) SetStatusForOwner(ctx context.Context, userID, invitationID, status string) (domain.Invitation, error) {
	const query = `
		UPDATE invitations
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, status, event_type, slug, title, template_key, allow_indexing, design_data, created_at, updated_at`
	invitation, err := scanInvitation(r.pool.QueryRow(ctx, query, invitationID, userID, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Invitation{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Hosts, err = r.findHosts(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Events, err = r.findEvents(ctx, invitation.ID)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Media, err = r.findMedia(ctx, invitation.ID)
	return invitation, err
}

type rowScanner interface {
	Scan(...any) error
}

func scanInvitation(row rowScanner) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := row.Scan(
		&invitation.ID,
		&invitation.Status,
		&invitation.EventType,
		&invitation.Slug,
		&invitation.Title,
		&invitation.TemplateKey,
		&invitation.AllowIndexing,
		&invitation.DesignData,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)
	if err != nil {
		return domain.Invitation{}, err
	}
	return invitation, nil
}

func replaceInvitationChildren(ctx context.Context, tx pgx.Tx, invitationID string, hosts []domain.InvitationHost, events []domain.InvitationEvent) error {
	if err := replaceHosts(ctx, tx, invitationID, hosts); err != nil {
		return err
	}
	return replaceEvents(ctx, tx, invitationID, events)
}

func replaceHosts(ctx context.Context, tx pgx.Tx, invitationID string, hosts []domain.InvitationHost) error {
	if _, err := tx.Exec(ctx, `DELETE FROM invitation_hosts WHERE invitation_id = $1`, invitationID); err != nil {
		return fmt.Errorf("delete invitation hosts: %w", err)
	}
	const query = `INSERT INTO invitation_hosts (invitation_id, name, role, sort_order) VALUES ($1, $2, $3, $4)`
	for index, host := range hosts {
		if _, err := tx.Exec(ctx, query, invitationID, host.Name, host.Role, index); err != nil {
			return fmt.Errorf("insert invitation host: %w", err)
		}
	}
	return nil
}

func replaceEvents(ctx context.Context, tx pgx.Tx, invitationID string, events []domain.InvitationEvent) error {
	if _, err := tx.Exec(ctx, `DELETE FROM invitation_events WHERE invitation_id = $1`, invitationID); err != nil {
		return fmt.Errorf("delete invitation events: %w", err)
	}
	const query = `
		INSERT INTO invitation_events
			(invitation_id, name, start_at, end_at, timezone, venue_name, venue_address, map_url, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)`
	for index, event := range events {
		if _, err := tx.Exec(ctx, query,
			invitationID, event.Name, event.StartAt, event.EndAt, event.Timezone,
			event.VenueName, event.VenueAddress, event.MapURL, index,
		); err != nil {
			return fmt.Errorf("insert invitation event: %w", err)
		}
	}
	return nil
}
