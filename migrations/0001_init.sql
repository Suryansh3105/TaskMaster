CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tasks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command      TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    picked_at    TIMESTAMPTZ,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at    TIMESTAMPTZ
);

CREATE INDEX idx_tasks_claimable
    ON tasks (scheduled_at)
    WHERE picked_at IS NULL;