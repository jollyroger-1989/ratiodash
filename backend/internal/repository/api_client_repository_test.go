package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/testutil"
)

func TestAPIClientRepository_CRUD(t *testing.T) {
	repo := repository.NewAPIClientRepository(testutil.NewDB(t))

	client := &domain.APIClient{
		Name:      "Mobile App",
		KeyPrefix: "rd_live_abcd12345678",
		KeyHash:   "hash-1",
		Enabled:   true,
	}
	require.NoError(t, repo.Create(client))
	require.NotZero(t, client.ID)

	found, err := repo.FindByID(client.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, client.Name, found.Name)

	byPrefix, err := repo.FindByKeyPrefix(client.KeyPrefix)
	require.NoError(t, err)
	require.NotNil(t, byPrefix)
	assert.Equal(t, client.ID, byPrefix.ID)

	clients, err := repo.FindAll()
	require.NoError(t, err)
	require.Len(t, clients, 1)

	found.Enabled = false
	require.NoError(t, repo.Update(found))

	updated, err := repo.FindByID(found.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.False(t, updated.Enabled)

	require.NoError(t, repo.Delete(found.ID))

	deleted, err := repo.FindByID(found.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestAPIClientRepository_FindByID_NotFound(t *testing.T) {
	repo := repository.NewAPIClientRepository(testutil.NewDB(t))

	client, err := repo.FindByID(999)
	require.NoError(t, err)
	assert.Nil(t, client)
}
