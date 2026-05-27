package domain

import "time"

// AppCredential holds the single set of login credentials for RatioDash.
// Only one row ever exists in the app_credentials table.
type AppCredential struct {
	ID           uint      `json:"-"          gorm:"primaryKey"`
	Username     string    `json:"username"   gorm:"not null"`
	PasswordHash string    `json:"-"          gorm:"column:password_hash;not null"`
	JWTSecret    string    `json:"-"          gorm:"column:jwt_secret;not null"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthRepository defines persistence for app credentials.
type AuthRepository interface {
	// Find returns the single credential row, or nil when none exists.
	Find() (*AppCredential, error)
	// Create persists a new credential (must only be called when none exist).
	Create(username, passwordHash, jwtSecret string) (*AppCredential, error)
	// Update persists a changed username and/or password hash.
	Update(id uint, username, passwordHash string) error
}

// AuthService defines business logic for authentication.
type AuthService interface {
	// IsSetup reports whether credentials have been configured.
	IsSetup() (bool, error)
	// Setup stores the first (and only) credential. Returns an error if one already exists.
	Setup(username, password string) error
	// Login verifies the credentials and returns a signed JWT.
	Login(username, password string) (string, error)
	// ValidateToken parses and validates a JWT, returning the subject (username).
	ValidateToken(token string) (string, error)
	// UpdateCredentials changes the stored username and/or password.
	// currentPassword must match the existing hash.
	UpdateCredentials(currentPassword, newUsername, newPassword string) error
}

// SetupInput is the request body for the one-time setup wizard.
type SetupInput struct {
	Username string `json:"username" required:"true" minLength:"1"`
	Password string `json:"password" required:"true" minLength:"8"`
}

// LoginInput is the request body for login.
type LoginInput struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
}
