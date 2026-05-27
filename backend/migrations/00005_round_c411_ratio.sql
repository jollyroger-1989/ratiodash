-- +goose Up
UPDATE tracker_stats
SET ratio = ROUND(ratio, 2)
WHERE tracker_id IN (
    SELECT id FROM trackers WHERE scraper_key = 'c411'
);

-- +goose Down
-- Rounding is lossy; no-op rollback.
