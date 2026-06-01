package domain

import "time"

// NotifierConfigField describes a single configuration input required by a
// notifier type. The frontend uses these definitions to render forms dynamically.
type NotifierConfigField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // "text" | "password" | "url" | "select"
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// NotifierTypeInfo describes a built-in notification backend.
// The frontend queries this list to populate the "add notifier" form.
type NotifierTypeInfo struct {
	Key          string                `json:"key"`
	Label        string                `json:"label"`
	ConfigFields []NotifierConfigField `json:"config_fields"`
}

// AvailableNotifierTypes lists every built-in notifier backend.
// Add a new entry here when adding a new transport implementation.
var AvailableNotifierTypes = []NotifierTypeInfo{
	{
		Key:   "ntfy",
		Label: "ntfy",
		ConfigFields: []NotifierConfigField{
			{Key: "url", Label: "Topic URL", Type: "url", Required: true},
			{Key: "token", Label: "Access Token", Type: "password", Required: false},
		},
	},
	{
		Key:   "email",
		Label: "Email (SMTP)",
		ConfigFields: []NotifierConfigField{
			{Key: "host", Label: "SMTP Host", Type: "text", Required: true},
			{Key: "port", Label: "SMTP Port", Type: "text", Required: true},
			{Key: "from", Label: "From Address", Type: "text", Required: true},
			{Key: "to", Label: "To Address(es)", Type: "text", Required: true},
			{Key: "username", Label: "Username", Type: "text", Required: false},
			{Key: "password", Label: "Password", Type: "password", Required: false},
			{Key: "tls_mode", Label: "TLS Mode", Type: "select", Required: false, Options: []string{"starttls", "tls"}},
		},
	},
}

// NotifierConfig is a persisted configuration for a single notification backend.
// Multiple configs of the same type are allowed (e.g. two ntfy topics).
type NotifierConfig struct {
	ID        uint      `json:"id"         gorm:"primaryKey"`
	Name      string    `json:"name"       gorm:"not null"`
	Type      string    `json:"type"       gorm:"not null"` // matches NotifierTypeInfo.Key
	Config    string    `json:"-"          gorm:"not null;default:'{}'"`
	Enabled   bool      `json:"enabled"    gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// PublicConfig exposes non-sensitive config fields (e.g. "url").
	// Computed at query time; never persisted.
	PublicConfig map[string]string `json:"public_config,omitempty" gorm:"-"`
}

// CreateNotifierConfigInput is the request body for creating a notifier config.
type CreateNotifierConfigInput struct {
	Name   string `json:"name"   required:"true" minLength:"1"`
	Type   string `json:"type"   required:"true" minLength:"1"`
	Config string `json:"config"` // type-specific JSON blob
}

// UpdateNotifierConfigInput carries the fields that may be patched.
// Nil fields are left unchanged.
type UpdateNotifierConfigInput struct {
	Name    *string `json:"name,omitempty"`
	Config  *string `json:"config,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// NotifierBuilder constructs a live Notifier from a type key and a JSON config blob.
// Implementations live in the notifier package and are wired via FX.
type NotifierBuilder interface {
	Build(cfgType, cfgJSON string) (Notifier, error)
}

// NotifierConfigRepository is the persistence abstraction for NotifierConfig.
type NotifierConfigRepository interface {
	FindAll() ([]NotifierConfig, error)
	FindByID(id uint) (*NotifierConfig, error)
	FindEnabled() ([]NotifierConfig, error)
	Create(cfg *NotifierConfig) error
	Update(cfg *NotifierConfig) error
	Delete(id uint) error
}

// NotifierConfigService is the business-logic abstraction for NotifierConfig.
type NotifierConfigService interface {
	GetAll() ([]NotifierConfig, error)
	GetByID(id uint) (*NotifierConfig, error)
	GetEnabled() ([]NotifierConfig, error)
	Create(input CreateNotifierConfigInput) (*NotifierConfig, error)
	Update(id uint, input UpdateNotifierConfigInput) (*NotifierConfig, error)
	Delete(id uint) error
	// Test builds a live notifier from the supplied type and JSON config and
	// sends a test notification. Use TestByID to test a persisted config
	// (which has access to the full, non-redacted credentials).
	Test(cfgType, cfgJSON string) error
	// TestByID loads the persisted config (full credentials), merges it with
	// configOverride (non-empty keys win), and fires a test notification.
	// Pass an empty string to test purely against the stored config.
	TestByID(id uint, configOverride string) error
}
