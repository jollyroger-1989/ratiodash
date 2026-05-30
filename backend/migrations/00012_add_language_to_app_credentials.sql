-- +goose Up
ALTER TABLE app_credentials ADD COLUMN language TEXT NOT NULL DEFAULT 'en';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; no-op for rollback.
