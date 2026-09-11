package service_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func TestMultisolverrConfigService_Get(t *testing.T) {
	t.Run("reports has_api_key when a key is stored", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1, APIKey: "secret"}, nil)

		cfg, err := service.NewMultisolverrConfigService(repo).Get()

		require.NoError(t, err)
		assert.True(t, cfg.HasAPIKey)
	})

	t.Run("reports no api key when none stored", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1}, nil)

		cfg, err := service.NewMultisolverrConfigService(repo).Get()

		require.NoError(t, err)
		assert.False(t, cfg.HasAPIKey)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(nil, errors.New("db error"))

		_, err := service.NewMultisolverrConfigService(repo).Get()

		assert.Error(t, err)
	})
}

func TestMultisolverrConfigService_Update(t *testing.T) {
	t.Run("updates base_url, api_key and timeout", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1, TimeoutSeconds: 60}, nil)
		repo.EXPECT().Update(mock.MatchedBy(func(cfg *domain.MultisolverrConfig) bool {
			return cfg.BaseURL == "https://solverr.example.com" && cfg.APIKey == "k" && cfg.TimeoutSeconds == 30
		})).Return(nil)

		cfg, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			BaseURL:        ptr("https://solverr.example.com/"),
			APIKey:         ptr("k"),
			TimeoutSeconds: ptr(30),
		})

		require.NoError(t, err)
		assert.True(t, cfg.HasAPIKey)
	})

	t.Run("rejects a non-http(s) base_url", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1}, nil)

		_, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			BaseURL: ptr("ftp://solverr.example.com"),
		})

		assert.Error(t, err)
	})

	t.Run("rejects a non-positive timeout", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1}, nil)

		_, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			TimeoutSeconds: ptr(0),
		})

		assert.Error(t, err)
	})

	t.Run("refuses to enable without a base_url", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1}, nil)

		_, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			Enabled: ptr(true),
		})

		assert.Error(t, err)
	})

	t.Run("enables when a base_url is already stored", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1, BaseURL: "https://solverr.example.com"}, nil)
		repo.EXPECT().Update(mock.Anything).Return(nil)

		cfg, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			Enabled: ptr(true),
		})

		require.NoError(t, err)
		assert.True(t, cfg.Enabled)
	})

	t.Run("propagates repository Get error", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(nil, errors.New("db error"))

		_, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{})

		assert.Error(t, err)
	})

	t.Run("propagates repository Update error", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)
		repo.EXPECT().Get().Return(&domain.MultisolverrConfig{ID: 1}, nil)
		repo.EXPECT().Update(mock.Anything).Return(errors.New("db error"))

		_, err := service.NewMultisolverrConfigService(repo).Update(domain.UpdateMultisolverrConfigInput{
			APIKey: ptr("k"),
		})

		assert.Error(t, err)
	})
}

func TestMultisolverrConfigService_Test(t *testing.T) {
	t.Run("rejects an empty base_url", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)

		err := service.NewMultisolverrConfigService(repo).Test("  ")

		assert.Error(t, err)
	})

	t.Run("rejects a non-http(s) base_url", func(t *testing.T) {
		repo := mocks.NewMockMultisolverrConfigRepository(t)

		err := service.NewMultisolverrConfigService(repo).Test("ftp://solverr.example.com")

		assert.Error(t, err)
	})

	t.Run("succeeds when the host is reachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		repo := mocks.NewMockMultisolverrConfigRepository(t)

		err := service.NewMultisolverrConfigService(repo).Test(srv.URL)

		assert.NoError(t, err)
	})

	t.Run("fails when the host is unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()
		repo := mocks.NewMockMultisolverrConfigRepository(t)

		err := service.NewMultisolverrConfigService(repo).Test(srv.URL)

		assert.Error(t, err)
	})
}
