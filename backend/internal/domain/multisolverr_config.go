package domain

import "time"

// MultisolverrConfig is the singleton configuration for an optional
// multisolverr (https://github.com/jollyroger-1989/multisolverr) proxy.
// When enabled, trackers with UseMultisolverr set route their scraper HTTP
// requests through it via multisolverr's FlareSolverr-compatible API, so a
// tracker sitting behind a WAF/anti-bot challenge can still be scraped.
type MultisolverrConfig struct {
	ID             uint      `json:"id"              gorm:"primaryKey"`
	Enabled        bool      `json:"enabled"         gorm:"not null;default:false"`
	BaseURL        string    `json:"base_url"        gorm:"not null;default:''"`
	TimeoutSeconds int       `json:"timeout_seconds" gorm:"not null;default:60"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UpdateMultisolverrConfigInput carries the fields that may be patched.
// Nil fields are left unchanged.
type UpdateMultisolverrConfigInput struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	BaseURL        *string `json:"base_url,omitempty"`
	TimeoutSeconds *int    `json:"timeout_seconds,omitempty"`
}

// MultisolverrConfigRepository is the persistence abstraction for the
// singleton MultisolverrConfig row. Get creates a default (disabled) row on
// first access so callers never have to handle a "no config yet" case.
type MultisolverrConfigRepository interface {
	Get() (*MultisolverrConfig, error)
	Update(cfg *MultisolverrConfig) error
}

// MultisolverrConfigService is the business-logic abstraction for
// MultisolverrConfig.
type MultisolverrConfigService interface {
	Get() (*MultisolverrConfig, error)
	Update(input UpdateMultisolverrConfigInput) (*MultisolverrConfig, error)
	// Test checks that a multisolverr instance is reachable at baseURL without
	// saving anything. It does not exercise the solving pipeline itself (that
	// could burn a paid solver's quota) — only host reachability.
	Test(baseURL string) error
}
