-- +goose Up
ALTER TABLE users
ALTER COLUMN phone_number SET NOT NULL;

CREATE TABLE IF NOT EXISTS auth_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    purpose VARCHAR(30) NOT NULL CHECK (purpose IN ('register')),
    otp_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_otps_phone_purpose_created_at
    ON auth_otps(phone_number, purpose, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_otps_expires_at
    ON auth_otps(expires_at);

-- +goose Down
DROP TABLE IF EXISTS auth_otps;

ALTER TABLE users
ALTER COLUMN phone_number DROP NOT NULL;
