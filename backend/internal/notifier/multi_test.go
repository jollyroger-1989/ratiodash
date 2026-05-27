package notifier_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/notifier"
)

// stubNotifier is a function-based domain.Notifier for use in MultiNotifier tests.
type stubNotifier func(context.Context, domain.Notification) error

func (s stubNotifier) Notify(ctx context.Context, n domain.Notification) error { return s(ctx, n) }

// buildMulti creates a MultiNotifier from a plain slice, bypassing FX.
func buildMulti(ns ...domain.Notifier) domain.Notifier {
	return notifier.NewMultiNotifier(notifier.MultiParams{Notifiers: ns})
}

func TestMultiNotifier_Notify(t *testing.T) {
	t.Run("calls all backends", func(t *testing.T) {
		called := 0
		stub := stubNotifier(func(_ context.Context, _ domain.Notification) error {
			called++
			return nil
		})

		err := buildMulti(stub, stub).Notify(context.Background(), domain.Notification{})

		require.NoError(t, err)
		assert.Equal(t, 2, called)
	})

	t.Run("calls remaining backends after one fails", func(t *testing.T) {
		called := 0
		failing := stubNotifier(func(_ context.Context, _ domain.Notification) error {
			return errors.New("boom")
		})
		succeeding := stubNotifier(func(_ context.Context, _ domain.Notification) error {
			called++
			return nil
		})

		err := buildMulti(failing, succeeding).Notify(context.Background(), domain.Notification{})

		assert.Error(t, err)
		assert.Equal(t, 1, called)
	})

	t.Run("joins errors from multiple failing backends", func(t *testing.T) {
		err1 := errors.New("first error")
		err2 := errors.New("second error")

		err := buildMulti(
			stubNotifier(func(_ context.Context, _ domain.Notification) error { return err1 }),
			stubNotifier(func(_ context.Context, _ domain.Notification) error { return err2 }),
		).Notify(context.Background(), domain.Notification{})

		assert.ErrorIs(t, err, err1)
		assert.ErrorIs(t, err, err2)
	})

	t.Run("skips nil backends", func(t *testing.T) {
		called := 0
		stub := stubNotifier(func(_ context.Context, _ domain.Notification) error {
			called++
			return nil
		})

		err := buildMulti(nil, stub, nil).Notify(context.Background(), domain.Notification{})

		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})

	t.Run("no-op with empty backend list", func(t *testing.T) {
		err := buildMulti().Notify(context.Background(), domain.Notification{})

		assert.NoError(t, err)
	})
}
