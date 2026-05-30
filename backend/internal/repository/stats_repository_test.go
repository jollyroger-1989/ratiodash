package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
	"gorm.io/gorm"
)

// seedTracker creates and persists a tracker, returning it with its assigned ID.
// Shared by all stats sub-tests that need a parent tracker row.
func seedTracker(t *testing.T, db *gorm.DB, name string) *domain.Tracker {
	t.Helper()
	tr := newTracker(name)
	require.NoError(t, repository.NewTrackerRepository(db).Create(tr))
	return tr
}

func newStats(trackerID uint) *domain.TrackerStats {
	return &domain.TrackerStats{
		TrackerID:  trackerID,
		Uploaded:   1_000_000,
		Downloaded: 500_000,
		Ratio:      2.0,
		FetchedAt:  time.Now().UTC(),
	}
}

func TestStatsRepository_FindByTrackerID(t *testing.T) {
	t.Run("returns empty slice when none exist", func(t *testing.T) {
		repo := repository.NewStatsRepository(testutil.NewDB(t))

		stats, err := repo.FindByTrackerID(999, 0)

		require.NoError(t, err)
		assert.Equal(t, []domain.TrackerStats{}, stats)
	})

	t.Run("returns all stats for tracker", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")
		require.NoError(t, repo.Create(newStats(tr.ID)))
		require.NoError(t, repo.Create(newStats(tr.ID)))

		stats, err := repo.FindByTrackerID(tr.ID, 0)

		require.NoError(t, err)
		assert.Len(t, stats, 2)
	})

	t.Run("respects limit", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")
		for i := 0; i < 5; i++ {
			require.NoError(t, repo.Create(newStats(tr.ID)))
		}

		stats, err := repo.FindByTrackerID(tr.ID, 3)

		require.NoError(t, err)
		assert.Len(t, stats, 3)
	})

	t.Run("does not return stats from another tracker", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr1 := seedTracker(t, db, "Alpha")
		tr2 := seedTracker(t, db, "Beta")
		require.NoError(t, repo.Create(newStats(tr1.ID)))

		stats, err := repo.FindByTrackerID(tr2.ID, 0)

		require.NoError(t, err)
		assert.Equal(t, []domain.TrackerStats{}, stats)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewStatsRepository(db)

		_, err = repo.FindByTrackerID(1, 0)

		assert.Error(t, err)
	})
}

func TestStatsRepository_FindLatestByTrackerID(t *testing.T) {
	t.Run("returns nil when no stats exist", func(t *testing.T) {
		repo := repository.NewStatsRepository(testutil.NewDB(t))

		s, err := repo.FindLatestByTrackerID(999)

		require.NoError(t, err)
		assert.Nil(t, s)
	})

	t.Run("returns the most recent snapshot", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")

		older := newStats(tr.ID)
		older.Ratio = 1.0
		older.FetchedAt = time.Now().UTC().Add(-time.Hour)
		require.NoError(t, repo.Create(older))

		latest := newStats(tr.ID)
		latest.Ratio = 3.0
		latest.FetchedAt = time.Now().UTC()
		require.NoError(t, repo.Create(latest))

		found, err := repo.FindLatestByTrackerID(tr.ID)

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, latest.ID, found.ID)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewStatsRepository(db)

		_, err = repo.FindLatestByTrackerID(1)

		assert.Error(t, err)
	})
}

func TestStatsRepository_FindLatestAll(t *testing.T) {
	t.Run("returns empty slice when no stats exist", func(t *testing.T) {
		repo := repository.NewStatsRepository(testutil.NewDB(t))

		stats, err := repo.FindLatestAll()

		require.NoError(t, err)
		assert.Equal(t, []domain.TrackerStats{}, stats)
	})

	t.Run("returns one entry per tracker", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr1 := seedTracker(t, db, "Alpha")
		tr2 := seedTracker(t, db, "Beta")

		// Two snapshots for tr1, one for tr2
		require.NoError(t, repo.Create(newStats(tr1.ID)))
		require.NoError(t, repo.Create(newStats(tr1.ID)))
		require.NoError(t, repo.Create(newStats(tr2.ID)))

		stats, err := repo.FindLatestAll()

		require.NoError(t, err)
		assert.Len(t, stats, 2)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewStatsRepository(db)

		_, err = repo.FindLatestAll()

		assert.Error(t, err)
	})
}

