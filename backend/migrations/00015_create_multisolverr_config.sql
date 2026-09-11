-- +goose Up
CREATE TABLE IF NOT EXISTS multisolverr_configs (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    enabled         BOOLEAN  NOT NULL DEFAULT 0,
    base_url        TEXT     NOT NULL DEFAULT '',
    api_key         TEXT     NOT NULL DEFAULT '',
    timeout_seconds INTEGER  NOT NULL DEFAULT 60,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE trackers ADD COLUMN use_multisolverr BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; recreate is omitted.
DROP TABLE IF EXISTS multisolverr_configs;
