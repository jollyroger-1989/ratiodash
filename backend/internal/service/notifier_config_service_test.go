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

func TestNotifierConfigService_GetAll(t *testing.T) {
	t.Run("returns configs with public config populated", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindAll().Return([]domain.NotifierConfig{
			{ID: 1, Name: "Ntfy", Type: "ntfy", Config: `{"url":"https://ntfy.sh/test","token":"secret"}`},
		}, nil)

		cfgs, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetAll()

		require.NoError(t, err)
		require.Len(t, cfgs, 1)
		assert.Equal(t, map[string]string{"url": "https://ntfy.sh/test"}, cfgs[0].PublicConfig)
	})

	t.Run("returns empty slice when none exist", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindAll().Return([]domain.NotifierConfig{}, nil)

		cfgs, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetAll()

		require.NoError(t, err)
		assert.Empty(t, cfgs)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindAll().Return(nil, errors.New("db error"))

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetAll()

		assert.Error(t, err)
	})
}

func TestNotifierConfigService_GetByID(t *testing.T) {
	t.Run("returns config with public config populated", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(
			&domain.NotifierConfig{ID: 1, Name: "Ntfy", Type: "ntfy", Config: `{"url":"https://ntfy.sh/test","token":"secret"}`},
			nil,
		)

		cfg, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetByID(1)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "Ntfy", cfg.Name)
		assert.Equal(t, map[string]string{"url": "https://ntfy.sh/test"}, cfg.PublicConfig)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetByID(99)

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetByID(1)

		assert.Error(t, err)
	})
}

func TestNotifierConfigService_GetEnabled(t *testing.T) {
	t.Run("returns enabled configs with public config populated", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindEnabled().Return([]domain.NotifierConfig{
			{ID: 1, Name: "Ntfy", Type: "ntfy", Enabled: true, Config: `{"url":"https://ntfy.sh/test","token":"secret"}`},
		}, nil)

		cfgs, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetEnabled()

		require.NoError(t, err)
		require.Len(t, cfgs, 1)
		assert.Equal(t, map[string]string{"url": "https://ntfy.sh/test"}, cfgs[0].PublicConfig)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindEnabled().Return(nil, errors.New("db error"))

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).GetEnabled()

		assert.Error(t, err)
	})
}

