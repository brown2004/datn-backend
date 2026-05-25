-- +goose Up
ALTER TABLE pairing_sessions
ADD COLUMN IF NOT EXISTS requested_pc_agent_id UUID;

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_requested_pc_agent_id
    ON pairing_sessions(requested_pc_agent_id);

-- +goose Down
DROP INDEX IF EXISTS idx_pairing_sessions_requested_pc_agent_id;

ALTER TABLE pairing_sessions
DROP COLUMN IF EXISTS requested_pc_agent_id;
