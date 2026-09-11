package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

func setupMultisolverrConfigHandler(t *testing.T) humatest.TestAPI {
	t.Helper()
	db := testutil.NewDB(t)
	repo := repository.NewMultisolverrConfigRepository(db)
	svc := service.NewMultisolverrConfigService(repo)
	h := handler.NewMultisolverrConfigHandler(svc)
	api := testutil.NewAPI(t)
	handler.RegisterMultisolverrConfigRoutes(api, h)
	return api
}

func setupMultisolverrConfigHandlerWithMock(t *testing.T) (humatest.TestAPI, *mocks.MockMultisolverrConfigService) {
	t.Helper()
	svc := mocks.NewMockMultisolverrConfigService(t)
	h := handler.NewMultisolverrConfigHandler(svc)
	api := testutil.NewAPI(t)
	handler.RegisterMultisolverrConfigRoutes(api, h)
	return api, svc
}

func TestMultisolverrConfigHandler_Get(t *testing.T) {
	t.Run("returns the default disabled config", func(t *testing.T) {
		api := setupMultisolverrConfigHandler(t)

		resp := api.Do(http.MethodGet, "/api/v1/settings/multisolverr")

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.MultisolverrConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.False(t, body.Enabled)
	})

	t.Run("returns 500 on unexpected service error", func(t *testing.T) {
		api, svc := setupMultisolverrConfigHandlerWithMock(t)
		svc.EXPECT().Get().Return(nil, errors.New("db error"))

		resp := api.Do(http.MethodGet, "/api/v1/settings/multisolverr")

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestMultisolverrConfigHandler_Update(t *testing.T) {
	t.Run("updates base_url", func(t *testing.T) {
		api := setupMultisolverrConfigHandler(t)

		resp := api.Do(http.MethodPatch, "/api/v1/settings/multisolverr",
			map[string]string{"base_url": "https://solverr.example.com"})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.MultisolverrConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "https://solverr.example.com", body.BaseURL)
	})

	t.Run("returns 422 when enabling without a base_url", func(t *testing.T) {
		api := setupMultisolverrConfigHandler(t)

		resp := api.Do(http.MethodPatch, "/api/v1/settings/multisolverr",
			map[string]any{"enabled": true})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestMultisolverrConfigHandler_Test(t *testing.T) {
	t.Run("returns 204 when the host is reachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)
		api := setupMultisolverrConfigHandler(t)

		resp := api.Do(http.MethodPost, "/api/v1/settings/multisolverr/test",
			map[string]string{"base_url": srv.URL})

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 422 for an invalid base_url", func(t *testing.T) {
		api := setupMultisolverrConfigHandler(t)

		resp := api.Do(http.MethodPost, "/api/v1/settings/multisolverr/test",
			map[string]string{"base_url": "not-a-url"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}
