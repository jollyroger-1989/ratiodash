-- +goose Up
-- Migrate unit3d tracker URLs into their credentials JSON before dropping the column.
UPDATE trackers
SET credentials = json_set(
    CASE WHEN credentials IS NULL OR credentials = '' THEN '{}' ELSE credentials END,
    '$.url',
    url
)
WHERE scraper_key = 'unit3d' AND url IS NOT NULL AND url != '';

-- Drop the url column (requires SQLite 3.35+).
ALTER TABLE trackers DROP COLUMN url;

-- +goose Down
ALTER TABLE trackers ADD COLUMN url TEXT NOT NULL DEFAULT '';

-- Restore unit3d tracker URLs from credentials.
UPDATE trackers
SET url = json_extract(credentials, '$.url')
WHERE scraper_key = 'unit3d' AND json_extract(credentials, '$.url') IS NOT NULL;
