package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/proxy"
)

type multisolverrConfigService struct {
	repo domain.MultisolverrConfigRepository
}

func NewMultisolverrConfigService(repo domain.MultisolverrConfigRepository) domain.MultisolverrConfigService {
	return &multisolverrConfigService{repo: repo}
}

func (s *multisolverrConfigService) Get() (*domain.MultisolverrConfig, error) {
	return s.repo.Get()
}

func (s *multisolverrConfigService) Update(input domain.UpdateMultisolverrConfigInput) (*domain.MultisolverrConfig, error) {
	cfg, err := s.repo.Get()
	if err != nil {
		return nil, err
	}

	if input.BaseURL != nil {
		trimmed := strings.TrimRight(strings.TrimSpace(*input.BaseURL), "/")
		if trimmed != "" {
			if err := proxy.ValidateBaseURL(trimmed); err != nil {
				return nil, err
			}
		}
		cfg.BaseURL = trimmed
	}
	if input.TimeoutSeconds != nil {
		if *input.TimeoutSeconds <= 0 {
			return nil, fmt.Errorf("timeout_seconds must be positive")
		}
		cfg.TimeoutSeconds = *input.TimeoutSeconds
	}
	if input.Enabled != nil {
		if *input.Enabled && cfg.BaseURL == "" {
			return nil, fmt.Errorf("base_url is required to enable the multisolverr proxy")
		}
		cfg.Enabled = *input.Enabled
	}

	if err := s.repo.Update(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *multisolverrConfigService) Test(baseURL string) error {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return fmt.Errorf("base_url is required")
	}
	if err := proxy.ValidateBaseURL(trimmed); err != nil {
		return err
	}
	client := proxy.NewClient(proxy.Config{BaseURL: trimmed, Timeout: 15 * time.Second})
	return client.Ping(context.Background())
}
