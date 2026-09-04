-- +goose Up
ALTER TABLE trackers ADD COLUMN session_data TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; recreate is omitted.
