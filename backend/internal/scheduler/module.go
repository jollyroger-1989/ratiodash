package scheduler

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(New),
	// Expose *Scheduler as domain.Refresher for handler injection.
	fx.Provide(AsRefresher),
	// Expose *Scheduler as domain.ReportScheduler for handler injection.
	fx.Provide(AsReportScheduler),
	fx.Invoke(Start),
)
