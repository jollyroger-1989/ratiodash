package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type alertConfigRepository struct {
	db *gorm.DB
}

func NewAlertConfigRepository(db *gorm.DB) domain.AlertConfigRepository {
	return &alertConfigRepository{db: db}
}

func (r *alertConfigRepository) preloaded() *gorm.DB {
	return r.db.Preload("NotifierConfigs").Preload("Trackers")
}

func (r *alertConfigRepository) FindAll() ([]domain.AlertConfig, error) {
	var configs []domain.AlertConfig
	if err := r.preloaded().Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("finding alert configs: %w", err)
	}
	if configs == nil {
		configs = []domain.AlertConfig{}
	}
	return configs, nil
}

func (r *alertConfigRepository) FindByID(id uint) (*domain.AlertConfig, error) {
	var config domain.AlertConfig
	err := r.preloaded().First(&config, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding alert config %d: %w", id, err)
	}
	return &config, nil
}

func (r *alertConfigRepository) FindAllEnabled() ([]domain.AlertConfig, error) {
	var configs []domain.AlertConfig
	if err := r.preloaded().Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("finding enabled alert configs: %w", err)
	}
	if configs == nil {
		configs = []domain.AlertConfig{}
	}
	return configs, nil
}

func (r *alertConfigRepository) Create(config *domain.AlertConfig) error {
	if err := r.db.Create(config).Error; err != nil {
		return fmt.Errorf("creating alert config: %w", err)
	}
	return nil
}

func (r *alertConfigRepository) Update(config *domain.AlertConfig) error {
	if err := r.db.Save(config).Error; err != nil {
		return fmt.Errorf("updating alert config %d: %w", config.ID, err)
	}
	return nil
}

// UpdateNotifierConfigs replaces the notifier config associations for an alert config.
func (r *alertConfigRepository) UpdateNotifierConfigs(alertConfigID uint, configIDs []uint) error {
	config := &domain.AlertConfig{ID: alertConfigID}

	var configs []domain.NotifierConfig
	for _, id := range configIDs {
		configs = append(configs, domain.NotifierConfig{ID: id})
	}

	if err := r.db.Model(config).Association("NotifierConfigs").Replace(configs); err != nil {
		return fmt.Errorf("updating notifier configs for alert config %d: %w", alertConfigID, err)
	}
	return nil
}

// UpdateTrackers replaces the tracker associations for an alert config.
func (r *alertConfigRepository) UpdateTrackers(alertConfigID uint, trackerIDs []uint) error {
	config := &domain.AlertConfig{ID: alertConfigID}

	var trackers []domain.Tracker
	for _, id := range trackerIDs {
		trackers = append(trackers, domain.Tracker{ID: id})
	}

	if err := r.db.Model(config).Association("Trackers").Replace(trackers); err != nil {
		return fmt.Errorf("updating trackers for alert config %d: %w", alertConfigID, err)
	}
	return nil
}

func (r *alertConfigRepository) Delete(id uint) error {
	res := r.db.Delete(&domain.AlertConfig{}, id)
	if res.Error != nil {
		return fmt.Errorf("deleting alert config %d: %w", id, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("alert config %d not found", id)
	}
	return nil
}

// GetSentState returns whether an alert has already been sent for the given
// (alertConfig, tracker) pair. Returns false, nil when no row exists yet.
func (r *alertConfigRepository) GetSentState(alertConfigID, trackerID uint) (bool, error) {
	var sent bool
	err := r.db.Raw(
		"SELECT sent FROM alert_sent_states WHERE alert_config_id = ? AND tracker_id = ?",
		alertConfigID, trackerID,
	).Scan(&sent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("getting sent state for alert config %d tracker %d: %w", alertConfigID, trackerID, err)
	}
	return sent, nil
}

// SetSentState upserts the sent flag for a (alertConfig, tracker) pair.
func (r *alertConfigRepository) SetSentState(alertConfigID, trackerID uint, sent bool) error {
	sentVal := 0
	if sent {
		sentVal = 1
	}
	err := r.db.Exec(
		`INSERT INTO alert_sent_states (alert_config_id, tracker_id, sent)
		 VALUES (?, ?, ?)
		 ON CONFLICT(alert_config_id, tracker_id) DO UPDATE SET sent = excluded.sent`,
		alertConfigID, trackerID, sentVal,
	).Error
	if err != nil {
		return fmt.Errorf("setting sent state for alert config %d tracker %d: %w", alertConfigID, trackerID, err)
	}
	return nil
}
