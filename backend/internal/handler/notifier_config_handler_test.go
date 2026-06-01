package handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/notifier"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

func setupNotifierConfigHandler(t *testing.T) humatest.TestAPI {
	t.Helper()
	db := testutil.NewDB(t)
	repo := repository.NewNotifierConfigRepository(db)
	svc := service.NewNotifierConfigService(repo, notifier.NewFactory())
	authRepo := repository.NewAuthRepository(db)
	h := handler.NewNotifierConfigHandler(svc, authRepo)
	api := testutil.NewAPI(t)
	handler.RegisterNotifierConfigRoutes(api, h)
	return api
}

func setupNotifierConfigHandlerWithMock(t *testing.T) (humatest.TestAPI, *mocks.MockNotifierConfigService) {
	t.Helper()
	svc := mocks.NewMockNotifierConfigService(t)
	h := handler.NewNotifierConfigHandler(svc, nil)
	api := testutil.NewAPI(t)
	handler.RegisterNotifierConfigRoutes(api, h)
	return api, svc
}

// startMockNtfy starts a lightweight HTTP server that accepts any ntfy notification
// and returns its base URL. The server is shut down automatically when the test ends.
func startMockNtfy(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestNotifierConfigHandler_ListNotifierTypes(t *testing.T) {
	t.Run("returns available notifier types", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodGet, "/api/v1/notifier-types")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.NotifierTypeInfo
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.NotEmpty(t, body)
		keys := make([]string, 0, len(body))
		for _, tpe := range body {
			keys = append(keys, tpe.Key)
			assert.NotEmpty(t, tpe.ConfigFields)
		}
		assert.Contains(t, keys, "ntfy")
		assert.Contains(t, keys, "email")
	})
}

func TestNotifierConfigHandler_List(t *testing.T) {
	t.Run("returns empty list when no configs exist", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodGet, "/api/v1/notifier-configs")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns existing configs", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)
		ntfyURL := startMockNtfy(t)
		api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "My Ntfy", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s"}`, ntfyURL)})

		resp := api.Do(http.MethodGet, "/api/v1/notifier-configs")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "My Ntfy", body[0].Name)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		api, svc := setupNotifierConfigHandlerWithMock(t)
		svc.EXPECT().GetAll().Return(nil, errors.New("db error"))

		resp := api.Do(http.MethodGet, "/api/v1/notifier-configs")

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestNotifierConfigHandler_Get(t *testing.T) {
	t.Run("returns config by ID", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)
		ntfyURL := startMockNtfy(t)
		createResp := api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "My Ntfy", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s"}`, ntfyURL)})
		require.Equal(t, http.StatusCreated, createResp.Code)

		resp := api.Do(http.MethodGet, "/api/v1/notifier-configs/1")

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "My Ntfy", body.Name)
		assert.Equal(t, "ntfy", body.Type)
	})

	t.Run("returns 404 for missing config", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodGet, "/api/v1/notifier-configs/999")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestNotifierConfigHandler_Create(t *testing.T) {
	t.Run("creates config and returns 201 with public config", func(t *testing.T) {
		ntfyURL := startMockNtfy(t)
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "My Ntfy", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s","token":"secret"}`, ntfyURL)})

		require.Equal(t, http.StatusCreated, resp.Code)
		var body domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "My Ntfy", body.Name)
		assert.Equal(t, "ntfy", body.Type)
		assert.True(t, body.Enabled)
		assert.NotZero(t, body.ID)
		assert.Equal(t, map[string]string{"url": ntfyURL}, body.PublicConfig)
	})

	t.Run("returns 422 for unknown notifier type", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "Bad", "type": "unknown", "config": "{}"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestNotifierConfigHandler_Update(t *testing.T) {
	t.Run("updates name", func(t *testing.T) {
		ntfyURL := startMockNtfy(t)
		api := setupNotifierConfigHandler(t)
		api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "Old", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s"}`, ntfyURL)})

		resp := api.Do(http.MethodPatch, "/api/v1/notifier-configs/1",
			map[string]string{"name": "New"})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "New", body.Name)
	})

	t.Run("toggles enabled flag", func(t *testing.T) {
		ntfyURL := startMockNtfy(t)
		api := setupNotifierConfigHandler(t)
		api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "Ntfy", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s"}`, ntfyURL)})

		resp := api.Do(http.MethodPatch, "/api/v1/notifier-configs/1",
			map[string]any{"enabled": false})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.NotifierConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.False(t, body.Enabled)
	})

	t.Run("returns 422 when config does not exist", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodPatch, "/api/v1/notifier-configs/999",
			map[string]string{"name": "X"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestNotifierConfigHandler_Delete(t *testing.T) {
	t.Run("deletes config and returns 204", func(t *testing.T) {
		ntfyURL := startMockNtfy(t)
		api := setupNotifierConfigHandler(t)
		api.Do(http.MethodPost, "/api/v1/notifier-configs",
			map[string]string{"name": "Ntfy", "type": "ntfy", "config": fmt.Sprintf(`{"url":"%s"}`, ntfyURL)})

		resp := api.Do(http.MethodDelete, "/api/v1/notifier-configs/1")

		require.Equal(t, http.StatusNoContent, resp.Code)
		getResp := api.Do(http.MethodGet, "/api/v1/notifier-configs/1")
		assert.Equal(t, http.StatusNotFound, getResp.Code)
	})

	t.Run("returns 404 when config does not exist", func(t *testing.T) {
		api := setupNotifierConfigHandler(t)

		resp := api.Do(http.MethodDelete, "/api/v1/notifier-configs/999")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
