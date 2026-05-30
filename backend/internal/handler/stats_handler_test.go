package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

type statsTestEnv struct {
	api         humatest.TestAPI
	statsRepo   domain.StatsRepository
	trackerRepo domain.TrackerRepository
}

func setupStatsHandler(t *testing.T) statsTestEnv {
	t.Helper()
	db := testutil.NewDB(t)
	trackerRepo := repository.NewTrackerRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	svc := service.NewStatsService(statsRepo, trackerRepo)
	h := handler.NewStatsHandler(svc)
	api := testutil.NewAPI(t)
	handler.RegisterStatsRoutes(api, h)
	return statsTestEnv{api: api, statsRepo: statsRepo, trackerRepo: trackerRepo}
}

// seedTracker creates a tracker row directly via the repository and returns it.
func seedTracker(t *testing.T, repo domain.TrackerRepository, name string) domain.Tracker {
	t.Helper()
	tr := &domain.Tracker{
		Name:        name,
		ScraperKey:  "generic",
		Credentials: "{}",
		CronExpr:    "@hourly",
		Active:      true,
	}
	require.NoError(t, repo.Create(tr))
	return *tr
}

// seedStats creates a stats snapshot directly via the repository and returns it.
func seedStats(t *testing.T, repo domain.StatsRepository, trackerID uint, uploaded, downloaded int64) domain.TrackerStats {
	t.Helper()
	s := &domain.TrackerStats{
		TrackerID:  trackerID,
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Ratio:      float64(uploaded) / float64(downloaded),
		FetchedAt:  time.Now().UTC(),
	}
	require.NoError(t, repo.Create(s))
	return *s
}

func TestStatsHandler_GetDashboard(t *testing.T) {
	t.Run("returns empty list when no trackers", func(t *testing.T) {
		env := setupStatsHandler(t)

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers/stats")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.DashboardEntry
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns tracker with nil stats when none scraped yet", func(t *testing.T) {
		env := setupStatsHandler(t)
		seedTracker(t, env.trackerRepo, "Alpha")

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers/stats")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.DashboardEntry
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "Alpha", body[0].Tracker.Name)
		assert.Nil(t, body[0].Stats)
	})

	t.Run("pairs tracker with latest stats", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")
		seedStats(t, env.statsRepo, tr.ID, 100, 50)

		resp := env.api.Do(http.MethodGet, "/api/v1/trackers/stats")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.DashboardEntry
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		require.NotNil(t, body[0].Stats)
		assert.Equal(t, int64(100), body[0].Stats.Uploaded)
	})
}

func TestStatsHandler_GetTrackerHistory(t *testing.T) {
	t.Run("returns history for a tracker", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")
		seedStats(t, env.statsRepo, tr.ID, 100, 50)
		seedStats(t, env.statsRepo, tr.ID, 200, 100)

		resp := env.api.Do(http.MethodGet, fmt.Sprintf("/api/v1/trackers/%d/stats", tr.ID))

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.TrackerStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Len(t, body, 2)
	})

	t.Run("respects the limit query parameter", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")
		for i := range 5 {
			seedStats(t, env.statsRepo, tr.ID, int64(i+1)*100, int64(i+1)*50)
		}

		resp := env.api.Do(http.MethodGet,
			fmt.Sprintf("/api/v1/trackers/%d/stats?limit=3", tr.ID))

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.TrackerStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Len(t, body, 3)
	})

	t.Run("returns empty list when tracker has no stats", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")

		resp := env.api.Do(http.MethodGet,
			fmt.Sprintf("/api/v1/trackers/%d/stats", tr.ID))

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.TrackerStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})
}

func TestStatsHandler_GetGlobalHistory(t *testing.T) {
	t.Run("returns empty list when no data exists", func(t *testing.T) {
		env := setupStatsHandler(t)

		resp := env.api.Do(http.MethodGet, "/api/v1/stats/global")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.GlobalStatsPoint
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns aggregated daily snapshots", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr1 := seedTracker(t, env.trackerRepo, "Alpha")
		tr2 := seedTracker(t, env.trackerRepo, "Beta")

		t0 := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
		require.NoError(t, env.statsRepo.Create(&domain.TrackerStats{TrackerID: tr1.ID, Uploaded: 100, Downloaded: 50, Ratio: 2, FetchedAt: t0.Add(10 * time.Hour)}))
		require.NoError(t, env.statsRepo.Create(&domain.TrackerStats{TrackerID: tr2.ID, Uploaded: 200, Downloaded: 100, Ratio: 2, FetchedAt: t0.Add(11 * time.Hour)}))

		resp := env.api.Do(http.MethodGet, "/api/v1/stats/global")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.GlobalStatsPoint
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, int64(300), body[0].Uploaded)
		assert.Equal(t, int64(150), body[0].Downloaded)
		assert.Equal(t, 2.0, body[0].Ratio)
	})

	t.Run("respects limit query parameter", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")

		for i := 0; i < 3; i++ {
			day := time.Date(2026, 5, 27+i, 0, 0, 0, 0, time.UTC)
			require.NoError(t, env.statsRepo.Create(&domain.TrackerStats{
				TrackerID: tr.ID, Uploaded: int64((i + 1) * 100), Downloaded: int64((i + 1) * 50), Ratio: 2, FetchedAt: day.Add(9 * time.Hour),
			}))
		}

		resp := env.api.Do(http.MethodGet, "/api/v1/stats/global?limit=2")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.GlobalStatsPoint
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Len(t, body, 2)
	})
}

func TestStatsHandler_GetLatestStats(t *testing.T) {
	t.Run("returns the most recent snapshot", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")
		seedStats(t, env.statsRepo, tr.ID, 100, 50)
		latest := seedStats(t, env.statsRepo, tr.ID, 200, 100)

		resp := env.api.Do(http.MethodGet,
			fmt.Sprintf("/api/v1/trackers/%d/stats/latest", tr.ID))

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.TrackerStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, latest.ID, body.ID)
		assert.Equal(t, int64(200), body.Uploaded)
	})

	t.Run("returns 404 when no stats exist for tracker", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")

		resp := env.api.Do(http.MethodGet,
			fmt.Sprintf("/api/v1/trackers/%d/stats/latest", tr.ID))

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestStatsHandler_DeleteStat(t *testing.T) {
	t.Run("deletes a stat entry", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")
		s := seedStats(t, env.statsRepo, tr.ID, 100, 50)

		resp := env.api.Do(http.MethodDelete,
			fmt.Sprintf("/api/v1/trackers/%d/stats/%d", tr.ID, s.ID))

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 404 for missing stat entry", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr := seedTracker(t, env.trackerRepo, "Alpha")

		resp := env.api.Do(http.MethodDelete,
			fmt.Sprintf("/api/v1/trackers/%d/stats/999", tr.ID))

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestStatsHandler_DeleteStat_WrongTracker(t *testing.T) {
	t.Run("returns 404 when stat belongs to a different tracker", func(t *testing.T) {
		env := setupStatsHandler(t)
		tr1 := seedTracker(t, env.trackerRepo, "Alpha")
		tr2 := seedTracker(t, env.trackerRepo, "Beta")
		s := seedStats(t, env.statsRepo, tr1.ID, 100, 50)

		resp := env.api.Do(http.MethodDelete,
			fmt.Sprintf("/api/v1/trackers/%d/stats/%d", tr2.ID, s.ID))

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
