package service

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(NewTrackerService),
	fx.Provide(NewStatsService),
	fx.Provide(NewAlertConfigService),
	fx.Provide(NewAlertServiceWithAuthRepo),
	fx.Provide(NewRefreshService),
	fx.Provide(NewAuthService),
	fx.Provide(NewNotifierConfigService),
	fx.Provide(NewReportServiceWithAuthRepo),
)
