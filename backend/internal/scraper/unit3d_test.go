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

func TestUnit3DScraper_Key(t *testing.T) {
	assert.Equal(t, "unit3d", scraper.NewUnit3DScraper().Key())
}

func TestUnit3DScraper_CredentialFields(t *testing.T) {
	fields := scraper.NewUnit3DScraper().CredentialFields()
	require.Len(t, fields, 2)
	assert.Equal(t, domain.CredentialField{Key: "url", Label: "Tracker URL", Type: "text", Required: true}, fields[0])
	assert.Equal(t, domain.CredentialField{Key: "token", Label: "API Token", Type: "password", Required: true}, fields[1])
}

func TestUnit3DScraper_Fetch(t *testing.T) {
	t.Run("returns stats from human-readable size strings", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user", r.URL.Path)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uploaded":"1.00 GiB","downloaded":"512.00 MiB","ratio":"2.0"}`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"test-token"}`, srv.URL),
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1073741824), stats.Uploaded)   // 1 GiB
		assert.Equal(t, int64(536870912), stats.Downloaded)  // 512 MiB
		assert.InDelta(t, 2.0, stats.Ratio, 0.001)
	})

	t.Run("parses TiB and PiB sizes", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uploaded":"1.00 TiB","downloaded":"1.00 PiB","ratio":"0.001"}`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1099511627776), stats.Uploaded)   // 1 TiB
		assert.Equal(t, int64(1125899906842624), stats.Downloaded) // 1 PiB
	})

	t.Run("handles zero / bare byte values", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uploaded":"0","downloaded":"512.00 B","ratio":"0"}`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Uploaded)
		assert.Equal(t, int64(512), stats.Downloaded)
		assert.InDelta(t, 0.0, stats.Ratio, 0.001)
	})

	t.Run("returns error when token is missing", func(t *testing.T) {
		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"url":"http://example.com"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("returns error when URL is missing", func(t *testing.T) {
		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"token":"tok"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "url")
	})

	t.Run("returns error when credentials JSON is invalid", func(t *testing.T) {
		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: "not-json",
		})

		require.Error(t, err)
	})

	t.Run("returns error on HTTP 401", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"bad-token"}`, srv.URL),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("returns error on HTTP 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "oops", http.StatusInternalServerError)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.Error(t, err)
	})

	t.Run("returns error on malformed JSON response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `not json`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.Error(t, err)
	})

	t.Run("returns error on malformed size string in uploaded", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uploaded":"bad-size","downloaded":"1.00 GiB","ratio":"1.0"}`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing uploaded")
	})

	t.Run("returns error on unknown size unit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uploaded":"1.00 ZiB","downloaded":"1.00 GiB","ratio":"1.0"}`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.Error(t, err)
	})

	t.Run("returns error when HTML page returned instead of JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body>Login page</body></html>`)
		}))
		defer srv.Close()

		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: fmt.Sprintf(`{"url":%q,"token":"tok"}`, srv.URL),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTML")
	})

	t.Run("returns error for non-http URL scheme", func(t *testing.T) {
		s := scraper.NewUnit3DScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"url":"file:///etc/passwd","token":"tok"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheme")
	})
}
