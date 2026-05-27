package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type authRepository struct {
	db *gorm.DB
}

// NewAuthRepository returns a domain.AuthRepository backed by GORM/SQLite.
func NewAuthRepository(db *gorm.DB) domain.AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) Find() (*domain.AppCredential, error) {
	var cred domain.AppCredential
	err := r.db.First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding app credential: %w", err)
	}
	return &cred, nil
}

func (r *authRepository) Create(username, passwordHash, jwtSecret string) (*domain.AppCredential, error) {
	cred := &domain.AppCredential{
		Username:     username,
		PasswordHash: passwordHash,
		JWTSecret:    jwtSecret,
	}
	if err := r.db.Create(cred).Error; err != nil {
		return nil, fmt.Errorf("creating app credential: %w", err)
	}
	return cred, nil
}

func (r *authRepository) Update(id uint, username, passwordHash string) error {
	if err := r.db.Model(&domain.AppCredential{}).Where("id = ?", id).
		Updates(map[string]any{"username": username, "password_hash": passwordHash}).Error; err != nil {
		return fmt.Errorf("updating app credential: %w", err)
	}
	return nil
}
