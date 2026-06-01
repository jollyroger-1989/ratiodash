package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

type APIClientHandler struct {
	service domain.APIClientService
}

func NewAPIClientHandler(svc domain.APIClientService) *APIClientHandler {
	return &APIClientHandler{service: svc}
}

type ListAPIClientsOutput struct {
	Body []domain.APIClient
}

type CreateAPIClientInput struct {
	Body domain.CreateAPIClientInput
}

type CreateAPIClientOutput struct {
	Body struct {
		Client *domain.APIClient `json:"client"`
		APIKey string            `json:"api_key" doc:"Plaintext API key shown only once at creation"`
	}
}

type DeleteAPIClientInput struct {
	ID uint `path:"id"`
}

func (h *APIClientHandler) ListAPIClients(_ context.Context, _ *struct{}) (*ListAPIClientsOutput, error) {
	clients, err := h.service.GetAll()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list api clients")
	}
	return &ListAPIClientsOutput{Body: clients}, nil
}

func (h *APIClientHandler) CreateAPIClient(_ context.Context, input *CreateAPIClientInput) (*CreateAPIClientOutput, error) {
	client, apiKey, err := h.service.Create(input.Body)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("failed to create api client")
	}
	out := &CreateAPIClientOutput{}
	out.Body.Client = client
	out.Body.APIKey = apiKey
	return out, nil
}

func (h *APIClientHandler) DeleteAPIClient(_ context.Context, input *DeleteAPIClientInput) (*struct{}, error) {
	if err := h.service.Delete(input.ID); err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("api client %d not found", input.ID))
	}
	return nil, nil
}

func RegisterAPIClientRoutes(api huma.API, h *APIClientHandler) {
	const prefix = "/api/v1"

	huma.Register(api, huma.Operation{
		OperationID: "list-api-clients",
		Method:      http.MethodGet,
		Path:        prefix + "/api-clients",
		Summary:     "List API clients used by external apps",
		Tags:        []string{"api-clients"},
	}, h.ListAPIClients)

	huma.Register(api, huma.Operation{
		OperationID:   "create-api-client",
		Method:        http.MethodPost,
		Path:          prefix + "/api-clients",
		Summary:       "Create an API client and return a one-time API key",
		Tags:          []string{"api-clients"},
		DefaultStatus: http.StatusCreated,
	}, h.CreateAPIClient)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-api-client",
		Method:        http.MethodDelete,
		Path:          prefix + "/api-clients/{id}",
		Summary:       "Revoke an API client key",
		Tags:          []string{"api-clients"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteAPIClient)
}
