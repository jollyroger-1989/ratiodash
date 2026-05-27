package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jose/ratiodash/internal/domain"
)

type trackerService struct {
	repo     domain.TrackerRepository
	registry domain.ScraperRegistry
}

func NewTrackerService(repo domain.TrackerRepository, registry domain.ScraperRegistry) domain.TrackerService {
	return &trackerService{repo: repo, registry: registry}
}

func (s *trackerService) GetAll() ([]domain.Tracker, error) {
	trackers, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for i := range trackers {
		trackers[i].PublicCredentials = domain.RedactCredentials(trackers[i].Credentials)
	}
	return trackers, nil
}

func (s *trackerService) GetByID(id uint) (*domain.Tracker, error) {
	tracker, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if tracker == nil {
		return nil, fmt.Errorf("tracker %d not found", id)
	}
	tracker.PublicCredentials = domain.RedactCredentials(tracker.Credentials)
	return tracker, nil
}

func (s *trackerService) GetActive() ([]domain.Tracker, error) {
	trackers, err := s.repo.FindActive()
	if err != nil {
		return nil, err
	}
	for i := range trackers {
		trackers[i].PublicCredentials = domain.RedactCredentials(trackers[i].Credentials)
	}
	return trackers, nil
}

func (s *trackerService) Create(input domain.CreateTrackerInput) (*domain.Tracker, error) {
	cronExpr := input.CronExpr
	if cronExpr == "" {
		cronExpr = "@hourly"
	}
	creds := input.Credentials
	if creds == "" {
		creds = "{}"
	}

	tracker := &domain.Tracker{
		Name:        input.Name,
		ScraperKey:  input.ScraperKey,
		Credentials: creds,
		CronExpr:    cronExpr,
		Active:      true,
	}
	if err := s.repo.Create(tracker); err != nil {
		return nil, err
	}
	tracker.PublicCredentials = domain.RedactCredentials(tracker.Credentials)
	return tracker, nil
}

func (s *trackerService) Update(id uint, input domain.UpdateTrackerInput) (*domain.Tracker, error) {
	tracker, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if tracker == nil {
		return nil, fmt.Errorf("tracker %d not found", id)
	}

	if input.Name != nil {
		tracker.Name = *input.Name
	}
	if input.Credentials != nil {
		merged, err := mergeCredentials(tracker.Credentials, *input.Credentials)
		if err != nil {
			return nil, err
		}
		tracker.Credentials = merged
	}
	if input.CronExpr != nil {
		tracker.CronExpr = *input.CronExpr
	}
	if input.Active != nil {
		tracker.Active = *input.Active
	}

	if err := s.repo.Update(tracker); err != nil {
		return nil, err
	}
	tracker.PublicCredentials = domain.RedactCredentials(tracker.Credentials)
	return tracker, nil
}

// mergeCredentials overlays incoming onto existing, preserving any keys absent
// in incoming (e.g. passwords not re-submitted by the client).
func mergeCredentials(existing, incoming string) (string, error) {
	base := map[string]string{}
	if existing != "" && existing != "{}" {
		if err := json.Unmarshal([]byte(existing), &base); err != nil {
			return "{}", fmt.Errorf("parsing existing credentials: %w", err)
		}
	}
	overlay := map[string]string{}
	if incoming != "" && incoming != "{}" {
		if err := json.Unmarshal([]byte(incoming), &overlay); err != nil {
			return "{}", fmt.Errorf("parsing incoming credentials: %w", err)
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

func (s *trackerService) Delete(id uint) error {
	tracker, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if tracker == nil {
		return fmt.Errorf("tracker %d not found", id)
	}
	return s.repo.Delete(id)
}

func (s *trackerService) Test(scraperKey, credentialsJSON string) error {
	sc, ok := s.registry.Get(scraperKey)
	if !ok {
		return fmt.Errorf("unknown scraper key %q", scraperKey)
	}
	creds := credentialsJSON
	if creds == "" {
		creds = "{}"
	}
	_, err := sc.Fetch(context.Background(), domain.Tracker{
		ScraperKey:  scraperKey,
		Credentials: creds,
	})
	return err
}

func (s *trackerService) TestByID(id uint, credentialsOverride string) error {
	tracker, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if tracker == nil {
		return fmt.Errorf("tracker %d not found", id)
	}

	effective := tracker.Credentials
	if credentialsOverride != "" && credentialsOverride != "{}" {
		effective, err = mergeCredentials(tracker.Credentials, credentialsOverride)
		if err != nil {
			return err
		}
	}

	sc, ok := s.registry.Get(tracker.ScraperKey)
	if !ok {
		return fmt.Errorf("unknown scraper key %q", tracker.ScraperKey)
	}

	testTracker := *tracker
	testTracker.Credentials = effective
	_, err = sc.Fetch(context.Background(), testTracker)
	return err
}
