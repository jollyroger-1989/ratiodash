-- +goose Up
ALTER TABLE trackers ADD COLUMN last_error      TEXT     NOT NULL DEFAULT '';
ALTER TABLE trackers ADD COLUMN last_scraped_at DATETIME;

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; recreate is omitted.