func TestNotifierConfigService_Create(t *testing.T) {
	t.Run("creates with provided config and populates public config", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().Create(mock.MatchedBy(func(cfg *domain.NotifierConfig) bool {
			return cfg.Name == "Ntfy" && cfg.Type == "ntfy" &&
				cfg.Config == `{"token":"s","url":"https://ntfy.sh/test"}` && cfg.Enabled
		})).Return(nil)
		mockNotifier := mocks.NewMockNotifier(t)
		mockNotifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mockNotifier, nil)

		cfg, err := service.NewNotifierConfigService(repo, builder).Create(domain.CreateNotifierConfigInput{
			Name:   "Ntfy",
			Type:   "ntfy",
			Config: `{"token":"s","url":"https://ntfy.sh/test"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, "Ntfy", cfg.Name)
		assert.Equal(t, map[string]string{"url": "https://ntfy.sh/test"}, cfg.PublicConfig)
	})

	t.Run("returns error when notification test fails", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		mockNotifier := mocks.NewMockNotifier(t)
		mockNotifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(errors.New("401 Unauthorized"))
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mockNotifier, nil)

		_, err := service.NewNotifierConfigService(repo, builder).Create(domain.CreateNotifierConfigInput{
			Name:   "Ntfy",
			Type:   "ntfy",
			Config: `{"url":"https://ntfy.jlfr.i/ratiodash"}`,
		})

		assert.ErrorContains(t, err, "notification test failed")
	})

	t.Run("rejects create when required config fields are missing", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build("ntfy", "{}").Return(nil, errors.New(`ntfy: missing required field "url"`))

		_, err := service.NewNotifierConfigService(repo, builder).Create(domain.CreateNotifierConfigInput{
			Name: "Ntfy",
			Type: "ntfy",
		})

		assert.ErrorContains(t, err, "invalid config")
	})

	t.Run("returns error for unknown notifier type", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Create(domain.CreateNotifierConfigInput{
			Name: "Bad",
			Type: "unknown",
		})

		assert.ErrorContains(t, err, "unknown notifier type")
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))
		mockNotifier := mocks.NewMockNotifier(t)
		mockNotifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mockNotifier, nil)

		_, err := service.NewNotifierConfigService(repo, builder).Create(domain.CreateNotifierConfigInput{
			Name:   "Ntfy",
			Type:   "ntfy",
			Config: `{"url":"https://ntfy.sh/test"}`,
		})

		assert.Error(t, err)
	})
}

func TestNotifierConfigService_Update(t *testing.T) {
	existing := func() *domain.NotifierConfig {
		return &domain.NotifierConfig{
			ID:      1,
			Name:    "Old Name",
			Type:    "ntfy",
			Config:  `{"url":"https://ntfy.sh/old","token":"oldsecret"}`,
			Enabled: true,
		}
	}

	t.Run("updates name only", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(existing(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(cfg *domain.NotifierConfig) bool {
			return cfg.Name == "New Name"
		})).Return(nil)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mocks.NewMockNotifier(t), nil)

		cfg, err := service.NewNotifierConfigService(repo, builder).Update(1, domain.UpdateNotifierConfigInput{
			Name: ptr("New Name"),
		})

		require.NoError(t, err)
		assert.Equal(t, "New Name", cfg.Name)
	})

	t.Run("merges config preserving omitted sensitive fields", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(existing(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(cfg *domain.NotifierConfig) bool {
			// url changed, token preserved from existing
			return cfg.Config == `{"token":"oldsecret","url":"https://ntfy.sh/new"}`
		})).Return(nil)
		mockNotifier := mocks.NewMockNotifier(t)
		mockNotifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mockNotifier, nil)

		cfg, err := service.NewNotifierConfigService(repo, builder).Update(1, domain.UpdateNotifierConfigInput{
			Config: ptr(`{"url":"https://ntfy.sh/new"}`),
		})

		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("updates enabled flag", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(existing(), nil)
		repo.EXPECT().Update(mock.MatchedBy(func(cfg *domain.NotifierConfig) bool {
			return !cfg.Enabled
		})).Return(nil)
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mocks.NewMockNotifier(t), nil)

		cfg, err := service.NewNotifierConfigService(repo, builder).Update(1, domain.UpdateNotifierConfigInput{
			Enabled: ptr(false),
		})

		require.NoError(t, err)
		assert.False(t, cfg.Enabled)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Update(99, domain.UpdateNotifierConfigInput{})

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates repository find error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Update(1, domain.UpdateNotifierConfigInput{})

		assert.Error(t, err)
	})

	t.Run("propagates repository update error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(existing(), nil)
		repo.EXPECT().Update(mock.Anything).Return(errors.New("db error"))
		builder := mocks.NewMockNotifierBuilder(t)
		builder.EXPECT().Build(mock.Anything, mock.Anything).Return(mocks.NewMockNotifier(t), nil)

		_, err := service.NewNotifierConfigService(repo, builder).Update(1, domain.UpdateNotifierConfigInput{
			Name: ptr("Changed"),
		})

		assert.Error(t, err)
	})

	t.Run("returns error when existing config JSON is invalid", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		bad := existing()
		bad.Config = "not-json"
		repo.EXPECT().FindByID(uint(1)).Return(bad, nil)

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Update(1, domain.UpdateNotifierConfigInput{
			Config: ptr(`{"url":"https://ntfy.sh/new"}`),
		})

		assert.ErrorContains(t, err, "parsing existing config")
	})

	t.Run("returns error when incoming config JSON is invalid", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(existing(), nil)

		_, err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Update(1, domain.UpdateNotifierConfigInput{
			Config: ptr("not-json"),
		})

		assert.ErrorContains(t, err, "parsing incoming config")
	})
}

func TestNotifierConfigService_Delete(t *testing.T) {
	t.Run("deletes existing config", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(
			&domain.NotifierConfig{ID: 1, Name: "Ntfy", Type: "ntfy"},
			nil,
		)
		repo.EXPECT().Delete(uint(1)).Return(nil)

		err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Delete(1)

		require.NoError(t, err)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Delete(99)

		assert.ErrorContains(t, err, "not found")
	})

	t.Run("propagates repository find error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, errors.New("db error"))

		err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Delete(1)

		assert.Error(t, err)
	})

	t.Run("propagates repository delete error", func(t *testing.T) {
		repo := mocks.NewMockNotifierConfigRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(
			&domain.NotifierConfig{ID: 1},
			nil,
		)
		repo.EXPECT().Delete(uint(1)).Return(errors.New("db error"))

		err := service.NewNotifierConfigService(repo, mocks.NewMockNotifierBuilder(t)).Delete(1)

		assert.Error(t, err)
	})
}
