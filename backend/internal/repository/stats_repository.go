package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type statsRepository struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) domain.StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) FindByTrackerID(trackerID uint, limit int) ([]domain.TrackerStats, error) {
	var stats []domain.TrackerStats
	q := r.db.Where("tracker_id = ?", trackerID).Order("fetched_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("finding stats for tracker %d: %w", trackerID, err)
	}
	return stats, nil
}

func (r *statsRepository) FindLatestByTrackerID(trackerID uint) (*domain.TrackerStats, error) {
	var s domain.TrackerStats
	err := r.db.Where("tracker_id = ?", trackerID).Order("fetched_at DESC").First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding latest stats for tracker %d: %w", trackerID, err)
	}
	return &s, nil
}

func (r *statsRepository) FindLatestAll() ([]domain.TrackerStats, error) {
	// Subquery: max(id) per tracker groups the monotonically increasing IDs to get
	// the most recent row without a self-join.
	sub := r.db.Model(&domain.TrackerStats{}).Select("MAX(id)").Group("tracker_id")

	var stats []domain.TrackerStats
	if err := r.db.Where("id IN (?)", sub).Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("finding latest stats for all trackers: %w", err)
	}
	return stats, nil
}

func (r *statsRepository) Create(s *domain.TrackerStats) error {
	if err := r.db.Create(s).Error; err != nil {
		return fmt.Errorf("creating stats: %w", err)
	}
	return nil
}

func (r *statsRepository) Delete(statID uint, trackerID uint) error {
	res := r.db.Where("id = ? AND tracker_id = ?", statID, trackerID).Delete(&domain.TrackerStats{})
	if res.Error != nil {
		return fmt.Errorf("deleting stat %d: %w", statID, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("stat %d not found for tracker %d", statID, trackerID)
	}
	return nil
}

func (r *statsRepository) FindNearestAtOrBefore(trackerID uint, t time.Time) (*domain.TrackerStats, error) {
	var s domain.TrackerStats
	err := r.db.
		Where("tracker_id = ? AND fetched_at <= ?", trackerID, t).
		Order("fetched_at DESC").
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding stats for tracker %d at or before %s: %w", trackerID, t, err)
	}
	return &s, nil
}
