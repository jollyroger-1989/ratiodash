package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/jose/ratiodash/internal/domain"
)

type apiClientRepository struct {
	db *gorm.DB
}

func NewAPIClientRepository(db *gorm.DB) domain.APIClientRepository {
	return &apiClientRepository{db: db}
}

func (r *apiClientRepository) FindAll() ([]domain.APIClient, error) {
	clients := []domain.APIClient{}
	if err := r.db.Order("id ASC").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("finding all api clients: %w", err)
	}
	if clients == nil {
		clients = []domain.APIClient{}
	}
	return clients, nil
}

func (r *apiClientRepository) FindByID(id uint) (*domain.APIClient, error) {
	var client domain.APIClient
	err := r.db.First(&client, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding api client %d: %w", id, err)
	}
	return &client, nil
}

func (r *apiClientRepository) FindByKeyPrefix(prefix string) (*domain.APIClient, error) {
	var client domain.APIClient
	err := r.db.Where("key_prefix = ?", prefix).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding api client by prefix %q: %w", prefix, err)
	}
	return &client, nil
}

func (r *apiClientRepository) Create(client *domain.APIClient) error {
	if err := r.db.Create(client).Error; err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}
	return nil
}

func (r *apiClientRepository) Update(client *domain.APIClient) error {
	if err := r.db.Save(client).Error; err != nil {
		return fmt.Errorf("updating api client %d: %w", client.ID, err)
	}
	return nil
}

func (r *apiClientRepository) Delete(id uint) error {
	if err := r.db.Delete(&domain.APIClient{}, id).Error; err != nil {
		return fmt.Errorf("deleting api client %d: %w", id, err)
	}
	return nil
}
