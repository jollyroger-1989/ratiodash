package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sirupsen/logrus"

	"github.com/jose/ratiodash/internal/domain"
)

// TrackerHandler holds the HTTP handlers for Tracker CRUD and manual refresh.
type TrackerHandler struct {
	service   domain.TrackerService
	refresh   domain.RefreshService
	refresher domain.Refresher
}

func NewTrackerHandler(svc domain.TrackerService, refresh domain.RefreshService, r domain.Refresher) *TrackerHandler {
	return &TrackerHandler{service: svc, refresh: refresh, refresher: r}
}

// --- I/O types ---

type ListTrackersOutput struct {
	Body []domain.Tracker
}

type GetTrackerInput struct {
	ID uint `path:"id"`
}
type GetTrackerOutput struct {
	Body *domain.Tracker
}

type CreateTrackerInput struct {
	Body domain.CreateTrackerInput
}
type CreateTrackerOutput struct {
	Body *domain.Tracker
}

type UpdateTrackerInput struct {
	ID   uint                      `path:"id"`
	Body domain.UpdateTrackerInput `doc:"Fields to update (all optional)"`
}
type UpdateTrackerOutput struct {
	Body *domain.Tracker
}

type DeleteTrackerInput struct {
	ID uint `path:"id"`
}

type RefreshTrackerInput struct {
	ID uint `path:"id"`
}

type TestTrackerInput struct {
	Body struct {
		ScraperKey  string `json:"scraper_key"  required:"true" minLength:"1" doc:"Scraper backend key"`
		Credentials string `json:"credentials,omitempty"  doc:"JSON credentials blob to test"`
	}
}

type TestTrackerByIDInput struct {
	ID   uint `path:"id"`
	Body struct {
		Credentials string `json:"credentials,omitempty" doc:"Partial credentials override; non-empty keys win over stored values"`
	}
}

// --- Handlers ---

func (h *TrackerHandler) ListTrackers(ctx context.Context, _ *struct{}) (*ListTrackersOutput, error) {
	trackers, err := h.service.GetAll()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list trackers")
	}
	return &ListTrackersOutput{Body: trackers}, nil
}

func (h *TrackerHandler) GetTracker(ctx context.Context, input *GetTrackerInput) (*GetTrackerOutput, error) {
	tracker, err := h.service.GetByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("tracker %d not found", input.ID))
	}
	return &GetTrackerOutput{Body: tracker}, nil
}

func (h *TrackerHandler) CreateTracker(ctx context.Context, input *CreateTrackerInput) (*CreateTrackerOutput, error) {
	tracker, err := h.service.Create(input.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create tracker")
	}
	if err := h.refresh.RefreshTracker(ctx, tracker.ID); err != nil {
		_ = h.service.Delete(tracker.ID)
		return nil, huma.Error422UnprocessableEntity("scrape failed — check the tracker URL and credentials")
	}
	if err := h.refresher.Schedule(*tracker); err != nil {
		logrus.WithError(err).WithField("tracker_id", tracker.ID).Warn("handler_schedule_tracker_failed")
	}
	return &CreateTrackerOutput{Body: tracker}, nil
}

func (h *TrackerHandler) UpdateTracker(ctx context.Context, input *UpdateTrackerInput) (*UpdateTrackerOutput, error) {
	old, err := h.service.GetByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("tracker %d not found", input.ID))
	}

	tracker, err := h.service.Update(input.ID, input.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update tracker")
	}

	if err := h.refresh.RefreshTracker(ctx, tracker.ID); err != nil {
		// Restore old state on scrape failure
		_, _ = h.service.Update(input.ID, domain.UpdateTrackerInput{
			Name:        &old.Name,
			Credentials: &old.Credentials,
			CronExpr:    &old.CronExpr,
			Active:      &old.Active,
		})
		return nil, huma.Error422UnprocessableEntity("scrape failed — check the tracker URL and credentials")
	}
	if err := h.refresher.Schedule(*tracker); err != nil {
		logrus.WithError(err).WithField("tracker_id", tracker.ID).Warn("handler_reschedule_tracker_failed")
	}
	return &UpdateTrackerOutput{Body: tracker}, nil
}

func (h *TrackerHandler) DeleteTracker(ctx context.Context, input *DeleteTrackerInput) (*struct{}, error) {
	if err := h.service.Delete(input.ID); err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("tracker %d not found", input.ID))
	}
	h.refresher.Unschedule(input.ID)
	return nil, nil
}

func (h *TrackerHandler) RefreshTracker(ctx context.Context, input *RefreshTrackerInput) (*struct{}, error) {
	if err := h.refresh.RefreshTracker(ctx, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("refresh failed")
	}
	return nil, nil
}

func (h *TrackerHandler) TestTracker(ctx context.Context, input *TestTrackerInput) (*struct{}, error) {
	if err := h.service.Test(input.Body.ScraperKey, input.Body.Credentials); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

func (h *TrackerHandler) TestTrackerByID(ctx context.Context, input *TestTrackerByIDInput) (*struct{}, error) {
	if err := h.service.TestByID(input.ID, input.Body.Credentials); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

// RegisterTrackerRoutes registers Tracker routes on the Huma API.
func RegisterTrackerRoutes(api huma.API, h *TrackerHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "list-trackers",
		Method:      http.MethodGet,
		Path:        prefix + "/trackers",
		Summary:     "List all tracked trackers",
		Tags:        []string{"trackers"},
	}, h.ListTrackers)

	huma.Register(api, huma.Operation{
		OperationID: "get-tracker",
		Method:      http.MethodGet,
		Path:        prefix + "/trackers/{id}",
		Summary:     "Get a tracker by ID",
		Tags:        []string{"trackers"},
	}, h.GetTracker)

	huma.Register(api, huma.Operation{
		OperationID:   "create-tracker",
		Method:        http.MethodPost,
		Path:          prefix + "/trackers",
		Summary:       "Register a new torrent tracker",
		Tags:          []string{"trackers"},
		DefaultStatus: http.StatusCreated,
	}, h.CreateTracker)

	huma.Register(api, huma.Operation{
		OperationID: "update-tracker",
		Method:      http.MethodPatch,
		Path:        prefix + "/trackers/{id}",
		Summary:     "Update a tracker's settings",
		Tags:        []string{"trackers"},
	}, h.UpdateTracker)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-tracker",
		Method:        http.MethodDelete,
		Path:          prefix + "/trackers/{id}",
		Summary:       "Remove a tracker",
		Tags:          []string{"trackers"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteTracker)

	huma.Register(api, huma.Operation{
		OperationID:   "refresh-tracker",
		Method:        http.MethodPost,
		Path:          prefix + "/trackers/{id}/refresh",
		Summary:       "Trigger an immediate stats refresh",
		Tags:          []string{"trackers"},
		DefaultStatus: http.StatusNoContent,
	}, h.RefreshTracker)

	huma.Register(api, huma.Operation{
		OperationID:   "test-tracker",
		Method:        http.MethodPost,
		Path:          prefix + "/trackers/test",
		Summary:       "Dry-run a scraper with provided credentials",
		Tags:          []string{"trackers"},
		DefaultStatus: http.StatusNoContent,
	}, h.TestTracker)

	huma.Register(api, huma.Operation{
		OperationID:   "test-tracker-by-id",
		Method:        http.MethodPost,
		Path:          prefix + "/trackers/{id}/test",
		Summary:       "Dry-run an existing tracker's scraper with optional credential override",
		Tags:          []string{"trackers"},
		DefaultStatus: http.StatusNoContent,
	}, h.TestTrackerByID)
}
