package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/proxy"
)

func TestValidateBaseURL(t *testing.T) {
	t.Run("accepts http and https URLs", func(t *testing.T) {
		assert.NoError(t, proxy.ValidateBaseURL("http://solverr.local:8191"))
		assert.NoError(t, proxy.ValidateBaseURL("https://solverr.example.com"))
	})

	t.Run("rejects non-http(s) schemes", func(t *testing.T) {
		assert.Error(t, proxy.ValidateBaseURL("ftp://solverr.example.com"))
	})

	t.Run("rejects relative or malformed URLs", func(t *testing.T) {
		assert.Error(t, proxy.ValidateBaseURL("not-a-url"))
		assert.Error(t, proxy.ValidateBaseURL("://bad"))
	})
}

func TestClient_Solve(t *testing.T) {
	t.Run("returns the solved page for a GET request", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1", r.URL.Path)
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "request.get", req["cmd"])
			assert.Equal(t, "https://tracker.example.com/stats", req["url"])

			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"solution": map[string]any{
					"url":      "https://tracker.example.com/stats",
					"status":   200,
					"response": "<html>ok</html>",
					"cookies":  []map[string]string{{"name": "session", "value": "abc"}},
				},
			})
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		sr, err := client.Solve(context.Background(), http.MethodGet, "https://tracker.example.com/stats", "", nil)

		require.NoError(t, err)
		assert.Equal(t, "<html>ok</html>", sr.Solution.Response)
		assert.Equal(t, 200, sr.Solution.Status)
	})

	t.Run("forwards postData for POST requests", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "request.post", req["cmd"])
			assert.Equal(t, "user=a&pass=b", req["postData"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":   "ok",
				"solution": map[string]any{"status": 200, "response": ""},
			})
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		_, err := client.Solve(context.Background(), http.MethodPost, "https://tracker.example.com/login", "user=a&pass=b", nil)

		require.NoError(t, err)
	})

	t.Run("rejects unsupported HTTP methods", func(t *testing.T) {
		client := proxy.NewClient(proxy.Config{BaseURL: "http://localhost"})

		_, err := client.Solve(context.Background(), http.MethodDelete, "https://tracker.example.com", "", nil)

		assert.Error(t, err)
	})

	t.Run("returns an error when multisolverr reports a solve failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "message": "no solver available"})
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		_, err := client.Solve(context.Background(), http.MethodGet, "https://tracker.example.com", "", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no solver available")
	})

	t.Run("returns an error on a non-2xx HTTP response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		_, err := client.Solve(context.Background(), http.MethodGet, "https://tracker.example.com", "", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("returns an error on a malformed JSON response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
		_, err := client.Solve(context.Background(), http.MethodGet, "https://tracker.example.com", "", nil)

		assert.Error(t, err)
	})

	t.Run("returns an error when the host is unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL, Timeout: 1 * time.Second})
		_, err := client.Solve(context.Background(), http.MethodGet, "https://tracker.example.com", "", nil)

		assert.Error(t, err)
	})
}

func TestClient_Ping(t *testing.T) {
	t.Run("succeeds for any non-5xx response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL})
		assert.NoError(t, client.Ping(context.Background()))
	})

	t.Run("fails on a 5xx response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL})
		assert.Error(t, client.Ping(context.Background()))
	})

	t.Run("fails when the host is unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		client := proxy.NewClient(proxy.Config{BaseURL: srv.URL})
		assert.Error(t, client.Ping(context.Background()))
	})
}
