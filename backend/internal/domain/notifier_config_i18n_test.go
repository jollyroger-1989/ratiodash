package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalizeNotifierTypes(t *testing.T) {
	t.Run("localizes to english", func(t *testing.T) {
		types := LocalizeNotifierTypes(AvailableNotifierTypes, "en")
		require.NotEmpty(t, types)
		assert.Equal(t, "Email (SMTP)", findTypeLabel(t, types, "email"))
		assert.Equal(t, "SMTP Host", findFieldLabel(t, types, "email", "host"))
	})

	t.Run("localizes to french", func(t *testing.T) {
		types := LocalizeNotifierTypes(AvailableNotifierTypes, "fr")
		require.NotEmpty(t, types)
		assert.Equal(t, "Email (SMTP)", findTypeLabel(t, types, "email"))
		assert.Equal(t, "Hôte SMTP", findFieldLabel(t, types, "email", "host"))
		assert.Equal(t, "Mode TLS", findFieldLabel(t, types, "email", "tls_mode"))
	})

	t.Run("falls back to english", func(t *testing.T) {
		types := LocalizeNotifierTypes(AvailableNotifierTypes, "es")
		require.NotEmpty(t, types)
		assert.Equal(t, "Topic URL", findFieldLabel(t, types, "ntfy", "url"))
	})
}

func findTypeLabel(t *testing.T, types []NotifierTypeInfo, key string) string {
	t.Helper()
	for _, tp := range types {
		if tp.Key == key {
			return tp.Label
		}
	}
	t.Fatalf("type %q not found", key)
	return ""
}

func findFieldLabel(t *testing.T, types []NotifierTypeInfo, typeKey, fieldKey string) string {
	t.Helper()
	for _, tp := range types {
		if tp.Key != typeKey {
			continue
		}
		for _, f := range tp.ConfigFields {
			if f.Key == fieldKey {
				return f.Label
			}
		}
	}
	t.Fatalf("field %q in type %q not found", fieldKey, typeKey)
	return ""
}
