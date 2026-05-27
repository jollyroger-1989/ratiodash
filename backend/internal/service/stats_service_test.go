package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func TestStatsService_GetHistory(t *testing.T) {
	t.Run("returns stats for tracker", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().FindByTrackerID(uint(1), 10).Return([]domain.TrackerStats{
			{ID: 1, TrackerID: 1, Uploaded: 100, Downloaded: 50, Ratio: 2.0},
		}, nil)

		stats, err := service.NewStatsService(statsRepo, trackerRepo).GetHistory(1, 10)

		require.NoError(t, err)
		assert.Len(t, stats, 1)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().FindByTrackerID(uint(1), 50).Return(nil, errors.New("db error"))

		_, err := service.NewStatsService(statsRepo, trackerRepo).GetHistory(1, 50)

		assert.Error(t, err)
	})
}

func TestStatsService_GetLatest(t *testing.T) {
	t.Run("returns latest snapshot", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().FindLatestByTrackerID(uint(1)).Return(
			&domain.TrackerStats{ID: 5, TrackerID: 1, Ratio: 1.5},
			nil,
		)

		stat, err := service.NewStatsService(statsRepo, trackerRepo).GetLatest(1)

		require.NoError(t, err)
		assert.Equal(t, uint(5), stat.ID)
	})

	t.Run("returns nil when no stats exist", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().FindLatestByTrackerID(uint(1)).Return(nil, nil)

		stat, err := service.NewStatsService(statsRepo, trackerRepo).GetLatest(1)

		require.NoError(t, err)
		assert.Nil(t, stat)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().FindLatestByTrackerID(uint(1)).Return(nil, errors.New("db error"))

		_, err := service.NewStatsService(statsRepo, trackerRepo).GetLatest(1)

		assert.Error(t, err)
	})
}

func TestStatsService_DeleteEntry(t *testing.T) {
	t.Run("deletes entry", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().Delete(uint(3), uint(1)).Return(nil)

		err := service.NewStatsService(statsRepo, trackerRepo).DeleteEntry(3, 1)

		require.NoError(t, err)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		statsRepo.EXPECT().Delete(uint(3), uint(1)).Return(errors.New("not found"))

		err := service.NewStatsService(statsRepo, trackerRepo).DeleteEntry(3, 1)

		assert.Error(t, err)
	})
}

func TestStatsService_GetDashboard(t *testing.T) {
	t.Run("pairs trackers with their latest stats", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)

		trackerRepo.EXPECT().FindAll().Return([]domain.Tracker{
			{ID: 1, Name: "Alpha", Credentials: `{"api_key":"s"}`},
			{ID: 2, Name: "Beta", Credentials: "{}"},
		}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{
			{ID: 10, TrackerID: 1, Ratio: 2.5},
		}, nil)

		entries, err := service.NewStatsService(statsRepo, trackerRepo).GetDashboard()

		require.NoError(t, err)
		require.Len(t, entries, 2)
		require.NotNil(t, entries[0].Stats)
		assert.Equal(t, 2.5, entries[0].Stats.Ratio)
		assert.Nil(t, entries[1].Stats)
	})

	t.Run("redacts credentials in dashboard entries", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)

		trackerRepo.EXPECT().FindAll().Return([]domain.Tracker{
			{ID: 1, Name: "Alpha", Credentials: `{"api_key":"secret","username":"user"}`},
		}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{}, nil)

		entries, err := service.NewStatsService(statsRepo, trackerRepo).GetDashboard()

		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, map[string]string{"username": "user"}, entries[0].Tracker.PublicCredentials)
	})

	t.Run("propagates trackerRepo error", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		trackerRepo.EXPECT().FindAll().Return(nil, errors.New("db error"))

		_, err := service.NewStatsService(statsRepo, trackerRepo).GetDashboard()

		assert.Error(t, err)
	})

	t.Run("propagates statsRepo error", func(t *testing.T) {
		statsRepo := mocks.NewMockStatsRepository(t)
		trackerRepo := mocks.NewMockTrackerRepository(t)
		trackerRepo.EXPECT().FindAll().Return([]domain.Tracker{{ID: 1}}, nil)
		statsRepo.EXPECT().FindLatestAll().Return(nil, errors.New("db error"))

		_, err := service.NewStatsService(statsRepo, trackerRepo).GetDashboard()

		assert.Error(t, err)
	})
}
