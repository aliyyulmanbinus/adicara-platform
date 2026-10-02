CREATE TABLE guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    group_name TEXT,
    phone TEXT,
    notes TEXT,
    public_token TEXT NOT NULL UNIQUE,
    rsvp_status TEXT NOT NULL DEFAULT 'belum_menjawab'
        CHECK (rsvp_status IN ('hadir', 'tidak_hadir', 'belum_menjawab')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX guests_invitation_id_idx ON guests (invitation_id);
CREATE INDEX guests_invitation_rsvp_idx ON guests (invitation_id, rsvp_status);

CREATE TABLE rsvps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL UNIQUE REFERENCES guests(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('hadir', 'tidak_hadir')),
    attendee_count SMALLINT NOT NULL DEFAULT 1 CHECK (attendee_count BETWEEN 0 AND 20),
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX rsvps_invitation_id_idx ON rsvps (invitation_id);
