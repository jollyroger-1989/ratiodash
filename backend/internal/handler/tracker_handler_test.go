package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

type trackerTestEnv struct {
	api     humatest.TestAPI
	refresh *mocks.MockRefreshService
	sched   *mocks.MockRefresher
	stats   domain.StatsRepository
}

func setupTrackerHandler(t *testing.T) trackerTestEnv {
	t.Helper()
	db := testutil.NewDB(t)
	repo := repository.NewTrackerRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	registry := mocks.NewMockScraperRegistry(t)
	svc := service.NewTrackerService(repo, registry)
	statsSvc := service.NewStatsService(statsRepo, repo)
	refresh := mocks.NewMockRefreshService(t)
	sched := mocks.NewMockRefresher(t)
	h := handler.NewTrackerHandler(svc, statsSvc, refresh, sched)
	api := testutil.NewAPI(t)
	handler.RegisterTrackerRoutes(api, h)
	return trackerTestEnv{api: api, refresh: refresh, sched: sched, stats: statsRepo}
}

func TestTrackerHandler_List(t *testing.T) {
	t.Run("returns empty list when no trackers", func(t *testing.T) {
		env := setupTrackerHandler(t)

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.Tracker
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns existing trackers", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil)
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil)
		createResp := env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})
		require.Equal(t, http.StatusCreated, createResp.Code)

		var created domain.Tracker
		require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
		err := env.stats.Create(&domain.TrackerStats{TrackerID: created.ID, Uploaded: 42, Downloaded: 21, Ratio: 2.0})
		require.NoError(t, err)

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.Tracker
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "Alpha", body[0].Name)
		require.NotNil(t, body[0].Stats)
		assert.Equal(t, int64(42), body[0].Stats.Uploaded)
		assert.Equal(t, int64(21), body[0].Stats.Downloaded)
		assert.Equal(t, 2.0, body[0].Stats.Ratio)
	})
}

func TestTrackerHandler_Get(t *testing.T) {
	t.Run("returns tracker by ID", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil)
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil)
		createResp := env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})
		require.Equal(t, http.StatusCreated, createResp.Code)
		var created domain.Tracker
		require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers/1")

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.Tracker
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Alpha", body.Name)
	})

	t.Run("returns 404 for missing tracker", func(t *testing.T) {
		env := setupTrackerHandler(t)

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers/999")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestTrackerHandler_Create(t *testing.T) {
	t.Run("creates tracker and triggers initial scrape", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil)
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil)

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})

		require.Equal(t, http.StatusCreated, resp.Code)
		var body domain.Tracker
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Alpha", body.Name)
		assert.Equal(t, "generic", body.ScraperKey)
		assert.NotZero(t, body.ID)
	})

	t.Run("rolls back and returns 422 when scrape fails", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).
			Return(errors.New("connection refused"))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Beta", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

		// Confirm tracker was rolled back
		listResp := env.api.Do(http.MethodGet, "/api/v1/trackers")
		var trackers []domain.Tracker
		require.NoError(t, json.NewDecoder(listResp.Body).Decode(&trackers))
		assert.Empty(t, trackers)
	})
}

func TestTrackerHandler_Update(t *testing.T) {
	t.Run("updates tracker and reschedules", func(t *testing.T) {
		env := setupTrackerHandler(t)
		// Create
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil).Once()
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil).Once()
		env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})

		// Update
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil).Once()
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil).Once()

		resp := env.api.Do(http.MethodPatch, "/api/v1/trackers/1",
			map[string]string{"name": "Alpha Updated"})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.Tracker
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Alpha Updated", body.Name)
	})

	t.Run("returns 404 for missing tracker", func(t *testing.T) {
		env := setupTrackerHandler(t)

		resp := env.api.Do(http.MethodPatch, "/api/v1/trackers/999",
			map[string]string{"name": "X"})

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("restores old values and returns 422 when scrape fails after update", func(t *testing.T) {
		env := setupTrackerHandler(t)
		// Create
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil).Once()
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil).Once()
		env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})

		// Update — scrape fails
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).
			Return(errors.New("scrape failed")).Once()

		resp := env.api.Do(http.MethodPatch, "/api/v1/trackers/1",
			map[string]string{"name": "Alpha Bad"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

		// Name should be restored
		getResp := env.api.Do(http.MethodGet, "/api/v1/trackers/1")
		var tracker domain.Tracker
		require.NoError(t, json.NewDecoder(getResp.Body).Decode(&tracker))
		assert.Equal(t, "Alpha", tracker.Name)
	})
}

