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

// yggRebornLoginHTML is a minimal login page containing a csrf_token input.
const yggRebornLoginHTML = `<html><body>
<form>
  <input type="hidden" name="csrf_token" value="test-csrf">
</form>
</body></html>`

// yggRebornStatsHTML is a minimal account page matching the production HTML structure:
// a value div followed by a label div with class "mt-1".
const yggRebornStatsHTML = `<html><body>
<div>
  <div>2.00 Go</div>
  <div class="mt-1">Upload</div>
</div>
<div>
  <div>1.00 Go</div>
  <div class="mt-1">Download</div>
</div>
<div>
  <div>2.0</div>
  <div class="mt-1">Ratio</div>
</div>
</body></html>`

func yggRebornServer(t *testing.T, loginHTML, statsHTML string, loginResponseStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, loginHTML)
		case http.MethodPost:
			if loginResponseStatus == http.StatusFound {
				http.Redirect(w, r, "/", http.StatusFound)
			} else {
				w.WriteHeader(loginResponseStatus)
				fmt.Fprint(w, "login failed")
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/account/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, statsHTML)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

func TestYggRebornScraper_Key(t *testing.T) {
	assert.Equal(t, "yggreborn", scraper.NewYggRebornScraper().Key())
}

func TestYggRebornScraper_CredentialFields(t *testing.T) {
	fields := scraper.NewYggRebornScraper().CredentialFields()
	require.Len(t, fields, 2)
	assert.Equal(t, domain.CredentialField{Key: "username", Label: "Email / Username", Type: "text", Required: true}, fields[0])
	assert.Equal(t, domain.CredentialField{Key: "password", Label: "Password", Type: "password", Required: true}, fields[1])
}

func TestYggRebornScraper_Fetch(t *testing.T) {
	t.Run("returns correct stats with Go (GiB) sizes", func(t *testing.T) {
		srv := yggRebornServer(t, yggRebornLoginHTML, yggRebornStatsHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2147483648), stats.Uploaded)   // 2 GiB
		assert.Equal(t, int64(1073741824), stats.Downloaded) // 1 GiB
		assert.InDelta(t, 2.0, stats.Ratio, 0.001)
	})

	t.Run("parses Mo (MiB) sizes", func(t *testing.T) {
		statsHTML := `<html><body>
<div><div>512.00 Mo</div><div class="mt-1">Upload</div></div>
<div><div>256.00 Mo</div><div class="mt-1">Download</div></div>
<div><div>2.0</div><div class="mt-1">Ratio</div></div>
</body></html>`
		srv := yggRebornServer(t, yggRebornLoginHTML, statsHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(536870912), stats.Uploaded)   // 512 MiB
		assert.Equal(t, int64(268435456), stats.Downloaded) // 256 MiB
	})

	t.Run("parses Ko (KiB) sizes", func(t *testing.T) {
		statsHTML := `<html><body>
<div><div>1024.00 Ko</div><div class="mt-1">Upload</div></div>
<div><div>512.00 Ko</div><div class="mt-1">Download</div></div>
<div><div>2.0</div><div class="mt-1">Ratio</div></div>
</body></html>`
		srv := yggRebornServer(t, yggRebornLoginHTML, statsHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1048576), stats.Uploaded)   // 1 MiB
		assert.Equal(t, int64(524288), stats.Downloaded)  // 512 KiB
	})

	t.Run("parses To (TiB) sizes", func(t *testing.T) {
		statsHTML := `<html><body>
<div><div>1.00 To</div><div class="mt-1">Upload</div></div>
<div><div>1.00 To</div><div class="mt-1">Download</div></div>
<div><div>1.0</div><div class="mt-1">Ratio</div></div>
</body></html>`
		srv := yggRebornServer(t, yggRebornLoginHTML, statsHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		stats, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1099511627776), stats.Uploaded)
		assert.Equal(t, int64(1099511627776), stats.Downloaded)
	})

	t.Run("returns error when username is missing", func(t *testing.T) {
		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("returns error when password is missing", func(t *testing.T) {
		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("returns error when credentials JSON is invalid", func(t *testing.T) {
		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: "not-json",
		})

		require.Error(t, err)
	})

	t.Run("returns error when CSRF token not found in login page", func(t *testing.T) {
		// Server returns an empty page with no CSRF token.
		srv := yggRebornServer(t, `<html><body></body></html>`, yggRebornStatsHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "csrf")
	})

	t.Run("returns error when login does not redirect (returns 200)", func(t *testing.T) {
		srv := yggRebornServer(t, yggRebornLoginHTML, yggRebornStatsHTML, http.StatusOK)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"wrong"}`,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("returns error when account page returns error status", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				fmt.Fprint(w, yggRebornLoginHTML)
			} else {
				http.Redirect(w, r, "/", http.StatusFound)
			}
		})
		mux.HandleFunc("/account/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
		srv := httptest.NewServer(mux)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
	})

	t.Run("returns error when account page is missing stat labels", func(t *testing.T) {
		// Page has Upload and Download but not Ratio.
		incompleteHTML := `<html><body>
<div><div>2.00 Go</div><div class="mt-1">Upload</div></div>
<div><div>1.00 Go</div><div class="mt-1">Download</div></div>
</body></html>`
		srv := yggRebornServer(t, yggRebornLoginHTML, incompleteHTML, http.StatusFound)
		defer srv.Close()
		t.Setenv("YGGREBORN_URL", srv.URL)

		s := scraper.NewYggRebornScraper()
		_, err := s.Fetch(context.Background(), domain.Tracker{
			Credentials: `{"username":"user","password":"pass"}`,
		})

		require.Error(t, err)
	})
}
