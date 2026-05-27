package domain

import (
	"context"
	"time"
)

const (
	// AlertTypeRatioAlert fires when a tracker's ratio drops below a threshold.
	AlertTypeRatioAlert = "ratio_alert"
	// AlertTypeSyncError fires when a scraper fails to fetch stats for a tracker.
	AlertTypeSyncError = "sync_error"
)

// AlertConfig is a named alert rule that fires notifications when a condition
// is met. It mirrors the Report resource in structure and wiring.
type AlertConfig struct {
	ID              uint             `json:"id"              gorm:"primaryKey"`
	Name            string           `json:"name"`
	AlertType       string           `json:"alert_type"`
	Enabled         bool             `json:"enabled"`
	RatioThreshold  float64          `json:"ratio_threshold"`
	AllTrackers     bool             `json:"all_trackers"`
	Trackers        []Tracker        `json:"trackers"        gorm:"many2many:alert_config_trackers"`
	NotifierConfigs []NotifierConfig `json:"notifier_configs" gorm:"many2many:alert_config_notifier_configs"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CreateAlertConfigInput holds the fields required to create a new alert config.
type CreateAlertConfigInput struct {
	Name              string  `json:"name"               required:"true" minLength:"1" doc:"Human-readable alert name"`
	AlertType         string  `json:"alert_type"         required:"true"              doc:"Alert type: ratio_alert or sync_error"`
	Enabled           bool    `json:"enabled"                                         doc:"Whether the alert is active"`
	RatioThreshold    float64 `json:"ratio_threshold"                                 doc:"Ratio threshold for ratio_alert (default 1.5)"`
	AllTrackers       bool    `json:"all_trackers"                                    doc:"If true, applies to all trackers"`
	TrackerIDs        []uint  `json:"tracker_ids"                                     doc:"Specific tracker IDs (used when all_trackers is false)"`
	NotifierConfigIDs []uint  `json:"notifier_config_ids"                             doc:"IDs of notifier configs to deliver the alert to"`
}

// UpdateAlertConfigInput holds the optional fields for patching an alert config.
type UpdateAlertConfigInput struct {
	Name              *string  `json:"name"`
	Enabled           *bool    `json:"enabled"`
	RatioThreshold    *float64 `json:"ratio_threshold"`
	AllTrackers       *bool    `json:"all_trackers"`
	TrackerIDs        *[]uint  `json:"tracker_ids"`
	NotifierConfigIDs *[]uint  `json:"notifier_config_ids"`
}

// AlertConfigRepository is the persistence abstraction for AlertConfig.
type AlertConfigRepository interface {
	FindAll() ([]AlertConfig, error)
	FindByID(id uint) (*AlertConfig, error)
	FindAllEnabled() ([]AlertConfig, error)
	Create(config *AlertConfig) error
	Update(config *AlertConfig) error
	UpdateNotifierConfigs(alertConfigID uint, configIDs []uint) error
	UpdateTrackers(alertConfigID uint, trackerIDs []uint) error
	Delete(id uint) error
	// GetSentState returns whether an alert has already been sent for the
	// given (alertConfig, tracker) pair. Returns false, nil if no row exists.
	GetSentState(alertConfigID, trackerID uint) (bool, error)
	// SetSentState upserts the sent flag for a (alertConfig, tracker) pair.
	SetSentState(alertConfigID, trackerID uint, sent bool) error
}

// AlertConfigService is the business-logic abstraction for alert configs.
type AlertConfigService interface {
	GetAll() ([]AlertConfig, error)
	GetByID(id uint) (*AlertConfig, error)
	Create(input CreateAlertConfigInput) (*AlertConfig, error)
	Update(id uint, input UpdateAlertConfigInput) (*AlertConfig, error)
	Delete(id uint) error
}

// AlertService evaluates all enabled alert configs after each tracker refresh
// and fires notifications when conditions are met, with per-(config, tracker)
// deduplication to avoid repeated alerts for the same incident.
type AlertService interface {
	// Process checks all enabled alert configs against the result of a tracker
	// refresh. fetchErr is nil on success; stats is nil when the fetch failed.
	Process(ctx context.Context, tracker *Tracker, fetchErr error, stats *TrackerStats) error
}
