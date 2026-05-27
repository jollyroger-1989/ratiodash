-- +goose Up
CREATE TABLE IF NOT EXISTS sites (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    name         TEXT     NOT NULL UNIQUE,
    url          TEXT     NOT NULL,
    scraper_key  TEXT     NOT NULL,
    credentials  TEXT     NOT NULL DEFAULT '{}',
    cron_expr    TEXT     NOT NULL DEFAULT '@hourly',
    active       INTEGER  NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS sites;
