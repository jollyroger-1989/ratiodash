package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jose/ratiodash/internal/domain"
)

type refreshService struct {
	trackers    domain.TrackerService
	statsRepo   domain.StatsRepository
	trackerRepo domain.TrackerRepository
	registry    domain.ScraperRegistry
	alertSvc    domain.AlertService
}

func NewRefreshService(
	trackers domain.TrackerService,
	statsRepo domain.StatsRepository,
	trackerRepo domain.TrackerRepository,
	registry domain.ScraperRegistry,
	alertSvc domain.AlertService,
) domain.RefreshService {
	return &refreshService{
		trackers:    trackers,
		statsRepo:   statsRepo,
		trackerRepo: trackerRepo,
		registry:    registry,
		alertSvc:    alertSvc,
	}
}

func (s *refreshService) RefreshTracker(ctx context.Context, trackerID uint) error {
	tracker, err := s.trackers.GetByID(trackerID)
	if err != nil {
		return err
	}

	sc, ok := s.registry.Get(tracker.ScraperKey)
	if !ok {
		scrapeErr := fmt.Errorf("no scraper registered for key %q", tracker.ScraperKey)
		_ = s.trackerRepo.UpdateScrapeStatus(trackerID, scrapeErr.Error())
		_ = s.alertSvc.Process(ctx, tracker, scrapeErr, nil)
		return scrapeErr
	}

	stats, fetchErr := sc.Fetch(ctx, *tracker)
	if fetchErr != nil {
		_ = s.trackerRepo.UpdateScrapeStatus(trackerID, fetchErr.Error())
		_ = s.alertSvc.Process(ctx, tracker, fetchErr, nil)
		return fmt.Errorf("scraping tracker %q: %w", tracker.Name, fetchErr)
	}

	stats.TrackerID = trackerID
	stats.FetchedAt = time.Now().UTC()
	stats.Ratio = math.Round(stats.Ratio*100) / 100

	if err := s.statsRepo.Create(stats); err != nil {
		_ = s.trackerRepo.UpdateScrapeStatus(trackerID, err.Error())
		return err
	}

	_ = s.alertSvc.Process(ctx, tracker, nil, stats)
	return s.trackerRepo.UpdateScrapeStatus(trackerID, "")
}

func (s *refreshService) RefreshAll(ctx context.Context) error {
	trackers, err := s.trackers.GetActive()
	if err != nil {
		return err
	}

	var errs []error
	for _, tracker := range trackers {
		if err := s.RefreshTracker(ctx, tracker.ID); err != nil {
			errs = append(errs, fmt.Errorf("tracker %q: %w", tracker.Name, err))
		}
	}
	return errors.Join(errs...)
}
