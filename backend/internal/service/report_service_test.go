package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

func newReportService(
	t *testing.T,
	repo *mocks.MockReportRepository,
	notifierRepo *mocks.MockNotifierConfigRepository,
	trackers *mocks.MockTrackerService,
	statsRepo *mocks.MockStatsRepository,
	builder *mocks.MockNotifierBuilder,
) domain.ReportService {
	t.Helper()
	return service.NewReportService(repo, notifierRepo, trackers, statsRepo, builder)
}

func TestReportService_GetAll(t *testing.T) {
	t.Run("returns all reports from repo", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindAll().Return([]domain.Report{{ID: 1, Name: "Weekly"}}, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		reports, err := svc.GetAll()

		require.NoError(t, err)
		require.Len(t, reports, 1)
		assert.Equal(t, "Weekly", reports[0].Name)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindAll().Return(nil, errors.New("db error"))

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.GetAll()

		assert.Error(t, err)
	})
}

func TestReportService_GetByID(t *testing.T) {
	t.Run("returns report when found", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Report{ID: 1, Name: "Daily"}, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		r, err := svc.GetByID(1)

		require.NoError(t, err)
		assert.Equal(t, "Daily", r.Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.GetByID(99)

		assert.ErrorContains(t, err, "report 99 not found")
	})
}

func TestReportService_Create(t *testing.T) {
	t.Run("creates report and reloads it", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().Create(mock.MatchedBy(func(r *domain.Report) bool {
			return r.Name == "Monthly" && r.CronExpr == "@monthly"
		})).Return(nil)
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.Report{ID: 1, Name: "Monthly"}, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		r, err := svc.Create(domain.CreateReportInput{
			Name:     "Monthly",
			CronExpr: "@monthly",
		})

		require.NoError(t, err)
		assert.Equal(t, "Monthly", r.Name)
	})

	t.Run("attaches notifier configs by ID", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().Create(mock.MatchedBy(func(r *domain.Report) bool {
			return len(r.NotifierConfigs) == 2 &&
				r.NotifierConfigs[0].ID == 3 &&
				r.NotifierConfigs[1].ID == 7
		})).Return(nil)
		repo.EXPECT().FindByID(mock.AnythingOfType("uint")).Return(&domain.Report{ID: 1}, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.Create(domain.CreateReportInput{
			Name:              "R",
			CronExpr:          "@hourly",
			NotifierConfigIDs: []uint{3, 7},
		})

		require.NoError(t, err)
	})

	t.Run("propagates Create error", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.Create(domain.CreateReportInput{Name: "X", CronExpr: "@hourly"})

		assert.Error(t, err)
	})
}

func TestReportService_Update(t *testing.T) {
	t.Run("returns error when not found", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindByID(uint(99)).Return(nil, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.Update(99, domain.UpdateReportInput{})

		assert.ErrorContains(t, err, "report 99 not found")
	})

	t.Run("applies partial updates and reloads", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		existing := &domain.Report{ID: 1, Name: "Old", CronExpr: "@hourly"}
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()
		repo.EXPECT().Update(mock.MatchedBy(func(r *domain.Report) bool {
			return r.Name == "New" && r.CronExpr == "@daily"
		})).Return(nil)
		repo.EXPECT().FindByID(uint(1)).Return(&domain.Report{ID: 1, Name: "New"}, nil).Once()

		newName := "New"
		newCron := "@daily"
		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		r, err := svc.Update(1, domain.UpdateReportInput{Name: &newName, CronExpr: &newCron})

		require.NoError(t, err)
		assert.Equal(t, "New", r.Name)
	})

	t.Run("updates notifier configs when provided", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		existing := &domain.Report{ID: 1, Name: "R"}
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()
		repo.EXPECT().Update(mock.Anything).Return(nil)
		ids := []uint{10, 11}
		repo.EXPECT().UpdateNotifierConfigs(uint(1), ids).Return(nil)
		repo.EXPECT().FindByID(uint(1)).Return(existing, nil).Once()

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		_, err := svc.Update(1, domain.UpdateReportInput{NotifierConfigIDs: &ids})

		require.NoError(t, err)
	})
}

