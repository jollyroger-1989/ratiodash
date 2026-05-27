-- +goose Up
CREATE TABLE IF NOT EXISTS notifier_configs (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    type       TEXT     NOT NULL,
    config     TEXT     NOT NULL DEFAULT '{}',
    enabled    INTEGER  NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS notifier_configs;
