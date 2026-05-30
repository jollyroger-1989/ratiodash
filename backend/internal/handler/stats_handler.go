package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// StatsHandler holds the HTTP handlers for SiteStats queries.
type StatsHandler struct {
	service domain.StatsService
}

func NewStatsHandler(svc domain.StatsService) *StatsHandler {
	return &StatsHandler{service: svc}
}

// --- I/O types ---

type GetDashboardOutput struct {
	Body []domain.DashboardEntry
}

type GetTrackerHistoryInput struct {
	TrackerID uint `path:"tracker_id"`
	Limit     int  `query:"limit" doc:"Maximum number of snapshots to return (default 50)"`
}
type GetTrackerHistoryOutput struct {
	Body []domain.TrackerStats
}

type GetGlobalHistoryInput struct {
	Limit int `query:"limit" doc:"Maximum number of daily snapshots to return"`
}
type GetGlobalHistoryOutput struct {
	Body []domain.GlobalStatsPoint
}

type GetLatestStatsInput struct {
	TrackerID uint `path:"tracker_id"`
}
type GetLatestStatsOutput struct {
	Body *domain.TrackerStats
}

// --- Handlers ---

func (h *StatsHandler) GetDashboard(ctx context.Context, _ *struct{}) (*GetDashboardOutput, error) {
	stats, err := h.service.GetDashboard()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load dashboard")
	}
	return &GetDashboardOutput{Body: stats}, nil
}

func (h *StatsHandler) GetTrackerHistory(ctx context.Context, input *GetTrackerHistoryInput) (*GetTrackerHistoryOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	stats, err := h.service.GetHistory(input.TrackerID, limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load history")
	}
	return &GetTrackerHistoryOutput{Body: stats}, nil
}

func (h *StatsHandler) GetGlobalHistory(ctx context.Context, input *GetGlobalHistoryInput) (*GetGlobalHistoryOutput, error) {
	stats, err := h.service.GetGlobalHistory(input.Limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load global history")
	}
	return &GetGlobalHistoryOutput{Body: stats}, nil
}

func (h *StatsHandler) GetLatestStats(ctx context.Context, input *GetLatestStatsInput) (*GetLatestStatsOutput, error) {
	stats, err := h.service.GetLatest(input.TrackerID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get stats")
	}
	if stats == nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("no stats found for tracker %d", input.TrackerID))
	}
	return &GetLatestStatsOutput{Body: stats}, nil
}

type DeleteStatInput struct {
	TrackerID uint `path:"tracker_id"`
	StatID    uint `path:"stat_id"`
}

func (h *StatsHandler) DeleteStat(ctx context.Context, input *DeleteStatInput) (*struct{}, error) {
	if err := h.service.DeleteEntry(input.StatID, input.TrackerID); err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	return nil, nil
}

// RegisterStatsRoutes registers Stats routes on the Huma API.
func RegisterStatsRoutes(api huma.API, h *StatsHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "get-trackers-stats",
		Method:      http.MethodGet,
		Path:        prefix + "/trackers/stats",
		Summary:     "Latest ratio snapshot for every tracked site",
		Tags:        []string{"stats"},
	}, h.GetDashboard)

	huma.Register(api, huma.Operation{
		OperationID: "get-tracker-history",
		Method:      http.MethodGet,
		Path:        prefix + "/trackers/{tracker_id}/stats",
		Summary:     "Historical stats for a tracker",
		Tags:        []string{"stats"},
	}, h.GetTrackerHistory)

	huma.Register(api, huma.Operation{
		OperationID: "get-global-history",
		Method:      http.MethodGet,
		Path:        prefix + "/stats/global",
		Summary:     "Daily aggregated stats across all trackers",
		Tags:        []string{"stats"},
	}, h.GetGlobalHistory)

	huma.Register(api, huma.Operation{
		OperationID: "get-latest-stats",
		Method:      http.MethodGet,
		Path:        prefix + "/trackers/{tracker_id}/stats/latest",
		Summary:     "Most recent stats snapshot for a tracker",
		Tags:        []string{"stats"},
	}, h.GetLatestStats)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-stat",
		Method:        http.MethodDelete,
		Path:          prefix + "/trackers/{tracker_id}/stats/{stat_id}",
		Summary:       "Delete a single history snapshot",
		Tags:          []string{"stats"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteStat)
}
