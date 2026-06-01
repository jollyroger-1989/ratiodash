package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/pkg/config"
)

type fakeAuthService struct {
	validateFn func(string) (string, error)
}

func (f fakeAuthService) IsSetup() (bool, error) {
	return false, nil
}
func (f fakeAuthService) Setup(string, string) error {
	return nil
}
func (f fakeAuthService) Login(string, string) (string, error) {
	return "", nil
}
func (f fakeAuthService) ValidateToken(token string) (string, error) {
	if f.validateFn == nil {
		return "", errors.New("invalid")
	}
	return f.validateFn(token)
}
func (f fakeAuthService) UpdateCredentials(string, string, string) error {
	return nil
}
func (f fakeAuthService) GetLanguage() (string, error) {
	return "en", nil
}
func (f fakeAuthService) UpdateLanguage(string) error {
	return nil
}

func TestJWTMiddleware(t *testing.T) {
	mw := jwtMiddleware(
		fakeAuthService{validateFn: func(token string) (string, error) {
			if token == "jwt-ok" {
				return "admin", nil
			}
			return "", errors.New("invalid")
		}},
	)

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	h := mw(next)

	t.Run("allows public auth routes", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		assert.True(t, nextCalled)
	})

	t.Run("rejects missing bearer header on protected route", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trackers", nil)
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.False(t, nextCalled)
	})

	t.Run("accepts valid jwt", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trackers", nil)
		req.Header.Set("Authorization", "Bearer jwt-ok")
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		assert.True(t, nextCalled)
	})

	t.Run("rejects invalid jwt", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trackers", nil)
		req.Header.Set("Authorization", "Bearer not-valid")
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.False(t, nextCalled)
	})
}

var _ domain.AuthService = fakeAuthService{}

func TestDocsRoute_ServesSwaggerUI(t *testing.T) {
	cfg := &config.Config{
		ServerAddr:     ":8080",
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	router, _ := NewRouter(cfg, fakeAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rr.Body.String(), "id=\"swagger-ui\"")
	assert.Contains(t, rr.Body.String(), "swagger-ui-bundle.js")
	assert.Contains(t, rr.Body.String(), "data-url=\"/openapi.json\"")
}
