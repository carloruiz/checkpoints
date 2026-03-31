CREATE TABLE IF NOT EXISTS checkpoints (
    key        VARCHAR(256) PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
