package domain

import "context"

// NotificationLevel indicates the urgency / severity of a notification.
// Implementations map this to their own concept (ntfy priority, e-mail subject
// prefix, log level, etc.).
type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "info"
	LevelWarning NotificationLevel = "warning"
	LevelError   NotificationLevel = "error"
)

// NotificationEvent identifies what triggered a notification.
// Clients may filter or route based on this value.
type NotificationEvent string

const (
	// EventReport is fired when a periodic stats summary is dispatched.
	EventReport NotificationEvent = "report"
	// EventRatioAlert is fired when a tracker's ratio drops below a threshold.
	EventRatioAlert NotificationEvent = "ratio_alert"
	// EventSyncError is fired when a scraper fails to fetch stats for a tracker.
	EventSyncError NotificationEvent = "sync_error"
)

// Notification is the transport-agnostic payload delivered by a Notifier.
type Notification struct {
	// Event identifies what triggered this notification.
	Event NotificationEvent
	// Level indicates urgency so backends can adjust presentation (priority,
	// subject prefix, colour, etc.).
	Level NotificationLevel
	// Title is a short, human-readable summary (used as subject or headline).
	Title string
	// Body is the full human-readable message.
	Body string
	// Tags are optional free-form labels (e.g. tracker name, scraper key).
	// Backends may surface them as metadata, filter on them, or ignore them.
	Tags []string
}

// Notifier dispatches a Notification to an external channel.
// Implementations live under internal/notifier/ (ntfy, mail, etc.) and are
// wired via the FX "notifiers" value group into a MultiNotifier.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}
