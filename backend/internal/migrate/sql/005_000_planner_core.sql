-- +goose Up
CREATE TABLE planner_projects (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'wedding', 'khitanan', 'aqiqah', 'peresmian', 'launching',
        'birthday', 'wisuda', 'seminar', 'other'
    )),
    title TEXT NOT NULL CHECK (
        char_length(title) BETWEEN 1 AND 255 AND title = btrim(title) AND title !~ '[[:cntrl:]]'
    ),
    event_date DATE NOT NULL,                           -- Hari-H (hari pertama bila acara beberapa hari)
    event_end_date DATE,                                -- NULL = acara satu hari
    prep_start_date DATE NOT NULL,                      -- diisi usecase (tanggal hari ini di zona waktu proyek)
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta' CHECK (timezone ~
        '^(UTC|(Africa|America|Antarctica|Arctic|Asia|Atlantic|Australia|Europe|Indian|Pacific)/[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)?)$'),
    target_budget BIGINT NOT NULL DEFAULT 0 CHECK (target_budget BETWEEN 0 AND 10000000000000),  -- rupiah utuh, maks 10^13
    savings_target BIGINT CHECK (savings_target BETWEEN 0 AND 10000000000000),                   -- NULL = ikut target_budget
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
    is_sample BOOLEAN NOT NULL DEFAULT FALSE,           -- proyek contoh dari dataset PDF
    skipped_vendor_categories TEXT[] NOT NULL DEFAULT '{}',  -- kategori vendor yang ditandai "tidak diperlukan"
    invitation_id UUID,                                 -- undangan digital terhubung (bundling); TANPA FK: tabel undangan belum ada
    client_token TEXT CHECK (char_length(client_token) BETWEEN 8 AND 64),  -- pengaman kirim-ganda dari wizard
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,        -- divalidasi usecase terhadap registry event type
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (prep_start_date <= event_date),
    CHECK (event_end_date IS NULL OR event_end_date >= event_date)
);

CREATE INDEX planner_projects_user_status_idx ON planner_projects (user_id, status);
CREATE INDEX planner_projects_user_date_idx ON planner_projects (user_id, event_date);
CREATE UNIQUE INDEX planner_projects_invitation_idx ON planner_projects (invitation_id) WHERE invitation_id IS NOT NULL;
CREATE UNIQUE INDEX planner_projects_sample_idx ON planner_projects (user_id) WHERE is_sample;
CREATE UNIQUE INDEX planner_projects_client_token_idx ON planner_projects (user_id, client_token) WHERE client_token IS NOT NULL;

-- Nama zona waktu harus dikenal PostgreSQL; satu nilai salah akan menggagalkan seluruh kueri Hub pengguna itu.
-- Bentuk kanonik (huruf besar-kecil, awalan wilayah) dijaga CHECK di atas; keberadaannya dijaga pemicu ini.
-- +goose StatementBegin
CREATE FUNCTION planner_check_timezone() RETURNS trigger AS $$
BEGIN
    PERFORM now() AT TIME ZONE NEW.timezone;
    RETURN NEW;
EXCEPTION WHEN invalid_parameter_value THEN
    RAISE EXCEPTION 'unknown time zone: %', NEW.timezone USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER planner_projects_timezone_trg
    BEFORE INSERT OR UPDATE OF timezone ON planner_projects
    FOR EACH ROW EXECUTE FUNCTION planner_check_timezone();

-- Master sesi/sub-acara per proyek. SATU-SATUNYA sumber nilai sub_event_scope
-- untuk modul 02-05; modul 06 memakai id-nya langsung.
CREATE TABLE planner_event_sessions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    scope_key TEXT NOT NULL CHECK (scope_key ~ '^[a-z0-9_]{1,40}$'),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100 AND name = btrim(name)),
    icon TEXT NOT NULL DEFAULT '📋' CHECK (char_length(icon) BETWEEN 1 AND 8),
    is_system BOOLEAN NOT NULL DEFAULT FALSE,           -- TRUE hanya untuk baris 'all'
    sort_order INT NOT NULL DEFAULT 0,
    session_date DATE,
    start_time TIME,
    end_time TIME,
    location_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, scope_key),
    UNIQUE (id, project_id),
    CHECK ((scope_key = 'all') = is_system)
);

-- Nama sesi unik per proyek: impor tamu dan ekspor memetakan sesi lewat nama.
CREATE UNIQUE INDEX planner_event_sessions_name_idx ON planner_event_sessions (project_id, lower(name));

-- +goose Down
DROP TABLE IF EXISTS planner_event_sessions;
DROP TABLE IF EXISTS planner_projects;
DROP FUNCTION IF EXISTS planner_check_timezone();
