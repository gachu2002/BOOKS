CREATE TABLE protected_counters (
    resource_name TEXT PRIMARY KEY,
    value BIGINT NOT NULL DEFAULT 0 CHECK (value >= 0),
    last_fencing_token BIGINT NOT NULL DEFAULT 0,
    last_holder TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
