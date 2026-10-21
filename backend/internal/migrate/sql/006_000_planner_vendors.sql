-- +goose Up
CREATE TABLE planner_vendors (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category_key TEXT NOT NULL CHECK (category_key ~ '^[a-z0-9_]{1,40}$'),  -- slug stabil: venue, catering, mua, ...
    category_label TEXT NOT NULL,                       -- teks tampilan; bebas untuk kategori kustom
    vendor_name TEXT NOT NULL CHECK (char_length(vendor_name) BETWEEN 1 AND 255),
    pic_name TEXT,
    phone_number TEXT,                                  -- input pengguna
    formatted_wa_number TEXT,                           -- 62xxxxxxxxxx, diisi usecase
    portfolio_url TEXT,
    budget_limit BIGINT NOT NULL DEFAULT 0 CHECK (budget_limit >= 0),   -- pagu dari skenario aktif atau manual
    agreed_price BIGINT NOT NULL DEFAULT 0 CHECK (agreed_price >= 0),   -- 0 = belum ada penawaran
    -- Positif = hemat, negatif = melebihi pagu; NULL bila pagu atau penawaran belum ada.
    price_delta BIGINT GENERATED ALWAYS AS (
        CASE WHEN agreed_price > 0 AND budget_limit > 0 THEN budget_limit - agreed_price END
    ) STORED,
    status TEXT NOT NULL DEFAULT 'rencana' CHECK (status IN ('rencana', 'negosiasi', 'fix', 'batal')),
    notes TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_vendors_project_status_idx ON planner_vendors (project_id, status);
CREATE INDEX planner_vendors_project_category_idx ON planner_vendors (project_id, category_key);

-- +goose Down
DROP TABLE IF EXISTS planner_vendors;
