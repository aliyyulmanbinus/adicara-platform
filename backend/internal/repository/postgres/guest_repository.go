package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GuestRepository struct {
	pool *pgxpool.Pool
}

func NewGuestRepository(pool *pgxpool.Pool) *GuestRepository { return &GuestRepository{pool: pool} }

const guestColumns = `g.id::text, g.name, COALESCE(g.group_name, ''), COALESCE(g.phone, ''), COALESCE(g.notes, ''), g.public_token, g.rsvp_status, g.created_at, g.updated_at`

func (r *GuestRepository) ListByOwner(ctx context.Context, userID, invitationID string) ([]domain.Guest, error) {
	query := `SELECT ` + guestColumns + ` FROM guests g JOIN invitations i ON i.id = g.invitation_id WHERE i.id = $1 AND i.user_id = $2 ORDER BY g.name, g.id`
	rows, err := r.pool.Query(ctx, query, invitationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list guests: %w", err)
	}
	defer rows.Close()
	guests := make([]domain.Guest, 0)
	for rows.Next() {
		guest, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		guests = append(guests, guest)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate guests: %w", err)
	}
	return guests, nil
}

func (r *GuestRepository) CreateForOwner(ctx context.Context, userID, invitationID, token string, input domain.GuestWrite) (domain.Guest, error) {
	query := `INSERT INTO guests (invitation_id, name, group_name, phone, notes, public_token)
		SELECT i.id, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7 FROM invitations i WHERE i.id = $1 AND i.user_id = $2
		RETURNING id::text, name, COALESCE(group_name, ''), COALESCE(phone, ''), COALESCE(notes, ''), public_token, rsvp_status, created_at, updated_at`
	guest, err := scanGuest(r.pool.QueryRow(ctx, query, invitationID, userID, input.Name, input.Group, input.Phone, input.Notes, token))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Guest{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.Guest{}, fmt.Errorf("create guest: %w", err)
	}
	return guest, nil
}

func (r *GuestRepository) UpdateForOwner(ctx context.Context, userID, invitationID, guestID string, input domain.GuestWrite) (domain.Guest, error) {
	query := `UPDATE guests g SET name=$4, group_name=NULLIF($5,''), phone=NULLIF($6,''), notes=NULLIF($7,''), updated_at=NOW()
		FROM invitations i WHERE g.id=$3 AND g.invitation_id=$1 AND i.id=g.invitation_id AND i.user_id=$2
		RETURNING g.id::text, g.name, COALESCE(g.group_name, ''), COALESCE(g.phone, ''), COALESCE(g.notes, ''), g.public_token, g.rsvp_status, g.created_at, g.updated_at`
	guest, err := scanGuest(r.pool.QueryRow(ctx, query, invitationID, userID, guestID, input.Name, input.Group, input.Phone, input.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Guest{}, domain.ErrGuestNotFound
	}
	if err != nil {
		return domain.Guest{}, fmt.Errorf("update guest: %w", err)
	}
	return guest, nil
}

func (r *GuestRepository) DeleteForOwner(ctx context.Context, userID, invitationID, guestID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM guests g USING invitations i WHERE g.id=$3 AND g.invitation_id=$1 AND i.id=g.invitation_id AND i.user_id=$2`, invitationID, userID, guestID)
	if err != nil {
		return fmt.Errorf("delete guest: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrGuestNotFound
	}
	return nil
}

func (r *GuestRepository) SubmitRSVP(ctx context.Context, slug string, input domain.RSVPWrite) (domain.RSVP, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RSVP{}, fmt.Errorf("begin rsvp: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var guestID, invitationID string
	err = tx.QueryRow(ctx, `SELECT g.id::text, i.id::text FROM guests g JOIN invitations i ON i.id=g.invitation_id WHERE i.slug=$1 AND i.status='published' AND g.public_token=$2`, slug, input.GuestToken).Scan(&guestID, &invitationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RSVP{}, domain.ErrGuestNotFound
	}
	if err != nil {
		return domain.RSVP{}, fmt.Errorf("find rsvp guest: %w", err)
	}
	var result domain.RSVP
	err = tx.QueryRow(ctx, `INSERT INTO rsvps (invitation_id, guest_id, status, attendee_count, message) VALUES ($1,$2,$3,$4,NULLIF($5,''))
		ON CONFLICT (guest_id) DO UPDATE SET status=EXCLUDED.status, attendee_count=EXCLUDED.attendee_count, message=EXCLUDED.message, updated_at=NOW()
		RETURNING status, attendee_count, COALESCE(message, ''), updated_at`, invitationID, guestID, input.Status, input.AttendeeCount, input.Message).Scan(&result.Status, &result.AttendeeCount, &result.Message, &result.UpdatedAt)
	if err != nil {
		return domain.RSVP{}, fmt.Errorf("upsert rsvp: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE guests SET rsvp_status=$2, updated_at=NOW() WHERE id=$1`, guestID, input.Status); err != nil {
		return domain.RSVP{}, fmt.Errorf("update guest rsvp: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.RSVP{}, fmt.Errorf("commit rsvp: %w", err)
	}
	return result, nil
}

func scanGuest(row rowScanner) (domain.Guest, error) {
	var guest domain.Guest
	err := row.Scan(&guest.ID, &guest.Name, &guest.Group, &guest.Phone, &guest.Notes, &guest.PublicToken, &guest.RSVPStatus, &guest.CreatedAt, &guest.UpdatedAt)
	return guest, err
}
