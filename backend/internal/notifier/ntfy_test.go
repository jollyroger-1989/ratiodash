package notifier_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/notifier"
	"github.com/jose/ratiodash/pkg/config"
)

// capturedRequest holds what the fake ntfy server received.
type capturedRequest struct {
	method  string
	headers http.Header
	body    string
}

// ntfyTestServer starts a local HTTP server that records the first incoming
// request and responds with status. The server is closed when the test ends.
func ntfyTestServer(t *testing.T, status int) (*httptest.Server, *capturedRequest) {
	t.Helper()
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		cap.method = r.Method
		cap.headers = r.Header.Clone()
		cap.body = string(body)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

// buildNtfy creates an ntfyNotifier pointed at url with an optional token.
func buildNtfy(t *testing.T, url, token string) domain.Notifier {
	t.Helper()
	n, err := notifier.NewNtfyNotifier(&config.Config{NtfyURL: url, NtfyToken: token})
	require.NoError(t, err)
	require.NotNil(t, n)
	return n
}

func TestNtfyNotifier_New(t *testing.T) {
	t.Run("returns nil when NTFY_URL is empty", func(t *testing.T) {
		n, err := notifier.NewNtfyNotifier(&config.Config{})

		require.NoError(t, err)
		assert.Nil(t, n)
	})

	t.Run("returns a notifier when NTFY_URL is set", func(t *testing.T) {
		n, err := notifier.NewNtfyNotifier(&config.Config{NtfyURL: "http://ntfy.example.com/my-topic"})

		require.NoError(t, err)
		assert.NotNil(t, n)
	})
}

func TestNtfyNotifier_Notify(t *testing.T) {
	t.Run("sends POST with title and body", func(t *testing.T) {
		srv, cap := ntfyTestServer(t, http.StatusOK)

		err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
			Level: domain.LevelInfo,
			Title: "hello",
			Body:  "world",
		})

		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, cap.method)
		assert.Equal(t, "hello", cap.headers.Get("Title"))
		assert.Equal(t, "world", cap.body)
	})

	priorityCases := []struct {
		level    domain.NotificationLevel
		priority string
	}{
		{domain.LevelInfo, "3"},
		{domain.LevelWarning, "4"},
		{domain.LevelError, "5"},
	}
	for _, tc := range priorityCases {
		t.Run("sets priority "+tc.priority+" for level "+string(tc.level), func(t *testing.T) {
			srv, cap := ntfyTestServer(t, http.StatusOK)

			err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
				Level: tc.level,
			})

			require.NoError(t, err)
			assert.Equal(t, tc.priority, cap.headers.Get("Priority"))
		})
	}

	t.Run("prepends event name to domain tags", func(t *testing.T) {
		srv, cap := ntfyTestServer(t, http.StatusOK)

		err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
			Event: domain.EventSyncError,
			Level: domain.LevelError,
			Tags:  []string{"Alpha", "unit3d"},
		})

		require.NoError(t, err)
		tags := strings.Split(cap.headers.Get("Tags"), ",")
		assert.Equal(t, "sync_error", tags[0])
		assert.Contains(t, tags, "Alpha")
		assert.Contains(t, tags, "unit3d")
	})

	t.Run("tags header contains only event name when Tags is empty", func(t *testing.T) {
		srv, cap := ntfyTestServer(t, http.StatusOK)

		err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
			Event: domain.EventReport,
			Level: domain.LevelInfo,
		})

		require.NoError(t, err)
		assert.Equal(t, "report", cap.headers.Get("Tags"))
	})

	t.Run("sets Authorization header when token is configured", func(t *testing.T) {
		srv, cap := ntfyTestServer(t, http.StatusOK)

		err := buildNtfy(t, srv.URL, "secret-token").Notify(context.Background(), domain.Notification{
			Level: domain.LevelInfo,
		})

		require.NoError(t, err)
		assert.Equal(t, "Bearer secret-token", cap.headers.Get("Authorization"))
	})

	t.Run("omits Authorization header when no token is configured", func(t *testing.T) {
		srv, cap := ntfyTestServer(t, http.StatusOK)

		err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
			Level: domain.LevelInfo,
		})

		require.NoError(t, err)
		assert.Empty(t, cap.headers.Get("Authorization"))
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		srv, _ := ntfyTestServer(t, http.StatusInternalServerError)

		err := buildNtfy(t, srv.URL, "").Notify(context.Background(), domain.Notification{
			Level: domain.LevelError,
		})

		assert.Error(t, err)
	})

	t.Run("returns error when server is unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // shut down before Notify is called

		n, err := notifier.NewNtfyNotifier(&config.Config{NtfyURL: url})
		require.NoError(t, err)
		require.NotNil(t, n)

		err = n.Notify(context.Background(), domain.Notification{Level: domain.LevelInfo})

		assert.Error(t, err)
	})

	t.Run("returns error on cancelled context", func(t *testing.T) {
		srv, _ := ntfyTestServer(t, http.StatusOK)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := buildNtfy(t, srv.URL, "").Notify(ctx, domain.Notification{Level: domain.LevelInfo})

		assert.Error(t, err)
	})

	t.Run("returns error when URL cannot be parsed into a request", func(t *testing.T) {
		// A null byte is a control character rejected by url.ParseRequestURI,
		// so http.NewRequestWithContext returns an error before any dial occurs.
		n, err := notifier.NewNtfyNotifier(&config.Config{NtfyURL: "http://\x00"})
		require.NoError(t, err)
		require.NotNil(t, n)

		err = n.Notify(context.Background(), domain.Notification{Level: domain.LevelInfo})

		assert.Error(t, err)
	})
}
