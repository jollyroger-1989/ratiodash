-- +goose Up
CREATE TABLE IF NOT EXISTS site_stats (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    site_id     INTEGER  NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    uploaded    INTEGER  NOT NULL DEFAULT 0,
    downloaded  INTEGER  NOT NULL DEFAULT 0,
    ratio       REAL     NOT NULL DEFAULT 0,
    fetched_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_site_stats_site_id ON site_stats(site_id);

-- +goose Down
DROP INDEX IF EXISTS idx_site_stats_site_id;
DROP TABLE IF EXISTS site_stats;
