-- +goose Up
UPDATE tracker_stats
SET ratio = ROUND(ratio, 2);

-- +goose Down
-- Rounding is lossy; no-op rollback.
