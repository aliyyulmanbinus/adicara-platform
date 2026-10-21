-- +goose Up
CREATE TABLE planner_budget_scenarios (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    total_target BIGINT NOT NULL CHECK (total_target > 0),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    active_sub_events TEXT[] NOT NULL DEFAULT '{}',     -- scope_key yang ikut disimulasikan; kosong = semua sesi non-sistem
    preset_key TEXT,                                    -- asal salinan, mis. wedding_75jt; preset itu sendiri ada di kode
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

-- Paling banyak satu skenario aktif per proyek. Batas 3 skenario ditegakkan usecase
-- di dalam transaksi yang mengunci baris proyek (SELECT ... FOR UPDATE).
CREATE UNIQUE INDEX planner_budget_scenarios_one_active_idx ON planner_budget_scenarios (project_id) WHERE is_active;

CREATE TABLE planner_budget_scenario_items (
    id UUID PRIMARY KEY,
    scenario_id UUID NOT NULL,
    project_id UUID NOT NULL,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    item_key TEXT,                                      -- kunci pencocok Smart Adjust; NULL = pos kustom
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 150),
    cost_amount BIGINT NOT NULL DEFAULT 0 CHECK (cost_amount >= 0),
    sort_order INT NOT NULL DEFAULT 0,
    FOREIGN KEY (scenario_id, project_id)
        REFERENCES planner_budget_scenarios (id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_budget_scenario_items_scenario_idx ON planner_budget_scenario_items (scenario_id, sub_event_scope);
CREATE UNIQUE INDEX planner_budget_scenario_items_key_idx ON planner_budget_scenario_items (scenario_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL;

CREATE TABLE planner_contributors (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    role_label TEXT NOT NULL DEFAULT 'other',           -- divalidasi terhadap registry event type
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

CREATE TABLE planner_savings_entries (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    contributor_id UUID NOT NULL,
    entry_date DATE NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (contributor_id, project_id)
        REFERENCES planner_contributors (id, project_id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX planner_savings_project_date_idx ON planner_savings_entries (project_id, entry_date, created_at);
CREATE INDEX planner_savings_contributor_idx ON planner_savings_entries (contributor_id);

CREATE TABLE planner_budget_items (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 150),
    item_key TEXT,                                      -- sama dengan planner_budget_scenario_items.item_key
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('template', 'scenario', 'vendor', 'admin_doc', 'manual')),
    vendor_id UUID,
    admin_doc_id UUID,
    rollup_source TEXT CHECK (rollup_source IN ('shopping')),  -- bila terisi, estimasi/terbayar dihitung dari daftar belanja
    pricing_mode TEXT NOT NULL DEFAULT 'lump' CHECK (pricing_mode IN ('lump', 'unit')),
    quantity NUMERIC(12, 2) CHECK (quantity > 0),
    unit_label TEXT,
    unit_price BIGINT CHECK (unit_price >= 0),
    initial_estimate BIGINT NOT NULL DEFAULT 0 CHECK (initial_estimate >= 0),

    dp_amount BIGINT NOT NULL DEFAULT 0 CHECK (dp_amount >= 0),
    dp_paid_at DATE,
    dp_receipt_url TEXT,
    term_1_amount BIGINT NOT NULL DEFAULT 0 CHECK (term_1_amount >= 0),
    term_1_paid_at DATE,
    term_1_receipt_url TEXT,
    term_2_amount BIGINT NOT NULL DEFAULT 0 CHECK (term_2_amount >= 0),
    term_2_paid_at DATE,
    term_2_receipt_url TEXT,

    paid_total BIGINT GENERATED ALWAYS AS (dp_amount + term_1_amount + term_2_amount) STORED,
    remaining_balance BIGINT GENERATED ALWAYS AS (initial_estimate - (dp_amount + term_1_amount + term_2_amount)) STORED,

    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    CHECK (pricing_mode = 'lump' OR (quantity IS NOT NULL AND unit_price IS NOT NULL)),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id),
    FOREIGN KEY (admin_doc_id, project_id)
        REFERENCES planner_admin_docs (id, project_id) ON DELETE SET NULL (admin_doc_id)
);

CREATE INDEX planner_budget_items_project_scope_idx ON planner_budget_items (project_id, sub_event_scope);
CREATE INDEX planner_budget_items_project_sort_idx ON planner_budget_items (project_id, sort_order);
CREATE UNIQUE INDEX planner_budget_items_key_idx ON planner_budget_items (project_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL;
CREATE UNIQUE INDEX planner_budget_items_vendor_idx ON planner_budget_items (vendor_id) WHERE vendor_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_budget_items;
DROP TABLE IF EXISTS planner_savings_entries;
DROP TABLE IF EXISTS planner_contributors;
DROP TABLE IF EXISTS planner_budget_scenario_items;
DROP TABLE IF EXISTS planner_budget_scenarios;
