package notifier

import (
	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
)

// Module wires the MultiNotifier and all registered notification backends.
//
// To register a new backend, append an fx.Provide call:
//
//	fx.Provide(
//	    fx.Annotate(
//	        NewMyNotifier,
//	        fx.As(new(domain.Notifier)),
//	        fx.ResultTags(`group:"notifiers"`),
//	    ),
//	),
var Module = fx.Options(
	fx.Provide(NewFactory),
	fx.Provide(
		fx.Annotate(
			NewMultiNotifier,
			fx.As(new(domain.Notifier)),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewNtfyNotifier,
			fx.As(new(domain.Notifier)),
			fx.ResultTags(`group:"notifiers"`),
		),
	),
)
