package repository

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(NewTrackerRepository),
	fx.Provide(NewStatsRepository),
	fx.Provide(NewAuthRepository),
	fx.Provide(NewAPIClientRepository),
	fx.Provide(NewNotifierConfigRepository),
	fx.Provide(NewReportRepository),
	fx.Provide(NewAlertConfigRepository),
)
