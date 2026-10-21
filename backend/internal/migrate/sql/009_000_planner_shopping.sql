-- +goose Up
CREATE TABLE planner_shopping_items (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL,
    item_name TEXT NOT NULL CHECK (char_length(item_name) BETWEEN 1 AND 255),
    quantity NUMERIC(12, 2) NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit TEXT,                                          -- pcs, set, box, ... (opsional)
    estimated_price BIGINT NOT NULL DEFAULT 0 CHECK (estimated_price >= 0),  -- HARGA PER SATUAN
    actual_price BIGINT NOT NULL DEFAULT 0 CHECK (actual_price >= 0),        -- HARGA PER SATUAN, 0 = belum dibeli
    estimated_total BIGINT GENERATED ALWAYS AS (round(quantity * estimated_price)::bigint) STORED,
    actual_total BIGINT GENERATED ALWAYS AS (round(quantity * actual_price)::bigint) STORED,
    -- Positif = hemat, negatif = lebih mahal; NULL bila belum ada harga riil.
    price_delta BIGINT GENERATED ALWAYS AS (
        CASE WHEN actual_price > 0 THEN round(quantity * estimated_price)::bigint - round(quantity * actual_price)::bigint END
    ) STORED,
    purchase_location TEXT,                             -- teks bebas: URL marketplace atau nama toko/alamat/telepon
    vendor_id UUID,
    budget_item_id UUID,                                -- pos anggaran tempat total belanja ini digulung (rollup)
    purpose TEXT CHECK (purpose IN ('souvenir')),       -- 'souvenir' = pembanding stok vs serah terima di Menu 05
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'proses', 'selesai')),
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id),
    FOREIGN KEY (budget_item_id, project_id)
        REFERENCES planner_budget_items (id, project_id) ON DELETE SET NULL (budget_item_id)
);

CREATE INDEX planner_shopping_project_category_idx ON planner_shopping_items (project_id, category);
CREATE INDEX planner_shopping_project_status_idx ON planner_shopping_items (project_id, status);
CREATE INDEX planner_shopping_budget_item_idx ON planner_shopping_items (budget_item_id) WHERE budget_item_id IS NOT NULL;
CREATE UNIQUE INDEX planner_shopping_template_key_idx ON planner_shopping_items (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_shopping_items;
