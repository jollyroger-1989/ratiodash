package service

import (
	"context"
	"fmt"

	"github.com/jose/ratiodash/internal/domain"
)

type alertService struct {
	alertConfigRepo domain.AlertConfigRepository
	builder         domain.NotifierBuilder
	authRepo        domain.AuthRepository
}

func NewAlertService(
	alertConfigRepo domain.AlertConfigRepository,
	builder domain.NotifierBuilder,
) domain.AlertService {
	return &alertService{
		alertConfigRepo: alertConfigRepo,
		builder:         builder,
		authRepo:        nil,
	}
}

func NewAlertServiceWithAuthRepo(
	alertConfigRepo domain.AlertConfigRepository,
	builder domain.NotifierBuilder,
	authRepo domain.AuthRepository,
) domain.AlertService {
	return &alertService{
		alertConfigRepo: alertConfigRepo,
		builder:         builder,
		authRepo:        authRepo,
	}
}

// Process checks all enabled alert configs against the outcome of a tracker
// refresh. fetchErr is non-nil when the scrape failed; stats is nil in that
// case. Notification errors are logged but never returned so they cannot
// interrupt the normal scrape flow.
func (s *alertService) Process(ctx context.Context, tracker *domain.Tracker, fetchErr error, stats *domain.TrackerStats) error {
	loc := newNotificationLocalizer(notificationLanguage(s.authRepo))

	configs, err := s.alertConfigRepo.FindAllEnabled()
	if err != nil {
		// Best-effort: log implicitly via the returned ignored error.
		return nil
	}

	for _, cfg := range configs {
		if !s.coversTracker(cfg, tracker.ID) {
			continue
		}
		switch cfg.AlertType {
		case domain.AlertTypeSyncError:
			s.handleSyncError(ctx, cfg, tracker, fetchErr, loc)
		case domain.AlertTypeRatioAlert:
			if stats != nil {
				s.handleRatioAlert(ctx, cfg, tracker, stats, loc)
			}
		}
	}
	return nil
}

// coversTracker returns true when the alert config applies to the given tracker.
func (s *alertService) coversTracker(cfg domain.AlertConfig, trackerID uint) bool {
	if cfg.AllTrackers {
		return true
	}
	for _, t := range cfg.Trackers {
		if t.ID == trackerID {
			return true
		}
	}
	return false
}

func (s *alertService) handleSyncError(ctx context.Context, cfg domain.AlertConfig, tracker *domain.Tracker, fetchErr error, loc notificationLocalizer) {
	if fetchErr == nil {
		// Scrape succeeded — clear any previous alert state (recovery).
		_ = s.alertConfigRepo.SetSentState(cfg.ID, tracker.ID, false)
		return
	}

	sent, _ := s.alertConfigRepo.GetSentState(cfg.ID, tracker.ID)
	if sent {
		// Already alerted for this incident; do not spam.
		return
	}

	notification := domain.Notification{
		Event: domain.EventSyncError,
		Level: domain.LevelError,
		Title: loc.msg("alert.sync_error.title", map[string]any{"TrackerName": tracker.Name}),
		Body:  loc.msg("alert.sync_error.body", map[string]any{"Error": fetchErr.Error()}),
		Tags:  []string{"sync_error", tracker.Name},
	}
	s.dispatch(ctx, cfg, notification)
	_ = s.alertConfigRepo.SetSentState(cfg.ID, tracker.ID, true)
}

func (s *alertService) handleRatioAlert(ctx context.Context, cfg domain.AlertConfig, tracker *domain.Tracker, stats *domain.TrackerStats, loc notificationLocalizer) {
	sent, _ := s.alertConfigRepo.GetSentState(cfg.ID, tracker.ID)

	if stats.Ratio < cfg.RatioThreshold {
		if sent {
			// Already alerted; wait for recovery.
			return
		}
		notification := domain.Notification{
			Event: domain.EventRatioAlert,
			Level: domain.LevelWarning,
			Title: loc.msg("alert.ratio_low.title", map[string]any{"TrackerName": tracker.Name}),
			Body: loc.msg("alert.ratio_low.body", map[string]any{
				"Ratio":     fmt.Sprintf("%.2f", stats.Ratio),
				"Threshold": fmt.Sprintf("%.2f", cfg.RatioThreshold),
			}),
			Tags: []string{"ratio_alert", tracker.Name},
		}
		s.dispatch(ctx, cfg, notification)
		_ = s.alertConfigRepo.SetSentState(cfg.ID, tracker.ID, true)
	} else if sent {
		// Ratio recovered above threshold; reset so the next drop triggers again.
		_ = s.alertConfigRepo.SetSentState(cfg.ID, tracker.ID, false)
	}
}

// dispatch sends a notification to all notifier configs attached to the alert.
func (s *alertService) dispatch(ctx context.Context, cfg domain.AlertConfig, n domain.Notification) {
	for _, nc := range cfg.NotifierConfigs {
		notifier, err := s.builder.Build(nc.Type, nc.Config)
		if err != nil {
			continue
		}
		_ = notifier.Notify(ctx, n)
	}
}
