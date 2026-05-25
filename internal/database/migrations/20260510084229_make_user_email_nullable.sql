-- +goose Up
ALTER TABLE users
ALTER COLUMN email DROP NOT NULL;

-- +goose Down
ALTER TABLE users
ALTER COLUMN email SET NOT NULL;
