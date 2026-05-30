package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNotificationLocalizer_FormatTimestamp(t *testing.T) {
	ts := time.Date(2026, time.May, 30, 14, 5, 0, 0, time.UTC)

	t.Run("english uses ISO-like format", func(t *testing.T) {
		loc := newNotificationLocalizer("en")
		assert.Equal(t, "2026-05-30 14:05 UTC", loc.formatTimestamp(ts))
	})

	t.Run("french uses day-first format", func(t *testing.T) {
		loc := newNotificationLocalizer("fr")
		assert.Equal(t, "30/05/2026 14:05 UTC", loc.formatTimestamp(ts))
	})

	t.Run("unknown language falls back to english format", func(t *testing.T) {
		loc := newNotificationLocalizer("es")
		assert.Equal(t, "2026-05-30 14:05 UTC", loc.formatTimestamp(ts))
	})
}
