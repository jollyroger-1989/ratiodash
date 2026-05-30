package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func newAlertService(t *testing.T, repo *mocks.MockAlertConfigRepository, builder *mocks.MockNotifierBuilder) domain.AlertService {
	t.Helper()
	return service.NewAlertService(repo, builder)
}

func TestAlertService_Process_RepoError(t *testing.T) {
	t.Run("returns nil when FindAllEnabled fails (best-effort)", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		repo.EXPECT().FindAllEnabled().Return(nil, errors.New("db error"))

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 1}, nil, nil)

		assert.NoError(t, err)
	})
}

func TestAlertService_Process_TrackerFiltering(t *testing.T) {
	t.Run("skips config not covering the tracker", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		// Config covers tracker 99, not tracker 1.
		cfg := domain.AlertConfig{
			ID:          1,
			AlertType:   domain.AlertTypeSyncError,
			Enabled:     true,
			AllTrackers: false,
			Trackers:    []domain.Tracker{{ID: 99}},
		}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 1, Name: "Alpha"}, errors.New("fail"), nil)

		require.NoError(t, err)
		// No GetSentState / dispatch calls expected.
	})

	t.Run("processes config when AllTrackers is true", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{
			ID:          1,
			AlertType:   domain.AlertTypeSyncError,
			Enabled:     true,
			AllTrackers: true,
		}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		// Successful scrape → clears state.
		repo.EXPECT().SetSentState(uint(1), uint(5), false).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 5, Name: "Beta"}, nil, nil)

		require.NoError(t, err)
	})

	t.Run("processes config with explicit tracker ID match", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{
			ID:          2,
			AlertType:   domain.AlertTypeSyncError,
			Enabled:     true,
			AllTrackers: false,
			Trackers:    []domain.Tracker{{ID: 3}},
		}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().SetSentState(uint(2), uint(3), false).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 3, Name: "Gamma"}, nil, nil)

		require.NoError(t, err)
	})
}

func TestAlertService_HandleSyncError(t *testing.T) {
	t.Run("clears sent state on successful scrape (recovery)", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{ID: 1, AlertType: domain.AlertTypeSyncError, AllTrackers: true}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().SetSentState(uint(1), uint(10), false).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 10, Name: "X"}, nil, nil)

		require.NoError(t, err)
	})

	t.Run("does not dispatch when alert already sent", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{ID: 1, AlertType: domain.AlertTypeSyncError, AllTrackers: true}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(1), uint(10)).Return(true, nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 10, Name: "X"}, errors.New("fetch failed"), nil)

		require.NoError(t, err)
		// No build / notify calls expected.
	})

	t.Run("dispatches and marks sent when first sync error", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		nc := domain.NotifierConfig{ID: 7, Type: "ntfy", Config: `{"url":"http://ntfy.sh/test"}`}
		cfg := domain.AlertConfig{
			ID:              1,
			AlertType:       domain.AlertTypeSyncError,
			AllTrackers:     true,
			NotifierConfigs: []domain.NotifierConfig{nc},
		}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(1), uint(10)).Return(false, nil)
		builder.EXPECT().Build("ntfy", nc.Config).Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return n.Event == domain.EventSyncError &&
				n.Level == domain.LevelError &&
				n.Title == "[RatioDash] Sync failed: X"
		})).Return(nil)
		repo.EXPECT().SetSentState(uint(1), uint(10), true).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 10, Name: "X"}, errors.New("timeout"), nil)

		require.NoError(t, err)
	})

	t.Run("skips notifier when Build fails", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		nc := domain.NotifierConfig{ID: 8, Type: "unknown", Config: "{}"}
		cfg := domain.AlertConfig{
			ID:              1,
			AlertType:       domain.AlertTypeSyncError,
			AllTrackers:     true,
			NotifierConfigs: []domain.NotifierConfig{nc},
		}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(1), uint(10)).Return(false, nil)
		builder.EXPECT().Build("unknown", "{}").Return(nil, errors.New("unknown type"))
		repo.EXPECT().SetSentState(uint(1), uint(10), true).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), &domain.Tracker{ID: 10, Name: "X"}, errors.New("fail"), nil)

		require.NoError(t, err)
	})
}

