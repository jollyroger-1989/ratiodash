package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func TestAuthRepository_Find(t *testing.T) {
	t.Run("returns nil when no credentials exist", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))

		cred, err := repo.Find()

		require.NoError(t, err)
		assert.Nil(t, cred)
	})

	t.Run("returns credential after creation", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))
		_, err := repo.Create("admin", "hash", "secret")
		require.NoError(t, err)

		cred, err := repo.Find()

		require.NoError(t, err)
		require.NotNil(t, cred)
		assert.Equal(t, "admin", cred.Username)
		assert.Equal(t, "hash", cred.PasswordHash)
		assert.Equal(t, "secret", cred.JWTSecret)
		assert.Equal(t, "en", cred.Language)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewAuthRepository(db)

		_, err = repo.Find()

		assert.Error(t, err)
	})
}

func TestAuthRepository_Create(t *testing.T) {
	t.Run("assigns ID and returns credential", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))

		cred, err := repo.Create("admin", "hash", "secret")

		require.NoError(t, err)
		assert.NotZero(t, cred.ID)
		assert.Equal(t, "admin", cred.Username)
		assert.Equal(t, "hash", cred.PasswordHash)
		assert.Equal(t, "secret", cred.JWTSecret)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewAuthRepository(db)

		_, err = repo.Create("admin", "hash", "secret")

		assert.Error(t, err)
	})
}

func TestAuthRepository_Update(t *testing.T) {
	t.Run("persists new username and password hash", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))
		created, err := repo.Create("admin", "old_hash", "secret")
		require.NoError(t, err)

		require.NoError(t, repo.Update(created.ID, "newadmin", "new_hash"))

		cred, err := repo.Find()
		require.NoError(t, err)
		require.NotNil(t, cred)
		assert.Equal(t, "newadmin", cred.Username)
		assert.Equal(t, "new_hash", cred.PasswordHash)
	})

	t.Run("does not change the JWT secret", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))
		created, err := repo.Create("admin", "hash", "my-jwt-secret")
		require.NoError(t, err)

		require.NoError(t, repo.Update(created.ID, "admin", "new_hash"))

		cred, err := repo.Find()
		require.NoError(t, err)
		assert.Equal(t, "my-jwt-secret", cred.JWTSecret)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewAuthRepository(db)

		err = repo.Update(1, "admin", "hash")

		assert.Error(t, err)
	})
}

func TestAuthRepository_UpdateLanguage(t *testing.T) {
	t.Run("persists language", func(t *testing.T) {
		repo := repository.NewAuthRepository(testutil.NewDB(t))
		created, err := repo.Create("admin", "hash", "secret")
		require.NoError(t, err)

		require.NoError(t, repo.UpdateLanguage(created.ID, "fr"))

		cred, err := repo.Find()
		require.NoError(t, err)
		require.NotNil(t, cred)
		assert.Equal(t, "fr", cred.Language)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := testutil.NewDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		repo := repository.NewAuthRepository(db)

		err = repo.UpdateLanguage(1, "en")

		assert.Error(t, err)
	})
}
