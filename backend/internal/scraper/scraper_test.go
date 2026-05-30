package scraper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/scraper"
)

func TestParseCredentials(t *testing.T) {
	t.Run("returns empty credentials for empty string", func(t *testing.T) {
		c, err := scraper.ParseCredentials("")

		require.NoError(t, err)
		assert.Equal(t, scraper.Credentials{}, c)
	})

	t.Run("returns empty credentials for empty JSON object", func(t *testing.T) {
		c, err := scraper.ParseCredentials("{}")

		require.NoError(t, err)
		assert.Equal(t, scraper.Credentials{}, c)
	})

	t.Run("parses full credentials", func(t *testing.T) {
		raw := `{"url":"http://example.com","cookie":"abc","username":"user","password":"pass","api_key":"key","token":"tok"}`
		c, err := scraper.ParseCredentials(raw)

		require.NoError(t, err)
		assert.Equal(t, "http://example.com", c.URL)
		assert.Equal(t, "abc", c.Cookie)
		assert.Equal(t, "user", c.Username)
		assert.Equal(t, "pass", c.Password)
		assert.Equal(t, "key", c.APIKey)
		assert.Equal(t, "tok", c.Token)
	})

	t.Run("parses custom headers", func(t *testing.T) {
		raw := `{"headers":{"X-Custom":"value"}}`
		c, err := scraper.ParseCredentials(raw)

		require.NoError(t, err)
		assert.Equal(t, "value", c.Headers["X-Custom"])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := scraper.ParseCredentials("not-json")

		assert.Error(t, err)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("GenericScraper is registered with key 'generic'", func(t *testing.T) {
		s := scraper.NewGenericScraper()
		assert.Equal(t, "generic", s.Key())
	})

	t.Run("GenericScraper has no credential fields", func(t *testing.T) {
		s := scraper.NewGenericScraper()
		assert.Nil(t, s.CredentialFields())
	})
}