func TestAlertService_HandleRatioAlert(t *testing.T) {
	tracker := &domain.Tracker{ID: 5, Name: "Delta"}

	t.Run("does nothing when ratio_alert type and stats is nil", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{ID: 1, AlertType: domain.AlertTypeRatioAlert, AllTrackers: true, RatioThreshold: 1.5}
		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), tracker, nil, nil)

		require.NoError(t, err)
		// No GetSentState calls — ratio_alert with nil stats is a no-op.
	})

	t.Run("dispatches when ratio below threshold and not yet sent", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		nc := domain.NotifierConfig{ID: 3, Type: "ntfy", Config: `{}`}
		cfg := domain.AlertConfig{
			ID:              2,
			AlertType:       domain.AlertTypeRatioAlert,
			AllTrackers:     true,
			RatioThreshold:  1.5,
			NotifierConfigs: []domain.NotifierConfig{nc},
		}
		stats := &domain.TrackerStats{Ratio: 0.9}

		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(2), uint(5)).Return(false, nil)
		builder.EXPECT().Build("ntfy", "{}").Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return n.Event == domain.EventRatioAlert &&
				n.Level == domain.LevelWarning &&
				n.Title == "[RatioDash] Low ratio: Delta"
		})).Return(nil)
		repo.EXPECT().SetSentState(uint(2), uint(5), true).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), tracker, nil, stats)

		require.NoError(t, err)
	})

	t.Run("does not dispatch again when ratio below threshold and already sent", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{
			ID:             2,
			AlertType:      domain.AlertTypeRatioAlert,
			AllTrackers:    true,
			RatioThreshold: 1.5,
		}
		stats := &domain.TrackerStats{Ratio: 0.9}

		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(2), uint(5)).Return(true, nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), tracker, nil, stats)

		require.NoError(t, err)
	})

	t.Run("resets sent state when ratio recovers above threshold", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{
			ID:             3,
			AlertType:      domain.AlertTypeRatioAlert,
			AllTrackers:    true,
			RatioThreshold: 1.5,
		}
		stats := &domain.TrackerStats{Ratio: 2.0}

		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(3), uint(5)).Return(true, nil)
		repo.EXPECT().SetSentState(uint(3), uint(5), false).Return(nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), tracker, nil, stats)

		require.NoError(t, err)
	})

	t.Run("does nothing when ratio above threshold and not previously sent", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg := domain.AlertConfig{
			ID:             4,
			AlertType:      domain.AlertTypeRatioAlert,
			AllTrackers:    true,
			RatioThreshold: 1.5,
		}
		stats := &domain.TrackerStats{Ratio: 3.0}

		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg}, nil)
		repo.EXPECT().GetSentState(uint(4), uint(5)).Return(false, nil)

		svc := newAlertService(t, repo, builder)
		err := svc.Process(context.Background(), tracker, nil, stats)

		require.NoError(t, err)
	})

	t.Run("processes multiple configs independently", func(t *testing.T) {
		repo := mocks.NewMockAlertConfigRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		cfg1 := domain.AlertConfig{
			ID:          1,
			AlertType:   domain.AlertTypeSyncError,
			AllTrackers: true,
		}
		cfg2 := domain.AlertConfig{
			ID:             2,
			AlertType:      domain.AlertTypeRatioAlert,
			AllTrackers:    true,
			RatioThreshold: 1.5,
		}

		repo.EXPECT().FindAllEnabled().Return([]domain.AlertConfig{cfg1, cfg2}, nil)
		// cfg1: sync_error with successful scrape → clears state
		repo.EXPECT().SetSentState(uint(1), uint(5), false).Return(nil)
		// cfg2: ratio_alert with good ratio, not previously sent → no-op
		repo.EXPECT().GetSentState(uint(2), uint(5)).Return(false, nil)

		svc := newAlertService(t, repo, builder)
		stats := &domain.TrackerStats{Ratio: 2.5}
		err := svc.Process(context.Background(), tracker, nil, stats)

		require.NoError(t, err)
	})
}
