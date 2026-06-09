package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jose/ratiodash/internal/domain"
)

type reportService struct {
	repo         domain.ReportRepository
	notifierRepo domain.NotifierConfigRepository
	trackers     domain.TrackerService
	statsRepo    domain.StatsRepository
	builder      domain.NotifierBuilder
	authRepo     domain.AuthRepository
}

func NewReportService(
	repo domain.ReportRepository,
	notifierRepo domain.NotifierConfigRepository,
	trackers domain.TrackerService,
	statsRepo domain.StatsRepository,
	builder domain.NotifierBuilder,
) domain.ReportService {
	return &reportService{
		repo:         repo,
		notifierRepo: notifierRepo,
		trackers:     trackers,
		statsRepo:    statsRepo,
		builder:      builder,
		authRepo:     nil,
	}
}

func NewReportServiceWithAuthRepo(
	repo domain.ReportRepository,
	notifierRepo domain.NotifierConfigRepository,
	trackers domain.TrackerService,
	statsRepo domain.StatsRepository,
	builder domain.NotifierBuilder,
	authRepo domain.AuthRepository,
) domain.ReportService {
	return &reportService{
		repo:         repo,
		notifierRepo: notifierRepo,
		trackers:     trackers,
		statsRepo:    statsRepo,
		builder:      builder,
		authRepo:     authRepo,
	}
}

func (s *reportService) GetAll() ([]domain.Report, error) {
	return s.repo.FindAll()
}

func (s *reportService) GetByID(id uint) (*domain.Report, error) {
	r, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("report %d not found", id)
	}
	return r, nil
}

func (s *reportService) Create(input domain.CreateReportInput) (*domain.Report, error) {
	r := &domain.Report{
		Name:     input.Name,
		CronExpr: input.CronExpr,
	}
	// Attach notifier configs.
	for _, id := range input.NotifierConfigIDs {
		r.NotifierConfigs = append(r.NotifierConfigs, domain.NotifierConfig{ID: id})
	}

	if err := s.repo.Create(r); err != nil {
		return nil, err
	}
	// Reload so NotifierConfigs are fully populated.
	return s.repo.FindByID(r.ID)
}

func (s *reportService) Update(id uint, input domain.UpdateReportInput) (*domain.Report, error) {
	r, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("report %d not found", id)
	}

	if input.Name != nil {
		r.Name = *input.Name
	}
	if input.CronExpr != nil {
		r.CronExpr = *input.CronExpr
	}
	if err := s.repo.Update(r); err != nil {
		return nil, err
	}
	if input.NotifierConfigIDs != nil {
		if err := s.repo.UpdateNotifierConfigs(r.ID, *input.NotifierConfigIDs); err != nil {
			return nil, err
		}
	}
	return s.repo.FindByID(r.ID)
}

