package notifier

import (
	"context"
	"errors"

	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
)

// multiNotifier fans a single Notification out to every registered Notifier.
// A failure from one backend does not prevent the others from being called;
// all errors are joined and returned together.
type multiNotifier struct {
	notifiers []domain.Notifier
}

// MultiParams is the FX-injectable params struct for NewMultiNotifier.
// It can also be constructed directly in tests.
type MultiParams struct {
	fx.In
	Notifiers []domain.Notifier `group:"notifiers"`
}

// NewMultiNotifier is the FX constructor for multiNotifier.
// It receives every implementation registered with group:"notifiers" and
// exposes the result as domain.Notifier.
// Nil entries (backends that returned nil, nil because they are disabled) are
// silently filtered out.
func NewMultiNotifier(p MultiParams) domain.Notifier {
	active := make([]domain.Notifier, 0, len(p.Notifiers))
	for _, n := range p.Notifiers {
		if n != nil {
			active = append(active, n)
		}
	}
	return &multiNotifier{notifiers: active}
}

func (m *multiNotifier) Notify(ctx context.Context, n domain.Notification) error {
	var errs []error
	for _, notifier := range m.notifiers {
		if err := notifier.Notify(ctx, n); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
