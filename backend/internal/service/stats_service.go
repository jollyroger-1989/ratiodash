package service

import "github.com/jose/ratiodash/internal/domain"

type statsService struct {
	repo        domain.StatsRepository
	trackerRepo domain.TrackerRepository
}

func NewStatsService(repo domain.StatsRepository, trackerRepo domain.TrackerRepository) domain.StatsService {
	return &statsService{repo: repo, trackerRepo: trackerRepo}
}

func (s *statsService) GetHistory(trackerID uint, limit int) ([]domain.TrackerStats, error) {
	return s.repo.FindByTrackerID(trackerID, limit)
}

func (s *statsService) GetLatest(trackerID uint) (*domain.TrackerStats, error) {
	return s.repo.FindLatestByTrackerID(trackerID)
}

func (s *statsService) DeleteEntry(statID uint, trackerID uint) error {
	return s.repo.Delete(statID, trackerID)
}

func (s *statsService) GetDashboard() ([]domain.DashboardEntry, error) {
	trackers, err := s.trackerRepo.FindAll()
	if err != nil {
		return nil, err
	}

	latestStats, err := s.repo.FindLatestAll()
	if err != nil {
		return nil, err
	}

	statsByTrackerID := make(map[uint]*domain.TrackerStats, len(latestStats))
	for i := range latestStats {
		statsByTrackerID[latestStats[i].TrackerID] = &latestStats[i]
	}

	entries := make([]domain.DashboardEntry, len(trackers))
	for i, tracker := range trackers {
		tracker.PublicCredentials = domain.RedactCredentials(tracker.Credentials)
		entries[i] = domain.DashboardEntry{
			Tracker: tracker,
			Stats:   statsByTrackerID[tracker.ID],
		}
	}
	return entries, nil
}
