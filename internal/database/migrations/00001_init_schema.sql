-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone_number VARCHAR(20) UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pc_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,
    os_type VARCHAR(50) NOT NULL,
    agent_secret_hash TEXT,
    agent_status VARCHAR(20) NOT NULL DEFAULT 'offline'
        CHECK (agent_status IN ('online', 'offline')),
    protection_status VARCHAR(20) NOT NULL DEFAULT 'disabled'
        CHECK (protection_status IN ('enabled', 'disabled')),
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pc_agent_id UUID NOT NULL REFERENCES pc_agents(id) ON DELETE CASCADE,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    alert_type VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS mobile_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token TEXT NOT NULL UNIQUE,
    platform VARCHAR(50) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pc_agents_user_id ON pc_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_pc_agent_id ON alerts(pc_agent_id);
CREATE INDEX IF NOT EXISTS idx_alerts_triggered_at ON alerts(triggered_at);
CREATE INDEX IF NOT EXISTS idx_mobile_devices_user_id ON mobile_devices(user_id);

-- +goose Down
DROP TABLE IF EXISTS mobile_devices;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS pc_agents;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS pgcrypto;
