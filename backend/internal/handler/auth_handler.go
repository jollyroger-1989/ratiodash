package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// AuthHandler handles authentication endpoints and the one-time setup wizard.
type AuthHandler struct {
	service domain.AuthService
}

func NewAuthHandler(svc domain.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// --- I/O types ---

type AuthStatusOutput struct {
	Body struct {
		Setup bool `json:"setup" doc:"true when credentials have been configured"`
	}
}

type SetupAuthInput struct {
	Body domain.SetupInput
}

type LoginAuthInput struct {
	Body domain.LoginInput
}

type LoginAuthOutput struct {
	Body struct {
		Token string `json:"token" doc:"Signed JWT — include as Bearer token on subsequent requests"`
	}
}

type UpdateCredentialsInput struct {
	Body struct {
		CurrentPassword string `json:"current_password" required:"true" minLength:"1"`
		NewUsername     string `json:"new_username"     required:"true" minLength:"1"`
		NewPassword     string `json:"new_password"     required:"true" minLength:"8"`
	}
}

type SettingsLanguageOutput struct {
	Body struct {
		Language string `json:"language" enum:"en,fr"`
	}
}

type UpdateSettingsLanguageInput struct {
	Body struct {
		Language string `json:"language" required:"true" enum:"en,fr"`
	}
}

// --- Handlers ---

// Status reports whether credentials have been set up.
func (h *AuthHandler) Status(_ context.Context, _ *struct{}) (*AuthStatusOutput, error) {
	setup, err := h.service.IsSetup()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to check setup status")
	}
	out := &AuthStatusOutput{}
	out.Body.Setup = setup
	return out, nil
}

// Setup creates the initial credentials. Fails with 409 if already configured.
func (h *AuthHandler) Setup(_ context.Context, input *SetupAuthInput) (*struct{}, error) {
	if err := h.service.Setup(input.Body.Username, input.Body.Password); err != nil {
		if err.Error() == "credentials already configured" {
			return nil, huma.NewError(http.StatusConflict, "credentials already configured")
		}
		return nil, huma.Error500InternalServerError("setup failed")
	}
	return nil, nil
}

// Login verifies credentials and returns a JWT.
func (h *AuthHandler) Login(_ context.Context, input *LoginAuthInput) (*LoginAuthOutput, error) {
	token, err := h.service.Login(input.Body.Username, input.Body.Password)
	if err != nil {
		return nil, huma.NewError(http.StatusUnauthorized, "invalid credentials")
	}
	out := &LoginAuthOutput{}
	out.Body.Token = token
	return out, nil
}

// UpdateCredentials changes the username and/or password. Requires a valid JWT (protected route).
func (h *AuthHandler) UpdateCredentials(_ context.Context, input *UpdateCredentialsInput) (*struct{}, error) {
	err := h.service.UpdateCredentials(
		input.Body.CurrentPassword,
		input.Body.NewUsername,
		input.Body.NewPassword,
	)
	if err != nil {
		if err.Error() == "invalid current password" {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "invalid current password")
		}
		return nil, huma.Error500InternalServerError("failed to update credentials")
	}
	return nil, nil
}

func (h *AuthHandler) GetLanguage(_ context.Context, _ *struct{}) (*SettingsLanguageOutput, error) {
	language, err := h.service.GetLanguage()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch language setting")
	}
	out := &SettingsLanguageOutput{}
	out.Body.Language = language
	return out, nil
}

func (h *AuthHandler) UpdateLanguage(_ context.Context, input *UpdateSettingsLanguageInput) (*struct{}, error) {
	if err := h.service.UpdateLanguage(input.Body.Language); err != nil {
		if err.Error() == "not configured" {
			return nil, huma.NewError(http.StatusConflict, "credentials not configured")
		}
		return nil, huma.Error500InternalServerError("failed to update language setting")
	}
	return nil, nil
}

// RegisterAuthRoutes mounts all auth endpoints on the Huma API.
func RegisterAuthRoutes(api huma.API, h *AuthHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-auth-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/status",
		Summary:     "Check whether credentials have been configured",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, h.Status)

	huma.Register(api, huma.Operation{
		OperationID: "setup-auth",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/setup",
		Summary:     "Create initial credentials (only works when none exist)",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, h.Setup)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Login and receive a signed JWT token",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, h.Login)

	huma.Register(api, huma.Operation{
		OperationID: "update-credentials",
		Method:      http.MethodPatch,
		Path:        "/api/v1/settings/credentials",
		Summary:     "Update the admin username and/or password",
		Tags:        []string{"settings"},
	}, h.UpdateCredentials)

	huma.Register(api, huma.Operation{
		OperationID: "get-settings-language",
		Method:      http.MethodGet,
		Path:        "/api/v1/settings/language",
		Summary:     "Get the persisted UI language",
		Tags:        []string{"settings"},
	}, h.GetLanguage)

	huma.Register(api, huma.Operation{
		OperationID: "update-settings-language",
		Method:      http.MethodPatch,
		Path:        "/api/v1/settings/language",
		Summary:     "Update the persisted UI language",
		Tags:        []string{"settings"},
	}, h.UpdateLanguage)
}
