ALTER TABLE invitations ADD COLUMN design_data JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE invitation_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('bride', 'groom', 'gallery')),
    position INTEGER NOT NULL CHECK (position BETWEEN 0 AND 4),
    mime_type TEXT NOT NULL CHECK (mime_type IN ('image/jpeg', 'image/png', 'image/webp')),
    content BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (invitation_id, kind, position)
);
CREATE INDEX invitation_media_invitation_idx ON invitation_media (invitation_id);
