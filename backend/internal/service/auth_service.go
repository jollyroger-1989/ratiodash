package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/jose/ratiodash/internal/domain"
)

type authService struct {
	repo domain.AuthRepository
}

// NewAuthService returns a domain.AuthService backed by the given repository.
func NewAuthService(repo domain.AuthRepository) domain.AuthService {
	return &authService{repo: repo}
}

func (s *authService) IsSetup() (bool, error) {
	cred, err := s.repo.Find()
	if err != nil {
		return false, err
	}
	return cred != nil, nil
}

func (s *authService) Setup(username, password string) error {
	existing, err := s.repo.Find()
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("credentials already configured")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return fmt.Errorf("generating jwt secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	_, err = s.repo.Create(username, string(hash), secret)
	return err
}

func (s *authService) Login(username, password string) (string, error) {
	cred, err := s.repo.Find()
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", errors.New("not configured")
	}
	if cred.Username != username {
		return "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(cred.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

func (s *authService) ValidateToken(tokenStr string) (string, error) {
	cred, err := s.repo.Find()
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", errors.New("not configured")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cred.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid subject claim")
	}
	return sub, nil
}

func (s *authService) UpdateCredentials(currentPassword, newUsername, newPassword string) error {
	cred, err := s.repo.Find()
	if err != nil {
		return err
	}
	if cred == nil {
		return errors.New("not configured")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("invalid current password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	return s.repo.Update(cred.ID, newUsername, string(hash))
}
