-- +goose Up
ALTER TABLE pc_agents
DROP COLUMN IF EXISTS device_code;

DELETE FROM pc_agents
WHERE user_id IS NULL;

ALTER TABLE pc_agents
ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE pc_agents
ADD COLUMN IF NOT EXISTS agent_secret_hash TEXT;

ALTER TABLE pc_agents
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS pairing_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_code VARCHAR(20) NOT NULL UNIQUE,
    device_name VARCHAR(255) NOT NULL,
    os_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_by_user_id UUID,
    pc_agent_id UUID,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_pairing_sessions_confirmed_user
        FOREIGN KEY (confirmed_by_user_id)
        REFERENCES users(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_pairing_sessions_pc_agent
        FOREIGN KEY (pc_agent_id)
        REFERENCES pc_agents(id)
        ON DELETE SET NULL,

    CONSTRAINT chk_pairing_sessions_status
        CHECK (status IN ('pending', 'confirmed', 'expired', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_pc_agents_user_id
    ON pc_agents(user_id);

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_device_code
    ON pairing_sessions(device_code);

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_status
    ON pairing_sessions(status);

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_expires_at
    ON pairing_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_confirmed_by_user_id
    ON pairing_sessions(confirmed_by_user_id);

CREATE INDEX IF NOT EXISTS idx_pairing_sessions_pc_agent_id
    ON pairing_sessions(pc_agent_id);

-- +goose Down
DROP TABLE IF EXISTS pairing_sessions;