func TestReportService_Delete(t *testing.T) {
	t.Run("delegates to repo", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().Delete(uint(1)).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		require.NoError(t, svc.Delete(1))
	})
}

func TestReportService_Send(t *testing.T) {
	ctx := context.Background()

	makeReport := func(id uint, name string, notifiers ...domain.NotifierConfig) *domain.Report {
		return &domain.Report{
			ID:              id,
			Name:            name,
			NotifierConfigs: notifiers,
		}
	}

	t.Run("returns error when report not found", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(nil, nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			mocks.NewMockTrackerService(t),
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		err := svc.Send(ctx, 1)

		assert.ErrorContains(t, err, "report 1 not found")
	})

	t.Run("returns error when GetAll trackers fails", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		repo.EXPECT().FindByID(uint(1)).Return(makeReport(1, "R"), nil)
		trackers.EXPECT().GetAll().Return(nil, errors.New("db error"))

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			mocks.NewMockStatsRepository(t),
			mocks.NewMockNotifierBuilder(t),
		)
		err := svc.Send(ctx, 1)

		assert.ErrorContains(t, err, "loading trackers")
	})

	t.Run("returns error when FindLatestAll fails", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		repo.EXPECT().FindByID(uint(1)).Return(makeReport(1, "R"), nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{}, nil)
		statsRepo.EXPECT().FindLatestAll().Return(nil, errors.New("db error"))

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			mocks.NewMockNotifierBuilder(t),
		)
		err := svc.Send(ctx, 1)

		assert.ErrorContains(t, err, "loading latest stats")
	})

	t.Run("sends notification and updates last_sent_at", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		nc := domain.NotifierConfig{ID: 1, Type: "ntfy", Config: `{"url":"http://ntfy.sh/t"}`}
		report := makeReport(1, "Weekly digest", nc)
		tracker := domain.Tracker{ID: 10, Name: "Alpha"}

		repo.EXPECT().FindByID(uint(1)).Return(report, nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{tracker}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{
			{TrackerID: 10, Uploaded: 1024 * 1024 * 1024, Downloaded: 512 * 1024 * 1024, Ratio: 2.0},
		}, nil)
		builder.EXPECT().Build("ntfy", nc.Config).Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return n.Event == domain.EventReport &&
				n.Level == domain.LevelInfo &&
				strings.Contains(n.Title, "Weekly digest") &&
				strings.Contains(n.Body, "Alpha")
		})).Return(nil)
		repo.EXPECT().UpdateLastSentAt(uint(1), mock.AnythingOfType("time.Time")).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			builder,
		)
		err := svc.Send(ctx, 1)

		require.NoError(t, err)
	})

	t.Run("returns combined error on partial notifier failures, still updates last_sent_at", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)

		nc := domain.NotifierConfig{ID: 2, Type: "bad", Config: "{}"}
		report := makeReport(1, "R", nc)

		repo.EXPECT().FindByID(uint(1)).Return(report, nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{}, nil)
		statsRepo.EXPECT().FindLatestAll().Return(nil, nil)
		builder.EXPECT().Build("bad", "{}").Return(nil, errors.New("unknown notifier"))
		repo.EXPECT().UpdateLastSentAt(uint(1), mock.AnythingOfType("time.Time")).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			builder,
		)
		err := svc.Send(ctx, 1)

		assert.ErrorContains(t, err, "report send errors")
		assert.ErrorContains(t, err, "build error")
	})

	t.Run("loads baseline stats when LastSentAt is set", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		lastSent := time.Now().Add(-24 * time.Hour)
		nc := domain.NotifierConfig{ID: 1, Type: "ntfy", Config: `{}`}
		report := &domain.Report{
			ID: 1, Name: "R",
			LastSentAt:      &lastSent,
			NotifierConfigs: []domain.NotifierConfig{nc},
		}
		tracker := domain.Tracker{ID: 5, Name: "Beta"}

		repo.EXPECT().FindByID(uint(1)).Return(report, nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{tracker}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{
			{TrackerID: 5, Uploaded: 2048, Downloaded: 1024, Ratio: 2.0},
		}, nil)
		statsRepo.EXPECT().FindNearestAtOrBefore(uint(5), lastSent).Return(
			&domain.TrackerStats{TrackerID: 5, Uploaded: 1024, Downloaded: 512, Ratio: 2.0}, nil,
		)
		builder.EXPECT().Build("ntfy", "{}").Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return strings.Contains(n.Body, "Evolution since")
		})).Return(nil)
		repo.EXPECT().UpdateLastSentAt(uint(1), mock.AnythingOfType("time.Time")).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			builder,
		)
		err := svc.Send(ctx, 1)

		require.NoError(t, err)
	})

	t.Run("report body includes no-data message for tracker without stats", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		nc := domain.NotifierConfig{ID: 1, Type: "ntfy", Config: `{}`}
		report := makeReport(1, "R", nc)

		repo.EXPECT().FindByID(uint(1)).Return(report, nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{{ID: 99, Name: "NoStats"}}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{}, nil)
		builder.EXPECT().Build("ntfy", "{}").Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return strings.Contains(n.Body, "No data yet")
		})).Return(nil)
		repo.EXPECT().UpdateLastSentAt(uint(1), mock.AnythingOfType("time.Time")).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			builder,
		)
		err := svc.Send(ctx, 1)

		require.NoError(t, err)
	})

	t.Run("report body uses expected evolution symbols", func(t *testing.T) {
		repo := mocks.NewMockReportRepository(t)
		trackers := mocks.NewMockTrackerService(t)
		statsRepo := mocks.NewMockStatsRepository(t)
		builder := mocks.NewMockNotifierBuilder(t)
		notifier := mocks.NewMockNotifier(t)

		lastSent := time.Now().Add(-12 * time.Hour)
		nc := domain.NotifierConfig{ID: 1, Type: "ntfy", Config: `{}`}
		report := &domain.Report{
			ID:              1,
			Name:            "R",
			LastSentAt:      &lastSent,
			NotifierConfigs: []domain.NotifierConfig{nc},
		}
		alpha := domain.Tracker{ID: 1, Name: "Alpha"}
		beta := domain.Tracker{ID: 2, Name: "Beta"}

		repo.EXPECT().FindByID(uint(1)).Return(report, nil)
		trackers.EXPECT().GetAll().Return([]domain.Tracker{alpha, beta}, nil)
		statsRepo.EXPECT().FindLatestAll().Return([]domain.TrackerStats{
			{TrackerID: 1, Uploaded: 2000, Downloaded: 500, Ratio: 4.0},
			{TrackerID: 2, Uploaded: 1000, Downloaded: 1000, Ratio: 1.0},
		}, nil)
		statsRepo.EXPECT().FindNearestAtOrBefore(uint(1), lastSent).Return(
			&domain.TrackerStats{TrackerID: 1, Uploaded: 1000, Downloaded: 1000, Ratio: 1.0}, nil,
		)
		statsRepo.EXPECT().FindNearestAtOrBefore(uint(2), lastSent).Return(
			&domain.TrackerStats{TrackerID: 2, Uploaded: 1000, Downloaded: 1000, Ratio: 1.0}, nil,
		)
		builder.EXPECT().Build("ntfy", "{}").Return(notifier, nil)
		notifier.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(n domain.Notification) bool {
			return strings.Contains(n.Body, "⏫") &&
				strings.Contains(n.Body, "⏬") &&
				strings.Contains(n.Body, "🟰")
		})).Return(nil)
		repo.EXPECT().UpdateLastSentAt(uint(1), mock.AnythingOfType("time.Time")).Return(nil)

		svc := newReportService(t, repo,
			mocks.NewMockNotifierConfigRepository(t),
			trackers,
			statsRepo,
			builder,
		)
		err := svc.Send(ctx, 1)

		require.NoError(t, err)
	})
}
