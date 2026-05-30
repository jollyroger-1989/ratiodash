package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func newAlertConfigService(t *testing.T, repo *mocks.MockAlertConfigRepository) domain.AlertConfigService {
	t.Helper()
	return service.NewAlertConfigService(repo)
}

func TestAlertConfigService_GetAll(t *testing.T) {
	t.Run("returns all configs from repo", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindAll().Return([]domain.AlertConfig{{ID: 1, Name: "Low ratio"}}, nil)

		configs, err := newAlertConfigService(t, repo).GetAll()

		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "Low ratio", configs[0].Name)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindAll().Return(nil, errors.New("db error"))

		_, err := newAlertConfigService(t, repo).GetAll()

		assert.Error(t, err)
	})
}

func TestAlertConfigService_GetByID(t *testing.T) {
	t.Run("returns config when found", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.AlertConfig{ID: 1, Name: "A"}, nil)

		c, err := newAlertConfigService(t, repo).GetByID(1)

		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, uint(1), c.ID)
	})

	t.Run("returns error when not found (nil from repo)", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := newAlertConfigService(t, repo).GetByID(99)

		assert.ErrorContains(t, err, "alert config 99 not found")
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		_, err := newAlertConfigService(t, repo).GetByID(1)

		assert.Error(t, err)
	})
}

func TestAlertConfigService_Create(t *testing.T) {
	t.Run("defaults threshold to 1.5 when <= 0", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{
			Name:               "Zero threshold",
			AlertType:          domain.AlertTypeRatioAlert,
			Enabled:            true,
			RatioThreshold:     0,
			AllTrackers:        true,
			NotifierConfigIDs:  []uint{},
		}
		repo.EXPECT().Create(mock.MatchedBy(func(c *domain.AlertConfig) bool {
			return c.RatioThreshold == 1.5
		})).Return(nil)
		repo.EXPECT().UpdateNotifierConfigs(mock.AnythingOfType("uint"), []uint{}).Return(nil)
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.AlertConfig{ID: 1, RatioThreshold: 1.5}, nil)

		c, err := newAlertConfigService(t, repo).Create(input)

		require.NoError(t, err)
		assert.Equal(t, 1.5, c.RatioThreshold)
	})

	t.Run("preserves explicit positive threshold", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{
			Name:              "Custom threshold",
			AlertType:         domain.AlertTypeRatioAlert,
			RatioThreshold:    2.0,
			AllTrackers:       true,
			NotifierConfigIDs: []uint{},
		}
		repo.EXPECT().Create(mock.MatchedBy(func(c *domain.AlertConfig) bool {
			return c.RatioThreshold == 2.0
		})).Return(nil)
		repo.EXPECT().UpdateNotifierConfigs(mock.AnythingOfType("uint"), []uint{}).Return(nil)
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.AlertConfig{ID: 1, RatioThreshold: 2.0}, nil)

		c, err := newAlertConfigService(t, repo).Create(input)

		require.NoError(t, err)
		assert.Equal(t, 2.0, c.RatioThreshold)
	})

	t.Run("skips UpdateTrackers when AllTrackers is true", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{
			Name:              "All trackers",
			AlertType:         domain.AlertTypeSyncError,
			AllTrackers:       true,
			RatioThreshold:    1.5,
			NotifierConfigIDs: []uint{},
			TrackerIDs:        []uint{1, 2}, // should be ignored
		}
		repo.EXPECT().Create(mock.Anything).Return(nil)
		repo.EXPECT().UpdateNotifierConfigs(mock.AnythingOfType("uint"), []uint{}).Return(nil)
		// UpdateTrackers must NOT be called.
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.AlertConfig{ID: 1}, nil)

		_, err := newAlertConfigService(t, repo).Create(input)

		require.NoError(t, err)
	})

	t.Run("calls UpdateTrackers when AllTrackers is false", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{
			Name:              "Specific trackers",
			AlertType:         domain.AlertTypeSyncError,
			AllTrackers:       false,
			RatioThreshold:    1.5,
			NotifierConfigIDs: []uint{},
			TrackerIDs:        []uint{3, 4},
		}
		repo.EXPECT().Create(mock.Anything).Return(nil)
		repo.EXPECT().UpdateNotifierConfigs(mock.AnythingOfType("uint"), []uint{}).Return(nil)
		repo.EXPECT().UpdateTrackers(mock.AnythingOfType("uint"), []uint{3, 4}).Return(nil)
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.AlertConfig{ID: 1}, nil)

		_, err := newAlertConfigService(t, repo).Create(input)

		require.NoError(t, err)
	})

	t.Run("returns error when Create fails", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{Name: "X", AlertType: domain.AlertTypeSyncError, RatioThreshold: 1.5}
		repo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))

		_, err := newAlertConfigService(t, repo).Create(input)

		assert.ErrorContains(t, err, "creating alert config")
	})

	t.Run("returns error when UpdateNotifierConfigs fails", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		input := domain.CreateAlertConfigInput{
			Name: "X", AlertType: domain.AlertTypeSyncError, RatioThreshold: 1.5,
			NotifierConfigIDs: []uint{},
		}
		repo.EXPECT().Create(mock.Anything).Return(nil)
		repo.EXPECT().UpdateNotifierConfigs(mock.AnythingOfType("uint"), []uint{}).Return(errors.New("db error"))

		_, err := newAlertConfigService(t, repo).Create(input)

		assert.ErrorContains(t, err, "attaching notifiers")
	})
}

