package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (r *InvitationRepository) PutMedia(ctx context.Context, userID, invitationID, kind string, position int, mime string, content []byte) (domain.InvitationMedia, error) {
	const query = `
		INSERT INTO invitation_media (invitation_id, kind, position, mime_type, content)
		SELECT id, $3, $4, $5, $6 FROM invitations WHERE id = $1 AND user_id = $2
		ON CONFLICT (invitation_id, kind, position)
		DO UPDATE SET mime_type = EXCLUDED.mime_type, content = EXCLUDED.content, created_at = now()
		RETURNING id::text, kind, position`
	var item domain.InvitationMedia
	err := r.pool.QueryRow(ctx, query, invitationID, userID, kind, position, mime, content).Scan(&item.ID, &item.Kind, &item.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.InvitationMedia{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.InvitationMedia{}, fmt.Errorf("store invitation media: %w", err)
	}
	item.URL = "/api/v1/media/" + item.ID
	return item, nil
}

func (r *InvitationRepository) PublicMedia(ctx context.Context, id string) (string, []byte, error) {
	var mime string
	var content []byte
	err := r.pool.QueryRow(ctx, `
		SELECT m.mime_type, m.content FROM invitation_media m
		JOIN invitations i ON i.id = m.invitation_id
		WHERE m.id = $1 AND i.status = 'published'`, id).Scan(&mime, &content)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, domain.ErrInvitationNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("read invitation media: %w", err)
	}
	return mime, content, nil
}
