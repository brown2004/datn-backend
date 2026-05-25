-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_pairing_sessions_requested_pc_agent_id_unique
    ON pairing_sessions(requested_pc_agent_id)
    WHERE requested_pc_agent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_pairing_sessions_requested_pc_agent_id_unique;
