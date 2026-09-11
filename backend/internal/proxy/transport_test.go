package proxy_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/proxy"
)

func TestTransport_RoundTrip(t *testing.T) {
	t.Run("returns the solved body and status for a GET request", func(t *testing.T) {
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"solution": map[string]any{
					"status":   200,
					"response": "<html>stats</html>",
				},
			})
		}))
		defer solverr.Close()

		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second})}
		resp, err := client.Get("https://tracker.example.com/stats")

		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "<html>stats</html>", string(body))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("feeds returned cookies back into the client's cookie jar", func(t *testing.T) {
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"solution": map[string]any{
					"status":   200,
					"response": "ok",
					"cookies":  []map[string]string{{"name": "session", "value": "abc123"}},
				},
			})
		}))
		defer solverr.Close()

		jar, err := cookiejar.New(nil)
		require.NoError(t, err)
		client := &http.Client{
			Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second}),
			Jar:       jar,
		}

		targetURL, _ := url.Parse("https://tracker.example.com/stats")
		_, err = client.Get(targetURL.String())
		require.NoError(t, err)

		cookies := jar.Cookies(targetURL)
		require.Len(t, cookies, 1)
		assert.Equal(t, "session", cookies[0].Name)
		assert.Equal(t, "abc123", cookies[0].Value)
	})

	t.Run("forwards the request body as postData for POST requests", func(t *testing.T) {
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "user=alice", req["postData"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":   "ok",
				"solution": map[string]any{"status": 200, "response": ""},
			})
		}))
		defer solverr.Close()

		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second})}
		resp, err := client.Post("https://tracker.example.com/login", "application/x-www-form-urlencoded",
			strings.NewReader("user=alice"))

		require.NoError(t, err)
		defer resp.Body.Close()
	})

	t.Run("propagates an error for unsupported HTTP methods", func(t *testing.T) {
		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: "http://localhost"})}

		req, err := http.NewRequest(http.MethodPut, "https://tracker.example.com", nil)
		require.NoError(t, err)

		_, err = client.Do(req)

		assert.Error(t, err)
	})

	t.Run("reports the URL multisolverr's browser actually landed on", func(t *testing.T) {
		// multisolverr's headless browser follows redirects itself and never
		// surfaces a raw 3xx, so callers that need to know a redirect
		// happened (e.g. a login flow whose success signal is "we were sent
		// somewhere else") have to read it off solution.url instead.
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"solution": map[string]any{
					"status":   200,
					"url":      "https://tracker.example.com/",
					"response": "<html>home</html>",
				},
			})
		}))
		defer solverr.Close()

		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second})}
		resp, err := client.Post("https://tracker.example.com/login", "application/x-www-form-urlencoded", strings.NewReader(""))

		require.NoError(t, err)
		defer resp.Body.Close()
		require.NotNil(t, resp.Request)
		require.NotNil(t, resp.Request.URL)
		assert.Equal(t, "https://tracker.example.com/", resp.Request.URL.String())
	})

	t.Run("falls back to the original request URL when solution.url is missing", func(t *testing.T) {
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":   "ok",
				"solution": map[string]any{"status": 200, "response": "<html>login</html>"},
			})
		}))
		defer solverr.Close()

		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second})}
		resp, err := client.Post("https://tracker.example.com/login", "application/x-www-form-urlencoded", strings.NewReader(""))

		require.NoError(t, err)
		defer resp.Body.Close()
		require.NotNil(t, resp.Request)
		require.NotNil(t, resp.Request.URL)
		assert.Equal(t, "https://tracker.example.com/login", resp.Request.URL.String())
	})

	t.Run("propagates the underlying multisolverr error", func(t *testing.T) {
		solverr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer solverr.Close()

		client := &http.Client{Transport: proxy.NewTransport(proxy.Config{BaseURL: solverr.URL, Timeout: 5 * time.Second})}

		_, err := client.Get("https://tracker.example.com")

		assert.Error(t, err)
	})
}