func (s *reportService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// Send generates and dispatches a report to all configured notifiers.
func (s *reportService) Send(ctx context.Context, id uint) error {
	loc := newNotificationLocalizer(notificationLanguage(s.authRepo))

	r, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("report %d not found", id)
	}

	// Get current stats for all trackers.
	allTrackers, err := s.trackers.GetAll(nil)
	if err != nil {
		return fmt.Errorf("loading trackers: %w", err)
	}

	latestStats, err := s.statsRepo.FindLatestAll()
	if err != nil {
		return fmt.Errorf("loading latest stats: %w", err)
	}
	latestByTracker := make(map[uint]*domain.TrackerStats, len(latestStats))
	for i := range latestStats {
		latestByTracker[latestStats[i].TrackerID] = &latestStats[i]
	}

	// Get baseline stats (at last_sent_at) for evolution computation.
	baselineByTracker := make(map[uint]*domain.TrackerStats)
	if r.LastSentAt != nil {
		for _, t := range allTrackers {
			baseline, err := s.statsRepo.FindNearestAtOrBefore(t.ID, *r.LastSentAt)
			if err != nil {
				return fmt.Errorf("loading baseline stats for tracker %d: %w", t.ID, err)
			}
			if baseline != nil {
				baselineByTracker[t.ID] = baseline
			}
		}
	}

	body := buildReportBody(allTrackers, latestByTracker, baselineByTracker, r.LastSentAt, loc)

	// Send to each configured notifier.
	var errs []string
	for _, cfg := range r.NotifierConfigs {
		n, err := s.builder.Build(cfg.Type, cfg.Config)
		if err != nil {
			errs = append(errs, fmt.Sprintf("notifier %d: build error: %v", cfg.ID, err))
			continue
		}
		notification := domain.Notification{
			Event: domain.EventReport,
			Level: domain.LevelInfo,
			Title: loc.msg("report.title", map[string]any{"ReportName": r.Name}),
			Body:  body,
			Tags:  []string{"report"},
		}
		if err := n.Notify(ctx, notification); err != nil {
			errs = append(errs, fmt.Sprintf("notifier %d: %v", cfg.ID, err))
		}
	}

	// Update last_sent_at even on partial send errors so evolution baseline advances.
	if updateErr := s.repo.UpdateLastSentAt(r.ID, time.Now()); updateErr != nil {
		errs = append(errs, fmt.Sprintf("updating last_sent_at: %v", updateErr))
	}

	if len(errs) > 0 {
		return fmt.Errorf("report send errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// buildReportBody formats the Markdown-style notification body with emojis.
func buildReportBody(
	trackers []domain.Tracker,
	latest map[uint]*domain.TrackerStats,
	baseline map[uint]*domain.TrackerStats,
	lastSentAt *time.Time,
	loc notificationLocalizer,
) string {
	var sb strings.Builder

	if lastSentAt != nil {
		sb.WriteString(loc.msg("report.body.evolution_since", map[string]any{
			"Timestamp": loc.formatTimestamp(*lastSentAt),
		}))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(loc.msg("report.body.first_report", nil))
		sb.WriteString("\n\n")
	}

	// Global aggregate section.
	var totalUploaded, totalDownloaded int64
	var baseUploaded, baseDownloaded int64
	hasData := false
	hasBaseline := false
	for _, t := range trackers {
		if cur := latest[t.ID]; cur != nil {
			totalUploaded += cur.Uploaded
			totalDownloaded += cur.Downloaded
			hasData = true
		}
		if base := baseline[t.ID]; base != nil {
			baseUploaded += base.Uploaded
			baseDownloaded += base.Downloaded
			hasBaseline = true
		}
	}
	if hasData {
		var globalRatio float64
		if totalDownloaded > 0 {
			globalRatio = float64(totalUploaded) / float64(totalDownloaded)
		}
		var baseRatio float64
		if hasBaseline && baseDownloaded > 0 {
			baseRatio = float64(baseUploaded) / float64(baseDownloaded)
		}

		var globalBase *domain.TrackerStats
		if hasBaseline {
			globalBase = &domain.TrackerStats{Uploaded: baseUploaded, Downloaded: baseDownloaded, Ratio: baseRatio}
		}
		globalCur := &domain.TrackerStats{Uploaded: totalUploaded, Downloaded: totalDownloaded, Ratio: globalRatio}

		sb.WriteString(fmt.Sprintf("🌐 %s\n", loc.msg("report.body.global", nil)))
		sb.WriteString(fmt.Sprintf("  ⬆️  %s: %s%s\n",
			loc.msg("report.body.up_label", nil), formatBytes(totalUploaded), formatDelta(globalBase, globalCur, fieldUploaded, loc)))
		sb.WriteString(fmt.Sprintf("  ⬇️  %s: %s%s\n",
			loc.msg("report.body.dl_label", nil), formatBytes(totalDownloaded), formatDelta(globalBase, globalCur, fieldDownloaded, loc)))
		sb.WriteString(fmt.Sprintf("  %s %s: %.2f%s\n",
			ratioEmoji(globalRatio), loc.msg("report.body.ratio_label", nil), globalRatio, formatRatio(globalRatio, globalBase, loc)))
		sb.WriteString("\n")
	}

	for _, t := range trackers {
		cur := latest[t.ID]
		if cur == nil {
			sb.WriteString(fmt.Sprintf("📡 %s\n  ❓ %s\n\n", t.Name, loc.msg("report.body.no_data_yet", nil)))
			continue
		}

		base := baseline[t.ID]

		sb.WriteString(fmt.Sprintf("📡 %s\n", t.Name))
		sb.WriteString(fmt.Sprintf("  ⬆️  %s: %s%s\n",
			loc.msg("report.body.up_label", nil), formatBytes(cur.Uploaded), formatDelta(base, cur, fieldUploaded, loc)))
		sb.WriteString(fmt.Sprintf("  ⬇️  %s: %s%s\n",
			loc.msg("report.body.dl_label", nil), formatBytes(cur.Downloaded), formatDelta(base, cur, fieldDownloaded, loc)))

		ratioLine := formatRatio(cur.Ratio, base, loc)
		sb.WriteString(fmt.Sprintf("  %s %s: %.2f%s\n", ratioEmoji(cur.Ratio), loc.msg("report.body.ratio_label", nil), cur.Ratio, ratioLine))
		sb.WriteString("\n")
	}

	sb.WriteString("🕐 ")
	sb.WriteString(loc.msg("report.body.generated_at", map[string]any{
		"Timestamp": loc.formatTimestamp(time.Now()),
	}))
	return sb.String()
}

type field int

const (
	fieldUploaded field = iota
	fieldDownloaded
)

func formatDelta(base, cur *domain.TrackerStats, f field, loc notificationLocalizer) string {
	if base == nil {
		return ""
	}
	var delta int64
	switch f {
	case fieldUploaded:
		delta = cur.Uploaded - base.Uploaded
	case fieldDownloaded:
		delta = cur.Downloaded - base.Downloaded
	}
	if delta == 0 {
		return fmt.Sprintf(" (🟰 %s)", loc.msg("report.body.no_change", nil))
	}
	arrow := "⏫"
	absD := delta
	if delta < 0 {
		arrow = "⏬"
		absD = -delta
	}
	return fmt.Sprintf(" (%s %s)", arrow, formatBytes(absD))
}

func formatRatio(cur float64, base *domain.TrackerStats, loc notificationLocalizer) string {
	if base == nil {
		return ""
	}
	delta := cur - base.Ratio
	if math.Abs(delta) < 0.005 {
		return fmt.Sprintf(" (🟰 %s)", loc.msg("report.body.no_change", nil))
	}
	arrow := "⏫"
	if delta < 0 {
		arrow = "⏬"
		delta = -delta
	}
	return fmt.Sprintf(" (%s %.2f)", arrow, delta)
}

func ratioEmoji(ratio float64) string {
	switch {
	case ratio >= 2.0:
		return "🟢"
	case ratio >= 1.0:
		return "🟡"
	default:
		return "🔴"
	}
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	abs := b
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= TB:
		return fmt.Sprintf("%.2f TiB", float64(b)/TB)
	case abs >= GB:
		return fmt.Sprintf("%.2f GiB", float64(b)/GB)
	case abs >= MB:
		return fmt.Sprintf("%.2f MiB", float64(b)/MB)
	case abs >= KB:
		return fmt.Sprintf("%.2f KiB", float64(b)/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