func TestTrackerHandler_Delete(t *testing.T) {
	t.Run("deletes tracker and unschedules", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, mock.AnythingOfType("uint")).Return(nil)
		env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil)
		env.api.Do(http.MethodPost, "/api/v1/trackers",
			map[string]string{"name": "Alpha", "scraper_key": "generic", "credentials": "{}", "cron_expr": "@hourly"})
		env.sched.EXPECT().Unschedule(uint(1))

		resp := env.api.Do(http.MethodDelete, "/api/v1/trackers/1")

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 404 for missing tracker", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.sched.EXPECT().Unschedule(mock.AnythingOfType("uint")).Maybe()

		resp := env.api.Do(http.MethodDelete, "/api/v1/trackers/999")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestTrackerHandler_Refresh(t *testing.T) {
	t.Run("returns 204 on successful refresh", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, uint(1)).Return(nil)

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/1/refresh")

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 500 when refresh fails", func(t *testing.T) {
		env := setupTrackerHandler(t)
		env.refresh.EXPECT().RefreshTracker(mock.Anything, uint(1)).
			Return(errors.New("scrape error"))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/1/refresh")

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

type trackerTestEnvWithMock struct {
	api     humatest.TestAPI
	service *mocks.MockTrackerService
}

func setupTrackerHandlerWithMock(t *testing.T) trackerTestEnvWithMock {
	t.Helper()
	svc := mocks.NewMockTrackerService(t)
	stats := mocks.NewMockStatsService(t)
	stats.EXPECT().GetDashboard().Return([]domain.DashboardEntry{}, nil).Maybe()
	h := handler.NewTrackerHandler(svc, stats, mocks.NewMockRefreshService(t), mocks.NewMockRefresher(t))
	api := testutil.NewAPI(t)
	handler.RegisterTrackerRoutes(api, h)
	return trackerTestEnvWithMock{api: api, service: svc}
}

func TestTrackerHandler_Test(t *testing.T) {
	t.Run("returns 204 on success", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().Test("generic", `{"cookie":"abc"}`).Return(nil)

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/test",
			map[string]string{"scraper_key": "generic", "credentials": `{"cookie":"abc"}`})

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 422 when scraper fails", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().Test("generic", mock.Anything).
			Return(errors.New("401 Unauthorized"))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/test",
			map[string]string{"scraper_key": "generic", "credentials": "{}"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})

	t.Run("returns 422 for unknown scraper key", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().Test("unknown", mock.Anything).
			Return(errors.New(`unknown scraper key "unknown"`))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/test",
			map[string]string{"scraper_key": "unknown", "credentials": "{}"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestTrackerHandler_TestByID(t *testing.T) {
	t.Run("returns 204 on success", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().TestByID(uint(1), `{"cookie":"new"}`).Return(nil)

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/1/test",
			map[string]string{"credentials": `{"cookie":"new"}`})

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 422 when scraper fails", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().TestByID(uint(1), mock.Anything).
			Return(errors.New("scrape failed"))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/1/test",
			map[string]string{"credentials": "{}"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})

	t.Run("returns 422 when tracker not found", func(t *testing.T) {
		env := setupTrackerHandlerWithMock(t)
		env.service.EXPECT().TestByID(uint(999), "").
			Return(errors.New("tracker 999 not found"))

		resp := env.api.Do(http.MethodPost, "/api/v1/trackers/999/test",
			map[string]string{})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}
