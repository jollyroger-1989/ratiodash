package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func newNotifierConfig(name, typ string) *domain.NotifierConfig {
	return &domain.NotifierConfig{
		Name:    name,
		Type:    typ,
		Config:  `{"url":"https://ntfy.sh/test"}`,
		Enabled: true,
	}
}

func TestNotifierConfigRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no configs exist", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))

		cfgs, err := repo.FindAll()

		require.NoError(t, err)
		assert.Equal(t, []domain.NotifierConfig{}, cfgs)
	})

	t.Run("returns all created configs", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		require.NoError(t, repo.Create(newNotifierConfig("Alpha", "ntfy")))
		require.NoError(t, repo.Create(newNotifierConfig("Beta", "ntfy")))

		cfgs, err := repo.FindAll()

		require.NoError(t, err)
		assert.Len(t, cfgs, 2)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		_, err = repo.FindAll()

		assert.Error(t, err)
	})
}

func TestNotifierConfigRepository_FindByID(t *testing.T) {
	t.Run("returns config when found", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		cfg := newNotifierConfig("Alpha", "ntfy")
		require.NoError(t, repo.Create(cfg))

		found, err := repo.FindByID(cfg.ID)

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Alpha", found.Name)
		assert.Equal(t, "ntfy", found.Type)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))

		found, err := repo.FindByID(999)

		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		_, err = repo.FindByID(1)

		assert.Error(t, err)
	})
}

func TestNotifierConfigRepository_FindEnabled(t *testing.T) {
	t.Run("returns only enabled configs", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		enabled := newNotifierConfig("Enabled", "ntfy")
		disabled := newNotifierConfig("Disabled", "ntfy")
		disabled.Enabled = false
		require.NoError(t, repo.Create(enabled))
		require.NoError(t, repo.Create(disabled))

		cfgs, err := repo.FindEnabled()

		require.NoError(t, err)
		require.Len(t, cfgs, 1)
		assert.Equal(t, "Enabled", cfgs[0].Name)
	})

	t.Run("returns empty slice when none are enabled", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		disabled := newNotifierConfig("Disabled", "ntfy")
		disabled.Enabled = false
		require.NoError(t, repo.Create(disabled))

		cfgs, err := repo.FindEnabled()

		require.NoError(t, err)
		assert.Equal(t, []domain.NotifierConfig{}, cfgs)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		_, err = repo.FindEnabled()

		assert.Error(t, err)
	})
}

func TestNotifierConfigRepository_Create(t *testing.T) {
	t.Run("persists and sets ID", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		cfg := newNotifierConfig("Alpha", "ntfy")

		err := repo.Create(cfg)

		require.NoError(t, err)
		assert.NotZero(t, cfg.ID)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		err = repo.Create(newNotifierConfig("Alpha", "ntfy"))

		assert.Error(t, err)
	})
}

func TestNotifierConfigRepository_Update(t *testing.T) {
	t.Run("updates fields", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		cfg := newNotifierConfig("Alpha", "ntfy")
		require.NoError(t, repo.Create(cfg))
		cfg.Name = "Updated"
		cfg.Enabled = false

		err := repo.Update(cfg)

		require.NoError(t, err)
		found, err := repo.FindByID(cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", found.Name)
		assert.False(t, found.Enabled)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		err = repo.Update(&domain.NotifierConfig{ID: 1})

		assert.Error(t, err)
	})
}

func TestNotifierConfigRepository_Delete(t *testing.T) {
	t.Run("removes config from database", func(t *testing.T) {
		repo := repository.NewNotifierConfigRepository(testutil.NewDB(t))
		cfg := newNotifierConfig("Alpha", "ntfy")
		require.NoError(t, repo.Create(cfg))

		err := repo.Delete(cfg.ID)

		require.NoError(t, err)
		found, err := repo.FindByID(cfg.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewNotifierConfigRepository(db)

		err = repo.Delete(1)

		assert.Error(t, err)
	})
}
