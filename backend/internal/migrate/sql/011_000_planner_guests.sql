-- +goose Up
-- Tabel tamu bersama untuk layanan Planner dan Undangan Digital (dual-key).
-- invitation_id sengaja TANPA foreign key: tabel undangan belum ada di backend.
-- Migrasi modul undangan menambahkan constraint-nya (lihat 05_BUKU_TAMU/BE.md bab 6).
CREATE TABLE planner_guests (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES planner_projects(id) ON DELETE CASCADE,
    invitation_id UUID,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    guest_side TEXT,                                    -- kunci sisi dari registry (wedding: pria/wanita); NULL = belum ditentukan
    category TEXT NOT NULL DEFAULT 'lainnya',           -- kunci kategori dari registry; kustom diperbolehkan
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    table_number TEXT,
    pax_count INT NOT NULL DEFAULT 2 CHECK (pax_count BETWEEN 1 AND 20),
    phone_number TEXT,
    formatted_wa_number TEXT,
    invitation_sent_status TEXT NOT NULL DEFAULT 'unsent' CHECK (invitation_sent_status IN ('unsent', 'sent', 'delivered')),
    invitation_sent_at TIMESTAMPTZ,
    rsvp_status TEXT NOT NULL DEFAULT 'unconfirmed' CHECK (rsvp_status IN ('unconfirmed', 'attending', 'declined', 'maybe')),
    pax_actual INT NOT NULL DEFAULT 0 CHECK (pax_actual >= 0),
    rsvp_submitted_at TIMESTAMPTZ,
    qr_code_token TEXT NOT NULL UNIQUE,                 -- 16 byte acak (hex), dibuat usecase
    is_checked_in BOOLEAN NOT NULL DEFAULT FALSE,
    checked_in_at TIMESTAMPTZ,
    souvenir_received BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (project_id IS NOT NULL OR invitation_id IS NOT NULL),
    -- Dengan project_id NULL (tamu milik undangan saja) FK komposit ini dilewati (MATCH SIMPLE).
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_guests_project_side_idx ON planner_guests (project_id, guest_side);
CREATE INDEX planner_guests_project_rsvp_idx ON planner_guests (project_id, rsvp_status);
CREATE INDEX planner_guests_project_category_idx ON planner_guests (project_id, category);
CREATE INDEX planner_guests_project_name_idx ON planner_guests (project_id, lower(name));
CREATE INDEX planner_guests_invitation_idx ON planner_guests (invitation_id) WHERE invitation_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_guests;
