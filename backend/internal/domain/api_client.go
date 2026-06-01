package domain

import "time"

// APIClient is a machine-to-machine credential used by external apps.
// Only key metadata is exposed; the raw key is returned once at creation time.
type APIClient struct {
	ID         uint       `json:"id"           gorm:"primaryKey"`
	Name       string     `json:"name"         gorm:"not null"`
	KeyPrefix  string     `json:"key_prefix"   gorm:"not null;uniqueIndex"`
	KeyHash    string     `json:"-"            gorm:"column:key_hash;not null;uniqueIndex"`
	Enabled    bool       `json:"enabled"      gorm:"not null;default:true"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateAPIClientInput struct {
	Name string `json:"name" required:"true" minLength:"1"`
}

// APIClientRepository is the persistence abstraction for API clients.
type APIClientRepository interface {
	FindAll() ([]APIClient, error)
	FindByID(id uint) (*APIClient, error)
	FindByKeyPrefix(prefix string) (*APIClient, error)
	Create(client *APIClient) error
	Update(client *APIClient) error
	Delete(id uint) error
}

// APIKeyAuthenticator validates external app API keys.
type APIKeyAuthenticator interface {
	AuthenticateAPIKey(token string) (bool, error)
}

// APIClientService provides API client lifecycle and authentication behavior.
type APIClientService interface {
	APIKeyAuthenticator
	GetAll() ([]APIClient, error)
	Create(input CreateAPIClientInput) (*APIClient, string, error)
	Delete(id uint) error
}
