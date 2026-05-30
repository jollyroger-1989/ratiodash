package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func newAlertConfig(name string) *domain.AlertConfig {
	return &domain.AlertConfig{
		Name:        name,
		AlertType:   domain.AlertTypeRatioAlert,
		Enabled:     true,
		AllTrackers: true,
	}
}

func TestAlertConfigRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no configs exist", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))

		configs, err := repo.FindAll()

		require.NoError(t, err)
		assert.Equal(t, []domain.AlertConfig{}, configs)
	})

	t.Run("returns all created configs", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		require.NoError(t, repo.Create(newAlertConfig("Alpha")))
		require.NoError(t, repo.Create(newAlertConfig("Beta")))

		configs, err := repo.FindAll()

		require.NoError(t, err)
		assert.Len(t, configs, 2)
	})
}

func TestAlertConfigRepository_FindByID(t *testing.T) {
	t.Run("returns config when found", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("Alpha")
		require.NoError(t, repo.Create(cfg))

		found, err := repo.FindByID(cfg.ID)

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Alpha", found.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))

		found, err := repo.FindByID(999)

		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestAlertConfigRepository_FindAllEnabled(t *testing.T) {
	t.Run("returns only enabled configs", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		enabled := newAlertConfig("Enabled")
		enabled.Enabled = true
		disabled := newAlertConfig("Disabled")
		disabled.Enabled = false
		require.NoError(t, repo.Create(enabled))
		require.NoError(t, repo.Create(disabled))

		configs, err := repo.FindAllEnabled()

		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "Enabled", configs[0].Name)
	})

	t.Run("returns empty slice when none enabled", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		disabled := newAlertConfig("Disabled")
		disabled.Enabled = false
		require.NoError(t, repo.Create(disabled))

		configs, err := repo.FindAllEnabled()

		require.NoError(t, err)
		assert.Empty(t, configs)
	})
}

func TestAlertConfigRepository_Create(t *testing.T) {
	t.Run("assigns ID and persists", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("New")

		require.NoError(t, repo.Create(cfg))

		assert.NotZero(t, cfg.ID)
	})
}

func TestAlertConfigRepository_Update(t *testing.T) {
	t.Run("persists changes", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("Original")
		require.NoError(t, repo.Create(cfg))

		cfg.Name = "Updated"
		require.NoError(t, repo.Update(cfg))

		found, err := repo.FindByID(cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", found.Name)
	})
}

func TestAlertConfigRepository_UpdateNotifierConfigs(t *testing.T) {
	t.Run("replaces notifier config associations (empty list)", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("A")
		require.NoError(t, repo.Create(cfg))

		// Passing no IDs should just clear any existing associations.
		err := repo.UpdateNotifierConfigs(cfg.ID, []uint{})

		require.NoError(t, err)
	})
}

func TestAlertConfigRepository_UpdateTrackers(t *testing.T) {
	t.Run("replaces tracker associations (empty list)", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("A")
		require.NoError(t, repo.Create(cfg))

		err := repo.UpdateTrackers(cfg.ID, []uint{})

		require.NoError(t, err)
	})
}

func TestAlertConfigRepository_Delete(t *testing.T) {
	t.Run("deletes existing config", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		cfg := newAlertConfig("ToDelete")
		require.NoError(t, repo.Create(cfg))

		require.NoError(t, repo.Delete(cfg.ID))

		found, err := repo.FindByID(cfg.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))

		err := repo.Delete(999)

		assert.Error(t, err)
	})
}

func TestAlertConfigRepository_SentState(t *testing.T) {
	t.Run("returns false for unknown pair", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))

		sent, err := repo.GetSentState(1, 1)

		require.NoError(t, err)
		assert.False(t, sent)
	})

	t.Run("upserts and retrieves sent state", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))

		require.NoError(t, repo.SetSentState(1, 1, true))

		sent, err := repo.GetSentState(1, 1)
		require.NoError(t, err)
		assert.True(t, sent)
	})

	t.Run("overwrites existing sent state", func(t *testing.T) {
		repo := repository.NewAlertConfigRepository(testutil.NewDB(t))
		require.NoError(t, repo.SetSentState(1, 2, true))

		require.NoError(t, repo.SetSentState(1, 2, false))

		sent, err := repo.GetSentState(1, 2)
		require.NoError(t, err)
		assert.False(t, sent)
	})
}
