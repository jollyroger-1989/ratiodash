package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

func TestAPIClientService_CreateAndAuthenticate(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewAPIClientRepository(db)
	svc := service.NewAPIClientService(repo)

	client, apiKey, err := svc.Create(domain.CreateAPIClientInput{Name: "Home Assistant"})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotEmpty(t, apiKey)
	assert.Contains(t, apiKey, "rd_live_")

	stored, err := repo.FindByID(client.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.NotEqual(t, apiKey, stored.KeyHash)
	assert.Nil(t, stored.LastUsedAt)

	ok, err := svc.AuthenticateAPIKey(apiKey)
	require.NoError(t, err)
	assert.True(t, ok)

	storedAfterAuth, err := repo.FindByID(client.ID)
	require.NoError(t, err)
	require.NotNil(t, storedAfterAuth)
	assert.NotNil(t, storedAfterAuth.LastUsedAt)
}

func TestAPIClientService_AuthenticateAPIKey(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewAPIClientRepository(db)
	svc := service.NewAPIClientService(repo)

	client, apiKey, err := svc.Create(domain.CreateAPIClientInput{Name: "Automation"})
	require.NoError(t, err)
	require.NotNil(t, client)

	ok, err := svc.AuthenticateAPIKey(apiKey)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = svc.AuthenticateAPIKey("rd_live_invalid")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAPIClientService_Delete(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewAPIClientRepository(db)
	svc := service.NewAPIClientService(repo)

	client, _, err := svc.Create(domain.CreateAPIClientInput{Name: "Worker"})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(client.ID))

	_, err = svc.GetAll()
	require.NoError(t, err)

	err = svc.Delete(client.ID)
	assert.Error(t, err)
}
