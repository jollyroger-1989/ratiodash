package scraper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/scraper"
)

func torr9Server(t *testing.T, token string, me map[string]any, loginStatus, meStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if loginStatus != http.StatusOK {
			http.Error(w, "error", loginStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"token":%q}`, token)
	})

	mux.HandleFunc("/api/v1/users/me", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
		if meStatus != http.StatusOK {
			http.Error(w, "error", meStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(me)
	})

	return httptest.NewServer(mux)
}

func TestTorr9Scraper_Key(t *testing.T) {
	assert.Equal(t, "torr9", scraper.NewTorr9Scraper().Key())
}

func TestTorr9Scraper_CredentialFields(t *testing.T) {
	fields := scraper.NewTorr9Scraper().CredentialFields()
	require.Len(t, fields, 2)
	assert.Equal(t, domain.CredentialField{Key: "username", Label: "Username", Type: "text", Required: true}, fields[0])
	assert.Equal(t, domain.CredentialField{Key: "password", Label: "Password", Type: "password", Required: true}, fields[1])
}

func TestTorr9Scraper_Fetch(t *testing.T) {
	t.Run("returns correct stats", func(t *testing.T) {
		me := map[string]any{
			"total_uploaded_bytes":   int64(1073741824),
			"total_downloaded_bytes": int64(536870912),
			"bonus_uploaded":         int64(0),
			"bonus_downloaded":       int64(0),
		}
		srv := torr9Server(t, "jwt-tok", me, http.StatusOK, http.StatusOK)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1073741824), stats.Uploaded)
		assert.Equal(t, int64(536870912), stats.Downloaded)
		assert.InDelta(t, 2.0, stats.Ratio, 0.001)
	})

	t.Run("adds bonus bytes to totals", func(t *testing.T) {
		me := map[string]any{
			"total_uploaded_bytes":   int64(1073741824),
			"total_downloaded_bytes": int64(536870912),
			"bonus_uploaded":         int64(536870912),
			"bonus_downloaded":       int64(536870912),
		}
		srv := torr9Server(t, "jwt-tok", me, http.StatusOK, http.StatusOK)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1610612736), stats.Uploaded)   // 1GiB + 512MiB
		assert.Equal(t, int64(1073741824), stats.Downloaded) // 512MiB + 512MiB
	})

	t.Run("ratio is zero when downloaded is zero", func(t *testing.T) {
		me := map[string]any{
			"total_uploaded_bytes":   int64(1073741824),
			"total_downloaded_bytes": int64(0),
			"bonus_uploaded":         int64(0),
			"bonus_downloaded":       int64(0),
		}
		srv := torr9Server(t, "jwt-tok", me, http.StatusOK, http.StatusOK)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.InDelta(t, 0.0, stats.Ratio, 0.001)
	})

	t.Run("returns error when username is missing", func(t *testing.T) {
		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("returns error when password is missing", func(t *testing.T) {
		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("returns error when credentials JSON is invalid", func(t *testing.T) {
		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: "not-json",
		})

		require.Error(t, err)
	})

	t.Run("returns error when login returns non-200", func(t *testing.T) {
		srv := torr9Server(t, "tok", nil, http.StatusUnauthorized, http.StatusOK)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"wrong"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("returns error when login response contains no token", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{}`)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("returns error when user stats endpoint returns non-200", func(t *testing.T) {
		srv := torr9Server(t, "jwt-tok", nil, http.StatusOK, http.StatusForbidden)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
	})

	t.Run("returns error on invalid JSON from user stats endpoint", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"token":"jwt-tok"}`)
		})
		mux.HandleFunc("/api/v1/users/me", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `not json`)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()
		t.Setenv("TORR9_URL", srv.URL)

		s := scraper.NewTorr9Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
	})
}
