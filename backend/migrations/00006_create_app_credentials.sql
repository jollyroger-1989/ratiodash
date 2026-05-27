-- +goose Up
CREATE TABLE IF NOT EXISTS app_credentials (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    username      TEXT     NOT NULL,
    password_hash TEXT     NOT NULL,
    jwt_secret    TEXT     NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS app_credentials;
