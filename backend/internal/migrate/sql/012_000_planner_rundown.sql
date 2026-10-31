-- +goose Up
CREATE TABLE planner_rundowns (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME,                                      -- NULL = "selesai" (terbuka), mis. "10.40 - selesai"
    ends_next_day BOOLEAN NOT NULL DEFAULT FALSE,
    duration_minutes INT GENERATED ALWAYS AS (
        CASE WHEN end_time IS NULL THEN NULL
             ELSE (EXTRACT(EPOCH FROM (end_time - start_time)) / 60)::int + CASE WHEN ends_next_day THEN 1440 ELSE 0 END
        END
    ) STORED,
    activity_name TEXT NOT NULL CHECK (char_length(activity_name) BETWEEN 1 AND 255),
    pic_name TEXT NOT NULL DEFAULT '',
    vendor_id UUID,
    description TEXT NOT NULL DEFAULT '',
    highlights TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    sort_order INT NOT NULL DEFAULT 0,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, project_id),
    CHECK (end_time IS NULL OR ends_next_day OR end_time >= start_time),
    FOREIGN KEY (session_id, project_id)
        REFERENCES planner_event_sessions (id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (vendor_id, project_id)
        REFERENCES planner_vendors (id, project_id) ON DELETE SET NULL (vendor_id)
);

CREATE INDEX planner_rundowns_session_sort_idx ON planner_rundowns (session_id, sort_order);
CREATE INDEX planner_rundowns_project_status_idx ON planner_rundowns (project_id, status);
CREATE INDEX planner_rundowns_vendor_idx ON planner_rundowns (vendor_id) WHERE vendor_id IS NOT NULL;
CREATE UNIQUE INDEX planner_rundowns_template_key_idx ON planner_rundowns (project_id, template_key) WHERE template_key IS NOT NULL;

CREATE TABLE planner_playlists (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES planner_projects(id) ON DELETE CASCADE,
    rundown_id UUID,
    song_title TEXT NOT NULL CHECK (char_length(song_title) BETWEEN 1 AND 255),
    artist TEXT NOT NULL CHECK (char_length(artist) BETWEEN 1 AND 255),
    moment_tag TEXT NOT NULL DEFAULT 'Lainnya',
    audio_format TEXT NOT NULL DEFAULT 'digital_playback' CHECK (audio_format IN ('digital_playback', 'live_band', 'acoustic', 'sound_fx')),
    spotify_url TEXT,
    youtube_url TEXT,
    audio_cue_notes TEXT,
    status TEXT NOT NULL DEFAULT 'needs_review' CHECK (status IN ('ready', 'needs_review', 'missing_file')),
    sort_order INT NOT NULL DEFAULT 0,
    template_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (rundown_id, project_id)
        REFERENCES planner_rundowns (id, project_id) ON DELETE SET NULL (rundown_id)
);

CREATE INDEX planner_playlists_project_sort_idx ON planner_playlists (project_id, sort_order);
CREATE INDEX planner_playlists_rundown_idx ON planner_playlists (rundown_id) WHERE rundown_id IS NOT NULL;
CREATE UNIQUE INDEX planner_playlists_template_key_idx ON planner_playlists (project_id, template_key) WHERE template_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS planner_playlists;
DROP TABLE IF EXISTS planner_rundowns;
