-- +goose Up
CREATE TABLE m_role (
    id SMALLINT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX m_role_name_lower_idx ON m_role (lower(name)) WHERE deleted_at IS NULL;

INSERT INTO m_role (id, name) VALUES
    (1, 'admin'),
    (2, 'customer');

-- +goose Down
DROP TABLE IF EXISTS m_role;
