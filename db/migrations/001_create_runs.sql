CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS runs (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    status        VARCHAR(20)  NOT NULL DEFAULT 'QUEUED',
    marketplace   VARCHAR(50)  NOT NULL,
    input_json    JSONB        NOT NULL,
    result_json   JSONB,
    error_message TEXT,
    item_count    INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_runs_status      ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_created_at  ON runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_marketplace ON runs(marketplace);
