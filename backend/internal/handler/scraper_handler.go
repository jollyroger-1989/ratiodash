package handler

import (
	"context"
	"net/http"
	"sort"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jose/ratiodash/internal/domain"
)

// ScraperHandler exposes the registry of available scrapers.
type ScraperHandler struct {
	registry domain.ScraperRegistry
}

func NewScraperHandler(r domain.ScraperRegistry) *ScraperHandler {
	return &ScraperHandler{registry: r}
}

// ScraperInfo is the API representation of a registered scraper.
type ScraperInfo struct {
	Key              string                   `json:"key"`
	CredentialFields []domain.CredentialField `json:"credential_fields"`
}

type ListScrapersOutput struct {
	Body []ScraperInfo
}

func (h *ScraperHandler) ListScrapers(_ context.Context, _ *struct{}) (*ListScrapersOutput, error) {
	keys := h.registry.Keys()
	sort.Strings(keys)

	infos := make([]ScraperInfo, 0, len(keys))
	for _, k := range keys {
		s, ok := h.registry.Get(k)
		if !ok {
			continue
		}
		infos = append(infos, ScraperInfo{
			Key:              k,
			CredentialFields: s.CredentialFields(),
		})
	}
	return &ListScrapersOutput{Body: infos}, nil
}

func RegisterScraperRoutes(api huma.API, h *ScraperHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "list-scrapers",
		Method:      http.MethodGet,
		Path:        "/api/v1/scrapers",
		Summary:     "List available scrapers with their credential fields",
		Tags:        []string{"scrapers"},
	}, h.ListScrapers)
}
