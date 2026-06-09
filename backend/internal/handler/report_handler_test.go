package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/internal/testutil"
)

func newReportAPI(t *testing.T, svc *mocks.MockReportService, sched *mocks.MockReportScheduler) humatest.TestAPI {
	t.Helper()
	h := handler.NewReportHandler(svc, sched)
	api := testutil.NewAPI(t)
	handler.RegisterReportRoutes(api, h)
	return api
}

func TestReportHandler_List(t *testing.T) {
	t.Run("returns empty list", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().GetAll().Return([]domain.Report{}, nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodGet, "/api/v1/reports")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.Report
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns existing reports", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().GetAll().Return([]domain.Report{
			{ID: 1, Name: "Weekly", CronExpr: "@weekly"},
		}, nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodGet, "/api/v1/reports")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []domain.Report
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "Weekly", body[0].Name)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().GetAll().Return(nil, errors.New("db error"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodGet, "/api/v1/reports")

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestReportHandler_Get(t *testing.T) {
	t.Run("returns report by ID", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().GetByID(uint(1)).Return(&domain.Report{ID: 1, Name: "Daily"}, nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodGet, "/api/v1/reports/1")

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.Report
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Daily", body.Name)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().GetByID(uint(99)).Return(nil, errors.New("not found"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodGet, "/api/v1/reports/99")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestReportHandler_Create(t *testing.T) {
	t.Run("creates report and returns 201", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		input := domain.CreateReportInput{
			Name:     "Monthly",
			CronExpr: "@monthly",
		}
		created := &domain.Report{ID: 1, Name: "Monthly", CronExpr: "@monthly"}
		svc.EXPECT().Create(input).Return(created, nil)
		sched.EXPECT().ScheduleReport(*created).Return(nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"name":      "Monthly",
			"cron_expr": "@monthly",
		})

		require.Equal(t, http.StatusCreated, resp.Code)
		var body domain.Report
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Monthly", body.Name)
	})

	t.Run("returns 422 when service fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Create(domain.CreateReportInput{Name: "X", CronExpr: "bad"}).
			Return(nil, errors.New("invalid cron"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"name":      "X",
			"cron_expr": "bad",
		})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})

	t.Run("returns 422 when scheduling fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		input := domain.CreateReportInput{Name: "Bad sched", CronExpr: "@daily"}
		created := &domain.Report{ID: 2, Name: "Bad sched", CronExpr: "@daily"}
		svc.EXPECT().Create(input).Return(created, nil)
		sched.EXPECT().ScheduleReport(*created).Return(errors.New("scheduler error"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"name":      "Bad sched",
			"cron_expr": "@daily",
		})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestReportHandler_Update(t *testing.T) {
	t.Run("updates report and reschedules", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		newName := "Renamed"
		updated := &domain.Report{ID: 1, Name: "Renamed", CronExpr: "@weekly"}
		svc.EXPECT().Update(uint(1), domain.UpdateReportInput{Name: &newName}).Return(updated, nil)
		sched.EXPECT().ScheduleReport(*updated).Return(nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodPatch, "/api/v1/reports/1", map[string]interface{}{
			"name": "Renamed",
		})

		require.Equal(t, http.StatusOK, resp.Code)
		var body domain.Report
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Renamed", body.Name)
	})

	t.Run("returns 422 when service fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Update(uint(1), domain.UpdateReportInput{}).Return(nil, errors.New("not found"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodPatch, "/api/v1/reports/1", map[string]interface{}{})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})

	t.Run("returns 422 when rescheduling fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		newCron := "@hourly"
		updated := &domain.Report{ID: 1, Name: "X", CronExpr: "@hourly"}
		svc.EXPECT().Update(uint(1), domain.UpdateReportInput{CronExpr: &newCron}).Return(updated, nil)
		sched.EXPECT().ScheduleReport(*updated).Return(errors.New("invalid cron"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodPatch, "/api/v1/reports/1", map[string]interface{}{
			"cron_expr": "@hourly",
		})

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

func TestReportHandler_Delete(t *testing.T) {
	t.Run("deletes report and returns 204", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Delete(uint(1)).Return(nil)
		sched.EXPECT().UnscheduleReport(uint(1))

		resp := newReportAPI(t, svc, sched).Do(http.MethodDelete, "/api/v1/reports/1")

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 404 when service fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Delete(uint(99)).Return(errors.New("not found"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodDelete, "/api/v1/reports/99")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestReportHandler_Send(t *testing.T) {
	t.Run("sends report and returns 200", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Send(mock.Anything, uint(1)).Return(nil)

		resp := newReportAPI(t, svc, sched).Do(http.MethodPost, "/api/v1/reports/1/send")

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("returns 422 when send fails", func(t *testing.T) {
		svc := mocks.NewMockReportService(t)
		sched := mocks.NewMockReportScheduler(t)
		svc.EXPECT().Send(mock.Anything, uint(1)).Return(errors.New("no notifiers configured"))

		resp := newReportAPI(t, svc, sched).Do(http.MethodPost, "/api/v1/reports/1/send")

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})
}

// Integration test using a real SQLite DB.
func TestReportHandler_Integration(t *testing.T) {
	t.Run("full CRUD lifecycle without scheduling", func(t *testing.T) {
		db := testutil.NewDB(t)
		sched := mocks.NewMockReportScheduler(t)
		repo := repository.NewReportRepository(db)
		trackerSvc := mocks.NewMockTrackerService(t)
		trackerSvc.EXPECT().GetAll(mock.Anything).Return([]domain.Tracker{}, nil).Maybe()
		svc := service.NewReportService(
			repo,
			repository.NewNotifierConfigRepository(db),
			trackerSvc,
			repository.NewStatsRepository(db),
			mocks.NewMockNotifierBuilder(t),
		)
		h := handler.NewReportHandler(svc, sched)
		api := testutil.NewAPI(t)
		handler.RegisterReportRoutes(api, h)

		// Create
		sched.EXPECT().ScheduleReport(mock.MatchedBy(func(r domain.Report) bool {
			return r.Name == "Weekly digest" && r.CronExpr == "@weekly"
		})).Return(nil)
		createResp := api.Do(http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"name":      "Weekly digest",
			"cron_expr": "@weekly",
		})
		require.Equal(t, http.StatusCreated, createResp.Code)
		var created domain.Report
		require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
		assert.Equal(t, "Weekly digest", created.Name)
		assert.NotZero(t, created.ID)

		// Get
		getResp := api.Do(http.MethodGet, "/api/v1/reports/1")
		require.Equal(t, http.StatusOK, getResp.Code)

		// List
		listResp := api.Do(http.MethodGet, "/api/v1/reports")
		require.Equal(t, http.StatusOK, listResp.Code)
		var reports []domain.Report
		require.NoError(t, json.NewDecoder(listResp.Body).Decode(&reports))
		require.Len(t, reports, 1)

		// Update
		sched.EXPECT().ScheduleReport(mock.MatchedBy(func(r domain.Report) bool {
			return r.ID == 1 && r.Name == "Updated"
		})).Return(nil)
		updateResp := api.Do(http.MethodPatch, "/api/v1/reports/1", map[string]interface{}{
			"name": "Updated",
		})
		require.Equal(t, http.StatusOK, updateResp.Code)
		var updated domain.Report
		require.NoError(t, json.NewDecoder(updateResp.Body).Decode(&updated))
		assert.Equal(t, "Updated", updated.Name)

		// Delete
		sched.EXPECT().UnscheduleReport(uint(1))
		deleteResp := api.Do(http.MethodDelete, "/api/v1/reports/1")
		assert.Equal(t, http.StatusNoContent, deleteResp.Code)
	})
}
