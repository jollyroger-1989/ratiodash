package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type notifierConfigRepository struct {
	db *gorm.DB
}

func NewNotifierConfigRepository(db *gorm.DB) domain.NotifierConfigRepository {
	return &notifierConfigRepository{db: db}
}

func (r *notifierConfigRepository) FindAll() ([]domain.NotifierConfig, error) {
	var cfgs []domain.NotifierConfig
	if err := r.db.Find(&cfgs).Error; err != nil {
		return nil, fmt.Errorf("finding all notifier configs: %w", err)
	}
	return cfgs, nil
}

func (r *notifierConfigRepository) FindByID(id uint) (*domain.NotifierConfig, error) {
	var cfg domain.NotifierConfig
	err := r.db.First(&cfg, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding notifier config %d: %w", id, err)
	}
	return &cfg, nil
}

func (r *notifierConfigRepository) FindEnabled() ([]domain.NotifierConfig, error) {
	var cfgs []domain.NotifierConfig
	if err := r.db.Where("enabled = ?", true).Find(&cfgs).Error; err != nil {
		return nil, fmt.Errorf("finding enabled notifier configs: %w", err)
	}
	return cfgs, nil
}

func (r *notifierConfigRepository) Create(cfg *domain.NotifierConfig) error {
	if err := r.db.Create(cfg).Error; err != nil {
		return fmt.Errorf("creating notifier config: %w", err)
	}
	return nil
}

func (r *notifierConfigRepository) Update(cfg *domain.NotifierConfig) error {
	if err := r.db.Save(cfg).Error; err != nil {
		return fmt.Errorf("updating notifier config: %w", err)
	}
	return nil
}

func (r *notifierConfigRepository) Delete(id uint) error {
	if err := r.db.Delete(&domain.NotifierConfig{}, id).Error; err != nil {
		return fmt.Errorf("deleting notifier config: %w", err)
	}
	return nil
}
