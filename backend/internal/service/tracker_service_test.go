package service_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func ptr[T any](v T) *T { return &v }

func TestTrackerService_GetAll(t *testing.T) {
	t.Run("returns trackers with redacted credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindAll().Return([]domain.Tracker{
			{ID: 1, Name: "Alpha", Credentials: `{"api_key":"secret","username":"user"}`},
		}, nil)

		trackers, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetAll()

		require.NoError(t, err)
		require.Len(t, trackers, 1)
		assert.Equal(t, map[string]string{"username": "user"}, trackers[0].PublicCredentials)
	})

	t.Run("returns empty slice when none exist", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindAll().Return([]domain.Tracker{}, nil)

		trackers, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetAll()

		require.NoError(t, err)
		assert.Empty(t, trackers)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindAll().Return(nil, errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetAll()

		assert.Error(t, err)
	})
}

func TestTrackerService_GetByID(t *testing.T) {
	t.Run("returns tracker with redacted credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(
			&domain.Tracker{ID: 1, Name: "Alpha", Credentials: `{"token":"secret","username":"user"}`},
			nil,
		)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetByID(1)

		require.NoError(t, err)
		require.NotNil(t, tracker)
		assert.Equal(t, "Alpha", tracker.Name)
		assert.Equal(t, map[string]string{"username": "user"}, tracker.PublicCredentials)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetByID(99)

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetByID(1)

		assert.Error(t, err)
	})
}

func TestTrackerService_GetActive(t *testing.T) {
	t.Run("returns active trackers with redacted credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindActive().Return([]domain.Tracker{
			{ID: 1, Active: true, Credentials: `{"api_key":"secret"}`},
		}, nil)

		trackers, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetActive()

		require.NoError(t, err)
		require.Len(t, trackers, 1)
		assert.Nil(t, trackers[0].PublicCredentials)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindActive().Return(nil, errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).GetActive()

		assert.Error(t, err)
	})
}

func TestTrackerService_Create(t *testing.T) {
	t.Run("applies default cron and credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().Create(&domain.Tracker{
			Name:        "Alpha",
			ScraperKey:  "generic",
			Credentials: "{}",
			CronExpr:    "@hourly",
			Active:      true,
		}).Return(nil)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Create(domain.CreateTrackerInput{
			Name:       "Alpha",
			ScraperKey: "generic",
		})

		require.NoError(t, err)
		assert.Equal(t, "@hourly", tracker.CronExpr)
		assert.Equal(t, "{}", tracker.Credentials)
	})

	t.Run("respects provided cron expression and credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().Create(&domain.Tracker{
			Name:        "Beta",
			ScraperKey:  "unit3d",
			Credentials: `{"api_key":"k"}`,
			CronExpr:    "@daily",
			Active:      true,
		}).Return(nil)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Create(domain.CreateTrackerInput{
			Name:        "Beta",
			ScraperKey:  "unit3d",
			Credentials: `{"api_key":"k"}`,
			CronExpr:    "@daily",
		})

		require.NoError(t, err)
		assert.Equal(t, "@daily", tracker.CronExpr)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().Create(&domain.Tracker{
			Name:        "Alpha",
			ScraperKey:  "generic",
			Credentials: "{}",
			CronExpr:    "@hourly",
			Active:      true,
		}).Return(errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Create(domain.CreateTrackerInput{
			Name:       "Alpha",
			ScraperKey: "generic",
		})

		assert.Error(t, err)
	})
}

func TestTrackerService_Update(t *testing.T) {
	newExisting := func() *domain.Tracker {
		return &domain.Tracker{
			ID:          1,
			Name:        "Alpha",
			ScraperKey:  "generic",
			Credentials: `{"username":"user","api_key":"old"}`,
			CronExpr:    "@hourly",
			Active:      true,
		}
	}

	t.Run("updates name and merges credentials", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(newExisting(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(tr *domain.Tracker) bool {
			return tr.Name == "New Name" && tr.Credentials == `{"api_key":"old","username":"newuser"}`
		})).Return(nil)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{
			Name:        ptr("New Name"),
			Credentials: ptr(`{"username":"newuser"}`),
		})

		require.NoError(t, err)
		assert.Equal(t, "New Name", tracker.Name)
	})

	t.Run("updates cron expression", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(newExisting(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(tr *domain.Tracker) bool {
			return tr.CronExpr == "@daily"
		})).Return(nil)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{
			CronExpr: ptr("@daily"),
		})

		require.NoError(t, err)
		assert.Equal(t, "@daily", tracker.CronExpr)
	})

	t.Run("updates active flag", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(newExisting(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(tr *domain.Tracker) bool {
			return !tr.Active
		})).Return(nil)

		tracker, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{
			Active: ptr(false),
		})

		require.NoError(t, err)
		assert.False(t, tracker.Active)
	})

	t.Run("returns error when incoming credentials are invalid JSON", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(newExisting(), nil)

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{
			Credentials: ptr(`not-json`),
		})

		assert.Error(t, err)
	})

	t.Run("returns error when existing credentials are invalid JSON", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		corrupt := &domain.Tracker{
			ID:          1,
			Name:        "Alpha",
			ScraperKey:  "generic",
			Credentials: `not-json`,
			CronExpr:    "@hourly",
			Active:      true,
		}
		repo.EXPECT().FindByID(uint(1)).Return(corrupt, nil)

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{
			Credentials: ptr(`{"api_key":"new"}`),
		})

		assert.Error(t, err)
	})

	t.Run("returns error when tracker not found", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(99, domain.UpdateTrackerInput{})

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates FindByID error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{})

		assert.Error(t, err)
	})

	t.Run("propagates Update error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(newExisting(), nil)
		repo.EXPECT().Update(mock.Anything).Return(errors.New("db error"))

		_, err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Update(1, domain.UpdateTrackerInput{})

		assert.Error(t, err)
	})
}

