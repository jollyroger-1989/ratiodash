package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) domain.ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) FindAll() ([]domain.Report, error) {
	var reports []domain.Report
	if err := r.db.Preload("NotifierConfigs").Find(&reports).Error; err != nil {
		return nil, fmt.Errorf("finding reports: %w", err)
	}
	return reports, nil
}

func (r *reportRepository) FindByID(id uint) (*domain.Report, error) {
	var report domain.Report
	err := r.db.Preload("NotifierConfigs").First(&report, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding report %d: %w", id, err)
	}
	return &report, nil
}

func (r *reportRepository) Create(report *domain.Report) error {
	if err := r.db.Create(report).Error; err != nil {
		return fmt.Errorf("creating report: %w", err)
	}
	return nil
}

func (r *reportRepository) Update(report *domain.Report) error {
	if err := r.db.Save(report).Error; err != nil {
		return fmt.Errorf("updating report %d: %w", report.ID, err)
	}
	return nil
}

// UpdateNotifierConfigs replaces the notifier config associations for a report.
func (r *reportRepository) UpdateNotifierConfigs(reportID uint, configIDs []uint) error {
	report := &domain.Report{ID: reportID}

	// Build the association targets from IDs.
	var configs []domain.NotifierConfig
	for _, id := range configIDs {
		configs = append(configs, domain.NotifierConfig{ID: id})
	}

	if err := r.db.Model(report).Association("NotifierConfigs").Replace(configs); err != nil {
		return fmt.Errorf("updating notifier configs for report %d: %w", reportID, err)
	}
	return nil
}

func (r *reportRepository) Delete(id uint) error {
	res := r.db.Delete(&domain.Report{}, id)
	if res.Error != nil {
		return fmt.Errorf("deleting report %d: %w", id, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("report %d not found", id)
	}
	return nil
}

func (r *reportRepository) UpdateLastSentAt(id uint, sentAt time.Time) error {
	res := r.db.Model(&domain.Report{}).Where("id = ?", id).Update("last_sent_at", sentAt)
	if res.Error != nil {
		return fmt.Errorf("updating last_sent_at for report %d: %w", id, res.Error)
	}
	return nil
}
