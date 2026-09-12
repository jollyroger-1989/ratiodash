-- +goose Up
ALTER TABLE multisolverr_configs DROP COLUMN api_key;

-- +goose Down
ALTER TABLE multisolverr_configs ADD COLUMN api_key TEXT NOT NULL DEFAULT '';
