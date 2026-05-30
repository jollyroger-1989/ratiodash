package scraper_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/scraper"
)

// c411Server builds a test server that simulates the full C411 auth flow.
// csrfToken is returned via X-Csrf-Token header (simplest extraction path).
// loginSuccess controls whether POST /api/auth/login returns {success:true}.
// statsStatus is the HTTP status returned by GET /api/auth/me.
func c411Server(t *testing.T, csrfToken string, loginSuccess bool, statsStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if csrfToken != "" {
			w.Header().Set("X-Csrf-Token", csrfToken)
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body></body></html>`)
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !loginSuccess {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success":false}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true}`)
	})

	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if statsStatus != http.StatusOK {
			http.Error(w, "error", statsStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"user":{"uploaded":2147483648,"downloaded":1073741824,"ratio":2.0}}`)
	})

	return httptest.NewServer(mux)
}

func TestC411Scraper_Key(t *testing.T) {
	assert.Equal(t, "c411", scraper.NewC411Scraper().Key())
}

func TestC411Scraper_CredentialFields(t *testing.T) {
	fields := scraper.NewC411Scraper().CredentialFields()
	require.Len(t, fields, 2)
	assert.Equal(t, domain.CredentialField{Key: "username", Label: "Username", Type: "text", Required: true}, fields[0])
	assert.Equal(t, domain.CredentialField{Key: "password", Label: "Password", Type: "password", Required: true}, fields[1])
}

func TestC411Scraper_Fetch(t *testing.T) {
	t.Run("returns stats for valid credentials", func(t *testing.T) {
		srv := c411Server(t, "csrf-abc", true, http.StatusOK)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2147483648), stats.Uploaded)
		assert.Equal(t, int64(1073741824), stats.Downloaded)
		assert.InDelta(t, 2.0, stats.Ratio, 0.001)
	})

	t.Run("returns error when username is missing", func(t *testing.T) {
		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("returns error when password is missing", func(t *testing.T) {
		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("returns error when credentials JSON is invalid", func(t *testing.T) {
		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: "not-json",
		})

		require.Error(t, err)
	})

	t.Run("returns error when CSRF token not found", func(t *testing.T) {
		// Server returns login page with no CSRF token anywhere.
		srv := c411Server(t, "" /* no token */, true, http.StatusOK)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSRF")
	})

	t.Run("returns error when login returns success:false", func(t *testing.T) {
		srv := c411Server(t, "csrf-xyz", false /* loginSuccess=false */, http.StatusOK)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"wrong"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("returns error when login endpoint returns non-200", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Csrf-Token", "csrf-xyz")
			fmt.Fprint(w, `<html></html>`)
		})
		mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("returns error when stats endpoint returns non-200", func(t *testing.T) {
		srv := c411Server(t, "csrf-abc", true, http.StatusForbidden)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("returns error on invalid JSON from stats endpoint", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Csrf-Token", "csrf-abc")
			fmt.Fprint(w, `<html></html>`)
		})
		mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success":true}`)
		})
		mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `not json`)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()
		t.Setenv("C411_URL", srv.URL)

		s := scraper.NewC411Scraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
	})
}
