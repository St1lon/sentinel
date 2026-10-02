CREATE TABLE users (
    id               UUID PRIMARY KEY,
    email            TEXT        NOT NULL,
    password_hash    TEXT        NOT NULL,
    status_page_slug TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_status_page_slug_unique UNIQUE (status_page_slug),
    CONSTRAINT users_email_not_empty CHECK (length(btrim(email)) > 0)
);

CREATE TYPE monitor_kind AS ENUM ('http', 'tls_cert', 'tcp_port');

CREATE TYPE monitor_status AS ENUM ('pending', 'up', 'down', 'paused');

CREATE TABLE monitors (
    id                   UUID PRIMARY KEY,
    user_id              UUID           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                 TEXT           NOT NULL,
    kind                 monitor_kind   NOT NULL DEFAULT 'http',
    target               TEXT           NOT NULL,
    method               TEXT           NOT NULL DEFAULT 'GET',
    interval_seconds     INTEGER        NOT NULL DEFAULT 60,
    timeout_seconds      INTEGER        NOT NULL DEFAULT 10,
    expected_status      INTEGER        NOT NULL DEFAULT 200,
    failure_threshold    INTEGER        NOT NULL DEFAULT 2,
    is_public            BOOLEAN        NOT NULL DEFAULT FALSE,
    paused               BOOLEAN        NOT NULL DEFAULT FALSE,
    status               monitor_status NOT NULL DEFAULT 'pending',
    consecutive_failures INTEGER        NOT NULL DEFAULT 0,
    last_checked_at      TIMESTAMPTZ,
    next_check_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT monitors_name_unique_per_user UNIQUE (user_id, name),
    CONSTRAINT monitors_name_not_empty CHECK (length(btrim(name)) > 0 AND length(name) <= 100),
    CONSTRAINT monitors_target_not_empty CHECK (length(btrim(target)) > 0 AND length(target) <= 2048),
    CONSTRAINT monitors_interval_range CHECK (interval_seconds BETWEEN 10 AND 86400),
    CONSTRAINT monitors_timeout_range CHECK (timeout_seconds BETWEEN 1 AND 120),
    CONSTRAINT monitors_timeout_below_interval CHECK (timeout_seconds < interval_seconds),
    CONSTRAINT monitors_threshold_range CHECK (failure_threshold BETWEEN 1 AND 10),
    CONSTRAINT monitors_expected_status_range CHECK (expected_status BETWEEN 100 AND 599),
    CONSTRAINT monitors_failures_non_negative CHECK (consecutive_failures >= 0)
);

CREATE INDEX monitors_due_idx ON monitors (next_check_at) WHERE paused = FALSE;

CREATE INDEX monitors_user_created_idx ON monitors (user_id, created_at DESC);

CREATE TABLE checks (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  UUID        NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    up          BOOLEAN     NOT NULL,
    status_code INTEGER,
    latency_ms  INTEGER     NOT NULL,
    error       TEXT,

    CONSTRAINT checks_latency_non_negative CHECK (latency_ms >= 0),
    CONSTRAINT checks_status_code_range CHECK (status_code IS NULL OR status_code BETWEEN 100 AND 599)
);

CREATE INDEX checks_monitor_time_idx ON checks (monitor_id, checked_at DESC);

CREATE TABLE incidents (
    id          UUID PRIMARY KEY,
    monitor_id  UUID        NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    cause       TEXT        NOT NULL,

    CONSTRAINT incidents_resolved_after_started CHECK (resolved_at IS NULL OR resolved_at >= started_at)
);

-- Не больше одного открытого инцидента на монитор.
CREATE UNIQUE INDEX incidents_single_open_per_monitor
    ON incidents (monitor_id)
    WHERE resolved_at IS NULL;

CREATE INDEX incidents_monitor_started_idx ON incidents (monitor_id, started_at DESC);
