package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// AlertConfigHandler holds the HTTP handlers for AlertConfig CRUD.
type AlertConfigHandler struct {
	service domain.AlertConfigService
}

func NewAlertConfigHandler(svc domain.AlertConfigService) *AlertConfigHandler {
	return &AlertConfigHandler{service: svc}
}

// --- I/O types ---

type ListAlertConfigsOutput struct {
	Body []domain.AlertConfig
}

type GetAlertConfigInput struct {
	ID uint `path:"id"`
}
type GetAlertConfigOutput struct {
	Body *domain.AlertConfig
}

type CreateAlertConfigInput struct {
	Body domain.CreateAlertConfigInput
}
type CreateAlertConfigOutput struct {
	Body *domain.AlertConfig
}

type UpdateAlertConfigInput struct {
	ID   uint                          `path:"id"`
	Body domain.UpdateAlertConfigInput `doc:"Fields to update (all optional)"`
}
type UpdateAlertConfigOutput struct {
	Body *domain.AlertConfig
}

type DeleteAlertConfigInput struct {
	ID uint `path:"id"`
}

// --- Handlers ---

func (h *AlertConfigHandler) ListAlertConfigs(_ context.Context, _ *struct{}) (*ListAlertConfigsOutput, error) {
	configs, err := h.service.GetAll()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list alert configs")
	}
	return &ListAlertConfigsOutput{Body: configs}, nil
}

func (h *AlertConfigHandler) GetAlertConfig(_ context.Context, input *GetAlertConfigInput) (*GetAlertConfigOutput, error) {
	c, err := h.service.GetByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("alert config %d not found", input.ID))
	}
	return &GetAlertConfigOutput{Body: c}, nil
}

func (h *AlertConfigHandler) CreateAlertConfig(_ context.Context, input *CreateAlertConfigInput) (*CreateAlertConfigOutput, error) {
	c, err := h.service.Create(input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return &CreateAlertConfigOutput{Body: c}, nil
}

func (h *AlertConfigHandler) UpdateAlertConfig(_ context.Context, input *UpdateAlertConfigInput) (*UpdateAlertConfigOutput, error) {
	c, err := h.service.Update(input.ID, input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return &UpdateAlertConfigOutput{Body: c}, nil
}

func (h *AlertConfigHandler) DeleteAlertConfig(_ context.Context, input *DeleteAlertConfigInput) (*struct{}, error) {
	if err := h.service.Delete(input.ID); err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("alert config %d not found", input.ID))
	}
	return nil, nil
}

// --- Route registration ---

func RegisterAlertConfigRoutes(api huma.API, h *AlertConfigHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "list-alert-configs",
		Method:      http.MethodGet,
		Path:        prefix + "/alert-configs",
		Summary:     "List all alert configs",
		Tags:        []string{"alert-configs"},
	}, h.ListAlertConfigs)

	huma.Register(api, huma.Operation{
		OperationID: "get-alert-config",
		Method:      http.MethodGet,
		Path:        prefix + "/alert-configs/{id}",
		Summary:     "Get an alert config by ID",
		Tags:        []string{"alert-configs"},
	}, h.GetAlertConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "create-alert-config",
		Method:        http.MethodPost,
		Path:          prefix + "/alert-configs",
		Summary:       "Create a new alert config",
		Tags:          []string{"alert-configs"},
		DefaultStatus: http.StatusCreated,
	}, h.CreateAlertConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-alert-config",
		Method:      http.MethodPatch,
		Path:        prefix + "/alert-configs/{id}",
		Summary:     "Update an alert config",
		Tags:        []string{"alert-configs"},
	}, h.UpdateAlertConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-alert-config",
		Method:        http.MethodDelete,
		Path:          prefix + "/alert-configs/{id}",
		Summary:       "Delete an alert config",
		Tags:          []string{"alert-configs"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteAlertConfig)
}
