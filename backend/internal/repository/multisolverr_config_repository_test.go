package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func TestMultisolverrConfigRepository_Get(t *testing.T) {
	t.Run("creates a default disabled row on first access", func(t *testing.T) {
		repo := repository.NewMultisolverrConfigRepository(testutil.NewDB(t))

		cfg, err := repo.Get()

		require.NoError(t, err)
		assert.False(t, cfg.Enabled)
		assert.Equal(t, "", cfg.BaseURL)
		assert.Equal(t, 60, cfg.TimeoutSeconds)
	})

	t.Run("returns the same row on subsequent calls", func(t *testing.T) {
		repo := repository.NewMultisolverrConfigRepository(testutil.NewDB(t))
		first, err := repo.Get()
		require.NoError(t, err)

		second, err := repo.Get()

		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewMultisolverrConfigRepository(db)

		_, err = repo.Get()

		assert.Error(t, err)
	})
}

func TestMultisolverrConfigRepository_Update(t *testing.T) {
	t.Run("persists changes", func(t *testing.T) {
		repo := repository.NewMultisolverrConfigRepository(testutil.NewDB(t))
		cfg, err := repo.Get()
		require.NoError(t, err)

		cfg.Enabled = true
		cfg.BaseURL = "https://solverr.example.com"
		require.NoError(t, repo.Update(cfg))

		reloaded, err := repo.Get()
		require.NoError(t, err)
		assert.True(t, reloaded.Enabled)
		assert.Equal(t, "https://solverr.example.com", reloaded.BaseURL)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewMultisolverrConfigRepository(db)
		cfg, err := repo.Get()
		require.NoError(t, err)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		err = repo.Update(cfg)

		assert.Error(t, err)
	})
}
