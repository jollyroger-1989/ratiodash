package domain

import "context"

// CredentialField describes a single credential input that a scraper requires.
// The frontend uses these definitions to render the form dynamically.
type CredentialField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"` // "text" | "password"
	Required bool   `json:"required"`
}

// TrackerScraper is the interface every torrent-site adapter must implement.
// Add new implementations under internal/scraper/ and register them in
// scraper.Module — the registry will pick them up automatically via FX.
type TrackerScraper interface {
	// Key returns the unique string that matches Tracker.ScraperKey (e.g. "ptp", "btn").
	Key() string
	// CredentialFields describes the credentials this scraper needs.
	// The frontend renders a form field for each entry.
	CredentialFields() []CredentialField
	// Deprecated reports whether this scraper should be hidden from the
	// tracker-creation UI. Trackers already using a deprecated scraper keep
	// working; it just can't be selected for new ones.
	Deprecated() bool
	// Fetch retrieves the current upload / download / ratio for the given tracker.
	// TrackerID and FetchedAt are set by the caller; the scraper only fills the
	// measurement fields.
	Fetch(ctx context.Context, tracker Tracker) (*TrackerStats, error)
}

// sessionPersistDisabledKey is the context key used by WithSessionPersistDisabled.
type sessionPersistDisabledKey struct{}

// WithSessionPersistDisabled returns a context instructing scrapers not to
// persist any renewed login session (cookies/auth captures) back to storage.
// Used by TrackerService.Test and TestByID, which validate credentials
// without side effects.
func WithSessionPersistDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionPersistDisabledKey{}, true)
}

// SessionPersistDisabled reports whether ctx disables session persistence.
func SessionPersistDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(sessionPersistDisabledKey{}).(bool)
	return disabled
}

// ScraperRegistry provides access to all registered TrackerScrapers by key.
type ScraperRegistry interface {
	Get(key string) (TrackerScraper, bool)
	Keys() []string
}

// RefreshService orchestrates fetching fresh stats for trackers via the registry.
type RefreshService interface {
	RefreshTracker(ctx context.Context, trackerID uint) error
	RefreshAll(ctx context.Context) error
}

// Refresher manages the live cron schedule of tracker scrapes.
// Implemented by internal/scheduler.
type Refresher interface {
	Schedule(tracker Tracker) error
	Unschedule(trackerID uint)
}
