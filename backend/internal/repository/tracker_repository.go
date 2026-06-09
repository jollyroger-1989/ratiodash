package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type trackerRepository struct {
	db *gorm.DB
}

func NewTrackerRepository(db *gorm.DB) domain.TrackerRepository {
	return &trackerRepository{db: db}
}

func (r *trackerRepository) FindAll(opts *domain.TrackerSortOptions) ([]domain.Tracker, error) {
	var trackers []domain.Tracker
	query := r.db

	if opts != nil && opts.SortBy != "" {
		allowedColumns := []string{"ratio", "uploaded", "downloaded"}
		col := ""
		for _, c := range allowedColumns {
			if c == opts.SortBy {
				col = c
				break
			}
		}
		if col != "" {
			dir := "ASC"
			if opts.SortOrder == "desc" {
				dir = "DESC"
			}
			query = query.
				Joins("LEFT JOIN tracker_stats ON tracker_stats.id = (SELECT MAX(id) FROM tracker_stats ts WHERE ts.tracker_id = trackers.id)").
				Order("COALESCE(tracker_stats." + col + ", 0) " + dir)
		}
	}

	if err := query.Find(&trackers).Error; err != nil {
		return nil, fmt.Errorf("finding all trackers: %w", err)
	}
	return trackers, nil
}

func (r *trackerRepository) FindByID(id uint) (*domain.Tracker, error) {
	var tracker domain.Tracker
	err := r.db.First(&tracker, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding tracker %d: %w", id, err)
	}
	return &tracker, nil
}

func (r *trackerRepository) FindActive() ([]domain.Tracker, error) {
	var trackers []domain.Tracker
	if err := r.db.Where("active = ?", true).Find(&trackers).Error; err != nil {
		return nil, fmt.Errorf("finding active trackers: %w", err)
	}
	return trackers, nil
}

func (r *trackerRepository) Create(tracker *domain.Tracker) error {
	if err := r.db.Create(tracker).Error; err != nil {
		return fmt.Errorf("creating tracker: %w", err)
	}
	return nil
}

func (r *trackerRepository) Update(tracker *domain.Tracker) error {
	if err := r.db.Save(tracker).Error; err != nil {
		return fmt.Errorf("updating tracker: %w", err)
	}
	return nil
}

func (r *trackerRepository) Delete(id uint) error {
	if err := r.db.Delete(&domain.Tracker{}, id).Error; err != nil {
		return fmt.Errorf("deleting tracker: %w", err)
	}
	return nil
}

func (r *trackerRepository) UpdateScrapeStatus(trackerID uint, lastError string) error {
	now := time.Now().UTC()
	if err := r.db.Model(&domain.Tracker{}).
		Where("id = ?", trackerID).
		Updates(map[string]interface{}{
			"last_error":      lastError,
			"last_scraped_at": now,
		}).Error; err != nil {
		return fmt.Errorf("updating scrape status for tracker %d: %w", trackerID, err)
	}
	return nil
}
