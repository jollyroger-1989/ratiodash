package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type multisolverrConfigRepository struct {
	db *gorm.DB
}

func NewMultisolverrConfigRepository(db *gorm.DB) domain.MultisolverrConfigRepository {
	return &multisolverrConfigRepository{db: db}
}

// Get returns the singleton config row, creating a default (disabled) one on
// first access.
func (r *multisolverrConfigRepository) Get() (*domain.MultisolverrConfig, error) {
	var cfg domain.MultisolverrConfig
	err := r.db.First(&cfg).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("finding multisolverr config: %w", err)
	}

	cfg = domain.MultisolverrConfig{TimeoutSeconds: 60}
	if err := r.db.Create(&cfg).Error; err != nil {
		return nil, fmt.Errorf("creating default multisolverr config: %w", err)
	}
	return &cfg, nil
}

func (r *multisolverrConfigRepository) Update(cfg *domain.MultisolverrConfig) error {
	if err := r.db.Save(cfg).Error; err != nil {
		return fmt.Errorf("updating multisolverr config: %w", err)
	}
	return nil
}
