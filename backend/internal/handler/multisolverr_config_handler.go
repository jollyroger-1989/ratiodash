package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// MultisolverrConfigHandler holds the HTTP handlers for the singleton
// multisolverr proxy configuration, exposed under /settings/multisolverr.
type MultisolverrConfigHandler struct {
	service domain.MultisolverrConfigService
}

func NewMultisolverrConfigHandler(svc domain.MultisolverrConfigService) *MultisolverrConfigHandler {
	return &MultisolverrConfigHandler{service: svc}
}

// --- I/O types ---

type GetMultisolverrConfigOutput struct {
	Body *domain.MultisolverrConfig
}

type UpdateMultisolverrConfigInput struct {
	Body domain.UpdateMultisolverrConfigInput `doc:"Fields to update (all optional)"`
}
type UpdateMultisolverrConfigOutput struct {
	Body *domain.MultisolverrConfig
}

type TestMultisolverrConfigInput struct {
	Body struct {
		BaseURL string `json:"base_url" required:"true" minLength:"1" doc:"Multisolverr instance base URL to check"`
	}
}

// --- Handlers ---

func (h *MultisolverrConfigHandler) GetMultisolverrConfig(_ context.Context, _ *struct{}) (*GetMultisolverrConfigOutput, error) {
	cfg, err := h.service.Get()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load multisolverr config")
	}
	return &GetMultisolverrConfigOutput{Body: cfg}, nil
}

func (h *MultisolverrConfigHandler) UpdateMultisolverrConfig(_ context.Context, input *UpdateMultisolverrConfigInput) (*UpdateMultisolverrConfigOutput, error) {
	cfg, err := h.service.Update(input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return &UpdateMultisolverrConfigOutput{Body: cfg}, nil
}

func (h *MultisolverrConfigHandler) TestMultisolverrConfig(_ context.Context, input *TestMultisolverrConfigInput) (*struct{}, error) {
	if err := h.service.Test(input.Body.BaseURL); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	return nil, nil
}

// --- Route registration ---

func RegisterMultisolverrConfigRoutes(api huma.API, h *MultisolverrConfigHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "get-multisolverr-config",
		Method:      http.MethodGet,
		Path:        prefix + "/settings/multisolverr",
		Summary:     "Get the multisolverr proxy configuration",
		Tags:        []string{"settings"},
	}, h.GetMultisolverrConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-multisolverr-config",
		Method:      http.MethodPatch,
		Path:        prefix + "/settings/multisolverr",
		Summary:     "Update the multisolverr proxy configuration",
		Tags:        []string{"settings"},
	}, h.UpdateMultisolverrConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "test-multisolverr-config",
		Method:        http.MethodPost,
		Path:          prefix + "/settings/multisolverr/test",
		Summary:       "Check that a multisolverr instance is reachable",
		Tags:          []string{"settings"},
		DefaultStatus: http.StatusNoContent,
	}, h.TestMultisolverrConfig)
}
