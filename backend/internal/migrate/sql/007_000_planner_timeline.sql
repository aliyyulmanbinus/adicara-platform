-- +goose Up
CREATE TABLE planner_tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
    sub_event_scope TEXT NOT NULL DEFAULT 'all',
    category TEXT NOT NULL DEFAULT 'Umum',
    phase_order INT NOT NULL DEFAULT 1,
    phase_name TEXT NOT NULL DEFAULT '',                -- nama fase ikut tersimpan; tidak ada tabel fase
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    -- Konvensi PDF Hal. 3: durasi = selesai - mulai (1 Feb -> 2 Feb = "1 Hari"), minimal 1.
    duration_days INT GENERATED ALWAYS AS (GREATEST(end_date - start_date, 1)) STORED,
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'proses', 'selesai')),
    pic TEXT NOT NULL DEFAULT '',
    vendor_id UUID,
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    template_key TEXT,                                  -- mis. wedding.pertemuan_keluarga_1; dasar mode "append"
    origin TEXT NOT NULL DEFAULT 'custom' CHECK (origin IN ('pdf', 'adicara', 'custom')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date),
    FOREIGN KEY (project_id, sub_event_scope)
        REFERENCES planner_event_sessions (project_id, scope_key) ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id)
);

CREATE INDEX planner_tasks_project_sort_idx ON planner_tasks (project_id, sort_order);
CREATE INDEX planner_tasks_project_dates_idx ON planner_tasks (project_id, start_date, end_date);
CREATE UNIQUE INDEX planner_tasks_template_key_idx ON planner_tasks (project_id, template_key) WHERE template_key IS NOT NULL;

CREATE TABLE planner_admin_docs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    document_name TEXT NOT NULL CHECK (char_length(document_name) BETWEEN 1 AND 255),
    subject_role TEXT NOT NULL DEFAULT 'bersama',       -- divalidasi usecase terhadap registry event type
    quantity_detail TEXT NOT NULL DEFAULT '',
    nominal_cost BIGINT NOT NULL DEFAULT 0 CHECK (nominal_cost >= 0),
    status TEXT NOT NULL DEFAULT 'belum' CHECK (status IN ('belum', 'selesai')),
    sort_order INT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    template_key TEXT,
    origin TEXT NOT NULL DEFAULT 'custom' CHECK (origin IN ('pdf', 'adicara', 'custom')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id)
);

CREATE INDEX planner_admin_docs_project_sort_idx ON planner_admin_docs (project_id, sort_order);
CREATE UNIQUE INDEX planner_admin_docs_template_key_idx ON planner_admin_docs (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_admin_docs;
DROP TABLE IF EXISTS planner_tasks;
