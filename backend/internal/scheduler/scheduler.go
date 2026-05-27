package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
)

// Scheduler manages cron-based scraping for all active trackers and report dispatches.
// It implements domain.Refresher and domain.ReportScheduler so handlers can
// reschedule jobs at runtime.
type Scheduler struct {
	cron          *cron.Cron
	mu            sync.Mutex
	entries       map[uint]cron.EntryID // tracker entries
	reportEntries map[uint]cron.EntryID // report entries
	refresh       domain.RefreshService
	trackers      domain.TrackerService
	reports       domain.ReportService
}

func New(refresh domain.RefreshService, trackers domain.TrackerService, reports domain.ReportService) *Scheduler {
	return &Scheduler{
		cron:          cron.New(),
		entries:       make(map[uint]cron.EntryID),
		reportEntries: make(map[uint]cron.EntryID),
		refresh:       refresh,
		trackers:      trackers,
		reports:       reports,
	}
}

// AsRefresher adapts *Scheduler to domain.Refresher for FX injection.
func AsRefresher(s *Scheduler) domain.Refresher { return s }

// AsReportScheduler adapts *Scheduler to domain.ReportScheduler for FX injection.
func AsReportScheduler(s *Scheduler) domain.ReportScheduler { return s }

// Schedule registers (or replaces) the cron job for the given tracker.
func (s *Scheduler) Schedule(tracker domain.Tracker) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.entries[tracker.ID]; ok {
		s.cron.Remove(id)
	}

	id, err := s.cron.AddFunc(tracker.CronExpr, func() {
		if err := s.refresh.RefreshTracker(context.Background(), tracker.ID); err != nil {
			log.Printf("scheduler: tracker %d (%s): %v", tracker.ID, tracker.Name, err)
		}
	})
	if err != nil {
		return fmt.Errorf("scheduling tracker %d with expr %q: %w", tracker.ID, tracker.CronExpr, err)
	}

	s.entries[tracker.ID] = id
	return nil
}

// Unschedule removes the cron job for the given tracker.
func (s *Scheduler) Unschedule(trackerID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.entries[trackerID]; ok {
		s.cron.Remove(id)
		delete(s.entries, trackerID)
	}
}

// ScheduleReport registers (or replaces) the cron job for the given report.
func (s *Scheduler) ScheduleReport(report domain.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.reportEntries[report.ID]; ok {
		s.cron.Remove(id)
	}

	id, err := s.cron.AddFunc(report.CronExpr, func() {
		if err := s.reports.Send(context.Background(), report.ID); err != nil {
			log.Printf("scheduler: report %d (%s): %v", report.ID, report.Name, err)
		}
	})
	if err != nil {
		return fmt.Errorf("scheduling report %d with expr %q: %w", report.ID, report.CronExpr, err)
	}

	s.reportEntries[report.ID] = id
	return nil
}

// UnscheduleReport removes the cron job for the given report.
func (s *Scheduler) UnscheduleReport(reportID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.reportEntries[reportID]; ok {
		s.cron.Remove(id)
		delete(s.reportEntries, reportID)
	}
}

// Start is an FX lifecycle hook that loads all active trackers and reports,
// schedules them, and starts the cron runner.
func Start(lc fx.Lifecycle, s *Scheduler) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			trackers, err := s.trackers.GetActive()
			if err != nil {
				return fmt.Errorf("scheduler: loading active trackers: %w", err)
			}
			for _, tracker := range trackers {
				if err := s.Schedule(tracker); err != nil {
					log.Printf("scheduler: skipping tracker %q: %v", tracker.Name, err)
				}
			}

			reports, err := s.reports.GetAll()
			if err != nil {
				return fmt.Errorf("scheduler: loading reports: %w", err)
			}
			for _, report := range reports {
				if err := s.ScheduleReport(report); err != nil {
					log.Printf("scheduler: skipping report %q: %v", report.Name, err)
				}
			}

			s.cron.Start()
			log.Printf("scheduler: started with %d tracker(s) and %d report(s)", len(trackers), len(reports))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.cron.Stop()
			return nil
		},
	})
}
