package handler_test

import (
	"encoding/json"
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

type authTestEnv struct {
	api humatest.TestAPI
}

func setupAuthHandler(t *testing.T) authTestEnv {
	t.Helper()
	db := testutil.NewDB(t)
	repo := repository.NewAuthRepository(db)
	svc := service.NewAuthService(repo)
	h := handler.NewAuthHandler(svc)
	api := testutil.NewAPI(t)
	handler.RegisterAuthRoutes(api, h)
	return authTestEnv{api: api}
}

func TestAuthHandler_Status(t *testing.T) {
	t.Run("returns false when not configured", func(t *testing.T) {
		env := setupAuthHandler(t)

		resp := env.api.Do(http.MethodGet, "/api/v1/auth/status")

		require.Equal(t, http.StatusOK, resp.Code)
		var body struct {
			Setup bool `json:"setup"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.False(t, body.Setup)
	})

	t.Run("returns true after credentials are configured", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodGet, "/api/v1/auth/status")

		require.Equal(t, http.StatusOK, resp.Code)
		var body struct {
			Setup bool `json:"setup"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.True(t, body.Setup)
	})
}

func TestAuthHandler_Setup(t *testing.T) {
	t.Run("creates credentials and returns 204", func(t *testing.T) {
		env := setupAuthHandler(t)

		resp := env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 409 when credentials already exist", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		assert.Equal(t, http.StatusConflict, resp.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("returns JWT on valid credentials", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"username": "admin", "password": "password123"})

		require.Equal(t, http.StatusOK, resp.Code)
		var body struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotEmpty(t, body.Token)
	})

	t.Run("returns 401 on wrong password", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"username": "admin", "password": "wrongpassword"})

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("returns 401 on wrong username", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"username": "wrong", "password": "password123"})

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

func TestAuthHandler_UpdateCredentials(t *testing.T) {
	t.Run("updates credentials with valid current password", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPatch, "/api/v1/settings/credentials",
			map[string]string{"current_password": "password123", "new_username": "newadmin", "new_password": "newpassword123"})

		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Confirm new credentials work
		loginResp := env.api.Do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"username": "newadmin", "password": "newpassword123"})
		assert.Equal(t, http.StatusOK, loginResp.Code)
	})

	t.Run("returns 422 on wrong current password", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPatch, "/api/v1/settings/credentials",
			map[string]string{"current_password": "wrongpassword", "new_username": "newadmin", "new_password": "newpassword123"})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestAuthHandler_SettingsLanguage(t *testing.T) {
	t.Run("returns default language when configured", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodGet, "/api/v1/settings/language")

		require.Equal(t, http.StatusOK, resp.Code)
		var body struct {
			Language string `json:"language"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "en", body.Language)
	})

	t.Run("updates language", func(t *testing.T) {
		env := setupAuthHandler(t)
		env.api.Do(http.MethodPost, "/api/v1/auth/setup",
			map[string]string{"username": "admin", "password": "password123"})

		resp := env.api.Do(http.MethodPatch, "/api/v1/settings/language",
			map[string]string{"language": "fr"})

		assert.Equal(t, http.StatusNoContent, resp.Code)

		readResp := env.api.Do(http.MethodGet, "/api/v1/settings/language")
		require.Equal(t, http.StatusOK, readResp.Code)
		var body struct {
			Language string `json:"language"`
		}
		require.NoError(t, json.NewDecoder(readResp.Body).Decode(&body))
		assert.Equal(t, "fr", body.Language)
	})
}
