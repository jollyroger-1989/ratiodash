package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

// newTracker builds a minimal valid Tracker. Shared by stats tests too.
func newTracker(name string) *domain.Tracker {
	return &domain.Tracker{
		Name:       name,
		ScraperKey: "generic",
		CronExpr:   "@hourly",
	}
}

func TestTrackerRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no trackers exist", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))

		trackers, err := repo.FindAll(nil)

		require.NoError(t, err)
		assert.Equal(t, []domain.Tracker{}, trackers)
	})

	t.Run("returns all created trackers", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		require.NoError(t, repo.Create(newTracker("Alpha")))
		require.NoError(t, repo.Create(newTracker("Beta")))

		trackers, err := repo.FindAll(nil)

		require.NoError(t, err)
		assert.Len(t, trackers, 2)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		_, err = repo.FindAll(nil)

		assert.Error(t, err)
	})
}

func TestTrackerRepository_FindByID(t *testing.T) {
	t.Run("returns tracker when found", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("Alpha")
		require.NoError(t, repo.Create(tr))

		found, err := repo.FindByID(tr.ID)

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Alpha", found.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))

		found, err := repo.FindByID(999)

		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		_, err = repo.FindByID(1)

		assert.Error(t, err)
	})
}

func TestTrackerRepository_FindActive(t *testing.T) {
	t.Run("returns empty slice when no active trackers", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))

		trackers, err := repo.FindActive()

		require.NoError(t, err)
		assert.Equal(t, []domain.Tracker{}, trackers)
	})

	t.Run("returns only active trackers", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))

		active := newTracker("Active")
		require.NoError(t, repo.Create(active)) // default Active=true

		inactive := newTracker("Inactive")
		require.NoError(t, repo.Create(inactive))
		inactive.Active = false
		require.NoError(t, repo.Update(inactive)) // persist Active=false

		trackers, err := repo.FindActive()

		require.NoError(t, err)
		assert.Len(t, trackers, 1)
		assert.Equal(t, "Active", trackers[0].Name)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		_, err = repo.FindActive()

		assert.Error(t, err)
	})
}

func TestTrackerRepository_Create(t *testing.T) {
	t.Run("assigns ID on success", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("Alpha")

		err := repo.Create(tr)

		require.NoError(t, err)
		assert.NotZero(t, tr.ID)
	})

	t.Run("returns error on duplicate name", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		require.NoError(t, repo.Create(newTracker("Alpha")))

		err := repo.Create(newTracker("Alpha"))

		assert.Error(t, err)
	})
}

func TestTrackerRepository_Update(t *testing.T) {
	t.Run("persists changed fields", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("Before")
		require.NoError(t, repo.Create(tr))

		tr.Name = "After"
		require.NoError(t, repo.Update(tr))

		found, err := repo.FindByID(tr.ID)
		require.NoError(t, err)
		assert.Equal(t, "After", found.Name)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		err = repo.Update(newTracker("Alpha"))

		assert.Error(t, err)
	})
}

func TestTrackerRepository_Delete(t *testing.T) {
	t.Run("removes the tracker", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("ToDelete")
		require.NoError(t, repo.Create(tr))

		require.NoError(t, repo.Delete(tr.ID))

		found, err := repo.FindByID(tr.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		err = repo.Delete(1)

		assert.Error(t, err)
	})
}

func TestTrackerRepository_UpdateScrapeStatus(t *testing.T) {
	t.Run("sets last_error and last_scraped_at on failure", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("Alpha")
		require.NoError(t, repo.Create(tr))

		require.NoError(t, repo.UpdateScrapeStatus(tr.ID, "timeout"))

		found, err := repo.FindByID(tr.ID)
		require.NoError(t, err)
		assert.Equal(t, "timeout", found.LastError)
		assert.NotNil(t, found.LastScrapedAt)
	})

	t.Run("clears last_error on success", func(t *testing.T) {
		repo := repository.NewTrackerRepository(testutil.NewDB(t))
		tr := newTracker("Alpha")
		require.NoError(t, repo.Create(tr))
		require.NoError(t, repo.UpdateScrapeStatus(tr.ID, "timeout"))

		require.NoError(t, repo.UpdateScrapeStatus(tr.ID, ""))

		found, err := repo.FindByID(tr.ID)
		require.NoError(t, err)
		assert.Empty(t, found.LastError)
		assert.NotNil(t, found.LastScrapedAt)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewTrackerRepository(db)

		err = repo.UpdateScrapeStatus(1, "timeout")

		assert.Error(t, err)
	})
}
