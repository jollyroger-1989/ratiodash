package domain

import (
	"encoding/json"
	"time"
)

// sensitiveCredentialKeys are never included in PublicCredentials.
var sensitiveCredentialKeys = map[string]bool{
	"password": true,
	"token":    true,
	"api_key":  true,
	"cookie":   true,
}

// RedactCredentials parses a credentials JSON blob and returns a map
// containing only the non-sensitive fields (e.g. "url", "username").
func RedactCredentials(raw string) map[string]string {
	if raw == "" || raw == "{}" {
		return nil
	}
	var all map[string]string
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		return nil
	}
	pub := make(map[string]string, len(all))
	for k, v := range all {
		if !sensitiveCredentialKeys[k] {
			pub[k] = v
		}
	}
	if len(pub) == 0 {
		return nil
	}
	return pub
}

// Tracker represents a registered torrent tracker whose stats are being tracked.
type Tracker struct {
	ID            uint       `json:"id"                  gorm:"primaryKey"`
	Name          string     `json:"name"                gorm:"uniqueIndex;not null"`
	ScraperKey    string     `json:"scraper_key"         gorm:"not null"`
	Credentials   string     `json:"-"                   gorm:"not null;default:'{}'"`
	CronExpr      string     `json:"cron_expr"           gorm:"not null;default:'@hourly'"`
	Active        bool       `json:"active"              gorm:"not null;default:true"`
	LastError     string     `json:"last_error"          gorm:"not null;default:''"`
	LastScrapedAt *time.Time `json:"last_scraped_at"     gorm:"default:null"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// PublicCredentials contains non-sensitive credential fields (e.g. "url").
	// It is computed at query time and never persisted.
	PublicCredentials map[string]string `json:"public_credentials,omitempty" gorm:"-"`
}

type CreateTrackerInput struct {
	Name        string `json:"name"`
	ScraperKey  string `json:"scraper_key"`
	Credentials string `json:"credentials"` // tracker-specific JSON blob
	CronExpr    string `json:"cron_expr"`
}

type UpdateTrackerInput struct {
	Name        *string `json:"name,omitempty"`
	Credentials *string `json:"credentials,omitempty"`
	CronExpr    *string `json:"cron_expr,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// TrackerRepository is the persistence abstraction for Tracker entities.
type TrackerRepository interface {
	FindAll() ([]Tracker, error)
	FindByID(id uint) (*Tracker, error)
	FindActive() ([]Tracker, error)
	Create(tracker *Tracker) error
	Update(tracker *Tracker) error
	Delete(id uint) error
	// UpdateScrapeStatus records the outcome of the most recent scrape attempt.
	// lastError is empty on success, non-empty on failure.
	UpdateScrapeStatus(trackerID uint, lastError string) error
}

// TrackerService is the business-logic abstraction for Tracker operations.
type TrackerService interface {
	GetAll() ([]Tracker, error)
	GetByID(id uint) (*Tracker, error)
	GetActive() ([]Tracker, error)
	Create(input CreateTrackerInput) (*Tracker, error)
	Update(id uint, input UpdateTrackerInput) (*Tracker, error)
	Delete(id uint) error
	// Test fetches stats using scraperKey and credentialsJSON without persisting anything.
	Test(scraperKey, credentialsJSON string) error
	// TestByID loads the stored tracker, merges credentialsOverride, and fetches
	// without persisting. Pass an empty string to test with the stored credentials as-is.
	TestByID(id uint, credentialsOverride string) error
}
