package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jose/ratiodash/internal/domain"
)

type notifierConfigService struct {
	repo    domain.NotifierConfigRepository
	builder domain.NotifierBuilder
}

func NewNotifierConfigService(repo domain.NotifierConfigRepository, builder domain.NotifierBuilder) domain.NotifierConfigService {
	return &notifierConfigService{repo: repo, builder: builder}
}

func (s *notifierConfigService) GetAll() ([]domain.NotifierConfig, error) {
	cfgs, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for i := range cfgs {
		cfgs[i].PublicConfig = domain.RedactCredentials(cfgs[i].Config)
	}
	return cfgs, nil
}

func (s *notifierConfigService) GetByID(id uint) (*domain.NotifierConfig, error) {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("notifier config %d not found", id)
	}
	cfg.PublicConfig = domain.RedactCredentials(cfg.Config)
	return cfg, nil
}

func (s *notifierConfigService) GetEnabled() ([]domain.NotifierConfig, error) {
	cfgs, err := s.repo.FindEnabled()
	if err != nil {
		return nil, err
	}
	for i := range cfgs {
		cfgs[i].PublicConfig = domain.RedactCredentials(cfgs[i].Config)
	}
	return cfgs, nil
}

func (s *notifierConfigService) Create(input domain.CreateNotifierConfigInput) (*domain.NotifierConfig, error) {
	if !isKnownNotifierType(input.Type) {
		return nil, fmt.Errorf("unknown notifier type %q", input.Type)
	}

	config := input.Config
	if config == "" {
		config = "{}"
	}

	n, err := s.builder.Build(input.Type, config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if err := n.Notify(context.Background(), testNotification()); err != nil {
		return nil, fmt.Errorf("notification test failed: %w", err)
	}

	cfg := &domain.NotifierConfig{
		Name:    input.Name,
		Type:    input.Type,
		Config:  config,
		Enabled: true,
	}
	if err := s.repo.Create(cfg); err != nil {
		return nil, fmt.Errorf("creating notifier config: %w", err)
	}
	cfg.PublicConfig = domain.RedactCredentials(cfg.Config)
	return cfg, nil
}

func (s *notifierConfigService) Update(id uint, input domain.UpdateNotifierConfigInput) (*domain.NotifierConfig, error) {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("notifier config %d not found", id)
	}

	if input.Name != nil {
		cfg.Name = *input.Name
	}
	if input.Config != nil {
		merged, err := mergeConfig(cfg.Config, *input.Config)
		if err != nil {
			return nil, err
		}
		cfg.Config = merged
		// Config changed: validate structure AND connectivity on the final merged config.
		n, err := s.builder.Build(cfg.Type, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
		if err := n.Notify(context.Background(), testNotification()); err != nil {
			return nil, fmt.Errorf("notification test failed: %w", err)
		}
	} else {
		// Config unchanged: structural check only (no notification spam).
		if _, err := s.builder.Build(cfg.Type, cfg.Config); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
	}
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}

	if err := s.repo.Update(cfg); err != nil {
		return nil, fmt.Errorf("updating notifier config: %w", err)
	}
	cfg.PublicConfig = domain.RedactCredentials(cfg.Config)
	return cfg, nil
}

func (s *notifierConfigService) Delete(id uint) error {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("notifier config %d not found", id)
	}
	return s.repo.Delete(id)
}

func (s *notifierConfigService) Test(cfgType, cfgJSON string) error {
	if cfgJSON == "" {
		cfgJSON = "{}"
	}
	n, err := s.builder.Build(cfgType, cfgJSON)
	if err != nil {
		return fmt.Errorf("build notifier: %w", err)
	}
	return n.Notify(context.Background(), testNotification())
}

func (s *notifierConfigService) TestByID(id uint, configOverride string) error {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("load notifier config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("notifier config %d not found", id)
	}
	effective := cfg.Config
	if configOverride != "" && configOverride != "{}" {
		effective, err = mergeConfig(cfg.Config, configOverride)
		if err != nil {
			return fmt.Errorf("merge config override: %w", err)
		}
	}
	n, err := s.builder.Build(cfg.Type, effective)
	if err != nil {
		return fmt.Errorf("build notifier: %w", err)
	}
	return n.Notify(context.Background(), testNotification())
}

func testNotification() domain.Notification {
	return domain.Notification{
		Event: domain.EventReport,
		Level: domain.LevelInfo,
		Title: "[RatioDash] test",
		Body:  "This is a test notification from RatioDash. Your notifier is working correctly.",
		Tags:  []string{"test"},
	}
}

// isKnownNotifierType returns true when key matches a registered notifier type.
func isKnownNotifierType(key string) bool {
	for _, t := range domain.AvailableNotifierTypes {
		if t.Key == key {
			return true
		}
	}
	return false
}

// mergeConfig overlays incoming onto existing, preserving keys absent in
// incoming so that clients can omit sensitive fields (e.g. token) on update.
func mergeConfig(existing, incoming string) (string, error) {
	base := map[string]string{}
	if existing != "" && existing != "{}" {
		if err := json.Unmarshal([]byte(existing), &base); err != nil {
			return "{}", fmt.Errorf("parsing existing config: %w", err)
		}
	}
	overlay := map[string]string{}
	if incoming != "" && incoming != "{}" {
		if err := json.Unmarshal([]byte(incoming), &overlay); err != nil {
			return "{}", fmt.Errorf("parsing incoming config: %w", err)
		}
	}
	for k, v := range overlay {
		base[k] = v
	}
	out, err := json.Marshal(base)
	if err != nil {
		return "{}", err
	}
	return string(out), nil
}
