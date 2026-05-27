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
	// Fetch retrieves the current upload / download / ratio for the given tracker.
	// TrackerID and FetchedAt are set by the caller; the scraper only fills the
	// measurement fields.
	Fetch(ctx context.Context, tracker Tracker) (*TrackerStats, error)
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
