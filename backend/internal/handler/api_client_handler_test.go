package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

type apiClientTestEnv struct {
	api humatest.TestAPI
}

func setupAPIClientHandler(t *testing.T) apiClientTestEnv {
	t.Helper()
	db := testutil.NewDB(t)
	repo := repository.NewAPIClientRepository(db)
	svc := service.NewAPIClientService(repo)
	h := handler.NewAPIClientHandler(svc)
	api := testutil.NewAPI(t)
	handler.RegisterAPIClientRoutes(api, h)
	return apiClientTestEnv{api: api}
}

func TestAPIClientHandler_CreateAndList(t *testing.T) {
	env := setupAPIClientHandler(t)

	createResp := env.api.Do(http.MethodPost, "/api/v1/api-clients", map[string]string{"name": "External Dashboard"})
	require.Equal(t, http.StatusCreated, createResp.Code)

	var created struct {
		Client struct {
			ID        uint   `json:"id"`
			Name      string `json:"name"`
			KeyPrefix string `json:"key_prefix"`
		} `json:"client"`
		APIKey string `json:"api_key"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	assert.NotZero(t, created.Client.ID)
	assert.Equal(t, "External Dashboard", created.Client.Name)
	assert.NotEmpty(t, created.Client.KeyPrefix)
	assert.NotEmpty(t, created.APIKey)

	listResp := env.api.Do(http.MethodGet, "/api/v1/api-clients")
	require.Equal(t, http.StatusOK, listResp.Code)

	var listed []struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		KeyHash   string `json:"key_hash"`
		APIKey    string `json:"api_key"`
		KeyPrefix string `json:"key_prefix"`
	}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listed))
	require.Len(t, listed, 1)
	assert.Equal(t, created.Client.ID, listed[0].ID)
	assert.Equal(t, "", listed[0].KeyHash)
	assert.Equal(t, "", listed[0].APIKey)
}

func TestAPIClientHandler_Delete(t *testing.T) {
	env := setupAPIClientHandler(t)

	createResp := env.api.Do(http.MethodPost, "/api/v1/api-clients", map[string]string{"name": "Worker"})
	require.Equal(t, http.StatusCreated, createResp.Code)

	var created struct {
		Client struct {
			ID uint `json:"id"`
		} `json:"client"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	deleteResp := env.api.Do(http.MethodDelete, "/api/v1/api-clients/"+itoa(created.Client.ID))
	assert.Equal(t, http.StatusNoContent, deleteResp.Code)

	notFoundResp := env.api.Do(http.MethodDelete, "/api/v1/api-clients/"+itoa(created.Client.ID))
	assert.Equal(t, http.StatusNotFound, notFoundResp.Code)
}

func itoa(v uint) string {
	return fmt.Sprintf("%d", v)
}
