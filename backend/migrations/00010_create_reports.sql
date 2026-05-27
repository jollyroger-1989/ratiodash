-- +goose Up

CREATE TABLE reports (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    cron_expr  TEXT     NOT NULL DEFAULT '@daily',
    last_sent_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE report_notifier_configs (
    report_id          INTEGER NOT NULL REFERENCES reports(id)          ON DELETE CASCADE,
    notifier_config_id INTEGER NOT NULL REFERENCES notifier_configs(id) ON DELETE CASCADE,
    PRIMARY KEY (report_id, notifier_config_id)
);

-- +goose Down

DROP TABLE report_notifier_configs;
DROP TABLE reports;
