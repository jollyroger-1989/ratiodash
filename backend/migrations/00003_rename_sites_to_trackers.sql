-- +goose Up
ALTER TABLE sites RENAME TO trackers;
ALTER TABLE site_stats RENAME TO tracker_stats;
ALTER TABLE tracker_stats RENAME COLUMN site_id TO tracker_id;
DROP INDEX IF EXISTS idx_site_stats_site_id;
CREATE INDEX IF NOT EXISTS idx_tracker_stats_tracker_id ON tracker_stats(tracker_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tracker_stats_tracker_id;
CREATE INDEX IF NOT EXISTS idx_site_stats_site_id ON site_stats(site_id);
ALTER TABLE tracker_stats RENAME COLUMN tracker_id TO site_id;
ALTER TABLE tracker_stats RENAME TO site_stats;
ALTER TABLE trackers RENAME TO sites;
