package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func newReport(name string) *domain.Report {
	return &domain.Report{
		Name:     name,
		CronExpr: "@hourly",
	}
}

func TestReportRepository_FindAll(t *testing.T) {
	t.Run("returns empty when no reports exist", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))

		reports, err := repo.FindAll()

		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("returns all created reports", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		require.NoError(t, repo.Create(newReport("A")))
		require.NoError(t, repo.Create(newReport("B")))

		reports, err := repo.FindAll()

		require.NoError(t, err)
		assert.Len(t, reports, 2)
	})
}

func TestReportRepository_FindByID(t *testing.T) {
	t.Run("returns report when found", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("Daily")
		require.NoError(t, repo.Create(r))

		found, err := repo.FindByID(r.ID)

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Daily", found.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))

		found, err := repo.FindByID(999)

		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestReportRepository_Create(t *testing.T) {
	t.Run("assigns ID and persists", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("New")

		require.NoError(t, repo.Create(r))

		assert.NotZero(t, r.ID)
	})
}

func TestReportRepository_Update(t *testing.T) {
	t.Run("persists changes", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("Original")
		require.NoError(t, repo.Create(r))

		r.Name = "Updated"
		require.NoError(t, repo.Update(r))

		found, err := repo.FindByID(r.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", found.Name)
	})
}

func TestReportRepository_UpdateNotifierConfigs(t *testing.T) {
	t.Run("clears associations with empty list", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("A")
		require.NoError(t, repo.Create(r))

		err := repo.UpdateNotifierConfigs(r.ID, []uint{})

		require.NoError(t, err)
	})
}

func TestReportRepository_Delete(t *testing.T) {
	t.Run("deletes existing report", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("ToDelete")
		require.NoError(t, repo.Create(r))

		require.NoError(t, repo.Delete(r.ID))

		found, err := repo.FindByID(r.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))

		err := repo.Delete(999)

		assert.Error(t, err)
	})
}

func TestReportRepository_UpdateLastSentAt(t *testing.T) {
	t.Run("sets last_sent_at timestamp", func(t *testing.T) {
		repo := repository.NewReportRepository(testutil.NewDB(t))
		r := newReport("Timed")
		require.NoError(t, repo.Create(r))
		sentAt := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, repo.UpdateLastSentAt(r.ID, sentAt))

		found, err := repo.FindByID(r.ID)
		require.NoError(t, err)
		require.NotNil(t, found.LastSentAt)
		assert.Equal(t, sentAt.Unix(), found.LastSentAt.Unix())
	})
}
