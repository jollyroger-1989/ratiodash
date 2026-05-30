package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

func TestAlertConfigHandler_List(t *testing.T) {
	t.Run("returns empty list", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().GetAll().Return([]domain.AlertConfig{}, nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodGet, "/api/v1/alert-configs")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.AlertConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns existing configs", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().GetAll().Return([]domain.AlertConfig{
			{ID: 1, Name: "Low ratio", AlertType: domain.AlertTypeRatioAlert},
		}, nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodGet, "/api/v1/alert-configs")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.AlertConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "Low ratio", body[0].Name)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().GetAll().Return(nil, errors.New("db error"))
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodGet, "/api/v1/alert-configs")

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestAlertConfigHandler_Get(t *testing.T) {
	t.Run("returns config by ID", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().GetByID(uint(1)).Return(&domain.AlertConfig{ID: 1, Name: "A"}, nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodGet, "/api/v1/alert-configs/1")

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.AlertConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "A", body.Name)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().GetByID(uint(99)).Return(nil, errors.New("alert config 99 not found"))
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodGet, "/api/v1/alert-configs/99")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestAlertConfigHandler_Create(t *testing.T) {
	t.Run("creates config and returns 201", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().Create(domain.CreateAlertConfigInput{
			Name:              "Sync error",
			AlertType:         domain.AlertTypeSyncError,
			Enabled:           true,
			RatioThreshold:    0,
			AllTrackers:       true,
			TrackerIDs:        nil,
			NotifierConfigIDs: nil,
		}).Return(&domain.AlertConfig{ID: 1, Name: "Sync error"}, nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodPost, "/api/v1/alert-configs", map[string]interface{}{
			"name":         "Sync error",
			"alert_type":   domain.AlertTypeSyncError,
			"enabled":      true,
			"all_trackers": true,
		})

		require.Equal(t, http.StatusCreated, resp.Code)
		var body domain.AlertConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Sync error", body.Name)
	})

	t.Run("returns 422 when service fails", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().Create(domain.CreateAlertConfigInput{
			Name:      "X",
			AlertType: "bad_type",
		}).Return(nil, errors.New("invalid alert type"))
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodPost, "/api/v1/alert-configs", map[string]interface{}{
			"name":       "X",
			"alert_type": "bad_type",
		})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestAlertConfigHandler_Update(t *testing.T) {
	t.Run("updates and returns config", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		newName := "Renamed"
		svc.EXPECT().Update(uint(1), domain.UpdateAlertConfigInput{
			Name: &newName,
		}).Return(&domain.AlertConfig{ID: 1, Name: "Renamed"}, nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodPatch, "/api/v1/alert-configs/1", map[string]interface{}{
			"name": "Renamed",
		})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.AlertConfig
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Renamed", body.Name)
	})

	t.Run("returns 422 when service fails", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().Update(uint(1), domain.UpdateAlertConfigInput{}).Return(nil, errors.New("not found"))
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodPatch, "/api/v1/alert-configs/1", map[string]interface{}{})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestAlertConfigHandler_Delete(t *testing.T) {
	t.Run("deletes config and returns 204", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().Delete(uint(1)).Return(nil)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodDelete, "/api/v1/alert-configs/1")

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 404 when service fails", func(t *testing.T) {
		svc := mocks.NewMockAlertConfigService(t)
		svc.EXPECT().Delete(uint(99)).Return(errors.New("not found"))
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		resp := api.Do(http.MethodDelete, "/api/v1/alert-configs/99")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

// Integration test using real SQLite DB to verify the full wiring.
func TestAlertConfigHandler_Integration(t *testing.T) {
	t.Run("full CRUD lifecycle", func(t *testing.T) {
		db := testutil.NewDB(t)
		repo := repository.NewAlertConfigRepository(db)
		svc := service.NewAlertConfigService(repo)
		h := handler.NewAlertConfigHandler(svc)
		api := testutil.NewAPI(t)
		handler.RegisterAlertConfigRoutes(api, h)

		// Create
		createResp := api.Do(http.MethodPost, "/api/v1/alert-configs", map[string]interface{}{
			"name":            "Ratio drop",
			"alert_type":      domain.AlertTypeRatioAlert,
			"enabled":         true,
			"ratio_threshold": 1.5,
			"all_trackers":    true,
		})
		require.Equal(t, http.StatusCreated, createResp.Code)
		var created domain.AlertConfig
		require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
		assert.Equal(t, "Ratio drop", created.Name)
		assert.Equal(t, 1.5, created.RatioThreshold)
		assert.NotZero(t, created.ID)

		// Get
		getResp := api.Do(http.MethodGet, "/api/v1/alert-configs/1")
		require.Equal(t, http.StatusOK, getResp.Code)

		// List
		listResp := api.Do(http.MethodGet, "/api/v1/alert-configs")
		require.Equal(t, http.StatusOK, listResp.Code)
		var configs []domain.AlertConfig
		require.NoError(t, json.NewDecoder(listResp.Body).Decode(&configs))
		require.Len(t, configs, 1)

		// Update
		updateResp := api.Do(http.MethodPatch, "/api/v1/alert-configs/1", map[string]interface{}{
			"name": "Updated name",
		})
		require.Equal(t, http.StatusOK, updateResp.Code)
		var updated domain.AlertConfig
		require.NoError(t, json.NewDecoder(updateResp.Body).Decode(&updated))
		assert.Equal(t, "Updated name", updated.Name)

		// Delete
		deleteResp := api.Do(http.MethodDelete, "/api/v1/alert-configs/1")
		assert.Equal(t, http.StatusNoContent, deleteResp.Code)

		// Verify gone
		listResp2 := api.Do(http.MethodGet, "/api/v1/alert-configs")
		var configs2 []domain.AlertConfig
		require.NoError(t, json.NewDecoder(listResp2.Body).Decode(&configs2))
		assert.Empty(t, configs2)
	})
}
