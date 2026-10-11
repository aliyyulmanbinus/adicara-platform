-- +goose Up
CREATE TABLE refresh_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_sessions_user_idx ON refresh_sessions (user_id, revoked_at);

-- +goose Down
DROP TABLE IF EXISTS refresh_sessions;
