package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func newRefreshService(
	t *testing.T,
	trackerSvc *mocks.MockTrackerService,
	statsRepo *mocks.MockStatsRepository,
	trackerRepo *mocks.MockTrackerRepository,
	registry *mocks.MockScraperRegistry,
) domain.RefreshService {
	alertSvc := mocks.NewMockAlertService(t)
	alertSvc.EXPECT().Process(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	return service.NewRefreshService(trackerSvc, statsRepo, trackerRepo, registry, alertSvc)
}

var testTracker = &domain.Tracker{ID: 1, Name: "Alpha", ScraperKey: "unit3d"}

func TestRefreshService_RefreshTracker(t *testing.T) {
	t.Run("success: stores stats and clears scrape status", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)
		scraper := mocks.NewMockTrackerScraper(t)

		trackerSvc.EXPECT().GetByID(uint(1)).Return(testTracker, nil)
		registry.EXPECT().Get("unit3d").Return(scraper, true)
		scraper.EXPECT().Fetch(mock.Anything, *testTracker).Return(
			&domain.TrackerStats{Uploaded: 200, Downloaded: 100, Ratio: 1.999},
			nil,
		)
		statsRepo.EXPECT().Create(mock.MatchedBy(func(s *domain.TrackerStats) bool {
			return s.TrackerID == 1 && s.Ratio == 2.0 && !s.FetchedAt.IsZero()
		})).Return(nil)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), "").Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshTracker(context.Background(), 1)

		require.NoError(t, err)
	})

	t.Run("scraper not found: records error and returns", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)

		trackerSvc.EXPECT().GetByID(uint(1)).Return(testTracker, nil)
		registry.EXPECT().Get("unit3d").Return(nil, false)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), mock.MatchedBy(func(s string) bool {
			return s != ""
		})).Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshTracker(context.Background(), 1)

		assert.ErrorContains(t, err, "no scraper registered")
	})

	t.Run("Fetch fails: records error and returns", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)
		scraper := mocks.NewMockTrackerScraper(t)

		fetchErr := errors.New("network timeout")
		trackerSvc.EXPECT().GetByID(uint(1)).Return(testTracker, nil)
		registry.EXPECT().Get("unit3d").Return(scraper, true)
		scraper.EXPECT().Fetch(mock.Anything, *testTracker).Return(nil, fetchErr)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), fetchErr.Error()).Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshTracker(context.Background(), 1)

		assert.ErrorContains(t, err, "network timeout")
	})

	t.Run("StatsRepository.Create fails: records error and returns", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)
		scraper := mocks.NewMockTrackerScraper(t)

		createErr := errors.New("insert failed")
		trackerSvc.EXPECT().GetByID(uint(1)).Return(testTracker, nil)
		registry.EXPECT().Get("unit3d").Return(scraper, true)
		scraper.EXPECT().Fetch(mock.Anything, *testTracker).Return(
			&domain.TrackerStats{Ratio: 1.5}, nil,
		)
		statsRepo.EXPECT().Create(mock.Anything).Return(createErr)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), createErr.Error()).Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshTracker(context.Background(), 1)

		assert.Error(t, err)
	})

	t.Run("GetByID fails: returns error", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)

		trackerSvc.EXPECT().GetByID(uint(1)).Return(nil, errors.New("db error"))

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshTracker(context.Background(), 1)

		assert.Error(t, err)
	})
}

func TestRefreshService_RefreshAll(t *testing.T) {
	t.Run("refreshes all active trackers", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)
		scraper := mocks.NewMockTrackerScraper(t)

		tr1 := domain.Tracker{ID: 1, Name: "A", ScraperKey: "unit3d"}
		tr2 := domain.Tracker{ID: 2, Name: "B", ScraperKey: "unit3d"}

		trackerSvc.EXPECT().GetActive().Return([]domain.Tracker{tr1, tr2}, nil)
		trackerSvc.EXPECT().GetByID(uint(1)).Return(&tr1, nil)
		trackerSvc.EXPECT().GetByID(uint(2)).Return(&tr2, nil)
		registry.EXPECT().Get("unit3d").Return(scraper, true).Times(2)
		scraper.EXPECT().Fetch(mock.Anything, tr1).Return(&domain.TrackerStats{Ratio: 1.0}, nil)
		scraper.EXPECT().Fetch(mock.Anything, tr2).Return(&domain.TrackerStats{Ratio: 2.0}, nil)
		statsRepo.EXPECT().Create(mock.Anything).Return(nil).Times(2)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), "").Return(nil)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(2), "").Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshAll(context.Background())

		require.NoError(t, err)
	})

	t.Run("accumulates errors from failed trackers", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)

		tr1 := domain.Tracker{ID: 1, Name: "A", ScraperKey: "unit3d"}

		trackerSvc.EXPECT().GetActive().Return([]domain.Tracker{tr1}, nil)
		trackerSvc.EXPECT().GetByID(uint(1)).Return(&tr1, nil)
		registry.EXPECT().Get("unit3d").Return(nil, false)
		trackerRepo.EXPECT().UpdateScrapeStatus(uint(1), mock.Anything).Return(nil)

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshAll(context.Background())

		assert.Error(t, err)
	})

	t.Run("propagates GetActive error", func(t *testing.T) {
		trackerSvc := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		registry := mocks.NewMockScraperRegistry(t)

		trackerSvc.EXPECT().GetActive().Return(nil, errors.New("db error"))

		err := newRefreshService(t, trackerSvc, statsRepo, trackerRepo, registry).
			RefreshAll(context.Background())

		assert.Error(t, err)
	})
}
