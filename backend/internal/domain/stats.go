package domain

import "time"

// TrackerStats is a point-in-time snapshot of upload / download / ratio for a Tracker.
type TrackerStats struct {
	ID         uint      `json:"id"          gorm:"primaryKey"`
	TrackerID  uint      `json:"tracker_id"  gorm:"not null;index"`
	Uploaded   int64     `json:"uploaded"`   // bytes
	Downloaded int64     `json:"downloaded"` // bytes
	Ratio      float64   `json:"ratio"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// DashboardEntry pairs a Tracker with its most recent stats snapshot.
// Stats is nil when no data has been fetched yet.
type DashboardEntry struct {
	Tracker Tracker       `json:"tracker"`
	Stats   *TrackerStats `json:"stats"`
}

// StatsRepository is the persistence abstraction for TrackerStats.
type StatsRepository interface {
	FindByTrackerID(trackerID uint, limit int) ([]TrackerStats, error)
	FindLatestByTrackerID(trackerID uint) (*TrackerStats, error)
	// FindLatestAll returns the most recent snapshot for every tracker.
	FindLatestAll() ([]TrackerStats, error)
	// FindNearestAtOrBefore returns the latest snapshot taken at or before t.
	// Returns nil, nil when no such snapshot exists.
	FindNearestAtOrBefore(trackerID uint, t time.Time) (*TrackerStats, error)
	Create(stats *TrackerStats) error
	// Delete removes a single snapshot. Returns ErrNotFound if it does not exist
	// or does not belong to trackerID.
	Delete(statID uint, trackerID uint) error
}

// StatsService is the business-logic abstraction for TrackerStats queries.
type StatsService interface {
	GetHistory(trackerID uint, limit int) ([]TrackerStats, error)
	GetLatest(trackerID uint) (*TrackerStats, error)
	// GetDashboard returns all trackers paired with their latest snapshot (nil if none yet).
	GetDashboard() ([]DashboardEntry, error)
	// DeleteEntry removes a single history snapshot.
	DeleteEntry(statID uint, trackerID uint) error
}