func TestTrackerService_Delete(t *testing.T) {
	t.Run("deletes existing tracker", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{ID: 1}, nil)
		repo.EXPECT().Delete(uint(1)).Return(nil)

		err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Delete(1)

		require.NoError(t, err)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Delete(99)

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates FindByID error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Delete(1)

		assert.Error(t, err)
	})

	t.Run("propagates Delete error", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{ID: 1}, nil)
		repo.EXPECT().Delete(uint(1)).Return(errors.New("db error"))

		err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).Delete(1)

		assert.Error(t, err)
	})
}

func TestTrackerService_Test(t *testing.T) {
	t.Run("fetches stats with provided credentials", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.MatchedBy(func(tr domain.Tracker) bool {
			return tr.ScraperKey == "generic" && tr.Credentials == `{"cookie":"abc"}`
		})).Return(&domain.TrackerStats{}, nil)
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)

		err := service.NewTrackerService(mocks.NewMockTrackerRepository(t), registry).
			Test("generic", `{"cookie":"abc"}`)

		assert.NoError(t, err)
	})

	t.Run("uses empty JSON object when credentials are empty", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.MatchedBy(func(tr domain.Tracker) bool {
			return tr.Credentials == "{}"
		})).Return(&domain.TrackerStats{}, nil)
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)

		err := service.NewTrackerService(mocks.NewMockTrackerRepository(t), registry).
			Test("generic", "")

		assert.NoError(t, err)
	})

	t.Run("returns error for unknown scraper key", func(t *testing.T) {
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("unknown").Return(nil, false)

		err := service.NewTrackerService(mocks.NewMockTrackerRepository(t), registry).
			Test("unknown", "{}")

		assert.ErrorContains(t, err, "unknown scraper key")
	})

	t.Run("propagates scraper fetch error", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.Anything).Return(nil, errors.New("401 Unauthorized"))
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)

		err := service.NewTrackerService(mocks.NewMockTrackerRepository(t), registry).
			Test("generic", "{}")

		assert.ErrorContains(t, err, "401 Unauthorized")
	})
}

func TestTrackerService_TestByID(t *testing.T) {
	t.Run("fetches with stored credentials", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.MatchedBy(func(tr domain.Tracker) bool {
			return tr.Credentials == `{"cookie":"abc"}`
		})).Return(&domain.TrackerStats{}, nil)
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{
			ID: 1, ScraperKey: "generic", Credentials: `{"cookie":"abc"}`,
		}, nil)

		err := service.NewTrackerService(repo, registry).TestByID(1, "")

		assert.NoError(t, err)
	})

	t.Run("merges credentials override, override keys win", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.MatchedBy(func(tr domain.Tracker) bool {
			var creds map[string]string
			_ = json.Unmarshal([]byte(tr.Credentials), &creds)
			return creds["cookie"] == "new-cookie" && creds["username"] == "user"
		})).Return(&domain.TrackerStats{}, nil)
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{
			ID: 1, ScraperKey: "generic",
			Credentials: `{"cookie":"stored","username":"user"}`,
		}, nil)

		err := service.NewTrackerService(repo, registry).TestByID(1, `{"cookie":"new-cookie"}`)

		assert.NoError(t, err)
	})

	t.Run("returns error when tracker not found", func(t *testing.T) {
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(999)).Return(nil, nil)

		err := service.NewTrackerService(repo, mocks.NewMockScraperRegistry(t)).TestByID(999, "")

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("returns error for unknown scraper key", func(t *testing.T) {
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("unknown").Return(nil, false)
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{
			ID: 1, ScraperKey: "unknown", Credentials: "{}",
		}, nil)

		err := service.NewTrackerService(repo, registry).TestByID(1, "")

		assert.ErrorContains(t, err, "unknown scraper key")
	})

	t.Run("propagates scraper fetch error", func(t *testing.T) {
		scraper := mocks.NewMockTrackerScraper(t)
		scraper.EXPECT().Fetch(mock.Anything, mock.Anything).Return(nil, errors.New("scrape failed"))
		registry := mocks.NewMockScraperRegistry(t)
		registry.EXPECT().Get("generic").Return(scraper, true)
		repo := mocks.NewMockTrackerRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Tracker{
			ID: 1, ScraperKey: "generic", Credentials: `{"cookie":"abc"}`,
		}, nil)

		err := service.NewTrackerService(repo, registry).TestByID(1, "")

		assert.ErrorContains(t, err, "scrape failed")
	})
}
