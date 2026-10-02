CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    template_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'archived')),
    allow_indexing BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (slug = LOWER(slug)),
    CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);

CREATE INDEX invitations_status_idx ON invitations (status);

CREATE TABLE invitation_hosts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX invitation_hosts_invitation_id_idx ON invitation_hosts (invitation_id);

CREATE TABLE invitation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ,
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    venue_name TEXT NOT NULL,
    venue_address TEXT NOT NULL,
    map_url TEXT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_at IS NULL OR end_at > start_at)
);

CREATE INDEX invitation_events_invitation_id_idx ON invitation_events (invitation_id);

