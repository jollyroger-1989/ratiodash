package service

import (
	"fmt"

	"github.com/jose/ratiodash/internal/domain"
)

type alertConfigService struct {
	repo domain.AlertConfigRepository
}

func NewAlertConfigService(repo domain.AlertConfigRepository) domain.AlertConfigService {
	return &alertConfigService{repo: repo}
}

func (s *alertConfigService) GetAll() ([]domain.AlertConfig, error) {
	return s.repo.FindAll()
}

func (s *alertConfigService) GetByID(id uint) (*domain.AlertConfig, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("alert config %d not found", id)
	}
	return c, nil
}

func (s *alertConfigService) Create(input domain.CreateAlertConfigInput) (*domain.AlertConfig, error) {
	threshold := input.RatioThreshold
	if threshold <= 0 {
		threshold = 1.5
	}

	config := &domain.AlertConfig{
		Name:           input.Name,
		AlertType:      input.AlertType,
		Enabled:        input.Enabled,
		RatioThreshold: threshold,
		AllTrackers:    input.AllTrackers,
	}

	if err := s.repo.Create(config); err != nil {
		return nil, fmt.Errorf("creating alert config: %w", err)
	}

	if err := s.repo.UpdateNotifierConfigs(config.ID, input.NotifierConfigIDs); err != nil {
		return nil, fmt.Errorf("attaching notifiers: %w", err)
	}
	if !input.AllTrackers {
		if err := s.repo.UpdateTrackers(config.ID, input.TrackerIDs); err != nil {
			return nil, fmt.Errorf("attaching trackers: %w", err)
		}
	}

	return s.repo.FindByID(config.ID)
}

func (s *alertConfigService) Update(id uint, input domain.UpdateAlertConfigInput) (*domain.AlertConfig, error) {
	config, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("alert config %d not found", id)
	}

	if input.Name != nil {
		config.Name = *input.Name
	}
	if input.Enabled != nil {
		config.Enabled = *input.Enabled
	}
	if input.RatioThreshold != nil {
		config.RatioThreshold = *input.RatioThreshold
	}
	if input.AllTrackers != nil {
		config.AllTrackers = *input.AllTrackers
	}

	if err := s.repo.Update(config); err != nil {
		return nil, err
	}

	if input.NotifierConfigIDs != nil {
		if err := s.repo.UpdateNotifierConfigs(id, *input.NotifierConfigIDs); err != nil {
			return nil, err
		}
	}
	if input.TrackerIDs != nil {
		if err := s.repo.UpdateTrackers(id, *input.TrackerIDs); err != nil {
			return nil, err
		}
	}

	return s.repo.FindByID(id)
}

func (s *alertConfigService) Delete(id uint) error {
	return s.repo.Delete(id)
}
