package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// NotifierConfigHandler holds the HTTP handlers for NotifierConfig CRUD
// and the notifier type catalogue.
type NotifierConfigHandler struct {
	service domain.NotifierConfigService
}

func NewNotifierConfigHandler(svc domain.NotifierConfigService) *NotifierConfigHandler {
	return &NotifierConfigHandler{service: svc}
}

// --- I/O types ---

type ListNotifierTypesOutput struct {
	Body []domain.NotifierTypeInfo
}

type ListNotifierConfigsOutput struct {
	Body []domain.NotifierConfig
}

type GetNotifierConfigInput struct {
	ID uint `path:"id"`
}
type GetNotifierConfigOutput struct {
	Body *domain.NotifierConfig
}

type CreateNotifierConfigInput struct {
	Body domain.CreateNotifierConfigInput
}
type CreateNotifierConfigOutput struct {
	Body *domain.NotifierConfig
}

type UpdateNotifierConfigInput struct {
	ID   uint                             `path:"id"`
	Body domain.UpdateNotifierConfigInput `doc:"Fields to update (all optional)"`
}
type UpdateNotifierConfigOutput struct {
	Body *domain.NotifierConfig
}

type DeleteNotifierConfigInput struct {
	ID uint `path:"id"`
}

type TestNotifierConfigInput struct {
	Body struct {
		Type   string `json:"type"   required:"true" minLength:"1" doc:"Notifier backend type key"`
		Config string `json:"config" doc:"Type-specific JSON config blob"`
	}
}

type TestNotifierConfigByIDInput struct {
	ID   uint `path:"id"`
	Body struct {
		Config string `json:"config" doc:"Partial config override; non-empty keys win over stored values"`
	}
}

// --- Handlers ---

func (h *NotifierConfigHandler) ListNotifierTypes(_ context.Context, _ *struct{}) (*ListNotifierTypesOutput, error) {
	return &ListNotifierTypesOutput{Body: domain.AvailableNotifierTypes}, nil
}

func (h *NotifierConfigHandler) ListNotifierConfigs(_ context.Context, _ *struct{}) (*ListNotifierConfigsOutput, error) {
	cfgs, err := h.service.GetAll()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list notifier configs")
	}
	return &ListNotifierConfigsOutput{Body: cfgs}, nil
}

func (h *NotifierConfigHandler) GetNotifierConfig(_ context.Context, input *GetNotifierConfigInput) (*GetNotifierConfigOutput, error) {
	cfg, err := h.service.GetByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("notifier config %d not found", input.ID))
	}
	return &GetNotifierConfigOutput{Body: cfg}, nil
}

func (h *NotifierConfigHandler) CreateNotifierConfig(_ context.Context, input *CreateNotifierConfigInput) (*CreateNotifierConfigOutput, error) {
	cfg, err := h.service.Create(input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return &CreateNotifierConfigOutput{Body: cfg}, nil
}

func (h *NotifierConfigHandler) UpdateNotifierConfig(_ context.Context, input *UpdateNotifierConfigInput) (*UpdateNotifierConfigOutput, error) {
	cfg, err := h.service.Update(input.ID, input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return &UpdateNotifierConfigOutput{Body: cfg}, nil
}

func (h *NotifierConfigHandler) DeleteNotifierConfig(_ context.Context, input *DeleteNotifierConfigInput) (*struct{}, error) {
	if err := h.service.Delete(input.ID); err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("notifier config %d not found", input.ID))
	}
	return nil, nil
}

func (h *NotifierConfigHandler) TestNotifierConfig(_ context.Context, input *TestNotifierConfigInput) (*struct{}, error) {
	if err := h.service.Test(input.Body.Type, input.Body.Config); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

func (h *NotifierConfigHandler) TestNotifierConfigByID(_ context.Context, input *TestNotifierConfigByIDInput) (*struct{}, error) {
	if err := h.service.TestByID(input.ID, input.Body.Config); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

// --- Route registration ---

func RegisterNotifierConfigRoutes(api huma.API, h *NotifierConfigHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "list-notifier-types",
		Method:      http.MethodGet,
		Path:        prefix + "/notifier-types",
		Summary:     "List available notifier backend types and their config fields",
		Tags:        []string{"notifiers"},
	}, h.ListNotifierTypes)

	huma.Register(api, huma.Operation{
		OperationID: "list-notifier-configs",
		Method:      http.MethodGet,
		Path:        prefix + "/notifier-configs",
		Summary:     "List all notifier configurations",
		Tags:        []string{"notifiers"},
	}, h.ListNotifierConfigs)

	huma.Register(api, huma.Operation{
		OperationID: "get-notifier-config",
		Method:      http.MethodGet,
		Path:        prefix + "/notifier-configs/{id}",
		Summary:     "Get a notifier configuration by ID",
		Tags:        []string{"notifiers"},
	}, h.GetNotifierConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "create-notifier-config",
		Method:        http.MethodPost,
		Path:          prefix + "/notifier-configs",
		Summary:       "Add a new notifier configuration",
		Tags:          []string{"notifiers"},
		DefaultStatus: http.StatusCreated,
	}, h.CreateNotifierConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-notifier-config",
		Method:      http.MethodPatch,
		Path:        prefix + "/notifier-configs/{id}",
		Summary:     "Update a notifier configuration",
		Tags:        []string{"notifiers"},
	}, h.UpdateNotifierConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-notifier-config",
		Method:        http.MethodDelete,
		Path:          prefix + "/notifier-configs/{id}",
		Summary:       "Remove a notifier configuration",
		Tags:          []string{"notifiers"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteNotifierConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "test-notifier-config",
		Method:        http.MethodPost,
		Path:          prefix + "/notifier-configs/test",
		Summary:       "Send a test notification using the supplied type and config (before saving)",
		Tags:          []string{"notifiers"},
		DefaultStatus: http.StatusNoContent,
	}, h.TestNotifierConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "test-notifier-config-by-id",
		Method:        http.MethodPost,
		Path:          prefix + "/notifier-configs/{id}/test",
		Summary:       "Send a test notification using a saved notifier configuration",
		Tags:          []string{"notifiers"},
		DefaultStatus: http.StatusNoContent,
	}, h.TestNotifierConfigByID)
}
