package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/testutil"
)

type scraperTestEnv struct {
	api      humatest.TestAPI
	registry *mocks.MockScraperRegistry
}

func setupScraperHandler(t *testing.T) scraperTestEnv {
	t.Helper()
	registry := mocks.NewMockScraperRegistry(t)
	h := handler.NewScraperHandler(registry)
	api := testutil.NewAPI(t)
	handler.RegisterScraperRoutes(api, h)
	return scraperTestEnv{api: api, registry: registry}
}

func TestScraperHandler_List(t *testing.T) {
	t.Run("returns empty list when registry is empty", func(t *testing.T) {
		env := setupScraperHandler(t)
		env.registry.EXPECT().Keys().Return([]string{})

		resp := env.api.Do(http.MethodGet, "/api/v1/scrapers")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []handler.ScraperInfo
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body)
	})

	t.Run("returns sorted scraper list with credential fields", func(t *testing.T) {
		env := setupScraperHandler(t)
		env.registry.EXPECT().Keys().Return([]string{"unit3d", "generic"})

		genericScraper := mocks.NewMockTrackerScraper(t)
		genericScraper.EXPECT().CredentialFields().Return([]domain.CredentialField{
			{Key: "api_key", Label: "API Key", Type: "password", Required: true},
		})
		unit3dScraper := mocks.NewMockTrackerScraper(t)
		unit3dScraper.EXPECT().CredentialFields().Return([]domain.CredentialField{
			{Key: "token", Label: "Token", Type: "password", Required: true},
		})

		env.registry.EXPECT().Get("generic").Return(genericScraper, true)
		env.registry.EXPECT().Get("unit3d").Return(unit3dScraper, true)

		resp := env.api.Do(http.MethodGet, "/api/v1/scrapers")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []handler.ScraperInfo
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 2)
		// Response must be sorted alphabetically
		assert.Equal(t, "generic", body[0].Key)
		assert.Equal(t, "unit3d", body[1].Key)
		assert.Len(t, body[0].CredentialFields, 1)
	})

	t.Run("skips keys not found in registry", func(t *testing.T) {
		env := setupScraperHandler(t)
		env.registry.EXPECT().Keys().Return([]string{"missing", "present"})
		env.registry.EXPECT().Get("missing").Return(nil, false)

		presentScraper := mocks.NewMockTrackerScraper(t)
		presentScraper.EXPECT().CredentialFields().Return([]domain.CredentialField{})
		env.registry.EXPECT().Get("present").Return(presentScraper, true)

		resp := env.api.Do(http.MethodGet, "/api/v1/scrapers")

		require.Equal(t, http.StatusOK, resp.Code)
		var body []handler.ScraperInfo
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body, 1)
		assert.Equal(t, "present", body[0].Key)
	})
}
