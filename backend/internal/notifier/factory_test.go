package notifier_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/notifier"
)

func TestFactory_Build(t *testing.T) {
	factory := notifier.NewFactory()

	t.Run("builds ntfy notifier with valid config", func(t *testing.T) {
		n, err := factory.Build("ntfy", `{"url":"http://ntfy.sh/test"}`)

		require.NoError(t, err)
		assert.NotNil(t, n)
	})

	t.Run("builds ntfy notifier with token", func(t *testing.T) {
		n, err := factory.Build("ntfy", `{"url":"http://ntfy.sh/test","token":"tok"}`)

		require.NoError(t, err)
		assert.NotNil(t, n)
	})

	t.Run("returns error for ntfy with missing url", func(t *testing.T) {
		_, err := factory.Build("ntfy", `{"url":""}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "url")
	})

	t.Run("returns error for ntfy with empty config", func(t *testing.T) {
		_, err := factory.Build("ntfy", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "url")
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		_, err := factory.Build("unknown", "{}")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown notifier type")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := factory.Build("ntfy", "not-json")

		require.Error(t, err)
	})
}
