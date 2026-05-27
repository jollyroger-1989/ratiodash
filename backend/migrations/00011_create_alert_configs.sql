-- +goose Up

CREATE TABLE alert_configs (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL,
    alert_type      TEXT     NOT NULL,
    enabled         INTEGER  NOT NULL DEFAULT 1,
    ratio_threshold REAL     NOT NULL DEFAULT 1.5,
    all_trackers    INTEGER  NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_config_notifier_configs (
    alert_config_id    INTEGER NOT NULL REFERENCES alert_configs(id)    ON DELETE CASCADE,
    notifier_config_id INTEGER NOT NULL REFERENCES notifier_configs(id) ON DELETE CASCADE,
    PRIMARY KEY (alert_config_id, notifier_config_id)
);

CREATE TABLE alert_config_trackers (
    alert_config_id INTEGER NOT NULL REFERENCES alert_configs(id) ON DELETE CASCADE,
    tracker_id      INTEGER NOT NULL REFERENCES trackers(id)      ON DELETE CASCADE,
    PRIMARY KEY (alert_config_id, tracker_id)
);

-- Deduplication state: tracks whether an alert has already been sent for a
-- given (alert_config, tracker) pair, so we only fire once per incident.
CREATE TABLE alert_sent_states (
    alert_config_id INTEGER NOT NULL REFERENCES alert_configs(id) ON DELETE CASCADE,
    tracker_id      INTEGER NOT NULL REFERENCES trackers(id)      ON DELETE CASCADE,
    sent            INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (alert_config_id, tracker_id)
);

-- +goose Down

DROP TABLE IF EXISTS alert_sent_states;
DROP TABLE IF EXISTS alert_config_trackers;
DROP TABLE IF EXISTS alert_config_notifier_configs;
DROP TABLE IF EXISTS alert_configs;
