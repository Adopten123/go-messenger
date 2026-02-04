-- +goose Up
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(255);

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_url;