func TestAlertConfigService_Update(t *testing.T) {
	t.Run("returns error when config not found", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := newAlertConfigService(t, repo).Update(99, domain.UpdateAlertConfigInput{})

		assert.ErrorContains(t, err, "alert config 99 not found")
	})

	t.Run("applies partial field updates", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		existing := &domain.AlertConfig{ID: 1, Name: "Old name", Enabled: false, RatioThreshold: 1.0}
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()
		repo.EXPECT().Update(mock.MatchedBy(func(c *domain.AlertConfig) bool {
			return c.Name == "New name" && c.Enabled == true && c.RatioThreshold == 2.5
		})).Return(nil)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.AlertConfig{ID: 1, Name: "New name"}, nil).Once()

		newName := "New name"
		enabled := true
		threshold := 2.5
		input := domain.UpdateAlertConfigInput{
			Name:           &newName,
			Enabled:        &enabled,
			RatioThreshold: &threshold,
		}

		c, err := newAlertConfigService(t, repo).Update(1, input)

		require.NoError(t, err)
		assert.Equal(t, "New name", c.Name)
	})

	t.Run("updates notifier configs when provided", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		existing := &domain.AlertConfig{ID: 1, Name: "A"}
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()
		repo.EXPECT().Update(mock.Anything).Return(nil)
		ids := []uint{5, 6}
		repo.EXPECT().UpdateNotifierConfigs(uint(1), ids).Return(nil)
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()

		_, err := newAlertConfigService(t, repo).Update(1, domain.UpdateAlertConfigInput{
			NotifierConfigIDs: &ids,
		})

		require.NoError(t, err)
	})

	t.Run("updates trackers when provided", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		existing := &domain.AlertConfig{ID: 1, Name: "A"}
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()
		repo.EXPECT().Update(mock.Anything).Return(nil)
		ids := []uint{7, 8}
		repo.EXPECT().UpdateTrackers(uint(1), ids).Return(nil)
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()

		_, err := newAlertConfigService(t, repo).Update(1, domain.UpdateAlertConfigInput{
			TrackerIDs: &ids,
		})

		require.NoError(t, err)
	})

	t.Run("propagates Update error", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.AlertConfig{ID: 1}, nil)
		repo.EXPECT().Update(mock.Anything).Return(errors.New("db error"))

		_, err := newAlertConfigService(t, repo).Update(1, domain.UpdateAlertConfigInput{})

		assert.Error(t, err)
	})
}

func TestAlertConfigService_Delete(t *testing.T) {
	t.Run("delegates to repo", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().Delete(uint(1)).Return(nil)

		err := newAlertConfigService(t, repo).Delete(1)

		require.NoError(t, err)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		repo.EXPECT().Delete(uint(1)).Return(errors.New("not found"))

		err := newAlertConfigService(t, repo).Delete(1)

		assert.Error(t, err)
	})
}
