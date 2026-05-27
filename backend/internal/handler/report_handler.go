package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// ReportHandler holds the HTTP handlers for Report CRUD and manual dispatch.
type ReportHandler struct {
	service   domain.ReportService
	scheduler domain.ReportScheduler
}

func NewReportHandler(svc domain.ReportService, scheduler domain.ReportScheduler) *ReportHandler {
	return &ReportHandler{service: svc, scheduler: scheduler}
}

// --- I/O types ---

type ListReportsOutput struct {
	Body []domain.Report
}

type GetReportInput struct {
	ID uint `path:"id"`
}
type GetReportOutput struct {
	Body *domain.Report
}

type CreateReportInput struct {
	Body domain.CreateReportInput
}
type CreateReportOutput struct {
	Body *domain.Report
}

type UpdateReportInput struct {
	ID   uint                     `path:"id"`
	Body domain.UpdateReportInput `doc:"Fields to update (all optional)"`
}
type UpdateReportOutput struct {
	Body *domain.Report
}

type DeleteReportInput struct {
	ID uint `path:"id"`
}

type SendReportInput struct {
	ID uint `path:"id"`
}

// --- Handlers ---

func (h *ReportHandler) ListReports(_ context.Context, _ *struct{}) (*ListReportsOutput, error) {
	reports, err := h.service.GetAll()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list reports")
	}
	return &ListReportsOutput{Body: reports}, nil
}

func (h *ReportHandler) GetReport(_ context.Context, input *GetReportInput) (*GetReportOutput, error) {
	r, err := h.service.GetByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("report %d not found", input.ID))
	}
	return &GetReportOutput{Body: r}, nil
}

func (h *ReportHandler) CreateReport(_ context.Context, input *CreateReportInput) (*CreateReportOutput, error) {
	r, err := h.service.Create(input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	if err := h.scheduler.ScheduleReport(*r); err != nil {
		// Non-fatal: report is persisted; log via error response message.
		return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("report created but scheduling failed: %v", err))
	}
	return &CreateReportOutput{Body: r}, nil
}

func (h *ReportHandler) UpdateReport(_ context.Context, input *UpdateReportInput) (*UpdateReportOutput, error) {
	r, err := h.service.Update(input.ID, input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	if err := h.scheduler.ScheduleReport(*r); err != nil {
		return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("report updated but rescheduling failed: %v", err))
	}
	return &UpdateReportOutput{Body: r}, nil
}

func (h *ReportHandler) DeleteReport(_ context.Context, input *DeleteReportInput) (*struct{}, error) {
	if err := h.service.Delete(input.ID); err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("report %d not found", input.ID))
	}
	h.scheduler.UnscheduleReport(input.ID)
	return nil, nil
}

func (h *ReportHandler) SendReport(ctx context.Context, input *SendReportInput) (*struct{}, error) {
	if err := h.service.Send(ctx, input.ID); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

// --- Route registration ---

func RegisterReportRoutes(api huma.API, h *ReportHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "list-reports",
		Method:      http.MethodGet,
		Path:        prefix + "/reports",
		Summary:     "List all reports",
		Tags:        []string{"reports"},
	}, h.ListReports)

	huma.Register(api, huma.Operation{
		OperationID: "get-report",
		Method:      http.MethodGet,
		Path:        prefix + "/reports/{id}",
		Summary:     "Get a report by ID",
		Tags:        []string{"reports"},
	}, h.GetReport)

	huma.Register(api, huma.Operation{
		OperationID:   "create-report",
		Method:        http.MethodPost,
		Path:          prefix + "/reports",
		Summary:       "Create a new report",
		Tags:          []string{"reports"},
		DefaultStatus: http.StatusCreated,
	}, h.CreateReport)

	huma.Register(api, huma.Operation{
		OperationID: "update-report",
		Method:      http.MethodPatch,
		Path:        prefix + "/reports/{id}",
		Summary:     "Update a report",
		Tags:        []string{"reports"},
	}, h.UpdateReport)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-report",
		Method:        http.MethodDelete,
		Path:          prefix + "/reports/{id}",
		Summary:       "Delete a report",
		Tags:          []string{"reports"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteReport)

	huma.Register(api, huma.Operation{
		OperationID: "send-report",
		Method:      http.MethodPost,
		Path:        prefix + "/reports/{id}/send",
		Summary:     "Immediately dispatch a report",
		Tags:        []string{"reports"},
	}, h.SendReport)
}