func TestStatsRepository_FindGlobalHistory(t *testing.T) {
	t.Run("returns empty slice when no stats exist", func(t *testing.T) {
		repo := repository.NewStatsRepository(testutil.NewDB(t))

		points, err := repo.FindGlobalHistory(0)

		require.NoError(t, err)
		assert.Equal(t, []domain.GlobalStatsPoint{}, points)
	})

	t.Run("aggregates latest snapshot per tracker per day", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr1 := seedTracker(t, db, "Alpha")
		tr2 := seedTracker(t, db, "Beta")

		day1 := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
		day2 := time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)

		require.NoError(t, repo.Create(&domain.TrackerStats{TrackerID: tr1.ID, Uploaded: 100, Downloaded: 50, Ratio: 2, FetchedAt: day1.Add(8 * time.Hour)}))
		require.NoError(t, repo.Create(&domain.TrackerStats{TrackerID: tr1.ID, Uploaded: 150, Downloaded: 75, Ratio: 2, FetchedAt: day1.Add(20 * time.Hour)}))
		require.NoError(t, repo.Create(&domain.TrackerStats{TrackerID: tr2.ID, Uploaded: 200, Downloaded: 100, Ratio: 2, FetchedAt: day1.Add(10 * time.Hour)}))
		require.NoError(t, repo.Create(&domain.TrackerStats{TrackerID: tr1.ID, Uploaded: 300, Downloaded: 100, Ratio: 3, FetchedAt: day2.Add(9 * time.Hour)}))

		points, err := repo.FindGlobalHistory(0)

		require.NoError(t, err)
		require.Len(t, points, 2)
		assert.Equal(t, day2, points[0].FetchedAt)
		assert.Equal(t, int64(300), points[0].Uploaded)
		assert.Equal(t, int64(100), points[0].Downloaded)
		assert.Equal(t, 3.0, points[0].Ratio)
		assert.Equal(t, day1, points[1].FetchedAt)
		assert.Equal(t, int64(350), points[1].Uploaded)
		assert.Equal(t, int64(175), points[1].Downloaded)
		assert.Equal(t, 2.0, points[1].Ratio)
	})

	t.Run("respects limit", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")

		for i := 0; i < 3; i++ {
			day := time.Date(2026, 5, 27+i, 0, 0, 0, 0, time.UTC)
			require.NoError(t, repo.Create(&domain.TrackerStats{TrackerID: tr.ID, Uploaded: int64((i + 1) * 100), Downloaded: int64((i + 1) * 50), Ratio: 2, FetchedAt: day.Add(12 * time.Hour)}))
		}

		points, err := repo.FindGlobalHistory(2)

		require.NoError(t, err)
		require.Len(t, points, 2)
		assert.Equal(t, time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC), points[0].FetchedAt)
		assert.Equal(t, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), points[1].FetchedAt)
	})
}

func TestStatsRepository_Create(t *testing.T) {
	t.Run("assigns ID on success", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")
		s := newStats(tr.ID)

		err := repo.Create(s)

		require.NoError(t, err)
		assert.NotZero(t, s.ID)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewStatsRepository(db)

		err = repo.Create(newStats(1))

		assert.Error(t, err)
	})
}

func TestStatsRepository_Delete(t *testing.T) {
	t.Run("removes the snapshot", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr := seedTracker(t, db, "Alpha")
		s := newStats(tr.ID)
		require.NoError(t, repo.Create(s))

		require.NoError(t, repo.Delete(s.ID, tr.ID))

		remaining, err := repo.FindByTrackerID(tr.ID, 0)
		require.NoError(t, err)
		assert.Empty(t, remaining)
	})

	t.Run("returns error when stat does not exist", func(t *testing.T) {
		repo := repository.NewStatsRepository(testutil.NewDB(t))

		err := repo.Delete(999, 999)

		assert.Error(t, err)
	})

	t.Run("returns error when stat belongs to a different tracker", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewStatsRepository(db)
		tr1 := seedTracker(t, db, "Alpha")
		tr2 := seedTracker(t, db, "Beta")
		s := newStats(tr1.ID)
		require.NoError(t, repo.Create(s))

		err := repo.Delete(s.ID, tr2.ID)

		assert.Error(t, err)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewStatsRepository(db)

		err = repo.Delete(1, 1)

		assert.Error(t, err)
	})
}
