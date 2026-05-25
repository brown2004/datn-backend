-- +goose Up
ALTER TABLE pc_agents
ADD COLUMN IF NOT EXISTS agent_secret_hash TEXT;

ALTER TABLE pc_agents
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
SELECT 1;
