package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jose/ratiodash/internal/domain"
)

const (
	apiKeyPrefix          = "rd_live_"
	apiKeyRandomBytes     = 32
	apiKeyLookupPrefixLen = 20
)

type apiClientService struct {
	repo domain.APIClientRepository
}

func NewAPIClientService(repo domain.APIClientRepository) *apiClientService {
	return &apiClientService{repo: repo}
}

func (s *apiClientService) GetAll() ([]domain.APIClient, error) {
	return s.repo.FindAll()
}

func (s *apiClientService) Create(input domain.CreateAPIClientInput) (*domain.APIClient, string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}

	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}

	client := &domain.APIClient{
		Name:      name,
		KeyPrefix: rawKey[:apiKeyLookupPrefixLen],
		KeyHash:   hashAPIKey(rawKey),
		Enabled:   true,
	}
	if err := s.repo.Create(client); err != nil {
		return nil, "", err
	}

	return client, rawKey, nil
}

func (s *apiClientService) Delete(id uint) error {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if client == nil {
		return fmt.Errorf("api client %d not found", id)
	}
	return s.repo.Delete(id)
}

func (s *apiClientService) AuthenticateAPIKey(token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" || !strings.HasPrefix(token, apiKeyPrefix) || len(token) < apiKeyLookupPrefixLen {
		return false, nil
	}

	prefix := token[:apiKeyLookupPrefixLen]
	client, err := s.repo.FindByKeyPrefix(prefix)
	if err != nil {
		return false, err
	}
	if client == nil || !client.Enabled {
		return false, nil
	}

	expected := client.KeyHash
	actual := hashAPIKey(token)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
		return false, nil
	}

	now := time.Now()
	client.LastUsedAt = &now
	if err := s.repo.Update(client); err != nil {
		return false, err
	}
	return true, nil
}

func generateAPIKey() (string, error) {
	raw := make([]byte, apiKeyRandomBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating api key: %w", err)
	}
	return apiKeyPrefix + hex.EncodeToString(raw), nil
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
