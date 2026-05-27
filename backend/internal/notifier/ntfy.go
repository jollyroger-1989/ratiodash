package notifier

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/pkg/config"
)

// levelPriority maps domain levels to ntfy numeric priorities (1–5).
// https://docs.ntfy.sh/publish/#message-priority
var levelPriority = map[domain.NotificationLevel]string{
	domain.LevelInfo:    "3", // default
	domain.LevelWarning: "4", // high
	domain.LevelError:   "5", // urgent / max
}

type ntfyNotifier struct {
	url    string
	token  string
	client *http.Client
}

// NewNtfyNotifier returns a Notifier that publishes to an ntfy topic.
// Returns (nil, nil) when NTFY_URL is not configured, which disables the backend.
func NewNtfyNotifier(cfg *config.Config) (domain.Notifier, error) {
	if cfg.NtfyURL == "" {
		return nil, nil
	}
	return &ntfyNotifier{
		url:    cfg.NtfyURL,
		token:  cfg.NtfyToken,
		client: &http.Client{},
	}, nil
}

func (n *ntfyNotifier) Notify(ctx context.Context, notif domain.Notification) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, strings.NewReader(notif.Body))
	if err != nil {
		return fmt.Errorf("ntfy notify: build request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", notif.Title)
	req.Header.Set("Priority", levelPriority[notif.Level])

	// ntfy tags appear as emoji/label decorations in the notification.
	// Prepend the event name so the source is always visible.
	tags := append([]string{string(notif.Event)}, notif.Tags...)
	req.Header.Set("Tags", strings.Join(tags, ","))

	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy notify: %w", err)
	}
	defer resp.Body.Close()
	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy notify: unexpected status %d", resp.StatusCode)
	}
	return nil
}
