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

	t.Run("builds email notifier with auth and starttls", func(t *testing.T) {
		n, err := factory.Build("email", `{"host":"smtp.example.com","port":"587","from":"alerts@example.com","to":"user@example.com","username":"smtp-user","password":"smtp-pass","tls_mode":"starttls"}`)

		require.NoError(t, err)
		assert.NotNil(t, n)
	})

	t.Run("builds email notifier without auth and defaults tls_mode", func(t *testing.T) {
		n, err := factory.Build("email", `{"host":"smtp.example.com","port":"465","from":"alerts@example.com","to":"user@example.com"}`)

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

	t.Run("returns error for email with missing host", func(t *testing.T) {
		_, err := factory.Build("email", `{"port":"587","from":"alerts@example.com","to":"user@example.com"}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("returns error for email with invalid port", func(t *testing.T) {
		_, err := factory.Build("email", `{"host":"smtp.example.com","port":"abc","from":"alerts@example.com","to":"user@example.com"}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("returns error for email with invalid address", func(t *testing.T) {
		_, err := factory.Build("email", `{"host":"smtp.example.com","port":"587","from":"bad","to":"user@example.com"}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid from address")
	})

	t.Run("returns error for email with invalid tls mode", func(t *testing.T) {
		_, err := factory.Build("email", `{"host":"smtp.example.com","port":"587","from":"alerts@example.com","to":"user@example.com","tls_mode":"auto"}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tls_mode")
	})

	t.Run("returns error for email when only username is provided", func(t *testing.T) {
		_, err := factory.Build("email", `{"host":"smtp.example.com","port":"587","from":"alerts@example.com","to":"user@example.com","username":"smtp-user"}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "username and password")
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

	t.Run("returns error for email invalid JSON", func(t *testing.T) {
		_, err := factory.Build("email", "not-json")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse email config")
	})
}
