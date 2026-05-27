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
	r, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("report %d not found", id)
	}

	// Get current stats for all trackers.
	allTrackers, err := s.trackers.GetAll()
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

	body := buildReportBody(allTrackers, latestByTracker, baselineByTracker, r.LastSentAt)

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
			Title: fmt.Sprintf("[RatioDash] 📊 %s", r.Name),
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
) string {
	var sb strings.Builder

	if lastSentAt != nil {
		sb.WriteString(fmt.Sprintf("Evolution since %s\n\n", lastSentAt.UTC().Format("2006-01-02 15:04 UTC")))
	} else {
		sb.WriteString("First report — no previous baseline\n\n")
	}

	for _, t := range trackers {
		cur := latest[t.ID]
		if cur == nil {
			sb.WriteString(fmt.Sprintf("📡 %s\n  ❓ No data yet\n\n", t.Name))
			continue
		}

		base := baseline[t.ID]

		sb.WriteString(fmt.Sprintf("📡 %s\n", t.Name))
		sb.WriteString(fmt.Sprintf("  ⬆️  Upload:   %s%s\n",
			formatBytes(cur.Uploaded), formatDelta(base, cur, fieldUploaded)))
		sb.WriteString(fmt.Sprintf("  ⬇️  Download: %s%s\n",
			formatBytes(cur.Downloaded), formatDelta(base, cur, fieldDownloaded)))

		ratioLine := formatRatio(cur.Ratio, base, cur)
		sb.WriteString(fmt.Sprintf("  %s Ratio:    %.2f%s\n", ratioEmoji(cur.Ratio), cur.Ratio, ratioLine))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("🕐 Generated at %s", time.Now().UTC().Format("2006-01-02 15:04 UTC")))
	return sb.String()
}

type field int

const (
	fieldUploaded field = iota
	fieldDownloaded
)

func formatDelta(base, cur *domain.TrackerStats, f field) string {
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
		return " (→ no change)"
	}
	arrow := "↑"
	absD := delta
	if delta < 0 {
		arrow = "↓"
		absD = -delta
	}
	sign := "+"
	if delta < 0 {
		sign = "-"
	}
	return fmt.Sprintf(" (%s%s %s)", sign, formatBytes(absD), arrow)
}

func formatRatio(cur float64, base, _ *domain.TrackerStats) string {
	if base == nil {
		return ""
	}
	delta := cur - base.Ratio
	if math.Abs(delta) < 0.005 {
		return " (→ no change)"
	}
	arrow := "↑"
	if delta < 0 {
		arrow = "↓"
	}
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	return fmt.Sprintf(" (%s%.2f %s)", sign, delta, arrow)
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
