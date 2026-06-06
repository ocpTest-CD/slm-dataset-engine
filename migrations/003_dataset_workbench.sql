ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS progress INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS job_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    run_id UUID REFERENCES runs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    stage TEXT NOT NULL DEFAULT '',
    progress INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dataset_version_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_version_id UUID NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    byte_size BIGINT NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_job_events_job ON job_events(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_events_project ON job_events(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dataset_version_files_version ON dataset_version_files(dataset_version_id);
