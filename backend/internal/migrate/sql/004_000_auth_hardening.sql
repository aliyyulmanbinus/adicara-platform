-- +goose Up
-- Both columns are nullable and unread by the previous release, so rolling the
-- image back after this migration is safe.

-- Access tokens issued before this instant are no longer accepted; set when the
-- password changes so a stolen access token dies with the old password.
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ;

-- Set when a refresh session is spent by rotation (not by logout or a password
-- change). Presenting a rotated token again later is treated as token theft.
ALTER TABLE refresh_sessions ADD COLUMN rotated_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE refresh_sessions DROP COLUMN IF EXISTS rotated_at;
ALTER TABLE users DROP COLUMN IF EXISTS password_changed_at;
